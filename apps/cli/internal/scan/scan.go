package scan

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/validate"
)

// resolveAndValidateRoot converts a path to absolute and verifies it exists.
func resolveAndValidateRoot(root string) (string, error) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return "", fmt.Errorf("directory does not exist: %s", rootPath)
	}
	return rootPath, nil
}

// RunIncremental performs an incremental scan, only processing changed files.
// Returns the number of changes applied and an error if any.
// If there are no changes, returns (0, nil).
func RunIncremental(root string) (index.IncrementalScanSummary, error) {
	rootPath, err := resolveAndValidateRoot(root)
	if err != nil {
		return index.IncrementalScanSummary{}, err
	}

	// Check if index exists - if not, need full scan
	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return index.IncrementalScanSummary{}, fmt.Errorf("no index found; run 'palace scan --full' first")
	}

	guardrails := config.LoadGuardrails(rootPath)

	db, err := index.Open(dbPath)
	if err != nil {
		return index.IncrementalScanSummary{}, err
	}
	defer db.Close()

	// Detect changes
	changes, err := index.DetectChanges(db, rootPath, guardrails)
	if err != nil {
		return index.IncrementalScanSummary{}, fmt.Errorf("detect changes: %w", err)
	}

	if len(changes) == 0 {
		// Count unchanged files
		var count int
		if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM files").Scan(&count); err != nil {
			return index.IncrementalScanSummary{}, fmt.Errorf("count files: %w", err)
		}
		return index.IncrementalScanSummary{
			FilesUnchanged: count,
		}, nil
	}

	// Count files before changes to calculate unchanged correctly
	var initialCount int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM files").Scan(&initialCount); err != nil {
		return index.IncrementalScanSummary{}, fmt.Errorf("count initial files: %w", err)
	}

	// Apply incremental changes
	summary, err := index.IncrementalScan(db, rootPath, changes)
	if err != nil {
		return summary, fmt.Errorf("incremental scan: %w", err)
	}

	// Calculate unchanged files:
	// Unchanged = (files that existed before) - (files that were modified) - (files that were deleted)
	// Note: Added files don't affect unchanged count since they weren't there before
	summary.FilesUnchanged = initialCount - summary.FilesModified - summary.FilesDeleted

	return summary, nil
}

// Run performs a full scan of the workspace
func Run(root string) (index.ScanSummary, int, error) {
	rootPath, err := resolveAndValidateRoot(root)
	if err != nil {
		return index.ScanSummary{}, 0, err
	}

	if _, err := config.EnsureLayout(rootPath); err != nil {
		return index.ScanSummary{}, 0, err
	}

	guardrails := config.LoadGuardrails(rootPath)
	startedAt := time.Now().UTC()

	records, err := index.BuildFileRecords(rootPath, guardrails)
	if err != nil {
		return index.ScanSummary{}, 0, err
	}

	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	db, err := index.Open(dbPath)
	if err != nil {
		return index.ScanSummary{}, 0, err
	}
	defer db.Close()

	summary, err := index.WriteScan(db, rootPath, records, startedAt)
	if err != nil {
		return index.ScanSummary{}, 0, err
	}

	scanArtifactPath := filepath.Join(rootPath, ".palace", "index", "scan.json")
	now := time.Now().UTC().Format(time.RFC3339)
	artifact := model.ScanSummary{
		SchemaVersion:     "1.0.0",
		Kind:              "palace/scan",
		ScanID:            uuid.NewString(),
		DBScanID:          summary.ID,
		StartedAt:         summary.StartedAt.UTC().Format(time.RFC3339),
		CompletedAt:       summary.CompletedAt.UTC().Format(time.RFC3339),
		FileCount:         summary.FileCount,
		ChunkCount:        summary.ChunkCount,
		SymbolCount:       summary.SymbolCount,
		RelationshipCount: summary.RelationshipCount,
		ScanHash:          summary.ScanHash,
		Provenance: model.Provenance{
			CreatedBy: "palace scan",
			CreatedAt: now,
		},
	}

	if err := model.WriteScanSummary(scanArtifactPath, artifact); err != nil {
		return index.ScanSummary{}, 0, err
	}
	if err := validate.JSON(scanArtifactPath, "scan"); err != nil {
		return index.ScanSummary{}, 0, err
	}

	return summary, len(records), nil
}
