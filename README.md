# Mind Palace

A deterministic context system for codebases, inspired by the [Method of Loci](https://en.wikipedia.org/wiki/Method_of_loci).

[![CI](https://github.com/koksalmehmet/mind-palace/actions/workflows/ci.yml/badge.svg)](https://github.com/koksalmehmet/mind-palace/actions/workflows/ci.yml)
[![Release](https://github.com/koksalmehmet/mind-palace/actions/workflows/release.yml/badge.svg)](https://github.com/koksalmehmet/mind-palace/actions/workflows/release.yml)

## Overview

Mind Palace provides a **deterministic, schema-validated index** for codebases that both humans and AI agents can trust. No embeddings. No guessing. Deterministic.

### Components

| Component             | Description                                      |
| --------------------- | ------------------------------------------------ |
| **Palace CLI**        | Core engine for scanning, indexing, and querying |
| **Dashboard**         | Web UI for visualization and monitoring          |
| **VS Code Extension** | HUD, sidebar, and auto-sync integration          |
| **MCP Server**        | AI agent integration via JSON-RPC                |

### Key Features

- **Session Memory** - Track agent sessions, activities, and learnings
- **Corridors** - Share knowledge across multiple projects
- **Dashboard** - Visual exploration of your codebase
- **MCP Integration** - First-class support for AI agents

## Quick Start

```sh
# Install
curl -L https://github.com/koksalmehmet/mind-palace/releases/latest/download/palace-darwin-arm64 -o palace
chmod +x palace && sudo mv palace /usr/local/bin/

# Initialize
palace init && palace detect && palace scan

# Query
palace ask "where is authentication handled"

# Start dashboard
palace dashboard
```

## Documentation

Full documentation is available at [koksalmehmet.github.io/mind-palace](https://koksalmehmet.github.io/mind-palace).

## Repository Structure

```
mind-palace/
├── apps/                    # All ecosystem applications
│   ├── cli/                 # Palace CLI (Go)
│   │   ├── main.go          # Entry point
│   │   ├── internal/        # Core engine packages
│   │   ├── pkg/             # Public Go API
│   │   ├── schemas/         # JSON schema definitions
│   │   ├── starter/         # Init templates
│   │   └── tests/           # Integration tests
│   ├── dashboard/           # Angular web dashboard
│   ├── docs/                # Next.js + Nextra documentation
│   └── vscode/              # VS Code extension
├── assets/                  # Shared branding assets
├── packaging/               # Installer scripts (DMG, DEB, MSI)
├── scripts/                 # Build & CI scripts
├── CHANGELOG.md
├── LICENSE
├── Makefile
├── README.md
└── VERSION
```

## Development

### Prerequisites

- Go 1.22+
- Node.js 20+
- npm

### Setup

```sh
# Clone the repository
git clone https://github.com/koksalmehmet/mind-palace.git
cd mind-palace

# Install all dependencies
make deps

# Build everything
make build

# Run tests
make test
```

### Development Mode

```sh
# Run Go backend in dev mode
make dev

# Run Angular dashboard dev server (separate terminal)
make dev-dashboard

# Watch VS Code extension (separate terminal)
make dev-vscode
```

### Makefile Targets

```sh
make build          # Build all components
make build-palace   # Build CLI only
make build-dashboard # Build Angular dashboard
make build-vscode   # Build VS Code extension

make test           # Run all tests
make test-go        # Run Go tests only
make test-dashboard # Run Angular tests

make lint           # Run all linters
make clean          # Clean build artifacts
make deps           # Install all dependencies
```

See `make help` for all available targets.

## Installation

### Via Go

```sh
go install github.com/koksalmehmet/mind-palace/apps/cli@latest
```

### Binary Releases

- **macOS (Apple Silicon)**:

  ```sh
  curl -L https://github.com/koksalmehmet/mind-palace/releases/latest/download/palace-darwin-arm64 -o palace
  chmod +x palace && sudo mv palace /usr/local/bin/
  ```

- **macOS (Intel)**:

  ```sh
  curl -L https://github.com/koksalmehmet/mind-palace/releases/latest/download/palace-darwin-amd64 -o palace
  chmod +x palace && sudo mv palace /usr/local/bin/
  ```

- **Linux (amd64)**:

  ```sh
  curl -L https://github.com/koksalmehmet/mind-palace/releases/latest/download/palace-linux-amd64 -o palace
  chmod +x palace && sudo mv palace /usr/local/bin/
  ```

- **Windows**: Download `palace-windows-amd64.exe` from [releases](https://github.com/koksalmehmet/mind-palace/releases).

### VS Code Extension

Download the `.vsix` file from [releases](https://github.com/koksalmehmet/mind-palace/releases) and install:

```sh
code --install-extension mind-palace-vscode-*.vsix
```

### Self-Update

```sh
palace update          # Download and install latest
palace version --check # Check for updates
```

## MCP Integration

Add to your MCP client configuration:

```json
{
  "mcpServers": {
    "mind-palace": {
      "command": "palace",
      "args": ["serve", "--root", "/path/to/your/project"]
    }
  }
}
```

Compatible with Claude Desktop, Cursor, and other MCP-enabled agents.

## Public API

External tools can import Mind Palace packages:

```go
import (
    "github.com/koksalmehmet/mind-palace/apps/cli/pkg/memory"
    "github.com/koksalmehmet/mind-palace/apps/cli/pkg/corridor"
    "github.com/koksalmehmet/mind-palace/apps/cli/pkg/types"
)

// Open workspace memory
mem, _ := memory.Open("/path/to/workspace")
defer mem.Close()

// Start a session
session, _ := mem.StartSession("my-agent", "instance-1", "Implement feature X")

// Log activity
mem.LogActivity(session.ID, memory.Activity{
    Kind:   memory.ActivityFileEdit,
    Target: "main.go",
})

// End session
mem.EndSession(session.ID, memory.SessionCompleted, "Done")
```

See [Public API Documentation](./docs/public-api.md) for details.

## Contributing

We welcome contributions! Please see [Contributing Guide](./docs/contributing.md) for:

- Development setup
- Code style guidelines
- Pull request process
- Testing requirements

## License

PolyForm Shield 1.0.0 — see [LICENSE](LICENSE) for details.
