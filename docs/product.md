---
layout: default
title: Product
nav_order: 12
---

# Product Overview

Mind Palace is the **contract layer** between a codebase, humans, and AI agents.

---

## Components

| Component | Role | Interface |
|-----------|------|-----------|
| **CLI** | Deterministic engine | Terminal, CI |
| **Butler** | Intent-based search | `palace ask`, `palace serve` |
| **Observer** | Visual HUD | VS Code extension |

The CLI is the engine. The extension is the interface. Both respect the same contracts.

---

## Problems Solved

### Context Drift

Without structure, agents operate on stale assumptions and humans forget implicit rules.

**Solution**: Verifiable freshness via `palace verify`.

### Unbounded Agent Scope

Agents either rescan everything or operate on incomplete context.

**Solution**: Explicit full/diff scope. No silent widening.

### Missing Provenance

Typical context has no answer to "what produced this?"

**Solution**: Embedded scan identity, timestamps, and hashes.

### Non-Deterministic Tooling

Heuristic tools can't be trusted in CI.

**Solution**: Deterministic, schema-validated, reproducible.

---

## Capabilities

| Capability | Description |
|------------|-------------|
| **Curated Model** | Rooms, Playbooks, guardrails under `.palace/` |
| **Tier-0 Index** | SQLite + FTS5, deterministic chunking, SHA-256 hashes |
| **Context Packs** | Machine-readable context with provenance |
| **Diff-Scoped Workflows** | Constrain to changed files only |
| **Staleness Detection** | Fast/strict verification modes |
| **Butler Search** | BM25 + entry point boosting, grouped by Room |
| **MCP Server** | JSON-RPC 2.0 for AI agents |
| **Corridors** | Multi-repo context sharing |
| **Observer** | VS Code HUD with auto-heal |

---

## What Mind Palace Is Not

| Not This | Because |
|----------|---------|
| Agent orchestrator | Provides context, not decisions |
| Task runner | Declarative, not imperative |
| Language analyzer | Language-agnostic indexing |
| Test/lint replacement | Complements, doesn't replace |
| Heuristic/probabilistic | Deterministic by design |

---

## Design Principles

1. **Determinism beats convenience** - Same input = same output
2. **Schemas are contracts** - Validation, not suggestions
3. **Scope must be explicit** - No silent widening
4. **Generated state is reproducible** - Delete and rebuild anytime
5. **Agents should never guess** - Contracts over heuristics

---

## Best For

- Agent-assisted refactors
- Long-lived, evolving repositories
- CI environments needing deterministic gating
- Teams wanting shared, enforceable rules

## Less Useful For

- One-off scripts
- Small throwaway repos
- Projects without agents or automation

---

## Future Directions

| Feature | Purpose |
|---------|---------|
| Sessions | Multi-step agent workflows |
| Query commands | Direct index inspection |
| Richer planning | Structured steps from Playbooks |
| CI templates | Drop-in verification pipelines |
| Versioned packs | Long-term stability |
