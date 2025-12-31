package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/flags"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/util"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/collect"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/lint"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/signal"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/verify"
)

func init() {
	Register(&Command{
		Name:        "check",
		Description: "Verify index freshness and optionally generate CI outputs",
		Run:         RunCheck,
	})
}

// CheckOptions contains the configuration for the check command.
type CheckOptions struct {
	Root       string
	DiffRange  string
	Strict     bool
	Collect    bool // Generate context pack
	Signal     bool // Generate change signal
	AllowStale bool // For collect: allow even if index is stale
}

// RunCheck executes the check command with parsed arguments.
func RunCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	diff := fs.String("diff", "", "diff range for scoped verification")
	var strictFlag flags.BoolFlag
	fs.Var(&strictFlag, "strict", "strict mode (hash all files; slower but thorough)")
	collectFlag := fs.Bool("collect", false, "also generate context pack from diff")
	signalFlag := fs.Bool("signal", false, "also generate change signal from diff")
	allowStale := fs.Bool("allow-stale", false, "for --collect: allow even if index is stale")
	if err := fs.Parse(args); err != nil {
		return err
	}

	return ExecuteCheck(CheckOptions{
		Root:       *root,
		DiffRange:  *diff,
		Strict:     strictFlag.WasSet && strictFlag.Value,
		Collect:    *collectFlag,
		Signal:     *signalFlag,
		AllowStale: *allowStale,
	})
}

// ExecuteCheck performs the check with the given options.
func ExecuteCheck(opts CheckOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
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
	if opts.Strict {
		mode = verify.ModeStrict
	}

	staleList, fullScope, source, candidateCount, err := verify.Run(db, verify.Options{Root: rootPath, DiffRange: opts.DiffRange, Mode: mode})
	if err != nil {
		return err
	}

	util.PrintScope("check", fullScope, source, opts.DiffRange, candidateCount, rootPath)

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

	// Generate context pack if requested
	if opts.Collect {
		result, err := collect.Run(opts.Root, opts.DiffRange, collect.Options{AllowStale: opts.AllowStale})
		if err != nil {
			return fmt.Errorf("collect failed: %w", err)
		}
		cp := result.ContextPack
		for _, warning := range result.CorridorWarnings {
			fmt.Fprintf(os.Stderr, "⚠️  %s\n", warning)
		}
		fmt.Printf("context pack updated from scan %s\n", cp.ScanHash)
	}

	// Generate change signal if requested
	if opts.Signal {
		if opts.DiffRange == "" {
			return errors.New("--signal requires --diff range")
		}
		if _, err := signal.Generate(opts.Root, opts.DiffRange); err != nil {
			return fmt.Errorf("signal generation failed: %w", err)
		}
		fmt.Println("change signal written to .palace/outputs/change-signal.json")
	}

	return nil
}
