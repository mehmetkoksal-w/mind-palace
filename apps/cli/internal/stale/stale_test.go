package stale

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/fsutil"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
)

func TestModeConstants(t *testing.T) {
	if ModeFast != "fast" {
		t.Errorf("expected ModeFast to be 'fast', got %q", ModeFast)
	}
	if ModeStrict != "strict" {
		t.Errorf("expected ModeStrict to be 'strict', got %q", ModeStrict)
	}
}

func TestDetectEmptyCandidates(t *testing.T) {
	stored := map[string]index.FileMetadata{
		"main.go": {Hash: "abc123", Size: 100},
	}

	// Empty candidates, includeMissing=false
	result := Detect("/tmp", []string{}, stored, config.Guardrails{}, ModeFast, false)
	if len(result) != 0 {
		t.Errorf("expected no stale files for empty candidates, got %v", result)
	}

	// Empty candidates, includeMissing=true - should report stored files as missing
	result = Detect("/tmp", []string{}, stored, config.Guardrails{}, ModeFast, true)
	if len(result) != 1 {
		t.Errorf("expected 1 missing file, got %v", result)
	}
}

func TestDetectNilCandidates(t *testing.T) {
	stored := map[string]index.FileMetadata{}
	result := Detect("/tmp", nil, stored, config.Guardrails{}, ModeFast, false)
	if len(result) != 0 {
		t.Errorf("expected no stale files for nil candidates, got %v", result)
	}
}

func TestDetectEmptyPathSkipped(t *testing.T) {
	stored := map[string]index.FileMetadata{}
	result := Detect("/tmp", []string{"", "", ""}, stored, config.Guardrails{}, ModeFast, false)
	if len(result) != 0 {
		t.Errorf("expected empty paths to be skipped, got %v", result)
	}
}

func TestDetectMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	stored := map[string]index.FileMetadata{
		"nonexistent.go": {Hash: "abc123", Size: 100},
	}

	result := Detect(tmpDir, []string{"nonexistent.go"}, stored, config.Guardrails{}, ModeFast, false)

	if len(result) != 1 {
		t.Fatalf("expected 1 stale entry, got %d: %v", len(result), result)
	}
	if !strings.Contains(result[0], "missing file") {
		t.Errorf("expected 'missing file' message, got %q", result[0])
	}
}

func TestDetectNewFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file that's not in stored metadata
	testFile := filepath.Join(tmpDir, "new.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	stored := map[string]index.FileMetadata{} // Empty - file is "new"

	result := Detect(tmpDir, []string{"new.go"}, stored, config.Guardrails{}, ModeFast, false)

	if len(result) != 1 {
		t.Fatalf("expected 1 stale entry, got %d: %v", len(result), result)
	}
	if !strings.Contains(result[0], "new file") {
		t.Errorf("expected 'new file' message, got %q", result[0])
	}
}

func TestDetectChangedFileFastMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(tmpDir, "changed.go")
	content := []byte("package main\nfunc main() {}\n")
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Get actual hash for comparison
	actualHash, _ := fsutil.HashFile(testFile)

	// Stored metadata has different hash (simulating change)
	stored := map[string]index.FileMetadata{
		"changed.go": {
			Hash:    "different_hash",
			Size:    int64(len(content)),
			ModTime: time.Now().Add(-time.Hour), // Different mod time to trigger hash check
		},
	}

	result := Detect(tmpDir, []string{"changed.go"}, stored, config.Guardrails{}, ModeFast, false)

	if actualHash == "different_hash" {
		t.Skip("hash collision, skipping test")
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 stale entry, got %d: %v", len(result), result)
	}
	if !strings.Contains(result[0], "changed file") {
		t.Errorf("expected 'changed file' message, got %q", result[0])
	}
}

func TestDetectChangedFileStrictMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(tmpDir, "strict.go")
	content := []byte("package main\n")
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	stat, _ := os.Stat(testFile)

	// Same size and modtime but different hash - strict mode should still catch it
	stored := map[string]index.FileMetadata{
		"strict.go": {
			Hash:    "wrong_hash",
			Size:    stat.Size(),
			ModTime: stat.ModTime(),
		},
	}

	result := Detect(tmpDir, []string{"strict.go"}, stored, config.Guardrails{}, ModeStrict, false)

	if len(result) != 1 {
		t.Fatalf("expected 1 stale entry, got %d: %v", len(result), result)
	}
	if !strings.Contains(result[0], "changed file") {
		t.Errorf("expected 'changed file' message, got %q", result[0])
	}
}

func TestDetectUnchangedFileFastMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(tmpDir, "unchanged.go")
	content := []byte("package main\n")
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	stat, _ := os.Stat(testFile)
	hash, _ := fsutil.HashFile(testFile)

	// Exact match - should not be reported as stale
	stored := map[string]index.FileMetadata{
		"unchanged.go": {
			Hash:    hash,
			Size:    stat.Size(),
			ModTime: stat.ModTime(),
		},
	}

	result := Detect(tmpDir, []string{"unchanged.go"}, stored, config.Guardrails{}, ModeFast, false)

	if len(result) != 0 {
		t.Errorf("expected no stale files for unchanged file, got %v", result)
	}
}

func TestDetectUnchangedFileStrictMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(tmpDir, "unchanged.go")
	content := []byte("package main\n")
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	hash, _ := fsutil.HashFile(testFile)

	// Hash matches - should not be reported as stale
	stored := map[string]index.FileMetadata{
		"unchanged.go": {
			Hash: hash,
		},
	}

	result := Detect(tmpDir, []string{"unchanged.go"}, stored, config.Guardrails{}, ModeStrict, false)

	if len(result) != 0 {
		t.Errorf("expected no stale files for unchanged file in strict mode, got %v", result)
	}
}

func TestDetectIncludeMissing(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only one file
	testFile := filepath.Join(tmpDir, "exists.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	stat, _ := os.Stat(testFile)
	hash, _ := fsutil.HashFile(testFile)

	// Stored has two files, but only one exists
	stored := map[string]index.FileMetadata{
		"exists.go": {
			Hash:    hash,
			Size:    stat.Size(),
			ModTime: stat.ModTime(),
		},
		"deleted.go": {
			Hash: "abc123",
			Size: 100,
		},
	}

	// Without includeMissing
	result := Detect(tmpDir, []string{"exists.go"}, stored, config.Guardrails{}, ModeFast, false)
	if len(result) != 0 {
		t.Errorf("expected no stale files without includeMissing, got %v", result)
	}

	// With includeMissing
	result = Detect(tmpDir, []string{"exists.go"}, stored, config.Guardrails{}, ModeFast, true)
	if len(result) != 1 {
		t.Fatalf("expected 1 missing file with includeMissing, got %d: %v", len(result), result)
	}
	if !strings.Contains(result[0], "missing file") || !strings.Contains(result[0], "deleted.go") {
		t.Errorf("expected 'missing file deleted.go' message, got %q", result[0])
	}
}

func TestDetectGuardrailsFilter(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files
	for _, name := range []string{"main.go", "node_modules/lib.js", "vendor/dep.go"} {
		path := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	stored := map[string]index.FileMetadata{} // All files are "new"

	guardrails := config.Guardrails{
		DoNotTouchGlobs: []string{"node_modules/**", "vendor/**"},
	}

	candidates := []string{"main.go", "node_modules/lib.js", "vendor/dep.go"}
	result := Detect(tmpDir, candidates, stored, guardrails, ModeFast, false)

	// Only main.go should be reported (others filtered by guardrails)
	if len(result) != 1 {
		t.Fatalf("expected 1 stale entry (main.go), got %d: %v", len(result), result)
	}
	if !strings.Contains(result[0], "main.go") {
		t.Errorf("expected main.go in result, got %q", result[0])
	}
}

func TestDetectPathNormalization(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(tmpDir, "src", "main.go")
	if err := os.MkdirAll(filepath.Dir(testFile), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(testFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	stored := map[string]index.FileMetadata{} // File is "new"

	// Test with forward slash path (normalized)
	candidates := []string{"src/main.go"}
	result := Detect(tmpDir, candidates, stored, config.Guardrails{}, ModeFast, false)

	if len(result) != 1 {
		t.Fatalf("expected 1 stale entry, got %d: %v", len(result), result)
	}
	// Path should appear in the result
	if !strings.Contains(result[0], "src/main.go") {
		t.Errorf("expected path src/main.go in result, got %q", result[0])
	}
}

func TestDetectResultsSorted(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files
	for _, name := range []string{"z.go", "a.go", "m.go"} {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	stored := map[string]index.FileMetadata{} // All are "new"

	candidates := []string{"z.go", "a.go", "m.go"}
	result := Detect(tmpDir, candidates, stored, config.Guardrails{}, ModeFast, false)

	if len(result) != 3 {
		t.Fatalf("expected 3 stale entries, got %d", len(result))
	}

	// Results should be sorted alphabetically
	expected := []string{"new file a.go", "new file m.go", "new file z.go"}
	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("position %d: expected %q, got %q", i, exp, result[i])
		}
	}
}

func TestDetectFastModeSkipsHashWhenSizeAndModTimeMatch(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(tmpDir, "fast.go")
	content := []byte("package main\n")
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Use fsutil.StatFile to get normalized modtime (truncated to seconds)
	fstat, err := fsutil.StatFile(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}

	// Size and modtime match (using normalized time), but hash is wrong
	// Fast mode should NOT detect this as changed (optimization)
	stored := map[string]index.FileMetadata{
		"fast.go": {
			Hash:    "totally_wrong_hash",
			Size:    fstat.Size,
			ModTime: fstat.ModTime,
		},
	}

	result := Detect(tmpDir, []string{"fast.go"}, stored, config.Guardrails{}, ModeFast, false)

	// Fast mode trusts size+modtime, so no change detected
	if len(result) != 0 {
		t.Errorf("expected fast mode to skip hash check when size+modtime match, got %v", result)
	}
}

func TestDetectMultipleIssues(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files
	existingFile := filepath.Join(tmpDir, "existing.go")
	if err := os.WriteFile(existingFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	newFile := filepath.Join(tmpDir, "new.go")
	if err := os.WriteFile(newFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	stat, _ := os.Stat(existingFile)
	hash, _ := fsutil.HashFile(existingFile)

	stored := map[string]index.FileMetadata{
		"existing.go": {Hash: hash, Size: stat.Size(), ModTime: stat.ModTime()},
		"deleted.go":  {Hash: "abc", Size: 50},
	}

	candidates := []string{"existing.go", "new.go", "missing.go"}

	result := Detect(tmpDir, candidates, stored, config.Guardrails{}, ModeFast, true)

	// Should have: missing.go (not on disk), new.go (not in stored), deleted.go (includeMissing)
	if len(result) != 3 {
		t.Errorf("expected 3 issues, got %d: %v", len(result), result)
	}

	// Verify each issue type is present
	var hasMissing, hasNew, hasDeleted bool
	for _, r := range result {
		if strings.Contains(r, "missing.go") && strings.Contains(r, "missing file") {
			hasMissing = true
		}
		if strings.Contains(r, "new.go") && strings.Contains(r, "new file") {
			hasNew = true
		}
		if strings.Contains(r, "deleted.go") && strings.Contains(r, "missing file") {
			hasDeleted = true
		}
	}

	if !hasMissing {
		t.Error("expected missing.go to be reported")
	}
	if !hasNew {
		t.Error("expected new.go to be reported")
	}
	if !hasDeleted {
		t.Error("expected deleted.go to be reported")
	}
}

func TestDetectDuplicateCandidates(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file
	testFile := filepath.Join(tmpDir, "dup.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	stored := map[string]index.FileMetadata{} // File is "new"

	// Same file listed multiple times - code processes each occurrence
	candidates := []string{"dup.go", "dup.go", "dup.go"}
	result := Detect(tmpDir, candidates, stored, config.Guardrails{}, ModeFast, false)

	// Each occurrence is processed (no input deduplication)
	// This documents current behavior - caller should deduplicate if needed
	if len(result) != 3 {
		t.Errorf("expected 3 stale entries for duplicate candidates, got %d: %v", len(result), result)
	}
}

// Benchmark tests
func BenchmarkDetectFastMode(b *testing.B) {
	tmpDir := b.TempDir()
	stored := make(map[string]index.FileMetadata)
	candidates := make([]string, 100)

	for i := 0; i < 100; i++ {
		name := filepath.Join("src", "file"+string(rune('a'+i%26))+".go")
		path := filepath.Join(tmpDir, name)
		os.MkdirAll(filepath.Dir(path), 0o755)
		content := []byte("package main\n// file " + string(rune('a'+i%26)) + "\n")
		os.WriteFile(path, content, 0o644)
		stat, _ := os.Stat(path)
		hash, _ := fsutil.HashFile(path)
		stored[filepath.ToSlash(name)] = index.FileMetadata{
			Hash:    hash,
			Size:    stat.Size(),
			ModTime: stat.ModTime(),
		}
		candidates[i] = name
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Detect(tmpDir, candidates, stored, config.Guardrails{}, ModeFast, false)
	}
}

func BenchmarkDetectStrictMode(b *testing.B) {
	tmpDir := b.TempDir()
	stored := make(map[string]index.FileMetadata)
	candidates := make([]string, 100)

	for i := 0; i < 100; i++ {
		name := filepath.Join("src", "file"+string(rune('a'+i%26))+".go")
		path := filepath.Join(tmpDir, name)
		os.MkdirAll(filepath.Dir(path), 0o755)
		content := []byte("package main\n// file " + string(rune('a'+i%26)) + "\n")
		os.WriteFile(path, content, 0o644)
		hash, _ := fsutil.HashFile(path)
		stored[filepath.ToSlash(name)] = index.FileMetadata{Hash: hash}
		candidates[i] = name
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Detect(tmpDir, candidates, stored, config.Guardrails{}, ModeStrict, false)
	}
}
