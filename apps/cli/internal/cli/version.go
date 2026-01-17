package cli

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/update"
)

var (
	buildVersion = ""
	buildCommit  = ""
	buildDate    = ""
)

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if buildCommit == "unknown" || buildCommit == "" {
					buildCommit = setting.Value
				}
			case "vcs.time":
				if buildDate == "unknown" || buildDate == "" {
					buildDate = setting.Value
				}
			}
		}
	}
}

// SetBuildInfo sets the build information from ldflags or other sources.
func SetBuildInfo(version, commit, date string) {
	if version != "" && version != "dev" {
		buildVersion = version
	}
	if commit != "" && commit != "unknown" {
		buildCommit = commit
	}
	if date != "" && date != "unknown" {
		buildDate = date
	}
}

// GetVersion returns the current build version.
func GetVersion() string {
	if buildVersion != "" && buildVersion != "dev" {
		return buildVersion
	}

	// Dynamic fallback for development: try to find the VERSION file
	// by climbing up from the current directory.
	if v, err := findVersionFile(); err == nil {
		return v
	}

	return "dev"
}

func findVersionFile() (string, error) {
	// Simple implementation: check common locations relative to project roots
	paths := []string{
		"VERSION",
		"../VERSION",
		"../../VERSION",
		"../../../VERSION",
		"../../../../VERSION",
	}

	for _, p := range paths {
		if content, err := os.ReadFile(p); err == nil {
			return strings.TrimSpace(string(content)), nil
		}
	}
	return "", fmt.Errorf("version file not found")
}

func cmdVersion(args []string) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	check := fs.Bool("check", false, "check for updates")
	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Printf("palace %s (commit %s, built %s)\n", GetVersion(), buildCommit, buildDate)

	if *check {
		result, err := update.Check(GetVersion())
		if err != nil {
			fmt.Printf("\nUpdate check failed: %v\n", err)
			return nil
		}

		if result.UpdateAvailable {
			fmt.Printf("\nUpdate available: v%s -> v%s\n", result.CurrentVersion, result.LatestVersion)
			fmt.Printf("Run 'palace update' to install\n")
		} else {
			fmt.Printf("\nYou are running the latest version.\n")
		}
	}

	return nil
}
