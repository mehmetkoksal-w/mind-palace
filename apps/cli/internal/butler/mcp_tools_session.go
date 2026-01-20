package butler

import (
	"fmt"
	"strings"
	"time"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/memory"
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

	// Track this session in the server (for auto-session detection)
	s.currentSessionID = session.ID
	s.autoSessionUsed = false

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

	// Get session info before ending
	mem := s.butler.Memory()
	session, _ := mem.GetSession(sessionID)

	// Record outcome if provided
	if outcome != "" {
		if err := s.butler.RecordOutcome(sessionID, outcome, summary); err != nil {
			return s.toolError(id, fmt.Sprintf("record outcome failed: %v", err))
		}
	}

	if err := s.butler.EndSession(sessionID, state, summary); err != nil {
		return s.toolError(id, fmt.Sprintf("end session failed: %v", err))
	}

	// Clear the tracked session if this was the current one
	if s.currentSessionID == sessionID {
		s.currentSessionID = ""
		s.autoSessionUsed = false
	}

	// Generate session summary
	var output strings.Builder
	output.WriteString("# Session Ended\n\n")
	fmt.Fprintf(&output, "**Session ID:** `%s`\n", sessionID)
	fmt.Fprintf(&output, "**State:** %s\n", state)

	if session != nil {
		fmt.Fprintf(&output, "**Agent:** %s\n", session.AgentType)
		if session.Goal != "" {
			fmt.Fprintf(&output, "**Goal:** %s\n", session.Goal)
		}
		duration := time.Since(session.StartedAt)
		fmt.Fprintf(&output, "**Duration:** %s\n", formatDuration(duration))
	}

	if summary != "" {
		fmt.Fprintf(&output, "\n**Summary:** %s\n", summary)
	}

	// Get activities summary
	activities, _ := mem.GetActivities(sessionID, "", 100)
	if len(activities) > 0 {
		output.WriteString("\n## Activity Summary\n\n")

		// Count activities by kind
		activityCounts := make(map[string]int)
		successCount := 0
		failureCount := 0
		var filesEdited []string
		filesSeen := make(map[string]bool)

		for i := range activities {
			act := &activities[i]
			activityCounts[act.Kind]++
			if act.Outcome == "success" {
				successCount++
			} else if act.Outcome == "failure" {
				failureCount++
			}
			if act.Kind == "file_edit" && act.Target != "" && !filesSeen[act.Target] {
				filesEdited = append(filesEdited, act.Target)
				filesSeen[act.Target] = true
			}
		}

		fmt.Fprintf(&output, "- **Total activities:** %d\n", len(activities))
		fmt.Fprintf(&output, "- **Successful:** %d | **Failed:** %d\n", successCount, failureCount)

		if len(activityCounts) > 0 {
			output.WriteString("- **By type:** ")
			first := true
			for kind, count := range activityCounts {
				if !first {
					output.WriteString(", ")
				}
				fmt.Fprintf(&output, "%s (%d)", kind, count)
				first = false
			}
			output.WriteString("\n")
		}

		if len(filesEdited) > 0 {
			output.WriteString("\n### Files Edited\n\n")
			for i, f := range filesEdited {
				if i >= 5 {
					fmt.Fprintf(&output, "- ... and %d more\n", len(filesEdited)-5)
					break
				}
				fmt.Fprintf(&output, "- `%s`\n", f)
			}
		}
	}

	// Get proposals created during session
	proposals, _ := mem.GetProposalsBySession(sessionID)
	if len(proposals) > 0 {
		output.WriteString("\n## Proposals Created\n\n")

		pendingCount := 0
		approvedCount := 0
		for i := range proposals {
			p := &proposals[i]
			statusIcon := "⏳"
			switch p.Status {
			case memory.ProposalStatusApproved:
				statusIcon = "✅"
				approvedCount++
			case memory.ProposalStatusRejected:
				statusIcon = "❌"
			case memory.ProposalStatusPending:
				pendingCount++
			}
			fmt.Fprintf(&output, "- %s `%s` (%s): %s\n", statusIcon, p.ID, p.ProposedAs, truncateString(p.Content, 60))
		}

		if pendingCount > 0 {
			fmt.Fprintf(&output, "\n⚠️ **%d proposal(s) pending review** - use `palace proposals` to review.\n", pendingCount)
		}
	}

	output.WriteString("\n---\n")
	output.WriteString("Session data has been preserved for future reference.\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

// toolRecall retrieves learnings, optionally filtered by scope or search query.
func (s *MCPServer) toolRecall(id any, args map[string]interface{}) jsonRPCResponse {
	// Support direct lookup by ID for route fetch_ref compatibility
	if idArg, ok := args["id"].(string); ok && idArg != "" {
		l, err := s.butler.memory.GetLearning(idArg)
		if err != nil {
			return s.toolError(id, fmt.Sprintf("get learning failed: %v", err))
		}

		var output strings.Builder
		scopeInfo := l.Scope
		if l.ScopePath != "" {
			scopeInfo = fmt.Sprintf("%s:%s", l.Scope, l.ScopePath)
		}
		fmt.Fprintf(&output, "# Learning `%s`\n\n", l.ID)
		fmt.Fprintf(&output, "- **Scope:** %s\n", scopeInfo)
		fmt.Fprintf(&output, "- **Confidence:** %.0f%%\n", l.Confidence*100)
		fmt.Fprintf(&output, "- **Source:** %s | Used: %d times\n", l.Source, l.UseCount)
		fmt.Fprintf(&output, "- **Content:** %s\n", l.Content)

		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: mcpToolResult{
				Content: []mcpContent{{Type: "text", Text: output.String()}},
			},
		}
	}

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

	// Related Knowledge Suggestions: when querying, also suggest related decisions and ideas
	if query != "" {
		cfg := s.butler.Config()
		// Check if proactive briefing (which includes related suggestions) is enabled
		if cfg != nil && cfg.Autonomy != nil && cfg.Autonomy.ProactiveBriefing {
			var relatedAdded bool

			// Find related decisions (max 3)
			if decisions, dErr := s.butler.SearchDecisions(query, 3); dErr == nil && len(decisions) > 0 {
				if !relatedAdded {
					output.WriteString("---\n\n## 💡 Related Knowledge\n\n")
					relatedAdded = true
				}
				output.WriteString("### Related Decisions\n\n")
				for i := range decisions {
					d := &decisions[i]
					fmt.Fprintf(&output, "- `%s`: %s\n", d.ID, truncateString(d.Content, 80))
				}
				output.WriteString("\n")
			}

			// Find related ideas (max 3)
			if ideas, iErr := s.butler.SearchIdeas(query, 3); iErr == nil && len(ideas) > 0 {
				if !relatedAdded {
					output.WriteString("---\n\n## 💡 Related Knowledge\n\n")
					relatedAdded = true
				}
				output.WriteString("### Related Ideas\n\n")
				for i := range ideas {
					idea := &ideas[i]
					fmt.Fprintf(&output, "- `%s`: %s\n", idea.ID, truncateString(idea.Content, 80))
				}
				output.WriteString("\n")
			}

			if relatedAdded {
				output.WriteString("Use `recall_decisions` or `recall_ideas` to explore these further.\n")
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

// toolSessionResume resumes a previous session for continuation.
func (s *MCPServer) toolSessionResume(id any, args map[string]interface{}) jsonRPCResponse {
	sessionID, _ := args["sessionId"].(string)
	if sessionID == "" {
		// Find the most recent session by this agent type
		agentType, _ := args["agentType"].(string)
		if agentType == "" {
			return s.toolError(id, "sessionId or agentType is required")
		}

		// Get recent sessions for this agent type
		sessions, err := s.butler.ListSessions(false, 10)
		if err != nil {
			return s.toolError(id, fmt.Sprintf("list sessions failed: %v", err))
		}

		// Find most recent non-completed session
		for i := range sessions {
			sess := &sessions[i]
			if sess.AgentType == agentType && sess.State != "completed" && sess.State != "abandoned" {
				sessionID = sess.ID
				break
			}
		}

		if sessionID == "" {
			return s.toolError(id, fmt.Sprintf("no resumable session found for agent type: %s", agentType))
		}
	}

	// Get the session
	mem := s.butler.Memory()
	session, err := mem.GetSession(sessionID)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get session failed: %v", err))
	}
	if session == nil {
		return s.toolError(id, "session not found")
	}

	// Reactivate the session if it was timed out
	if session.State == "timeout" {
		if err := mem.UpdateSessionState(sessionID, "active"); err != nil {
			return s.toolError(id, fmt.Sprintf("reactivate session failed: %v", err))
		}
		session.State = "active"
	}

	// Set as current session
	s.currentSessionID = sessionID
	s.autoSessionUsed = false

	var output strings.Builder
	output.WriteString("# Session Resumed\n\n")
	fmt.Fprintf(&output, "**Session ID:** `%s`\n", session.ID)
	fmt.Fprintf(&output, "**Agent:** %s\n", session.AgentType)
	if session.Goal != "" {
		fmt.Fprintf(&output, "**Original Task:** %s\n", session.Goal)
	}
	fmt.Fprintf(&output, "**Started:** %s\n", session.StartedAt.Format(time.RFC3339))
	fmt.Fprintf(&output, "**State:** %s\n\n", session.State)

	// Get activities summary
	activities, _ := mem.GetActivities(sessionID, "", 20)
	if len(activities) > 0 {
		output.WriteString("## Previous Activity\n\n")

		activityCounts := make(map[string]int)
		for i := range activities {
			activityCounts[activities[i].Kind]++
		}

		for kind, count := range activityCounts {
			fmt.Fprintf(&output, "- %s: %d\n", kind, count)
		}

		// Show last 3 activities
		output.WriteString("\n### Recent:\n\n")
		showCount := minInt(3, len(activities))
		for i := 0; i < showCount; i++ {
			act := &activities[i]
			fmt.Fprintf(&output, "- %s on `%s` (%s)\n", act.Kind, act.Target, act.Outcome)
		}
		output.WriteString("\n")
	}

	// Check for context focus
	if s.currentTaskFocus != "" {
		output.WriteString("## Context Focus\n\n")
		fmt.Fprintf(&output, "**Task:** %s\n", s.currentTaskFocus)
		if len(s.focusKeywords) > 0 {
			fmt.Fprintf(&output, "**Keywords:** %s\n", strings.Join(s.focusKeywords, ", "))
		}
		output.WriteString("\n")
	}

	// Check for accepted handoff
	handoffMu.RLock()
	var acceptedHandoff *Handoff
	for _, h := range handoffStore {
		if h.AcceptedBy == sessionID && h.Status == "accepted" {
			acceptedHandoff = h
			break
		}
	}
	handoffMu.RUnlock()

	if acceptedHandoff != nil {
		output.WriteString("## Active Handoff\n\n")
		fmt.Fprintf(&output, "**Handoff ID:** `%s`\n", acceptedHandoff.ID)
		fmt.Fprintf(&output, "**Task:** %s\n", acceptedHandoff.Task)
		if len(acceptedHandoff.PendingWork) > 0 {
			output.WriteString("**Pending Work:**\n")
			for _, item := range acceptedHandoff.PendingWork {
				fmt.Fprintf(&output, "- [ ] %s\n", item)
			}
		}
		output.WriteString("\n")
	}

	output.WriteString("---\n")
	output.WriteString("Session resumed. Continue where you left off.\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolSessionStatus gets the current session status and context.
func (s *MCPServer) toolSessionStatus(id any, args map[string]interface{}) jsonRPCResponse {
	var output strings.Builder
	output.WriteString("# Session Status\n\n")

	if s.currentSessionID == "" {
		output.WriteString("**No active session.**\n\n")
		output.WriteString("Use `session_init` to start a new session or `session_resume` to continue a previous one.\n")

		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: mcpToolResult{
				Content: []mcpContent{{Type: "text", Text: output.String()}},
			},
		}
	}

	mem := s.butler.Memory()
	session, _ := mem.GetSession(s.currentSessionID)

	if session != nil {
		output.WriteString("## Current Session\n\n")
		fmt.Fprintf(&output, "- **ID:** `%s`\n", session.ID)
		fmt.Fprintf(&output, "- **Agent:** %s\n", session.AgentType)
		if session.Goal != "" {
			fmt.Fprintf(&output, "- **Task:** %s\n", session.Goal)
		}
		fmt.Fprintf(&output, "- **State:** %s\n", session.State)
		fmt.Fprintf(&output, "- **Duration:** %s\n", formatDuration(time.Since(session.StartedAt)))
		output.WriteString("\n")
	}

	// Context focus
	if s.currentTaskFocus != "" {
		output.WriteString("## Context Focus\n\n")
		fmt.Fprintf(&output, "- **Task:** %s\n", s.currentTaskFocus)
		if len(s.focusKeywords) > 0 {
			fmt.Fprintf(&output, "- **Keywords:** %s\n", strings.Join(s.focusKeywords, ", "))
		}
		if len(s.contextPriorityUp) > 0 {
			fmt.Fprintf(&output, "- **Pinned Records:** %d\n", len(s.contextPriorityUp))
		}
		output.WriteString("\n")
	}

	// Tracked files
	if len(s.trackedFiles) > 0 {
		output.WriteString("## Tracked Files\n\n")
		count := 0
		for path := range s.trackedFiles {
			if count >= 5 {
				fmt.Fprintf(&output, "- ... and %d more\n", len(s.trackedFiles)-5)
				break
			}
			fmt.Fprintf(&output, "- `%s`\n", path)
			count++
		}
		output.WriteString("\n")
	}

	// Activity summary
	if session != nil {
		activities, _ := mem.GetActivities(s.currentSessionID, "", 50)
		if len(activities) > 0 {
			output.WriteString("## Activity Summary\n\n")
			successCount := 0
			failureCount := 0
			for i := range activities {
				if activities[i].Outcome == "success" {
					successCount++
				} else if activities[i].Outcome == "failure" {
					failureCount++
				}
			}
			fmt.Fprintf(&output, "- **Total:** %d activities\n", len(activities))
			fmt.Fprintf(&output, "- **Successful:** %d | **Failed:** %d\n", successCount, failureCount)
			output.WriteString("\n")
		}
	}

	// Active handoff
	handoffMu.RLock()
	var activeHandoff *Handoff
	for _, h := range handoffStore {
		if h.AcceptedBy == s.currentSessionID && h.Status == "accepted" {
			activeHandoff = h
			break
		}
	}
	handoffMu.RUnlock()

	if activeHandoff != nil {
		output.WriteString("## Active Handoff\n\n")
		fmt.Fprintf(&output, "- **ID:** `%s`\n", activeHandoff.ID)
		fmt.Fprintf(&output, "- **From:** %s\n", activeHandoff.FromAgent)
		fmt.Fprintf(&output, "- **Pending Items:** %d\n", len(activeHandoff.PendingWork))
		output.WriteString("\n")
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
