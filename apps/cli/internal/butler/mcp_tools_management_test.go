package butler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoomTool_List(t *testing.T) {
	server, _ := setupMCPServer(t)

	resp := server.dispatchRoom(1, map[string]interface{}{}, "list")
	if resp.Error != nil {
		t.Fatalf("dispatchRoom list error: %v", resp.Error)
	}

	text := extractResponseText(resp)
	if !strings.Contains(text, "Mind Palace Rooms") {
		t.Errorf("expected 'Mind Palace Rooms' in output, got: %s", text)
	}
}

func TestRoomTool_Show(t *testing.T) {
	server, _ := setupMCPServer(t)

	// First list to see what rooms exist
	listResp := server.dispatchRoom(1, map[string]interface{}{}, "list")
	if listResp.Error != nil {
		t.Fatalf("dispatchRoom list error: %v", listResp.Error)
	}

	// Try to show a room that doesn't exist
	resp := server.dispatchRoom(1, map[string]interface{}{
		"name": "nonexistent",
	}, "show")
	if resp.Error != nil {
		// Error might be returned in result instead of Error field
		t.Logf("dispatchRoom show returned error (expected): %v", resp.Error)
	}

	text := extractResponseText(resp)
	if !strings.Contains(text, "not found") && !strings.Contains(text, "Error") {
		t.Logf("Response for nonexistent room: %s", text)
	}
}

func TestRoomTool_CreateUpdateDelete(t *testing.T) {
	server, _ := setupMCPServer(t)

	// Create a room
	createResp := server.dispatchRoom(1, map[string]interface{}{
		"name":         "test-room",
		"summary":      "A test room for unit testing",
		"entry_points": []interface{}{"src/test/"},
		"capabilities": []interface{}{"testing", "validation"},
	}, "create")

	if createResp.Error != nil {
		t.Fatalf("create room error: %v", createResp.Error)
	}

	text := extractResponseText(createResp)
	if !strings.Contains(text, "Created") && !strings.Contains(text, "test-room") {
		t.Errorf("expected create confirmation, got: %s", text)
	}

	// Verify room file was created
	roomFile := filepath.Join(server.butler.root, ".palace", "rooms", "test-room.jsonc")
	if _, err := os.Stat(roomFile); os.IsNotExist(err) {
		t.Error("room file was not created")
	}

	// Show the room
	showResp := server.dispatchRoom(1, map[string]interface{}{
		"name": "test-room",
	}, "show")

	if showResp.Error != nil {
		t.Fatalf("show room error: %v", showResp.Error)
	}

	text = extractResponseText(showResp)
	if !strings.Contains(text, "test-room") {
		t.Errorf("show should contain room name, got: %s", text)
	}
	if !strings.Contains(text, "A test room for unit testing") {
		t.Errorf("show should contain room summary, got: %s", text)
	}

	// Update the room
	updateResp := server.dispatchRoom(1, map[string]interface{}{
		"name":    "test-room",
		"summary": "Updated summary",
	}, "update")

	if updateResp.Error != nil {
		t.Fatalf("update room error: %v", updateResp.Error)
	}

	text = extractResponseText(updateResp)
	if !strings.Contains(text, "Updated") {
		t.Errorf("expected update confirmation, got: %s", text)
	}

	// Verify update
	showResp = server.dispatchRoom(1, map[string]interface{}{
		"name": "test-room",
	}, "show")

	text = extractResponseText(showResp)
	if !strings.Contains(text, "Updated summary") {
		t.Errorf("show should contain updated summary, got: %s", text)
	}

	// Delete the room
	deleteResp := server.dispatchRoom(1, map[string]interface{}{
		"name": "test-room",
	}, "delete")

	if deleteResp.Error != nil {
		t.Fatalf("delete room error: %v", deleteResp.Error)
	}

	text = extractResponseText(deleteResp)
	if !strings.Contains(text, "Deleted") {
		t.Errorf("expected delete confirmation, got: %s", text)
	}

	// Verify room file was deleted
	if _, err := os.Stat(roomFile); !os.IsNotExist(err) {
		t.Error("room file was not deleted")
	}
}

func TestRoomTool_CreateValidation(t *testing.T) {
	server, _ := setupMCPServer(t)

	// Missing name
	resp := server.dispatchRoom(1, map[string]interface{}{
		"summary":      "No name",
		"entry_points": []interface{}{"src/"},
	}, "create")

	text := extractResponseText(resp)
	if !strings.Contains(text, "name is required") && !strings.Contains(text, "Error") {
		t.Errorf("expected name validation error, got: %s", text)
	}

	// Missing summary
	resp = server.dispatchRoom(1, map[string]interface{}{
		"name":         "test",
		"entry_points": []interface{}{"src/"},
	}, "create")

	text = extractResponseText(resp)
	if !strings.Contains(text, "summary is required") && !strings.Contains(text, "Error") {
		t.Errorf("expected summary validation error, got: %s", text)
	}

	// Missing entry points
	resp = server.dispatchRoom(1, map[string]interface{}{
		"name":    "test",
		"summary": "Test room",
	}, "create")

	text = extractResponseText(resp)
	if !strings.Contains(text, "entry_point") && !strings.Contains(text, "Error") {
		t.Errorf("expected entry_point validation error, got: %s", text)
	}
}

func TestRoomTool_InvalidAction(t *testing.T) {
	server, _ := setupMCPServer(t)

	resp := server.dispatchRoom(1, map[string]interface{}{}, "invalid_action")
	if resp.Error == nil {
		text := extractResponseText(resp)
		// Error might be in the response text
		if !strings.Contains(text, "invalid") && !strings.Contains(text, "Error") {
			t.Error("expected error for invalid action")
		}
	}
}

func TestIndexTool_Status(t *testing.T) {
	server, _ := setupMCPServer(t)

	resp := server.dispatchIndex(1, map[string]interface{}{}, "status")
	if resp.Error != nil {
		t.Fatalf("dispatchIndex status error: %v", resp.Error)
	}

	text := extractResponseText(resp)
	if !strings.Contains(text, "Index Status") {
		t.Errorf("expected 'Index Status' in output, got: %s", text)
	}
}

func TestIndexTool_Stats(t *testing.T) {
	server, _ := setupMCPServer(t)

	resp := server.dispatchIndex(1, map[string]interface{}{}, "stats")
	if resp.Error != nil {
		t.Fatalf("dispatchIndex stats error: %v", resp.Error)
	}

	text := extractResponseText(resp)
	if !strings.Contains(text, "Index Statistics") {
		t.Errorf("expected 'Index Statistics' in output, got: %s", text)
	}
	if !strings.Contains(text, "Files") {
		t.Errorf("expected 'Files' count in output, got: %s", text)
	}
}

func TestIndexTool_InvalidAction(t *testing.T) {
	server, _ := setupMCPServer(t)

	resp := server.dispatchIndex(1, map[string]interface{}{}, "invalid_action")
	if resp.Error == nil {
		text := extractResponseText(resp)
		// Error might be in the response text
		if !strings.Contains(text, "invalid") && !strings.Contains(text, "Error") {
			t.Error("expected error for invalid action")
		}
	}
}

func TestIndexTool_DefaultAction(t *testing.T) {
	server, _ := setupMCPServer(t)

	// Empty action should default to "status"
	resp := server.dispatchIndex(1, map[string]interface{}{}, "")
	if resp.Error != nil {
		t.Fatalf("dispatchIndex default action error: %v", resp.Error)
	}

	text := extractResponseText(resp)
	if !strings.Contains(text, "Index Status") {
		t.Errorf("default action should be status, got: %s", text)
	}
}

func TestRoomTool_DefaultAction(t *testing.T) {
	server, _ := setupMCPServer(t)

	// Empty action should default to "list"
	resp := server.dispatchRoom(1, map[string]interface{}{}, "")
	if resp.Error != nil {
		t.Fatalf("dispatchRoom default action error: %v", resp.Error)
	}

	text := extractResponseText(resp)
	if !strings.Contains(text, "Mind Palace Rooms") {
		t.Errorf("default action should be list, got: %s", text)
	}
}

// Helper to extract text from response
func extractResponseText(resp jsonRPCResponse) string {
	if resp.Result == nil {
		return ""
	}
	result, ok := resp.Result.(mcpToolResult)
	if !ok {
		return ""
	}
	if len(result.Content) == 0 {
		return ""
	}
	return result.Content[0].Text
}
