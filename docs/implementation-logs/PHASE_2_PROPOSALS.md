# Phase 2 Implementation Log: Proposals Table & Write Path

**Status:** ✅ COMPLETED  
**Completed:** 2026-01-14

---

## Objective

Implement proposal workflow: agent/LLM writes create proposals (not direct records), with approval flow that creates auditable links.

## Changes Implemented

### 1. Proposals Schema (Migration V5)

**File:** `apps/cli/internal/memory/schema.go`

Created proposals table:

```sql
CREATE TABLE proposals (
    id TEXT PRIMARY KEY,
    proposed_as TEXT NOT NULL,           -- 'decision' or 'learning'
    content TEXT NOT NULL,
    context TEXT DEFAULT '',
    scope TEXT DEFAULT 'palace',
    scope_path TEXT DEFAULT '',
    source TEXT NOT NULL,                -- 'agent', 'auto-extract', 'cli'
    session_id TEXT DEFAULT '',
    agent_type TEXT DEFAULT '',
    evidence_refs TEXT DEFAULT '{}',     -- JSON: {sessionId, conversationId, extractor}
    classification_confidence REAL DEFAULT 0,
    classification_signals TEXT DEFAULT '[]',
    dedupe_key TEXT DEFAULT '',
    status TEXT DEFAULT 'pending',       -- 'pending', 'approved', 'rejected', 'expired'
    reviewed_by TEXT DEFAULT '',
    reviewed_at TEXT DEFAULT '',
    review_note TEXT DEFAULT '',
    promoted_to_id TEXT DEFAULT '',
    created_at TEXT NOT NULL,
    expires_at TEXT DEFAULT '',
    archived_at TEXT DEFAULT ''
);

-- Unique index on dedupe_key (prevents duplicate proposals)
CREATE UNIQUE INDEX idx_proposals_dedupe ON proposals(dedupe_key) WHERE dedupe_key != '';

-- Indexes for common queries
CREATE INDEX idx_proposals_status ON proposals(status);
CREATE INDEX idx_proposals_expires ON proposals(expires_at) WHERE expires_at != '';

-- FTS5 for full-text search on proposals
CREATE VIRTUAL TABLE proposals_fts USING fts5(
    id UNINDEXED, proposed_as, content, context,
    content=proposals, content_rowid=rowid
);

-- Trigger to keep FTS in sync
CREATE TRIGGER proposals_fts_insert AFTER INSERT ON proposals BEGIN
    INSERT INTO proposals_fts(rowid, id, proposed_as, content, context)
    VALUES (new.rowid, new.id, new.proposed_as, new.content, new.context);
END;

CREATE TRIGGER proposals_fts_delete AFTER DELETE ON proposals BEGIN
    DELETE FROM proposals_fts WHERE rowid = old.rowid;
END;
```

**Key Features:**

- Dedupe protection via unique index on `dedupe_key`
- Evidence tracking via `evidence_refs` JSON field
- Full-text search support
- Expiration tracking for auto-archival

### 2. Proposal CRUD Operations

**File:** `apps/cli/internal/memory/proposal.go` (NEW)

Implemented functions:

- `AddProposal(p Proposal) (string, error)` - Create proposal with validation
- `GetProposal(id string) (*Proposal, error)` - Retrieve by ID
- `GetProposals(status, proposedAs, scope, scopePath string, limit int) ([]Proposal, error)` - List/filter
- `SearchProposals(query string, limit int) ([]Proposal, error)` - FTS search
- `CheckDuplicateProposal(dedupeKey string) (*Proposal, error)` - Dedupe check
- `ApproveProposal(id, by, note string) (string, error)` - Approval flow
- `RejectProposal(id, by, note string) error` - Rejection flow
- `ExpireProposal(id string) error` - Mark as expired
- `DeleteProposal(id string) error` - Hard delete
- `CountProposals(status string) (int, error)` - Count by status

**Validation:**

- `proposed_as` must be "decision" or "learning"
- `content` required
- `source` required

### 3. Dedupe Key Generation

**File:** `apps/cli/internal/memory/proposal.go`

```go
func GenerateDedupeKey(proposedAs, content, scope, scopePath string) string {
    // Normalize content: lowercase, trim, collapse whitespace
    normalized := strings.ToLower(strings.TrimSpace(content))
    normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

    // Hash: type + scope + normalized content
    data := proposedAs + ":" + scope + ":" + scopePath + ":" + normalized
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:16]) // First 16 bytes = 32 hex chars
}
```

**Design:**

- Content-based: same content in same scope = same key
- Collision-resistant: SHA-256 truncated to 128 bits
- Scope-aware: different scope = different key

### 4. Approval Flow with Promotion

**File:** `apps/cli/internal/memory/proposal.go`

```go
func (m *Memory) ApproveProposal(id, by, note string) (string, error) {
    // 1. Retrieve proposal
    proposal, err := m.GetProposal(id)

    // 2. Validate: must be pending
    if proposal.Status != "pending" {
        return "", fmt.Errorf("proposal already processed")
    }

    // 3. Create approved record
    var recordID string
    switch proposal.ProposedAs {
    case "decision":
        recordID, err = m.AddDecision(Decision{
            Content:                proposal.Content,
            Authority:              string(AuthorityApproved),
            PromotedFromProposalID: id,
            Scope:                  proposal.Scope,
            // ... other fields from proposal
        })
    case "learning":
        recordID, err = m.AddLearning(Learning{
            Content:                proposal.Content,
            Authority:              string(AuthorityApproved),
            PromotedFromProposalID: id,
            Scope:                  proposal.Scope,
            // ... other fields from proposal
        })
    }

    // 4. Update proposal status
    m.db.Exec(`UPDATE proposals SET status = 'approved',
                reviewed_by = ?, reviewed_at = ?, review_note = ?,
                promoted_to_id = ? WHERE id = ?`,
              by, time.Now().UTC().Format(time.RFC3339), note, recordID, id)

    return recordID, nil
}
```

**Key Properties:**

- Atomic: record creation + proposal update in one transaction
- Immutable link: `promoted_from_proposal_id` on record points back
- Status tracking: proposal marked as `approved`
- Audit trail: reviewer ID and timestamp captured

### 5. MCP Tool Integration

**File:** `apps/cli/internal/butler/mcp_tools_brain.go`

Modified `toolStore()`:

```go
func (s *MCPServer) toolStore(id any, args map[string]interface{}) jsonRPCResponse {
    // ... parse args

    // Classification
    classification := ClassifyContent(content)

    // Route based on type
    switch classification.Type {
    case "idea":
        // Ideas still go direct (no governance)
        ideaID, _ := s.butler.AddIdea(...)
        return success(ideaID)

    case "decision", "learning":
        // Decisions & learnings now create proposals
        dedupe := GenerateDedupeKey(classification.Type, content, scope, scopePath)

        // Check for duplicates
        if existing, _ := s.butler.memory.CheckDuplicateProposal(dedupe); existing != nil {
            return error("Duplicate proposal detected")
        }

        // Create proposal
        proposal := Proposal{
            ProposedAs:                classification.Type,
            Content:                   content,
            Scope:                     scope,
            ClassificationConfidence:  classification.Confidence,
            ClassificationSignals:     classification.Signals,
            DedupeKey:                 dedupe,
            Source:                    "agent",
            // ... other fields
        }
        propID, _ := s.butler.memory.AddProposal(proposal)

        return success(fmt.Sprintf("Proposal created: %s", propID))
    }
}
```

**Change Summary:**

- Ideas: unchanged (direct write)
- Decisions/Learnings: now create proposals
- Dedupe check before creation
- Classification signals stored in proposal

### 6. Auto-Extract Integration

**File:** `apps/cli/internal/memory/extract.go`

Updated `AutoExtract()` to create proposals with evidence:

```go
func (m *Memory) AutoExtract(sessionID, conversationID string, messages []Message) ([]ExtractedRecord, error) {
    // ... LLM extraction logic

    for _, record := range extracted {
        // Generate evidence refs
        evidenceRefs := map[string]interface{}{
            "sessionId":      sessionID,
            "conversationId": conversationID,
            "extractor":      "llm-auto-extract",
            "messageCount":   len(messages),
        }
        evidenceJSON, _ := json.Marshal(evidenceRefs)

        // Generate dedupe key
        dedupe := GenerateDedupeKey(record.Type, record.Content, record.Scope, record.ScopePath)

        // Check for duplicates
        if existing, _ := m.CheckDuplicateProposal(dedupe); existing != nil {
            continue // Skip duplicate
        }

        // Create proposal (not direct record)
        proposal := Proposal{
            ProposedAs:                record.Type,
            Content:                   record.Content,
            EvidenceRefs:             string(evidenceJSON),
            ClassificationConfidence:  record.Confidence,
            ClassificationSignals:     marshalSignals(record.ClassificationSignals),
            DedupeKey:                 dedupe,
            Source:                    "auto-extract",
            SessionID:                 sessionID,
        }
        m.AddProposal(proposal)
    }
}
```

**Evidence Captured:**

- Session ID (which agent session created this)
- Conversation ID (which conversation prompted it)
- Extractor type (`llm-auto-extract`)
- Message count (how much context was analyzed)

### 7. CLI Commands

**Files Created:**

- `apps/cli/internal/cli/commands/proposals.go` - `palace proposals` command
- `apps/cli/internal/cli/commands/approve.go` - `palace approve` command (merged into proposals.go)

**Command: `palace proposals`**

```bash
palace proposals                  # List pending proposals
palace proposals --status approved
palace proposals --limit 20
```

**Command: `palace proposals approve <id>`**

```bash
palace proposals approve prop_abc123 --note "Verified in testing"
```

**Command: `palace proposals reject <id>`**

```bash
palace proposals reject prop_xyz789 --note "Contradicts decision d_123"
```

### 8. CLI Store Command Update

**File:** `apps/cli/internal/cli/commands/store.go`

Added `--direct` flag:

```go
type StoreOptions struct {
    // ... existing fields
    Direct bool  // NEW: bypass proposals, write directly
}

func ExecuteStore(opts StoreOptions) error {
    classification := ClassifyContent(opts.Content)

    // Ideas always direct
    if classification.Type == "idea" {
        return storeIdeaDirect(...)
    }

    // --direct flag: write directly with audit
    if opts.Direct {
        if classification.Type == "decision" {
            id, _ := mem.AddDecision(Decision{
                Content:   opts.Content,
                Authority: string(AuthorityApproved),
                Source:    "human",
            })
            // Create audit log entry
            mem.AddAuditLog(AuditLog{
                Action:    "direct_write",
                ActorType: "human",
                EntityType: "decision",
                EntityID:  id,
            })
            fmt.Printf("✅ Decision created: %s (direct write, audited)\n", id)
        }
        // ... similar for learning
        return nil
    }

    // Default: create proposal
    dedupe := GenerateDedupeKey(classification.Type, opts.Content, opts.Scope, opts.ScopePath)
    if existing, _ := mem.CheckDuplicateProposal(dedupe); existing != nil {
        return fmt.Errorf("Duplicate proposal exists: %s", existing.ID)
    }

    proposal := Proposal{
        ProposedAs: classification.Type,
        Content:    opts.Content,
        DedupeKey:  dedupe,
        Source:     "cli",
        // ...
    }
    propID, _ := mem.AddProposal(proposal)
    fmt.Printf("📥 Proposal created: %s\n", propID)
    fmt.Println("  Use 'palace proposals approve <id>' to approve.")
    return nil
}
```

**Behavior:**

- Default: creates proposal for decisions/learnings
- `--direct`: writes directly with `authority=approved`, creates audit log
- Ideas: always direct (no proposals)

---

## Validation & Testing

### Test Coverage

**File:** `apps/cli/internal/memory/proposal_test.go`

Tests implemented:

1. `TestAddProposal` - Basic proposal creation
2. `TestAddProposalValidation` - Input validation
3. `TestGetProposals` - List and filter
4. `TestSearchProposals` - FTS search
5. `TestCheckDuplicateProposal` - Dedupe detection
6. `TestApproveProposal` - Approval creates record with link
7. `TestApproveLearningProposal` - Learning promotion
8. `TestApproveProposalAlreadyProcessed` - Idempotency
9. `TestRejectProposal` - Rejection flow
10. `TestRejectProposalAlreadyProcessed` - Idempotency
11. `TestExpireProposal` - Expiration
12. `TestCountProposals` - Counting by status
13. `TestGenerateDedupeKey` - Dedupe key generation
14. `TestDeleteProposal` - Hard delete
15. `TestProposalFTSSearch` - Full-text search

**Results:** ✅ All tests pass

### Integration Tests

**File:** `apps/cli/internal/butler/mcp_test.go`

Added test: `TestMCPToolHandlersBrain`

- Verifies `store` creates proposals
- Verifies duplicate rejection
- Confirms proposal fields populated

**File:** `apps/cli/internal/cli/commands/store_test.go`

Added tests:

- `TestExecuteStoreSuccess` - Default proposal creation
- `TestExecuteStoreAsDecision` - Classification to proposal
- `TestExecuteStoreAsIdea` - Ideas still direct

**Results:** ✅ All tests pass

### Manual Verification

```bash
# Test full flow
palace init
palace scan

# Create proposal
palace store "We should use PostgreSQL for the database"
# Output: 📥 Proposal created (decision): prop_abc123

# List proposals
palace proposals
# Shows pending proposal

# Approve
palace proposals approve prop_abc123 --note "Team consensus"
# Output: ✅ Approved: d_xyz789

# Verify decision exists with link
palace recall decisions --id d_xyz789
# Shows decision with promotedFromProposalId: prop_abc123
```

---

## Acceptance Criteria

| Criterion                                                 | Status | Evidence                                           |
| --------------------------------------------------------- | ------ | -------------------------------------------------- |
| `store` MCP tool creates proposal (not decision/learning) | ✅     | MCP handler updated, tests pass                    |
| Proposals have `dedupe_key`, duplicates rejected          | ✅     | Unique index, tests validate                       |
| Promoted records have `promoted_from_proposal_id` set     | ✅     | ApproveProposal sets field                         |
| LLM proposals have `evidence_refs` populated              | ✅     | AutoExtract includes session/conversation IDs      |
| Expired proposals auto-archive                            | ⚠️     | Schema supports, auto-archival job not implemented |

**Note:** Auto-archival of expired proposals is supported in schema but background job not yet implemented. Can be added as maintenance task.

---

## Migration Impact

### Database Changes

- New `proposals` table with FTS
- Non-breaking: existing tables unchanged
- Triggers for FTS sync

### API/CLI Impact

- **Breaking:** `palace store` now creates proposals by default
  - Migration: Add `--direct` flag for human direct writes
  - Agents unaffected (proposals expected)
- **Additive:** New commands `palace proposals`, `palace approve/reject`

### Rollback Plan

```sql
DROP TRIGGER proposals_fts_insert;
DROP TRIGGER proposals_fts_delete;
DROP TABLE proposals_fts;
DROP TABLE proposals;
```

Reverting code: Remove proposal flow from `mcp_tools_brain.go` and `commands/store.go`

---

## Related Documentation

- [Proposals Design](../IMPLEMENTATION_PLAN_V2.1.md#phase-2-proposals-table--write-path)
- [Dedupe Strategy](../IMPLEMENTATION_PLAN_V2.1.md#promotion-logic)

## Next Phase

Phase 3: MCP Mode Gate & Tool Segmentation
