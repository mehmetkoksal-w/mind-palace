package commands

import (
	"testing"
)

func TestRunDashboardInvalidFlag(t *testing.T) {
	err := RunDashboard([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestRunDashboardInvalidPort(t *testing.T) {
	err := RunDashboard([]string{"--port", "0"})
	if err == nil {
		t.Error("expected error for invalid port 0")
	}

	err = RunDashboard([]string{"--port", "-1"})
	if err == nil {
		t.Error("expected error for negative port")
	}

	err = RunDashboard([]string{"--port", "70000"})
	if err == nil {
		t.Error("expected error for port > 65535")
	}
}

// Note: Full dashboard test would start an HTTP server which is complex.
// ExecuteDashboard is best tested via integration tests.
