# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0-alpha] - 2026-01-07

### Added

- **Postmortem Feature**: New postmortem commands and webview for capturing lessons learned from tasks, bugs, and incidents
- **Butler Registry Architecture**: Refactored VS Code extension to use centralized CommandRegistry, ProviderRegistry, ViewRegistry, and EventBus patterns
- **Knowledge Tree Enhancements**: Added postmortem category and status grouping in knowledge panel
- **Onboarding Flow**: First-run onboarding experience in dashboard with sample data creation
- **LLM Hardening**: Comprehensive test suite for LLM clients (Ollama, OpenAI, Anthropic) with 90.6% coverage
- **Cache Service**: LRU cache implementation with bridge integration for performance optimization
- **Logger Services**: Unified logging infrastructure across VS Code extension and dashboard

### Changed

- **VS Code Extension Architecture**: Migrated to registry-based component organization for better maintainability
- **Bridge API**: Added public postmortem methods (getPostmortem, resolvePostmortem, postmortemToLearnings)
- **Parser Strategy**: Documented LSP-first parsing approach with tree-sitter/regex fallback for future adaptations
- **Build Configuration**: MinGW configured for CGO support in CI environments

### Fixed

- **TypeScript Compilation**: Resolved all 48 TypeScript errors in VS Code extension
- **Provider Registration**: Fixed knowledge tree provider syntax and rendering logic
- **Config Watcher**: Added graceful handling for missing workspace folders
- **MCP Client**: Improved connection handling and error recovery

### Testing

- **VS Code Extension**: 41/49 tests passing (84% - 8 failures related to workspace/timing in test environment)
- **Dashboard**: 205/211 tests passing (97% - 6 failures in onboarding feature specs)
- **Go CLI**: Core packages validated (config, corridor, LLM, signal, project, validate)

### Known Issues

- CGO-dependent packages (analysis, butler, scan) fail on Windows without MinGW; Linux CI provides full coverage
- Cytoscape import warning in blueprint webview (requires esModuleInterop)
- Benchmark execution pending due to package structure consolidation needed

---

## [0.0.2-alpha] - 2026-01-01

### Fixed

- **Dashboard Embedding**: Fixed panic on startup when dashboard assets were embedded with different directory structures (local vs CI builds)
- **Recall Link Flags**: Fixed flag parsing for `palace recall link` - flags must now come before the source ID (documented correctly)
- **Dart Call Graph**: Deep analysis (LSP-based call tracking) now runs automatically for Dart/Flutter projects

### Added

- **Explore Rooms**: New `palace explore --rooms` flag to list all configured rooms in the workspace
- **Auto Deep Analysis**: Dart/Flutter projects are automatically detected and deep analysis runs during scan

### Changed

- Dashboard upgraded from Angular 17 to Angular 21
- Dashboard components updated to Angular 21 standalone defaults
- TypeScript upgraded to 5.9.3
- Zone.js upgraded to 0.15.1

---

## [0.0.1-alpha] - 2026-01-01

### Welcome to the Mind Palace

This is the maiden release of Mind Palace, a "Second Brain" for developers and AI agents. It transforms your codebase from a collection of files into a living, searchable memory palace.

### Core Features

#### Unified Intelligence CLI

A "Clean and Neat" interface designed for high-velocity development.

- **`init`**: Effortless workspace setup with project auto-detection.
- **`scan`**: High-performance structural indexing powered by Tree-sitter and SQLite FTS5.
- **`explore`**: Intent-based semantic search and recursive call-graph tracing.
- **`brief`**: Instant structural intelligence for file-level context and edit history.

#### The Developer's Second Brain

Store what code can't: the "Why" behind your decisions.

- **Knowledge Capture**: Natural language storage for Ideas, Decisions, and Learnings.
- **Decision Lifecycle**: Track decision outcomes (Success, Failure, Iteration) to build a project narrative.
- **Knowledge Linking**: Build a graph of relationships between records (e.g., "this learning supports that decision").

#### AI Protocol & Ecosystem

Seamlessly integrated with existing AI tools.

- **MCP Server**: Native Model Context Protocol (MCP) support for Claude, Cursor, and custom agents.
- **Session Memory**: Automated tracking of agent interactions and learning extraction.
- **Corridors**: Secure, cross-workspace knowledge sharing for poly-repo ecosystems.

#### Web Dashboard & Visualization

- **Mind Palace UI**: A lush, interactive dashboard for visualizing codebase maps, knowledge timelines, and hotspot patterns.
- **Interactive Graphs**: Dynamic call-chain and knowledge inheritance visualizations.

### Technical Foundation

- **Language Support**: Deep parsing for 15+ languages including Go, TS/JS, Python, Rust, and C#.
- **ACID Reliable**: Concurrent, WAL-mode SQLite indexing.
- **CI/CD Ready**: Deterministic context generation and Git-scoped verification.

---

_Mind Palace: Because code is only half the story._
