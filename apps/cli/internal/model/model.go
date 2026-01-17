// Package model defines the core data structures used by Mind Palace.
package model

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/jsonc"
)

// Capability defines a task or tool that can be executed within a room.
type Capability struct {
	Command          string            `json:"command"`
	Description      string            `json:"description"`
	WorkingDirectory string            `json:"workingDirectory,omitempty"`
	Env              map[string]string `json:"env,omitempty"`
}

// ProjectProfile describes the project's structure, languages, and capabilities.
type ProjectProfile struct {
	SchemaVersion string                `json:"schemaVersion"`
	Kind          string                `json:"kind"`
	ProjectRoot   string                `json:"projectRoot"`
	Languages     []string              `json:"languages"`
	Capabilities  map[string]Capability `json:"capabilities"`
	Guardrails    config.Guardrails     `json:"guardrails"`
	Provenance    map[string]string     `json:"provenance"`
}

// ScopeInfo tracks the analysis scope (full scan vs incremental diff).
type ScopeInfo struct {
	Mode      string `json:"mode"`                // "full" | "diff"
	Source    string `json:"source"`              // "full-scan" | "git-diff" | "change-signal"
	FileCount int    `json:"fileCount"`           // candidate count (full or diff)
	DiffRange string `json:"diffRange,omitempty"` // when mode="diff"
}

// ContextPack is a portable bundle of context, findings, and plans for a task.
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

	Scope     *ScopeInfo     `json:"scope,omitempty"`
	Corridors []CorridorInfo `json:"corridors,omitempty"` // Remote context from neighbors

	ScanID   string `json:"scanId"`
	ScanHash string `json:"scanHash"`
	ScanTime string `json:"scanTime"`

	Provenance Provenance `json:"provenance"`
}

// CorridorInfo describes context available from a neighboring project.
type CorridorInfo struct {
	Name      string   `json:"name"`            // Neighbor name
	Source    string   `json:"source"`          // URL or local path
	Goal      string   `json:"goal,omitempty"`  // Remote pack's goal
	Files     []string `json:"files"`           // Namespaced: corridor://{name}/{path}
	Rooms     []string `json:"rooms,omitempty"` // Remote room names
	FromCache bool     `json:"fromCache"`       // True if loaded from cache
	FetchedAt string   `json:"fetchedAt"`       // When this was fetched
	Error     string   `json:"error,omitempty"` // Any fetch errors (non-fatal)
}

// Finding represents an observation made during analysis.
type Finding struct {
	Summary  string `json:"summary"`
	Detail   string `json:"detail,omitempty"`
	Severity string `json:"severity"`
	File     string `json:"file,omitempty"`
}

// PlanStep represents a single step in a task execution plan.
type PlanStep struct {
	Step   string `json:"step"`
	Status string `json:"status"`
}

// VerificationResult represents the outcome of a verification check.
type VerificationResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Provenance tracks the origin and creation details of an object.
type Provenance struct {
	CreatedBy        string `json:"createdBy"`
	CreatedAt        string `json:"createdAt"`
	UpdatedBy        string `json:"updatedBy,omitempty"`
	UpdatedAt        string `json:"updatedAt,omitempty"`
	Generator        string `json:"generator,omitempty"`
	GeneratorVersion string `json:"generatorVersion,omitempty"`
}

// Room defines a logical grouping of context and tools for specific tasks.
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

// Playbook is a sequence of rooms to be visited for a high-level goal.
type Playbook struct {
	SchemaVersion string   `json:"schemaVersion"`
	Kind          string   `json:"kind"`
	Name          string   `json:"name"`
	Summary       string   `json:"summary"`
	Rooms         []string `json:"rooms"`
}

// RoomArtifact describes a specific file or object produced/consumed in a room.
type RoomArtifact struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	PathHint    string `json:"pathHint,omitempty"`
}

// RoomStep defines a specific action to take within a room.
type RoomStep struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Capability  string `json:"capability,omitempty"`
	Evidence    string `json:"evidence,omitempty"`
}

// LoadContextPack loads a context pack from a JSONC file.
func LoadContextPack(path string) (ContextPack, error) {
	var cp ContextPack
	if err := jsonc.DecodeFile(path, &cp); err != nil {
		return cp, err
	}
	return cp, nil
}

// WriteContextPack writes a context pack to a JSON file.
func WriteContextPack(path string, cp ContextPack) error {
	normalizeContextPack(&cp)
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// NewContextPack creates a fresh context pack for a new task.
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

// Clone creates a deep copy of the context pack.
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
