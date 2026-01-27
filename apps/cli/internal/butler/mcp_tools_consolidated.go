package butler

// buildConsolidatedToolsList returns the consolidated list of MCP tools.
// Consolidated from 89 tools to 26 tools using action-based dispatching.
//
// Tool categories:
//   - Composite: session_init, file_context (critical workflows)
//   - Core: explore, store, recall, brief, session
//   - Search: search, embedding
//   - Intelligence: context, contradiction, decay
//   - Workflow: postmortem, handoff, conversation
//   - Cross-workspace: corridor
//   - Analysis: pattern, contract, analytics
//   - Governance: govern, route
//   - Management: room, index, playbook
func buildConsolidatedToolsList() []mcpTool {
	return []mcpTool{
		// ============================================================
		// COMPOSITE TOOLS - Critical workflows (kept separate)
		// ============================================================
		buildSessionInitTool(),
		buildFileContextTool(),

		// ============================================================
		// CORE TOOLS - Primary functionality
		// ============================================================
		buildExploreTool(),
		buildStoreTool(),
		buildRecallTool(),
		buildBriefTool(),
		buildSessionTool(),

		// ============================================================
		// SEARCH & EMBEDDING TOOLS
		// ============================================================
		buildSearchTool(),
		buildEmbeddingTool(),

		// ============================================================
		// INTELLIGENCE TOOLS
		// ============================================================
		buildContextTool(),
		buildContradictionTool(),
		buildDecayTool(),

		// ============================================================
		// WORKFLOW TOOLS
		// ============================================================
		buildPostmortemTool(),
		buildHandoffTool(),
		buildConversationTool(),

		// ============================================================
		// CROSS-WORKSPACE TOOLS
		// ============================================================
		buildCorridorTool(),

		// ============================================================
		// ANALYSIS TOOLS
		// ============================================================
		buildPatternTool(),
		buildContractTool(),
		buildAnalyticsTool(),

		// ============================================================
		// GOVERNANCE TOOLS
		// ============================================================
		buildGovernTool(),
		buildRouteTool(),

		// ============================================================
		// MANAGEMENT TOOLS - Room, Index, and Playbook management
		// ============================================================
		buildRoomTool(),
		buildIndexTool(),
		buildPlaybookTool(),
	}
}

// ============================================================
// COMPOSITE TOOLS (Critical workflows - kept as separate tools)
// ============================================================

func buildSessionInitTool() mcpTool {
	return mcpTool{
		Name: "session_init",
		Description: `🔴 **CRITICAL - THE FIRST CALL** Initialize a session with full context.

Combines: session start + workspace briefing + room listing.

**CALL THIS FIRST** at the start of every conversation or task.

Returns:
- Session ID (use for all subsequent calls)
- Workspace briefing (active agents, learnings, hotspots)
- Project structure (rooms and entry points)`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"agent_name": map[string]interface{}{
					"type":        "string",
					"description": "Your agent name (e.g., 'claude-code', 'cursor', 'copilot')",
				},
				"task": map[string]interface{}{
					"type":        "string",
					"description": "Brief description of what you're working on",
				},
				"agent_id": map[string]interface{}{
					"type":        "string",
					"description": "Optional unique identifier for this agent instance",
				},
			},
			"required": []string{"agent_name"},
		},
		Autonomy: &mcpToolAutonomy{
			Level:     "required",
			Triggers:  []string{"session_start", "task_begin", "conversation_start"},
			Frequency: "once_per_session",
		},
	}
}

func buildFileContextTool() mcpTool {
	return mcpTool{
		Name: "file_context",
		Description: `🔴 **CRITICAL - CALL BEFORE EVERY FILE EDIT** Get complete context for a file.

Combines: file learnings + conflict detection + failure history.

**MUST CALL** before editing any file.

Returns:
- Conflict warnings (if another agent is editing)
- File-scoped learnings and decisions
- Known failures and their severity
- File edit history`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file you're about to edit",
				},
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Your session ID from session_init",
				},
			},
			"required": []string{"file_path"},
		},
		Autonomy: &mcpToolAutonomy{
			Level:         "required",
			Prerequisites: []string{"session_init"},
			Triggers:      []string{"before_file_edit"},
			Frequency:     "per_file",
		},
	}
}

// ============================================================
// CORE TOOLS
// ============================================================

func buildExploreTool() mcpTool {
	return mcpTool{
		Name: "explore",
		Description: `🟡 **IMPORTANT** Search and explore the codebase.

Actions:
- search: Search by intent/keywords (default)
- rooms: List all available rooms
- context: Get complete context for a task
- impact: Analyze change impact
- symbols: List symbols by kind
- symbol: Get specific symbol details
- file: Get exported symbols in a file
- deps: Get dependency graph
- callers: Find function callers
- callees: Find function callees
- graph: Get complete call graph`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"search", "rooms", "context", "impact", "symbols", "symbol", "file", "deps", "callers", "callees", "graph"},
					"description": "Explore action (default: search)",
					"default":     "search",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query (for action=search)",
				},
				"task": map[string]interface{}{
					"type":        "string",
					"description": "Task description (for action=context)",
				},
				"target": map[string]interface{}{
					"type":        "string",
					"description": "Target file/symbol (for action=impact)",
				},
				"kind": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"class", "interface", "function", "method", "constant", "type", "enum"},
					"description": "Symbol kind (for action=symbols)",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Symbol name (for action=symbol)",
				},
				"symbol": map[string]interface{}{
					"type":        "string",
					"description": "Symbol name (for action=callers/callees)",
				},
				"file": map[string]interface{}{
					"type":        "string",
					"description": "File path (for action=file/deps/graph/callees)",
				},
				"files": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "File paths (for action=deps)",
				},
				"room": map[string]interface{}{
					"type":        "string",
					"description": "Filter to specific room (for action=search)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results (default: 10)",
					"default":     10,
				},
				"fuzzy": map[string]interface{}{
					"type":        "boolean",
					"description": "Enable fuzzy matching (for action=search)",
					"default":     false,
				},
				"includeTests": map[string]interface{}{
					"type":        "boolean",
					"description": "Include test files (for action=context)",
					"default":     false,
				},
			},
		},
		Autonomy: &mcpToolAutonomy{
			Level:         "recommended",
			Prerequisites: []string{"session_init"},
			Triggers:      []string{"unknown_location", "feature_discovery", "code_search"},
			Frequency:     "as_needed",
		},
	}
}

func buildStoreTool() mcpTool {
	return mcpTool{
		Name: "store",
		Description: `🟡 **IMPORTANT** Store knowledge in the palace.

Types:
- auto: Auto-classify based on content (default)
- decision: Store as decision
- idea: Store as idea
- learning: Store as learning

Set direct=true for immediate storage (human mode only).
Otherwise creates a proposal for human review.`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The knowledge to store",
				},
				"as": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"auto", "decision", "idea", "learning"},
					"description": "Type of knowledge (default: auto)",
					"default":     "auto",
				},
				"scope": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"palace", "room", "file"},
					"description": "Scope: palace, room, or file (default: palace)",
					"default":     "palace",
				},
				"scope_path": map[string]interface{}{
					"type":        "string",
					"description": "Path for room/file scope",
				},
				"direct": map[string]interface{}{
					"type":        "boolean",
					"description": "Bypass proposal system (human mode only)",
					"default":     false,
				},
				"context": map[string]interface{}{
					"type":        "string",
					"description": "Additional context for the knowledge",
				},
				"supersedes": map[string]interface{}{
					"type":        "string",
					"description": "ID of decision this supersedes",
				},
			},
			"required": []string{"content"},
		},
	}
}

func buildRecallTool() mcpTool {
	return mcpTool{
		Name: "recall",
		Description: `🟡 **IMPORTANT** Retrieve and manage knowledge.

Actions for retrieval:
- get: Get records by type (default)
- links: Get links for a record

Actions for management (human mode):
- link: Create relationship between records
- unlink: Remove a link
- outcome: Record decision outcome
- obsolete: Mark learning as obsolete
- archive: Archive old learnings

Types (for action=get):
- learnings, decisions, ideas`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"get", "links", "link", "unlink", "outcome", "obsolete", "archive"},
					"description": "Recall action (default: get)",
					"default":     "get",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"learnings", "decisions", "ideas"},
					"description": "Record type (for action=get)",
					"default":     "learnings",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query (for action=get)",
				},
				"record_id": map[string]interface{}{
					"type":        "string",
					"description": "Record ID (for action=links/link/obsolete)",
				},
				"decision_id": map[string]interface{}{
					"type":        "string",
					"description": "Decision ID (for action=outcome/link)",
				},
				"source_id": map[string]interface{}{
					"type":        "string",
					"description": "Source record ID (for action=link)",
				},
				"target_id": map[string]interface{}{
					"type":        "string",
					"description": "Target record ID (for action=link)",
				},
				"relation": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"supports", "contradicts", "supersedes", "implements", "related"},
					"description": "Relationship type (for action=link)",
				},
				"link_id": map[string]interface{}{
					"type":        "string",
					"description": "Link ID to remove (for action=unlink)",
				},
				"outcome": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"successful", "failed", "mixed", "abandoned"},
					"description": "Decision outcome (for action=outcome)",
				},
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "Reason (for action=obsolete/outcome)",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Filter by status (for action=get)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results (default: 10)",
					"default":     10,
				},
			},
		},
	}
}

func buildBriefTool() mcpTool {
	return mcpTool{
		Name: "brief",
		Description: `🟢 **RECOMMENDED** Get workspace or file briefing.

Modes:
- workspace: Full workspace briefing (default)
- file: File-specific intelligence
- smart: LLM-powered contextual briefing`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"mode": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"workspace", "file", "smart"},
					"description": "Briefing mode (default: workspace)",
					"default":     "workspace",
				},
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "File path (for mode=file)",
				},
				"context": map[string]interface{}{
					"type":        "string",
					"description": "Context type (for mode=smart): 'file', 'room', 'task'",
				},
				"style": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"brief", "detailed", "technical"},
					"description": "Briefing style (for mode=smart)",
					"default":     "brief",
				},
			},
		},
	}
}

func buildSessionTool() mcpTool {
	return mcpTool{
		Name: "session",
		Description: `🟢 **RECOMMENDED** Manage agent sessions.

Actions:
- start: Start a new session
- end: End current session
- log: Log activity within session
- conflict: Check for file conflicts
- list: List all sessions
- resume: Resume a previous session
- status: Get current session status`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"start", "end", "log", "conflict", "list", "resume", "status"},
					"description": "Session action",
				},
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session ID (for end/log/resume)",
				},
				"agent_name": map[string]interface{}{
					"type":        "string",
					"description": "Agent name (for start)",
				},
				"agent_id": map[string]interface{}{
					"type":        "string",
					"description": "Agent ID (for start)",
				},
				"goal": map[string]interface{}{
					"type":        "string",
					"description": "Session goal (for start)",
				},
				"message": map[string]interface{}{
					"type":        "string",
					"description": "Log message (for log)",
				},
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "File path (for conflict)",
				},
				"outcome": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"completed", "partial", "failed", "abandoned"},
					"description": "Session outcome (for end)",
				},
				"summary": map[string]interface{}{
					"type":        "string",
					"description": "Session summary (for end)",
				},
				"include_inactive": map[string]interface{}{
					"type":        "boolean",
					"description": "Include inactive sessions (for list)",
					"default":     false,
				},
			},
			"required": []string{"action"},
		},
	}
}

// ============================================================
// SEARCH & EMBEDDING TOOLS
// ============================================================

func buildSearchTool() mcpTool {
	return mcpTool{
		Name: "search",
		Description: `🟢 **RECOMMENDED** Advanced search capabilities.

Modes:
- semantic: Semantic search with embeddings
- hybrid: Combined keyword + semantic search
- similar: Find similar records`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"mode": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"semantic", "hybrid", "similar"},
					"description": "Search mode (default: hybrid)",
					"default":     "hybrid",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query",
				},
				"record_id": map[string]interface{}{
					"type":        "string",
					"description": "Record ID (for mode=similar)",
				},
				"kinds": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Record kinds to search",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results (default: 10)",
					"default":     10,
				},
			},
		},
	}
}

func buildEmbeddingTool() mcpTool {
	return mcpTool{
		Name: "embedding",
		Description: `🟢 **OPTIONAL** Manage embeddings for semantic search.

Actions:
- sync: Generate/update embeddings
- stats: Get embedding statistics`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"sync", "stats"},
					"description": "Embedding action",
				},
				"kinds": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Record kinds to sync (for action=sync)",
				},
				"force": map[string]interface{}{
					"type":        "boolean",
					"description": "Force regeneration (for action=sync)",
					"default":     false,
				},
			},
			"required": []string{"action"},
		},
	}
}

// ============================================================
// INTELLIGENCE TOOLS
// ============================================================

func buildContextTool() mcpTool {
	return mcpTool{
		Name: "context",
		Description: `🟢 **RECOMMENDED** Smart context management.

Actions:
- inject: Get auto-injected context for a file
- explain: Explain scope inheritance
- focus: Set task focus for prioritization
- get: Get prioritized context
- pin: Pin/unpin records for priority`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"inject", "explain", "focus", "get", "pin"},
					"description": "Context action",
				},
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "File path (for inject/explain/get)",
				},
				"task": map[string]interface{}{
					"type":        "string",
					"description": "Task description (for focus)",
				},
				"record_id": map[string]interface{}{
					"type":        "string",
					"description": "Record ID (for pin)",
				},
				"unpin": map[string]interface{}{
					"type":        "boolean",
					"description": "Unpin instead of pin (for pin)",
					"default":     false,
				},
			},
			"required": []string{"action"},
		},
	}
}

func buildContradictionTool() mcpTool {
	return mcpTool{
		Name: "contradiction",
		Description: `🟢 **OPTIONAL** Find and manage contradicting records.

Actions:
- find: Find contradictions for a record
- check: Check if two records contradict
- summary: Get contradiction summary`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"find", "check", "summary"},
					"description": "Contradiction action",
				},
				"record_id": map[string]interface{}{
					"type":        "string",
					"description": "Record ID (for find)",
				},
				"record1_id": map[string]interface{}{
					"type":        "string",
					"description": "First record ID (for check)",
				},
				"record2_id": map[string]interface{}{
					"type":        "string",
					"description": "Second record ID (for check)",
				},
			},
			"required": []string{"action"},
		},
	}
}

func buildDecayTool() mcpTool {
	return mcpTool{
		Name: "decay",
		Description: `🟢 **OPTIONAL** Manage learning confidence decay.

Actions:
- stats: Get decay statistics
- preview: Preview decay candidates
- apply: Apply confidence decay
- reinforce: Reinforce a learning
- boost: Boost learning confidence`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"stats", "preview", "apply", "reinforce", "boost"},
					"description": "Decay action",
				},
				"learning_id": map[string]interface{}{
					"type":        "string",
					"description": "Learning ID (for reinforce/boost)",
				},
				"amount": map[string]interface{}{
					"type":        "number",
					"description": "Boost amount 0.0-1.0 (for boost)",
				},
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "Reason for reinforcement",
				},
			},
			"required": []string{"action"},
		},
	}
}

// ============================================================
// WORKFLOW TOOLS
// ============================================================

func buildPostmortemTool() mcpTool {
	return mcpTool{
		Name: "postmortem",
		Description: `🟢 **RECOMMENDED** Create and manage failure postmortems.

Actions:
- create: Create a new postmortem
- list: List postmortems
- get: Get specific postmortem
- resolve: Mark as resolved
- stats: Get postmortem statistics
- to_learnings: Convert to learnings`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"create", "list", "get", "resolve", "stats", "to_learnings"},
					"description": "Postmortem action",
				},
				"id": map[string]interface{}{
					"type":        "string",
					"description": "Postmortem ID (for get/resolve/to_learnings)",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Title (for create)",
				},
				"what_happened": map[string]interface{}{
					"type":        "string",
					"description": "What happened (for create)",
				},
				"root_cause": map[string]interface{}{
					"type":        "string",
					"description": "Root cause (for create)",
				},
				"resolution": map[string]interface{}{
					"type":        "string",
					"description": "Resolution (for create/resolve)",
				},
				"severity": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"low", "medium", "high", "critical"},
					"description": "Severity (for create)",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Filter by status (for list)",
				},
			},
			"required": []string{"action"},
		},
	}
}

func buildHandoffTool() mcpTool {
	return mcpTool{
		Name: "handoff",
		Description: `🟢 **RECOMMENDED** Create and manage task handoffs between agents.

Actions:
- create: Create a new handoff
- list: List available handoffs
- accept: Accept a handoff
- complete: Mark handoff complete`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"create", "list", "accept", "complete"},
					"description": "Handoff action",
				},
				"id": map[string]interface{}{
					"type":        "string",
					"description": "Handoff ID (for accept/complete)",
				},
				"task": map[string]interface{}{
					"type":        "string",
					"description": "Task description (for create)",
				},
				"context": map[string]interface{}{
					"type":        "string",
					"description": "Context/notes (for create)",
				},
				"files": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Related files (for create)",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Filter by status (for list)",
				},
			},
			"required": []string{"action"},
		},
	}
}

func buildConversationTool() mcpTool {
	return mcpTool{
		Name: "conversation",
		Description: `🟢 **OPTIONAL** Store and search conversation history.

Actions:
- store: Store a conversation
- search: Search past conversations
- extract: Extract insights from conversation`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"store", "search", "extract"},
					"description": "Conversation action",
				},
				"conversation_id": map[string]interface{}{
					"type":        "string",
					"description": "Conversation ID (for extract)",
				},
				"summary": map[string]interface{}{
					"type":        "string",
					"description": "Conversation summary (for store)",
				},
				"messages": map[string]interface{}{
					"type":        "array",
					"description": "Messages array (for store)",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query (for search)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results (for search)",
					"default":     10,
				},
			},
			"required": []string{"action"},
		},
	}
}

// ============================================================
// CROSS-WORKSPACE TOOLS
// ============================================================

func buildCorridorTool() mcpTool {
	return mcpTool{
		Name: "corridor",
		Description: `🟢 **OPTIONAL** Cross-workspace knowledge sharing.

Actions:
- learnings: Get corridor learnings
- links: Get linked workspaces
- stats: Get corridor statistics
- promote: Promote learning to corridor
- reinforce: Reinforce corridor learning`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"learnings", "links", "stats", "promote", "reinforce"},
					"description": "Corridor action",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query (for learnings)",
				},
				"learning_id": map[string]interface{}{
					"type":        "string",
					"description": "Learning ID (for promote/reinforce)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results",
					"default":     10,
				},
			},
			"required": []string{"action"},
		},
	}
}

// ============================================================
// ANALYSIS TOOLS
// ============================================================

func buildPatternTool() mcpTool {
	return mcpTool{
		Name: "pattern",
		Description: `🟢 **OPTIONAL** Detect and manage code patterns.

Actions:
- list: Get detected patterns
- show: Show pattern details
- approve: Approve a pattern
- ignore: Ignore a pattern
- stats: Get pattern statistics`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"list", "show", "approve", "ignore", "stats"},
					"description": "Pattern action",
				},
				"pattern_id": map[string]interface{}{
					"type":        "string",
					"description": "Pattern ID (for show/approve/ignore)",
				},
				"category": map[string]interface{}{
					"type":        "string",
					"description": "Filter by category (for list)",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Filter by status (for list)",
				},
			},
			"required": []string{"action"},
		},
	}
}

func buildContractTool() mcpTool {
	return mcpTool{
		Name: "contract",
		Description: `🟢 **OPTIONAL** Manage API contracts.

Actions:
- list: Get API contracts
- show: Show contract details
- verify: Verify a contract
- ignore: Ignore a contract
- stats: Get contract statistics
- mismatches: Get contracts with mismatches`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"list", "show", "verify", "ignore", "stats", "mismatches"},
					"description": "Contract action",
				},
				"contract_id": map[string]interface{}{
					"type":        "string",
					"description": "Contract ID (for show/verify/ignore)",
				},
				"method": map[string]interface{}{
					"type":        "string",
					"description": "Filter by HTTP method (for list)",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Filter by path (for list)",
				},
			},
			"required": []string{"action"},
		},
	}
}

func buildAnalyticsTool() mcpTool {
	return mcpTool{
		Name: "analytics",
		Description: `🟢 **OPTIONAL** Get workspace analytics.

Types:
- sessions: Session analytics
- learnings: Learning effectiveness
- health: Overall workspace health`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"sessions", "learnings", "health"},
					"description": "Analytics type",
				},
				"days": map[string]interface{}{
					"type":        "integer",
					"description": "Days to analyze (for sessions)",
					"default":     30,
				},
				"sort": map[string]interface{}{
					"type":        "string",
					"description": "Sort order (for learnings)",
				},
			},
			"required": []string{"type"},
		},
	}
}

// ============================================================
// GOVERNANCE TOOLS
// ============================================================

func buildGovernTool() mcpTool {
	return mcpTool{
		Name: "govern",
		Description: `🔒 **HUMAN ONLY** Proposal governance actions.

Actions:
- approve: Approve a proposal
- reject: Reject a proposal
- list: List pending proposals`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"approve", "reject", "list"},
					"description": "Governance action",
				},
				"proposal_id": map[string]interface{}{
					"type":        "string",
					"description": "Proposal ID (for approve/reject)",
				},
				"note": map[string]interface{}{
					"type":        "string",
					"description": "Review note (for approve/reject)",
				},
				"reviewer": map[string]interface{}{
					"type":        "string",
					"description": "Reviewer identifier",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Filter by status (for list)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum results (for list)",
					"default":     20,
				},
			},
			"required": []string{"action"},
		},
	}
}

func buildRouteTool() mcpTool {
	return mcpTool{
		Name: "route",
		Description: `🟡 **IMPORTANT** Get a navigation route for understanding a topic.

Use this when you need to learn about a system/feature before modifying it.
Returns an ordered path through rooms, files, decisions, and learnings.`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"intent": map[string]interface{}{
					"type":        "string",
					"description": "What you want to understand",
				},
				"scope": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"file", "room", "palace"},
					"description": "Starting scope (default: palace)",
					"default":     "palace",
				},
				"scope_path": map[string]interface{}{
					"type":        "string",
					"description": "Path for file/room scope",
				},
			},
			"required": []string{"intent"},
		},
	}
}

// ============================================================
// MANAGEMENT TOOLS - Room and Index management
// ============================================================

func buildRoomTool() mcpTool {
	return mcpTool{
		Name: "room",
		Description: `🟡 **IMPORTANT** Manage Mind Palace rooms.

Rooms are logical groupings of code with entry points and capabilities.

Actions:
- list: List all rooms (default)
- show: Show room details
- create: Create a new room
- update: Update room properties
- delete: Delete a room

Use rooms to organize your codebase and provide context for different features.`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"list", "show", "create", "update", "delete"},
					"description": "Room action (default: list)",
					"default":     "list",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Room name (required for show/create/update/delete)",
				},
				"summary": map[string]interface{}{
					"type":        "string",
					"description": "Room summary (required for create, optional for update)",
				},
				"entry_points": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Entry point paths (required for create, optional for update)",
				},
				"capabilities": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Room capabilities (optional)",
				},
			},
		},
		Autonomy: &mcpToolAutonomy{
			Level:     "recommended",
			Triggers:  []string{"organizing_code", "feature_scoping"},
			Frequency: "as_needed",
		},
	}
}

func buildIndexTool() mcpTool {
	return mcpTool{
		Name: "index",
		Description: `🟡 **IMPORTANT** Manage the code index.

The index contains parsed files, symbols, and relationships.

Actions:
- status: Get index freshness status (default)
- scan: Trigger incremental scan
- rescan: Force full rescan (clears existing index)
- stats: Get detailed index statistics

Use this tool to ensure the index is up-to-date before searching.`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"status", "scan", "rescan", "stats"},
					"description": "Index action (default: status)",
					"default":     "status",
				},
				"workers": map[string]interface{}{
					"type":        "integer",
					"description": "Number of parallel workers for scan/rescan (0=auto)",
					"default":     0,
				},
			},
		},
		Autonomy: &mcpToolAutonomy{
			Level:     "recommended",
			Triggers:  []string{"stale_results", "index_outdated", "before_search"},
			Frequency: "as_needed",
		},
	}
}

func buildPlaybookTool() mcpTool {
	return mcpTool{
		Name: "playbook",
		Description: `🟡 **IMPORTANT** Execute guided task playbooks.

Playbooks are sequences of rooms with steps to complete high-level goals.

Actions:
- list: List all available playbooks (default)
- show: Show playbook details
- start: Start playbook execution
- status: Get current execution status
- guidance: Get guidance for current step
- advance: Mark current step complete, move to next
- evidence: Record evidence collected
- verify: Run verification checks
- complete: Mark playbook complete, clear state

Playbooks guide agents through structured workflows with rooms, steps, and evidence collection.`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"action": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"list", "show", "start", "status", "guidance", "advance", "evidence", "verify", "complete"},
					"description": "Playbook action (default: list)",
					"default":     "list",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Playbook name (for show/start)",
				},
				"evidence_id": map[string]interface{}{
					"type":        "string",
					"description": "Evidence ID to record (for evidence action)",
				},
				"value": map[string]interface{}{
					"type":        "string",
					"description": "Evidence value (for evidence action)",
				},
			},
		},
		Autonomy: &mcpToolAutonomy{
			Level:     "recommended",
			Triggers:  []string{"onboarding", "guided_task", "structured_workflow"},
			Frequency: "as_needed",
		},
	}
}
