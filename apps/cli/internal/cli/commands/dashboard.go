package commands

import (
	"fmt"
)

func init() {
	Register(&Command{
		Name:        "dashboard",
		Description: "Start web dashboard for visualization (deprecated)",
		Run:         RunDashboard,
	})
}

// RunDashboard shows a deprecation notice.
func RunDashboard(args []string) error {
	fmt.Println("The dashboard has been removed in v0.4.2.")
	fmt.Println()
	fmt.Println("Use CLI commands instead:")
	fmt.Println("  palace status           Show workspace status")
	fmt.Println("  palace status --full    Detailed statistics")
	fmt.Println("  palace recall           List knowledge")
	fmt.Println("  palace explore          Search codebase")
	fmt.Println()
	fmt.Println("For AI agents, use the MCP server:")
	fmt.Println("  palace serve            Start MCP server")
	return nil
}
