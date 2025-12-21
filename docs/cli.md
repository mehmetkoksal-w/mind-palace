---
layout: default
title: CLI
nav_order: 5
---

# CLI Reference

All commands are deterministic. Validation uses embedded schemas.

---

## Quick Reference

| Command | Purpose | Modifies State |
|---------|---------|----------------|
| `init` | Create .palace/ structure | Yes |
| `detect` | Auto-detect project profile | Yes |
| `scan` | Build/refresh Index | Yes |
| `collect` | Assemble context pack | Yes |
| `verify` | Check freshness | No |
| `signal` | Generate change signal | Yes |
| `ask` | Query Butler | No |
| `serve` | Start MCP server | No |
| `lint` | Validate configs | No |

---

## init

Create `.palace/` scaffolding.

```sh
palace init [--with-outputs] [--force]
```

**Creates**:
- `palace.jsonc` - Root config
- `rooms/` - Room templates
- `playbooks/` - Playbook templates
- `schemas/` - Exported schemas (read-only copies)

Does not scan or index.

---

## detect

Generate project profile from workspace analysis.

```sh
palace detect
```

**Creates**: `.palace/project-profile.json`

Detects language, framework, and structure hints.

---

## scan

Build the Index (Tier-0).

```sh
palace scan
```

**Creates**:
- `.palace/index/palace.db` - SQLite + FTS5
- `.palace/index/scan.json` - Scan metadata

Respects guardrails. Deterministic output.

---

## collect

Assemble context pack from Index + Rooms.

```sh
palace collect [--diff <range>] [--goal "<text>"] [--allow-stale]
```

| Flag | Effect |
|------|--------|
| `--diff` | Scope to changed files only |
| `--goal` | Set context pack goal |
| `--allow-stale` | Skip freshness check |

**Creates**: `.palace/outputs/context-pack.json`

With Corridors configured, fetches and merges neighbor context.

---

## verify

Check Index freshness.

```sh
palace verify [--fast|--strict] [--diff <range>]
```

| Mode | Behavior |
|------|----------|
| `--fast` (default) | mtime/size check, hash only suspects |
| `--strict` | Hash all indexed files |

**Exit codes**:
- `0` - Fresh
- `1` - Stale
- `2` - Error

---

## signal

Generate change signal from git diff.

```sh
palace signal --diff <range>
```

**Creates**: `.palace/outputs/change-signal.json`

Contains sorted file list with hashes. Enables reproducible diff-scoped workflows.

---

## ask

Query the Index via Butler.

```sh
palace ask "<query>" [--room <name>] [--limit <n>]
```

**Examples**:
```sh
palace ask "where is authentication"
palace ask "AuthService"
palace ask --room api "rate limiting"
```

**Ranking**: BM25 + entry point boost (3x) + path match boost (2.5x)

Results grouped by Room.

---

## serve

Start MCP server for AI agents.

```sh
palace serve [--root <path>]
```

JSON-RPC 2.0 over stdio.

**Tools**:
- `search_mind_palace` - Query Index
- `list_rooms` - List Rooms

**Resources**:
- `palace://files/{path}` - Read file
- `palace://rooms/{name}` - Read Room manifest

---

## lint

Validate curated configs.

```sh
palace lint
```

Checks `palace.jsonc`, `rooms/*.jsonc`, `playbooks/*.jsonc` against embedded schemas.

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Stale / Verification failed |
| 2 | Config or schema error |
| 3 | File system error |
