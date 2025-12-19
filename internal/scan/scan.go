package scan

import (
	"path/filepath"
	"time"

	"mind-palace/internal/config"
	"mind-palace/internal/index"
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

	summary, err := index.WriteScan(db, rootPath, records, time.Now())
	if err != nil {
		return index.ScanSummary{}, 0, err
	}
	return summary, len(records), nil
}
