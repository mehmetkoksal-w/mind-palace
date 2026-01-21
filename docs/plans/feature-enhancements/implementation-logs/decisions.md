# Architectural Decisions Log

> Record of key decisions made during the feature enhancement implementation

---

## Decision Format

```markdown
### DEC-XXX: [Title]

**Date**: YYYY-MM-DD
**Status**: Proposed | Accepted | Superseded | Deprecated
**Sprint**: N

**Context**: Why this decision was needed

**Decision**: What we decided

**Consequences**: What this means for the implementation

**Alternatives Considered**: What else we looked at
```

---

## Decisions

### DEC-001: Patterns Become Learnings

**Date**: 2025-01-21
**Status**: Accepted
**Sprint**: 1

**Context**: Drift treats patterns as standalone entities with approve/ignore status. Mind Palace already has a rich knowledge model (Ideas, Decisions, Learnings) with governance workflow. We need to decide how detected patterns integrate.

**Decision**: Detected patterns will automatically create Learnings when approved:
- Pattern detected → stored in `patterns` table with `status: discovered`
- User approves pattern → creates Learning with `authority: proposed`
- Learning goes through normal governance (if enabled)
- Pattern links to Learning via `learning_id`

**Consequences**:
- Patterns benefit from existing governance workflow
- Learnings can reference pattern for context
- Confidence decay applies to pattern-derived learnings
- Search includes patterns via learning content

**Alternatives Considered**:
1. **Patterns as separate knowledge type**: More flexibility but duplicates governance logic
2. **Patterns replace learnings**: Too disruptive to existing model
3. **No integration**: Misses opportunity to leverage existing infrastructure

---

### DEC-002: Confidence Scoring Weights

**Date**: 2025-01-21
**Status**: Accepted
**Sprint**: 1

**Context**: Drift uses multi-factor confidence scoring. We need to define the weights for Mind Palace.

**Decision**: Use weighted factors:
- Frequency: 30% (how often pattern appears)
- Consistency: 30% (uniformity of implementations)
- Spread: 25% (number of files using it)
- Age: 15% (how long pattern existed)

Thresholds:
- High: ≥ 0.85
- Medium: 0.70 - 0.84
- Low: 0.50 - 0.69
- Uncertain: < 0.50

**Consequences**:
- Matches Drift's approach for consistency
- Weights configurable in `palace.jsonc` if needed later
- Bulk approve defaults to 0.95 threshold (very conservative)

**Alternatives Considered**:
1. **Equal weights**: Simpler but less nuanced
2. **ML-based scoring**: Too complex for v1, consider later

---

### DEC-003: SQLite for Pattern Storage

**Date**: 2025-01-21
**Status**: Accepted
**Sprint**: 1

**Context**: Drift uses JSON files in `.drift/patterns/`. Mind Palace uses SQLite for memory.db. Need to decide storage strategy.

**Decision**: Store patterns in `memory.db` SQLite database:
- New `patterns` table with full metadata
- New `pattern_locations` table for file references
- Leverages existing SQLite infrastructure
- Enables SQL queries for filtering/aggregation

**Consequences**:
- Consistent with Mind Palace architecture
- Better query performance for large pattern sets
- Requires database migration (v8)
- Transaction safety for bulk operations

**Alternatives Considered**:
1. **JSON files like Drift**: Simpler but inconsistent with architecture
2. **Separate patterns.db**: Unnecessary complexity
3. **Embedded in index (palace.db)**: Mixing concerns

---

### DEC-004: Start with AST-Only Detection

**Date**: 2025-01-21
**Status**: Accepted
**Sprint**: 1

**Context**: Drift has AST-based, regex-based, semantic-based, and custom detectors. We need to decide scope for Sprint 1.

**Decision**: Sprint 1 implements AST-based detection only:
- Leverage existing Tree-sitter integration
- Deterministic results (aligns with Mind Palace philosophy)
- No LLM dependency for basic patterns

**Consequences**:
- Simpler implementation
- Predictable performance
- Some patterns (semantic) deferred to later sprints
- Can add regex-based easily if needed

**Alternatives Considered**:
1. **All detection types**: Too much scope for Sprint 1
2. **Regex-only**: Too limited for structural patterns

---

### DEC-005: Detector Registry as Singleton

**Date**: 2025-01-21
**Status**: Accepted
**Sprint**: 1

**Context**: Need to manage detector lifecycle and discovery.

**Decision**: Single global registry initialized at startup:
- Built-in detectors auto-registered
- Thread-safe with RWMutex
- Detectors are stateless (context passed per detection)
- No plugin system for v1 (built-ins only)

**Consequences**:
- Simple, predictable initialization
- Easy to test with mock registry
- Future: plugin system can extend registry

**Alternatives Considered**:
1. **Per-scan registry**: Unnecessary overhead
2. **Plugin-based from start**: Over-engineering for v1

---

### DEC-006: Parallel Detection with Worker Pool

**Date**: 2025-01-21
**Status**: Accepted
**Sprint**: 1

**Context**: Large codebases need efficient pattern detection.

**Decision**: Use worker pool for parallel file processing:
- Default workers = CPU cores
- Configurable via `palace.jsonc`
- Share parsed ASTs from index when possible
- Aggregate results with mutex protection

**Consequences**:
- Good performance on multi-core machines
- Memory usage scales with workers
- Need careful result aggregation

**Alternatives Considered**:
1. **Sequential processing**: Too slow for large codebases
2. **Goroutine per file**: Uncontrolled concurrency

---

### DEC-007: TypeSchema as Universal Type Representation

**Date**: 2026-01-21
**Status**: Accepted
**Sprint**: 2

**Context**: FE-BE contract detection needs to compare types across languages (Go structs, TypeScript interfaces, Python Pydantic models). Need a common representation.

**Decision**: Use JSON Schema-inspired TypeSchema struct:
- Type field: object, array, string, number, integer, boolean, null, any
- Recursive Properties map for objects
- Items for arrays
- Required tracking via IsRequired() method
- Nullable and Enum support

**Consequences**:
- Language-agnostic type comparison
- Easy serialization to JSON for storage
- Recursive comparison algorithm handles nested types
- Type extractors convert language-specific types to TypeSchema

**Alternatives Considered**:
1. **Language-specific comparisons**: Too complex, N^2 comparisons
2. **JSON Schema directly**: Too complex, more than needed
3. **Protocol Buffers style**: Overkill for REST API contracts

---

### DEC-008: Tree-sitter for All Extractors

**Date**: 2026-01-21
**Status**: Accepted
**Sprint**: 2

**Context**: Need to extract endpoints and API calls from multiple languages (Go, TypeScript, Python).

**Decision**: Use Tree-sitter AST parsing for all extractors:
- Consistent parsing approach across languages
- Reuse existing Tree-sitter infrastructure from index
- Query-based pattern matching for each extractor
- ExtractEndpointsFromContent/ExtractCallsFromContent interface

**Consequences**:
- Fast, reliable parsing
- Language grammars already available
- Some edge cases (template strings, dynamic URLs) hard to handle
- Framework detection via AST patterns

**Alternatives Considered**:
1. **Regex-based extraction**: Too fragile, can't handle nesting
2. **Language-specific parsers**: More maintenance burden
3. **LLM-based extraction**: Non-deterministic, slow

---

### DEC-009: Endpoint Matcher with Confidence Scoring

**Date**: 2026-01-21
**Status**: Accepted
**Sprint**: 2

**Context**: Frontend URLs may not exactly match backend routes (e.g., `/api/users/123` vs `/api/users/:id`).

**Decision**: Implement multi-factor endpoint matcher:
- Normalize paths (`:id`, `{id}` → parameter regex)
- Method matching (case-insensitive)
- Confidence based on: method match, path segments, parameter alignment
- Support for both `:param` (Express) and `{param}` (FastAPI/OpenAPI) formats

**Consequences**:
- Handles path parameter variations
- Confidence helps filter uncertain matches
- Dynamic URLs flagged as low confidence
- Can match template literals with variables

**Alternatives Considered**:
1. **Exact string match only**: Too strict, misses valid contracts
2. **Regex for all paths**: Complex, error-prone
3. **LLM for matching**: Overkill for URL matching

---

### DEC-010: Five Mismatch Types for Contract Validation

**Date**: 2026-01-21
**Status**: Accepted
**Sprint**: 2

**Context**: Need to categorize type mismatches between FE and BE for actionable feedback.

**Decision**: Define 5 mismatch types with severity:
- `missing_in_frontend` (warning): BE has field FE doesn't expect
- `missing_in_backend` (error): FE expects field BE doesn't provide
- `type_mismatch` (error): Different types for same field
- `optionality_mismatch` (warning): Required vs optional difference
- `nullability_mismatch` (warning): Nullable vs non-nullable

**Consequences**:
- Clear categorization for developers
- Errors require action, warnings are informational
- Field path tracking (e.g., "user.profile.email")
- Human-readable descriptions generated automatically

**Alternatives Considered**:
1. **Single "mismatch" type**: Not actionable enough
2. **More granular types**: Over-classification
3. **Severity per-instance**: Too complex to configure

---

## Template for New Decisions

Copy this template when adding new decisions:

```markdown
### DEC-XXX: [Title]

**Date**: YYYY-MM-DD
**Status**: Proposed
**Sprint**: N

**Context**: [Why this decision is needed]

**Decision**: [What we decided]

**Consequences**: [What this means]

**Alternatives Considered**:
1. [Option 1]: [Why not chosen]
2. [Option 2]: [Why not chosen]
```
