---
layout: default
title: Collaboration
nav_order: 8
---

# Git & Collaboration

Mind Palace separates **curated** (committed) from **generated** (gitignored) artifacts.

---

## What to Commit

```
.palace/
├── palace.jsonc          ✓ Commit
├── rooms/*.jsonc         ✓ Commit
├── playbooks/*.jsonc     ✓ Commit
├── project-profile.json  ✓ Commit
├── schemas/              ✓ Commit (export-only, don't edit)
├── index/                ✗ Gitignore
├── outputs/              ✗ Gitignore
└── cache/                ✗ Gitignore
```

### .gitignore

```gitignore
.palace/index/
.palace/outputs/
.palace/cache/
.palace/maps/
*.db
```

---

## Team Workflow

### Developer Loop

```sh
# After pulling changes
palace scan              # Rebuild Index
palace collect           # Refresh context pack
palace verify            # Confirm freshness

# After making changes
palace scan && palace collect
```

### Code Review

Curated changes (Rooms, Playbooks, config) should be reviewed. Add to CI:

```yaml
- name: Lint Mind Palace
  run: palace lint
```

---

## CI Integration

### Basic Verification

```yaml
jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Palace
        run: |
          curl -L https://github.com/koksalmehmet/mind-palace/releases/latest/download/palace-linux-amd64 -o palace
          chmod +x palace && sudo mv palace /usr/local/bin/

      - name: Lint
        run: palace lint

      - name: Scan
        run: palace scan

      - name: Verify
        run: palace verify --strict
```

### Diff-Scoped Verification

```yaml
- name: Verify PR Changes
  run: |
    palace signal --diff ${{ github.event.pull_request.base.sha }}..${{ github.sha }}
    palace verify --strict --diff ${{ github.event.pull_request.base.sha }}..${{ github.sha }}
```

### Caching Index

```yaml
- name: Cache Palace Index
  uses: actions/cache@v3
  with:
    path: .palace/index
    key: palace-index-${{ hashFiles('**/*.ts', '**/*.go', '**/*.py') }}
```

---

## Merge Conflicts

After merge/rebase:

```sh
palace scan && palace collect
```

The Index is derived from source files. Regenerate, don't merge.

---

## Multi-Repo (Corridors)

For teams with multiple repositories:

```jsonc
// frontend/.palace/palace.jsonc
{
  "neighbors": {
    "backend": {
      "url": "https://storage.example.com/backend/context-pack.json",
      "ttl": "24h"
    }
  }
}
```

Publish context packs in CI:

```yaml
- name: Publish Context Pack
  run: |
    palace collect
    aws s3 cp .palace/outputs/context-pack.json s3://bucket/project/context-pack.json
```

Neighbor files are namespaced: `corridor://backend/src/api.ts`
