---
layout: default
title: Workflows
nav_order: 4
---

# Workflows

## Setup (Once)

```sh
palace init          # Create .palace/ structure
palace detect        # Auto-detect language/framework
palace scan          # Build initial index
```

## Daily Loop

### Terminal

```sh
# After changes
palace scan          # Rebuild index
palace collect       # Refresh context pack

# Before agent work
palace verify        # Confirm freshness
palace ask "query"   # Find relevant code
```

### VS Code (Automatic)

```
Edit files → Save → Extension runs verify → Auto-heal if stale
```

The Observer extension handles the loop automatically:

1. **Save file** - triggers debounced verify
2. **If stale** - runs `palace scan && palace collect`
3. **HUD updates** - green (fresh) or red (stale)

## Diff-Scoped Workflows

Constrain context to only changed files:

```sh
# Generate change signal
palace signal --diff HEAD~1..HEAD

# Collect only affected context
palace collect --diff HEAD~1..HEAD

# Verify only changed files
palace verify --diff HEAD~1..HEAD
```

Useful for:
- CI pipelines (verify only PR changes)
- Large monorepos (avoid full scans)
- Focused agent sessions

## CI Integration

```yaml
# .github/workflows/verify.yml
- name: Verify Mind Palace
  run: |
    palace verify --strict --diff ${{ github.event.pull_request.base.sha }}..${{ github.sha }}
```

Exit codes:
- `0` - Fresh, proceed
- `1` - Stale, fail CI
- `2` - Config error

## Agent Sessions

### Via MCP (Recommended)

```json
{
  "mcpServers": {
    "mind-palace": {
      "command": "palace",
      "args": ["serve", "--root", "/path/to/project"]
    }
  }
}
```

Agent can then:
1. Call `search_mind_palace` to find code
2. Call `list_rooms` to understand structure
3. Read files via `palace://files/{path}`

### Via Context Pack

```sh
palace collect --goal "Fix the auth bug"
# Hand context-pack.json to agent
```

---

## Command Reference

| Command | Purpose | Key Flags |
|---------|---------|-----------|
| `palace init` | Create .palace/ | `--with-outputs` |
| `palace detect` | Auto-detect project | - |
| `palace scan` | Build/rebuild index | - |
| `palace collect` | Assemble context pack | `--diff`, `--goal`, `--allow-stale` |
| `palace verify` | Check freshness | `--fast`, `--strict`, `--diff` |
| `palace signal` | Generate change signal | `--diff` |
| `palace ask` | Search index | `--room` |
| `palace serve` | Start MCP server | `--root` |
| `palace lint` | Validate configs | - |

---

## VS Code Extension

### HUD States

| State | Meaning | Action |
|-------|---------|--------|
| Green | Index is fresh | None needed |
| Red | Files changed | Auto-heal or manual scan |
| Amber | Scanning | Wait |

### Configuration

Settings in `.palace/palace.jsonc` override VS Code settings:

```jsonc
{
  "vscode": {
    "autoSync": true,
    "autoSyncDelay": 3000,
    "decorations": { "enabled": true },
    "sidebar": { "defaultView": "tree" }
  }
}
```

### Sidebar

- **Tree view** - Rooms as folders, files as entries
- **Graph view** - Cytoscape visualization of connections
- **Search** - Query Butler directly from sidebar
