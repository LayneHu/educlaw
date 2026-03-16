package llm

import (
	"context"
	"fmt"
)

// DisabledProvider returns a stable error instead of silently falling back to OpenAI defaults.
type DisabledProvider struct {
	reason string
}

func NewDisabledProvider(reason string) *DisabledProvider {
	return &DisabledProvider{reason: reason}
}

func (p *DisabledProvider) ModelName() string {
	return "disabled"
}

func (p *DisabledProvider) Complete(_ context.Context, _ CompletionRequest) (*CompletionResponse, error) {
	return nil, fmt.Errorf("LLM is not configured: %s", p.reason)
}

func (p *DisabledProvider) StreamComplete(_ context.Context, _ CompletionRequest, _ func(string)) (*CompletionResponse, error) {
	return nil, fmt.Errorf("LLM is not configured: %s", p.reason)
}
