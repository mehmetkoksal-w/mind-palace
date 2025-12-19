package index

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"mind-palace/internal/fsutil"
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
