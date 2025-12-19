# mind-palace

A deterministic CLI for constructing and maintaining a structured “mind palace” over a codebase. Contracts are JSON Schema–first, the Tier-0 index is SQLite (WAL + FTS5), and curated artifacts are JSONC.

## .palace/ contract
- **Curated (commit)**: `.palace/palace.jsonc`, `.palace/rooms/*.jsonc`, `.palace/playbooks/*.jsonc`, `.palace/project-profile.json`, `.palace/schemas/*`, `.palace/maps` (if present).
- **Generated (ignore)**: `.palace/index/`, `.palace/outputs/`, `.palace/sessions/`, any `*.db` artifacts.
- **Embedded schemas**: Validation always uses the embedded schemas; `.palace/schemas` are export-only for transparency.

## Command expectations
- `palace lint`: validates curated artifacts only (palace.jsonc, rooms, playbooks, project-profile) against embedded schemas.
- `palace scan`: builds/refreshes the Tier-0 index (files, hashes, FTS chunks) and writes `.palace/index/palace.db`.
- `palace collect`: reads the existing index + curated manifests to refresh `.palace/outputs/context-pack.json` (no scan).
- `palace verify --fast|--strict [--diff range]`: runs lint, then staleness checks (fast = mtime/size with selective hashing; strict = hash all). `--fast` and `--strict` cannot be combined.
- `palace signal --diff range`: generates `.palace/outputs/change-signal.json` from git diff output; validation uses embedded schemas.

## Quick start
```sh
go run ./cmd/palace init          # curated scaffolding only
go run ./cmd/palace detect
go run ./cmd/palace scan
```

Then iterate with `palace plan` or `palace collect` (these write `.palace/outputs/context-pack.json`) and `palace verify` as you work. Only `palace scan` mutates the index; `palace collect` and `palace verify` assume an up-to-date scan.
