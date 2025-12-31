package commands

import (
	"flag"
	"fmt"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/update"
)

// BuildVersion is set by the main package to provide version info
var BuildVersion = "dev"

func init() {
	Register(&Command{
		Name:        "update",
		Description: "Update palace to latest version",
		Run:         RunUpdate,
	})
}

// RunUpdate executes the update command.
func RunUpdate(args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	return ExecuteUpdate(BuildVersion)
}

// ExecuteUpdate performs the update with the given current version.
func ExecuteUpdate(currentVersion string) error {
	err := update.Update(currentVersion, func(msg string) {
		fmt.Println(msg)
	})
	if err != nil {
		if err.Error() == "already at latest version" {
			fmt.Printf("palace %s is already the latest version.\n", currentVersion)
			return nil
		}
		return err
	}
	return nil
}
