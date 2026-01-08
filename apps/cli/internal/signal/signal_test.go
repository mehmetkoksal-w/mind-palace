package signal

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/fsutil"
)

func TestGenerateChangeSignal(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.CommandContext(context.Background(), "git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v output: %s", args, err, string(out))
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "tester")

	fpath := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(fpath, []byte("one"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("add", "file.txt")
	run("commit", "-m", "initial")

	if err := os.WriteFile(fpath, []byte("two"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("add", "file.txt")
	run("commit", "-m", "update")

	sig, err := Generate(dir, "HEAD~1..HEAD")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(sig.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(sig.Changes))
	}
	change := sig.Changes[0]
	if change.Status != "modified" || change.Path != "file.txt" {
		t.Fatalf("unexpected change: %+v", change)
	}
	expectedHash, err := fsutil.HashFile(fpath)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if change.Hash != expectedHash {
		t.Fatalf("hash mismatch: %s vs %s", change.Hash, expectedHash)
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		token    string
		expected string
	}{
		{"A", "added"},
		{"M", "modified"},
		{"D", "deleted"},
		{"R100", "modified"}, // Renamed
		{"R050", "modified"}, // Renamed with similarity
		{"C100", "modified"}, // Copied
		{"C050", "modified"}, // Copied with similarity
		{"X", "modified"},    // Unknown defaults to modified
		{"", "modified"},     // Empty defaults to modified
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			result := parseStatus(tt.token)
			if result != tt.expected {
				t.Errorf("parseStatus(%q) = %q, want %q", tt.token, result, tt.expected)
			}
		})
	}
}

func TestGenerateRequiresDiffRange(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty diff range
	_, err := Generate(tmpDir, "")
	if err == nil {
		t.Error("expected error for empty diff range")
	}

	// Whitespace only diff range
	_, err = Generate(tmpDir, "   ")
	if err == nil {
		t.Error("expected error for whitespace-only diff range")
	}
}

func TestGenerateWithAddedFile(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.CommandContext(context.Background(), "git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v output: %s", args, err, string(out))
		}
	}

	// Initialize git repo
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "tester")

	// Initial commit with one file
	fpath := filepath.Join(dir, "initial.txt")
	if err := os.WriteFile(fpath, []byte("initial"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("add", "initial.txt")
	run("commit", "-m", "initial")

	// Add a new file in second commit
	newFile := filepath.Join(dir, "new_file.txt")
	if err := os.WriteFile(newFile, []byte("new content"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("add", "new_file.txt")
	run("commit", "-m", "add new file")

	sig, err := Generate(dir, "HEAD~1..HEAD")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if len(sig.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(sig.Changes))
	}

	if sig.Changes[0].Status != "added" {
		t.Errorf("expected status 'added', got %q", sig.Changes[0].Status)
	}
	if sig.Changes[0].Path != "new_file.txt" {
		t.Errorf("expected path 'new_file.txt', got %q", sig.Changes[0].Path)
	}
}

func TestGenerateWithDeletedFile(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.CommandContext(context.Background(), "git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v output: %s", args, err, string(out))
		}
	}

	// Initialize git repo
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "tester")

	// Initial commit with two files
	f1 := filepath.Join(dir, "keep.txt")
	f2 := filepath.Join(dir, "delete_me.txt")
	if err := os.WriteFile(f1, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(f2, []byte("delete"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "initial")

	// Delete file in second commit
	run("rm", "delete_me.txt")
	run("commit", "-m", "delete file")

	sig, err := Generate(dir, "HEAD~1..HEAD")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if len(sig.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(sig.Changes))
	}

	if sig.Changes[0].Status != "deleted" {
		t.Errorf("expected status 'deleted', got %q", sig.Changes[0].Status)
	}
	if sig.Changes[0].Path != "delete_me.txt" {
		t.Errorf("expected path 'delete_me.txt', got %q", sig.Changes[0].Path)
	}
	// Deleted files should not have a hash
	if sig.Changes[0].Hash != "" {
		t.Errorf("deleted files should have empty hash, got %q", sig.Changes[0].Hash)
	}
}

func TestGenerateWithMultipleChanges(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.CommandContext(context.Background(), "git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v output: %s", args, err, string(out))
		}
	}

	// Initialize git repo
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "tester")

	// Initial commit
	f1 := filepath.Join(dir, "modify.txt")
	f2 := filepath.Join(dir, "delete.txt")
	if err := os.WriteFile(f1, []byte("original"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(f2, []byte("to delete"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "initial")

	// Make multiple changes
	if err := os.WriteFile(f1, []byte("modified"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("rm", "delete.txt")
	f3 := filepath.Join(dir, "added.txt")
	if err := os.WriteFile(f3, []byte("new file"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "multiple changes")

	sig, err := Generate(dir, "HEAD~1..HEAD")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if len(sig.Changes) != 3 {
		t.Fatalf("expected 3 changes, got %d", len(sig.Changes))
	}

	// Changes should be sorted by path
	for i := 0; i < len(sig.Changes)-1; i++ {
		if sig.Changes[i].Path > sig.Changes[i+1].Path {
			t.Errorf("changes not sorted: %s > %s", sig.Changes[i].Path, sig.Changes[i+1].Path)
		}
	}
}

func TestChangeSignalMetadata(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.CommandContext(context.Background(), "git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v output: %s", args, err, string(out))
		}
	}

	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "tester")

	fpath := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(fpath, []byte("one"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "initial")

	if err := os.WriteFile(fpath, []byte("two"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "update")

	sig, err := Generate(dir, "HEAD~1..HEAD")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Check metadata
	if sig.SchemaVersion != "1.0.0" {
		t.Errorf("SchemaVersion = %q, want %q", sig.SchemaVersion, "1.0.0")
	}
	if sig.Kind != "palace/change-signal" {
		t.Errorf("Kind = %q, want %q", sig.Kind, "palace/change-signal")
	}
	if sig.DiffRange != "HEAD~1..HEAD" {
		t.Errorf("DiffRange = %q, want %q", sig.DiffRange, "HEAD~1..HEAD")
	}
	if sig.GeneratedAt == "" {
		t.Error("GeneratedAt should not be empty")
	}
	if sig.Provenance.CreatedBy != "palace signal" {
		t.Errorf("CreatedBy = %q, want %q", sig.Provenance.CreatedBy, "palace signal")
	}
}
func TestPathsFromSignal(t *testing.T) {
	dir := t.TempDir()

	// Setup git repo
	run := func(args ...string) {
		cmd := exec.CommandContext(context.Background(), "git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v output: %s", args, err, string(out))
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "tester")

	path := filepath.Join(dir, "file.txt")
	os.WriteFile(path, []byte("v1"), 0o644)
	run("add", ".")
	run("commit", "-m", "init")
	os.WriteFile(path, []byte("v2"), 0o644)
	run("add", ".")
	run("commit", "-m", "mod")

	// Generate a signal
	diffRange := "HEAD~1..HEAD"
	_, err := Generate(dir, diffRange)
	if err != nil {
		t.Fatal(err)
	}

	// Test retrieving paths from signal
	paths, fromSignal, err := Paths(dir, diffRange, config.Guardrails{})
	if err != nil {
		t.Fatal(err)
	}
	if !fromSignal {
		t.Error("expected paths from signal")
	}
	if len(paths) != 1 || paths[0] != "file.txt" {
		t.Errorf("expected [file.txt], got %v", paths)
	}

	// Test with mismatching diff range (should fallback to git diff)
	_, fromSignal, err = Paths(dir, "HEAD", config.Guardrails{})
	if err != nil {
		t.Fatal(err)
	}
	if fromSignal {
		t.Error("expected paths from git diff, not signal")
	}
}

func TestGenerateErrors(t *testing.T) {
	// Invalid root
	_, err := Generate("/non/existent", "HEAD")
	if err == nil {
		t.Error("expected error for invalid root")
	}
}

func TestPathsFromSignalErrors(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, ".palace", "outputs")
	os.MkdirAll(outDir, 0o755)

	// Create invalid signal file
	sigPath := filepath.Join(outDir, "change-signal.json")
	os.WriteFile(sigPath, []byte("not valid json"), 0o644)

	_, fromSignal, err := Paths(dir, "HEAD", config.Guardrails{})
	if err == nil {
		t.Error("expected error for invalid signal file")
	}
	if !fromSignal {
		t.Error("expected fromSignal=true when file exists but fails load")
	}
}

func TestPathsWithGuardrails(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.CommandContext(context.Background(), "git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v output: %s", args, err, string(out))
		}
	}
	run("init")
	run("config", "user.email", "t@example.com")
	run("config", "user.name", "t")

	// Create an initial commit so HEAD is valid
	os.WriteFile(filepath.Join(dir, "root"), []byte("root"), 0o644)
	run("add", ".")
	run("commit", "-m", "root")

	// Add secret file and commit it
	os.WriteFile(filepath.Join(dir, "secret.key"), []byte("key"), 0o644)
	run("add", ".")
	run("commit", "-m", "secret")

	// Check diff from previous commit (HEAD~1) to HEAD. Expect secret.key, but guardrail ignores it.
	g := config.Guardrails{DoNotTouchGlobs: []string{"*.key"}}
	paths, _, err := Paths(dir, "HEAD~1", g)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Errorf("expected 0 paths, got %v", paths)
	}
}

func TestGenerateWithInvalidDiffRange(t *testing.T) {
	dir := t.TempDir()
	// Initialize repo so it's a valid git root
	exec.CommandContext(context.Background(), "git", "-C", dir, "init").Run()

	// Pass invalid range that git will reject
	_, err := Generate(dir, "INVALID..RANGE")
	if err == nil {
		t.Error("expected error for invalid git diff range")
	}
}
