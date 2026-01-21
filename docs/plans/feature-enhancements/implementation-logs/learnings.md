# Implementation Learnings

> Insights and lessons learned during feature enhancement implementation

---

## Format

```markdown
### LRN-XXX: [Title]

**Date**: YYYY-MM-DD
**Sprint**: N
**Category**: Architecture | Performance | Testing | Integration | Other

**Learning**: What we learned

**Context**: How we discovered this

**Application**: How this applies to future work
```

---

## Learnings

### LRN-001: Drift's Detector Design is Clean

**Date**: 2025-01-21
**Sprint**: Pre-Sprint (Audit)
**Category**: Architecture

**Learning**: Drift's detector interface is well-designed for extensibility. The separation of detector metadata (ID, category, name) from detection logic makes it easy to register and query detectors.

**Context**: Analyzed `packages/detectors/src/index.ts` and detector implementations.

**Application**: Adopt similar interface design for Mind Palace. Keep detectors stateless with context passed per invocation.

---

### LRN-002: Confidence Scoring Needs Multiple Factors

**Date**: 2025-01-21
**Sprint**: Pre-Sprint (Audit)
**Category**: Architecture

**Learning**: Single-factor confidence (just frequency) produces poor results. Drift's multi-factor approach (frequency, consistency, spread, age) produces more meaningful scores.

**Context**: Reviewed Drift's confidence scoring implementation.

**Application**: Implement weighted multi-factor scoring. Make weights configurable for tuning.

---

### LRN-003: Bulk Operations Need Dry-Run

**Date**: 2025-01-21
**Sprint**: Pre-Sprint (Audit)
**Category**: Integration

**Learning**: Bulk approve operations can affect many patterns. Users need to preview before committing.

**Context**: Drift provides `--dry-run` flag for bulk operations.

**Application**: All bulk CLI commands should support `--dry-run` to show what would change without making changes.

---

### LRN-004: Pattern Locations Are First-Class

**Date**: 2025-01-21
**Sprint**: Pre-Sprint (Audit)
**Category**: Architecture

**Learning**: Storing exact locations (file, line, snippet) for each pattern instance is crucial for:
- Showing users where patterns exist
- Detecting outliers
- Enabling navigation from violations

**Context**: Drift's `pattern_locations` separate from patterns.

**Application**: Create `pattern_locations` table with proper indexes. Store snippets for context.

---

## Template for New Learnings

```markdown
### LRN-XXX: [Title]

**Date**: YYYY-MM-DD
**Sprint**: N
**Category**: Architecture | Performance | Testing | Integration | Other

**Learning**: [What we learned]

**Context**: [How we discovered this]

**Application**: [How this applies to future work]
```
