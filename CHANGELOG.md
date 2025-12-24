# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
