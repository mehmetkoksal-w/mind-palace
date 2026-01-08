# Mind Palace Observer

<!-- TODO: Add logo/icon here -->

VS Code extension for the [Mind Palace](https://github.com/koksalmehmet/mind-palace) ecosystem.

## Features

- **Traffic Light HUD**: Status bar indicator showing index freshness (Fresh/Stale/Scanning)
- **Blueprint Sidebar**: Interactive visualization of Rooms and files (tree & graph views)
- **Auto-Healing**: Automatic scan & collect on file save
- **Butler Search**: Search your codebase by intent directly from VS Code

## Requirements

This extension requires the Mind Palace CLI. Install it first:

```sh
# macOS
curl -L https://github.com/koksalmehmet/mind-palace/releases/latest/download/palace-darwin-arm64 -o palace
chmod +x palace && sudo mv palace /usr/local/bin/

# Linux
curl -L https://github.com/koksalmehmet/mind-palace/releases/latest/download/palace-linux-amd64 -o palace
chmod +x palace && sudo mv palace /usr/local/bin/
```

## Quick Start

1. Install this extension
2. Open a project with Mind Palace initialized (`palace init`)
3. The status bar shows sync status automatically
4. Open the Blueprint sidebar to explore your project structure

## Configuration

### VS Code Settings

| Setting                            | Default  | Description              |
| ---------------------------------- | -------- | ------------------------ |
| `mindPalace.binaryPath`            | `palace` | Path to CLI binary       |
| `mindPalace.autoSync`              | `true`   | Auto-heal on file save   |
| `mindPalace.autoSyncDelay`         | `3000`   | Debounce delay (ms)      |
| `mindPalace.waitForCleanWorkspace` | `true`   | Wait for all files saved |

### Project Configuration

Settings can also be specified in `.palace/palace.jsonc`:

```jsonc
{
  "vscode": {
    "autoSync": true,
    "autoSyncDelay": 5000,
    "decorations": { "enabled": true },
    "sidebar": { "defaultView": "graph" }
  }
}
```

Project config takes precedence over VS Code settings.

## Documentation

Full documentation lives in the CLI repository:

- [Ecosystem Overview](https://github.com/koksalmehmet/mind-palace/blob/main/docs/ecosystem.md)
- [Extension Guide](https://github.com/koksalmehmet/mind-palace/blob/main/docs/extension.md)
- [Compatibility Matrix](https://github.com/koksalmehmet/mind-palace/blob/main/docs/COMPATIBILITY.md)
- [Workflows](https://github.com/koksalmehmet/mind-palace/blob/main/docs/workflows.md)

## Version Compatibility

| CLI Version | Extension Version |
| ----------- | ----------------- |
| 0.0.1-alpha | 0.0.1-alpha       |

The extension checks CLI version on startup and warns if incompatible.

## Related Projects

- [Mind Palace CLI](https://github.com/koksalmehmet/mind-palace) - Core engine

## License

MIT License — see [LICENSE](LICENSE) for details.
