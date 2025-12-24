# Changelog

All notable changes to the Mind Palace Observer VS Code extension will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
