package bus

// InboundMessage represents a message coming into the system from a user.
type InboundMessage struct {
	Channel   string `json:"channel"`
	ActorID   string `json:"actor_id"`
	ActorType string `json:"actor_type"`
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
}

// OutboundMessage represents a message going out to a user session.
type OutboundMessage struct {
	SessionID   string `json:"session_id"`
	ActorID     string `json:"actor_id"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"` // "text", "rendered", "error", "tool_call"
	Done        bool   `json:"done"`
}

// ToolEvent carries a tool-call or tool-result event for the frontend log panel.
type ToolEvent struct {
	Phase   string `json:"phase"`   // "call" | "result"
	Tool    string `json:"tool"`
	Summary string `json:"summary"` // human-readable one-liner
}

// RenderedContent represents interactive content to be rendered in the browser.
type RenderedContent struct {
	ID      string `json:"id"`
	Type    string `json:"type"`  // game, quiz, visual, embed, video, report
	Title   string `json:"title"`
	Content string `json:"content"` // HTML content
}
