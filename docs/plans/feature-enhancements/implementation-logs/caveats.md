# Known Caveats & Limitations

> Issues, workarounds, and limitations discovered during implementation

---

## Format

```markdown
### CAV-XXX: [Title]

**Severity**: Low | Medium | High
**Sprint**: N
**Status**: Open | Mitigated | Resolved

**Description**: What the issue is

**Impact**: How it affects users/development

**Workaround**: Temporary solution if any

**Resolution Plan**: How we plan to fix it
```

---

## Caveats

### CAV-001: Tree-sitter Language Coverage

**Severity**: Medium
**Sprint**: 1
**Status**: Open

**Description**: Mind Palace supports 20+ languages via Tree-sitter, but not all languages have equal query support for pattern detection.

**Impact**: Some detectors may not work for all languages. API response patterns work for Go/TypeScript but may miss Python/Ruby edge cases.

**Workaround**: Detectors declare supported languages. Unsupported languages are skipped gracefully.

**Resolution Plan**: Add language-specific detector variants as needed. Community contributions welcome.

---

### CAV-002: Cross-File Pattern Detection Performance

**Severity**: Medium
**Sprint**: 1
**Status**: Open

**Description**: Some patterns require analyzing relationships across multiple files (e.g., "all API handlers follow this structure"). This can be expensive.

**Impact**: Large codebases may see slower pattern detection when cross-file detectors run.

**Workaround**:
- Cross-file detection is opt-in via detector config
- Use index relationships instead of re-parsing
- Incremental detection only processes changed files

**Resolution Plan**: Optimize cross-file queries using index. Consider caching intermediate results.

---

### CAV-003: No Semantic Detection in Sprint 1

**Severity**: Low
**Sprint**: 1
**Status**: Open

**Description**: Drift's semantic detection (understanding code meaning) requires LLM integration. Sprint 1 is AST-only.

**Impact**: Some nuanced patterns that require understanding intent won't be detected.

**Workaround**: Users can manually create learnings for semantic patterns.

**Resolution Plan**: Add semantic detection in Sprint 4+ when LLM integration is scoped.

---

### CAV-004: Confidence Decay Not Applied to Patterns

**Severity**: Low
**Sprint**: 1
**Status**: Open

**Description**: Mind Palace has confidence decay for learnings. Patterns have static confidence based on detection.

**Impact**: Pattern confidence doesn't decrease if pattern isn't seen in recent scans.

**Workaround**: Linked learnings inherit pattern confidence at creation time.

**Resolution Plan**: Consider adding pattern freshness factor in future sprint.

---

## Template for New Caveats

```markdown
### CAV-XXX: [Title]

**Severity**: Low | Medium | High
**Sprint**: N
**Status**: Open

**Description**: [What the issue is]

**Impact**: [How it affects users/development]

**Workaround**: [Temporary solution if any]

**Resolution Plan**: [How we plan to fix it]
```
