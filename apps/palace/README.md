# Palace CLI

This directory provides build tooling for the Palace CLI.

## Structure

The actual Go source code is in `cmd/palace/` (following Go convention).
This directory contains:
- `Makefile` - Build commands for the CLI

## Building

From this directory:
```bash
make build    # Build the CLI binary
make run      # Run the CLI (use ARGS="command" for arguments)
make test     # Run tests
make install  # Install to GOPATH/bin
```

Or from the project root:
```bash
make build-palace
```

## Source Code

See `../../cmd/palace/main.go` for the entry point.
