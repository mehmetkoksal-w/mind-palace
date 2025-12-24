package memory

import (
	"fmt"
	"strings"
	"time"
)

// parseTimeOrZero parses a time string, returning zero time on failure.
// This is intentional: database timestamps are trusted; zero time is safe fallback.
func parseTimeOrZero(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// AddLearning stores a new learning in the database.
func (m *Memory) AddLearning(l Learning) (string, error) {
	if l.ID == "" {
		l.ID = generateID("lrn")
	}
	if l.Scope == "" {
		l.Scope = "palace"
	}
	if l.Confidence == 0 {
		l.Confidence = 0.5
	}
	if l.Source == "" {
		l.Source = "agent"
	}
	now := time.Now().UTC()
	if l.CreatedAt.IsZero() {
		l.CreatedAt = now
	}
	if l.LastUsed.IsZero() {
		l.LastUsed = now
	}

	_, err := m.db.Exec(`
		INSERT INTO learnings (id, session_id, scope, scope_path, content, confidence, source, created_at, last_used, use_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, l.ID, l.SessionID, l.Scope, l.ScopePath, l.Content, l.Confidence, l.Source,
		l.CreatedAt.Format(time.RFC3339), l.LastUsed.Format(time.RFC3339), l.UseCount)
	if err != nil {
		return "", fmt.Errorf("insert learning: %w", err)
	}

	return l.ID, nil
}

// GetLearning retrieves a learning by ID.
func (m *Memory) GetLearning(id string) (*Learning, error) {
	row := m.db.QueryRow(`
		SELECT id, session_id, scope, scope_path, content, confidence, source, created_at, last_used, use_count
		FROM learnings WHERE id = ?
	`, id)

	var l Learning
	var createdAt, lastUsed string
	err := row.Scan(&l.ID, &l.SessionID, &l.Scope, &l.ScopePath, &l.Content, &l.Confidence, &l.Source, &createdAt, &lastUsed, &l.UseCount)
	if err != nil {
		return nil, fmt.Errorf("scan learning: %w", err)
	}

	l.CreatedAt = parseTimeOrZero(createdAt)
	l.LastUsed = parseTimeOrZero(lastUsed)
	return &l, nil
}

// GetLearnings retrieves learnings matching the given criteria.
func (m *Memory) GetLearnings(scope, scopePath string, limit int) ([]Learning, error) {
	query := `SELECT id, session_id, scope, scope_path, content, confidence, source, created_at, last_used, use_count FROM learnings WHERE 1=1`
	args := []interface{}{}

	if scope != "" {
		query += ` AND scope = ?`
		args = append(args, scope)
	}
	if scopePath != "" {
		query += ` AND scope_path = ?`
		args = append(args, scopePath)
	}
	query += ` ORDER BY confidence DESC, use_count DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query learnings: %w", err)
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

// SearchLearnings searches learnings by content.
func (m *Memory) SearchLearnings(query string, limit int) ([]Learning, error) {
	sqlQuery := `
		SELECT id, session_id, scope, scope_path, content, confidence, source, created_at, last_used, use_count
		FROM learnings
		WHERE content LIKE ?
		ORDER BY confidence DESC, use_count DESC
	`
	args := []interface{}{"%" + query + "%"}
	if limit > 0 {
		sqlQuery += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search learnings: %w", err)
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

// ReinforceLearning increases confidence and use count of a learning.
func (m *Memory) ReinforceLearning(id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := m.db.Exec(`
		UPDATE learnings
		SET confidence = MIN(1.0, confidence + 0.1),
		    use_count = use_count + 1,
		    last_used = ?
		WHERE id = ?
	`, now, id)
	return err
}

// WeakenLearning decreases confidence of a learning (when it proves unhelpful).
func (m *Memory) WeakenLearning(id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := m.db.Exec(`
		UPDATE learnings
		SET confidence = MAX(0.0, confidence - 0.1),
		    last_used = ?
		WHERE id = ?
	`, now, id)
	return err
}

// DeleteLearning removes a learning from the database.
func (m *Memory) DeleteLearning(id string) error {
	_, err := m.db.Exec(`DELETE FROM learnings WHERE id = ?`, id)
	return err
}

// GetRelevantLearnings finds learnings relevant to a given file or query.
func (m *Memory) GetRelevantLearnings(filePath, query string, limit int) ([]Learning, error) {
	var allLearnings []Learning

	// Get palace-wide learnings
	palaceLearnings, err := m.GetLearnings("palace", "", limit)
	if err != nil {
		return nil, err
	}
	allLearnings = append(allLearnings, palaceLearnings...)

	// Get file-specific learnings if file path provided
	if filePath != "" {
		fileLearnings, err := m.GetLearnings("file", filePath, limit)
		if err != nil {
			return nil, err
		}
		allLearnings = append(allLearnings, fileLearnings...)

		// Get room learnings if file is in a room
		parts := strings.Split(filePath, "/")
		if len(parts) > 1 {
			roomPath := parts[0]
			roomLearnings, err := m.GetLearnings("room", roomPath, limit)
			if err != nil {
				return nil, err
			}
			allLearnings = append(allLearnings, roomLearnings...)
		}
	}

	// Search by query if provided
	if query != "" {
		searchResults, err := m.SearchLearnings(query, limit)
		if err != nil {
			return nil, err
		}
		// Merge results, avoiding duplicates
		seen := make(map[string]bool)
		for _, l := range allLearnings {
			seen[l.ID] = true
		}
		for _, l := range searchResults {
			if !seen[l.ID] {
				allLearnings = append(allLearnings, l)
				seen[l.ID] = true
			}
		}
	}

	// Sort by confidence and limit
	if limit > 0 && len(allLearnings) > limit {
		allLearnings = allLearnings[:limit]
	}

	return allLearnings, nil
}

// GetHighConfidenceLearnings returns learnings ready for promotion to corridor.
func (m *Memory) GetHighConfidenceLearnings(minConfidence float64, minUseCount int) ([]Learning, error) {
	rows, err := m.db.Query(`
		SELECT id, session_id, scope, scope_path, content, confidence, source, created_at, last_used, use_count
		FROM learnings
		WHERE confidence >= ? AND use_count >= ?
		ORDER BY confidence DESC, use_count DESC
	`, minConfidence, minUseCount)
	if err != nil {
		return nil, fmt.Errorf("query high confidence learnings: %w", err)
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

// DecayUnusedLearnings reduces confidence of learnings not used recently.
func (m *Memory) DecayUnusedLearnings(unusedDays int, decayAmount float64) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -unusedDays).Format(time.RFC3339)
	result, err := m.db.Exec(`
		UPDATE learnings
		SET confidence = MAX(0.1, confidence - ?)
		WHERE last_used < ? AND confidence > 0.1
	`, decayAmount, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// PruneLowConfidenceLearnings removes learnings below threshold.
func (m *Memory) PruneLowConfidenceLearnings(minConfidence float64) (int64, error) {
	result, err := m.db.Exec(`DELETE FROM learnings WHERE confidence < ?`, minConfidence)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
