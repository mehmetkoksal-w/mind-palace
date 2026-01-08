package model_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

func TestWriteAndLoadChangeSignal(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "change-signal.json")

	original := model.ChangeSignal{
		SchemaVersion: "1.0.0",
		Kind:          "palace/change-signal",
		DiffRange:     "HEAD~1..HEAD",
		GeneratedAt:   "2024-01-01T00:00:00Z",
		Changes: []model.Change{
			{Path: "file1.go", Status: "modified", Hash: "abc123"},
			{Path: "file2.go", Status: "added"},
		},
		RequiredArtifacts: []string{"scan.json"},
		Provenance: model.Provenance{
			CreatedBy: "test",
			CreatedAt: "2024-01-01T00:00:00Z",
		},
	}

	// Write
	if err := model.WriteChangeSignal(path, original); err != nil {
		t.Fatalf("WriteChangeSignal failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("change signal file was not created")
	}

	// Load
	loaded, err := model.LoadChangeSignal(path)
	if err != nil {
		t.Fatalf("LoadChangeSignal failed: %v", err)
	}

	// Verify contents
	if loaded.DiffRange != original.DiffRange {
		t.Errorf("diffRange mismatch: got %q, want %q", loaded.DiffRange, original.DiffRange)
	}
	if len(loaded.Changes) != len(original.Changes) {
		t.Errorf("changes length mismatch: got %d, want %d", len(loaded.Changes), len(original.Changes))
	}
	if loaded.Changes[0].Path != original.Changes[0].Path {
		t.Errorf("first change path mismatch: got %q, want %q", loaded.Changes[0].Path, original.Changes[0].Path)
	}
}

func TestLoadChangeSignalNotFound(t *testing.T) {
	_, err := model.LoadChangeSignal("/nonexistent/change-signal.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoadChangeSignalInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(path, []byte("not valid json {"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := model.LoadChangeSignal(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestChangeSignalNormalizesNilChanges(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "change-signal.json")

	// Create with nil changes
	sig := model.ChangeSignal{
		SchemaVersion: "1.0.0",
		Kind:          "palace/change-signal",
		DiffRange:     "HEAD~1..HEAD",
		// Changes is nil
	}

	if err := model.WriteChangeSignal(path, sig); err != nil {
		t.Fatalf("WriteChangeSignal failed: %v", err)
	}

	// Load and verify changes is empty array, not nil
	loaded, err := model.LoadChangeSignal(path)
	if err != nil {
		t.Fatalf("LoadChangeSignal failed: %v", err)
	}

	if loaded.Changes == nil {
		t.Error("Changes should be empty slice, not nil")
	}
}

func TestChangeType(t *testing.T) {
	change := model.Change{
		Path:   "test/file.go",
		Status: "modified",
		Hash:   "sha256:abc123",
	}

	if change.Path != "test/file.go" {
		t.Error("Change path not set correctly")
	}
	if change.Status != "modified" {
		t.Error("Change status not set correctly")
	}
	if change.Hash != "sha256:abc123" {
		t.Error("Change hash not set correctly")
	}
}

func TestWriteChangeSignalError(t *testing.T) {
	sig := model.ChangeSignal{
		SchemaVersion: "1.0.0",
		Kind:          "palace/change-signal",
	}

	// Write to invalid path
	err := model.WriteChangeSignal("/nonexistent/dir/change-signal.json", sig)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}
