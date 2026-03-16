package commands

import "context"

// ReplyFunc sends command output back to the caller.
type ReplyFunc func(string) error

// Request is the execution input for a command.
type Request struct {
	Text      string
	SessionID string
	ActorID   string
	ActorType string
	Reply     ReplyFunc
}

// Handler executes a command.
type Handler func(context.Context, Request, *Runtime) error
