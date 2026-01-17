package commands

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/butler"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/flags"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
)

func init() {
	Register(&Command{
		Name:        "serve",
		Description: "Start MCP server for AI agents",
		Run:         RunServe,
	})
}

// ServeOptions contains the configuration for the serve command.
type ServeOptions struct {
	Root string
	Mode string // MCP mode: "agent" or "human"
}

// RunServe executes the serve command with parsed arguments.
func RunServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	mode := fs.String("mode", "agent", "MCP mode: 'agent' (restricted, default) or 'human' (full access)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	return ExecuteServe(ServeOptions{Root: *root, Mode: *mode})
}

// ExecuteServe starts the MCP server with the given options.
func ExecuteServe(opts ServeOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	// Validate mode
	if !butler.IsValidMCPMode(opts.Mode) {
		return fmt.Errorf("invalid mode %q; must be 'agent' or 'human'", opts.Mode)
	}
	mcpMode := butler.MCPMode(opts.Mode)

	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("index missing; run 'palace scan' first: %w", err)
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

	server := butler.NewMCPServerWithMode(b, mcpMode)

	modeDesc := "agent (restricted)"
	if mcpMode == butler.MCPModeHuman {
		modeDesc = "human (full access)"
	}
	fmt.Fprintf(os.Stderr, "Mind Palace MCP server started in %s mode. Reading JSON-RPC from stdin...\n", modeDesc)
	if mcpMode == butler.MCPModeHuman {
		fmt.Fprintln(os.Stderr, "WARNING: Human mode enables direct-write and governance bypass tools.")
	}

	// Log restricted tools in agent mode for visibility
	if mcpMode == butler.MCPModeAgent {
		restricted := []string{}
		for tool := range butler.GetAdminOnlyTools() {
			restricted = append(restricted, tool)
		}
		if len(restricted) > 0 {
			fmt.Fprintf(os.Stderr, "Restricted tools (human mode only): %s\n", strings.Join(restricted, ", "))
		}
	}

	return server.Serve()
}
