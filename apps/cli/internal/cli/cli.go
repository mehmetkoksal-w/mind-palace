package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/butler"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/collect"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/corridor"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/dashboard"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/jsonc"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/lint"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/project"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/scan"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/signal"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/update"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/validate"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/verify"
)

func init() {
	butler.SetJSONCDecoder(jsonc.DecodeFile)
}

// ============================================================================
// Validation Helpers
// ============================================================================

func validateConfidence(v float64) error {
	if v < 0.0 || v > 1.0 {
		return fmt.Errorf("confidence must be between 0.0 and 1.0, got %f", v)
	}
	return nil
}

func validateLimit(v int) error {
	if v < 0 {
		return fmt.Errorf("limit must be non-negative, got %d", v)
	}
	return nil
}

func validateScope(v string) error {
	valid := map[string]bool{"file": true, "room": true, "palace": true}
	if !valid[v] {
		return fmt.Errorf("scope must be file, room, or palace, got %q", v)
	}
	return nil
}

func validatePort(v int) error {
	if v < 1 || v > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", v)
	}
	return nil
}

func Run(args []string) error {
	if len(args) == 0 {
		return usage()
	}

	checkForUpdates(args)
	switch args[0] {
	case "version", "--version", "-v":
		return cmdVersion(args[1:])
	case "update":
		return cmdUpdate(args[1:])
	case "init":
		return cmdInit(args[1:])
	case "scan":
		return cmdScan(args[1:])
	case "check":
		return cmdCheck(args[1:])
	case "query":
		return cmdQuery(args[1:])
	case "context":
		return cmdContext(args[1:])
	case "graph":
		return cmdGraph(args[1:])
	case "serve":
		return cmdServe(args[1:])
	case "ci":
		return cmdCI(args[1:])
	case "session":
		return cmdSession(args[1:])
	case "learn":
		return cmdLearn(args[1:])
	case "recall":
		return cmdRecall(args[1:])
	case "intel":
		return cmdIntel(args[1:])
	case "brief":
		return cmdBrief(args[1:])
	case "corridor":
		return cmdCorridor(args[1:])
	case "maintenance":
		return cmdMaintenance(args[1:])
	case "dashboard":
		return cmdDashboard(args[1:])
	case "help", "-h", "--help":
		if len(args) > 1 {
			return cmdHelp(args[1:])
		}
		return usage()
	default:
		return fmt.Errorf("unknown command: %s\nRun 'palace help' for usage", args[0])
	}
}

func usage() error {
	fmt.Print(`palace - AI-friendly codebase indexing and search

COMMANDS
  init      Initialize palace in current directory
  scan      Build/refresh the code index
  check     Verify index freshness (combines lint + verify)
  query     Search the codebase
  context   Generate AI context for a goal
  graph     Explore call relationships
  serve     Start MCP server for AI agents
  ci        CI/CD specific commands

SESSION MEMORY
  session   Manage agent sessions (start, end, list, show)
  learn     Store a learning for future reference
  recall    Search and retrieve learnings
  intel     Show file intelligence (edit history, failures)
  brief     Get a briefing before working on files

CORRIDORS
  corridor  Cross-workspace knowledge sharing (list, link, unlink, personal, promote, search)

DASHBOARD
  dashboard Start web dashboard for visualization

HOUSEKEEPING
  maintenance Cleanup stale data (sessions, agents, learnings, links)
  update      Update palace to latest version
  help        Show help for a command or topic
  version     Show version information

EXAMPLES
  palace init                          # Initialize palace
  palace scan                          # Index the codebase
  palace check                         # Verify index is fresh
  palace query "auth logic"            # Search for auth logic
  palace context "add login feature"   # Get AI context for a task
  palace graph handleAuth              # Who calls handleAuth?
  palace serve                         # Start MCP server

SESSION EXAMPLES
  palace session start --agent claude  # Start a new session
  palace session list --active         # List active sessions
  palace learn "Always run tests"      # Store a learning
  palace recall "testing"              # Find learnings
  palace intel src/auth.go             # File edit history
  palace brief src/auth.go             # Full briefing

CORRIDOR EXAMPLES
  palace corridor list                 # List linked workspaces
  palace corridor link api ../api      # Link another workspace
  palace corridor personal             # Show personal learnings
  palace corridor promote <id>         # Promote learning to personal
  palace corridor search "auth"        # Search across corridors

Run 'palace help <command>' for detailed help on a command.
`)
	return nil
}

// ============================================================================
// Core Commands
// ============================================================================

func cmdUpdate(args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	err := update.Update(buildVersion, func(msg string) {
		fmt.Println(msg)
	})
	if err != nil {
		if err.Error() == "already at latest version" {
			fmt.Printf("palace %s is already the latest version.\n", buildVersion)
			return nil
		}
		return err
	}
	return nil
}

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	force := fs.Bool("force", false, "overwrite existing curated files")
	withOutputs := fs.Bool("with-outputs", false, "also create generated outputs (context-pack)")
	detect := fs.Bool("detect", false, "auto-detect project type and generate profile")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	if _, err := config.EnsureLayout(rootPath); err != nil {
		return err
	}
	if err := config.CopySchemas(rootPath, *force); err != nil {
		return err
	}

	// Auto-detect project type if requested
	language := "unknown"
	if *detect {
		profile := project.BuildProfile(rootPath)
		profilePath := filepath.Join(rootPath, ".palace", "project-profile.json")
		if err := config.WriteJSON(profilePath, profile); err != nil {
			return err
		}
		if len(profile.Languages) > 0 {
			language = profile.Languages[0]
		}
		fmt.Printf("detected project type: %s\n", language)
	}

	replacements := map[string]string{
		"projectName": filepath.Base(rootPath),
		"language":    language,
	}
	if err := config.WriteTemplate(filepath.Join(rootPath, ".palace", "palace.jsonc"), "palace.jsonc", replacements, *force); err != nil {
		return err
	}
	if err := config.WriteTemplate(filepath.Join(rootPath, ".palace", "rooms", "project-overview.jsonc"), "rooms/project-overview.jsonc", map[string]string{}, *force); err != nil {
		return err
	}
	if err := config.WriteTemplate(filepath.Join(rootPath, ".palace", "playbooks", "default.jsonc"), "playbooks/default.jsonc", map[string]string{}, *force); err != nil {
		return err
	}
	if err := config.WriteTemplate(filepath.Join(rootPath, ".palace", "project-profile.json"), "project-profile.json", map[string]string{}, *force); err != nil {
		return err
	}

	if *withOutputs {
		cpPath := filepath.Join(rootPath, ".palace", "outputs", "context-pack.json")
		if _, err := os.Stat(cpPath); os.IsNotExist(err) || *force {
			cpReplacements := map[string]string{
				"goal":      "unspecified",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			if err := config.WriteTemplate(cpPath, "outputs/context-pack.json", cpReplacements, *force); err != nil {
				return err
			}
		}
	}

	fmt.Printf("initialized palace in %s\n", filepath.Join(rootPath, ".palace"))
	return nil
}

func cmdScan(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	full := fs.Bool("full", false, "force full rescan (default: incremental)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *full {
		// Full scan
		summary, fileCount, err := scan.Run(*root)
		if err != nil {
			return err
		}
		fmt.Printf("full scan: indexed %d files, %d symbols, %d relationships\n", fileCount, summary.SymbolCount, summary.RelationshipCount)
		fmt.Printf("scan hash: %s\n", summary.ScanHash)
		fmt.Printf("scan artifact written to %s\n", filepath.Join(summary.Root, ".palace", "index", "scan.json"))
		return nil
	}

	// Incremental scan (default)
	summary, err := scan.RunIncremental(*root)
	if err != nil {
		// If no index exists, fall back to full scan
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		fmt.Fprintf(os.Stderr, "Running full scan instead...\n")
		fullSummary, fileCount, err := scan.Run(*root)
		if err != nil {
			return err
		}
		fmt.Printf("full scan: indexed %d files, %d symbols, %d relationships\n", fileCount, fullSummary.SymbolCount, fullSummary.RelationshipCount)
		fmt.Printf("scan hash: %s\n", fullSummary.ScanHash)
		fmt.Printf("scan artifact written to %s\n", filepath.Join(fullSummary.Root, ".palace", "index", "scan.json"))
		return nil
	}

	totalChanges := summary.FilesAdded + summary.FilesModified + summary.FilesDeleted
	if totalChanges == 0 {
		fmt.Printf("no changes detected (%d files unchanged)\n", summary.FilesUnchanged)
	} else {
		fmt.Printf("incremental scan: +%d added, ~%d modified, -%d deleted (took %v)\n",
			summary.FilesAdded, summary.FilesModified, summary.FilesDeleted, summary.Duration.Round(time.Millisecond))
		fmt.Printf("%d files unchanged\n", summary.FilesUnchanged)
	}
	return nil
}

func cmdCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	diff := fs.String("diff", "", "diff range for scoped verification")
	var strictFlag boolFlag
	fs.Var(&strictFlag, "strict", "strict mode (hash all files; slower but thorough)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	// Run lint first
	if err := lint.Run(rootPath); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("index missing; run 'palace scan' first: %w", err)
	}
	db, err := index.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	summary, err := index.LatestScan(db)
	if err != nil {
		return err
	}
	if summary.ID == 0 {
		return errors.New("no scan records found; run 'palace scan'")
	}

	mode := verify.ModeFast
	if strictFlag.set && strictFlag.value {
		mode = verify.ModeStrict
	}

	staleList, fullScope, source, candidateCount, err := verify.Run(db, verify.Options{Root: rootPath, DiffRange: *diff, Mode: mode})
	if err != nil {
		return err
	}

	printScope("check", fullScope, source, *diff, candidateCount, rootPath)

	if len(staleList) > 0 {
		fmt.Println("stale files detected:")
		preview := staleList
		if len(preview) > 20 {
			preview = preview[:20]
		}
		for _, s := range preview {
			fmt.Printf("- %s\n", s)
		}
		if len(staleList) > len(preview) {
			fmt.Printf("... and %d more\n", len(staleList)-len(preview))
		}
		return errors.New("index is stale; run 'palace scan'")
	}

	fmt.Printf("check ok; latest scan %s at %s\n", summary.ScanHash, summary.CompletedAt.Format(time.RFC3339))
	return nil
}

// ============================================================================
// Query & Context Commands
// ============================================================================

func cmdQuery(args []string) error {
	fs := flag.NewFlagSet("query", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	room := fs.String("room", "", "filter to specific room")
	limit := fs.Int("limit", 10, "maximum results")
	fuzzy := fs.Bool("fuzzy", false, "enable fuzzy matching for typo tolerance")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate inputs
	if err := validateLimit(*limit); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("usage: palace query <search terms>\n\nExample: palace query \"auth logic\"")
	}
	query := strings.Join(remaining, " ")

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("index missing; run 'palace scan' first: %w", err)
	}
	db, err := index.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	b, err := butler.New(db, rootPath)
	if err != nil {
		return fmt.Errorf("initialize butler: %w", err)
	}

	results, err := b.Search(query, butler.SearchOptions{
		Limit:      *limit,
		RoomFilter: *room,
		FuzzyMatch: *fuzzy,
	})
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	fmt.Printf("\n🔍 Search results for: \"%s\"\n", query)
	fmt.Println(strings.Repeat("─", 60))

	for _, group := range results {
		roomDisplay := group.Room
		if roomDisplay == "" {
			roomDisplay = "(ungrouped)"
		}
		fmt.Printf("\n📁 Room: %s\n", roomDisplay)
		if group.Summary != "" {
			fmt.Printf("   %s\n", group.Summary)
		}
		fmt.Println()

		for _, r := range group.Results {
			entryMark := ""
			if r.IsEntry {
				entryMark = " ⭐"
			}
			fmt.Printf("  📄 %s%s\n", r.Path, entryMark)
			fmt.Printf("     Lines %d-%d  (score: %.2f)\n", r.StartLine, r.EndLine, r.Score)

			snippet := r.Snippet
			lines := strings.Split(snippet, "\n")
			if len(lines) > 5 {
				lines = lines[:5]
				lines = append(lines, "...")
			}
			for _, line := range lines {
				fmt.Printf("     │ %s\n", truncateLine(line, 70))
			}
			fmt.Println()
		}
	}

	return nil
}

func cmdContext(args []string) error {
	fs := flag.NewFlagSet("context", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	goal := fs.String("goal", "", "goal for context pack")
	if err := fs.Parse(args); err != nil {
		return err
	}
	remaining := fs.Args()
	if *goal == "" && len(remaining) > 0 {
		*goal = strings.Join(remaining, " ")
	}
	if *goal == "" {
		return errors.New("usage: palace context <goal>\n\nExample: palace context \"add user authentication\"")
	}
	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	if _, err := config.EnsureLayout(rootPath); err != nil {
		return err
	}

	cpPath := filepath.Join(rootPath, ".palace", "outputs", "context-pack.json")
	cp := model.NewContextPack(*goal)
	if existing, err := model.LoadContextPack(cpPath); err == nil {
		cp = existing
	}
	cp.Goal = *goal
	cp.Provenance.UpdatedBy = "palace context"
	cp.Provenance.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if cp.Provenance.CreatedBy == "" {
		cp.Provenance.CreatedBy = "palace context"
	}
	if cp.Provenance.CreatedAt == "" {
		cp.Provenance.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	if db, err := index.Open(dbPath); err == nil {
		if summary, err := index.LatestScan(db); err == nil && summary.ID != 0 {
			cp.ScanHash = summary.ScanHash
			cp.ScanTime = summary.CompletedAt.Format(time.RFC3339)
			cp.ScanID = fmt.Sprintf("scan-%d", summary.ID)
		}
		db.Close()
	}

	if err := model.WriteContextPack(cpPath, cp); err != nil {
		return err
	}
	if err := validate.JSON(cpPath, "context-pack"); err != nil {
		return err
	}
	fmt.Printf("updated context pack at %s\n", cpPath)
	return nil
}

// ============================================================================
// Graph Command (unified call graph)
// ============================================================================

func cmdGraph(args []string) error {
	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	out := fs.String("out", "", "show outgoing calls (requires file path)")
	file := fs.String("file", "", "show full call graph for a file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()

	// Determine mode based on flags and arguments
	if *file != "" {
		// Full call graph for a file
		return cmdGraphFileWithPath(*root, *file)
	}

	if len(remaining) == 0 {
		return errors.New(`usage: palace graph <symbol> [options]

Show who calls a symbol:
  palace graph handleAuth              # Who calls handleAuth?

Show what a symbol calls:
  palace graph Search --out butler.go  # What does Search call?

Show full call graph for a file:
  palace graph --file cli.go           # All calls in/out of cli.go`)
	}

	symbolName := remaining[0]

	// If --out is specified or we have a second argument, show callees
	if *out != "" || len(remaining) >= 2 {
		filePath := *out
		if filePath == "" && len(remaining) >= 2 {
			filePath = remaining[1]
		}
		return cmdGraphCalleesWithArgs(*root, symbolName, filePath)
	}

	// Default: show callers
	return cmdGraphCallersWithArgs(*root, symbolName)
}

func cmdGraphCallersWithArgs(root, symbolName string) error {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("index missing; run 'palace scan' first: %w", err)
	}
	db, err := index.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	calls, err := index.GetIncomingCalls(db, symbolName)
	if err != nil {
		return fmt.Errorf("get callers: %w", err)
	}

	if len(calls) == 0 {
		fmt.Printf("No callers found for '%s'\n", symbolName)
		fmt.Println("This symbol may not be called anywhere, or call tracking may not be available for this language.")
		return nil
	}

	fmt.Printf("\n🔍 Callers of '%s'\n", symbolName)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Found %d call sites:\n\n", len(calls))

	for _, call := range calls {
		callerInfo := ""
		if call.CallerSymbol != "" {
			callerInfo = fmt.Sprintf(" (in %s)", call.CallerSymbol)
		}
		fmt.Printf("  📍 %s:%d%s\n", call.FilePath, call.Line, callerInfo)
	}

	return nil
}

func cmdGraphCalleesWithArgs(root, symbolName, filePath string) error {
	if filePath == "" {
		return errors.New("file path required for outgoing calls\n\nUsage: palace graph <symbol> --out <file>")
	}

	rootPath, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("index missing; run 'palace scan' first: %w", err)
	}
	db, err := index.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	calls, err := index.GetOutgoingCalls(db, symbolName, filePath)
	if err != nil {
		return fmt.Errorf("get callees: %w", err)
	}

	if len(calls) == 0 {
		fmt.Printf("No outgoing calls found for '%s' in '%s'\n", symbolName, filePath)
		return nil
	}

	fmt.Printf("\n🔍 Functions called by '%s'\n", symbolName)
	fmt.Printf("   File: %s\n", filePath)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Found %d function calls:\n\n", len(calls))

	for _, call := range calls {
		fmt.Printf("  📞 %s (line %d)\n", call.CalleeSymbol, call.Line)
	}

	return nil
}

func cmdGraphFileWithPath(root, filePath string) error {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("index missing; run 'palace scan' first: %w", err)
	}
	db, err := index.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	graph, err := index.GetCallGraph(db, filePath)
	if err != nil {
		return fmt.Errorf("get call graph: %w", err)
	}

	fmt.Printf("\n📊 Call Graph for '%s'\n", filePath)
	fmt.Println(strings.Repeat("═", 60))

	fmt.Println("\n⬅️  INCOMING CALLS (who calls functions in this file)")
	fmt.Println(strings.Repeat("─", 50))
	if len(graph.IncomingCalls) == 0 {
		fmt.Println("  No incoming calls from other files.")
	} else {
		for _, call := range graph.IncomingCalls {
			callerInfo := ""
			if call.CallerSymbol != "" {
				callerInfo = fmt.Sprintf(" (from %s)", call.CallerSymbol)
			}
			fmt.Printf("  📍 %s called from %s:%d%s\n", call.CalleeSymbol, call.FilePath, call.Line, callerInfo)
		}
	}

	fmt.Println("\n➡️  OUTGOING CALLS (what this file calls)")
	fmt.Println(strings.Repeat("─", 50))
	if len(graph.OutgoingCalls) == 0 {
		fmt.Println("  No outgoing calls tracked.")
	} else {
		for _, call := range graph.OutgoingCalls {
			callerInfo := ""
			if call.CallerSymbol != "" {
				callerInfo = fmt.Sprintf(" (from %s)", call.CallerSymbol)
			}
			fmt.Printf("  📞 %s at line %d%s\n", call.CalleeSymbol, call.Line, callerInfo)
		}
	}

	return nil
}

// ============================================================================
// Serve Command
// ============================================================================

func cmdServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("index missing; run 'palace scan' first: %w", err)
	}
	db, err := index.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	b, err := butler.New(db, rootPath)
	if err != nil {
		return fmt.Errorf("initialize butler: %w", err)
	}

	server := butler.NewMCPServer(b)

	fmt.Fprintln(os.Stderr, "Mind Palace MCP server started. Reading JSON-RPC from stdin...")

	return server.Serve()
}

// ============================================================================
// CI Commands
// ============================================================================

func cmdCI(args []string) error {
	if len(args) == 0 {
		return errors.New(`usage: palace ci <command>

Commands:
  verify   Check if index is fresh (for CI gates)
  collect  Generate context pack from diff
  signal   Generate change signal from git diff

Examples:
  palace ci verify --diff HEAD~1..HEAD
  palace ci collect --diff HEAD~1..HEAD
  palace ci signal --diff HEAD~1..HEAD`)
	}

	switch args[0] {
	case "verify":
		return cmdCIVerify(args[1:])
	case "collect":
		return cmdCICollect(args[1:])
	case "signal":
		return cmdCISignal(args[1:])
	default:
		return fmt.Errorf("unknown ci command: %s\nRun 'palace ci' for usage", args[0])
	}
}

func cmdCIVerify(args []string) error {
	// Same as check but intended for CI context
	return cmdCheck(args)
}

func cmdCICollect(args []string) error {
	fs := flag.NewFlagSet("ci collect", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	diff := fs.String("diff", "", "diff range or matching change signal")
	allowStale := fs.Bool("allow-stale", false, "allow collecting even if index is stale")
	if err := fs.Parse(args); err != nil {
		return err
	}

	fullScope := strings.TrimSpace(*diff) == ""
	result, err := collect.Run(*root, *diff, collect.Options{AllowStale: *allowStale})
	if err != nil {
		return err
	}

	cp := result.ContextPack
	source := ""
	if cp.Scope != nil {
		source = cp.Scope.Source
	}
	printScope("collect", fullScope, source, *diff, scopeFileCount(cp), mustAbs(*root))

	for _, warning := range result.CorridorWarnings {
		fmt.Fprintf(os.Stderr, "⚠️  %s\n", warning)
	}

	fmt.Printf("context pack updated from scan %s\n", cp.ScanHash)
	return nil
}

func cmdCISignal(args []string) error {
	fs := flag.NewFlagSet("ci signal", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	diff := fs.String("diff", "", "diff range (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*diff) == "" {
		return errors.New("--diff range is required\n\nUsage: palace ci signal --diff HEAD~1..HEAD")
	}
	if _, err := signal.Generate(*root, *diff); err != nil {
		return err
	}
	fmt.Println("change signal written to .palace/outputs/change-signal.json")
	return nil
}

// ============================================================================
// Session Memory Commands
// ============================================================================

func cmdSession(args []string) error {
	if len(args) == 0 {
		return errors.New(`usage: palace session <command>

Commands:
  start    Start a new session
  end      End a session
  list     List sessions
  show     Show session details

Examples:
  palace session start --agent claude --goal "implement auth"
  palace session end SESSION_ID
  palace session list --active
  palace session show SESSION_ID`)
	}

	switch args[0] {
	case "start":
		return cmdSessionStart(args[1:])
	case "end":
		return cmdSessionEnd(args[1:])
	case "list":
		return cmdSessionList(args[1:])
	case "show":
		return cmdSessionShow(args[1:])
	default:
		return fmt.Errorf("unknown session command: %s\nRun 'palace help session' for usage", args[0])
	}
}

func cmdSessionStart(args []string) error {
	fs := flag.NewFlagSet("session start", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	agent := fs.String("agent", "cli", "agent type (claude, cursor, aider, etc.)")
	agentID := fs.String("agent-id", "", "unique agent instance ID")
	goal := fs.String("goal", "", "session goal")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Use remaining args as goal if not set via flag
	if *goal == "" && len(fs.Args()) > 0 {
		*goal = strings.Join(fs.Args(), " ")
	}

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	session, err := mem.StartSession(*agent, *agentID, *goal)
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}

	fmt.Printf("Session started: %s\n", session.ID)
	fmt.Printf("  Agent: %s\n", session.AgentType)
	if session.Goal != "" {
		fmt.Printf("  Goal: %s\n", session.Goal)
	}
	return nil
}

func cmdSessionEnd(args []string) error {
	fs := flag.NewFlagSet("session end", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	state := fs.String("state", "completed", "final state (completed, abandoned)")
	summary := fs.String("summary", "", "session summary")
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("usage: palace session end SESSION_ID [--state completed|abandoned] [--summary \"...\"]")
	}
	sessionID := remaining[0]

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	if err := mem.EndSession(sessionID, *state, *summary); err != nil {
		return fmt.Errorf("end session: %w", err)
	}

	fmt.Printf("Session %s ended (state: %s)\n", sessionID, *state)
	return nil
}

func cmdSessionList(args []string) error {
	fs := flag.NewFlagSet("session list", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	active := fs.Bool("active", false, "show only active sessions")
	limit := fs.Int("limit", 10, "maximum sessions to show")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate inputs
	if err := validateLimit(*limit); err != nil {
		return err
	}

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	sessions, err := mem.ListSessions(*active, *limit)
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	fmt.Printf("\n📋 Sessions\n")
	fmt.Println(strings.Repeat("─", 60))

	for _, s := range sessions {
		stateIcon := "✅"
		switch s.State {
		case "active":
			stateIcon = "🔄"
		case "abandoned":
			stateIcon = "❌"
		}
		fmt.Printf("%s %s [%s] %s\n", stateIcon, s.ID, s.AgentType, s.State)
		if s.Goal != "" {
			fmt.Printf("   Goal: %s\n", truncateLine(s.Goal, 50))
		}
		fmt.Printf("   Started: %s\n", s.StartedAt.Format(time.RFC3339))
		fmt.Println()
	}

	return nil
}

func cmdSessionShow(args []string) error {
	fs := flag.NewFlagSet("session show", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("usage: palace session show SESSION_ID")
	}
	sessionID := remaining[0]

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	session, err := mem.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	fmt.Printf("\n📋 Session: %s\n", session.ID)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Agent:      %s\n", session.AgentType)
	if session.AgentID != "" {
		fmt.Printf("Agent ID:   %s\n", session.AgentID)
	}
	fmt.Printf("State:      %s\n", session.State)
	fmt.Printf("Started:    %s\n", session.StartedAt.Format(time.RFC3339))
	fmt.Printf("Last Active: %s\n", session.LastActivity.Format(time.RFC3339))
	if session.Goal != "" {
		fmt.Printf("Goal:       %s\n", session.Goal)
	}
	if session.Summary != "" {
		fmt.Printf("Summary:    %s\n", session.Summary)
	}

	// Show recent activities
	activities, err := mem.GetActivities(sessionID, "", 10)
	if err == nil && len(activities) > 0 {
		fmt.Printf("\n📝 Recent Activities:\n")
		for _, a := range activities {
			outcomeIcon := "•"
			switch a.Outcome {
			case "success":
				outcomeIcon = "✓"
			case "failure":
				outcomeIcon = "✗"
			}
			target := ""
			if a.Target != "" {
				target = fmt.Sprintf(" → %s", a.Target)
			}
			fmt.Printf("  %s [%s] %s%s\n", outcomeIcon, a.Kind, a.Timestamp.Format("15:04:05"), target)
		}
	}

	return nil
}

func cmdLearn(args []string) error {
	// Extract content (non-flag args) and flags separately
	// This allows: palace learn "content" --scope room
	// as well as:  palace learn --scope room "content"
	var contentParts []string
	var flagArgs []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			// It's a flag - add it and its value if present
			flagArgs = append(flagArgs, arg)
			// Check if next arg is the flag value (not another flag)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				// Check if this flag expects a value
				if arg == "--scope" || arg == "-scope" ||
					arg == "--path" || arg == "-path" ||
					arg == "--root" || arg == "-root" ||
					arg == "--confidence" || arg == "-confidence" {
					i++
					flagArgs = append(flagArgs, args[i])
				}
			}
		} else {
			contentParts = append(contentParts, arg)
		}
	}

	fs := flag.NewFlagSet("learn", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	scope := fs.String("scope", "palace", "learning scope (file, room, palace)")
	path := fs.String("path", "", "scope path (file path or room name)")
	confidence := fs.Float64("confidence", 0.5, "initial confidence (0.0-1.0)")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	// Validate inputs
	if err := validateScope(*scope); err != nil {
		return err
	}
	if err := validateConfidence(*confidence); err != nil {
		return err
	}

	if len(contentParts) == 0 {
		return errors.New(`usage: palace learn "content" [options]

Options:
  --scope <scope>       Scope: file, room, palace (default: palace)
  --path <path>         Scope path (file path or room name)
  --confidence <n>      Initial confidence 0.0-1.0 (default: 0.5)

Examples:
  palace learn "Always run tests before committing"
  palace learn "This file has fragile regex" --scope file --path src/parser.go
  palace learn "Auth module requires database connection" --scope room --path auth`)
	}
	content := strings.Join(contentParts, " ")

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	learning := memory.Learning{
		Scope:      *scope,
		ScopePath:  *path,
		Content:    content,
		Confidence: *confidence,
		Source:     "user",
	}

	id, err := mem.AddLearning(learning)
	if err != nil {
		return fmt.Errorf("add learning: %w", err)
	}

	fmt.Printf("Learning stored: %s\n", id)
	fmt.Printf("  Scope: %s", *scope)
	if *path != "" {
		fmt.Printf(" (%s)", *path)
	}
	fmt.Println()
	fmt.Printf("  Content: %s\n", truncateLine(content, 60))
	return nil
}

func cmdRecall(args []string) error {
	fs := flag.NewFlagSet("recall", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	scope := fs.String("scope", "", "filter by scope (file, room, palace)")
	path := fs.String("path", "", "filter by scope path")
	limit := fs.Int("limit", 10, "maximum results")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate inputs
	if err := validateLimit(*limit); err != nil {
		return err
	}
	if *scope != "" {
		if err := validateScope(*scope); err != nil {
			return err
		}
	}

	remaining := fs.Args()
	query := ""
	if len(remaining) > 0 {
		query = strings.Join(remaining, " ")
	}

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	var learnings []memory.Learning
	if query != "" {
		learnings, err = mem.SearchLearnings(query, *limit)
	} else {
		learnings, err = mem.GetLearnings(*scope, *path, *limit)
	}
	if err != nil {
		return fmt.Errorf("recall learnings: %w", err)
	}

	if len(learnings) == 0 {
		fmt.Println("No learnings found.")
		return nil
	}

	fmt.Printf("\n💡 Learnings\n")
	fmt.Println(strings.Repeat("─", 60))

	for _, l := range learnings {
		confidenceBar := strings.Repeat("█", int(l.Confidence*10)) + strings.Repeat("░", 10-int(l.Confidence*10))
		scopeInfo := l.Scope
		if l.ScopePath != "" {
			scopeInfo = fmt.Sprintf("%s:%s", l.Scope, l.ScopePath)
		}
		fmt.Printf("\n[%s] %s (%.0f%%)\n", l.ID, confidenceBar, l.Confidence*100)
		fmt.Printf("  Scope:  %s\n", scopeInfo)
		fmt.Printf("  Source: %s | Used: %d times\n", l.Source, l.UseCount)
		fmt.Printf("  %s\n", l.Content)
	}
	fmt.Println()

	return nil
}

func cmdIntel(args []string) error {
	fs := flag.NewFlagSet("intel", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New(`usage: palace intel <file-path>

Shows file intelligence including:
  - Edit count and history
  - Failure rate
  - Associated learnings

Example:
  palace intel src/auth/login.go`)
	}
	filePath := remaining[0]

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	intel, err := mem.GetFileIntel(filePath)
	if err != nil {
		return fmt.Errorf("get file intel: %w", err)
	}

	fmt.Printf("\n📊 File Intelligence: %s\n", filePath)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Edit Count:    %d\n", intel.EditCount)
	fmt.Printf("Failure Count: %d\n", intel.FailureCount)
	if intel.EditCount > 0 {
		failureRate := float64(intel.FailureCount) / float64(intel.EditCount) * 100
		fmt.Printf("Failure Rate:  %.1f%%\n", failureRate)
	}
	if !intel.LastEdited.IsZero() {
		fmt.Printf("Last Edited:   %s\n", intel.LastEdited.Format(time.RFC3339))
	}
	if intel.LastEditor != "" {
		fmt.Printf("Last Editor:   %s\n", intel.LastEditor)
	}

	// Show associated learnings
	learnings, err := mem.GetFileLearnings(filePath)
	if err == nil && len(learnings) > 0 {
		fmt.Printf("\n💡 Associated Learnings:\n")
		for _, l := range learnings {
			fmt.Printf("  • [%.0f%%] %s\n", l.Confidence*100, truncateLine(l.Content, 50))
		}
	}

	return nil
}

func cmdBrief(args []string) error {
	fs := flag.NewFlagSet("brief", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	filePath := ""
	if len(remaining) > 0 {
		filePath = remaining[0]
	}

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	fmt.Printf("\n📋 Briefing")
	if filePath != "" {
		fmt.Printf(" for: %s", filePath)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("═", 60))

	// Show active agents
	agents, err := mem.GetActiveAgents(5 * time.Minute)
	if err == nil && len(agents) > 0 {
		fmt.Printf("\n🤖 Active Agents:\n")
		for _, a := range agents {
			currentFile := ""
			if a.CurrentFile != "" {
				currentFile = fmt.Sprintf(" working on %s", a.CurrentFile)
			}
			fmt.Printf("  • %s (%s)%s\n", a.AgentType, a.SessionID[:12], currentFile)
		}
	}

	// Show conflict warning if file specified
	if filePath != "" {
		conflict, err := mem.CheckConflict("", filePath)
		if err == nil && conflict != nil {
			fmt.Printf("\n⚠️  Conflict Warning:\n")
			fmt.Printf("  Another agent (%s) touched this file recently\n", conflict.OtherAgent)
			fmt.Printf("  Session: %s | Last touched: %s\n", conflict.OtherSession[:12], conflict.LastTouched.Format("15:04:05"))
		}

		// Show file intel
		intel, err := mem.GetFileIntel(filePath)
		if err == nil && intel.EditCount > 0 {
			fmt.Printf("\n📊 File History:\n")
			fmt.Printf("  Edits: %d | Failures: %d", intel.EditCount, intel.FailureCount)
			if intel.EditCount > 0 {
				failureRate := float64(intel.FailureCount) / float64(intel.EditCount) * 100
				if failureRate > 20 {
					fmt.Printf(" (⚠️ %.0f%% failure rate)", failureRate)
				}
			}
			fmt.Println()
		}
	}

	// Show relevant learnings
	learnings, err := mem.GetRelevantLearnings(filePath, "", 5)
	if err == nil && len(learnings) > 0 {
		fmt.Printf("\n💡 Relevant Learnings:\n")
		for _, l := range learnings {
			scopeInfo := ""
			if l.Scope != "palace" {
				scopeInfo = fmt.Sprintf(" [%s]", l.Scope)
			}
			fmt.Printf("  • [%.0f%%]%s %s\n", l.Confidence*100, scopeInfo, truncateLine(l.Content, 45))
		}
	}

	// Show hotspots
	hotspots, err := mem.GetFileHotspots(5)
	if err == nil && len(hotspots) > 0 {
		fmt.Printf("\n🔥 Hotspots (most edited files):\n")
		for _, h := range hotspots {
			warning := ""
			if h.FailureCount > 0 {
				warning = fmt.Sprintf(" (⚠️ %d failures)", h.FailureCount)
			}
			fmt.Printf("  • %s (%d edits)%s\n", h.Path, h.EditCount, warning)
		}
	}

	fmt.Println()
	return nil
}

// ============================================================================
// Corridor Commands
// ============================================================================

func cmdCorridor(args []string) error {
	if len(args) == 0 {
		return errors.New(`usage: palace corridor <subcommand> [options]

Subcommands:
  list      List linked workspaces
  link      Link another workspace
  unlink    Remove a workspace link
  personal  Show personal corridor learnings
  promote   Promote a learning to personal corridor
  search    Search across all corridors

Run 'palace corridor <subcommand> --help' for subcommand help.`)
	}

	switch args[0] {
	case "list":
		return cmdCorridorList(args[1:])
	case "link":
		return cmdCorridorLink(args[1:])
	case "unlink":
		return cmdCorridorUnlink(args[1:])
	case "personal":
		return cmdCorridorPersonal(args[1:])
	case "promote":
		return cmdCorridorPromote(args[1:])
	case "search":
		return cmdCorridorSearch(args[1:])
	default:
		return fmt.Errorf("unknown corridor command: %s\nRun 'palace help corridor' for usage", args[0])
	}
}

func cmdCorridorList(args []string) error {
	gc, err := corridor.OpenGlobal()
	if err != nil {
		return fmt.Errorf("open global corridor: %w", err)
	}
	defer gc.Close()

	links, err := gc.GetLinks()
	if err != nil {
		return fmt.Errorf("get links: %w", err)
	}

	stats, err := gc.Stats()
	if err != nil {
		return fmt.Errorf("get stats: %w", err)
	}

	fmt.Printf("\n🚪 Corridors\n")
	fmt.Println(strings.Repeat("─", 60))

	// Personal corridor stats
	fmt.Printf("\n📌 Personal Corridor\n")
	fmt.Printf("  Learnings: %d\n", stats["learningCount"])
	if avg, ok := stats["averageConfidence"].(float64); ok && avg > 0 {
		fmt.Printf("  Avg Confidence: %.0f%%\n", avg*100)
	}

	// Linked workspaces
	fmt.Printf("\n🔗 Linked Workspaces (%d)\n", len(links))
	if len(links) == 0 {
		fmt.Printf("  No workspaces linked yet.\n")
		fmt.Printf("  Use 'palace corridor link <name> <path>' to link one.\n")
	} else {
		for _, link := range links {
			lastAccess := "never"
			if !link.LastAccessed.IsZero() {
				lastAccess = link.LastAccessed.Format("2006-01-02")
			}
			fmt.Printf("  • %s\n", link.Name)
			fmt.Printf("    Path: %s\n", link.Path)
			fmt.Printf("    Added: %s | Last accessed: %s\n", link.AddedAt.Format("2006-01-02"), lastAccess)
		}
	}

	fmt.Println()
	return nil
}

func cmdCorridorLink(args []string) error {
	if len(args) < 2 {
		return errors.New(`usage: palace corridor link <name> <path>

Links another workspace to enable cross-workspace learning sharing.
The workspace must have a .palace directory.

Examples:
  palace corridor link api ../api-service
  palace corridor link shared ~/code/shared-lib`)
	}

	name := args[0]
	path := args[1]

	gc, err := corridor.OpenGlobal()
	if err != nil {
		return fmt.Errorf("open global corridor: %w", err)
	}
	defer gc.Close()

	if err := gc.Link(name, path); err != nil {
		return fmt.Errorf("link workspace: %w", err)
	}

	absPath, _ := filepath.Abs(path)
	fmt.Printf("Linked workspace '%s' → %s\n", name, absPath)
	return nil
}

func cmdCorridorUnlink(args []string) error {
	if len(args) < 1 {
		return errors.New(`usage: palace corridor unlink <name>

Removes a workspace link.

Example:
  palace corridor unlink api`)
	}

	name := args[0]

	gc, err := corridor.OpenGlobal()
	if err != nil {
		return fmt.Errorf("open global corridor: %w", err)
	}
	defer gc.Close()

	if err := gc.Unlink(name); err != nil {
		return fmt.Errorf("unlink workspace: %w", err)
	}

	fmt.Printf("Unlinked workspace '%s'\n", name)
	return nil
}

func cmdCorridorPersonal(args []string) error {
	fs := flag.NewFlagSet("personal", flag.ContinueOnError)
	query := fs.String("query", "", "search query")
	limit := fs.Int("limit", 20, "maximum results")
	if err := fs.Parse(args); err != nil {
		return err
	}

	gc, err := corridor.OpenGlobal()
	if err != nil {
		return fmt.Errorf("open global corridor: %w", err)
	}
	defer gc.Close()

	learnings, err := gc.GetPersonalLearnings(*query, *limit)
	if err != nil {
		return fmt.Errorf("get personal learnings: %w", err)
	}

	fmt.Printf("\n📌 Personal Corridor Learnings\n")
	fmt.Println(strings.Repeat("─", 60))

	if len(learnings) == 0 {
		fmt.Printf("\nNo personal learnings yet.\n")
		fmt.Printf("Use 'palace corridor promote <id>' to promote workspace learnings.\n\n")
		return nil
	}

	for _, l := range learnings {
		confidenceBar := strings.Repeat("█", int(l.Confidence*10)) + strings.Repeat("░", 10-int(l.Confidence*10))
		origin := ""
		if l.OriginWorkspace != "" {
			origin = fmt.Sprintf(" (from: %s)", l.OriginWorkspace)
		}
		fmt.Printf("\n[%s] %s (%.0f%%)%s\n", l.ID[:12], confidenceBar, l.Confidence*100, origin)
		fmt.Printf("  Used: %d times | Last: %s\n", l.UseCount, l.LastUsed.Format("2006-01-02"))
		fmt.Printf("  %s\n", l.Content)
	}
	fmt.Println()

	return nil
}

func cmdCorridorPromote(args []string) error {
	fs := flag.NewFlagSet("promote", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New(`usage: palace corridor promote <learning-id>

Promotes a workspace learning to your personal corridor.
The learning will then be available across all your projects.

Example:
  palace corridor promote abc123def456`)
	}

	learningID := remaining[0]

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	// Get the learning from workspace memory
	learnings, err := mem.GetLearnings("", "", 1000)
	if err != nil {
		return fmt.Errorf("get learnings: %w", err)
	}

	var found *memory.Learning
	for _, l := range learnings {
		if l.ID == learningID || strings.HasPrefix(l.ID, learningID) {
			found = &l
			break
		}
	}

	if found == nil {
		return fmt.Errorf("learning not found: %s", learningID)
	}

	gc, err := corridor.OpenGlobal()
	if err != nil {
		return fmt.Errorf("open global corridor: %w", err)
	}
	defer gc.Close()

	workspaceName := filepath.Base(rootPath)
	if err := gc.PromoteFromWorkspace(workspaceName, *found); err != nil {
		return fmt.Errorf("promote learning: %w", err)
	}

	fmt.Printf("Promoted learning to personal corridor:\n")
	fmt.Printf("  ID: %s\n", found.ID)
	fmt.Printf("  Content: %s\n", truncateLine(found.Content, 50))
	fmt.Printf("  From: %s\n", workspaceName)
	return nil
}

func cmdCorridorSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	all := fs.Bool("all", false, "search all linked workspaces too")
	limit := fs.Int("limit", 20, "maximum results")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate inputs
	if err := validateLimit(*limit); err != nil {
		return err
	}

	remaining := fs.Args()
	query := ""
	if len(remaining) > 0 {
		query = strings.Join(remaining, " ")
	}

	gc, err := corridor.OpenGlobal()
	if err != nil {
		return fmt.Errorf("open global corridor: %w", err)
	}
	defer gc.Close()

	fmt.Printf("\n🔍 Corridor Search")
	if query != "" {
		fmt.Printf(": %s", query)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))

	// Personal learnings
	personalLearnings, err := gc.GetPersonalLearnings(query, *limit)
	if err == nil && len(personalLearnings) > 0 {
		fmt.Printf("\n📌 Personal Corridor (%d results)\n", len(personalLearnings))
		for _, l := range personalLearnings {
			fmt.Printf("  • [%.0f%%] %s\n", l.Confidence*100, truncateLine(l.Content, 50))
		}
	}

	// Linked workspace learnings
	if *all {
		linkedLearnings, err := gc.GetAllLinkedLearnings(*limit)
		if err == nil && len(linkedLearnings) > 0 {
			fmt.Printf("\n🔗 Linked Workspaces (%d results)\n", len(linkedLearnings))
			for _, l := range linkedLearnings {
				scope := l.Scope
				if l.ScopePath != "" {
					scope = fmt.Sprintf("%s:%s", l.Scope, l.ScopePath)
				}
				fmt.Printf("  • [%.0f%%] [%s] %s\n", l.Confidence*100, scope, truncateLine(l.Content, 40))
			}
		}
	}

	fmt.Println()
	return nil
}

// ============================================================================
// Dashboard Command
// ============================================================================

func cmdDashboard(args []string) error {
	fs := flag.NewFlagSet("dashboard", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	port := fs.Int("port", 3001, "server port")
	noBrowser := fs.Bool("no-browser", false, "don't open browser automatically")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate inputs
	if err := validatePort(*port); err != nil {
		return err
	}

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	// Open memory database
	mem, err := memory.Open(rootPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not open memory database: %v\n", err)
	}

	// Open global corridor
	gc, err := corridor.OpenGlobal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not open global corridor: %v\n", err)
	}

	// Open butler for code search
	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	var b *butler.Butler
	if _, err := os.Stat(dbPath); err == nil {
		db, err := index.Open(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not open index database: %v\n", err)
		} else {
			b, err = butler.New(db, rootPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Could not open butler: %v\n", err)
			}
		}
	}

	server := dashboard.New(dashboard.Config{
		Butler:   b,
		Memory:   mem,
		Corridor: gc,
		Port:     *port,
		Root:     rootPath,
	})

	fmt.Printf("Starting Mind Palace Dashboard...\n")
	return server.Start(!*noBrowser)
}

// ============================================================================
// Maintenance Command
// ============================================================================

func cmdMaintenance(args []string) error {
	fs := flag.NewFlagSet("maintenance", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	dryRun := fs.Bool("dry-run", false, "show what would be done without making changes")
	if err := fs.Parse(args); err != nil {
		return err
	}

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	fmt.Println("\n🧹 Palace Maintenance")
	fmt.Println(strings.Repeat("─", 60))
	if *dryRun {
		fmt.Println("(dry-run mode - no changes will be made)")
		fmt.Println()
	}

	// 1. Cleanup abandoned sessions and stale agents in workspace memory
	mem, err := memory.Open(rootPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not open memory database: %v\n", err)
	} else {
		defer mem.Close()

		// Cleanup abandoned sessions (>24h inactive)
		if *dryRun {
			fmt.Println("Would mark active sessions >24h inactive as abandoned")
		} else {
			n, err := mem.CleanupAbandonedSessions(24 * time.Hour)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to cleanup sessions: %v\n", err)
			} else if n > 0 {
				fmt.Printf("✓ Marked %d abandoned sessions\n", n)
			}
		}

		// Cleanup stale agents (>5min inactive)
		if *dryRun {
			fmt.Println("Would remove agents with no heartbeat for >5 minutes")
		} else {
			n, err := mem.CleanupStaleAgents(5 * time.Minute)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to cleanup agents: %v\n", err)
			} else if n > 0 {
				fmt.Printf("✓ Removed %d stale agents\n", n)
			}
		}

		// Decay unused learnings (>30 days, reduce by 0.1)
		if *dryRun {
			fmt.Println("Would decay confidence of learnings unused for >30 days")
		} else {
			n, err := mem.DecayUnusedLearnings(30, 0.1)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to decay learnings: %v\n", err)
			} else if n > 0 {
				fmt.Printf("✓ Decayed %d unused learnings\n", n)
			}
		}

		// Prune low-confidence learnings (<0.1)
		if *dryRun {
			fmt.Println("Would remove learnings with confidence <0.1")
		} else {
			n, err := mem.PruneLowConfidenceLearnings(0.1)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to prune learnings: %v\n", err)
			} else if n > 0 {
				fmt.Printf("✓ Pruned %d low-confidence learnings\n", n)
			}
		}
	}

	// 2. Validate and report stale corridor links
	gc, err := corridor.OpenGlobal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not open global corridor: %v\n", err)
	} else {
		defer gc.Close()

		staleLinks, err := gc.ValidateLinks()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to validate links: %v\n", err)
		} else if len(staleLinks) > 0 {
			if *dryRun {
				fmt.Printf("Would remove %d stale corridor links:\n", len(staleLinks))
				for _, name := range staleLinks {
					fmt.Printf("  - %s\n", name)
				}
			} else {
				pruned, err := gc.PruneStaleLinks()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to prune links: %v\n", err)
				} else {
					fmt.Printf("✓ Removed %d stale corridor links\n", len(pruned))
				}
			}
		}
	}

	fmt.Println()
	if *dryRun {
		fmt.Println("Run without --dry-run to apply changes.")
	} else {
		fmt.Println("✓ Maintenance complete")
	}
	return nil
}

// ============================================================================
// Help Command
// ============================================================================

func cmdHelp(args []string) error {
	if len(args) == 0 {
		return usage()
	}

	topic := strings.ToLower(strings.TrimSpace(args[0]))
	switch topic {
	case "init":
		fmt.Print(`palace init - Initialize palace scaffolding

Usage: palace init [options]

Options:
  --root <path>     Workspace root (default: current directory)
  --force           Overwrite existing curated files
  --detect          Auto-detect project type and generate profile
  --with-outputs    Also create generated outputs

Examples:
  palace init
  palace init --detect
  palace init --force --detect
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
  --root <path>     Workspace root (default: current directory)
  --diff <range>    Scope verification to git diff range
  --strict          Hash all files (slower but thorough)

The check command:
  - Validates palace configuration files (lint)
  - Compares workspace files against the index
  - Reports stale files that need re-scanning

Examples:
  palace check
  palace check --strict
  palace check --diff HEAD~1..HEAD
`)
	case "query", "ask":
		fmt.Print(`palace query - Search the codebase

Usage: palace query <search terms> [options]

Options:
  --root <path>    Workspace root (default: current directory)
  --room <name>    Filter to specific room
  --limit <n>      Maximum results (default: 10)
  --fuzzy          Enable fuzzy matching for typo tolerance

The query command searches for:
  - Code symbols (functions, classes, methods)
  - Content matching your search terms
  - Semantic matches (synonyms, CamelCase splitting)

Examples:
  palace query "auth logic"
  palace query handleAuth
  palace query "user service" --room auth
  palace query authetncation --fuzzy   # typo-tolerant
`)
	case "context", "plan":
		fmt.Print(`palace context - Generate AI context for a goal

Usage: palace context <goal> [options]

Options:
  --root <path>    Workspace root (default: current directory)
  --goal <text>    Goal description (can also be positional)

The context command creates a context-pack.json with:
  - Goal description
  - Relevant scan metadata
  - Provenance information

Examples:
  palace context "add user authentication"
  palace context "fix the login bug"
  palace context --goal "refactor payment flow"
`)
	case "graph", "callers", "callees", "callgraph":
		fmt.Print(`palace graph - Explore call relationships

Usage: palace graph <symbol> [options]
       palace graph --file <path>

Options:
  --root <path>    Workspace root (default: current directory)
  --out <file>     Show outgoing calls (what symbol calls)
  --file <path>    Show full call graph for a file

Modes:
  palace graph <symbol>              # Who calls this symbol?
  palace graph <symbol> --out <file> # What does this symbol call?
  palace graph --file <path>         # Full call graph for file

Examples:
  palace graph handleAuth            # Who calls handleAuth?
  palace graph Search --out butler.go # What does Search call?
  palace graph --file internal/cli/cli.go
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
	case "learn":
		fmt.Print(`palace learn - Store a learning for future reference

Usage: palace learn "content" [options]

Options:
  --scope <scope>      Scope: file, room, palace (default: palace)
  --path <path>        Scope path (file path or room name)
  --confidence <n>     Initial confidence 0.0-1.0 (default: 0.5)

Learnings are knowledge captured during sessions:
  - Palace scope: Applies workspace-wide
  - Room scope: Applies to a logical module/room
  - File scope: Applies to a specific file

Confidence increases when learnings prove helpful,
and decreases when they prove unhelpful.

Examples:
  palace learn "Always run tests before committing"
  palace learn "This file has fragile regex" --scope file --path src/parser.go
  palace learn "Auth module requires DB" --scope room --path auth
`)
	case "recall":
		fmt.Print(`palace recall - Search and retrieve learnings

Usage: palace recall [query] [options]

Options:
  --scope <scope>    Filter by scope (file, room, palace)
  --path <path>      Filter by scope path
  --limit <n>        Maximum results (default: 10)

Without a query, returns all learnings matching filters.
With a query, searches learning content.

Examples:
  palace recall                           # All learnings
  palace recall "testing"                 # Search for testing-related
  palace recall --scope file              # All file-scoped learnings
  palace recall --scope room --path auth  # Auth room learnings
`)
	case "intel":
		fmt.Print(`palace intel - Show file intelligence

Usage: palace intel <file-path>

Shows file intelligence including:
  - Edit count (how many times this file was edited)
  - Failure count (how many edits led to failures)
  - Failure rate percentage
  - Last edit time and editor
  - Associated learnings

This helps identify "fragile" files that often cause problems.

Example:
  palace intel src/auth/login.go
`)
	case "brief":
		fmt.Print(`palace brief - Get a briefing before working

Usage: palace brief [file-path]

Provides a comprehensive briefing including:
  - Active agents in the workspace
  - Conflict warnings (if file specified)
  - File history and failure rate
  - Relevant learnings
  - Hotspot files (most edited)

Use before starting work to understand the current state.

Examples:
  palace brief                    # General workspace briefing
  palace brief src/auth/login.go  # Briefing for specific file
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
		fmt.Println(explainAll())
	default:
		return fmt.Errorf("unknown help topic: %s\n\nAvailable topics: init, scan, check, query, context, graph, serve, ci, session, learn, recall, intel, brief, artifacts", topic)
	}
	return nil
}

// ============================================================================
// Utility Functions
// ============================================================================

func checkForUpdates(args []string) {
	if len(args) == 0 {
		return
	}
	cmd := args[0]
	if cmd == "version" || cmd == "--version" || cmd == "-v" || cmd == "update" {
		return
	}

	cacheDir, err := update.GetCacheDir()
	if err != nil {
		return
	}

	result, err := update.CheckCached(buildVersion, cacheDir)
	if err != nil {
		return
	}

	if result.UpdateAvailable {
		fmt.Fprintf(os.Stderr, "Update available: v%s -> v%s (run 'palace update')\n\n", result.CurrentVersion, result.LatestVersion)
	}
}

type boolFlag struct {
	value bool
	set   bool
}

func (b *boolFlag) Set(s string) error {
	if s == "" {
		b.value = true
		b.set = true
		return nil
	}
	switch strings.ToLower(s) {
	case "true", "1":
		b.value = true
	case "false", "0":
		b.value = false
	default:
		return fmt.Errorf("invalid boolean %q", s)
	}
	b.set = true
	return nil
}

func (b *boolFlag) String() string {
	if b.value {
		return "true"
	}
	return "false"
}

func (b *boolFlag) IsBoolFlag() bool { return true }

func printScope(cmd string, fullScope bool, source string, diffRange string, fileCount int, rootPath string) {
	mode := "diff"
	if fullScope {
		mode = "full"
	}
	if source == "" {
		if fullScope {
			source = "full-scan"
		} else {
			source = "git-diff/change-signal"
		}
	}
	fmt.Printf("Scope (%s):\n", cmd)
	fmt.Printf("  root: %s\n", rootPath)
	fmt.Printf("  mode: %s\n", mode)
	fmt.Printf("  source: %s\n", source)
	fmt.Printf("  fileCount: %d\n", fileCount)
	if !fullScope {
		fmt.Printf("  diffRange: %s\n", strings.TrimSpace(diffRange))
	}
}

func mustAbs(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

func scopeFileCount(cp model.ContextPack) int {
	if cp.Scope == nil {
		return 0
	}
	return cp.Scope.FileCount
}

func truncateLine(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func explainAll() string {
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
