package commands

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/butler"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/flags"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/util"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/config"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/jsonc"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/validate"
)

func init() {
	// Set up butler JSONC decoder
	butler.SetJSONCDecoder(jsonc.DecodeFile)

	Register(&Command{
		Name:        "explore",
		Description: "Search code, get context, or trace call relationships",
		Run:         RunExplore,
	})
}

// ExploreOptions contains the configuration for the explore command.
type ExploreOptions struct {
	Root       string
	Query      string
	Room       string
	Limit      int
	FuzzyMatch bool
	// Mode flags
	Full      bool   // --full: get full context with learnings, decisions
	Map       string // --map: trace call graph (symbol name or "file")
	File      string // --file: file path for map mode
	Depth     int    // --depth: recursion depth for call chain tracing
	Direction string // --direction: up, down, or both
}

// RunExplore executes the explore command with parsed arguments.
func RunExplore(args []string) error {
	fs := flag.NewFlagSet("explore", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	room := fs.String("room", "", "filter to specific room")
	limit := fs.Int("limit", 10, "maximum results")
	fuzzy := fs.Bool("fuzzy", false, "enable fuzzy matching for typo tolerance")
	full := fs.Bool("full", false, "get full context including learnings and decisions")
	mapSymbol := fs.String("map", "", "trace call graph for a symbol or use --map --file for file graph")
	file := fs.String("file", "", "file path for --map mode")
	depth := fs.Int("depth", 0, "recursion depth for call chain tracing (1-10)")
	direction := fs.String("direction", "up", "trace direction: up (callers), down (callees), or both")
	listRooms := fs.Bool("rooms", false, "list all configured rooms")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate inputs
	if err := flags.ValidateLimit(*limit); err != nil {
		return err
	}

	// List rooms mode
	if *listRooms {
		return runExploreListRooms(*root)
	}

	remaining := fs.Args()

	// Determine mode based on flags
	if *mapSymbol != "" || *file != "" {
		// Map mode: trace call relationships
		// If depth > 0, use call chain tracing
		if *depth > 0 {
			return runExploreCallChain(*root, *mapSymbol, *file, *depth, *direction)
		}
		return runExploreMap(*root, *mapSymbol, *file)
	}

	// Need a query for search or full context
	if len(remaining) == 0 {
		return errors.New(`usage: palace explore <query> [options]

Search the codebase:
  palace explore "auth logic"
  palace explore "auth" --room auth
  palace explore "auth" --fuzzy

Get full context for a task:
  palace explore "add user authentication" --full

List configured rooms:
  palace explore --rooms

Trace call relationships (direct):
  palace explore --map handleAuth              # Who calls handleAuth?
  palace explore --map Search --file butler.go # What does Search call?
  palace explore --map --file cli.go           # Full call graph for file

Trace call chains (recursive):
  palace explore --map save --depth 3                    # Trace 3 levels of callers
  palace explore --map save --depth 3 --direction up     # Who calls save's callers?
  palace explore --map init --depth 3 --direction down   # What does init call, recursively?
  palace explore --map auth --depth 2 --direction both   # Both directions`)
	}
	query := strings.Join(remaining, " ")

	if *full {
		// Full context mode
		return runExploreFull(*root, query)
	}

	// Default: search mode
	return runExploreSearch(ExploreOptions{
		Root:       *root,
		Query:      query,
		Room:       *room,
		Limit:      *limit,
		FuzzyMatch: *fuzzy,
	})
}

// runExploreSearch performs a code search.
func runExploreSearch(opts ExploreOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
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

	results, err := b.Search(opts.Query, butler.SearchOptions{
		Limit:      opts.Limit,
		RoomFilter: opts.Room,
		FuzzyMatch: opts.FuzzyMatch,
	})
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	fmt.Printf("\n🔍 Search results for: \"%s\"\n", opts.Query)
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
				fmt.Printf("     │ %s\n", util.TruncateLine(line, 70))
			}
			fmt.Println()
		}
	}

	return nil
}

// runExploreFull generates full context for a task.
func runExploreFull(root, goal string) error {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if _, err := config.EnsureLayout(rootPath); err != nil {
		return err
	}

	cpPath := filepath.Join(rootPath, ".palace", "outputs", "context-pack.json")
	cp := model.NewContextPack(goal)
	if existing, err := model.LoadContextPack(cpPath); err == nil {
		cp = existing
	}
	cp.Goal = goal
	cp.Provenance.UpdatedBy = "palace explore --full"
	cp.Provenance.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if cp.Provenance.CreatedBy == "" {
		cp.Provenance.CreatedBy = "palace explore --full"
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
	fmt.Printf("Context pack updated at %s\n", cpPath)
	fmt.Printf("Goal: %s\n", goal)
	return nil
}

// runExploreMap traces call relationships.
func runExploreMap(root, symbol, filePath string) error {
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

	// If only --file is provided (no symbol), show full file graph
	if symbol == "" && filePath != "" {
		return runExploreMapFile(db, filePath)
	}

	// If symbol and file provided, show callees (what does symbol call?)
	if symbol != "" && filePath != "" {
		return runExploreMapCallees(db, symbol, filePath)
	}

	// If only symbol provided, show callers (who calls symbol?)
	if symbol != "" {
		return runExploreMapCallers(db, symbol)
	}

	return errors.New("usage: palace explore --map <symbol> or palace explore --map --file <file>")
}

// runExploreMapCallers shows who calls a symbol.
func runExploreMapCallers(db *sql.DB, symbol string) error {
	calls, err := index.GetIncomingCalls(db, symbol)
	if err != nil {
		return fmt.Errorf("get callers: %w", err)
	}

	if len(calls) == 0 {
		fmt.Printf("No callers found for '%s'\n", symbol)
		fmt.Println("This symbol may not be called anywhere, or call tracking may not be available for this language.")
		return nil
	}

	fmt.Printf("\n🔍 Callers of '%s'\n", symbol)
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

// runExploreMapCallees shows what a symbol calls.
func runExploreMapCallees(db *sql.DB, symbol, filePath string) error {
	calls, err := index.GetOutgoingCalls(db, symbol, filePath)
	if err != nil {
		return fmt.Errorf("get callees: %w", err)
	}

	if len(calls) == 0 {
		fmt.Printf("No outgoing calls found for '%s' in '%s'\n", symbol, filePath)
		return nil
	}

	fmt.Printf("\n🔍 Functions called by '%s'\n", symbol)
	fmt.Printf("   File: %s\n", filePath)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Found %d function calls:\n\n", len(calls))

	for _, call := range calls {
		fmt.Printf("  📞 %s (line %d)\n", call.CalleeSymbol, call.Line)
	}

	return nil
}

// runExploreMapFile shows the full call graph for a file.
func runExploreMapFile(db *sql.DB, filePath string) error {
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

// runExploreCallChain traces call chains recursively.
func runExploreCallChain(root, symbol, filePath string, depth int, direction string) error {
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

	if symbol == "" {
		return errors.New("symbol is required for call chain tracing")
	}

	// Validate direction
	if direction != "up" && direction != "down" && direction != "both" {
		direction = "up"
	}

	result, err := index.GetCallChain(db, symbol, filePath, direction, depth)
	if err != nil {
		return fmt.Errorf("trace call chain: %w", err)
	}

	// Print header
	dirLabel := map[string]string{
		"up":   "⬆️  UPSTREAM (who calls this)",
		"down": "⬇️  DOWNSTREAM (what this calls)",
		"both": "⬆️⬇️ BOTH DIRECTIONS",
	}[direction]

	fmt.Printf("\n🔗 Call Chain for '%s'\n", symbol)
	fmt.Println(strings.Repeat("═", 60))
	fmt.Printf("Direction: %s | Max Depth: %d\n", dirLabel, depth)
	fmt.Println(strings.Repeat("─", 60))

	if len(result.Chains) == 0 {
		fmt.Printf("\nNo call chains found for '%s'\n", symbol)
		fmt.Println("The symbol may not be called/calling anything, or call tracking may not be available.")
		return nil
	}

	// Print the tree
	fmt.Printf("\n📍 %s ← TARGET\n", symbol)
	printCallChainTree(result.Chains, "")

	// Print summary
	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Found %d call paths", result.TotalPaths)
	if result.Truncated {
		fmt.Print(" (truncated, max 100 paths)")
	}
	fmt.Println()

	return nil
}

// printCallChainTree recursively prints the call chain as a tree.
func printCallChainTree(nodes []*index.CallChainNode, prefix string) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1

		// Choose the branch character
		branch := "├─"
		if isLast {
			branch = "└─"
		}

		// Print the node
		location := ""
		if node.FilePath != "" {
			location = fmt.Sprintf(" (%s:%d)", node.FilePath, node.Line)
		}
		fmt.Printf("%s%s %s%s\n", prefix, branch, node.Symbol, location)

		// Calculate prefix for children
		childPrefix := prefix
		if isLast {
			childPrefix += "   "
		} else {
			childPrefix += "│  "
		}

		// Recursively print children
		if len(node.Children) > 0 {
			printCallChainTree(node.Children, childPrefix)
		}
	}
}

// runExploreListRooms lists all configured rooms.
func runExploreListRooms(root string) error {
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

	b, err := butler.New(db, rootPath)
	if err != nil {
		return fmt.Errorf("initialize butler: %w", err)
	}

	rooms := b.ListRooms()

	if len(rooms) == 0 {
		fmt.Println("No rooms configured.")
		fmt.Println("Rooms are defined in .palace/rooms/*.jsonc files.")
		return nil
	}

	fmt.Printf("\n🏠 Configured Rooms (%d)\n", len(rooms))
	fmt.Println(strings.Repeat("─", 60))

	for i := range rooms {
		room := &rooms[i]
		fmt.Printf("\n📁 %s\n", room.Name)
		if room.Summary != "" {
			fmt.Printf("   %s\n", room.Summary)
		}
		if len(room.EntryPoints) > 0 {
			fmt.Printf("   Entry points: %d\n", len(room.EntryPoints))
		}
		if len(room.Capabilities) > 0 {
			fmt.Printf("   Capabilities: %s\n", strings.Join(room.Capabilities, ", "))
		}
	}

	return nil
}
