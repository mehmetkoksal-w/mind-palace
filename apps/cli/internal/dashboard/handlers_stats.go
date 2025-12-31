package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// handleBrief returns a workspace briefing.
func (s *Server) handleBrief(w http.ResponseWriter, r *http.Request) {
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

	result := map[string]any{
		"agents":        []memory.ActiveAgent{},
		"learnings":     []memory.Learning{},
		"hotspots":      []memory.FileIntel{},
		"fileLearnings": []memory.Learning{},
	}

	// Active agents
	agents, _ := mem.GetActiveAgents(5 * time.Minute)
	if agents != nil {
		result["agents"] = agents
	}

	// Conflict check
	if path != "" {
		conflict, _ := mem.CheckConflict("", path)
		if conflict != nil {
			result["conflict"] = conflict
		}

		// File intel
		intel, _ := mem.GetFileIntel(path)
		result["fileIntel"] = intel

		// File learnings
		learnings, _ := mem.GetFileLearnings(path)
		if learnings != nil {
			result["fileLearnings"] = learnings
		}
	}

	// Relevant learnings
	relevantLearnings, _ := mem.GetRelevantLearnings(path, "", 10)
	if relevantLearnings != nil {
		result["learnings"] = relevantLearnings
	}

	// Hotspots
	hotspots, _ := mem.GetFileHotspots(5)
	if hotspots != nil {
		result["hotspots"] = hotspots
	}

	writeJSON(w, result)
}

// handleStats returns overall statistics.
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	// Get thread-safe copies of resources
	s.mu.RLock()
	mem := s.memory
	corr := s.corridor
	b := s.butler
	root := s.root
	s.mu.RUnlock()

	result := map[string]any{}

	// Workspace info
	result["workspace"] = s.getWorkspaceInfoSafe(root, b)

	// Memory stats - use count queries for efficiency
	if mem != nil {
		totalSessions, _ := mem.CountSessions(false)
		activeSessions, _ := mem.CountSessions(true)
		learningCount, _ := mem.CountLearnings()
		filesTracked, _ := mem.CountFilesTracked()

		result["sessions"] = map[string]any{
			"total":  totalSessions,
			"active": activeSessions,
		}
		result["learnings"] = learningCount
		result["filesTracked"] = filesTracked
	}

	// Corridor stats
	if corr != nil {
		corridorStats, _ := corr.Stats()
		result["corridor"] = corridorStats
	}

	// Butler stats
	if b != nil {
		rooms := b.ListRooms()
		result["rooms"] = len(rooms)
	}

	writeJSON(w, result)
}

// handleDecayStats returns confidence decay statistics.
// GET /api/decay/stats
func (s *Server) handleDecayStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	s.mu.RLock()
	mem := s.memory
	b := s.butler
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "memory not available")
		return
	}

	// Get decay config
	cfg := memory.DefaultDecayConfig()
	if b != nil {
		palaceCfg := b.Config()
		if palaceCfg != nil && palaceCfg.ConfidenceDecay != nil {
			dc := palaceCfg.ConfidenceDecay
			cfg.Enabled = dc.Enabled
			if dc.DecayDays > 0 {
				cfg.DecayDays = dc.DecayDays
			}
			if dc.DecayRate > 0 {
				cfg.DecayRate = dc.DecayRate
			}
			if dc.DecayInterval > 0 {
				cfg.DecayInterval = dc.DecayInterval
			}
			if dc.MinConfidence > 0 {
				cfg.MinConfidence = dc.MinConfidence
			}
		}
	}

	stats, err := mem.GetDecayStats(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, map[string]any{
		"config": map[string]any{
			"enabled":       cfg.Enabled,
			"decayDays":     cfg.DecayDays,
			"decayRate":     cfg.DecayRate,
			"decayInterval": cfg.DecayInterval,
			"minConfidence": cfg.MinConfidence,
		},
		"stats": stats,
	})
}

// handleDecayPreview returns a preview of what would be decayed.
// GET /api/decay/preview?limit=20
func (s *Server) handleDecayPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	s.mu.RLock()
	mem := s.memory
	b := s.butler
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "memory not available")
		return
	}

	// Get decay config
	cfg := memory.DefaultDecayConfig()
	if b != nil {
		palaceCfg := b.Config()
		if palaceCfg != nil && palaceCfg.ConfidenceDecay != nil {
			dc := palaceCfg.ConfidenceDecay
			cfg.Enabled = dc.Enabled
			if dc.DecayDays > 0 {
				cfg.DecayDays = dc.DecayDays
			}
			if dc.DecayRate > 0 {
				cfg.DecayRate = dc.DecayRate
			}
			if dc.DecayInterval > 0 {
				cfg.DecayInterval = dc.DecayInterval
			}
			if dc.MinConfidence > 0 {
				cfg.MinConfidence = dc.MinConfidence
			}
		}
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if n, err := parseInt(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	result, err := mem.PreviewDecay(cfg, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, result)
}

// parseInt parses a string to int, returning error on failure
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// handleSmartBriefing generates a smart briefing for the given context.
// POST /api/briefings/smart
// Body: { "context": "file|room|task|workspace", "contextPath": "...", "style": "summary|detailed|actionable" }
func (s *Server) handleSmartBriefing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	s.mu.RLock()
	mem := s.memory
	b := s.butler
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "memory not available")
		return
	}

	var req struct {
		Context     string `json:"context"`
		ContextPath string `json:"contextPath"`
		Style       string `json:"style"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Context == "" {
		req.Context = "workspace"
	}
	if req.Style == "" {
		req.Style = "summary"
	}

	// Build basic briefing data
	result := map[string]any{
		"context":     req.Context,
		"contextPath": req.ContextPath,
		"style":       req.Style,
		"generatedAt": time.Now(),
	}

	// Gather relevant data based on context
	switch req.Context {
	case "file":
		learnings, _ := mem.GetFileLearnings(req.ContextPath)
		result["learnings"] = learnings
		intel, _ := mem.GetFileIntel(req.ContextPath)
		result["fileIntel"] = intel

	case "room":
		learnings, _ := mem.GetRelevantLearnings("", req.ContextPath, 20)
		result["learnings"] = learnings

	case "task":
		if b != nil && b.GetEmbedder() != nil {
			opts := memory.DefaultSemanticSearchOptions()
			opts.Limit = 20
			results, _ := mem.SemanticSearch(b.GetEmbedder(), req.ContextPath, opts)
			result["searchResults"] = results
		}

	default: // workspace
		totalLearnings, _ := mem.CountLearnings()
		totalSessions, _ := mem.CountSessions(false)
		learnings, _ := mem.GetRelevantLearnings("", "", 10)
		result["stats"] = map[string]any{
			"totalLearnings": totalLearnings,
			"totalSessions":  totalSessions,
		}
		result["learnings"] = learnings
	}

	// Get warnings
	warnings := []map[string]any{}

	// Decay warnings
	cfg := memory.DefaultDecayConfig()
	if b != nil {
		palaceCfg := b.Config()
		if palaceCfg != nil && palaceCfg.ConfidenceDecay != nil {
			dc := palaceCfg.ConfidenceDecay
			cfg.Enabled = dc.Enabled
			if dc.DecayDays > 0 {
				cfg.DecayDays = dc.DecayDays
			}
		}
	}
	decayStats, _ := mem.GetDecayStats(cfg)
	if decayStats != nil && decayStats.AtRiskCount > 0 {
		warnings = append(warnings, map[string]any{
			"type":    "decay",
			"message": fmt.Sprintf("%d learnings at risk of decay", decayStats.AtRiskCount),
			"count":   decayStats.AtRiskCount,
		})
	}

	// Contradiction warnings
	contradictions, _ := mem.GetContradictionSummary(5)
	if contradictions != nil && contradictions.TotalContradictionLinks > 0 {
		warnings = append(warnings, map[string]any{
			"type":    "contradiction",
			"message": fmt.Sprintf("%d contradictions detected", contradictions.TotalContradictionLinks),
			"count":   contradictions.TotalContradictionLinks,
		})
	}

	result["warnings"] = warnings

	writeJSON(w, result)
}
