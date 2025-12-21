# mind-palace

A deterministic CLI for constructing and maintaining a structured “mind palace” over a codebase. Contracts are JSON Schema–first, the Tier-0 index is SQLite (WAL + FTS5), and curated artifacts are JSONC.

## Documentation

- Canonical docs live in `/docs` (Markdown, schema-first, deterministic).
- GitHub Pages can publish `/docs` directly (see `/docs/pages.md` for recommended setup and workflow triggers).
- Start with `/docs/index.md` for overview/philosophy, `/docs/product.md` for positioning and feature set, then `/docs/concepts.md` and `/docs/workflows.md` for day-to-day usage.

## .palace/ contract

- **Curated (commit)**: `.palace/palace.jsonc`, `.palace/rooms/*.jsonc`, `.palace/playbooks/*.jsonc`, `.palace/project-profile.json`, `.palace/schemas/*`.
- **Generated (ignore)**: `.palace/index/`, `.palace/outputs/`, `.palace/maps/`, any `*.db` artifacts. A future `.palace/sessions/` may appear but is not created today.
- **Embedded schemas**: Validation always uses the embedded schemas; `.palace/schemas` are export-only for transparency.

## Key artifacts

- `.palace/index/palace.db`: Tier-0 index (files + metadata + chunked FTS content)
- `.palace/index/scan.json`: validated scan summary (UUID scanId + dbScanId + counts + scanHash)
- `.palace/outputs/context-pack.json`: validated context pack used as “authoritative working context”
- `.palace/outputs/change-signal.json`: validated diff signal for strict diff workflows

## Command expectations

- `palace lint`: validates curated artifacts only (palace.jsonc, rooms, playbooks, project-profile) against embedded schemas.
- `palace scan`: builds/refreshes the Tier-0 index (files, hashes, FTS chunks), writes `.palace/index/palace.db`, and emits `.palace/index/scan.json`.
- `palace collect [--diff range] [--allow-stale]`: reads existing index + curated manifests to refresh `.palace/outputs/context-pack.json` (no scan).
  - **Full scope** (no `--diff`): fails if index is stale unless `--allow-stale`.
  - **Diff scope** (`--diff`): uses git diff or a matching change-signal; never widens scope silently.
- `palace verify --fast|--strict [--diff range]`: runs lint, then staleness checks.
  - **Diff strictness**: if diff cannot be computed, verify errors (no fallback to full repo).
  - Empty diffs verify zero candidates without fallback.
- `palace signal --diff range`: generates `.palace/outputs/change-signal.json` from git diff output; validation uses embedded schemas.
- `palace explain [scan|collect|verify|signal|artifacts|all]`: prints behavior, invariants, and outputs.

### Butler (Intent-Based Search)

- `palace ask <query>`: Search the codebase by intent or keywords. Returns ranked results grouped by "Room".
  - Supports natural language: `palace ask "where is the authentication logic"`
  - Supports code symbols: `palace ask "SearchChunks"` or `palace ask "func_name"`
  - Filter by room: `palace ask --room project-overview "entry points"`
- `palace serve`: Start an MCP (Model Context Protocol) server over stdio for AI agent integration.
  - Exposes `search_mind_palace` and `list_rooms` tools
  - Exposes `palace://files/{path}` and `palace://rooms/{name}` resources
  - Compatible with Claude Desktop, Cursor, and other MCP-enabled agents

## MCP Integration (AI Agents)

To connect Claude Desktop or other MCP clients to Mind Palace, add to your MCP config:

```json
{
  "mcpServers": {
    "mind-palace": {
      "command": "palace",
      "args": ["serve", "--root", "/path/to/your/project"]
    }
  }
}
```

This enables AI agents to:

1. **Search by intent**: "Where is the auth logic?" → Ranked code snippets grouped by Room
2. **Discover structure**: List available Rooms with their entry points and capabilities
3. **Read files**: Access full file content for deeper understanding

## Corridors (Multi-Repo Support)

Corridors enable distributed context collection across multiple repositories. Frontend can import Backend's "public contract" without cloning the source.

### Configuration

Add `neighbors` to your `palace.jsonc`:

```jsonc
{
  "neighbors": {
    "backend": {
      "url": "https://storage.example.com/backend/context-pack.json",
      "auth": {
        "type": "bearer",
        "token": "$BACKEND_API_TOKEN"
      },
      "ttl": "24h"
    },
    "core": {
      "localPath": "../core-lib" // For monorepos
    }
  }
}
```

### Features

- **Namespace Isolation**: Remote files are prefixed as `corridor://backend/src/api.ts` to prevent collisions
- **Resilient Caching**: Works offline after first fetch; cached in `.palace/cache/neighbors/`
- **Flexible Auth**: Supports `bearer`, `basic`, and custom `header` authentication with `$ENV_VAR` expansion
- **TTL Control**: Configure refresh intervals per neighbor (default: 24h)

When you run `palace collect`, corridors are automatically fetched and merged into your context pack.

## Quick start

```sh
go run ./cmd/palace init
go run ./cmd/palace detect
go run ./cmd/palace scan
go run ./cmd/palace collect --allow-stale   # optional first run convenience
go run ./cmd/palace verify

# Butler (search by intent)
go run ./cmd/palace ask "where is the search logic"
```

## Install (no Go toolchain required)

### Via Go (developer machines)

```sh
go install github.com/koksalmehmet/mind-palace/cmd/palace@latest
```

### Latest release (binaries)

- macOS arm64:
  ```sh
  curl -L https://github.com/koksalmehmet/mind-palace/releases/latest/download/palace-darwin-arm64 -o palace
  chmod +x palace && sudo mv palace /usr/local/bin/
  ```
- Linux amd64:
  ```sh
  curl -L https://github.com/koksalmehmet/mind-palace/releases/latest/download/palace-linux-amd64 -o palace
  chmod +x palace && sudo mv palace /usr/local/bin/
  ```
- Windows amd64: download `https://github.com/koksalmehmet/mind-palace/releases/latest/download/palace-windows-amd64.exe` and place on PATH.

Asset names (no extensions except Windows):

- macOS: `palace-darwin-arm64`
- Linux: `palace-linux-amd64`
- Windows: `palace-windows-amd64.exe`

### Pinned version (recommended for CI)

Replace `${VERSION}` (e.g., `v0.0.1-rc1`):

- macOS arm64:
  ```sh
  curl -L https://github.com/koksalmehmet/mind-palace/releases/download/${VERSION}/palace-darwin-arm64 -o palace
  chmod +x palace && sudo mv palace /usr/local/bin/
  ```
- Linux amd64:
  ```sh
  curl -L https://github.com/koksalmehmet/mind-palace/releases/download/${VERSION}/palace-linux-amd64 -o palace
  chmod +x palace && sudo mv palace /usr/local/bin/
  ```
- Windows amd64: download `https://github.com/koksalmehmet/mind-palace/releases/download/${VERSION}/palace-windows-amd64.exe`.

### Checksums

Each release includes `SHA256SUMS`.

- macOS: `shasum -a 256 palace`
- Linux: `sha256sum palace`
- Or verify all: `sha256sum --check SHA256SUMS`

## Releasing

- Tag and push: `git tag v0.1.0 && git push origin v0.1.0`
- CI builds cross-platform binaries, generates SHA256SUMS, and publishes a GitHub Release automatically.
