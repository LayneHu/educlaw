package serve

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	cmdinternal "github.com/pingjie/educlaw/cmd/educlaw/internal"
	"github.com/pingjie/educlaw/pkg/agents"
	"github.com/pingjie/educlaw/pkg/bus"
	"github.com/pingjie/educlaw/pkg/config"
	"github.com/pingjie/educlaw/pkg/cron"
	"github.com/pingjie/educlaw/pkg/health"
	"github.com/pingjie/educlaw/pkg/heartbeat"
	"github.com/pingjie/educlaw/pkg/llm"
	"github.com/pingjie/educlaw/pkg/skills"
	"github.com/pingjie/educlaw/pkg/storage"
	"github.com/pingjie/educlaw/pkg/web"
	"github.com/pingjie/educlaw/pkg/workspace"
)

// SetupServer initializes all server components and returns the web server.
func SetupServer(configPath string) (*web.Server, string, error) {
	cfg, resolvedConfigPath, err := cmdinternal.LoadConfigWithPath(configPath)
	if err != nil {
		return nil, "", fmt.Errorf("loading config: %w", err)
	}
	log.Printf("Config: %s", resolvedConfigPath)

	// Ensure workspace directories exist
	wksp := cfg.WorkspacePath()
	for _, dir := range []string{
		wksp,
		cfg.StudentsDir(),
		cfg.FamiliesDir(),
		cfg.TeachersDir(),
		cfg.AgentsDir(),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, "", fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	// Initialize database
	db, err := storage.InitDB(cfg.DBPath())
	if err != nil {
		return nil, "", fmt.Errorf("initializing database: %w", err)
	}
	log.Printf("Database: %s", cfg.DBPath())

	// Initialize workspace manager
	wm := workspace.NewManager(wksp)

	// Copy built-in agent files to workspace agents dir if not present
	agentsDir := cfg.AgentsDir()
	if err := initAgentFiles(agentsDir); err != nil {
		log.Printf("Warning: could not init agent files: %v", err)
	}

	// Initialize message bus
	msgBus := bus.NewMessageBus()

	// Build LLM provider (multi-provider with fallback if configured)
	llmProvider := buildLLMProvider(cfg)
	if mc, name, err := cfg.ResolveModelSelection(); err == nil {
		log.Printf("LLM selection: model_name=%s provider=%s model=%s api_base=%s has_key=%t",
			name, mc.Provider, mc.Model, mc.APIBase, mc.APIKey != "")
	} else {
		log.Printf("LLM selection: unresolved (%v)", err)
	}

	healthMgr := health.NewManager()
	healthMgr.RegisterCheck("database", func() (bool, string) {
		if err := db.Ping(); err != nil {
			return false, err.Error()
		}
		return true, "connected"
	})
	healthMgr.RegisterCheck("llm", func() (bool, string) {
		if mc, _, err := cfg.ResolveModelSelection(); err == nil {
			if mc.APIKey != "" || strings.EqualFold(mc.Provider, "ollama") {
				return true, mc.Model
			}
		}
		return false, "missing model_list api key"
	})

	// Auto-discover builtin skills dir if not configured
	if cfg.Skills.BuiltinDir == "" {
		candidates := []string{"./skills"}
		if exe, err := os.Executable(); err == nil {
			candidates = append(candidates, filepath.Join(filepath.Dir(exe), "skills"))
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				cfg.Skills.BuiltinDir = c
				log.Printf("Auto-discovered skills dir: %s", c)
				break
			}
		}
	}

	healthMgr.RegisterCheck("skills", func() (bool, string) {
		if cfg.Skills.BuiltinDir == "" {
			return false, "builtin skills dir not configured"
		}
		if _, err := os.Stat(cfg.Skills.BuiltinDir); err != nil {
			return false, err.Error()
		}
		return true, cfg.Skills.BuiltinDir
	})

	// Initialize skills loader
	globalSkillsDir := ""
	if home, err := os.UserHomeDir(); err == nil {
		globalSkillsDir = filepath.Join(home, ".educlaw", "skills")
		if globalSkillsDir == filepath.Join(wksp, "skills") {
			globalSkillsDir = ""
		}
	}
	skillsLoader := skills.NewLoader(filepath.Join(wksp, "skills"), globalSkillsDir, cfg.Skills.BuiltinDir)

	// Create shared agent loop
	agentLoop := agents.NewAgentLoop(cfg, llmProvider, db, wm, msgBus, skillsLoader)

	// Initialize and wire cron service if enabled
	if cfg.Cron.Enabled {
		cronStorePath := filepath.Join(wksp, "cron", "jobs.json")
		cronSvc := cron.NewService(cronStorePath)
		cronSvc.SetHandler(func(job *cron.Job) error {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			msg := bus.InboundMessage{
				ActorID:   job.Payload.ActorID,
				ActorType: job.Payload.ActorType,
				Content:   job.Payload.Message,
			}
			return agentLoop.Process(ctx, msg)
		})
		agentLoop.SetCronService(cronSvc)
		if err := cronSvc.Start(); err != nil {
			log.Printf("Warning: cron service failed to start: %v", err)
		} else {
			log.Printf("Cron service started")
		}
	}

	// Create web server
	srv := web.NewServer(cfg, resolvedConfigPath, db, wm, msgBus, healthMgr, llmProvider, skillsLoader, agentLoop)

	// Start heartbeat service if enabled
	if cfg.Heartbeat.Enabled {
		hb := heartbeat.NewService(db, msgBus, cfg.Heartbeat)
		hb.SetHandler(func(ctx context.Context, sessionID, actorID, actorType string) error {
			prompt := heartbeat.BuildHeartbeatPrompt(actorType)
			msg := bus.InboundMessage{
				ActorID:   actorID,
				ActorType: actorType,
				SessionID: sessionID,
				Content:   prompt,
			}
			return agentLoop.Process(ctx, msg)
		})
		hb.Start()
		log.Printf("Heartbeat service started (interval: %d min)", cfg.Heartbeat.IntervalMinutes)
	}

	healthMgr.SetReady(true)

	return srv, cfg.Address(), nil
}

// buildLLMProvider constructs a Provider (single or multi-fallback) from model_list config.
func buildLLMProvider(cfg *config.Config) llm.Provider {
	var providers []llm.Provider

	if len(cfg.ModelList) > 0 {
		primaryCfg, modelName, err := cfg.ResolveModelSelection()
		if err == nil {
			if primaryCfg.APIKey != "" || strings.EqualFold(primaryCfg.Provider, "ollama") {
				providers = append(providers, llm.NewClient(primaryCfg))
			}
			fallbacks, fbErr := cfg.ResolveFallbackSelections()
			if fbErr != nil {
				log.Printf("Warning: resolving fallback models failed: %v", fbErr)
			} else {
				for _, fb := range fallbacks {
					if fb.APIKey != "" || strings.EqualFold(fb.Provider, "ollama") {
						modelCfg := fb
						providers = append(providers, llm.NewClient(&modelCfg))
					}
				}
			}
			log.Printf("LLM: primary=%s -> %s, fallbacks=%d", modelName, primaryCfg.Model, max(len(providers)-1, 0))
		} else {
			log.Printf("Warning: resolving model_list failed: %v", err)
		}
	}

	if len(providers) == 0 {
		log.Printf("Warning: no LLM API key configured — AI responses will fail")
		return llm.NewDisabledProvider("missing model_list api key")
	}

	if len(providers) == 1 {
		return providers[0]
	}
	return llm.NewMultiProvider(providers...)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func initAgentFiles(agentsDir string) error {
	agentsMDPath := agentsDir + "/AGENTS.md"
	if _, err := os.Stat(agentsMDPath); os.IsNotExist(err) {
		content := defaultAgentsMD()
		return os.WriteFile(agentsMDPath, []byte(content), 0644)
	}
	return nil
}

func defaultAgentsMD() string {
	return `# EduClaw AI教学原则

## 核心身份
你是"知知"，一位充满智慧和耐心的AI学习伙伴。

## 教学哲学
1. **苏格拉底式引导**: 不直接给出答案，通过提问引导学生自己发现
2. **三级脚手架**: 提示 → 引导 → 直接讲解
3. **积极强化**: 及时表扬进步，温柔纠正错误
4. **个性化**: 结合学生兴趣和生活经验解释概念

## 工具使用规则
- 每次对话后用 add_daily_note 记录学习内容
- 用 record_answer 记录学生答题情况
- 发现学生困难时考虑使用 game-generator 或 visual-explainer
- 重要发现及时更新 KNOWLEDGE.md 和 ERRORS.md

## 语言风格
- 使用鼓励性语言
- 避免专业术语，用学生能理解的语言
- 保持轻松愉快的氛围
`
}
