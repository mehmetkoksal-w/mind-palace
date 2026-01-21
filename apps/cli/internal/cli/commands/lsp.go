package commands

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/cli/flags"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/lsp"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/memory"
)

func init() {
	Register(&Command{
		Name:        "lsp",
		Description: "Start Language Server Protocol server for editor integration",
		Run:         RunLSP,
	})
}

// LSPOptions contains the configuration for the lsp command.
type LSPOptions struct {
	Root    string
	LogFile string
	Stdio   bool
}

// RunLSP executes the lsp command with parsed arguments.
func RunLSP(args []string) error {
	fs := flag.NewFlagSet("lsp", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	logFile := fs.String("log", "", "Log file path (default: none)")
	stdio := fs.Bool("stdio", true, "Use stdio transport (default)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	return ExecuteLSP(LSPOptions{
		Root:    *root,
		LogFile: *logFile,
		Stdio:   *stdio,
	})
}

// ExecuteLSP starts the LSP server with the given options.
func ExecuteLSP(opts LSPOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}

	// Set up logging
	var logWriter io.Writer = io.Discard // Don't log to stderr by default (it's for JSON-RPC)
	if opts.LogFile != "" {
		logFile, err := os.OpenFile(opts.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("open log file: %w", err)
		}
		defer logFile.Close()
		logWriter = logFile
	}

	// Open memory database if available
	var mem *memory.Memory
	mem, err = memory.Open(rootPath)
	if err != nil {
		// LSP can work without memory, just with limited functionality
		logWriter.Write([]byte(fmt.Sprintf("Memory database not available: %v; diagnostics will be limited\n", err)))
	} else {
		defer mem.Close()
	}

	// Create LSP server
	server := lsp.NewServerWithIO(os.Stdin, os.Stdout)
	server.SetLogger(logWriter)

	// Set up diagnostics provider if we have memory
	if mem != nil {
		adapter := lsp.NewButlerAdapter(mem)
		server.SetDiagnosticsProvider(adapter)
	}

	// Set up context with cancellation on signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		logWriter.Write([]byte("Received shutdown signal\n"))
		cancel()
	}()

	logWriter.Write([]byte(fmt.Sprintf("Mind Palace LSP server started (root: %s)\n", rootPath)))

	// Run the server
	return server.Run(ctx)
}
