package serve

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pingjie/educlaw/pkg/config"
	"github.com/pingjie/educlaw/pkg/llm"
)

func TestBuildLLMProviderWithoutAPIKeyReturnsDisabledProvider(t *testing.T) {
	cfg := config.Default()
	cfg.ModelList[0].APIKey = ""

	provider := buildLLMProvider(cfg)
	if provider.ModelName() != "disabled" {
		t.Fatalf("ModelName() = %q, want %q", provider.ModelName(), "disabled")
	}

	_, err := provider.Complete(context.Background(), llm.CompletionRequest{})
	if err == nil || !strings.Contains(err.Error(), "LLM is not configured") {
		t.Fatalf("Complete() error = %v", err)
	}
}

func TestBuildLLMProviderWithRepoConfigUsesConfiguredModel(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", "..", "..", "config.json"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	mc, name, err := cfg.ResolveModelSelection()
	if err != nil {
		t.Fatalf("ResolveModelSelection() error = %v", err)
	}
	if !strings.EqualFold(name, "minimax-default") {
		t.Fatalf("model_name = %q, want %q", name, "minimax-default")
	}
	if !strings.EqualFold(mc.Provider, "minimax") {
		t.Fatalf("provider = %q, want %q", mc.Provider, "minimax")
	}
	if mc.APIKey == "" {
		t.Fatalf("api key is empty")
	}

	provider := buildLLMProvider(cfg)
	if provider.ModelName() == "disabled" {
		t.Fatalf("provider unexpectedly disabled")
	}
}
