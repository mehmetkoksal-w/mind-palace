package memory

import (
	"math"
	"os"
	"testing"
)

func TestDefaultEmbeddingConfig(t *testing.T) {
	cfg := DefaultEmbeddingConfig()
	if cfg.Backend != "disabled" {
		t.Errorf("Expected backend 'disabled', got %s", cfg.Backend)
	}
}

func TestNewEmbedderDisabled(t *testing.T) {
	cfg := EmbeddingConfig{Backend: "disabled"}
	embedder, err := NewEmbedder(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if embedder != nil {
		t.Error("Expected nil embedder for disabled backend")
	}
}

func TestNewEmbedderOllama(t *testing.T) {
	cfg := EmbeddingConfig{
		Backend: "ollama",
		Model:   "nomic-embed-text",
		URL:     "http://localhost:11434",
	}
	embedder, err := NewEmbedder(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if embedder == nil {
		t.Fatal("Expected embedder, got nil")
	}
	if embedder.Model() != "nomic-embed-text" {
		t.Errorf("Expected model 'nomic-embed-text', got %s", embedder.Model())
	}
}

func TestNewEmbedderOpenAIMissingKey(t *testing.T) {
	// Clear env var if set
	oldKey := os.Getenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	defer func() {
		if oldKey != "" {
			os.Setenv("OPENAI_API_KEY", oldKey)
		}
	}()

	cfg := EmbeddingConfig{
		Backend: "openai",
		Model:   "text-embedding-3-small",
	}
	_, err := NewEmbedder(cfg)
	if err == nil {
		t.Error("Expected error for missing API key")
	}
}

func TestNewEmbedderOpenAIWithKey(t *testing.T) {
	cfg := EmbeddingConfig{
		Backend: "openai",
		Model:   "text-embedding-3-small",
		APIKey:  "test-key",
	}
	embedder, err := NewEmbedder(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if embedder == nil {
		t.Fatal("Expected embedder, got nil")
	}
	if embedder.Model() != "text-embedding-3-small" {
		t.Errorf("Expected model 'text-embedding-3-small', got %s", embedder.Model())
	}
}

func TestNewEmbedderUnknown(t *testing.T) {
	cfg := EmbeddingConfig{Backend: "unknown"}
	_, err := NewEmbedder(cfg)
	if err == nil {
		t.Error("Expected error for unknown backend")
	}
}

func TestStoreAndGetEmbedding(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "embedding-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}

	err := mem.StoreEmbedding("i_test123", "idea", embedding, "test-model")
	if err != nil {
		t.Fatalf("Failed to store embedding: %v", err)
	}

	retrieved, err := mem.GetEmbedding("i_test123")
	if err != nil {
		t.Fatalf("Failed to get embedding: %v", err)
	}

	if len(retrieved) != len(embedding) {
		t.Fatalf("Expected %d floats, got %d", len(embedding), len(retrieved))
	}

	for i := range embedding {
		if math.Abs(float64(retrieved[i]-embedding[i])) > 0.0001 {
			t.Errorf("Index %d: expected %f, got %f", i, embedding[i], retrieved[i])
		}
	}
}

func TestDeleteEmbedding(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "embedding-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	mem.StoreEmbedding("i_test", "idea", []float32{0.1, 0.2}, "model")
	mem.DeleteEmbedding("i_test")

	_, err := mem.GetEmbedding("i_test")
	if err == nil {
		t.Error("Expected error getting deleted embedding")
	}
}

func TestGetAllEmbeddings(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "embedding-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	mem.StoreEmbedding("i_1", "idea", []float32{0.1, 0.2}, "model")
	mem.StoreEmbedding("i_2", "idea", []float32{0.3, 0.4}, "model")
	mem.StoreEmbedding("d_1", "decision", []float32{0.5, 0.6}, "model")

	ideas, err := mem.GetAllEmbeddings("idea")
	if err != nil {
		t.Fatalf("Failed to get embeddings: %v", err)
	}

	if len(ideas) != 2 {
		t.Errorf("Expected 2 idea embeddings, got %d", len(ideas))
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float32
	}{
		{
			name:     "identical vectors",
			a:        []float32{1, 0, 0},
			b:        []float32{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1, 0},
			b:        []float32{0, 1},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float32{1, 0},
			b:        []float32{-1, 0},
			expected: -1.0,
		},
		{
			name:     "similar vectors",
			a:        []float32{0.8, 0.6},
			b:        []float32{0.9, 0.5},
			expected: 0.9922, // approximately
		},
		{
			name:     "different lengths",
			a:        []float32{1, 0, 0},
			b:        []float32{1, 0},
			expected: 0.0, // invalid
		},
		{
			name:     "empty vectors",
			a:        []float32{},
			b:        []float32{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.a, tt.b)
			if math.Abs(float64(result-tt.expected)) > 0.01 {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestFindSimilarEmbeddings(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "embedding-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Store some embeddings
	mem.StoreEmbedding("i_1", "idea", []float32{0.9, 0.1, 0.0}, "model")
	mem.StoreEmbedding("i_2", "idea", []float32{0.8, 0.2, 0.1}, "model")
	mem.StoreEmbedding("i_3", "idea", []float32{0.1, 0.9, 0.0}, "model")
	mem.StoreEmbedding("d_1", "decision", []float32{0.85, 0.15, 0.05}, "model")

	// Query similar to i_1 and i_2
	query := []float32{0.85, 0.15, 0.05}

	results, err := mem.FindSimilarEmbeddings(query, "idea", 10, 0.0)
	if err != nil {
		t.Fatalf("Failed to find similar: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// First result should be most similar (i_1 or i_2)
	if results[0].Similarity < 0.9 {
		t.Errorf("Expected high similarity for first result, got %f", results[0].Similarity)
	}

	// With min similarity filter - very high threshold should filter most out
	filtered, _ := mem.FindSimilarEmbeddings(query, "idea", 10, 0.999)
	if len(filtered) > 1 {
		t.Errorf("Expected at most 1 result with very high threshold, got %d", len(filtered))
	}
}

func TestCountEmbeddings(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "embedding-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	count, _ := mem.CountEmbeddings()
	if count != 0 {
		t.Errorf("Expected 0 embeddings, got %d", count)
	}

	mem.StoreEmbedding("i_1", "idea", []float32{0.1}, "model")
	mem.StoreEmbedding("i_2", "idea", []float32{0.2}, "model")

	count, _ = mem.CountEmbeddings()
	if count != 2 {
		t.Errorf("Expected 2 embeddings, got %d", count)
	}
}

func TestFloat32Conversion(t *testing.T) {
	original := []float32{0.123456, -0.789012, 1.234567, -2.345678}

	bytes := float32sToBytes(original)
	recovered := bytesToFloat32s(bytes)

	if len(recovered) != len(original) {
		t.Fatalf("Length mismatch: %d vs %d", len(recovered), len(original))
	}

	for i := range original {
		if math.Abs(float64(original[i]-recovered[i])) > 0.000001 {
			t.Errorf("Index %d: expected %f, got %f", i, original[i], recovered[i])
		}
	}
}

func TestBytesToFloat32sEmpty(t *testing.T) {
	result := bytesToFloat32s(nil)
	if result != nil {
		t.Error("Expected nil for empty input")
	}

	result = bytesToFloat32s([]byte{})
	if result != nil {
		t.Error("Expected nil for empty slice")
	}
}
