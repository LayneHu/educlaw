package commands

import (
	"context"
	"fmt"
	"strings"
)

func showCommand() Definition {
	return Definition{
		Name:        "show",
		Description: "Show model info or a skill description",
		Usage:       "/show model | /show skill <name>",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			fields := strings.Fields(req.Text)
			if len(fields) < 2 {
				return req.Reply("Usage: /show model | /show skill <name>")
			}
			switch strings.ToLower(fields[1]) {
			case "model":
				if rt == nil || rt.GetModelInfo == nil {
					return req.Reply("Model info is unavailable.")
				}
				model, provider := rt.GetModelInfo()
				return req.Reply(fmt.Sprintf("Current model: %s\nProvider: %s", model, provider))
			case "skill":
				if len(fields) < 3 {
					return req.Reply("Usage: /show skill <name>")
				}
				if rt == nil || rt.ShowSkill == nil {
					return req.Reply("Skill details are unavailable.")
				}
				content, ok := rt.ShowSkill(fields[2])
				if !ok {
					return req.Reply(fmt.Sprintf("Skill %s not found.", fields[2]))
				}
				return req.Reply(content)
			default:
				return req.Reply("Usage: /show model | /show skill <name>")
			}
		},
	}
}
