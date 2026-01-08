package collect

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

func TestCollectEntryPoints(t *testing.T) {
	root := t.TempDir()
	roomDir := filepath.Join(root, ".palace", "rooms")
	if err := os.MkdirAll(roomDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	room := `{
  "schemaVersion":"1.0.0",
  "kind":"palace/room",
  "name":"core",
  "summary":"Core room",
  "entryPoints":["src/main.go","lib/util.go"],
  "provenance":{"createdBy":"test","createdAt":"2024-01-01T00:00:00Z"}
}`
	if err := os.WriteFile(filepath.Join(roomDir, "core.jsonc"), []byte(room), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	entries := collectEntryPoints(root, "core")
	if len(entries) != 2 {
		t.Fatalf("entries length = %d, want 2", len(entries))
	}
	if entries[0] != "src/main.go" {
		t.Fatalf("entry[0] = %q, want %q", entries[0], "src/main.go")
	}
}

func TestFilterExisting(t *testing.T) {
	stored := map[string]index.FileMetadata{
		"a.go": {Hash: "a", Size: 1, ModTime: time.Now()},
	}
	paths := []string{"a.go", "b.go"}
	out := filterExisting(paths, stored)
	if len(out) != 1 || out[0] != "a.go" {
		t.Fatalf("filterExisting() = %v, want [a.go]", out)
	}
}

func TestMergeOrderedUnique(t *testing.T) {
	out := mergeOrderedUnique([]string{"a", "b", "a"}, []string{"b", "c", ""})
	if len(out) != 3 {
		t.Fatalf("mergeOrderedUnique() length = %d, want 3", len(out))
	}
	if out[0] != "a" || out[1] != "b" || out[2] != "c" {
		t.Fatalf("mergeOrderedUnique() = %v, want [a b c]", out)
	}
}

func TestPrioritizeHits(t *testing.T) {
	hits := []index.ChunkHit{
		{Path: "a.go"},
		{Path: "b.go"},
		{Path: "c.go"},
	}
	out := prioritizeHits(hits, []string{"b.go"})
	if out[0].Path != "b.go" {
		t.Fatalf("prioritizeHits()[0] = %s, want b.go", out[0].Path)
	}
}

func TestRunFullScopeAllowStale(t *testing.T) {
	root := t.TempDir()
	if _, err := config.EnsureLayout(root); err != nil {
		t.Fatalf("EnsureLayout() error = %v", err)
	}

	if err := config.WriteTemplate(filepath.Join(root, ".palace", "palace.jsonc"), "palace.jsonc", map[string]string{
		"projectName": "test",
		"language":    "go",
	}, true); err != nil {
		t.Fatalf("WriteTemplate(palace) error = %v", err)
	}
	if err := config.WriteTemplate(filepath.Join(root, ".palace", "rooms", "project-overview.jsonc"), "rooms/project-overview.jsonc", nil, true); err != nil {
		t.Fatalf("WriteTemplate(room) error = %v", err)
	}

	mainPath := filepath.Join(root, "main.go")
	if err := os.WriteFile(mainPath, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	dbPath := filepath.Join(root, ".palace", "index", "palace.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("index.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.ExecContext(context.Background(), `INSERT INTO files (path, hash, size, mod_time, indexed_at, language) VALUES (?, ?, ?, ?, ?, ?)`,
		"main.go", "hash", 1, now, now, "go"); err != nil {
		t.Fatalf("insert file error = %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO scans (root, scan_hash, started_at, completed_at) VALUES (?, ?, ?, ?)`,
		root, "scanhash", now, now); err != nil {
		t.Fatalf("insert scan error = %v", err)
	}

	result, err := Run(root, "", Options{AllowStale: true})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.ContextPack.Scope == nil || result.ContextPack.Scope.FileCount == 0 {
		t.Fatalf("expected scope file count > 0")
	}
}

func TestRunDiffScopeFromSignalWithCorridorWarning(t *testing.T) {
	root := t.TempDir()
	if _, err := config.EnsureLayout(root); err != nil {
		t.Fatalf("EnsureLayout() error = %v", err)
	}

	palaceConfig := `{
  "schemaVersion": "1.0.0",
  "kind": "palace/config",
  "project": {
    "name": "test",
    "description": "",
    "language": "go",
    "repository": ""
  },
  "defaultRoom": "project-overview",
  "guardrails": {},
  "neighbors": {
    "missing": {
      "localPath": "missing-neighbor"
    }
  },
  "provenance": {
    "createdBy": "test",
    "createdAt": "2024-01-01T00:00:00Z"
  }
}`
	if err := os.WriteFile(filepath.Join(root, ".palace", "palace.jsonc"), []byte(palaceConfig), 0o644); err != nil {
		t.Fatalf("WriteFile(palace.jsonc) error = %v", err)
	}
	if err := config.WriteTemplate(filepath.Join(root, ".palace", "rooms", "project-overview.jsonc"), "rooms/project-overview.jsonc", map[string]string{}, true); err != nil {
		t.Fatalf("WriteTemplate(room) error = %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile(README) error = %v", err)
	}

	dbPath := filepath.Join(root, ".palace", "index", "palace.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("index.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.ExecContext(context.Background(), `INSERT INTO files (path, hash, size, mod_time, indexed_at, language) VALUES (?, ?, ?, ?, ?, ?)`,
		"README.md", "hash", 1, now, now, "md"); err != nil {
		t.Fatalf("insert file error = %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO scans (root, scan_hash, started_at, completed_at) VALUES (?, ?, ?, ?)`,
		root, "scanhash", now, now); err != nil {
		t.Fatalf("insert scan error = %v", err)
	}

	sig := model.ChangeSignal{
		SchemaVersion: "1.0.0",
		Kind:          "palace/change-signal",
		DiffRange:     "HEAD~1..HEAD",
		GeneratedAt:   now,
		Changes: []model.Change{
			{Path: "README.md", Status: "modified", Hash: "hash"},
		},
		Provenance: model.Provenance{
			CreatedBy: "palace signal",
			CreatedAt: now,
		},
	}
	sigPath := filepath.Join(root, ".palace", "outputs", "change-signal.json")
	if err := model.WriteChangeSignal(sigPath, sig); err != nil {
		t.Fatalf("WriteChangeSignal() error = %v", err)
	}

	result, err := Run(root, "HEAD~1..HEAD", Options{AllowStale: true})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.ContextPack.Scope == nil || result.ContextPack.Scope.Mode != "diff" {
		t.Fatalf("expected diff scope, got %+v", result.ContextPack.Scope)
	}
	if result.ContextPack.Scope.Source != "change-signal" {
		t.Fatalf("expected change-signal source, got %q", result.ContextPack.Scope.Source)
	}
	if len(result.CorridorWarnings) == 0 {
		t.Fatalf("expected corridor warning for missing neighbor")
	}
}
