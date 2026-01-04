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
  brief     Get briefing on workspace or file

SETUP & INDEX
  init      Initialize the palace in the current directory
  scan      Build/refresh the code index
  check     Verify index freshness and optionally generate CI outputs

SERVICES
  serve     Start MCP server for AI agents
  dashboard Start web dashboard for visualization

AGENTS & SESSIONS
  session   Manage agent sessions

CROSS-WORKSPACE
  corridor  Cross-workspace knowledge sharing

HOUSEKEEPING
  clean     Clean up stale data
  update    Update palace to latest version
  help      Show help for a command or topic
  version   Show version information

EXPLORE EXAMPLES
  palace explore "auth logic"              # Search for code
  palace explore "add auth" --full         # Full context with learnings
  palace explore --map handleAuth          # Who calls handleAuth?

STORE & RECALL EXAMPLES
  palace store "Let's use JWT for auth"    # Auto-classified as decision
  palace recall                            # List learnings
  palace recall "auth"                     # Search knowledge

BRIEF EXAMPLES
  palace brief                             # Workspace briefing
  palace brief src/auth.go                 # File-specific briefing

Run 'palace help <command>' for detailed help on a command.
`)
	return nil
}

// ShowHelpTopic displays help for a specific topic.
func ShowHelpTopic(topic string) error {
	switch topic {
	case "init":
		fmt.Print(`palace init - Initialize the palace

Usage: palace init [options]

Options:
  --root <path>     Workspace root (default: current directory)
  --force           Overwrite existing curated files
  --detect          Auto-detect project type and generate profile
  --with-outputs    Also create generated outputs

Examples:
  palace init
  palace init --detect
`)
	case "scan":
		fmt.Print(`palace scan - Build or refresh the code index

Usage: palace scan [options]

Options:
  --root <path>    Workspace root (default: current directory)
  --full           Force full rescan (default: incremental)
  --deep           Enable LSP-based deep analysis for call tracking

The scan command parses your codebase using Tree-sitter and builds a structural index.
For Dart/Flutter projects, deep analysis runs automatically to extract accurate call graphs.
`)
	case "check":
		fmt.Print(`palace check - Verify index freshness

Usage: palace check [options]

Options:
  --root <path>       Workspace root (default: current directory)
  --diff <range>      Scope verification to git diff range
  --strict            Hash all files (slower but thorough)
  --collect           Also generate context pack from diff
  --signal            Also generate change signal from diff

The check command ensures the index is up-to-date and validates configuration.
`)
	case "explore":
		fmt.Print(`palace explore - Search code, get context, or trace calls

Usage: palace explore <query> [options]
       palace explore --map <symbol> [--file <path>]
       palace explore --rooms

Options:
  --root <path>     Workspace root (default: current directory)
  --room <name>     Filter to specific room
  --rooms           List all configured rooms
  --limit <n>       Maximum results (default: 10)
  --fuzzy           Enable fuzzy matching
  --full            Get full context (generates context-pack.json)
  --map <symbol>    Trace call graph for a symbol
  --file <path>     File path for --map mode
  --depth <n>       Recursion depth for call chain (1-10)
  --direction <d>   Trace direction: up, down, or both (default: up)

Examples:
  palace explore "auth logic"
  palace explore --rooms
  palace explore --map handleAuth
`)
	case "store":
		fmt.Print(`palace store - Store knowledge in the palace

Usage: palace store "content" [options]

Options:
  --root <path>        Workspace root (default: current directory)
  --scope <scope>      Scope: file, room, palace (default: palace)
  --path <path>        Scope path (file path or room name)
  --as <type>          Force type: decision, idea, or learning

Content is auto-classified (e.g., "Let's..." -> decision).
`)
	case "recall":
		fmt.Print(`palace recall - Retrieve knowledge from the palace

Usage: palace recall [query] [options]
       palace recall update <decision-id> <outcome>
       palace recall link --<relation> <target> <source-id>

Options:
  --root <path>       Workspace root (default: current directory)
  --type <type>       Filter by type: decision, idea, learning
  --pending           Show decisions awaiting outcome

Subcommands:
  update    Record decision outcome
  link      Create relationship between records
`)
	case "serve":
		fmt.Print(`palace serve - Start MCP server for AI agents

Starts a Model Context Protocol server on stdio.
`)
	case "session":
		fmt.Print(`palace session - Manage agent sessions

Usage: palace session <command> [options]

Commands:
  start, end, list, show
`)
	case "brief":
		fmt.Print(`palace brief - Get briefing on workspace or file

Usage: palace brief [file-path] [options]

Options:
  --sessions        Show detailed session information
`)
	case "corridor":
		fmt.Print(`palace corridor - Cross-workspace knowledge sharing

Usage: palace corridor <subcommand> [options]

Subcommands:
  list, link, unlink, personal, promote, search
`)
	case "dashboard":
		fmt.Print(`palace dashboard - Web dashboard for visualization

Usage: palace dashboard [options]

Options:
  --port <n>        Server port (default: 3001)
`)
	case "clean":
		fmt.Print(`palace clean - Clean up stale data

Usage: palace clean [options]

Options:
  --dry-run         Show what would be done
`)
	case "artifacts":
		fmt.Print(`Mind Palace Artifacts

Curated: .palace/palace.jsonc, rooms/*.jsonc, playbooks/*.jsonc
Generated: .palace/index/*, .palace/outputs/*
`)
	case "all":
		fmt.Println(ExplainAll())
	default:
		return fmt.Errorf("unknown help topic: %s\n\nAvailable topics: explore, store, recall, brief, init, scan, check, serve, session, corridor, dashboard, clean, artifacts", topic)
	}
	return nil
}

// ExplainAll returns a comprehensive reference of all features.
func ExplainAll() string {
	return `Mind Palace - Complete Reference

SCAN
  Purpose: Build/refresh the SQLite index and symbols.
  Outputs: .palace/index/palace.db

CHECK (formerly verify + lint)
  Purpose: Detect staleness and validate configuration.
  Modes: Default (fast mtime check), --strict (full hash check)

EXPLORE
  Purpose: Search codebase and trace relationships.
  Features: Full-text search, call graphs (--map), full context (--full).

BRIEF
  Purpose: Get briefing on workspace or file.

STORE
  Purpose: Capture ideas, decisions, and learnings.

RECALL
  Purpose: Retrieve knowledge and record outcomes.

CLEAN
  Purpose: Cleanup stale sessions and decay old learnings.

CI INTEGRATION
  Check:   palace check --diff <range>
  Context: palace check --collect --diff <range>
  Signal:  palace check --signal --diff <range>
`
}
