# Git & Collaboration Model

Mind Palace separates curated (committed) state from generated (ignored) artifacts to keep provenance clean and runs reproducible.

## Commit vs Ignore
- **Commit**: `.palace/palace.jsonc`, `.palace/rooms/*.jsonc`, `.palace/playbooks/*.jsonc`, `.palace/project-profile.json`, `.palace/schemas/*`, documentation under `/docs`.
- **Ignore**: `.palace/index/*`, `.palace/outputs/*`, `.palace/maps/*`, `*.db`. A future `.palace/sessions/` may appear but is not created today.
- Rationale: Generated artifacts depend on the current workspace; keeping them out of VCS avoids noisy diffs and stale trust.

## Team workflow
- Curated changes (rooms, playbooks, palace.jsonc) should go through code review with `palace lint` in CI.
- Developers run `palace scan` locally after edits to refresh the index, then `palace collect` to update the context pack, followed by `palace verify` to ensure freshness.
- Diff-scoped work should include a `palace signal --diff <range>` artifact for reproducible verification and agent runs.

## CI usage
- Recommended CI gates:
  - `palace lint` (curated validation)
  - `palace verify --strict` (or `--fast` when acceptable) after ensuring an index exists; in diff-based pipelines, supply a change-signal or git diff range.
- If the CI environment lacks a prebuilt index, run `palace scan` before `verify` or cache `.palace/index/` between runs.
- Fail CI when verification reports staleness or when diff scope cannot be resolved.

## Collaboration tips
- Treat `.palace/schemas` as export-only; never edit them directly—changes must go through embedded schemas.
- When rebasing or merging, rerun `palace scan` and `palace collect` locally to refresh outputs before further work.
- Encourage agents to surface scope and scan identity (from context-pack) in their explanations to avoid silent widening.

## Migration note
- If an existing `.palace/palace.jsonc` contains an `outputs` block, remove it. Output paths are fixed conventions (`.palace/index/palace.db`, `.palace/outputs/context-pack.json`) and are not configurable.
