// Package types provides shared type definitions for Mind Palace.
// External tools can import these types to interact with Mind Palace data structures.
package types

import "time"

// Session represents an agent work session in a workspace.
type Session struct {
	ID           string    `json:"id"`
	AgentType    string    `json:"agentType"` // "claude-code", "cursor", "aider", etc.
	AgentID      string    `json:"agentId"`   // Unique agent instance identifier
	Goal         string    `json:"goal"`      // What the agent is trying to accomplish
	StartedAt    time.Time `json:"startedAt"`
	LastActivity time.Time `json:"lastActivity"`
	State        string    `json:"state"`   // "active", "completed", "abandoned"
	Summary      string    `json:"summary"` // Summary of what was accomplished
}

// Activity represents an action taken during a session.
type Activity struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId"`
	Kind      string    `json:"kind"`    // "file_read", "file_edit", "search", "command"
	Target    string    `json:"target"`  // File path or search query
	Details   string    `json:"details"` // JSON with specifics
	Timestamp time.Time `json:"timestamp"`
	Outcome   string    `json:"outcome"` // "success", "failure", "unknown"
}

// Learning represents an emerged pattern or heuristic.
type Learning struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"sessionId,omitempty"` // Optional - can be manual
	Scope      string    `json:"scope"`               // "file", "room", "palace", "corridor"
	ScopePath  string    `json:"scopePath"`           // e.g., "auth/login.go" or "auth" or ""
	Content    string    `json:"content"`             // The learning itself
	Confidence float64   `json:"confidence"`          // 0.0-1.0, increases with reinforcement
	Source     string    `json:"source"`              // "agent", "user", "inferred"
	CreatedAt  time.Time `json:"createdAt"`
	LastUsed   time.Time `json:"lastUsed"`
	UseCount   int       `json:"useCount"`
}

// FileIntel represents intelligence gathered about a specific file.
type FileIntel struct {
	Path         string    `json:"path"`
	EditCount    int       `json:"editCount"`
	LastEdited   time.Time `json:"lastEdited,omitempty"`
	LastEditor   string    `json:"lastEditor"` // Agent type
	FailureCount int       `json:"failureCount"`
	Learnings    []string  `json:"learnings"` // Learning IDs associated with this file
}

// PersonalLearning represents a learning in the personal corridor (global).
type PersonalLearning struct {
	ID              string    `json:"id"`
	OriginWorkspace string    `json:"originWorkspace"`
	Content         string    `json:"content"`
	Confidence      float64   `json:"confidence"`
	Source          string    `json:"source"`
	CreatedAt       time.Time `json:"createdAt"`
	LastUsed        time.Time `json:"lastUsed"`
	UseCount        int       `json:"useCount"`
	Tags            []string  `json:"tags"`
}

// LinkedWorkspace represents a linked workspace in the corridor.
type LinkedWorkspace struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	AddedAt      time.Time `json:"addedAt"`
	LastAccessed time.Time `json:"lastAccessed,omitempty"`
}

// ActiveAgent represents an agent currently working in the workspace.
type ActiveAgent struct {
	AgentType   string    `json:"agentType"`
	AgentID     string    `json:"agentId"`
	SessionID   string    `json:"sessionId"`
	LastSeen    time.Time `json:"lastSeen"`
	CurrentFile string    `json:"currentFile,omitempty"`
}

// Conflict represents a potential conflict between agents.
type Conflict struct {
	Path         string    `json:"path"`
	OtherSession string    `json:"otherSession"`
	OtherAgent   string    `json:"otherAgent"`
	LastTouched  time.Time `json:"lastTouched"`
	Severity     string    `json:"severity"` // "warning", "critical"
}

// ActivityKind constants for activity types.
const (
	ActivityFileRead = "file_read"
	ActivityFileEdit = "file_edit"
	ActivitySearch   = "search"
	ActivityCommand  = "command"
)

// SessionState constants for session states.
const (
	SessionActive    = "active"
	SessionCompleted = "completed"
	SessionAbandoned = "abandoned"
)

// LearningScope constants for learning scopes.
const (
	ScopeFile     = "file"
	ScopeRoom     = "room"
	ScopePalace   = "palace"
	ScopeCorridor = "corridor"
)

// LearningSource constants for learning sources.
const (
	SourceAgent    = "agent"
	SourceUser     = "user"
	SourceInferred = "inferred"
	SourcePromoted = "promoted"
)

// Outcome constants for activity outcomes.
const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
	OutcomeUnknown = "unknown"
)
