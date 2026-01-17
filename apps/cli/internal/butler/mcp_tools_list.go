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
			Name: "explore",
			Description: `🟡 **IMPORTANT** Search the codebase by intent or keywords. Returns ranked results grouped by 'Rooms'.

**WHEN TO USE:**
- When user asks 'where is X?' or 'find Y'
- When looking for code related to a feature or concept
- When asked to modify something but don't know where it is
- To discover what exists before implementing something new

**AUTONOMOUS BEHAVIOR:**
Agents should use this proactively when encountering unfamiliar code areas or when user mentions features/concepts without file paths. No need to ask - searching is research, not action.

**BEST FOR:**
Quick discovery of relevant files and code locations. Like a 'smart grep' that understands intent and ranks by relevance.`,
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
			Name: "explore_rooms",
			Description: `🔴 **CRITICAL - CALL EARLY** List all available Rooms in the Mind Palace.

**WHEN TO USE:**
- At the start of a session (after brief) to understand project structure
- When user asks 'what areas does this project have?' or 'show me the structure'
- Before implementing features to understand which Room they belong in
- When you're unfamiliar with the codebase organization

**AUTONOMOUS BEHAVIOR:**
Agents should call this automatically early in the session, especially when working in an unfamiliar codebase. This is 'loading the map' - essential orientation.

**BEST FOR:**
Understanding the conceptual organization of the project. Rooms are curated areas with entry points and capabilities.`,
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name: "explore_context",
			Description: `🔴 **CRITICAL - USE BEFORE IMPLEMENTATION** Get complete context for a task. Returns all relevant files, symbols, imports, decisions, learnings, and ideas.

**WHEN TO USE:**
- BEFORE implementing any feature or fix
- When user provides a task description ('add authentication', 'fix bug X')
- When you need comprehensive understanding of a feature area
- Before making architectural changes

**AUTONOMOUS BEHAVIOR:**
Agents MUST call this before implementation work. This is the 'gather intel' step - never skip it.

**BEST FOR:**
Assembling complete context for a specific goal. This is more comprehensive than explore (which finds files) - it builds a full context pack with learnings, decisions, and relationships.`,
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
			Name: "explore_impact",
			Description: `🟡 **IMPORTANT** Analyze the impact of changing a file or symbol.

**WHEN TO USE:**
- BEFORE modifying shared/critical functions or files
- When refactoring or renaming symbols
- When user asks 'what would break if I change X?'
- Before deleting or moving code

**AUTONOMOUS BEHAVIOR:**
Agents should call this before making changes to unfamiliar code, especially in shared modules. Use proactively for safety.

**BEST FOR:**
Impact analysis. Shows what depends on a target (dependents) and what it depends on (dependencies). Critical for preventing breaking changes.`,
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
			Name: "explore_symbols",
			Description: `🟢 **RECOMMENDED** List symbols of a specific kind in the codebase (e.g., all classes, all functions).

**WHEN TO USE:**
- When user asks 'show me all X' (classes, interfaces, etc.)
- To get an overview of available types/functions
- When looking for naming patterns or conventions
- For documentation or refactoring planning

**AUTONOMOUS BEHAVIOR:**
Agents should use this when explicitly requested or when needing a catalog of symbols. This is specialized discovery.

**BEST FOR:**
Getting a catalog of symbols by type. Useful for overview and discovery of available entities in the codebase.`,
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
			Name: "explore_symbol",
			Description: `🟡 **IMPORTANT** Get detailed information about a specific symbol (function, class, method, etc.).

**WHEN TO USE:**
- When you see a symbol name but need to know its signature/location
- When user asks 'what is X?' or 'where is function Y?'
- Before calling or modifying a symbol you're unfamiliar with
- To verify a symbol's parameters before using it

**AUTONOMOUS BEHAVIOR:**
Agents should call this proactively when encountering unfamiliar symbols in code. No need to ask - this is documentation lookup.

**BEST FOR:**
Getting full details about a specific named entity. Returns signature, documentation, location, and type information.`,
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
			Name: "explore_file",
			Description: `🟡 **IMPORTANT** Get all exported symbols in a specific file.

**WHEN TO USE:**
- When you know the file path but need to see what it contains
- Before importing from a file to see available exports
- When user asks 'what's in file X?'
- To understand a file's public API

**AUTONOMOUS BEHAVIOR:**
Agents should call this when examining unfamiliar files or when needing to know available symbols for import.

**BEST FOR:**
Discovering what a file exposes. Returns all exported/public symbols - the file's API surface.`,
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
			Name: "explore_deps",
			Description: `🟢 **RECOMMENDED** Get the dependency graph for files, showing what they import.

**WHEN TO USE:**
- When analyzing module dependencies
- Before moving files to avoid breaking imports
- When user asks 'what does this file depend on?'
- For understanding dependency chains

**AUTONOMOUS BEHAVIOR:**
Use when explicitly needed for dependency analysis. Not required for routine work.

**BEST FOR:**
Dependency visualization. Shows what a file imports and where those imports come from.`,
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
			Name: "explore_callers",
			Description: `🟡 **IMPORTANT** Find all locations that call a function or method. Answers 'who calls this?'

**WHEN TO USE:**
- Before modifying a function to see who depends on it
- When user asks 'where is this function used?'
- To understand usage patterns of a function
- Before changing function signatures

**AUTONOMOUS BEHAVIOR:**
Agents should call this before modifying functions, especially when the function appears to be shared.

**BEST FOR:**
Upstream impact analysis. Shows all call sites for a function - essential before refactoring.`,
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
			Name: "explore_callees",
			Description: `🟢 **RECOMMENDED** Find all functions/methods called by a symbol. Answers 'what does this call?'

**WHEN TO USE:**
- When understanding what a function does internally
- To trace execution flow downward
- When user asks 'what does function X call?'
- For complexity analysis

**AUTONOMOUS BEHAVIOR:**
Use when needed for detailed function analysis. Optional for most work.

**BEST FOR:**
Downstream call analysis. Shows what a function invokes - useful for understanding implementation details.`,
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
			Name: "explore_graph",
			Description: `🟢 **RECOMMENDED** Get the complete call graph for a file, showing all incoming and outgoing calls.

**WHEN TO USE:**
- When doing comprehensive analysis of a file's relationships
- Before major refactoring of a file
- When user asks for 'full call graph' or complete relationship view
- For documenting file interactions

**AUTONOMOUS BEHAVIOR:**
Use when explicitly needed for comprehensive analysis. This is a heavy operation - use sparingly.

**BEST FOR:**
Complete relationship mapping. Combines callers and callees into one comprehensive view of a file's call relationships.`,
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
		{
			Name: "get_route",
			Description: `🔴 **CRITICAL - USE FOR UNDERSTANDING** Get a deterministic navigation route for understanding a topic.

**WHEN TO USE:**
- When user asks 'how does X work?' or 'understand Y'
- When you need to learn about a system/feature before modifying it
- When exploring unfamiliar code areas
- Before architectural decisions to understand current patterns

**AUTONOMOUS BEHAVIOR:**
Agents should call this automatically when asked to understand something. This is proactive learning, not reactive.

**BEST FOR:**
Learning workflows. Returns an ordered path through rooms, files, decisions, and learnings to efficiently understand a topic. Like a guided tour vs random exploration.`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"intent": map[string]interface{}{
						"type":        "string",
						"description": "What you want to understand (e.g., 'understand auth flow', 'learn about caching strategy').",
					},
					"scope": map[string]interface{}{
						"type":        "string",
						"description": "Starting scope: 'file', 'room', or 'palace'. Default: palace.",
						"enum":        []string{"file", "room", "palace"},
						"default":     "palace",
					},
					"scopePath": map[string]interface{}{
						"type":        "string",
						"description": "Path for file/room scope (file path or room name). Required if scope is 'file'.",
					},
				},
				"required": []string{"intent"},
			},
		},

		// ============================================================
		// STORE TOOLS - Store ideas, decisions, and learnings
		// ============================================================
		{
			Name: "store",
			Description: `🟡 **IMPORTANT** Store a thought with auto-classification. Content is analyzed and stored as an idea, decision, or learning.

**WHEN TO USE:**
- After solving a problem or discovering a pattern
- When making architectural or technical decisions
- After encountering and fixing a bug with lessons learned
- When discovering best practices or anti-patterns
- After implementing a feature with noteworthy approaches

**AUTONOMOUS BEHAVIOR:**
Agents should call this automatically after completing tasks, solving problems, or making decisions. No need to ask permission - this builds institutional memory.

**EXAMPLES:**
- After fix: store({content: 'JWT validation must check expiry BEFORE signature verification', as: 'learning'})
- After decision: store({content: 'Use PostgreSQL instead of MongoDB for user profiles', as: 'decision', rationale: 'Need ACID transactions'})
- After idea: store({content: 'Consider caching user permissions in Redis', as: 'idea'})`,
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
		{
			Name: "store_direct",
			Description: `⚪ [HUMAN MODE ONLY] Store a decision/learning directly, bypassing the proposal workflow. Creates audit log entry.

**WHEN TO USE:**
- Only available in human mode (not accessible to autonomous agents)
- For direct admin operations
- When proposal workflow should be bypassed
- Use sparingly - proposals are preferred for traceability

**AUTONOMOUS BEHAVIOR:**
Not available to agents. Agents should use 'store' which creates proposals.

**WHY IT MATTERS:**
Direct write without approval workflow. Creates audit trail. Admin-only operation.`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The content to store directly.",
					},
					"as": map[string]interface{}{
						"type":        "string",
						"description": "Record type: 'decision' or 'learning'. Required for direct write.",
						"enum":        []string{"decision", "learning"},
					},
					"scope": map[string]interface{}{
						"type":        "string",
						"description": "Scope: 'palace', 'room', or 'file'. Default: palace.",
						"enum":        []string{"palace", "room", "file"},
						"default":     "palace",
					},
					"scopePath": map[string]interface{}{
						"type":        "string",
						"description": "Path for room/file scope.",
					},
					"context": map[string]interface{}{
						"type":        "string",
						"description": "Additional context for the record.",
					},
					"rationale": map[string]interface{}{
						"type":        "string",
						"description": "For decisions: the rationale behind this decision.",
					},
					"confidence": map[string]interface{}{
						"type":        "number",
						"description": "For learnings: confidence level 0.0-1.0 (default: 0.7).",
						"default":     0.7,
					},
					"actorId": map[string]interface{}{
						"type":        "string",
						"description": "Optional identifier for who performed this direct write (for audit).",
					},
				},
				"required": []string{"content", "as"},
			},
		},

		// ============================================================
		// GOVERNANCE TOOLS - Approve/reject proposals (human mode only)
		// ============================================================
		{
			Name: "approve",
			Description: `⚪ [HUMAN MODE ONLY] Approve a pending proposal, creating the corresponding decision/learning with 'approved' authority.

**WHEN TO USE:**
- Only available in human mode
- To approve agent-created proposals
- For governance workflow
- When reviewing pending proposals

**AUTONOMOUS BEHAVIOR:**
Not available to agents. Humans use this to approve agent proposals.

**WHY IT MATTERS:**
Human approval mechanism for agent proposals. Enables collaborative knowledge curation.`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"proposalId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the proposal to approve (e.g., 'prop_abc123').",
					},
					"by": map[string]interface{}{
						"type":        "string",
						"description": "Name or identifier of who is approving.",
					},
					"note": map[string]interface{}{
						"type":        "string",
						"description": "Optional note about why this was approved.",
					},
				},
				"required": []string{"proposalId"},
			},
		},
		{
			Name: "reject",
			Description: `⚪ [HUMAN MODE ONLY] Reject a pending proposal with reason. Proposal marked as rejected and not promoted.

**WHEN TO USE:**
- Only available in human mode
- To reject agent-created proposals
- When proposal is incorrect/unwanted
- For governance workflow

**AUTONOMOUS BEHAVIOR:**
Not available to agents. Humans use this to reject agent proposals.

**WHY IT MATTERS:**
Human rejection mechanism for agent proposals. Maintains quality control.`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"proposalId": map[string]interface{}{
						"type":        "string",
						"description": "ID of the proposal to reject (e.g., 'prop_abc123').",
					},
					"by": map[string]interface{}{
						"type":        "string",
						"description": "Name or identifier of who is rejecting.",
					},
					"note": map[string]interface{}{
						"type":        "string",
						"description": "Reason for rejection (recommended).",
					},
				},
				"required": []string{"proposalId"},
			},
		},

		// ============================================================
		// RECALL TOOLS - Retrieve knowledge and manage relationships
		// ============================================================
		{
			Name: "recall",
			Description: `🟢 **RECOMMENDED** Retrieve learnings, optionally filtered by scope or search query.

**WHEN TO USE:**
- When you need past learnings related to current work
- If user asks 'what have we learned about X?'
- Before implementing to check for existing patterns/solutions
- When you want to reference historical knowledge

**AUTONOMOUS BEHAVIOR:**
Agents should proactively search learnings when working in areas that might have related knowledge. Use when context might help.

**EXAMPLES:**
- recall({query: 'authentication'}) - Find auth-related learnings
- recall({scope: 'file', scopePath: 'auth/jwt.go'}) - File-specific learnings`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "ID of a specific learning to retrieve. If provided, returns only that record.",
					},
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
			Name: "recall_decisions",
			Description: `🟢 **RECOMMENDED** Retrieve decisions, optionally filtered by status, scope, or search query.

**WHEN TO USE:**
- Before making architectural choices to see existing decisions
- When user asks 'why did we choose X?'
- To check if a decision already exists for something you're about to decide
- When implementing features to follow established patterns

**AUTONOMOUS BEHAVIOR:**
Agents should check for relevant decisions before making new ones. This prevents contradicting existing decisions.

**EXAMPLES:**
- recall_decisions({query: 'database'}) - Find DB-related decisions
- recall_decisions({status: 'active', scope: 'room', scopePath: 'api'}) - Active API decisions`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "ID of a specific decision to retrieve. If provided, returns only that record.",
					},
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
			Name: "recall_ideas",
			Description: `🟢 **RECOMMENDED** Retrieve ideas, optionally filtered by status, scope, or search query.

**WHEN TO USE:**
- Before implementing features to check if already explored
- When user asks 'what ideas do we have for X?'
- To avoid duplicate exploration of ideas
- When looking for inspiration or alternatives

**AUTONOMOUS BEHAVIOR:**
Agents can check ideas when relevant, but it's not critical for most workflows.

**EXAMPLES:**
- recall_ideas({status: 'exploring'}) - Active ideas being explored
- recall_ideas({query: 'performance'}) - Performance-related ideas`,
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
			Name: "recall_outcome",
			Description: `🟡 **IMPORTANT** Record the outcome of a decision. Tracks whether decisions worked out.

**WHEN TO USE:**
- After implementing a decision and seeing results
- When decision outcome is clear (success/failed/mixed)
- To enable decision feedback loop
- When user says 'that approach worked' or 'that decision was wrong'

**AUTONOMOUS BEHAVIOR:**
Offer to record decision outcome when implementation is complete and results are known. Ask user for outcome assessment.

**WHY IT MATTERS:**
Enables learning from decisions. Tracks which decisions worked. Feeds into recall_learning_link for automatic confidence updates.`,
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
			Name: "recall_link",
			Description: `🟢 **RECOMMENDED** Create a relationship between records (ideas, decisions, learnings, code files).

**WHEN TO USE:**
- When two records are related (supports, contradicts, implements, etc.)
- To build knowledge graph connections
- When decision implements an idea
- When learning supersedes another learning

**AUTONOMOUS BEHAVIOR:**
Create links when relationship is explicit. Example: when storing decision that implements an idea, link them with 'implements' relation.

**WHY IT MATTERS:**
Builds knowledge graph. Enables navigation between related concepts. Shows how ideas become decisions become learnings.`,
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
			Name: "recall_links",
			Description: `🟢 **RECOMMENDED** Get all links for a record (as source, target, or both).

**WHEN TO USE:**
- To see what's related to current record
- When exploring knowledge graph
- To understand relationships between concepts
- When user asks 'what's connected to this?'

**AUTONOMOUS BEHAVIOR:**
Optional. Use when user wants to explore relationships or when understanding context requires seeing connected records.

**BEST FOR:**
Navigating knowledge graph. Discovering related ideas, decisions, learnings.`,
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
			Name: "recall_unlink",
			Description: `🟢 **RECOMMENDED** Delete a link by its ID.

**WHEN TO USE:**
- When link is incorrect or no longer valid
- To clean up knowledge graph
- When user says 'those aren't actually related'
- For maintaining accurate relationships

**AUTONOMOUS BEHAVIOR:**
Use only with user confirmation. Don't auto-delete links.

**WHY IT MATTERS:**
Maintains knowledge graph accuracy by removing incorrect relationships.`,
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
			Name: "brief",
			Description: `🔴 **CRITICAL - CALL SECOND** Get a comprehensive briefing before working. Shows active agents, conflicts, learnings, hotspots.

**WHEN TO USE:**
- IMMEDIATELY after starting a session (right after session_start)
- Before beginning work on any file or feature
- When the user asks for context about the project/workspace
- After long periods of inactivity (to catch up on changes)
- When switching between different areas of the codebase

**AUTONOMOUS BEHAVIOR:**
Agents MUST call this automatically at the start of each conversation/task without asking. This is essential context gathering, not optional.

**WHY IT MATTERS:**
Briefings prevent redundant work, surface relevant learnings and decisions, warn about conflicts, and show hotspots that need careful handling. Working without a briefing is like coding blindfolded - you miss critical context that could save time and prevent mistakes.`,
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
			Name: "brief_file",
			Description: `🟡 **IMPORTANT** Get intelligence about a file: edit history, failure rate, associated learnings.

**WHEN TO USE:**
- Before editing a file you haven't touched before in this session
- When asked to modify a file flagged as a 'hotspot' in the briefing
- Before making significant changes to critical/fragile files
- When the user mentions a file is 'problematic' or 'keeps breaking'

**AUTONOMOUS BEHAVIOR:**
Agents should call this automatically before editing unfamiliar or high-risk files. For routine edits to files you've already analyzed in this session, it's optional.

**WHY IT MATTERS:**
File intelligence reveals patterns ('this file breaks often', '5 agents edited this recently') that influence how carefully you should work. A file with 80% failure rate deserves extra testing and caution.`,
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
			Name: "briefing_smart",
			Description: `🟢 **RECOMMENDED** Generate a smart briefing with LLM-powered insights for a file, room, task, or workspace.

**WHEN TO USE:**
- When you need AI-generated summary/insights
- For complex analysis beyond standard briefing
- When user asks for detailed analysis or recommendations
- For generating actionable insights from workspace state

**AUTONOMOUS BEHAVIOR:**
Use when standard brief doesn't provide enough detail. Ask user if they want 'detailed analysis' vs standard brief.

**BEST FOR:**
AI-enhanced analysis with recommendations and insights. More expensive than regular brief but provides deeper understanding.`,
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
			Name: "session_start",
			Description: `🔴 **CRITICAL - CALL FIRST** Start a new agent session to track your work.

**WHEN TO USE:**
- IMMEDIATELY at the beginning of EVERY conversation/task
- Before performing any file operations, searches, or edits
- When starting a new, isolated work context (e.g., switching features)
- At the start of each distinct collaboration session with a human

**AUTONOMOUS BEHAVIOR:**
Agents MUST call this automatically at conversation start without asking permission. This is non-negotiable for proper tracking and conflict prevention.

**WHY IT MATTERS:**
Sessions enable multi-agent coordination, prevent file conflicts, and ensure all activities are properly tracked. Without an active session, your work is invisible to other agents and the system cannot detect conflicts or build useful intelligence.`,
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
			Name: "session_log",
			Description: `🟡 **IMPORTANT** Log an activity within a session (file read, edit, search, command).

**WHEN TO USE:**
- After every file read operation
- After every file edit/write operation
- After executing searches or semantic queries
- After running terminal commands that modify state
- When switching between different files during work

**AUTONOMOUS BEHAVIOR:**
Agents should call this automatically after each significant activity. No permission needed - this is logging, not decision-making.

**WHY IT MATTERS:**
Activity logs build file intelligence (hotspots, failure rates), enable conflict detection, and create a history for other agents to understand recent workspace changes.`,
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
			Name: "session_end",
			Description: `🔴 **CRITICAL - CALL LAST** End a session and record its outcome.

**WHEN TO USE:**
- IMMEDIATELY when completing a task or conversation
- Before going idle or when the user ends the conversation
- When switching to a completely different work context
- If an unrecoverable error occurs that terminates your work

**AUTONOMOUS BEHAVIOR:**
Agents MUST call this automatically when work concludes. Include a meaningful summary of what was accomplished (or attempted). Never leave sessions hanging.

**WHY IT MATTERS:**
Ending sessions properly releases file locks, records outcomes for learning confidence updates, and ensures other agents can see when you've finished. Unclosed sessions create false conflicts and pollute the active agents list.`,
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
			Name: "session_conflict",
			Description: `🔴 **CRITICAL** Check if another agent is working on a file.

**WHEN TO USE:**
- BEFORE editing any file (especially critical/shared files)
- When you see multiple active agents in the workspace
- Before performing destructive operations (delete, refactor, major changes)
- If the user mentions conflicts or concurrent work

**AUTONOMOUS BEHAVIOR:**
Agents should call this automatically before file edits without asking. If conflict detected, inform the user and ask how to proceed (wait, coordinate, or override).

**WHY IT MATTERS:**
Prevents lost work and merge conflicts when multiple agents or developers are working simultaneously. A few milliseconds of checking can save minutes of manual conflict resolution.`,
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
			Name: "session_list",
			Description: `🟢 **RECOMMENDED** List all agent sessions, optionally filtered by status.

**WHEN TO USE:**
- When debugging session-related issues
- If the user asks 'who else is working here?' or 'what agents are active?'
- Before starting work in a busy/shared workspace to understand context
- When investigating unexpected conflicts or behaviors

**AUTONOMOUS BEHAVIOR:**
Agents should ask before calling unless investigating a specific issue the user reported. This is informational, not required for normal workflows.

**WHY IT MATTERS:**
Provides visibility into workspace activity and helps diagnose coordination issues. Useful for understanding who's working on what, but not needed for every task.`,
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
			Name: "conversation_store",
			Description: `🟢 **RECOMMENDED** Store the current conversation for future context.

**WHEN TO USE:**
- At the end of important/complex discussions
- When user explicitly asks to 'save this conversation'
- After solving tricky problems with valuable context
- When conversation contains decisions or insights worth preserving

**AUTONOMOUS BEHAVIOR:**
Agents should ask if user wants to save conversation after significant work. Don't auto-save every chat - be selective.

**BEST FOR:**
Preserving important discussions for later context recall and analysis.`,
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
			Name: "conversation_search",
			Description: `🟢 **RECOMMENDED** Search past conversations by summary or content.

**WHEN TO USE:**
- When user asks 'what did we discuss about X?'
- To find previous conversation context
- When trying to recall past discussions
- For retrieving historical conversation insights

**AUTONOMOUS BEHAVIOR:**
Use when user explicitly asks about past conversations. Don't search conversations unless user requests it.

**BEST FOR:**
Finding previously saved conversations by topic or content.`,
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
			Name: "conversation_extract",
			Description: `🟡 **IMPORTANT** Extract ideas, decisions, and learnings from a stored conversation using AI analysis. Automatically stores extracted records.

**WHEN TO USE:**
- After storing an important conversation
- When user says 'extract insights from that conversation'
- To mine stored conversations for knowledge
- For converting discussions into actionable records

**AUTONOMOUS BEHAVIOR:**
Offer to extract insights after storing important conversations. Ask user if they want AI to mine the conversation for ideas/decisions/learnings.

**WHY IT MATTERS:**
Automatically converts conversation into structured knowledge. Saves manual work of identifying and storing insights.`,
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
			Name: "search_semantic",
			Description: `🟢 **RECOMMENDED** Perform semantic search using AI embeddings. Finds conceptually similar content even without exact word matches.

**WHEN TO USE:**
- When keyword search (explore) returns no results
- When searching for concepts vs exact terms
- When user asks abstract questions like 'error handling patterns'
- For finding similar solutions to current problem

**AUTONOMOUS BEHAVIOR:**
Use as fallback when exact search fails, or when user query is conceptual. Requires embedding backend configured.

**BEST FOR:**
Conceptual search. Example: searching 'retry logic' finds records about 'exponential backoff' even without keyword match.`,
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
			Name: "search_hybrid",
			Description: `🟢 **RECOMMENDED** Combined keyword + semantic search. Returns results matching either exact words or conceptual meaning.

**WHEN TO USE:**
- When you want comprehensive search (both exact and conceptual)
- When unsure if exact terms exist in knowledge base
- For best-of-both-worlds search coverage
- As default search for learnings/decisions when embeddings are available

**AUTONOMOUS BEHAVIOR:**
Prefer this over search_semantic when available - it's more comprehensive. Falls back to keyword-only if embeddings not configured.

**BEST FOR:**
Maximum recall. Combines precision of keyword search with breadth of semantic search.`,
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
			Name: "search_similar",
			Description: `🟢 **RECOMMENDED** Find records similar to a given record ID. Uses semantic similarity to find related content.

**WHEN TO USE:**
- When you want 'more like this' functionality
- To find records similar to current context
- When user asks 'what else is like this?'
- For discovering related knowledge based on similarity

**AUTONOMOUS BEHAVIOR:**
Use when exploring related content. Requires embeddings to be configured.

**BEST FOR:**
Discovering conceptually similar records. Like 'related articles' feature.`,
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
			Name: "embedding_sync",
			Description: `🟢 **RECOMMENDED** Generate embeddings for records that don't have them. Useful for backfilling after enabling embeddings or after bulk imports.

**WHEN TO USE:**
- When semantic search returns incomplete results
- After enabling embeddings feature
- Following bulk knowledge imports
- When user asks 'why is semantic search not working?'

**AUTONOMOUS BEHAVIOR:**
Suggest running this if semantic search seems incomplete. Requires embedding backend configured.

**BEST FOR:**
Ensuring all records have embeddings for semantic search. One-time or periodic maintenance operation.`,
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
			Name: "embedding_stats",
			Description: `🟢 **RECOMMENDED** Get statistics about the embedding system including total embeddings, pending records, and pipeline status.

**WHEN TO USE:**
- When debugging semantic search issues
- If user asks 'is semantic search working?'
- To check embedding coverage
- For monitoring embedding system health

**AUTONOMOUS BEHAVIOR:**
Use when user has questions about semantic search availability or coverage.

**BEST FOR:**
Monitoring embedding system health and coverage statistics.`,
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},

		// ============================================================
		// LEARNING LIFECYCLE TOOLS - Manage learning status and feedback
		// ============================================================
		{
			Name: "recall_learning_link",
			Description: `🟢 **RECOMMENDED** Link a learning to a decision for outcome feedback. When decision outcome is recorded, learning confidence auto-updates.

**WHEN TO USE:**
- When storing learning based on a decision
- To enable automatic confidence updates when decision outcomes are known
- When user wants learning quality to improve based on decision results
- For creating feedback loops between decisions and learnings

**AUTONOMOUS BEHAVIOR:**
Offer to link learning to decision when both are related. This enables future automatic confidence adjustment.

**WHY IT MATTERS:**
Creates feedback loop: if decision succeeds, linked learning's confidence increases. If decision fails, confidence decreases. Self-improving knowledge system.`,
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
			Name: "recall_obsolete",
			Description: `🟡 **IMPORTANT** Mark a learning as obsolete with reason. Obsolete learnings are preserved but hidden from active queries.

**WHEN TO USE:**
- When learning is no longer valid/applicable
- If approach has been superseded by better solution
- When user says 'we don't do this anymore'
- To maintain historical context while removing from active knowledge

**AUTONOMOUS BEHAVIOR:**
When storing new learning that contradicts existing one, ask user if old learning should be marked obsolete.

**WHY IT MATTERS:**
Preserves history without polluting active knowledge. Better than deleting - maintains 'why we stopped doing X' context.`,
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
			Name: "recall_archive",
			Description: `🟢 **RECOMMENDED** Archive old, low-confidence learnings. Archived learnings are preserved but hidden from active queries.

**WHEN TO USE:**
- For routine knowledge base cleanup
- When learnings haven't been accessed in months
- If low-confidence learnings are cluttering results
- As maintenance operation (not critical for daily work)

**AUTONOMOUS BEHAVIOR:**
Suggest archival only if user complains about too many/irrelevant learnings. Never auto-archive without permission.

**BEST FOR:**
Maintaining clean knowledge base by hiding unused, low-confidence content while preserving historical record.`,
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
			Name: "recall_learnings_by_status",
			Description: `🟢 **RECOMMENDED** Retrieve learnings by lifecycle status: 'active', 'obsolete', or 'archived'.

**WHEN TO USE:**
- When user asks 'what did we stop doing?' (obsolete)
- To review archived learnings
- For auditing learning lifecycle
- When debugging why learning isn't appearing (might be obsolete/archived)

**AUTONOMOUS BEHAVIOR:**
Use when user explicitly asks about historical/obsolete knowledge. Default queries should use active learnings only.

**BEST FOR:**
Exploring learning history and understanding knowledge evolution over time.`,
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
			Name: "recall_contradictions",
			Description: `🟢 **RECOMMENDED** Find records that contradict a given record using AI semantic analysis. Uses embeddings + LLM to detect contradictions.

**WHEN TO USE:**
- Before storing new learning/decision to check for conflicts
- When user mentions conflicting information
- For knowledge base quality assurance
- When debugging inconsistent behavior

**AUTONOMOUS BEHAVIOR:**
Optional but valuable. Consider running when storing important decisions/learnings. Requires embedding backend configured.

**BEST FOR:**
Maintaining knowledge base consistency. Finds semantic contradictions that keyword search would miss.`,
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
			Name: "recall_contradiction_check",
			Description: `🟢 **RECOMMENDED** Check if two specific records contradict each other using AI analysis.

**WHEN TO USE:**
- When user suspects two records conflict
- To validate consistency between related learnings/decisions
- For debugging contradictory information
- When user asks 'do these contradict?'

**AUTONOMOUS BEHAVIOR:**
Use only when user explicitly questions consistency between two records.

**BEST FOR:**
Directed contradiction checking between specific records.`,
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
			Name: "recall_contradiction_summary",
			Description: `🟢 **RECOMMENDED** Get summary of all known contradictions in the system.

**WHEN TO USE:**
- For knowledge base health checks
- When user asks about knowledge quality/consistency
- As periodic maintenance task
- When debugging conflicting behaviors

**AUTONOMOUS BEHAVIOR:**
Optional quality check. Suggest running periodically but not required for daily work.

**BEST FOR:**
Knowledge base health monitoring and conflict resolution.`,
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
			Name: "decay_stats",
			Description: `🟢 **RECOMMENDED** Get statistics about confidence decay for learnings (at-risk count, decayed count, averages).

**WHEN TO USE:**
- When user asks about knowledge health/staleness
- Before deciding to run decay_apply
- For monitoring knowledge base freshness
- To understand decay system state

**AUTONOMOUS BEHAVIOR:**
Optional monitoring tool. Use when user asks about decay status or knowledge staleness.

**BEST FOR:**
Monitoring decay system metrics and knowledge base health statistics.`,
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name: "decay_preview",
			Description: `🟢 **RECOMMENDED** Preview which learnings would be affected by applying decay. Does not modify anything.

**WHEN TO USE:**
- Before running decay_apply to see impact
- When user asks 'what knowledge is becoming stale?'
- For testing decay configuration
- To understand decay system behavior

**AUTONOMOUS BEHAVIOR:**
ALWAYS call this before decay_apply. Never apply decay without previewing first.

**WHY IT MATTERS:**
Safe preview of decay candidates. Shows what would be affected without making changes. Essential safety check.`,
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
			Name: "decay_apply",
			Description: `🟡 **IMPORTANT** Apply confidence decay to inactive learnings. Reduces confidence of learnings that haven't been accessed.

**WHEN TO USE:**
- After reviewing decay_preview results
- When user approves decay application
- As maintenance task to keep knowledge fresh
- When knowledge base has stale content

**AUTONOMOUS BEHAVIOR:**
NEVER call without user permission. MUST call decay_preview first, show results, get approval, then apply.

**WHY IT MATTERS:**
Reduces confidence of inactive learnings. Semi-destructive - lowers confidence scores based on staleness.`,
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name: "decay_reinforce",
			Description: `🟢 **RECOMMENDED** Reinforce a learning to prevent decay (marks it as recently accessed).

**WHEN TO USE:**
- When learning proves valuable during current work
- To prevent important learning from decaying
- When user says 'this is still relevant'
- After successfully applying a learning

**AUTONOMOUS BEHAVIOR:**
Consider reinforcing learnings that help solve current problem. Shows they're still valuable.

**WHY IT MATTERS:**
Resets decay timer. Indicates learning is still useful, preventing confidence degradation over time.`,
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
			Name: "decay_boost",
			Description: `🟢 **RECOMMENDED** Boost a learning's confidence (opposite of decay).

**WHEN TO USE:**
- When learning proves especially valuable
- To manually increase confidence of undervalued knowledge
- When user says 'this is really important'
- To promote high-quality learnings

**AUTONOMOUS BEHAVIOR:**
Rarely needed. Consider boosting when learning is exceptionally helpful, but ask user first.

**WHY IT MATTERS:**
Manually increases confidence score. Opposite of decay - promotes valuable knowledge.`,
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
			Name: "context_auto_inject",
			Description: `🔴 **CRITICAL - CALL BEFORE EVERY FILE EDIT** Get automatically assembled context for a file. Returns prioritized learnings, decisions, failures, and warnings.

**WHEN TO USE:**
- BEFORE editing ANY file (this is non-negotiable)
- When opening a file for the first time in a session
- When the user asks 'what should I know about this file?'
- Before making significant changes or refactoring

**AUTONOMOUS BEHAVIOR:**
Agents MUST call this automatically before every file edit. Treat this as mandatory 'read file metadata' - just as you wouldn't edit a file without reading it first, don't edit without getting its context.

**WHY IT MATTERS:**
This tool provides file-scoped learnings ('never use X in this file'), decisions ('this file must use Y pattern'), and failure warnings ('this breaks 40% of the time'). Editing without this context is like performing surgery without reading the patient's chart.`,
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
			Name: "scope_explain",
			Description: `🟢 **RECOMMENDED** Explain the scope inheritance chain for a file. Shows which learnings/decisions apply at each level.

**WHEN TO USE:**
- When debugging why certain context appears for a file
- If user asks 'why am I seeing these learnings?'
- To understand scope hierarchy (file → room → palace → corridor)
- When troubleshooting context issues

**AUTONOMOUS BEHAVIOR:**
Use only when debugging or when user asks. Not needed for routine work.

**BEST FOR:**
Understanding how scope inheritance works and why certain context is included for a file.`,
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
			Name: "store_postmortem",
			Description: `🟡 **IMPORTANT** Create a postmortem record to document a failure.

**WHEN TO USE:**
- After fixing a significant bug or outage
- When a decision leads to production issues
- After discovering a major flaw in implementation
- When user says 'that didn't work' or 'we need to remember this failure'

**AUTONOMOUS BEHAVIOR:**
Agents should offer to create postmortems after fixing issues, especially if lessons were learned. Ask user if they want to document it.

**EXAMPLES:**
- After fixing security bug: store_postmortem({title: 'JWT validation bypass', what_happened: '...', root_cause: '...', lessons_learned: [...]})`,
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
			Name: "get_postmortems",
			Description: `🟢 **RECOMMENDED** Retrieve postmortem records with optional filters for status and severity.

**WHEN TO USE:**
- When user asks 'what failures have we had?'
- To learn from past mistakes before implementing similar features
- For reviewing historical failure patterns
- When investigating recurring issues

**AUTONOMOUS BEHAVIOR:**
Use when working in areas with known failures. Check postmortems before implementing features similar to past failures.

**BEST FOR:**
Learning from past failures. Understanding what went wrong and how it was prevented.`,
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
			Name: "get_postmortem",
			Description: `🟢 **RECOMMENDED** Get detailed information about a specific postmortem.

**WHEN TO USE:**
- When user references a specific failure
- To get full details of a postmortem found in search
- For deep-diving into past failure
- When investigating root causes

**AUTONOMOUS BEHAVIOR:**
Use when user asks about specific past failure or when postmortem ID is referenced.

**BEST FOR:**
Getting complete postmortem details including lessons learned and prevention steps.`,
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
			Name: "resolve_postmortem",
			Description: `🟡 **IMPORTANT** Mark a postmortem as resolved.

**WHEN TO USE:**
- After implementing prevention steps from postmortem
- When root cause is fully addressed
- When user confirms issue is resolved
- For closing out postmortem lifecycle

**AUTONOMOUS BEHAVIOR:**
Offer to resolve postmortem after implementing prevention steps. Get user confirmation first.

**WHY IT MATTERS:**
Tracks postmortem lifecycle. Shows which failures have been addressed vs still open.`,
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
			Name: "postmortem_stats",
			Description: `🟢 **RECOMMENDED** Get aggregated statistics about postmortems (counts by status and severity).

**WHEN TO USE:**
- When user asks about failure patterns/trends
- For project health monitoring
- To understand failure landscape
- When user wants overview of failure history

**AUTONOMOUS BEHAVIOR:**
Optional. Use when user asks about project health or failure statistics.

**BEST FOR:**
High-level postmortem metrics and failure trend analysis.`,
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name: "postmortem_to_learnings",
			Description: `🟡 **IMPORTANT** Convert postmortem's lessons learned into individual learning records. Creates learnings linked to the postmortem.

**WHEN TO USE:**
- After creating a postmortem with valuable lessons
- To formalize learnings from failure
- When user wants to extract knowledge from postmortem
- For converting failure insights into actionable learnings

**AUTONOMOUS BEHAVIOR:**
Offer to extract learnings after creating postmortem. Ask user if they want lessons converted to formal learning records.

**WHY IT MATTERS:**
Transforms postmortem insights into queryable, reusable learnings. Prevents same mistakes by making lessons discoverable.`,
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
			Name: "corridor_learnings",
			Description: `🟢 **RECOMMENDED** Get personal learnings from the global corridor (cross-workspace knowledge).

**WHEN TO USE:**
- When workspace-specific learnings don't provide answers
- To leverage knowledge from other projects
- When user asks 'have we dealt with this before elsewhere?'
- For general best practices that apply across projects

**AUTONOMOUS BEHAVIOR:**
Use when workspace learnings are insufficient. This is advanced/optional feature.

**BEST FOR:**
Accessing your personal knowledge base that follows you across all projects. Like 'global learnings'.`,
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
			Name: "corridor_links",
			Description: `🟢 **RECOMMENDED** Get all linked workspaces in the global corridor.

**WHEN TO USE:**
- When user asks 'what projects am I tracking?'
- To see which workspaces contribute to corridor knowledge
- For understanding corridor scope
- When debugging corridor behavior

**AUTONOMOUS BEHAVIOR:**
Optional informational tool. Use when user wants to understand corridor setup.

**BEST FOR:**
Inspecting corridor configuration and workspace linkage.`,
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name: "corridor_stats",
			Description: `🟢 **RECOMMENDED** Get statistics about the personal corridor (learning count, linked workspaces).

**WHEN TO USE:**
- When user asks 'how much corridor knowledge do I have?'
- For understanding corridor knowledge base size
- To monitor cross-workspace learning growth
- When user wants corridor overview

**AUTONOMOUS BEHAVIOR:**
Optional informational tool. Use when user asks about corridor metrics.

**BEST FOR:**
Corridor health monitoring and metrics overview.`,
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name: "corridor_promote",
			Description: `🟡 **IMPORTANT** Promote a workspace learning to the personal corridor for cross-workspace use.

**WHEN TO USE:**
- When learning applies to all your projects, not just current one
- If user says 'I want to use this everywhere'
- For general best practices worth sharing across workspaces
- When creating universal patterns/standards

**AUTONOMOUS BEHAVIOR:**
Offer to promote learning when it's clearly general-purpose (e.g., 'always validate user input'). Ask user permission.

**WHY IT MATTERS:**
Makes learning available across all your projects. Creates personal knowledge that follows you everywhere.`,
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
			Name: "corridor_reinforce",
			Description: `🟢 **RECOMMENDED** Increase confidence for a personal corridor learning (marks it as useful).

**WHEN TO USE:**
- When corridor learning proves valuable in current project
- To boost confidence of general-purpose knowledge
- When user says 'this corridor advice was helpful'
- After successfully applying corridor learning

**AUTONOMOUS BEHAVIOR:**
Reinforce corridor learnings that help solve problems. Shows they're valuable across projects.

**WHY IT MATTERS:**
Increases confidence of cross-workspace knowledge. Signals that general knowledge is working well.`,
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
