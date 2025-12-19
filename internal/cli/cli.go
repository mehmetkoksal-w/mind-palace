package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mind-palace/internal/collect"
	"mind-palace/internal/config"
	"mind-palace/internal/index"
	"mind-palace/internal/lint"
	"mind-palace/internal/model"
	"mind-palace/internal/project"
	"mind-palace/internal/scan"
	"mind-palace/internal/signal"
	"mind-palace/internal/validate"
	"mind-palace/internal/verify"
)

// Run dispatches CLI commands.
func Run(args []string) error {
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
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
	case "help", "-h", "--help":
		return usage()
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func usage() error {
	fmt.Println("palace commands: init | detect | scan | lint | verify | plan | collect | signal")
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
	fs.Var(&fastFlag, "fast", "fast mode (mtime/size with selective hashing)")
	fs.Var(&strictFlag, "strict", "strict mode (hash all)")
	fastFlag.value = true // default fast
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

	stale, fallback, err := verify.Run(db, verify.Options{Root: rootPath, DiffRange: *diff, Mode: mode})
	if err != nil {
		return err
	}
	if *diff != "" && fallback {
		fmt.Println("git diff not available; verified entire workspace")
	}
	if len(stale) > 0 {
		fmt.Println("stale artifacts detected:")
		for _, s := range stale {
			fmt.Printf("- %s\n", s)
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
	if err := fs.Parse(args); err != nil {
		return err
	}
	cp, err := collect.Run(*root, *diff)
	if err != nil {
		return err
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
