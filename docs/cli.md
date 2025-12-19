# CLI Command Reference

Each command is deterministic and schema-driven. Validation always uses embedded schemas.

## init
- **Purpose**: Create `.palace/` curated scaffolding and export embedded schemas.
- **Inputs**: None (reads workspace root).
- **Outputs**: `.palace/palace.jsonc`, default room/playbook templates, `.palace/project-profile.json` template, `.palace/schemas/*` (export-only). Optional `.palace/outputs/context-pack.json` when `--with-outputs`.
- **Side effects**: Creates directories. Does not scan or index.
- **When to use**: First run in a repo. Re-runnable with `--force` to refresh templates.

## detect
- **Purpose**: Generate `.palace/project-profile.json`.
- **Inputs**: Workspace files (lightweight heuristics only).
- **Outputs**: JSON project profile (curated; commit it).
- **Side effects**: None beyond writing the profile.
- **When to use**: After init, or when project structure changes significantly.

## scan
- **Purpose**: Build/refresh Tier-0 index.
- **Inputs**: Workspace files outside guardrails.
- **Outputs**: `.palace/index/palace.db` (SQLite WAL + FTS5) and `.palace/index/scan.json` (validated scan summary with scanId/dbScanId/counts/hash).
- **Side effects**: Replaces existing index content deterministically.
- **When to use**: After code changes to refresh the index; before collect/verify if stale.

## lint
- **Purpose**: Validate curated artifacts only.
- **Inputs**: `.palace/palace.jsonc`, `.palace/rooms/*.jsonc`, `.palace/playbooks/*.jsonc`, `.palace/project-profile.json`.
- **Outputs**: None (validation errors on failure).
- **Side effects**: None.
- **When to use**: Before verify/collect, or in CI to gate curated edits.

## plan
- **Purpose**: Set or update the context-pack goal and provenance.
- **Inputs**: Goal string; optionally existing context-pack to preserve fields.
- **Outputs**: `.palace/outputs/context-pack.json` (goal/provenance updated).
- **Side effects**: Writes context-pack; does not scan or collect evidence.
- **When to use**: To declare intent before collect/agent work.

## collect
- **Purpose**: Assemble `.palace/outputs/context-pack.json` from the existing index + curated manifests.
- **Inputs**: `.palace/index/palace.db`, curated manifests, optional `--diff <range>`.
- **Outputs**: Updated context-pack (validated).
- **Side effects**: None beyond writing context-pack.
- **When to use**:
  - Full scope: after scan, to refresh working context (fails if index is stale unless `--allow-stale`).
  - Diff scope: to focus on changed files via git diff or matching change-signal (no scope widening).

## verify
- **Purpose**: Validate curated state (lint) and check staleness vs the index.
- **Inputs**: `.palace/index/palace.db`, curated manifests, optional `--diff <range>`.
- **Outputs**: None on success; lists stale items on failure.
- **Side effects**: None; does not modify index.
- **Modes**:
  - `--fast` (default): mtime/size shortcut; selective hashing for mismatches.
  - `--strict`: hash all candidates.
- **Diff strictness**: If diff cannot be computed, verify errors (no fallback); empty diffs verify zero candidates.
- **When to use**: Before CI gating, before trusting the context, after edits.

## signal
- **Purpose**: Generate `.palace/outputs/change-signal.json` from git diff output.
- **Inputs**: `--diff <range>`; reads git repository; respects guardrails on chosen paths.
- **Outputs**: Validated change-signal with sorted changes and hashes for non-deleted files.
- **Side effects**: None beyond writing change-signal.
- **When to use**: For deterministic diff-scoped workflows (agents/CI), or to feed verify/collect diff scope without re-running git.

## explain
- **Purpose**: Print short behavioral summaries for scan/collect/verify/signal/artifacts.
- **Inputs/Outputs**: Stdout text only.
- **Side effects**: None.
- **When to use**: Quick reference to invariants and artifacts.
