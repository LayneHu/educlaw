package commands

import (
	"context"
	"testing"
)

func TestExecutorHelp(t *testing.T) {
	reg := NewRegistry(Builtins())
	rt := &Runtime{
		ListDefinitions: func() []Definition { return Builtins() },
	}
	exec := NewExecutor(reg, rt)
	var got string
	res := exec.Execute(context.Background(), Request{
		Text:  "/help",
		Reply: func(s string) error { got = s; return nil },
	})
	if !res.Handled {
		t.Fatal("expected command to be handled")
	}
	if got == "" {
		t.Fatal("expected help output")
	}
}
