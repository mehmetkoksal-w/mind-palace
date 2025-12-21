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
| CLI       | 0.0.1-rc1 | 1.0.0 |
| Extension | 0.0.1-rc1 | 1.0.0 |

Versions are **synced** across components.

---

## Version Source

Both repos use a `VERSION` file as the single source of truth:

```
mind-palace/VERSION          → CLI version
mind-palace-vscode/VERSION   → Extension version
```

CI reads `VERSION` to determine if a release is needed.

---

## Semver Rules

| Bump | Meaning | Example |
|------|---------|---------|
| Major | Breaking changes (CLI output, MCP protocol, schemas) | 1.0.0 → 2.0.0 |
| Minor | New features, backwards compatible | 0.1.0 → 0.2.0 |
| Patch | Bug fixes | 0.1.0 → 0.1.1 |
| Pre-release | Testing releases | 0.1.0-rc1 |

---

## Release Process

Both repos use identical automated release pipelines:

```
VERSION file change → Push to main → CI runs → Review approval → GitHub Release
```

**To release**:
1. Update `VERSION` file (e.g., `0.2.0`)
2. Update `CHANGELOG.md`
3. Update `package.json` version (extension only)
4. Commit and push to `main`
5. CI detects new version, requests approval
6. Approve in GitHub Actions
7. Release created with artifacts

**Artifacts**:
- CLI: binaries (darwin/linux/windows) + macOS .pkg installers
- Extension: `.vsix` file

---

## Installation

### CLI

Download from [GitHub Releases](https://github.com/koksalmehmet/mind-palace/releases):

```sh
# macOS/Linux
curl -L https://github.com/koksalmehmet/mind-palace/releases/latest/download/palace-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m) -o palace
chmod +x palace
sudo mv palace /usr/local/bin/
```

### Extension

Download `.vsix` from [GitHub Releases](https://github.com/koksalmehmet/mind-palace-vscode/releases):

```sh
code --install-extension mind-palace-0.0.1-rc1.vsix
```

---

## Version Check

The extension validates CLI version on activation:

```typescript
// src/version.ts
const COMPATIBILITY_MATRIX = {
  '0.0.1-rc1': '0.0.1-rc1',  // extension → required CLI
};
```

Incompatible versions trigger a warning notification.

---

## Changelog

### 0.0.1-rc1

- Pre-release
- Schema version 1.0.0 (preview)
- MCP tools: `search_mind_palace`, `list_rooms`
