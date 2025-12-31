package commands

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/butler"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/cli/flags"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/corridor"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/dashboard"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

func init() {
	Register(&Command{
		Name:        "dashboard",
		Description: "Start web dashboard for visualization",
		Run:         RunDashboard,
	})
}

// DashboardOptions contains the configuration for the dashboard command.
type DashboardOptions struct {
	Root      string
	Port      int
	NoBrowser bool
}

// RunDashboard executes the dashboard command with parsed arguments.
func RunDashboard(args []string) error {
	fs := flag.NewFlagSet("dashboard", flag.ContinueOnError)
	root := fs.String("root", ".", "workspace root")
	port := fs.Int("port", 3001, "server port")
	noBrowser := fs.Bool("no-browser", false, "don't open browser automatically")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate inputs
	if err := flags.ValidatePort(*port); err != nil {
		return err
	}

	return ExecuteDashboard(DashboardOptions{
		Root:      *root,
		Port:      *port,
		NoBrowser: *noBrowser,
	})
}

// ExecuteDashboard starts the dashboard server with the given options.
func ExecuteDashboard(opts DashboardOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	// Open memory database
	mem, err := memory.Open(rootPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not open memory database: %v\n", err)
	}

	// Open global corridor
	gc, err := corridor.OpenGlobal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not open global corridor: %v\n", err)
	}

	// Open butler for code search
	dbPath := filepath.Join(rootPath, ".palace", "index", "palace.db")
	var b *butler.Butler
	if _, err := os.Stat(dbPath); err == nil {
		db, err := index.Open(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not open index database: %v\n", err)
		} else {
			b, err = butler.New(db, rootPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Could not open butler: %v\n", err)
			}
		}
	}

	server := dashboard.New(dashboard.Config{
		Butler:   b,
		Memory:   mem,
		Corridor: gc,
		Port:     opts.Port,
		Root:     rootPath,
	})

	fmt.Printf("Starting Mind Palace Dashboard...\n")
	return server.Start(!opts.NoBrowser)
}
