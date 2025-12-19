# End-to-End Workflow

This is a realistic daily loop for using Mind Palace on an active repository.

## 1) Initialize palace
`palace init` creates `.palace/` scaffolding and exports embedded schemas. Optional `--with-outputs` creates a starter context-pack; defaults avoid generating outputs.

## 2) Detect project profile
`palace detect` writes `.palace/project-profile.json` (JSON) based on the current repo. Curated files remain untouched.

## 3) Scan workspace (Tier-0)
`palace scan` reads files outside guardrails, chunks content, hashes files, and writes:
- `.palace/index/palace.db` (SQLite WAL + FTS5)
- `.palace/index/scan.json` (validated scan summary with IDs/counts/hash)
Always run scan after meaningful file changes to keep the index fresh.

## 4) Plan with a goal
`palace plan --goal "<goal>"` seeds or updates the context pack goal and provenance. It does not scan or collect evidence.

## 5) Collect context
`palace collect [--diff <range>] [--allow-stale]` assembles `.palace/outputs/context-pack.json` from the existing index + curated manifests.
- **Full scope (no --diff)**: fails if the index is stale unless `--allow-stale`.
- **Diff scope (--diff)**: uses git diff or a matching change-signal; never widens scope.

## 6) Run agent / perform changes
Use context-pack + rooms/playbooks to guide changes. Agents should not rescan; they should consume the generated artifacts.

## 7) Capture change signal (diff-scoped work)
`palace signal --diff <range>` writes `.palace/outputs/change-signal.json` (sorted, hashed for non-deleted paths). Useful for CI, code review, and deterministic agent runs.

## 8) Verify consistency
`palace verify [--fast|--strict] [--diff <range>]` runs lint, then staleness checks.
- **Fast (default)**: mtime/size shortcut with selective hashing.
- **Strict**: hash all candidates.
- **Diff strictness**: if diff cannot be computed, verification errors instead of widening scope. Empty diffs verify zero candidates.

## 9) Iterate
After merges or new changes, rerun scan → collect → verify. Keep curated files in VCS; keep generated outputs ignored but reproducible.

## Releasing
- Tag and push: `git tag v0.1.0 && git push origin v0.1.0`
- CI builds cross-platform binaries, generates SHA256SUMS, and publishes a GitHub Release automatically.
