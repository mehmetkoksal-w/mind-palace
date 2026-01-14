# Mind Palace Governance Implementation Plan v2.1

## Decisions (Confirmed)

**DP-1: Human direct-write behavior**

- Humans may write `approved` directly via CLI using `--direct` flag
- Only available in human mode (`palace serve --mode human` or CLI invocation)
- Always audited with `actor_type: human`, `action: direct_write`
- **Not available via MCP in agent mode** - tool does not exist in agent tool list

**DP-2: Legacy authority value**

- Use `legacy_approved` as backfill value for all existing records
- Do not collapse into `approved` - preserves audit trail of pre-governance data

---

## Phase 1: Authority Field & Legacy Compatibility

**Status:** ✅ DONE

**Goal:** Add `authority` field with clean semantics. Centralize authority logic.

### Adjustments Applied

- **Authority enum centralization:** Define `IsAuthoritative()` helper as single source of truth for authority resolution.

### Schema Changes (Migration V4)

```sql
ALTER TABLE decisions ADD COLUMN authority TEXT DEFAULT 'proposed';
ALTER TABLE decisions ADD COLUMN promoted_from_proposal_id TEXT DEFAULT '';
ALTER TABLE learnings ADD COLUMN authority TEXT DEFAULT 'proposed';
ALTER TABLE learnings ADD COLUMN promoted_from_proposal_id TEXT DEFAULT '';

-- Backfill with legacy marker (DP-2 confirmed)
UPDATE decisions SET authority = 'legacy_approved' WHERE authority = 'proposed';
UPDATE learnings SET authority = 'legacy_approved' WHERE authority = 'proposed';

CREATE INDEX idx_decisions_authority ON decisions(authority);
CREATE INDEX idx_learnings_authority ON learnings(authority);
```

### Authority Enum & Helper (Adjustment 1)

```go
// memory/authority.go - SINGLE SOURCE OF TRUTH

type Authority string

const (
    AuthorityProposed       Authority = "proposed"
    AuthorityApproved       Authority = "approved"
    AuthorityLegacyApproved Authority = "legacy_approved"
)

// ValidAuthorities is the complete set of valid authority values
var ValidAuthorities = []Authority{
    AuthorityProposed,
    AuthorityApproved,
    AuthorityLegacyApproved,
}

// IsAuthoritative returns true if the authority value represents
// trusted, human-approved state. ALL queries must use this helper.
func IsAuthoritative(auth Authority) bool {
    return auth == AuthorityApproved || auth == AuthorityLegacyApproved
}

// AuthoritativeValues returns the list for SQL IN clauses.
// Queries MUST call this, not hard-code values.
func AuthoritativeValues() []Authority {
    return []Authority{AuthorityApproved, AuthorityLegacyApproved}
}
```

### Query Pattern (All Queries Must Follow)

```go
// CORRECT - uses centralized helper
func (m *Memory) GetAuthoritativeDecisions(...) ([]Decision, error) {
    authValues := AuthoritativeValues()
    placeholders := sqlPlaceholders(len(authValues))
    query := fmt.Sprintf(`
        SELECT ... FROM decisions
        WHERE authority IN (%s) AND status = 'active'
        ...`, placeholders)
    // ...
}

// INCORRECT - hard-coded list (not allowed)
// WHERE authority IN ('approved', 'legacy_approved')  // BAD
```

### Files Changed

- `memory/schema.go` - Migration V4
- `memory/authority.go` - **New file** (centralized authority enum + helpers)
- `memory/decision.go` - Use `AuthoritativeValues()` in all queries
- `memory/learning.go` - Use `AuthoritativeValues()` in all queries

### Acceptance Criteria

1. All existing records have `authority = 'legacy_approved'`
2. New agent-created records have `authority = 'proposed'`
3. **No hard-coded authority lists in queries** - all use `AuthoritativeValues()`
4. `IsAuthoritative()` is the single resolution function

---

## Phase 2: Proposals Table & Write Path

**Status:** ✅ DONE

**Goal:** Agent/LLM writes go to proposals. Promotion creates auditable link.

### Schema Changes (Migration V5)

```sql
CREATE TABLE proposals (
    id TEXT PRIMARY KEY,
    proposed_as TEXT NOT NULL,
    content TEXT NOT NULL,
    context TEXT DEFAULT '',
    scope TEXT DEFAULT 'palace',
    scope_path TEXT DEFAULT '',
    source TEXT NOT NULL,
    session_id TEXT DEFAULT '',
    agent_type TEXT DEFAULT '',
    evidence_refs TEXT DEFAULT '{}',
    classification_confidence REAL DEFAULT 0,
    classification_signals TEXT DEFAULT '[]',
    dedupe_key TEXT DEFAULT '',
    status TEXT DEFAULT 'pending',
    reviewed_by TEXT DEFAULT '',
    reviewed_at TEXT DEFAULT '',
    review_note TEXT DEFAULT '',
    promoted_to_id TEXT DEFAULT '',
    created_at TEXT NOT NULL,
    expires_at TEXT DEFAULT '',
    archived_at TEXT DEFAULT ''
);

CREATE UNIQUE INDEX idx_proposals_dedupe ON proposals(dedupe_key) WHERE dedupe_key != '';
CREATE INDEX idx_proposals_status ON proposals(status);
CREATE INDEX idx_proposals_expires ON proposals(expires_at) WHERE expires_at != '';
```

### Promotion Logic

- Approved proposals create records with `authority = 'approved'`
- Promoted records have `promoted_from_proposal_id` set (immutable reference)
- Proposals become immutable after review (`status` changes, content does not)

### LLM Evidence Requirements

- `AutoExtract` proposals must include: `session_id`, `conversation_id`, `extractor`
- `ContradictionAutoLink` proposals must include: `source_record`, `target_record`, `confidence`, `explanation`

### Files Changed

- `memory/schema.go` - Migration V5
- `memory/proposal.go` - New file
- `butler/mcp_tools_brain.go` - `store` writes to proposals
- `butler/butler_session.go` - `autoExtractForSession` adds evidence
- `cli/commands/approve.go` - New command
- `cli/commands/proposals.go` - New command

### Acceptance Criteria

1. `store` MCP tool creates proposal (not decision/learning)
2. Proposals have `dedupe_key`, duplicates rejected
3. Promoted records have `promoted_from_proposal_id` set
4. LLM proposals have `evidence_refs` populated
5. Expired proposals auto-archive

---

## Phase 3: MCP Mode Gate & Tool Segmentation

**Status:** ✅ DONE

**Goal:** Enforce governance by mode. Direct-write only in human mode.

### Adjustments Applied

- **CLI direct-write tightening:** `--direct` explicitly documented as human-mode only, always audited, not in MCP agent mode.

### Mode Gate Design

```go
type MCPMode string

const (
    MCPModeAgent MCPMode = "agent"  // Restricted: no admin tools, no direct write
    MCPModeHuman MCPMode = "human"  // Full access: admin tools + direct write
)
```

### Tool Access Matrix

| Tool                            | Agent Mode | Human Mode | Notes                             |
| ------------------------------- | ---------- | ---------- | --------------------------------- |
| `explore`, `recall`, `brief`    | Yes        | Yes        | Read-only                         |
| `store`                         | Yes        | Yes        | Creates proposal                  |
| `session_*`                     | Yes        | Yes        | Process artifacts                 |
| `get_route`                     | Yes        | Yes        | Navigation                        |
| `store_direct`                  | **No**     | Yes        | Bypasses proposals (Adjustment 2) |
| `approve`, `reject`             | No         | Yes        | Admin                             |
| `recall_outcome`, `recall_link` | No         | Yes        | Mutates truth                     |

### Direct-Write Specification (Adjustment 2)

**`store_direct` / `--direct` flag behavior:**

1. **Availability:**

   - CLI: `palace store "..." --direct` - available when running interactively
   - MCP human mode: `store_direct` tool available
   - MCP agent mode: **Tool does not exist** - not in `tools/list`, rejected if called

2. **Audit requirements:**

   - Always creates audit entry with `action: direct_write`
   - Records `actor_type: human`
   - Captures content hash for traceability

3. **Failure behavior in agent mode:**

   - If agent calls `store_direct`: returns error `-32602: Tool "store_direct" not available in agent mode`
   - If agent passes `--direct` argument to `store`: argument ignored, proposal created normally
   - No silent fallback - explicit rejection

4. **Created record:**
   - `authority = 'approved'` (not `proposed`)
   - `promoted_from_proposal_id = ''` (empty - no proposal)
   - `source = 'human'`

### Files Changed

- `butler/mcp.go` - Mode field, tool filtering
- `butler/mcp_tools.go` - Tool access registry
- `butler/mcp_tools_brain.go` - `store_direct` implementation
- `cli/commands/serve.go` - `--mode` flag
- `cli/commands/store.go` - `--direct` flag (CLI only)
- `memory/schema.go` - Migration V6 (audit_log)
- `memory/audit.go` - New file

### Acceptance Criteria

1. `palace serve --mode agent` excludes `store_direct` and admin tools
2. `palace serve --mode human` includes all tools
3. Agent calling `store_direct` gets explicit error (not silent failure)
4. `--direct` writes always create audit entry
5. Direct-written records have `authority = 'approved'`, empty `promoted_from_proposal_id`

---

## Phase 4: Authoritative State Query Surface

**Status:** ✅ DONE

**Goal:** Deterministic, bounded query for "what is true."

### Scope Expansion

- Centralized in Go (`ExpandScope()` function)
- Views only filter by authority using `AuthoritativeValues()`
- No scope logic in SQL

### Bounded Query Config

```go
type AuthoritativeQueryConfig struct {
    MaxDecisions   int  // Default: 10
    MaxLearnings   int  // Default: 10
    MaxContentLen  int  // Default: 500 chars per item
}
```

- No token estimation - uses explicit item counts and character limits
- Deterministic truncation (first N characters + "...")

### Files Changed

- `memory/schema.go` - Migration V7 (views using `AuthoritativeValues()`)
- `memory/scope.go` - Scope expansion logic
- `butler/butler_context.go` - Bounded query builder

### Acceptance Criteria

1. Scope expansion is deterministic
2. Views use `AuthoritativeValues()`, not hard-coded list
3. Query bounded by max items and max chars
4. No token heuristics (`len/4` removed)

---

## Phase 5: Route/Polyline Query

**Status:** ✅ DONE

**Goal:** Add `get_route` MCP tool for deterministic navigation guidance.

### Adjustments Applied

- **Route output contract refinement:** Output includes `fetch_ref` field describing how to retrieve details.

### MCP Tool Contract

**Input:**

```json
{
  "intent": "understand auth flow",
  "scope": "file",
  "scopePath": "src/auth/jwt.go"
}
```

**Output (Adjustment 3):**

```json
{
  "nodes": [
    {
      "order": 1,
      "kind": "room",
      "id": "authentication",
      "reason": "Room name matches intent",
      "fetch_ref": "explore_rooms"
    },
    {
      "order": 2,
      "kind": "decision",
      "id": "d_abc123",
      "reason": "Decision content matches intent",
      "fetch_ref": "recall_decisions --id d_abc123"
    },
    {
      "order": 3,
      "kind": "file",
      "id": "src/auth/jwt.go",
      "reason": "Room entry point",
      "fetch_ref": "explore_file --path src/auth/jwt.go"
    },
    {
      "order": 4,
      "kind": "learning",
      "id": "l_xyz789",
      "reason": "High-confidence learning (85%)",
      "fetch_ref": "recall --id l_xyz789"
    }
  ],
  "meta": {
    "rule_version": "v1.0",
    "node_count": 4
  }
}
```

### Fetch Reference Mapping (Adjustment 3)

| Node Kind  | `fetch_ref` Format           |
| ---------- | ---------------------------- |
| `room`     | `explore_rooms`              |
| `decision` | `recall_decisions --id {id}` |
| `learning` | `recall --id {id}`           |
| `file`     | `explore_file --path {id}`   |

- `fetch_ref` tells the agent exactly which tool to call to get full details
- Minimal - just tool name and required argument
- Deterministic - same node kind always produces same ref pattern

### Derivation Rules (Unchanged)

1. Match intent keywords to room names/summaries
2. Include relevant decisions from scope chain
3. Include high-confidence learnings (>=0.7)
4. Max 10 nodes, deterministic ordering

### Files Changed

- `butler/butler_route.go` - New file
- `butler/mcp_tools_route.go` - New file
- `butler/mcp.go` - Register `get_route` tool

### Acceptance Criteria

1. `get_route` returns ordered list of max 10 nodes
2. Each node includes `fetch_ref` with tool invocation pattern
3. Derivation is deterministic (same input → same output)
4. Rule version included in response

---

## Summary of Adjustments in v2.1

| Adjustment                    | Phase | What Changed                                                                                       |
| ----------------------------- | ----- | -------------------------------------------------------------------------------------------------- |
| Authority enum centralization | 1     | Added `authority.go` with `IsAuthoritative()` helper; all queries must use `AuthoritativeValues()` |
| CLI direct-write tightening   | 3     | Explicit spec: `store_direct` not in agent mode, always audited, explicit error on violation       |
| Route output contract         | 5     | Added `fetch_ref` field to each node with tool invocation pattern                                  |

## Minimal Viable Governance Invariants

1. **Agents can only create proposals** - `store_direct` not available in agent mode
2. **Authority resolution is centralized** - `IsAuthoritative()` is single source of truth
3. **Promoted records link back to proposals** - `promoted_from_proposal_id`
4. **LLM outputs carry evidence** - `evidence_refs` required
5. **Direct writes are audited** - `action: direct_write` in audit log
6. **Mode is set at startup, not runtime** - No elevation attack
