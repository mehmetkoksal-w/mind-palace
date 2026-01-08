package model_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

func TestWriteAndLoadScanSummary(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "scan.json")

	original := model.ScanSummary{
		SchemaVersion:     "1.0.0",
		Kind:              "palace/scan",
		ScanID:            "scan-uuid-123",
		DBScanID:          42,
		StartedAt:         "2024-01-01T00:00:00Z",
		CompletedAt:       "2024-01-01T00:01:00Z",
		FileCount:         100,
		ChunkCount:        500,
		SymbolCount:       250,
		RelationshipCount: 75,
		ScanHash:          "sha256:abcdef123456",
		Provenance: model.Provenance{
			CreatedBy: "palace scan",
			CreatedAt: "2024-01-01T00:01:00Z",
		},
	}

	// Write
	if err := model.WriteScanSummary(path, original); err != nil {
		t.Fatalf("WriteScanSummary failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("scan summary file was not created")
	}

	// Load
	loaded, err := model.LoadScanSummary(path)
	if err != nil {
		t.Fatalf("LoadScanSummary failed: %v", err)
	}

	// Verify contents
	if loaded.ScanID != original.ScanID {
		t.Errorf("scanId mismatch: got %q, want %q", loaded.ScanID, original.ScanID)
	}
	if loaded.DBScanID != original.DBScanID {
		t.Errorf("dbScanId mismatch: got %d, want %d", loaded.DBScanID, original.DBScanID)
	}
	if loaded.FileCount != original.FileCount {
		t.Errorf("fileCount mismatch: got %d, want %d", loaded.FileCount, original.FileCount)
	}
	if loaded.ChunkCount != original.ChunkCount {
		t.Errorf("chunkCount mismatch: got %d, want %d", loaded.ChunkCount, original.ChunkCount)
	}
	if loaded.SymbolCount != original.SymbolCount {
		t.Errorf("symbolCount mismatch: got %d, want %d", loaded.SymbolCount, original.SymbolCount)
	}
	if loaded.ScanHash != original.ScanHash {
		t.Errorf("scanHash mismatch: got %q, want %q", loaded.ScanHash, original.ScanHash)
	}
	if loaded.RelationshipCount != original.RelationshipCount {
		t.Errorf("relationshipCount mismatch: got %d, want %d", loaded.RelationshipCount, original.RelationshipCount)
	}
	if loaded.Provenance.CreatedBy != original.Provenance.CreatedBy {
		t.Errorf("provenance.createdBy mismatch: got %q, want %q", loaded.Provenance.CreatedBy, original.Provenance.CreatedBy)
	}
}

func TestLoadScanSummaryNotFound(t *testing.T) {
	_, err := model.LoadScanSummary("/nonexistent/scan.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoadScanSummaryInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(path, []byte("not valid json {"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := model.LoadScanSummary(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestScanSummaryType(t *testing.T) {
	summary := model.ScanSummary{
		SchemaVersion:     "1.0.0",
		Kind:              "palace/scan",
		ScanID:            "test-id",
		DBScanID:          1,
		StartedAt:         "2024-01-01T00:00:00Z",
		CompletedAt:       "2024-01-01T00:00:01Z",
		FileCount:         10,
		ChunkCount:        50,
		SymbolCount:       25,
		RelationshipCount: 5,
		ScanHash:          "hash",
	}

	if summary.SchemaVersion != "1.0.0" || summary.Kind != "palace/scan" || summary.ScanID != "test-id" || summary.DBScanID != 1 {
		t.Error("ScanSummary basic fields not set correctly")
	}
	if summary.StartedAt != "2024-01-01T00:00:00Z" || summary.CompletedAt != "2024-01-01T00:00:01Z" {
		t.Error("ScanSummary time fields not set correctly")
	}
	if summary.FileCount != 10 || summary.ChunkCount != 50 || summary.SymbolCount != 25 || summary.RelationshipCount != 5 || summary.ScanHash != "hash" {
		t.Error("ScanSummary metric fields not set correctly")
	}
}

func TestWriteScanSummaryError(t *testing.T) {
	summary := model.ScanSummary{
		SchemaVersion: "1.0.0",
		Kind:          "palace/scan",
		ScanID:        "test-id",
	}

	// Write to invalid path (directory that doesn't exist)
	err := model.WriteScanSummary("/nonexistent/dir/scan.json", summary)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}
