package butler

import (
	"fmt"
	"strings"
	"time"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/config"
)

// toolSessionInit is a composite tool that combines session_start + brief + explore_rooms.
// This is the recommended first call for any agent session.
//
func (s *MCPServer) toolSessionInit(id any, args map[string]interface{}) jsonRPCResponse {
	agentType, _ := args["agent_name"].(string)
	if agentType == "" {
		agentType, _ = args["agentType"].(string) // fallback to old param name
	}
	if agentType == "" {
		return s.toolError(id, "agent_name is required")
	}

	agentID, _ := args["agent_id"].(string)
	if agentID == "" {
		agentID, _ = args["agentId"].(string) // fallback
	}
	task, _ := args["task"].(string)
	if task == "" {
		task, _ = args["goal"].(string) // fallback
	}

	var output strings.Builder

	// 1. Start session
	session, err := s.butler.StartSession(agentType, agentID, task)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("start session failed: %v", err))
	}

	// Track this session in the server (for auto-session detection)
	s.currentSessionID = session.ID
	s.autoSessionUsed = false

	output.WriteString("# Session Initialized\n\n")
	output.WriteString("## Session Info\n\n")
	fmt.Fprintf(&output, "- **Session ID:** `%s`\n", session.ID)
	fmt.Fprintf(&output, "- **Agent:** %s\n", session.AgentType)
	if session.Goal != "" {
		fmt.Fprintf(&output, "- **Task:** %s\n", session.Goal)
	}
	fmt.Fprintf(&output, "- **Started:** %s\n\n", session.StartedAt.Format(time.RFC3339))

	// 1.5. Check for pending handoffs
	pendingHandoffs := getPendingHandoffsForAgent(agentType)
	if len(pendingHandoffs) > 0 {
		output.WriteString("## 📬 Pending Handoffs\n\n")
		urgentCount := 0
		highCount := 0
		for _, h := range pendingHandoffs {
			priorityIcon := "🔵"
			switch h.Priority {
			case "high":
				priorityIcon = "🟠"
				highCount++
			case "urgent":
				priorityIcon = "🔴"
				urgentCount++
			}
			fmt.Fprintf(&output, "- %s `%s`: %s\n", priorityIcon, h.ID, truncateString(h.Task, 60))
			fmt.Fprintf(&output, "  From: %s | Created: %s\n", h.FromAgent, h.CreatedAt.Format("Jan 2 15:04"))
		}
		if urgentCount > 0 {
			fmt.Fprintf(&output, "\n⚠️ **%d urgent handoff(s)** waiting for attention!\n", urgentCount)
		} else if highCount > 0 {
			fmt.Fprintf(&output, "\n⚡ **%d high-priority handoff(s)** available.\n", highCount)
		}
		output.WriteString("\nUse `handoff_accept({id: '...'})` to take over a task.\n\n")
	}

	// 2. Get briefing
	brief, err := s.butler.GetBrief("")
	if err == nil {
		output.WriteString("## Workspace Briefing\n\n")

		// Active agents (exclude self)
		activeOthers := 0
		for i := range brief.ActiveAgents {
			a := &brief.ActiveAgents[i]
			if a.SessionID != session.ID {
				activeOthers++
			}
		}
		if activeOthers > 0 {
			output.WriteString("### Active Agents\n\n")
			for i := range brief.ActiveAgents {
				a := &brief.ActiveAgents[i]
				if a.SessionID == session.ID {
					continue // Skip self
				}
				currentFile := ""
				if a.CurrentFile != "" {
					currentFile = fmt.Sprintf(" working on `%s`", a.CurrentFile)
				}
				sessionShort := a.SessionID
				if len(sessionShort) > 12 {
					sessionShort = sessionShort[:12]
				}
				fmt.Fprintf(&output, "- **%s** (session: `%s`)%s\n", a.AgentType, sessionShort, currentFile)
			}
			output.WriteString("\n")
		}

		// Relevant learnings
		if len(brief.Learnings) > 0 {
			output.WriteString("### Key Learnings\n\n")
			count := minInt(5, len(brief.Learnings))
			for i := 0; i < count; i++ {
				l := &brief.Learnings[i]
				fmt.Fprintf(&output, "- [%.0f%%] %s\n", l.Confidence*100, l.Content)
			}
			if len(brief.Learnings) > 5 {
				fmt.Fprintf(&output, "- ... and %d more\n", len(brief.Learnings)-5)
			}
			output.WriteString("\n")
		}

		// Hotspots
		if len(brief.Hotspots) > 0 {
			output.WriteString("### Hotspots (Frequently Edited)\n\n")
			count := minInt(5, len(brief.Hotspots))
			for i := 0; i < count; i++ {
				h := &brief.Hotspots[i]
				warning := ""
				if h.FailureCount > 0 {
					warning = fmt.Sprintf(" ⚠️ %d failures", h.FailureCount)
				}
				fmt.Fprintf(&output, "- `%s` (%d edits)%s\n", h.Path, h.EditCount, warning)
			}
			output.WriteString("\n")
		}
	}

	// 3. List rooms
	rooms := s.butler.ListRooms()
	if len(rooms) > 0 {
		output.WriteString("## Project Structure (Rooms)\n\n")
		for i := range rooms {
			room := &rooms[i]
			fmt.Fprintf(&output, "### %s\n", room.Name)
			fmt.Fprintf(&output, "%s\n", room.Summary)
			if len(room.EntryPoints) > 0 {
				output.WriteString("Entry points: ")
				eps := make([]string, 0, minInt(3, len(room.EntryPoints)))
				for j := 0; j < minInt(3, len(room.EntryPoints)); j++ {
					eps = append(eps, fmt.Sprintf("`%s`", room.EntryPoints[j]))
				}
				output.WriteString(strings.Join(eps, ", "))
				if len(room.EntryPoints) > 3 {
					fmt.Fprintf(&output, " +%d more", len(room.EntryPoints)-3)
				}
				output.WriteString("\n")
			}
			output.WriteString("\n")
		}
	}

	// Add next steps
	output.WriteString("---\n\n")
	output.WriteString("## Next Steps\n\n")
	nextSteps := []string{
		fmt.Sprintf("Use session ID `%s` for all subsequent calls", session.ID),
		"Call `explore({intent: '...'})` to find relevant code",
		"Call `file_context({file_path: '...'})` before editing any file",
		"Call `store({content: '...', as: 'learning|decision|idea'})` to save knowledge",
		fmt.Sprintf("Call `session_end({sessionId: '%s', outcome: '...'})` when done", session.ID),
	}
	for _, step := range nextSteps {
		fmt.Fprintf(&output, "1. %s\n", step)
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolFileContext is a composite tool that combines context_auto_inject + session_conflict.
// This should be called before editing any file.
//
func (s *MCPServer) toolFileContext(id any, args map[string]interface{}) jsonRPCResponse {
	filePath, _ := args["file_path"].(string)
	if filePath == "" {
		filePath, _ = args["filePath"].(string) // fallback
	}
	if filePath == "" {
		return s.toolError(id, "file_path is required")
	}

	sessionID, _ := args["session_id"].(string)
	if sessionID == "" {
		sessionID, _ = args["sessionId"].(string) // fallback
	}

	var output strings.Builder

	output.WriteString(fmt.Sprintf("# File Context: `%s`\n\n", filePath))

	// 1. Check for conflicts using butler method
	hasConflict := false
	if sessionID != "" {
		conflict, err := s.butler.CheckConflict(sessionID, filePath)
		if err == nil && conflict != nil {
			hasConflict = true
			output.WriteString("## ⚠️ Conflict Warning\n\n")
			fmt.Fprintf(&output, "Another agent (**%s**) is working on this file!\n\n", conflict.OtherAgent)
			fmt.Fprintf(&output, "- Session: `%s`\n", conflict.OtherSession)
			fmt.Fprintf(&output, "- Last activity: %s\n", conflict.LastTouched.Format("15:04:05"))
			fmt.Fprintf(&output, "- Severity: **%s**\n\n", conflict.Severity)
			output.WriteString("**Recommendation:** Coordinate with the other agent or wait.\n\n")
		}
	}

	// 2. Get auto-injection context
	cfg := s.butler.Config()
	var autoInjectCfg *config.AutoInjectionConfig
	if cfg != nil && cfg.AutoInjection != nil {
		autoInjectCfg = cfg.AutoInjection
	} else {
		autoInjectCfg = config.DefaultAutoInjectionConfig()
	}

	ctx, err := s.butler.GetAutoInjectionContext(filePath, autoInjectCfg)
	if err != nil {
		output.WriteString("## Context\n\n")
		output.WriteString("*No specific context available for this file.*\n\n")
	} else {
		// Learnings
		if len(ctx.Learnings) > 0 {
			output.WriteString("## Learnings\n\n")
			for i := range ctx.Learnings {
				pl := &ctx.Learnings[i]
				l := &pl.Learning
				scopeTag := ""
				switch l.Scope {
				case "file":
					scopeTag = " 📍"
				case "room":
					scopeTag = " 🏠"
				}
				fmt.Fprintf(&output, "- [%.0f%%]%s %s\n", l.Confidence*100, scopeTag, l.Content)
				if pl.Reason != "" {
					fmt.Fprintf(&output, "  *Why: %s*\n", pl.Reason)
				}
			}
			output.WriteString("\n")
		}

		// Decisions
		if len(ctx.Decisions) > 0 {
			output.WriteString("## Active Decisions\n\n")
			for i := range ctx.Decisions {
				d := &ctx.Decisions[i]
				fmt.Fprintf(&output, "- **%s**: %s\n", d.ID, d.Content)
				if d.Rationale != "" {
					fmt.Fprintf(&output, "  *Rationale: %s*\n", d.Rationale)
				}
			}
			output.WriteString("\n")
		}

		// Failures
		if len(ctx.Failures) > 0 {
			output.WriteString("## ⚠️ Known Failures\n\n")
			for i := range ctx.Failures {
				f := &ctx.Failures[i]
				fmt.Fprintf(&output, "- **%s** [%s]: %s\n", f.Path, f.Severity, f.LastFailure)
			}
			output.WriteString("\n")
		}

		// Warnings
		if len(ctx.Warnings) > 0 {
			output.WriteString("## Warnings\n\n")
			for i := range ctx.Warnings {
				w := &ctx.Warnings[i]
				fmt.Fprintf(&output, "- %s\n", w.Message)
			}
			output.WriteString("\n")
		}
	}

	// 3. Get file intel
	mem := s.butler.Memory()
	if mem != nil {
		intel, err := mem.GetFileIntel(filePath)
		if err == nil && intel != nil && intel.EditCount > 0 {
			output.WriteString("## File History\n\n")
			fmt.Fprintf(&output, "- **Total edits:** %d\n", intel.EditCount)
			fmt.Fprintf(&output, "- **Failures:** %d\n", intel.FailureCount)
			if intel.EditCount > 0 {
				failureRate := float64(intel.FailureCount) / float64(intel.EditCount) * 100
				if failureRate > 20 {
					fmt.Fprintf(&output, "- ⚠️ **High failure rate:** %.0f%% - proceed with caution\n", failureRate)
				}
			}
			output.WriteString("\n")
		}
	}

	// Add next steps
	output.WriteString("---\n\n")
	output.WriteString("## Next Steps\n\n")
	var nextSteps []string
	if hasConflict {
		nextSteps = []string{
			"Wait for the other agent to finish, OR",
			"Coordinate via `session_log` to avoid conflicts",
			"If proceeding, make changes quickly and call `session_log` immediately after",
		}
	} else {
		nextSteps = []string{
			"You are clear to edit this file",
			"After editing, call `session_log({activity: 'file_edit', path: '...', description: '...'})`",
			"If you learn something, call `store({content: '...', as: 'learning'})`",
		}
	}
	for _, step := range nextSteps {
		fmt.Fprintf(&output, "- %s\n", step)
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// minInt is defined in fuzzy.go
