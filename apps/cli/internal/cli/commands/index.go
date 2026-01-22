package commands

import (
	"flag"
	"fmt"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/cli/flags"
)

func init() {
	Register(&Command{
		Name:        "index",
		Aliases:     []string{"idx"},
		Description: "Manage the code index: scan, check, or show stats",
		Run:         RunIndex,
	})
}

// RunIndex executes the index command with subcommand routing.
func RunIndex(args []string) error {
	// Check for subcommands first
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "scan":
			return RunScan(args[1:])
		case "check":
			return RunCheck(args[1:])
		case "stats":
			return runIndexStats(args[1:])
		}
	}

	// No subcommand provided - show help for index
	return indexUsage()
}

// indexUsage shows the usage for the index command.
func indexUsage() error {
	fmt.Println(`Usage: palace index <subcommand> [options]

Manage the code index for your workspace.

Subcommands:
  scan      Build or refresh the code index
  check     Verify index freshness and optionally generate CI outputs
  stats     Show detailed index statistics

Examples:
  palace index scan                  # Incremental scan (auto-detects git)
  palace index scan --full           # Force full rescan
  palace index check                 # Verify index is fresh
  palace index check --diff HEAD~5   # Check against a diff range
  palace index stats                 # Show index statistics

Aliases:
  palace scan    → palace index scan
  palace check   → palace index check

For more details on a subcommand:
  palace index scan --help
  palace index check --help`)
	return nil
}

// runIndexStats shows index statistics (subset of status --full).
func runIndexStats(args []string) error {
	fs := flag.NewFlagSet("index stats", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Reuse the status full functionality but only index stats
	return executeIndexStats(*root)
}

// executeIndexStats shows only index statistics.
func executeIndexStats(root string) error {
	// Reuse getStatusIndexStats from status.go
	indexStats, err := getStatusIndexStats(root)
	if err != nil {
		return fmt.Errorf("index not available: %w", err)
	}

	fmt.Println()
	fmt.Println("Index Statistics")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("  Files indexed:      %d\n", indexStats.FileCount)
	fmt.Printf("  Chunks:             %d\n", indexStats.ChunkCount)
	fmt.Printf("  Symbols:            %d\n", indexStats.SymbolCount)

	// Print symbols by kind
	if len(indexStats.SymbolsByKind) > 0 {
		for kind, count := range indexStats.SymbolsByKind {
			fmt.Printf("    - %-14s %d\n", kind+":", count)
		}
	}

	fmt.Printf("  Relationships:      %d\n", indexStats.RelationshipCount)

	// Print relationships by kind
	if len(indexStats.RelationshipsByKind) > 0 {
		for kind, count := range indexStats.RelationshipsByKind {
			fmt.Printf("    - %-14s %d\n", kind+":", count)
		}
	}

	if !indexStats.LastScan.IsZero() {
		fmt.Printf("  Last scan:          %s\n", indexStats.LastScan.Format("2006-01-02 15:04:05"))
	}
	if indexStats.ScanHash != "" {
		hash := indexStats.ScanHash
		if len(hash) > 12 {
			hash = hash[:12] + "..."
		}
		fmt.Printf("  Scan hash:          %s\n", hash)
	}

	fmt.Println()
	return nil
}
