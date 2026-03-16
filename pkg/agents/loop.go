package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/pingjie/educlaw/pkg/agents/orchestrator"
	"github.com/pingjie/educlaw/pkg/bus"
	"github.com/pingjie/educlaw/pkg/config"
	"github.com/pingjie/educlaw/pkg/cron"
	"github.com/pingjie/educlaw/pkg/llm"
	"github.com/pingjie/educlaw/pkg/memory"
	"github.com/pingjie/educlaw/pkg/skills"
	"github.com/pingjie/educlaw/pkg/tools"
	"github.com/pingjie/educlaw/pkg/workspace"
)

// AgentLoop implements the core ReAct loop for processing messages.
type AgentLoop struct {
	cfg          *config.Config
	llm          llm.Provider
	db           *sql.DB
	wm           *workspace.Manager
	msgBus       *bus.MessageBus
	sessionStore *memory.SQLiteStore
	skillsLoader *skills.Loader
	skillsReg    *skills.RegistryManager
	skillsInst   *skills.Installer
	cronSvc      *cron.Service // optional; enables scheduling tools
}

// NewAgentLoop creates a new AgentLoop.
func NewAgentLoop(
	cfg *config.Config,
	llmClient llm.Provider,
	db *sql.DB,
	wm *workspace.Manager,
	msgBus *bus.MessageBus,
	skillsLoader *skills.Loader,
) *AgentLoop {
	return &AgentLoop{
		cfg:          cfg,
		llm:          llmClient,
		db:           db,
		wm:           wm,
		msgBus:       msgBus,
		sessionStore: memory.NewSQLiteStore(db),
		skillsLoader: skillsLoader,
		skillsReg:    skills.NewRegistryManager(skillsLoader),
		skillsInst:   skills.NewInstaller(skillsLoader.WorkspaceDir()),
	}
}

// SetCronService attaches a cron service so agents can schedule reminders.
func (al *AgentLoop) SetCronService(svc *cron.Service) {
	al.cronSvc = svc
}

// Process handles an inbound message through the full ReAct loop.
func (al *AgentLoop) Process(ctx context.Context, msg bus.InboundMessage) error {
	// 1. Get or create session
	sessionID := msg.SessionID
	var messages []llm.Message
	var err error

	if sessionID == "" {
		sessionID, messages, err = al.sessionStore.GetOrCreateSession(ctx, msg.ActorID, msg.ActorType)
	} else {
		messages, err = al.sessionStore.GetOrCreateSessionByID(ctx, sessionID, msg.ActorID, msg.ActorType)
	}
	if err != nil {
		return fmt.Errorf("getting session: %w", err)
	}

	// Sanitize and truncate history to prevent 400 errors and context overflow
	messages = sanitizeHistory(messages)
	messages = truncateHistory(messages, 20)

	// 2. Route to appropriate agent and build system prompt
	agentType := orchestrator.Route(msg.ActorType, msg.Content)
	actorDir := al.actorDirForType(msg.ActorID, msg.ActorType)
	agentDir := al.wm.AgentDir()
	cb := NewContextBuilder(al.wm, actorDir, agentDir, al.cfg.Skills.BuiltinDir)
	systemPrompt := cb.Build(msg.ActorType, agentType)
	log.Printf("[agent] actor=%s type=%s → agent=%s", msg.ActorID, msg.ActorType, agentType)

	// 3. Build tool registry
	registry := al.buildRegistry(sessionID, msg.ActorID, msg.ActorType, actorDir)

	// 4. Append user message
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: msg.Content,
	})

	// 5. ReAct loop
	maxIter := al.cfg.Agent.MaxIterations
	if maxIter == 0 {
		maxIter = 15
	}

	allMessages := append([]llm.Message{{Role: "system", Content: systemPrompt}}, messages...)

	for i := 0; i < maxIter; i++ {
		req := llm.CompletionRequest{
			Messages:    allMessages,
			Tools:       registry.AsLLMTools(),
			Temperature: al.cfg.Agent.Temperature,
			MaxTokens:   al.cfg.Agent.MaxTokens,
		}

		// Stream tokens to message bus
		resp, err := al.llm.StreamComplete(ctx, req, func(token string) {
			al.msgBus.Publish(bus.OutboundMessage{
				SessionID:   sessionID,
				ActorID:     msg.ActorID,
				Content:     token,
				ContentType: "text",
				Done:        false,
			})
		})
		if err != nil {
			return fmt.Errorf("LLM call iteration %d: %w", i, err)
		}

		// Append assistant message
		assistantMsg := llm.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		allMessages = append(allMessages, assistantMsg)

		// If no tool calls, we're done
		if len(resp.ToolCalls) == 0 {
			break
		}

		// Execute tool calls
		for _, tc := range resp.ToolCalls {
			toolResult := al.executeToolCall(ctx, registry, tc, sessionID, msg.ActorID)

			// Append tool result
			allMessages = append(allMessages, llm.Message{
				Role:       "tool",
				Content:    toolResult,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			})
		}
	}

	// 6. Save session (without system prompt)
	userAndAssistantMessages := allMessages[1:] // skip system prompt
	_ = al.sessionStore.SetHistory(ctx, sessionID, userAndAssistantMessages)

	// 7. Publish done message
	al.msgBus.Publish(bus.OutboundMessage{
		SessionID:   sessionID,
		ActorID:     msg.ActorID,
		Content:     "",
		ContentType: "text",
		Done:        true,
	})

	return nil
}

func (al *AgentLoop) actorDirForType(actorID, actorType string) string {
	switch actorType {
	case "student":
		return al.wm.StudentDir(actorID)
	case "family", "parent":
		return al.wm.FamilyDir(actorID)
	case "teacher":
		return al.wm.TeacherDir(actorID)
	default:
		return al.wm.StudentDir(actorID)
	}
}

func (al *AgentLoop) buildRegistry(sessionID, actorID, actorType, actorDir string) *tools.Registry {
	registry := tools.NewRegistry()
	registry.Register(tools.NewListWorkspaceFilesTool(al.wm, actorDir))
	registry.Register(tools.NewReadWorkspaceTool(al.wm, actorDir))
	registry.Register(tools.NewWriteWorkspaceTool(al.wm, actorDir))
	registry.Register(tools.NewAppendDailyTool(al.wm, actorDir))
	registry.Register(tools.NewRecordEventTool(al.db, actorID))
	registry.Register(tools.NewQueryKnowledgeTool(al.db, actorID))
	registry.Register(tools.NewRenderContentTool(al.db, al.msgBus, sessionID, actorID))
	registry.Register(tools.NewReadSkillTool(al.skillsLoader))
	registry.Register(tools.NewFindSkillsTool(al.skillsReg))
	if al.skillsLoader.WorkspaceDir() != "" {
		registry.Register(tools.NewInstallSkillTool(al.skillsInst))
	}
	registry.Register(tools.NewWebFetchTool())
	registry.Register(tools.NewWebSearchTool())
	if al.cronSvc != nil {
		registry.Register(tools.NewScheduleReminderTool(al.cronSvc, actorID, actorType))
		registry.Register(tools.NewListRemindersTool(al.cronSvc, actorID))
		registry.Register(tools.NewCancelReminderTool(al.cronSvc))
	}
	return registry
}

func (al *AgentLoop) executeToolCall(ctx context.Context, registry *tools.Registry, tc llm.ToolCall, sessionID, actorID string) string {
	var args map[string]any
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		log.Printf("failed to parse tool args for %s: %v", tc.Function.Name, err)
		return fmt.Sprintf("Error: failed to parse arguments: %v", err)
	}

	// Publish tool-call event
	al.publishToolEvent(sessionID, actorID, bus.ToolEvent{
		Phase:   "call",
		Tool:    tc.Function.Name,
		Summary: toolArgsSummary(args),
	})

	result, err := registry.Execute(ctx, tc.Function.Name, args)
	if err != nil {
		log.Printf("tool %s error: %v", tc.Function.Name, err)
		al.publishToolEvent(sessionID, actorID, bus.ToolEvent{
			Phase:   "result",
			Tool:    tc.Function.Name,
			Summary: fmt.Sprintf("错误: %v", err),
		})
		return fmt.Sprintf("Error executing %s: %v", tc.Function.Name, err)
	}

	// Publish result summary (truncate long results)
	summary := result
	if len(summary) > 120 {
		summary = summary[:120] + "..."
	}
	al.publishToolEvent(sessionID, actorID, bus.ToolEvent{
		Phase:   "result",
		Tool:    tc.Function.Name,
		Summary: summary,
	})

	return result
}

func (al *AgentLoop) publishToolEvent(sessionID, actorID string, ev bus.ToolEvent) {
	data, _ := json.Marshal(ev)
	al.msgBus.Publish(bus.OutboundMessage{
		SessionID:   sessionID,
		ActorID:     actorID,
		Content:     string(data),
		ContentType: "tool_call",
		Done:        false,
	})
}

// sanitizeHistory removes invalid message sequences that would cause LLM 400 errors:
// - orphaned tool messages (no preceding assistant tool_call)
// - assistant tool_call messages without complete matching tool results
func sanitizeHistory(messages []llm.Message) []llm.Message {
	if len(messages) == 0 {
		return messages
	}

	result := make([]llm.Message, 0, len(messages))
	// seenCallIDs tracks tool_call IDs from assistant messages already added
	seenCallIDs := make(map[string]bool)

	for i, m := range messages {
		switch m.Role {
		case "tool":
			// Drop tool results with no matching assistant tool_call
			if m.ToolCallID == "" || !seenCallIDs[m.ToolCallID] {
				continue
			}
		case "assistant":
			if len(m.ToolCalls) > 0 {
				// Verify all expected tool results follow this message
				expectedIDs := make(map[string]bool)
				for _, tc := range m.ToolCalls {
					if tc.ID != "" {
						expectedIDs[tc.ID] = true
					}
				}
				foundIDs := make(map[string]bool)
				for j := i + 1; j < len(messages); j++ {
					next := messages[j]
					if next.Role == "tool" && expectedIDs[next.ToolCallID] {
						foundIDs[next.ToolCallID] = true
					} else if next.Role != "tool" {
						break
					}
				}
				if len(foundIDs) < len(expectedIDs) {
					// Incomplete tool results — skip this assistant turn entirely.
					// The orphaned tool messages will be filtered out above since
					// their IDs won't be in seenCallIDs.
					continue
				}
				for id := range expectedIDs {
					seenCallIDs[id] = true
				}
			}
		}
		result = append(result, m)
	}
	return result
}

// truncateHistory keeps the last maxPairs user/assistant exchange pairs to
// prevent context window overflow on long sessions.
func truncateHistory(messages []llm.Message, maxPairs int) []llm.Message {
	if maxPairs <= 0 || len(messages) == 0 {
		return messages
	}
	var userIndices []int
	for i, m := range messages {
		if m.Role == "user" {
			userIndices = append(userIndices, i)
		}
	}
	if len(userIndices) <= maxPairs {
		return messages
	}
	cutIdx := userIndices[len(userIndices)-maxPairs]
	return messages[cutIdx:]
}

// toolArgsSummary builds a short human-readable summary of tool arguments.
func toolArgsSummary(args map[string]any) string {
	if len(args) == 0 {
		return ""
	}
	parts := make([]string, 0, len(args))
	for k, v := range args {
		val := fmt.Sprintf("%v", v)
		if len(val) > 60 {
			val = val[:60] + "..."
		}
		parts = append(parts, k+"="+val)
	}
	return strings.Join(parts, ", ")
}
