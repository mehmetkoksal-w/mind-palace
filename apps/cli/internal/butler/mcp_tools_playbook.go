package butler

import (
	"fmt"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/playbook"
)

// ============================================================
// PLAYBOOK TOOL - Guided task execution with rooms and steps
// ============================================================

// playbookExecutor caches the playbook executor instance.
var playbookState *playbook.ExecutionState

// dispatchPlaybook handles the playbook tool with action parameter.
func (s *MCPServer) dispatchPlaybook(id any, args map[string]interface{}, action string) jsonRPCResponse {
	if action == "" {
		action = "list" // default action
	}

	switch action {
	case "list":
		return s.toolPlaybookList(id)
	case "show":
		return s.toolPlaybookShow(id, args)
	case "start":
		return s.toolPlaybookStart(id, args)
	case "status":
		return s.toolPlaybookStatus(id)
	case "guidance":
		return s.toolPlaybookGuidance(id)
	case "advance":
		return s.toolPlaybookAdvance(id)
	case "evidence":
		return s.toolPlaybookEvidence(id, args)
	case "verify":
		return s.toolPlaybookVerify(id)
	case "complete":
		return s.toolPlaybookComplete(id)
	default:
		return consolidatedToolError(id, "playbook", "action", action)
	}
}

// getPlaybookExecutor returns the playbook executor.
func (s *MCPServer) getPlaybookExecutor() *playbook.Executor {
	rooms := s.butler.ListRooms()
	return playbook.NewExecutor(s.butler.root, rooms)
}

// toolPlaybookList lists all available playbooks.
func (s *MCPServer) toolPlaybookList(id any) jsonRPCResponse {
	executor := s.getPlaybookExecutor()
	playbooks, err := executor.ListPlaybooks()
	if err != nil {
		return s.toolError(id, fmt.Sprintf("list playbooks: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Available Playbooks\n\n")

	if len(playbooks) == 0 {
		output.WriteString("No playbooks found. Create playbooks in `.palace/playbooks/`.\n\n")
		output.WriteString("Example playbook structure:\n```json\n")
		output.WriteString(`{
  "schemaVersion": "1.0.0",
  "kind": "palace/playbook",
  "name": "my-playbook",
  "summary": "Description of the playbook",
  "rooms": ["room1", "room2"]
}`)
		output.WriteString("\n```\n")
	} else {
		output.WriteString("| Playbook | Summary | Rooms |\n")
		output.WriteString("|----------|---------|-------|\n")
		for _, pb := range playbooks {
			rooms := strings.Join(pb.Rooms, ", ")
			if len(rooms) > 30 {
				rooms = rooms[:27] + "..."
			}
			fmt.Fprintf(&output, "| %s | %s | %s |\n",
				pb.Name,
				truncateSnippet(pb.Summary, 40),
				rooms)
		}
		fmt.Fprintf(&output, "\n**Total:** %d playbooks\n", len(playbooks))
		output.WriteString("\nUse `playbook` with `action=start` and `name=<playbook>` to begin execution.\n")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolPlaybookShow shows details for a specific playbook.
func (s *MCPServer) toolPlaybookShow(id any, args map[string]interface{}) jsonRPCResponse {
	name := getStringArg(args, "name", "")
	if name == "" {
		return s.toolError(id, "playbook name is required")
	}

	executor := s.getPlaybookExecutor()
	pb, err := executor.LoadPlaybook(name)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("load playbook: %v", err))
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# Playbook: %s\n\n", pb.Name)
	fmt.Fprintf(&output, "**Summary:** %s\n\n", pb.Summary)

	output.WriteString("## Rooms\n")
	for i, roomName := range pb.Rooms {
		fmt.Fprintf(&output, "%d. %s\n", i+1, roomName)
	}

	if len(pb.RequiredEvidence) > 0 {
		output.WriteString("\n## Required Evidence\n")
		for _, ev := range pb.RequiredEvidence {
			fmt.Fprintf(&output, "- **%s**: %s", ev.ID, ev.Description)
			if ev.Room != "" {
				fmt.Fprintf(&output, " (from: %s)", ev.Room)
			}
			output.WriteString("\n")
		}
	}

	if len(pb.Verification) > 0 {
		output.WriteString("\n## Verification Checks\n")
		for _, v := range pb.Verification {
			fmt.Fprintf(&output, "- **%s**: %s", v.Name, v.Expectation)
			if v.Capability != "" {
				fmt.Fprintf(&output, " (%s)", v.Capability)
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

// toolPlaybookStart starts playbook execution.
func (s *MCPServer) toolPlaybookStart(id any, args map[string]interface{}) jsonRPCResponse {
	name := getStringArg(args, "name", "")
	if name == "" {
		return s.toolError(id, "playbook name is required")
	}

	executor := s.getPlaybookExecutor()

	// Check for existing execution
	if playbookState != nil && playbookState.Status == "running" {
		return s.toolError(id, fmt.Sprintf("playbook '%s' is already running. Use action=status or action=complete first.", playbookState.PlaybookName))
	}

	state, err := executor.Start(name)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("start playbook: %v", err))
	}

	// Save state
	playbookState = state
	if err := executor.SaveState(state); err != nil {
		// Non-fatal, log but continue
	}

	var output strings.Builder
	fmt.Fprintf(&output, "# ✅ Playbook Started: %s\n\n", name)
	fmt.Fprintf(&output, "**Summary:** %s\n\n", state.Playbook.Summary)
	fmt.Fprintf(&output, "**Rooms to visit:** %d\n", len(state.Playbook.Rooms))
	fmt.Fprintf(&output, "**Status:** %s\n\n", state.Status)

	// Show first step guidance
	guidance, err := executor.GetCurrentGuidance(state)
	if err == nil && guidance != nil {
		output.WriteString("## Current Step\n")
		fmt.Fprintf(&output, "**Room:** %s\n", guidance.RoomName)
		fmt.Fprintf(&output, "**Step %d/%d:** %s\n", guidance.StepNumber, guidance.TotalSteps, guidance.StepName)
		if guidance.StepDescription != "" {
			fmt.Fprintf(&output, "**Description:** %s\n", guidance.StepDescription)
		}
		if len(guidance.EntryPoints) > 0 {
			output.WriteString("\n**Entry Points:**\n")
			for _, ep := range guidance.EntryPoints {
				fmt.Fprintf(&output, "- `%s`\n", ep)
			}
		}
		output.WriteString("\n**Next:** " + guidance.NextAction + "\n")
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolPlaybookStatus shows current execution status.
func (s *MCPServer) toolPlaybookStatus(id any) jsonRPCResponse {
	if playbookState == nil {
		// Try to load from disk
		executor := s.getPlaybookExecutor()
		state, err := executor.LoadState()
		if err != nil {
			return s.toolError(id, "no playbook execution in progress. Use action=start to begin.")
		}
		playbookState = state
	}

	executor := s.getPlaybookExecutor()
	progress := executor.GetProgress(playbookState)

	var output strings.Builder
	fmt.Fprintf(&output, "# Playbook Status: %s\n\n", playbookState.PlaybookName)
	fmt.Fprintf(&output, "**Status:** %s\n", playbookState.Status)
	fmt.Fprintf(&output, "**Progress:** %.1f%%\n", progress)
	fmt.Fprintf(&output, "**Started:** %s\n", playbookState.StartedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&output, "**Updated:** %s\n\n", playbookState.UpdatedAt.Format("2006-01-02 15:04:05"))

	output.WriteString("## Rooms Progress\n")
	for i, roomName := range playbookState.Playbook.Rooms {
		status := "⬜"
		if i < playbookState.CurrentRoomIdx {
			status = "✅"
		} else if i == playbookState.CurrentRoomIdx && playbookState.Status == "running" {
			status = "▶️"
		}
		fmt.Fprintf(&output, "%s %s\n", status, roomName)
	}

	if playbookState.Status == "completed" {
		output.WriteString("\n## Evidence Collected\n")
		if len(playbookState.Evidence) > 0 {
			for k := range playbookState.Evidence {
				fmt.Fprintf(&output, "- %s ✅\n", k)
			}
		} else {
			output.WriteString("No evidence collected.\n")
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

// toolPlaybookGuidance shows guidance for the current step.
func (s *MCPServer) toolPlaybookGuidance(id any) jsonRPCResponse {
	if playbookState == nil || playbookState.Status != "running" {
		return s.toolError(id, "no playbook running. Use action=start first.")
	}

	executor := s.getPlaybookExecutor()
	guidance, err := executor.GetCurrentGuidance(playbookState)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get guidance: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Current Step Guidance\n\n")
	fmt.Fprintf(&output, "**Room:** %s\n", guidance.RoomName)
	fmt.Fprintf(&output, "**Room Summary:** %s\n\n", guidance.RoomSummary)
	fmt.Fprintf(&output, "## Step %d of %d: %s\n\n", guidance.StepNumber, guidance.TotalSteps, guidance.StepName)

	if guidance.StepDescription != "" {
		fmt.Fprintf(&output, "**Description:** %s\n\n", guidance.StepDescription)
	}

	if guidance.Capability != "" {
		fmt.Fprintf(&output, "**Capability:** `%s`\n", guidance.Capability)
	}

	if guidance.EvidenceID != "" {
		fmt.Fprintf(&output, "**Evidence to collect:** `%s`\n", guidance.EvidenceID)
	}

	if len(guidance.EntryPoints) > 0 {
		output.WriteString("\n**Entry Points to review:**\n")
		for _, ep := range guidance.EntryPoints {
			fmt.Fprintf(&output, "- `%s`\n", ep)
		}
	}

	output.WriteString("\n---\n**Next Action:** " + guidance.NextAction + "\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolPlaybookAdvance advances to the next step.
func (s *MCPServer) toolPlaybookAdvance(id any) jsonRPCResponse {
	if playbookState == nil || playbookState.Status != "running" {
		return s.toolError(id, "no playbook running. Use action=start first.")
	}

	executor := s.getPlaybookExecutor()
	prevRoom := ""
	if playbookState.CurrentRoom != nil {
		prevRoom = playbookState.CurrentRoom.Name
	}
	prevStep := playbookState.CurrentStepIdx

	if err := executor.AdvanceStep(playbookState); err != nil {
		return s.toolError(id, fmt.Sprintf("advance: %v", err))
	}

	// Save state
	_ = executor.SaveState(playbookState)

	var output strings.Builder

	if playbookState.Status == "completed" {
		output.WriteString("# 🎉 Playbook Completed!\n\n")
		fmt.Fprintf(&output, "**Playbook:** %s\n", playbookState.PlaybookName)
		fmt.Fprintf(&output, "**Rooms completed:** %d\n\n", len(playbookState.CompletedRooms))

		output.WriteString("Use `action=verify` to run verification checks.\n")
	} else {
		output.WriteString("# ✅ Step Advanced\n\n")

		currentRoom := ""
		if playbookState.CurrentRoom != nil {
			currentRoom = playbookState.CurrentRoom.Name
		}

		if prevRoom != currentRoom {
			fmt.Fprintf(&output, "**Room completed:** %s\n", prevRoom)
			fmt.Fprintf(&output, "**Now in room:** %s\n\n", currentRoom)
		} else {
			fmt.Fprintf(&output, "**Step %d completed** in room `%s`\n\n", prevStep+1, prevRoom)
		}

		// Show next step guidance
		guidance, err := executor.GetCurrentGuidance(playbookState)
		if err == nil && guidance != nil {
			output.WriteString("## Next Step\n")
			fmt.Fprintf(&output, "**Step %d/%d:** %s\n", guidance.StepNumber, guidance.TotalSteps, guidance.StepName)
			if guidance.StepDescription != "" {
				fmt.Fprintf(&output, "**Description:** %s\n", guidance.StepDescription)
			}
			output.WriteString("\n**Next:** " + guidance.NextAction + "\n")
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

// toolPlaybookEvidence records evidence.
func (s *MCPServer) toolPlaybookEvidence(id any, args map[string]interface{}) jsonRPCResponse {
	if playbookState == nil {
		return s.toolError(id, "no playbook execution in progress")
	}

	evidenceID := getStringArg(args, "evidence_id", "")
	if evidenceID == "" {
		return s.toolError(id, "evidence_id is required")
	}

	data := args["data"]
	if data == nil {
		data = getStringArg(args, "value", "collected")
	}

	executor := s.getPlaybookExecutor()
	if err := executor.CollectEvidence(playbookState, evidenceID, data); err != nil {
		return s.toolError(id, fmt.Sprintf("collect evidence: %v", err))
	}

	_ = executor.SaveState(playbookState)

	var output strings.Builder
	fmt.Fprintf(&output, "# ✅ Evidence Collected: %s\n\n", evidenceID)
	fmt.Fprintf(&output, "**Total evidence collected:** %d\n", len(playbookState.Evidence))

	// Show which required evidence is still missing
	if len(playbookState.Playbook.RequiredEvidence) > 0 {
		output.WriteString("\n## Required Evidence Status\n")
		for _, req := range playbookState.Playbook.RequiredEvidence {
			status := "⬜"
			if _, ok := playbookState.Evidence[req.ID]; ok {
				status = "✅"
			}
			fmt.Fprintf(&output, "%s %s: %s\n", status, req.ID, req.Description)
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

// toolPlaybookVerify runs verification checks.
func (s *MCPServer) toolPlaybookVerify(id any) jsonRPCResponse {
	if playbookState == nil {
		return s.toolError(id, "no playbook execution to verify")
	}

	executor := s.getPlaybookExecutor()
	results, err := executor.RunVerification(playbookState)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("verify: %v", err))
	}

	var output strings.Builder
	output.WriteString("# Verification Results\n\n")

	if len(results) == 0 {
		output.WriteString("No verification checks defined for this playbook.\n")
	} else {
		output.WriteString("| Check | Status | Detail |\n")
		output.WriteString("|-------|--------|--------|\n")
		for _, r := range results {
			statusIcon := "⬜"
			switch r.Status {
			case "passed":
				statusIcon = "✅"
			case "failed":
				statusIcon = "❌"
			case "skipped":
				statusIcon = "⏭️"
			}
			detail := r.Detail
			if r.Error != "" {
				detail = r.Error
			}
			fmt.Fprintf(&output, "| %s | %s %s | %s |\n", r.Name, statusIcon, r.Status, detail)
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

// toolPlaybookComplete marks the playbook as complete and clears state.
func (s *MCPServer) toolPlaybookComplete(id any) jsonRPCResponse {
	if playbookState == nil {
		return s.toolError(id, "no playbook execution to complete")
	}

	executor := s.getPlaybookExecutor()
	playbookName := playbookState.PlaybookName
	progress := executor.GetProgress(playbookState)
	evidenceCount := len(playbookState.Evidence)

	// Clear state
	_ = executor.ClearState()
	playbookState = nil

	var output strings.Builder
	output.WriteString("# Playbook Execution Completed\n\n")
	fmt.Fprintf(&output, "**Playbook:** %s\n", playbookName)
	fmt.Fprintf(&output, "**Final Progress:** %.1f%%\n", progress)
	fmt.Fprintf(&output, "**Evidence Collected:** %d items\n\n", evidenceCount)
	output.WriteString("Execution state has been cleared. You can start a new playbook.\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}
