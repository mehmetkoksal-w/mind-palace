package memory

import (
	"context"
	"fmt"
	"time"
)

// SemanticSearchResult represents a result from semantic search.
type SemanticSearchResult struct {
	ID         string    `json:"id"`
	Kind       string    `json:"kind"` // "idea", "decision", "learning"
	Content    string    `json:"content"`
	Similarity float32   `json:"similarity"`
	CreatedAt  time.Time `json:"createdAt"`
}

// SemanticSearchOptions configures semantic search behavior.
type SemanticSearchOptions struct {
	Kinds         []string // Filter by record kinds (empty = all)
	Limit         int      // Maximum results (default 10)
	MinSimilarity float32  // Minimum similarity threshold (default 0.5)
	Scope         string   // Filter by scope
	ScopePath     string   // Filter by scope path
}

// DefaultSemanticSearchOptions returns default search options.
func DefaultSemanticSearchOptions() SemanticSearchOptions {
	return SemanticSearchOptions{
		Kinds:         []string{}, // All kinds
		Limit:         10,
		MinSimilarity: 0.5,
	}
}

// SemanticSearch performs semantic search using embeddings.
// Unlike FTS5, this finds conceptually similar content even without word overlap.
func (m *Memory) SemanticSearch(embedder Embedder, query string, opts SemanticSearchOptions) ([]SemanticSearchResult, error) {
	if embedder == nil {
		return nil, fmt.Errorf("embedder not available")
	}

	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	if opts.MinSimilarity <= 0 {
		opts.MinSimilarity = 0.5
	}

	// Generate query embedding
	queryEmb, err := embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	var results []SemanticSearchResult

	// Search each enabled kind
	kinds := opts.Kinds
	if len(kinds) == 0 {
		kinds = []string{"idea", "decision", "learning"}
	}

	for _, kind := range kinds {
		kindResults, err := m.searchKind(queryEmb, kind, opts)
		if err != nil {
			continue // Skip on error
		}
		results = append(results, kindResults...)
	}

	// Sort by similarity
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Similarity > results[i].Similarity {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Apply limit
	if len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}

// searchKind searches a specific record kind.
func (m *Memory) searchKind(queryEmb []float32, kind string, opts SemanticSearchOptions) ([]SemanticSearchResult, error) {
	// Get all embeddings for this kind
	similar, err := m.FindSimilarEmbeddings(queryEmb, kind, opts.Limit*2, opts.MinSimilarity)
	if err != nil {
		return nil, err
	}

	var results []SemanticSearchResult

	for _, s := range similar {
		// Fetch the actual record content
		content, createdAt, err := m.GetRecordContent(s.RecordID, kind)
		if err != nil {
			continue
		}

		// Apply scope filter if specified
		if opts.Scope != "" || opts.ScopePath != "" {
			scope, scopePath := m.getRecordScope(s.RecordID, kind)
			if opts.Scope != "" && scope != opts.Scope {
				continue
			}
			if opts.ScopePath != "" && scopePath != opts.ScopePath {
				continue
			}
		}

		results = append(results, SemanticSearchResult{
			ID:         s.RecordID,
			Kind:       kind,
			Content:    content,
			Similarity: s.Similarity,
			CreatedAt:  createdAt,
		})
	}

	return results, nil
}

// GetRecordContent fetches content for a record.
func (m *Memory) GetRecordContent(id, kind string) (string, time.Time, error) {
	var content, createdAtStr string

	switch kind {
	case "idea":
		err := m.db.QueryRowContext(context.Background(), `SELECT content, created_at FROM ideas WHERE id = ?`, id).Scan(&content, &createdAtStr)
		if err != nil {
			return "", time.Time{}, err
		}
	case "decision":
		err := m.db.QueryRowContext(context.Background(), `SELECT content, created_at FROM decisions WHERE id = ?`, id).Scan(&content, &createdAtStr)
		if err != nil {
			return "", time.Time{}, err
		}
	case "learning":
		err := m.db.QueryRowContext(context.Background(), `SELECT content, created_at FROM learnings WHERE id = ?`, id).Scan(&content, &createdAtStr)
		if err != nil {
			return "", time.Time{}, err
		}
	default:
		return "", time.Time{}, fmt.Errorf("unknown kind: %s", kind)
	}

	createdAt := parseTimeOrZero(createdAtStr)
	return content, createdAt, nil
}

// getRecordScope fetches scope info for a record.
func (m *Memory) getRecordScope(id, kind string) (string, string) {
	var scope, scopePath string

	switch kind {
	case "idea":
		m.db.QueryRowContext(context.Background(), `SELECT scope, scope_path FROM ideas WHERE id = ?`, id).Scan(&scope, &scopePath)
	case "decision":
		m.db.QueryRowContext(context.Background(), `SELECT scope, scope_path FROM decisions WHERE id = ?`, id).Scan(&scope, &scopePath)
	case "learning":
		m.db.QueryRowContext(context.Background(), `SELECT scope, scope_path FROM learnings WHERE id = ?`, id).Scan(&scope, &scopePath)
	}

	return scope, scopePath
}

// HybridSearchResult contains results from combined keyword and semantic search.
type HybridSearchResult struct {
	SemanticSearchResult
	MatchType string  `json:"matchType"` // "keyword", "semantic", "both"
	FTSScore  float64 `json:"ftsScore,omitempty"`
}

// HybridSearch performs combined keyword + semantic search.
func (m *Memory) HybridSearch(embedder Embedder, query string, opts SemanticSearchOptions) ([]HybridSearchResult, error) {
	results := make(map[string]*HybridSearchResult)

	// Default to all kinds if none specified
	kinds := opts.Kinds
	if len(kinds) == 0 {
		kinds = []string{"idea", "decision", "learning"}
	}

	// First, do FTS5 keyword search
	for _, kind := range kinds {
		ftsResults := m.ftsSearch(query, kind, opts.Limit)
		for _, r := range ftsResults {
			if existing, ok := results[r.ID]; ok {
				existing.MatchType = "both"
				existing.FTSScore = r.FTSScore
			} else {
				results[r.ID] = &HybridSearchResult{
					SemanticSearchResult: SemanticSearchResult{
						ID:        r.ID,
						Kind:      kind,
						Content:   r.Content,
						CreatedAt: r.CreatedAt,
					},
					MatchType: "keyword",
					FTSScore:  r.FTSScore,
				}
			}
		}
	}

	// Then do semantic search if embedder available
	if embedder != nil {
		semanticResults, err := m.SemanticSearch(embedder, query, opts)
		if err == nil {
			for _, r := range semanticResults {
				if existing, ok := results[r.ID]; ok {
					existing.MatchType = "both"
					existing.Similarity = r.Similarity
				} else {
					results[r.ID] = &HybridSearchResult{
						SemanticSearchResult: r,
						MatchType:            "semantic",
					}
				}
			}
		}
	}

	// Convert to slice and sort
	var finalResults []HybridSearchResult
	for _, r := range results {
		finalResults = append(finalResults, *r)
	}

	// Sort: "both" first, then by similarity/score
	for i := 0; i < len(finalResults)-1; i++ {
		for j := i + 1; j < len(finalResults); j++ {
			// "both" matches should come first
			if finalResults[j].MatchType == "both" && finalResults[i].MatchType != "both" {
				finalResults[i], finalResults[j] = finalResults[j], finalResults[i]
			} else if finalResults[i].MatchType == finalResults[j].MatchType {
				// Within same match type, sort by score
				scoreI := float64(finalResults[i].Similarity) + finalResults[i].FTSScore
				scoreJ := float64(finalResults[j].Similarity) + finalResults[j].FTSScore
				if scoreJ > scoreI {
					finalResults[i], finalResults[j] = finalResults[j], finalResults[i]
				}
			}
		}
	}

	if len(finalResults) > opts.Limit {
		finalResults = finalResults[:opts.Limit]
	}

	return finalResults, nil
}

// ftsResult is a helper for FTS5 results.
type ftsResult struct {
	ID        string
	Content   string
	CreatedAt time.Time
	FTSScore  float64
}

// ftsSearch performs FTS5 search for a specific kind.
func (m *Memory) ftsSearch(query, kind string, limit int) []ftsResult {
	var results []ftsResult

	switch kind {
	case "idea":
		rows, err := m.db.QueryContext(context.Background(), `
			SELECT i.id, i.content, i.created_at, rank
			FROM ideas i
			JOIN ideas_fts fts ON i.rowid = fts.rowid
			WHERE ideas_fts MATCH ?
			ORDER BY rank
			LIMIT ?`, query, limit)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var r ftsResult
				var createdAt string
				rows.Scan(&r.ID, &r.Content, &createdAt, &r.FTSScore)
				r.CreatedAt = parseTimeOrZero(createdAt)
				results = append(results, r)
			}
			if err := rows.Err(); err != nil {
				_ = rows.Err()
			}
		}
	case "decision":
		rows, err := m.db.QueryContext(context.Background(), `
			SELECT d.id, d.content, d.created_at, rank
			FROM decisions d
			JOIN decisions_fts fts ON d.rowid = fts.rowid
			WHERE decisions_fts MATCH ?
			ORDER BY rank
			LIMIT ?`, query, limit)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var r ftsResult
				var createdAt string
				rows.Scan(&r.ID, &r.Content, &createdAt, &r.FTSScore)
				r.CreatedAt = parseTimeOrZero(createdAt)
				results = append(results, r)
			}
		}
	case "learning":
		rows, err := m.db.QueryContext(context.Background(), `
			SELECT l.id, l.content, l.created_at, rank
			FROM learnings l
			JOIN learnings_fts fts ON l.rowid = fts.rowid
			WHERE learnings_fts MATCH ?
			ORDER BY rank
			LIMIT ?`, query, limit)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var r ftsResult
				var createdAt string
				rows.Scan(&r.ID, &r.Content, &createdAt, &r.FTSScore)
				r.CreatedAt = parseTimeOrZero(createdAt)
				results = append(results, r)
			}
		}
	}

	return results
}
