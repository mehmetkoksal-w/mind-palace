---
layout: default
title: Deployment
nav_order: 17
---

# GitHub Pages Deployment

Documentation is published from `/docs` via GitHub Pages.

---

## Configuration

**GitHub Settings**:
- Source: GitHub Actions
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

Documentation is deployed as part of the main pipeline (`.github/workflows/pipeline.yml`):

```yaml
docs:
  name: Deploy Docs
  if: github.event_name == 'push'
  needs: ci
  runs-on: ubuntu-latest
  environment:
    name: production-docs
    url: ${{ steps.deployment.outputs.page_url }}
  steps:
    - uses: actions/checkout@v4
    - uses: actions/configure-pages@v4
    - uses: actions/jekyll-build-pages@v1
      with:
        source: ./docs
        destination: ./_site
    - uses: actions/upload-pages-artifact@v3
      with:
        path: ./_site
    - uses: actions/deploy-pages@v4
```

Docs are deployed automatically when:
1. CI passes on main branch
2. Push event (not workflow_dispatch)

---

## CI Validation

Documentation is validated during CI (`.github/workflows/ci.yml`):

```yaml
build-docs:
  name: Build Docs
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: ruby/setup-ruby@v1
      with:
        ruby-version: '3.2'
    - run: gem install bundler jekyll
    - run: cd docs && bundle exec jekyll build
```

This ensures docs build successfully before merge.

---

## Doc Structure

### Getting Started
| File | Purpose | nav_order |
|------|---------|-----------|
| `index.md` | Landing page | 1 |
| `concepts.md` | Core concepts | 2 |
| `workflows.md` | Usage patterns | 3 |
| `cli.md` | Command reference | 4 |

### Features
| File | Purpose | nav_order |
|------|---------|-----------|
| `session-memory.md` | Agent sessions | 5 |
| `corridors.md` | Cross-project knowledge | 6 |
| `dashboard.md` | Web UI guide | 7 |
| `agents.md` | AI/MCP integration | 8 |

### Architecture
| File | Purpose | nav_order |
|------|---------|-----------|
| `architecture.md` | System design | 9 |
| `ecosystem.md` | Architecture details | 10 |
| `extension.md` | VS Code extension | 11 |
| `extensibility.md` | Schema evolution | 12 |
| `COMPATIBILITY.md` | Version matrix | 13 |
| `public-api.md` | Go packages API | 14 |

### Contributing
| File | Purpose | nav_order |
|------|---------|-----------|
| `development.md` | Development setup | 15 |
| `contributing.md` | How to contribute | 16 |
| `pages.md` | This file | 17 |

### Reference
| File | Purpose | nav_order |
|------|---------|-----------|
| `errors.md` | Error codes | 18 |
| `branding.md` | Visual identity | 19 |
| `product.md` | Product overview | 20 |
| `collaboration.md` | Git/CI model | 21 |

---

## Local Development

To preview docs locally:

```sh
cd docs
bundle install
bundle exec jekyll serve
# Open http://localhost:4000
```

Or using Docker:

```sh
docker run --rm -v "$PWD/docs:/srv/jekyll" -p 4000:4000 jekyll/jekyll jekyll serve
```

---

## Guidelines

- Update docs when CLI behavior changes
- Keep examples deterministic (no timestamps)
- Don't commit generated artifacts to `/docs`
- Use front matter for navigation order
- Add new pages to this index
- Ensure all internal links work
