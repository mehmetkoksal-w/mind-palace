# Governance Quick Reference

This guide provides quick reference for using the governance features implemented in Plan V2.1.

## Overview

The governance system provides human-in-the-loop approval for knowledge updates, ensuring all recorded knowledge is validated before becoming authoritative.

## Authority States

All knowledge records (decisions, learnings, fragments, postmortems) have an `authority` field:

| State      | Description                | Visible to Agents |
| ---------- | -------------------------- | ----------------- |
| `proposed` | Pending review             | ❌ No             |
| `approved` | Verified and authoritative | ✅ Yes            |
| `rejected` | Declined during review     | ❌ No             |

**Key Principle**: Agents only see `authority=approved` records in all recall queries.

## MCP Modes

The MCP server operates in two modes:

| Mode    | Description         | Tool Access                     |
| ------- | ------------------- | ------------------------------- |
| `agent` | AI agent mode       | Read-only, creates proposals    |
| `human` | Human operator mode | Full access, can approve/reject |

### Mode-Based Tool Access

**Agent Mode Tools:**

- ✅ `recall`, `recall_decisions`, `recall_learnings`, etc. (read-only, approved records)
- ✅ `get_route` (returns navigation paths)
- ✅ `store` (creates proposals for review)
- ❌ `store_direct` (admin-only)
- ❌ `approve_proposal` (admin-only)
- ❌ `reject_proposal` (admin-only)

**Human Mode Tools:**

- ✅ All agent mode tools
- ✅ `store_direct` (bypass proposals for verified updates)
- ✅ `approve_proposal` (review and approve proposals)
- ✅ `reject_proposal` (review and reject proposals)

## CLI Commands

### Create Proposal

```bash
palace propose --type decision --content "Use PostgreSQL for database"
palace propose --type learning --content "Always validate JWTs" --confidence 0.9
```

### List Proposals

```bash
palace proposals                    # All proposals
palace proposals --status proposed  # Pending review
palace proposals --status approved  # Approved
palace proposals --status rejected  # Rejected
```

### Review Proposals

```bash
palace approve <proposal-id>        # Approve proposal
palace reject <proposal-id>         # Reject proposal
```

## MCP Tool Usage

### Store Knowledge (Creates Proposal)

```json
{
  "tool": "store",
  "arguments": {
    "type": "decision",
    "content": "Use JWT for authentication",
    "scope": "palace"
  }
}
```

This creates a proposal with `authority=proposed`. A human must approve it before agents can see it.

### Store Direct (Human Only)

```json
{
  "tool": "store_direct",
  "arguments": {
    "type": "decision",
    "content": "Use JWT for authentication",
    "scope": "palace"
  }
}
```

This creates a record with `authority=approved` immediately. Only available in human mode.

### Recall by ID

```json
{
  "tool": "recall_decisions",
  "arguments": {
    "id": "dec_abc123"
  }
}
```

Fetches a specific decision by ID. Only returns if `authority=approved`.

### Get Route

```json
{
  "tool": "get_route",
  "arguments": {
    "intent": "understand authentication",
    "scope": "palace"
  }
}
```

Returns route with `fetch_ref` for each node:

```json
{
  "nodes": [
    {
      "id": "dec_abc123",
      "kind": "decision",
      "fetch_ref": "recall_decisions --id dec_abc123",
      "score": 0.95
    },
    {
      "id": "lrn_xyz789",
      "kind": "learning",
      "fetch_ref": "recall --id lrn_xyz789",
      "score": 0.87
    }
  ]
}
```

Agent can follow `fetch_ref` to retrieve full content of each node.

## Workflow Examples

### Agent Discovery Flow

1. Agent calls `get_route` with intent
2. Receives route with `fetch_ref` for each node
3. Agent calls recall tool with `--id` parameter to fetch node content
4. Only `authority=approved` records are returned

### Human Approval Flow

1. Agent uses `store` tool to create proposal
2. Proposal stored with `authority=proposed`
3. Human reviews with `palace proposals`
4. Human approves with `palace approve <id>`
5. Record updated to `authority=approved`
6. Now visible to all agents via recall tools

### Human Direct Update Flow

1. Human uses `store_direct` MCP tool (or CLI equivalent)
2. Record created with `authority=approved` immediately
3. Visible to agents immediately
4. Use for human-verified information

## Scope Governance

All recall queries are bounded by scope:

```
palace > project > room
```

When querying scope `room`, results include:

- Records with scope=room
- Records with scope=project (inherited)
- Records with scope=palace (inherited)

Truncation ensures bounded result sets (default max 20 nodes per route).

## Determinism Guarantees

All queries produce deterministic, reproducible results:

- Consistent ordering by relevance score
- Stable tie-breaking by ID
- Fixed truncation limits
- Same input → same output (always)

## Database Migrations

The governance system uses migrations V4-V7:

| Migration | Purpose                                     |
| --------- | ------------------------------------------- |
| V4        | Add authority field to all knowledge tables |
| V5        | Create proposals table                      |
| V6        | Add MCP audit logging                       |
| V7        | Create authoritative state views            |

All migrations preserve existing data and provide rollback support.

## Best Practices

### For Agents

- ✅ Always use `store` to create proposals
- ✅ Use `get_route` to discover knowledge
- ✅ Follow `fetch_ref` to retrieve node content
- ✅ Trust that recall results are approved and authoritative

### For Humans

- ✅ Review proposals regularly (`palace proposals`)
- ✅ Use `store_direct` for verified information you add manually
- ✅ Reject proposals with incorrect or low-quality content
- ✅ Approve proposals that provide valuable, accurate knowledge

### For System Designers

- ✅ All recall queries filter by `authority=approved`
- ✅ All new records default to `authority=proposed`
- ✅ Use scope inheritance to organize knowledge hierarchies
- ✅ Implement audit logging for all MCP operations

## Troubleshooting

### "No results found" when agent recalls

- Check if records have `authority=approved`
- Use `palace proposals` to see pending proposals
- Approve relevant proposals

### "Tool not available" error

- Check MCP mode (agent vs human)
- Admin-only tools require human mode
- Verify tool name matches schema

### Duplicate proposals

- System uses dedupe keys to prevent duplicates
- Dedupe key = hash(type, content, scope)
- Updating existing record creates new proposal

## Testing

Run governance tests:

```bash
cd apps/cli
go test ./internal/butler -run Route     # Route tests
go test ./internal/memory -run Authority # Authority tests
go test ./internal/butler -run E2E       # E2E tests
```

All tests validate:

- ✅ Authority filtering
- ✅ Scope expansion
- ✅ Deterministic ordering
- ✅ fetch_ref generation
- ✅ Mode-based access control

## Documentation

- [Implementation Plan V2.1](../IMPLEMENTATION_PLAN_V2.1.md) - Complete plan
- [Phase Logs](../docs/implementation-logs/) - Detailed implementation logs
- [Governance Summary](../docs/implementation-logs/GOVERNANCE_COMPLETE_SUMMARY.md) - Completion summary
- [E2E Tests](../apps/cli/internal/butler/butler_route_e2e_test.go) - Test examples

## Support

For issues or questions:

1. Check implementation logs for design decisions
2. Review E2E tests for usage examples
3. Run tests to validate system state
4. Check CHANGELOG.md for recent changes
