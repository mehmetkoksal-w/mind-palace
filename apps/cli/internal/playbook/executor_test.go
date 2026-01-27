package playbook

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/model"
)

func setupTestPlaybook(t *testing.T) (string, *Executor) {
	t.Helper()

	root := t.TempDir()

	// Create directories
	os.MkdirAll(filepath.Join(root, ".palace", "rooms"), 0o755)
	os.MkdirAll(filepath.Join(root, ".palace", "playbooks"), 0o755)
	os.MkdirAll(filepath.Join(root, ".palace", "outputs"), 0o755)

	// Create test rooms
	room1 := `{
  "schemaVersion": "1.0.0",
  "kind": "palace/room",
  "name": "room1",
  "summary": "First test room",
  "entryPoints": ["src/"],
  "capabilities": ["read.file"],
  "steps": [
    {"name": "Step 1", "description": "First step"},
    {"name": "Step 2", "description": "Second step"}
  ]
}`
	os.WriteFile(filepath.Join(root, ".palace", "rooms", "room1.jsonc"), []byte(room1), 0o644)

	room2 := `{
  "schemaVersion": "1.0.0",
  "kind": "palace/room",
  "name": "room2",
  "summary": "Second test room",
  "entryPoints": ["lib/"],
  "steps": [
    {"name": "Step A", "description": "Step A desc", "evidence": "artifact-a"}
  ]
}`
	os.WriteFile(filepath.Join(root, ".palace", "rooms", "room2.jsonc"), []byte(room2), 0o644)

	// Create test playbook
	playbook := `{
  "schemaVersion": "1.0.0",
  "kind": "palace/playbook",
  "name": "test-playbook",
  "summary": "A test playbook",
  "rooms": ["room1", "room2"],
  "requiredEvidence": [
    {"id": "artifact-a", "description": "Test artifact", "room": "room2"}
  ],
  "verification": [
    {"name": "lint", "expectation": "lint passes", "capability": "lint.run"}
  ]
}`
	os.WriteFile(filepath.Join(root, ".palace", "playbooks", "test-playbook.jsonc"), []byte(playbook), 0o644)

	// Load rooms
	rooms := []model.Room{
		{
			SchemaVersion: "1.0.0",
			Kind:          "palace/room",
			Name:          "room1",
			Summary:       "First test room",
			EntryPoints:   []string{"src/"},
			Capabilities:  []string{"read.file"},
			Steps: []model.RoomStep{
				{Name: "Step 1", Description: "First step"},
				{Name: "Step 2", Description: "Second step"},
			},
		},
		{
			SchemaVersion: "1.0.0",
			Kind:          "palace/room",
			Name:          "room2",
			Summary:       "Second test room",
			EntryPoints:   []string{"lib/"},
			Steps: []model.RoomStep{
				{Name: "Step A", Description: "Step A desc", Evidence: "artifact-a"},
			},
		},
	}

	executor := NewExecutor(root, rooms)
	return root, executor
}

func TestListPlaybooks(t *testing.T) {
	_, executor := setupTestPlaybook(t)

	playbooks, err := executor.ListPlaybooks()
	if err != nil {
		t.Fatalf("ListPlaybooks() error: %v", err)
	}

	if len(playbooks) != 1 {
		t.Errorf("expected 1 playbook, got %d", len(playbooks))
	}

	if playbooks[0].Name != "test-playbook" {
		t.Errorf("expected playbook name 'test-playbook', got '%s'", playbooks[0].Name)
	}
}

func TestLoadPlaybook(t *testing.T) {
	_, executor := setupTestPlaybook(t)

	pb, err := executor.LoadPlaybook("test-playbook")
	if err != nil {
		t.Fatalf("LoadPlaybook() error: %v", err)
	}

	if pb.Name != "test-playbook" {
		t.Errorf("expected name 'test-playbook', got '%s'", pb.Name)
	}

	if len(pb.Rooms) != 2 {
		t.Errorf("expected 2 rooms, got %d", len(pb.Rooms))
	}

	if len(pb.RequiredEvidence) != 1 {
		t.Errorf("expected 1 required evidence, got %d", len(pb.RequiredEvidence))
	}
}

func TestStart(t *testing.T) {
	_, executor := setupTestPlaybook(t)

	state, err := executor.Start("test-playbook")
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	if state.Status != "running" {
		t.Errorf("expected status 'running', got '%s'", state.Status)
	}

	if state.CurrentRoomIdx != 0 {
		t.Errorf("expected current room index 0, got %d", state.CurrentRoomIdx)
	}

	if state.CurrentRoom == nil {
		t.Fatal("expected current room to be set")
	}

	if state.CurrentRoom.Name != "room1" {
		t.Errorf("expected current room 'room1', got '%s'", state.CurrentRoom.Name)
	}
}

func TestGetCurrentGuidance(t *testing.T) {
	_, executor := setupTestPlaybook(t)

	state, _ := executor.Start("test-playbook")
	guidance, err := executor.GetCurrentGuidance(state)
	if err != nil {
		t.Fatalf("GetCurrentGuidance() error: %v", err)
	}

	if guidance.RoomName != "room1" {
		t.Errorf("expected room 'room1', got '%s'", guidance.RoomName)
	}

	if guidance.StepNumber != 1 {
		t.Errorf("expected step 1, got %d", guidance.StepNumber)
	}

	if guidance.StepName != "Step 1" {
		t.Errorf("expected step name 'Step 1', got '%s'", guidance.StepName)
	}
}

func TestAdvanceStep(t *testing.T) {
	_, executor := setupTestPlaybook(t)

	state, _ := executor.Start("test-playbook")

	// Advance first step in room1
	if err := executor.AdvanceStep(state); err != nil {
		t.Fatalf("AdvanceStep() error: %v", err)
	}

	if state.CurrentStepIdx != 1 {
		t.Errorf("expected step index 1, got %d", state.CurrentStepIdx)
	}

	// Advance second step in room1 (should move to room2)
	if err := executor.AdvanceStep(state); err != nil {
		t.Fatalf("AdvanceStep() error: %v", err)
	}

	if state.CurrentRoom.Name != "room2" {
		t.Errorf("expected current room 'room2', got '%s'", state.CurrentRoom.Name)
	}

	if state.CurrentStepIdx != 0 {
		t.Errorf("expected step index 0 in new room, got %d", state.CurrentStepIdx)
	}
}

func TestAdvanceRoom(t *testing.T) {
	_, executor := setupTestPlaybook(t)

	state, _ := executor.Start("test-playbook")

	// Complete all steps and rooms
	executor.AdvanceStep(state) // room1 step 1
	executor.AdvanceStep(state) // room1 step 2 -> room2
	executor.AdvanceStep(state) // room2 step A -> complete

	if state.Status != "completed" {
		t.Errorf("expected status 'completed', got '%s'", state.Status)
	}

	if len(state.CompletedRooms) != 2 {
		t.Errorf("expected 2 completed rooms, got %d", len(state.CompletedRooms))
	}
}

func TestCollectEvidence(t *testing.T) {
	_, executor := setupTestPlaybook(t)

	state, _ := executor.Start("test-playbook")

	if err := executor.CollectEvidence(state, "artifact-a", "collected value"); err != nil {
		t.Fatalf("CollectEvidence() error: %v", err)
	}

	if _, ok := state.Evidence["artifact-a"]; !ok {
		t.Error("expected evidence to be stored")
	}
}

func TestRunVerification(t *testing.T) {
	_, executor := setupTestPlaybook(t)

	state, _ := executor.Start("test-playbook")

	results, err := executor.RunVerification(state)
	if err != nil {
		t.Fatalf("RunVerification() error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 verification result, got %d", len(results))
	}

	if results[0].Name != "lint" {
		t.Errorf("expected verification name 'lint', got '%s'", results[0].Name)
	}
}

func TestGetProgress(t *testing.T) {
	_, executor := setupTestPlaybook(t)

	state, _ := executor.Start("test-playbook")

	// Initial progress should be 0
	progress := executor.GetProgress(state)
	if progress != 0 {
		t.Errorf("expected initial progress 0, got %f", progress)
	}

	// Complete one step
	executor.AdvanceStep(state)
	progress = executor.GetProgress(state)
	if progress < 20 || progress > 40 {
		t.Errorf("expected progress around 33%%, got %f%%", progress)
	}
}

func TestSaveAndLoadState(t *testing.T) {
	_, executor := setupTestPlaybook(t)

	state, _ := executor.Start("test-playbook")
	state.Evidence["test-evidence"] = "test-value"

	// Save state
	if err := executor.SaveState(state); err != nil {
		t.Fatalf("SaveState() error: %v", err)
	}

	// Load state
	loaded, err := executor.LoadState()
	if err != nil {
		t.Fatalf("LoadState() error: %v", err)
	}

	if loaded.PlaybookName != state.PlaybookName {
		t.Errorf("expected playbook name '%s', got '%s'", state.PlaybookName, loaded.PlaybookName)
	}

	if _, ok := loaded.Evidence["test-evidence"]; !ok {
		t.Error("expected evidence to be persisted")
	}
}

func TestClearState(t *testing.T) {
	root, executor := setupTestPlaybook(t)

	state, _ := executor.Start("test-playbook")
	executor.SaveState(state)

	if err := executor.ClearState(); err != nil {
		t.Fatalf("ClearState() error: %v", err)
	}

	statePath := filepath.Join(root, ".palace", "outputs", "playbook-state.json")
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Error("expected state file to be deleted")
	}
}

func TestStartNonexistentPlaybook(t *testing.T) {
	_, executor := setupTestPlaybook(t)

	_, err := executor.Start("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent playbook")
	}
}

func TestStartWithMissingRoom(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".palace", "playbooks"), 0o755)

	// Create playbook with nonexistent room
	playbook := `{
  "schemaVersion": "1.0.0",
  "kind": "palace/playbook",
  "name": "bad-playbook",
  "summary": "Bad playbook",
  "rooms": ["nonexistent-room"]
}`
	os.WriteFile(filepath.Join(root, ".palace", "playbooks", "bad-playbook.jsonc"), []byte(playbook), 0o644)

	executor := NewExecutor(root, []model.Room{})

	_, err := executor.Start("bad-playbook")
	if err == nil {
		t.Error("expected error for missing room")
	}
}
