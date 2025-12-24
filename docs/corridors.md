# Corridors

Corridors enable knowledge sharing across Mind Palace workspaces. They allow learnings to flow between projects, so wisdom gained in one codebase can benefit all your work.

## Overview

```
~/.palace/                          # Global storage
├── config.yaml                     # Global settings
└── corridors/
    └── personal.db                 # Your personal learnings

~/project-a/.palace/                # Project A workspace
└── memory.db                       # Project A learnings

~/project-b/.palace/                # Project B workspace
└── memory.db                       # Project B learnings

# Corridors connect these, sharing knowledge
```

## Personal Corridor

The personal corridor stores learnings that follow you across all projects. Located at `~/.palace/corridors/personal.db`.

### View Personal Learnings

```bash
palace corridor personal
```

### Add Personal Learning

```bash
palace learn "Global best practice" --corridor
```

Or promote a workspace learning:
```bash
palace corridor promote LEARNING_ID
```

## Workspace Links

Link other workspaces to share learnings between projects.

### Link a Workspace

```bash
palace corridor link api-service ~/code/api-service
```

### List Links

```bash
palace corridor list
```

Output:
```
Personal Corridor: ~/.palace/corridors/personal.db
  Learnings: 15
  Avg Confidence: 0.82

Linked Workspaces:
  api-service    ~/code/api-service/.palace    (23 learnings)
  shared-lib     ~/code/shared-lib/.palace     (8 learnings)
  web-frontend   ~/code/web-frontend/.palace   (12 learnings)
```

### Unlink a Workspace

```bash
palace corridor unlink api-service
```

## Knowledge Flow

```
          ┌─────────────────────┐
          │  Personal Corridor  │
          │    (global DB)      │
          └────────┬────────────┘
                   │
          ┌────────┼────────┐
          │        │        │
          ▼        ▼        ▼
      ┌───────┐ ┌───────┐ ┌───────┐
      │Project│ │Project│ │Project│
      │   A   │ │   B   │ │   C   │
      └───────┘ └───────┘ └───────┘
```

### Promotion

High-confidence learnings can be promoted from workspace to personal corridor:

```bash
# Manually promote
palace corridor promote LEARNING_ID

# Automatic promotion (checks for high-confidence learnings)
palace corridor auto-promote
```

Promotion criteria:
- Confidence >= 0.8
- Use count >= 3
- Not already in personal corridor

### Search Across Corridors

```bash
# Search all linked workspaces
palace corridor search "error handling" --all

# Search specific workspace
palace corridor search "authentication" --workspace api-service
```

## Context Assembly

When using `get_context`, corridor learnings are included automatically:

```json
{
  "tool": "get_context",
  "arguments": {
    "query": "user authentication",
    "includeCorridors": true
  }
}
```

Response includes:
```json
{
  "files": [...],
  "symbols": [...],
  "learnings": [...],
  "corridorLearnings": [
    {
      "content": "Always hash passwords with bcrypt",
      "confidence": 0.95,
      "source": "personal"
    },
    {
      "content": "JWT tokens should expire within 24 hours",
      "confidence": 0.88,
      "source": "api-service"
    }
  ]
}
```

## Configuration

Global corridor configuration in `~/.palace/config.yaml`:

```yaml
corridors:
  personal: ~/.palace/corridors/personal.db
  links:
    api-service: /Users/me/code/api-service/.palace
    shared-lib: /Users/me/code/shared-lib/.palace

  settings:
    auto_promote: true
    min_confidence_for_promotion: 0.8
    min_use_count_for_promotion: 3
```

## Use Cases

### Solo Developer

Personal corridor carries your best practices between projects:

```bash
# In project A, learn something valuable
palace learn "Use structured logging for all errors"

# It gets promoted to personal corridor
palace corridor promote abc123

# In project B, it's automatically available
palace recall "logging"
# Returns the learning from personal corridor
```

### Team Knowledge Sharing

Share learnings with your team by committing the `.palace` directory:

```bash
# Project repository
.palace/
├── palace.jsonc      # Project config (commit this)
├── index/palace.db   # Code index (gitignore)
└── memory.db         # Learnings (commit this for team sharing)
```

Team members automatically get shared learnings when they clone.

### Cross-Project Patterns

Link related projects to share domain knowledge:

```bash
# Link your microservices
palace corridor link users-service ~/code/users-service
palace corridor link orders-service ~/code/orders-service
palace corridor link payments-service ~/code/payments-service

# Learnings from any service are searchable
palace corridor search "database migrations" --all
```

## Database Schema

Personal corridor schema (`~/.palace/corridors/personal.db`):

```sql
-- Learnings in personal corridor
CREATE TABLE learnings (
    id TEXT PRIMARY KEY,
    origin_workspace TEXT DEFAULT '',
    content TEXT NOT NULL,
    confidence REAL DEFAULT 0.5,
    source TEXT DEFAULT 'promoted',
    created_at TEXT NOT NULL,
    last_used TEXT NOT NULL,
    use_count INTEGER DEFAULT 0,
    tags TEXT DEFAULT '[]'
);

-- Linked workspace registry
CREATE TABLE links (
    name TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    added_at TEXT NOT NULL,
    last_accessed TEXT
);
```

## CLI Reference

| Command | Description |
|---------|-------------|
| `palace corridor list` | Show all corridors and links |
| `palace corridor personal` | Show personal corridor learnings |
| `palace corridor link NAME PATH` | Link a workspace |
| `palace corridor unlink NAME` | Remove a workspace link |
| `palace corridor promote ID` | Promote learning to personal |
| `palace corridor search QUERY` | Search across corridors |

## Best Practices

1. **Start personal**: Use the personal corridor for truly universal patterns
2. **Link related projects**: Connect microservices, libraries, and related codebases
3. **Let confidence guide promotion**: High-confidence learnings are more reliable
4. **Review periodically**: Clean up outdated learnings
5. **Share with team**: Commit workspace learnings to share team knowledge
