package butler

import (
	"fmt"
	"strings"
)

// toolExplore searches the codebase by intent or keywords.
func (s *MCPServer) toolExplore(id any, args map[string]interface{}) jsonRPCResponse {
	query, _ := args["query"].(string)
	if query == "" {
		return s.toolError(id, "query is required")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
		if limit > 50 {
			limit = 50
		}
	}

	roomFilter, _ := args["room"].(string)
	fuzzyMatch, _ := args["fuzzy"].(bool)

	results, err := s.butler.Search(query, SearchOptions{
		Limit:      limit,
		RoomFilter: roomFilter,
		FuzzyMatch: fuzzyMatch,
	})
	if err != nil {
		return s.toolError(id, fmt.Sprintf("search failed: %v", err))
	}

	// Format results as readable text
	var output strings.Builder
	for _, group := range results {
		fmt.Fprintf(&output, "## Room: %s\n", group.Room)
		if group.Summary != "" {
			fmt.Fprintf(&output, "_Summary: %s_\n", group.Summary)
		}
		output.WriteString("\n")

		for _, r := range group.Results {
			entryMark := ""
			if r.IsEntry {
				entryMark = " ⭐ (entry point)"
			}
			fmt.Fprintf(&output, "### %s%s\n", r.Path, entryMark)
			fmt.Fprintf(&output, "Lines %d-%d (score: %.2f)\n", r.StartLine, r.EndLine, r.Score)
			fmt.Fprintf(&output, "```\n%s\n```\n\n", truncateSnippet(r.Snippet, 500))
		}
	}

	if len(results) == 0 {
		output.WriteString("No results found for the query.")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolExploreRooms lists all available Rooms in the Mind Palace.
func (s *MCPServer) toolExploreRooms(id any) jsonRPCResponse {
	rooms := s.butler.ListRooms()

	var output strings.Builder
	output.WriteString("# Mind Palace Rooms\n\n")

	for _, room := range rooms {
		fmt.Fprintf(&output, "## %s\n", room.Name)
		fmt.Fprintf(&output, "%s\n\n", room.Summary)
		output.WriteString("**Entry Points:**\n")
		for _, ep := range room.EntryPoints {
			fmt.Fprintf(&output, "- `%s`\n", ep)
		}
		if len(room.Capabilities) > 0 {
			output.WriteString("\n**Capabilities:**\n")
			for _, cap := range room.Capabilities {
				fmt.Fprintf(&output, "- %s\n", cap)
			}
		}
		output.WriteString("\n")
	}

	if len(rooms) == 0 {
		output.WriteString("No rooms defined. Run `palace init` to create default rooms.")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}
