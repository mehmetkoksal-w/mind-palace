# Mind Palace - Autonomous Agent Instructions

You have access to Mind Palace MCP tools for deterministic codebase context management.

## Core Workflow: Session → Brief → Context → Work → End

### 1. CRITICAL: Start Every Task with a Session

**BEFORE doing ANY work:**

```
1. Call session_start({agent_name: "gemini", task: "brief description"})
2. IMMEDIATELY call brief() to get workspace context
3. Review briefing for active agents, conflicts, learnings, hotspots
```

**WHY:** Sessions track your work, prevent conflicts with other agents, and enable workspace coordination.

### 2. CRITICAL: Get Context Before Every File Edit

**BEFORE editing ANY file:**

```
Call context_auto_inject({file_path: "path/to/file.ts"})
```

**WHY:** This provides:

- File-scoped learnings ("never use X in this file")
- File-scoped decisions ("this file must use Y pattern")
- Failure warnings ("this breaks 40% of the time")
- Hotspot alerts (frequently modified, high failure rate)

### 3. IMPORTANT: Check for Conflicts

**When editing files:**

```
Call session_conflict({path: "path/to/file.ts"})
```

**WHY:** Detects if another agent is working on the same file. Prevents merge conflicts and wasted work.

### 4. IMPORTANT: Store Knowledge as You Work

**After solving problems or making decisions:**

```
Call store({content: "description", as: "learning|decision|idea"})
```

**Examples:**

- Learning: "Authentication middleware should validate JWT expiration before checking permissions"
- Decision: "Use Zod for all API input validation to maintain consistency"
- Idea: "Consider caching user permissions in Redis to reduce DB queries"

**WHY:** Builds institutional knowledge. Future you (or other agents) can recall this context.

### 5. IMPORTANT: Log Your Activities

**Throughout the session, log what you're doing:**

```
session_log({
  activity: "file_edit|file_read|search|command|test",
  path: "path/to/file.ts",
  description: "what you did"
})
```

**WHY:** Creates audit trail. Helps debug issues and understand workflow patterns.

### 6. CRITICAL: End Your Session

**When task is complete:**

```
Call session_end({outcome: "success|partial|failed", summary: "what was accomplished"})
```

**WHY:** Releases locks, marks task complete, enables session analysis.

## Exploration & Discovery Tools

### When You Need to Find Code

**Search by intent/keywords:**

```
explore({intent: "authentication logic"})
```

**Explore by room (logical component):**

```
explore_rooms()  # List all rooms (e.g., "api", "auth", "database")
explore_context({room: "auth"})  # Get auth-related files
```

**Find symbol definitions:**

```
explore_symbol({query: "authenticateUser"})
```

**Understand impact:**

```
explore_impact({path: "auth/jwt.ts"})  # What depends on this file?
explore_callers({symbol: "validateToken", file: "auth/jwt.ts"})  # Who calls this?
```

## Knowledge Recall

### Before Implementing Features

**Check existing knowledge:**

```
recall({query: "authentication"})  # Find related learnings
recall_decisions({query: "validation", status: "active"})  # Find active decisions
recall_ideas({status: "exploring"})  # See ideas being explored
```

**Check for past failures:**

```
get_postmortems({severity: "high"})  # Learn from past mistakes
```

### When Working in Specific Areas

**Get room-scoped knowledge:**

```
recall({scope: "room", scopePath: "auth", query: "jwt"})
```

**Get file-scoped knowledge:**

```
recall({scope: "file", scopePath: "auth/jwt.ts"})
```

## Advanced Workflows

### Postmortem After Failures

**When you fix a significant bug:**

```
store_postmortem({
  title: "JWT expiration not validated",
  what_happened: "Tokens accepted even after expiration",
  root_cause: "Missing expiration check in middleware",
  lessons_learned: ["Always validate JWT expiration", "Add integration tests for expired tokens"],
  prevention_steps: ["Add expiration validation", "Add test coverage"],
  severity: "high",
  affected_files: ["middleware/auth.ts"]
})
```

**Then extract learnings:**

```
postmortem_to_learnings({postmortemId: "pm_abc123"})
```

### Cross-Workspace Knowledge (Corridor)

**If you have personal learnings across projects:**

```
corridor_learnings({query: "error handling"})  # Your personal best practices
```

**Promote workspace learning to personal corridor:**

```
corridor_promote({learningId: "l_abc123"})  # Make it available everywhere
```

## Priority System

- **CRITICAL**: Must do these (session_start, brief, context_auto_inject, session_end)
- **IMPORTANT**: Should do these (store, session_log, session_conflict)
- **RECOMMENDED**: Optional but valuable (recall, explore, postmortems)
- **ADMIN**: Human-mode only (approve, reject, store_direct)

## Anti-Patterns (DON'T DO THIS)

- Start work without calling session_start
- Edit files without calling context_auto_inject
- Forget to call session_end
- Work without calling brief first
- Ignore session_conflict warnings
- Store vague learnings ("this is good" - too generic)
- Never log activities (breaks audit trail)

## Best Practices

- Start with session_start, brief
- Get context before every file edit
- Check conflicts before editing
- Store specific, actionable learnings
- Log major activities
- End session when done
- Learn from postmortems
- Use explore tools to understand codebase
- Check decisions before implementing

## Example: Complete Workflow

```
1. session_start({agent_name: "gemini", task: "Add JWT refresh token support"})
2. brief()  # Get workspace overview
3. explore({intent: "authentication jwt"})  # Find relevant code
4. recall({query: "jwt authentication"})  # Check existing knowledge
5. get_postmortems({severity: "high"})  # Learn from past failures
6. context_auto_inject({file_path: "auth/jwt.ts"})  # Get file context
7. session_conflict({path: "auth/jwt.ts"})  # Check for conflicts
8. [Make changes to file]
9. session_log({activity: "file_edit", path: "auth/jwt.ts", description: "Added refresh token logic"})
10. store({content: "JWT refresh tokens should expire in 7 days, access tokens in 1 hour", as: "decision"})
11. session_end({outcome: "success", summary: "Added JWT refresh token support"})
```

## Scope Hierarchy (Most Specific Wins)

```
File (most specific) → Room → Palace (workspace) → Corridor (personal/global)
```

**When storing knowledge, choose appropriate scope:**

- File: "This specific file should never import from parent directories"
- Room: "All auth files must use bcrypt for password hashing"
- Palace: "Use TypeScript strict mode across entire project"
- Corridor: "Always validate input at API boundaries" (applies to all your projects)

## Semantic Search (If Embeddings Enabled)

```
search_semantic({query: "retry logic with exponential backoff"})  # Conceptual search
search_hybrid({query: "error handling"})  # Keyword + semantic (best of both)
search_similar({recordId: "l_abc123"})  # Find similar learnings
```

## Decision Lifecycle

```
1. Store idea: store({content: "...", as: "idea"})
2. Decide: store({content: "...", as: "decision", rationale: "..."})
3. Implement: [make changes]
4. Record outcome: recall_outcome({decisionId: "d_abc123", outcome: "success", note: "..."})
5. If failed: store_postmortem({related_decision: "d_abc123", ...})
```

## Conflict Resolution

**If session_conflict returns active sessions:**

1. Check what they're working on
2. Coordinate or wait
3. Don't force concurrent edits to same file

**If you must edit conflicting file:**

- Communicate intent via session_log
- Work quickly to minimize overlap
- End session ASAP to release lock

---

**Remember:** Mind Palace is your institutional memory. Use it to avoid repeating mistakes, preserve context, and enable autonomous coordination with other agents.
