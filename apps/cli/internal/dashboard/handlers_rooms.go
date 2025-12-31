package dashboard

import (
	"net/http"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

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
