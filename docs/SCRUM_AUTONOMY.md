# Mind Palace Agent Autonomy - Scrum Backlog

## Vision
Make AI agents truly autonomous when using Mind Palace by reducing manual tool calls, adding proactive behaviors, and streamlining the critical workflow.

---

## Epics

### E1: Composite Tools (Quick Wins)
Reduce cognitive load by combining frequently-used tool sequences.

### E2: Auto-Suggestions & Guidance
Help agents know what to do next without explicit instructions.

### E3: Lifecycle Automation
Automate session management and activity logging.

### E4: Proactive Notifications
Push information to agents instead of waiting for requests.

### E5: Relaxed Governance
Allow agents more autonomy for low-risk operations.

### E6: MCP Protocol Enhancements
Leverage MCP capabilities for better agent integration.

---

## Sprint 1: Foundation & Quick Wins
**Goal:** Immediate autonomy improvements with minimal risk

### S1.1 - Composite `session_init` Tool
**Epic:** E1 | **Points:** 3 | **Priority:** P0

**Description:**
Create a single tool that combines `session_start` + `brief` + `explore_rooms` into one call.

**Acceptance Criteria:**
- [ ] New tool `session_init` registered in MCP
- [ ] Returns combined output: session ID, briefing, rooms list
- [ ] Tool description emphasizes "CALL THIS FIRST"
- [ ] Existing individual tools remain available
- [ ] Unit tests pass

**Technical Notes:**
- Location: `apps/cli/internal/butler/mcp_tools_session.go`
- Reuse existing `toolSessionStart`, `toolBrief`, `toolExploreRooms` logic

---

### S1.2 - Composite `file_context` Tool
**Epic:** E1 | **Points:** 2 | **Priority:** P0

**Description:**
Create a tool that combines `context_auto_inject` + `session_conflict` for a file.

**Acceptance Criteria:**
- [ ] New tool `file_context` registered in MCP
- [ ] Takes `file_path` parameter
- [ ] Returns: auto-injected context + conflict check result
- [ ] Tool description: "CALL BEFORE EVERY FILE EDIT"
- [ ] Unit tests pass

---

### S1.3 - Add "Next Steps" to Tool Responses
**Epic:** E2 | **Points:** 3 | **Priority:** P1

**Description:**
Add a `next_steps` field to tool responses suggesting what agents should do next.

**Acceptance Criteria:**
- [ ] `session_init` suggests: "Now explore with `explore({intent: '...'})`"
- [ ] `file_context` suggests: "Ready to edit. After changes, call `session_log`"
- [ ] `store` suggests: "Consider linking to related decisions with `recall_link`"
- [ ] `session_end` suggests: "Session complete. Start new session for next task"
- [ ] Next steps are contextual (different suggestions based on state)

---

### S1.4 - Enhanced Tool Metadata
**Epic:** E2 | **Points:** 2 | **Priority:** P1

**Description:**
Add structured metadata to tool definitions for autonomy guidance.

**Acceptance Criteria:**
- [ ] Add `autonomy_level` field: "required" | "recommended" | "optional"
- [ ] Add `prerequisites` field: list of tools that should be called first
- [ ] Add `triggers` field: when this tool should be called
- [ ] Metadata included in `tools/list` response
- [ ] Documentation updated

**Schema:**
```json
{
  "name": "context_auto_inject",
  "autonomy": {
    "level": "required",
    "prerequisites": ["session_init"],
    "triggers": ["before_file_edit"],
    "frequency": "per_file"
  }
}
```

---

## Sprint 2: Lifecycle Automation
**Goal:** Reduce manual session and logging overhead

### S2.1 - Auto-Session Detection
**Epic:** E3 | **Points:** 5 | **Priority:** P0

**Description:**
Automatically create a session if agent calls any tool without an active session.

**Acceptance Criteria:**
- [ ] Track "current session" in MCP server state
- [ ] If no session exists and tool is called, auto-create session
- [ ] Auto-session uses agent name from first tool call or "unknown"
- [ ] Return session ID in response metadata
- [ ] Log warning: "Auto-created session. Call session_init for better tracking"
- [ ] Config option to disable auto-session

---

### S2.2 - Auto-Activity Logging
**Epic:** E3 | **Points:** 5 | **Priority:** P1

**Description:**
Automatically log activities when certain tools are called.

**Acceptance Criteria:**
- [ ] `file_context` auto-logs "file_focus" activity
- [ ] `explore` auto-logs "search" activity
- [ ] `store` auto-logs "knowledge_create" activity
- [ ] `recall` auto-logs "knowledge_query" activity
- [ ] Activity logging is transparent (doesn't change tool response)
- [ ] Config option to disable auto-logging

---

### S2.3 - Session Auto-End with Timeout
**Epic:** E3 | **Points:** 3 | **Priority:** P2

**Description:**
Automatically end sessions that have been inactive for too long.

**Acceptance Criteria:**
- [ ] Config option: `session_timeout_minutes` (default: 30)
- [ ] Background goroutine checks for stale sessions
- [ ] Stale sessions auto-ended with outcome "timeout"
- [ ] Summary includes: "Auto-ended due to inactivity"
- [ ] Notification in next tool response: "Previous session auto-ended"

---

### S2.4 - Session Cleanup on MCP Disconnect
**Epic:** E3 | **Points:** 2 | **Priority:** P1

**Description:**
Clean up sessions when MCP connection is closed.

**Acceptance Criteria:**
- [ ] Detect MCP stdin/stdout close
- [ ] End all active sessions for this connection
- [ ] Outcome: "disconnected"
- [ ] Log cleanup action

---

## Sprint 3: Proactive Intelligence
**Goal:** Push useful information to agents without explicit requests

### S3.1 - Conflict Detection Background Monitor
**Epic:** E4 | **Points:** 5 | **Priority:** P0

**Description:**
Monitor for file conflicts and include warnings in tool responses.

**Acceptance Criteria:**
- [ ] Track files touched by current session
- [ ] On each tool call, check if any tracked files have new conflicts
- [ ] Include `warnings` array in response if conflicts detected
- [ ] Warning format: "File {path} modified by {agent} since you last accessed it"
- [ ] Config option to enable/disable

---

### S3.2 - Proactive Briefing Updates
**Epic:** E4 | **Points:** 3 | **Priority:** P1

**Description:**
Include relevant briefing updates in tool responses when context changes.

**Acceptance Criteria:**
- [ ] Track "last briefing time" for session
- [ ] If new learnings/decisions added since last brief, include summary
- [ ] If new postmortems created, include alert
- [ ] Updates appear in `context_updates` field of responses
- [ ] Don't spam: max 1 update per 5 minutes

---

### S3.3 - Contradiction Pre-Check on Store
**Epic:** E4 | **Points:** 3 | **Priority:** P1

**Description:**
Before storing, check for contradictions and warn the agent.

**Acceptance Criteria:**
- [ ] When `store` is called, run contradiction check first
- [ ] If contradictions found, include in response (don't block)
- [ ] Suggest: "This may contradict {record_id}. Review before finalizing."
- [ ] Config option for contradiction sensitivity threshold

---

### S3.4 - Related Knowledge Suggestions
**Epic:** E4 | **Points:** 3 | **Priority:** P2

**Description:**
Suggest related learnings/decisions when agents access files.

**Acceptance Criteria:**
- [ ] `file_context` includes `related_knowledge` section
- [ ] Show learnings from similar files (same room, similar name)
- [ ] Show decisions that might apply (room/palace scope)
- [ ] Limit to top 3 most relevant
- [ ] Include relevance score

---

## Sprint 4: Governance Relaxation
**Goal:** Allow agents more autonomy for safe operations

### S4.1 - Auto-Approve High-Confidence Learnings
**Epic:** E5 | **Points:** 5 | **Priority:** P0

**Description:**
Automatically approve learnings with high confidence scores.

**Acceptance Criteria:**
- [ ] Config option: `auto_approve_threshold` (default: 0.85)
- [ ] Learnings with confidence >= threshold bypass proposal
- [ ] Audit log entry: "Auto-approved due to high confidence"
- [ ] Decisions still require human approval (governance)
- [ ] Agent mode respects this; human mode unaffected

---

### S4.2 - Agent-Accessible Linking
**Epic:** E5 | **Points:** 3 | **Priority:** P1

**Description:**
Allow agents to create links between records without admin mode.

**Acceptance Criteria:**
- [ ] New tool `suggest_link` (not admin-only)
- [ ] Creates link with `source: "agent"` and `status: "suggested"`
- [ ] Human can later approve/reject suggested links
- [ ] Suggested links visible but marked as unconfirmed
- [ ] Existing `recall_link` remains admin-only for confirmed links

---

### S4.3 - Agent-Accessible Obsolete Marking
**Epic:** E5 | **Points:** 2 | **Priority:** P2

**Description:**
Allow agents to mark learnings as potentially obsolete.

**Acceptance Criteria:**
- [ ] New tool `suggest_obsolete` (not admin-only)
- [ ] Marks learning with `obsolete_suggested: true` and reason
- [ ] Human can confirm or reject
- [ ] Suggested obsolete items shown with warning in recalls
- [ ] Existing `recall_obsolete` remains admin-only for confirmation

---

## Sprint 5: MCP Protocol Enhancements
**Goal:** Leverage advanced MCP capabilities

### S5.1 - Initialization Instructions
**Epic:** E6 | **Points:** 3 | **Priority:** P0

**Description:**
Send initialization instructions when agent connects.

**Acceptance Criteria:**
- [ ] On MCP `initialize` request, include `instructions` field
- [ ] Instructions contain critical workflow summary
- [ ] Include list of required tools with order
- [ ] Include workspace name and status
- [ ] Format compatible with MCP spec

**Response format:**
```json
{
  "protocolVersion": "2024-11-05",
  "serverInfo": {...},
  "instructions": "Mind Palace workflow: 1) session_init 2) file_context before edits..."
}
```

---

### S5.2 - Resource Templates for Context
**Epic:** E6 | **Points:** 3 | **Priority:** P1

**Description:**
Provide MCP resource templates for common context patterns.

**Acceptance Criteria:**
- [ ] `palace://init` - Returns session_init equivalent data
- [ ] `palace://context/{file_path}` - Returns file_context data
- [ ] `palace://status` - Returns current session status
- [ ] Resources are read-only (safe for agents to access anytime)
- [ ] Documentation updated

---

### S5.3 - Tool Annotations for Autonomy
**Epic:** E6 | **Points:** 2 | **Priority:** P2

**Description:**
Use MCP tool annotations to convey autonomy metadata.

**Acceptance Criteria:**
- [ ] Add `x-autonomy-level` annotation to tools
- [ ] Add `x-prerequisites` annotation
- [ ] Add `x-auto-trigger` annotation
- [ ] Annotations follow MCP extension conventions
- [ ] Compatible with standard MCP clients

---

## Sprint 6: Testing & Documentation
**Goal:** Ensure quality and adoption

### S6.1 - Integration Tests for Composite Tools
**Epic:** E1 | **Points:** 3 | **Priority:** P0

**Acceptance Criteria:**
- [ ] Test `session_init` returns all three components
- [ ] Test `file_context` returns context + conflict check
- [ ] Test error handling for each component
- [ ] Test with missing prerequisites

---

### S6.2 - Integration Tests for Lifecycle Automation
**Epic:** E3 | **Points:** 3 | **Priority:** P0

**Acceptance Criteria:**
- [ ] Test auto-session creation
- [ ] Test auto-activity logging
- [ ] Test session timeout
- [ ] Test cleanup on disconnect

---

### S6.3 - Update Agent Rules Files
**Epic:** E2 | **Points:** 2 | **Priority:** P1

**Acceptance Criteria:**
- [ ] Update CLAUDE.md with new composite tools
- [ ] Update .cursorrules with new workflow
- [ ] Update all rule files consistently
- [ ] Simplify instructions (fewer manual steps)

---

### S6.4 - Update Documentation Site
**Epic:** E2 | **Points:** 3 | **Priority:** P1

**Acceptance Criteria:**
- [ ] Update autonomous-agents.mdx with new tools
- [ ] Add "Autonomy Features" section
- [ ] Document configuration options
- [ ] Add migration guide from old workflow

---

## Backlog Summary

| Sprint | Focus | Stories | Total Points |
|--------|-------|---------|--------------|
| 1 | Foundation & Quick Wins | 4 | 10 |
| 2 | Lifecycle Automation | 4 | 15 |
| 3 | Proactive Intelligence | 4 | 14 |
| 4 | Governance Relaxation | 3 | 10 |
| 5 | MCP Protocol | 3 | 8 |
| 6 | Testing & Docs | 4 | 11 |
| **Total** | | **22** | **68** |

---

## Definition of Done

- [ ] Code implemented and compiles
- [ ] Unit tests written and passing
- [ ] Integration tests passing (if applicable)
- [ ] Code reviewed
- [ ] Documentation updated
- [ ] Agent rules files updated (if applicable)
- [ ] Feature flag/config option added (if applicable)
- [ ] No regressions in existing tests

---

## Configuration Schema (New Options)

```jsonc
// palace.jsonc additions
{
  "autonomy": {
    "autoSession": true,           // S2.1: Auto-create sessions
    "autoActivityLog": true,       // S2.2: Auto-log activities
    "sessionTimeoutMinutes": 30,   // S2.3: Auto-end timeout
    "conflictMonitoring": true,    // S3.1: Background conflict check
    "proactiveBriefing": true,     // S3.2: Include updates in responses
    "contradictionPreCheck": true, // S3.3: Check before store
    "autoApproveThreshold": 0.85,  // S4.1: Auto-approve learnings
    "suggestedLinksEnabled": true, // S4.2: Agent link suggestions
    "suggestedObsoleteEnabled": true // S4.3: Agent obsolete suggestions
  }
}
```

---

## Next Steps

1. Review and prioritize backlog
2. Estimate team velocity
3. Commit to Sprint 1
4. Create branch `plc-004` for Sprint 1 work
