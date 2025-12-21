---
layout: default
title: Ecosystem
nav_order: 2
---

# Ecosystem

Mind Palace consists of two repositories working together.

| Repository | Language | Purpose |
|------------|----------|---------|
| [mind-palace](https://github.com/koksalmehmet/mind-palace) | Go | CLI + MCP server |
| [mind-palace-vscode](https://github.com/koksalmehmet/mind-palace-vscode) | TypeScript | VS Code extension |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         DEVELOPER                                    │
└─────────────────────────────────────────────────────────────────────┘
         │                    │                         │
         ▼                    ▼                         ▼
┌─────────────────┐  ┌─────────────────┐      ┌─────────────────┐
│    Terminal     │  │     VS Code     │      │    AI Agent     │
│                 │  │   (Observer)    │      │ (Claude, etc.)  │
└────────┬────────┘  └────────┬────────┘      └────────┬────────┘
         │                    │                         │
         │ CLI commands       │ Spawns CLI              │ MCP protocol
         │                    │                         │
         └────────────────────┼─────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      MIND PALACE CLI (Go)                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│  │  scan    │ │ collect  │ │  verify  │ │   ask    │ │  serve   │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        .palace/ DIRECTORY                            │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ CURATED (commit to git)                                     │    │
│  │  palace.jsonc, rooms/*.jsonc, playbooks/*.jsonc             │    │
│  └─────────────────────────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ GENERATED (gitignored)                                      │    │
│  │  index/palace.db, outputs/context-pack.json                 │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

---

## CLI Internals

### Storage: SQLite + FTS5

```
palace.db
├── files        # path, hash, size, mtime
├── chunks       # file_id, content, line_start, line_end
└── chunks_fts   # FTS5 virtual table for full-text search
```

**Why SQLite?**
- Deterministic (same input = same output)
- WAL mode allows concurrent reads
- FTS5 provides BM25 ranking out of the box
- Single file, no server process

### Search: BM25 + Boosting

```
Score = BM25(query, content)
      + entry_point_boost (if file is room entry point)
      + path_match_boost (if path contains query terms)
```

Results grouped by Room, sorted by score within each group.

### Hashing: SHA-256

Every indexed file has a SHA-256 hash. Verification compares stored hashes against current file hashes.

```
Stored: sha256:abc123...
Current: sha256:abc123... → Fresh ✓
Current: sha256:def456... → Stale ✗
```

---

## Extension Internals

### Bridge Pattern

The extension never maintains state. All operations go through the CLI:

```typescript
// bridge.ts
async runVerify(): Promise<boolean> {
  const { exitCode } = await exec('palace verify --fast');
  return exitCode === 0;
}

async runHeal(): Promise<void> {
  await exec('palace scan && palace collect');
}
```

### MCP Client

For search, the extension spawns `palace serve` and communicates via JSON-RPC:

```typescript
// Simplified
const result = await mcpClient.request('tools/call', {
  name: 'search_mind_palace',
  arguments: { query: 'auth' }
});
```

### Configuration Merge

```
.palace/palace.jsonc  →  Highest priority
         ↓
VS Code settings      →  Medium priority
         ↓
Extension defaults    →  Lowest priority
```

---

## Protocol: MCP

[Model Context Protocol](https://modelcontextprotocol.io/) - JSON-RPC 2.0 over stdio.

### Tools

```json
{"method": "tools/call", "params": {"name": "search_mind_palace", "arguments": {"query": "auth"}}}
{"method": "tools/call", "params": {"name": "list_rooms"}}
```

### Resources

```
palace://files/src/auth/login.ts  → File content
palace://rooms/auth               → Room manifest
```

---

## Version Compatibility

| CLI | Extension | Schema |
|-----|-----------|--------|
| 0.0.1-rc1 | 0.0.1 | 1.0.0 |

Extension checks CLI version on startup. Warns if incompatible.

See [COMPATIBILITY.md](./COMPATIBILITY.html) for upgrade guidelines.
