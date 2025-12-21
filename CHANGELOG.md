# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.1-rc2] - 2025-12-21

### Fixed

- **CLI: Symlink Handling**: Fixed scanner failing on symlinked directories (common in Flutter/iOS projects). Symlinked directories are now skipped gracefully, and broken symlinks are handled without errors.
- **CLI: Language Detection**: `palace detect` now correctly identifies Flutter/Dart projects (`pubspec.yaml`) along with Rust, Python, Ruby, Java, C#, Swift, and PHP projects instead of returning "unknown".
- **CLI: Room Template**: Removed Go-specific `go.mod` from default `project-overview.jsonc` template entry points. Now uses only `README.md` which is universal across all project types.

### Changed

- **CLI: Extended Language Support**: Added detection and default commands (test, lint, deps) for 10+ languages:
  - Dart/Flutter (`pubspec.yaml`)
  - Rust (`Cargo.toml`)
  - Python (`pyproject.toml`, `setup.py`, `requirements.txt`)
  - Ruby (`Gemfile`)
  - Java/Kotlin (`pom.xml`, `build.gradle`)
  - C#/.NET (`.csproj`, `.sln`)
  - Swift (`Package.swift`)
  - PHP (`composer.json`)

## [0.0.1-rc1] - 2025-12-21

### Added

- **Mind Palace Initial Release**: A unified workspace for high-fidelity context management.
- **Butler (Intent Engine)**:
  - Integrated intent-based search with `palace ask`.
  - MCP Server (`palace serve`) for deep integration with AI agents like Claude and Cursor.
  - Advanced ranking using BM25 and structural heuristics (entry points, file types).
- **Corridors (Distributed Context)**:
  - Native support for multi-repo architectures via `neighbors` configuration.
  - Virtualized namespaces (`corridor://`) and resilient remote artifact caching.
- **CLI Workspace & Lifecycle**:
  - `init`, `scan`, `collect`, `verify`, `plan`, and `signal` for deterministic context management.
  - **Palace Verify**: Fast and strict staleness checks for CI/CD safety.
  - **Palace Lint**: Schema-first validation for all curated artifacts (rooms, playbooks, project profiles).
  - **Palace Explain**: Built-in inspector for project invariants, behaviors, and expected outputs.
  - **Palace Signal**: Automated diff-signal generation for strict incremental workflows.
- **Architecture & Tooling**:
  - **Tier-0 Index**: SQLite-backed indexing (WAL + FTS5) with full-text search capabilities.
  - **Embedded Schemas**: Built-in JSON Schema validation for all configuration files.
  - **Documentation Workflow**: Canonical Markdown docs in `/docs` with ready-to-use GitHub Pages publishing setup.
  - Curated starter templates for rapid project onboarding.
