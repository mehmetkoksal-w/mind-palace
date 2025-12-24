package cli

import (
	"flag"
	"fmt"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/update"
)

var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildDate    = "unknown"
)

func SetBuildInfo(version, commit, date string) {
	if version != "" {
		buildVersion = version
	}
	if commit != "" {
		buildCommit = commit
	}
	if date != "" {
		buildDate = date
	}
}

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
