# Mind Palace - GitHub Copilot Instructions

This project uses Mind Palace for deterministic codebase context management. When you have access to Mind Palace MCP tools, follow these instructions.

## Core Workflow: Session → Brief → Context → Work → End

### 1. CRITICAL: Start Every Task with a Session

**BEFORE doing ANY work:**

```
1. Call session_start({agent_name: "copilot", task: "brief description"})
2. IMMEDIATELY call brief() to get workspace context
3. Review briefing for active agents, conflicts, learnings, hotspots
```

Sessions track your work, prevent conflicts with other agents, and enable workspace coordination.

### 2. CRITICAL: Get Context Before Every File Edit

**BEFORE editing ANY file:**

```
Call context_auto_inject({file_path: "path/to/file.ts"})
```

This provides:

- File-scoped learnings ("never use X in this file")
- File-scoped decisions ("this file must use Y pattern")
- Failure warnings ("this breaks 40% of the time")
- Hotspot alerts (frequently modified, high failure rate)

### 3. IMPORTANT: Check for Conflicts

**When editing files:**

```
Call session_conflict({path: "path/to/file.ts"})
```

Detects if another agent is working on the same file. Prevents merge conflicts and wasted work.

### 4. IMPORTANT: Store Knowledge as You Work

**After solving problems or making decisions:**

```
Call store({content: "description", as: "learning|decision|idea"})
```

Examples:

- Learning: "Authentication middleware should validate JWT expiration before checking permissions"
- Decision: "Use Zod for all API input validation to maintain consistency"
- Idea: "Consider caching user permissions in Redis to reduce DB queries"

### 5. IMPORTANT: Log Your Activities

**Throughout the session, log what you're doing:**

```
session_log({
  activity: "file_edit|file_read|search|command|test",
  path: "path/to/file.ts",
  description: "what you did"
})
```

### 6. CRITICAL: End Your Session

**When task is complete:**

```
Call session_end({outcome: "success|partial|failed", summary: "what was accomplished"})
```

## Exploration & Discovery Tools

### When You Need to Find Code

```
explore({intent: "authentication logic"})          # Search by intent
explore_rooms()                                     # List all rooms
explore_context({room: "auth"})                    # Get room context
explore_symbol({query: "authenticateUser"})        # Find symbol
explore_impact({path: "auth/jwt.ts"})              # Analyze impact
explore_callers({symbol: "validateToken"})         # Find callers
```

## Knowledge Recall

### Before Implementing Features

```
recall({query: "authentication"})                   # Find related learnings
recall_decisions({query: "validation"})             # Find active decisions
get_postmortems({severity: "high"})                # Learn from past failures
```

### Scoped Knowledge

```
recall({scope: "room", scopePath: "auth", query: "jwt"})    # Room-scoped
recall({scope: "file", scopePath: "auth/jwt.ts"})           # File-scoped
```

## Advanced Workflows

### Postmortem After Failures

```
store_postmortem({
  title: "JWT expiration not validated",
  what_happened: "Tokens accepted even after expiration",
  root_cause: "Missing expiration check in middleware",
  lessons_learned: ["Always validate JWT expiration"],
  prevention_steps: ["Add expiration validation"],
  severity: "high",
  affected_files: ["middleware/auth.ts"]
})
```

### Cross-Workspace Knowledge (Corridor)

```
corridor_learnings({query: "error handling"})      # Personal best practices
corridor_promote({learningId: "l_abc123"})         # Make available everywhere
```

## Priority System

- **CRITICAL**: Must do (session_start, brief, context_auto_inject, session_end)
- **IMPORTANT**: Should do (store, session_log, session_conflict)
- **RECOMMENDED**: Optional but valuable (recall, explore, postmortems)
- **ADMIN**: Human-mode only (approve, reject, store_direct)

## Anti-Patterns

- Start work without calling session_start
- Edit files without calling context_auto_inject
- Forget to call session_end
- Work without calling brief first
- Ignore session_conflict warnings
- Store vague learnings ("this is good" - too generic)

## Example: Complete Workflow

```
1. session_start({agent_name: "copilot", task: "Add JWT refresh token support"})
2. brief()
3. explore({intent: "authentication jwt"})
4. recall({query: "jwt authentication"})
5. get_postmortems({severity: "high"})
6. context_auto_inject({file_path: "auth/jwt.ts"})
7. session_conflict({path: "auth/jwt.ts"})
8. [Make changes to file]
9. session_log({activity: "file_edit", path: "auth/jwt.ts", description: "Added refresh token logic"})
10. store({content: "JWT refresh tokens should expire in 7 days", as: "decision"})
11. session_end({outcome: "success", summary: "Added JWT refresh token support"})
```

## Scope Hierarchy

```
File (most specific) → Room → Palace (workspace) → Corridor (personal/global)
```

When storing knowledge, choose the narrowest applicable scope.

---

**Remember:** Mind Palace is your institutional memory. Use it to avoid repeating mistakes, preserve context, and enable autonomous coordination with other agents.
