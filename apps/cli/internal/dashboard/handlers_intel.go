package dashboard

import (
	"net/http"
	"strconv"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// handleFileIntel returns file intelligence.
func (s *Server) handleFileIntel(w http.ResponseWriter, r *http.Request) {
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

	path := r.URL.Query().Get("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path parameter required")
		return
	}

	intel, err := mem.GetFileIntel(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	learnings, _ := mem.GetFileLearnings(path)
	if learnings == nil {
		learnings = []memory.Learning{}
	}

	writeJSON(w, map[string]any{
		"intel":     intel,
		"learnings": learnings,
	})
}

// handleHotspots returns file hotspots.
func (s *Server) handleHotspots(w http.ResponseWriter, r *http.Request) {
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
		if n, err := strconv.Atoi(limitStr); err == nil {
			limit = n
		}
	}

	hotspots, err := mem.GetFileHotspots(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if hotspots == nil {
		hotspots = []memory.FileIntel{}
	}

	fragile, _ := mem.GetFragileFiles(limit)
	if fragile == nil {
		fragile = []memory.FileIntel{}
	}

	writeJSON(w, map[string]any{
		"hotspots": hotspots,
		"fragile":  fragile,
	})
}

// handleAgents returns active agents.
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
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

	agents, err := mem.GetActiveAgents(5 * time.Minute)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if agents == nil {
		agents = []memory.ActiveAgent{}
	}

	writeJSON(w, map[string]any{
		"agents": agents,
		"count":  len(agents),
	})
}
