# Mind Palace Governance Implementation Plan v2

## Decision Points (Requires Product Input)

**DP-1: Can humans write approved state directly via CLI?**
- **(A) Yes** - `palace store --direct` writes `authority=approved` (human CLI only)
- **(B) No** - All inputs go through proposals, even human CLI
- **Default:** (A) - humans can bypass proposal workflow for their own input

**DP-2: Legacy record authority value**
- **(A) `legacy_approved`** - Distinct value, queries include it with approved
- **(B) `approved`** - Treat as approved (simpler, but loses provenance)
- **Default:** (A) - preserves audit trail of what existed before governance

---

## Phase 1: Authority Field & Legacy Compatibility

**Goal:** Add `authority` field with clean semantics. No inference from source.

### Fix Applied
- **Fix 1:** No backfill from `source=cli`. All existing records get `authority='legacy_approved'`.
- **Fix 2:** Authority is only `proposed | approved | legacy_approved`. Lifecycle stays in `status`.

### Schema Changes (Migration V4)

```sql
-- Add authority column (no inference from source)
ALTER TABLE decisions ADD COLUMN authority TEXT DEFAULT 'proposed';
ALTER TABLE decisions ADD COLUMN promoted_from_proposal_id TEXT DEFAULT '';
ALTER TABLE learnings ADD COLUMN authority TEXT DEFAULT 'proposed';
ALTER TABLE learnings ADD COLUMN promoted_from_proposal_id TEXT DEFAULT '';

-- Backfill existing records with legacy marker (Fix 1)
UPDATE decisions SET authority = 'legacy_approved' WHERE authority = 'proposed';
UPDATE learnings SET authority = 'legacy_approved' WHERE authority = 'proposed';

-- Indexes
CREATE INDEX idx_decisions_authority ON decisions(authority);
CREATE INDEX idx_learnings_authority ON learnings(authority);
```

**Authority values:**
| Value | Meaning |
|-------|---------|
| `proposed` | Awaiting review |
| `approved` | Human-approved |
| `legacy_approved` | Existed before governance system |

**Lifecycle stays in `status`:**
- Decisions: `active | superseded | reversed`
- Learnings: `active | obsolete | archived`

### Query Changes

```go
// Authoritative = approved OR legacy_approved
var authoritativeValues = []string{"approved", "legacy_approved"}

func (m *Memory) GetAuthoritativeDecisions(scope, scopePath string, limit int) ([]Decision, error) {
    return m.queryDecisions(`
        SELECT ... FROM decisions
        WHERE authority IN ('approved', 'legacy_approved')
          AND status = 'active'
          AND scope = ? AND scope_path = ?
        LIMIT ?`, scope, scopePath, limit)
}
```

### Files Changed
- `memory/schema.go` - Migration V4
- `memory/decision.go` - Add Authority field, update queries
- `memory/learning.go` - Add Authority field, update queries

### Acceptance Criteria
1. All existing records have `authority = 'legacy_approved'`
2. New agent-created records have `authority = 'proposed'`
3. Default queries return `approved` OR `legacy_approved`
4. `status` field unchanged (lifecycle separate from authority)

### Risks
| Risk | Mitigation |
|------|------------|
| Queries must check two values | Use constant list, indexed |

---

## Phase 2: Proposals Table & Write Path

**Goal:** Agent/LLM writes go to proposals. Promotion creates auditable link.

### Fixes Applied
- **Fix 3:** `promoted_from_proposal_id` on promoted records. Proposals immutable.
- **Fix 7:** `dedupe_key` and `expires_at` / `archived_at` for inbox hygiene.
- **Fix 8:** Evidence refs required for LLM proposals (`evidence_refs` JSON field).

### Schema Changes (Migration V5)

```sql
CREATE TABLE proposals (
    id TEXT PRIMARY KEY,
    proposed_as TEXT NOT NULL,           -- 'decision', 'learning', 'link'
    content TEXT NOT NULL,
    context TEXT DEFAULT '',

    -- Scope
    scope TEXT DEFAULT 'palace',
    scope_path TEXT DEFAULT '',

    -- Provenance
    source TEXT NOT NULL,                -- 'agent', 'llm', 'human'
    session_id TEXT DEFAULT '',
    agent_type TEXT DEFAULT '',

    -- Evidence (Fix 8)
    evidence_refs TEXT DEFAULT '{}',     -- JSON: {"session_id": "...", "activity_ids": [...], "conversation_id": "..."}

    -- Classification (if auto-classified)
    classification_confidence REAL DEFAULT 0,
    classification_signals TEXT DEFAULT '[]',

    -- Deduplication (Fix 7)
    dedupe_key TEXT DEFAULT '',          -- Hash of (content, scope, proposed_as)

    -- Lifecycle
    status TEXT DEFAULT 'pending',       -- 'pending', 'approved', 'rejected', 'expired', 'archived'
    reviewed_by TEXT DEFAULT '',
    reviewed_at TEXT DEFAULT '',
    review_note TEXT DEFAULT '',
    promoted_to_id TEXT DEFAULT '',

    -- Expiry (Fix 7)
    created_at TEXT NOT NULL,
    expires_at TEXT DEFAULT '',          -- Optional TTL
    archived_at TEXT DEFAULT ''
);

CREATE UNIQUE INDEX idx_proposals_dedupe ON proposals(dedupe_key) WHERE dedupe_key != '';
CREATE INDEX idx_proposals_status ON proposals(status);
CREATE INDEX idx_proposals_expires ON proposals(expires_at) WHERE expires_at != '';
```

### Dedupe Algorithm (Fix 7)

```go
func (p *Proposal) ComputeDedupeKey() string {
    h := sha256.New()
    h.Write([]byte(p.ProposedAs))
    h.Write([]byte(p.Scope))
    h.Write([]byte(p.ScopePath))
    h.Write([]byte(normalizeContent(p.Content)))
    return hex.EncodeToString(h.Sum(nil))[:16]
}

func (m *Memory) AddProposal(p Proposal) (string, error) {
    p.DedupeKey = p.ComputeDedupeKey()

    // Check for existing pending proposal with same key
    existing, _ := m.GetProposalByDedupeKey(p.DedupeKey)
    if existing != nil && existing.Status == "pending" {
        return existing.ID, nil // Return existing, don't create duplicate
    }
    // ... insert
}
```

### Promotion with Referential Link (Fix 3)

```go
func (m *Memory) ApproveProposal(proposalID, reviewedBy, note string) (*PromotionResult, error) {
    tx, _ := m.db.BeginTx(ctx, nil)
    defer tx.Rollback()

    proposal, _ := m.getProposalTx(tx, proposalID)
    if proposal.Status != "pending" {
        return nil, fmt.Errorf("proposal status is %s, not pending", proposal.Status)
    }

    var promotedID string
    switch proposal.ProposedAs {
    case "decision":
        dec := Decision{
            Content:                 proposal.Content,
            Authority:               "approved",
            PromotedFromProposalID:  proposalID,  // Fix 3: referential link
            // ...
        }
        promotedID, _ = m.addDecisionTx(tx, dec)
    // ... learning, link cases
    }

    // Proposal becomes immutable (Fix 3)
    tx.ExecContext(ctx, `
        UPDATE proposals
        SET status = 'approved', promoted_to_id = ?, reviewed_by = ?, reviewed_at = ?, review_note = ?
        WHERE id = ?`,
        promotedID, reviewedBy, time.Now().UTC().Format(time.RFC3339), note, proposalID)

    tx.Commit()
    return &PromotionResult{ProposalID: proposalID, PromotedID: promotedID}, nil
}
```

### LLM Evidence Requirements (Fix 8)

```go
// AutoExtract must include evidence
func (b *Butler) autoExtractForSession(sessionID string, llm LLMClient) {
    conv, _ := b.memory.GetConversation(sessionID)
    results := llm.Extract(conv.Messages)

    for _, r := range results {
        p := Proposal{
            ProposedAs: r.Kind,
            Content:    r.Content,
            Source:     "llm",
            EvidenceRefs: map[string]any{  // Fix 8
                "session_id":      sessionID,
                "conversation_id": conv.ID,
                "extractor":       "auto-extract",
            },
        }
        b.memory.AddProposal(p)
    }
}

// ContradictionAutoLink must include evidence
func autoLinkContradiction(recordID, contradictingID string, analysis ContradictionResult) Proposal {
    return Proposal{
        ProposedAs: "link",
        Content:    fmt.Sprintf("Contradiction: %s vs %s", recordID, contradictingID),
        Source:     "llm",
        EvidenceRefs: map[string]any{  // Fix 8
            "analyzer":      "contradiction-detection",
            "source_record": recordID,
            "target_record": contradictingID,
            "confidence":    analysis.Confidence,
            "explanation":   analysis.Explanation,
        },
    }
}
```

### Expiry Cleanup (Fix 7)

```go
// Run periodically or on palace clean
func (m *Memory) ArchiveExpiredProposals() (int, error) {
    now := time.Now().UTC().Format(time.RFC3339)
    result, _ := m.db.ExecContext(ctx, `
        UPDATE proposals
        SET status = 'expired', archived_at = ?
        WHERE status = 'pending' AND expires_at != '' AND expires_at < ?`,
        now, now)
    count, _ := result.RowsAffected()
    return int(count), nil
}
```

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

### Risks
| Risk | Mitigation |
|------|------------|
| Proposal backlog | Expiry policy, `palace proposals --archive-old` |
| Dedupe false positives | Content normalization, manual override |

---

## Phase 3: MCP Mode Gate & Tool Segmentation

**Goal:** Enforce governance by mode, not just tool filtering.

### Fix Applied
- **Fix 4:** Explicit mode gate (`agent_mode` vs `human_mode`), not just hidden tools.

### Mode Gate Design

```go
type MCPMode string

const (
    MCPModeAgent MCPMode = "agent"  // Default: restricted
    MCPModeHuman MCPMode = "human"  // Elevated: full access
)

type MCPServer struct {
    butler *Butler
    mode   MCPMode  // Set at initialization
    // ...
}

// Mode is set via initialization parameter, not runtime
func NewMCPServer(butler *Butler, mode MCPMode) *MCPServer {
    return &MCPServer{butler: butler, mode: mode}
}
```

**CLI invocation:**
```bash
# Agent mode (default for MCP stdio)
palace serve --mode agent

# Human mode (requires explicit flag, e.g., for dashboard backend)
palace serve --mode human --require-auth
```

### Tool Categories

```go
type ToolAccess struct {
    AgentMode bool  // Available in agent mode
    HumanMode bool  // Available in human mode
}

var toolAccess = map[string]ToolAccess{
    // Read tools - both modes
    "explore":          {AgentMode: true,  HumanMode: true},
    "recall":           {AgentMode: true,  HumanMode: true},
    "recall_decisions": {AgentMode: true,  HumanMode: true},
    "brief":            {AgentMode: true,  HumanMode: true},
    "get_route":        {AgentMode: true,  HumanMode: true},  // Fix 9

    // Propose tools - both modes
    "store":            {AgentMode: true,  HumanMode: true},
    "session_start":    {AgentMode: true,  HumanMode: true},
    "session_log":      {AgentMode: true,  HumanMode: true},
    "session_end":      {AgentMode: true,  HumanMode: true},

    // Admin tools - human mode only
    "approve":          {AgentMode: false, HumanMode: true},
    "reject":           {AgentMode: false, HumanMode: true},
    "recall_outcome":   {AgentMode: false, HumanMode: true},
    "recall_link":      {AgentMode: false, HumanMode: true},
    "store_direct":     {AgentMode: false, HumanMode: true},  // Bypass proposals
}
```

### Enforcement

```go
func (s *MCPServer) handleToolsList(req jsonRPCRequest) jsonRPCResponse {
    tools := []mcpTool{}
    for _, t := range allTools {
        access := toolAccess[t.Name]
        if (s.mode == MCPModeAgent && access.AgentMode) ||
           (s.mode == MCPModeHuman && access.HumanMode) {
            tools = append(tools, t)
        }
    }
    return jsonRPCResponse{Result: map[string]any{"tools": tools}}
}

func (s *MCPServer) handleToolsCall(req jsonRPCRequest) jsonRPCResponse {
    // ... parse params ...

    access := toolAccess[params.Name]
    allowed := (s.mode == MCPModeAgent && access.AgentMode) ||
               (s.mode == MCPModeHuman && access.HumanMode)

    if !allowed {
        return jsonRPCResponse{
            Error: &rpcError{
                Code:    -32602,
                Message: fmt.Sprintf("Tool %q not available in %s mode", params.Name, s.mode),
            },
        }
    }
    // ... execute tool ...
}
```

### Audit Log (from v1, unchanged)

```sql
CREATE TABLE audit_log (
    id TEXT PRIMARY KEY,
    timestamp TEXT NOT NULL,
    actor_type TEXT NOT NULL,      -- 'human', 'agent', 'llm', 'system'
    actor_id TEXT DEFAULT '',
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    mcp_mode TEXT DEFAULT '',      -- 'agent', 'human'
    details TEXT DEFAULT '{}'
);
```

### Files Changed
- `butler/mcp.go` - Add mode field, tool filtering by mode
- `butler/mcp_tools.go` - Tool access registry
- `cli/commands/serve.go` - Add `--mode` flag
- `memory/schema.go` - Migration V6 (audit_log)
- `memory/audit.go` - New file

### Acceptance Criteria
1. `palace serve --mode agent` excludes admin tools
2. `palace serve --mode human` includes all tools
3. Tool call rejected if mode doesn't allow
4. Mode logged in audit entries
5. No way to switch mode at runtime

### Risks
| Risk | Mitigation |
|------|------------|
| Human mode accidentally exposed | Require `--require-auth` flag for human mode |

---

## Phase 4: Authoritative State Query Surface

**Goal:** Deterministic, bounded query for "what is true."

### Fixes Applied
- **Fix 5:** Scope expansion in Go, not SQL. Simple view.
- **Fix 6:** Deterministic bounds (max items, max chars), no token heuristics.

### Simple View (no scope logic)

```sql
-- Migration V7
CREATE VIEW authoritative_decisions AS
SELECT id, content, scope, scope_path, created_at
FROM decisions
WHERE authority IN ('approved', 'legacy_approved')
  AND status = 'active';

CREATE VIEW authoritative_learnings AS
SELECT id, content, scope, scope_path, confidence, created_at
FROM learnings
WHERE authority IN ('approved', 'legacy_approved')
  AND status = 'active'
  AND confidence >= 0.3;
```

### Scope Expansion in Go (Fix 5)

```go
type ScopeChain struct {
    Scopes []ScopeLevel
}

type ScopeLevel struct {
    Scope     string
    ScopePath string
}

// Deterministic scope expansion
func ExpandScope(scope, scopePath string) ScopeChain {
    chain := ScopeChain{}

    switch scope {
    case "file":
        // file → room (inferred) → palace
        chain.Scopes = append(chain.Scopes, ScopeLevel{"file", scopePath})
        room := inferRoomFromPath(scopePath)
        if room != "" {
            chain.Scopes = append(chain.Scopes, ScopeLevel{"room", room})
        }
        chain.Scopes = append(chain.Scopes, ScopeLevel{"palace", ""})
    case "room":
        chain.Scopes = append(chain.Scopes, ScopeLevel{"room", scopePath})
        chain.Scopes = append(chain.Scopes, ScopeLevel{"palace", ""})
    case "palace":
        chain.Scopes = append(chain.Scopes, ScopeLevel{"palace", ""})
    }

    return chain
}

func inferRoomFromPath(path string) string {
    // Deterministic: first directory component
    parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
    if len(parts) > 0 {
        return parts[0]
    }
    return ""
}
```

### Bounded Query (Fix 6)

```go
type AuthoritativeQueryConfig struct {
    MaxDecisions   int  // Default: 10
    MaxLearnings   int  // Default: 10
    MaxContentLen  int  // Default: 500 chars per item
}

func DefaultQueryConfig() AuthoritativeQueryConfig {
    return AuthoritativeQueryConfig{
        MaxDecisions:  10,
        MaxLearnings:  10,
        MaxContentLen: 500,
    }
}

func (b *Butler) GetAuthoritativeState(scope, scopePath string, cfg AuthoritativeQueryConfig) AuthoritativeState {
    chain := ExpandScope(scope, scopePath)

    var decisions []Decision
    var learnings []Learning

    // Query each scope level in order, stop when limits reached
    for _, level := range chain.Scopes {
        if len(decisions) < cfg.MaxDecisions {
            decs, _ := b.memory.QueryAuthoritativeDecisions(level.Scope, level.ScopePath, cfg.MaxDecisions - len(decisions))
            decisions = append(decisions, decs...)
        }
        if len(learnings) < cfg.MaxLearnings {
            learns, _ := b.memory.QueryAuthoritativeLearnings(level.Scope, level.ScopePath, cfg.MaxLearnings - len(learnings))
            learnings = append(learnings, learns...)
        }
    }

    // Truncate content deterministically (Fix 6)
    for i := range decisions {
        if len(decisions[i].Content) > cfg.MaxContentLen {
            decisions[i].Content = decisions[i].Content[:cfg.MaxContentLen] + "..."
        }
    }
    for i := range learnings {
        if len(learnings[i].Content) > cfg.MaxContentLen {
            learnings[i].Content = learnings[i].Content[:cfg.MaxContentLen] + "..."
        }
    }

    return AuthoritativeState{Decisions: decisions, Learnings: learnings}
}
```

### Files Changed
- `memory/schema.go` - Migration V7 (views)
- `memory/scope.go` - New file (scope expansion)
- `butler/butler_context.go` - Bounded query builder

### Acceptance Criteria
1. Scope expansion is deterministic (same input → same output)
2. No token estimation (removed `len/4`)
3. Query bounded by max items and max chars
4. View only filters authority, no scope logic

---

## Phase 5: Route/Polyline Query

**Goal:** Add `get_route` MCP tool for deterministic navigation guidance.

### Fix Applied
- **Fix 9:** Rule-based route derivation from rooms + profile + authoritative state.

### MCP Tool Contract

```go
// Input
type GetRouteParams struct {
    Intent string `json:"intent"`      // e.g., "understand auth flow", "add new endpoint"
    Scope  string `json:"scope"`       // Starting scope
}

// Output
type RouteResult struct {
    Nodes []RouteNode `json:"nodes"`
    Meta  RouteMeta   `json:"meta"`
}

type RouteNode struct {
    Order       int      `json:"order"`
    Kind        string   `json:"kind"`       // "room", "file", "decision", "learning"
    ID          string   `json:"id"`         // Room name, file path, or record ID
    Reason      string   `json:"reason"`     // Why included
    EntryPoints []string `json:"entryPoints,omitempty"` // For rooms
}

type RouteMeta struct {
    RuleVersion string `json:"ruleVersion"`
    NodeCount   int    `json:"nodeCount"`
}
```

### Rule-Based Derivation

```go
const RouteRuleVersion = "v1.0"
const MaxRouteNodes = 10

func (b *Butler) GetRoute(intent, scope string) RouteResult {
    nodes := []RouteNode{}

    // Rule 1: Match intent keywords to room names
    rooms := b.ListRooms()
    intentWords := tokenize(strings.ToLower(intent))

    for _, room := range rooms {
        if matchesAny(room.Name, intentWords) || matchesAny(room.Summary, intentWords) {
            nodes = append(nodes, RouteNode{
                Order:       len(nodes) + 1,
                Kind:        "room",
                ID:          room.Name,
                Reason:      "Room name/summary matches intent",
                EntryPoints: room.EntryPoints,
            })
        }
        if len(nodes) >= MaxRouteNodes {
            break
        }
    }

    // Rule 2: Include relevant decisions from scope
    if len(nodes) < MaxRouteNodes {
        chain := ExpandScope(scope, "")
        for _, level := range chain.Scopes {
            decs, _ := b.memory.QueryAuthoritativeDecisions(level.Scope, level.ScopePath, 3)
            for _, d := range decs {
                if matchesAny(d.Content, intentWords) {
                    nodes = append(nodes, RouteNode{
                        Order:  len(nodes) + 1,
                        Kind:   "decision",
                        ID:     d.ID,
                        Reason: "Decision content matches intent",
                    })
                }
                if len(nodes) >= MaxRouteNodes {
                    break
                }
            }
        }
    }

    // Rule 3: Include high-confidence learnings
    if len(nodes) < MaxRouteNodes {
        learns, _ := b.memory.QueryAuthoritativeLearnings(scope, "", 5)
        for _, l := range learns {
            if l.Confidence >= 0.7 && matchesAny(l.Content, intentWords) {
                nodes = append(nodes, RouteNode{
                    Order:  len(nodes) + 1,
                    Kind:   "learning",
                    ID:     l.ID,
                    Reason: fmt.Sprintf("High-confidence learning (%.0f%%)", l.Confidence*100),
                })
            }
            if len(nodes) >= MaxRouteNodes {
                break
            }
        }
    }

    return RouteResult{
        Nodes: nodes,
        Meta:  RouteMeta{RuleVersion: RouteRuleVersion, NodeCount: len(nodes)},
    }
}

func tokenize(s string) []string {
    return strings.Fields(strings.ToLower(s))
}

func matchesAny(text string, words []string) bool {
    lower := strings.ToLower(text)
    for _, w := range words {
        if strings.Contains(lower, w) {
            return true
        }
    }
    return false
}
```

### Files Changed
- `butler/butler_route.go` - New file
- `butler/mcp_tools_route.go` - New file
- `butler/mcp.go` - Register `get_route` tool

### Acceptance Criteria
1. `get_route` returns ordered list of max 10 nodes
2. Derivation is deterministic (same input → same output)
3. Rule version included in response
4. No LLM or heavy static analysis

---

## Summary

| Phase | Goal | Key Fix Applied |
|-------|------|-----------------|
| 1 | Authority field | Fix 1 (no inference), Fix 2 (authority vs lifecycle) |
| 2 | Proposals table | Fix 3 (referential), Fix 7 (dedupe/expiry), Fix 8 (evidence) |
| 3 | MCP mode gate | Fix 4 (mode vs tool filtering) |
| 4 | Authoritative query | Fix 5 (scope in Go), Fix 6 (deterministic bounds) |
| 5 | Route query | Fix 9 (rule-based navigation) |

## Minimal Viable Governance Invariants

1. **Agents can only create proposals** - Enforced by mode gate + tool access
2. **Promoted records link back to proposals** - `promoted_from_proposal_id`
3. **LLM outputs carry evidence** - `evidence_refs` required
4. **Default queries return approved + legacy_approved only**
5. **Mode is set at startup, not runtime** - No elevation attack
