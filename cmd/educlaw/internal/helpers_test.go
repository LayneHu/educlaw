package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigPathPrefersWorkingDirConfig(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	defer os.Chdir(wd)

	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loadPath, savePath := ResolveConfigPath("")
	if loadPath != "config.json" || savePath != "config.json" {
		t.Fatalf("ResolveConfigPath() = %q, %q", loadPath, savePath)
	}
}
