package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunStatsInvalidFlag(t *testing.T) {
	err := RunStats([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestExecuteStatsNoIndex(t *testing.T) {
	root := t.TempDir()

	// Create .palace directory but no index
	palaceDir := filepath.Join(root, ".palace")
	os.MkdirAll(palaceDir, 0o755)

	// This should not error, just report "not available"
	err := ExecuteStats(StatsOptions{Root: root})
	if err != nil {
		t.Errorf("ExecuteStats() error: %v", err)
	}
}

func TestExecuteStatsEmptyDir(t *testing.T) {
	root := t.TempDir()

	// This should not error, just report "not available"
	err := ExecuteStats(StatsOptions{Root: root})
	if err != nil {
		t.Errorf("ExecuteStats() error: %v", err)
	}
}
