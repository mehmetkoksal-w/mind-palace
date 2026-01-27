package butler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/model"
)

// ============================================================
// ROOM TOOL - Manage Mind Palace rooms
// ============================================================

// dispatchRoom handles the room tool with action parameter.
func (s *MCPServer) dispatchRoom(id any, args map[string]interface{}, action string) jsonRPCResponse {
	if action == "" {
		action = "list" // default action
	}

	switch action {
	case "list":
		return s.toolRoomList(id)
	case "show":
		return s.toolRoomShow(id, args)
	case "create":
		return s.toolRoomCreate(id, args)
	case "update":
		return s.toolRoomUpdate(id, args)
	case "delete":
		return s.toolRoomDelete(id, args)
	default:
		return consolidatedToolError(id, "room", "action", action)
	}
}

// toolRoomList lists all available rooms.
func (s *MCPServer) toolRoomList(id any) jsonRPCResponse {
	rooms := s.butler.ListRooms()

	var output strings.Builder
	output.WriteString("# Mind Palace Rooms\n\n")

	if len(rooms) == 0 {
		output.WriteString("No rooms defined. Use `room` tool with `action=create` to create a room.\n")
		output.WriteString("\nExample:\n```json\n")
		output.WriteString(`{"action": "create", "name": "auth", "summary": "Authentication module", "entry_points": ["src/auth/"]}`)
		output.WriteString("\n```\n")
	} else {
		output.WriteString("| Room | Summary | Entry Points |\n")
		output.WriteString("|------|---------|-------------|\n")
		for _, room := range rooms {
			eps := strings.Join(room.EntryPoints, ", ")
			if len(eps) > 40 {
				eps = eps[:37] + "..."
			}
			fmt.Fprintf(&output, "| %s | %s | %s |\n",
				room.Name,
				truncateSnippet(room.Summary, 50),
				eps)
		}
		fmt.Fprintf(&output, "\n**Total:** %d rooms\n", len(rooms))
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolRoomShow shows details for a specific room.
func (s *MCPServer) toolRoomShow(id any, args map[string]interface{}) jsonRPCResponse {
	name := getStringArg(args, "name", "")
	if name == "" {
		return s.toolError(id, "room name is required (set 'name' parameter)")
	}

	rooms := s.butler.ListRooms()
	var room *model.Room
	for i := range rooms {
		if rooms[i].Name == name {
			room = &rooms[i]
			break
		}
	}

	if room == nil {
		// Suggest similar names
		var suggestions []string
		for _, r := range rooms {
			if strings.Contains(strings.ToLower(r.Name), strings.ToLower(name)) {
				suggestions = append(suggestions, r.Name)
			}
		}
		msg := fmt.Sprintf("room '%s' not found", name)
		if len(suggestions) > 0 {
			msg += fmt.Sprintf(". Did you mean: %s?", strings.Join(suggestions, ", "))
		}
		return s.toolError(id, msg)
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# Room: %s\n\n", room.Name)
	fmt.Fprintf(&output, "**Summary:** %s\n\n", room.Summary)

	output.WriteString("## Entry Points\n")
	for _, ep := range room.EntryPoints {
		fmt.Fprintf(&output, "- `%s`\n", ep)
	}

	if len(room.Capabilities) > 0 {
		output.WriteString("\n## Capabilities\n")
		for _, cap := range room.Capabilities {
			fmt.Fprintf(&output, "- %s\n", cap)
		}
	}

	if len(room.Artifacts) > 0 {
		output.WriteString("\n## Artifacts\n")
		for _, art := range room.Artifacts {
			if art.PathHint != "" {
				fmt.Fprintf(&output, "- **%s** (`%s`): %s\n", art.Name, art.PathHint, art.Description)
			} else {
				fmt.Fprintf(&output, "- **%s**: %s\n", art.Name, art.Description)
			}
		}
	}

	if len(room.Steps) > 0 {
		output.WriteString("\n## Steps\n")
		for i, step := range room.Steps {
			fmt.Fprintf(&output, "%d. **%s**", i+1, step.Name)
			if step.Description != "" {
				fmt.Fprintf(&output, ": %s", step.Description)
			}
			output.WriteString("\n")
			if step.Evidence != "" {
				fmt.Fprintf(&output, "   - Evidence: %s\n", step.Evidence)
			}
		}
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolRoomCreate creates a new room.
func (s *MCPServer) toolRoomCreate(id any, args map[string]interface{}) jsonRPCResponse {
	name := getStringArg(args, "name", "")
	if name == "" {
		return s.toolError(id, "room name is required")
	}

	summary := getStringArg(args, "summary", "")
	if summary == "" {
		return s.toolError(id, "room summary is required")
	}

	// Parse entry points
	var entryPoints []string
	if eps, ok := args["entry_points"].([]interface{}); ok {
		for _, ep := range eps {
			if s, ok := ep.(string); ok {
				entryPoints = append(entryPoints, s)
			}
		}
	}
	if len(entryPoints) == 0 {
		return s.toolError(id, "at least one entry_point is required")
	}

	// Parse optional capabilities
	var capabilities []string
	if caps, ok := args["capabilities"].([]interface{}); ok {
		for _, cap := range caps {
			if s, ok := cap.(string); ok {
				capabilities = append(capabilities, s)
			}
		}
	}

	// Check if room already exists
	rooms := s.butler.ListRooms()
	for _, r := range rooms {
		if r.Name == name {
			return s.toolError(id, fmt.Sprintf("room '%s' already exists. Use action=update to modify it.", name))
		}
	}

	// Create room
	room := model.Room{
		SchemaVersion: "1.0.0",
		Kind:          "palace/room",
		Name:          name,
		Summary:       summary,
		EntryPoints:   entryPoints,
		Capabilities:  capabilities,
	}

	// Write to file
	roomsDir := filepath.Join(s.butler.root, ".palace", "rooms")
	if err := os.MkdirAll(roomsDir, 0o755); err != nil {
		return s.toolError(id, fmt.Sprintf("create rooms directory: %v", err))
	}

	roomFile := filepath.Join(roomsDir, name+".jsonc")
	data, err := json.MarshalIndent(room, "", "  ")
	if err != nil {
		return s.toolError(id, fmt.Sprintf("marshal room: %v", err))
	}

	if err := os.WriteFile(roomFile, data, 0o644); err != nil {
		return s.toolError(id, fmt.Sprintf("write room file: %v", err))
	}

	// Reload rooms
	s.butler.reloadRooms()

	var output strings.Builder
	fmt.Fprintf(&output, "# ✅ Room Created: %s\n\n", name)
	fmt.Fprintf(&output, "**Summary:** %s\n\n", summary)
	output.WriteString("**Entry Points:**\n")
	for _, ep := range entryPoints {
		fmt.Fprintf(&output, "- `%s`\n", ep)
	}
	if len(capabilities) > 0 {
		output.WriteString("\n**Capabilities:**\n")
		for _, cap := range capabilities {
			fmt.Fprintf(&output, "- %s\n", cap)
		}
	}
	fmt.Fprintf(&output, "\n**File:** `.palace/rooms/%s.jsonc`\n", name)

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolRoomUpdate updates an existing room.
func (s *MCPServer) toolRoomUpdate(id any, args map[string]interface{}) jsonRPCResponse {
	name := getStringArg(args, "name", "")
	if name == "" {
		return s.toolError(id, "room name is required")
	}

	// Find existing room
	rooms := s.butler.ListRooms()
	var existingRoom *model.Room
	for i := range rooms {
		if rooms[i].Name == name {
			existingRoom = &rooms[i]
			break
		}
	}

	if existingRoom == nil {
		return s.toolError(id, fmt.Sprintf("room '%s' not found. Use action=create to create it.", name))
	}

	// Apply updates
	updated := *existingRoom

	if summary := getStringArg(args, "summary", ""); summary != "" {
		updated.Summary = summary
	}

	if eps, ok := args["entry_points"].([]interface{}); ok && len(eps) > 0 {
		updated.EntryPoints = nil
		for _, ep := range eps {
			if s, ok := ep.(string); ok {
				updated.EntryPoints = append(updated.EntryPoints, s)
			}
		}
	}

	if caps, ok := args["capabilities"].([]interface{}); ok {
		updated.Capabilities = nil
		for _, cap := range caps {
			if s, ok := cap.(string); ok {
				updated.Capabilities = append(updated.Capabilities, s)
			}
		}
	}

	// Write updated room
	roomFile := filepath.Join(s.butler.root, ".palace", "rooms", name+".jsonc")
	data, err := json.MarshalIndent(updated, "", "  ")
	if err != nil {
		return s.toolError(id, fmt.Sprintf("marshal room: %v", err))
	}

	if err := os.WriteFile(roomFile, data, 0o644); err != nil {
		return s.toolError(id, fmt.Sprintf("write room file: %v", err))
	}

	// Reload rooms
	s.butler.reloadRooms()

	var output strings.Builder
	fmt.Fprintf(&output, "# ✅ Room Updated: %s\n\n", name)
	fmt.Fprintf(&output, "**Summary:** %s\n\n", updated.Summary)
	output.WriteString("**Entry Points:**\n")
	for _, ep := range updated.EntryPoints {
		fmt.Fprintf(&output, "- `%s`\n", ep)
	}
	if len(updated.Capabilities) > 0 {
		output.WriteString("\n**Capabilities:**\n")
		for _, cap := range updated.Capabilities {
			fmt.Fprintf(&output, "- %s\n", cap)
		}
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolRoomDelete deletes a room.
func (s *MCPServer) toolRoomDelete(id any, args map[string]interface{}) jsonRPCResponse {
	name := getStringArg(args, "name", "")
	if name == "" {
		return s.toolError(id, "room name is required")
	}

	// Verify room exists
	rooms := s.butler.ListRooms()
	found := false
	for _, r := range rooms {
		if r.Name == name {
			found = true
			break
		}
	}

	if !found {
		return s.toolError(id, fmt.Sprintf("room '%s' not found", name))
	}

	// Delete room file
	roomFile := filepath.Join(s.butler.root, ".palace", "rooms", name+".jsonc")
	if err := os.Remove(roomFile); err != nil {
		return s.toolError(id, fmt.Sprintf("delete room file: %v", err))
	}

	// Reload rooms
	s.butler.reloadRooms()

	var output strings.Builder
	fmt.Fprintf(&output, "# ✅ Room Deleted: %s\n\n", name)
	output.WriteString("The room has been removed from the Mind Palace.\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}
