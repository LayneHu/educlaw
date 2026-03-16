package llm

import "testing"

func TestUsesMaxCompletionTokens(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{model: "gpt-4o", want: false},
		{model: "gpt-5", want: true},
		{model: "gpt-5-mini", want: true},
		{model: "o1", want: true},
		{model: "o3-mini", want: true},
		{model: "o4-mini", want: true},
		{model: "gemini-2.5-pro", want: false},
	}

	for _, tt := range tests {
		if got := usesMaxCompletionTokens(tt.model); got != tt.want {
			t.Fatalf("usesMaxCompletionTokens(%q) = %v, want %v", tt.model, got, tt.want)
		}
	}
}
