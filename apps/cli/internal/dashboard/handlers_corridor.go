package dashboard

import (
	"net/http"
	"strconv"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/corridor"
)

// handleCorridors returns corridor overview.
func (s *Server) handleCorridors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	s.mu.RLock()
	corr := s.corridor
	s.mu.RUnlock()

	if corr == nil {
		writeError(w, http.StatusServiceUnavailable, "corridor not available")
		return
	}

	stats, err := corr.Stats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	links, _ := corr.GetLinks()
	if links == nil {
		links = []corridor.LinkedWorkspace{}
	}

	writeJSON(w, map[string]any{
		"stats": stats,
		"links": links,
	})
}

// handleCorridorPersonal returns personal corridor learnings.
func (s *Server) handleCorridorPersonal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	s.mu.RLock()
	corr := s.corridor
	s.mu.RUnlock()

	if corr == nil {
		writeError(w, http.StatusServiceUnavailable, "corridor not available")
		return
	}

	query := r.URL.Query().Get("query")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			limit = n
		}
	}

	learnings, err := corr.GetPersonalLearnings(query, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if learnings == nil {
		learnings = []corridor.PersonalLearning{}
	}

	writeJSON(w, map[string]any{
		"learnings": learnings,
		"count":     len(learnings),
	})
}

// handleCorridorLinks returns linked workspaces.
func (s *Server) handleCorridorLinks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	s.mu.RLock()
	corr := s.corridor
	s.mu.RUnlock()

	if corr == nil {
		writeError(w, http.StatusServiceUnavailable, "corridor not available")
		return
	}

	links, err := corr.GetLinks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if links == nil {
		links = []corridor.LinkedWorkspace{}
	}

	writeJSON(w, map[string]any{
		"links": links,
		"count": len(links),
	})
}
