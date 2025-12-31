package commands

import (
	"testing"
)

func TestRegisterAndGet(t *testing.T) {
	// Clear registry for test isolation
	registry = make(map[string]*Command)

	cmd := &Command{
		Name:        "test",
		Aliases:     []string{"t", "tst"},
		Description: "Test command",
		Run:         func(args []string) error { return nil },
	}

	Register(cmd)

	// Test getting by name
	got, ok := Get("test")
	if !ok {
		t.Error("expected to find command by name")
	}
	if got.Name != "test" {
		t.Errorf("Name = %q, want %q", got.Name, "test")
	}

	// Test getting by alias
	got, ok = Get("t")
	if !ok {
		t.Error("expected to find command by alias 't'")
	}
	if got.Name != "test" {
		t.Errorf("Name = %q, want %q", got.Name, "test")
	}

	got, ok = Get("tst")
	if !ok {
		t.Error("expected to find command by alias 'tst'")
	}

	// Test getting non-existent command
	_, ok = Get("nonexistent")
	if ok {
		t.Error("expected not to find non-existent command")
	}
}

func TestList(t *testing.T) {
	// Clear registry for test isolation
	registry = make(map[string]*Command)

	cmd1 := &Command{Name: "cmd1", Aliases: []string{"c1"}, Run: func(args []string) error { return nil }}
	cmd2 := &Command{Name: "cmd2", Run: func(args []string) error { return nil }}

	Register(cmd1)
	Register(cmd2)

	commands := List()

	// Should return 2 unique commands (not 3 with alias)
	if len(commands) != 2 {
		t.Errorf("List() returned %d commands, want 2", len(commands))
	}
}

func TestCommandExecution(t *testing.T) {
	called := false
	cmd := &Command{
		Name: "exec",
		Run: func(args []string) error {
			called = true
			if len(args) != 2 {
				t.Errorf("Expected 2 args, got %d", len(args))
			}
			return nil
		},
	}
	if cmd.Name != "exec" {
		t.Errorf("Name = %q, want %q", cmd.Name, "exec")
	}

	err := cmd.Run([]string{"arg1", "arg2"})
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
	if !called {
		t.Error("Command was not executed")
	}
}
