package skills

import (
	"sort"
	"strings"
)

// SearchResult represents a skill search result.
type SearchResult struct {
	Score   float64
	Name    string
	Summary string
	Source  string
}

// RegistryManager searches across skills visible to the current loader.
type RegistryManager struct {
	loader *Loader
}

// NewRegistryManager creates a registry manager backed by the local loader.
func NewRegistryManager(loader *Loader) *RegistryManager {
	return &RegistryManager{loader: loader}
}

// SearchLocal searches workspace/global/builtin skills with a simple token scorer.
func (rm *RegistryManager) SearchLocal(query string, limit int) []SearchResult {
	if rm == nil || rm.loader == nil {
		return nil
	}
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return nil
	}

	terms := strings.Fields(query)
	if len(terms) == 0 {
		terms = []string{query}
	}

	var results []SearchResult
	for _, skill := range rm.loader.ListSkills() {
		score := scoreSkillMatch(skill, terms)
		if score <= 0 {
			continue
		}
		results = append(results, SearchResult{
			Score:   score,
			Name:    skill.Name,
			Summary: skill.Description,
			Source:  classifySkillSource(skill, rm.loader),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Name < results[j].Name
		}
		return results[i].Score > results[j].Score
	})
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results
}

func scoreSkillMatch(skill SkillInfo, terms []string) float64 {
	name := strings.ToLower(skill.Name)
	desc := strings.ToLower(skill.Description)
	score := 0.0
	for _, term := range terms {
		switch {
		case name == term:
			score += 10
		case strings.Contains(name, term):
			score += 4
		}
		if desc != "" && strings.Contains(desc, term) {
			score += 2
		}
	}
	return score
}

func classifySkillSource(skill SkillInfo, loader *Loader) string {
	switch {
	case loader.workspaceDir != "" && strings.HasPrefix(skill.Dir, loader.workspaceDir):
		return "workspace"
	case loader.globalDir != "" && strings.HasPrefix(skill.Dir, loader.globalDir):
		return "global"
	default:
		return "builtin"
	}
}
