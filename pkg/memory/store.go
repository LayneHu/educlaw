package memory

import (
	"context"
	"time"

	"github.com/pingjie/educlaw/pkg/llm"
)

// Store defines the persistent conversation storage contract.
type Store interface {
	AddMessage(ctx context.Context, sessionID, role, content string) error
	AddFullMessage(ctx context.Context, sessionID string, msg llm.Message) error
	GetHistory(ctx context.Context, sessionID string) ([]llm.Message, error)
	GetSummary(ctx context.Context, sessionID string) (string, error)
	SetSummary(ctx context.Context, sessionID, summary string) error
	TruncateHistory(ctx context.Context, sessionID string, keepLast int) error
	SetHistory(ctx context.Context, sessionID string, history []llm.Message) error
	Compact(ctx context.Context, sessionID string) error
	Close() error
}

// ActiveSession holds lightweight metadata for recently active sessions.
type ActiveSession struct {
	ID        string
	ActorID   string
	ActorType string
	UpdatedAt time.Time
}
