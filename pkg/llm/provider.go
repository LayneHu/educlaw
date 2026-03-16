package llm

import (
	"context"
	"log"
)

// Provider is the interface for LLM backends.
// Both Client and MultiProvider implement this interface.
type Provider interface {
	StreamComplete(ctx context.Context, req CompletionRequest, onToken func(string)) (*CompletionResponse, error)
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	ModelName() string
}

// MultiProvider tries providers in order, using the next one on error.
// This provides transparent fallback across different LLM services.
type MultiProvider struct {
	providers []Provider
}

// NewMultiProvider creates a MultiProvider from an ordered list of providers.
// The first provider is primary; subsequent ones are fallbacks.
func NewMultiProvider(providers ...Provider) *MultiProvider {
	return &MultiProvider{providers: providers}
}

// ModelName returns the primary provider's model name.
func (mp *MultiProvider) ModelName() string {
	if len(mp.providers) > 0 {
		return mp.providers[0].ModelName()
	}
	return "none"
}

// StreamComplete tries each provider in order, falling back on error.
// Note: if the primary provider has already streamed tokens before failing,
// the fallback will re-stream from scratch (tokens may be duplicated in that
// edge case). For the heartbeat/cron path, use Complete instead.
func (mp *MultiProvider) StreamComplete(ctx context.Context, req CompletionRequest, onToken func(string)) (*CompletionResponse, error) {
	var lastErr error
	for i, p := range mp.providers {
		resp, err := p.StreamComplete(ctx, req, onToken)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if i < len(mp.providers)-1 {
			log.Printf("[llm] provider %s failed (attempt %d/%d): %v — trying fallback",
				p.ModelName(), i+1, len(mp.providers), err)
		}
	}
	return nil, lastErr
}

// Complete tries each provider in order, falling back on error.
func (mp *MultiProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	var lastErr error
	for i, p := range mp.providers {
		resp, err := p.Complete(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if i < len(mp.providers)-1 {
			log.Printf("[llm] provider %s failed (attempt %d/%d): %v — trying fallback",
				p.ModelName(), i+1, len(mp.providers), err)
		}
	}
	return nil, lastErr
}
