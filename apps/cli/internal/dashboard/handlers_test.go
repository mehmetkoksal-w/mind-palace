package dashboard

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// createTestServer creates a server for testing with nil dependencies
func createTestServer() *Server {
	return &Server{
		butler:   nil,
		memory:   nil,
		corridor: nil,
		port:     0,
		root:     "/test/workspace",
	}
}

func TestHandleHealth(t *testing.T) {
	server := createTestServer()

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", response["status"])
	}

	if _, ok := response["timestamp"]; !ok {
		t.Error("expected timestamp in response")
	}
}

func TestHandleRooms(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("POST", "/api/rooms", nil)
		w := httptest.NewRecorder()

		server.handleRooms(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("butler not available", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/rooms", nil)
		w := httptest.NewRecorder()

		server.handleRooms(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})
}

func TestHandleSessions(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("POST", "/api/sessions", nil)
		w := httptest.NewRecorder()

		server.handleSessions(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("memory not available", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/sessions", nil)
		w := httptest.NewRecorder()

		server.handleSessions(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("with query parameters", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/sessions?active=true&limit=10", nil)
		w := httptest.NewRecorder()

		server.handleSessions(w, req)

		// Should still fail due to no memory, but should parse params without panic
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})
}

func TestHandleSessionDetail(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("POST", "/api/sessions/123", nil)
		w := httptest.NewRecorder()

		server.handleSessionDetail(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("memory not available", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/sessions/123", nil)
		w := httptest.NewRecorder()

		server.handleSessionDetail(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("missing session ID", func(t *testing.T) {
		server := createTestServer()
		// Temporarily set memory to non-nil (we'll use a mock in integration tests)
		// For now, test the path parsing by checking request handling
		req := httptest.NewRequest("GET", "/api/sessions/", nil)
		w := httptest.NewRecorder()

		server.handleSessionDetail(w, req)

		// Will fail due to no memory, not bad request
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})
}

func TestHandleActivity(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("DELETE", "/api/activity", nil)
		w := httptest.NewRecorder()

		server.handleActivity(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("memory not available", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/activity", nil)
		w := httptest.NewRecorder()

		server.handleActivity(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("with query parameters", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/activity?sessionId=sess1&path=src/main.go&limit=25", nil)
		w := httptest.NewRecorder()

		server.handleActivity(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})
}

func TestHandleLearnings(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("PUT", "/api/learnings", nil)
		w := httptest.NewRecorder()

		server.handleLearnings(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("memory not available", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/learnings", nil)
		w := httptest.NewRecorder()

		server.handleLearnings(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("with query search", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/learnings?query=authentication", nil)
		w := httptest.NewRecorder()

		server.handleLearnings(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("with scope parameters", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/learnings?scope=file&scopePath=src/main.go&limit=10", nil)
		w := httptest.NewRecorder()

		server.handleLearnings(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})
}

func TestHandleFileIntel(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("POST", "/api/file-intel", nil)
		w := httptest.NewRecorder()

		server.handleFileIntel(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("memory not available", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/file-intel?path=main.go", nil)
		w := httptest.NewRecorder()

		server.handleFileIntel(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("missing path parameter", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/file-intel", nil)
		w := httptest.NewRecorder()

		server.handleFileIntel(w, req)

		// First checks memory, then path
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})
}

func TestHandleCorridors(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("POST", "/api/corridors", nil)
		w := httptest.NewRecorder()

		server.handleCorridors(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("corridor not available", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/corridors", nil)
		w := httptest.NewRecorder()

		server.handleCorridors(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})
}

func TestHandleCorridorPersonal(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("DELETE", "/api/corridors/personal", nil)
		w := httptest.NewRecorder()

		server.handleCorridorPersonal(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("corridor not available", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/corridors/personal", nil)
		w := httptest.NewRecorder()

		server.handleCorridorPersonal(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("with query parameters", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/corridors/personal?query=test&limit=30", nil)
		w := httptest.NewRecorder()

		server.handleCorridorPersonal(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})
}

func TestHandleCorridorLinks(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("PUT", "/api/corridors/links", nil)
		w := httptest.NewRecorder()

		server.handleCorridorLinks(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("corridor not available", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/corridors/links", nil)
		w := httptest.NewRecorder()

		server.handleCorridorLinks(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})
}

func TestHandleSearch(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("POST", "/api/search?q=test", nil)
		w := httptest.NewRecorder()

		server.handleSearch(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("missing query parameter", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/search", nil)
		w := httptest.NewRecorder()

		server.handleSearch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("empty query parameter", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/search?q=", nil)
		w := httptest.NewRecorder()

		server.handleSearch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("valid query with no resources", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/search?q=function&limit=10", nil)
		w := httptest.NewRecorder()

		server.handleSearch(w, req)

		// Should succeed with empty results
		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Check that empty arrays are returned
		if symbols, ok := response["symbols"].([]any); !ok || symbols == nil {
			t.Error("expected symbols array in response")
		}
		if learnings, ok := response["learnings"].([]any); !ok || learnings == nil {
			t.Error("expected learnings array in response")
		}
		if corridor, ok := response["corridor"].([]any); !ok || corridor == nil {
			t.Error("expected corridor array in response")
		}
	})
}

func TestHandleGraph(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("POST", "/api/graph/main", nil)
		w := httptest.NewRecorder()

		server.handleGraph(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("butler not available", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/graph/main", nil)
		w := httptest.NewRecorder()

		server.handleGraph(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("missing symbol", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/graph/", nil)
		w := httptest.NewRecorder()

		server.handleGraph(w, req)

		// First checks butler, then symbol
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})
}

func TestHandleHotspots(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("POST", "/api/hotspots", nil)
		w := httptest.NewRecorder()

		server.handleHotspots(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("memory not available", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/hotspots", nil)
		w := httptest.NewRecorder()

		server.handleHotspots(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("with limit parameter", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/hotspots?limit=5", nil)
		w := httptest.NewRecorder()

		server.handleHotspots(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})
}

func TestHandleAgents(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("DELETE", "/api/agents", nil)
		w := httptest.NewRecorder()

		server.handleAgents(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("memory not available", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/agents", nil)
		w := httptest.NewRecorder()

		server.handleAgents(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})
}

func TestHandleBrief(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("POST", "/api/brief", nil)
		w := httptest.NewRecorder()

		server.handleBrief(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("memory not available", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/brief", nil)
		w := httptest.NewRecorder()

		server.handleBrief(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("with path parameter", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/brief?path=src/main.go", nil)
		w := httptest.NewRecorder()

		server.handleBrief(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})
}

func TestHandleStats(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("POST", "/api/stats", nil)
		w := httptest.NewRecorder()

		server.handleStats(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("success with no resources", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/stats", nil)
		w := httptest.NewRecorder()

		server.handleStats(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Should have workspace info
		if _, ok := response["workspace"]; !ok {
			t.Error("expected workspace in response")
		}
	})
}

func TestHandleWorkspaces(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("POST", "/api/workspaces", nil)
		w := httptest.NewRecorder()

		server.handleWorkspaces(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("returns current workspace", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/workspaces", nil)
		w := httptest.NewRecorder()

		server.handleWorkspaces(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		workspaces, ok := response["workspaces"].([]any)
		if !ok {
			t.Fatal("expected workspaces array")
		}

		if len(workspaces) < 1 {
			t.Error("expected at least one workspace (current)")
		}

		// Check current workspace
		current, ok := response["current"].(string)
		if !ok {
			t.Error("expected current path")
		}
		if current != "/test/workspace" {
			t.Errorf("expected current to be '/test/workspace', got %q", current)
		}
	})
}

func TestHandleWorkspaceSwitch(t *testing.T) {
	t.Run("method not allowed", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("GET", "/api/workspace/switch", nil)
		w := httptest.NewRecorder()

		server.handleWorkspaceSwitch(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		server := createTestServer()
		req := httptest.NewRequest("POST", "/api/workspace/switch", bytes.NewReader([]byte("invalid json")))
		w := httptest.NewRecorder()

		server.handleWorkspaceSwitch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		server := createTestServer()
		body := `{"path": ""}`
		req := httptest.NewRequest("POST", "/api/workspace/switch", bytes.NewReader([]byte(body)))
		w := httptest.NewRecorder()

		server.handleWorkspaceSwitch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("path does not exist", func(t *testing.T) {
		server := createTestServer()
		body := `{"path": "/nonexistent/path/that/does/not/exist"}`
		req := httptest.NewRequest("POST", "/api/workspace/switch", bytes.NewReader([]byte(body)))
		w := httptest.NewRecorder()

		server.handleWorkspaceSwitch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("switch to current workspace", func(t *testing.T) {
		server := &Server{
			root: "/tmp", // Use /tmp which exists on most systems
		}
		body := `{"path": "/tmp"}`
		req := httptest.NewRequest("POST", "/api/workspace/switch", bytes.NewReader([]byte(body)))
		w := httptest.NewRecorder()

		server.handleWorkspaceSwitch(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if response["message"] != "already on this workspace" {
			t.Errorf("expected 'already on this workspace' message, got %v", response["message"])
		}
	})
}

func TestGetWorkspaceName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"empty path", "", "Unknown"},
		{"simple path", "/home/user/project", "project"},
		{"path with trailing slash", "/home/user/project/", "project/"}, // Trailing slash preserved
		{"single component", "project", "project"},
		{"root path", "/", "/"},   // Root returns itself
		{"windows style", "C:\\Users\\project", "project"},
		{"mixed separators", "/home/user\\project", "project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getWorkspaceName(tt.path)
			if result != tt.expected {
				t.Errorf("getWorkspaceName(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"message": "hello"}

	writeJSON(w, data)

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", w.Header().Get("Content-Type"))
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if response["message"] != "hello" {
		t.Errorf("expected message 'hello', got %q", response["message"])
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		message string
	}{
		{"bad request", http.StatusBadRequest, "invalid input"},
		{"not found", http.StatusNotFound, "resource not found"},
		{"server error", http.StatusInternalServerError, "something went wrong"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			writeError(w, tt.status, tt.message)

			if w.Code != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, w.Code)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got %q", w.Header().Get("Content-Type"))
			}

			var response map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if response["error"] != tt.message {
				t.Errorf("expected error %q, got %q", tt.message, response["error"])
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := corsMiddleware(handler)

	t.Run("adds CORS headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("expected Access-Control-Allow-Origin header")
		}
		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("expected Access-Control-Allow-Methods header")
		}
		if w.Header().Get("Access-Control-Allow-Headers") == "" {
			t.Error("expected Access-Control-Allow-Headers header")
		}
	})

	t.Run("handles OPTIONS preflight", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/test", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d for OPTIONS, got %d", http.StatusOK, w.Code)
		}
	})
}

// Test concurrent access to handlers
func TestHandlersConcurrency(t *testing.T) {
	server := createTestServer()

	// Run multiple requests concurrently
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/api/health", nil)
			w := httptest.NewRecorder()
			server.handleHealth(w, req)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
