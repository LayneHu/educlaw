package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SkillInfo holds metadata about a skill.
type SkillInfo struct {
	Name        string
	Description string
	Dir         string
}

// Loader scans directories for skills and loads their content.
type Loader struct {
	workspaceDir string
	globalDir    string
	builtinDir   string
}

// NewLoader creates a new skills Loader.
func NewLoader(workspaceDir, globalDir, builtinDir string) *Loader {
	return &Loader{
		workspaceDir: workspaceDir,
		globalDir:    globalDir,
		builtinDir:   builtinDir,
	}
}

// ListSkills scans all skill directories and returns available skills.
func (l *Loader) ListSkills() []SkillInfo {
	var skills []SkillInfo
	seen := make(map[string]bool)

	for _, dir := range []string{l.workspaceDir, l.globalDir, l.builtinDir} {
		if dir == "" {
			continue
		}
		found := l.scanDir(dir)
		for _, s := range found {
			if !seen[s.Name] {
				skills = append(skills, s)
				seen[s.Name] = true
			}
		}
	}
	return skills
}

// WorkspaceDir returns the workspace skill root.
func (l *Loader) WorkspaceDir() string { return l.workspaceDir }

// GlobalDir returns the global skill root.
func (l *Loader) GlobalDir() string { return l.globalDir }

// BuiltinDir returns the builtin skill root.
func (l *Loader) BuiltinDir() string { return l.builtinDir }

func (l *Loader) scanDir(dir string) []SkillInfo {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var skills []SkillInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillDir := filepath.Join(dir, entry.Name())
		skillFile := filepath.Join(skillDir, "SKILL.md")
		data, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}

		info := SkillInfo{
			Name: entry.Name(),
			Dir:  skillDir,
		}
		info.Description = parseFrontmatterDescription(string(data))
		skills = append(skills, info)
	}
	return skills
}

// parseFrontmatterDescription extracts the description from YAML frontmatter.
func parseFrontmatterDescription(content string) string {
	if !strings.HasPrefix(content, "---") {
		return ""
	}
	// Find closing ---
	rest := content[3:]
	end := strings.Index(rest, "---")
	if end < 0 {
		return ""
	}
	frontmatter := rest[:end]
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "description:") {
			desc := strings.TrimPrefix(line, "description:")
			return strings.TrimSpace(desc)
		}
	}
	return ""
}

// LoadSkill reads a skill's full content (without frontmatter).
func (l *Loader) LoadSkill(name string) (string, bool) {
	for _, dir := range []string{l.workspaceDir, l.globalDir, l.builtinDir} {
		if dir == "" {
			continue
		}
		skillFile := filepath.Join(dir, name, "SKILL.md")
		data, err := os.ReadFile(skillFile)
		if err != nil {
			continue
		}
		return stripFrontmatter(string(data)), true
	}
	return "", false
}

// LoadSkillAsset reads an asset file from within a skill directory.
func (l *Loader) LoadSkillAsset(skillName, assetFile string) (string, bool) {
	for _, dir := range []string{l.workspaceDir, l.globalDir, l.builtinDir} {
		if dir == "" {
			continue
		}
		assetPath := filepath.Join(dir, skillName, assetFile)
		data, err := os.ReadFile(assetPath)
		if err != nil {
			continue
		}
		return string(data), true
	}
	return "", false
}

// stripFrontmatter removes the YAML frontmatter from markdown content.
func stripFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---") {
		return content
	}
	rest := content[3:]
	// Skip the first newline after ---
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return content
	}
	after := rest[end+4:]
	if strings.HasPrefix(after, "\n") {
		after = after[1:]
	}
	return after
}

// BuildSummary returns an XML-formatted summary of all available skills.
func (l *Loader) BuildSummary() string {
	skills := l.ListSkills()
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<available_skills>\n")
	for _, s := range skills {
		sb.WriteString(fmt.Sprintf("  <skill name=%q description=%q />\n", s.Name, s.Description))
	}
	sb.WriteString("</available_skills>")
	return sb.String()
}
