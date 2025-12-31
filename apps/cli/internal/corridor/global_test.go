package corridor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// Note: These tests use the real home directory via OpenGlobal().
// For CI environments, you may want to mock the home directory
// or skip these tests with t.Skip().

func TestPersonalLearning(t *testing.T) {
	// Skip if we can't access home directory
	gc, err := OpenGlobal()
	if err != nil {
		t.Skipf("Skipping test: cannot open global corridor: %v", err)
	}
	defer gc.Close()

	// Add a test learning
	learning := PersonalLearning{
		Content:         "Test learning for corridor test",
		Confidence:      0.8,
		Source:          "test",
		OriginWorkspace: "test-workspace",
	}

	err = gc.AddPersonalLearning(learning)
	if err != nil {
		t.Fatalf("Failed to add personal learning: %v", err)
	}

	// Get learnings
	learnings, err := gc.GetPersonalLearnings("", 100)
	if err != nil {
		t.Fatalf("Failed to get personal learnings: %v", err)
	}

	// Find our test learning
	found := false
	for _, l := range learnings {
		if l.Content == "Test learning for corridor test" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Test learning was not found in results")
	}

	// Search learnings
	searchResults, err := gc.GetPersonalLearnings("corridor test", 10)
	if err != nil {
		t.Fatalf("Failed to search personal learnings: %v", err)
	}

	if len(searchResults) == 0 {
		t.Error("Expected at least one search result")
	}
}

func TestStats(t *testing.T) {
	gc, err := OpenGlobal()
	if err != nil {
		t.Skipf("Skipping test: cannot open global corridor: %v", err)
	}
	defer gc.Close()

	stats, err := gc.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	// Verify stats has expected keys
	if stats == nil {
		t.Fatal("Stats returned nil")
	}

	// Check that learningCount is present (could be 0)
	if _, ok := stats["learningCount"]; !ok {
		t.Error("Stats missing 'learningCount' key")
	}

	// Check linkedWorkspaces
	if _, ok := stats["linkedWorkspaces"]; !ok {
		t.Error("Stats missing 'linkedWorkspaces' key")
	}
}

func TestGetLinks(t *testing.T) {
	gc, err := OpenGlobal()
	if err != nil {
		t.Skipf("Skipping test: cannot open global corridor: %v", err)
	}
	defer gc.Close()

	// GetLinks should not error even with no links
	links, err := gc.GetLinks()
	if err != nil {
		t.Fatalf("Failed to get links: %v", err)
	}

	// links could be empty, that's fine
	_ = links
}

func TestGlobalCorridorLinksAndLearnings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	gc, err := OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal() error = %v", err)
	}
	defer gc.Close()

	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, ".palace"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if err := gc.Link("ws1", workspace); err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	links, err := gc.GetLinks()
	if err != nil || len(links) == 0 {
		t.Fatalf("GetLinks() = %v, err = %v", links, err)
	}

	if err := gc.Unlink("ws1"); err != nil {
		t.Fatalf("Unlink() error = %v", err)
	}
	if err := gc.Unlink("ws1"); err == nil {
		t.Fatalf("expected error unlinking missing link")
	}
}

func TestLinkedLearningsAndPromotion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	gc, err := OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal() error = %v", err)
	}
	defer gc.Close()

	workspace := t.TempDir()
	mem, err := memory.Open(workspace)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	defer mem.Close()

	_, err = mem.AddLearning(memory.Learning{
		Scope:      "palace",
		Content:    "shared learning",
		Confidence: 0.9,
		UseCount:   3,
	})
	if err != nil {
		t.Fatalf("AddLearning() error = %v", err)
	}

	if err := gc.Link("ws1", workspace); err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	linked, err := gc.GetLinkedLearnings("ws1", 10)
	if err != nil || len(linked) == 0 {
		t.Fatalf("GetLinkedLearnings() = %v, err = %v", linked, err)
	}

	allLinked, err := gc.GetAllLinkedLearnings(10)
	if err != nil || len(allLinked) == 0 {
		t.Fatalf("GetAllLinkedLearnings() = %v, err = %v", allLinked, err)
	}

	promoted, err := gc.AutoPromote("ws1", mem)
	if err != nil || len(promoted) == 0 {
		t.Fatalf("AutoPromote() = %v, err = %v", promoted, err)
	}

	if err := gc.PromoteFromWorkspace("ws1", memory.Learning{ID: "manual", Content: "manual learning", Confidence: 0.9}); err != nil {
		t.Fatalf("PromoteFromWorkspace() error = %v", err)
	}

	personal, err := gc.GetPersonalLearnings("shared", 10)
	if err != nil || len(personal) == 0 {
		t.Fatalf("GetPersonalLearnings() = %v, err = %v", personal, err)
	}

	if err := gc.ReinforceLearning(personal[0].ID); err != nil {
		t.Fatalf("ReinforceLearning() error = %v", err)
	}
	if err := gc.DeleteLearning(personal[0].ID); err != nil {
		t.Fatalf("DeleteLearning() error = %v", err)
	}
}

func TestValidateAndPruneLinks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	gc, err := OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal() error = %v", err)
	}
	defer gc.Close()

	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, ".palace"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if err := gc.Link("stale", workspace); err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	stale, err := gc.ValidateLinks()
	if err != nil || len(stale) == 0 {
		t.Fatalf("ValidateLinks() = %v, err = %v", stale, err)
	}

	pruned, err := gc.PruneStaleLinks()
	if err != nil || len(pruned) == 0 {
		t.Fatalf("PruneStaleLinks() = %v, err = %v", pruned, err)
	}
}
