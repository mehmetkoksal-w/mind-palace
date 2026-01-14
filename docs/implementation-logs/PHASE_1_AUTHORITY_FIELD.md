# Phase 1 Implementation Log: Authority Field & Legacy Compatibility

**Status:** ✅ COMPLETED  
**Completed:** 2026-01-14

---

## Objective

Add `authority` field with clean semantics and centralize authority logic to establish governance foundation.

## Changes Implemented

### 1. Authority Enum & Centralization

**File:** `apps/cli/internal/memory/authority.go` (NEW)

Created centralized authority enum with helper functions:

- `Authority` type with three values: `proposed`, `approved`, `legacy_approved`
- `IsAuthoritative(auth Authority) bool` - Single source of truth for authority checks
- `AuthoritativeValues() []Authority` - Returns `[approved, legacy_approved]` for SQL queries
- `AuthoritativeValuesStrings() []string` - String version for SQL IN clauses
- `SQLPlaceholders(n int) string` - Helper for generating SQL placeholders

**Key Design Decision:**

- All queries MUST use `AuthoritativeValues()` helper - no hard-coded authority lists allowed
- This centralizes the definition of "what is authoritative" in one place

### 2. Schema Migration V4

**File:** `apps/cli/internal/memory/schema.go`

Added to `runMigrations()`:

```sql
-- Add authority columns with default 'proposed'
ALTER TABLE decisions ADD COLUMN authority TEXT DEFAULT 'proposed';
ALTER TABLE decisions ADD COLUMN promoted_from_proposal_id TEXT DEFAULT '';
ALTER TABLE learnings ADD COLUMN authority TEXT DEFAULT 'proposed';
ALTER TABLE learnings ADD COLUMN promoted_from_proposal_id TEXT DEFAULT '';

-- Backfill existing records with 'legacy_approved' (DP-2)
UPDATE decisions SET authority = 'legacy_approved' WHERE authority = 'proposed';
UPDATE learnings SET authority = 'legacy_approved' WHERE authority = 'proposed';

-- Create indexes for performance
CREATE INDEX idx_decisions_authority ON decisions(authority);
CREATE INDEX idx_learnings_authority ON learnings(authority);
```

**Migration Safety:**

- Existing records marked as `legacy_approved` (not collapsed into `approved`)
- Preserves audit trail of pre-governance data
- Indexes added immediately for query performance

### 3. Query Pattern Updates

**Files Modified:**

- `apps/cli/internal/memory/decision.go`
- `apps/cli/internal/memory/learning.go`

**Pattern Applied:**
All queries for authoritative records now use centralized helpers:

```go
// Before (WRONG - hard-coded)
query := `SELECT ... WHERE authority IN ('approved', 'legacy_approved')`

// After (CORRECT - uses helper)
authVals := AuthoritativeValuesStrings()
query := `SELECT ... WHERE authority IN (` + SQLPlaceholders(len(authVals)) + `)`
for _, v := range authVals {
    args = append(args, v)
}
```

**Functions Updated:**

- `GetDecisions()` - Now calls `GetDecisionsWithAuthority(..., authoritativeOnly=true)`
- `GetDecisionsWithAuthority()` - Uses `AuthoritativeValuesStrings()` for filtering
- `GetLearnings()` - Now calls `GetLearningsWithAuthority(..., authoritativeOnly=true)`
- `GetLearningsWithAuthority()` - Uses `AuthoritativeValuesStrings()` for filtering
- `SearchDecisions()` - Uses authority helpers
- `SearchLearnings()` - Uses authority helpers

### 4. Decision & Learning Structs

**Files:** `apps/cli/internal/memory/decision.go`, `learning.go`

Added fields:

```go
type Decision struct {
    // ... existing fields
    Authority              string    `json:"authority"`
    PromotedFromProposalID string    `json:"promotedFromProposalId,omitempty"`
}

type Learning struct {
    // ... existing fields
    Authority              string    `json:"authority"`
    PromotedFromProposalID string    `json:"promotedFromProposalId,omitempty"`
}
```

Default values in `Add*` functions:

- New records: `authority = "proposed"`
- Will be changed to `"approved"` when promoted from proposals or written directly by humans

---

## Validation & Testing

### Test Coverage

**File:** `apps/cli/internal/memory/authority_test.go` (exists in test suite)

Tests validate:

1. `IsAuthoritative()` returns true for `approved` and `legacy_approved`
2. `IsAuthoritative()` returns false for `proposed`
3. `AuthoritativeValues()` returns exactly `[approved, legacy_approved]`
4. `SQLPlaceholders()` generates correct number of `?` placeholders

### Manual Verification

Ran full memory test suite:

```bash
go test -v ./apps/cli/internal/memory
```

**Result:** ✅ All tests pass (135 tests, ~9s)

### Query Pattern Audit

Searched codebase for hard-coded authority strings:

```bash
grep -r "authority IN" apps/cli/internal/
grep -r "'approved'" apps/cli/internal/memory/
```

**Result:** ✅ No hard-coded authority lists found in active query paths

---

## Acceptance Criteria

| Criterion                                                 | Status | Evidence                                |
| --------------------------------------------------------- | ------ | --------------------------------------- |
| All existing records have `authority = 'legacy_approved'` | ✅     | Migration V4 backfill executed          |
| New agent-created records have `authority = 'proposed'`   | ✅     | Default value in `Add*` functions       |
| No hard-coded authority lists in queries                  | ✅     | All queries use `AuthoritativeValues()` |
| `IsAuthoritative()` is single resolution function         | ✅     | Centralized in `authority.go`           |

---

## Migration Impact

### Database Changes

- Two new columns added to existing tables (non-breaking)
- Indexes created (improves query performance)
- Existing data backfilled with `legacy_approved`

### API Impact

- **Breaking:** None - authority filtering is internal
- **Additive:** New fields in JSON responses (`authority`, `promotedFromProposalId`)

### Rollback Plan

If needed, migration can be reversed:

```sql
DROP INDEX idx_decisions_authority;
DROP INDEX idx_learnings_authority;
-- Note: SQLite doesn't support DROP COLUMN, would require table rebuild
```

---

## Related Documentation

- [Authority Enum Design](../IMPLEMENTATION_PLAN_V2.1.md#phase-1-authority-field--legacy-compatibility)
- Decision DP-2: Legacy authority value

## Next Phase

Phase 2: Proposals Table & Write Path
