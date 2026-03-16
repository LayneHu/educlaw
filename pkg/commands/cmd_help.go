package commands

import (
	"context"
	"fmt"
	"strings"
)

func helpCommand() Definition {
	return Definition{
		Name:        "help",
		Description: "Show available slash commands",
		Usage:       "/help",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			defs := Builtins()
			if rt != nil && rt.ListDefinitions != nil {
				defs = rt.ListDefinitions()
			}
			lines := make([]string, 0, len(defs)+1)
			lines = append(lines, "Available commands:")
			for _, def := range defs {
				lines = append(lines, fmt.Sprintf("- %s: %s", def.effectiveUsage(), def.Description))
			}
			return req.Reply(strings.Join(lines, "\n"))
		},
	}
}
