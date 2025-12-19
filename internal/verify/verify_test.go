package verify

import (
	"os"
	"os/exec"
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

	staleList := detectStale(dir, []string{"file.txt"}, stored, config.Guardrails{}, ModeFast, true)
	if len(staleList) != 0 {
		t.Fatalf("expected no stale entries, got %v", staleList)
	}
}

func TestVerifyEmptyDiffDoesNotFallback(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v output: %s", args, err, string(out))
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "tester")

	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("add", "file.txt")
	run("commit", "-m", "initial")
	run("commit", "--allow-empty", "-m", "noop")

	if _, err := config.EnsureLayout(dir); err != nil {
		t.Fatalf("layout: %v", err)
	}
	records, err := index.BuildFileRecords(dir, config.LoadGuardrails(dir))
	if err != nil {
		t.Fatalf("build records: %v", err)
	}
	dbPath := filepath.Join(dir, ".palace", "index", "palace.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if _, err := index.WriteScan(db, dir, records, time.Now()); err != nil {
		t.Fatalf("write scan: %v", err)
	}

	stale, fallback, _, count, err := Run(db, Options{Root: dir, DiffRange: "HEAD..HEAD", Mode: ModeFast})
	if err != nil {
		t.Fatalf("verify run: %v", err)
	}
	if fallback {
		t.Fatalf("expected no fallback for empty diff")
	}
	if count != 0 {
		t.Fatalf("expected zero candidates for empty diff, got %d", count)
	}
	if len(stale) != 0 {
		t.Fatalf("expected no stale for empty diff, got %v", stale)
	}
}
