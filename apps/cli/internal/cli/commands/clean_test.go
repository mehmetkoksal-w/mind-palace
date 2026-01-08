package commands_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/commands"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

func TestRunCleanInvalidFlag(t *testing.T) {
	err := commands.RunClean([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestExecuteCleanDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	palaceDir := filepath.Join(tmpDir, ".palace")
	os.MkdirAll(palaceDir, 0o755)

	// Initialize memory
	mem, err := memory.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open memory: %v", err)
	}
	mem.Close()

	err = commands.ExecuteClean(commands.CleanOptions{
		Root:   tmpDir,
		DryRun: true,
	})
	if err != nil {
		t.Errorf("ExecuteClean dry-run failed: %v", err)
	}
}

func TestExecuteCleanNoMemory(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't initialize memory - it should print a warning but not fail

	err := commands.ExecuteClean(commands.CleanOptions{
		Root:   tmpDir,
		DryRun: true,
	})
	if err != nil {
		t.Errorf("ExecuteClean should not fail when memory doesn't exist: %v", err)
	}
}

func TestExecuteCleanActual(t *testing.T) {
	tmpDir := t.TempDir()
	palaceDir := filepath.Join(tmpDir, ".palace")
	os.MkdirAll(palaceDir, 0o755)

	// Initialize memory
	mem, err := memory.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open memory: %v", err)
	}
	mem.Close()

	err = commands.ExecuteClean(commands.CleanOptions{
		Root:   tmpDir,
		DryRun: false,
	})
	if err != nil {
		t.Errorf("ExecuteClean failed: %v", err)
	}
}
