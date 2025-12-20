---
layout: home
title: Home
nav_order: 1
---

# Mind Palace
{: .fs-9 }

The contract layer between your codebase and AI agents.
{: .fs-6 .fw-300 }

[Get Started](./workflows.html){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 } [View on GitHub](https://github.com/mehmetkoksal-w/mind-palace){: .btn .fs-5 .mb-4 .mb-md-0 }

---

## Why Mind Palace?

Agents drift. Context rots. Heuristics fail.
Mind Palace provides a **deterministic, schema-first index** (Tier-0) that serves as the single source of truth for both humans and AI.

### Key Features

| **Deterministic** | **Schema-First** | **Agent-Ready** |
|:--- |:--- |:--- |
| SQLite (WAL + FTS5) index built from file hashes. No guessing. | JSONC manifests for rooms and playbooks are validated against strict schemas. | Context packs provide machine-readable, provenance-tracked context. |

---

## Quick Start

### 1. Install
```sh
# macOS
curl -L https://github.com/mehmetkoksal-w/mind-palace/releases/latest/download/palace-darwin-arm64 -o palace
chmod +x palace && sudo mv palace /usr/local/bin/
```

### 2. Initialize
```sh
palace init
palace detect
palace scan
```

### 3. Collect Context
```sh
palace collect --diff HEAD~1..HEAD
```
