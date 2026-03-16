package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config is the root configuration structure.
type Config struct {
	LLM       LLMConfig       `json:"llm"`
	ModelList []ModelConfig   `json:"model_list,omitempty"`
	Workspace string          `json:"workspace"`
	Server    ServerConfig    `json:"server"`
	Agent     AgentConfig     `json:"agent"`
	Health    HealthConfig    `json:"health"`
	Skills    SkillsConfig    `json:"skills"`
	Heartbeat HeartbeatConfig `json:"heartbeat"`
	Cron      CronConfig      `json:"cron"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// AgentConfig holds agent behavior settings.
type AgentConfig struct {
	MaxIterations int     `json:"max_iterations"`
	Temperature   float64 `json:"temperature"`
	MaxTokens     int     `json:"max_tokens"`
}

// HealthConfig controls HTTP health endpoints on the main web server.
type HealthConfig struct {
	Enabled bool `json:"enabled"`
}

// SkillsConfig holds skills-related settings.
type SkillsConfig struct {
	BuiltinDir string `json:"builtin_dir"`
}

// HeartbeatConfig holds heartbeat settings.
type HeartbeatConfig struct {
	Enabled         bool `json:"enabled"`
	IntervalMinutes int  `json:"interval_minutes"`
}

// LLMProviderConfig holds settings for a single LLM provider.
// Provider values: openai, gemini, deepseek, openrouter, anthropic, groq, ollama, zhipu, mistral, moonshot, etc.
// If Provider is empty, it is inferred from APIBase or Model name.
type LLMProviderConfig struct {
	ModelName string `json:"model_name,omitempty"`
	Provider  string `json:"provider"` // e.g. "openai", "gemini", "deepseek"
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	APIBase   string `json:"api_base"`
	Proxy     string `json:"proxy"`
}

// LLMConfig supports a primary provider plus optional fallbacks.
type LLMConfig struct {
	Primary   LLMProviderConfig   `json:"primary"`
	Fallbacks []LLMProviderConfig `json:"fallbacks"`
}

// ModelConfig represents a model-centric provider entry.
// When model_list is used, llm.primary.model / fallbacks[].model refer to model_name.
type ModelConfig struct {
	ModelName      string `json:"model_name"`
	Provider       string `json:"provider,omitempty"`
	Model          string `json:"model"`
	APIKey         string `json:"api_key"`
	APIBase        string `json:"api_base,omitempty"`
	Proxy          string `json:"proxy,omitempty"`
	MaxTokensField string `json:"max_tokens_field,omitempty"`
	ThinkingLevel  string `json:"thinking_level,omitempty"`
}

// CronConfig holds settings for the cron scheduler.
type CronConfig struct {
	Enabled bool `json:"enabled"`
}

// Default returns a config with sensible defaults for first-run setup.
func Default() *Config {
	return &Config{
		LLM: LLMConfig{
			Primary: LLMProviderConfig{
				Model: "minimax-default",
			},
		},
		ModelList: []ModelConfig{
			{
				ModelName: "minimax-default",
				Provider:  "minimax",
				Model:     "MiniMax-M2.5",
				APIBase:   "https://api.minimaxi.com/v1",
			},
		},
		Workspace: "~/.educlaw",
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 18080,
		},
		Agent: AgentConfig{
			MaxIterations: 15,
			Temperature:   0.7,
			MaxTokens:     8192,
		},
		Health: HealthConfig{
			Enabled: true,
		},
		Skills: SkillsConfig{
			BuiltinDir: "./skills",
		},
	}
}

// Load reads configuration from a JSON file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	cfg := Default()
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		if _, ok := raw["model_list"]; !ok {
			cfg.ModelList = nil
		}
		if _, ok := raw["llm"]; !ok {
			cfg.LLM = LLMConfig{}
		}
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	cfg.normalizeModelList()
	if err := cfg.ValidateModelList(); err != nil {
		return nil, fmt.Errorf("validating model_list: %w", err)
	}

	// Set defaults
	return cfg, nil
}

// Save writes configuration to disk in pretty-printed JSON format.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config file %s: %w", path, err)
	}
	return nil
}

// GetModelConfig resolves a model alias from model_list.
func (c *Config) GetModelConfig(modelName string) (*ModelConfig, error) {
	for i := range c.ModelList {
		if c.ModelList[i].ModelName == modelName {
			return &c.ModelList[i], nil
		}
	}
	return nil, fmt.Errorf("model %q not found in model_list", modelName)
}

// ResolveModelSelection resolves the active model alias to a model config.
func (c *Config) ResolveModelSelection() (*ModelConfig, string, error) {
	if len(c.ModelList) == 0 {
		return nil, "", fmt.Errorf("model_list is empty")
	}
	name := strings.TrimSpace(c.LLM.Primary.Model)
	if name == "" {
		name = c.ModelList[0].ModelName
	}
	mc, err := c.GetModelConfig(name)
	if err != nil {
		return nil, "", err
	}
	return mc, name, nil
}

func (c *Config) ResolveFallbackSelections() ([]ModelConfig, error) {
	if len(c.ModelList) == 0 {
		return nil, nil
	}
	items := make([]ModelConfig, 0, len(c.LLM.Fallbacks))
	for _, fb := range c.LLM.Fallbacks {
		if strings.TrimSpace(fb.Model) == "" {
			continue
		}
		mc, err := c.GetModelConfig(strings.TrimSpace(fb.Model))
		if err != nil {
			return nil, err
		}
		items = append(items, *mc)
	}
	return items, nil
}

func (c *Config) ValidateModelList() error {
	if len(c.ModelList) == 0 {
		return fmt.Errorf("model_list is empty")
	}
	seen := make(map[string]struct{}, len(c.ModelList))
	for i, item := range c.ModelList {
		if strings.TrimSpace(item.ModelName) == "" {
			return fmt.Errorf("model_list[%d].model_name is required", i)
		}
		if strings.TrimSpace(item.Model) == "" {
			return fmt.Errorf("model_list[%d].model is required", i)
		}
		if _, ok := seen[item.ModelName]; ok {
			return fmt.Errorf("duplicate model_name %q", item.ModelName)
		}
		seen[item.ModelName] = struct{}{}
	}
	return nil
}

func (c *Config) normalizeModelList() {
	for i := range c.ModelList {
		if c.ModelList[i].Provider == "" {
			c.ModelList[i].Provider = inferProvider(c.ModelList[i].Model)
		}
		if c.ModelList[i].APIBase == "" {
			c.ModelList[i].APIBase = defaultAPIBase(c.ModelList[i].Provider)
		}
	}
}

func inferProvider(model string) string {
	lm := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasPrefix(lm, "gpt"), strings.HasPrefix(lm, "o1"), strings.HasPrefix(lm, "o3"), strings.HasPrefix(lm, "o4"):
		return "openai"
	case strings.HasPrefix(lm, "gemini"):
		return "gemini"
	case strings.HasPrefix(lm, "claude"):
		return "anthropic"
	case strings.HasPrefix(lm, "deepseek"):
		return "deepseek"
	case strings.Contains(lm, "glm"), strings.Contains(lm, "zhipu"):
		return "zhipu"
	case strings.Contains(lm, "mistral"), strings.Contains(lm, "mixtral"):
		return "mistral"
	case strings.Contains(lm, "moonshot"), strings.Contains(lm, "kimi"):
		return "moonshot"
	case strings.Contains(lm, "minimax"), strings.Contains(lm, "abab"), strings.Contains(lm, "m2.5"):
		return "minimax"
	case strings.Contains(lm, "qwen"), strings.Contains(lm, "llama"):
		return "ollama"
	}
	return ""
}

func defaultAPIBase(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "openai", "gpt":
		return "https://api.openai.com/v1"
	case "gemini", "google":
		return "https://generativelanguage.googleapis.com/v1beta/openai"
	case "deepseek":
		return "https://api.deepseek.com/v1"
	case "anthropic", "claude":
		return "https://api.anthropic.com/v1"
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	case "groq":
		return "https://api.groq.com/openai/v1"
	case "ollama":
		return "http://localhost:11434/v1"
	case "zhipu", "glm":
		return "https://open.bigmodel.cn/api/paas/v4"
	case "mistral":
		return "https://api.mistral.ai/v1"
	case "moonshot", "kimi":
		return "https://api.moonshot.cn/v1"
	case "nvidia":
		return "https://integrate.api.nvidia.com/v1"
	case "minimax":
		return "https://api.minimaxi.com/v1"
	case "litellm":
		return "http://localhost:4000/v1"
	}
	return ""
}

// WorkspacePath resolves the workspace path, expanding ~ to home directory.
func (c *Config) WorkspacePath() string {
	p := c.Workspace
	if p == "" {
		p = "~/.educlaw"
	}
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			p = filepath.Join(home, p[2:])
		}
	}
	return p
}

// StudentsDir returns the path for student workspaces.
func (c *Config) StudentsDir() string {
	return filepath.Join(c.WorkspacePath(), "students")
}

// FamiliesDir returns the path for family workspaces.
func (c *Config) FamiliesDir() string {
	return filepath.Join(c.WorkspacePath(), "families")
}

// TeachersDir returns the path for teacher workspaces.
func (c *Config) TeachersDir() string {
	return filepath.Join(c.WorkspacePath(), "teachers")
}

// DBPath returns the path for the SQLite database.
func (c *Config) DBPath() string {
	return filepath.Join(c.WorkspacePath(), "educlaw.db")
}

// AgentsDir returns the path for agent configuration files.
func (c *Config) AgentsDir() string {
	return filepath.Join(c.WorkspacePath(), "agents")
}

// Address returns the server address string.
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
