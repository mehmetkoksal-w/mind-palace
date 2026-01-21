# Mind Palace - Autonomous Agent Instructions

You have access to Mind Palace MCP tools for deterministic codebase context management.

## Quick Start: Two Essential Commands

### 1. START HERE: `session_init`

**Call this FIRST, before doing anything else:**

```
session_init({agent_name: "claude-code", task: "brief description"})
```

This single call:
- Starts your session (gets you a session ID)
- Provides workspace briefing (active agents, learnings, hotspots)
- Shows project structure (rooms and entry points)
- Gives you next steps guidance

### 2. BEFORE EVERY FILE EDIT: `file_context`

**Call this before editing ANY file:**

```
file_context({file_path: "path/to/file.ts", session_id: "your-session-id"})
```

This single call:
- Checks for conflicts (is another agent editing this?)
- Provides file-scoped learnings and decisions
- Shows known failures and warnings
- Gives file history and failure rate

## Complete Workflow

```
1. session_init({agent_name: "claude-code", task: "Add JWT refresh token support"})
   → Get session ID, workspace context, project structure

2. explore({intent: "authentication jwt"})
   → Find relevant code

3. file_context({file_path: "auth/jwt.ts", session_id: "..."})
   → Get file context before editing

4. [Make your changes]

5. session_log({activity: "file_edit", path: "auth/jwt.ts", description: "Added refresh token logic"})
   → Log what you did

6. store({content: "JWT refresh tokens should expire in 7 days", as: "decision"})
   → Save knowledge for future

7. session_end({sessionId: "...", outcome: "success", summary: "Added JWT refresh token support"})
   → End session when done
```

## Knowledge Management

### Store Knowledge as You Work

```
store({content: "description", as: "learning|decision|idea"})
```

**Examples:**
- Learning: "Authentication middleware should validate JWT expiration before checking permissions"
- Decision: "Use Zod for all API input validation to maintain consistency"
- Idea: "Consider caching user permissions in Redis to reduce DB queries"

### Recall Existing Knowledge

```
recall({query: "authentication"})           # Find related learnings
recall_decisions({query: "validation"})     # Find active decisions
get_postmortems({severity: "high"})         # Learn from past failures
```

## Exploration Tools

```
explore({intent: "authentication logic"})   # Search by intent
explore_rooms()                              # List project structure
explore_symbol({query: "authenticateUser"}) # Find symbol
explore_impact({path: "auth/jwt.ts"})       # Analyze change impact
```

## Priority System

- **CRITICAL**: `session_init` (first), `file_context` (before edits), `session_end` (last)
- **IMPORTANT**: `store`, `session_log`
- **RECOMMENDED**: `recall`, `explore`, `get_postmortems`

## Anti-Patterns

- Starting work without `session_init`
- Editing files without `file_context`
- Forgetting to call `session_end`
- Ignoring conflict warnings
- Storing vague learnings ("this is good" - too generic)

## Scope Hierarchy

```
File (most specific) → Room → Palace (workspace) → Corridor (personal/global)
```

Choose the narrowest applicable scope when storing knowledge.

---

**Remember:** Mind Palace is your institutional memory. Use it to avoid repeating mistakes, preserve context, and coordinate with other agents.
