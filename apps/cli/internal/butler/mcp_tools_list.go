package butler

// buildToolsList returns the list of all available MCP tools.
// Tool names are prefixed to match CLI commands:
//   - explore_*: search, context, graph, symbols
//   - store_*: remember ideas/decisions/learnings
//   - recall_*: retrieve knowledge, manage links
//   - brief_*: get briefings and file intel
//   - session_*: manage agent sessions
//   - conversation_*: store/search conversations
func buildToolsList() []mcpTool {
	return []mcpTool{
		// ============================================================
		// EXPLORE TOOLS - Search, context, symbols, and call graphs
		// ============================================================
		{
			Name:        "explore",
			Description: "Search the codebase by intent or keywords. Returns ranked results grouped by 'Rooms' (conceptual areas of the project).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query - can be natural language ('where is auth logic') or code symbols ('handleAuth', 'AuthService').",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results (default: 10, max: 50)",
						"default":     10,
					},
					"room": map[string]interface{}{
						"type":        "string",
						"description": "Optional: filter results to a specific Room name",
					},
					"fuzzy": map[string]interface{}{
						"type":        "boolean",
						"description": "Enable fuzzy matching for typo tolerance (default: false). Useful when unsure of exact spelling.",
						"default":     false,
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "explore_rooms",
			Description: "List all available Rooms in the Mind Palace. Rooms are curated conceptual areas of the project with entry points and capabilities.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "explore_context",
			Description: "Get complete context for a task. Returns all relevant files, symbols, imports, decisions, learnings, and ideas.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task": map[string]interface{}{
						"type":        "string",
						"description": "Description of the task you want to accomplish (e.g., 'add user authentication', 'fix the login bug', 'understand the payment flow').",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of files/symbols to return (default: 20)",
						"default":     20,
					},
					"maxTokens": map[string]interface{}{
						"type":        "integer",
						"description": "Token budget for the response. Context will be truncated to fit within this budget.",
					},
					"includeTests": map[string]interface{}{
						"type":        "boolean",
						"description": "Include test files in the results (default: false).",
						"default":     false,
					},
					"includeLearnings": map[string]interface{}{
						"type":        "boolean",
						"description": "Include relevant learnings from session memory (default: true).",
						"default":     true,
					},
					"includeFileIntel": map[string]interface{}{
						"type":        "boolean",
						"description": "Include file intelligence (edit history, failure rates) for files in context (default: true).",
						"default":     true,
					},
					"includeIdeas": map[string]interface{}{
						"type":        "boolean",
						"description": "Include relevant ideas (default: true).",
						"default":     true,
					},
					"includeDecisions": map[string]interface{}{
						"type":        "boolean",
						"description": "Include relevant decisions (default: true).",
						"default":     true,
					},
				},
				"required": []string{"task"},
			},
		},
		{
			Name:        "explore_impact",
			Description: "Analyze the impact of changing a file or symbol. Returns what depends on it (dependents) and what it depends on (dependencies).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"target": map[string]interface{}{
						"type":        "string",
						"description": "File path or symbol name to analyze (e.g., 'internal/auth/handler.go', 'AuthService').",
					},
				},
				"required": []string{"target"},
			},
		},
		{
			Name:        "explore_symbols",
			Description: "List symbols of a specific kind in the codebase (e.g., all classes, all functions, all interfaces).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"kind": map[string]interface{}{
						"type":        "string",
						"description": "Symbol kind: 'class', 'interface', 'function', 'method', 'constant', 'type', 'enum', 'property', 'constructor'.",
						"enum":        []string{"class", "interface", "function", "method", "constant", "type", "enum", "property", "constructor"},
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of symbols to return (default: 50)",
						"default":     50,
					},
				},
				"required": []string{"kind"},
			},
		},
		{
			Name:        "explore_symbol",
			Description: "Get detailed information about a specific symbol (function, class, method, etc.) including its signature, documentation, and location.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the symbol to look up.",
					},
					"file": map[string]interface{}{
						"type":        "string",
						"description": "Optional: file path to narrow down the search if the symbol name is common.",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "explore_file",
			Description: "Get all exported symbols in a specific file.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file": map[string]interface{}{
						"type":        "string",
						"description": "File path to get symbols from.",
					},
				},
				"required": []string{"file"},
			},
		},
		{
			Name:        "explore_deps",
			Description: "Get the dependency graph for one or more files, showing what they import.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"files": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "List of file paths to analyze.",
					},
				},
				"required": []string{"files"},
			},
		},
		{
			Name:        "explore_callers",
			Description: "Find all locations that call a function or method. Answers 'who calls this function?'",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"symbol": map[string]interface{}{
						"type":        "string",
						"description": "Function or method name to find callers of (e.g., 'handleAuth', 'config.Parse', 'User.Save').",
					},
				},
				"required": []string{"symbol"},
			},
		},
		{
			Name:        "explore_callees",
			Description: "Find all functions/methods called by a symbol. Answers 'what does this function call?'",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"symbol": map[string]interface{}{
						"type":        "string",
						"description": "Function or method name to analyze.",
					},
					"file": map[string]interface{}{
						"type":        "string",
						"description": "File path where the symbol is defined (required if symbol name is ambiguous).",
					},
				},
				"required": []string{"symbol", "file"},
			},
		},
		{
			Name:        "explore_graph",
			Description: "Get the complete call graph for a file, showing all incoming and outgoing calls.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file": map[string]interface{}{
						"type":        "string",
						"description": "File path to analyze.",
					},
				},
				"required": []string{"file"},
			},
		},

		// ============================================================
		// STORE TOOLS - Store ideas, decisions, and learnings
		// ============================================================
		{
			Name:        "store",
			Description: "Store a thought with auto-classification. Content is analyzed and stored as an idea, decision, or learning based on natural language signals.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The content to store. Use phrases like 'Let's...' for decisions, 'What if...' for ideas, 'TIL...' for learnings.",
					},
					"as": map[string]interface{}{
						"type":        "string",
						"description": "Optional: force classification as 'decision', 'idea', or 'learning'. If not provided, auto-classifies.",
						"enum":        []string{"decision", "idea", "learning"},
					},
					"scope": map[string]interface{}{
						"type":        "string",
						"description": "Scope: 'palace' (workspace-wide), 'room' (module), 'file' (specific file). Default: palace.",
						"enum":        []string{"palace", "room", "file"},
						"default":     "palace",
					},
					"scopePath": map[string]interface{}{
						"type":        "string",
						"description": "Path for room/file scope (room name or file path).",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Optional tags to categorize this record.",
					},
					"confidence": map[string]interface{}{
						"type":        "number",
						"description": "For learnings: confidence level 0.0-1.0 (default: 0.5).",
						"default":     0.5,
					},
				},
				"required": []string{"content"},
			},
		},

		// ============================================================
		// RECALL TOOLS - Retrieve knowledge and manage relationships
		// ============================================================
		{
			Name:        "recall",
			Description: "Retrieve learnings, optionally filtered by scope or search query.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Optional search query to filter learnings.",
					},
					"scope": map[string]interface{}{
						"type":        "string",
						"description": "Filter by scope: 'palace', 'room', 'file'.",
						"enum":        []string{"palace", "room", "file"},
					},
					"scopePath": map[string]interface{}{
						"type":        "string",
						"description": "Filter by scope path.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum learnings to return (default: 10).",
						"default":     10,
					},
				},
			},
		},
		{
			Name:        "recall_decisions",
			Description: "Retrieve decisions, optionally filtered by status, scope, or search query.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Optional search query to filter decisions.",
					},
					"status": map[string]interface{}{
						"type":        "string",
						"description": "Filter by status: 'active', 'superseded', 'reversed'.",
						"enum":        []string{"active", "superseded", "reversed"},
					},
					"scope": map[string]interface{}{
						"type":        "string",
						"description": "Filter by scope: 'palace', 'room', 'file'.",
						"enum":        []string{"palace", "room", "file"},
					},
					"scopePath": map[string]interface{}{
						"type":        "string",
						"description": "Filter by scope path.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum decisions to return (default: 10).",
						"default":     10,
					},
				},
			},
		},
		{
			Name:        "recall_ideas",
			Description: "Retrieve ideas, optionally filtered by status, scope, or search query.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Optional search query to filter ideas.",
					},
					"status": map[string]interface{}{
						"type":        "string",
						"description": "Filter by status: 'active', 'exploring', 'implemented', 'dropped'.",
						"enum":        []string{"active", "exploring", "implemented", "dropped"},
					},
					"scope": map[string]interface{}{
						"type":        "string",
						"description": "Filter by scope: 'palace', 'room', 'file'.",
						"enum":        []string{"palace", "room", "file"},
					},
					"scopePath": map[string]interface{}{
						"type":        "string",
						"description": "Filter by scope path.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum ideas to return (default: 10).",
						"default":     10,
					},
				},
			},
		},
		{
			Name:        "recall_outcome",
			Description: "Record the outcome of a decision. Use this to track whether decisions worked out.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"decisionId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the decision (e.g., 'd_abc123').",
					},
					"outcome": map[string]interface{}{
						"type":        "string",
						"description": "Outcome: 'success', 'failed', or 'mixed'.",
						"enum":        []string{"success", "failed", "mixed"},
					},
					"note": map[string]interface{}{
						"type":        "string",
						"description": "Optional note about the outcome (what happened, lessons learned).",
					},
				},
				"required": []string{"decisionId", "outcome"},
			},
		},
		{
			Name:        "recall_link",
			Description: "Create a relationship between records (ideas, decisions, learnings, code files).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sourceId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the source record (e.g., 'i_abc123', 'd_xyz789').",
					},
					"targetId": map[string]interface{}{
						"type":        "string",
						"description": "ID of target record or code reference (e.g., 'd_abc123', 'auth/jwt.go:15-45').",
					},
					"relation": map[string]interface{}{
						"type":        "string",
						"description": "Type of relationship.",
						"enum":        []string{"supports", "contradicts", "implements", "supersedes", "inspired_by", "related"},
					},
				},
				"required": []string{"sourceId", "targetId", "relation"},
			},
		},
		{
			Name:        "recall_links",
			Description: "Get all links for a record (as source, target, or both).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"recordId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the record to get links for.",
					},
					"direction": map[string]interface{}{
						"type":        "string",
						"description": "Filter by direction: 'from' (source), 'to' (target), 'all' (both). Default: all.",
						"enum":        []string{"from", "to", "all"},
						"default":     "all",
					},
				},
				"required": []string{"recordId"},
			},
		},
		{
			Name:        "recall_unlink",
			Description: "Delete a link by its ID.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"linkId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the link to delete (e.g., 'lnk_abc123').",
					},
				},
				"required": []string{"linkId"},
			},
		},

		// ============================================================
		// BRIEF TOOLS - Get briefings and file intelligence
		// ============================================================
		{
			Name:        "brief",
			Description: "Get a comprehensive briefing before working. Shows active agents, conflicts, learnings, hotspots.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file": map[string]interface{}{
						"type":        "string",
						"description": "Optional file path for file-specific briefing.",
					},
				},
			},
		},
		{
			Name:        "brief_file",
			Description: "Get intelligence about a file: edit history, failure rate, associated learnings.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File path to get intelligence for.",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "briefing_smart",
			Description: "Generate a smart briefing with LLM-powered insights for a file, room, task, or workspace.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"context": map[string]interface{}{
						"type":        "string",
						"description": "Context type: 'file', 'room', 'task', or 'workspace' (default: workspace).",
						"enum":        []string{"file", "room", "task", "workspace"},
					},
					"contextPath": map[string]interface{}{
						"type":        "string",
						"description": "Context path: file path, room name, or task description.",
					},
					"style": map[string]interface{}{
						"type":        "string",
						"description": "Briefing style: 'summary', 'detailed', or 'actionable' (default: summary).",
						"enum":        []string{"summary", "detailed", "actionable"},
					},
				},
			},
		},

		// ============================================================
		// SESSION TOOLS - Manage agent sessions
		// ============================================================
		{
			Name:        "session_start",
			Description: "Start a new agent session. Sessions track activities and learnings within a work context.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"agentType": map[string]interface{}{
						"type":        "string",
						"description": "Type of agent (e.g., 'claude', 'cursor', 'aider').",
					},
					"agentId": map[string]interface{}{
						"type":        "string",
						"description": "Optional unique agent instance ID.",
					},
					"goal": map[string]interface{}{
						"type":        "string",
						"description": "What this session aims to accomplish.",
					},
				},
				"required": []string{"agentType"},
			},
		},
		{
			Name:        "session_log",
			Description: "Log an activity within a session (file read, edit, search, command).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "Session ID from session_start.",
					},
					"kind": map[string]interface{}{
						"type":        "string",
						"description": "Activity type: 'file_read', 'file_edit', 'search', 'command'.",
						"enum":        []string{"file_read", "file_edit", "search", "command"},
					},
					"target": map[string]interface{}{
						"type":        "string",
						"description": "File path or search query.",
					},
					"outcome": map[string]interface{}{
						"type":        "string",
						"description": "Result: 'success', 'failure', 'unknown'.",
						"enum":        []string{"success", "failure", "unknown"},
					},
					"details": map[string]interface{}{
						"type":        "string",
						"description": "Optional JSON details about the activity.",
					},
				},
				"required": []string{"sessionId", "kind", "target"},
			},
		},
		{
			Name:        "session_end",
			Description: "End a session and record its outcome.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "Session ID to end.",
					},
					"outcome": map[string]interface{}{
						"type":        "string",
						"description": "Session outcome: 'success', 'failure', 'partial'.",
						"enum":        []string{"success", "failure", "partial"},
					},
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "Brief summary of what was accomplished.",
					},
				},
				"required": []string{"sessionId"},
			},
		},
		{
			Name:        "session_conflict",
			Description: "Check if another agent is working on a file.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File path to check for conflicts.",
					},
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "Optional current session ID to exclude from conflict check.",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "session_list",
			Description: "List all agent sessions, optionally filtered by status.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"active": map[string]interface{}{
						"type":        "boolean",
						"description": "If true, only return active sessions. Default: false (returns all sessions).",
						"default":     false,
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum sessions to return (default: 20).",
						"default":     20,
					},
				},
			},
		},

		// ============================================================
		// CONVERSATION TOOLS - Store and search conversations
		// ============================================================
		{
			Name:        "conversation_store",
			Description: "Store the current conversation for future context. Use this to preserve important discussions for later recall.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"summary": map[string]interface{}{
						"type":        "string",
						"description": "A concise summary of the conversation (1-2 sentences).",
					},
					"messages": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"role":    map[string]interface{}{"type": "string", "enum": []string{"user", "assistant", "system"}},
								"content": map[string]interface{}{"type": "string"},
							},
							"required": []string{"role", "content"},
						},
						"description": "Array of conversation messages.",
					},
					"agentType": map[string]interface{}{
						"type":        "string",
						"description": "Type of AI agent (e.g., 'claude-code', 'cursor').",
					},
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "Optional session ID to link the conversation to.",
					},
				},
				"required": []string{"summary", "messages"},
			},
		},
		{
			Name:        "conversation_search",
			Description: "Search past conversations by summary or content.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query to match against conversation summaries.",
					},
					"sessionId": map[string]interface{}{
						"type":        "string",
						"description": "Filter by session ID.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum results (default 10).",
					},
				},
			},
		},
		{
			Name:        "conversation_extract",
			Description: "Extract ideas, decisions, and learnings from a stored conversation using AI analysis. Automatically stores extracted records in the memory.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"conversationId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the conversation to extract from (e.g., 'conv_abc123').",
					},
				},
				"required": []string{"conversationId"},
			},
		},

		// ============================================================
		// SEMANTIC SEARCH TOOLS - AI-powered search
		// ============================================================
		{
			Name:        "search_semantic",
			Description: "Perform semantic search using AI embeddings. Finds conceptually similar content even without exact word matches. Requires embedding backend to be configured.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Natural language query (e.g., 'error handling patterns', 'authentication best practices').",
					},
					"kinds": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Filter by record kinds: 'idea', 'decision', 'learning'. Default: all kinds.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum results to return (default: 10).",
						"default":     10,
					},
					"minSimilarity": map[string]interface{}{
						"type":        "number",
						"description": "Minimum similarity threshold 0.0-1.0 (default: 0.5).",
						"default":     0.5,
					},
					"scope": map[string]interface{}{
						"type":        "string",
						"description": "Filter by scope: 'palace', 'room', 'file'.",
						"enum":        []string{"palace", "room", "file"},
					},
					"scopePath": map[string]interface{}{
						"type":        "string",
						"description": "Filter by scope path.",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "search_hybrid",
			Description: "Combined keyword + semantic search. Returns results matching either exact words or conceptual meaning. Falls back to keyword-only if embeddings are not configured.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query - can be keywords or natural language.",
					},
					"kinds": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Filter by record kinds: 'idea', 'decision', 'learning'. Default: all kinds.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum results to return (default: 10).",
						"default":     10,
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "search_similar",
			Description: "Find records similar to a given record ID. Uses semantic similarity to find related content.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"recordId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the record to find similar records for (e.g., 'l_abc123', 'i_xyz789').",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum results to return (default: 5).",
						"default":     5,
					},
					"minSimilarity": map[string]interface{}{
						"type":        "number",
						"description": "Minimum similarity threshold 0.0-1.0 (default: 0.6).",
						"default":     0.6,
					},
				},
				"required": []string{"recordId"},
			},
		},

		// ============================================================
		// EMBEDDING MANAGEMENT TOOLS - Auto-embedding pipeline control
		// ============================================================
		{
			Name:        "embedding_sync",
			Description: "Generate embeddings for records that don't have them. Useful for backfilling after enabling embeddings or after bulk imports.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"kinds": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Record kinds to process: 'idea', 'decision', 'learning'. Default: all kinds.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum records to process (default: 100).",
						"default":     100,
					},
				},
			},
		},
		{
			Name:        "embedding_stats",
			Description: "Get statistics about the embedding system including total embeddings, pending records, and pipeline status.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},

		// ============================================================
		// LEARNING LIFECYCLE TOOLS - Manage learning status and feedback
		// ============================================================
		{
			Name:        "recall_learning_link",
			Description: "Link a learning to a decision for outcome feedback. When the decision's outcome is recorded, the learning's confidence is automatically updated.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"decisionId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the decision to link (e.g., 'd_abc123').",
					},
					"learningId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the learning to link (e.g., 'lrn_xyz789').",
					},
				},
				"required": []string{"decisionId", "learningId"},
			},
		},
		{
			Name:        "recall_obsolete",
			Description: "Mark a learning as obsolete with a reason. Obsolete learnings are preserved but hidden from active queries.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"learningId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the learning to mark obsolete.",
					},
					"reason": map[string]interface{}{
						"type":        "string",
						"description": "Reason for obsolescence (e.g., 'superseded by new approach', 'no longer applicable').",
					},
				},
				"required": []string{"learningId", "reason"},
			},
		},
		{
			Name:        "recall_archive",
			Description: "Archive old, low-confidence learnings. Archived learnings are preserved but hidden from active queries.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"unusedDays": map[string]interface{}{
						"type":        "integer",
						"description": "Archive learnings unused for this many days (default: 90).",
						"default":     90,
					},
					"maxConfidence": map[string]interface{}{
						"type":        "number",
						"description": "Only archive learnings with confidence at or below this threshold (default: 0.3).",
						"default":     0.3,
					},
				},
			},
		},
		{
			Name:        "recall_learnings_by_status",
			Description: "Retrieve learnings by lifecycle status: 'active', 'obsolete', or 'archived'.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{
						"type":        "string",
						"description": "Lifecycle status: 'active', 'obsolete', or 'archived'. Default: 'active'.",
						"default":     "active",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum learnings to return (default: 20).",
						"default":     20,
					},
				},
			},
		},

		// ============================================================
		// CONTRADICTION DETECTION TOOLS - AI-powered contradiction analysis
		// ============================================================
		{
			Name:        "recall_contradictions",
			Description: "Find records that contradict a given record using AI semantic analysis. Uses embeddings to find similar content, then LLM to detect contradictions.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"recordId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the record to check for contradictions (e.g., 'i_abc123', 'd_xyz789').",
					},
					"minConfidence": map[string]interface{}{
						"type":        "number",
						"description": "Minimum confidence threshold 0.0-1.0 for including contradictions (default: 0.7).",
						"default":     0.7,
					},
					"autoLink": map[string]interface{}{
						"type":        "boolean",
						"description": "Automatically create 'contradicts' links for high-confidence findings (default: true).",
						"default":     true,
					},
				},
				"required": []string{"recordId"},
			},
		},
		{
			Name:        "recall_contradiction_check",
			Description: "Check if two specific records contradict each other using AI analysis.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"record1Id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the first record to compare.",
					},
					"record2Id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the second record to compare.",
					},
				},
				"required": []string{"record1Id", "record2Id"},
			},
		},
		{
			Name:        "recall_contradiction_summary",
			Description: "Get a summary of all known contradictions in the system.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of contradiction pairs to show (default: 10).",
						"default":     10,
					},
				},
			},
		},

		// ============================================================
		// DECAY TOOLS - Confidence decay management
		// ============================================================
		{
			Name:        "decay_stats",
			Description: "Get statistics about confidence decay for learnings (at-risk count, decayed count, averages).",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "decay_preview",
			Description: "Preview what learnings would be affected by applying decay. Does not modify anything.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum records to show in preview (default: 20).",
						"default":     20,
					},
				},
			},
		},
		{
			Name:        "decay_apply",
			Description: "Apply confidence decay to inactive learnings. Reduces confidence of learnings that haven't been accessed.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "decay_reinforce",
			Description: "Reinforce a learning to prevent decay (marks it as recently accessed).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"learningId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the learning to reinforce (e.g., 'l_abc123').",
					},
				},
				"required": []string{"learningId"},
			},
		},
		{
			Name:        "decay_boost",
			Description: "Boost a learning's confidence (opposite of decay).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"learningId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the learning to boost (e.g., 'l_abc123').",
					},
					"amount": map[string]interface{}{
						"type":        "number",
						"description": "Amount to boost confidence (default: 0.1).",
						"default":     0.1,
					},
				},
				"required": []string{"learningId"},
			},
		},

		// ============================================================
		// CONTEXT & SCOPE TOOLS - Auto-injection and scope management
		// ============================================================
		{
			Name:        "context_auto_inject",
			Description: "Get automatically assembled context for a file. Returns prioritized learnings, decisions, failures, and warnings relevant to the file based on scope inheritance.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "File path to get context for.",
					},
					"maxTokens": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum tokens for context (default: 2000).",
						"default":     2000,
					},
					"includeLearnings": map[string]interface{}{
						"type":        "boolean",
						"description": "Include learnings in context (default: true).",
						"default":     true,
					},
					"includeDecisions": map[string]interface{}{
						"type":        "boolean",
						"description": "Include decisions in context (default: true).",
						"default":     true,
					},
					"includeFailures": map[string]interface{}{
						"type":        "boolean",
						"description": "Include failure information in context (default: true).",
						"default":     true,
					},
				},
				"required": []string{"file_path"},
			},
		},
		{
			Name:        "scope_explain",
			Description: "Explain the scope inheritance chain for a file. Shows which learnings/decisions apply at each scope level (file, room, palace, corridor).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "File path to explain scope for.",
					},
				},
				"required": []string{"file_path"},
			},
		},

		// ============================================================
		// POSTMORTEM TOOLS - Failure memory and analysis
		// ============================================================
		{
			Name:        "store_postmortem",
			Description: "Create a postmortem record to document a failure, its root cause, lessons learned, and prevention steps.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Brief title describing the failure (e.g., 'Authentication bypass in JWT validation').",
					},
					"what_happened": map[string]interface{}{
						"type":        "string",
						"description": "Detailed description of what went wrong.",
					},
					"root_cause": map[string]interface{}{
						"type":        "string",
						"description": "Analysis of why the failure occurred.",
					},
					"lessons_learned": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Key takeaways from this failure.",
					},
					"prevention_steps": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Steps to prevent this failure from recurring.",
					},
					"severity": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"low", "medium", "high", "critical"},
						"description": "Severity of the failure (default: medium).",
						"default":     "medium",
					},
					"affected_files": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "List of files affected by this failure.",
					},
					"related_decision": map[string]interface{}{
						"type":        "string",
						"description": "ID of a decision that led to this failure (e.g., 'd_abc123').",
					},
					"related_session": map[string]interface{}{
						"type":        "string",
						"description": "ID of the session where failure occurred.",
					},
				},
				"required": []string{"title", "what_happened"},
			},
		},
		{
			Name:        "get_postmortems",
			Description: "Retrieve postmortem records with optional filters for status and severity.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"open", "resolved", "recurring"},
						"description": "Filter by status.",
					},
					"severity": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"low", "medium", "high", "critical"},
						"description": "Filter by severity.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum postmortems to return (default: 20).",
						"default":     20,
					},
				},
			},
		},
		{
			Name:        "get_postmortem",
			Description: "Get detailed information about a specific postmortem.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the postmortem (e.g., 'pm_abc123').",
					},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "resolve_postmortem",
			Description: "Mark a postmortem as resolved.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the postmortem to resolve.",
					},
				},
				"required": []string{"id"},
			},
		},
		{
			Name:        "postmortem_stats",
			Description: "Get aggregated statistics about postmortems (counts by status and severity).",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "postmortem_to_learnings",
			Description: "Convert a postmortem's lessons learned into individual learning records. Creates learnings linked to the postmortem.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the postmortem to convert lessons from.",
					},
				},
				"required": []string{"id"},
			},
		},

		// ============================================================
		// CORRIDOR TOOLS - Personal cross-workspace learnings
		// ============================================================
		{
			Name:        "corridor_learnings",
			Description: "Get personal learnings from the global corridor (cross-workspace knowledge).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Optional search query to filter learnings.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum learnings to return (default: 20).",
						"default":     20,
					},
				},
			},
		},
		{
			Name:        "corridor_links",
			Description: "Get all linked workspaces in the global corridor.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "corridor_stats",
			Description: "Get statistics about the personal corridor (learning count, linked workspaces).",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "corridor_promote",
			Description: "Promote a workspace learning to the personal corridor for cross-workspace use.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"learningId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the learning to promote (e.g., 'l_abc123').",
					},
				},
				"required": []string{"learningId"},
			},
		},
		{
			Name:        "corridor_reinforce",
			Description: "Increase confidence for a personal corridor learning (marks it as useful).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"learningId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the personal learning to reinforce.",
					},
				},
				"required": []string{"learningId"},
			},
		},
	}
}
