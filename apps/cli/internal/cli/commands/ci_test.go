package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunCINoArgs(t *testing.T) {
	err := RunCI([]string{})
	if err == nil {
		t.Error("expected error for missing subcommand")
	}
}

func TestRunCIUnknownSubcommand(t *testing.T) {
	err := RunCI([]string{"unknown"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

func TestRunCICollectInvalidFlag(t *testing.T) {
	// ci collect delegates to check --collect, so invalid flag should error
	err := RunCI([]string{"collect", "--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestRunCISignalMissingDiff(t *testing.T) {
	root := t.TempDir()

	// Initialize and scan first
	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	goFile := filepath.Join(root, "main.go")
	os.WriteFile(goFile, []byte("package main\nfunc main() {}\n"), 0644)

	err = ExecuteScan(ScanOptions{Root: root, Full: true})
	if err != nil {
		t.Fatalf("ExecuteScan() error: %v", err)
	}

	// ci signal without --diff should error (signal requires diff)
	err = RunCI([]string{"signal", "--root", root})
	if err == nil {
		t.Error("expected error for missing diff range")
	}
}

func TestExecuteCheckCollectSuccess(t *testing.T) {
	root := t.TempDir()

	// Initialize and scan
	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	goFile := filepath.Join(root, "main.go")
	os.WriteFile(goFile, []byte("package main\nfunc main() {}\n"), 0644)

	err = ExecuteScan(ScanOptions{Root: root, Full: true})
	if err != nil {
		t.Fatalf("ExecuteScan() error: %v", err)
	}

	// Collect without diff (full scope) via check --collect
	err = ExecuteCheck(CheckOptions{
		Root:    root,
		Collect: true,
	})
	if err != nil {
		t.Fatalf("ExecuteCheck(Collect) error: %v", err)
	}

	// Verify context pack was created
	cpPath := filepath.Join(root, ".palace", "outputs", "context-pack.json")
	if _, err := os.Stat(cpPath); err != nil {
		t.Fatalf("context-pack.json not created: %v", err)
	}
}

func TestExecuteCheckCollectAllowStale(t *testing.T) {
	root := t.TempDir()

	// Initialize and scan
	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	goFile := filepath.Join(root, "main.go")
	os.WriteFile(goFile, []byte("package main\nfunc main() {}\n"), 0644)

	err = ExecuteScan(ScanOptions{Root: root, Full: true})
	if err != nil {
		t.Fatalf("ExecuteScan() error: %v", err)
	}

	// Collect with allow-stale flag
	err = ExecuteCheck(CheckOptions{
		Root:       root,
		Collect:    true,
		AllowStale: true,
	})
	if err != nil {
		t.Fatalf("ExecuteCheck(Collect, AllowStale) error: %v", err)
	}
}

func TestRunCIVerifyDelegates(t *testing.T) {
	root := t.TempDir()

	// Initialize and scan
	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	goFile := filepath.Join(root, "main.go")
	os.WriteFile(goFile, []byte("package main\nfunc main() {}\n"), 0644)

	err = ExecuteScan(ScanOptions{Root: root, Full: true})
	if err != nil {
		t.Fatalf("ExecuteScan() error: %v", err)
	}

	// CI verify should work (delegates to check)
	err = RunCI([]string{"verify", "--root", root})
	if err != nil {
		t.Fatalf("RunCI(verify) error: %v", err)
	}
}

func TestRunCIDispatch(t *testing.T) {
	root := t.TempDir()

	// Initialize and scan
	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	goFile := filepath.Join(root, "main.go")
	os.WriteFile(goFile, []byte("package main\nfunc main() {}\n"), 0644)

	err = ExecuteScan(ScanOptions{Root: root, Full: true})
	if err != nil {
		t.Fatalf("ExecuteScan() error: %v", err)
	}

	// Test dispatch to verify
	err = RunCI([]string{"verify", "--root", root})
	if err != nil {
		t.Fatalf("RunCI(verify) error: %v", err)
	}

	// Test dispatch to collect
	err = RunCI([]string{"collect", "--root", root})
	if err != nil {
		t.Fatalf("RunCI(collect) error: %v", err)
	}
}

func TestExecuteCheckSignalNoDiff(t *testing.T) {
	root := t.TempDir()

	// Initialize palace
	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	// Try signal without diff - should error
	err = ExecuteCheck(CheckOptions{
		Root:   root,
		Signal: true,
	})
	// This should error because --signal requires --diff
	if err == nil {
		t.Error("expected error for --signal without --diff")
	}
}
