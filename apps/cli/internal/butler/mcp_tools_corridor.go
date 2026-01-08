package butler

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/corridor"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// toolCorridorLearnings retrieves personal learnings from the global corridor.
func (s *MCPServer) toolCorridorLearnings(id any, args map[string]interface{}) jsonRPCResponse {
	gc, err := corridor.OpenGlobal()
	if err != nil {
		return s.toolError(id, "open corridor: "+err.Error())
	}
	defer gc.Close()

	query := ""
	if q, ok := args["query"].(string); ok {
		query = q
	}

	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	learnings, err := gc.GetPersonalLearnings(query, limit)
	if err != nil {
		return s.toolError(id, "get learnings: "+err.Error())
	}
	if learnings == nil {
		learnings = []corridor.PersonalLearning{}
	}

	var output strings.Builder
	output.WriteString("# Personal Corridor Learnings\n\n")
	if len(learnings) == 0 {
		output.WriteString("*No personal learnings found.*\n")
	} else {
		for i := range learnings {
			l := &learnings[i]
			fmt.Fprintf(&output, "- **[%s]** %s (confidence: %.1f)\n", l.ID, l.Content, l.Confidence)
			if l.OriginWorkspace != "" {
				fmt.Fprintf(&output, "  - Origin: %s\n", l.OriginWorkspace)
			}
		}
	}
	fmt.Fprintf(&output, "\nTotal: %d learnings", len(learnings))

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolCorridorLinks retrieves linked workspaces.
func (s *MCPServer) toolCorridorLinks(id any, _ map[string]interface{}) jsonRPCResponse {
	gc, err := corridor.OpenGlobal()
	if err != nil {
		return s.toolError(id, "open corridor: "+err.Error())
	}
	defer gc.Close()

	links, err := gc.GetLinks()
	if err != nil {
		return s.toolError(id, "get links: "+err.Error())
	}
	if links == nil {
		links = []corridor.LinkedWorkspace{}
	}

	var output strings.Builder
	output.WriteString("# Linked Workspaces\n\n")
	if len(links) == 0 {
		output.WriteString("*No workspaces linked.*\n\n")
		output.WriteString("Use `palace corridor link <name> <path>` to link a workspace.\n")
	} else {
		for _, l := range links {
			fmt.Fprintf(&output, "- **%s**: `%s`\n", l.Name, l.Path)
			if !l.LastAccessed.IsZero() {
				fmt.Fprintf(&output, "  - Last accessed: %s\n", l.LastAccessed.Format("2006-01-02 15:04"))
			}
		}
	}
	fmt.Fprintf(&output, "\nTotal: %d linked workspaces", len(links))

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolCorridorStats retrieves corridor statistics.
func (s *MCPServer) toolCorridorStats(id any, _ map[string]interface{}) jsonRPCResponse {
	gc, err := corridor.OpenGlobal()
	if err != nil {
		return s.toolError(id, "open corridor: "+err.Error())
	}
	defer gc.Close()

	stats, err := gc.Stats()
	if err != nil {
		return s.toolError(id, "get stats: "+err.Error())
	}

	// Return as JSON for easy parsing
	data, _ := json.MarshalIndent(stats, "", "  ")

	var output strings.Builder
	output.WriteString("# Corridor Statistics\n\n")
	output.WriteString("```json\n")
	output.Write(data)
	output.WriteString("\n```\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolCorridorPromote promotes a learning to the personal corridor.
func (s *MCPServer) toolCorridorPromote(id any, args map[string]interface{}) jsonRPCResponse {
	learningID, ok := args["learningId"].(string)
	if !ok || learningID == "" {
		return s.toolError(id, "learningId is required")
	}

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not available")
	}

	// Get the learning from workspace memory
	learnings, err := mem.GetLearnings("palace", "", 100)
	if err != nil {
		return s.toolError(id, "get learning: "+err.Error())
	}

	var found bool
	var targetLearning memory.Learning
	for i := range learnings {
		if learnings[i].ID == learningID {
			found = true
			targetLearning = learnings[i]
			break
		}
	}

	if found {
		// Promote to personal corridor
		gc, err := corridor.OpenGlobal()
		if err != nil {
			return s.toolError(id, "open corridor: "+err.Error())
		}
		defer gc.Close()

		workspaceName := s.butler.GetWorkspaceName()
		if err := gc.PromoteFromWorkspace(workspaceName, targetLearning); err != nil {
			return s.toolError(id, "promote: "+err.Error())
		}
	}

	if !found {
		return s.toolError(id, "learning not found: "+learningID)
	}

	var output strings.Builder
	output.WriteString("# Learning Promoted\n\n")
	fmt.Fprintf(&output, "**ID:** %s\n", learningID)
	fmt.Fprintf(&output, "**Content:** %s\n", targetLearning.Content)
	fmt.Fprintf(&output, "**Origin:** %s\n\n", s.butler.GetWorkspaceName())
	output.WriteString("This learning is now available in your personal corridor across all workspaces.")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolCorridorReinforce increases confidence for a personal learning.
func (s *MCPServer) toolCorridorReinforce(id any, args map[string]interface{}) jsonRPCResponse {
	learningID, ok := args["learningId"].(string)
	if !ok || learningID == "" {
		return s.toolError(id, "learningId is required")
	}

	gc, err := corridor.OpenGlobal()
	if err != nil {
		return s.toolError(id, "open corridor: "+err.Error())
	}
	defer gc.Close()

	if err := gc.ReinforceLearning(learningID); err != nil {
		return s.toolError(id, "reinforce: "+err.Error())
	}

	var output strings.Builder
	output.WriteString("# Learning Reinforced\n\n")
	fmt.Fprintf(&output, "**ID:** %s\n\n", learningID)
	output.WriteString("Confidence increased. This learning will rank higher in future queries.")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}
