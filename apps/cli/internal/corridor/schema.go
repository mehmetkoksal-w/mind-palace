package corridor

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// schemaVersionTable creates the schema version tracking table
const corridorSchemaVersionTable = `
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL
);
`

// corridorMigrations is an ordered list of database migrations for the corridor DB.
// Each migration is a function that takes a transaction and applies schema changes.
// Migrations are applied in order, starting from version 0.
// IMPORTANT: Never modify existing migrations, only add new ones.
var corridorMigrations = []func(*sql.Tx) error{
	// Migration 0: Initial schema
	corridorMigrateV0,
}

// corridorMigrateV0 creates the initial corridor schema (version 0)
func corridorMigrateV0(tx *sql.Tx) error {
	schema := `
-- Personal corridor learnings
CREATE TABLE IF NOT EXISTS learnings (
    id TEXT PRIMARY KEY,
    origin_workspace TEXT DEFAULT '',   -- Where it came from
    content TEXT NOT NULL,
    confidence REAL DEFAULT 0.5,
    source TEXT DEFAULT 'promoted',
    created_at TEXT NOT NULL,
    last_used TEXT NOT NULL,
    use_count INTEGER DEFAULT 0,
    tags TEXT DEFAULT '[]'              -- JSON array for categorization
);
CREATE INDEX IF NOT EXISTS idx_learnings_confidence ON learnings(confidence DESC);

-- Linked workspace registry
CREATE TABLE IF NOT EXISTS links (
    name TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    added_at TEXT NOT NULL,
    last_accessed TEXT
);
`
	_, err := tx.ExecContext(context.Background(), schema)
	return err
}

func initCorridorDB(db *sql.DB) error {
	// Create schema version table first
	if _, err := db.ExecContext(context.Background(), corridorSchemaVersionTable); err != nil {
		return fmt.Errorf("create schema_version table: %w", err)
	}

	// Get current schema version
	var currentVersion int
	row := db.QueryRowContext(context.Background(), "SELECT COALESCE(MAX(version), -1) FROM schema_version")
	if err := row.Scan(&currentVersion); err != nil {
		return fmt.Errorf("get schema version: %w", err)
	}

	// Run pending migrations
	for i := currentVersion + 1; i < len(corridorMigrations); i++ {
		if err := runCorridorMigration(db, i); err != nil {
			return fmt.Errorf("run migration %d: %w", i, err)
		}
	}

	return nil
}

// runCorridorMigration executes a single migration in a transaction
func runCorridorMigration(db *sql.DB, version int) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Run the migration
	if err := corridorMigrations[version](tx); err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}

	// Record the migration
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(context.Background(), "INSERT INTO schema_version (version, applied_at) VALUES (?, ?)", version, now); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	return tx.Commit()
}

// GetCorridorSchemaVersion returns the current corridor schema version
func GetCorridorSchemaVersion(db *sql.DB) (int, error) {
	var version int
	row := db.QueryRowContext(context.Background(), "SELECT COALESCE(MAX(version), -1) FROM schema_version")
	err := row.Scan(&version)
	return version, err
}
