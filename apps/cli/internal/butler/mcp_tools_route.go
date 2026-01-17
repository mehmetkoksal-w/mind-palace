package butler

import (
	"encoding/json"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// toolGetRoute handles the get_route MCP tool call.
// It derives a deterministic navigation route based on intent and scope.
func (s *MCPServer) toolGetRoute(id any, args map[string]interface{}) jsonRPCResponse {
	// Parse intent (required)
	intent, ok := args["intent"].(string)
	if !ok || intent == "" {
		return s.toolError(id, "Missing required parameter: intent")
	}

	// Parse scope (optional, default: palace)
	scopeStr := "palace"
	if scopeArg, ok := args["scope"].(string); ok && scopeArg != "" {
		scopeStr = scopeArg
	}

	// Validate scope
	var scope memory.Scope
	switch scopeStr {
	case "file":
		scope = memory.ScopeFile
	case "room":
		scope = memory.ScopeRoom
	case "palace":
		scope = memory.ScopePalace
	default:
		return s.toolError(id, "Invalid scope: must be 'file', 'room', or 'palace'")
	}

	// Parse scopePath (optional)
	scopePath := ""
	if sp, ok := args["scopePath"].(string); ok {
		scopePath = sp
	}

	// Validate: file scope requires scopePath
	if scope == memory.ScopeFile && scopePath == "" {
		return s.toolError(id, "scopePath is required when scope is 'file'")
	}

	// Derive the route
	result, err := s.butler.GetRoute(intent, scope, scopePath, nil)
	if err != nil {
		return s.toolError(id, "Failed to derive route: "+err.Error())
	}

	// Serialize result
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return s.toolError(id, "Failed to serialize route: "+err.Error())
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{
				{Type: "text", Text: string(resultJSON)},
			},
		},
	}
}
