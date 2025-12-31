# Changelog

All notable changes to the Mind Palace Observer VS Code extension will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0-alpha] - 2025-12-31

> **Note**: This release requires CLI v0.1.0-alpha or later.

### Added

- **Knowledge View**: Browse and manage ideas, decisions, and learnings
  - Tree structure grouped by status (ideas/decisions) or scope (learnings)
  - Detail panels with full content and metadata
  - Context menu for links management

- **Sessions View**: Track AI agent sessions
  - View active, completed, and abandoned sessions
  - Session detail panels with activity history
  - Start/End session commands

- **Corridor View**: Personal cross-workspace learnings
  - Browse personal learnings from the global corridor
  - View linked workspaces
  - Reinforce learning confidence

- **Conversation Storage**: Store and search past AI conversations
  - `Mind Palace: Search Conversations` command
  - Detail view for conversation history

- **Links Management**: Create relationships between records
  - `Mind Palace: Show Links` - view all links for a record
  - `Mind Palace: Create Link` - create relationships (supports, contradicts, implements, etc.)

- **Semantic Search**: AI-powered search for knowledge
  - `Mind Palace: Semantic Search` command (`Cmd/Ctrl+Shift+F`)
  - Hybrid keyword + semantic search
  - Filter by record type (ideas, decisions, learnings)
  - Requires embedding backend configuration in `palace.jsonc`

- **Keyboard Shortcuts**:
  - `Cmd/Ctrl+Shift+H`: Heal context
  - `Cmd/Ctrl+Shift+S`: Check status
  - `Cmd/Ctrl+Shift+B`: Open blueprint
  - `Cmd/Ctrl+Shift+K`: Quick Store selected text
  - `Cmd/Ctrl+Shift+F`: Semantic Search
  - `Cmd/Ctrl+Shift+M`: Show Mind Palace Menu

- **Configuration Options**:
  - `mindPalace.binaryPath`: Path to palace CLI
  - `mindPalace.autoSync`: Enable/disable auto-heal
  - `mindPalace.autoSyncDelay`: Debounce timing
  - `mindPalace.waitForCleanWorkspace`: Wait for all saves
  - `mindPalace.showStatusBarItem`: Toggle status bar visibility
  - `mindPalace.showFileDecorations`: Toggle editor decorations
  - `mindPalace.enableSemanticSearch`: Toggle semantic search features

- **Status Bar HUD**: Real-time palace status indicator
  - Fresh/Stale/Scanning states with color coding
  - Current room context display

- **Blueprint Sidebar**: Visual context explorer
  - Tree View: Hierarchical room and entry point browser
  - Map View: Graph visualization with Cytoscape.js

- **Auto-Heal**: Automatic scan on file save with debouncing

### Performance

- TTL-based caching for all tree providers
  - Knowledge provider: 1-minute cache
  - Session provider: 30-second cache
  - Corridor provider: 1-minute cache
- Request deduplication to prevent redundant MCP calls

### MCP Tools

- 3 new semantic search tools:
  - `search_semantic`: Pure AI embedding search
  - `search_hybrid`: Combined keyword + semantic search
  - `search_similar`: Find similar records

## [0.0.1-alpha] - 2025-12-23

### Added

- **Status Bar HUD**: Real-time palace status indicator
  - Fresh/Stale/Scanning states with color coding
  - Current room context display
  - Countdown timer during heal debounce

- **Blueprint Sidebar**: Visual context explorer
  - Tree View: Hierarchical room and entry point browser
  - Map View: Graph visualization with Cytoscape.js
  - Click-to-navigate to files

- **Commands**:
  - `Mind Palace: Heal` - Scan and collect context
  - `Mind Palace: Check Status` - Verify staleness
  - `Mind Palace: Open Blueprint` - Show sidebar

- **Keyboard Shortcuts**:
  - `Cmd+Shift+H` / `Ctrl+Shift+H`: Heal context
  - `Cmd+Shift+S` / `Ctrl+Shift+S`: Check status
  - `Cmd+Shift+B` / `Ctrl+Shift+B`: Open blueprint

- **Auto-Heal**: Automatic scan on file save
  - Configurable debounce delay (default 1500ms)
  - Optional wait for clean workspace
  - Non-blocking background execution

- **Configuration**:
  - `mindPalace.binaryPath`: Path to palace CLI
  - `mindPalace.autoSync`: Enable/disable auto-heal
  - `mindPalace.autoSyncDelay`: Debounce timing
  - `mindPalace.waitForCleanWorkspace`: Wait for all saves

- **Version Compatibility**: Automatic CLI version checking with upgrade prompts
