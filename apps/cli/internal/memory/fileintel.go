package memory

import (
	"database/sql"
	"fmt"
	"time"
)


// GetFileIntel retrieves intelligence about a specific file.
func (m *Memory) GetFileIntel(path string) (*FileIntel, error) {
	row := m.db.QueryRow(`
		SELECT path, edit_count, last_edited, last_editor, failure_count
		FROM file_intel WHERE path = ?
	`, path)

	var fi FileIntel
	var lastEdited sql.NullString
	err := row.Scan(&fi.Path, &fi.EditCount, &lastEdited, &fi.LastEditor, &fi.FailureCount)
	if err == sql.ErrNoRows {
		// Return empty intel for files not yet tracked
		return &FileIntel{Path: path}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan file intel: %w", err)
	}

	if lastEdited.Valid {
		fi.LastEdited = parseTimeOrZero(lastEdited.String)
	}

	// Get associated learnings
	rows, err := m.db.Query(`
		SELECT learning_id FROM file_learnings WHERE file_path = ?
	`, path)
	if err != nil {
		return nil, fmt.Errorf("query file learnings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var learningID string
		if err := rows.Scan(&learningID); err != nil {
			return nil, fmt.Errorf("scan learning id: %w", err)
		}
		fi.Learnings = append(fi.Learnings, learningID)
	}

	return &fi, nil
}

// RecordFileEdit records that a file was edited.
func (m *Memory) RecordFileEdit(path, agentType string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	// Upsert file_intel
	_, err := m.db.Exec(`
		INSERT INTO file_intel (path, edit_count, last_edited, last_editor, failure_count)
		VALUES (?, 1, ?, ?, 0)
		ON CONFLICT(path) DO UPDATE SET
			edit_count = edit_count + 1,
			last_edited = excluded.last_edited,
			last_editor = excluded.last_editor
	`, path, now, agentType)
	return err
}

// RecordFileFailure records that an edit to a file led to a failure.
func (m *Memory) RecordFileFailure(path string) error {
	_, err := m.db.Exec(`
		UPDATE file_intel SET failure_count = failure_count + 1 WHERE path = ?
	`, path)
	return err
}

// AssociateLearningWithFile links a learning to a file.
func (m *Memory) AssociateLearningWithFile(filePath, learningID string) error {
	// Ensure file_intel entry exists
	if err := m.ensureFileIntel(filePath); err != nil {
		return err
	}

	_, err := m.db.Exec(`
		INSERT OR IGNORE INTO file_learnings (file_path, learning_id)
		VALUES (?, ?)
	`, filePath, learningID)
	return err
}

// ensureFileIntel creates a file_intel entry if it doesn't exist.
func (m *Memory) ensureFileIntel(path string) error {
	_, err := m.db.Exec(`
		INSERT OR IGNORE INTO file_intel (path, edit_count, failure_count)
		VALUES (?, 0, 0)
	`, path)
	return err
}

// GetFileHotspots returns files with the most edits.
func (m *Memory) GetFileHotspots(limit int) ([]FileIntel, error) {
	query := `
		SELECT path, edit_count, last_edited, last_editor, failure_count
		FROM file_intel
		ORDER BY edit_count DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query file hotspots: %w", err)
	}
	defer rows.Close()

	var files []FileIntel
	for rows.Next() {
		var fi FileIntel
		var lastEdited sql.NullString
		if err := rows.Scan(&fi.Path, &fi.EditCount, &lastEdited, &fi.LastEditor, &fi.FailureCount); err != nil {
			return nil, fmt.Errorf("scan file intel: %w", err)
		}
		if lastEdited.Valid {
			fi.LastEdited = parseTimeOrZero(lastEdited.String)
		}
		files = append(files, fi)
	}
	return files, nil
}

// GetFragileFiles returns files with the most failures.
func (m *Memory) GetFragileFiles(limit int) ([]FileIntel, error) {
	query := `
		SELECT path, edit_count, last_edited, last_editor, failure_count
		FROM file_intel
		WHERE failure_count > 0
		ORDER BY failure_count DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query fragile files: %w", err)
	}
	defer rows.Close()

	var files []FileIntel
	for rows.Next() {
		var fi FileIntel
		var lastEdited sql.NullString
		if err := rows.Scan(&fi.Path, &fi.EditCount, &lastEdited, &fi.LastEditor, &fi.FailureCount); err != nil {
			return nil, fmt.Errorf("scan file intel: %w", err)
		}
		if lastEdited.Valid {
			fi.LastEdited = parseTimeOrZero(lastEdited.String)
		}
		files = append(files, fi)
	}
	return files, nil
}

// GetFileLearnings returns all learnings associated with a file.
func (m *Memory) GetFileLearnings(path string) ([]Learning, error) {
	rows, err := m.db.Query(`
		SELECT l.id, l.session_id, l.scope, l.scope_path, l.content, l.confidence, l.source, l.created_at, l.last_used, l.use_count
		FROM learnings l
		JOIN file_learnings fl ON l.id = fl.learning_id
		WHERE fl.file_path = ?
		ORDER BY l.confidence DESC
	`, path)
	if err != nil {
		return nil, fmt.Errorf("query file learnings: %w", err)
	}
	defer rows.Close()

	var learnings []Learning
	for rows.Next() {
		var l Learning
		var createdAt, lastUsed string
		if err := rows.Scan(&l.ID, &l.SessionID, &l.Scope, &l.ScopePath, &l.Content, &l.Confidence, &l.Source, &createdAt, &lastUsed, &l.UseCount); err != nil {
			return nil, fmt.Errorf("scan learning: %w", err)
		}
		l.CreatedAt = parseTimeOrZero(createdAt)
		l.LastUsed = parseTimeOrZero(lastUsed)
		learnings = append(learnings, l)
	}
	return learnings, nil
}
