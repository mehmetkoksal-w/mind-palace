package commands

import (
	"fmt"
	"strings"
)

func init() {
	Register(&Command{
		Name:        "help",
		Aliases:     []string{"-h", "--help"},
		Description: "Show help for a command or topic",
		Run:         RunHelp,
	})
}

// RunHelp executes the help command with parsed arguments.
func RunHelp(args []string) error {
	if len(args) == 0 {
		return ShowUsage()
	}

	topic := strings.ToLower(strings.TrimSpace(args[0]))
	return ShowHelpTopic(topic)
}

// ShowUsage displays the main usage message.
func ShowUsage() error {
	fmt.Print(`palace - AI-friendly codebase memory and search

CORE COMMANDS
  explore   Search code, get context, or trace call relationships
  store     Store knowledge in the palace (idea, decision, or learning)
  recall    Retrieve knowledge from the palace
  brief     Get briefing on workspace or file (status, agents, intel)

SETUP & INDEX
  build     Build the palace (initialize in current directory)
  scan      Build/refresh the code index
  check     Verify index freshness and optionally generate CI outputs

SERVICES
  serve     Start MCP server for AI agents
  dashboard Start web dashboard for visualization

AGENTS & SESSIONS
  session   Manage agent sessions (start, end, list, show)

CROSS-WORKSPACE
  corridor  Cross-workspace knowledge sharing

CI/CD
  ci        CI/CD integration commands (shortcuts for check flags)

HOUSEKEEPING
  clean     Clean up stale data (sessions, agents, learnings)
  update    Update palace to latest version
  help      Show help for a command or topic
  version   Show version information

EXPLORE EXAMPLES
  palace explore "auth logic"              # Search for code
  palace explore "add auth" --full         # Full context with learnings
  palace explore --map handleAuth          # Who calls handleAuth?
  palace explore --map Search --file x.go  # What does Search call?
  palace explore --map --file cli.go       # Full file call graph

STORE & RECALL EXAMPLES
  palace store "Let's use JWT for auth"    # Auto-classified as decision
  palace store "What if we cache?" --as idea
  palace store "Always test first" --as learning

  palace recall                            # List learnings
  palace recall "auth"                     # Search knowledge
  palace recall --type decision            # List decisions
  palace recall --pending                  # Decisions awaiting outcome
  palace recall update d_123 success       # Record decision outcome
  palace recall link i_1 --supports d_2    # Link records

BRIEF EXAMPLES
  palace brief                             # Workspace briefing
  palace brief src/auth.go                 # File-specific briefing
  palace brief --sessions                  # Include session details

OTHER EXAMPLES
  palace build                             # Initialize palace
  palace scan                              # Index the codebase
  palace check                             # Verify index is fresh
  palace check --collect --diff HEAD~1     # CI: verify and collect
  palace serve                             # Start MCP server
  palace session start --agent claude      # Start a session
  palace corridor search "auth"            # Search across workspaces

Run 'palace help <command>' for detailed help on a command.
`)
	return nil
}

// ShowHelpTopic displays help for a specific topic.
func ShowHelpTopic(topic string) error {
	switch topic {
	case "build", "enter", "init":
		fmt.Print(`palace build - Build the palace (initialize)

Usage: palace build [options]

Options:
  --root <path>     Workspace root (default: current directory)
  --force           Overwrite existing curated files
  --detect          Auto-detect project type and generate profile
  --with-outputs    Also create generated outputs

Examples:
  palace build
  palace build --detect
  palace build --force --detect
  palace init           # Alias for build
  palace enter          # Alias for build
`)
	case "scan":
		fmt.Print(`palace scan - Build or refresh the code index

Usage: palace scan [options]

Options:
  --root <path>    Workspace root (default: current directory)

The scan command:
  - Parses all code files using Tree-sitter
  - Extracts symbols (functions, classes, methods, etc.)
  - Tracks relationships (imports, calls)
  - Stores everything in .palace/index/palace.db

Examples:
  palace scan
  palace scan --root /path/to/project
`)
	case "check", "verify":
		fmt.Print(`palace check - Verify index freshness

Usage: palace check [options]

Options:
  --root <path>       Workspace root (default: current directory)
  --diff <range>      Scope verification to git diff range
  --strict            Hash all files (slower but thorough)
  --collect           Also generate context pack from diff
  --signal            Also generate change signal from diff
  --allow-stale       For --collect: allow even if index is stale

The check command:
  - Validates palace configuration files (lint)
  - Compares workspace files against the index
  - Reports stale files that need re-scanning
  - Optionally generates CI outputs (context pack, change signal)

Examples:
  palace check
  palace check --strict
  palace check --diff HEAD~1..HEAD
  palace check --diff HEAD~1..HEAD --collect --signal   # CI mode
`)
	case "explore", "query", "context", "graph":
		fmt.Print(`palace explore - Search code, get context, or trace calls

Usage: palace explore <query> [options]
       palace explore --map <symbol> [--file <path>]
       palace explore --map --file <path>

Options:
  --root <path>     Workspace root (default: current directory)
  --room <name>     Filter to specific room
  --limit <n>       Maximum results (default: 10)
  --fuzzy           Enable fuzzy matching for typo tolerance
  --full            Get full context including learnings and decisions
  --map <symbol>    Trace call graph for a symbol
  --file <path>     File path for --map mode

Modes:
  palace explore "query"              # Search code
  palace explore "query" --full       # Full context with learnings
  palace explore --map <symbol>       # Who calls this symbol?
  palace explore --map <sym> --file f # What does symbol call?
  palace explore --map --file <path>  # Full file call graph

Examples:
  palace explore "auth logic"
  palace explore "add auth" --full
  palace explore --map handleAuth
  palace explore --map Search --file butler.go
  palace explore --map --file internal/cli/cli.go
`)
	case "store", "remember", "learn":
		fmt.Print(`palace store - Store knowledge in the palace

Usage: palace store "content" [options]

Options:
  --root <path>        Workspace root (default: current directory)
  --scope <scope>      Scope: file, room, palace (default: palace)
  --path <path>        Scope path (file path or room name)
  --tag <tags>         Comma-separated tags
  --as <type>          Force type: decision, idea, or learning
  --confidence <n>     Confidence for learnings, 0.0-1.0 (default: 0.5)

Content is auto-classified based on natural language signals:
  - "Let's...", "We should..." → decision
  - "What if...", "Maybe we could..." → idea
  - "TIL...", "Always...", "Never..." → learning

Examples:
  palace store "Let's use JWT for auth"          # Auto: decision
  palace store "What if we add caching?"         # Auto: idea
  palace store "Always run tests first"          # Auto: learning
  palace store "Use JWT" --as decision           # Force type
  palace store "Config in /etc" --as learning --confidence 0.9
`)
	case "recall", "review", "outcome", "link":
		fmt.Print(`palace recall - Retrieve knowledge from the palace

Usage: palace recall [query] [options]
       palace recall update <decision-id> <outcome>
       palace recall link <source-id> --<relation> <target>

Options:
  --root <path>       Workspace root (default: current directory)
  --scope <scope>     Filter by scope (file, room, palace)
  --path <path>       Filter by scope path
  --limit <n>         Maximum results (default: 10)
  --type <type>       Filter by type: decision, idea, learning
  --pending           Show decisions awaiting outcome

Subcommands:
  recall update <id> <outcome>   Record decision outcome (success/failed/mixed)
  recall link <src> --rel <tgt>  Create relationship between records

Link Relations:
  --supersedes, --implements, --supports, --contradicts, --inspired-by, --related

Examples:
  palace recall                            # List learnings
  palace recall "auth"                     # Search knowledge
  palace recall --type decision            # List decisions
  palace recall --type idea                # List ideas
  palace recall --pending                  # Decisions awaiting review
  palace recall update d_abc123 success    # Record outcome
  palace recall link d_123 --supersedes d_old
  palace recall link i_456 --implements auth.go:10-20
`)
	case "serve":
		fmt.Print(`palace serve - Start MCP server for AI agents

Usage: palace serve [options]

Options:
  --root <path>    Workspace root (default: current directory)

The serve command starts a Model Context Protocol (MCP) server that:
  - Reads JSON-RPC requests from stdin
  - Provides tools for AI agents to search and explore code
  - Supports search, call graph, and context retrieval

Configure in Claude Desktop or other MCP clients:
  {
    "mcpServers": {
      "mind-palace": {
        "command": "palace",
        "args": ["serve", "--root", "/path/to/project"]
      }
    }
  }
`)
	case "ci":
		fmt.Print(`palace ci - CI/CD specific commands

Usage: palace ci <command> [options]

Commands:
  verify     Check if index is fresh (gate CI pipelines)
  collect    Generate context pack from diff scope
  signal     Generate change signal from git diff

These commands are optimized for CI pipelines with --diff support.

Examples:
  palace ci verify --diff HEAD~1..HEAD
  palace ci collect --diff HEAD~1..HEAD
  palace ci signal --diff HEAD~1..HEAD
`)
	case "session":
		fmt.Print(`palace session - Manage agent sessions

Usage: palace session <command> [options]

Commands:
  start    Start a new session
  end      End a session
  list     List sessions
  show     Show session details

Options for 'start':
  --agent <type>     Agent type (claude, cursor, aider, cli)
  --agent-id <id>    Unique agent instance ID
  --goal <text>      Session goal

Options for 'end':
  --state <state>    Final state (completed, abandoned)
  --summary <text>   Session summary

Options for 'list':
  --active           Show only active sessions
  --limit <n>        Maximum sessions (default: 10)

Examples:
  palace session start --agent claude --goal "implement auth"
  palace session list --active
  palace session show ses_abc123
  palace session end ses_abc123 --state completed
`)
	case "brief", "intel":
		fmt.Print(`palace brief - Get briefing on workspace or file

Usage: palace brief [file-path] [options]

Options:
  --root <path>     Workspace root (default: current directory)
  --sessions        Show detailed session information

Provides a comprehensive briefing including:
  - Active agents in the workspace
  - Recent sessions (with --sessions)
  - Conflict warnings (if file specified)
  - File intelligence (edit count, failure rate, last editor)
  - File-specific learnings
  - Relevant learnings
  - Active ideas and decisions
  - Hotspot files (most edited)

Examples:
  palace brief                    # Workspace briefing
  palace brief src/auth/login.go  # File-specific briefing
  palace brief --sessions         # Include session details
`)
	case "corridor":
		fmt.Print(`palace corridor - Cross-workspace knowledge sharing

Usage: palace corridor <subcommand> [options]

Subcommands:
  list      List linked workspaces and personal corridor stats
  link      Link another workspace for cross-project learning
  unlink    Remove a workspace link
  personal  View learnings in your personal corridor
  promote   Promote a workspace learning to personal corridor
  search    Search learnings across corridors

The personal corridor (~/.palace/corridors/personal.db) stores learnings
that follow you across all your projects. High-confidence, frequently-used
learnings can be promoted from workspace memory to your personal corridor.

Linked workspaces allow you to access learnings from other projects directly.

Examples:
  palace corridor list                      # View all corridors
  palace corridor link api ../api-service   # Link another project
  palace corridor personal                  # View personal learnings
  palace corridor promote abc123            # Promote a learning
  palace corridor search "auth" --all       # Search all corridors
`)
	case "dashboard":
		fmt.Print(`palace dashboard - Web dashboard for visualization

Usage: palace dashboard [options]

Options:
  --root <path>     Workspace root (default: current directory)
  --port <n>        Server port (default: 3001)
  --no-browser      Don't open browser automatically

The dashboard provides a web interface to:
  - View sessions and activity
  - Browse learnings
  - See file hotspots and intelligence
  - Manage corridors
  - Visualize call graphs
  - Monitor active agents

Examples:
  palace dashboard                  # Start on port 3001
  palace dashboard --port 8080      # Start on custom port
  palace dashboard --no-browser     # Start without opening browser
`)
	case "clean", "maintenance":
		fmt.Print(`palace clean - Clean up stale data

Usage: palace clean [options]

Options:
  --root <path>     Workspace root (default: current directory)
  --dry-run         Show what would be done without making changes

The clean command:
  - Marks abandoned sessions (>24h inactive) as abandoned
  - Removes stale agents (>5min without heartbeat)
  - Decays confidence of unused learnings (>30 days)
  - Prunes low-confidence learnings (<10%)
  - Validates and removes stale corridor links

Examples:
  palace clean                # Run cleanup
  palace clean --dry-run      # Preview changes
  palace maintenance          # Alias for clean
`)
	case "artifacts":
		fmt.Print(`Mind Palace Artifacts

Curated (commit to git):
  .palace/palace.jsonc              Main configuration
  .palace/rooms/*.jsonc             Room definitions
  .palace/playbooks/*.jsonc         Playbook definitions
  .palace/project-profile.json      Project metadata
  .palace/schemas/*                 JSON schemas (reference)

Generated (add to .gitignore):
  .palace/index/*                   SQLite database and scan artifacts
  .palace/outputs/*                 Generated context packs, signals
  .palace/sessions/*                Session data (if present)
`)
	case "all":
		fmt.Println(ExplainAll())
	default:
		return fmt.Errorf("unknown help topic: %s\n\nAvailable topics: explore, store, recall, brief, build, scan, check, serve, session, corridor, dashboard, clean, ci, artifacts", topic)
	}
	return nil
}

// ExplainAll returns a comprehensive reference of all features.
func ExplainAll() string {
	return `Mind Palace - Complete Reference

SCAN
  Purpose: Build/refresh the SQLite index and emit an auditable scan summary.
  Inputs: workspace files (excluding guardrails)
  Outputs:
    - .palace/index/palace.db (SQLite WAL + FTS)
    - .palace/index/scan.json (includes UUID + counts + hash)
  Behavior:
    - Always rebuilds index from disk; deterministic chunking + hashing.

CHECK (formerly verify + lint)
  Purpose: Detect staleness between workspace and indexed metadata.
  Modes:
    - Default: mtime/size shortcut with selective hashing
    - --strict: hash all candidates
  Scope:
    - No --diff: verifies full workspace against stored index.
    - With --diff: verifies only changed paths.

QUERY (formerly ask)
  Purpose: Search the codebase by intent or code symbols.
  Features:
    - Full-text search with BM25 ranking
    - CamelCase/snake_case tokenization
    - Synonym expansion for programming terms
    - Optional fuzzy matching for typo tolerance

CONTEXT (formerly plan)
  Purpose: Generate context-pack.json for AI consumption.
  Outputs:
    - .palace/outputs/context-pack.json

GRAPH (formerly callers/callees/callgraph)
  Purpose: Explore call relationships in the codebase.
  Modes:
    - Show callers (who calls X?)
    - Show callees (what does X call?)
    - Full file call graph

CI COMMANDS
  ci verify:  Check freshness with --diff support
  ci collect: Generate context pack from diff scope
  ci signal:  Generate change-signal.json from git diff

ARTIFACTS
  Curated (commit):
    - .palace/palace.jsonc, rooms/*.jsonc, playbooks/*.jsonc
    - .palace/project-profile.json
    - .palace/schemas/*
  Generated (ignore):
    - .palace/index/*, .palace/outputs/*`
}
