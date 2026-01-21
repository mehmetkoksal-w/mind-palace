package dashboard

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/memory"
)

// PatternResponse represents a pattern in API responses.
type PatternResponse struct {
	ID               string                 `json:"id"`
	Category         string                 `json:"category"`
	Subcategory      string                 `json:"subcategory"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	DetectorID       string                 `json:"detectorId"`
	Confidence       float64                `json:"confidence"`
	FrequencyScore   float64                `json:"frequencyScore"`
	ConsistencyScore float64                `json:"consistencyScore"`
	SpreadScore      float64                `json:"spreadScore"`
	AgeScore         float64                `json:"ageScore"`
	Status           string                 `json:"status"`
	Authority        string                 `json:"authority"`
	LearningID       string                 `json:"learningId"`
	Locations        []LocationResponse     `json:"locations"`
	Outliers         []LocationResponse     `json:"outliers"`
	Metadata         map[string]interface{} `json:"metadata"`
	FirstSeen        string                 `json:"firstSeen"`
	LastSeen         string                 `json:"lastSeen"`
	CreatedAt        string                 `json:"createdAt"`
	UpdatedAt        string                 `json:"updatedAt"`
}

// LocationResponse represents a pattern location in API responses.
type LocationResponse struct {
	ID            string `json:"id"`
	PatternID     string `json:"patternId"`
	FilePath      string `json:"filePath"`
	LineStart     int    `json:"lineStart"`
	LineEnd       int    `json:"lineEnd"`
	Snippet       string `json:"snippet"`
	IsOutlier     bool   `json:"isOutlier"`
	OutlierReason string `json:"outlierReason"`
	CreatedAt     string `json:"createdAt"`
}

// PatternStatsResponse represents pattern statistics.
type PatternStatsResponse struct {
	Total             int            `json:"total"`
	Discovered        int            `json:"discovered"`
	Approved          int            `json:"approved"`
	Ignored           int            `json:"ignored"`
	ByCategory        map[string]int `json:"byCategory"`
	AverageConfidence float64        `json:"averageConfidence"`
}

// handlePatterns handles GET /api/patterns
func (s *Server) handlePatterns(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "Memory not initialized")
		return
	}

	// Parse query parameters
	category := r.URL.Query().Get("category")
	status := r.URL.Query().Get("status")
	minConfidenceStr := r.URL.Query().Get("minConfidence")
	limitStr := r.URL.Query().Get("limit")

	limit := 50
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	minConfidence := 0.0
	if minConfidenceStr != "" {
		if parsed, err := strconv.ParseFloat(minConfidenceStr, 64); err == nil {
			minConfidence = parsed
		}
	}

	// Build filters
	filters := memory.PatternFilters{
		Category:      category,
		Status:        status,
		MinConfidence: minConfidence,
		Limit:         limit,
	}

	// Get patterns from memory
	patterns, err := mem.GetPatterns(filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get patterns: "+err.Error())
		return
	}

	// Convert to response format
	response := make([]PatternResponse, 0, len(patterns))
	for i := range patterns {
		p := &patterns[i]
		// Get locations for this pattern
		allLocations, _ := mem.GetPatternLocations(p.ID)
		locations, outliers := splitLocations(allLocations)

		pr := patternToResponse(*p, locations, outliers)
		response = append(response, pr)
	}

	writeJSON(w, map[string]any{
		"patterns": response,
		"count":    len(response),
	})
}

// handlePatternDetail handles GET/POST /api/patterns/{id}/*
func (s *Server) handlePatternDetail(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "Memory not initialized")
		return
	}

	// Parse path: /api/patterns/{id} or /api/patterns/{id}/approve or /api/patterns/{id}/ignore
	path := strings.TrimPrefix(r.URL.Path, "/api/patterns/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "Pattern ID required")
		return
	}

	id := parts[0]

	// Check for action
	if len(parts) > 1 {
		action := parts[1]
		switch action {
		case "approve":
			s.handlePatternApprove(w, r, mem, id)
			return
		case "ignore":
			s.handlePatternIgnore(w, r, mem, id)
			return
		}
	}

	// GET single pattern
	if r.Method == http.MethodGet {
		pattern, err := mem.GetPattern(id)
		if err != nil {
			writeError(w, http.StatusNotFound, "Pattern not found")
			return
		}

		allLocations, _ := mem.GetPatternLocations(id)
		locations, outliers := splitLocations(allLocations)

		writeJSON(w, patternToResponse(*pattern, locations, outliers))
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// handlePatternApprove handles POST /api/patterns/{id}/approve
func (s *Server) handlePatternApprove(w http.ResponseWriter, r *http.Request, mem *memory.Memory, id string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse request body for optional withLearning flag
	var req struct {
		WithLearning bool `json:"withLearning"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req) // Ignore error for empty body

	var learningID string
	if req.WithLearning {
		// Approve with learning creation
		var err error
		learningID, err = mem.ApprovePatternWithLearning(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to approve pattern with learning: "+err.Error())
			return
		}
	} else {
		// Simple approval
		if err := mem.ApprovePattern(id, ""); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to approve pattern: "+err.Error())
			return
		}
	}

	pattern, err := mem.GetPattern(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Pattern approved but failed to fetch: "+err.Error())
		return
	}

	allLocations, _ := mem.GetPatternLocations(id)
	locations, outliers := splitLocations(allLocations)

	response := patternToResponse(*pattern, locations, outliers)
	if learningID != "" {
		writeJSON(w, map[string]any{
			"pattern":    response,
			"learningId": learningID,
		})
	} else {
		writeJSON(w, response)
	}
}

// handlePatternIgnore handles POST /api/patterns/{id}/ignore
func (s *Server) handlePatternIgnore(w http.ResponseWriter, r *http.Request, mem *memory.Memory, id string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := mem.IgnorePattern(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to ignore pattern: "+err.Error())
		return
	}

	pattern, err := mem.GetPattern(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Pattern ignored but failed to fetch: "+err.Error())
		return
	}

	allLocations, _ := mem.GetPatternLocations(id)
	locations, outliers := splitLocations(allLocations)

	writeJSON(w, patternToResponse(*pattern, locations, outliers))
}

// handlePatternBulkApprove handles POST /api/patterns/bulk-approve
func (s *Server) handlePatternBulkApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "Memory not initialized")
		return
	}

	// Parse request body
	var req struct {
		IDs          []string `json:"ids"`
		WithLearning bool     `json:"withLearning"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "No pattern IDs provided")
		return
	}

	// Approve each pattern
	approved := 0
	var patterns []PatternResponse
	var learningIDs []string
	for _, id := range req.IDs {
		var err error
		if req.WithLearning {
			var learningID string
			learningID, err = mem.ApprovePatternWithLearning(id)
			if err == nil {
				learningIDs = append(learningIDs, learningID)
			}
		} else {
			err = mem.ApprovePattern(id, "")
		}

		if err == nil {
			approved++
			if pattern, err := mem.GetPattern(id); err == nil {
				allLocations, _ := mem.GetPatternLocations(id)
				locations, outliers := splitLocations(allLocations)
				patterns = append(patterns, patternToResponse(*pattern, locations, outliers))
			}
		}
	}

	response := map[string]any{
		"approved": approved,
		"patterns": patterns,
	}
	if req.WithLearning && len(learningIDs) > 0 {
		response["learningIds"] = learningIDs
	}
	writeJSON(w, response)
}

// handlePatternStats handles GET /api/patterns/stats
func (s *Server) handlePatternStats(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "Memory not initialized")
		return
	}

	// Get all patterns to calculate stats
	allPatterns, err := mem.GetPatterns(memory.PatternFilters{Limit: 10000})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get patterns: "+err.Error())
		return
	}

	stats := PatternStatsResponse{
		Total:      len(allPatterns),
		ByCategory: make(map[string]int),
	}

	totalConfidence := 0.0
	for i := range allPatterns {
		p := &allPatterns[i]
		switch p.Status {
		case "discovered":
			stats.Discovered++
		case "approved":
			stats.Approved++
		case "ignored":
			stats.Ignored++
		}

		stats.ByCategory[p.Category]++
		totalConfidence += p.Confidence
	}

	if len(allPatterns) > 0 {
		stats.AverageConfidence = totalConfidence / float64(len(allPatterns))
	}

	writeJSON(w, stats)
}

// splitLocations separates locations into regular locations and outliers.
func splitLocations(all []memory.PatternLocation) (locations, outliers []memory.PatternLocation) {
	for i := range all {
		if all[i].IsOutlier {
			outliers = append(outliers, all[i])
		} else {
			locations = append(locations, all[i])
		}
	}
	return
}

// patternToResponse converts a memory.Pattern to a PatternResponse.
func patternToResponse(p memory.Pattern, locations, outliers []memory.PatternLocation) PatternResponse {
	locResponses := make([]LocationResponse, 0, len(locations))
	for i := range locations {
		locResponses = append(locResponses, locationToResponse(locations[i]))
	}

	outlierResponses := make([]LocationResponse, 0, len(outliers))
	for i := range outliers {
		outlierResponses = append(outlierResponses, locationToResponse(outliers[i]))
	}

	return PatternResponse{
		ID:               p.ID,
		Category:         p.Category,
		Subcategory:      p.Subcategory,
		Name:             p.Name,
		Description:      p.Description,
		DetectorID:       p.DetectorID,
		Confidence:       p.Confidence,
		FrequencyScore:   p.FrequencyScore,
		ConsistencyScore: p.ConsistencyScore,
		SpreadScore:      p.SpreadScore,
		AgeScore:         p.AgeScore,
		Status:           p.Status,
		Authority:        p.Authority,
		LearningID:       p.LearningID,
		Locations:        locResponses,
		Outliers:         outlierResponses,
		Metadata:         p.Metadata,
		FirstSeen:        p.FirstSeen.Format("2006-01-02T15:04:05Z"),
		LastSeen:         p.LastSeen.Format("2006-01-02T15:04:05Z"),
		CreatedAt:        p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:        p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// locationToResponse converts a memory.PatternLocation to a LocationResponse.
func locationToResponse(loc memory.PatternLocation) LocationResponse {
	return LocationResponse{
		ID:            loc.ID,
		PatternID:     loc.PatternID,
		FilePath:      loc.FilePath,
		LineStart:     loc.LineStart,
		LineEnd:       loc.LineEnd,
		Snippet:       loc.Snippet,
		IsOutlier:     loc.IsOutlier,
		OutlierReason: loc.OutlierReason,
		CreatedAt:     loc.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
