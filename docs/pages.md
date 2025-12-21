---
layout: default
title: Deployment
nav_order: 13
---

# GitHub Pages Deployment

Documentation is published from `/docs` via GitHub Pages.

---

## Configuration

**GitHub Settings**:
- Source: `/docs` folder
- Branch: `main`
- Theme: just-the-docs (remote)

**Jekyll Config** (`_config.yml`):
```yaml
title: Mind Palace
description: Deterministic context for AI agents
remote_theme: pmarsceill/just-the-docs
color_scheme: dark
```

---

## Deployment Workflow

`.github/workflows/docs.yml`:

```yaml
name: Deploy Docs

on:
  push:
    branches: [main]
    paths: ['docs/**', 'README.md']
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: pages
  cancel-in-progress: true

jobs:
  deploy:
    runs-on: ubuntu-latest
    environment:
      name: github-pages
    steps:
      - uses: actions/checkout@v4
      - uses: actions/configure-pages@v4
      - uses: actions/upload-pages-artifact@v3
        with:
          path: docs
      - uses: actions/deploy-pages@v4
```

---

## Doc Structure

| File | Purpose | nav_order |
|------|---------|-----------|
| `index.md` | Landing page | 1 |
| `ecosystem.md` | Architecture | 2 |
| `concepts.md` | Core concepts | 3 |
| `workflows.md` | Usage patterns | 4 |
| `cli.md` | Command reference | 5 |
| `agents.md` | AI integration | 6 |
| `extension.md` | VS Code extension | 7 |
| `collaboration.md` | Git/CI model | 8 |
| `extensibility.md` | Schema evolution | 9 |
| `COMPATIBILITY.md` | Version matrix | 10 |
| `branding.md` | Visual identity | 11 |
| `product.md` | Product overview | 12 |
| `pages.md` | This file | 13 |
| `errors.md` | Error codes | 14 |

---

## Guidelines

- Update docs when CLI behavior changes
- Keep examples deterministic (no timestamps)
- Don't commit generated artifacts to `/docs`
- Use front matter for navigation order
