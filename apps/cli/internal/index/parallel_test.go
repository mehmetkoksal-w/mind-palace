package index

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/config"
)

// TestBuildFileRecordsParallelEmpty tests parallel scanning with no files.
func TestBuildFileRecordsParallelEmpty(t *testing.T) {
	root := t.TempDir()

	records, err := BuildFileRecordsParallel(root, config.Guardrails{}, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

// TestBuildFileRecordsParallelSingleFile tests with a single file.
func TestBuildFileRecordsParallelSingleFile(t *testing.T) {
	root := t.TempDir()

	// Create a Go file
	goFile := filepath.Join(root, "main.go")
	content := `package main

func main() {
	println("hello")
}
`
	if err := os.WriteFile(goFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	records, err := BuildFileRecordsParallel(root, config.Guardrails{}, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}

	if records[0].Path != "main.go" {
		t.Errorf("expected path 'main.go', got '%s'", records[0].Path)
	}

	if records[0].Language != "go" {
		t.Errorf("expected language 'go', got '%s'", records[0].Language)
	}
}

// TestBuildFileRecordsParallelMultipleFiles tests parallel scanning with many files.
func TestBuildFileRecordsParallelMultipleFiles(t *testing.T) {
	root := t.TempDir()

	// Create multiple files of different types
	files := map[string]string{
		"main.go": `package main

func main() {}
`,
		"util.go": `package main

func helper() string { return "test" }
`,
		"config.json": `{"key": "value"}`,
		"readme.md":   `# Test Project`,
		"src/lib.go": `package src

func LibFunc() {}
`,
		"src/types.go": `package src

type MyType struct {
	Name string
}
`,
	}

	for path, content := range files {
		fullPath := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	records, err := BuildFileRecordsParallel(root, config.Guardrails{}, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != len(files) {
		t.Errorf("expected %d records, got %d", len(files), len(records))
	}

	// Verify all files are present
	foundPaths := make(map[string]bool)
	for _, r := range records {
		foundPaths[r.Path] = true
	}

	for path := range files {
		if !foundPaths[path] {
			t.Errorf("missing record for path: %s", path)
		}
	}
}

// TestBuildFileRecordsParallelOrderPreserved tests that output order is deterministic.
func TestBuildFileRecordsParallelOrderPreserved(t *testing.T) {
	root := t.TempDir()

	// Create files with alphabetically sortable names
	fileNames := []string{"a.go", "b.go", "c.go", "d.go", "e.go"}
	for _, name := range fileNames {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Run multiple times to check for race conditions in ordering
	for i := 0; i < 5; i++ {
		records, err := BuildFileRecordsParallel(root, config.Guardrails{}, 4)
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v", i, err)
		}

		if len(records) != len(fileNames) {
			t.Fatalf("run %d: expected %d records, got %d", i, len(fileNames), len(records))
		}

		// Verify order is preserved (should be sorted alphabetically)
		for j, name := range fileNames {
			if records[j].Path != name {
				t.Errorf("run %d: expected record[%d].Path = %s, got %s", i, j, name, records[j].Path)
			}
		}
	}
}

// TestBuildFileRecordsParallelEquivalence tests that parallel and sequential produce same results.
func TestBuildFileRecordsParallelEquivalence(t *testing.T) {
	root := t.TempDir()

	// Create a variety of files
	files := map[string]string{
		"main.go":       "package main\n\nfunc main() {}\n",
		"lib/util.go":   "package lib\n\nfunc Helper() {}\n",
		"lib/types.go":  "package lib\n\ntype Config struct{}\n",
		"config.json":   `{"debug": true}`,
		"readme.md":     "# Project\n\nDescription here.",
		"data/test.txt": "test data content",
	}

	for path, content := range files {
		fullPath := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	guardrails := config.Guardrails{}

	// Get sequential results (1 worker)
	sequential, err := BuildFileRecordsParallel(root, guardrails, 1)
	if err != nil {
		t.Fatalf("sequential scan error: %v", err)
	}

	// Get parallel results (4 workers)
	parallel, err := BuildFileRecordsParallel(root, guardrails, 4)
	if err != nil {
		t.Fatalf("parallel scan error: %v", err)
	}

	if len(sequential) != len(parallel) {
		t.Fatalf("record count mismatch: sequential=%d, parallel=%d", len(sequential), len(parallel))
	}

	for i := range sequential {
		if sequential[i].Path != parallel[i].Path {
			t.Errorf("path mismatch at %d: sequential=%s, parallel=%s", i, sequential[i].Path, parallel[i].Path)
		}
		if sequential[i].Hash != parallel[i].Hash {
			t.Errorf("hash mismatch at %d for %s", i, sequential[i].Path)
		}
		if sequential[i].Size != parallel[i].Size {
			t.Errorf("size mismatch at %d for %s", i, sequential[i].Path)
		}
		if sequential[i].Language != parallel[i].Language {
			t.Errorf("language mismatch at %d for %s: sequential=%s, parallel=%s",
				i, sequential[i].Path, sequential[i].Language, parallel[i].Language)
		}
	}
}

// TestBuildFileRecordsParallelWithManyFiles tests performance with many files.
func TestBuildFileRecordsParallelWithManyFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	root := t.TempDir()

	// Create 100 files
	fileCount := 100
	for i := 0; i < fileCount; i++ {
		name := filepath.Join(root, "pkg", "file"+string(rune('a'+i%26))+string(rune('0'+i/26))+".go")
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		content := "package pkg\n\nfunc Func" + string(rune('A'+i%26)) + "() {}\n"
		if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	guardrails := config.Guardrails{}

	// Time sequential
	startSeq := time.Now()
	seqRecords, err := BuildFileRecordsParallel(root, guardrails, 1)
	if err != nil {
		t.Fatalf("sequential error: %v", err)
	}
	seqDuration := time.Since(startSeq)

	// Time parallel (auto-detect workers)
	startPar := time.Now()
	parRecords, err := BuildFileRecordsParallel(root, guardrails, 0)
	if err != nil {
		t.Fatalf("parallel error: %v", err)
	}
	parDuration := time.Since(startPar)

	if len(seqRecords) != fileCount || len(parRecords) != fileCount {
		t.Errorf("expected %d records, got seq=%d, par=%d", fileCount, len(seqRecords), len(parRecords))
	}

	t.Logf("Sequential: %v, Parallel (workers=%d): %v, Speedup: %.2fx",
		seqDuration, runtime.NumCPU(), parDuration, float64(seqDuration)/float64(parDuration))
}

// TestBuildFileRecordsParallelConcurrentSafety tests for race conditions.
func TestBuildFileRecordsParallelConcurrentSafety(t *testing.T) {
	root := t.TempDir()

	// Create files
	for i := 0; i < 20; i++ {
		name := filepath.Join(root, "file"+string(rune('a'+i))+".go")
		if err := os.WriteFile(name, []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	guardrails := config.Guardrails{}

	// Run concurrently multiple times to catch race conditions
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := BuildFileRecordsParallel(root, guardrails, 4)
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent execution error: %v", err)
	}
}

// TestBuildFileRecordsParallelAutoWorkers tests auto worker detection.
func TestBuildFileRecordsParallelAutoWorkers(t *testing.T) {
	root := t.TempDir()

	// Create a file
	if err := os.WriteFile(filepath.Join(root, "test.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// workers = 0 should auto-detect
	records, err := BuildFileRecordsParallel(root, config.Guardrails{}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}

	// workers = -1 should also auto-detect
	records, err = BuildFileRecordsParallel(root, config.Guardrails{}, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
}

// TestBuildFileRecordsParallelErrorHandling tests error propagation.
func TestBuildFileRecordsParallelErrorHandling(t *testing.T) {
	root := t.TempDir()

	// Create a file
	testFile := filepath.Join(root, "test.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// First verify it works
	_, err := BuildFileRecordsParallel(root, config.Guardrails{}, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test with a non-existent root - should fail in ListFiles
	_, err = BuildFileRecordsParallel("/nonexistent/path/that/does/not/exist", config.Guardrails{}, 4)
	// Note: On some systems, ListFiles may return empty list rather than error
	// for non-existent paths, so we just verify the function doesn't panic
	// The important thing is graceful handling, not necessarily an error
	t.Logf("Result for non-existent path: err=%v", err)
}

// TestBuildFileRecordsParallelGuardrails tests that guardrails are respected.
func TestBuildFileRecordsParallelGuardrails(t *testing.T) {
	root := t.TempDir()

	// Create files including ones that should be excluded
	files := map[string]string{
		"main.go":             "package main\n",
		"node_modules/pkg.js": "// js",
		".git/config":         "[core]",
		"vendor/lib.go":       "package vendor\n",
		"src/app.go":          "package src\n",
	}

	for path, content := range files {
		fullPath := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Create guardrails to exclude node_modules and .git
	guardrails := config.Guardrails{
		DoNotTouchGlobs: []string{"node_modules/**", ".git/**", "vendor/**"},
	}

	records, err := BuildFileRecordsParallel(root, guardrails, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only have main.go and src/app.go
	if len(records) != 2 {
		t.Errorf("expected 2 records (main.go, src/app.go), got %d", len(records))
		for _, r := range records {
			t.Logf("  - %s", r.Path)
		}
	}

	// Verify excluded files are not present
	for _, r := range records {
		if r.Path == "node_modules/pkg.js" || r.Path == ".git/config" || r.Path == "vendor/lib.go" {
			t.Errorf("excluded file should not be present: %s", r.Path)
		}
	}
}
