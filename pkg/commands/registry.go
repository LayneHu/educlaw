package commands

import "strings"

// Registry stores commands by normalized name.
type Registry struct {
	defs  []Definition
	index map[string]int
}

func NewRegistry(defs []Definition) *Registry {
	index := make(map[string]int, len(defs))
	stored := make([]Definition, len(defs))
	copy(stored, defs)
	for i, def := range stored {
		index[normalize(def.Name)] = i
	}
	return &Registry{defs: stored, index: index}
}

func (r *Registry) Definitions() []Definition {
	out := make([]Definition, len(r.defs))
	copy(out, r.defs)
	return out
}

func (r *Registry) Lookup(name string) (Definition, bool) {
	idx, ok := r.index[normalize(name)]
	if !ok {
		return Definition{}, false
	}
	return r.defs[idx], true
}

func normalize(s string) string {
	s = strings.TrimSpace(strings.TrimPrefix(s, "/"))
	return strings.ToLower(s)
}
