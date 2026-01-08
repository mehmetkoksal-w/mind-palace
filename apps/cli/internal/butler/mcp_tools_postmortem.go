package butler

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// toolStorePostmortem creates a new postmortem record.
func (s *MCPServer) toolStorePostmortem(id any, args map[string]interface{}) jsonRPCResponse {
	title, _ := args["title"].(string)
	whatHappened, _ := args["what_happened"].(string)
	rootCause, _ := args["root_cause"].(string)
	severity, _ := args["severity"].(string)

	if title == "" {
		return s.toolError(id, "title is required")
	}
	if whatHappened == "" {
		return s.toolError(id, "what_happened is required")
	}

	input := memory.PostmortemInput{
		Title:        title,
		WhatHappened: whatHappened,
		RootCause:    rootCause,
		Severity:     severity,
	}

	// Parse lessons learned
	if lessons, ok := args["lessons_learned"].([]interface{}); ok {
		for _, l := range lessons {
			if s, ok := l.(string); ok {
				input.LessonsLearned = append(input.LessonsLearned, s)
			}
		}
	}

	// Parse prevention steps
	if steps, ok := args["prevention_steps"].([]interface{}); ok {
		for _, s := range steps {
			if str, ok := s.(string); ok {
				input.PreventionSteps = append(input.PreventionSteps, str)
			}
		}
	}

	// Parse affected files
	if files, ok := args["affected_files"].([]interface{}); ok {
		for _, f := range files {
			if s, ok := f.(string); ok {
				input.AffectedFiles = append(input.AffectedFiles, s)
			}
		}
	}

	// Optional related decision/session
	if rd, ok := args["related_decision"].(string); ok {
		input.RelatedDecision = rd
	}
	if rs, ok := args["related_session"].(string); ok {
		input.RelatedSession = rs
	}

	pm, err := s.butler.memory.StorePostmortem(input)
	if err != nil {
		return s.toolError(id, err.Error())
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{
				Type: "text",
				Text: fmt.Sprintf("Postmortem created: %s\nTitle: %s\nSeverity: %s\nStatus: %s", pm.ID, pm.Title, pm.Severity, pm.Status),
			}},
		},
	}
}

// toolGetPostmortems retrieves postmortems with optional filters.
func (s *MCPServer) toolGetPostmortems(id any, args map[string]interface{}) jsonRPCResponse {
	status, _ := args["status"].(string)
	severity, _ := args["severity"].(string)
	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	postmortems, err := s.butler.memory.GetPostmortems(status, severity, limit)
	if err != nil {
		return s.toolError(id, err.Error())
	}

	if len(postmortems) == 0 {
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: mcpToolResult{
				Content: []mcpContent{{
					Type: "text",
					Text: "No postmortems found.",
				}},
			},
		}
	}

	// Format as readable text
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d postmortem(s):\n\n", len(postmortems)))

	for i := range postmortems {
		pm := &postmortems[i]
		sb.WriteString(fmt.Sprintf("## %s [%s] - %s\n", pm.Title, pm.Severity, pm.Status))
		sb.WriteString(fmt.Sprintf("ID: %s | Created: %s\n", pm.ID, pm.CreatedAt.Format("2006-01-02")))
		sb.WriteString(fmt.Sprintf("What happened: %s\n", truncate(pm.WhatHappened, 200)))
		if pm.RootCause != "" {
			sb.WriteString(fmt.Sprintf("Root cause: %s\n", truncate(pm.RootCause, 100)))
		}
		if len(pm.LessonsLearned) > 0 {
			sb.WriteString(fmt.Sprintf("Lessons: %d\n", len(pm.LessonsLearned)))
		}
		sb.WriteString("\n")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{
				Type: "text",
				Text: sb.String(),
			}},
		},
	}
}

// toolGetPostmortem retrieves a single postmortem by ID.
func (s *MCPServer) toolGetPostmortem(id any, args map[string]interface{}) jsonRPCResponse {
	postmortemID, _ := args["id"].(string)
	if postmortemID == "" {
		return s.toolError(id, "id is required")
	}

	pm, err := s.butler.memory.GetPostmortem(postmortemID)
	if err != nil {
		return s.toolError(id, err.Error())
	}
	if pm == nil {
		return s.toolError(id, "postmortem not found")
	}

	// Format as detailed view
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", pm.Title))
	sb.WriteString(fmt.Sprintf("**ID:** %s\n", pm.ID))
	sb.WriteString(fmt.Sprintf("**Severity:** %s\n", pm.Severity))
	sb.WriteString(fmt.Sprintf("**Status:** %s\n", pm.Status))
	sb.WriteString(fmt.Sprintf("**Created:** %s\n", pm.CreatedAt.Format("2006-01-02 15:04")))
	if !pm.ResolvedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("**Resolved:** %s\n", pm.ResolvedAt.Format("2006-01-02 15:04")))
	}
	sb.WriteString("\n## What Happened\n")
	sb.WriteString(pm.WhatHappened)
	sb.WriteString("\n")

	if pm.RootCause != "" {
		sb.WriteString("\n## Root Cause\n")
		sb.WriteString(pm.RootCause)
		sb.WriteString("\n")
	}

	if len(pm.LessonsLearned) > 0 {
		sb.WriteString("\n## Lessons Learned\n")
		for _, lesson := range pm.LessonsLearned {
			sb.WriteString(fmt.Sprintf("- %s\n", lesson))
		}
	}

	if len(pm.PreventionSteps) > 0 {
		sb.WriteString("\n## Prevention Steps\n")
		for _, step := range pm.PreventionSteps {
			sb.WriteString(fmt.Sprintf("- %s\n", step))
		}
	}

	if len(pm.AffectedFiles) > 0 {
		sb.WriteString("\n## Affected Files\n")
		for _, f := range pm.AffectedFiles {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
	}

	if pm.RelatedDecision != "" {
		sb.WriteString(fmt.Sprintf("\n**Related Decision:** %s\n", pm.RelatedDecision))
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{
				Type: "text",
				Text: sb.String(),
			}},
		},
	}
}

// toolResolvePostmortem marks a postmortem as resolved.
func (s *MCPServer) toolResolvePostmortem(id any, args map[string]interface{}) jsonRPCResponse {
	postmortemID, _ := args["id"].(string)
	if postmortemID == "" {
		return s.toolError(id, "id is required")
	}

	err := s.butler.memory.ResolvePostmortem(postmortemID)
	if err != nil {
		return s.toolError(id, err.Error())
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{
				Type: "text",
				Text: fmt.Sprintf("Postmortem %s marked as resolved.", postmortemID),
			}},
		},
	}
}

// toolPostmortemStats returns aggregated postmortem statistics.
func (s *MCPServer) toolPostmortemStats(id any, _ map[string]interface{}) jsonRPCResponse {
	stats, err := s.butler.memory.GetPostmortemStats()
	if err != nil {
		return s.toolError(id, err.Error())
	}

	data, _ := json.MarshalIndent(stats, "", "  ")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{
				Type: "text",
				Text: fmt.Sprintf("Postmortem Statistics:\n\nTotal: %d\nOpen: %d\nResolved: %d\nRecurring: %d\n\nBy Severity:\n%s",
					stats.Total, stats.Open, stats.Resolved, stats.Recurring, string(data)),
			}},
		},
	}
}

// toolPostmortemToLearnings converts a postmortem's lessons to learnings.
func (s *MCPServer) toolPostmortemToLearnings(id any, args map[string]interface{}) jsonRPCResponse {
	postmortemID, _ := args["id"].(string)
	if postmortemID == "" {
		return s.toolError(id, "id is required")
	}

	learningIDs, err := s.butler.memory.ConvertPostmortemToLearning(postmortemID)
	if err != nil {
		return s.toolError(id, err.Error())
	}

	if len(learningIDs) == 0 {
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: mcpToolResult{
				Content: []mcpContent{{
					Type: "text",
					Text: "No lessons to convert. Add lessons_learned to the postmortem first.",
				}},
			},
		}
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{
				Type: "text",
				Text: fmt.Sprintf("Created %d learning(s) from postmortem:\n%s", len(learningIDs), strings.Join(learningIDs, "\n")),
			}},
		},
	}
}
