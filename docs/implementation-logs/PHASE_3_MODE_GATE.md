# Phase 3 Implementation Log: MCP Mode Gate & Tool Segmentation

**Status:** ✅ COMPLETED  
**Completed:** 2026-01-14

---

## Objective

Enforce governance by mode: restrict admin tools and direct writes to human mode only, with explicit errors for violations.

## Changes Implemented

### 1. MCP Mode Enum

**File:** `apps/cli/internal/butler/mcp.go`

Added mode type and constants:

```go
type MCPMode string

const (
    MCPModeAgent MCPMode = "agent"  // Restricted: no admin tools
    MCPModeHuman MCPMode = "human"  // Full access: all tools
)

type MCPServer struct {
    // ... existing fields
    mode MCPMode  // NEW: server mode set at startup
}
```

**Design:**

- Mode set at server initialization (not runtime)
- Cannot be changed after startup (no elevation attack)
- Defaults to `agent` mode (most restrictive)

### 2. Tool Access Registry

**File:** `apps/cli/internal/butler/mcp_tools_list.go`

Created admin-only tool list:

```go
var adminOnlyTools = map[string]bool{
    "store_direct":    true,
    "approve":         true,
    "reject":          true,
    "recall_outcome":  true,  // Records decision outcomes
    "recall_link":     true,  // Creates relationships
    "recall_unlink":   true,  // Deletes relationships
}

func IsAdminOnlyTool(name string) bool {
    return adminOnlyTools[name]
}
```

**Tool Access Matrix:**

| Tool                                         | Agent Mode | Human Mode | Rationale             |
| -------------------------------------------- | ---------- | ---------- | --------------------- |
| `explore`, `explore_*`                       | ✅         | ✅         | Read-only             |
| `recall`, `recall_decisions`, `recall_ideas` | ✅         | ✅         | Read-only             |
| `brief`, `brief_file`                        | ✅         | ✅         | Read-only             |
| `store`                                      | ✅         | ✅         | Creates proposals     |
| `session_*`                                  | ✅         | ✅         | Process artifacts     |
| `get_route`                                  | ✅         | ✅         | Navigation            |
| `store_direct`                               | ❌         | ✅         | Bypasses proposals    |
| `approve`                                    | ❌         | ✅         | Admin action          |
| `reject`                                     | ❌         | ✅         | Admin action          |
| `recall_outcome`                             | ❌         | ✅         | Mutates truth         |
| `recall_link`                                | ❌         | ✅         | Mutates relationships |
| `recall_unlink`                              | ❌         | ✅         | Mutates relationships |

### 3. Tool List Filtering

**File:** `apps/cli/internal/butler/mcp.go`

Modified `handleToolsList()`:

```go
func (s *MCPServer) handleToolsList(req jsonRPCRequest) jsonRPCResponse {
    allTools := buildToolsList()

    // Filter based on mode
    var availableTools []mcpTool
    for _, tool := range allTools {
        // Exclude admin-only tools in agent mode
        if s.mode == MCPModeAgent && IsAdminOnlyTool(tool.Name) {
            continue
        }
        availableTools = append(availableTools, tool)
    }

    return jsonRPCResponse{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result:  map[string]interface{}{"tools": availableTools},
    }
}
```

**Effect:**

- Agent mode: `store_direct`, `approve`, `reject`, etc. not in tool list
- Human mode: all tools visible
- Client cannot discover hidden tools in agent mode

### 4. Tool Call Enforcement

**File:** `apps/cli/internal/butler/mcp.go`

Added mode check in `handleToolsCall()`:

```go
func (s *MCPServer) handleToolsCall(req jsonRPCRequest) jsonRPCResponse {
    var params mcpToolCallParams
    json.Unmarshal(req.Params, &params)

    // Enforce mode restrictions
    if s.mode == MCPModeAgent && IsAdminOnlyTool(params.Name) {
        return jsonRPCResponse{
            JSONRPC: "2.0",
            ID:      req.ID,
            Error: &rpcError{
                Code:    -32602,
                Message: fmt.Sprintf("Tool %q not available in agent mode", params.Name),
            },
        }
    }

    // Dispatch to handler
    switch params.Name {
    case "store_direct":
        return s.toolStoreDirect(req.ID, params.Arguments)
    // ... other tools
    }
}
```

**Failure Behavior:**

- Agent calls `store_direct` → explicit error `-32602`
- Error message: `"Tool \"store_direct\" not available in agent mode"`
- No silent fallback or degradation

### 5. store_direct Tool Implementation

**File:** `apps/cli/internal/butler/mcp_tools_governance.go` (NEW)

```go
func (s *MCPServer) toolStoreDirect(id any, args map[string]interface{}) jsonRPCResponse {
    // Parse args
    content, _ := args["content"].(string)
    recordType, _ := args["as"].(string)  // "decision" or "learning"
    scope, _ := args["scope"].(string)
    scopePath, _ := args["scopePath"].(string)
    actorId, _ := args["actorId"].(string)

    // Validate
    if content == "" {
        return s.toolError(id, "content is required")
    }
    if recordType != "decision" && recordType != "learning" {
        return s.toolError(id, "as must be 'decision' or 'learning'")
    }

    // Create approved record directly
    var recordID string
    var err error

    switch recordType {
    case "decision":
        recordID, err = s.butler.memory.AddDecision(memory.Decision{
            Content:   content,
            Authority: string(memory.AuthorityApproved),
            Source:    "human",
            Scope:     scope,
            ScopePath: scopePath,
            // ... other fields from args
        })
    case "learning":
        recordID, err = s.butler.memory.AddLearning(memory.Learning{
            Content:   content,
            Authority: string(memory.AuthorityApproved),
            Source:    "human",
            Scope:     scope,
            ScopePath: scopePath,
            // ... other fields from args
        })
    }

    if err != nil {
        return s.toolError(id, err.Error())
    }

    // Create audit log entry
    s.butler.memory.AddAuditLog(memory.AuditLog{
        Action:     "direct_write",
        ActorType:  "human",
        ActorID:    actorId,
        EntityType: recordType,
        EntityID:   recordID,
        Metadata:   fmt.Sprintf(`{"content_hash":"%s"}`, hashContent(content)),
    })

    return jsonRPCResponse{
        JSONRPC: "2.0",
        ID:      id,
        Result: mcpToolResult{
            Content: []mcpContent{{
                Type: "text",
                Text: fmt.Sprintf("✅ %s created: %s (direct write, audited)", recordType, recordID),
            }},
        },
    }
}
```

**Key Properties:**

- Creates record with `authority = "approved"`
- `promoted_from_proposal_id` left empty (no proposal)
- `source = "human"`
- Always creates audit log entry
- Only available in human mode (checked before dispatch)

### 6. Audit Log Schema & Implementation

**File:** `apps/cli/internal/memory/schema.go` (Migration V6)

```sql
CREATE TABLE audit_log (
    id TEXT PRIMARY KEY,
    action TEXT NOT NULL,           -- 'direct_write', 'approve', 'reject'
    actor_type TEXT NOT NULL,       -- 'human', 'system'
    actor_id TEXT DEFAULT '',       -- user identifier
    entity_type TEXT NOT NULL,      -- 'decision', 'learning', 'proposal'
    entity_id TEXT NOT NULL,        -- ID of affected record
    metadata TEXT DEFAULT '{}',     -- JSON: additional context
    created_at TEXT NOT NULL
);

CREATE INDEX idx_audit_action ON audit_log(action);
CREATE INDEX idx_audit_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_created ON audit_log(created_at);
```

**File:** `apps/cli/internal/memory/audit.go` (NEW)

```go
type AuditLog struct {
    ID         string
    Action     string    // direct_write, approve, reject
    ActorType  string    // human, system
    ActorID    string    // user identifier
    EntityType string    // decision, learning, proposal
    EntityID   string    // affected record ID
    Metadata   string    // JSON
    CreatedAt  time.Time
}

func (m *Memory) AddAuditLog(log AuditLog) (string, error)
func (m *Memory) GetAuditLogs(action, entityType string, limit int) ([]AuditLog, error)
func (m *Memory) GetAuditLogsForEntity(entityType, entityID string) ([]AuditLog, error)
```

**Audit Events:**

- `direct_write`: Human bypassed proposals
- `approve`: Proposal approved
- `reject`: Proposal rejected

### 7. CLI Serve Command

**File:** `apps/cli/internal/cli/commands/serve.go`

Added `--mode` flag:

```go
type ServeOptions struct {
    Root string
    Mode string  // "agent" or "human"
}

func ExecuteServe(opts ServeOptions) error {
    // Validate mode
    var mode butler.MCPMode
    switch opts.Mode {
    case "agent", "":
        mode = butler.MCPModeAgent
    case "human":
        mode = butler.MCPModeHuman
    default:
        return fmt.Errorf("invalid mode: %s (must be 'agent' or 'human')", opts.Mode)
    }

    // Initialize server with mode
    server := butler.NewMCPServer(b, mode)

    // Log mode
    if mode == butler.MCPModeHuman {
        fmt.Fprintln(os.Stderr, "⚠️  MCP server starting in HUMAN mode (full access)")
    } else {
        fmt.Fprintln(os.Stderr, "🤖 MCP server starting in AGENT mode (restricted)")
    }

    server.Serve()
}
```

**Usage:**

```bash
palace serve                  # Defaults to agent mode
palace serve --mode agent     # Explicit agent mode
palace serve --mode human     # Human mode (all tools)
```

### 8. approve/reject Tool Implementations

**File:** `apps/cli/internal/butler/mcp_tools_governance.go`

```go
func (s *MCPServer) toolApprove(id any, args map[string]interface{}) jsonRPCResponse {
    proposalID, _ := args["proposalId"].(string)
    by, _ := args["by"].(string)
    note, _ := args["note"].(string)

    if proposalID == "" {
        return s.toolError(id, "proposalId is required")
    }

    // Approve and promote
    recordID, err := s.butler.memory.ApproveProposal(proposalID, by, note)
    if err != nil {
        return s.toolError(id, err.Error())
    }

    // Create audit log
    s.butler.memory.AddAuditLog(memory.AuditLog{
        Action:     "approve",
        ActorType:  "human",
        ActorID:    by,
        EntityType: "proposal",
        EntityID:   proposalID,
        Metadata:   fmt.Sprintf(`{"promoted_to":"%s"}`, recordID),
    })

    return success(fmt.Sprintf("✅ Approved: %s", recordID))
}

func (s *MCPServer) toolReject(id any, args map[string]interface{}) jsonRPCResponse {
    proposalID, _ := args["proposalId"].(string)
    by, _ := args["by"].(string)
    note, _ := args["note"].(string)

    if proposalID == "" {
        return s.toolError(id, "proposalId is required")
    }

    err := s.butler.memory.RejectProposal(proposalID, by, note)
    if err != nil {
        return s.toolError(id, err.Error())
    }

    // Create audit log
    s.butler.memory.AddAuditLog(memory.AuditLog{
        Action:     "reject",
        ActorType:  "human",
        ActorID:    by,
        EntityType: "proposal",
        EntityID:   proposalID,
        Metadata:   fmt.Sprintf(`{"reason":"%s"}`, note),
    })

    return success("❌ Proposal rejected")
}
```

---

## Validation & Testing

### Test Coverage

**File:** `apps/cli/internal/butler/mcp_test.go`

Tests added:

1. `TestMCPModeEnumValidation` - Mode enum values
2. `TestMCPAdminOnlyToolsFiltering` - Admin tool registry
3. `TestMCPToolsListFilteringByMode` - Tool list varies by mode
4. `TestMCPToolCallModeEnforcement` - Explicit error in agent mode
5. `TestMCPStoreDirectHumanMode` - store_direct works in human mode
6. `TestMCPApproveRejectHumanMode` - approve/reject work in human mode
7. `TestMCPServerModeGetter` - Mode accessor
8. `TestMCPDefaultModeIsAgent` - Default mode is agent

**Results:** ✅ All tests pass

**File:** `apps/cli/internal/memory/audit_test.go`

Tests added:

1. `TestAddAuditLog` - Basic audit creation
2. `TestGetAuditLogs` - Query by action/entity type
3. `TestGetAuditLogsForEntity` - Query by specific entity

**Results:** ✅ All tests pass

### Integration Tests

**File:** `apps/cli/internal/cli/commands/serve_test.go`

Test: `TestExecuteServeMissingIndex`

- Verifies serve command initializes with mode

**File:** `apps/cli/internal/cli/commands/store_test.go`

Test: `TestExecuteStoreSuccess`

- Verifies `--direct` creates audit log entry

### Manual Verification

```bash
# Agent mode (default)
palace serve
# Output: 🤖 MCP server starting in AGENT mode (restricted)

# Request tools list
# → store_direct NOT in list

# Try to call store_direct
# → Error: Tool "store_direct" not available in agent mode

# Human mode
palace serve --mode human
# Output: ⚠️  MCP server starting in HUMAN mode (full access)

# Request tools list
# → store_direct IS in list

# Call store_direct
# → Success, creates approved record + audit log
```

---

## Acceptance Criteria

| Criterion                                                                   | Status | Evidence                          |
| --------------------------------------------------------------------------- | ------ | --------------------------------- |
| `palace serve --mode agent` excludes admin tools                            | ✅     | Tool list filtered, tests pass    |
| `palace serve --mode human` includes all tools                              | ✅     | Tool list complete, tests pass    |
| Agent calling `store_direct` gets explicit error                            | ✅     | Error `-32602`, tests validate    |
| `--direct` writes always create audit entry                                 | ✅     | Audit log created, tests validate |
| Direct records have `authority=approved`, empty `promoted_from_proposal_id` | ✅     | Implementation verified           |

---

## Migration Impact

### Database Changes

- New `audit_log` table (non-breaking)
- Indexes for common audit queries

### API/CLI Impact

- **Additive:** New `palace serve --mode` flag
  - Default: agent mode (most restrictive)
  - Explicit human mode required for admin tools
- **Additive:** New MCP tools `store_direct`, `approve`, `reject`
  - Only available in human mode
- **Breaking:** None (new functionality only)

### Security Implications

- **Hardened:** Agents cannot bypass governance
- **Auditable:** All direct writes logged
- **Transparent:** Mode visible in server startup message
- **Defense in depth:**
  - Tool not in list (discovery prevention)
  - Tool call rejected (execution prevention)
  - Audit log (detection)

### Rollback Plan

```sql
DROP TABLE audit_log;
```

Code: Remove mode checks from `mcp.go`, revert `serve.go` changes

---

## Related Documentation

- [MCP Mode Design](../IMPLEMENTATION_PLAN_V2.1.md#phase-3-mcp-mode-gate--tool-segmentation)
- Decision DP-1: Human direct-write behavior

## Next Phase

Phase 4: Authoritative State Query Surface
