package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/cli/flags"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/cli/util"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/memory"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/patterns"

	// Import detectors to register them
	_ "github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/patterns/detectors"
)

func init() {
	Register(&Command{
		Name:        "patterns",
		Aliases:     []string{"pattern", "pat"},
		Description: "Manage detected code patterns",
		Run:         RunPatterns,
	})
}

// PatternsOptions contains the configuration for the patterns command.
type PatternsOptions struct {
	Root          string
	Subcommand    string
	Category      string
	Status        string
	MinConfidence float64
	Limit         int
	PatternID     string
	Bulk          bool
	DryRun        bool
	WithLearning  bool
}

// RunPatterns executes the patterns command.
func RunPatterns(args []string) error {
	if len(args) == 0 {
		return showPatternsHelp()
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "scan":
		return RunPatternsScan(subArgs)
	case "list", "ls":
		return RunPatternsList(subArgs)
	case "approve":
		return RunPatternsApprove(subArgs)
	case "ignore":
		return RunPatternsIgnore(subArgs)
	case "show":
		return RunPatternsShow(subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\nRun 'palace patterns' for usage", subcommand)
	}
}

func showPatternsHelp() error {
	fmt.Println(`Usage: palace patterns <subcommand> [options]

Manage detected code patterns.

Subcommands:
  scan      Scan codebase for patterns
  list      List detected patterns
  approve   Approve a pattern for enforcement
  ignore    Ignore a pattern
  show      Show pattern details

Examples:
  palace patterns scan
  palace patterns list --status discovered
  palace patterns list --min-confidence 0.85
  palace patterns approve pat_abc123
  palace patterns approve --bulk --min-confidence 0.95
  palace patterns ignore pat_xyz789`)
	return nil
}

// RunPatternsScan runs pattern detection on the codebase.
func RunPatternsScan(args []string) error {
	fs := flag.NewFlagSet("patterns scan", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	category := fs.String("category", "", "filter by category (api, errors, structural, etc.)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	return ExecutePatternsScan(PatternsOptions{
		Root:     *root,
		Category: *category,
	})
}

// ExecutePatternsScan scans the codebase for patterns.
func ExecutePatternsScan(opts PatternsOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	// Verify palace is initialized
	palacePath := filepath.Join(rootPath, ".palace")
	if _, err := os.Stat(palacePath); os.IsNotExist(err) {
		return fmt.Errorf("palace not initialized in %s. Run 'palace init' first", rootPath)
	}

	fmt.Println("Scanning for patterns...")

	// Open memory
	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	// Collect files
	files, err := patterns.CollectFiles(rootPath, nil)
	if err != nil {
		return fmt.Errorf("collect files: %w", err)
	}

	fmt.Printf("Found %d files to analyze\n", len(files))

	// Create engine
	engine := patterns.NewEngine(patterns.DefaultRegistry, mem, rootPath)

	// Configure
	cfg := patterns.DefaultEngineConfig()
	if opts.Category != "" {
		cfg.Categories = []patterns.PatternCategory{patterns.PatternCategory(opts.Category)}
	}
	engine.WithConfig(cfg)

	// Run scan
	ctx := context.Background()
	result, err := engine.Scan(ctx, files)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	// Save results
	if err := engine.SaveResults(ctx, result.Patterns); err != nil {
		return fmt.Errorf("save results: %w", err)
	}

	// Print summary
	fmt.Println()
	fmt.Printf("Scan complete in %s\n", result.Duration.Round(100))
	fmt.Printf("  Files scanned: %d\n", result.FilesScanned)
	fmt.Printf("  Detectors run: %d\n", result.DetectorsRun)
	fmt.Printf("  Patterns found: %d\n", len(result.Patterns))

	if len(result.Errors) > 0 {
		fmt.Printf("  Errors: %d\n", len(result.Errors))
	}

	// Show discovered patterns
	if len(result.Patterns) > 0 {
		fmt.Println()
		fmt.Println("Discovered patterns:")
		for _, p := range result.Patterns {
			level := patterns.GetConfidenceLevel(p.Confidence)
			icon := "?"
			switch level {
			case patterns.ConfidenceHigh:
				icon = "+"
			case patterns.ConfidenceMedium:
				icon = "~"
			case patterns.ConfidenceLow:
				icon = "-"
			}
			fmt.Printf("  [%s] %s (%.0f%%) - %d locations\n",
				icon, p.Name, p.Confidence*100, len(p.Locations))
		}
		fmt.Println()
		fmt.Println("Use 'palace patterns list' to see all patterns")
		fmt.Println("Use 'palace patterns approve <id>' to approve patterns")
	}

	return nil
}

// RunPatternsList lists detected patterns.
func RunPatternsList(args []string) error {
	fs := flag.NewFlagSet("patterns list", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	category := fs.String("category", "", "filter by category")
	status := fs.String("status", "", "filter by status: discovered, approved, ignored")
	minConfidence := fs.Float64("min-confidence", 0, "minimum confidence threshold (0-1)")
	limit := fs.Int("limit", 50, "maximum patterns to show")
	if err := fs.Parse(args); err != nil {
		return err
	}

	return ExecutePatternsList(PatternsOptions{
		Root:          *root,
		Category:      *category,
		Status:        *status,
		MinConfidence: *minConfidence,
		Limit:         *limit,
	})
}

// ExecutePatternsList lists patterns matching criteria.
func ExecutePatternsList(opts PatternsOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	patternList, err := mem.GetPatterns(memory.PatternFilters{
		Category:      opts.Category,
		Status:        opts.Status,
		MinConfidence: opts.MinConfidence,
		Limit:         opts.Limit,
	})
	if err != nil {
		return fmt.Errorf("get patterns: %w", err)
	}

	if len(patternList) == 0 {
		fmt.Println("No patterns found.")
		fmt.Println("Run 'palace patterns scan' to detect patterns.")
		return nil
	}

	// Count by status
	discovered, approved, ignored := 0, 0, 0
	for _, p := range patternList {
		switch p.Status {
		case "discovered":
			discovered++
		case "approved":
			approved++
		case "ignored":
			ignored++
		}
	}

	fmt.Printf("Patterns: %d discovered, %d approved, %d ignored\n",
		discovered, approved, ignored)
	fmt.Println(strings.Repeat("=", 60))

	// Group by category
	byCategory := make(map[string][]memory.Pattern)
	for _, p := range patternList {
		byCategory[p.Category] = append(byCategory[p.Category], p)
	}

	for cat, pats := range byCategory {
		fmt.Printf("\n%s:\n", strings.ToUpper(cat))
		for _, p := range pats {
			statusIcon := "?"
			switch p.Status {
			case "discovered":
				statusIcon = "?"
			case "approved":
				statusIcon = "+"
			case "ignored":
				statusIcon = "x"
			}

			level := patterns.GetConfidenceLevel(p.Confidence)
			confIcon := ""
			switch level {
			case patterns.ConfidenceHigh:
				confIcon = "HIGH"
			case patterns.ConfidenceMedium:
				confIcon = "MED"
			case patterns.ConfidenceLow:
				confIcon = "LOW"
			default:
				confIcon = "???"
			}

			fmt.Printf("  [%s] %s  %s\n", statusIcon, p.ID, p.Name)
			fmt.Printf("      Confidence: %.0f%% (%s) | Detector: %s\n",
				p.Confidence*100, confIcon, p.DetectorID)
		}
	}

	fmt.Println()
	if discovered > 0 {
		fmt.Println("Use 'palace patterns approve <id>' to approve a pattern")
		fmt.Println("Use 'palace patterns approve --bulk --min-confidence 0.95' for bulk approval")
	}

	return nil
}

// RunPatternsApprove approves a pattern.
func RunPatternsApprove(args []string) error {
	fs := flag.NewFlagSet("patterns approve", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	bulk := fs.Bool("bulk", false, "approve all patterns meeting confidence threshold")
	minConfidence := fs.Float64("min-confidence", 0.95, "minimum confidence for bulk approval")
	dryRun := fs.Bool("dry-run", false, "show what would be approved without making changes")
	withLearning := fs.Bool("with-learning", false, "create a learning from the approved pattern(s)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	opts := PatternsOptions{
		Root:          *root,
		Bulk:          *bulk,
		MinConfidence: *minConfidence,
		DryRun:        *dryRun,
		WithLearning:  *withLearning,
	}

	remaining := fs.Args()
	if !opts.Bulk && len(remaining) == 0 {
		return errors.New(`usage: palace patterns approve <pattern-id> [options]
       palace patterns approve --bulk [--min-confidence 0.95] [--dry-run]

Approve pattern(s) for enforcement.

Arguments:
  <pattern-id>       ID of pattern to approve (e.g., pat_abc123)

Options:
  --bulk             Approve all patterns meeting confidence threshold
  --min-confidence   Minimum confidence for bulk approval (default: 0.95)
  --dry-run          Show what would be approved without making changes
  --with-learning    Create a learning from each approved pattern

Examples:
  palace patterns approve pat_abc123
  palace patterns approve pat_abc123 --with-learning
  palace patterns approve --bulk --min-confidence 0.90
  palace patterns approve --bulk --with-learning --dry-run`)
	}

	if len(remaining) > 0 {
		opts.PatternID = remaining[0]
	}

	return ExecutePatternsApprove(opts)
}

// ExecutePatternsApprove approves patterns.
func ExecutePatternsApprove(opts PatternsOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	if opts.Bulk {
		// Bulk approval
		if opts.DryRun {
			// Show what would be approved
			toApprove, err := mem.GetPatterns(memory.PatternFilters{
				Status:        "discovered",
				MinConfidence: opts.MinConfidence,
			})
			if err != nil {
				return err
			}

			if len(toApprove) == 0 {
				fmt.Printf("No patterns with confidence >= %.0f%% to approve.\n",
					opts.MinConfidence*100)
				return nil
			}

			action := "approve"
			if opts.WithLearning {
				action = "approve and create learnings for"
			}
			fmt.Printf("Would %s %d patterns (dry-run):\n", action, len(toApprove))
			for _, p := range toApprove {
				fmt.Printf("  + %s: %s (%.0f%%)\n", p.ID, p.Name, p.Confidence*100)
			}
			return nil
		}

		var count int
		if opts.WithLearning {
			// Bulk approve with learning creation
			var learningMap map[string]string
			count, learningMap, err = mem.BulkApprovePatternsWithLearnings(opts.MinConfidence)
			if err != nil {
				return fmt.Errorf("bulk approve with learnings: %w", err)
			}

			if count == 0 {
				fmt.Printf("No patterns with confidence >= %.0f%% to approve.\n",
					opts.MinConfidence*100)
			} else {
				fmt.Printf("+ Approved %d patterns with confidence >= %.0f%% and created %d learnings\n",
					count, opts.MinConfidence*100, len(learningMap))
			}
		} else {
			count, err = mem.BulkApprovePatterns(opts.MinConfidence)
			if err != nil {
				return fmt.Errorf("bulk approve: %w", err)
			}

			if count == 0 {
				fmt.Printf("No patterns with confidence >= %.0f%% to approve.\n",
					opts.MinConfidence*100)
			} else {
				fmt.Printf("+ Approved %d patterns with confidence >= %.0f%%\n",
					count, opts.MinConfidence*100)
			}
		}
		return nil
	}

	// Single pattern approval
	pattern, err := mem.GetPattern(opts.PatternID)
	if err != nil {
		return fmt.Errorf("get pattern: %w", err)
	}
	if pattern == nil {
		return fmt.Errorf("pattern not found: %s", opts.PatternID)
	}

	if pattern.Status == "approved" {
		fmt.Printf("Pattern %s is already approved.\n", opts.PatternID)
		return nil
	}

	// Show pattern details
	fmt.Printf("Approving pattern: %s\n", opts.PatternID)
	fmt.Printf("Name: %s\n", pattern.Name)
	fmt.Printf("Category: %s\n", pattern.Category)
	fmt.Printf("Confidence: %.0f%%\n", pattern.Confidence*100)

	// Approve
	if opts.WithLearning {
		learningID, err := mem.ApprovePatternWithLearning(opts.PatternID)
		if err != nil {
			return fmt.Errorf("approve pattern with learning: %w", err)
		}
		fmt.Printf("\n+ Approved pattern: %s\n", opts.PatternID)
		fmt.Printf("+ Created learning: %s\n", learningID)
	} else {
		if err := mem.ApprovePattern(opts.PatternID, ""); err != nil {
			return fmt.Errorf("approve pattern: %w", err)
		}
		fmt.Printf("\n+ Approved pattern: %s\n", opts.PatternID)
	}

	return nil
}

// RunPatternsIgnore ignores a pattern.
func RunPatternsIgnore(args []string) error {
	fs := flag.NewFlagSet("patterns ignore", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New(`usage: palace patterns ignore <pattern-id>

Ignore a pattern (won't be enforced or shown in future scans).

Arguments:
  <pattern-id>    ID of pattern to ignore (e.g., pat_abc123)

Examples:
  palace patterns ignore pat_abc123`)
	}

	return ExecutePatternsIgnore(PatternsOptions{
		Root:      *root,
		PatternID: remaining[0],
	})
}

// ExecutePatternsIgnore ignores a pattern.
func ExecutePatternsIgnore(opts PatternsOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	pattern, err := mem.GetPattern(opts.PatternID)
	if err != nil {
		return fmt.Errorf("get pattern: %w", err)
	}
	if pattern == nil {
		return fmt.Errorf("pattern not found: %s", opts.PatternID)
	}

	if pattern.Status == "ignored" {
		fmt.Printf("Pattern %s is already ignored.\n", opts.PatternID)
		return nil
	}

	// Show pattern details
	fmt.Printf("Ignoring pattern: %s\n", opts.PatternID)
	fmt.Printf("Name: %s\n", pattern.Name)

	// Ignore
	if err := mem.IgnorePattern(opts.PatternID); err != nil {
		return fmt.Errorf("ignore pattern: %w", err)
	}

	fmt.Printf("\nx Ignored pattern: %s\n", opts.PatternID)
	return nil
}

// RunPatternsShow shows details of a pattern.
func RunPatternsShow(args []string) error {
	fs := flag.NewFlagSet("patterns show", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New(`usage: palace patterns show <pattern-id>

Show detailed information about a pattern.

Arguments:
  <pattern-id>    ID of pattern to show (e.g., pat_abc123)`)
	}

	return ExecutePatternsShow(PatternsOptions{
		Root:      *root,
		PatternID: remaining[0],
	})
}

// ExecutePatternsShow shows pattern details.
func ExecutePatternsShow(opts PatternsOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	mem, err := memory.Open(rootPath)
	if err != nil {
		return fmt.Errorf("open memory: %w", err)
	}
	defer mem.Close()

	pattern, err := mem.GetPattern(opts.PatternID)
	if err != nil {
		return fmt.Errorf("get pattern: %w", err)
	}
	if pattern == nil {
		return fmt.Errorf("pattern not found: %s", opts.PatternID)
	}

	// Get locations
	locations, err := mem.GetPatternLocations(opts.PatternID)
	if err != nil {
		return fmt.Errorf("get locations: %w", err)
	}

	// Display
	fmt.Printf("Pattern: %s\n", pattern.ID)
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Name:        %s\n", pattern.Name)
	fmt.Printf("Category:    %s/%s\n", pattern.Category, pattern.Subcategory)
	fmt.Printf("Description: %s\n", pattern.Description)
	fmt.Printf("Detector:    %s\n", pattern.DetectorID)
	fmt.Println()
	fmt.Printf("Status:      %s\n", pattern.Status)
	fmt.Printf("Authority:   %s\n", pattern.Authority)
	fmt.Println()
	fmt.Printf("Confidence:  %.1f%%\n", pattern.Confidence*100)
	fmt.Printf("  Frequency:   %.1f%%\n", pattern.FrequencyScore*100)
	fmt.Printf("  Consistency: %.1f%%\n", pattern.ConsistencyScore*100)
	fmt.Printf("  Spread:      %.1f%%\n", pattern.SpreadScore*100)
	fmt.Printf("  Age:         %.1f%%\n", pattern.AgeScore*100)
	fmt.Println()
	fmt.Printf("First seen:  %s\n", pattern.FirstSeen.Format("2006-01-02 15:04"))
	fmt.Printf("Last seen:   %s\n", pattern.LastSeen.Format("2006-01-02 15:04"))

	if len(locations) > 0 {
		matches := 0
		outliers := 0
		for _, loc := range locations {
			if loc.IsOutlier {
				outliers++
			} else {
				matches++
			}
		}

		fmt.Println()
		fmt.Printf("Locations: %d matches, %d outliers\n", matches, outliers)

		// Show first few locations
		shown := 0
		for _, loc := range locations {
			if loc.IsOutlier {
				continue
			}
			if shown >= 5 {
				fmt.Printf("  ... and %d more\n", matches-shown)
				break
			}
			fmt.Printf("  %s:%d\n", loc.FilePath, loc.LineStart)
			if loc.Snippet != "" {
				fmt.Printf("    %s\n", util.TruncateLine(loc.Snippet, 60))
			}
			shown++
		}

		// Show outliers
		if outliers > 0 {
			fmt.Println()
			fmt.Println("Outliers (deviations):")
			shown = 0
			for _, loc := range locations {
				if !loc.IsOutlier {
					continue
				}
				if shown >= 3 {
					fmt.Printf("  ... and %d more\n", outliers-shown)
					break
				}
				fmt.Printf("  %s:%d\n", loc.FilePath, loc.LineStart)
				if loc.OutlierReason != "" {
					fmt.Printf("    Reason: %s\n", loc.OutlierReason)
				}
				shown++
			}
		}
	}

	return nil
}
