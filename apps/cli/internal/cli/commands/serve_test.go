package commands

import (
	"testing"
)

func TestRunServeInvalidFlag(t *testing.T) {
	err := RunServe([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestExecuteServeMissingIndex(t *testing.T) {
	root := t.TempDir()

	// Initialize palace but don't scan
	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	err = ExecuteServe(ServeOptions{Root: root})
	if err == nil {
		t.Error("expected error for missing index")
	}
}

// Note: Full serve test would require mocking stdin/stdout
// which is complex. For now, we test flag parsing and error cases.
