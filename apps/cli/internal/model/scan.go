package model

import (
	"encoding/json"
	"fmt"
	"os"
)

// ScanSummary is the JSON artifact describing a scan.
type ScanSummary struct {
	SchemaVersion     string     `json:"schemaVersion"`
	Kind              string     `json:"kind"`
	ScanID            string     `json:"scanId"`   // UUID
	DBScanID          int64      `json:"dbScanId"` // SQLite scans.id
	StartedAt         string     `json:"startedAt"`
	CompletedAt       string     `json:"completedAt"`
	FileCount         int        `json:"fileCount"`
	ChunkCount        int        `json:"chunkCount"`
	SymbolCount       int        `json:"symbolCount"`
	RelationshipCount int        `json:"relationshipCount"`
	ScanHash          string     `json:"scanHash"`
	Provenance        Provenance `json:"provenance"`
}

// WriteScanSummary writes the summary to disk.
func WriteScanSummary(path string, summary ScanSummary) error {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// LoadScanSummary reads the summary from disk.
func LoadScanSummary(path string) (ScanSummary, error) {
	var s ScanSummary
	data, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("parse %s: %w", path, err)
	}
	return s, nil
}
