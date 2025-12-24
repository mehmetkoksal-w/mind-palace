package integration_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const sqliteSignature = "SQLite format 3\x00"

func TestPalaceLifecycle(t *testing.T) {
	repoRoot := repoRoot(t)
	binPath := filepath.Join(t.TempDir(), "palace")
	buildPalace(t, repoRoot, binPath)

	workspace := t.TempDir()
	seedPath := filepath.Join(workspace, "notes.txt")
	if err := os.WriteFile(seedPath, []byte("hello world\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	runPalace(t, binPath, workspace, "init", "--root", workspace)
	runPalace(t, binPath, workspace, "scan", "--root", workspace)

	dbPath := filepath.Join(workspace, ".palace", "index", "palace.db")
	assertSQLiteFile(t, dbPath)

	runPalace(t, binPath, workspace, "ci", "collect", "--root", workspace)

	contextPath := filepath.Join(workspace, ".palace", "outputs", "context-pack.json")
	data, err := os.ReadFile(contextPath)
	if err != nil {
		t.Fatalf("read context pack: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode context pack: %v", err)
	}
	scanHash, ok := payload["scanHash"].(string)
	if !ok || scanHash == "" {
		t.Fatalf("expected scanHash in context pack")
	}

	runPalace(t, binPath, workspace, "ci", "verify", "--root", workspace)

	if err := os.WriteFile(seedPath, []byte("hello world\nupdated\n"), 0o644); err != nil {
		t.Fatalf("modify seed file: %v", err)
	}

	output := runPalaceExpectFail(t, binPath, workspace, "ci", "verify", "--root", workspace)
	if !bytes.Contains([]byte(output), []byte("stale")) {
		t.Fatalf("expected stale verification output, got: %s", output)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// tests/ is at apps/cli/tests/, so go up 3 levels to repo root
	return filepath.Dir(filepath.Dir(filepath.Dir(cwd)))
}

func buildPalace(t *testing.T, repoRoot, binPath string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", binPath, "./apps/cli")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build palace: %v\n%s", err, string(out))
	}
}

func runPalace(t *testing.T, binPath, dir string, args ...string) string {
	t.Helper()
	out, err := runCommand(binPath, dir, args...)
	if err != nil {
		t.Fatalf("palace %v failed: %v\n%s", args, err, string(out))
	}
	return string(out)
}

func runPalaceExpectFail(t *testing.T, binPath, dir string, args ...string) string {
	t.Helper()
	out, err := runCommand(binPath, dir, args...)
	if err == nil {
		t.Fatalf("expected palace %v to fail", args)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected exit error for palace %v: %v", args, err)
	}
	return string(out)
}

func runCommand(binPath, dir string, args ...string) ([]byte, error) {
	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

func assertSQLiteFile(t *testing.T, path string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open sqlite file: %v", err)
	}
	defer file.Close()

	header := make([]byte, len(sqliteSignature))
	if _, err := io.ReadFull(file, header); err != nil {
		t.Fatalf("read sqlite header: %v", err)
	}
	if string(header) != sqliteSignature {
		t.Fatalf("unexpected sqlite header: %q", string(header))
	}
}
