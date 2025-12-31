package dashboard

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// handleSessions returns session list.
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
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

	activeOnly := r.URL.Query().Get("active") == "true"
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			limit = n
		}
	}

	sessions, err := mem.ListSessions(activeOnly, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sessions == nil {
		sessions = []memory.Session{}
	}

	writeJSON(w, map[string]any{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// handleSessionDetail returns details for a specific session.
func (s *Server) handleSessionDetail(w http.ResponseWriter, r *http.Request) {
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

	// Extract session ID from path: /api/sessions/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	sessionID := strings.TrimSuffix(path, "/")

	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "session ID required")
		return
	}

	session, err := mem.GetSession(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if session == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	activities, _ := mem.GetActivities(sessionID, "", 100)
	if activities == nil {
		activities = []memory.Activity{}
	}

	writeJSON(w, map[string]any{
		"session":    session,
		"activities": activities,
	})
}
