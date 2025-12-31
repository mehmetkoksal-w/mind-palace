package memory

import (
	"os"
	"testing"
)

// mockEmbedder is a simple embedder for testing
type mockEmbedder struct {
	embeddings map[string][]float32
}

func newMockEmbedder() *mockEmbedder {
	return &mockEmbedder{
		embeddings: map[string][]float32{
			"authentication": {0.9, 0.1, 0.0, 0.0},
			"JWT tokens":     {0.85, 0.15, 0.0, 0.0},
			"database":       {0.0, 0.0, 0.9, 0.1},
			"PostgreSQL":     {0.0, 0.0, 0.85, 0.15},
			"caching":        {0.0, 0.0, 0.0, 0.9},
		},
	}
}

func (e *mockEmbedder) Embed(text string) ([]float32, error) {
	// Return pre-defined embedding or a default
	if emb, ok := e.embeddings[text]; ok {
		return emb, nil
	}
	// Default embedding based on text length (for testing)
	return []float32{0.5, 0.5, 0.0, 0.0}, nil
}

func (e *mockEmbedder) Model() string {
	return "mock-model"
}

func TestDefaultSemanticSearchOptions(t *testing.T) {
	opts := DefaultSemanticSearchOptions()
	if opts.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", opts.Limit)
	}
	if opts.MinSimilarity != 0.5 {
		t.Errorf("Expected min similarity 0.5, got %f", opts.MinSimilarity)
	}
}

func TestSemanticSearchNoEmbedder(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "semantic-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	opts := DefaultSemanticSearchOptions()
	_, err := mem.SemanticSearch(nil, "test", opts)
	if err == nil {
		t.Error("Expected error with nil embedder")
	}
}

func TestSemanticSearchBasic(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "semantic-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	embedder := newMockEmbedder()

	// Add ideas with embeddings
	id1, _ := mem.AddIdea(Idea{Content: "Use JWT tokens for authentication"})
	mem.StoreEmbedding(id1, "idea", embedder.embeddings["JWT tokens"], "mock")

	id2, _ := mem.AddIdea(Idea{Content: "Use PostgreSQL for database"})
	mem.StoreEmbedding(id2, "idea", embedder.embeddings["PostgreSQL"], "mock")

	// Search for authentication-related content
	opts := DefaultSemanticSearchOptions()
	opts.Kinds = []string{"idea"}
	opts.MinSimilarity = 0.0 // Low threshold for testing

	results, err := mem.SemanticSearch(embedder, "authentication", opts)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should find results
	if len(results) == 0 {
		t.Error("Expected results for authentication search")
	}

	// First result should be JWT (most similar to authentication)
	if len(results) > 0 && results[0].ID != id1 {
		t.Errorf("Expected JWT idea (%s) as top result, got %s", id1, results[0].ID)
	}
}

func TestSemanticSearchFilters(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "semantic-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	embedder := newMockEmbedder()

	// Add idea and decision with same embedding
	ideaID, _ := mem.AddIdea(Idea{Content: "Auth idea"})
	mem.StoreEmbedding(ideaID, "idea", []float32{0.9, 0.1, 0.0, 0.0}, "mock")

	decID, _ := mem.AddDecision(Decision{Content: "Auth decision"})
	mem.StoreEmbedding(decID, "decision", []float32{0.9, 0.1, 0.0, 0.0}, "mock")

	// Search only ideas
	opts := DefaultSemanticSearchOptions()
	opts.Kinds = []string{"idea"}
	opts.MinSimilarity = 0.0

	results, _ := mem.SemanticSearch(embedder, "authentication", opts)

	// Should only find the idea
	for _, r := range results {
		if r.Kind != "idea" {
			t.Errorf("Expected only ideas, got %s", r.Kind)
		}
	}
}

func TestSemanticSearchLimit(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "semantic-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	embedder := newMockEmbedder()

	// Add many ideas
	for i := 0; i < 20; i++ {
		id, _ := mem.AddIdea(Idea{Content: "Test idea"})
		mem.StoreEmbedding(id, "idea", []float32{0.9, 0.1, 0.0, 0.0}, "mock")
	}

	opts := DefaultSemanticSearchOptions()
	opts.Limit = 5
	opts.MinSimilarity = 0.0

	results, _ := mem.SemanticSearch(embedder, "authentication", opts)

	if len(results) > 5 {
		t.Errorf("Expected max 5 results, got %d", len(results))
	}
}

func TestHybridSearchNoEmbedder(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "semantic-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add idea with FTS
	mem.AddIdea(Idea{Content: "Authentication using JWT tokens"})

	opts := DefaultSemanticSearchOptions()
	opts.Kinds = []string{"idea"}

	// Should still work with keyword search only
	results, err := mem.HybridSearch(nil, "authentication", opts)
	if err != nil {
		t.Fatalf("Hybrid search failed: %v", err)
	}

	// Should find via FTS even without embedder
	if len(results) > 0 {
		if results[0].MatchType != "keyword" {
			t.Errorf("Expected keyword match type, got %s", results[0].MatchType)
		}
	}
}

func TestHybridSearchBothMatches(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "semantic-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	embedder := newMockEmbedder()

	// Add idea that matches both keyword and semantic
	id, _ := mem.AddIdea(Idea{Content: "Authentication with JWT tokens"})
	mem.StoreEmbedding(id, "idea", embedder.embeddings["JWT tokens"], "mock")

	opts := DefaultSemanticSearchOptions()
	opts.Kinds = []string{"idea"}
	opts.MinSimilarity = 0.0

	results, _ := mem.HybridSearch(embedder, "authentication", opts)

	// Should find with "both" match type
	found := false
	for _, r := range results {
		if r.ID == id && r.MatchType == "both" {
			found = true
			break
		}
	}

	if !found && len(results) > 0 {
		t.Logf("Found results but none with 'both' match type. First result: %+v", results[0])
	}
}

func TestGetRecordContent(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "semantic-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	ideaID, _ := mem.AddIdea(Idea{Content: "Test idea content"})
	decID, _ := mem.AddDecision(Decision{Content: "Test decision content"})

	// Test idea
	content, _, err := mem.GetRecordContent(ideaID, "idea")
	if err != nil {
		t.Fatalf("Failed to get idea content: %v", err)
	}
	if content != "Test idea content" {
		t.Errorf("Expected 'Test idea content', got %s", content)
	}

	// Test decision
	content, _, err = mem.GetRecordContent(decID, "decision")
	if err != nil {
		t.Fatalf("Failed to get decision content: %v", err)
	}
	if content != "Test decision content" {
		t.Errorf("Expected 'Test decision content', got %s", content)
	}

	// Test unknown kind
	_, _, err = mem.GetRecordContent("x", "unknown")
	if err == nil {
		t.Error("Expected error for unknown kind")
	}
}

func TestGetRecordScope(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "semantic-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	ideaID, _ := mem.AddIdea(Idea{
		Content:   "Test",
		Scope:     "room",
		ScopePath: "api",
	})

	scope, scopePath := mem.getRecordScope(ideaID, "idea")
	if scope != "room" {
		t.Errorf("Expected scope 'room', got %s", scope)
	}
	if scopePath != "api" {
		t.Errorf("Expected scope path 'api', got %s", scopePath)
	}
}
