package dashboard

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/butler"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/corridor"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

func decodeJSON(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return result
}

func TestHandleHealth(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)

	s.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := decodeJSON(t, rec.Body)
	if body["status"] != "ok" {
		t.Errorf("status = %v, want ok", body["status"])
	}
}

func TestHandleRoomsMethodNotAllowed(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/rooms", nil)

	s.handleRooms(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleSessionsNoMemory(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)

	s.handleSessions(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleWorkspaces(t *testing.T) {
	root := t.TempDir()
	s := New(Config{Root: root})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/workspaces", nil)

	s.handleWorkspaces(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := decodeJSON(t, rec.Body)
	if body["current"] != root {
		t.Errorf("current = %v, want %s", body["current"], root)
	}
	workspaces, ok := body["workspaces"].([]any)
	if !ok || len(workspaces) != 1 {
		t.Fatalf("workspaces length = %v, want 1", body["workspaces"])
	}
}

func TestHandleWorkspaceSwitch(t *testing.T) {
	root := t.TempDir()
	s := New(Config{Root: root})

	t.Run("method not allowed", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/workspace/switch", nil)

		s.handleWorkspaceSwitch(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/workspace/switch", bytes.NewBufferString("{bad-json"))

		s.handleWorkspaceSwitch(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("missing path", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/workspace/switch", bytes.NewBufferString(`{"path":""}`))

		s.handleWorkspaceSwitch(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("path does not exist", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/workspace/switch", bytes.NewBufferString(`{"path":"/nope/does-not-exist"}`))

		s.handleWorkspaceSwitch(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("already on workspace", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/workspace/switch", bytes.NewBufferString(`{"path":"`+root+`"}`))

		s.handleWorkspaceSwitch(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		body := decodeJSON(t, rec.Body)
		if success, ok := body["success"].(bool); !ok || !success {
			t.Fatalf("success = %v, want true", body["success"])
		}
	})
}

func TestGetWorkspaceName(t *testing.T) {
	if name := getWorkspaceName(""); name != "Unknown" {
		t.Errorf("getWorkspaceName(\"\") = %q, want %q", name, "Unknown")
	}
	if name := getWorkspaceName("/tmp/project"); name != "project" {
		t.Errorf("getWorkspaceName() = %q, want %q", name, "project")
	}
}

func TestHandleSearchMissingQuery(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/search", nil)

	s.handleSearch(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleGraphMissingSymbol(t *testing.T) {
	s := New(Config{Root: t.TempDir(), Butler: &butler.Butler{}})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/graph/", nil)

	s.handleGraph(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleFileIntelMissingPath(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/file-intel", nil)

	s.handleFileIntel(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleStatsNoResources(t *testing.T) {
	s := New(Config{Root: t.TempDir()})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)

	s.handleStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := decodeJSON(t, rec.Body)
	if body["workspace"] == nil {
		t.Fatalf("workspace info missing in stats response")
	}
}

func TestHandleSessionsDetailAndActivity(t *testing.T) {
	root := t.TempDir()
	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	sess, err := mem.StartSession("agent", "id-1", "goal")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	if err := mem.LogActivity(sess.ID, memory.Activity{Kind: "file_edit", Target: "main.go", Outcome: "success"}); err != nil {
		t.Fatalf("LogActivity() error = %v", err)
	}

	s := New(Config{Root: root, Memory: mem})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions?active=true", nil)
	s.handleSessions(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/sessions/"+sess.ID, nil)
	s.handleSessionDetail(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/activity?sessionId="+sess.ID, nil)
	s.handleActivity(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleLearningsAndFileIntel(t *testing.T) {
	root := t.TempDir()
	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	lrnID, err := mem.AddLearning(memory.Learning{Scope: "file", ScopePath: "src/main.go", Content: "note"})
	if err != nil {
		t.Fatalf("AddLearning() error = %v", err)
	}
	if err := mem.AssociateLearningWithFile("src/main.go", lrnID); err != nil {
		t.Fatalf("AssociateLearningWithFile() error = %v", err)
	}
	if err := mem.RecordFileEdit("src/main.go", "agent"); err != nil {
		t.Fatalf("RecordFileEdit() error = %v", err)
	}
	if err := mem.RecordFileFailure("src/main.go"); err != nil {
		t.Fatalf("RecordFileFailure() error = %v", err)
	}

	s := New(Config{Root: root, Memory: mem})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/learnings?scope=file&scopePath=src/main.go", nil)
	s.handleLearnings(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/file-intel?path=src/main.go", nil)
	s.handleFileIntel(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleCorridorEndpoints(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	gc, err := corridor.OpenGlobal()
	if err != nil {
		t.Fatalf("OpenGlobal() error = %v", err)
	}
	t.Cleanup(func() { _ = gc.Close() })

	if err := gc.AddPersonalLearning(corridor.PersonalLearning{Content: "corridor note"}); err != nil {
		t.Fatalf("AddPersonalLearning() error = %v", err)
	}

	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, ".palace"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := gc.Link("ws", workspace); err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	s := New(Config{Root: t.TempDir(), Corridor: gc})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/corridors", nil)
	s.handleCorridors(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/corridors/personal", nil)
	s.handleCorridorPersonal(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/corridors/links", nil)
	s.handleCorridorLinks(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleHotspotsAgentsAndBrief(t *testing.T) {
	root := t.TempDir()
	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	sess, err := mem.StartSession("agent", "id-1", "goal")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	if err := mem.RegisterAgent("agent", "id-1", sess.ID); err != nil {
		t.Fatalf("RegisterAgent() error = %v", err)
	}
	if err := mem.RecordFileEdit("hot.go", "agent"); err != nil {
		t.Fatalf("RecordFileEdit() error = %v", err)
	}
	if err := mem.RecordFileFailure("hot.go"); err != nil {
		t.Fatalf("RecordFileFailure() error = %v", err)
	}
	if err := mem.LogActivity(sess.ID, memory.Activity{Kind: "file_edit", Target: "hot.go", Outcome: "success"}); err != nil {
		t.Fatalf("LogActivity() error = %v", err)
	}

	s := New(Config{Root: root, Memory: mem})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/hotspots?limit=5", nil)
	s.handleHotspots(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	s.handleAgents(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/brief?path=hot.go", nil)
	s.handleBrief(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleRoomsWithButler(t *testing.T) {
	root := t.TempDir()
	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	// Create a palace.json to make butler work
	palaceJSON := filepath.Join(root, "palace.json")
	if err := os.WriteFile(palaceJSON, []byte(`{"name":"test","rooms":[]}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	b, err := butler.New(mem.DB(), root)
	if err != nil {
		t.Fatalf("butler.New() error = %v", err)
	}
	t.Cleanup(func() { _ = b.Close() })

	s := New(Config{Root: root, Butler: b})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/rooms", nil)
	s.handleRooms(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := decodeJSON(t, rec.Body)
	if _, ok := body["rooms"]; !ok {
		t.Error("rooms field missing from response")
	}
}

func TestHandleRoomsNoButler(t *testing.T) {
	s := New(Config{Root: t.TempDir()})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/rooms", nil)
	s.handleRooms(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleSearchWithQuery(t *testing.T) {
	root := t.TempDir()
	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	// Add some learnings to search
	_, err = mem.AddLearning(memory.Learning{Content: "test learning content", Scope: "palace"})
	if err != nil {
		t.Fatalf("AddLearning() error = %v", err)
	}

	s := New(Config{Root: root, Memory: mem})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/search?q=test&limit=10", nil)
	s.handleSearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := decodeJSON(t, rec.Body)
	if _, ok := body["symbols"]; !ok {
		t.Error("symbols field missing from response")
	}
	if _, ok := body["learnings"]; !ok {
		t.Error("learnings field missing from response")
	}
}

func TestHandleSearchMethodNotAllowed(t *testing.T) {
	s := New(Config{Root: t.TempDir()})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/search?q=test", nil)
	s.handleSearch(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleGraphWithSymbol(t *testing.T) {
	root := t.TempDir()
	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	// Create a palace.json
	palaceJSON := filepath.Join(root, "palace.json")
	if err := os.WriteFile(palaceJSON, []byte(`{"name":"test","rooms":[]}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	b, err := butler.New(mem.DB(), root)
	if err != nil {
		t.Fatalf("butler.New() error = %v", err)
	}
	t.Cleanup(func() { _ = b.Close() })

	s := New(Config{Root: root, Butler: b})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/graph/testSymbol?file=test.go", nil)
	s.handleGraph(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := decodeJSON(t, rec.Body)
	if body["symbol"] != "testSymbol" {
		t.Errorf("symbol = %v, want testSymbol", body["symbol"])
	}
}

func TestHandleGraphMethodNotAllowed(t *testing.T) {
	s := New(Config{Root: t.TempDir(), Butler: &butler.Butler{}})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/graph/symbol", nil)
	s.handleGraph(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleGraphNoButler(t *testing.T) {
	s := New(Config{Root: t.TempDir()})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/graph/symbol", nil)
	s.handleGraph(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleStatsComprehensive(t *testing.T) {
	root := t.TempDir()
	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	// Add some data for stats
	_, _ = mem.StartSession("agent", "id-1", "goal")
	_, _ = mem.AddLearning(memory.Learning{Content: "test", Scope: "palace"})
	_, _ = mem.AddIdea(memory.Idea{Content: "test idea"})
	_, _ = mem.AddDecision(memory.Decision{Content: "test decision"})

	// Create a palace.json
	palaceJSON := filepath.Join(root, "palace.json")
	if err := os.WriteFile(palaceJSON, []byte(`{"name":"test","rooms":[]}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	b, err := butler.New(mem.DB(), root)
	if err != nil {
		t.Fatalf("butler.New() error = %v", err)
	}
	t.Cleanup(func() { _ = b.Close() })

	s := New(Config{Root: root, Memory: mem, Butler: b})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	s.handleStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := decodeJSON(t, rec.Body)
	if body["workspace"] == nil {
		t.Error("workspace field missing from response")
	}
}

func TestHandleHotspotsMethodNotAllowed(t *testing.T) {
	s := New(Config{Root: t.TempDir()})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/hotspots", nil)
	s.handleHotspots(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleHotspotsNoMemory(t *testing.T) {
	s := New(Config{Root: t.TempDir()})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/hotspots", nil)
	s.handleHotspots(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleAgentsNoMemory(t *testing.T) {
	s := New(Config{Root: t.TempDir()})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	s.handleAgents(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleBriefNoPath(t *testing.T) {
	root := t.TempDir()
	mem, _ := memory.Open(root)
	t.Cleanup(func() { _ = mem.Close() })

	s := New(Config{Root: root, Memory: mem})

	// Path is optional in handleBrief, so this should succeed
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/brief", nil)
	s.handleBrief(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleBriefNoMemory(t *testing.T) {
	s := New(Config{Root: t.TempDir()})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/brief", nil)
	s.handleBrief(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleSessionDetailNotFound(t *testing.T) {
	root := t.TempDir()
	mem, _ := memory.Open(root)
	t.Cleanup(func() { _ = mem.Close() })

	s := New(Config{Root: root, Memory: mem})

	// GetSession returns error for non-existent session, so handler returns 500
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/nonexistent", nil)
	s.handleSessionDetail(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandleSessionDetailNoMemory(t *testing.T) {
	s := New(Config{Root: t.TempDir()})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/test", nil)
	s.handleSessionDetail(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleSessionDetailMissingID(t *testing.T) {
	root := t.TempDir()
	mem, _ := memory.Open(root)
	t.Cleanup(func() { _ = mem.Close() })

	s := New(Config{Root: root, Memory: mem})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/", nil)
	s.handleSessionDetail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleActivityNoMemory(t *testing.T) {
	s := New(Config{Root: t.TempDir()})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/activity", nil)
	s.handleActivity(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleLearningsNoMemory(t *testing.T) {
	s := New(Config{Root: t.TempDir()})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/learnings", nil)
	s.handleLearnings(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleCorridorsNoCorr(t *testing.T) {
	s := New(Config{Root: t.TempDir()})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/corridors", nil)
	s.handleCorridors(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleCorridorPersonalNoCorr(t *testing.T) {
	s := New(Config{Root: t.TempDir()})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/corridors/personal", nil)
	s.handleCorridorPersonal(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleCorridorLinksNoCorr(t *testing.T) {
	s := New(Config{Root: t.TempDir()})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/corridors/links", nil)
	s.handleCorridorLinks(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleIdeas(t *testing.T) {
	root := t.TempDir()
	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	_, _ = mem.AddIdea(memory.Idea{Content: "test idea", Status: "active"})

	s := New(Config{Root: root, Memory: mem})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/ideas?status=active&limit=10", nil)
	s.handleIdeas(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleDecisions(t *testing.T) {
	root := t.TempDir()
	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	_, _ = mem.AddDecision(memory.Decision{Content: "test decision", Status: "active"})

	s := New(Config{Root: root, Memory: mem})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/decisions?status=active&limit=10", nil)
	s.handleDecisions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleLinks(t *testing.T) {
	root := t.TempDir()
	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	id, _ := mem.AddIdea(memory.Idea{Content: "idea 1"})
	_, _ = mem.AddLink(memory.Link{SourceID: id, SourceKind: "idea", TargetID: "file.go", TargetKind: "file", Relation: "related"})

	s := New(Config{Root: root, Memory: mem})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/links?limit=10", nil)
	s.handleLinks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleConversations(t *testing.T) {
	root := t.TempDir()
	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	_, _ = mem.AddConversation(memory.Conversation{
		AgentType: "claude-code",
		Summary:   "test conversation",
	})

	s := New(Config{Root: root, Memory: mem})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/conversations?limit=10", nil)
	s.handleConversations(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandleRemember(t *testing.T) {
	root := t.TempDir()
	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	s := New(Config{Root: root, Memory: mem})

	t.Run("method not allowed", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/remember", nil)
		s.handleRemember(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("no memory", func(t *testing.T) {
		s2 := New(Config{Root: t.TempDir()})
		rec := httptest.NewRecorder()
		body := bytes.NewBufferString(`{"content": "test"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/remember", body)
		s2.handleRemember(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := bytes.NewBufferString(`not json`)
		req := httptest.NewRequest(http.MethodPost, "/api/remember", body)
		s.handleRemember(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("missing content", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := bytes.NewBufferString(`{"kind": "idea"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/remember", body)
		s.handleRemember(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("remember idea", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := bytes.NewBufferString(`{"content": "This is an idea", "kind": "idea", "scope": "palace"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/remember", body)
		s.handleRemember(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		result := decodeJSON(t, rec.Body)
		if result["kind"] != "idea" {
			t.Errorf("kind = %v, want idea", result["kind"])
		}
	})

	t.Run("remember decision", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := bytes.NewBufferString(`{"content": "We decided to use Go", "kind": "decision"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/remember", body)
		s.handleRemember(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		result := decodeJSON(t, rec.Body)
		if result["kind"] != "decision" {
			t.Errorf("kind = %v, want decision", result["kind"])
		}
	})

	t.Run("remember learning", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := bytes.NewBufferString(`{"content": "Learned something important", "kind": "learning"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/remember", body)
		s.handleRemember(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		result := decodeJSON(t, rec.Body)
		if result["kind"] != "learning" {
			t.Errorf("kind = %v, want learning", result["kind"])
		}
	})

	t.Run("auto-classify", func(t *testing.T) {
		rec := httptest.NewRecorder()
		body := bytes.NewBufferString(`{"content": "Maybe we should consider refactoring"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/remember", body)
		s.handleRemember(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

func TestHandleBrainSearch(t *testing.T) {
	root := t.TempDir()
	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	// Add some test data
	_, _ = mem.AddIdea(memory.Idea{Content: "test idea about auth", Status: "active"})
	_, _ = mem.AddDecision(memory.Decision{Content: "decided to use JWT for auth", Status: "active"})
	_, _ = mem.AddLearning(memory.Learning{Content: "auth should be stateless"})

	s := New(Config{Root: root, Memory: mem})

	t.Run("method not allowed", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/brain/search", nil)
		s.handleBrainSearch(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("no memory", func(t *testing.T) {
		s2 := New(Config{Root: t.TempDir()})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/brain/search?q=test", nil)
		s2.handleBrainSearch(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("search all", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/brain/search?q=auth", nil)
		s.handleBrainSearch(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("search ideas only", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/brain/search?q=auth&kind=idea", nil)
		s.handleBrainSearch(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("search decisions only", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/brain/search?q=auth&kind=decision", nil)
		s.handleBrainSearch(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("search learnings only", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/brain/search?q=auth&kind=learning", nil)
		s.handleBrainSearch(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("list without query", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/brain/search?status=active", nil)
		s.handleBrainSearch(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

func TestHandleBrainContext(t *testing.T) {
	root := t.TempDir()

	t.Run("method not allowed", func(t *testing.T) {
		s := New(Config{Root: root})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/brain/context", nil)
		s.handleBrainContext(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("no butler", func(t *testing.T) {
		s := New(Config{Root: root})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/brain/context?query=test", nil)
		s.handleBrainContext(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("missing query", func(t *testing.T) {
		butlerRoot := t.TempDir()
		mem, err := memory.Open(butlerRoot)
		if err != nil {
			t.Fatalf("memory.Open() error = %v", err)
		}
		t.Cleanup(func() { _ = mem.Close() })

		// Create a palace.json to make butler work
		palaceJSON := filepath.Join(butlerRoot, "palace.json")
		if err := os.WriteFile(palaceJSON, []byte(`{"name":"test","rooms":[]}`), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		b, err := butler.New(mem.DB(), butlerRoot)
		if err != nil {
			t.Fatalf("butler.New() error = %v", err)
		}
		t.Cleanup(func() { _ = b.Close() })

		s := New(Config{Root: butlerRoot, Butler: b})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/brain/context", nil)
		s.handleBrainContext(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("handles context error from butler", func(t *testing.T) {
		butlerRoot := t.TempDir()
		mem, err := memory.Open(butlerRoot)
		if err != nil {
			t.Fatalf("memory.Open() error = %v", err)
		}
		t.Cleanup(func() { _ = mem.Close() })

		// Create a palace.json to make butler work
		palaceJSON := filepath.Join(butlerRoot, "palace.json")
		if err := os.WriteFile(palaceJSON, []byte(`{"name":"test","rooms":[]}`), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		b, err := butler.New(mem.DB(), butlerRoot)
		if err != nil {
			t.Fatalf("butler.New() error = %v", err)
		}
		t.Cleanup(func() { _ = b.Close() })

		// Without a proper index, GetEnhancedContext returns an error
		// This tests the error handling path
		s := New(Config{Root: butlerRoot, Butler: b, Memory: mem})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/brain/context?query=test", nil)
		s.handleBrainContext(rec, req)

		// We expect 500 since the index tables aren't set up
		// This still tests the error handling code path
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("parses boolean flags", func(t *testing.T) {
		butlerRoot := t.TempDir()
		mem, err := memory.Open(butlerRoot)
		if err != nil {
			t.Fatalf("memory.Open() error = %v", err)
		}
		t.Cleanup(func() { _ = mem.Close() })

		// Create a palace.json to make butler work
		palaceJSON := filepath.Join(butlerRoot, "palace.json")
		if err := os.WriteFile(palaceJSON, []byte(`{"name":"test","rooms":[]}`), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		b, err := butler.New(mem.DB(), butlerRoot)
		if err != nil {
			t.Fatalf("butler.New() error = %v", err)
		}
		t.Cleanup(func() { _ = b.Close() })

		// Test with boolean flags - will still error but exercises the flag parsing code
		s := New(Config{Root: butlerRoot, Butler: b, Memory: mem})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/brain/context?query=test&includeIdeas=false&includeDecisions=false&includeLearnings=false", nil)
		s.handleBrainContext(rec, req)

		// We expect 500 since the index tables aren't set up
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}
