package dashboard

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// handlePostmortems handles GET/POST /api/postmortems
func (s *Server) handlePostmortems(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "memory not available")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.listPostmortems(w, r, mem)
	case http.MethodPost:
		s.createPostmortem(w, r, mem)
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or POST required")
	}
}

// listPostmortems returns postmortems with optional filters.
func (s *Server) listPostmortems(w http.ResponseWriter, r *http.Request, mem *memory.Memory) {
	status := r.URL.Query().Get("status")
	severity := r.URL.Query().Get("severity")

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			limit = n
		}
	}

	postmortems, err := mem.GetPostmortems(status, severity, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if postmortems == nil {
		postmortems = []memory.Postmortem{}
	}

	writeJSON(w, map[string]any{
		"postmortems": postmortems,
		"count":       len(postmortems),
	})
}

// createPostmortem creates a new postmortem.
func (s *Server) createPostmortem(w http.ResponseWriter, r *http.Request, mem *memory.Memory) {
	var input memory.PostmortemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if input.WhatHappened == "" {
		writeError(w, http.StatusBadRequest, "whatHappened is required")
		return
	}

	pm, err := mem.StorePostmortem(input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, pm)
}

// handlePostmortemDetail handles GET/PUT/DELETE /api/postmortems/{id}
func (s *Server) handlePostmortemDetail(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "memory not available")
		return
	}

	// Extract ID from path: /api/postmortems/{id} or /api/postmortems/{id}/resolve
	path := r.URL.Path[len("/api/postmortems/"):]
	parts := strings.Split(path, "/")
	id := parts[0]
	if id == "" {
		writeError(w, http.StatusBadRequest, "postmortem ID required")
		return
	}

	// Check for /resolve action
	if len(parts) > 1 && parts[1] == "resolve" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required for resolve")
			return
		}
		s.resolvePostmortem(w, id, mem)
		return
	}

	// Check for /learnings action
	if len(parts) > 1 && parts[1] == "learnings" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "POST required for learnings conversion")
			return
		}
		s.convertPostmortemToLearnings(w, id, mem)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getPostmortem(w, id, mem)
	case http.MethodPut:
		s.updatePostmortem(w, r, id, mem)
	case http.MethodDelete:
		s.deletePostmortem(w, id, mem)
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET, PUT, or DELETE required")
	}
}

// getPostmortem retrieves a single postmortem.
func (s *Server) getPostmortem(w http.ResponseWriter, id string, mem *memory.Memory) {
	pm, err := mem.GetPostmortem(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if pm == nil {
		writeError(w, http.StatusNotFound, "postmortem not found")
		return
	}

	writeJSON(w, pm)
}

// updatePostmortem updates a postmortem.
func (s *Server) updatePostmortem(w http.ResponseWriter, r *http.Request, id string, mem *memory.Memory) {
	var input memory.PostmortemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	err := mem.UpdatePostmortem(id, input)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Return updated postmortem
	pm, _ := mem.GetPostmortem(id)
	writeJSON(w, pm)
}

// deletePostmortem deletes a postmortem.
func (s *Server) deletePostmortem(w http.ResponseWriter, id string, mem *memory.Memory) {
	err := mem.DeletePostmortem(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, map[string]any{"deleted": true, "id": id})
}

// resolvePostmortem marks a postmortem as resolved.
func (s *Server) resolvePostmortem(w http.ResponseWriter, id string, mem *memory.Memory) {
	err := mem.ResolvePostmortem(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Return updated postmortem
	pm, _ := mem.GetPostmortem(id)
	writeJSON(w, pm)
}

// convertPostmortemToLearnings converts a postmortem's lessons to learnings.
func (s *Server) convertPostmortemToLearnings(w http.ResponseWriter, id string, mem *memory.Memory) {
	learningIDs, err := mem.ConvertPostmortemToLearning(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, map[string]any{
		"created":     len(learningIDs),
		"learningIds": learningIDs,
	})
}

// handlePostmortemStats returns aggregated postmortem statistics.
// GET /api/postmortems/stats
func (s *Server) handlePostmortemStats(w http.ResponseWriter, r *http.Request) {
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

	stats, err := mem.GetPostmortemStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, stats)
}
