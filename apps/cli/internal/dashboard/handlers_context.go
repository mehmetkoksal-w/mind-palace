package dashboard

import (
	"encoding/json"
	"net/http"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/butler"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
)

// handleContextPreview returns auto-injection context preview for a file.
// POST /api/context/preview
func (s *Server) handleContextPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var req struct {
		FilePath         string  `json:"filePath"`
		MaxTokens        int     `json:"maxTokens,omitempty"`
		IncludeLearnings *bool   `json:"includeLearnings,omitempty"`
		IncludeDecisions *bool   `json:"includeDecisions,omitempty"`
		IncludeFailures  *bool   `json:"includeFailures,omitempty"`
		MinConfidence    float64 `json:"minConfidence,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.FilePath == "" {
		writeError(w, http.StatusBadRequest, "filePath is required")
		return
	}

	s.mu.RLock()
	b := s.butler
	s.mu.RUnlock()

	if b == nil {
		writeError(w, http.StatusServiceUnavailable, "butler not available")
		return
	}

	// Build config from request
	cfg := config.DefaultAutoInjectionConfig()
	if req.MaxTokens > 0 {
		cfg.MaxTokens = req.MaxTokens
	}
	if req.IncludeLearnings != nil {
		cfg.IncludeLearnings = *req.IncludeLearnings
	}
	if req.IncludeDecisions != nil {
		cfg.IncludeDecisions = *req.IncludeDecisions
	}
	if req.IncludeFailures != nil {
		cfg.IncludeFailures = *req.IncludeFailures
	}
	if req.MinConfidence > 0 {
		cfg.MinConfidence = req.MinConfidence
	}

	ctx, err := b.GetAutoInjectionContext(req.FilePath, cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, ctx)
}

// handleScopeExplain returns scope explanation for a file.
// POST /api/scope/explain
func (s *Server) handleScopeExplain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var req struct {
		FilePath string `json:"filePath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.FilePath == "" {
		writeError(w, http.StatusBadRequest, "filePath is required")
		return
	}

	s.mu.RLock()
	b := s.butler
	s.mu.RUnlock()

	if b == nil {
		writeError(w, http.StatusServiceUnavailable, "butler not available")
		return
	}

	// Get scope config from butler's config
	var scopeCfg *config.ScopeConfig
	if b.Config() != nil && b.Config().Scope != nil {
		scopeCfg = b.Config().Scope
	}

	explanation, err := b.GetScopeExplanation(req.FilePath, scopeCfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, explanation)
}

// handleScopeHierarchy returns full scope hierarchy data.
// GET /api/scope/hierarchy
func (s *Server) handleScopeHierarchy(w http.ResponseWriter, r *http.Request) {
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

	// Build hierarchy view
	hierarchy := butler.ScopeHierarchyView{
		Levels: []butler.ScopeLevelDetail{},
	}

	// File-level records (we can't enumerate all files, skip for now)

	// Room-level records - get unique rooms
	roomLearnings, _ := mem.GetLearnings("room", "", 100)
	roomDecisions, _ := mem.GetDecisions("", "", "room", "", 100)
	roomIdeas, _ := mem.GetIdeas("", "room", "", 100)

	if len(roomLearnings) > 0 || len(roomDecisions) > 0 || len(roomIdeas) > 0 {
		hierarchy.Levels = append(hierarchy.Levels, butler.ScopeLevelDetail{
			Scope:     "room",
			Learnings: roomLearnings,
			Decisions: roomDecisions,
			Ideas:     roomIdeas,
		})
	}

	// Palace-level records
	palaceLearnings, _ := mem.GetLearnings("palace", "", 100)
	palaceDecisions, _ := mem.GetDecisions("", "", "palace", "", 100)
	palaceIdeas, _ := mem.GetIdeas("", "palace", "", 100)

	hierarchy.Levels = append(hierarchy.Levels, butler.ScopeLevelDetail{
		Scope:     "palace",
		Learnings: palaceLearnings,
		Decisions: palaceDecisions,
		Ideas:     palaceIdeas,
	})

	writeJSON(w, hierarchy)
}
