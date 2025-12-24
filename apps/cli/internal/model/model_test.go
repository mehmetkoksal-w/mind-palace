package model_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

func TestNewContextPack(t *testing.T) {
	goal := "Test the new feature"
	cp := model.NewContextPack(goal)

	if cp.Goal != goal {
		t.Errorf("expected goal %q, got %q", goal, cp.Goal)
	}
	if cp.SchemaVersion != "1.0.0" {
		t.Errorf("expected schema version 1.0.0, got %s", cp.SchemaVersion)
	}
	if cp.Kind != "palace/context-pack" {
		t.Errorf("expected kind palace/context-pack, got %s", cp.Kind)
	}
	if cp.RoomsVisited == nil {
		t.Error("RoomsVisited should be initialized")
	}
	if cp.FilesReferenced == nil {
		t.Error("FilesReferenced should be initialized")
	}
	if cp.SymbolsReferenced == nil {
		t.Error("SymbolsReferenced should be initialized")
	}
	if cp.Findings == nil {
		t.Error("Findings should be initialized")
	}
	if cp.Plan == nil {
		t.Error("Plan should be initialized")
	}
	if cp.Verification == nil {
		t.Error("Verification should be initialized")
	}
	if cp.Provenance.CreatedBy != "palace" {
		t.Errorf("expected createdBy palace, got %s", cp.Provenance.CreatedBy)
	}
	if cp.Provenance.CreatedAt == "" {
		t.Error("CreatedAt should be set")
	}
}

func TestContextPackClone(t *testing.T) {
	cp := model.NewContextPack("Original goal")
	cp.ScanID = "scan-123"
	cp.ScanHash = "hash-456"
	cp.RoomsVisited = []string{"room1", "room2"}

	cloned := cp.Clone()

	if cloned.Goal != cp.Goal {
		t.Errorf("cloned goal mismatch: got %q, want %q", cloned.Goal, cp.Goal)
	}
	if cloned.ScanID != cp.ScanID {
		t.Errorf("cloned scanId mismatch: got %q, want %q", cloned.ScanID, cp.ScanID)
	}
	if len(cloned.RoomsVisited) != len(cp.RoomsVisited) {
		t.Errorf("cloned roomsVisited length mismatch: got %d, want %d", len(cloned.RoomsVisited), len(cp.RoomsVisited))
	}
}

func TestWriteAndLoadContextPack(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "context-pack.json")

	original := model.NewContextPack("Test goal")
	original.ScanID = "test-scan-id"
	original.ScanHash = "test-hash"
	original.RoomsVisited = []string{"room1"}
	original.FilesReferenced = []string{"file1.go", "file2.go"}
	original.SymbolsReferenced = []string{"func1", "func2"}
	original.Findings = []model.Finding{
		{Summary: "Found something", Severity: "info"},
	}
	original.Plan = []model.PlanStep{
		{Step: "Step 1", Status: "pending"},
	}
	original.Verification = []model.VerificationResult{
		{Name: "Check 1", Status: "pass"},
	}

	// Write
	if err := model.WriteContextPack(path, original); err != nil {
		t.Fatalf("WriteContextPack failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("context pack file was not created")
	}

	// Load
	loaded, err := model.LoadContextPack(path)
	if err != nil {
		t.Fatalf("LoadContextPack failed: %v", err)
	}

	// Verify contents
	if loaded.Goal != original.Goal {
		t.Errorf("goal mismatch: got %q, want %q", loaded.Goal, original.Goal)
	}
	if loaded.ScanID != original.ScanID {
		t.Errorf("scanId mismatch: got %q, want %q", loaded.ScanID, original.ScanID)
	}
	if len(loaded.RoomsVisited) != len(original.RoomsVisited) {
		t.Errorf("roomsVisited length mismatch: got %d, want %d", len(loaded.RoomsVisited), len(original.RoomsVisited))
	}
	if len(loaded.FilesReferenced) != len(original.FilesReferenced) {
		t.Errorf("filesReferenced length mismatch: got %d, want %d", len(loaded.FilesReferenced), len(original.FilesReferenced))
	}
}

func TestLoadContextPackNotFound(t *testing.T) {
	_, err := model.LoadContextPack("/nonexistent/path/context.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoadContextPackInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(path, []byte("not valid json {"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := model.LoadContextPack(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestWriteContextPackNormalizesNils(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "context-pack.json")

	// Create a context pack with nil slices
	cp := model.ContextPack{
		SchemaVersion: "1.0.0",
		Kind:          "palace/context-pack",
		Goal:          "Test",
		// Leave slices nil
	}

	if err := model.WriteContextPack(path, cp); err != nil {
		t.Fatalf("WriteContextPack failed: %v", err)
	}

	// Read and verify empty arrays (not null)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	// Check that arrays are present (not null)
	content := string(data)
	if !contains(content, `"roomsVisited": []`) {
		t.Error("roomsVisited should be empty array, not null")
	}
	if !contains(content, `"filesReferenced": []`) {
		t.Error("filesReferenced should be empty array, not null")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestContextPackTypes(t *testing.T) {
	// Test Finding
	finding := model.Finding{
		Summary:  "Test finding",
		Detail:   "Detailed description",
		Severity: "warning",
		File:     "test.go",
	}
	if finding.Summary != "Test finding" {
		t.Error("Finding summary not set correctly")
	}

	// Test PlanStep
	step := model.PlanStep{
		Step:   "Do something",
		Status: "pending",
	}
	if step.Status != "pending" {
		t.Error("PlanStep status not set correctly")
	}

	// Test VerificationResult
	result := model.VerificationResult{
		Name:   "Test check",
		Status: "pass",
		Detail: "All good",
	}
	if result.Status != "pass" {
		t.Error("VerificationResult status not set correctly")
	}

	// Test Provenance
	prov := model.Provenance{
		CreatedBy:        "test",
		CreatedAt:        time.Now().Format(time.RFC3339),
		UpdatedBy:        "test2",
		UpdatedAt:        time.Now().Format(time.RFC3339),
		Generator:        "palace",
		GeneratorVersion: "1.0.0",
	}
	if prov.Generator != "palace" {
		t.Error("Provenance generator not set correctly")
	}

	// Test ScopeInfo
	scope := model.ScopeInfo{
		Mode:      "diff",
		Source:    "git-diff",
		FileCount: 10,
		DiffRange: "HEAD~1..HEAD",
	}
	if scope.Mode != "diff" {
		t.Error("ScopeInfo mode not set correctly")
	}

	// Test CorridorInfo
	corridor := model.CorridorInfo{
		Name:      "neighbor1",
		Source:    "https://example.com/context.json",
		Goal:      "Remote goal",
		Files:     []string{"file1.go"},
		Rooms:     []string{"room1"},
		FromCache: true,
		FetchedAt: time.Now().Format(time.RFC3339),
		Error:     "",
	}
	if corridor.Name != "neighbor1" {
		t.Error("CorridorInfo name not set correctly")
	}
}

func TestRoomAndPlaybookTypes(t *testing.T) {
	// Test Room
	room := model.Room{
		SchemaVersion: "1.0.0",
		Kind:          "palace/room",
		Name:          "auth",
		Summary:       "Authentication logic",
		EntryPoints:   []string{"login.go", "logout.go"},
		Artifacts: []model.RoomArtifact{
			{Name: "handler", Description: "Main handler", PathHint: "handler.go"},
		},
		Capabilities: []string{"test", "build"},
		Steps: []model.RoomStep{
			{Name: "setup", Description: "Setup deps", Capability: "install", Evidence: "package.json"},
		},
	}
	if room.Name != "auth" {
		t.Error("Room name not set correctly")
	}
	if len(room.EntryPoints) != 2 {
		t.Error("Room entryPoints not set correctly")
	}
	if len(room.Artifacts) != 1 {
		t.Error("Room artifacts not set correctly")
	}

	// Test Playbook
	playbook := model.Playbook{
		SchemaVersion: "1.0.0",
		Kind:          "palace/playbook",
		Name:          "deploy",
		Summary:       "Deployment playbook",
		Rooms:         []string{"auth", "api"},
	}
	if playbook.Name != "deploy" {
		t.Error("Playbook name not set correctly")
	}
	if len(playbook.Rooms) != 2 {
		t.Error("Playbook rooms not set correctly")
	}
}

func TestCapabilityType(t *testing.T) {
	cap := model.Capability{
		Command:          "npm test",
		Description:      "Run tests",
		WorkingDirectory: "./src",
		Env:              map[string]string{"NODE_ENV": "test"},
	}
	if cap.Command != "npm test" {
		t.Error("Capability command not set correctly")
	}
	if cap.Env["NODE_ENV"] != "test" {
		t.Error("Capability env not set correctly")
	}
}
