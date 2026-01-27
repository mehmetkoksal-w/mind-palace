# Changelog

All notable changes to the Mind Palace VS Code extension will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.2-alpha] - 2026-01-27

### Changed

- **Simplified to Status Bar + LSP**: Extension now focuses on two core features
- **Minimal Footprint**: Removed sidebar, webviews, and most commands

### Removed

- **Sidebar Views**: Blueprint, Knowledge, Sessions, and Corridor views removed
- **Webviews**: Knowledge graph and blueprint visualizations removed
- **Commands**: Store, session management, semantic search commands removed
- **Keyboard Shortcuts**: Removed all shortcuts except status check
- **File Decorations**: Inline decorations removed
- **Auto-Heal**: Auto-sync on file save removed (use CLI `palace scan --watch` instead)

### Kept

- **Status Bar**: Shows Palace status with knowledge counts (✓ Palace 2D/1I/3L)
- **LSP Integration**: Real-time pattern and contract diagnostics
- **2 Commands**: `checkStatus` and `restartLsp`

---

## [0.4.1-alpha] - 2026-01-21

### Added

- **LSP Integration**: Full Language Server Protocol client
  - Pattern violation diagnostics with confidence scores
  - Contract mismatch diagnostics between frontend/backend
  - Hover information showing pattern/contract details
  - Code actions to approve, ignore, or verify issues
  - Code lens showing issue counts per file
  - Crash recovery with automatic server restart

---

## [0.0.1-alpha] - 2026-01-01

### Initial Release

- Status bar indicator for index freshness
- Blueprint sidebar with tree and graph views
- Knowledge view for ideas, decisions, learnings
- Session tracking for AI agents
- Semantic search integration
- Auto-heal on file save

---

_Mind Palace: Context at the speed of thought._
