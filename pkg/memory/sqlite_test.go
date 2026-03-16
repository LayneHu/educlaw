package memory

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/pingjie/educlaw/pkg/llm"
	"github.com/pingjie/educlaw/pkg/storage"
)

func TestSQLiteStoreHistoryAndSummary(t *testing.T) {
	db, err := storage.InitDB(filepath.Join(t.TempDir(), "educlaw.db"))
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store := NewSQLiteStore(db)
	ctx := context.Background()

	sessionID, history, err := store.GetOrCreateSession(ctx, "student-1", "student")
	if err != nil {
		t.Fatalf("GetOrCreateSession() error = %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("initial history len = %d, want 0", len(history))
	}

	if err := store.AddFullMessage(ctx, sessionID, llm.Message{Role: "user", Content: "hello"}); err != nil {
		t.Fatalf("AddFullMessage() error = %v", err)
	}
	if err := store.AddMessage(ctx, sessionID, "assistant", "world"); err != nil {
		t.Fatalf("AddMessage() error = %v", err)
	}

	history, err = store.GetHistory(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetHistory() error = %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("history len = %d, want 2", len(history))
	}

	if err := store.SetSummary(ctx, sessionID, "summary"); err != nil {
		t.Fatalf("SetSummary() error = %v", err)
	}
	summary, err := store.GetSummary(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetSummary() error = %v", err)
	}
	if summary != "summary" {
		t.Fatalf("summary = %q, want %q", summary, "summary")
	}

	if err := store.TruncateHistory(ctx, sessionID, 1); err != nil {
		t.Fatalf("TruncateHistory() error = %v", err)
	}
	history, err = store.GetHistory(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetHistory() after truncate error = %v", err)
	}
	if len(history) != 1 || history[0].Content != "world" {
		t.Fatalf("truncated history = %#v, want last message only", history)
	}
}
