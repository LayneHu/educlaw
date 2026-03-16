package commands

import "context"

func clearCommand() Definition {
	return Definition{
		Name:        "clear",
		Description: "Clear the current session history",
		Usage:       "/clear",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.ClearHistory == nil {
				return req.Reply("Clear history is unavailable.")
			}
			if req.SessionID == "" {
				return req.Reply("No active session to clear.")
			}
			if err := rt.ClearHistory(req.SessionID); err != nil {
				return req.Reply("Failed to clear chat history: " + err.Error())
			}
			return req.Reply("Chat history cleared.")
		},
	}
}
