---
layout: default
title: Compatibility
nav_order: 10
---

# Compatibility Matrix

Version compatibility between ecosystem components.

---

## Current Versions

| Component | Version | Schema |
|-----------|---------|--------|
| CLI       | 0.0.1-alpha | 1.0.0 |
| Extension | 0.0.1-alpha | 1.0.0 |

Versions are **synced** across components.

---

## Version Source

The monorepo uses a `VERSION` file as the single source of truth:

```
mind-palace/VERSION              → CLI version
mind-palace/apps/vscode/package.json → Extension version
```

CI reads `VERSION` to determine release version for tags.

---

## Semver Rules

| Bump | Meaning | Example |
|------|---------|---------|
| Major | Breaking changes (CLI output, MCP protocol, schemas) | 1.0.0 → 2.0.0 |
| Minor | New features, backwards compatible | 0.1.0 → 0.2.0 |
| Patch | Bug fixes | 0.1.0 → 0.1.1 |
| Pre-release | Testing releases | 0.1.0-alpha, 0.1.0-beta |

---

## Release Process

Automated release pipeline triggered by tags:

```
Create tag v0.0.1-alpha → Push → CI builds → GitHub Release
```

**To release**:
1. Update `VERSION` file
2. Update `CHANGELOG.md`
3. Update `apps/vscode/package.json` version
4. Commit and push to `main`
5. Create and push tag: `git tag v0.0.1-alpha && git push origin v0.0.1-alpha`
6. CI builds all artifacts and creates release

**Artifacts**:
- CLI: binaries (darwin/linux/windows)
- Extension: `.vsix` file
- Dashboard: embedded in CLI binary

---

## Installation

### CLI

Download from [GitHub Releases](https://github.com/mehmetkoksal-w/mind-palace/releases):

```sh
# macOS (Apple Silicon)
curl -L https://github.com/mehmetkoksal-w/mind-palace/releases/latest/download/palace-darwin-arm64 -o palace
chmod +x palace
sudo mv palace /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/mehmetkoksal-w/mind-palace/releases/latest/download/palace-darwin-amd64 -o palace
chmod +x palace
sudo mv palace /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/mehmetkoksal-w/mind-palace/releases/latest/download/palace-linux-amd64 -o palace
chmod +x palace
sudo mv palace /usr/local/bin/
```

### Extension

Download `.vsix` from [GitHub Releases](https://github.com/mehmetkoksal-w/mind-palace/releases):

```sh
code --install-extension mind-palace-vscode-0.0.1-alpha.vsix
```

---

## Version Check

The extension validates CLI version on activation:

```typescript
// src/version.ts
const COMPATIBILITY_MATRIX = {
  '0.0.1-alpha': '0.0.1-alpha',  // extension → required CLI
};
```

Incompatible versions trigger a warning notification.

---

## Changelog

See [CHANGELOG.md](https://github.com/mehmetkoksal-w/mind-palace/blob/main/CHANGELOG.md) for full release history.

### 0.0.1-alpha

- Initial public release
- Schema version 1.0.0
- Full CLI, Dashboard, and VS Code extension
