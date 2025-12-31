package commands_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/commands"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

func TestRunMaintenanceInvalidFlag(t *testing.T) {
	err := commands.RunMaintenance([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestExecuteMaintenanceDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	palaceDir := filepath.Join(tmpDir, ".palace")
	os.MkdirAll(palaceDir, 0755)

	// Initialize memory
	mem, err := memory.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open memory: %v", err)
	}
	mem.Close()

	err = commands.ExecuteMaintenance(commands.MaintenanceOptions{
		Root:   tmpDir,
		DryRun: true,
	})
	if err != nil {
		t.Errorf("ExecuteMaintenance dry-run failed: %v", err)
	}
}

func TestExecuteMaintenanceNoMemory(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't initialize memory - it should print a warning but not fail

	err := commands.ExecuteMaintenance(commands.MaintenanceOptions{
		Root:   tmpDir,
		DryRun: true,
	})
	if err != nil {
		t.Errorf("ExecuteMaintenance should not fail when memory doesn't exist: %v", err)
	}
}

func TestExecuteMaintenanceActual(t *testing.T) {
	tmpDir := t.TempDir()
	palaceDir := filepath.Join(tmpDir, ".palace")
	os.MkdirAll(palaceDir, 0755)

	// Initialize memory
	mem, err := memory.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open memory: %v", err)
	}
	mem.Close()

	err = commands.ExecuteMaintenance(commands.MaintenanceOptions{
		Root:   tmpDir,
		DryRun: false,
	})
	if err != nil {
		t.Errorf("ExecuteMaintenance failed: %v", err)
	}
}
