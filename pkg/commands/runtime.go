package commands

// Runtime provides command handlers with app state.
type Runtime struct {
	ListDefinitions func() []Definition
	GetModelInfo    func() (string, string)
	ListSkills      func() []string
	ShowSkill       func(name string) (string, bool)
	ClearHistory    func(sessionID string) error
}
