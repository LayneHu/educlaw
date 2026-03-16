package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoadPreservesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := Default()
	cfg.ModelList[0].APIKey = "sk-test"

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Server.Host != "127.0.0.1" {
		t.Fatalf("Server.Host = %q, want %q", loaded.Server.Host, "127.0.0.1")
	}
	if loaded.Agent.MaxTokens != 8192 {
		t.Fatalf("Agent.MaxTokens = %d, want %d", loaded.Agent.MaxTokens, 8192)
	}
	if loaded.LLM.Primary.Model != "minimax-default" {
		t.Fatalf("LLM.Primary.Model = %q", loaded.LLM.Primary.Model)
	}
	mc, name, err := loaded.ResolveModelSelection()
	if err != nil {
		t.Fatalf("ResolveModelSelection() error = %v", err)
	}
	if name != "minimax-default" || mc.Model != "MiniMax-M2.5" {
		t.Fatalf("resolved model = %q -> %q", name, mc.Model)
	}
}

func TestLoadRejectsConfigWithoutModelList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := []byte(`{
  "llm": {
    "primary": {
      "provider": "deepseek",
      "api_key": "sk-test",
      "model": "deepseek-chat",
      "api_base": "https://api.deepseek.com/v1"
    }
  }
}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err == nil {
		t.Fatalf("Load() error = nil, want model_list validation error")
	}
	if cfg != nil {
		t.Fatalf("Load() cfg != nil, want nil on error")
	}
	if !strings.Contains(err.Error(), "model_list is empty") {
		t.Fatalf("Load() error = %q, want model_list is empty", err.Error())
	}
}
