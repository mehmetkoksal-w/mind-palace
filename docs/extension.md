---
layout: default
title: Extension
nav_order: 7
---

# Mind Palace Observer

VS Code extension for the Mind Palace ecosystem.

**Repository**: [mind-palace-vscode](https://github.com/koksalmehmet/mind-palace-vscode)

---

## Install

1. Install CLI first (required):
   ```sh
   curl -L https://github.com/koksalmehmet/mind-palace/releases/latest/download/palace-darwin-arm64 -o palace
   chmod +x palace && sudo mv palace /usr/local/bin/
   ```

2. Install extension from VS Code Marketplace: "Mind Palace Observer"

---

## Features

### Status Bar (HUD)

| State | Color | Meaning |
|-------|-------|---------|
| Fresh | Green | Index matches filesystem |
| Stale | Red | Files changed since last scan |
| Scanning | Amber | Heal in progress |

Click to open action menu.

### Sidebar (Blueprint)

- **Tree view**: Rooms as folders, entry points as files
- **Graph view**: Cytoscape visualization
- **Search**: Query Butler directly

### Auto-Heal

On file save:
1. Wait for debounce (default: 3s)
2. Run `palace scan && palace collect`
3. Update HUD

---

## Configuration

### VS Code Settings

```json
{
  "mindPalace.binaryPath": "palace",
  "mindPalace.autoSync": true,
  "mindPalace.autoSyncDelay": 3000,
  "mindPalace.waitForCleanWorkspace": true
}
```

### Project Config (Higher Priority)

In `.palace/palace.jsonc`:

```jsonc
{
  "vscode": {
    "autoSync": true,
    "autoSyncDelay": 5000,
    "decorations": {
      "enabled": true,
      "style": "gutter"  // "gutter" | "inline" | "both"
    },
    "statusBar": {
      "position": "left",
      "priority": 100
    },
    "sidebar": {
      "defaultView": "tree",  // "tree" | "graph"
      "graphLayout": "cose"   // "cose" | "circle" | "grid"
    }
  }
}
```

Project config overrides VS Code settings.

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│              VS CODE EXTENSION                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │    HUD      │  │   Sidebar   │  │  Decorator  │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘ │
│         └────────────────┼────────────────┘         │
│                    ┌─────▼─────┐                    │
│                    │  Bridge   │                    │
│                    └─────┬─────┘                    │
└──────────────────────────┼──────────────────────────┘
                           │
          ┌────────────────┼────────────────┐
          ▼                ▼                ▼
      palace           palace          palace
      verify            scan            serve
```

**Key principle**: Extension has no state. CLI is source of truth.

---

## Files

| File | Purpose |
|------|---------|
| `extension.ts` | Activation, commands, events |
| `bridge.ts` | CLI invocation, MCP client |
| `config.ts` | Merged config reader |
| `version.ts` | CLI version check |
| `hud.ts` | Status bar component |
| `sidebar.ts` | Webview with graph/tree |
| `decorator.ts` | Code gutters |

---

## Troubleshooting

**"Palace binary not found"**
- Check: `palace --version`
- Set `mindPalace.binaryPath` to absolute path

**HUD stuck on Scanning**
- Check Output panel → "Mind Palace"
- Try `palace scan` manually

**Search returns nothing**
- Verify index exists: `ls .palace/index/palace.db`
- Rebuild: `palace scan`
