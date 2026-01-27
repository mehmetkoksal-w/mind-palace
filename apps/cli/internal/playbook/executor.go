// Package playbook provides playbook execution and management.
package playbook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/jsonc"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/model"
)

// ExecutionState tracks playbook execution progress.
type ExecutionState struct {
	ID             string                 `json:"id"`
	PlaybookName   string                 `json:"playbookName"`
	Playbook       *model.Playbook        `json:"playbook"`
	CurrentRoomIdx int                    `json:"currentRoomIdx"`
	CurrentStepIdx int                    `json:"currentStepIdx"`
	CurrentRoom    *model.Room            `json:"currentRoom,omitempty"`
	CompletedRooms []string               `json:"completedRooms"`
	CompletedSteps map[string][]int       `json:"completedSteps"` // room -> step indices
	Evidence       map[string]interface{} `json:"evidence"`
	StartedAt      time.Time              `json:"startedAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
	Status         string                 `json:"status"` // "running", "paused", "completed", "failed"
	SessionID      string                 `json:"sessionId,omitempty"`
}

// VerificationResult holds the result of a verification check.
type VerificationResult struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "passed", "failed", "skipped", "pending"
	Detail string `json:"detail,omitempty"`
	Error  string `json:"error,omitempty"`
}

// StepGuidance provides guidance for completing the current step.
type StepGuidance struct {
	RoomName        string   `json:"roomName"`
	RoomSummary     string   `json:"roomSummary"`
	StepNumber      int      `json:"stepNumber"`
	TotalSteps      int      `json:"totalSteps"`
	StepName        string   `json:"stepName"`
	StepDescription string   `json:"stepDescription,omitempty"`
	Capability      string   `json:"capability,omitempty"`
	EvidenceID      string   `json:"evidenceId,omitempty"`
	EntryPoints     []string `json:"entryPoints"`
	NextAction      string   `json:"nextAction"`
}

// Executor manages playbook execution.
type Executor struct {
	rootPath string
	rooms    map[string]model.Room
}

// NewExecutor creates a playbook executor.
func NewExecutor(rootPath string, rooms []model.Room) *Executor {
	roomMap := make(map[string]model.Room)
	for _, r := range rooms {
		roomMap[r.Name] = r
	}
	return &Executor{
		rootPath: rootPath,
		rooms:    roomMap,
	}
}

// ListPlaybooks returns all available playbooks.
func (e *Executor) ListPlaybooks() ([]model.Playbook, error) {
	playbooksDir := filepath.Join(e.rootPath, ".palace", "playbooks")
	entries, err := filepath.Glob(filepath.Join(playbooksDir, "*.jsonc"))
	if err != nil {
		return nil, err
	}

	playbooks := make([]model.Playbook, 0, len(entries))
	for _, path := range entries {
		pb, err := model.LoadPlaybook(path)
		if err != nil {
			continue // Skip invalid playbooks
		}
		playbooks = append(playbooks, pb)
	}
	return playbooks, nil
}

// LoadPlaybook loads a playbook by name from .palace/playbooks/.
func (e *Executor) LoadPlaybook(name string) (*model.Playbook, error) {
	path := filepath.Join(e.rootPath, ".palace", "playbooks", name+".jsonc")
	pb, err := model.LoadPlaybook(path)
	if err != nil {
		return nil, fmt.Errorf("load playbook %s: %w", name, err)
	}
	return &pb, nil
}

// Start begins execution of a playbook.
func (e *Executor) Start(name string) (*ExecutionState, error) {
	pb, err := e.LoadPlaybook(name)
	if err != nil {
		return nil, err
	}

	// Validate all rooms exist
	for _, roomName := range pb.Rooms {
		if _, ok := e.rooms[roomName]; !ok {
			return nil, fmt.Errorf("room %q not found", roomName)
		}
	}

	now := time.Now().UTC()
	state := &ExecutionState{
		ID:             generateExecutionID(),
		PlaybookName:   name,
		Playbook:       pb,
		CurrentRoomIdx: 0,
		CurrentStepIdx: 0,
		CompletedRooms: []string{},
		CompletedSteps: make(map[string][]int),
		Evidence:       make(map[string]interface{}),
		StartedAt:      now,
		UpdatedAt:      now,
		Status:         "running",
	}

	// Load first room
	if len(pb.Rooms) > 0 {
		room := e.rooms[pb.Rooms[0]]
		state.CurrentRoom = &room
	}

	return state, nil
}

// GetCurrentGuidance returns guidance for the current step.
func (e *Executor) GetCurrentGuidance(state *ExecutionState) (*StepGuidance, error) {
	if state.Status == "completed" {
		return nil, fmt.Errorf("playbook already completed")
	}

	if state.CurrentRoom == nil {
		return nil, fmt.Errorf("no current room")
	}

	guidance := &StepGuidance{
		RoomName:    state.CurrentRoom.Name,
		RoomSummary: state.CurrentRoom.Summary,
		TotalSteps:  len(state.CurrentRoom.Steps),
		EntryPoints: state.CurrentRoom.EntryPoints,
	}

	if state.CurrentStepIdx < len(state.CurrentRoom.Steps) {
		step := state.CurrentRoom.Steps[state.CurrentStepIdx]
		guidance.StepNumber = state.CurrentStepIdx + 1
		guidance.StepName = step.Name
		guidance.StepDescription = step.Description
		guidance.Capability = step.Capability
		guidance.EvidenceID = step.Evidence
		guidance.NextAction = "Complete this step, then call playbook with action=advance"
	} else {
		guidance.StepNumber = len(state.CurrentRoom.Steps)
		guidance.NextAction = "All steps complete. Call playbook with action=advance to move to next room"
	}

	return guidance, nil
}

// AdvanceStep moves to the next step or room.
func (e *Executor) AdvanceStep(state *ExecutionState) error {
	if state.CurrentRoom == nil {
		return fmt.Errorf("no current room")
	}

	// Mark current step as completed
	roomName := state.CurrentRoom.Name
	if _, ok := state.CompletedSteps[roomName]; !ok {
		state.CompletedSteps[roomName] = []int{}
	}
	state.CompletedSteps[roomName] = append(state.CompletedSteps[roomName], state.CurrentStepIdx)

	state.CurrentStepIdx++
	state.UpdatedAt = time.Now().UTC()

	// Check if room completed
	if state.CurrentStepIdx >= len(state.CurrentRoom.Steps) {
		return e.AdvanceRoom(state)
	}
	return nil
}

// AdvanceRoom moves to the next room.
func (e *Executor) AdvanceRoom(state *ExecutionState) error {
	// Mark current room complete
	if state.CurrentRoom != nil {
		state.CompletedRooms = append(state.CompletedRooms, state.CurrentRoom.Name)
	}

	state.CurrentRoomIdx++
	state.CurrentStepIdx = 0
	state.UpdatedAt = time.Now().UTC()

	// Check if playbook completed
	if state.CurrentRoomIdx >= len(state.Playbook.Rooms) {
		state.Status = "completed"
		state.CurrentRoom = nil
		return nil
	}

	// Load next room
	roomName := state.Playbook.Rooms[state.CurrentRoomIdx]
	room := e.rooms[roomName]
	state.CurrentRoom = &room
	return nil
}

// CollectEvidence stores evidence for a required evidence ID.
func (e *Executor) CollectEvidence(state *ExecutionState, evidenceID string, data interface{}) error {
	state.Evidence[evidenceID] = data
	state.UpdatedAt = time.Now().UTC()
	return nil
}

// RunVerification executes verification checks.
func (e *Executor) RunVerification(state *ExecutionState) ([]VerificationResult, error) {
	if state.Playbook == nil {
		return nil, fmt.Errorf("no playbook loaded")
	}

	results := make([]VerificationResult, 0, len(state.Playbook.Verification))

	for _, check := range state.Playbook.Verification {
		result := VerificationResult{
			Name:   check.Name,
			Status: "pending",
		}

		// Execute capability-based verification
		switch check.Capability {
		case "lint.run":
			// TODO: Run palace lint
			result.Status = "skipped"
			result.Detail = "Run 'palace lint' to verify"
		case "tests.run":
			// TODO: Run tests
			result.Status = "skipped"
			result.Detail = "Run project tests to verify"
		default:
			result.Status = "skipped"
			result.Detail = fmt.Sprintf("Manual verification: %s", check.Expectation)
		}

		results = append(results, result)
	}

	return results, nil
}

// GetProgress returns overall progress percentage.
func (e *Executor) GetProgress(state *ExecutionState) float64 {
	if state.Playbook == nil || len(state.Playbook.Rooms) == 0 {
		return 100.0
	}

	// Calculate based on rooms and steps
	totalSteps := 0
	completedSteps := 0

	for _, roomName := range state.Playbook.Rooms {
		room, ok := e.rooms[roomName]
		if !ok {
			continue
		}
		roomSteps := len(room.Steps)
		if roomSteps == 0 {
			roomSteps = 1 // Count room itself if no steps
		}
		totalSteps += roomSteps

		// Count completed steps for this room
		if completed, ok := state.CompletedSteps[roomName]; ok {
			completedSteps += len(completed)
		}
	}

	if totalSteps == 0 {
		return 100.0
	}
	return float64(completedSteps) / float64(totalSteps) * 100
}

// SaveState persists execution state to disk.
func (e *Executor) SaveState(state *ExecutionState) error {
	outputDir := filepath.Join(e.rootPath, ".palace", "outputs")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	path := filepath.Join(outputDir, "playbook-state.json")
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// LoadState loads persisted execution state.
func (e *Executor) LoadState() (*ExecutionState, error) {
	path := filepath.Join(e.rootPath, ".palace", "outputs", "playbook-state.json")
	var state ExecutionState
	if err := jsonc.DecodeFile(path, &state); err != nil {
		return nil, err
	}

	// Reload current room from rooms map
	if state.CurrentRoomIdx < len(state.Playbook.Rooms) {
		roomName := state.Playbook.Rooms[state.CurrentRoomIdx]
		if room, ok := e.rooms[roomName]; ok {
			state.CurrentRoom = &room
		}
	}

	return &state, nil
}

// ClearState removes persisted execution state.
func (e *Executor) ClearState() error {
	path := filepath.Join(e.rootPath, ".palace", "outputs", "playbook-state.json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// generateExecutionID generates a unique execution ID.
func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}
