package dashboard

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/contracts"
)

// ContractResponse represents a contract in API responses.
type ContractResponse struct {
	ID              string             `json:"id"`
	Method          string             `json:"method"`
	Endpoint        string             `json:"endpoint"`
	EndpointPattern string             `json:"endpointPattern"`
	Backend         BackendResponse    `json:"backend"`
	FrontendCalls   []FrontendCallResp `json:"frontendCalls"`
	Mismatches      []MismatchResponse `json:"mismatches"`
	Status          string             `json:"status"`
	Authority       string             `json:"authority"`
	Confidence      float64            `json:"confidence"`
	FirstSeen       string             `json:"firstSeen"`
	LastSeen        string             `json:"lastSeen"`
	CreatedAt       string             `json:"createdAt"`
	UpdatedAt       string             `json:"updatedAt"`
}

// BackendResponse represents backend endpoint info.
type BackendResponse struct {
	File           string `json:"file"`
	Line           int    `json:"line"`
	Framework      string `json:"framework"`
	Handler        string `json:"handler"`
	RequestSchema  any    `json:"requestSchema,omitempty"`
	ResponseSchema any    `json:"responseSchema,omitempty"`
}

// FrontendCallResp represents a frontend API call.
type FrontendCallResp struct {
	ID             string `json:"id"`
	File           string `json:"file"`
	Line           int    `json:"line"`
	CallType       string `json:"callType"`
	ExpectedSchema any    `json:"expectedSchema,omitempty"`
}

// MismatchResponse represents a contract mismatch.
type MismatchResponse struct {
	FieldPath    string `json:"fieldPath"`
	Type         string `json:"type"`
	Severity     string `json:"severity"`
	Description  string `json:"description"`
	BackendType  string `json:"backendType,omitempty"`
	FrontendType string `json:"frontendType,omitempty"`
}

// ContractStatsResponse represents contract statistics.
type ContractStatsResponse struct {
	Total         int            `json:"total"`
	Discovered    int            `json:"discovered"`
	Verified      int            `json:"verified"`
	Mismatch      int            `json:"mismatch"`
	Ignored       int            `json:"ignored"`
	ByMethod      map[string]int `json:"byMethod"`
	TotalErrors   int            `json:"totalErrors"`
	TotalWarnings int            `json:"totalWarnings"`
	TotalCalls    int            `json:"totalCalls"`
}

// handleContracts handles GET /api/contracts
func (s *Server) handleContracts(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "Memory not initialized")
		return
	}

	store := contracts.NewStore(mem.DB())

	// Ensure tables exist (defensive)
	if err := store.CreateTables(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to initialize contracts: "+err.Error())
		return
	}

	// Parse query parameters
	method := r.URL.Query().Get("method")
	status := r.URL.Query().Get("status")
	endpoint := r.URL.Query().Get("endpoint")
	hasMismatchesStr := r.URL.Query().Get("hasMismatches")
	limitStr := r.URL.Query().Get("limit")

	limit := 50
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	hasMismatches := false
	if hasMismatchesStr == "true" {
		hasMismatches = true
	}

	// Build filter
	filter := contracts.ContractFilter{
		Method:        method,
		Endpoint:      endpoint,
		HasMismatches: hasMismatches,
		Limit:         limit,
	}
	if status != "" {
		filter.Status = contracts.ContractStatus(status)
	}

	// Get contracts from store
	contractList, err := store.ListContracts(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get contracts: "+err.Error())
		return
	}

	// Convert to response format
	response := make([]ContractResponse, 0, len(contractList))
	for _, c := range contractList {
		response = append(response, contractToResponse(c))
	}

	writeJSON(w, map[string]any{
		"contracts": response,
		"count":     len(response),
	})
}

// handleContractDetail handles GET/POST /api/contracts/{id}/*
func (s *Server) handleContractDetail(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "Memory not initialized")
		return
	}

	store := contracts.NewStore(mem.DB())

	// Ensure tables exist (defensive)
	if err := store.CreateTables(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to initialize contracts: "+err.Error())
		return
	}

	// Parse path: /api/contracts/{id} or /api/contracts/{id}/verify or /api/contracts/{id}/ignore
	path := strings.TrimPrefix(r.URL.Path, "/api/contracts/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "Contract ID required")
		return
	}

	id := parts[0]

	// Check for action
	if len(parts) > 1 {
		action := parts[1]
		switch action {
		case "verify":
			s.handleContractVerify(w, r, store, id)
			return
		case "ignore":
			s.handleContractIgnore(w, r, store, id)
			return
		}
	}

	// GET single contract
	if r.Method == http.MethodGet {
		contract, err := store.GetContract(id)
		if err != nil || contract == nil {
			writeError(w, http.StatusNotFound, "Contract not found")
			return
		}

		writeJSON(w, contractToResponse(contract))
		return
	}

	writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// handleContractVerify handles POST /api/contracts/{id}/verify
func (s *Server) handleContractVerify(w http.ResponseWriter, r *http.Request, store *contracts.Store, id string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := store.UpdateStatus(id, contracts.ContractVerified); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to verify contract: "+err.Error())
		return
	}

	writeJSON(w, map[string]any{
		"success": true,
		"message": "Contract verified",
	})
}

// handleContractIgnore handles POST /api/contracts/{id}/ignore
func (s *Server) handleContractIgnore(w http.ResponseWriter, r *http.Request, store *contracts.Store, id string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := store.UpdateStatus(id, contracts.ContractIgnored); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to ignore contract: "+err.Error())
		return
	}

	writeJSON(w, map[string]any{
		"success": true,
		"message": "Contract ignored",
	})
}

// handleContractStats handles GET /api/contracts/stats
func (s *Server) handleContractStats(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "Memory not initialized")
		return
	}

	store := contracts.NewStore(mem.DB())

	// Ensure tables exist (defensive)
	if err := store.CreateTables(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to initialize contracts: "+err.Error())
		return
	}

	stats, err := store.GetStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get contract stats: "+err.Error())
		return
	}

	writeJSON(w, ContractStatsResponse{
		Total:         stats.Total,
		Discovered:    stats.Discovered,
		Verified:      stats.Verified,
		Mismatch:      stats.Mismatch,
		Ignored:       stats.Ignored,
		ByMethod:      stats.ByMethod,
		TotalErrors:   stats.TotalErrors,
		TotalWarnings: stats.TotalWarnings,
		TotalCalls:    stats.TotalCalls,
	})
}

// handleContractMismatches handles GET /api/contracts/mismatches
func (s *Server) handleContractMismatches(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "Memory not initialized")
		return
	}

	store := contracts.NewStore(mem.DB())

	// Ensure tables exist (defensive)
	if err := store.CreateTables(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to initialize contracts: "+err.Error())
		return
	}

	// Get contracts with mismatches
	filter := contracts.ContractFilter{
		HasMismatches: true,
		Limit:         100,
	}

	contractList, err := store.ListContracts(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get contracts: "+err.Error())
		return
	}

	// Collect all mismatches with contract context
	type MismatchWithContext struct {
		MismatchResponse
		ContractID string `json:"contractId"`
		Method     string `json:"method"`
		Endpoint   string `json:"endpoint"`
	}

	var mismatches []MismatchWithContext
	for _, c := range contractList {
		for _, m := range c.Mismatches {
			mismatches = append(mismatches, MismatchWithContext{
				MismatchResponse: MismatchResponse{
					FieldPath:    m.FieldPath,
					Type:         string(m.Type),
					Severity:     string(m.Severity),
					Description:  m.Description,
					BackendType:  m.BackendType,
					FrontendType: m.FrontendType,
				},
				ContractID: c.ID,
				Method:     c.Method,
				Endpoint:   c.Endpoint,
			})
		}
	}

	writeJSON(w, map[string]any{
		"mismatches": mismatches,
		"count":      len(mismatches),
	})
}

// contractToResponse converts a Contract to ContractResponse.
func contractToResponse(c *contracts.Contract) ContractResponse {
	frontendCalls := make([]FrontendCallResp, 0, len(c.FrontendCalls))
	for _, fc := range c.FrontendCalls {
		frontendCalls = append(frontendCalls, FrontendCallResp{
			ID:             fc.ID,
			File:           fc.File,
			Line:           fc.Line,
			CallType:       fc.CallType,
			ExpectedSchema: fc.ExpectedSchema,
		})
	}

	mismatches := make([]MismatchResponse, 0, len(c.Mismatches))
	for _, m := range c.Mismatches {
		mismatches = append(mismatches, MismatchResponse{
			FieldPath:    m.FieldPath,
			Type:         string(m.Type),
			Severity:     string(m.Severity),
			Description:  m.Description,
			BackendType:  m.BackendType,
			FrontendType: m.FrontendType,
		})
	}

	return ContractResponse{
		ID:              c.ID,
		Method:          c.Method,
		Endpoint:        c.Endpoint,
		EndpointPattern: c.EndpointPattern,
		Backend: BackendResponse{
			File:           c.Backend.File,
			Line:           c.Backend.Line,
			Framework:      c.Backend.Framework,
			Handler:        c.Backend.Handler,
			RequestSchema:  c.Backend.RequestSchema,
			ResponseSchema: c.Backend.ResponseSchema,
		},
		FrontendCalls: frontendCalls,
		Mismatches:    mismatches,
		Status:        string(c.Status),
		Authority:     c.Authority,
		Confidence:    c.Confidence,
		FirstSeen:     c.FirstSeen.Format("2006-01-02T15:04:05Z"),
		LastSeen:      c.LastSeen.Format("2006-01-02T15:04:05Z"),
		CreatedAt:     c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:     c.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
