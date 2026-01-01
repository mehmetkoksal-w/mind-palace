# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
