package signal

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/koksalmehmet/mind-palace/internal/fsutil"
)

func TestGenerateChangeSignal(t *testing.T) {
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
