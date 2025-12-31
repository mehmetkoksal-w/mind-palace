package dashboard

import (
	"net/http"
	"strconv"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// handleActivity returns recent activities.
func (s *Server) handleActivity(w http.ResponseWriter, r *http.Request) {
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

	sessionID := r.URL.Query().Get("sessionId")
	path := r.URL.Query().Get("path")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			limit = n
		}
	}

	activities, err := mem.GetActivities(sessionID, path, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if activities == nil {
		activities = []memory.Activity{}
	}

	writeJSON(w, map[string]any{
		"activities": activities,
		"count":      len(activities),
	})
}

// handleLearnings returns learnings.
func (s *Server) handleLearnings(w http.ResponseWriter, r *http.Request) {
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

	scope := r.URL.Query().Get("scope")
	scopePath := r.URL.Query().Get("scopePath")
	query := r.URL.Query().Get("query")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			limit = n
		}
	}

	var learnings []memory.Learning
	var err error

	if query != "" {
		learnings, err = mem.SearchLearnings(query, limit)
	} else {
		learnings, err = mem.GetLearnings(scope, scopePath, limit)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if learnings == nil {
		learnings = []memory.Learning{}
	}

	writeJSON(w, map[string]any{
		"learnings": learnings,
		"count":     len(learnings),
	})
}
