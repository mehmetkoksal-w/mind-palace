package dashboard

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/butler"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// handleRemember captures ideas/decisions with auto-classification.
// POST /api/remember
// Body: { "content": "...", "kind": "idea|decision|learning" (optional), "scope": "palace|room|file", "scopePath": "...", "tags": ["..."] }
func (s *Server) handleRemember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "memory not available")
		return
	}

	var req struct {
		Content   string   `json:"content"`
		Kind      string   `json:"kind"`
		Scope     string   `json:"scope"`
		ScopePath string   `json:"scopePath"`
		Tags      []string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	if req.Scope == "" {
		req.Scope = "palace"
	}

	// Determine kind via auto-classification or explicit
	var kind memory.RecordKind
	var classification memory.Classification

	if req.Kind != "" {
		kind = memory.RecordKind(req.Kind)
		classification = memory.Classification{Kind: kind, Confidence: 1.0, Signals: []string{"explicit"}}
	} else {
		classification = memory.Classify(req.Content)
		kind = classification.Kind
	}

	// Extract additional tags from content
	extractedTags := memory.ExtractTags(req.Content)
	req.Tags = append(req.Tags, extractedTags...)

	// Store based on kind
	var recordID string
	var err error
	needsAudit := false

	switch kind {
	case memory.RecordKindIdea:
		idea := memory.Idea{
			Content:   req.Content,
			Scope:     req.Scope,
			ScopePath: req.ScopePath,
			Source:    "dashboard",
		}
		recordID, err = mem.AddIdea(idea)
	case memory.RecordKindDecision:
		// Dashboard writes from humans are treated as approved (direct authority)
		dec := memory.Decision{
			Content:   req.Content,
			Scope:     req.Scope,
			ScopePath: req.ScopePath,
			Source:    "dashboard",
			Authority: string(memory.AuthorityApproved),
		}
		recordID, err = mem.AddDecision(dec)
		needsAudit = true
	case memory.RecordKindLearning:
		// Dashboard writes from humans are treated as approved (direct authority)
		learning := memory.Learning{
			Content:    req.Content,
			Scope:      req.Scope,
			ScopePath:  req.ScopePath,
			Source:     "dashboard",
			Confidence: 0.5,
			Authority:  string(memory.AuthorityApproved),
		}
		recordID, err = mem.AddLearning(learning)
		needsAudit = true
	default:
		// Default to idea if unknown
		idea := memory.Idea{
			Content:   req.Content,
			Scope:     req.Scope,
			ScopePath: req.ScopePath,
			Source:    "dashboard",
		}
		recordID, err = mem.AddIdea(idea)
		kind = memory.RecordKindIdea
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Audit log for direct writes (decisions/learnings)
	if needsAudit {
		hash := sha256.Sum256([]byte(req.Content))
		contentHash := hex.EncodeToString(hash[:])
		targetKind := string(kind)
		_, _ = mem.AddAuditLog(memory.AuditLogEntry{
			Action:     memory.AuditActionDirectWrite,
			ActorType:  memory.AuditActorHuman,
			ActorID:    "dashboard",
			TargetID:   recordID,
			TargetKind: targetKind,
			Details:    fmt.Sprintf(`{"scope":"%s","scope_path":"%s","content_hash":"%s"}`, req.Scope, req.ScopePath, contentHash), //nolint:gocritic // JSON template uses %s for values inside quotes
		})
	}

	// Set tags if any
	if len(req.Tags) > 0 {
		mem.SetTags(recordID, string(kind), req.Tags)
	}

	writeJSON(w, map[string]any{
		"id":         recordID,
		"kind":       string(kind),
		"confidence": classification.Confidence,
		"signals":    classification.Signals,
		"scope":      req.Scope,
		"scopePath":  req.ScopePath,
		"tags":       req.Tags,
	})
}

// handleBrainSearch searches ideas and decisions.
// GET /api/brain/search?q=...&kind=idea|decision&status=active&limit=20
func (s *Server) handleBrainSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "memory not available")
		return
	}

	query := r.URL.Query().Get("q")
	kind := r.URL.Query().Get("kind")
	status := r.URL.Query().Get("status")
	scope := r.URL.Query().Get("scope")
	scopePath := r.URL.Query().Get("scopePath")

	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			limit = n
		}
	}

	result := map[string]any{
		"ideas":     []any{},
		"decisions": []any{},
		"learnings": []any{},
	}

	// Search ideas
	if kind == "" || kind == "idea" {
		if query != "" {
			ideas, _ := mem.SearchIdeas(query, limit)
			if ideas != nil {
				result["ideas"] = ideas
			}
		} else {
			ideas, _ := mem.GetIdeas(status, scope, scopePath, limit)
			if ideas != nil {
				result["ideas"] = ideas
			}
		}
	}

	// Search decisions
	if kind == "" || kind == "decision" {
		if query != "" {
			decisions, _ := mem.SearchDecisions(query, limit)
			if decisions != nil {
				result["decisions"] = decisions
			}
		} else {
			decisions, _ := mem.GetDecisions(status, "", scope, scopePath, limit) // empty outcome filter
			if decisions != nil {
				result["decisions"] = decisions
			}
		}
	}

	// Search learnings
	if kind == "" || kind == "learning" {
		if query != "" {
			learnings, _ := mem.SearchLearnings(query, limit)
			if learnings != nil {
				result["learnings"] = learnings
			}
		} else {
			learnings, _ := mem.GetLearnings(scope, scopePath, limit)
			if learnings != nil {
				result["learnings"] = learnings
			}
		}
	}

	writeJSON(w, result)
}

// handleBrainContext returns assembled context for a topic.
// GET /api/brain/context?query=...&includeIdeas=true&includeDecisions=true&includeLearnings=true&includeCode=true
func (s *Server) handleBrainContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	s.mu.RLock()
	b := s.butler
	s.mu.RUnlock()

	if b == nil {
		writeError(w, http.StatusServiceUnavailable, "butler not available")
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter required")
		return
	}

	// Parse boolean flags
	includeIdeas := r.URL.Query().Get("includeIdeas") != "false"
	includeDecisions := r.URL.Query().Get("includeDecisions") != "false"
	includeLearnings := r.URL.Query().Get("includeLearnings") != "false"

	opts := butler.EnhancedContextOptions{
		Query:            query,
		IncludeIdeas:     includeIdeas,
		IncludeDecisions: includeDecisions,
		IncludeLearnings: includeLearnings,
		Limit:            10,
	}

	ctx, err := b.GetEnhancedContext(opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := map[string]any{}

	// Code context from embedded ContextResult
	if ctx.ContextResult != nil {
		if len(ctx.Symbols) > 0 {
			result["symbols"] = ctx.Symbols
		}
		if len(ctx.Decisions) > 0 {
			result["codeDecisions"] = ctx.Decisions
		}
	}

	// Brain data
	if len(ctx.BrainIdeas) > 0 {
		result["ideas"] = ctx.BrainIdeas
	}
	if len(ctx.BrainDecisions) > 0 {
		result["decisions"] = ctx.BrainDecisions
	}
	if len(ctx.Learnings) > 0 {
		result["learnings"] = ctx.Learnings
	}
	if len(ctx.RelatedLinks) > 0 {
		result["links"] = ctx.RelatedLinks
	}
	if len(ctx.DecisionConflicts) > 0 {
		result["conflicts"] = ctx.DecisionConflicts
	}

	writeJSON(w, result)
}

// handleContradictions returns contradictions summary.
// GET /api/contradictions?limit=20
func (s *Server) handleContradictions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "memory not available")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	summary, err := mem.GetContradictionSummary(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, summary)
}
