package butler

import (
	"fmt"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// toolSessionStart starts a new agent session.
func (s *MCPServer) toolSessionStart(id any, args map[string]interface{}) jsonRPCResponse {
	agentType, _ := args["agentType"].(string)
	if agentType == "" {
		return s.toolError(id, "agentType is required")
	}

	agentID, _ := args["agentId"].(string)
	goal, _ := args["goal"].(string)

	session, err := s.butler.StartSession(agentType, agentID, goal)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("start session failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Session Started\n\n")
	fmt.Fprintf(&output, "**Session ID:** `%s`\n", session.ID)
	fmt.Fprintf(&output, "**Agent Type:** %s\n", session.AgentType)
	if session.Goal != "" {
		fmt.Fprintf(&output, "**Goal:** %s\n", session.Goal)
	}
	fmt.Fprintf(&output, "**Started:** %s\n", session.StartedAt.Format(time.RFC3339))
	output.WriteString("\nUse this session ID to log activities and end the session.")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolSessionLog logs an activity within a session.
func (s *MCPServer) toolSessionLog(id any, args map[string]interface{}) jsonRPCResponse {
	sessionID, _ := args["sessionId"].(string)
	if sessionID == "" {
		return s.toolError(id, "sessionId is required")
	}

	kind, _ := args["kind"].(string)
	if kind == "" {
		return s.toolError(id, "kind is required")
	}

	target, _ := args["target"].(string)
	outcome, _ := args["outcome"].(string)
	details, _ := args["details"].(string)

	if outcome == "" {
		outcome = "unknown"
	}
	if details == "" {
		details = "{}"
	}

	act := memory.Activity{
		Kind:    kind,
		Target:  target,
		Outcome: outcome,
		Details: details,
	}

	err := s.butler.LogActivity(sessionID, act)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("log activity failed: %v", err))
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("Activity logged: %s on %s (%s)", kind, target, outcome)}},
		},
	}
}

// toolSessionEnd ends a session and records its outcome.
func (s *MCPServer) toolSessionEnd(id any, args map[string]interface{}) jsonRPCResponse {
	sessionID, _ := args["sessionId"].(string)
	if sessionID == "" {
		return s.toolError(id, "sessionId is required")
	}

	outcome, _ := args["outcome"].(string)
	summary, _ := args["summary"].(string)

	state := "completed"
	if outcome == "failure" {
		state = "abandoned"
	}

	// Record outcome if provided
	if outcome != "" {
		if err := s.butler.RecordOutcome(sessionID, outcome, summary); err != nil {
			return s.toolError(id, fmt.Sprintf("record outcome failed: %v", err))
		}
	}

	if err := s.butler.EndSession(sessionID, state, summary); err != nil {
		return s.toolError(id, fmt.Sprintf("end session failed: %v", err))
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("Session %s ended (state: %s)", sessionID, state)}},
		},
	}
}

// toolRecall retrieves learnings, optionally filtered by scope or search query.
func (s *MCPServer) toolRecall(id any, args map[string]interface{}) jsonRPCResponse {
	query, _ := args["query"].(string)
	scope, _ := args["scope"].(string)
	scopePath, _ := args["scopePath"].(string)

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	var learnings []memory.Learning
	var err error

	if query != "" {
		learnings, err = s.butler.SearchLearnings(query, limit)
	} else {
		learnings, err = s.butler.GetLearnings(scope, scopePath, limit)
	}

	if err != nil {
		return s.toolError(id, fmt.Sprintf("get learnings failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Learnings\n\n")

	if len(learnings) == 0 {
		output.WriteString("No learnings found.\n")
	} else {
		for i := range learnings {
			l := &learnings[i]
			scopeInfo := l.Scope
			if l.ScopePath != "" {
				scopeInfo = fmt.Sprintf("%s:%s", l.Scope, l.ScopePath)
			}
			fmt.Fprintf(&output, "## `%s` (%.0f%% confidence)\n", l.ID, l.Confidence*100)
			fmt.Fprintf(&output, "- **Scope:** %s\n", scopeInfo)
			fmt.Fprintf(&output, "- **Source:** %s | Used: %d times\n", l.Source, l.UseCount)
			fmt.Fprintf(&output, "- **Content:** %s\n\n", l.Content)
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

// toolBriefFile gets intelligence about a file.
func (s *MCPServer) toolBriefFile(id any, args map[string]interface{}) jsonRPCResponse {
	path, _ := args["path"].(string)
	if path == "" {
		return s.toolError(id, "path is required")
	}

	intel, err := s.butler.GetFileIntel(path)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get file intel failed: %v", err))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# File Intelligence: `%s`\n\n", path)
	fmt.Fprintf(&output, "**Edit Count:** %d\n", intel.EditCount)
	fmt.Fprintf(&output, "**Failure Count:** %d\n", intel.FailureCount)
	if intel.EditCount > 0 {
		failureRate := float64(intel.FailureCount) / float64(intel.EditCount) * 100
		fmt.Fprintf(&output, "**Failure Rate:** %.1f%%\n", failureRate)
	}
	if !intel.LastEdited.IsZero() {
		fmt.Fprintf(&output, "**Last Edited:** %s\n", intel.LastEdited.Format(time.RFC3339))
	}
	if intel.LastEditor != "" {
		fmt.Fprintf(&output, "**Last Editor:** %s\n", intel.LastEditor)
	}

	if len(intel.Learnings) > 0 {
		output.WriteString("\n## Associated Learnings\n\n")
		for _, learningID := range intel.Learnings {
			fmt.Fprintf(&output, "- `%s`\n", learningID)
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

// toolBrief gets a comprehensive briefing before working.
func (s *MCPServer) toolBrief(id any, args map[string]interface{}) jsonRPCResponse {
	filePath, _ := args["filePath"].(string)

	brief, err := s.butler.GetBrief(filePath)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get brief failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Briefing")
	if filePath != "" {
		fmt.Fprintf(&output, " for `%s`", filePath)
	}
	output.WriteString("\n\n")

	// Active agents
	if len(brief.ActiveAgents) > 0 {
		output.WriteString("## Active Agents\n\n")
		for i := range brief.ActiveAgents {
			a := &brief.ActiveAgents[i]
			currentFile := ""
			if a.CurrentFile != "" {
				currentFile = fmt.Sprintf(" working on `%s`", a.CurrentFile)
			}
			fmt.Fprintf(&output, "- **%s** (session: `%s`)%s\n", a.AgentType, a.SessionID[:12], currentFile)
		}
		output.WriteString("\n")
	}

	// Conflict warning
	if brief.Conflict != nil {
		output.WriteString("## ⚠️ Conflict Warning\n\n")
		fmt.Fprintf(&output, "Another agent (**%s**) touched this file recently.\n", brief.Conflict.OtherAgent)
		fmt.Fprintf(&output, "- Session: `%s`\n", brief.Conflict.OtherSession[:12])
		fmt.Fprintf(&output, "- Last touched: %s\n", brief.Conflict.LastTouched.Format("15:04:05"))
		fmt.Fprintf(&output, "- Severity: %s\n\n", brief.Conflict.Severity)
	}

	// File intel
	if brief.FileIntel != nil && brief.FileIntel.EditCount > 0 {
		output.WriteString("## File History\n\n")
		fmt.Fprintf(&output, "- **Edits:** %d\n", brief.FileIntel.EditCount)
		fmt.Fprintf(&output, "- **Failures:** %d\n", brief.FileIntel.FailureCount)
		if brief.FileIntel.EditCount > 0 {
			failureRate := float64(brief.FileIntel.FailureCount) / float64(brief.FileIntel.EditCount) * 100
			if failureRate > 20 {
				fmt.Fprintf(&output, "- ⚠️ **High failure rate:** %.0f%%\n", failureRate)
			}
		}
		output.WriteString("\n")
	}

	// Relevant learnings
	if len(brief.Learnings) > 0 {
		output.WriteString("## Relevant Learnings\n\n")
		for i := range brief.Learnings {
			l := &brief.Learnings[i]
			scopeInfo := ""
			if l.Scope != "palace" {
				scopeInfo = fmt.Sprintf(" [%s]", l.Scope)
			}
			fmt.Fprintf(&output, "- [%.0f%%]%s %s\n", l.Confidence*100, scopeInfo, l.Content)
		}
		output.WriteString("\n")
	}

	// Brain ideas
	if len(brief.BrainIdeas) > 0 {
		output.WriteString("## 💭 Active Ideas\n\n")
		for i := range brief.BrainIdeas {
			idea := &brief.BrainIdeas[i]
			fmt.Fprintf(&output, "- `%s` %s\n", idea.ID, idea.Content)
		}
		output.WriteString("\n")
	}

	// Brain decisions
	if len(brief.BrainDecisions) > 0 {
		output.WriteString("## 📋 Active Decisions\n\n")
		for i := range brief.BrainDecisions {
			d := &brief.BrainDecisions[i]
			outcomeIcon := ""
			switch d.Outcome {
			case "successful":
				outcomeIcon = " ✅"
			case "failed":
				outcomeIcon = " ❌"
			case "mixed":
				outcomeIcon = " ⚖️"
			}
			fmt.Fprintf(&output, "- `%s`%s %s\n", d.ID, outcomeIcon, d.Content)
		}
		output.WriteString("\n")
	}

	// Hotspots
	if len(brief.Hotspots) > 0 {
		output.WriteString("## Hotspots (Most Edited Files)\n\n")
		for i := range brief.Hotspots {
			h := &brief.Hotspots[i]
			warning := ""
			if h.FailureCount > 0 {
				warning = fmt.Sprintf(" (⚠️ %d failures)", h.FailureCount)
			}
			fmt.Fprintf(&output, "- `%s` (%d edits)%s\n", h.Path, h.EditCount, warning)
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

// toolSessionList lists all sessions.
func (s *MCPServer) toolSessionList(id any, args map[string]interface{}) jsonRPCResponse {
	activeOnly := false
	if a, ok := args["active"].(bool); ok {
		activeOnly = a
	}

	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	sessions, err := s.butler.ListSessions(activeOnly, limit)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("list sessions failed: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Sessions\n\n")

	if len(sessions) == 0 {
		output.WriteString("No sessions found.\n")
	} else {
		for i := range sessions {
			sess := &sessions[i]
			statusIcon := "🟢"
			switch sess.State {
			case "completed":
				statusIcon = "✅"
			case "abandoned":
				statusIcon = "❌"
			}
			fmt.Fprintf(&output, "## %s `%s`\n", statusIcon, sess.ID)
			fmt.Fprintf(&output, "- **Agent:** %s\n", sess.AgentType)
			if sess.Goal != "" {
				fmt.Fprintf(&output, "- **Goal:** %s\n", sess.Goal)
			}
			fmt.Fprintf(&output, "- **State:** %s\n", sess.State)
			fmt.Fprintf(&output, "- **Started:** %s\n", sess.StartedAt.Format(time.RFC3339))
			if sess.State != "active" && !sess.LastActivity.IsZero() {
				fmt.Fprintf(&output, "- **Last Activity:** %s\n", sess.LastActivity.Format(time.RFC3339))
			}
			output.WriteString("\n")
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

// toolSessionConflict checks if another agent is working on a file.
func (s *MCPServer) toolSessionConflict(id any, args map[string]interface{}) jsonRPCResponse {
	path, _ := args["path"].(string)
	if path == "" {
		return s.toolError(id, "path is required")
	}

	sessionID, _ := args["sessionId"].(string)

	conflict, err := s.butler.CheckConflict(sessionID, path)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("check conflict failed: %v", err))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# Conflict Check: `%s`\n\n", path)

	if conflict == nil {
		output.WriteString("✅ **No conflicts detected.** Safe to proceed.\n")
	} else {
		output.WriteString("⚠️ **Conflict detected!**\n\n")
		fmt.Fprintf(&output, "- **Other Agent:** %s\n", conflict.OtherAgent)
		fmt.Fprintf(&output, "- **Session:** `%s`\n", conflict.OtherSession[:12])
		fmt.Fprintf(&output, "- **Last Touched:** %s\n", conflict.LastTouched.Format(time.RFC3339))
		fmt.Fprintf(&output, "- **Severity:** %s\n", conflict.Severity)
		output.WriteString("\nConsider coordinating with the other agent before making changes.")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}
