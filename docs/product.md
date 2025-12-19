# Mind Palace — Product Overview & Feature Set

## What Mind Palace Is

Mind Palace is a deterministic, schema-first CLI tool for constructing, maintaining, and validating a structured mental model over a codebase.

It exists to solve a specific, recurring problem in modern development:

Humans and AI agents need shared, reliable context about a repository — and that context must be correct, reproducible, and scoped.

Mind Palace provides that context as contracts, not guesses.

It does this by:
- Separating curated intent from derived state
- Indexing code deterministically
- Enforcing explicit scope and provenance
- Producing machine-readable artifacts that agents and CI can trust

It is not an agent, an orchestrator, or a heuristic analyzer. It is the ground truth layer agents operate on.

## What Problems It Solves

### 1. Context Drift

Large or long-lived repositories change constantly. Without a structured system:
- Agents operate on stale assumptions
- Humans forget implicit rules and guardrails
- Refactors silently widen scope

Mind Palace makes context freshness explicit and verifiable.

### 2. Unbounded Agent Scope

Most agents either:
- Rescan entire repositories every time, or
- Operate on incomplete prompt context

Mind Palace enforces:
- Full-scope vs diff-scope explicitly
- No silent widening
- Guardrails that apply equally to humans, CI, and agents

### 3. Missing Provenance

Typical “context” has no answer to:
- What scan produced this?
- What files were included?
- What changed since last time?

Mind Palace embeds:
- Scan identity (IDs, hashes, timestamps)
- Scope metadata
- Change provenance (via change-signal)

### 4. Non-deterministic Tooling

Heuristic tools are hard to trust and impossible to gate in CI.

Mind Palace is:
- Deterministic
- Schema-validated
- Reproducible from workspace state

## Core Capabilities (Current Features)

### 1. Curated Project Model

Mind Palace introduces a curated model under `.palace/`:
- `palace.jsonc` — project definition, guardrails, defaults
- `rooms/*.jsonc` — where to look
- `playbooks/*.jsonc` — how to execute certain classes of change
- `project-profile.json` — detected language/framework signals

These files:
- Are human-authored
- Are version-controlled
- Are validated against embedded JSON Schemas

### 2. Deterministic Workspace Index (Tier-0)

`palace scan` builds a deterministic index:
- SQLite (WAL + FTS5)
- Normalized paths
- Deterministic chunking
- Hashes + normalized mtimes
- Validated scan summary

This index is:
- Generated (ignored in git)
- Reproducible
- The single source for derived knowledge
- Generated outputs live under `.palace/index` and `.palace/outputs`; `.palace/maps` is created by layout as a reserved internal directory.

### 3. Context Pack Generation

`palace collect` assembles a context pack:
- Goal and provenance
- Scan identity
- Explicit scope (full / diff)
- Referenced files
- Findings from the index

The context pack is:
- Validated
- Machine-readable
- Designed to be consumed directly by agents

Agents should never invent context outside this pack.

### 4. Diff-Scoped Workflows

Mind Palace supports strict diff-based workflows:
- Git diff ranges
- Explicit change-signal artifacts
- No silent fallback to full scope
- Empty diffs are valid and verified

This enables:
- Deterministic agent runs
- Reproducible CI verification
- Safer refactors

### 5. Staleness Detection & Verification

`palace verify`:
- Validates curated manifests
- Detects stale files vs the index
- Supports fast and strict modes
- Errors instead of widening scope

This allows:
- CI gating
- Human confidence before running agents
- Clear remediation steps

### 6. Guardrails as First-Class Contracts

Guardrails:
- Are defined once
- Apply everywhere (scan, collect, verify, signal)
- Protect both humans and agents
- Cannot be overridden by agents

### 7. Agent-Friendly by Design

Mind Palace is designed to be used by:
- CLI agents (Codex, Claude, Gemini, Cursor CLI)
- IDE agents (Cursor, Windsurf, Copilot)

Agents are expected to:
- Read curated manifests
- Consume context packs
- Respect scope and guardrails
- React to verification failures instead of guessing

## What Mind Palace Is Not

Explicit non-goals:
- ❌ Not an agent orchestrator
- ❌ Not a task runner
- ❌ Not a language-specific analyzer
- ❌ Not a replacement for tests or linters
- ❌ Not heuristic or probabilistic

It provides structure and truth, not decisions.

## Intended Usage Patterns

Mind Palace is designed for:
- Agent-assisted refactors
- Long-lived, evolving repositories
- CI environments that need deterministic gating
- Teams that want shared, enforceable project rules

It is less useful for:
- One-off scripts
- Small throwaway repos
- Projects without agents or automation

## Planned / Future Features (Intentional, Not Promises)

The following are deliberate next steps, not speculative ideas.

### 1. Short-Term / Session Memory (planned)
- Non-curated, non-committed session artifacts
- Agent iteration logs
- Intermediate reasoning snapshots
- Would live under `.palace/sessions/` (not created today)

Purpose: support multi-step agent workflows without polluting curated state.

### 2. Query & Inspection Commands
- `palace query "<fts query>"`
- `palace show <file|chunk>`

Purpose: allow humans to inspect the index directly and debug context selection.

### 3. Richer Planning Output
- Structured plan steps derived from playbooks
- Explicit expected evidence per step
- Deterministic plan artifacts

Purpose: make plan more than goal-setting without becoming orchestration.

### 4. CI-First Enhancements
- Built-in CI templates
- Scan/index caching strategies
- Deterministic verification pipelines

Purpose: make Mind Palace a drop-in CI primitive.

### 5. Versioned Context Packs
- Explicit compatibility guarantees
- Tooling to validate older packs
- Clear migration paths

Purpose: long-term stability for agent tooling.

## Design Philosophy (Summary)

Mind Palace is built on a few hard rules:
- Determinism beats convenience
- Schemas are contracts
- Scope must be explicit
- Generated state must be reproducible
- Agents should never guess

Everything in the tool flows from these principles.

## In One Sentence

Mind Palace is the contract layer between a codebase, humans, and AI agents — ensuring everyone operates on the same, verifiable understanding of reality.
