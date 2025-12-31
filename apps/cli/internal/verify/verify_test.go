package verify

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/fsutil"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
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

func TestVerifyFullScan(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := config.EnsureLayout(dir); err != nil {
		t.Fatalf("layout: %v", err)
	}
	dbPath := filepath.Join(dir, ".palace", "index", "palace.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// Initial scan - USE LOADED GUARDRAILS instead of empty
	guardrails := config.LoadGuardrails(dir)
	records, _ := index.BuildFileRecords(dir, guardrails)
	index.WriteScan(db, dir, records, time.Now())

	// Run verify with empty diff (should trigger full scan)
	staleList, fullScope, source, count, err := Run(db, Options{Root: dir, DiffRange: "", Mode: ModeFast})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !fullScope {
		t.Error("expected full scope for empty diff range")
	}
	if source != "full-scan" {
		t.Errorf("expected source 'full-scan', got %q", source)
	}
	if count != 1 {
		t.Errorf("expected 1 candidate, got %d", count)
	}
	if len(staleList) != 0 {
		t.Errorf("expected 0 stale files, got %d: %v", len(staleList), staleList)
	}
}

func TestVerifyDetectsStale(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := config.EnsureLayout(dir); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, ".palace", "index", "palace.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	guardrails := config.LoadGuardrails(dir)
	records, _ := index.BuildFileRecords(dir, guardrails)
	index.WriteScan(db, dir, records, time.Now())

	// Modify file and set mod time back or forward by a minute to bypass second truncation
	if err := os.WriteFile(path, []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}
	newTime := time.Now().Add(1 * time.Minute)
	if err := os.Chtimes(path, newTime, newTime); err != nil {
		t.Fatal(err)
	}

	staleList, _, _, _, err := Run(db, Options{Root: dir, DiffRange: "", Mode: ModeFast})
	if err != nil {
		t.Fatal(err)
	}

	if len(staleList) != 1 || !strings.Contains(staleList[0], "changed file file.txt") {
		t.Errorf("expected file.txt to be stale, got %v", staleList)
	}
}

func TestVerifyStrictMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := config.EnsureLayout(dir); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, ".palace", "index", "palace.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	guardrails := config.LoadGuardrails(dir)
	records, _ := index.BuildFileRecords(dir, guardrails)
	index.WriteScan(db, dir, records, time.Now())

	staleList, _, _, _, err := Run(db, Options{Root: dir, DiffRange: "", Mode: ModeStrict})
	if err != nil {
		t.Fatal(err)
	}

	if len(staleList) != 0 {
		t.Errorf("expected 0 stale files in strict mode, got %v", staleList)
	}
}

func TestSourceFrom(t *testing.T) {
	if sourceFrom(true) != "change-signal" {
		t.Errorf("expected change-signal, got %s", sourceFrom(true))
	}
	if sourceFrom(false) != "git-diff" {
		t.Errorf("expected git-diff, got %s", sourceFrom(false))
	}
}

func TestVerifyInvalidRoot(t *testing.T) {
	// Create an empty DB so LoadFileMetadata works but Run fails later due to invalid path
	dir := t.TempDir()
	db, err := index.Open(filepath.Join(dir, "empty.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, _, _, _, err = Run(db, Options{Root: "/non/existent/path/mind/palace/test"})
	if err == nil {
		t.Error("expected error for invalid root")
	}
}

func TestVerifyWithInvalidDiff(t *testing.T) {
	dir := t.TempDir()
	if _, err := config.EnsureLayout(dir); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, ".palace", "index", "palace.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Invalid diff range should return error
	_, _, _, _, err = Run(db, Options{Root: dir, DiffRange: "INVALID..DIFF"})
	if err == nil {
		t.Error("expected error for invalid diff range")
	}
}
