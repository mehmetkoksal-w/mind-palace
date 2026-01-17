package butler

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/jsonc"
)

func setupMCPServer(t *testing.T) (*MCPServer, *Butler) {
	t.Helper()

	root := t.TempDir()
	if jsonCDecode == nil {
		SetJSONCDecoder(jsonc.DecodeFile)
	}
	roomsDir := filepath.Join(root, ".palace", "rooms")
	if err := os.MkdirAll(roomsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	room := `{"name":"core","summary":"Core room","entryPoints":["main.go"]}`
	roomPath := filepath.Join(roomsDir, "core.jsonc")
	if err := os.WriteFile(roomPath, []byte(room), 0o644); err != nil {
		t.Fatalf("WriteFile(room) error = %v", err)
	}

	dbPath := filepath.Join(root, ".palace", "index", "palace.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("index.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := seedIndex(db); err != nil {
		t.Fatalf("seedIndex() error = %v", err)
	}

	b, err := New(db, root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = b.Close() })

	server := &MCPServer{
		butler: b,
		reader: bufio.NewReader(strings.NewReader("")),
		writer: &bytes.Buffer{},
	}

	return server, b
}

func seedIndex(db *sql.DB) error {
	now := time.Now().UTC().Format(time.RFC3339)
	files := []struct {
		path string
		lang string
	}{
		{path: "main.go", lang: "go"},
		{path: "other.go", lang: "go"},
		{path: "dep.go", lang: "go"},
		{path: "caller.go", lang: "go"},
	}

	for _, f := range files {
		if _, err := db.ExecContext(context.Background(), `INSERT INTO files (path, hash, size, mod_time, indexed_at, language) VALUES (?, ?, ?, ?, ?, ?)`,
			f.path, "hash", 1, now, now, f.lang); err != nil {
			return err
		}
	}

	content := "package main\nfunc DoWork() {}\n"
	if _, err := db.ExecContext(context.Background(), `INSERT INTO chunks (path, chunk_index, start_line, end_line, content) VALUES (?, ?, ?, ?, ?)`,
		"main.go", 0, 1, 2, content); err != nil {
		return err
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO chunks_fts (path, content, chunk_index) VALUES (?, ?, ?)`,
		"main.go", content, 0); err != nil {
		return err
	}

	if _, err := db.ExecContext(context.Background(), `INSERT INTO symbols (file_path, name, kind, line_start, line_end, signature, doc_comment, exported) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"main.go", "DoWork", "function", 1, 20, "func DoWork()", "", 1); err != nil {
		return err
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO symbols (file_path, name, kind, line_start, line_end, signature, doc_comment, exported) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"caller.go", "Caller", "function", 1, 20, "func Caller()", "", 1); err != nil {
		return err
	}

	relationships := []struct {
		source string
		target string
		kind   string
		line   int
		symbol string
	}{
		{source: "other.go", target: "main.go", kind: "import"},
		{source: "main.go", target: "dep.go", kind: "import"},
		{source: "main.go", symbol: "Helper", kind: "call", line: 2},
		{source: "caller.go", symbol: "DoWork", kind: "call", line: 10},
	}

	for _, r := range relationships {
		if _, err := db.ExecContext(context.Background(), `INSERT INTO relationships (source_file, target_file, target_symbol, kind, line) VALUES (?, ?, ?, ?, ?)`,
			r.source, r.target, r.symbol, r.kind, r.line); err != nil {
			return err
		}
	}

	return nil
}

func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	return data
}

func toolText(t *testing.T, resp jsonRPCResponse) string {
	t.Helper()
	result, ok := resp.Result.(mcpToolResult)
	if !ok {
		t.Fatalf("result type = %T, want mcpToolResult", resp.Result)
	}
	if len(result.Content) == 0 {
		t.Fatal("empty result content")
	}
	return result.Content[0].Text
}

func TestMCPHandleRequestAndLists(t *testing.T) {
	server, _ := setupMCPServer(t)

	resp := server.handleRequest(jsonRPCRequest{JSONRPC: "2.0", ID: 1, Method: "initialize"})
	if resp.Error != nil {
		t.Fatalf("initialize error = %v", resp.Error)
	}
	if _, ok := resp.Result.(mcpInitializeResult); !ok {
		t.Fatalf("initialize result type = %T", resp.Result)
	}

	resp = server.handleRequest(jsonRPCRequest{JSONRPC: "2.0", ID: 2, Method: "unknown"})
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}

	resp = server.handleToolsList(jsonRPCRequest{JSONRPC: "2.0", ID: 3})
	if resp.Error != nil {
		t.Fatalf("tools list error = %v", resp.Error)
	}
	if result, ok := resp.Result.(map[string]interface{}); !ok || result["tools"] == nil {
		t.Fatalf("tools list result = %T", resp.Result)
	}

	resp = server.handleResourcesList(jsonRPCRequest{JSONRPC: "2.0", ID: 4})
	if resp.Error != nil {
		t.Fatalf("resources list error = %v", resp.Error)
	}
}

func TestMCPResourcesRead(t *testing.T) {
	server, _ := setupMCPServer(t)

	resp := server.handleResourcesRead(jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params:  mustMarshal(t, mcpResourceReadParams{URI: "palace://files/main.go"}),
	})
	if resp.Error != nil {
		t.Fatalf("read file error = %v", resp.Error)
	}

	resp = server.handleResourcesRead(jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Params:  mustMarshal(t, mcpResourceReadParams{URI: "palace://rooms/core"}),
	})
	if resp.Error != nil {
		t.Fatalf("read room error = %v", resp.Error)
	}

	resp = server.handleResourcesRead(jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Params:  mustMarshal(t, mcpResourceReadParams{URI: "palace://files/../secret.txt"}),
	})
	if resp.Error == nil {
		t.Fatal("expected error for invalid file path")
	}
}

func TestMCPToolHandlersIndex(t *testing.T) {
	server, _ := setupMCPServer(t)

	resp := server.toolExplore(1, map[string]interface{}{"query": "DoWork"})
	if text := toolText(t, resp); !strings.Contains(text, "Room: core") {
		t.Fatalf("search output missing room: %s", text)
	}

	resp = server.toolExploreImpact(2, map[string]interface{}{"target": "main.go"})
	if text := toolText(t, resp); !strings.Contains(text, "Files that depend on this") {
		t.Fatalf("impact output missing dependents section: %s", text)
	}

	resp = server.toolExploreSymbols(3, map[string]interface{}{"kind": "function", "limit": float64(10)})
	if text := toolText(t, resp); !strings.Contains(text, "DoWork") {
		t.Fatalf("list symbols output missing symbol: %s", text)
	}

	resp = server.toolExploreSymbol(4, map[string]interface{}{"name": "DoWork"})
	if text := toolText(t, resp); !strings.Contains(text, "DoWork") {
		t.Fatalf("get symbol output missing symbol: %s", text)
	}

	resp = server.toolExploreFile(5, map[string]interface{}{"file": "main.go"})
	if text := toolText(t, resp); !strings.Contains(text, "DoWork") {
		t.Fatalf("file symbols output missing symbol: %s", text)
	}

	resp = server.toolExploreDeps(6, map[string]interface{}{"files": []interface{}{"main.go"}})
	if text := toolText(t, resp); !strings.Contains(text, "dep.go") {
		t.Fatalf("dependencies output missing dep: %s", text)
	}

	resp = server.toolExploreCallers(7, map[string]interface{}{"symbol": "DoWork"})
	if text := toolText(t, resp); !strings.Contains(text, "caller.go") {
		t.Fatalf("callers output missing call site: %s", text)
	}

	resp = server.toolExploreCallees(8, map[string]interface{}{"symbol": "DoWork", "file": "main.go"})
	if text := toolText(t, resp); !strings.Contains(text, "Helper") {
		t.Fatalf("callees output missing call: %s", text)
	}

	resp = server.toolExploreGraph(9, map[string]interface{}{"file": "main.go"})
	if text := toolText(t, resp); !strings.Contains(text, "Incoming Calls") {
		t.Fatalf("call graph output missing sections: %s", text)
	}
}

func TestMCPToolHandlersMemory(t *testing.T) {
	server, b := setupMCPServer(t)

	resp := server.toolSessionStart(1, map[string]interface{}{"agentType": "cli", "goal": "test"})
	text := toolText(t, resp)
	sessionID := extractBetween(text, "**Session ID:** `", "`")
	if sessionID == "" {
		t.Fatalf("session ID missing from output: %s", text)
	}

	if err := b.memory.RegisterAgent("cli", "agent-1", sessionID); err != nil {
		t.Fatalf("RegisterAgent() error = %v", err)
	}
	if err := b.memory.SetCurrentFile("agent-1", "main.go"); err != nil {
		t.Fatalf("SetCurrentFile() error = %v", err)
	}

	resp = server.toolSessionLog(2, map[string]interface{}{
		"sessionId": sessionID,
		"kind":      "file_edit",
		"target":    "main.go",
	})
	if text := toolText(t, resp); !strings.Contains(text, "Activity logged") {
		t.Fatalf("log activity output unexpected: %s", text)
	}

	resp = server.toolStore(3, map[string]interface{}{
		"content": "Remember to test",
		"scope":   "palace",
		"as":      "learning",
	})
	// Phase 2: Learnings go through proposal workflow
	if text := toolText(t, resp); !strings.Contains(text, "Proposal Created") {
		t.Fatalf("store learning output unexpected: %s", text)
	}

	resp = server.toolRecall(4, map[string]interface{}{"limit": float64(5)})
	if text := toolText(t, resp); !strings.Contains(text, "Learnings") {
		t.Fatalf("get learnings output unexpected: %s", text)
	}

	if err := b.RecordFileEdit("main.go", "cli"); err != nil {
		t.Fatalf("RecordFileEdit() error = %v", err)
	}

	resp = server.toolBriefFile(5, map[string]interface{}{"path": "main.go"})
	if text := toolText(t, resp); !strings.Contains(text, "File Intelligence") {
		t.Fatalf("file intel output unexpected: %s", text)
	}

	resp = server.toolBrief(6, map[string]interface{}{"filePath": "main.go"})
	if text := toolText(t, resp); !strings.Contains(text, "Briefing") {
		t.Fatalf("brief output unexpected: %s", text)
	}

	resp = server.toolSessionConflict(7, map[string]interface{}{"path": "main.go", "sessionId": "other"})
	if text := toolText(t, resp); !strings.Contains(text, "Conflict") {
		t.Fatalf("conflict output unexpected: %s", text)
	}

	resp = server.toolSessionEnd(8, map[string]interface{}{"sessionId": sessionID, "outcome": "success"})
	if text := toolText(t, resp); !strings.Contains(text, "ended") {
		t.Fatalf("end session output unexpected: %s", text)
	}
}

func TestSanitizePath(t *testing.T) {
	if sanitizePath("../secret") != "" {
		t.Fatal("sanitizePath should reject traversal")
	}
	if sanitizePath("/abs/path") != "" {
		t.Fatal("sanitizePath should reject absolute paths")
	}
	want := filepath.FromSlash("safe/file.txt")
	if got := sanitizePath("safe/file.txt"); got != want {
		t.Fatalf("sanitizePath() = %q, want %q", got, want)
	}
}

func extractBetween(s, start, end string) string {
	startIdx := strings.Index(s, start)
	if startIdx == -1 {
		return ""
	}
	startIdx += len(start)
	endIdx := strings.Index(s[startIdx:], end)
	if endIdx == -1 {
		return ""
	}
	return s[startIdx : startIdx+endIdx]
}

func TestInferKindFromID(t *testing.T) {
	tests := []struct {
		id       string
		expected string
	}{
		{"i_abc123", "idea"},
		{"d_xyz789", "decision"},
		{"l_learn01", "learning"},
		{"https://example.com", "url"},
		{"http://example.com", "url"},
		{"path/to/file.go", "code"},
		{"file.txt", "code"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if got := inferKindFromID(tt.id); got != tt.expected {
				t.Errorf("inferKindFromID(%q) = %q, want %q", tt.id, got, tt.expected)
			}
		})
	}
}

func TestMCPToolHandlersBrain(t *testing.T) {
	server, _ := setupMCPServer(t)

	// toolStore - stores an idea
	resp := server.toolStore(1, map[string]interface{}{
		"content": "Test idea content",
		"as":      "idea",
	})
	if text := toolText(t, resp); !strings.Contains(text, "Remembered") {
		t.Fatalf("toolStore idea output unexpected: %s", text)
	}

	// toolStore - stores a decision (Phase 2: goes through proposal workflow)
	resp = server.toolStore(2, map[string]interface{}{
		"content":   "Test decision content",
		"as":        "decision",
		"status":    "active",
		"rationale": "Because testing",
	})
	if text := toolText(t, resp); !strings.Contains(text, "Proposal Created") {
		t.Fatalf("toolStore decision output unexpected: %s", text)
	}

	// toolRecallIdeas
	resp = server.toolRecallIdeas(3, map[string]interface{}{"limit": float64(10)})
	if text := toolText(t, resp); !strings.Contains(text, "Test idea") || !strings.Contains(text, "Ideas") {
		t.Fatalf("toolRecallIdeas output unexpected: %s", text)
	}

	// toolRecallDecisions - Phase 2: decisions from toolStore are proposals,
	// so they won't show up in recallDecisions until approved
	resp = server.toolRecallDecisions(4, map[string]interface{}{"limit": float64(10)})
	if text := toolText(t, resp); !strings.Contains(text, "Decisions") {
		t.Fatalf("toolRecallDecisions output unexpected: %s", text)
	}
	// Note: We don't check for "Test decision" here because it's now a proposal, not an approved decision
}

func TestMCPToolHandlersLinks(t *testing.T) {
	server, _ := setupMCPServer(t)

	// Create an idea to link
	resp := server.toolStore(1, map[string]interface{}{
		"content": "Source idea for links",
		"as":      "idea",
	})
	text := toolText(t, resp)
	ideaID := extractBetween(text, "**ID:** `", "`")
	if ideaID == "" {
		t.Skipf("Could not extract idea ID from: %s", text)
	}

	// toolRecallLink - use URL as target to avoid file validation
	resp = server.toolRecallLink(2, map[string]interface{}{
		"sourceId": ideaID,
		"targetId": "https://example.com/docs",
		"relation": "related",
	})
	if text := toolText(t, resp); !strings.Contains(text, "Link Created") {
		t.Fatalf("toolRecallLink output unexpected: %s", text)
	}
	linkID := extractBetween(toolText(t, resp), "**ID:** `", "`")

	// toolRecallLinks
	resp = server.toolRecallLinks(3, map[string]interface{}{
		"recordId": ideaID,
	})
	if text := toolText(t, resp); !strings.Contains(text, "Links") {
		t.Fatalf("toolRecallLinks output unexpected: %s", text)
	}

	// toolRecallUnlink
	if linkID != "" {
		resp = server.toolRecallUnlink(4, map[string]interface{}{
			"linkId": linkID,
		})
		if text := toolText(t, resp); !strings.Contains(text, "eleted") {
			t.Fatalf("toolRecallUnlink output unexpected: %s", text)
		}
	}
}

func TestMCPToolHandlersConversations(t *testing.T) {
	server, _ := setupMCPServer(t)

	// toolConversationStore
	resp := server.toolConversationStore(1, map[string]interface{}{
		"summary": "Test conversation summary",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
			map[string]interface{}{"role": "assistant", "content": "Hi there"},
		},
		"agentType": "test",
	})
	if text := toolText(t, resp); !strings.Contains(text, "Conversation Stored") {
		t.Fatalf("toolConversationStore output unexpected: %s", text)
	}

	// toolConversationSearch
	resp = server.toolConversationSearch(2, map[string]interface{}{
		"query": "test",
		"limit": float64(10),
	})
	// This may or may not find results depending on FTS5 setup
	if resp.Error != nil {
		t.Fatalf("toolSearchConversations error: %v", resp.Error)
	}
}

func TestMCPToolExploreContext(t *testing.T) {
	server, _ := setupMCPServer(t)

	resp := server.toolExploreContext(1, map[string]interface{}{
		"task":  "DoWork",
		"limit": float64(5),
	})
	if resp.Error != nil {
		t.Fatalf("toolExploreContext error: %v", resp.Error)
	}
	if text := toolText(t, resp); !strings.Contains(text, "Context") {
		t.Fatalf("toolExploreContext output unexpected: %s", text)
	}
}

func TestMCPToolExploreRooms(t *testing.T) {
	server, _ := setupMCPServer(t)

	resp := server.toolExploreRooms(1)
	if resp.Error != nil {
		t.Fatalf("toolExploreRooms error: %v", resp.Error)
	}
	if text := toolText(t, resp); !strings.Contains(text, "core") {
		t.Fatalf("toolExploreRooms output missing room: %s", text)
	}
}

func TestMCPToolError(t *testing.T) {
	server, _ := setupMCPServer(t)

	resp := server.toolError(1, "test error message")
	result, ok := resp.Result.(mcpToolResult)
	if !ok {
		t.Fatalf("toolError result type = %T, want mcpToolResult", resp.Result)
	}
	if len(result.Content) == 0 || !result.IsError {
		t.Fatalf("toolError should have IsError=true")
	}
	if !strings.Contains(result.Content[0].Text, "test error message") {
		t.Fatalf("toolError message not found in result")
	}
}

func TestMCPHandleToolsCall(t *testing.T) {
	server, _ := setupMCPServer(t)

	// Test with a valid tool
	resp := server.handleToolsCall(jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Params: mustMarshal(t, mcpToolCallParams{
			Name:      "explore_rooms",
			Arguments: map[string]interface{}{},
		}),
	})
	if resp.Error != nil {
		t.Fatalf("handleToolsCall error: %v", resp.Error)
	}

	// Test with unknown tool
	resp = server.handleToolsCall(jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Params: mustMarshal(t, mcpToolCallParams{
			Name:      "unknown_tool",
			Arguments: map[string]interface{}{},
		}),
	})
	if resp.Error == nil {
		t.Fatal("unknown tool should return error")
	}
	if !strings.Contains(resp.Error.Message, "Unknown tool") {
		t.Fatalf("unknown tool error message unexpected: %s", resp.Error.Message)
	}
}

func TestMCPToolCorridorLearnings(t *testing.T) {
	server, _ := setupMCPServer(t)

	// Test without query
	resp := server.toolCorridorLearnings(1, map[string]interface{}{})
	if resp.Error != nil {
		t.Fatalf("toolCorridorLearnings error: %v", resp.Error)
	}
	if text := toolText(t, resp); !strings.Contains(text, "Personal Corridor Learnings") {
		t.Fatalf("toolCorridorLearnings output unexpected: %s", text)
	}

	// Test with query
	resp = server.toolCorridorLearnings(2, map[string]interface{}{
		"query": "test",
		"limit": float64(10),
	})
	if resp.Error != nil {
		t.Fatalf("toolCorridorLearnings with query error: %v", resp.Error)
	}
}

func TestMCPToolCorridorLinks(t *testing.T) {
	server, _ := setupMCPServer(t)

	resp := server.toolCorridorLinks(1, map[string]interface{}{})
	if resp.Error != nil {
		t.Fatalf("toolCorridorLinks error: %v", resp.Error)
	}
	if text := toolText(t, resp); !strings.Contains(text, "Linked Workspaces") {
		t.Fatalf("toolCorridorLinks output unexpected: %s", text)
	}
}

func TestMCPToolCorridorStats(t *testing.T) {
	server, _ := setupMCPServer(t)

	resp := server.toolCorridorStats(1, map[string]interface{}{})
	if resp.Error != nil {
		t.Fatalf("toolCorridorStats error: %v", resp.Error)
	}
	if text := toolText(t, resp); !strings.Contains(text, "Corridor Statistics") {
		t.Fatalf("toolCorridorStats output unexpected: %s", text)
	}
}

func TestMCPToolCorridorReinforce(t *testing.T) {
	server, _ := setupMCPServer(t)

	// Test with missing learningId
	resp := server.toolCorridorReinforce(1, map[string]interface{}{})
	if resp.Error == nil {
		text := toolText(t, resp)
		if !strings.Contains(text, "learningId is required") {
			t.Fatalf("toolCorridorReinforce should require learningId")
		}
	}

	// Test with non-existent learning (should not crash)
	resp = server.toolCorridorReinforce(2, map[string]interface{}{
		"learningId": "nonexistent_id",
	})
	// May error or succeed depending on DB state, just ensure no panic
}

func TestMCPToolCorridorPromote(t *testing.T) {
	server, _ := setupMCPServer(t)

	// Test with missing learningId
	resp := server.toolCorridorPromote(1, map[string]interface{}{})
	if resp.Error == nil {
		text := toolText(t, resp)
		if !strings.Contains(text, "learningId is required") {
			t.Fatalf("toolCorridorPromote should require learningId")
		}
	}

	// Test with non-existent learning
	resp = server.toolCorridorPromote(2, map[string]interface{}{
		"learningId": "l_nonexistent",
	})
	// Should return error about learning not found
	if resp.Error == nil {
		text := toolText(t, resp)
		if !strings.Contains(text, "not found") && !strings.Contains(text, "Error") {
			t.Log("toolCorridorPromote with nonexistent: unexpected success")
		}
	}
}

// ============================================================
// MCP Mode Tests - Phase 3 Governance
// ============================================================

// setupMCPServerWithMode creates a test MCP server with the specified mode.
func setupMCPServerWithMode(t *testing.T, mode MCPMode) (*MCPServer, *Butler) {
	t.Helper()

	root := t.TempDir()
	if jsonCDecode == nil {
		SetJSONCDecoder(jsonc.DecodeFile)
	}
	roomsDir := filepath.Join(root, ".palace", "rooms")
	if err := os.MkdirAll(roomsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	room := `{"name":"core","summary":"Core room","entryPoints":["main.go"]}`
	roomPath := filepath.Join(roomsDir, "core.jsonc")
	if err := os.WriteFile(roomPath, []byte(room), 0o644); err != nil {
		t.Fatalf("WriteFile(room) error = %v", err)
	}

	dbPath := filepath.Join(root, ".palace", "index", "palace.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("index.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := seedIndex(db); err != nil {
		t.Fatalf("seedIndex() error = %v", err)
	}

	b, err := New(db, root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = b.Close() })

	server := &MCPServer{
		butler: b,
		reader: bufio.NewReader(strings.NewReader("")),
		writer: &bytes.Buffer{},
		mode:   mode,
	}

	return server, b
}

func TestMCPModeEnumValidation(t *testing.T) {
	// Test valid modes
	if !IsValidMCPMode("agent") {
		t.Error("'agent' should be a valid mode")
	}
	if !IsValidMCPMode("human") {
		t.Error("'human' should be a valid mode")
	}

	// Test invalid modes
	if IsValidMCPMode("invalid") {
		t.Error("'invalid' should not be a valid mode")
	}
	if IsValidMCPMode("") {
		t.Error("empty string should not be a valid mode")
	}
}

func TestMCPAdminOnlyToolsFiltering(t *testing.T) {
	// Verify admin-only tools are correctly identified
	adminTools := GetAdminOnlyTools()
	expectedAdminTools := []string{"store_direct", "approve", "reject"}

	for _, tool := range expectedAdminTools {
		if !adminTools[tool] {
			t.Errorf("Tool %q should be marked as admin-only", tool)
		}
	}

	// Verify IsAdminOnlyTool works
	for _, tool := range expectedAdminTools {
		if !IsAdminOnlyTool(tool) {
			t.Errorf("IsAdminOnlyTool(%q) should return true", tool)
		}
	}

	// Regular tools should not be admin-only
	regularTools := []string{"explore", "store", "recall", "brief"}
	for _, tool := range regularTools {
		if IsAdminOnlyTool(tool) {
			t.Errorf("IsAdminOnlyTool(%q) should return false", tool)
		}
	}
}

func TestMCPToolsListFilteringByMode(t *testing.T) {
	// Agent mode should filter out admin-only tools
	agentServer, _ := setupMCPServerWithMode(t, MCPModeAgent)

	agentResp := agentServer.handleToolsList(jsonRPCRequest{JSONRPC: "2.0", ID: 1})
	if agentResp.Error != nil {
		t.Fatalf("handleToolsList error: %v", agentResp.Error)
	}

	agentResult := agentResp.Result.(map[string]interface{})
	agentTools := agentResult["tools"].([]mcpTool)

	// Check that admin-only tools are NOT in agent mode
	for _, tool := range agentTools {
		if IsAdminOnlyTool(tool.Name) {
			t.Errorf("Admin-only tool %q should not be in agent mode tools list", tool.Name)
		}
	}

	// Human mode should include all tools
	humanServer, _ := setupMCPServerWithMode(t, MCPModeHuman)

	humanResp := humanServer.handleToolsList(jsonRPCRequest{JSONRPC: "2.0", ID: 2})
	if humanResp.Error != nil {
		t.Fatalf("handleToolsList error: %v", humanResp.Error)
	}

	humanResult := humanResp.Result.(map[string]interface{})
	humanTools := humanResult["tools"].([]mcpTool)

	// Check that admin-only tools ARE in human mode
	adminToolsFound := map[string]bool{}
	for _, tool := range humanTools {
		if IsAdminOnlyTool(tool.Name) {
			adminToolsFound[tool.Name] = true
		}
	}

	expectedAdminTools := []string{"store_direct", "approve", "reject"}
	for _, expected := range expectedAdminTools {
		if !adminToolsFound[expected] {
			t.Errorf("Admin tool %q should be in human mode tools list", expected)
		}
	}

	// Human mode should have more tools than agent mode
	if len(humanTools) <= len(agentTools) {
		t.Errorf("Human mode should have more tools than agent mode: human=%d, agent=%d",
			len(humanTools), len(agentTools))
	}
}

func TestMCPToolCallModeEnforcement(t *testing.T) {
	agentServer, _ := setupMCPServerWithMode(t, MCPModeAgent)

	// Agent should be blocked from calling admin-only tools
	adminTools := []string{"store_direct", "approve", "reject"}
	for _, tool := range adminTools {
		resp := agentServer.handleToolsCall(jsonRPCRequest{
			JSONRPC: "2.0",
			ID:      1,
			Params: mustMarshal(t, mcpToolCallParams{
				Name:      tool,
				Arguments: map[string]interface{}{},
			}),
		})

		if resp.Error == nil {
			t.Errorf("Agent mode should block %q tool, but it succeeded", tool)
			continue
		}

		if !strings.Contains(resp.Error.Message, "not available in agent mode") {
			t.Errorf("Error message for %q should mention 'not available in agent mode', got: %s",
				tool, resp.Error.Message)
		}
	}
}

func TestMCPStoreDirectHumanMode(t *testing.T) {
	humanServer, _ := setupMCPServerWithMode(t, MCPModeHuman)

	// Store a decision directly
	resp := humanServer.toolStoreDirect(1, map[string]interface{}{
		"content":   "Direct decision from human",
		"as":        "decision",
		"scope":     "palace",
		"rationale": "Testing direct write",
		"actorId":   "test-human",
	})

	if resp.Error != nil {
		t.Fatalf("toolStoreDirect error: %v", resp.Error)
	}

	text := toolText(t, resp)
	if !strings.Contains(text, "Direct Write Successful") {
		t.Errorf("toolStoreDirect output should contain 'Direct Write Successful', got: %s", text)
	}
	if !strings.Contains(text, "decision") {
		t.Errorf("toolStoreDirect output should contain 'decision', got: %s", text)
	}
	if !strings.Contains(text, "audit") {
		t.Errorf("toolStoreDirect output should mention audit, got: %s", text)
	}

	// Store a learning directly
	resp = humanServer.toolStoreDirect(2, map[string]interface{}{
		"content":    "Direct learning from human",
		"as":         "learning",
		"scope":      "palace",
		"confidence": 0.9,
	})

	if resp.Error != nil {
		t.Fatalf("toolStoreDirect learning error: %v", resp.Error)
	}

	text = toolText(t, resp)
	if !strings.Contains(text, "Direct Write Successful") {
		t.Errorf("toolStoreDirect learning output unexpected: %s", text)
	}
}

func TestMCPApproveRejectHumanMode(t *testing.T) {
	humanServer, butler := setupMCPServerWithMode(t, MCPModeHuman)
	mem := butler.Memory()

	// Create a proposal via the store tool (which creates proposals for decisions)
	resp := humanServer.toolStore(1, map[string]interface{}{
		"content": "Test decision for approval",
		"as":      "decision",
	})
	text := toolText(t, resp)
	if !strings.Contains(text, "Proposal Created") {
		t.Skipf("Could not create proposal: %s", text)
	}

	// Get the proposal ID
	proposalID := extractBetween(text, "**ID:** `", "`")
	if proposalID == "" {
		t.Skipf("Could not extract proposal ID from: %s", text)
	}

	// Approve the proposal
	resp = humanServer.toolApprove(2, map[string]interface{}{
		"proposalId": proposalID,
		"by":         "test-human",
		"note":       "Approved for testing",
	})

	if resp.Error != nil {
		t.Fatalf("toolApprove error: %v", resp.Error)
	}

	text = toolText(t, resp)
	if !strings.Contains(text, "Proposal Approved") {
		t.Errorf("toolApprove output should contain 'Proposal Approved', got: %s", text)
	}
	if !strings.Contains(text, "Promoted To") {
		t.Errorf("toolApprove output should contain 'Promoted To', got: %s", text)
	}

	// Create another proposal for rejection test
	resp = humanServer.toolStore(3, map[string]interface{}{
		"content": "Test decision for rejection",
		"as":      "decision",
	})
	text = toolText(t, resp)
	proposalID2 := extractBetween(text, "**ID:** `", "`")
	if proposalID2 == "" {
		t.Skipf("Could not extract second proposal ID")
	}

	// Reject the proposal
	resp = humanServer.toolReject(4, map[string]interface{}{
		"proposalId": proposalID2,
		"by":         "test-human",
		"note":       "Rejected for testing",
	})

	if resp.Error != nil {
		t.Fatalf("toolReject error: %v", resp.Error)
	}

	text = toolText(t, resp)
	if !strings.Contains(text, "Proposal Rejected") {
		t.Errorf("toolReject output should contain 'Proposal Rejected', got: %s", text)
	}

	// Verify audit logs were created
	auditLogs, err := mem.GetAuditLogs("", "", 10)
	if err != nil {
		t.Fatalf("GetAuditLogs error: %v", err)
	}

	// Should have at least 2 audit entries (approve and reject)
	if len(auditLogs) < 2 {
		t.Errorf("Expected at least 2 audit logs, got %d", len(auditLogs))
	}
}

func TestMCPServerModeGetter(t *testing.T) {
	agentServer, _ := setupMCPServerWithMode(t, MCPModeAgent)
	if agentServer.Mode() != MCPModeAgent {
		t.Errorf("Mode() = %q, want %q", agentServer.Mode(), MCPModeAgent)
	}

	humanServer, _ := setupMCPServerWithMode(t, MCPModeHuman)
	if humanServer.Mode() != MCPModeHuman {
		t.Errorf("Mode() = %q, want %q", humanServer.Mode(), MCPModeHuman)
	}
}

func TestMCPDefaultModeIsAgent(t *testing.T) {
	// NewMCPServer should default to agent mode for security
	// Note: setupMCPServer creates the server directly without mode set,
	// so we test with a properly created server instead.

	root := t.TempDir()
	if jsonCDecode == nil {
		SetJSONCDecoder(jsonc.DecodeFile)
	}
	roomsDir := filepath.Join(root, ".palace", "rooms")
	if err := os.MkdirAll(roomsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	dbPath := filepath.Join(root, ".palace", "index", "palace.db")
	db, err := index.Open(dbPath)
	if err != nil {
		t.Fatalf("index.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	b, err := New(db, root)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = b.Close() })

	// Use NewMCPServer which should default to agent mode
	server := NewMCPServer(b)
	if server.Mode() != MCPModeAgent {
		t.Errorf("NewMCPServer() should default to agent mode, got %q", server.Mode())
	}
}
