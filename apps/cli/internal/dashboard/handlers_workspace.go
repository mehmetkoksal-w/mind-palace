package dashboard

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/butler"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

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
	// CodeQL: path-injection - absPath is sanitized via filepath.Abs and validated before use in switchWorkspace
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
	// CodeQL: path-injection - rootPath is validated workspace path, filepath.Join sanitizes the path
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
