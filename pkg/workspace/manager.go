package workspace

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Manager manages workspace files for students, families, and teachers.
type Manager struct {
	baseDir string
}

// NewManager creates a new workspace Manager.
func NewManager(baseDir string) *Manager {
	return &Manager{baseDir: baseDir}
}

// BaseDir returns the base workspace directory.
func (m *Manager) BaseDir() string {
	return m.baseDir
}

// StudentDir returns the workspace directory for a student.
func (m *Manager) StudentDir(id string) string {
	return filepath.Join(m.baseDir, "students", id)
}

// FamilyDir returns the workspace directory for a family.
func (m *Manager) FamilyDir(id string) string {
	return filepath.Join(m.baseDir, "families", id)
}

// TeacherDir returns the workspace directory for a teacher.
func (m *Manager) TeacherDir(id string) string {
	return filepath.Join(m.baseDir, "teachers", id)
}

// AgentDir returns the workspace directory for agents.
func (m *Manager) AgentDir() string {
	return filepath.Join(m.baseDir, "agents")
}

// ReadFile reads a file from the given directory and returns its content.
// Returns empty string if file doesn't exist.
func (m *Manager) ReadFile(dir, filename string) string {
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		return ""
	}
	return string(data)
}

// WriteFile writes content to a file in the given directory using an atomic
// temp-file-then-rename to prevent data corruption on power loss or crash.
func (m *Manager) WriteFile(dir, filename, content string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	path := filepath.Join(dir, filename)
	if err := atomicWriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", path, err)
	}
	return nil
}

// atomicWriteFile writes data to path atomically via a temp file + rename.
func atomicWriteFile(path string, data []byte, _ os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op if rename succeeded

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// AppendDailyLog appends content to today's daily log file.
// Log is stored at memory/YYYYMM/YYYYMMDD.md
func (m *Manager) AppendDailyLog(dir, content string) error {
	now := time.Now()
	monthDir := filepath.Join(dir, "memory", now.Format("200601"))
	if err := os.MkdirAll(monthDir, 0755); err != nil {
		return fmt.Errorf("creating month directory: %w", err)
	}
	logFile := filepath.Join(monthDir, now.Format("20060102")+".md")

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer f.Close()

	timestamp := now.Format("15:04")
	entry := fmt.Sprintf("\n## %s\n%s\n", timestamp, content)
	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("writing log entry: %w", err)
	}
	return nil
}

// ReadMemory reads the long-term memory file (MEMORY.md) for an actor workspace.
func (m *Manager) ReadMemory(dir string) string {
	return m.ReadFile(dir, "MEMORY.md")
}

// GetRecentDailyNotes reads daily notes from the last N days, newest first.
// Returns all found notes joined with "---" separators.
func (m *Manager) GetRecentDailyNotes(dir string, days int) string {
	var parts []string
	now := time.Now()
	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i)
		logFile := filepath.Join(dir, "memory", date.Format("200601"), date.Format("20060102")+".md")
		data, err := os.ReadFile(logFile)
		if err != nil {
			continue
		}
		if content := strings.TrimSpace(string(data)); content != "" {
			parts = append(parts, content)
		}
	}
	return strings.Join(parts, "\n\n---\n\n")
}

// ReadDailyLog reads today's daily log content.
func (m *Manager) ReadDailyLog(dir string) string {
	now := time.Now()
	logFile := filepath.Join(dir, "memory", now.Format("200601"), now.Format("20060102")+".md")
	data, err := os.ReadFile(logFile)
	if err != nil {
		return ""
	}
	return string(data)
}

// InitFromTemplate copies template files to the target directory.
// Does not overwrite files that already exist.
func (m *Manager) InitFromTemplate(dir, templateDir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating target directory: %w", err)
	}

	entries, err := os.ReadDir(templateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading template directory: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(templateDir, entry.Name())
		dstPath := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			if err := m.InitFromTemplate(dstPath, srcPath); err != nil {
				return err
			}
			continue
		}

		// Don't overwrite existing files
		if _, err := os.Stat(dstPath); err == nil {
			continue
		}

		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("copying %s: %w", entry.Name(), err)
		}
	}
	return nil
}

// InitFromEmbeddedTemplate copies files from the embedded templates into the
// target directory. The templateType must be one of "student", "family", or
// "teacher". Existing files in targetDir are never overwritten.
func (m *Manager) InitFromEmbeddedTemplate(targetDir, templateType string) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("creating target directory: %w", err)
	}
	subFS, err := fs.Sub(EmbeddedTemplates, "templates/"+templateType)
	if err != nil {
		return fmt.Errorf("accessing embedded template %s: %w", templateType, err)
	}
	return fs.WalkDir(subFS, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == "." {
			return nil
		}
		dstPath := filepath.Join(targetDir, path)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}
		// Don't overwrite existing files
		if _, err := os.Stat(dstPath); err == nil {
			return nil
		}
		data, err := fs.ReadFile(subFS, path)
		if err != nil {
			return fmt.Errorf("reading embedded file %s: %w", path, err)
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0644)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
