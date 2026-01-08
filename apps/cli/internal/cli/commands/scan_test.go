package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunScanInvalidFlag(t *testing.T) {
	err := RunScan([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestExecuteScanFullOnEmptyDir(t *testing.T) {
	root := t.TempDir()

	// Initialize palace first
	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	// Create a simple Go file
	goFile := filepath.Join(root, "main.go")
	os.WriteFile(goFile, []byte("package main\nfunc main() {}\n"), 0o644)

	err = ExecuteScan(ScanOptions{Root: root, Full: true})
	if err != nil {
		t.Fatalf("ExecuteScan(Full) error: %v", err)
	}

	// Verify database was created
	dbPath := filepath.Join(root, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected palace.db to exist: %v", err)
	}
}

func TestExecuteScanIncrementalFallsBackToFull(t *testing.T) {
	root := t.TempDir()

	// Initialize palace first
	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	// Create a simple Go file
	goFile := filepath.Join(root, "main.go")
	os.WriteFile(goFile, []byte("package main\nfunc main() {}\n"), 0o644)

	// Incremental scan on fresh workspace should fall back to full
	err = ExecuteScan(ScanOptions{Root: root, Full: false})
	if err != nil {
		t.Fatalf("ExecuteScan(Incremental) error: %v", err)
	}

	// Verify database was created
	dbPath := filepath.Join(root, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected palace.db to exist: %v", err)
	}
}

func TestExecuteScanIncrementalNoChanges(t *testing.T) {
	root := t.TempDir()

	// Initialize and do first full scan
	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	goFile := filepath.Join(root, "main.go")
	os.WriteFile(goFile, []byte("package main\nfunc main() {}\n"), 0o644)

	err = ExecuteScan(ScanOptions{Root: root, Full: true})
	if err != nil {
		t.Fatalf("First ExecuteScan() error: %v", err)
	}

	// Second incremental scan should report no changes
	err = ExecuteScan(ScanOptions{Root: root, Full: false})
	if err != nil {
		t.Fatalf("Second ExecuteScan() error: %v", err)
	}
}
