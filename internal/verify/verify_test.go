package verify

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"mind-palace/internal/config"
	"mind-palace/internal/fsutil"
	"mind-palace/internal/index"
)

func TestDetectStaleIgnoresUnchangedFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	stat, err := fsutil.StatFile(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	hash, err := fsutil.HashFile(path)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	stored := map[string]index.FileMetadata{
		"file.txt": {
			Hash:    hash,
			Size:    stat.Size,
			ModTime: stat.ModTime,
		},
	}

	// Ensure timestamps after write are normalized the same way.
	time.Sleep(10 * time.Millisecond)
	stale := detectStale(dir, []string{"file.txt"}, stored, config.Guardrails{}, ModeFast, true)
	if len(stale) != 0 {
		t.Fatalf("expected no stale entries, got %v", stale)
	}
}
