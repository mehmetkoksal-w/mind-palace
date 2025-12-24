# Session Memory

Session Memory is Mind Palace's system for tracking agent work sessions, activities, learnings, and file intelligence. It enables agents to learn from past experiences and share knowledge across sessions.

## Overview

Session Memory provides:
- **Session Tracking**: Record agent sessions with goals and outcomes
- **Activity Logging**: Track file reads, edits, searches, and commands
- **Learnings**: Capture and recall patterns, heuristics, and best practices
- **File Intelligence**: Track edit history, failure rates, and file-specific learnings
- **Multi-Agent Coordination**: Detect conflicts when multiple agents work on the same files

## Sessions

A session represents a single agent's work period in your codebase.

### Starting a Session

```bash
palace session start --agent claude-code --goal "Implement user authentication"
```

Or via MCP:
```json
{
  "tool": "start_session",
  "arguments": {
    "agentType": "claude-code",
    "goal": "Implement user authentication"
  }
}
```

### Session States

- `active`: Session is currently running
- `completed`: Session ended successfully
- `abandoned`: Session was terminated unexpectedly

### Listing Sessions

```bash
# List all sessions
palace session list

# List only active sessions
palace session list --active

# Limit results
palace session list --limit 10
```

### Ending a Session

```bash
palace session end SESSION_ID
```

## Activities

Activities record what agents do during sessions.

### Activity Types

- `file_read`: Agent read a file
- `file_edit`: Agent modified a file
- `search`: Agent searched for symbols or patterns
- `command`: Agent ran a shell command

### Logging Activities

Via MCP:
```json
{
  "tool": "log_activity",
  "arguments": {
    "sessionId": "session-123",
    "kind": "file_edit",
    "target": "src/auth/login.go",
    "outcome": "success",
    "details": {"linesChanged": 25}
  }
}
```

### Viewing Activities

```bash
# Recent activities
palace activity --limit 20

# Activities for a specific session
palace activity --session SESSION_ID

# Activities for a specific file
palace activity --file src/main.go
```

## Learnings

Learnings capture knowledge that emerged from agent work.

### Scope Levels

- `file`: Applies to a specific file
- `room`: Applies to a logical module/directory
- `palace`: Applies to the entire project
- `corridor`: Applies across projects (via corridors)

### Adding Learnings

```bash
# Add a project-wide learning
palace learn "Always run tests before committing"

# Add a file-specific learning
palace learn "This file handles rate limiting" --scope file --path src/middleware/rate.go

# Add a room-level learning
palace learn "Use prepared statements for all queries" --scope room --path database
```

Via MCP:
```json
{
  "tool": "add_learning",
  "arguments": {
    "content": "This module requires special error handling",
    "scope": "room",
    "scopePath": "payment-processing",
    "confidence": 0.85
  }
}
```

### Recalling Learnings

```bash
# Search for relevant learnings
palace recall "authentication"

# Get learnings for a specific scope
palace recall --scope file --path auth/login.go
```

### Confidence

Learnings have a confidence score (0.0 to 1.0) that increases with reinforcement:
- Initial confidence: 0.5
- User-provided learnings start at 0.8
- Reinforced learnings increase towards 1.0

## File Intelligence

File intelligence tracks the history of changes to each file.

### Metrics Tracked

- **Edit Count**: Number of times the file was modified
- **Failure Count**: Number of times edits led to failures
- **Last Editor**: Which agent last modified the file
- **Associated Learnings**: Learnings specific to this file

### Viewing File Intelligence

```bash
palace intel src/auth/login.go
```

Output:
```
File: src/auth/login.go
Edit Count: 15
Failure Count: 2
Failure Rate: 13.3%
Last Editor: claude-code
Last Edited: 2024-01-15 14:30:00

Associated Learnings:
- Requires integration tests after changes (confidence: 0.9)
- Sensitive security code - review carefully (confidence: 0.85)
```

### Hotspots and Fragile Files

```bash
# See most frequently edited files
palace hotspots

# See files with high failure rates
palace fragile
```

## Multi-Agent Coordination

When multiple agents work on the same codebase, Mind Palace helps prevent conflicts.

### Active Agents

```bash
palace agents
```

### Conflict Detection

Via MCP:
```json
{
  "tool": "check_conflict",
  "arguments": {
    "path": "src/auth/login.go"
  }
}
```

Response when conflict detected:
```json
{
  "hasConflict": true,
  "conflict": {
    "path": "src/auth/login.go",
    "otherSession": "session-456",
    "otherAgent": "cursor",
    "lastTouched": "2024-01-15T14:30:00Z",
    "severity": "warning"
  }
}
```

## Briefing

Get a quick summary before starting work:

```bash
palace brief
```

Or for a specific file:

```bash
palace brief src/auth/login.go
```

The briefing includes:
- Active agents in the workspace
- Potential conflicts for the file
- Relevant learnings
- File intelligence
- Recent hotspots

## Database Schema

Session memory is stored in `.palace/memory.db` (SQLite):

```sql
-- Sessions
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    agent_type TEXT NOT NULL,
    agent_id TEXT DEFAULT '',
    goal TEXT DEFAULT '',
    started_at TEXT NOT NULL,
    last_activity TEXT NOT NULL,
    state TEXT DEFAULT 'active',
    summary TEXT DEFAULT ''
);

-- Activities
CREATE TABLE activities (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    target TEXT DEFAULT '',
    details TEXT DEFAULT '{}',
    timestamp TEXT NOT NULL,
    outcome TEXT DEFAULT 'unknown'
);

-- Learnings
CREATE TABLE learnings (
    id TEXT PRIMARY KEY,
    session_id TEXT DEFAULT '',
    scope TEXT NOT NULL,
    scope_path TEXT DEFAULT '',
    content TEXT NOT NULL,
    confidence REAL DEFAULT 0.5,
    source TEXT DEFAULT 'agent',
    created_at TEXT NOT NULL,
    last_used TEXT NOT NULL,
    use_count INTEGER DEFAULT 0
);

-- File Intelligence
CREATE TABLE file_intel (
    path TEXT PRIMARY KEY,
    edit_count INTEGER DEFAULT 0,
    last_edited TEXT,
    last_editor TEXT DEFAULT '',
    failure_count INTEGER DEFAULT 0
);
```

## MCP Tools Reference

| Tool | Description |
|------|-------------|
| `start_session` | Start a new agent session |
| `end_session` | End an active session |
| `log_activity` | Log an activity in a session |
| `record_outcome` | Record session outcome |
| `add_learning` | Add a new learning |
| `get_learnings` | Retrieve learnings by scope |
| `get_file_intel` | Get intelligence for a file |
| `get_activity` | Get recent activities |
| `check_conflict` | Check for conflicts on a file |
| `get_active_agents` | List active agents |

## Best Practices

1. **Start sessions with clear goals**: Helps track what was accomplished
2. **Log activities consistently**: Better file intelligence and conflict detection
3. **Add learnings as you discover patterns**: Knowledge compounds over time
4. **Check for conflicts before editing shared files**: Prevent merge conflicts
5. **Review briefings before starting work**: Stay informed about recent changes
