package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	reHTMLTag    = regexp.MustCompile(`<[^>]+>`)
	reWhitespace = regexp.MustCompile(`\s{2,}`)
	reScriptCSS  = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
)

// stripHTML removes HTML tags and returns cleaned plain text.
func stripHTML(html string) string {
	s := reScriptCSS.ReplaceAllString(html, " ")
	s = reHTMLTag.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = reWhitespace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func httpGet(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; EduClaw/1.0)")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, rawURL)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024)) // max 512 KB
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// ---- WebFetchTool ----

// WebFetchTool fetches a URL and returns its text content.
type WebFetchTool struct{}

// NewWebFetchTool creates a new WebFetchTool.
func NewWebFetchTool() *WebFetchTool { return &WebFetchTool{} }

func (t *WebFetchTool) Name() string { return "web_fetch" }
func (t *WebFetchTool) Description() string {
	return "Fetch the content of a web page and return its text. Useful for getting educational articles, Wikipedia pages, or reference material."
}
func (t *WebFetchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch",
			},
			"max_chars": map[string]any{
				"type":        "integer",
				"description": "Maximum characters to return (default 4000)",
			},
		},
		"required": []string{"url"},
	}
}
func (t *WebFetchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	rawURL, _ := args["url"].(string)
	if rawURL == "" {
		return "", fmt.Errorf("url is required")
	}
	maxChars := 4000
	if v, ok := args["max_chars"].(float64); ok && v > 0 {
		maxChars = int(v)
	}

	body, err := httpGet(ctx, rawURL)
	if err != nil {
		return "", fmt.Errorf("fetching %s: %w", rawURL, err)
	}

	text := stripHTML(body)
	if len(text) > maxChars {
		text = text[:maxChars] + "\n[... content truncated ...]"
	}
	return text, nil
}

// ---- WebSearchTool ----

// WebSearchTool searches the web using DuckDuckGo Lite.
type WebSearchTool struct{}

// NewWebSearchTool creates a new WebSearchTool.
func NewWebSearchTool() *WebSearchTool { return &WebSearchTool{} }

func (t *WebSearchTool) Name() string { return "web_search" }
func (t *WebSearchTool) Description() string {
	return "Search the web for educational content. Returns a list of relevant results with titles, URLs, and snippets."
}
func (t *WebSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query",
			},
		},
		"required": []string{"query"},
	}
}

// reSearchResult matches DuckDuckGo Lite result snippets.
var (
	reSearchTitle   = regexp.MustCompile(`(?i)<a[^>]+class="result-link"[^>]*>([^<]+)</a>`)
	reSearchSnippet = regexp.MustCompile(`(?i)<td[^>]+class="result-snippet"[^>]*>(.*?)</td>`)
	reSearchURL     = regexp.MustCompile(`(?i)<a[^>]+href="([^"]+)"[^>]*class="result-link"`)
)

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return "", fmt.Errorf("query is required")
	}

	searchURL := "https://lite.duckduckgo.com/lite/?q=" + url.QueryEscape(query)
	body, err := httpGet(ctx, searchURL)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	titles := reSearchTitle.FindAllStringSubmatch(body, 8)
	snippets := reSearchSnippet.FindAllStringSubmatch(body, 8)
	urls := reSearchURL.FindAllStringSubmatch(body, 8)

	if len(titles) == 0 {
		return "No search results found.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for: %s\n\n", query))
	for i := range titles {
		title := strings.TrimSpace(titles[i][1])
		link := ""
		if i < len(urls) {
			link = strings.TrimSpace(urls[i][1])
		}
		snippet := ""
		if i < len(snippets) {
			snippet = strings.TrimSpace(stripHTML(snippets[i][1]))
		}
		sb.WriteString(fmt.Sprintf("%d. **%s**\n   %s\n   %s\n\n", i+1, title, snippet, link))
	}
	return sb.String(), nil
}
