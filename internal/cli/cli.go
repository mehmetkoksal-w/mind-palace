package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/internal/butler"
	"github.com/koksalmehmet/mind-palace/internal/collect"
	"github.com/koksalmehmet/mind-palace/internal/config"
	"github.com/koksalmehmet/mind-palace/internal/index"
	"github.com/koksalmehmet/mind-palace/internal/jsonc"
	"github.com/koksalmehmet/mind-palace/internal/lint"
	"github.com/koksalmehmet/mind-palace/internal/model"
	"github.com/koksalmehmet/mind-palace/internal/project"
	"github.com/koksalmehmet/mind-palace/internal/scan"
	"github.com/koksalmehmet/mind-palace/internal/signal"
	"github.com/koksalmehmet/mind-palace/internal/validate"
	"github.com/koksalmehmet/mind-palace/internal/verify"
)

func init() {
	butler.SetJSONCDecoder(jsonc.DecodeFile)
}

func Run(args []string) error {
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "version", "--version", "-v":
		return cmdVersion()
	case "init":
		return cmdInit(args[1:])
	case "detect":
		return cmdDetect(args[1:])
	case "scan":
		return cmdScan(args[1:])
	case "lint":
		return cmdLint(args[1:])
	case "verify":
		return cmdVerify(args[1:])
	case "plan":
		return cmdPlan(args[1:])
	case "collect":
		return cmdCollect(args[1:])
	case "signal":
		return cmdSignal(args[1:])
	case "explain":
		return cmdExplain(args[1:])
	case "ask":
		return cmdAsk(args[1:])
	case "serve":
		return cmdServe(args[1:])
	case "help", "-h", "--help":
		return usage()
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func usage() error {
	fmt.Println(`palace commands: init | detect | scan | lint | verify | plan | collect | signal | explain | ask | serve

Examples:
  palace init
  palace scan
  palace verify --diff HEAD~1..HEAD
  palace collect --diff HEAD~1..HEAD
  palace explain verify

  # Butler:
  palace ask "where is the auth logic"
  palace ask --room project-overview "entry points"
  palace serve   # Start MCP server for AI agents`)
	return nil
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

func resolveVerifyMode(fastFlag, strictFlag boolFlag) (verify.Mode, error) {
	if fastFlag.set && strictFlag.set && fastFlag.value && strictFlag.value {
		return "", errors.New("verify: --fast and --strict are mutually exclusive")
	}
	if fastFlag.set && !fastFlag.value && !(strictFlag.set && strictFlag.value) {
		return "", errors.New("verify: --fast=false requires --strict")
	}
	if strictFlag.set && strictFlag.value {
		return verify.ModeStrict, nil
	}
	return verify.ModeFast, nil
}

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	force := fs.Bool("force", false, "overwrite existing curated files")
	withOutputs := fs.Bool("with-outputs", false, "also create generated outputs (context-pack)")
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

	replacements := map[string]string{
		"projectName": filepath.Base(rootPath),
		"language":    "unknown",
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

	fmt.Printf("initialized palace scaffolding in %s\n", filepath.Join(rootPath, ".palace"))
	return nil
}

func cmdDetect(args []string) error {
	fs := flag.NewFlagSet("detect", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
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

	profile := project.BuildProfile(rootPath)
	profilePath := filepath.Join(rootPath, ".palace", "project-profile.json")
	if err := config.WriteJSON(profilePath, profile); err != nil {
		return err
	}
	fmt.Printf("generated project profile at %s\n", profilePath)
	return nil
}

func cmdScan(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	summary, fileCount, err := scan.Run(*root)
	if err != nil {
		return err
	}
	fmt.Printf("indexed %d files; scan hash %s\n", fileCount, summary.ScanHash)
	fmt.Printf("scan artifact written to %s\n", filepath.Join(summary.Root, ".palace", "index", "scan.json"))
	return nil
}

func cmdLint(args []string) error {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	if err := lint.Run(rootPath); err != nil {
		return err
	}
	fmt.Println("lint ok")
	return nil
}

func cmdVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	diff := fs.String("diff", "", "diff range for scoped verification")
	var fastFlag boolFlag
	var strictFlag boolFlag
	fs.Var(&fastFlag, "fast", "fast mode (default; mtime/size with selective hashing)")
	fs.Var(&strictFlag, "strict", "strict mode (hash all; disables fast)")
	fastFlag.value = true
	if err := fs.Parse(args); err != nil {
		return err
	}

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	if err := lint.Run(rootPath); err != nil {
		return fmt.Errorf("lint failed: %w", err)
	}

	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("index missing; run palace scan first: %w", err)
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
		return errors.New("no scan records found; run palace scan")
	}

	mode, err := resolveVerifyMode(fastFlag, strictFlag)
	if err != nil {
		return err
	}

	staleList, fullScope, source, candidateCount, err := verify.Run(db, verify.Options{Root: rootPath, DiffRange: *diff, Mode: mode})
	if err != nil {
		return err
	}

	printScope("verify", fullScope, source, *diff, candidateCount, rootPath)

	if len(staleList) > 0 {
		fmt.Println("stale artifacts detected:")
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
		return errors.New("index is stale; rerun palace scan")
	}

	fmt.Printf("verify ok; latest scan %s at %s\n", summary.ScanHash, summary.CompletedAt.Format(time.RFC3339))
	return nil
}

func cmdPlan(args []string) error {
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
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
		return errors.New("goal is required for palace plan")
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
	cp.Provenance.UpdatedBy = "palace plan"
	cp.Provenance.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if cp.Provenance.CreatedBy == "" {
		cp.Provenance.CreatedBy = "palace plan"
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

func cmdCollect(args []string) error {
	fs := flag.NewFlagSet("collect", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	diff := fs.String("diff", "", "optional diff range or matching change signal")
	allowStale := fs.Bool("allow-stale", false, "allow collecting even if the index is stale (full scope only)")
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

func cmdSignal(args []string) error {
	fs := flag.NewFlagSet("signal", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	diff := fs.String("diff", "", "diff range (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*diff) == "" {
		return errors.New("signal requires --diff range")
	}
	if _, err := signal.Generate(*root, *diff); err != nil {
		return err
	}
	fmt.Println("change signal written to .palace/outputs/change-signal.json")
	return nil
}

func cmdExplain(args []string) error {
	topic := "all"
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		topic = strings.TrimSpace(args[0])
	}
	switch topic {
	case "all":
		fmt.Println(explainAll())
	case "scan":
		fmt.Println(explainScan())
	case "collect":
		fmt.Println(explainCollect())
	case "verify":
		fmt.Println(explainVerify())
	case "signal":
		fmt.Println(explainSignal())
	case "artifacts":
		fmt.Println(explainArtifacts())
	default:
		return fmt.Errorf("unknown explain topic: %s (try: scan|collect|verify|signal|artifacts|all)", topic)
	}
	return nil
}


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


func explainAll() string {
	return strings.Join([]string{
		explainScan(),
		"",
		explainCollect(),
		"",
		explainVerify(),
		"",
		explainSignal(),
		"",
		explainArtifacts(),
	}, "\n")
}

func explainScan() string {
	return `scan
  Purpose: Build/refresh the Tier-0 SQLite index and emit an auditable scan summary.
  Inputs: workspace files (excluding guardrails)
  Outputs:
    - .palace/index/palace.db (SQLite WAL + FTS)
    - .palace/index/scan.json (validated, includes UUID + dbScanId + counts + hash)
  Behavior:
    - Always rebuilds index from disk; deterministic chunking + hashing.`
}

func explainCollect() string {
	return `collect
  Purpose: Refresh .palace/outputs/context-pack.json using existing index + curated manifests.
  Inputs:
    - .palace/index/palace.db (must exist)
    - curated manifests (.palace/palace.jsonc, rooms, playbooks)
  Outputs:
    - .palace/outputs/context-pack.json (validated)
  Freshness:
    - Full scope (no --diff): fails if index is stale unless --allow-stale.
    - Diff scope (--diff): uses git diff or a matching change-signal, never widens scope silently.`
}

func explainVerify() string {
	return `verify
  Purpose: Detect staleness between workspace and the latest indexed metadata.
  Modes:
    - --fast  (default): mtime/size shortcut with selective hashing
    - --strict: hash all candidates
  Scope:
    - No --diff: verifies full workspace against stored index.
    - With --diff: verifies only changed paths (git diff or matching change-signal).
  Diff behavior:
    - If diff cannot be computed, verify returns an error (no widening).`
}

func explainSignal() string {
	return `signal
  Purpose: Generate a deterministic change-signal artifact from a git diff range.
  Inputs:
    - git diff --name-status <range>
  Outputs:
    - .palace/outputs/change-signal.json (validated)
  Notes:
    - Handles rename/copy formats; hashes non-deleted files; normalizes paths; sorts changes.`
}

func explainArtifacts() string {
	return `artifacts
  Curated (commit):
    - .palace/palace.jsonc
    - .palace/rooms/*.jsonc
    - .palace/playbooks/*.jsonc
    - .palace/project-profile.json
    - .palace/schemas/* (export-only for transparency; embedded are canonical)
  Generated (ignore):
    - .palace/index/*
    - .palace/outputs/*
    - .palace/sessions/* (if present)
    - *.db artifacts`
}


func cmdAsk(args []string) error {
	fs := flag.NewFlagSet("ask", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	room := fs.String("room", "", "filter to specific room")
	limit := fs.Int("limit", 10, "maximum results")
	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("usage: palace ask <query>")
	}
	query := strings.Join(remaining, " ")

	rootPath, err := filepath.Abs(*root)
	if err != nil {
		return err
	}

	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("index missing; run palace scan first: %w", err)
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
		return fmt.Errorf("index missing; run palace scan first: %w", err)
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

func truncateLine(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
