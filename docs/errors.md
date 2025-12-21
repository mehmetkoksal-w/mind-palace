---
layout: default
title: Error Codes
nav_order: 14
---

# Shared Error Codes

Mind Palace ecosystem components use standardized error codes for consistent error handling across CLI, extension, and MCP integrations.

## Error Code Format

```
MP-{CATEGORY}{NUMBER}
```

- **MP**: Mind Palace prefix
- **CATEGORY**: Single letter (see below)
- **NUMBER**: 3-digit code

## Categories

| Category | Letter | Description |
|----------|--------|-------------|
| Configuration | C | Config file errors |
| Index | I | Database/indexing errors |
| Schema | S | Validation errors |
| File System | F | File access errors |
| MCP | M | Protocol errors |
| Runtime | R | Execution errors |

## Error Reference

### Configuration Errors (MP-Cxxx)

| Code | Message | Description | Resolution |
|------|---------|-------------|------------|
| MP-C001 | Config not found | `.palace/palace.jsonc` missing | Run `palace init` |
| MP-C002 | Invalid config | Config fails schema validation | Check schema errors in output |
| MP-C003 | Missing required field | Required field not present | Add missing field to config |
| MP-C010 | Room not found | Referenced room doesn't exist | Create room or fix reference |
| MP-C011 | Playbook not found | Referenced playbook doesn't exist | Create playbook or fix reference |
| MP-C020 | Invalid neighbor config | Corridor neighbor misconfigured | Check URL/path and auth settings |

### Index Errors (MP-Ixxx)

| Code | Message | Description | Resolution |
|------|---------|-------------|------------|
| MP-I001 | Index not found | `.palace/index/palace.db` missing | Run `palace scan` |
| MP-I002 | Index stale | Files changed since last scan | Run `palace scan` |
| MP-I003 | Index corrupted | Database integrity check failed | Delete `.palace/index/` and rescan |
| MP-I010 | Scan failed | Error during indexing | Check file permissions |
| MP-I020 | FTS query error | Invalid search query | Simplify query syntax |

### Schema Errors (MP-Sxxx)

| Code | Message | Description | Resolution |
|------|---------|-------------|------------|
| MP-S001 | Schema version mismatch | Artifact version != CLI version | Upgrade CLI or regenerate artifacts |
| MP-S002 | Validation failed | Artifact doesn't match schema | Fix artifact per error details |
| MP-S003 | Unknown schema | Schema not embedded in CLI | Check artifact `kind` field |
| MP-S010 | Invalid JSON | Malformed JSON/JSONC | Fix syntax error at reported location |
| MP-S011 | Invalid JSONC | Malformed JSONC syntax | Check for unclosed comments |

### File System Errors (MP-Fxxx)

| Code | Message | Description | Resolution |
|------|---------|-------------|------------|
| MP-F001 | File not found | Referenced file doesn't exist | Update reference or create file |
| MP-F002 | Permission denied | Cannot read/write file | Check file permissions |
| MP-F003 | Path outside workspace | Path escapes project root | Use relative paths within project |
| MP-F010 | Guardrail violation | File matches do-not-touch pattern | Remove from guardrails or skip file |

### MCP Errors (MP-Mxxx)

| Code | Message | Description | Resolution |
|------|---------|-------------|------------|
| MP-M001 | Server not running | `palace serve` not started | Start MCP server |
| MP-M002 | Connection lost | MCP connection dropped | Reconnect or restart server |
| MP-M003 | Request timeout | MCP request took too long | Check server status, retry |
| MP-M010 | Unknown tool | Tool name not recognized | Check available tools |
| MP-M011 | Invalid arguments | Tool arguments malformed | Check tool schema |
| MP-M020 | Protocol error | Invalid JSON-RPC message | Check message format |

### Runtime Errors (MP-Rxxx)

| Code | Message | Description | Resolution |
|------|---------|-------------|------------|
| MP-R001 | Binary not found | `palace` not in PATH | Install CLI or set path |
| MP-R002 | Version mismatch | CLI/Extension incompatible | Upgrade to compatible versions |
| MP-R003 | Workspace not open | No workspace folder | Open a folder in VS Code |
| MP-R010 | Command failed | CLI command returned error | Check command output |
| MP-R020 | Git not available | Git commands failed | Install git or check PATH |

## Exit Codes

The CLI uses these exit codes:

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Configuration error |
| 3 | Validation error |
| 4 | Index error (stale/missing) |
| 5 | File system error |

## Structured Error Output

When `--json` flag is used (where supported), errors are returned as:

```json
{
  "error": {
    "code": "MP-I002",
    "message": "Index stale",
    "details": {
      "staleFiles": ["src/main.ts", "src/utils.ts"],
      "lastScan": "2025-01-15T10:30:00Z"
    }
  }
}
```

## Extension Error Handling

The VS Code extension maps error codes to user-friendly notifications:

| Error Code | Notification Type | Action Offered |
|------------|-------------------|----------------|
| MP-C001 | Warning | "Run palace init" button |
| MP-I002 | Info | "Heal" button |
| MP-R001 | Error | "Configure path" link |
| MP-R002 | Warning | Version mismatch details |

## Adding New Error Codes

When adding new errors:

1. Choose appropriate category
2. Use next available number in category
3. Add to this document
4. Update CLI error definitions
5. Update extension error mapping (if user-facing)
