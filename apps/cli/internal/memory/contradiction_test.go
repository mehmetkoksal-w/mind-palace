package memory

import (
	"os"
	"testing"
)

func TestStubContradictionAnalyzer(t *testing.T) {
	analyzer := NewStubContradictionAnalyzer()

	r1 := RecordForAnalysis{
		ID:      "i_1",
		Kind:    "idea",
		Content: "Use JWT for authentication",
	}
	r2 := RecordForAnalysis{
		ID:      "i_2",
		Kind:    "idea",
		Content: "Use session cookies for authentication",
	}

	result, err := analyzer.AnalyzeContradiction(r1, r2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Record1ID != "i_1" {
		t.Errorf("Expected record1 ID 'i_1', got %s", result.Record1ID)
	}
	if result.Record2ID != "i_2" {
		t.Errorf("Expected record2 ID 'i_2', got %s", result.Record2ID)
	}
	// Stub should indicate it needs LLM
	if result.Explanation == "" {
		t.Error("Expected explanation from stub analyzer")
	}
}

func TestStubFindContradictions(t *testing.T) {
	analyzer := NewStubContradictionAnalyzer()

	record := RecordForAnalysis{
		ID:      "i_main",
		Kind:    "idea",
		Content: "Main idea",
	}

	candidates := []RecordForAnalysis{
		{ID: "i_1", Kind: "idea", Content: "Candidate 1"},
		{ID: "i_2", Kind: "idea", Content: "Candidate 2"},
		{ID: "i_main", Kind: "idea", Content: "Main idea"}, // Self - should be skipped
	}

	results, err := analyzer.FindContradictions(record, candidates)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should skip self
	if len(results) != 2 {
		t.Errorf("Expected 2 results (excluding self), got %d", len(results))
	}
}

func TestDefaultContradictionOptions(t *testing.T) {
	opts := DefaultContradictionOptions()

	if !opts.UseEmbeddings {
		t.Error("Expected UseEmbeddings to be true")
	}
	if opts.MinSimilarity != 0.6 {
		t.Errorf("Expected MinSimilarity 0.6, got %f", opts.MinSimilarity)
	}
	if !opts.IncludeIdeas {
		t.Error("Expected IncludeIdeas to be true")
	}
	if !opts.IncludeDecisions {
		t.Error("Expected IncludeDecisions to be true")
	}
	if opts.IncludeLearnings {
		t.Error("Expected IncludeLearnings to be false")
	}
}

func TestFindPotentialContradictionsByFTS(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "contradiction-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add some ideas
	mem.AddIdea(Idea{Content: "Use JWT authentication"})
	mem.AddIdea(Idea{Content: "Use session cookies for authentication"})
	mem.AddIdea(Idea{Content: "Database migration plan"}) // Unrelated

	opts := DefaultContradictionOptions()
	opts.UseEmbeddings = false // Force FTS

	candidates, err := mem.FindPotentialContradictions("authentication", nil, opts)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should find the two authentication-related ideas
	if len(candidates) < 2 {
		t.Errorf("Expected at least 2 candidates, got %d", len(candidates))
	}
}

func TestFindPotentialContradictionsWithEmbeddings(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "contradiction-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Custom embedder that returns auth-like embedding for auth-related queries
	embedder := &customMockEmbedder{
		queryResponse: []float32{0.9, 0.1, 0.0, 0.0}, // auth-like
	}

	// Add ideas with embeddings
	id1, _ := mem.AddIdea(Idea{Content: "Use JWT authentication"})
	mem.StoreEmbedding(id1, "idea", []float32{0.9, 0.1, 0.0, 0.0}, "mock") // Similar

	id2, _ := mem.AddIdea(Idea{Content: "Use session cookies"})
	mem.StoreEmbedding(id2, "idea", []float32{0.85, 0.15, 0.0, 0.0}, "mock") // Similar

	id3, _ := mem.AddIdea(Idea{Content: "Database migration"})
	mem.StoreEmbedding(id3, "idea", []float32{0.0, 0.0, 0.9, 0.0}, "mock") // Different

	opts := DefaultContradictionOptions()
	opts.MinSimilarity = 0.5

	candidates, err := mem.FindPotentialContradictions("authentication", embedder, opts)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should find similar ideas (at least the auth-related ones)
	if len(candidates) < 2 {
		t.Errorf("Expected at least 2 candidates, got %d", len(candidates))
	}
}

type customMockEmbedder struct {
	queryResponse []float32
}

func (e *customMockEmbedder) Embed(text string) ([]float32, error) {
	return e.queryResponse, nil
}

func (e *customMockEmbedder) Model() string {
	return "custom-mock"
}

func TestGetContradictingRecords(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "contradiction-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Create ideas
	id1, _ := mem.AddIdea(Idea{Content: "Use JWT"})
	id2, _ := mem.AddIdea(Idea{Content: "Use sessions"})
	id3, _ := mem.AddIdea(Idea{Content: "Unrelated idea"})

	// Create contradiction links
	mem.AddLink(Link{
		SourceID:   id1,
		SourceKind: "idea",
		TargetID:   id2,
		TargetKind: "idea",
		Relation:   RelationContradicts,
	})

	// Also create a non-contradiction link
	mem.AddLink(Link{
		SourceID:   id1,
		SourceKind: "idea",
		TargetID:   id3,
		TargetKind: "idea",
		Relation:   RelationRelated,
	})

	contradicting, err := mem.GetContradictingRecords(id1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(contradicting) != 1 {
		t.Errorf("Expected 1 contradicting record, got %d", len(contradicting))
	}

	if len(contradicting) > 0 && contradicting[0].ID != id2 {
		t.Errorf("Expected contradicting record %s, got %s", id2, contradicting[0].ID)
	}
}

func TestGetContradictingRecordsBidirectional(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "contradiction-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id1, _ := mem.AddIdea(Idea{Content: "Idea A"})
	id2, _ := mem.AddIdea(Idea{Content: "Idea B"})

	// B contradicts A (incoming link for A)
	mem.AddLink(Link{
		SourceID:   id2,
		SourceKind: "idea",
		TargetID:   id1,
		TargetKind: "idea",
		Relation:   RelationContradicts,
	})

	// Query from A's perspective - should find B
	contradicting, _ := mem.GetContradictingRecords(id1)
	if len(contradicting) != 1 {
		t.Errorf("Expected 1 contradicting record, got %d", len(contradicting))
	}
}

func TestGetContradictionSummary(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "contradiction-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Create ideas with different statuses
	id1, _ := mem.AddIdea(Idea{Content: "Active idea 1", Status: "active"})
	id2, _ := mem.AddIdea(Idea{Content: "Active idea 2", Status: "active"})
	id3, _ := mem.AddIdea(Idea{Content: "Archived idea", Status: "archived"})

	// Create contradiction links
	mem.AddLink(Link{
		SourceID: id1, SourceKind: "idea",
		TargetID: id2, TargetKind: "idea",
		Relation: RelationContradicts,
	})
	mem.AddLink(Link{
		SourceID: id1, SourceKind: "idea",
		TargetID: id3, TargetKind: "idea",
		Relation: RelationContradicts,
	})

	summary, err := mem.GetContradictionSummary(10)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if summary.TotalContradictionLinks != 2 {
		t.Errorf("Expected 2 total links, got %d", summary.TotalContradictionLinks)
	}

	if len(summary.TopContradictions) != 2 {
		t.Errorf("Expected 2 top contradictions, got %d", len(summary.TopContradictions))
	}
}

func TestRecordForAnalysis(t *testing.T) {
	record := RecordForAnalysis{
		ID:      "i_test",
		Kind:    "idea",
		Content: "Test content",
		Context: "Test context",
		Status:  "active",
	}

	if record.ID != "i_test" {
		t.Error("ID mismatch")
	}
	if record.Kind != "idea" {
		t.Error("Kind mismatch")
	}
}

func TestContradictionResult(t *testing.T) {
	result := ContradictionResult{
		Record1ID:       "i_1",
		Record2ID:       "i_2",
		IsContradiction: true,
		Confidence:      0.9,
		Explanation:     "Direct conflict",
		ContradictType:  "direct",
	}

	if !result.IsContradiction {
		t.Error("Expected IsContradiction to be true")
	}
	if result.Confidence != 0.9 {
		t.Errorf("Expected confidence 0.9, got %f", result.Confidence)
	}
}
