package cli

import "fmt"

var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildDate    = "unknown"
)

// SetBuildInfo configures build metadata for the CLI.
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

func cmdVersion() error {
	fmt.Printf("palace %s (commit %s, built %s)\n", buildVersion, buildCommit, buildDate)
	return nil
}
