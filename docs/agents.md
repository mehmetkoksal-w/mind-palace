# Agent Integration Guide

Mind Palace is built for agent-assisted work. Agents must treat schemas and artifacts as contracts, not suggestions.

## What agents should read
- `.palace/palace.jsonc`: project definition, default room, guardrails.
- `.palace/rooms/*.jsonc`: entry points and evidence guidance.
- `.palace/playbooks/*.jsonc`: change procedures and required evidence.
- `.palace/project-profile.json`: language/framework hints.
- `.palace/outputs/context-pack.json`: current goal, scan identity, findings, scope.
- `.palace/outputs/change-signal.json` (if present): authoritative diff.
- `.palace/index/scan.json`: scan identity and counts (generated).

## What agents must not touch
- `.palace/index/*` and `.palace/outputs/*`: generated; never edited directly.
- `.palace/schemas/*`: export-only copies; validation uses embedded schemas.
- Files matching guardrails (`.git/**`, `.palace/**`, `node_modules/**`, etc.).

## How to use context-pack.json
- Treat it as the authoritative working context: goal, scan hash/id/time, scope (full/diff), referenced files, findings.
- If scope is diff, limit actions to the listed files unless a human broadens scope.
- If scan hash is stale, request a fresh `palace scan` + `palace collect`.

## Respecting guardrails
- Always check guardrails before reading/writing. Do not propose edits in guarded paths.
- For path matching, normalize to forward slashes to align with guardrail semantics.

## Reacting to verify failures
- If `palace verify` reports stale files: ask the user to run `palace scan` (or do it if permitted), then `palace collect`, then retry.
- If `palace verify --diff` errors because diff cannot be computed: request a change-signal (`palace signal --diff <range>`) or a valid git diff range.

## Using change-signal
- Prefer an existing `.palace/outputs/change-signal.json` that matches the target diff range.
- If absent, request `palace signal --diff <range>` instead of running your own git diff.
- Use the change-signal paths to scope work; treat deleted files as non-editable.

## CLI vs IDE agents
- **CLI agents (Codex/Claude/Gemini/Cursor CLI)**: run `palace plan/collect/verify/signal` as needed, never mutate generated artifacts directly, and obey guardrails when proposing patches.
- **IDE agents (Cursor/Windsurf/Copilot)**: load curated files + context-pack; avoid auto-saving changes in `.palace/index` or `.palace/outputs`; surface verification results to the user before proceeding.

## Schema contract
- Embedded JSON Schemas are the single source of truth. Assume validation will run in CI.
- Do not invent fields; treat missing optional fields as absent, not null.

## When unsure
- Prefer to ask for `palace explain <topic>` to confirm behavior.
- When diff scope is unclear, request a change-signal or explicit diff range instead of assuming full-scope permission.
