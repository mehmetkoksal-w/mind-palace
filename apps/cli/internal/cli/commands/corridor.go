package commands

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/flags"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/util"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/corridor"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

func init() {
	Register(&Command{
		Name:        "corridor",
		Description: "Cross-workspace knowledge sharing",
		Run:         RunCorridor,
	})
}

// RunCorridor is the main entry point for the corridor command
func RunCorridor(args []string) error {
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
		return ExecuteCorridorList(args[1:])
	case "link":
		return ExecuteCorridorLink(args[1:])
	case "unlink":
		return ExecuteCorridorUnlink(args[1:])
	case "personal":
		return ExecuteCorridorPersonal(args[1:])
	case "promote":
		return ExecuteCorridorPromote(args[1:])
	case "search":
		return ExecuteCorridorSearch(args[1:])
	default:
		return fmt.Errorf("unknown corridor command: %s\nRun 'palace help corridor' for usage", args[0])
	}
}

// ExecuteCorridorList lists linked workspaces and personal corridor stats
func ExecuteCorridorList(_ []string) error {
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

// ExecuteCorridorLink links another workspace
func ExecuteCorridorLink(args []string) error {
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

// ExecuteCorridorUnlink removes a workspace link
func ExecuteCorridorUnlink(args []string) error {
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

// ExecuteCorridorPersonal shows personal corridor learnings
func ExecuteCorridorPersonal(args []string) error {
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

// ExecuteCorridorPromote promotes a workspace learning to personal corridor
func ExecuteCorridorPromote(args []string) error {
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
	fmt.Printf("  Content: %s\n", util.TruncateLine(found.Content, 50))
	fmt.Printf("  From: %s\n", workspaceName)
	return nil
}

// ExecuteCorridorSearch searches learnings across corridors
func ExecuteCorridorSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	all := fs.Bool("all", false, "search all linked workspaces too")
	limit := fs.Int("limit", 20, "maximum results")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate inputs
	if err := flags.ValidateLimit(*limit); err != nil {
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
			fmt.Printf("  • [%.0f%%] %s\n", l.Confidence*100, util.TruncateLine(l.Content, 50))
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
				fmt.Printf("  • [%.0f%%] [%s] %s\n", l.Confidence*100, scope, util.TruncateLine(l.Content, 40))
			}
		}
	}

	fmt.Println()
	return nil
}
