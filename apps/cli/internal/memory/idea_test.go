package memory

import (
	"os"
	"strings"
	"testing"
)

func TestAddIdea(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "idea-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mem, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	defer mem.Close()

	// Test adding a basic idea
	idea := Idea{
		Content: "What if we used GraphQL instead of REST?",
		Context: "Discussing API design",
	}
	id, err := mem.AddIdea(idea)
	if err != nil {
		t.Fatalf("Failed to add idea: %v", err)
	}
	if !strings.HasPrefix(id, "i_") {
		t.Errorf("Expected ID to start with 'i_', got %s", id)
	}

	// Verify defaults
	retrieved, err := mem.GetIdea(id)
	if err != nil {
		t.Fatalf("Failed to get idea: %v", err)
	}
	if retrieved.Status != IdeaStatusActive {
		t.Errorf("Expected status 'active', got '%s'", retrieved.Status)
	}
	if retrieved.Scope != "palace" {
		t.Errorf("Expected scope 'palace', got '%s'", retrieved.Scope)
	}
	if retrieved.Source != "cli" {
		t.Errorf("Expected source 'cli', got '%s'", retrieved.Source)
	}
}

func TestAddIdeaWithScope(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "idea-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Test idea with file scope
	idea := Idea{
		Content:   "This file needs caching",
		Scope:     "file",
		ScopePath: "api/handler.go",
		Status:    IdeaStatusExploring,
	}
	id, err := mem.AddIdea(idea)
	if err != nil {
		t.Fatalf("Failed to add idea: %v", err)
	}

	retrieved, _ := mem.GetIdea(id)
	if retrieved.Scope != "file" {
		t.Errorf("Expected scope 'file', got '%s'", retrieved.Scope)
	}
	if retrieved.ScopePath != "api/handler.go" {
		t.Errorf("Expected scope_path 'api/handler.go', got '%s'", retrieved.ScopePath)
	}
	if retrieved.Status != IdeaStatusExploring {
		t.Errorf("Expected status 'exploring', got '%s'", retrieved.Status)
	}
}

func TestGetIdeas(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "idea-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add multiple ideas
	ideas := []Idea{
		{Content: "Idea 1", Status: IdeaStatusActive, Scope: "palace"},
		{Content: "Idea 2", Status: IdeaStatusExploring, Scope: "palace"},
		{Content: "Idea 3", Status: IdeaStatusActive, Scope: "room", ScopePath: "auth"},
		{Content: "Idea 4", Status: IdeaStatusDropped, Scope: "palace"},
	}
	for _, idea := range ideas {
		_, err := mem.AddIdea(idea)
		if err != nil {
			t.Fatalf("Failed to add idea: %v", err)
		}
	}

	// Get all ideas
	all, err := mem.GetIdeas("", "", "", 10)
	if err != nil {
		t.Fatalf("Failed to get ideas: %v", err)
	}
	if len(all) != 4 {
		t.Errorf("Expected 4 ideas, got %d", len(all))
	}

	// Filter by status
	active, _ := mem.GetIdeas(IdeaStatusActive, "", "", 10)
	if len(active) != 2 {
		t.Errorf("Expected 2 active ideas, got %d", len(active))
	}

	// Filter by scope
	roomIdeas, _ := mem.GetIdeas("", "room", "", 10)
	if len(roomIdeas) != 1 {
		t.Errorf("Expected 1 room idea, got %d", len(roomIdeas))
	}

	// Filter by scope path
	authIdeas, _ := mem.GetIdeas("", "room", "auth", 10)
	if len(authIdeas) != 1 {
		t.Errorf("Expected 1 auth room idea, got %d", len(authIdeas))
	}

	// Test limit
	limited, _ := mem.GetIdeas("", "", "", 2)
	if len(limited) != 2 {
		t.Errorf("Expected 2 ideas with limit, got %d", len(limited))
	}
}

func TestSearchIdeas(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "idea-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add ideas with different content
	ideas := []Idea{
		{Content: "Use GraphQL for the API", Context: "API design discussion"},
		{Content: "Add caching layer", Context: "Performance optimization"},
		{Content: "Implement rate limiting", Context: "API security"},
	}
	for _, idea := range ideas {
		mem.AddIdea(idea)
	}

	// Search by content
	results, err := mem.SearchIdeas("API", 10)
	if err != nil {
		t.Fatalf("Failed to search ideas: %v", err)
	}
	if len(results) < 1 {
		t.Errorf("Expected at least 1 result for 'API', got %d", len(results))
	}

	// Search with no results
	noResults, _ := mem.SearchIdeas("nonexistent", 10)
	if len(noResults) != 0 {
		t.Errorf("Expected 0 results for 'nonexistent', got %d", len(noResults))
	}
}

func TestUpdateIdeaStatus(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "idea-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddIdea(Idea{Content: "Test idea"})

	// Update status
	err := mem.UpdateIdeaStatus(id, IdeaStatusImplemented)
	if err != nil {
		t.Fatalf("Failed to update idea status: %v", err)
	}

	idea, _ := mem.GetIdea(id)
	if idea.Status != IdeaStatusImplemented {
		t.Errorf("Expected status 'implemented', got '%s'", idea.Status)
	}

	// Verify updated_at changed
	if idea.UpdatedAt.IsZero() {
		t.Error("Expected updated_at to be set")
	}

	// Update non-existent idea
	err = mem.UpdateIdeaStatus("nonexistent", IdeaStatusActive)
	if err == nil {
		t.Error("Expected error for non-existent idea")
	}
}

func TestUpdateIdea(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "idea-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddIdea(Idea{Content: "Original content", Context: "Original context"})

	// Update content and context
	err := mem.UpdateIdea(id, "Updated content", "Updated context")
	if err != nil {
		t.Fatalf("Failed to update idea: %v", err)
	}

	idea, _ := mem.GetIdea(id)
	if idea.Content != "Updated content" {
		t.Errorf("Expected content 'Updated content', got '%s'", idea.Content)
	}
	if idea.Context != "Updated context" {
		t.Errorf("Expected context 'Updated context', got '%s'", idea.Context)
	}

	// Update non-existent idea
	err = mem.UpdateIdea("nonexistent", "content", "context")
	if err == nil {
		t.Error("Expected error for non-existent idea")
	}
}

func TestDeleteIdea(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "idea-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddIdea(Idea{Content: "To be deleted"})

	// Delete the idea
	err := mem.DeleteIdea(id)
	if err != nil {
		t.Fatalf("Failed to delete idea: %v", err)
	}

	// Verify it's gone
	_, err = mem.GetIdea(id)
	if err == nil {
		t.Error("Expected error getting deleted idea")
	}
}

func TestCountIdeas(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "idea-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add ideas with different statuses
	mem.AddIdea(Idea{Content: "Active 1", Status: IdeaStatusActive})
	mem.AddIdea(Idea{Content: "Active 2", Status: IdeaStatusActive})
	mem.AddIdea(Idea{Content: "Dropped", Status: IdeaStatusDropped})

	// Count all
	total, err := mem.CountIdeas("")
	if err != nil {
		t.Fatalf("Failed to count ideas: %v", err)
	}
	if total != 3 {
		t.Errorf("Expected 3 total ideas, got %d", total)
	}

	// Count by status
	active, _ := mem.CountIdeas(IdeaStatusActive)
	if active != 2 {
		t.Errorf("Expected 2 active ideas, got %d", active)
	}

	dropped, _ := mem.CountIdeas(IdeaStatusDropped)
	if dropped != 1 {
		t.Errorf("Expected 1 dropped idea, got %d", dropped)
	}
}

func TestIdeaWithSession(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "idea-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Create a session first
	session, _ := mem.StartSession("claude-code", "test-agent", "Testing ideas")

	// Add idea linked to session
	idea := Idea{
		Content:   "Session-linked idea",
		SessionID: session.ID,
	}
	id, _ := mem.AddIdea(idea)

	retrieved, _ := mem.GetIdea(id)
	if retrieved.SessionID != session.ID {
		t.Errorf("Expected session ID '%s', got '%s'", session.ID, retrieved.SessionID)
	}
}

func TestIdeaFTSIntegration(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "idea-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add idea
	id, _ := mem.AddIdea(Idea{
		Content: "Implement authentication with JWT tokens",
		Context: "Security requirements discussion",
	})

	// Update idea (should update FTS via trigger)
	err := mem.UpdateIdea(id, "Implement OAuth2 authentication", "Updated security approach")
	if err != nil {
		t.Fatalf("Failed to update idea: %v", err)
	}

	// Search should find updated content
	results, _ := mem.SearchIdeas("OAuth2", 10)
	found := false
	for _, r := range results {
		if r.ID == id {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find updated idea via FTS search")
	}

	// Old content should not be found (FTS trigger should have updated)
	oldResults, _ := mem.SearchIdeas("JWT", 10)
	for _, r := range oldResults {
		if r.ID == id {
			t.Error("Found old content in FTS after update - trigger may not be working")
		}
	}
}

func TestSearchIdeasLikeFallback(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "idea-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add ideas
	mem.AddIdea(Idea{Content: "Performance optimization techniques", Context: "Backend development"})
	mem.AddIdea(Idea{Content: "API design patterns", Context: "Architecture discussion"})

	// The searchIdeasLike function is a fallback - test it directly
	results, err := mem.searchIdeasLike("optimization", 10)
	if err != nil {
		t.Fatalf("searchIdeasLike failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Search in context
	contextResults, _ := mem.searchIdeasLike("Backend", 10)
	if len(contextResults) != 1 {
		t.Errorf("Expected 1 result for context search, got %d", len(contextResults))
	}

	// No results
	noResults, _ := mem.searchIdeasLike("nonexistent", 10)
	if len(noResults) != 0 {
		t.Errorf("Expected 0 results, got %d", len(noResults))
	}
}
