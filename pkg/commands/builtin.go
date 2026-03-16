package commands

// Builtins returns the supported slash commands for the current web app.
func Builtins() []Definition {
	return []Definition{
		helpCommand(),
		listCommand(),
		showCommand(),
		clearCommand(),
	}
}
