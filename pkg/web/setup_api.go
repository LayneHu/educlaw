package web

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pingjie/educlaw/pkg/config"
	"github.com/pingjie/educlaw/pkg/storage"
)

type modelGuide struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIBase  string `json:"api_base"`
	Note     string `json:"note"`
}

type setupStatus struct {
	NeedsSetup   bool           `json:"needs_setup"`
	IsConfigured bool           `json:"is_configured"`
	ConfigPath   string         `json:"config_path"`
	Workspace    string         `json:"workspace"`
	ModelName    string         `json:"model_name"`
	Provider     string         `json:"provider"`
	Model        string         `json:"model"`
	HasAPIKey    bool           `json:"has_api_key"`
	ActorCounts  map[string]int `json:"actor_counts"`
	Recommended  []modelGuide   `json:"recommended_models"`
}

type setupApplyRequest struct {
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	APIKey         string `json:"api_key"`
	APIBase        string `json:"api_base"`
	Proxy          string `json:"proxy"`
	TeacherName    string `json:"teacher_name"`
	TeacherSubject string `json:"teacher_subject"`
	SchoolName     string `json:"school_name"`
	TeacherGrade   string `json:"teacher_grade"`
}

func recommendedModels() []modelGuide {
	return []modelGuide{
		{ID: "minimax-default", Label: "MiniMax M2.5", Provider: "minimax", Model: "MiniMax-M2.5", APIBase: "https://api.minimaxi.com/v1", Note: "默认推荐，OpenAI 兼容，适合教学对话和内容生成。"},
		{ID: "minimax-fast", Label: "MiniMax M2.5 Highspeed", Provider: "minimax", Model: "MiniMax-M2.5-highspeed", APIBase: "https://api.minimaxi.com/v1", Note: "更偏速度，适合高频课堂问答。"},
		{ID: "gpt-4o-mini", Label: "OpenAI GPT-4o mini", Provider: "openai", Model: "gpt-4o-mini", APIBase: "https://api.openai.com/v1", Note: "成本低，响应快，适合日常陪学和报告草稿。"},
		{ID: "gpt-5", Label: "OpenAI GPT-5", Provider: "openai", Model: "gpt-5", APIBase: "https://api.openai.com/v1", Note: "推理更强，适合复杂备课和结构化生成。"},
		{ID: "deepseek-chat", Label: "DeepSeek Chat", Provider: "deepseek", Model: "deepseek-chat", APIBase: "https://api.deepseek.com/v1", Note: "中文表现稳，适合本地化教学场景。"},
		{ID: "gemini-flash", Label: "Gemini 2.0 Flash", Provider: "gemini", Model: "gemini-2.0-flash", APIBase: "https://generativelanguage.googleapis.com/v1beta/openai", Note: "多模态和速度都不错，适合轻量课堂助手。"},
		{ID: "ollama-qwen", Label: "Ollama Qwen", Provider: "ollama", Model: "qwen2.5:7b-instruct", APIBase: "http://localhost:11434/v1", Note: "本地模型，无需云 API key，但需要本机先装 Ollama。"},
	}
}

func (s *Server) currentProvider() config.LLMProviderConfig {
	if mc, name, err := s.cfg.ResolveModelSelection(); err == nil {
		return config.LLMProviderConfig{
			ModelName: name,
			Provider:  mc.Provider,
			APIKey:    mc.APIKey,
			Model:     mc.Model,
			APIBase:   mc.APIBase,
			Proxy:     mc.Proxy,
		}
	}
	return config.LLMProviderConfig{}
}

func (s *Server) actorCounts() map[string]int {
	counts := map[string]int{
		"student": 0,
		"family":  0,
		"teacher": 0,
	}
	for _, actorType := range []string{"student", "family", "teacher"} {
		items, err := storage.ListActors(s.db, actorType)
		if err == nil {
			counts[actorType] = len(items)
		}
	}
	return counts
}

func (s *Server) setupSnapshot() setupStatus {
	provider := s.currentProvider()
	counts := s.actorCounts()
	isConfigured := provider.Provider != "" && provider.Model != "" && (provider.APIKey != "" || strings.EqualFold(provider.Provider, "ollama"))
	return setupStatus{
		NeedsSetup:   !isConfigured || counts["teacher"] == 0,
		IsConfigured: isConfigured,
		ConfigPath:   s.configPath,
		Workspace:    s.cfg.WorkspacePath(),
		ModelName:    provider.ModelName,
		Provider:     provider.Provider,
		Model:        provider.Model,
		HasAPIKey:    provider.APIKey != "",
		ActorCounts:  counts,
		Recommended:  recommendedModels(),
	}
}

func (s *Server) needsSetup() bool {
	return s.setupSnapshot().NeedsSetup
}

// HandleSetupStatus returns first-run setup information for the frontend wizard.
func (s *Server) HandleSetupStatus(c *gin.Context) {
	c.JSON(http.StatusOK, s.setupSnapshot())
}

// HandleSetupApply persists model settings and optional teacher profile bootstrap.
func (s *Server) HandleSetupApply(c *gin.Context) {
	var req setupApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Provider) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is required"})
		return
	}
	if strings.TrimSpace(req.Model) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}
	if strings.TrimSpace(req.APIKey) == "" && !strings.EqualFold(strings.TrimSpace(req.Provider), "ollama") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_key is required"})
		return
	}

	cfg := config.Default()
	if _, err := os.Stat(s.configPath); err == nil {
		loaded, err := config.Load(s.configPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		cfg = loaded
	}
	savePath := s.configPath

	modelName := normalizeModelName(req.Provider, req.Model)
	cfg.LLM.Primary = config.LLMProviderConfig{Model: modelName}
	cfg.ModelList = upsertModelConfig(cfg.ModelList, config.ModelConfig{
		ModelName: modelName,
		Provider:  strings.TrimSpace(req.Provider),
		Model:     strings.TrimSpace(req.Model),
		APIKey:    strings.TrimSpace(req.APIKey),
		APIBase:   strings.TrimSpace(req.APIBase),
		Proxy:     strings.TrimSpace(req.Proxy),
	})
	if cfg.Workspace == "" {
		cfg.Workspace = "~/.educlaw"
	}

	if err := config.Save(savePath, cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	teacherID := ""
	teacherName := strings.TrimSpace(req.TeacherName)
	if teacherName != "" {
		existing, _ := storage.ListActors(s.db, "teacher")
		for _, actor := range existing {
			if actor.Name == teacherName && actor.Subject == strings.TrimSpace(req.TeacherSubject) {
				teacherID = actor.ID
				break
			}
		}
		if teacherID == "" {
			teacherID = uuid.New().String()
			if err := storage.SaveActor(s.db, teacherID, "teacher", teacherName, strings.TrimSpace(req.TeacherGrade), strings.TrimSpace(req.TeacherSubject), ""); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		dir := s.wm.TeacherDir(teacherID)
		_ = os.MkdirAll(dir, 0755)
		_ = s.wm.InitFromTemplate(dir, filepath.Join("workspace_templates", "teacher"))
		_ = s.wm.WriteFile(dir, "PROFILE.md", buildTeacherProfile(teacherName, strings.TrimSpace(req.TeacherSubject), strings.TrimSpace(req.SchoolName), strings.TrimSpace(req.TeacherGrade)))
	}

	s.cfg = cfg
	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"config_path": savePath,
		"teacher_id":  teacherID,
		"status":      s.setupSnapshot(),
	})
}

func buildTeacherProfile(name, subject, school, grade string) string {
	return fmt.Sprintf(`# 教师档案

## 基本信息
- 姓名: %s
- 科目: %s
- 学校: %s
- 年级: %s

## 教学偏好
- 课堂风格: 启发式
- 备课重点: 围绕班级薄弱点设计练习与讲解
`, blankFallback(name, "(待填写)"), blankFallback(subject, "(待填写)"), blankFallback(school, "(待填写)"), blankFallback(grade, "(待填写)"))
}

func normalizeModelName(provider, model string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	model = strings.TrimSpace(model)
	model = strings.NewReplacer(".", "-", ":", "-", "/", "-").Replace(model)
	model = strings.ToLower(model)
	if provider == "" {
		return model
	}
	return provider + "-" + model
}

func upsertModelConfig(items []config.ModelConfig, incoming config.ModelConfig) []config.ModelConfig {
	for i := range items {
		if items[i].ModelName == incoming.ModelName {
			items[i] = incoming
			return items
		}
	}
	return append(items, incoming)
}

func blankFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
