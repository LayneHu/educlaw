package commands

import "fmt"

// Definition describes a single slash command.
type Definition struct {
	Name        string
	Description string
	Usage       string
	Handler     Handler
}

func (d Definition) effectiveUsage() string {
	if d.Usage != "" {
		return d.Usage
	}
	return fmt.Sprintf("/%s", d.Name)
}
