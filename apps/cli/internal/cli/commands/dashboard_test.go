package commands

import (
	"testing"
)

func TestRunDashboard(t *testing.T) {
	// Dashboard has been removed in v0.4.2 and now just shows a deprecation message.
	// Verify it returns nil regardless of arguments.
	err := RunDashboard([]string{})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	err = RunDashboard([]string{"--invalid-flag"})
	if err != nil {
		t.Errorf("expected no error for deprecated command, got: %v", err)
	}

	err = RunDashboard([]string{"--port", "70000"})
	if err != nil {
		t.Errorf("expected no error for deprecated command, got: %v", err)
	}
}
