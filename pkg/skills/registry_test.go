package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryManagerSearchLocal(t *testing.T) {
	root := t.TempDir()
	workspaceDir := filepath.Join(root, "workspace")
	builtinDir := filepath.Join(root, "builtin")
	mustWriteSkill(t, filepath.Join(workspaceDir, "math-helper", "SKILL.md"), "---\ndescription: math games and math drills\n---\nbody")
	mustWriteSkill(t, filepath.Join(builtinDir, "story-problem", "SKILL.md"), "---\ndescription: turn problems into stories\n---\nbody")

	loader := NewLoader(workspaceDir, "", builtinDir)
	rm := NewRegistryManager(loader)
	results := rm.SearchLocal("math game", 10)
	if len(results) == 0 {
		t.Fatal("SearchLocal() returned no results")
	}
	if results[0].Name != "math-helper" {
		t.Fatalf("top result = %q, want math-helper", results[0].Name)
	}
}

func mustWriteSkill(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
