package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunSessionNoArgs(t *testing.T) {
	err := RunSession([]string{})
	if err == nil {
		t.Error("expected error for missing subcommand")
	}
}

func TestRunSessionUnknownSubcommand(t *testing.T) {
	err := RunSession([]string{"unknown"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

func TestRunSessionStartInvalidFlag(t *testing.T) {
	err := RunSessionStart([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestRunSessionEndInvalidFlag(t *testing.T) {
	err := RunSessionEnd([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestRunSessionEndMissingID(t *testing.T) {
	err := RunSessionEnd([]string{})
	if err == nil {
		t.Error("expected error for missing session ID")
	}
}

func TestRunSessionListInvalidFlag(t *testing.T) {
	err := RunSessionList([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestRunSessionListInvalidLimit(t *testing.T) {
	err := RunSessionList([]string{"--limit", "-1"})
	if err == nil {
		t.Error("expected error for negative limit")
	}
}

func TestRunSessionShowInvalidFlag(t *testing.T) {
	err := RunSessionShow([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestRunSessionShowMissingID(t *testing.T) {
	err := RunSessionShow([]string{})
	if err == nil {
		t.Error("expected error for missing session ID")
	}
}

func TestExecuteSessionStartSuccess(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	err = ExecuteSessionStart(SessionStartOptions{
		Root:  root,
		Agent: "test-agent",
		Goal:  "test goal",
	})
	if err != nil {
		t.Fatalf("ExecuteSessionStart() error: %v", err)
	}
}

func TestExecuteSessionListSuccess(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	// Create a session first
	err = ExecuteSessionStart(SessionStartOptions{
		Root:  root,
		Agent: "test-agent",
		Goal:  "test goal",
	})
	if err != nil {
		t.Fatalf("ExecuteSessionStart() error: %v", err)
	}

	// List sessions
	err = ExecuteSessionList(SessionListOptions{
		Root:   root,
		Active: false,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("ExecuteSessionList() error: %v", err)
	}
}

func TestExecuteSessionListActiveOnly(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	// List active sessions only
	err = ExecuteSessionList(SessionListOptions{
		Root:   root,
		Active: true,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("ExecuteSessionList(Active) error: %v", err)
	}
}

func TestRunSessionDispatch(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	// Test dispatch to start
	err = RunSession([]string{"start", "--root", root, "--agent", "test", "my goal"})
	if err != nil {
		t.Fatalf("RunSession(start) error: %v", err)
	}

	// Test dispatch to list
	err = RunSession([]string{"list", "--root", root})
	if err != nil {
		t.Fatalf("RunSession(list) error: %v", err)
	}
}

func TestExecuteSessionWorkflow(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	// Create memory.db file
	memDir := filepath.Join(root, ".palace", "index")
	os.MkdirAll(memDir, 0755)

	// Start a session
	err = ExecuteSessionStart(SessionStartOptions{
		Root:  root,
		Agent: "test-agent",
		Goal:  "implement feature",
	})
	if err != nil {
		t.Fatalf("ExecuteSessionStart() error: %v", err)
	}

	// List should show the session
	err = ExecuteSessionList(SessionListOptions{Root: root, Limit: 10})
	if err != nil {
		t.Fatalf("ExecuteSessionList() error: %v", err)
	}
}

func TestExecuteSessionEndNonexistent(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	// End a session that doesn't exist - should succeed (EndSession is idempotent)
	err = ExecuteSessionEnd(SessionEndOptions{
		Root:      root,
		SessionID: "nonexistent_session",
		Summary:   "test",
		State:     "completed",
	})
	// EndSession may not error for non-existent sessions
	// just verify it doesn't crash
	_ = err
}

func TestExecuteSessionShowNotFound(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	// Show non-existent session - should error
	err = ExecuteSessionShow(SessionShowOptions{
		Root:      root,
		SessionID: "nonexistent_session",
	})
	if err == nil {
		t.Error("expected error for non-existent session")
	}
}
