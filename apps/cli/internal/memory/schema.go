package memory

import (
	"context"
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
	// Migration 1: Brain tables (ideas, decisions, conversations, links, tags, embeddings)
	migrateV1,
	// Migration 2: Learning lifecycle (status, obsolete, archived, decision-learning links)
	migrateV2,
	// Migration 3: Postmortems table for failure memory
	migrateV3,
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
	_, err := tx.ExecContext(context.Background(), schema)
	return err
}

// migrateV1 adds Brain tables: ideas, decisions, conversations, links, tags, embeddings
func migrateV1(tx *sql.Tx) error {
	schema := `
-- Ideas table (with scope system like Learnings)
CREATE TABLE IF NOT EXISTS ideas (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    context TEXT DEFAULT '',
    status TEXT DEFAULT 'active',
    scope TEXT DEFAULT 'palace',
    scope_path TEXT DEFAULT '',
    session_id TEXT DEFAULT '',
    source TEXT DEFAULT 'cli',
    created_at TEXT NOT NULL,
    updated_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_ideas_status ON ideas(status);
CREATE INDEX IF NOT EXISTS idx_ideas_scope ON ideas(scope, scope_path);

-- Decisions table (with scope system and outcome tracking)
CREATE TABLE IF NOT EXISTS decisions (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    rationale TEXT DEFAULT '',
    context TEXT DEFAULT '',
    status TEXT DEFAULT 'active',
    outcome TEXT DEFAULT 'unknown',
    outcome_note TEXT DEFAULT '',
    outcome_at TEXT,
    scope TEXT DEFAULT 'palace',
    scope_path TEXT DEFAULT '',
    session_id TEXT DEFAULT '',
    source TEXT DEFAULT 'cli',
    created_at TEXT NOT NULL,
    updated_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_decisions_status ON decisions(status);
CREATE INDEX IF NOT EXISTS idx_decisions_outcome ON decisions(outcome);
CREATE INDEX IF NOT EXISTS idx_decisions_scope ON decisions(scope, scope_path);

-- Normalized tags table (shared by ideas, decisions, learnings)
CREATE TABLE IF NOT EXISTS record_tags (
    record_id TEXT NOT NULL,
    record_kind TEXT NOT NULL,
    tag TEXT NOT NULL,
    PRIMARY KEY (record_id, tag)
);
CREATE INDEX IF NOT EXISTS idx_record_tags_tag ON record_tags(tag);
CREATE INDEX IF NOT EXISTS idx_record_tags_kind ON record_tags(record_kind);

-- Conversations table (auto-captured on session end)
CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    agent_type TEXT DEFAULT '',
    summary TEXT DEFAULT '',
    messages TEXT DEFAULT '[]',
    extracted TEXT DEFAULT '[]',
    session_id TEXT DEFAULT '',
    created_at TEXT NOT NULL
);

-- Links table with staleness tracking for code links
CREATE TABLE IF NOT EXISTS links (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    source_kind TEXT NOT NULL,
    target_id TEXT NOT NULL,
    target_kind TEXT NOT NULL,
    relation TEXT NOT NULL,
    target_mtime TEXT DEFAULT '',
    is_stale INTEGER DEFAULT 0,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_links_source ON links(source_id);
CREATE INDEX IF NOT EXISTS idx_links_target ON links(target_id);
CREATE INDEX IF NOT EXISTS idx_links_relation ON links(relation);

-- Embeddings table (for sqlite-vss semantic search)
CREATE TABLE IF NOT EXISTS embeddings (
    record_id TEXT PRIMARY KEY,
    record_kind TEXT NOT NULL,
    embedding BLOB NOT NULL,
    model TEXT DEFAULT '',
    created_at TEXT NOT NULL
);

-- FTS5 for ideas with triggers for automatic sync
CREATE VIRTUAL TABLE IF NOT EXISTS ideas_fts USING fts5(
    content, context,
    content=ideas, content_rowid=rowid
);

CREATE TRIGGER IF NOT EXISTS ideas_ai AFTER INSERT ON ideas BEGIN
    INSERT INTO ideas_fts(rowid, content, context) VALUES (new.rowid, new.content, new.context);
END;
CREATE TRIGGER IF NOT EXISTS ideas_ad AFTER DELETE ON ideas BEGIN
    INSERT INTO ideas_fts(ideas_fts, rowid, content, context) VALUES('delete', old.rowid, old.content, old.context);
END;
CREATE TRIGGER IF NOT EXISTS ideas_au AFTER UPDATE ON ideas BEGIN
    INSERT INTO ideas_fts(ideas_fts, rowid, content, context) VALUES('delete', old.rowid, old.content, old.context);
    INSERT INTO ideas_fts(rowid, content, context) VALUES (new.rowid, new.content, new.context);
END;

-- FTS5 for decisions with triggers
CREATE VIRTUAL TABLE IF NOT EXISTS decisions_fts USING fts5(
    content, rationale, context,
    content=decisions, content_rowid=rowid
);

CREATE TRIGGER IF NOT EXISTS decisions_ai AFTER INSERT ON decisions BEGIN
    INSERT INTO decisions_fts(rowid, content, rationale, context) VALUES (new.rowid, new.content, new.rationale, new.context);
END;
CREATE TRIGGER IF NOT EXISTS decisions_ad AFTER DELETE ON decisions BEGIN
    INSERT INTO decisions_fts(decisions_fts, rowid, content, rationale, context) VALUES('delete', old.rowid, old.content, old.rationale, old.context);
END;
CREATE TRIGGER IF NOT EXISTS decisions_au AFTER UPDATE ON decisions BEGIN
    INSERT INTO decisions_fts(decisions_fts, rowid, content, rationale, context) VALUES('delete', old.rowid, old.content, old.rationale, old.context);
    INSERT INTO decisions_fts(rowid, content, rationale, context) VALUES (new.rowid, new.content, new.rationale, new.context);
END;

-- FTS5 for conversations with triggers
CREATE VIRTUAL TABLE IF NOT EXISTS conversations_fts USING fts5(
    summary,
    content=conversations, content_rowid=rowid
);

CREATE TRIGGER IF NOT EXISTS conversations_ai AFTER INSERT ON conversations BEGIN
    INSERT INTO conversations_fts(rowid, summary) VALUES (new.rowid, new.summary);
END;
CREATE TRIGGER IF NOT EXISTS conversations_ad AFTER DELETE ON conversations BEGIN
    INSERT INTO conversations_fts(conversations_fts, rowid, summary) VALUES('delete', old.rowid, old.summary);
END;
CREATE TRIGGER IF NOT EXISTS conversations_au AFTER UPDATE ON conversations BEGIN
    INSERT INTO conversations_fts(conversations_fts, rowid, summary) VALUES('delete', old.rowid, old.summary);
    INSERT INTO conversations_fts(rowid, summary) VALUES (new.rowid, new.summary);
END;
`
	_, err := tx.ExecContext(context.Background(), schema)
	return err
}

// ensureSchema creates the schema version table and runs any pending migrations
func (m *Memory) ensureSchema() error {
	// Create schema version table first
	if _, err := m.db.ExecContext(context.Background(), schemaVersionTable); err != nil {
		return fmt.Errorf("create schema_version table: %w", err)
	}

	// Get current schema version
	var currentVersion int
	row := m.db.QueryRowContext(context.Background(), "SELECT COALESCE(MAX(version), -1) FROM schema_version")
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
	tx, err := m.db.BeginTx(context.Background(), nil)
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
	if _, err := tx.ExecContext(context.Background(), "INSERT INTO schema_version (version, applied_at) VALUES (?, ?)", version, now); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	return tx.Commit()
}

// GetSchemaVersion returns the current schema version
func (m *Memory) GetSchemaVersion() (int, error) {
	var version int
	row := m.db.QueryRowContext(context.Background(), "SELECT COALESCE(MAX(version), -1) FROM schema_version")
	err := row.Scan(&version)
	return version, err
}

// migrateV2 adds learning lifecycle features: status, obsolescence, archival, decision-learning links
func migrateV2(tx *sql.Tx) error {
	// Add lifecycle columns to learnings table
	alterStatements := []string{
		// Status column for lifecycle tracking
		`ALTER TABLE learnings ADD COLUMN status TEXT DEFAULT 'active'`,
		// Reason for obsolescence
		`ALTER TABLE learnings ADD COLUMN obsolete_reason TEXT DEFAULT ''`,
		// When the learning was archived
		`ALTER TABLE learnings ADD COLUMN archived_at TEXT DEFAULT ''`,
	}

	for _, stmt := range alterStatements {
		// SQLite doesn't support IF NOT EXISTS for ALTER TABLE, so we ignore errors
		// if the column already exists
		_, _ = tx.ExecContext(context.Background(), stmt)
	}

	// Create decision-learning relationship table
	schema := `
-- Decision-Learning links for outcome feedback
-- When a decision's outcome is recorded, linked learnings' confidence is updated
CREATE TABLE IF NOT EXISTS decision_learnings (
    decision_id TEXT NOT NULL,
    learning_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (decision_id, learning_id),
    FOREIGN KEY (decision_id) REFERENCES decisions(id) ON DELETE CASCADE,
    FOREIGN KEY (learning_id) REFERENCES learnings(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_decision_learnings_decision ON decision_learnings(decision_id);
CREATE INDEX IF NOT EXISTS idx_decision_learnings_learning ON decision_learnings(learning_id);

-- Index for learning status queries
CREATE INDEX IF NOT EXISTS idx_learnings_status ON learnings(status);
`
	_, err := tx.ExecContext(context.Background(), schema)
	return err
}

// migrateV3 adds the postmortems table for failure memory
func migrateV3(tx *sql.Tx) error {
	schema := `
-- Postmortems table for tracking and analyzing failures
CREATE TABLE IF NOT EXISTS postmortems (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    what_happened TEXT NOT NULL,
    root_cause TEXT DEFAULT '',
    lessons_learned TEXT DEFAULT '[]',
    prevention_steps TEXT DEFAULT '[]',
    severity TEXT DEFAULT 'medium',
    status TEXT DEFAULT 'open',
    affected_files TEXT DEFAULT '[]',
    related_decision TEXT DEFAULT '',
    related_session TEXT DEFAULT '',
    created_at TEXT NOT NULL,
    resolved_at TEXT DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_postmortems_status ON postmortems(status);
CREATE INDEX IF NOT EXISTS idx_postmortems_severity ON postmortems(severity);
CREATE INDEX IF NOT EXISTS idx_postmortems_created ON postmortems(created_at DESC);

-- FTS5 for postmortems search
CREATE VIRTUAL TABLE IF NOT EXISTS postmortems_fts USING fts5(
    title, what_happened, root_cause, lessons_learned,
    content=postmortems, content_rowid=rowid
);

CREATE TRIGGER IF NOT EXISTS postmortems_ai AFTER INSERT ON postmortems BEGIN
    INSERT INTO postmortems_fts(rowid, title, what_happened, root_cause, lessons_learned)
    VALUES (new.rowid, new.title, new.what_happened, new.root_cause, new.lessons_learned);
END;

CREATE TRIGGER IF NOT EXISTS postmortems_ad AFTER DELETE ON postmortems BEGIN
    INSERT INTO postmortems_fts(postmortems_fts, rowid, title, what_happened, root_cause, lessons_learned)
    VALUES('delete', old.rowid, old.title, old.what_happened, old.root_cause, old.lessons_learned);
END;

CREATE TRIGGER IF NOT EXISTS postmortems_au AFTER UPDATE ON postmortems BEGIN
    INSERT INTO postmortems_fts(postmortems_fts, rowid, title, what_happened, root_cause, lessons_learned)
    VALUES('delete', old.rowid, old.title, old.what_happened, old.root_cause, old.lessons_learned);
    INSERT INTO postmortems_fts(rowid, title, what_happened, root_cause, lessons_learned)
    VALUES (new.rowid, new.title, new.what_happened, new.root_cause, new.lessons_learned);
END;
`
	_, err := tx.ExecContext(context.Background(), schema)
	return err
}
