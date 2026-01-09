package commands

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/flags"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/corridor"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

func init() {
	Register(&Command{
		Name:        "clean",
		Aliases:     []string{"maintenance"}, // Keep maintenance as alias for backwards compatibility
		Description: "Clean up stale data (sessions, agents, learnings, links)",
		Run:         RunClean,
	})
}

// CleanOptions contains the configuration for the clean command.
type CleanOptions struct {
	Root   string
	DryRun bool
}

// RunClean executes the clean command with parsed arguments.
func RunClean(args []string) error {
	fs := flag.NewFlagSet("clean", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	dryRun := fs.Bool("dry-run", false, "show what would be done without making changes")
	if err := fs.Parse(args); err != nil {
		return err
	}

	return ExecuteClean(CleanOptions{
		Root:   *root,
		DryRun: *dryRun,
	})
}

// ExecuteClean performs clean tasks on the workspace.
func ExecuteClean(opts CleanOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	fmt.Println("\n🧹 Palace Maintenance (Clean)")
	fmt.Println(strings.Repeat("─", 60))
	if opts.DryRun {
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
		if opts.DryRun {
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
		if opts.DryRun {
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
		if opts.DryRun {
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
		if opts.DryRun {
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
			if opts.DryRun {
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
	if opts.DryRun {
		fmt.Println("Run without --dry-run to apply changes.")
	} else {
		fmt.Println("✓ Maintenance complete")
	}
	return nil
}
