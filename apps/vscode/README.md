# Mind Palace

Minimal VS Code extension for the [Mind Palace](https://github.com/mehmetkoksal-w/mind-palace) ecosystem.

## Features

### Status Bar

Shows Mind Palace status at a glance:

| State | Display | Meaning |
|-------|---------|--------|
| Fresh | `✓ Palace 2D/1I/3L` | Index fresh, shows knowledge counts |
| Stale | `⚠ Palace` | Files changed since last scan |
| Not initialized | `ℹ Palace: Not initialized` | Run `palace init` |

Click the status bar item to refresh.

### LSP Integration

Real-time diagnostics powered by the Language Server Protocol:

- Pattern violation warnings with confidence scores
- Contract mismatch errors between frontend and backend
- Hover information showing pattern/contract details
- Code actions to approve, ignore, or verify issues
- Code lens showing issue counts per file

## Requirements

This extension requires the Mind Palace CLI. Install it first:

```sh
# macOS
curl -L https://github.com/mehmetkoksal-w/mind-palace/releases/latest/download/palace-darwin-arm64 -o palace
chmod +x palace && sudo mv palace /usr/local/bin/

# Linux
curl -L https://github.com/mehmetkoksal-w/mind-palace/releases/latest/download/palace-linux-amd64 -o palace
chmod +x palace && sudo mv palace /usr/local/bin/
```

## Quick Start

1. Install this extension
2. Open a project with Mind Palace initialized (`palace init`)
3. The status bar shows index status automatically
4. Use CLI for full functionality (`palace explore`, `palace recall`, etc.)

## Commands

| Command | Description |
|---------|-------------|
| `Mind Palace: Check Status` | Refresh status bar |
| `Mind Palace: Restart LSP Server` | Restart LSP if needed |

## Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `mindPalace.binaryPath` | `palace` | Path to CLI binary |
| `mindPalace.showStatusBarItem` | `true` | Show status bar indicator |
| `mindPalace.lsp.enabled` | `true` | Enable LSP diagnostics |
| `mindPalace.lsp.diagnostics.patterns` | `true` | Show pattern diagnostics |
| `mindPalace.lsp.diagnostics.contracts` | `true` | Show contract diagnostics |
| `mindPalace.lsp.codeLens.enabled` | `true` | Show code lens |

## Philosophy

This extension is intentionally minimal. Mind Palace is designed for **AI agents**, not humans.

- **CLI** is the source of truth for all operations
- **MCP Server** (`palace serve`) provides AI agent integration
- **LSP Server** (`palace lsp`) provides real-time editor diagnostics
- **Extension** provides status visibility and LSP client

For full functionality, use the CLI directly or connect your AI agent via MCP.

## Documentation

- [CLI Reference](https://mind-palace.dev/reference/cli)
- [MCP Server](https://mind-palace.dev/reference/mcp)
- [LSP Server](https://mind-palace.dev/reference/lsp)

## Version Compatibility

| CLI Version | Extension Version |
| ----------- | ----------------- |
| 0.0.1-alpha | 0.0.1-alpha       |

The extension checks CLI version on startup and warns if incompatible.

## Related Projects

- [Mind Palace CLI](https://github.com/mehmetkoksal-w/mind-palace) - Core engine

## License

MIT License — see [LICENSE](LICENSE) for details.
