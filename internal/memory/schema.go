package memory

import (
	"database/sql"
	"fmt"
	"time"
)

// schemaVersionTable creates the schema version tracking table
const schemaVersionTable = `
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL
);
`

// migrations is an ordered list of database migrations.
// Each migration is a function that takes a transaction and applies schema changes.
// Migrations are applied in order, starting from version 0.
// IMPORTANT: Never modify existing migrations, only add new ones.
var migrations = []func(*sql.Tx) error{
	// Migration 0: Initial schema
	migrateV0,
}

// migrateV0 creates the initial database schema (version 0)
func migrateV0(tx *sql.Tx) error {
	schema := `
-- Sessions: Track agent work sessions
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    agent_type TEXT NOT NULL,
    agent_id TEXT DEFAULT '',
    goal TEXT DEFAULT '',
    started_at TEXT NOT NULL,
    last_activity TEXT NOT NULL,
    state TEXT DEFAULT 'active',
    summary TEXT DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_sessions_state ON sessions(state);
CREATE INDEX IF NOT EXISTS idx_sessions_agent ON sessions(agent_type);

-- Activities: What agents did during sessions
CREATE TABLE IF NOT EXISTS activities (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    target TEXT DEFAULT '',
    details TEXT DEFAULT '{}',
    timestamp TEXT NOT NULL,
    outcome TEXT DEFAULT 'unknown'
);
CREATE INDEX IF NOT EXISTS idx_activities_session ON activities(session_id);
CREATE INDEX IF NOT EXISTS idx_activities_target ON activities(target);
CREATE INDEX IF NOT EXISTS idx_activities_kind ON activities(kind);

-- Learnings: Emerged patterns and heuristics
CREATE TABLE IF NOT EXISTS learnings (
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
CREATE INDEX IF NOT EXISTS idx_learnings_scope ON learnings(scope, scope_path);
CREATE INDEX IF NOT EXISTS idx_learnings_confidence ON learnings(confidence DESC);

-- File Intelligence: Per-file history
CREATE TABLE IF NOT EXISTS file_intel (
    path TEXT PRIMARY KEY,
    edit_count INTEGER DEFAULT 0,
    last_edited TEXT,
    last_editor TEXT DEFAULT '',
    failure_count INTEGER DEFAULT 0
);

-- Learning-File associations
CREATE TABLE IF NOT EXISTS file_learnings (
    file_path TEXT NOT NULL,
    learning_id TEXT NOT NULL,
    PRIMARY KEY (file_path, learning_id),
    FOREIGN KEY (file_path) REFERENCES file_intel(path) ON DELETE CASCADE,
    FOREIGN KEY (learning_id) REFERENCES learnings(id) ON DELETE CASCADE
);

-- Active agents registry (for multi-agent coordination)
CREATE TABLE IF NOT EXISTS active_agents (
    agent_id TEXT PRIMARY KEY,
    agent_type TEXT NOT NULL,
    session_id TEXT REFERENCES sessions(id) ON DELETE CASCADE,
    last_heartbeat TEXT NOT NULL,
    current_file TEXT DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_active_agents_file ON active_agents(current_file);
`
	_, err := tx.Exec(schema)
	return err
}

// ensureSchema creates the schema version table and runs any pending migrations
func (m *Memory) ensureSchema() error {
	// Create schema version table first
	if _, err := m.db.Exec(schemaVersionTable); err != nil {
		return fmt.Errorf("create schema_version table: %w", err)
	}

	// Get current schema version
	var currentVersion int
	row := m.db.QueryRow("SELECT COALESCE(MAX(version), -1) FROM schema_version")
	if err := row.Scan(&currentVersion); err != nil {
		return fmt.Errorf("get schema version: %w", err)
	}

	// Run pending migrations
	for i := currentVersion + 1; i < len(migrations); i++ {
		if err := m.runMigration(i); err != nil {
			return fmt.Errorf("run migration %d: %w", i, err)
		}
	}

	return nil
}

// runMigration executes a single migration in a transaction
func (m *Memory) runMigration(version int) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Run the migration
	if err := migrations[version](tx); err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}

	// Record the migration
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.Exec("INSERT INTO schema_version (version, applied_at) VALUES (?, ?)", version, now); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	return tx.Commit()
}

// GetSchemaVersion returns the current schema version
func (m *Memory) GetSchemaVersion() (int, error) {
	var version int
	row := m.db.QueryRow("SELECT COALESCE(MAX(version), -1) FROM schema_version")
	err := row.Scan(&version)
	return version, err
}
