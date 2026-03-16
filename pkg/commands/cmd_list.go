package commands

import (
	"context"
	"strings"
)

func listCommand() Definition {
	return Definition{
		Name:        "list",
		Description: "List available skills",
		Usage:       "/list",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.ListSkills == nil {
				return req.Reply("Listing skills is unavailable.")
			}
			skills := rt.ListSkills()
			if len(skills) == 0 {
				return req.Reply("No skills available.")
			}
			return req.Reply("Available skills:\n- " + strings.Join(skills, "\n- "))
		},
	}
}
