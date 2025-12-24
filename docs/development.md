---
layout: default
title: Development
nav_order: 15
---

# Development Guide

This guide covers setting up the development environment and working with the Mind Palace monorepo.

## Prerequisites

- **Go 1.22+** - [Download](https://go.dev/dl/)
- **Node.js 20+** - [Download](https://nodejs.org/)
- **npm** - Comes with Node.js
- **Git** - [Download](https://git-scm.com/)

Optional:
- **golangci-lint** - For Go linting: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
- **vsce** - For VS Code extension packaging: `npm install -g @vscode/vsce`

## Repository Structure

```
mind-palace/
├── cmd/palace/              # CLI entry point
├── internal/                # Private Go packages
│   ├── analysis/            # Language parsers (Go, TS, Python, etc.)
│   ├── butler/              # MCP server & coordinator
│   ├── cli/                 # CLI command implementations
│   ├── config/              # Configuration management
│   ├── corridor/            # Cross-workspace sharing
│   ├── dashboard/           # HTTP server + embedded assets
│   ├── index/               # Code index & oracle
│   └── memory/              # Session memory
├── pkg/                     # Public Go packages (importable by external tools)
│   ├── types/               # Shared type definitions
│   ├── memory/              # Session memory public API
│   └── corridor/            # Corridor public API
├── apps/
│   ├── dashboard/           # Angular 17+ web dashboard
│   ├── vscode/              # VS Code extension
│   └── palace/              # CLI build wrapper (Makefile)
├── schemas/                 # JSON schemas for validation
├── docs/                    # Documentation (Jekyll)
├── scripts/                 # Build & test scripts
├── .github/workflows/       # CI/CD workflows
├── Makefile                 # Root build orchestration
├── go.mod                   # Go module definition
└── VERSION                  # Version file for releases
```

## Initial Setup

```sh
# Clone the repository
git clone https://github.com/koksalmehmet/mind-palace.git
cd mind-palace

# Install all dependencies
make deps

# Verify the build
make build

# Run tests
make test
```

## Development Workflows

### Working on the Go Backend

```sh
# Build the CLI
make build-palace

# Run in dev mode (hot reloading not supported, rebuild required)
make dev

# Run specific commands during development
go run ./cmd/palace <command>

# Run Go tests
make test-go

# Run Go linter
make lint-go
```

### Working on the Dashboard

```sh
# Install dashboard dependencies (if not done via make deps)
cd apps/dashboard && npm install

# Start the dev server with hot reload
make dev-dashboard
# Or directly: cd apps/dashboard && npm start

# The dashboard runs at http://localhost:4200
# It proxies API requests to the palace backend (must be running)

# Run dashboard tests
make test-dashboard

# Build for production
make build-dashboard
```

### Working on the VS Code Extension

```sh
# Install dependencies
cd apps/vscode && npm install

# Compile the extension
make build-vscode
# Or directly: cd apps/vscode && npm run compile

# Watch for changes
make dev-vscode
# Or directly: cd apps/vscode && npm run watch

# Package as .vsix
cd apps/vscode && npx vsce package

# To test in VS Code:
# 1. Open apps/vscode in VS Code
# 2. Press F5 to launch Extension Development Host
```

## Build Commands

| Command | Description |
|---------|-------------|
| `make build` | Build all components |
| `make build-palace` | Build CLI binary only |
| `make build-dashboard` | Build dashboard and embed assets |
| `make build-vscode` | Compile VS Code extension |
| `make release` | Build optimized release binary with embedded dashboard |

## Test Commands

| Command | Description |
|---------|-------------|
| `make test` | Run all tests |
| `make test-go` | Run Go tests with race detection |
| `make test-dashboard` | Run Angular tests (headless) |
| `make test-vscode` | Run VS Code extension tests |
| `make e2e` | Run end-to-end tests |

## Lint Commands

| Command | Description |
|---------|-------------|
| `make lint` | Run all linters |
| `make lint-go` | Run golangci-lint |
| `make lint-dashboard` | Run Angular lint |
| `make lint-vscode` | Run VS Code extension lint |

## Other Commands

| Command | Description |
|---------|-------------|
| `make deps` | Install all dependencies |
| `make deps-go` | Download Go modules |
| `make deps-dashboard` | Install dashboard npm packages |
| `make deps-vscode` | Install VS Code extension npm packages |
| `make clean` | Remove build artifacts |
| `make clean-all` | Remove build artifacts and node_modules |
| `make info` | Show project info (version, commit, directories) |
| `make verify` | Build and verify the binary works |
| `make help` | Show all available targets |

## Running the Full Stack

For full development, you need three terminals:

```sh
# Terminal 1: Go backend
make dev

# Terminal 2: Dashboard dev server
make dev-dashboard

# Terminal 3: VS Code extension (if working on it)
make dev-vscode
```

The dashboard at `http://localhost:4200` will proxy API requests to the backend at `http://localhost:3001`.

## Database Files

Mind Palace uses SQLite databases:

| Database | Location | Purpose |
|----------|----------|---------|
| `palace.db` | `.palace/index/` | Code index (symbols, files, calls) |
| `memory.db` | `.palace/` | Session memory (sessions, activities, learnings) |
| `personal.db` | `~/.palace/corridors/` | Personal corridor (cross-project learnings) |

To reset:
```sh
rm -rf .palace/index/palace.db  # Reset code index
rm -rf .palace/memory.db        # Reset session memory
rm -rf ~/.palace/corridors/     # Reset personal corridor
```

## Adding a New Language Parser

1. Create a new file in `internal/analysis/`:
   ```go
   // internal/analysis/ruby.go
   package analysis

   type RubyAnalyzer struct{}

   func (a *RubyAnalyzer) Analyze(path string, content []byte) (*FileAnalysis, error) {
       // Parse Ruby code and extract symbols
   }
   ```

2. Register in the analyzer factory:
   ```go
   // internal/analysis/factory.go
   func GetAnalyzer(lang string) Analyzer {
       switch lang {
       case "ruby":
           return &RubyAnalyzer{}
       // ...
       }
   }
   ```

3. Add language detection in scanner.

## Adding MCP Tools

1. Define the tool schema in `internal/butler/mcp.go`:
   ```go
   {
       Name: "my_tool",
       Description: "Does something useful",
       InputSchema: map[string]any{
           "type": "object",
           "properties": map[string]any{
               "param": map[string]any{"type": "string"},
           },
       },
   }
   ```

2. Implement the handler:
   ```go
   func (b *Butler) handleMyTool(params map[string]any) (any, error) {
       // Implementation
   }
   ```

3. Register in the tool dispatcher.

## Adding Dashboard Features

1. Generate a new component:
   ```sh
   cd apps/dashboard
   ng generate component features/my-feature --standalone
   ```

2. Add route in `src/app/app.routes.ts`:
   ```typescript
   { path: 'my-feature', component: MyFeatureComponent }
   ```

3. Add navigation in `src/app/app.component.ts`.

4. Implement API calls via `ApiService`.

## Debugging

### Go Backend

```sh
# Run with verbose logging
go run ./cmd/palace --debug <command>

# Use delve for debugging
dlv debug ./cmd/palace -- <command>
```

### Dashboard

Use Chrome DevTools (F12) or VS Code's debugger with the Angular debug configuration.

### VS Code Extension

1. Open `apps/vscode` in VS Code
2. Set breakpoints
3. Press F5 to launch Extension Development Host
4. Trigger extension functionality

## Environment Variables

| Variable | Description |
|----------|-------------|
| `PALACE_DEBUG` | Enable debug logging |
| `PALACE_HOME` | Override global palace directory (default: `~/.palace`) |

## Continuous Integration

The project uses GitHub Actions for CI/CD:

- **ci.yml** - Runs on all PRs: tests, linting, build verification
- **pipeline.yml** - Runs on main: full CI + auto-release if VERSION changes
- **release.yml** - Runs on tags: creates GitHub release with all assets

### Release Process

1. Update `VERSION` file with new version (e.g., `0.2.0`)
2. Commit and push to main
3. Pipeline detects new version, creates tag, and releases

Or manually:
```sh
git tag v0.2.0
git push origin v0.2.0
```

## Troubleshooting

### "Dashboard dependencies not installed"
```sh
make deps-dashboard
```

### "golangci-lint not found"
```sh
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### "vsce not found"
```sh
npm install -g @vscode/vsce
```

### Tests failing due to missing Chrome
Install Chrome or use Firefox:
```sh
npm test -- --browsers=Firefox
```
