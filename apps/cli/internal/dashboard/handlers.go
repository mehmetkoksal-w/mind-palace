package dashboard

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/butler"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/corridor"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

// handleHealth returns server health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
	})
}

// handleRooms returns the list of rooms.
func (s *Server) handleRooms(w http.ResponseWriter, r *http.Request) {
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

	rooms := b.ListRooms()
	if rooms == nil {
		rooms = []model.Room{}
	}

	writeJSON(w, map[string]any{
		"rooms": rooms,
		"count": len(rooms),
	})
}

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

// handleGraph returns call graph data.
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

// getWorkspaceInfo returns information about the current workspace.
// Note: This acquires the lock internally, use getWorkspaceInfoSafe if you already have the values.
func (s *Server) getWorkspaceInfo() map[string]any {
	s.mu.RLock()
	root := s.root
	b := s.butler
	s.mu.RUnlock()

	return s.getWorkspaceInfoSafe(root, b)
}

// getWorkspaceInfoSafe returns workspace info using pre-fetched values (thread-safe).
func (s *Server) getWorkspaceInfoSafe(root string, b *butler.Butler) map[string]any {
	info := map[string]any{
		"path": root,
		"name": getWorkspaceName(root),
	}

	// Get index info from butler if available
	if b != nil {
		indexInfo := b.GetIndexInfo()
		if indexInfo != nil {
			info["lastScan"] = indexInfo.LastScan
			info["fileCount"] = indexInfo.FileCount
			info["status"] = indexInfo.Status
		}
	}

	return info
}

// getWorkspaceName extracts workspace name from path.
func getWorkspaceName(path string) string {
	if path == "" {
		return "Unknown"
	}
	// Get last component of path
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			if i < len(path)-1 {
				return path[i+1:]
			}
		}
	}
	return path
}

// WorkspaceInfo represents a workspace for the API.
type WorkspaceInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	IsCurrent bool   `json:"isCurrent"`
	HasPalace bool   `json:"hasPalace"`
}

// handleWorkspaces returns available workspaces (current + linked corridors).
func (s *Server) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	s.mu.RLock()
	currentRoot := s.root
	corr := s.corridor
	s.mu.RUnlock()

	workspaces := []WorkspaceInfo{}

	// Add current workspace
	workspaces = append(workspaces, WorkspaceInfo{
		Name:      getWorkspaceName(currentRoot),
		Path:      currentRoot,
		IsCurrent: true,
		HasPalace: true,
	})

	// Add linked workspaces from corridors
	if corr != nil {
		links, err := corr.GetLinks()
		if err == nil {
			for _, link := range links {
				// Check if the linked workspace has a .palace directory
				hasPalace := false
				palacePath := filepath.Join(link.Path, ".palace")
				if _, err := os.Stat(palacePath); err == nil {
					hasPalace = true
				}

				workspaces = append(workspaces, WorkspaceInfo{
					Name:      link.Name,
					Path:      link.Path,
					IsCurrent: link.Path == currentRoot,
					HasPalace: hasPalace,
				})
			}
		}
	}

	writeJSON(w, map[string]any{
		"workspaces": workspaces,
		"current":    currentRoot,
	})
}

// handleWorkspaceSwitch switches to a different workspace.
func (s *Server) handleWorkspaceSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid path: "+err.Error())
		return
	}

	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		writeError(w, http.StatusBadRequest, "path does not exist")
		return
	}

	// Check if already on this workspace
	s.mu.RLock()
	currentRoot := s.root
	s.mu.RUnlock()

	if absPath == currentRoot {
		writeJSON(w, map[string]any{
			"success":   true,
			"message":   "already on this workspace",
			"workspace": getWorkspaceName(absPath),
			"path":      absPath,
		})
		return
	}

	// Switch workspace
	if err := s.switchWorkspace(absPath); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to switch: "+err.Error())
		return
	}

	writeJSON(w, map[string]any{
		"success":   true,
		"message":   "switched workspace",
		"workspace": getWorkspaceName(absPath),
		"path":      absPath,
	})
}

// switchWorkspace switches the server to a different workspace.
// Opens new resources FIRST before closing old ones to prevent data loss.
func (s *Server) switchWorkspace(rootPath string) error {
	// Open new resources FIRST (before acquiring lock or closing old ones)
	var newMem *memory.Memory
	var newButler *butler.Butler

	// Try to open new memory database
	mem, err := memory.Open(rootPath)
	if err == nil {
		newMem = mem
	}
	// Non-fatal if memory fails - continue without it

	// Try to open new butler for code search
	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); err == nil {
		db, err := index.Open(dbPath)
		if err == nil {
			b, err := butler.New(db, rootPath)
			if err == nil {
				newButler = b
			} else {
				// butler.New failed - close the db to prevent leak
				db.Close()
			}
		}
	}
	// Non-fatal if butler fails - continue without it

	// Now acquire lock and swap resources
	s.mu.Lock()
	defer s.mu.Unlock()

	// Close old resources
	if s.butler != nil {
		s.butler.Close()
	}
	if s.memory != nil {
		s.memory.Close()
	}

	// Assign new resources
	s.butler = newButler
	s.memory = newMem
	s.root = rootPath

	return nil
}
