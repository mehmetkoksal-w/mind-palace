package butler

import (
	"fmt"
)

// useConsolidatedTools controls whether to use the consolidated tool list (23 tools)
// or the legacy tool list (89 tools). Set to true to enable consolidation.
var useConsolidatedTools = true

// getToolsList returns the appropriate tools list based on configuration.
func getToolsList() []mcpTool {
	if useConsolidatedTools {
		return buildConsolidatedToolsList()
	}
	return buildToolsList()
}

// adminOnlyToolsConsolidated lists consolidated tools that are only available in human mode.
var adminOnlyToolsConsolidated = map[string]bool{
	"govern": true, // approve/reject proposals
}

// adminOnlyActionsConsolidated lists tool actions that require human mode.
var adminOnlyActionsConsolidated = map[string]map[string]bool{
	"store": {
		"direct": true, // store with direct=true
	},
	"recall": {
		"outcome":  true, // mark decision outcomes
		"link":     true, // create links
		"unlink":   true, // remove links
		"obsolete": true, // mark learnings obsolete
		"archive":  true, // archive learnings
	},
}

// IsAdminOnlyToolConsolidated returns true if the tool requires human mode.
func IsAdminOnlyToolConsolidated(toolName string) bool {
	if useConsolidatedTools {
		return adminOnlyToolsConsolidated[toolName]
	}
	return adminOnlyTools[toolName]
}

// IsAdminOnlyActionConsolidated checks if a specific action within a tool requires human mode.
func IsAdminOnlyActionConsolidated(toolName string, args map[string]interface{}) bool {
	if !useConsolidatedTools {
		return false
	}

	// Check for direct=true on store
	if toolName == "store" {
		if direct, ok := args["direct"].(bool); ok && direct {
			return true
		}
	}

	// Check for admin-only actions
	if actions, ok := adminOnlyActionsConsolidated[toolName]; ok {
		if action, ok := args["action"].(string); ok {
			return actions[action]
		}
	}

	return false
}

// dispatchConsolidatedTool routes consolidated tool calls with action-based dispatching.
func (s *MCPServer) dispatchConsolidatedTool(id any, params mcpToolCallParams) jsonRPCResponse {
	args := params.Arguments
	action := getStringArg(args, "action", "")

	switch params.Name {
	// ============================================================
	// COMPOSITE TOOLS (kept as-is)
	// ============================================================
	case "session_init":
		return s.toolSessionInit(id, args)
	case "file_context":
		return s.toolFileContext(id, args)

	// ============================================================
	// EXPLORE - consolidated with action parameter
	// ============================================================
	case "explore":
		return s.dispatchExplore(id, args, action)

	// ============================================================
	// STORE - consolidated with type/direct parameters
	// ============================================================
	case "store":
		if direct, ok := args["direct"].(bool); ok && direct {
			return s.toolStoreDirect(id, args)
		}
		return s.toolStore(id, args)

	// ============================================================
	// RECALL - consolidated with action/type parameters
	// ============================================================
	case "recall":
		return s.dispatchRecall(id, args, action)

	// ============================================================
	// BRIEF - consolidated with mode parameter
	// ============================================================
	case "brief":
		return s.dispatchBrief(id, args)

	// ============================================================
	// SESSION - consolidated with action parameter
	// ============================================================
	case "session":
		return s.dispatchSession(id, args, action)

	// ============================================================
	// SEARCH - consolidated with mode parameter
	// ============================================================
	case "search":
		return s.dispatchSearch(id, args)

	// ============================================================
	// EMBEDDING - consolidated with action parameter
	// ============================================================
	case "embedding":
		switch action {
		case "sync":
			return s.toolEmbeddingSync(id, args)
		case "stats":
			return s.toolEmbeddingStats(id, args)
		default:
			return consolidatedToolError(id, "embedding", "action", action)
		}

	// ============================================================
	// CONTEXT - consolidated with action parameter
	// ============================================================
	case "context":
		return s.dispatchContext(id, args, action)

	// ============================================================
	// CONTRADICTION - consolidated with action parameter
	// ============================================================
	case "contradiction":
		switch action {
		case "find":
			return s.toolRecallContradictions(id, args)
		case "check":
			return s.toolRecallContradictionCheck(id, args)
		case "summary":
			return s.toolRecallContradictionSummary(id, args)
		default:
			return consolidatedToolError(id, "contradiction", "action", action)
		}

	// ============================================================
	// DECAY - consolidated with action parameter
	// ============================================================
	case "decay":
		switch action {
		case "stats":
			return s.toolDecayStats(id, args)
		case "preview":
			return s.toolDecayPreview(id, args)
		case "apply":
			return s.toolDecayApply(id, args)
		case "reinforce":
			return s.toolDecayReinforce(id, args)
		case "boost":
			return s.toolDecayBoost(id, args)
		default:
			return consolidatedToolError(id, "decay", "action", action)
		}

	// ============================================================
	// POSTMORTEM - consolidated with action parameter
	// ============================================================
	case "postmortem":
		return s.dispatchPostmortem(id, args, action)

	// ============================================================
	// HANDOFF - consolidated with action parameter
	// ============================================================
	case "handoff":
		switch action {
		case "create":
			return s.toolHandoffCreate(id, args)
		case "list":
			return s.toolHandoffList(id, args)
		case "accept":
			return s.toolHandoffAccept(id, args)
		case "complete":
			return s.toolHandoffComplete(id, args)
		default:
			return consolidatedToolError(id, "handoff", "action", action)
		}

	// ============================================================
	// CONVERSATION - consolidated with action parameter
	// ============================================================
	case "conversation":
		switch action {
		case "store":
			return s.toolConversationStore(id, args)
		case "search":
			return s.toolConversationSearch(id, args)
		case "extract":
			return s.toolConversationExtract(id, args)
		default:
			return consolidatedToolError(id, "conversation", "action", action)
		}

	// ============================================================
	// CORRIDOR - consolidated with action parameter
	// ============================================================
	case "corridor":
		switch action {
		case "learnings":
			return s.toolCorridorLearnings(id, args)
		case "links":
			return s.toolCorridorLinks(id, args)
		case "stats":
			return s.toolCorridorStats(id, args)
		case "promote":
			return s.toolCorridorPromote(id, args)
		case "reinforce":
			return s.toolCorridorReinforce(id, args)
		default:
			return consolidatedToolError(id, "corridor", "action", action)
		}

	// ============================================================
	// PATTERN - consolidated with action parameter
	// ============================================================
	case "pattern":
		switch action {
		case "list":
			return s.toolPatternsGet(id, args)
		case "show":
			return s.toolPatternShow(id, args)
		case "approve":
			return s.toolPatternApprove(id, args)
		case "ignore":
			return s.toolPatternIgnore(id, args)
		case "stats":
			return s.toolPatternStats(id, args)
		default:
			return consolidatedToolError(id, "pattern", "action", action)
		}

	// ============================================================
	// CONTRACT - consolidated with action parameter
	// ============================================================
	case "contract":
		switch action {
		case "list":
			return s.toolContractsGet(id, args)
		case "show":
			return s.toolContractShow(id, args)
		case "verify":
			return s.toolContractVerify(id, args)
		case "ignore":
			return s.toolContractIgnore(id, args)
		case "stats":
			return s.toolContractStats(id, args)
		case "mismatches":
			return s.toolContractMismatches(id, args)
		default:
			return consolidatedToolError(id, "contract", "action", action)
		}

	// ============================================================
	// ANALYTICS - consolidated with type parameter
	// ============================================================
	case "analytics":
		analyticsType := getStringArg(args, "type", "health")
		switch analyticsType {
		case "sessions":
			return s.toolSessionAnalytics(id, args)
		case "learnings":
			return s.toolLearningEffectiveness(id, args)
		case "health":
			return s.toolWorkspaceHealth(id, args)
		default:
			return consolidatedToolError(id, "analytics", "type", analyticsType)
		}

	// ============================================================
	// GOVERN - consolidated with action parameter
	// ============================================================
	case "govern":
		switch action {
		case "approve":
			return s.toolApprove(id, args)
		case "reject":
			return s.toolReject(id, args)
		case "list":
			return s.toolListProposals(id, args)
		default:
			return consolidatedToolError(id, "govern", "action", action)
		}

	// ============================================================
	// ROUTE - kept as separate tool
	// ============================================================
	case "route":
		return s.toolGetRoute(id, args)

	default:
		return jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &rpcError{Code: -32602, Message: fmt.Sprintf("Unknown tool: %s", params.Name)},
		}
	}
}

// dispatchExplore handles the consolidated explore tool with action parameter.
func (s *MCPServer) dispatchExplore(id any, args map[string]interface{}, action string) jsonRPCResponse {
	if action == "" {
		action = "search" // default action
	}

	switch action {
	case "search":
		return s.toolExplore(id, args)
	case "rooms":
		return s.toolExploreRooms(id)
	case "context":
		return s.toolExploreContext(id, args)
	case "impact":
		return s.toolExploreImpact(id, args)
	case "symbols":
		return s.toolExploreSymbols(id, args)
	case "symbol":
		return s.toolExploreSymbol(id, args)
	case "file":
		return s.toolExploreFile(id, args)
	case "deps":
		return s.toolExploreDeps(id, args)
	case "callers":
		return s.toolExploreCallers(id, args)
	case "callees":
		return s.toolExploreCallees(id, args)
	case "graph":
		return s.toolExploreGraph(id, args)
	default:
		return consolidatedToolError(id, "explore", "action", action)
	}
}

// dispatchRecall handles the consolidated recall tool with action/type parameters.
func (s *MCPServer) dispatchRecall(id any, args map[string]interface{}, action string) jsonRPCResponse {
	if action == "" {
		action = "get" // default action
	}

	switch action {
	case "get":
		recallType := getStringArg(args, "type", "learnings")
		switch recallType {
		case "learnings":
			return s.toolRecall(id, args)
		case "decisions":
			return s.toolRecallDecisions(id, args)
		case "ideas":
			return s.toolRecallIdeas(id, args)
		default:
			return consolidatedToolError(id, "recall", "type", recallType)
		}
	case "links":
		return s.toolRecallLinks(id, args)
	case "link":
		// Check if it's a learning link (has decision_id and learning_id)
		if _, hasDecision := args["decision_id"]; hasDecision {
			if _, hasLearning := args["learning_id"]; hasLearning {
				return s.toolRecallLearningLink(id, args)
			}
		}
		return s.toolRecallLink(id, args)
	case "unlink":
		return s.toolRecallUnlink(id, args)
	case "outcome":
		return s.toolRecallOutcome(id, args)
	case "obsolete":
		return s.toolRecallObsolete(id, args)
	case "archive":
		return s.toolRecallArchive(id, args)
	default:
		return consolidatedToolError(id, "recall", "action", action)
	}
}

// dispatchBrief handles the consolidated brief tool with mode parameter.
func (s *MCPServer) dispatchBrief(id any, args map[string]interface{}) jsonRPCResponse {
	mode := getStringArg(args, "mode", "workspace")

	switch mode {
	case "workspace":
		return s.toolBrief(id, args)
	case "file":
		return s.toolBriefFile(id, args)
	case "smart":
		return s.toolBriefingSmart(id, args)
	default:
		return consolidatedToolError(id, "brief", "mode", mode)
	}
}

// dispatchSession handles the consolidated session tool with action parameter.
func (s *MCPServer) dispatchSession(id any, args map[string]interface{}, action string) jsonRPCResponse {
	switch action {
	case "start":
		return s.toolSessionStart(id, args)
	case "end":
		return s.toolSessionEnd(id, args)
	case "log":
		return s.toolSessionLog(id, args)
	case "conflict":
		return s.toolSessionConflict(id, args)
	case "list":
		return s.toolSessionList(id, args)
	case "resume":
		return s.toolSessionResume(id, args)
	case "status":
		return s.toolSessionStatus(id, args)
	default:
		return consolidatedToolError(id, "session", "action", action)
	}
}

// dispatchSearch handles the consolidated search tool with mode parameter.
func (s *MCPServer) dispatchSearch(id any, args map[string]interface{}) jsonRPCResponse {
	mode := getStringArg(args, "mode", "hybrid")

	switch mode {
	case "semantic":
		return s.toolSearchSemantic(id, args)
	case "hybrid":
		return s.toolSearchHybrid(id, args)
	case "similar":
		return s.toolSearchSimilar(id, args)
	default:
		return consolidatedToolError(id, "search", "mode", mode)
	}
}

// dispatchContext handles the consolidated context tool with action parameter.
func (s *MCPServer) dispatchContext(id any, args map[string]interface{}, action string) jsonRPCResponse {
	switch action {
	case "inject":
		return s.toolContextAutoInject(id, args)
	case "explain":
		return s.toolScopeExplain(id, args)
	case "focus":
		return s.toolContextFocus(id, args)
	case "get":
		return s.toolContextGet(id, args)
	case "pin":
		return s.toolContextPin(id, args)
	default:
		return consolidatedToolError(id, "context", "action", action)
	}
}

// dispatchPostmortem handles the consolidated postmortem tool with action parameter.
func (s *MCPServer) dispatchPostmortem(id any, args map[string]interface{}, action string) jsonRPCResponse {
	switch action {
	case "create":
		return s.toolStorePostmortem(id, args)
	case "list":
		return s.toolGetPostmortems(id, args)
	case "get":
		return s.toolGetPostmortem(id, args)
	case "resolve":
		return s.toolResolvePostmortem(id, args)
	case "stats":
		return s.toolPostmortemStats(id, args)
	case "to_learnings":
		return s.toolPostmortemToLearnings(id, args)
	default:
		return consolidatedToolError(id, "postmortem", "action", action)
	}
}

// consolidatedToolError returns a standard error response for invalid action/type.
func consolidatedToolError(id any, tool, param, value string) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: -32602, Message: fmt.Sprintf("Invalid %s for %s: %q", param, tool, value)},
	}
}

// getStringArg safely extracts a string argument with a default value.
func getStringArg(args map[string]interface{}, key, defaultValue string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return defaultValue
}

// toolListProposals lists proposals for the govern tool.
func (s *MCPServer) toolListProposals(id any, args map[string]interface{}) jsonRPCResponse {
	status := getStringArg(args, "status", "pending")
	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	proposals, err := s.butler.Memory().GetProposals(status, "", limit)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("Failed to get proposals: %v", err))
	}

	var result string
	if len(proposals) == 0 {
		result = fmt.Sprintf("No proposals with status '%s'.", status)
	} else {
		result = fmt.Sprintf("## Proposals (%d %s)\n\n", len(proposals), status)
		for _, p := range proposals {
			summary := p.Content
			if len(summary) > 60 {
				summary = summary[:57] + "..."
			}
			result += fmt.Sprintf("- **%s** [%s]: %s\n", p.ID, p.ProposedAs, summary)
		}
	}

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: result}},
		},
	}
}
