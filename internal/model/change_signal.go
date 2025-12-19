package model

import (
	"encoding/json"
	"fmt"
	"os"
)

// Change describes a single file change in a diff.
type Change struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Hash   string `json:"hash,omitempty"`
}

// ChangeSignal represents a deterministic change signal artifact.
type ChangeSignal struct {
	SchemaVersion     string     `json:"schemaVersion"`
	Kind              string     `json:"kind"`
	DiffRange         string     `json:"diffRange"`
	GeneratedAt       string     `json:"generatedAt"`
	Changes           []Change   `json:"changes"`
	RequiredArtifacts []string   `json:"requiredArtifacts,omitempty"`
	Provenance        Provenance `json:"provenance"`
}

// WriteChangeSignal writes the change signal to disk.
func WriteChangeSignal(path string, sig ChangeSignal) error {
	normalizeChangeSignal(&sig)
	data, err := json.MarshalIndent(sig, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// LoadChangeSignal reads a change signal from disk.
func LoadChangeSignal(path string) (ChangeSignal, error) {
	var sig ChangeSignal
	data, err := os.ReadFile(path)
	if err != nil {
		return sig, err
	}
	if err := json.Unmarshal(data, &sig); err != nil {
		return sig, fmt.Errorf("parse %s: %w", path, err)
	}
	normalizeChangeSignal(&sig)
	return sig, nil
}

func normalizeChangeSignal(sig *ChangeSignal) {
	if sig.Changes == nil {
		sig.Changes = []Change{}
	}
}
