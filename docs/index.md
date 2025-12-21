---
layout: home
title: Home
nav_order: 1
---

# Mind Palace

A deterministic context system for codebases, inspired by the [Method of Loci](https://en.wikipedia.org/wiki/Method_of_loci).

[Get Started](./workflows.html){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 } [Concepts](./concepts.html){: .btn .fs-5 .mb-4 .mb-md-0 }

---

## The Problem

AI agents working with codebases face three failures:

1. **Context rot** - Agents receive stale snapshots that diverge from reality
2. **Scope drift** - Without boundaries, agents wander into irrelevant code
3. **Heuristic fragility** - File-matching guesses break as codebases evolve

Current solutions (RAG, embeddings, "just dump everything") are probabilistic. They work until they don't.

## The Solution

Mind Palace provides a **deterministic, schema-validated index** that both humans and AI agents can trust.

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   You curate    │     │   CLI indexes   │     │  Agents query   │
│   Rooms +       │ ──► │   with hashes   │ ──► │  with contracts │
│   Playbooks     │     │   (SQLite+FTS5) │     │  (MCP protocol) │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

**No embeddings. No guessing. Deterministic.**

## Method of Loci

The [Method of Loci](https://en.wikipedia.org/wiki/Method_of_loci) (memory palace technique) is a mnemonic strategy from ancient Greece. You mentally place information in specific locations within an imagined building, then "walk through" to recall it.

Mind Palace applies this to code:

| Memory Palace | Mind Palace |
|---------------|-------------|
| Imagined building | Your codebase |
| Rooms | Logical groupings (auth, api, cli) |
| Objects in rooms | Entry points, key files |
| Walking through | Querying with Butler |
| Recalling | Assembling context packs |

The metaphor isn't decorative—it's the architecture.

## Key Properties

| Property | How |
|----------|-----|
| **Deterministic** | SHA-256 hashes, SQLite with WAL, reproducible scans |
| **Schema-first** | JSON Schema validation on all artifacts |
| **Verifiable** | `palace verify` proves freshness |
| **Agent-ready** | MCP server for direct AI integration |
| **Diff-scoped** | Constrain context to changed files only |

## Quick Start

```sh
# Install
curl -L https://github.com/koksalmehmet/mind-palace/releases/latest/download/palace-darwin-arm64 -o palace
chmod +x palace && sudo mv palace /usr/local/bin/

# Initialize
palace init && palace detect && palace scan

# Query
palace ask "where is authentication handled"
```

## Components

| Component | Purpose |
|-----------|---------|
| [CLI](./cli.html) | Core engine: `palace scan`, `collect`, `verify`, `ask`, `serve` |
| [Observer](./extension.html) | VS Code extension: HUD, sidebar, auto-sync |
| [MCP Server](./agents.html) | AI agent integration via JSON-RPC |

---

## Next

- [Concepts](./concepts.html) - Core terminology
- [Workflows](./workflows.html) - Day-to-day usage
- [Ecosystem](./ecosystem.html) - Architecture details
