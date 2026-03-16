package tools

import (
	"context"
	"fmt"

	"github.com/pingjie/educlaw/pkg/llm"
)

// Tool defines the interface for agent tools.
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, args map[string]any) (string, error)
}

// Registry holds registered tools.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a new tool Registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// Execute runs a tool by name with the given arguments.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (string, error) {
	t, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}
	return t.Execute(ctx, args)
}

// AsLLMTools returns all registered tools in LLM format.
func (r *Registry) AsLLMTools() []llm.Tool {
	var tools []llm.Tool
	for _, t := range r.tools {
		tools = append(tools, llm.Tool{
			Type: "function",
			Function: llm.ToolFunc{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}
	return tools
}
