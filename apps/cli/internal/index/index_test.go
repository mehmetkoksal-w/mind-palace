package index

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/fsutil"
)

func TestSearchChunksHandlesSpaces(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "palace.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	content := "hello world\nsecond line"
	sum := sha256.Sum256([]byte(content))
	records := []FileRecord{
		{
			Path:    "file.txt",
			Hash:    fmt.Sprintf("%x", sum[:]),
			Size:    int64(len(content)),
			ModTime: fsutil.NormalizeModTime(time.Now()),
			Chunks:  fsutil.ChunkContent(content, 120, 8*1024),
		},
	}

	if _, err := WriteScan(db, dir, records, time.Now()); err != nil {
		t.Fatalf("write scan: %v", err)
	}

	hits, err := SearchChunks(db, "hello world", 5)
	if err != nil {
		t.Fatalf("search chunks: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if hits[0].Path != "file.txt" {
		t.Fatalf("unexpected path %s", hits[0].Path)
	}
}

// Database Open tests
func TestOpen(t *testing.T) {
	t.Run("creates new database", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "test.db")

		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		// Verify file was created
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("database file was not created")
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "subdir", "nested", "test.db")

		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("database file was not created in nested directory")
		}
	})

	t.Run("opens existing database", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "test.db")

		// Create database
		db1, err := Open(dbPath)
		if err != nil {
			t.Fatalf("first Open failed: %v", err)
		}
		db1.Close()

		// Re-open
		db2, err := Open(dbPath)
		if err != nil {
			t.Fatalf("second Open failed: %v", err)
		}
		defer db2.Close()
	})
}

// WriteScan tests
func TestWriteScan(t *testing.T) {
	t.Run("writes file records", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "palace.db")
		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		content := "package main\n\nfunc main() {}\n"
		sum := sha256.Sum256([]byte(content))
		records := []FileRecord{
			{
				Path:    "main.go",
				Hash:    fmt.Sprintf("%x", sum[:]),
				Size:    int64(len(content)),
				ModTime: fsutil.NormalizeModTime(time.Now()),
				Chunks:  fsutil.ChunkContent(content, 120, 8*1024),
			},
		}

		summary, err := WriteScan(db, dir, records, time.Now())
		if err != nil {
			t.Fatalf("WriteScan failed: %v", err)
		}

		if summary.FileCount != 1 {
			t.Errorf("expected FileCount=1, got %d", summary.FileCount)
		}
	})

	t.Run("empty records", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "palace.db")
		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		summary, err := WriteScan(db, dir, []FileRecord{}, time.Now())
		if err != nil {
			t.Fatalf("WriteScan with empty records failed: %v", err)
		}

		if summary.FileCount != 0 {
			t.Errorf("expected FileCount=0, got %d", summary.FileCount)
		}
	})
}

// LatestScan tests
func TestLatestScan(t *testing.T) {
	t.Run("returns scan summary", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "palace.db")
		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		content := "package main\n"
		sum := sha256.Sum256([]byte(content))
		records := []FileRecord{
			{
				Path:    "main.go",
				Hash:    fmt.Sprintf("%x", sum[:]),
				Size:    int64(len(content)),
				ModTime: fsutil.NormalizeModTime(time.Now()),
				Chunks:  fsutil.ChunkContent(content, 120, 8*1024),
			},
		}

		_, err = WriteScan(db, dir, records, time.Now())
		if err != nil {
			t.Fatalf("WriteScan failed: %v", err)
		}

		latest, err := LatestScan(db)
		if err != nil {
			t.Fatalf("LatestScan failed: %v", err)
		}

		// LatestScan returns scan metadata (ID, Root, ScanHash, CompletedAt)
		// not file counts - those come from WriteScan
		if latest.ID == 0 {
			t.Error("expected non-zero scan ID")
		}
		if latest.Root != dir {
			t.Errorf("expected Root=%q, got %q", dir, latest.Root)
		}
	})

	t.Run("returns empty for new database", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "palace.db")
		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		latest, err := LatestScan(db)
		if err != nil {
			t.Fatalf("LatestScan failed: %v", err)
		}

		if latest.ID != 0 {
			t.Errorf("expected ID=0 for new db, got %d", latest.ID)
		}
	})
}

// LoadFileMetadata tests
func TestLoadFileMetadata(t *testing.T) {
	t.Run("loads metadata from database", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "palace.db")
		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		content := "package main\n"
		sum := sha256.Sum256([]byte(content))
		hash := fmt.Sprintf("%x", sum[:])
		modTime := fsutil.NormalizeModTime(time.Now())

		records := []FileRecord{
			{
				Path:    "main.go",
				Hash:    hash,
				Size:    int64(len(content)),
				ModTime: modTime,
				Chunks:  fsutil.ChunkContent(content, 120, 8*1024),
			},
		}

		_, err = WriteScan(db, dir, records, time.Now())
		if err != nil {
			t.Fatalf("WriteScan failed: %v", err)
		}

		metadata, err := LoadFileMetadata(db)
		if err != nil {
			t.Fatalf("LoadFileMetadata failed: %v", err)
		}

		if len(metadata) != 1 {
			t.Errorf("expected 1 file in metadata, got %d", len(metadata))
		}

		meta, ok := metadata["main.go"]
		if !ok {
			t.Fatal("main.go not found in metadata")
		}

		if meta.Hash != hash {
			t.Errorf("expected hash %s, got %s", hash, meta.Hash)
		}

		if meta.Size != int64(len(content)) {
			t.Errorf("expected size %d, got %d", len(content), meta.Size)
		}
	})

	t.Run("empty database returns empty map", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "palace.db")
		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		metadata, err := LoadFileMetadata(db)
		if err != nil {
			t.Fatalf("LoadFileMetadata failed: %v", err)
		}

		if len(metadata) != 0 {
			t.Errorf("expected empty metadata, got %d entries", len(metadata))
		}
	})
}

// DetectChanges tests
func TestDetectChanges(t *testing.T) {
	t.Run("detects new files", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, ".palace", "index", "palace.db")

		// Create palace structure
		os.MkdirAll(filepath.Dir(dbPath), 0755)

		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		// Create a file on disk that's not in the database
		testFile := filepath.Join(dir, "new.go")
		if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		changes, err := DetectChanges(db, dir, config.Guardrails{})
		if err != nil {
			t.Fatalf("DetectChanges failed: %v", err)
		}

		// Should have detected the new file
		var foundNew bool
		for _, c := range changes {
			if c.Path == "new.go" && c.Action == "added" {
				foundNew = true
				break
			}
		}

		if !foundNew {
			t.Errorf("expected to detect new.go as added, got changes: %v", changes)
		}
	})

	t.Run("detects deleted files", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, ".palace", "index", "palace.db")
		os.MkdirAll(filepath.Dir(dbPath), 0755)

		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		// Create and index a file
		testFile := filepath.Join(dir, "todelete.go")
		content := "package main\n"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		sum := sha256.Sum256([]byte(content))
		records := []FileRecord{
			{
				Path:    "todelete.go",
				Hash:    fmt.Sprintf("%x", sum[:]),
				Size:    int64(len(content)),
				ModTime: fsutil.NormalizeModTime(time.Now()),
				Chunks:  fsutil.ChunkContent(content, 120, 8*1024),
			},
		}

		_, err = WriteScan(db, dir, records, time.Now())
		if err != nil {
			t.Fatalf("WriteScan failed: %v", err)
		}

		// Delete the file
		os.Remove(testFile)

		changes, err := DetectChanges(db, dir, config.Guardrails{})
		if err != nil {
			t.Fatalf("DetectChanges failed: %v", err)
		}

		var foundDeleted bool
		for _, c := range changes {
			if c.Path == "todelete.go" && c.Action == "deleted" {
				foundDeleted = true
				break
			}
		}

		if !foundDeleted {
			t.Errorf("expected to detect todelete.go as deleted, got changes: %v", changes)
		}
	})

	t.Run("detects modified files", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, ".palace", "index", "palace.db")
		os.MkdirAll(filepath.Dir(dbPath), 0755)

		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		// Create and index a file
		testFile := filepath.Join(dir, "modify.go")
		content := "package main\n"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		sum := sha256.Sum256([]byte(content))
		records := []FileRecord{
			{
				Path:    "modify.go",
				Hash:    fmt.Sprintf("%x", sum[:]),
				Size:    int64(len(content)),
				ModTime: fsutil.NormalizeModTime(time.Now()),
				Chunks:  fsutil.ChunkContent(content, 120, 8*1024),
			},
		}

		_, err = WriteScan(db, dir, records, time.Now())
		if err != nil {
			t.Fatalf("WriteScan failed: %v", err)
		}

		// Modify the file
		newContent := "package main\n\nfunc main() {}\n"
		if err := os.WriteFile(testFile, []byte(newContent), 0644); err != nil {
			t.Fatalf("failed to modify file: %v", err)
		}

		changes, err := DetectChanges(db, dir, config.Guardrails{})
		if err != nil {
			t.Fatalf("DetectChanges failed: %v", err)
		}

		var foundModified bool
		for _, c := range changes {
			if c.Path == "modify.go" && c.Action == "modified" {
				foundModified = true
				break
			}
		}

		if !foundModified {
			t.Errorf("expected to detect modify.go as modified, got changes: %v", changes)
		}
	})

	t.Run("no changes when unchanged", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, ".palace", "index", "palace.db")
		os.MkdirAll(filepath.Dir(dbPath), 0755)

		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		// Create and index a file
		testFile := filepath.Join(dir, "unchanged.go")
		content := "package main\n"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		sum := sha256.Sum256([]byte(content))
		records := []FileRecord{
			{
				Path:    "unchanged.go",
				Hash:    fmt.Sprintf("%x", sum[:]),
				Size:    int64(len(content)),
				ModTime: fsutil.NormalizeModTime(time.Now()),
				Chunks:  fsutil.ChunkContent(content, 120, 8*1024),
			},
		}

		_, err = WriteScan(db, dir, records, time.Now())
		if err != nil {
			t.Fatalf("WriteScan failed: %v", err)
		}

		// Exclude .palace directory from detection
		guardrails := config.Guardrails{
			DoNotTouchGlobs: []string{".palace/**"},
		}

		changes, err := DetectChanges(db, dir, guardrails)
		if err != nil {
			t.Fatalf("DetectChanges failed: %v", err)
		}

		if len(changes) != 0 {
			t.Errorf("expected no changes for unchanged file, got %v", changes)
		}
	})
}

// IncrementalScan tests
func TestIncrementalScan(t *testing.T) {
	t.Run("processes empty changes", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "palace.db")
		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		summary, err := IncrementalScan(db, dir, []FileChange{})
		if err != nil {
			t.Fatalf("IncrementalScan failed: %v", err)
		}

		if summary.FilesAdded != 0 || summary.FilesModified != 0 || summary.FilesDeleted != 0 {
			t.Errorf("expected all zeros for empty changes, got %+v", summary)
		}
	})

	t.Run("handles added files", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "palace.db")
		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		// Create a test file
		testFile := filepath.Join(dir, "added.go")
		content := "package main\n"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		sum := sha256.Sum256([]byte(content))
		changes := []FileChange{
			{
				Path:    "added.go",
				Action:  "added",
				NewHash: fmt.Sprintf("%x", sum[:]),
			},
		}

		summary, err := IncrementalScan(db, dir, changes)
		if err != nil {
			t.Fatalf("IncrementalScan failed: %v", err)
		}

		if summary.FilesAdded != 1 {
			t.Errorf("expected FilesAdded=1, got %d", summary.FilesAdded)
		}
	})

	t.Run("handles deleted files", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "palace.db")
		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		// First add a file
		testFile := filepath.Join(dir, "todelete.go")
		content := "package main\n"
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		sum := sha256.Sum256([]byte(content))
		records := []FileRecord{
			{
				Path:    "todelete.go",
				Hash:    fmt.Sprintf("%x", sum[:]),
				Size:    int64(len(content)),
				ModTime: fsutil.NormalizeModTime(time.Now()),
				Chunks:  fsutil.ChunkContent(content, 120, 8*1024),
			},
		}

		_, err = WriteScan(db, dir, records, time.Now())
		if err != nil {
			t.Fatalf("WriteScan failed: %v", err)
		}

		// Now delete it via incremental scan
		changes := []FileChange{
			{
				Path:    "todelete.go",
				Action:  "deleted",
				OldHash: fmt.Sprintf("%x", sum[:]),
			},
		}

		summary, err := IncrementalScan(db, dir, changes)
		if err != nil {
			t.Fatalf("IncrementalScan failed: %v", err)
		}

		if summary.FilesDeleted != 1 {
			t.Errorf("expected FilesDeleted=1, got %d", summary.FilesDeleted)
		}
	})
}

// SearchChunks additional tests
func TestSearchChunks(t *testing.T) {
	t.Run("empty query returns no results", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "palace.db")
		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		hits, err := SearchChunks(db, "", 5)
		if err != nil {
			t.Fatalf("SearchChunks failed: %v", err)
		}

		if len(hits) != 0 {
			t.Errorf("expected no hits for empty query, got %d", len(hits))
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "palace.db")
		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		// Create multiple files with same content
		var records []FileRecord
		for i := 0; i < 10; i++ {
			content := "common content here\n"
			sum := sha256.Sum256([]byte(content))
			records = append(records, FileRecord{
				Path:    fmt.Sprintf("file%d.txt", i),
				Hash:    fmt.Sprintf("%x", sum[:]),
				Size:    int64(len(content)),
				ModTime: fsutil.NormalizeModTime(time.Now()),
				Chunks:  fsutil.ChunkContent(content, 120, 8*1024),
			})
		}

		_, err = WriteScan(db, dir, records, time.Now())
		if err != nil {
			t.Fatalf("WriteScan failed: %v", err)
		}

		hits, err := SearchChunks(db, "common", 3)
		if err != nil {
			t.Fatalf("SearchChunks failed: %v", err)
		}

		if len(hits) > 3 {
			t.Errorf("expected at most 3 hits due to limit, got %d", len(hits))
		}
	})
}

// FileChange struct tests
func TestFileChange(t *testing.T) {
	t.Run("added file", func(t *testing.T) {
		change := FileChange{
			Path:    "new.go",
			Action:  "added",
			OldHash: "",
			NewHash: "abc123",
		}

		if change.Action != "added" {
			t.Errorf("expected action 'added', got %q", change.Action)
		}
		if change.OldHash != "" {
			t.Errorf("expected empty OldHash for added file")
		}
	})

	t.Run("deleted file", func(t *testing.T) {
		change := FileChange{
			Path:    "old.go",
			Action:  "deleted",
			OldHash: "abc123",
			NewHash: "",
		}

		if change.Action != "deleted" {
			t.Errorf("expected action 'deleted', got %q", change.Action)
		}
		if change.NewHash != "" {
			t.Errorf("expected empty NewHash for deleted file")
		}
	})

	t.Run("modified file", func(t *testing.T) {
		change := FileChange{
			Path:    "changed.go",
			Action:  "modified",
			OldHash: "abc123",
			NewHash: "def456",
		}

		if change.Action != "modified" {
			t.Errorf("expected action 'modified', got %q", change.Action)
		}
		if change.OldHash == change.NewHash {
			t.Errorf("expected different hashes for modified file")
		}
	})
}

// IncrementalScanSummary struct tests
func TestIncrementalScanSummary(t *testing.T) {
	summary := IncrementalScanSummary{
		FilesAdded:     5,
		FilesModified:  3,
		FilesDeleted:   2,
		FilesUnchanged: 100,
		Duration:       time.Second,
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

// sanitizeFTSQuery tests
func TestSanitizeFTSQuery(t *testing.T) {
	// The function wraps queries in double quotes and escapes internal quotes
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "\"hello\""},
		{"hello world", "\"hello world\""},
		{"hello*", "\"hello*\""},
		{"test\"quote", "\"test\"\"quote\""},   // quotes are doubled for escaping
		{"special@#$chars", "\"special@#$chars\""}, // special chars preserved
		{"", "\"\""},
		{"   spaces   ", "\"spaces\""},  // trimmed then quoted
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFTSQuery(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFTSQuery(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// GetChunksForFile tests
func TestGetChunksForFile(t *testing.T) {
	t.Run("returns chunks for file", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "palace.db")
		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		content := "line 1\nline 2\nline 3\nline 4\nline 5\n"
		sum := sha256.Sum256([]byte(content))
		records := []FileRecord{
			{
				Path:    "chunked.txt",
				Hash:    fmt.Sprintf("%x", sum[:]),
				Size:    int64(len(content)),
				ModTime: fsutil.NormalizeModTime(time.Now()),
				Chunks:  fsutil.ChunkContent(content, 2, 100),
			},
		}

		_, err = WriteScan(db, dir, records, time.Now())
		if err != nil {
			t.Fatalf("WriteScan failed: %v", err)
		}

		chunks, err := GetChunksForFile(db, "chunked.txt")
		if err != nil {
			t.Fatalf("GetChunksForFile failed: %v", err)
		}

		if len(chunks) == 0 {
			t.Error("expected at least one chunk")
		}

		for _, chunk := range chunks {
			if chunk.Content == "" {
				t.Error("chunk content should not be empty")
			}
			if chunk.StartLine < 1 {
				t.Error("chunk StartLine should be at least 1")
			}
		}
	})

	t.Run("returns empty for nonexistent file", func(t *testing.T) {
		dir := t.TempDir()
		dbPath := filepath.Join(dir, "palace.db")
		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer db.Close()

		chunks, err := GetChunksForFile(db, "nonexistent.txt")
		if err != nil {
			t.Fatalf("GetChunksForFile failed: %v", err)
		}

		if len(chunks) != 0 {
			t.Errorf("expected no chunks for nonexistent file, got %d", len(chunks))
		}
	})
}

// FileMetadata struct tests
func TestFileMetadata(t *testing.T) {
	meta := FileMetadata{
		Hash:    "abc123",
		Size:    1024,
		ModTime: time.Now(),
	}

	if meta.Hash != "abc123" {
		t.Errorf("expected hash 'abc123', got %q", meta.Hash)
	}
	if meta.Size != 1024 {
		t.Errorf("expected size 1024, got %d", meta.Size)
	}
}

// ScanSummary struct tests
func TestScanSummary(t *testing.T) {
	summary := ScanSummary{
		FileCount:   100,
		SymbolCount: 500,
		ChunkCount:  200,
	}

	if summary.FileCount != 100 {
		t.Errorf("expected FileCount=100, got %d", summary.FileCount)
	}
	if summary.SymbolCount != 500 {
		t.Errorf("expected SymbolCount=500, got %d", summary.SymbolCount)
	}
	if summary.ChunkCount != 200 {
		t.Errorf("expected ChunkCount=200, got %d", summary.ChunkCount)
	}
}

// ChunkHit tests
func TestChunkHit(t *testing.T) {
	hit := ChunkHit{
		Path:       "file.go",
		ChunkIndex: 2,
		StartLine:  10,
		EndLine:    20,
		Content:    "func main() {}",
	}

	if hit.Path != "file.go" {
		t.Errorf("expected path 'file.go', got %q", hit.Path)
	}
	if hit.ChunkIndex != 2 {
		t.Errorf("expected ChunkIndex=2, got %d", hit.ChunkIndex)
	}
	if hit.StartLine != 10 || hit.EndLine != 20 {
		t.Errorf("expected lines 10-20, got %d-%d", hit.StartLine, hit.EndLine)
	}
	if !strings.Contains(hit.Content, "func") {
		t.Error("expected content to contain 'func'")
	}
}
