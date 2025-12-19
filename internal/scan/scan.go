package scan

import (
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"mind-palace/internal/config"
	"mind-palace/internal/index"
	"mind-palace/internal/model"
	"mind-palace/internal/validate"
)

// Run executes the Tier 0 scan pipeline.
func Run(root string) (index.ScanSummary, int, error) {
	rootPath, err := filepath.Abs(root)
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

	// Emit scan.json (deterministic artifact describing the scan).
	scanArtifactPath := filepath.Join(rootPath, ".palace", "index", "scan.json")
	now := time.Now().UTC().Format(time.RFC3339)
	artifact := model.ScanSummary{
		SchemaVersion: "1.0.0",
		Kind:          "palace/scan",
		ScanID:        uuid.NewString(),
		DBScanID:      summary.ID,
		StartedAt:     summary.StartedAt.UTC().Format(time.RFC3339),
		CompletedAt:   summary.CompletedAt.UTC().Format(time.RFC3339),
		FileCount:     summary.FileCount,
		ChunkCount:    summary.ChunkCount,
		ScanHash:      summary.ScanHash,
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