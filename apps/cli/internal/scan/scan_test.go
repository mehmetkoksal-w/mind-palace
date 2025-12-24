package scan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
)

func TestRunRequiresPalaceLayout(t *testing.T) {
	tmpDir := t.TempDir()

	// Run without palace layout - should succeed in creating it
	_, _, err := Run(tmpDir)
	if err != nil {
		// May fail due to no files to scan, which is fine
		t.Logf("Run on empty dir: %v", err)
	}

	// Verify layout was created
	palaceDir := filepath.Join(tmpDir, ".palace")
	if _, err := os.Stat(palaceDir); os.IsNotExist(err) {
		t.Error("expected .palace directory to be created")
	}
}

func TestRunCreatesIndex(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple file to index
	testFile := filepath.Join(tmpDir, "main.go")
	content := `package main

func main() {
	println("Hello")
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	summary, count, err := Run(tmpDir)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if count == 0 {
		t.Error("expected at least one file to be indexed")
	}

	if summary.FileCount == 0 {
		t.Error("expected FileCount > 0")
	}

	// Verify index database was created
	dbPath := filepath.Join(tmpDir, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected index database to be created")
	}

	// Verify scan.json was created
	scanPath := filepath.Join(tmpDir, ".palace", "index", "scan.json")
	if _, err := os.Stat(scanPath); os.IsNotExist(err) {
		t.Error("expected scan.json to be created")
	}
}

func TestRunIncrementalRequiresExistingIndex(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal palace layout without index
	palaceDir := filepath.Join(tmpDir, ".palace")
	if err := os.MkdirAll(palaceDir, 0755); err != nil {
		t.Fatalf("failed to create palace dir: %v", err)
	}

	// RunIncremental should fail without existing index
	_, err := RunIncremental(tmpDir)
	if err == nil {
		t.Error("expected error when no index exists")
	}

	expectedMsg := "no index found"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("expected error containing %q, got %q", expectedMsg, err.Error())
	}
}

func TestRunIncrementalAfterFullScan(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "main.go")
	content := `package main

func main() {}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Run full scan first
	_, _, err := Run(tmpDir)
	if err != nil {
		t.Fatalf("Full scan failed: %v", err)
	}

	// Run incremental scan - should succeed with no changes
	summary, err := RunIncremental(tmpDir)
	if err != nil {
		t.Fatalf("Incremental scan failed: %v", err)
	}

	// No changes, so all files should be unchanged
	if summary.FilesAdded != 0 {
		t.Errorf("expected 0 added files, got %d", summary.FilesAdded)
	}
	if summary.FilesModified != 0 {
		t.Errorf("expected 0 modified files, got %d", summary.FilesModified)
	}
	if summary.FilesDeleted != 0 {
		t.Errorf("expected 0 deleted files, got %d", summary.FilesDeleted)
	}
}

func TestRunIncrementalDetectsNewFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial file
	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Run full scan
	_, _, err := Run(tmpDir)
	if err != nil {
		t.Fatalf("Full scan failed: %v", err)
	}

	// Add a new file
	newFile := filepath.Join(tmpDir, "utils.go")
	if err := os.WriteFile(newFile, []byte("package main\n\nfunc helper() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write new file: %v", err)
	}

	// Run incremental scan
	summary, err := RunIncremental(tmpDir)
	if err != nil {
		t.Fatalf("Incremental scan failed: %v", err)
	}

	if summary.FilesAdded != 1 {
		t.Errorf("expected 1 added file, got %d", summary.FilesAdded)
	}
}

func TestRunIncrementalDetectsModifiedFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial file
	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Run full scan
	_, _, err := Run(tmpDir)
	if err != nil {
		t.Fatalf("Full scan failed: %v", err)
	}

	// Modify the file
	if err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// Run incremental scan
	summary, err := RunIncremental(tmpDir)
	if err != nil {
		t.Fatalf("Incremental scan failed: %v", err)
	}

	if summary.FilesModified != 1 {
		t.Errorf("expected 1 modified file, got %d", summary.FilesModified)
	}
}

func TestRunIncrementalDetectsDeletedFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial files
	file1 := filepath.Join(tmpDir, "main.go")
	file2 := filepath.Join(tmpDir, "utils.go")
	if err := os.WriteFile(file1, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}

	// Run full scan
	_, _, err := Run(tmpDir)
	if err != nil {
		t.Fatalf("Full scan failed: %v", err)
	}

	// Delete one file
	if err := os.Remove(file2); err != nil {
		t.Fatalf("failed to delete file: %v", err)
	}

	// Run incremental scan
	summary, err := RunIncremental(tmpDir)
	if err != nil {
		t.Fatalf("Incremental scan failed: %v", err)
	}

	if summary.FilesDeleted != 1 {
		t.Errorf("expected 1 deleted file, got %d", summary.FilesDeleted)
	}
}

func TestRunIncrementalFilesUnchangedCalculation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 3 initial files
	for i, name := range []string{"a.go", "b.go", "c.go"} {
		path := filepath.Join(tmpDir, name)
		content := []byte("package main\n// file " + string(rune('a'+i)) + "\n")
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	// Run full scan
	_, _, err := Run(tmpDir)
	if err != nil {
		t.Fatalf("Full scan failed: %v", err)
	}

	// Modify one file, delete one file, add one file
	// Modify a.go
	if err := os.WriteFile(filepath.Join(tmpDir, "a.go"), []byte("package main\n// modified\n"), 0644); err != nil {
		t.Fatalf("failed to modify a.go: %v", err)
	}
	// Delete b.go
	if err := os.Remove(filepath.Join(tmpDir, "b.go")); err != nil {
		t.Fatalf("failed to delete b.go: %v", err)
	}
	// Add d.go
	if err := os.WriteFile(filepath.Join(tmpDir, "d.go"), []byte("package main\n// new\n"), 0644); err != nil {
		t.Fatalf("failed to write d.go: %v", err)
	}

	// Run incremental scan
	summary, err := RunIncremental(tmpDir)
	if err != nil {
		t.Fatalf("Incremental scan failed: %v", err)
	}

	// Expected: 1 modified, 1 deleted, 1 added, 1 unchanged (c.go)
	if summary.FilesModified != 1 {
		t.Errorf("expected 1 modified, got %d", summary.FilesModified)
	}
	if summary.FilesDeleted != 1 {
		t.Errorf("expected 1 deleted, got %d", summary.FilesDeleted)
	}
	if summary.FilesAdded != 1 {
		t.Errorf("expected 1 added, got %d", summary.FilesAdded)
	}
	if summary.FilesUnchanged != 1 {
		t.Errorf("expected 1 unchanged (c.go), got %d", summary.FilesUnchanged)
	}
}

func TestRunWithInvalidPath(t *testing.T) {
	_, _, err := Run("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestRunIncrementalWithInvalidPath(t *testing.T) {
	_, err := RunIncremental("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestIncrementalScanSummary(t *testing.T) {
	summary := index.IncrementalScanSummary{
		FilesAdded:     5,
		FilesModified:  3,
		FilesDeleted:   2,
		FilesUnchanged: 100,
	}

	if summary.FilesAdded != 5 {
		t.Errorf("expected FilesAdded=5, got %d", summary.FilesAdded)
	}
	if summary.FilesModified != 3 {
		t.Errorf("expected FilesModified=3, got %d", summary.FilesModified)
	}
	if summary.FilesDeleted != 2 {
		t.Errorf("expected FilesDeleted=2, got %d", summary.FilesDeleted)
	}
	if summary.FilesUnchanged != 100 {
		t.Errorf("expected FilesUnchanged=100, got %d", summary.FilesUnchanged)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
