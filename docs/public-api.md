---
layout: default
title: Public API
nav_order: 14
---

# Public API

Mind Palace provides public Go packages that external tools can import to interact with workspace memory and corridors.

## Installation

```sh
go get github.com/mehmetkoksal-w/mind-palace/pkg/memory
go get github.com/mehmetkoksal-w/mind-palace/pkg/corridor
go get github.com/mehmetkoksal-w/mind-palace/pkg/types
```

## Packages

| Package | Description |
|---------|-------------|
| `pkg/types` | Shared type definitions and constants |
| `pkg/memory` | Workspace session memory API |
| `pkg/corridor` | Global corridor (cross-workspace) API |

---

## pkg/types

Shared type definitions used across memory and corridor packages.

### Types

```go
import "github.com/mehmetkoksal-w/mind-palace/pkg/types"

// Session represents an agent work session
type Session struct {
    ID           string
    AgentType    string    // "claude-code", "cursor", "aider", etc.
    AgentID      string    // Unique agent instance identifier
    Goal         string    // What the agent is trying to accomplish
    StartedAt    time.Time
    LastActivity time.Time
    State        string    // "active", "completed", "abandoned"
    Summary      string
}

// Activity represents an action taken during a session
type Activity struct {
    ID        string
    SessionID string
    Kind      string    // "file_read", "file_edit", "search", "command"
    Target    string    // File path or search query
    Details   string    // JSON with specifics
    Timestamp time.Time
    Outcome   string    // "success", "failure", "unknown"
}

// Learning represents an emerged pattern or heuristic
type Learning struct {
    ID         string
    SessionID  string    // Optional
    Scope      string    // "file", "room", "palace", "corridor"
    ScopePath  string    // e.g., "auth/login.go" or "auth"
    Content    string    // The learning itself
    Confidence float64   // 0.0-1.0
    Source     string    // "agent", "user", "inferred"
    CreatedAt  time.Time
    LastUsed   time.Time
    UseCount   int
}

// FileIntel represents intelligence about a file
type FileIntel struct {
    Path         string
    EditCount    int
    LastEdited   time.Time
    LastEditor   string
    FailureCount int
    Learnings    []string
}

// PersonalLearning represents a learning in the personal corridor
type PersonalLearning struct {
    ID              string
    OriginWorkspace string
    Content         string
    Confidence      float64
    Source          string
    CreatedAt       time.Time
    LastUsed        time.Time
    UseCount        int
    Tags            []string
}

// LinkedWorkspace represents a linked workspace
type LinkedWorkspace struct {
    Name         string
    Path         string
    AddedAt      time.Time
    LastAccessed time.Time
}

// Conflict represents a potential conflict between agents
type Conflict struct {
    Path         string
    OtherSession string
    OtherAgent   string
    LastTouched  time.Time
    Severity     string    // "warning", "critical"
}
```

### Constants

```go
// Activity kinds
const (
    ActivityFileRead  = "file_read"
    ActivityFileEdit  = "file_edit"
    ActivitySearch    = "search"
    ActivityCommand   = "command"
)

// Session states
const (
    SessionActive    = "active"
    SessionCompleted = "completed"
    SessionAbandoned = "abandoned"
)

// Learning scopes
const (
    ScopeFile     = "file"
    ScopeRoom     = "room"
    ScopePalace   = "palace"
    ScopeCorridor = "corridor"
)

// Learning sources
const (
    SourceAgent    = "agent"
    SourceUser     = "user"
    SourceInferred = "inferred"
    SourcePromoted = "promoted"
)

// Activity outcomes
const (
    OutcomeSuccess = "success"
    OutcomeFailure = "failure"
    OutcomeUnknown = "unknown"
)
```

---

## pkg/memory

Workspace session memory API. Tracks sessions, activities, learnings, and file intelligence.

### Opening Memory

```go
import "github.com/mehmetkoksal-w/mind-palace/pkg/memory"

// Open workspace memory (creates .palace/memory.db if needed)
mem, err := memory.Open("/path/to/workspace")
if err != nil {
    log.Fatal(err)
}
defer mem.Close()
```

### Sessions

```go
// Start a new session
session, err := mem.StartSession("my-agent", "instance-123", "Implement authentication")

// Get session by ID
session, err := mem.GetSession(sessionID)

// List sessions
sessions, err := mem.ListSessions(true, 10)  // activeOnly=true, limit=10

// End session
err := mem.EndSession(sessionID, memory.SessionCompleted, "Authentication implemented")
```

### Activities

```go
// Log an activity
err := mem.LogActivity(session.ID, memory.Activity{
    Kind:    memory.ActivityFileEdit,
    Target:  "auth/login.go",
    Details: `{"lines_changed": 45}`,
    Outcome: memory.OutcomeSuccess,
})

// Get activities for a session
activities, err := mem.GetActivities(sessionID, "", 50)

// Get activities for a specific file
activities, err := mem.GetActivities("", "auth/login.go", 50)
```

### Learnings

```go
// Add a learning
id, err := mem.AddLearning(memory.Learning{
    Scope:      memory.ScopeFile,
    ScopePath:  "auth/login.go",
    Content:    "Always validate JWT expiration before processing",
    Confidence: 0.8,
    Source:     memory.SourceAgent,
})

// Get learnings by scope
learnings, err := mem.GetLearnings("file", "auth/login.go", 10)

// Get learnings for palace scope
learnings, err := mem.GetLearnings("palace", "", 10)

// Search learnings by content
learnings, err := mem.SearchLearnings("authentication", 10)

// Get relevant learnings for a file and room
learnings, err := mem.GetRelevantLearnings("auth/login.go", "authentication", 10)

// Reinforce a learning (increase confidence)
err := mem.ReinforceLearning(learningID)
```

### File Intelligence

```go
// Record a file edit
err := mem.RecordFileEdit("auth/login.go", "claude-code")

// Record a failure related to a file
err := mem.RecordFileFailure("auth/login.go")

// Get file intelligence
intel, err := mem.GetFileIntel("auth/login.go")
fmt.Printf("Edit count: %d, Failures: %d\n", intel.EditCount, intel.FailureCount)

// Get file hotspots (most edited files)
hotspots, err := mem.GetFileHotspots(10)
```

### Conflict Detection

```go
// Check if another agent is working on the same file
conflict, err := mem.CheckConflict(mySessionID, "auth/login.go")
if conflict != nil {
    fmt.Printf("Warning: %s is also editing this file\n", conflict.OtherAgent)
}
```

### Complete Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/mehmetkoksal-w/mind-palace/pkg/memory"
)

func main() {
    // Open workspace memory
    mem, err := memory.Open("/home/user/myproject")
    if err != nil {
        log.Fatal(err)
    }
    defer mem.Close()

    // Start a session
    session, err := mem.StartSession("my-agent", "agent-001", "Fix authentication bug")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Started session: %s\n", session.ID)

    // Log reading a file
    mem.LogActivity(session.ID, memory.Activity{
        Kind:    memory.ActivityFileRead,
        Target:  "auth/jwt.go",
        Outcome: memory.OutcomeSuccess,
    })

    // Check for conflicts before editing
    conflict, _ := mem.CheckConflict(session.ID, "auth/jwt.go")
    if conflict != nil {
        fmt.Printf("Warning: %s is working on this file\n", conflict.OtherAgent)
    }

    // Log editing a file
    mem.LogActivity(session.ID, memory.Activity{
        Kind:    memory.ActivityFileEdit,
        Target:  "auth/jwt.go",
        Outcome: memory.OutcomeSuccess,
    })
    mem.RecordFileEdit("auth/jwt.go", "my-agent")

    // Add a learning
    mem.AddLearning(memory.Learning{
        Scope:      memory.ScopeFile,
        ScopePath:  "auth/jwt.go",
        Content:    "JWT secret must be at least 256 bits",
        Confidence: 0.9,
        Source:     memory.SourceAgent,
    })

    // End session
    mem.EndSession(session.ID, memory.SessionCompleted, "Fixed JWT validation bug")
}
```

---

## pkg/corridor

Global corridor API for cross-workspace knowledge sharing.

### Opening Corridor

```go
import "github.com/mehmetkoksal-w/mind-palace/pkg/corridor"

// Open global corridor (~/.palace/corridors/personal.db)
cor, err := corridor.OpenGlobal()
if err != nil {
    log.Fatal(err)
}
defer cor.Close()
```

### Personal Learnings

```go
// Add a personal learning (available across all workspaces)
err := cor.AddPersonalLearning(corridor.PersonalLearning{
    Content:         "Always use context.Context for cancellation",
    Confidence:      0.9,
    Source:          "promoted",
    OriginWorkspace: "api-service",
    Tags:            []string{"go", "best-practice"},
})

// Get personal learnings
learnings, err := cor.GetPersonalLearnings("", 20)  // query="", limit=20

// Search personal learnings
learnings, err := cor.GetPersonalLearnings("context", 10)

// Reinforce a learning
err := cor.ReinforceLearning(learningID)

// Delete a learning
err := cor.DeleteLearning(learningID)
```

### Workspace Links

```go
// Link another workspace
err := cor.Link("api-service", "/home/user/api-service")

// List linked workspaces
links, err := cor.GetLinks()
for _, link := range links {
    fmt.Printf("%s -> %s\n", link.Name, link.Path)
}

// Get learnings from a specific linked workspace
learnings, err := cor.GetLinkedLearnings("api-service", 10)

// Get learnings from all linked workspaces
learnings, err := cor.GetAllLinkedLearnings(20)

// Unlink a workspace
err := cor.Unlink("api-service")
```

### Statistics

```go
stats, err := cor.Stats()
fmt.Printf("Learnings: %v\n", stats["learningCount"])
fmt.Printf("Linked workspaces: %v\n", stats["linkedWorkspaces"])
fmt.Printf("Average confidence: %v\n", stats["averageConfidence"])
```

### Utility Functions

```go
// Get global palace path
path, err := corridor.GlobalPath()  // Returns ~/.palace

// Ensure global directory structure exists
path, err := corridor.EnsureGlobalLayout()
```

### Complete Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/mehmetkoksal-w/mind-palace/pkg/corridor"
)

func main() {
    // Open global corridor
    cor, err := corridor.OpenGlobal()
    if err != nil {
        log.Fatal(err)
    }
    defer cor.Close()

    // Add a personal learning
    cor.AddPersonalLearning(corridor.PersonalLearning{
        Content:         "Use structured logging in production",
        Confidence:      0.85,
        Source:          "user",
        OriginWorkspace: "backend",
        Tags:            []string{"logging", "production"},
    })

    // Link another workspace
    cor.Link("frontend", "/home/user/frontend-app")

    // Get all linked workspaces
    links, _ := cor.GetLinks()
    fmt.Println("Linked workspaces:")
    for _, link := range links {
        fmt.Printf("  %s: %s\n", link.Name, link.Path)
    }

    // Search across all corridors
    learnings, _ := cor.GetPersonalLearnings("logging", 10)
    fmt.Println("\nLearnings about logging:")
    for _, l := range learnings {
        fmt.Printf("  - %s (confidence: %.2f)\n", l.Content, l.Confidence)
    }

    // Show stats
    stats, _ := cor.Stats()
    fmt.Printf("\nStats: %v learnings, %v linked workspaces\n",
        stats["learningCount"], stats["linkedWorkspaces"])
}
```

---

## Integration Patterns

### AI Agent Integration

```go
// In your agent's initialization
mem, _ := memory.Open(workspacePath)
session, _ := mem.StartSession("my-agent", instanceID, userGoal)

// Before reading a file
mem.LogActivity(session.ID, memory.Activity{
    Kind: memory.ActivityFileRead, Target: filePath,
})

// Get relevant context
learnings, _ := mem.GetRelevantLearnings(filePath, currentRoom, 5)

// Before editing a file
conflict, _ := mem.CheckConflict(session.ID, filePath)
if conflict != nil {
    // Warn user or handle conflict
}

mem.LogActivity(session.ID, memory.Activity{
    Kind: memory.ActivityFileEdit, Target: filePath, Outcome: memory.OutcomeSuccess,
})
mem.RecordFileEdit(filePath, "my-agent")

// When discovering something useful
mem.AddLearning(memory.Learning{
    Scope: memory.ScopeFile, ScopePath: filePath,
    Content: "Discovered pattern...", Confidence: 0.7, Source: memory.SourceAgent,
})

// On completion
mem.EndSession(session.ID, memory.SessionCompleted, summary)
```

### Cross-Project Knowledge

```go
// Promote high-confidence workspace learnings to personal corridor
cor, _ := corridor.OpenGlobal()
mem, _ := memory.Open(workspacePath)

learnings, _ := mem.GetLearnings("palace", "", 100)
for _, l := range learnings {
    if l.Confidence >= 0.8 && l.UseCount >= 3 {
        cor.AddPersonalLearning(corridor.PersonalLearning{
            Content:         l.Content,
            Confidence:      l.Confidence,
            Source:          "promoted",
            OriginWorkspace: workspaceName,
        })
    }
}
```

---

## Error Handling

All functions return errors that should be checked:

```go
mem, err := memory.Open(path)
if err != nil {
    // Handle: database creation failed, invalid path, etc.
}

session, err := mem.StartSession(agentType, agentID, goal)
if err != nil {
    // Handle: database error, validation error, etc.
}
```

Common errors:
- Invalid workspace path
- Database locked (another process has exclusive access)
- Invalid session ID
- Linked workspace doesn't exist or has no `.palace` directory
