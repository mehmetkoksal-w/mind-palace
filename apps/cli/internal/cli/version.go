package cli

import (
	"flag"
	"fmt"
	"runtime/debug"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/update"
)

var (
	buildVersion = "0.0.2-alpha"
	buildCommit  = "unknown"
	buildDate    = "unknown"
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
	return buildVersion
}

func cmdVersion(args []string) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	check := fs.Bool("check", false, "check for updates")
	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Printf("palace %s (commit %s, built %s)\n", buildVersion, buildCommit, buildDate)

	if *check {
		result, err := update.Check(buildVersion)
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
