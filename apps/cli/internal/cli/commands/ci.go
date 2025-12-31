package commands

import (
	"errors"
	"fmt"
)

func init() {
	Register(&Command{
		Name:        "ci",
		Description: "CI/CD integration commands (shortcuts for check --collect/--signal)",
		Run:         RunCI,
	})
}

// RunCI dispatches to the appropriate CI subcommand.
// These are shortcuts for the check command with specific flags.
func RunCI(args []string) error {
	if len(args) == 0 {
		return errors.New(`usage: palace ci <command>

Commands:
  verify   Check if index is fresh (alias for: palace check)
  collect  Generate context pack from diff (alias for: palace check --collect)
  signal   Generate change signal from diff (alias for: palace check --signal)

Examples:
  palace ci verify --diff HEAD~1..HEAD
  palace ci collect --diff HEAD~1..HEAD
  palace ci signal --diff HEAD~1..HEAD

Note: These are shortcuts. You can also use:
  palace check --diff HEAD~1..HEAD --collect --signal`)
	}

	switch args[0] {
	case "verify":
		// Just run check
		return RunCheck(args[1:])
	case "collect":
		// Run check with --collect flag
		return RunCheck(append([]string{"--collect"}, args[1:]...))
	case "signal":
		// Run check with --signal flag
		return RunCheck(append([]string{"--signal"}, args[1:]...))
	default:
		return fmt.Errorf("unknown ci command: %s\nRun 'palace ci' for usage", args[0])
	}
}
