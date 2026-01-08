package memory

import (
	"context"
	"fmt"
	"strings"
)

// SetTags sets the tags for a record, replacing any existing tags.
func (m *Memory) SetTags(recordID, recordKind string, tags []string) error {
	// Start a transaction
	tx, err := m.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing tags
	_, err = tx.ExecContext(context.Background(), `DELETE FROM record_tags WHERE record_id = ? AND record_kind = ?`, recordID, recordKind)
	if err != nil {
		return fmt.Errorf("delete existing tags: %w", err)
	}

	// Insert new tags
	for _, tag := range tags {
		tag = normalizeTag(tag)
		if tag == "" {
			continue
		}
		_, err = tx.ExecContext(context.Background(), `INSERT OR IGNORE INTO record_tags (record_id, record_kind, tag) VALUES (?, ?, ?)`,
			recordID, recordKind, tag)
		if err != nil {
			return fmt.Errorf("insert tag: %w", err)
		}
	}

	return tx.Commit()
}

// AddTag adds a single tag to a record.
func (m *Memory) AddTag(recordID, recordKind, tag string) error {
	tag = normalizeTag(tag)
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}

	_, err := m.db.ExecContext(context.Background(), `INSERT OR IGNORE INTO record_tags (record_id, record_kind, tag) VALUES (?, ?, ?)`,
		recordID, recordKind, tag)
	if err != nil {
		return fmt.Errorf("insert tag: %w", err)
	}
	return nil
}

// RemoveTag removes a single tag from a record.
func (m *Memory) RemoveTag(recordID, recordKind, tag string) error {
	tag = normalizeTag(tag)
	_, err := m.db.ExecContext(context.Background(), `DELETE FROM record_tags WHERE record_id = ? AND record_kind = ? AND tag = ?`,
		recordID, recordKind, tag)
	return err
}

// GetTags returns all tags for a record.
func (m *Memory) GetTags(recordID, recordKind string) ([]string, error) {
	rows, err := m.db.QueryContext(context.Background(), `SELECT tag FROM record_tags WHERE record_id = ? AND record_kind = ? ORDER BY tag`,
		recordID, recordKind)
	if err != nil {
		return nil, fmt.Errorf("query tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}
	return tags, nil
}

// GetRecordsByTag returns all record IDs with a given tag.
func (m *Memory) GetRecordsByTag(tag, recordKind string) ([]string, error) {
	tag = normalizeTag(tag)
	query := `SELECT record_id FROM record_tags WHERE tag = ?`
	args := []interface{}{tag}

	if recordKind != "" {
		query += ` AND record_kind = ?`
		args = append(args, recordKind)
	}
	query += ` ORDER BY record_id`

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query records by tag: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan record id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate record ids: %w", err)
	}
	return ids, nil
}

// GetAllTags returns all unique tags in the database.
func (m *Memory) GetAllTags(recordKind string) ([]string, error) {
	query := `SELECT DISTINCT tag FROM record_tags`
	args := []interface{}{}

	if recordKind != "" {
		query += ` WHERE record_kind = ?`
		args = append(args, recordKind)
	}
	query += ` ORDER BY tag`

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query all tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}
	return tags, nil
}

// GetTagCounts returns tag usage counts.
func (m *Memory) GetTagCounts(recordKind string, limit int) (map[string]int, error) {
	query := `SELECT tag, COUNT(*) as count FROM record_tags`
	args := []interface{}{}

	if recordKind != "" {
		query += ` WHERE record_kind = ?`
		args = append(args, recordKind)
	}
	query += ` GROUP BY tag ORDER BY count DESC, tag`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tag counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var tag string
		var count int
		if err := rows.Scan(&tag, &count); err != nil {
			return nil, fmt.Errorf("scan tag count: %w", err)
		}
		counts[tag] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tag counts: %w", err)
	}
	return counts, nil
}

// SearchByTags returns records that have ALL the specified tags.
func (m *Memory) SearchByTags(tags []string, recordKind string, limit int) ([]string, error) {
	if len(tags) == 0 {
		return nil, nil
	}

	// Normalize tags
	normalizedTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = normalizeTag(tag)
		if tag != "" {
			normalizedTags = append(normalizedTags, tag)
		}
	}
	if len(normalizedTags) == 0 {
		return nil, nil
	}

	// Build query to find records with ALL tags
	placeholders := make([]string, len(normalizedTags))
	args := make([]interface{}, len(normalizedTags))
	for i, tag := range normalizedTags {
		placeholders[i] = "?"
		args[i] = tag
	}

	query := fmt.Sprintf(`
		SELECT record_id FROM record_tags
		WHERE tag IN (%s)
	`, strings.Join(placeholders, ", "))

	if recordKind != "" {
		query += ` AND record_kind = ?`
		args = append(args, recordKind)
	}

	query += fmt.Sprintf(`
		GROUP BY record_id
		HAVING COUNT(DISTINCT tag) = %d
		ORDER BY record_id
	`, len(normalizedTags))

	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("search by tags: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan record id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate record ids: %w", err)
	}
	return ids, nil
}

// DeleteTagsForRecord removes all tags for a record.
func (m *Memory) DeleteTagsForRecord(recordID, recordKind string) error {
	_, err := m.db.ExecContext(context.Background(), `DELETE FROM record_tags WHERE record_id = ? AND record_kind = ?`, recordID, recordKind)
	return err
}

// normalizeTag lowercases and trims a tag.
func normalizeTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}
