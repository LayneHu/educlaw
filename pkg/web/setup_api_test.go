package web

import (
	"path/filepath"
	"testing"

	"github.com/pingjie/educlaw/pkg/config"
	"github.com/pingjie/educlaw/pkg/storage"
	"github.com/pingjie/educlaw/pkg/workspace"
)

func TestSetupSnapshotTracksConfigAndTeacher(t *testing.T) {
	dir := t.TempDir()
	db, err := storage.InitDB(filepath.Join(dir, "educlaw.db"))
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	defer db.Close()

	cfg := config.Default()
	cfg.Workspace = dir
	srv := &Server{
		cfg:        cfg,
		configPath: filepath.Join(dir, "config.json"),
		db:         db,
		wm:         workspace.NewManager(dir),
	}

	initial := srv.setupSnapshot()
	if !initial.NeedsSetup {
		t.Fatalf("NeedsSetup = false, want true for empty config")
	}
	if len(initial.Recommended) == 0 {
		t.Fatalf("Recommended models is empty")
	}

	cfg.ModelList[0].APIKey = "sk-test"
	if err := storage.SaveActor(db, "teacher-1", "teacher", "张老师", "五年级", "数学", "", ""); err != nil {
		t.Fatalf("SaveActor() error = %v", err)
	}

	ready := srv.setupSnapshot()
	if ready.NeedsSetup {
		t.Fatalf("NeedsSetup = true, want false after model + teacher setup")
	}
	if ready.ActorCounts["teacher"] != 1 {
		t.Fatalf("teacher count = %d, want 1", ready.ActorCounts["teacher"])
	}
	if ready.ModelName != "minimax-default" {
		t.Fatalf("ModelName = %q, want %q", ready.ModelName, "minimax-default")
	}
}
