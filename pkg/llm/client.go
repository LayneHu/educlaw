package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/pingjie/educlaw/pkg/config"
	openai "github.com/sashabaranov/go-openai"
)

// thoughtSignatureTransport handles Gemini thinking-model quirks transparently:
//
//  1. On outgoing requests: injects stored thought_signatures back into
//     assistant-role messages that contain tool_calls (Gemini requires these
//     to be round-tripped or the next call returns 400).
//
//  2. On incoming SSE responses: scans each "data:" chunk for
//     thought_signature fields inside tool_call deltas and stores them keyed
//     by tool call ID.
type thoughtSignatureTransport struct {
	base http.RoundTripper
	mu   sync.Mutex
	sigs map[string]string // tool call id → thought_signature
}

func (t *thoughtSignatureTransport) store(id, sig string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.sigs == nil {
		t.sigs = make(map[string]string)
	}
	t.sigs[id] = sig
	log.Printf("[thought_signature] stored sig for id=%s (len=%d)", id, len(sig))
}

func (t *thoughtSignatureTransport) get(id string) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	s, ok := t.sigs[id]
	return s, ok
}

func (t *thoughtSignatureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body == nil || !strings.Contains(req.URL.Path, "chat/completions") {
		return t.base.RoundTrip(req)
	}

	bodyBytes, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}

	// Inject thought_signatures into any assistant messages with tool_calls
	bodyBytes = t.injectSignatures(bodyBytes)
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	req.ContentLength = int64(len(bodyBytes))

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Wrap the response body to capture thought_signatures from either
	// streaming (SSE line-by-line) or non-streaming (JSON body on Close).
	resp.Body = &sigCapturingBody{
		ReadCloser: resp.Body,
		transport:  t,
		indexToID:  make(map[int]string),
		// all and lineBuf start as nil slices, appended on first Read
	}
	return resp, nil
}

// injectSignatures rewrites the "messages" array in the JSON body, adding
// thought_signature to tool_calls in assistant messages when we have a stored
// signature for that tool call ID.
func (t *thoughtSignatureTransport) injectSignatures(body []byte) []byte {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(body, &doc); err != nil {
		return body
	}
	messagesRaw, ok := doc["messages"]
	if !ok {
		return body
	}
	var messages []json.RawMessage
	if err := json.Unmarshal(messagesRaw, &messages); err != nil {
		return body
	}

	modified := false
	for i, msgRaw := range messages {
		var msg map[string]json.RawMessage
		if err := json.Unmarshal(msgRaw, &msg); err != nil {
			continue
		}
		var role string
		json.Unmarshal(msg["role"], &role) //nolint:errcheck
		if role != "assistant" {
			continue
		}
		toolCallsRaw, ok := msg["tool_calls"]
		if !ok {
			continue
		}
		var toolCalls []map[string]json.RawMessage
		if err := json.Unmarshal(toolCallsRaw, &toolCalls); err != nil {
			continue
		}
		tcChanged := false
		for j, tc := range toolCalls {
			if _, already := tc["thought_signature"]; already {
				continue
			}
			var id string
			json.Unmarshal(tc["id"], &id) //nolint:errcheck
			if id == "" {
				continue
			}
			sig, ok := t.get(id)
			if !ok {
				log.Printf("[thought_signature] inject: no sig found for id=%s", id)
				continue
			}
			tc["thought_signature"], _ = json.Marshal(sig)
			toolCalls[j] = tc
			tcChanged = true
			modified = true
			log.Printf("[thought_signature] inject: injected sig for id=%s", id)
		}
		if tcChanged {
			newTCs, _ := json.Marshal(toolCalls)
			msg["tool_calls"] = newTCs
			messages[i], _ = json.Marshal(msg)
		}
	}

	if !modified {
		return body
	}
	doc["messages"], _ = json.Marshal(messages)
	out, err := json.Marshal(doc)
	if err != nil {
		return body
	}
	return out
}

// sigCapturingBody wraps a response body (streaming SSE or plain JSON) and
// extracts thought_signatures so they can be round-tripped in subsequent calls.
type sigCapturingBody struct {
	io.ReadCloser
	transport *thoughtSignatureTransport
	all       []byte         // complete copy of all bytes received (never consumed)
	lineBuf   []byte         // sliding window for SSE line parsing (consumed)
	indexToID map[int]string // SSE: tool_call index → id across chunks
}

func (b *sigCapturingBody) Read(p []byte) (n int, err error) {
	n, err = b.ReadCloser.Read(p)
	if n > 0 {
		b.all = append(b.all, p[:n]...) // preserve for JSON close-time parse
		b.lineBuf = append(b.lineBuf, p[:n]...)
		b.drainLines()
	}
	return
}

func (b *sigCapturingBody) Close() error {
	trimmed := bytes.TrimSpace(b.all)
	if len(trimmed) > 0 {
		if trimmed[0] == '{' {
			// Non-streaming: plain JSON body — extract thought_signatures directly.
			b.extractFromJSON(b.all)
		} else {
			// Streaming SSE: do a full re-scan of all accumulated data as a
			// safety net in case drainLines() missed anything mid-read.
			for _, line := range bytes.Split(b.all, []byte("\n")) {
				line = bytes.TrimRight(line, "\r")
				if !bytes.HasPrefix(line, []byte("data: ")) {
					continue
				}
				data := bytes.TrimPrefix(line, []byte("data: "))
				data = bytes.TrimSpace(data)
				if len(data) == 0 || string(data) == "[DONE]" {
					continue
				}
				b.extractFromSSEChunk(data)
			}
		}
	}
	return b.ReadCloser.Close()
}

// drainLines processes complete lines from b.lineBuf (SSE streams).
// Non-data lines are consumed but ignored; only "data: {...}" lines are parsed.
func (b *sigCapturingBody) drainLines() {
	for {
		idx := bytes.IndexByte(b.lineBuf, '\n')
		if idx < 0 {
			return
		}
		line := bytes.TrimRight(b.lineBuf[:idx], "\r")
		b.lineBuf = b.lineBuf[idx+1:]

		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := bytes.TrimPrefix(line, []byte("data: "))
		data = bytes.TrimSpace(data)
		if len(data) == 0 || string(data) == "[DONE]" {
			continue
		}
		b.extractFromSSEChunk(data)
	}
}

// sseToolDelta is the minimal shape to extract thought_signatures from streaming chunks.
type sseToolDelta struct {
	Choices []struct {
		Delta struct {
			// Some models put thought_signature on the delta itself
			ThoughtSignature string `json:"thought_signature"`
			ToolCalls        []struct {
				Index            *int   `json:"index"`
				ID               string `json:"id"`
				ThoughtSignature string `json:"thought_signature"`
			} `json:"tool_calls"`
		} `json:"delta"`
		// Some models put it on the choice
		ThoughtSignature string `json:"thought_signature"`
	} `json:"choices"`
	// Some models put it at the root
	ThoughtSignature string `json:"thought_signature"`
}

func (b *sigCapturingBody) extractFromSSEChunk(data []byte) {
	var chunk sseToolDelta
	if err := json.Unmarshal(data, &chunk); err != nil {
		return
	}
	for _, choice := range chunk.Choices {
		for _, tc := range choice.Delta.ToolCalls {
			// Build index → ID map
			if tc.Index != nil && tc.ID != "" {
				b.indexToID[*tc.Index] = tc.ID
			}
			// Capture thought_signature at tool_call level
			if tc.ThoughtSignature != "" {
				id := tc.ID
				if id == "" && tc.Index != nil {
					id = b.indexToID[*tc.Index]
				}
				if id != "" {
					b.transport.store(id, tc.ThoughtSignature)
				} else {
					log.Printf("[thought_signature] SSE chunk: sig present but no id (index=%v)", tc.Index)
				}
			} else if tc.ID != "" {
				log.Printf("[thought_signature] SSE chunk: tool_call id=%s index=%v no sig", tc.ID, tc.Index)
			}
		}
		// Also check if sig is at delta level (some Gemini variants)
		if choice.Delta.ThoughtSignature != "" && len(choice.Delta.ToolCalls) == 0 {
			log.Printf("[thought_signature] SSE chunk: delta-level sig (no tool_calls in this chunk)")
		}
	}
}

// extractFromJSON handles non-streaming (plain JSON) response bodies.
func (b *sigCapturingBody) extractFromJSON(data []byte) {
	var resp struct {
		Choices []struct {
			Message struct {
				ToolCalls []struct {
					ID               string `json:"id"`
					ThoughtSignature string `json:"thought_signature"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return
	}
	for _, choice := range resp.Choices {
		for _, tc := range choice.Message.ToolCalls {
			if tc.ID != "" && tc.ThoughtSignature != "" {
				b.transport.store(tc.ID, tc.ThoughtSignature)
			} else if tc.ID != "" {
				log.Printf("[thought_signature] JSON body: tool_call id=%s has NO thought_signature", tc.ID)
			}
		}
	}
}

// Message represents a chat message.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ToolCall represents a tool invocation by the LLM.
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// Tool represents a tool definition for the LLM.
type Tool struct {
	Type     string   `json:"type"`
	Function ToolFunc `json:"function"`
}

// ToolFunc holds the tool function definition.
type ToolFunc struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// CompletionRequest holds parameters for a completion request.
type CompletionRequest struct {
	Messages    []Message
	Tools       []Tool
	Temperature float64
	MaxTokens   int
}

// CompletionResponse holds the result of a completion.
type CompletionResponse struct {
	Content   string
	ToolCalls []ToolCall
}

// Client wraps an OpenAI-compatible LLM client.
// It implements the Provider interface.
type Client struct {
	client    *openai.Client
	model     string
	transport *thoughtSignatureTransport // non-nil only for Gemini (thought_signature handling)
}

// ModelName returns the model identifier for this client.
func (c *Client) ModelName() string { return c.model }

// isGeminiProvider returns true when the config is for a Gemini/Google backend
// that requires thought_signature round-tripping.
func isGeminiProvider(cfg *config.ModelConfig) bool {
	p := strings.ToLower(cfg.Provider)
	if p == "gemini" || p == "google" {
		return true
	}
	if p != "" {
		return false // explicit non-Gemini provider
	}
	// Infer from API base URL or model name
	if strings.Contains(cfg.APIBase, "googleapis.com") {
		return true
	}
	lm := strings.ToLower(cfg.Model)
	return strings.HasPrefix(lm, "gemini")
}

// NewClient creates a new OpenAI-compatible LLM client.
// For Gemini backends it wraps the transport to handle thought_signature round-tripping.
func NewClient(cfg *config.ModelConfig) *Client {
	ocfg := openai.DefaultConfig(cfg.APIKey)
	if cfg.APIBase != "" {
		ocfg.BaseURL = cfg.APIBase
	}

	// Build base transport (with optional proxy)
	var baseTransport http.RoundTripper = http.DefaultTransport
	if cfg.Proxy != "" {
		if proxyURL, err := url.Parse(cfg.Proxy); err == nil {
			baseTransport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
		}
	}

	if isGeminiProvider(cfg) {
		// Gemini requires thought_signature injection for tool calls
		tst := &thoughtSignatureTransport{base: baseTransport}
		ocfg.HTTPClient = &http.Client{Transport: tst}
		return &Client{
			client:    openai.NewClientWithConfig(ocfg),
			model:     cfg.Model,
			transport: tst,
		}
	}

	// Non-Gemini providers (OpenAI, DeepSeek, etc.) use plain transport
	ocfg.HTTPClient = &http.Client{Transport: baseTransport}
	return &Client{
		client: openai.NewClientWithConfig(ocfg),
		model:  cfg.Model,
	}
}

// hasSigsForAll returns true if every tool call in tcs has a stored thought_signature.
// Returns true immediately for non-Gemini clients (no transport = no sig needed).
func (c *Client) hasSigsForAll(tcs []ToolCall) bool {
	if c.transport == nil {
		return true // non-Gemini: thought_signatures not required
	}
	c.transport.mu.Lock()
	defer c.transport.mu.Unlock()
	if c.transport.sigs == nil {
		log.Printf("[thought_signature] hasSigsForAll: sigs map is nil")
		return false
	}
	for _, tc := range tcs {
		if tc.ID != "" {
			if _, ok := c.transport.sigs[tc.ID]; !ok {
				log.Printf("[thought_signature] hasSigsForAll: missing sig for id=%s", tc.ID)
				return false
			}
		}
	}
	return true
}

func convertMessages(msgs []Message) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, 0, len(msgs))
	for _, m := range msgs {
		msg := openai.ChatCompletionMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
			Name:       m.Name,
		}
		if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				msg.ToolCalls = append(msg.ToolCalls, openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolType(tc.Type),
					Function: openai.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}
		result = append(result, msg)
	}
	return result
}

func convertTools(tools []Tool) []openai.Tool {
	result := make([]openai.Tool, 0, len(tools))
	for _, t := range tools {
		result = append(result, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  t.Function.Parameters,
			},
		})
	}
	return result
}

func usesMaxCompletionTokens(model string) bool {
	lm := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasPrefix(lm, "o1"),
		strings.HasPrefix(lm, "o3"),
		strings.HasPrefix(lm, "o4"),
		strings.HasPrefix(lm, "gpt-5"):
		return true
	default:
		return false
	}
}

func (c *Client) applyTokenLimit(creq *openai.ChatCompletionRequest, maxTokens int) {
	if maxTokens <= 0 {
		return
	}
	if usesMaxCompletionTokens(c.model) {
		creq.MaxCompletionTokens = maxTokens
		return
	}
	creq.MaxTokens = maxTokens
}

// Complete sends a non-streaming completion request.
func (c *Client) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	msgs := convertMessages(req.Messages)
	tools := convertTools(req.Tools)

	creq := openai.ChatCompletionRequest{
		Model:       c.model,
		Messages:    msgs,
		Temperature: float32(req.Temperature),
	}
	c.applyTokenLimit(&creq, req.MaxTokens)
	if len(tools) > 0 {
		creq.Tools = tools
	}

	resp, err := c.client.CreateChatCompletion(ctx, creq)
	if err != nil {
		return nil, fmt.Errorf("LLM completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return &CompletionResponse{}, nil
	}

	choice := resp.Choices[0]
	result := &CompletionResponse{
		Content: choice.Message.Content,
	}

	for _, tc := range choice.Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:   tc.ID,
			Type: string(tc.Type),
			Function: struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			}{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}

	return result, nil
}

// StreamComplete sends a true SSE streaming request, calling onToken for each
// text token as it arrives.  After the stream closes, sigCapturingBody.Close()
// does a full re-scan of the raw SSE bytes for thought_signatures.
// If Gemini's SSE stream does not include thought_signatures (they may only
// appear in non-streaming responses), a silent non-streaming call is made as a
// fallback so that the next turn can inject them back.
func (c *Client) StreamComplete(ctx context.Context, req CompletionRequest, onToken func(string)) (*CompletionResponse, error) {
	msgs := convertMessages(req.Messages)
	tools := convertTools(req.Tools)

	creq := openai.ChatCompletionRequest{
		Model:       c.model,
		Messages:    msgs,
		Temperature: float32(req.Temperature),
		Stream:      true,
	}
	c.applyTokenLimit(&creq, req.MaxTokens)
	if len(tools) > 0 {
		creq.Tools = tools
	}

	stream, err := c.client.CreateChatCompletionStream(ctx, creq)
	if err != nil {
		return nil, fmt.Errorf("LLM stream: %w", err)
	}

	var fullContent string
	toolCallMap := make(map[int]*ToolCall)

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			stream.Close()
			return nil, fmt.Errorf("stream recv: %w", err)
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta

		// Stream text tokens to caller
		if delta.Content != "" {
			fullContent += delta.Content
			if onToken != nil {
				onToken(delta.Content)
			}
		}

		// Accumulate tool calls by index
		for di, dtc := range delta.ToolCalls {
			var i int
			if dtc.Index != nil {
				i = *dtc.Index
			} else {
				i = di
			}
			if _, exists := toolCallMap[i]; !exists {
				toolCallMap[i] = &ToolCall{Type: "function"}
			}
			tc := toolCallMap[i]
			if dtc.ID != "" {
				tc.ID = dtc.ID
			}
			if dtc.Function.Name != "" {
				tc.Function.Name += dtc.Function.Name
			}
			if dtc.Function.Arguments != "" {
				tc.Function.Arguments += dtc.Function.Arguments
			}
		}
	}

	// Explicit close triggers sigCapturingBody.Close() → full SSE re-scan
	// for thought_signatures on the accumulated raw buffer.
	stream.Close()

	result := &CompletionResponse{Content: fullContent}
	for i := 0; i < len(toolCallMap); i++ {
		if tc, ok := toolCallMap[i]; ok {
			if tc.ID == "" {
				tc.ID = fmt.Sprintf("call_%d", i)
			}
			result.ToolCalls = append(result.ToolCalls, *tc)
		}
	}

	// Safety net: if tool calls were returned but thought_signatures are
	// still missing (Gemini SSE may not include them in stream chunks),
	// make a non-streaming call to get the authoritative response which
	// DOES include thought_signatures in the JSON body.
	//
	// CRITICAL: we MUST use the non-streaming response's tool calls
	// (not the streaming ones), because the thought_signatures are stored
	// keyed by tool call ID — and the IDs from non-streaming will match
	// what we stored, while the streaming IDs would not.
	if len(result.ToolCalls) > 0 && !c.hasSigsForAll(result.ToolCalls) {
		log.Printf("[thought_signature] SSE missing sigs for %d tool calls, falling back to non-streaming", len(result.ToolCalls))
		nonStreamResp, err := c.Complete(ctx, req)
		if err == nil && len(nonStreamResp.ToolCalls) > 0 {
			// Use non-streaming tool calls — their IDs match stored sigs
			result.ToolCalls = nonStreamResp.ToolCalls
			log.Printf("[thought_signature] using non-streaming tool calls (count=%d)", len(result.ToolCalls))
		} else if err != nil {
			log.Printf("[thought_signature] non-streaming fallback error: %v", err)
		}
	}

	return result, nil
}
