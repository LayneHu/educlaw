package agents

// AgentType constants
const (
	AgentTypeOrchestrator = "orchestrator"
	AgentTypeTutor        = "tutor"
	AgentTypePlanner      = "planner"
	AgentTypeCompanion    = "companion"
	AgentTypeAnalyst      = "analyst"
	AgentTypeParent       = "parent"
	AgentTypeTeacher      = "teacher"
)

// AgentConfig holds configuration for an agent.
type AgentConfig struct {
	Name             string
	Type             string
	WorkspaceDir     string
	SystemPromptFile string
}
