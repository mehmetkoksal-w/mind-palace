# Governance Layer Implementation - Handoff Summary

## Current State

**Branch:** `plc-002`
**Last Phase Completed:** Phase 3 of 5
**All Tests:** PASSING

---

## What Was Implemented (Phase 3)

### Goal
Enforce governance by MCP mode. Direct-write only in human mode. Agent mode cannot bypass proposals.

### New Files Created

| File | Purpose |
|------|---------|
| `apps/cli/internal/memory/audit.go` | Audit log struct, CRUD operations, action types |
| `apps/cli/internal/butler/mcp_tools_governance.go` | `store_direct`, `approve`, `reject` MCP tool handlers |

### Schema Changes (Migration V6)

Added to `apps/cli/internal/memory/schema.go`:

```sql
CREATE TABLE audit_log (
    id TEXT PRIMARY KEY,
    action TEXT NOT NULL,           -- 'direct_write', 'approve', 'reject'
    actor_type TEXT NOT NULL,       -- 'human', 'agent'
    actor_id TEXT DEFAULT '',       -- Optional identifier (username, session ID)
    target_id TEXT NOT NULL,        -- ID of affected record
    target_kind TEXT NOT NULL,      -- 'decision', 'learning', 'proposal'
    details TEXT DEFAULT '{}',      -- JSON details about the action
    created_at TEXT NOT NULL
);

CREATE INDEX idx_audit_log_action ON audit_log(action);
CREATE INDEX idx_audit_log_target ON audit_log(target_id);
CREATE INDEX idx_audit_log_created ON audit_log(created_at DESC);
CREATE INDEX idx_audit_log_actor ON audit_log(actor_type, actor_id);
```

### Modified Files

| File | Changes |
|------|---------|
| `apps/cli/internal/butler/mcp.go` | Added `MCPMode` enum, mode field, tool filtering in `handleToolsList` and `handleToolsCall` |
| `apps/cli/internal/butler/mcp_tools_list.go` | Added tool definitions for `store_direct`, `approve`, `reject` |
| `apps/cli/internal/cli/commands/serve.go` | Added `--mode` flag (agent/human), mode validation |
| `apps/cli/internal/memory/schema.go` | Added Migration V6 for `audit_log` table |
| `apps/cli/internal/memory/decision_test.go` | Updated schema version test to expect v6 |
| `apps/cli/internal/butler/mcp_test.go` | Added 8 new tests for mode-based access |

### Key Implementation Details

**MCPMode enum (`mcp.go`):**
```go
type MCPMode string

const (
    MCPModeAgent MCPMode = "agent"  // Restricted: no admin tools
    MCPModeHuman MCPMode = "human"  // Full access: admin tools
)

var adminOnlyTools = map[string]bool{
    "store_direct": true,
    "approve":      true,
    "reject":       true,
}
```

**Tool filtering (two enforcement points):**
1. `handleToolsList` - Filters admin-only tools from `tools/list` in agent mode
2. `handleToolsCall` - Returns explicit error if agent calls admin-only tool

**Audit logging:**
- All governance actions (`direct_write`, `approve`, `reject`) create audit entries
- Records actor type (`human`/`agent`), actor ID, target details
- JSON details field for extensibility

### Tool Access Matrix (Implemented)

| Tool | Agent Mode | Human Mode |
|------|------------|------------|
| `explore`, `recall`, `brief` | Yes | Yes |
| `store` | Yes (creates proposal) | Yes (creates proposal) |
| `store_direct` | **No** (filtered + rejected) | Yes |
| `approve`, `reject` | **No** (filtered + rejected) | Yes |

### Usage

```bash
# Start in restricted agent mode (default - secure by default)
palace serve

# Start with explicit agent mode
palace serve --mode agent

# Start in human mode with full access (admin tools available)
palace serve --mode human
```

### Critical Invariant Maintained

**Agent mode cannot access `store_direct`, `approve`, or `reject` tools.**

Enforced at:
- **Discovery:** Tools not returned in `tools/list` response
- **Execution:** Explicit error `-32602: Tool "store_direct" not available in agent mode`

---

## Previous Phases Summary

### Phase 1 (Complete)
- Added `authority` field to decisions/learnings
- Created `memory/authority.go` with `IsAuthoritative()` helper
- Backfilled existing records with `legacy_approved`
- Migration V4

### Phase 2 (Complete)
- Created `proposals` table for agent write path
- MCP `store` tool creates proposals for decisions/learnings
- CLI commands: `proposals`, `approve`, `reject`
- Promotion logic links proposals to created records
- Migration V5

### Phase 3 (Complete)
- MCP mode enum (`agent`/`human`)
- Tool filtering by mode
- `store_direct` tool (human mode only)
- `approve`/`reject` MCP tools (human mode only)
- Audit logging for all governance actions
- Migration V6

---

## What Comes Next (Phase 4)

**Goal:** Deterministic, bounded query for "what is true."

### Key Tasks

1. **Scope expansion centralization:**
   - Create `memory/scope.go` with `ExpandScope()` function
   - File scope → Room scope → Palace scope chain

2. **Bounded query configuration:**
   ```go
   type AuthoritativeQueryConfig struct {
       MaxDecisions   int  // Default: 10
       MaxLearnings   int  // Default: 10
       MaxContentLen  int  // Default: 500 chars per item
   }
   ```

3. **Create authoritative state views (Migration V7):**
   - Views that filter by `AuthoritativeValues()` (from `authority.go`)
   - No hard-coded authority lists in SQL

4. **Update `butler_context.go`:**
   - Bounded query builder using config
   - Deterministic truncation

### Files to Create/Modify

| File | Purpose |
|------|---------|
| `memory/scope.go` | **New** - Scope expansion logic |
| `memory/schema.go` | Migration V7 for authoritative views |
| `butler/butler_context.go` | Bounded query builder |

### Acceptance Criteria
1. Scope expansion is deterministic
2. Views use `AuthoritativeValues()`, not hard-coded list
3. Query bounded by max items and max chars
4. No token heuristics

---

## Reference Documents

- `IMPLEMENTATION_PLAN_V2.1.md` - Full 5-phase implementation plan
- `ALIGNMENT_ASSESSMENT.md` - Gap analysis and requirements

---

## Running Tests

```bash
cd apps/cli
go test ./internal/... ./pkg/... -count=1
```

All tests should pass. Schema version is now 6.

---

## Quick Start for Phase 4

1. Read `IMPLEMENTATION_PLAN_V2.1.md` Phase 4 section
2. Create `apps/cli/internal/memory/scope.go` with `ExpandScope()` function
3. Add Migration V7 with authoritative state views
4. Update `butler_context.go` for bounded queries
5. Add tests for scope expansion and bounded queries
6. Run all tests to verify
