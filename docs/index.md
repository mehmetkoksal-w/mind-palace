# Mind Palace

Mind Palace is a deterministic CLI for constructing and maintaining a structured “mind palace” over any codebase. It is designed for long-lived projects, schema-first workflows, and agent-assisted refactors where correctness and provenance matter more than heuristics.

## Overview
- **What it is**: A contract-driven CLI that indexes a workspace (Tier-0 SQLite + WAL + FTS5), validates curated JSONC manifests, and produces deterministic context packs and change signals for humans and agents.
- **Problem it solves**: Keeps agents and humans in sync about what exists, what changed, and what is allowed to touch—without probabilistic guessing or ad-hoc scans.
- **When to use it**: Before and during significant refactors, agent-assisted changes, or CI validation to ensure context freshness and guardrail adherence.
- **Non-goals**: No orchestration of agents, no heuristic ranking, no language-specific logic in the core, no YAML.
- **Product overview**: See `./product.md` for positioning, feature set, and future plans.

## Mental Model & Philosophy
- **Determinism over heuristics**: Everything derivable from code is computed (hashes, mtimes, chunks). No best-effort scanning.
- **Schema-first**: Embedded JSON Schemas define contracts; validation always uses embedded copies. Exported schemas are for transparency only.
- **Curated vs generated**: Curated JSONC is committed; generated SQLite/outputs are ignored to keep provenance clear and reproducible.
- **SQLite (WAL + FTS5)**: Stable, portable Tier-0 index with full-text search over chunked content.
- **Agents never scan blindly**: Agents should rely on context packs, change signals, and the index—not re-scan the repo.
- **Ignored outputs**: `.palace/index` and `.palace/outputs` are generated so runs remain reproducible and diffs stay clean.

## Installation
Set once for copy/paste:
```sh
OWNER_REPO="koksalmehmet/mind-palace"
```

### Latest release (no Go toolchain required)
- macOS arm64:
  ```sh
  curl -L https://github.com/${OWNER_REPO}/releases/latest/download/palace-darwin-arm64 -o palace
  chmod +x palace && sudo mv palace /usr/local/bin/
  ```
- Linux amd64:
  ```sh
  curl -L https://github.com/${OWNER_REPO}/releases/latest/download/palace-linux-amd64 -o palace
  chmod +x palace && sudo mv palace /usr/local/bin/
  ```
- Windows amd64: download `https://github.com/${OWNER_REPO}/releases/latest/download/palace-windows-amd64.exe` and place on PATH.

### Pinned version (recommended for CI)
Replace `${VERSION}` (e.g., `v0.0.1-rc1`):
- macOS arm64:
  ```sh
  curl -L https://github.com/${OWNER_REPO}/releases/download/${VERSION}/palace-darwin-arm64 -o palace
  chmod +x palace && sudo mv palace /usr/local/bin/
  ```
- Linux amd64:
  ```sh
  curl -L https://github.com/${OWNER_REPO}/releases/download/${VERSION}/palace-linux-amd64 -o palace
  chmod +x palace && sudo mv palace /usr/local/bin/
  ```
- Windows amd64: download `https://github.com/${OWNER_REPO}/releases/download/${VERSION}/palace-windows-amd64.exe`.

### Checksums
Each release includes `SHA256SUMS`.
- macOS: `shasum -a 256 palace`
- Linux: `sha256sum palace`
- Or verify all: `sha256sum --check SHA256SUMS`

### From source
Requires Go 1.25+ and Git:
```sh
go run ./cmd/palace init
go run ./cmd/palace detect
go run ./cmd/palace scan
```
See `./workflows.md` for a fuller daily loop.

## Core Concepts
See `./concepts.md` for a deep dive into Palace, Rooms, Playbooks, Project Profile, Index (Tier-0), Context Pack, Change Signal, Guardrails, and curated vs generated artifacts.

## End-to-End Workflow
A realistic daily loop is documented in `./workflows.md` (init → detect → scan → plan → collect → agent work → signal/verify).

## CLI Reference
Detailed command behavior, inputs, outputs, and side effects live in `./cli.md`.

## Agent Integration
How CLI/IDE agents should consume artifacts, respect guardrails, and react to verification results is covered in `./agents.md`.

## Collaboration & CI
Commit/ignore rules, team workflows, and CI guidance are in `./collaboration.md`.

## Extensibility & Versioning
How to add rooms/playbooks and evolve schemas safely is in `./extensibility.md`.

## Documentation & GitHub Pages
Docs live under `/docs` in Markdown. Recommended GitHub Pages configuration is described in `./pages.md`. Use `/docs` as the Pages source and trigger deployments on `master` when docs or `README.md` change.
