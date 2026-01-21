# Sprint 1: Pattern Detection Foundation

> **Goal**: Implement automated pattern detection with confidence scoring and bulk approvals

## Sprint Overview

| Attribute | Value |
|-----------|-------|
| Sprint Number | 1 |
| Status | ✅ Complete |
| Depends On | Audit Phase (Complete) |
| Blocks | Sprint 2 (Contract Detection) |

---

## Objectives

### Primary
- [x] Implement pattern detection engine with detector registry
- [x] Add confidence scoring algorithm (multi-factor weighted)
- [x] Integrate patterns with existing governance workflow

### Secondary
- [x] Add bulk approval CLI commands
- [x] Dashboard pattern review interface
- [x] Initial set of 10-15 core detectors

---

## Scope

### In Scope
- Detector interface and registry in Go
- Confidence scoring with frequency, consistency, spread, age factors
- Pattern storage in memory.db (new table)
- Patterns automatically create Learnings with `authority: proposed`
- CLI commands: `palace patterns scan`, `palace patterns list`, `palace patterns approve`
- Bulk approval with `--bulk --min-confidence` flags
- Dashboard patterns view

### Out of Scope
- Full 101 detectors from Drift (future sprints)
- Contract detection (Sprint 2)
- LSP integration (Sprint 3)
- Semantic detection (requires LLM, future)

---

## Architecture

### New Packages

```
apps/cli/internal/
├── patterns/
│   ├── detector.go          # Detector interface & registry
│   ├── confidence.go        # Confidence scoring algorithm
│   ├── engine.go            # Pattern detection orchestration
│   ├── store.go             # Pattern persistence
│   └── detectors/           # Built-in detectors
│       ├── api/
│       │   ├── response_envelope.go
│       │   ├── error_handling.go
│       │   └── route_structure.go
│       ├── structural/
│       │   ├── file_naming.go
│       │   └── import_order.go
│       └── types/
│           ├── interface_naming.go
│           └── type_exports.go
```

### Database Schema

```sql
-- Add to memory.db migrations

CREATE TABLE patterns (
    id TEXT PRIMARY KEY,
    category TEXT NOT NULL,
    subcategory TEXT,
    name TEXT NOT NULL,
    description TEXT,
    detector_id TEXT NOT NULL,
    confidence REAL DEFAULT 0.0,
    frequency_score REAL DEFAULT 0.0,
    consistency_score REAL DEFAULT 0.0,
    spread_score REAL DEFAULT 0.0,
    age_score REAL DEFAULT 0.0,
    status TEXT DEFAULT 'discovered',  -- discovered, approved, ignored
    authority TEXT DEFAULT 'proposed',
    learning_id TEXT REFERENCES learnings(id),  -- Link to learning when approved
    first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE pattern_locations (
    id TEXT PRIMARY KEY,
    pattern_id TEXT REFERENCES patterns(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    line_start INTEGER NOT NULL,
    line_end INTEGER,
    snippet TEXT,
    is_outlier BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_patterns_category ON patterns(category);
CREATE INDEX idx_patterns_status ON patterns(status);
CREATE INDEX idx_patterns_confidence ON patterns(confidence);
CREATE INDEX idx_pattern_locations_pattern ON pattern_locations(pattern_id);
CREATE INDEX idx_pattern_locations_file ON pattern_locations(file_path);
```

### Core Types

```go
// internal/patterns/detector.go

type DetectionContext struct {
    File       *analysis.ParsedFile
    Index      *index.Index
    AllFiles   []*analysis.ParsedFile  // For cross-file patterns
}

type Detector interface {
    // Metadata
    ID() string
    Category() string
    Subcategory() string
    Name() string
    Description() string

    // Detection
    Detect(ctx *DetectionContext) (*DetectionResult, error)

    // Languages this detector supports
    Languages() []string
}

type DetectionResult struct {
    Locations   []Location
    Outliers    []Location
    Confidence  ConfidenceFactors
    Metadata    map[string]any
}

type Location struct {
    FilePath  string
    LineStart int
    LineEnd   int
    Snippet   string
}

type ConfidenceFactors struct {
    Frequency   float64  // 0.0-1.0: How often pattern appears
    Consistency float64  // 0.0-1.0: Uniformity of implementations
    Spread      float64  // 0.0-1.0: Number of files using it
    Age         float64  // 0.0-1.0: How long pattern existed
}

func (cf ConfidenceFactors) Score() float64 {
    return cf.Frequency*0.30 +
           cf.Consistency*0.30 +
           cf.Spread*0.25 +
           cf.Age*0.15
}
```

```go
// internal/patterns/registry.go

type Registry struct {
    detectors map[string]Detector
    mu        sync.RWMutex
}

func NewRegistry() *Registry {
    r := &Registry{
        detectors: make(map[string]Detector),
    }
    r.registerBuiltins()
    return r
}

func (r *Registry) Register(d Detector) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.detectors[d.ID()] = d
}

func (r *Registry) Get(id string) (Detector, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    d, ok := r.detectors[id]
    return d, ok
}

func (r *Registry) ByCategory(category string) []Detector {
    r.mu.RLock()
    defer r.mu.RUnlock()
    var result []Detector
    for _, d := range r.detectors {
        if d.Category() == category {
            result = append(result, d)
        }
    }
    return result
}
```

---

## Tasks

### Week 1: Core Infrastructure ✅

#### 1.1 Database Migration
- [x] Create migration v8 with patterns and pattern_locations tables
- [x] Add indexes for performance
- [x] Update memory.Store with pattern methods
- [x] Write migration tests

#### 1.2 Detector Interface & Registry
- [x] Define Detector interface
- [x] Implement Registry with thread-safe registration
- [x] Add detector discovery (auto-register built-ins)
- [x] Write registry tests

#### 1.3 Confidence Scoring
- [x] Implement ConfidenceFactors struct
- [x] Create scoring algorithm with weighted factors
- [x] Add threshold constants (High/Medium/Low/Uncertain)
- [x] Write scoring tests with edge cases

### Week 2: Detection Engine ✅

#### 2.1 Engine Implementation
- [x] Create Engine struct that orchestrates detection
- [x] Implement file iteration with parallel processing
- [x] Add pattern deduplication logic
- [x] Integrate with existing index/analysis packages
- [x] Write engine tests

#### 2.2 Pattern Store
- [x] CRUD operations for patterns
- [x] Location management (add, query, remove)
- [x] Status transitions (discovered → approved/ignored)
- [x] Learning linkage on approval
- [x] Write store tests

### Week 3: CLI Commands ✅

#### 3.1 Pattern Scan Command
- [x] `palace patterns scan` - Run all detectors
- [x] `palace patterns scan --category api` - Filter by category
- [x] `palace patterns scan --detector api/response-envelope` - Single detector
- [x] Progress indication for large codebases
- [x] JSON output option

#### 3.2 Pattern List Command
- [x] `palace patterns list` - Show all patterns
- [x] `palace patterns list --status discovered` - Filter by status
- [x] `palace patterns list --min-confidence 0.8` - Filter by confidence
- [x] Table output with confidence indicators

#### 3.3 Pattern Approve/Ignore Commands
- [x] `palace patterns approve <id>` - Approve single pattern
- [x] `palace patterns ignore <id>` - Ignore single pattern
- [x] `palace patterns approve --bulk --min-confidence 0.95` - Bulk approve
- [x] `palace patterns review` - Interactive review mode
- [x] Dry-run flag for bulk operations

### Week 4: Initial Detectors + Dashboard ✅

#### 4.1 Core Detectors (11 implemented)
- [x] `go_error_handling` - Go error handling patterns
- [x] `go_naming` - Go naming conventions
- [x] `go_imports` - Go import organization
- [x] `go_function_length` - Function length patterns
- [x] `go_comments` - Comment patterns
- [x] `go_tests` - Test file patterns
- [x] `go_logging` - Logging patterns
- [x] `go_config` - Configuration patterns
- [x] `go_api` - API patterns
- [x] `go_di` - Dependency injection patterns
- [x] `go_file_organization` - File organization patterns

#### 4.2 Dashboard Integration
- [x] Add Patterns view to Angular dashboard
- [x] Pattern list with filters (category, status, confidence)
- [x] Bulk selection with checkbox
- [x] Approve/Ignore actions
- [x] Pattern detail view with locations
- [x] Create Learning toggle on approve

### Week 5: Integration & Polish ✅

#### 5.1 Governance Integration
- [x] Approved patterns auto-create Learnings
- [x] Learning inherits pattern metadata
- [x] Link back from Learning to Pattern
- [x] Respect governance authority levels
- [x] ApprovePatternWithLearning and BulkApprovePatternsWithLearnings

#### 5.2 MCP Tools
- [x] `patterns_get` - Query patterns with filters
- [x] `pattern_show` - Show pattern details
- [x] `pattern_approve` - Approve patterns
- [x] `pattern_ignore` - Ignore patterns
- [x] `pattern_stats` - Get pattern statistics

#### 5.3 Testing & Documentation
- [x] Integration tests for full flow
- [x] Update CLI help text
- [x] Add patterns section to docs
- [x] Update CHANGELOG

---

## Definition of Done

- [x] All database migrations applied cleanly
- [x] Detector registry with 10+ detectors (11 implemented)
- [x] Confidence scoring with configurable weights
- [x] CLI commands: scan, list, approve, ignore, show
- [x] Bulk approval working with confidence threshold
- [x] Dashboard patterns view functional
- [x] Governance integration: patterns → learnings
- [x] MCP tools exposed and working
- [x] All existing tests passing
- [x] New tests for patterns package (>70% coverage)
- [x] Documentation updated

---

## Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Performance on large codebases | High | Medium | Parallel processing, incremental detection |
| False positive patterns | Medium | High | Conservative confidence thresholds, manual review |
| Detector complexity | Medium | Medium | Start with simple AST-based detectors |
| Dashboard scope creep | Low | Medium | MVP interface, iterate later |

---

## Dependencies

### Internal
- `internal/analysis` - Tree-sitter parsing (existing)
- `internal/index` - Code index (existing)
- `internal/memory` - SQLite storage (existing)
- `internal/model` - Data structures (existing)

### External
- None new (uses existing Tree-sitter)

---

## Notes

- Start with AST-based detection only (no semantic/LLM)
- Confidence algorithm weights can be tuned later
- Focus on Go and TypeScript detectors first (Mind Palace's stack)
- Dashboard can be minimal - bulk approve is key UX
