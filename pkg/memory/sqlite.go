package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/pingjie/educlaw/pkg/llm"
)

// SQLiteStore persists sessions in the main educlaw SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a session store backed by SQLite.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

// GetOrCreateSession returns the latest session for an actor or creates one.
func (s *SQLiteStore) GetOrCreateSession(ctx context.Context, actorID, actorType string) (string, []llm.Message, error) {
	var sessionID string
	var messagesJSON string

	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, messages FROM sessions WHERE actor_id = ? ORDER BY updated_at DESC LIMIT 1`,
		actorID,
	).Scan(&sessionID, &messagesJSON)

	if err == sql.ErrNoRows {
		sessionID = uuid.New().String()
		if _, err = s.db.ExecContext(
			ctx,
			`INSERT INTO sessions (id, actor_id, actor_type, messages, summary) VALUES (?, ?, ?, '[]', '')`,
			sessionID, actorID, actorType,
		); err != nil {
			return "", nil, fmt.Errorf("creating session: %w", err)
		}
		return sessionID, []llm.Message{}, nil
	}
	if err != nil {
		return "", nil, fmt.Errorf("querying session: %w", err)
	}

	history, err := decodeMessages(messagesJSON)
	if err != nil {
		return "", nil, nil
	}
	return sessionID, history, nil
}

// GetOrCreateSessionByID returns a specific session or creates it.
func (s *SQLiteStore) GetOrCreateSessionByID(ctx context.Context, sessionID, actorID, actorType string) ([]llm.Message, error) {
	var messagesJSON string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT messages FROM sessions WHERE id = ?`,
		sessionID,
	).Scan(&messagesJSON)

	if err == sql.ErrNoRows {
		if _, err = s.db.ExecContext(
			ctx,
			`INSERT INTO sessions (id, actor_id, actor_type, messages, summary) VALUES (?, ?, ?, '[]', '')`,
			sessionID, actorID, actorType,
		); err != nil {
			return nil, fmt.Errorf("creating session: %w", err)
		}
		return []llm.Message{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying session: %w", err)
	}

	history, err := decodeMessages(messagesJSON)
	if err != nil {
		return nil, nil
	}
	return history, nil
}

// GetRecentlyActiveSessions returns sessions updated within the last N hours.
func (s *SQLiteStore) GetRecentlyActiveSessions(ctx context.Context, hours int) ([]ActiveSession, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, actor_id, actor_type, updated_at FROM sessions
		 WHERE updated_at >= datetime('now', ? || ' hours')
		 ORDER BY updated_at DESC`,
		fmt.Sprintf("-%d", hours),
	)
	if err != nil {
		return nil, fmt.Errorf("querying recent sessions: %w", err)
	}
	defer rows.Close()

	var sessions []ActiveSession
	for rows.Next() {
		var item ActiveSession
		if err := rows.Scan(&item.ID, &item.ActorID, &item.ActorType, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning session: %w", err)
		}
		sessions = append(sessions, item)
	}
	return sessions, rows.Err()
}

func (s *SQLiteStore) AddMessage(ctx context.Context, sessionID, role, content string) error {
	return s.AddFullMessage(ctx, sessionID, llm.Message{Role: role, Content: content})
}

func (s *SQLiteStore) AddFullMessage(ctx context.Context, sessionID string, msg llm.Message) error {
	history, err := s.GetHistory(ctx, sessionID)
	if err != nil {
		return err
	}
	history = append(history, msg)
	return s.SetHistory(ctx, sessionID, history)
}

func (s *SQLiteStore) GetHistory(ctx context.Context, sessionID string) ([]llm.Message, error) {
	var messagesJSON string
	err := s.db.QueryRowContext(ctx, `SELECT messages FROM sessions WHERE id = ?`, sessionID).Scan(&messagesJSON)
	if err == sql.ErrNoRows {
		return []llm.Message{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying history: %w", err)
	}
	history, err := decodeMessages(messagesJSON)
	if err != nil {
		return []llm.Message{}, nil
	}
	return history, nil
}

func (s *SQLiteStore) GetSummary(ctx context.Context, sessionID string) (string, error) {
	var summary string
	err := s.db.QueryRowContext(ctx, `SELECT summary FROM sessions WHERE id = ?`, sessionID).Scan(&summary)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("querying summary: %w", err)
	}
	return summary, nil
}

func (s *SQLiteStore) SetSummary(ctx context.Context, sessionID, summary string) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE sessions SET summary = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		summary, sessionID,
	)
	if err != nil {
		return fmt.Errorf("saving summary: %w", err)
	}
	return nil
}

func (s *SQLiteStore) TruncateHistory(ctx context.Context, sessionID string, keepLast int) error {
	history, err := s.GetHistory(ctx, sessionID)
	if err != nil {
		return err
	}
	switch {
	case keepLast <= 0:
		history = []llm.Message{}
	case len(history) > keepLast:
		history = history[len(history)-keepLast:]
	}
	return s.SetHistory(ctx, sessionID, history)
}

func (s *SQLiteStore) SetHistory(ctx context.Context, sessionID string, history []llm.Message) error {
	data, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("marshaling messages: %w", err)
	}
	_, err = s.db.ExecContext(
		ctx,
		`UPDATE sessions SET messages = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		string(data), sessionID,
	)
	if err != nil {
		return fmt.Errorf("saving session: %w", err)
	}
	return nil
}

func (s *SQLiteStore) Compact(context.Context, string) error {
	return nil
}

func (s *SQLiteStore) Close() error {
	return nil
}

func decodeMessages(messagesJSON string) ([]llm.Message, error) {
	if messagesJSON == "" || messagesJSON == "[]" {
		return []llm.Message{}, nil
	}
	var messages []llm.Message
	if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
		return nil, err
	}
	return messages, nil
}
