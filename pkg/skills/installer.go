package skills

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Installer installs skills into the workspace skill directory.
type Installer struct {
	targetDir string
	client    *http.Client
}

// NewInstaller creates a new skill installer.
func NewInstaller(targetDir string) *Installer {
	return &Installer{
		targetDir: targetDir,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

// InstallFromGitHub installs a skill from a GitHub repo path like "owner/repo".
func (i *Installer) InstallFromGitHub(ctx context.Context, repo string, force bool) (string, error) {
	repo = strings.TrimSpace(repo)
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("repo must be in owner/repo format")
	}

	skillName := parts[1]
	skillDir := filepath.Join(i.targetDir, skillName)
	if !force {
		if _, err := os.Stat(skillDir); err == nil {
			return "", fmt.Errorf("skill %q already exists", skillName)
		}
	}
	if force {
		if err := os.RemoveAll(skillDir); err != nil {
			return "", fmt.Errorf("removing existing skill: %w", err)
		}
	}

	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/SKILL.md", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	resp, err := i.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching skill: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching skill: unexpected HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading skill body: %w", err)
	}
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return "", fmt.Errorf("creating skill directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), body, 0o644); err != nil {
		return "", fmt.Errorf("writing SKILL.md: %w", err)
	}
	return skillName, nil
}
