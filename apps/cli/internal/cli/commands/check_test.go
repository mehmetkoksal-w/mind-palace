package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunCheckInvalidFlag(t *testing.T) {
	err := RunCheck([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestExecuteCheckMissingIndex(t *testing.T) {
	root := t.TempDir()

	// Initialize palace but don't scan
	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	err = ExecuteCheck(CheckOptions{Root: root})
	if err == nil {
		t.Error("expected error for missing index")
	}
}

func TestExecuteCheckSuccess(t *testing.T) {
	root := t.TempDir()

	// Initialize and scan
	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	goFile := filepath.Join(root, "main.go")
	os.WriteFile(goFile, []byte("package main\nfunc main() {}\n"), 0o644)

	err = ExecuteScan(ScanOptions{Root: root, Full: true})
	if err != nil {
		t.Fatalf("ExecuteScan() error: %v", err)
	}

	// Check should pass
	err = ExecuteCheck(CheckOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteCheck() error: %v", err)
	}
}

func TestExecuteCheckStrictMode(t *testing.T) {
	root := t.TempDir()

	// Initialize and scan
	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	goFile := filepath.Join(root, "main.go")
	os.WriteFile(goFile, []byte("package main\nfunc main() {}\n"), 0o644)

	err = ExecuteScan(ScanOptions{Root: root, Full: true})
	if err != nil {
		t.Fatalf("ExecuteScan() error: %v", err)
	}

	// Check in strict mode
	err = ExecuteCheck(CheckOptions{Root: root, Strict: true})
	if err != nil {
		t.Fatalf("ExecuteCheck(Strict) error: %v", err)
	}
}
