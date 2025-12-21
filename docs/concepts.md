---
layout: default
title: Concepts
nav_order: 3
---

# Concepts

Mind Palace has two types of artifacts: **curated** (you write, commit to git) and **generated** (CLI creates, gitignored).

```
.palace/
├── palace.jsonc          # Curated: project config
├── rooms/*.jsonc         # Curated: room definitions
├── playbooks/*.jsonc     # Curated: workflow templates
├── index/
│   ├── palace.db         # Generated: SQLite + FTS5 index
│   └── scan.json         # Generated: scan metadata
└── outputs/
    ├── context-pack.json # Generated: assembled context
    └── change-signal.json# Generated: diff information
```

---

## Curated Artifacts

### Palace Config

`.palace/palace.jsonc` - The root manifest.

```jsonc
{
  "schemaVersion": "1.0.0",
  "kind": "palace/config",
  "project": { "name": "my-app" },
  "defaultRoom": "core",
  "guardrails": {
    "doNotTouchGlobs": ["*.secret", "vendor/**"]
  },
  "neighbors": { /* corridors to other repos */ },
  "vscode": { /* extension settings */ }
}
```

### Rooms

`.palace/rooms/*.jsonc` - Logical groupings answering "where should I look?"

```jsonc
// rooms/auth.jsonc
{
  "schemaVersion": "1.0.0",
  "kind": "palace/room",
  "name": "auth",
  "purpose": "Authentication and session management",
  "entryPoints": ["src/auth/login.ts", "src/middleware/auth.ts"],
  "includeGlobs": ["src/auth/**", "src/middleware/auth*"]
}
```

### Playbooks

`.palace/playbooks/*.jsonc` - Workflow contracts answering "how do I execute this change?"

```jsonc
// playbooks/add-endpoint.jsonc
{
  "schemaVersion": "1.0.0",
  "kind": "palace/playbook",
  "name": "add-endpoint",
  "rooms": ["api", "validation"],
  "steps": [
    "Define route in src/routes/",
    "Add validation schema",
    "Write handler",
    "Add tests"
  ]
}
```

---

## Generated Artifacts

### Index (Tier-0)

`palace.db` - SQLite database with:
- **WAL mode** - Concurrent reads during writes
- **FTS5** - Full-text search with BM25 ranking
- **File hashes** - SHA-256 for staleness detection

Created by `palace scan`. This is the source of truth.

### Context Pack

`context-pack.json` - Portable context for agents.

```json
{
  "goal": "Fix auth bug",
  "scanId": "abc-123",
  "files": [
    {"path": "src/auth/login.ts", "hash": "sha256:...", "snippet": "..."}
  ],
  "rooms": ["auth"],
  "provenance": { "createdAt": "2025-01-15T10:00:00Z" }
}
```

Created by `palace collect`. Schema-validated.

### Change Signal

`change-signal.json` - Diff metadata for scoped workflows.

Created by `palace signal --diff HEAD~1..HEAD`. Enables CI to verify only changed files.

---

## Runtime Components

### Butler

The search engine. Uses FTS5 with BM25 ranking + entry-point boosting.

```sh
palace ask "where is authentication"
palace ask --room api "rate limiting"
```

### MCP Server

JSON-RPC 2.0 server for AI agents.

```sh
palace serve
```

Exposes:
- `search_mind_palace` - Query the index
- `list_rooms` - Enumerate rooms
- `palace://files/{path}` - Read files
- `palace://rooms/{name}` - Read room manifests

### Observer

VS Code extension. Automates verify/scan/collect on file save.

---

## Data Flow

```
┌──────────────┐   scan    ┌──────────────┐  collect  ┌──────────────┐
│ Source Files │ ────────► │  palace.db   │ ────────► │ context-pack │
│   + Rooms    │           │   (Index)    │           │    .json     │
└──────────────┘           └──────────────┘           └──────────────┘
                                  │
                                  │ ask / serve
                                  ▼
                           ┌──────────────┐
                           │   Results    │
                           │ (ranked,     │
                           │  grouped)    │
                           └──────────────┘
```

---

## Verification Model

```
palace verify [--fast|--strict]

--fast (default):
  1. Check mtime/size of files
  2. Hash only candidates that changed
  3. Compare against stored hashes

--strict:
  1. Hash all indexed files
  2. Compare against stored hashes
  3. Fail if any mismatch

Exit codes:
  0 = Fresh (index matches filesystem)
  1 = Stale (files changed)
  2 = Error (config/schema issues)
```

---

## Guardrails

Patterns the CLI will never touch:

```jsonc
{
  "guardrails": {
    "doNotTouchGlobs": [".git/**", "node_modules/**", ".env*"],
    "readOnlyGlobs": ["package-lock.json"]
  }
}
```

Defaults are always applied. Your patterns extend, never replace.

---

## Corridors

Import context from other Mind Palace repositories.

```jsonc
{
  "neighbors": {
    "backend": {
      "url": "https://storage.example.com/backend/context-pack.json",
      "ttl": "24h"
    },
    "shared-lib": {
      "localPath": "../shared-lib"  // monorepo
    }
  }
}
```

Remote files are namespaced: `corridor://backend/src/api.ts`
