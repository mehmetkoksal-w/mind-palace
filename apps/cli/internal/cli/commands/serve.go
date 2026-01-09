package commands

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

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
}

// RunServe executes the serve command with parsed arguments.
func RunServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}

	return ExecuteServe(ServeOptions{Root: *root})
}

// ExecuteServe starts the MCP server with the given options.
func ExecuteServe(opts ServeOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

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

	server := butler.NewMCPServer(b)

	fmt.Fprintln(os.Stderr, "Mind Palace MCP server started. Reading JSON-RPC from stdin...")

	return server.Serve()
}
