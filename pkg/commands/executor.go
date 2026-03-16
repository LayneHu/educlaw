package commands

import (
	"context"
	"strings"
)

// ExecuteResult captures the outcome of command dispatch.
type ExecuteResult struct {
	Handled bool
	Command string
	Err     error
}

// Executor dispatches slash commands.
type Executor struct {
	reg *Registry
	rt  *Runtime
}

func NewExecutor(reg *Registry, rt *Runtime) *Executor {
	return &Executor{reg: reg, rt: rt}
}

func (e *Executor) Execute(ctx context.Context, req Request) ExecuteResult {
	if !strings.HasPrefix(strings.TrimSpace(req.Text), "/") {
		return ExecuteResult{}
	}
	fields := strings.Fields(strings.TrimSpace(req.Text))
	if len(fields) == 0 {
		return ExecuteResult{}
	}
	name := normalize(fields[0])
	def, ok := e.reg.Lookup(name)
	if !ok {
		return ExecuteResult{}
	}
	if req.Reply == nil {
		req.Reply = func(string) error { return nil }
	}
	err := def.Handler(ctx, req, e.rt)
	return ExecuteResult{Handled: true, Command: def.Name, Err: err}
}
