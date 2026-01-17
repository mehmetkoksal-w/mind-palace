// Package memory provides session tracking, learning management, and file intelligence
// for AI agents working in a Mind Palace workspace.
package memory

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Memory manages the session memory database for a workspace.
type Memory struct {
	db       *sql.DB
	root     string
	pipeline *EmbeddingPipeline // optional, may be nil
}

// Session represents an agent work session in the workspace.
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
	ID                     string    `json:"id"`
	SessionID              string    `json:"sessionId,omitempty"`              // Optional - can be manual
	Scope                  string    `json:"scope"`                            // "file", "room", "palace", "corridor"
	ScopePath              string    `json:"scopePath"`                        // e.g., "auth/login.go" or "auth" or ""
	Content                string    `json:"content"`                          // The learning itself
	Confidence             float64   `json:"confidence"`                       // 0.0-1.0, increases with reinforcement
	Source                 string    `json:"source"`                           // "agent", "user", "inferred"
	Authority              string    `json:"authority"`                        // "proposed", "approved", "legacy_approved"
	PromotedFromProposalID string    `json:"promotedFromProposalId,omitempty"` // ID of proposal that was promoted to create this
	CreatedAt              time.Time `json:"createdAt"`
	LastUsed               time.Time `json:"lastUsed"`
	UseCount               int       `json:"useCount"`
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

// Open opens or creates the memory database at the given workspace root.
func Open(root string) (*Memory, error) {
	dbDir := filepath.Join(root, ".palace")
	if err := os.MkdirAll(dbDir, 0o755); err != nil { // lgtm[go/path-injection] root is trusted CLI workspace path
		return nil, fmt.Errorf("create .palace dir: %w", err)
	}

	dbPath := filepath.Join(dbDir, "memory.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode and foreign keys
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(context.Background(), pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("set pragma: %w", err)
		}
	}

	m := &Memory{db: db, root: root}
	if err := m.ensureSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	return m, nil
}

// Close closes the memory database and stops the embedding pipeline.
func (m *Memory) Close() error {
	if m.pipeline != nil {
		m.pipeline.Stop()
	}
	return m.db.Close()
}

// SetEmbeddingPipeline sets the embedding pipeline for auto-embedding.
func (m *Memory) SetEmbeddingPipeline(p *EmbeddingPipeline) {
	m.pipeline = p
}

// GetEmbeddingPipeline returns the embedding pipeline (may be nil).
func (m *Memory) GetEmbeddingPipeline() *EmbeddingPipeline {
	return m.pipeline
}

// enqueueEmbedding adds a record to the embedding queue if pipeline is enabled.
func (m *Memory) enqueueEmbedding(recordID, kind, content string) {
	if m.pipeline != nil {
		m.pipeline.Enqueue(recordID, kind, content)
	}
}

// DB returns the underlying database connection for advanced queries.
func (m *Memory) DB() *sql.DB {
	return m.db
}

// CountSessions returns the number of sessions, optionally filtered to active only.
func (m *Memory) CountSessions(activeOnly bool) (int, error) {
	query := "SELECT COUNT(*) FROM sessions"
	if activeOnly {
		query += " WHERE state = 'active'"
	}
	var count int
	err := m.db.QueryRowContext(context.Background(), query).Scan(&count)
	return count, err
}

// CountLearnings returns the total number of learnings.
func (m *Memory) CountLearnings() (int, error) {
	var count int
	err := m.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM learnings").Scan(&count)
	return count, err
}

// CountFilesTracked returns the number of unique files with recorded activity.
func (m *Memory) CountFilesTracked() (int, error) {
	var count int
	err := m.db.QueryRowContext(context.Background(), "SELECT COUNT(DISTINCT path) FROM file_intel").Scan(&count)
	return count, err
}
