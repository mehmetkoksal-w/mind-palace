# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0-alpha] - 2025-12-31

> **Breaking Changes**: This release includes command renames. See the [Migration Guide](https://koksalmehmet.github.io/mind-palace/reference/migration) for upgrade instructions.

### Breaking Changes

- **Command Renames**: Several commands have been renamed for clarity
  - `palace ask` → `palace query` (search the codebase)
  - `palace collect` → `palace context` (generate context packs)
  - `palace verify` → `palace check` (verify freshness, now includes lint)
  - `palace signal` → `palace ci signal` (moved under ci subcommand)
  - `palace detect` removed (merged into `palace init`)

### Changed

- **Codebase Restructuring**: Major refactoring to package-based architecture
  - Separated internal packages for better modularity
  - Created public API packages (`pkg/memory`, `pkg/corridor`, `pkg/types`)
  - Improved code organization following Go best practices

### Added

- **Postmortems**: Failure memory system for tracking and learning from failures
  - Create postmortems with structured analysis (what happened, root cause, lessons)
  - Severity levels: low, medium, high, critical
  - Status tracking: open, resolved, recurring
  - Convert postmortem lessons to learnings automatically
  - Link postmortems to related decisions and sessions
  - MCP tools: `store_postmortem`, `get_postmortems`, `resolve_postmortem`

- **AI Context Preview**: Preview what context AI agents will receive
  - Preview auto-injection context for any file path
  - Token budget visualization with breakdown
  - Toggle inclusion of learnings, decisions, and failures
  - Priority scoring with "why included" explanations
  - API: `POST /api/context/preview`

- **Decision Timeline**: Visual decision evolution tracking
  - Horizontal timeline visualization with outcome color coding
  - Decision chain tracking (predecessors and successors)
  - Scope filtering (file, room, palace)
  - "Review needed" indicators for decisions with unknown outcomes
  - API: `GET /api/decisions/timeline`, `GET /api/decisions/:id/chain`

- **Scope Explorer**: Knowledge inheritance visualization
  - 4-level scope hierarchy: file → room → palace → corridor
  - Visual inheritance chain with record counts
  - Toggle inheritance per scope level
  - Explains what knowledge applies to any file path
  - API: `POST /api/scope/explain`, `GET /api/scope/hierarchy`

- **Brain System**: Track ideas, decisions, and outcomes
  - `palace remember` - Auto-classify and store ideas/decisions/learnings
  - `palace outcome` - Record decision outcomes (successful, failed, etc.)
  - `palace review` - Review old decisions awaiting feedback
  - `palace link` - Create relationships between records

- **Call Graph**: Explore function relationships
  - `palace graph <symbol>` - Visualize callers and callees

- **File Intelligence**: Enhanced file tracking
  - `palace intel <file>` - Show edit history and failure rates
  - `palace brief <file>` - Get briefing before editing

- **CI Commands**: Dedicated CI/CD commands
  - `palace ci verify` - Verify index freshness in CI
  - `palace ci collect` - Generate context for CI
  - `palace ci signal` - Generate change signals

- **Housekeeping**:
  - `palace maintenance` - Clean up stale sessions, learnings, and links

- **Comprehensive Test Suite**: Added tests for all packages
  - Unit tests for all internal packages
  - Integration tests for full system workflows
  - End-to-end tests covering complete user workflows

- **Public API packages**: Stable Go API for external tools
  - `pkg/memory` - Session and learning management
  - `pkg/corridor` - Cross-workspace knowledge sharing
  - `pkg/types` - Common type definitions

- **Dashboard Enhancements**:
  - New Insights sub-pages: Context Preview, Decision Timeline, Postmortems, Scope Explorer
  - SPA routing support for client-side navigation
  - WebSocket support for real-time updates

### Fixed

- Fixed postmortem time.Time scanning error when retrieving records from SQLite
- Fixed SPA routing - dashboard now properly serves Angular routes
- Fixed embed.go build issue with dist/.gitkeep
- Fixed GitHub Actions workflow for macOS and Windows builds
- Fixed MSI installer build with WiX UI extension

## [0.0.1-alpha] - 2025-12-23

### Added

- **Mind Palace CLI**: A deterministic context system for codebases
  - `palace init` - Initialize a new Mind Palace workspace
  - `palace scan` - Index codebase with full-text search (SQLite FTS5)
  - `palace scan --incremental` - Fast delta scanning for large codebases
  - `palace collect` - Generate context packs from rooms
  - `palace ask` - Intent-based semantic search
  - `palace verify` - Staleness detection for CI/CD
  - `palace signal` - Diff-based change signals
  - `palace lint` - Schema validation for all artifacts
  - `palace dashboard` - Web UI for visualization
  - `palace serve` - MCP server for AI agent integration
  - `palace update` - Self-update from GitHub releases

- **Session Memory**: Track agent sessions and learnings
  - Session lifecycle (start, pause, resume, end)
  - Activity logging (file edits, tool calls, decisions)
  - Learning capture with confidence scoring
  - Recall API for retrieving past learnings

- **Corridors**: Cross-project knowledge sharing
  - Link multiple workspaces
  - Global search across linked projects
  - Automatic discovery promotion

- **Dashboard**: Web UI for visualization
  - Codebase graph visualization
  - Session history browser
  - File hotspot analysis
  - Search interface

- **VS Code Extension**: IDE integration
  - Status bar HUD (Fresh/Stale indicator)
  - Blueprint sidebar (Rooms, Map views)
  - Auto-heal on file save
  - Keyboard shortcuts

- **Multi-Language Analysis**: Code parsing for 15+ languages
  - Go, TypeScript, JavaScript, Python, Rust
  - Java, C#, C/C++, Swift, Ruby
  - SQL, YAML, HCL, Dockerfile, Bash

- **MCP Integration**: AI agent protocol support
  - Compatible with Claude Desktop, Cursor
  - Full Butler API exposure via JSON-RPC
