package web

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/pingjie/educlaw/pkg/agents"
	"github.com/pingjie/educlaw/pkg/bus"
	"github.com/pingjie/educlaw/pkg/commands"
	"github.com/pingjie/educlaw/pkg/config"
	"github.com/pingjie/educlaw/pkg/health"
	"github.com/pingjie/educlaw/pkg/llm"
	"github.com/pingjie/educlaw/pkg/memory"
	"github.com/pingjie/educlaw/pkg/skills"
	"github.com/pingjie/educlaw/pkg/workspace"
)

//go:embed static
var staticFiles embed.FS

// Server holds the web server state.
type Server struct {
	cfg        *config.Config
	configPath string
	db         *sql.DB
	wm         *workspace.Manager
	msgBus     *bus.MessageBus
	agentLoop  *agents.AgentLoop
	health     *health.Manager
	llm        llm.Provider
	skills     *skills.Loader
	sessions   *memory.SQLiteStore
	commands   *commands.Executor
	router     *gin.Engine
}

// NewServer creates and configures the HTTP server.
// agentLoop is the shared loop used by all handlers (also shared with heartbeat/cron).
// llmClient is kept in the signature for future direct-LLM handler use.
func NewServer(
	cfg *config.Config,
	configPath string,
	db *sql.DB,
	wm *workspace.Manager,
	msgBus *bus.MessageBus,
	healthMgr *health.Manager,
	llmClient llm.Provider,
	skillsLoader *skills.Loader,
	agentLoop *agents.AgentLoop,
) *Server {
	s := &Server{
		cfg:        cfg,
		configPath: configPath,
		db:         db,
		wm:         wm,
		msgBus:     msgBus,
		agentLoop:  agentLoop,
		health:     healthMgr,
		llm:        llmClient,
		skills:     skillsLoader,
		sessions:   memory.NewSQLiteStore(db),
		router:     gin.Default(),
	}
	s.commands = commands.NewExecutor(commands.NewRegistry(commands.Builtins()), s.commandRuntime())

	s.setupRoutes()
	return s
}

func (s *Server) commandRuntime() *commands.Runtime {
	return &commands.Runtime{
		ListDefinitions: func() []commands.Definition {
			return commands.Builtins()
		},
		GetModelInfo: func() (string, string) {
			if mc, name, err := s.cfg.ResolveModelSelection(); err == nil {
				return mc.Model, mc.Provider + " (" + name + ")"
			}
			if s.llm != nil {
				return s.llm.ModelName(), "unknown"
			}
			return "", "unknown"
		},
		ListSkills: func() []string {
			if s.skills == nil {
				return nil
			}
			items := s.skills.ListSkills()
			names := make([]string, 0, len(items))
			for _, item := range items {
				names = append(names, item.Name)
			}
			sort.Strings(names)
			return names
		},
		ShowSkill: func(name string) (string, bool) {
			if s.skills == nil {
				return "", false
			}
			return s.skills.LoadSkill(name)
		},
		ClearHistory: func(sessionID string) error {
			return s.sessions.SetHistory(context.Background(), sessionID, []llm.Message{})
		},
	}
}

func (s *Server) setupRoutes() {
	// Static files
	staticFS, _ := fs.Sub(staticFiles, "static")
	s.router.StaticFS("/static", http.FS(staticFS))

	// Page routes
	s.router.GET("/", func(c *gin.Context) {
		if s.needsSetup() {
			c.Redirect(http.StatusFound, "/setup")
			return
		}
		c.Redirect(http.StatusFound, "/student")
	})

	s.router.GET("/setup", func(c *gin.Context) {
		data, _ := staticFiles.ReadFile("static/setup.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	s.router.GET("/student", func(c *gin.Context) {
		data, _ := staticFiles.ReadFile("static/student.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	s.router.GET("/parent", func(c *gin.Context) {
		data, _ := staticFiles.ReadFile("static/parent.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	s.router.GET("/teacher", func(c *gin.Context) {
		data, _ := staticFiles.ReadFile("static/teacher.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	// API routes
	api := s.router.Group("/api")
	{
		// Chat
		api.POST("/chat", s.HandleChat)
		api.GET("/chat/stream/:session_id", s.HandleStream)

		// Student
		api.GET("/student/:id/summary", s.HandleStudentSummary)

		// Parent
		api.GET("/parent/:id/report", s.HandleParentReport)

		// Teacher
		api.GET("/teacher/:id/class-report", s.HandleClassReport)

		// Onboard
		api.POST("/onboard", s.HandleOnboard)
		api.GET("/setup/status", s.HandleSetupStatus)
		api.POST("/setup/apply", s.HandleSetupApply)

		// Actors
		api.GET("/actors/:type", s.HandleListActors)
	}

	if s.cfg.Health.Enabled && s.health != nil {
		s.router.GET("/health", s.HandleHealth)
		s.router.GET("/ready", s.HandleReady)
	}
}

// Run starts the HTTP server.
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
