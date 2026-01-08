package memory

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"time"
)

// Embedder defines the interface for generating text embeddings.
type Embedder interface {
	// Embed generates a vector embedding for the given text.
	Embed(text string) ([]float32, error)
	// Model returns the name of the model being used.
	Model() string
}

// EmbeddingConfig holds configuration for the embedding backend.
type EmbeddingConfig struct {
	Backend string `json:"backend"` // "ollama", "openai", or "disabled"
	Model   string `json:"model"`   // e.g., "nomic-embed-text", "text-embedding-3-small"
	URL     string `json:"url"`     // Base URL for the API
	APIKey  string `json:"apiKey"`  // API key (for OpenAI)
}

// DefaultEmbeddingConfig returns the default embedding configuration.
func DefaultEmbeddingConfig() EmbeddingConfig {
	return EmbeddingConfig{
		Backend: "disabled", // Disabled by default until configured
		Model:   "",
		URL:     "",
	}
}

// NewEmbedder creates an Embedder based on the configuration.
func NewEmbedder(cfg EmbeddingConfig) (Embedder, error) {
	switch cfg.Backend {
	case "ollama":
		url := cfg.URL
		if url == "" {
			url = "http://localhost:11434"
		}
		model := cfg.Model
		if model == "" {
			model = "nomic-embed-text"
		}
		return &OllamaEmbedder{
			url:   url,
			model: model,
		}, nil
	case "openai":
		apiKey := cfg.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI API key required (set apiKey or OPENAI_API_KEY)")
		}
		model := cfg.Model
		if model == "" {
			model = "text-embedding-3-small"
		}
		return &OpenAIEmbedder{
			apiKey: apiKey,
			model:  model,
		}, nil
	case "disabled", "":
		return nil, nil // No embedder
	default:
		return nil, fmt.Errorf("unknown embedding backend: %s", cfg.Backend)
	}
}

// ============================================================================
// Ollama Embedder (local, free, default)
// ============================================================================

// OllamaEmbedder generates embeddings using a local Ollama instance.
type OllamaEmbedder struct {
	url    string
	model  string
	client *http.Client
}

// Embed generates an embedding using Ollama.
func (e *OllamaEmbedder) Embed(text string) ([]float32, error) {
	if e.client == nil {
		e.client = &http.Client{Timeout: 30 * time.Second}
	}

	reqBody := map[string]string{
		"model":  e.model,
		"prompt": text,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", e.url+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}

	return result.Embedding, nil
}

// Model returns the Ollama model name.
func (e *OllamaEmbedder) Model() string {
	return e.model
}

// ============================================================================
// OpenAI Embedder
// ============================================================================

// OpenAIEmbedder generates embeddings using the OpenAI API.
type OpenAIEmbedder struct {
	apiKey string
	model  string
	client *http.Client
}

// Embed generates an embedding using OpenAI.
func (e *OpenAIEmbedder) Embed(text string) ([]float32, error) {
	if e.client == nil {
		e.client = &http.Client{Timeout: 30 * time.Second}
	}

	reqBody := map[string]interface{}{
		"model": e.model,
		"input": text,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode openai response: %w", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return result.Data[0].Embedding, nil
}

// Model returns the OpenAI model name.
func (e *OpenAIEmbedder) Model() string {
	return e.model
}

// ============================================================================
// Embedding Storage
// ============================================================================

// StoreEmbedding stores an embedding for a record.
func (m *Memory) StoreEmbedding(recordID, recordKind string, embedding []float32, model string) error {
	blob := float32sToBytes(embedding)
	_, err := m.db.ExecContext(context.Background(), `
		INSERT OR REPLACE INTO embeddings (record_id, record_kind, embedding, model, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		recordID, recordKind, blob, model, time.Now().UTC().Format(time.RFC3339))
	return err
}

// GetEmbedding retrieves the embedding for a record.
func (m *Memory) GetEmbedding(recordID string) ([]float32, error) {
	var blob []byte
	err := m.db.QueryRowContext(context.Background(), `SELECT embedding FROM embeddings WHERE record_id = ?`, recordID).Scan(&blob)
	if err != nil {
		return nil, err
	}
	return bytesToFloat32s(blob), nil
}

// DeleteEmbedding removes the embedding for a record.
func (m *Memory) DeleteEmbedding(recordID string) error {
	_, err := m.db.ExecContext(context.Background(), `DELETE FROM embeddings WHERE record_id = ?`, recordID)
	return err
}

// GetAllEmbeddings returns all embeddings of a specific kind.
func (m *Memory) GetAllEmbeddings(recordKind string) (map[string][]float32, error) {
	rows, err := m.db.QueryContext(context.Background(), `SELECT record_id, embedding FROM embeddings WHERE record_kind = ?`, recordKind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	embeddings := make(map[string][]float32)
	for rows.Next() {
		var id string
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			return nil, err
		}
		embeddings[id] = bytesToFloat32s(blob)
	}

	return embeddings, nil
}

// ============================================================================
// Vector Operations
// ============================================================================

// CosineSimilarity computes the cosine similarity between two vectors.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / float32(math.Sqrt(float64(normA))*math.Sqrt(float64(normB)))
}

// SimilarityResult represents a record with its similarity score to a query.
type SimilarityResult struct {
	RecordID   string  `json:"recordId"`
	RecordKind string  `json:"recordKind"`
	Similarity float32 `json:"similarity"`
}

// FindSimilarEmbeddings finds records with embeddings similar to the query.
func (m *Memory) FindSimilarEmbeddings(queryEmbedding []float32, recordKind string, limit int, minSimilarity float32) ([]SimilarityResult, error) {
	query := `SELECT record_id, record_kind, embedding FROM embeddings`
	args := []interface{}{}

	if recordKind != "" {
		query += ` WHERE record_kind = ?`
		args = append(args, recordKind)
	}

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SimilarityResult
	for rows.Next() {
		var id, kind string
		var blob []byte
		if err := rows.Scan(&id, &kind, &blob); err != nil {
			continue
		}

		embedding := bytesToFloat32s(blob)
		similarity := CosineSimilarity(queryEmbedding, embedding)

		if similarity >= minSimilarity {
			results = append(results, SimilarityResult{
				RecordID:   id,
				RecordKind: kind,
				Similarity: similarity,
			})
		}
	}

	// Sort by similarity (descending)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Similarity > results[i].Similarity {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// CountEmbeddings returns the total number of embeddings.
func (m *Memory) CountEmbeddings() (int, error) {
	var count int
	err := m.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM embeddings`).Scan(&count)
	return count, err
}

// ============================================================================
// Helper Functions
// ============================================================================

// float32sToBytes converts a slice of float32 to bytes for storage.
func float32sToBytes(floats []float32) []byte {
	buf := new(bytes.Buffer)
	for _, f := range floats {
		binary.Write(buf, binary.LittleEndian, f)
	}
	return buf.Bytes()
}

// bytesToFloat32s converts bytes back to a slice of float32.
func bytesToFloat32s(data []byte) []float32 {
	if len(data) == 0 {
		return nil
	}
	// Ensure data length is a multiple of 4 (size of float32)
	if len(data)%4 != 0 {
		return nil
	}
	floats := make([]float32, len(data)/4)
	buf := bytes.NewReader(data)
	for i := range floats {
		if err := binary.Read(buf, binary.LittleEndian, &floats[i]); err != nil {
			// Return partial results on error (shouldn't happen with valid data)
			return floats[:i]
		}
	}
	return floats
}
