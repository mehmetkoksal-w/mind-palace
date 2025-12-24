package corridor

import (
	"testing"
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
