package model

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"mind-palace/internal/config"
	"mind-palace/internal/jsonc"
)

// Capability represents a runnable command for a capability.
type Capability struct {
	Command          string            `json:"command"`
	Description      string            `json:"description"`
	WorkingDirectory string            `json:"workingDirectory,omitempty"`
	Env              map[string]string `json:"env,omitempty"`
}

// ProjectProfile describes detected project capabilities and guardrails.
type ProjectProfile struct {
	SchemaVersion string                `json:"schemaVersion"`
	Kind          string                `json:"kind"`
	ProjectRoot   string                `json:"projectRoot"`
	Languages     []string              `json:"languages"`
	Capabilities  map[string]Capability `json:"capabilities"`
	Guardrails    config.Guardrails     `json:"guardrails"`
	Provenance    map[string]string     `json:"provenance"`
}

// ScopeInfo describes what inputs were considered for an operation.
type ScopeInfo struct {
	Mode      string `json:"mode"`                // "full" | "diff"
	Source    string `json:"source"`              // "full-scan" | "git-diff" | "change-signal"
	FileCount int    `json:"fileCount"`           // candidate count (full or diff)
	DiffRange string `json:"diffRange,omitempty"` // when mode="diff"
}

// ContextPack is the authoritative context artifact.
type ContextPack struct {
	SchemaVersion     string               `json:"schemaVersion"`
	Kind              string               `json:"kind"`
	Goal              string               `json:"goal"`
	RoomsVisited      []string             `json:"roomsVisited"`
	FilesReferenced   []string             `json:"filesReferenced"`
	SymbolsReferenced []string             `json:"symbolsReferenced"`
	Findings          []Finding            `json:"findings"`
	Plan              []PlanStep           `json:"plan"`
	Verification      []VerificationResult `json:"verification"`

	Scope *ScopeInfo `json:"scope,omitempty"`

	ScanID   string `json:"scanId"`
	ScanHash string `json:"scanHash"`
	ScanTime string `json:"scanTime"`

	Provenance Provenance `json:"provenance"`
}

// Finding captures deterministic findings.
type Finding struct {
	Summary  string `json:"summary"`
	Detail   string `json:"detail,omitempty"`
	Severity string `json:"severity"`
	File     string `json:"file,omitempty"`
}

// PlanStep tracks execution progress.
type PlanStep struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

// VerificationResult tracks verification outcomes.
type VerificationResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Provenance records who generated the artifact.
type Provenance struct {
	CreatedBy        string `json:"createdBy"`
	CreatedAt        string `json:"createdAt"`
	UpdatedBy        string `json:"updatedBy,omitempty"`
	UpdatedAt        string `json:"updatedAt,omitempty"`
	Generator        string `json:"generator,omitempty"`
	GeneratorVersion string `json:"generatorVersion,omitempty"`
}

// Room describes a curated room manifest.
type Room struct {
	SchemaVersion string         `json:"schemaVersion"`
	Kind          string         `json:"kind"`
	Name          string         `json:"name"`
	Summary       string         `json:"summary"`
	EntryPoints   []string       `json:"entryPoints"`
	Artifacts     []RoomArtifact `json:"artifacts,omitempty"`
	Capabilities  []string       `json:"capabilities,omitempty"`
	Steps         []RoomStep     `json:"steps,omitempty"`
}

// Playbook describes a curated playbook manifest.
type Playbook struct {
	SchemaVersion string   `json:"schemaVersion"`
	Kind          string   `json:"kind"`
	Name          string   `json:"name"`
	Summary       string   `json:"summary"`
	Rooms         []string `json:"rooms"`
}

// RoomArtifact captures declared artifacts in a room.
type RoomArtifact struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	PathHint    string `json:"pathHint,omitempty"`
}

// RoomStep captures declared steps in a room.
type RoomStep struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Capability  string `json:"capability,omitempty"`
	Evidence    string `json:"evidence,omitempty"`
}

// LoadContextPack reads a context pack from disk.
func LoadContextPack(path string) (ContextPack, error) {
	var cp ContextPack
	if err := jsonc.DecodeFile(path, &cp); err != nil {
		return cp, err
	}
	return cp, nil
}

// WriteContextPack writes a context pack as JSON.
func WriteContextPack(path string, cp ContextPack) error {
	normalizeContextPack(&cp)
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// NewContextPack creates a bare context pack for a goal.
func NewContextPack(goal string) ContextPack {
	now := time.Now().UTC().Format(time.RFC3339)
	return ContextPack{
		SchemaVersion:     "1.0.0",
		Kind:              "palace/context-pack",
		Goal:              goal,
		RoomsVisited:      []string{},
		FilesReferenced:   []string{},
		SymbolsReferenced: []string{},
		Findings:          []Finding{},
		Plan:              []PlanStep{},
		Verification:      []VerificationResult{},
		Scope:             nil,
		ScanID:            "",
		ScanHash:          "",
		ScanTime:          now,
		Provenance: Provenance{
			CreatedBy: "palace",
			CreatedAt: now,
		},
	}
}

// Clone returns a shallow copy of the context pack.
func (cp ContextPack) Clone() ContextPack {
	return cp
}

func normalizeContextPack(cp *ContextPack) {
	if cp.RoomsVisited == nil {
		cp.RoomsVisited = []string{}
	}
	if cp.FilesReferenced == nil {
		cp.FilesReferenced = []string{}
	}
	if cp.SymbolsReferenced == nil {
		cp.SymbolsReferenced = []string{}
	}
	if cp.Findings == nil {
		cp.Findings = []Finding{}
	}
	if cp.Plan == nil {
		cp.Plan = []PlanStep{}
	}
	if cp.Verification == nil {
		cp.Verification = []VerificationResult{}
	}
}
