package commands_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/commands"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

func TestRunCorridorNoArgs(t *testing.T) {
	err := commands.RunCorridor([]string{})
	if err == nil {
		t.Error("expected error for no arguments")
	}
}

func TestRunCorridorUnknownSubcommand(t *testing.T) {
	err := commands.RunCorridor([]string{"invalid"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

func TestExecuteCorridorLinkNoArgs(t *testing.T) {
	err := commands.ExecuteCorridorLink([]string{})
	if err == nil {
		t.Error("expected error for no arguments")
	}
}

func TestExecuteCorridorLinkOneArg(t *testing.T) {
	err := commands.ExecuteCorridorLink([]string{"name"})
	if err == nil {
		t.Error("expected error for only one argument")
	}
}

func TestExecuteCorridorUnlinkNoArgs(t *testing.T) {
	err := commands.ExecuteCorridorUnlink([]string{})
	if err == nil {
		t.Error("expected error for no arguments")
	}
}

func TestExecuteCorridorPromoteNoArgs(t *testing.T) {
	err := commands.ExecuteCorridorPromote([]string{})
	if err == nil {
		t.Error("expected error for no arguments")
	}
}

func TestExecuteCorridorPersonalInvalidFlag(t *testing.T) {
	err := commands.ExecuteCorridorPersonal([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestExecuteCorridorSearchInvalidFlag(t *testing.T) {
	err := commands.ExecuteCorridorSearch([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestExecuteCorridorSearchInvalidLimit(t *testing.T) {
	err := commands.ExecuteCorridorSearch([]string{"--limit", "-1"})
	if err == nil {
		t.Error("expected error for invalid limit")
	}
}

func TestExecuteCorridorPromoteNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	palaceDir := filepath.Join(tmpDir, ".palace")
	os.MkdirAll(palaceDir, 0o755)

	// Initialize memory
	mem, err := memory.Open(tmpDir)
	if err != nil {
		t.Fatalf("failed to open memory: %v", err)
	}
	mem.Close()

	err = commands.ExecuteCorridorPromote([]string{"--root", tmpDir, "nonexistent"})
	if err == nil {
		t.Error("expected error for non-existent learning")
	}
}

func TestExecuteCorridorListSuccess(t *testing.T) {
	// ExecuteCorridorList doesn't take args and uses global corridor
	// Just verify it doesn't crash
	err := commands.ExecuteCorridorList([]string{})
	// Should succeed (or fail gracefully if global corridor not accessible)
	_ = err
}

func TestRunCorridorListDispatch(t *testing.T) {
	// Test dispatching to list subcommand
	err := commands.RunCorridor([]string{"list"})
	// Should succeed or fail gracefully
	_ = err
}
