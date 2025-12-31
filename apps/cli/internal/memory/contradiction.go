package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/llm"
)

// ContradictionAnalyzer defines the interface for detecting contradictions.
// This is designed to be implemented by LLM-based analyzers.
type ContradictionAnalyzer interface {
	// AnalyzeContradiction checks if two records contradict each other.
	// Returns a ContradictionResult with analysis details.
	AnalyzeContradiction(record1, record2 RecordForAnalysis) (*ContradictionResult, error)

	// FindContradictions analyzes a record against a set of candidates.
	// Returns potential contradictions found.
	FindContradictions(record RecordForAnalysis, candidates []RecordForAnalysis) ([]ContradictionResult, error)
}

// RecordForAnalysis represents a record prepared for contradiction analysis.
type RecordForAnalysis struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`    // "idea", "decision", "learning"
	Content   string    `json:"content"` // Main content
	Context   string    `json:"context"` // Additional context
	Status    string    `json:"status"`  // For decisions: "active", "superseded"
	CreatedAt time.Time `json:"createdAt"`
}

// ContradictionResult represents the result of contradiction analysis.
type ContradictionResult struct {
	Record1ID       string  `json:"record1Id"`
	Record2ID       string  `json:"record2Id"`
	IsContradiction bool    `json:"isContradiction"`
	Confidence      float64 `json:"confidence"`  // 0.0 to 1.0
	Explanation     string  `json:"explanation"` // Why they contradict (or don't)
	ContradictType  string  `json:"contradictType,omitempty"` // "direct", "implicit", "temporal"
}

// ============================================================================
// Stub Analyzer (for testing and as default)
// ============================================================================

// StubContradictionAnalyzer is a simple analyzer that uses basic heuristics.
// Real contradiction detection should use an LLM for nuanced analysis.
type StubContradictionAnalyzer struct{}

// NewStubContradictionAnalyzer creates a new stub analyzer.
func NewStubContradictionAnalyzer() ContradictionAnalyzer {
	return &StubContradictionAnalyzer{}
}

// AnalyzeContradiction performs basic contradiction analysis.
// This stub uses simple heuristics - a real implementation would use an LLM.
func (a *StubContradictionAnalyzer) AnalyzeContradiction(r1, r2 RecordForAnalysis) (*ContradictionResult, error) {
	result := &ContradictionResult{
		Record1ID:       r1.ID,
		Record2ID:       r2.ID,
		IsContradiction: false,
		Confidence:      0.5,
		Explanation:     "Stub analyzer cannot determine contradiction - requires LLM",
	}
	return result, nil
}

// FindContradictions finds potential contradictions using basic heuristics.
func (a *StubContradictionAnalyzer) FindContradictions(record RecordForAnalysis, candidates []RecordForAnalysis) ([]ContradictionResult, error) {
	var results []ContradictionResult

	for _, candidate := range candidates {
		if candidate.ID == record.ID {
			continue // Skip self
		}

		result, err := a.AnalyzeContradiction(record, candidate)
		if err != nil {
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// ============================================================================
// LLM-Based Contradiction Analyzer
// ============================================================================

// LLMContradictionAnalyzer uses an LLM to detect semantic contradictions.
type LLMContradictionAnalyzer struct {
	llm llm.Client
}

// NewLLMContradictionAnalyzer creates a new LLM-based analyzer.
func NewLLMContradictionAnalyzer(client llm.Client) ContradictionAnalyzer {
	return &LLMContradictionAnalyzer{llm: client}
}

// contradictionPrompt is the prompt for analyzing contradictions.
const contradictionPrompt = `Compare these two statements for contradictions. Consider:
- Direct contradictions (opposite assertions)
- Implicit contradictions (incompatible approaches/strategies)
- Temporal contradictions (outdated vs current)

Statement 1 (%s, created %s):
"%s"

Statement 2 (%s, created %s):
"%s"

Return a JSON object with this exact structure:
{
  "isContradiction": true or false,
  "confidence": 0.0 to 1.0,
  "type": "direct" | "implicit" | "temporal" | "none",
  "explanation": "Brief explanation of why they do or don't contradict"
}

Be conservative - only mark as contradiction if there's genuine conflict.`

// LLMContradictionResponse is the expected JSON response from the LLM.
type LLMContradictionResponse struct {
	IsContradiction bool    `json:"isContradiction"`
	Confidence      float64 `json:"confidence"`
	Type            string  `json:"type"`
	Explanation     string  `json:"explanation"`
}

// AnalyzeContradiction uses the LLM to analyze if two records contradict.
func (a *LLMContradictionAnalyzer) AnalyzeContradiction(r1, r2 RecordForAnalysis) (*ContradictionResult, error) {
	prompt := fmt.Sprintf(contradictionPrompt,
		r1.Kind, r1.CreatedAt.Format("2006-01-02"), r1.Content,
		r2.Kind, r2.CreatedAt.Format("2006-01-02"), r2.Content,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	opts := llm.DefaultCompletionOptions()
	opts.SystemPrompt = "You are a contradiction detection assistant. Analyze statements for logical conflicts. Always respond with valid JSON."

	var response LLMContradictionResponse
	if err := a.llm.CompleteJSON(ctx, prompt, opts, &response); err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	return &ContradictionResult{
		Record1ID:       r1.ID,
		Record2ID:       r2.ID,
		IsContradiction: response.IsContradiction,
		Confidence:      response.Confidence,
		Explanation:     response.Explanation,
		ContradictType:  response.Type,
	}, nil
}

// FindContradictions analyzes a record against multiple candidates.
func (a *LLMContradictionAnalyzer) FindContradictions(record RecordForAnalysis, candidates []RecordForAnalysis) ([]ContradictionResult, error) {
	var results []ContradictionResult

	for _, candidate := range candidates {
		if candidate.ID == record.ID {
			continue // Skip self
		}

		result, err := a.AnalyzeContradiction(record, candidate)
		if err != nil {
			continue // Skip failed analyses
		}

		// Only include actual contradictions
		if result.IsContradiction {
			results = append(results, *result)
		}
	}

	return results, nil
}

// ============================================================================
// Auto-Contradiction Detection
// ============================================================================

// AutoCheckContradictions finds contradictions for a new record and optionally auto-links.
func (m *Memory) AutoCheckContradictions(recordID, kind, content string, analyzer ContradictionAnalyzer, embedder Embedder, autoLink bool, minConfidence float64) ([]ContradictionResult, error) {
	// Create record for analysis
	record := RecordForAnalysis{
		ID:        recordID,
		Kind:      kind,
		Content:   content,
		CreatedAt: time.Now(),
	}

	// Find potential candidates
	opts := DefaultContradictionOptions()
	candidates, err := m.FindPotentialContradictions(content, embedder, opts)
	if err != nil {
		return nil, fmt.Errorf("find candidates: %w", err)
	}

	// Filter out self
	var filteredCandidates []RecordForAnalysis
	for _, c := range candidates {
		if c.ID != recordID {
			filteredCandidates = append(filteredCandidates, c)
		}
	}

	if len(filteredCandidates) == 0 {
		return []ContradictionResult{}, nil
	}

	// Analyze with LLM
	contradictions, err := analyzer.FindContradictions(record, filteredCandidates)
	if err != nil {
		return nil, fmt.Errorf("analyze: %w", err)
	}

	// Auto-link high-confidence contradictions
	if autoLink && minConfidence > 0 {
		for _, c := range contradictions {
			if c.Confidence >= minConfidence {
				// Create a link marking the contradiction
				link := Link{
					SourceID:   recordID,
					SourceKind: kind,
					TargetID:   c.Record2ID,
					TargetKind: m.getKindForID(c.Record2ID),
					Relation:   RelationContradicts,
				}
				_, _ = m.AddLink(link)
			}
		}
	}

	return contradictions, nil
}

// GetRecordForAnalysis retrieves a record for contradiction analysis (exported wrapper).
func (m *Memory) GetRecordForAnalysis(id, kind string) (*RecordForAnalysis, error) {
	return m.getRecordForAnalysis(id, kind)
}

// getKindForID determines the record kind from the ID prefix.
func (m *Memory) getKindForID(id string) string {
	if len(id) < 2 {
		return ""
	}
	switch {
	case id[0] == 'i' && id[1] == '_':
		return "idea"
	case id[0] == 'd' && id[1] == '_':
		return "decision"
	case id[0] == 'l' && id[1] == '_':
		return "learning"
	default:
		return ""
	}
}

// ============================================================================
// Contradiction Detection Methods
// ============================================================================

// ContradictionCheckOptions configures contradiction checking.
type ContradictionCheckOptions struct {
	UseEmbeddings   bool    // Use semantic similarity to find candidates
	MinSimilarity   float32 // Minimum similarity to consider as candidate
	MaxCandidates   int     // Maximum candidates to analyze
	IncludeIdeas    bool    // Check ideas
	IncludeDecisions bool   // Check decisions
	IncludeLearnings bool   // Check learnings
}

// DefaultContradictionOptions returns default options.
func DefaultContradictionOptions() ContradictionCheckOptions {
	return ContradictionCheckOptions{
		UseEmbeddings:    true,
		MinSimilarity:    0.6,
		MaxCandidates:    20,
		IncludeIdeas:     true,
		IncludeDecisions: true,
		IncludeLearnings: false, // Learnings rarely contradict
	}
}

// FindPotentialContradictions finds records that may contradict the given content.
// Uses embeddings to find semantically similar content, then returns candidates.
func (m *Memory) FindPotentialContradictions(content string, embedder Embedder, opts ContradictionCheckOptions) ([]RecordForAnalysis, error) {
	var candidates []RecordForAnalysis

	if embedder == nil || !opts.UseEmbeddings {
		// Fall back to FTS search
		return m.findContradictionsByFTS(content, opts)
	}

	// Get embedding for content
	queryEmb, err := embedder.Embed(content)
	if err != nil {
		return m.findContradictionsByFTS(content, opts)
	}

	// Find similar records
	var kinds []string
	if opts.IncludeIdeas {
		kinds = append(kinds, "idea")
	}
	if opts.IncludeDecisions {
		kinds = append(kinds, "decision")
	}
	if opts.IncludeLearnings {
		kinds = append(kinds, "learning")
	}

	for _, kind := range kinds {
		similar, err := m.FindSimilarEmbeddings(queryEmb, kind, opts.MaxCandidates, opts.MinSimilarity)
		if err != nil {
			continue
		}

		for _, s := range similar {
			record, err := m.getRecordForAnalysis(s.RecordID, kind)
			if err != nil {
				continue
			}
			candidates = append(candidates, *record)
		}
	}

	// Limit total candidates
	if len(candidates) > opts.MaxCandidates {
		candidates = candidates[:opts.MaxCandidates]
	}

	return candidates, nil
}

// findContradictionsByFTS uses FTS to find potential contradictions.
func (m *Memory) findContradictionsByFTS(content string, opts ContradictionCheckOptions) ([]RecordForAnalysis, error) {
	var candidates []RecordForAnalysis

	if opts.IncludeIdeas {
		ideas, _ := m.SearchIdeas(content, opts.MaxCandidates)
		for _, idea := range ideas {
			candidates = append(candidates, RecordForAnalysis{
				ID:        idea.ID,
				Kind:      "idea",
				Content:   idea.Content,
				Context:   idea.Context,
				Status:    idea.Status,
				CreatedAt: idea.CreatedAt,
			})
		}
	}

	if opts.IncludeDecisions {
		decisions, _ := m.SearchDecisions(content, opts.MaxCandidates)
		for _, dec := range decisions {
			candidates = append(candidates, RecordForAnalysis{
				ID:        dec.ID,
				Kind:      "decision",
				Content:   dec.Content,
				Context:   dec.Context,
				Status:    dec.Status,
				CreatedAt: dec.CreatedAt,
			})
		}
	}

	// Limit total
	if len(candidates) > opts.MaxCandidates {
		candidates = candidates[:opts.MaxCandidates]
	}

	return candidates, nil
}

// getRecordForAnalysis converts a record ID to RecordForAnalysis.
func (m *Memory) getRecordForAnalysis(id, kind string) (*RecordForAnalysis, error) {
	switch kind {
	case "idea":
		idea, err := m.GetIdea(id)
		if err != nil || idea == nil {
			return nil, err
		}
		return &RecordForAnalysis{
			ID:        idea.ID,
			Kind:      "idea",
			Content:   idea.Content,
			Context:   idea.Context,
			Status:    idea.Status,
			CreatedAt: idea.CreatedAt,
		}, nil

	case "decision":
		dec, err := m.GetDecision(id)
		if err != nil || dec == nil {
			return nil, err
		}
		return &RecordForAnalysis{
			ID:        dec.ID,
			Kind:      "decision",
			Content:   dec.Content,
			Context:   dec.Context,
			Status:    dec.Status,
			CreatedAt: dec.CreatedAt,
		}, nil

	case "learning":
		var content, scopePath, createdAt string
		err := m.db.QueryRow(`SELECT content, scope_path, created_at FROM learnings WHERE id = ?`, id).
			Scan(&content, &scopePath, &createdAt)
		if err != nil {
			return nil, err
		}
		return &RecordForAnalysis{
			ID:        id,
			Kind:      "learning",
			Content:   content,
			Context:   scopePath,
			CreatedAt: parseTimeOrZero(createdAt),
		}, nil
	}

	return nil, nil
}

// ============================================================================
// Link-based Contradiction Detection (existing functionality enhancement)
// ============================================================================

// GetContradictingRecords returns records that are explicitly marked as contradicting.
func (m *Memory) GetContradictingRecords(recordID string) ([]RecordForAnalysis, error) {
	links, err := m.GetLinksForSource(recordID)
	if err != nil {
		return nil, err
	}

	var contradicting []RecordForAnalysis
	for _, link := range links {
		if link.Relation != RelationContradicts {
			continue
		}

		record, err := m.getRecordForAnalysis(link.TargetID, link.TargetKind)
		if err != nil || record == nil {
			continue
		}
		contradicting = append(contradicting, *record)
	}

	// Also check incoming links (records that contradict this one)
	incomingLinks, err := m.GetLinksForTarget(recordID)
	if err != nil {
		return contradicting, nil
	}

	for _, link := range incomingLinks {
		if link.Relation != RelationContradicts {
			continue
		}

		record, err := m.getRecordForAnalysis(link.SourceID, link.SourceKind)
		if err != nil || record == nil {
			continue
		}
		contradicting = append(contradicting, *record)
	}

	return contradicting, nil
}

// ContradictionSummary provides an overview of contradictions in the system.
type ContradictionSummary struct {
	TotalContradictionLinks int                  `json:"totalContradictionLinks"`
	ActiveConflicts         int                  `json:"activeConflicts"`
	ResolvedConflicts       int                  `json:"resolvedConflicts"`
	TopContradictions       []ContradictionPair  `json:"topContradictions"`
}

// ContradictionPair represents a pair of contradicting records.
type ContradictionPair struct {
	Record1 RecordForAnalysis `json:"record1"`
	Record2 RecordForAnalysis `json:"record2"`
	LinkID  string            `json:"linkId"`
}

// GetContradictionSummary returns an overview of contradictions.
func (m *Memory) GetContradictionSummary(limit int) (*ContradictionSummary, error) {
	if limit <= 0 {
		limit = 10
	}

	summary := &ContradictionSummary{}

	// Count total contradiction links
	var totalLinks int
	m.db.QueryRow(`SELECT COUNT(*) FROM links WHERE relation = ?`, RelationContradicts).Scan(&totalLinks)
	summary.TotalContradictionLinks = totalLinks

	// Get contradiction pairs
	links, err := m.GetLinksByRelation(RelationContradicts, limit)
	if err != nil {
		return summary, err
	}

	for _, link := range links {
		source, _ := m.getRecordForAnalysis(link.SourceID, link.SourceKind)
		target, _ := m.getRecordForAnalysis(link.TargetID, link.TargetKind)

		if source != nil && target != nil {
			summary.TopContradictions = append(summary.TopContradictions, ContradictionPair{
				Record1: *source,
				Record2: *target,
				LinkID:  link.ID,
			})

			// Count active vs resolved based on status
			if source.Status == "active" && target.Status == "active" {
				summary.ActiveConflicts++
			} else {
				summary.ResolvedConflicts++
			}
		}
	}

	return summary, nil
}
