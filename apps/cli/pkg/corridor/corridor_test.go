package corridor

import (
	"os"
	"testing"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/corridor"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

func TestGlobalPath(t *testing.T) {
	path, err := GlobalPath()
	if err != nil {
		t.Fatalf("GlobalPath() error: %v", err)
	}
	if path == "" {
		t.Error("GlobalPath() should return non-empty path")
	}
}

func TestEnsureGlobalLayout(t *testing.T) {
	path, err := EnsureGlobalLayout()
	if err != nil {
		t.Fatalf("EnsureGlobalLayout() error: %v", err)
	}
	if path == "" {
		t.Error("EnsureGlobalLayout() should return non-empty path")
	}
}

func TestConvertPersonalLearnings(t *testing.T) {
	now := time.Now()
	input := []corridor.PersonalLearning{
		{
			ID:              "test-id-1",
			OriginWorkspace: "workspace1",
			Content:         "Test content 1",
			Confidence:      0.8,
			Source:          "promoted",
			CreatedAt:       now,
			LastUsed:        now,
			UseCount:        5,
			Tags:            []string{"tag1", "tag2"},
		},
		{
			ID:              "test-id-2",
			OriginWorkspace: "workspace2",
			Content:         "Test content 2",
			Confidence:      0.6,
			Source:          "manual",
			CreatedAt:       now,
			LastUsed:        now,
			UseCount:        3,
			Tags:            []string{"tag3"},
		},
	}

	result := convertPersonalLearnings(input)

	if len(result) != len(input) {
		t.Errorf("expected %d learnings, got %d", len(input), len(result))
	}

	for i, r := range result {
		if r.ID != input[i].ID {
			t.Errorf("[%d] ID mismatch: got %q, want %q", i, r.ID, input[i].ID)
		}
		if r.OriginWorkspace != input[i].OriginWorkspace {
			t.Errorf("[%d] OriginWorkspace mismatch: got %q, want %q", i, r.OriginWorkspace, input[i].OriginWorkspace)
		}
		if r.Content != input[i].Content {
			t.Errorf("[%d] Content mismatch: got %q, want %q", i, r.Content, input[i].Content)
		}
		if r.Confidence != input[i].Confidence {
			t.Errorf("[%d] Confidence mismatch: got %f, want %f", i, r.Confidence, input[i].Confidence)
		}
	}
}

func TestConvertPersonalLearningsEmpty(t *testing.T) {
	result := convertPersonalLearnings([]corridor.PersonalLearning{})
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d items", len(result))
	}
}

func TestConvertLinkedWorkspaces(t *testing.T) {
	now := time.Now()
	input := []corridor.LinkedWorkspace{
		{
			Name:         "api-project",
			Path:         "/path/to/api",
			AddedAt:      now,
			LastAccessed: now,
		},
		{
			Name:         "web-project",
			Path:         "/path/to/web",
			AddedAt:      now,
			LastAccessed: now,
		},
	}

	result := convertLinkedWorkspaces(input)

	if len(result) != len(input) {
		t.Errorf("expected %d workspaces, got %d", len(input), len(result))
	}

	for i, r := range result {
		if r.Name != input[i].Name {
			t.Errorf("[%d] Name mismatch: got %q, want %q", i, r.Name, input[i].Name)
		}
		if r.Path != input[i].Path {
			t.Errorf("[%d] Path mismatch: got %q, want %q", i, r.Path, input[i].Path)
		}
	}
}

func TestConvertLinkedWorkspacesEmpty(t *testing.T) {
	result := convertLinkedWorkspaces([]corridor.LinkedWorkspace{})
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d items", len(result))
	}
}

func TestConvertInternalLearnings(t *testing.T) {
	now := time.Now()
	input := []memory.Learning{
		{
			ID:         "learn-1",
			SessionID:  "session-1",
			Scope:      "file",
			ScopePath:  "src/main.go",
			Content:    "Always check errors",
			Confidence: 0.9,
			Source:     "agent",
			CreatedAt:  now,
			LastUsed:   now,
			UseCount:   10,
		},
		{
			ID:         "learn-2",
			SessionID:  "session-2",
			Scope:      "room",
			ScopePath:  "auth",
			Content:    "Validate tokens",
			Confidence: 0.7,
			Source:     "manual",
			CreatedAt:  now,
			LastUsed:   now,
			UseCount:   5,
		},
	}

	result := convertInternalLearnings(input)

	if len(result) != len(input) {
		t.Errorf("expected %d learnings, got %d", len(input), len(result))
	}

	for i, r := range result {
		if r.ID != input[i].ID {
			t.Errorf("[%d] ID mismatch: got %q, want %q", i, r.ID, input[i].ID)
		}
		if r.SessionID != input[i].SessionID {
			t.Errorf("[%d] SessionID mismatch: got %q, want %q", i, r.SessionID, input[i].SessionID)
		}
		if r.Scope != input[i].Scope {
			t.Errorf("[%d] Scope mismatch: got %q, want %q", i, r.Scope, input[i].Scope)
		}
		if r.Content != input[i].Content {
			t.Errorf("[%d] Content mismatch: got %q, want %q", i, r.Content, input[i].Content)
		}
	}
}

func TestConvertInternalLearningsEmpty(t *testing.T) {
	result := convertInternalLearnings([]memory.Learning{})
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d items", len(result))
	}
}

func TestTypeAliases(t *testing.T) {
	// Test that type aliases work correctly
	pl := PersonalLearning{
		ID:      "test-id",
		Content: "test content",
	}
	if pl.ID != "test-id" {
		t.Error("PersonalLearning type alias not working")
	}
	if pl.Content != "test content" {
		t.Error("PersonalLearning Content not set correctly")
	}

	lw := LinkedWorkspace{
		Name: "test-workspace",
		Path: "/path/to/workspace",
	}
	if lw.Name != "test-workspace" {
		t.Error("LinkedWorkspace type alias not working")
	}
	if lw.Path != "/path/to/workspace" {
		t.Error("LinkedWorkspace Path not set correctly")
	}

	l := Learning{
		ID:      "learn-id",
		Content: "learn content",
	}
	if l.ID != "learn-id" {
		t.Error("Learning type alias not working")
	}
	if l.Content != "learn content" {
		t.Error("Learning Content not set correctly")
	}
}

func TestOpenGlobalAndClose(t *testing.T) {
	gc, err := OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal() error: %v", err)
	}

	if gc == nil {
		t.Fatal("OpenGlobal() returned nil GlobalCorridor")
	}

	if err := gc.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}
}

func TestOpenGlobalStats(t *testing.T) {
	gc, err := OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal() error: %v", err)
	}
	defer gc.Close()

	stats, err := gc.Stats()
	if err != nil {
		t.Errorf("Stats() error: %v", err)
	}

	if stats == nil {
		t.Error("Stats() should return non-nil map")
	}
}

func TestOpenGlobalGetLinks(t *testing.T) {
	gc, err := OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal() error: %v", err)
	}
	defer gc.Close()

	links, err := gc.GetLinks()
	if err != nil {
		t.Errorf("GetLinks() error: %v", err)
	}

	// Links should not be nil (may be empty slice)
	if links == nil {
		t.Error("GetLinks() should return non-nil slice")
	}
}

func TestOpenGlobalGetPersonalLearnings(t *testing.T) {
	gc, err := OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal() error: %v", err)
	}
	defer gc.Close()

	// Should work with empty query
	learnings, err := gc.GetPersonalLearnings("", 10)
	if err != nil {
		t.Errorf("GetPersonalLearnings() error: %v", err)
	}

	// Learnings should not be nil (may be empty slice)
	if learnings == nil {
		t.Error("GetPersonalLearnings() should return non-nil slice")
	}
}

func TestOpenGlobalAddPersonalLearning(t *testing.T) {
	gc, err := OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal() error: %v", err)
	}
	defer gc.Close()

	// Generate a unique ID for the learning
	testID := "test-learning-coverage-" + time.Now().Format("20060102150405")

	// Add a personal learning
	learning := PersonalLearning{
		ID:              testID,
		Content:         "Test personal learning for coverage",
		OriginWorkspace: "test-workspace",
		Source:          "manual",
		Confidence:      0.8,
		Tags:            []string{"test", "coverage"},
	}

	err = gc.AddPersonalLearning(learning)
	if err != nil {
		t.Fatalf("AddPersonalLearning() error: %v", err)
	}

	// Clean up: delete the learning we just added
	if err := gc.DeleteLearning(testID); err != nil {
		t.Errorf("DeleteLearning() cleanup error: %v", err)
	}
}

func TestOpenGlobalReinforceLearning(t *testing.T) {
	gc, err := OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal() error: %v", err)
	}
	defer gc.Close()

	// Generate a unique ID
	testID := "test-reinforce-" + time.Now().Format("20060102150405")

	// First add a learning to reinforce
	learning := PersonalLearning{
		ID:              testID,
		Content:         "Reinforcement test learning",
		OriginWorkspace: "test-workspace",
		Source:          "manual",
		Confidence:      0.5,
	}

	err = gc.AddPersonalLearning(learning)
	if err != nil {
		t.Fatalf("AddPersonalLearning() error: %v", err)
	}

	// Reinforce it
	if err := gc.ReinforceLearning(testID); err != nil {
		t.Errorf("ReinforceLearning() error: %v", err)
	}

	// Clean up
	if err := gc.DeleteLearning(testID); err != nil {
		t.Errorf("DeleteLearning() cleanup error: %v", err)
	}
}

func TestOpenGlobalDeleteLearning(t *testing.T) {
	gc, err := OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal() error: %v", err)
	}
	defer gc.Close()

	// Generate a unique ID
	testID := "test-delete-" + time.Now().Format("20060102150405")

	// First add a learning
	learning := PersonalLearning{
		ID:              testID,
		Content:         "Delete test learning",
		OriginWorkspace: "test-workspace",
		Source:          "manual",
		Confidence:      0.6,
	}

	err = gc.AddPersonalLearning(learning)
	if err != nil {
		t.Fatalf("AddPersonalLearning() error: %v", err)
	}

	// Delete it
	if err := gc.DeleteLearning(testID); err != nil {
		t.Errorf("DeleteLearning() error: %v", err)
	}
}

func TestOpenGlobalLinkUnlink(t *testing.T) {
	gc, err := OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal() error: %v", err)
	}
	defer gc.Close()

	tmpDir := t.TempDir()
	workspaceName := "test-link-workspace"

	// Create .palace directory to make it a valid workspace
	palaceDir := tmpDir + "/.palace"
	if err := os.MkdirAll(palaceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}

	// Link a workspace
	if err := gc.Link(workspaceName, tmpDir); err != nil {
		t.Fatalf("Link() error: %v", err)
	}

	// Verify it's in the links
	links, err := gc.GetLinks()
	if err != nil {
		t.Fatalf("GetLinks() error: %v", err)
	}

	found := false
	for _, link := range links {
		if link.Name == workspaceName {
			found = true
			break
		}
	}
	if !found {
		t.Error("Link() did not add workspace to links")
	}

	// Unlink
	if err := gc.Unlink(workspaceName); err != nil {
		t.Errorf("Unlink() error: %v", err)
	}
}

func TestOpenGlobalGetLinkedLearnings(t *testing.T) {
	gc, err := OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal() error: %v", err)
	}
	defer gc.Close()

	// Get learnings from a non-existent link - should error
	_, err = gc.GetLinkedLearnings("nonexistent-workspace", 10)
	// Expect an error for non-existent workspace
	if err == nil {
		t.Error("GetLinkedLearnings() should error for non-existent workspace")
	}
}

func TestOpenGlobalGetAllLinkedLearnings(t *testing.T) {
	gc, err := OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal() error: %v", err)
	}
	defer gc.Close()

	learnings, err := gc.GetAllLinkedLearnings(10)
	if err != nil {
		t.Errorf("GetAllLinkedLearnings() error: %v", err)
	}

	// The result may be empty or nil if no linked workspaces exist
	_ = learnings
}
