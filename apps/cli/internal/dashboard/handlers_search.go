package dashboard

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/butler"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// handleSearch performs a unified search.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter required")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			limit = n
		}
	}

	// Get thread-safe copies of resources
	s.mu.RLock()
	b := s.butler
	mem := s.memory
	corr := s.corridor
	s.mu.RUnlock()

	result := map[string]any{
		"symbols":   []any{},
		"learnings": []any{},
		"corridor":  []any{},
	}

	// Search code symbols
	if b != nil {
		opts := butler.SearchOptions{
			Limit: limit,
		}
		symbols, err := b.Search(query, opts)
		if err == nil && symbols != nil {
			result["symbols"] = symbols
		}
	}

	// Search learnings
	if mem != nil {
		learnings, err := mem.SearchLearnings(query, limit)
		if err == nil && learnings != nil {
			result["learnings"] = learnings
		}
	}

	// Search personal corridor
	if corr != nil {
		personal, err := corr.GetPersonalLearnings(query, limit)
		if err == nil && personal != nil {
			result["corridor"] = personal
		}
	}

	writeJSON(w, result)
}

// handleSemanticSearch performs semantic/hybrid search.
func (s *Server) handleSemanticSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter required")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	kinds := r.URL.Query()["kind"] // Can have multiple: ?kind=idea&kind=learning

	s.mu.RLock()
	b := s.butler
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "memory not available")
		return
	}

	// Try to get embedder from butler
	var embedder memory.Embedder
	if b != nil {
		embedder = b.GetEmbedder()
	}

	// Build search options
	opts := memory.DefaultSemanticSearchOptions()
	opts.Limit = limit
	if len(kinds) > 0 {
		opts.Kinds = kinds
	}

	// Perform hybrid search (falls back to keyword if no embedder)
	results, err := mem.HybridSearch(embedder, query, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	searchMode := "hybrid"
	if embedder == nil {
		searchMode = "keyword"
	}

	writeJSON(w, map[string]any{
		"query":      query,
		"searchMode": searchMode,
		"count":      len(results),
		"results":    results,
	})
}

// handleGraph returns call graph data.
// If depth parameter is provided, returns recursive call chain instead of direct callers/callees.
func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
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

	// Extract symbol from path: /api/graph/{symbol}
	pathPart := strings.TrimPrefix(r.URL.Path, "/api/graph/")
	symbol := strings.TrimSuffix(pathPart, "/")

	if symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol required")
		return
	}

	filePath := r.URL.Query().Get("file")

	// Check if depth parameter is provided for call chain tracing
	depthStr := r.URL.Query().Get("depth")
	if depthStr != "" {
		depth, err := strconv.Atoi(depthStr)
		if err != nil || depth < 1 {
			depth = 3
		}
		if depth > 10 {
			depth = 10
		}

		direction := r.URL.Query().Get("direction")
		if direction == "" {
			direction = "up"
		}

		chain, err := b.GetCallChain(symbol, filePath, direction, depth)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, chain)
		return
	}

	// Default: return direct callers/callees
	callers, _ := b.GetIncomingCalls(symbol)
	callees, _ := b.GetOutgoingCalls(symbol, filePath)

	if callers == nil {
		callers = []index.CallSite{}
	}
	if callees == nil {
		callees = []index.CallSite{}
	}

	writeJSON(w, map[string]any{
		"symbol":  symbol,
		"callers": callers,
		"callees": callees,
	})
}
