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

	// Enqueue embedding generation (non-blocking)
	m.enqueueEmbedding(l.ID, "learning", l.Content)

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

// ============================================================================
// Learning Lifecycle Management
// ============================================================================

// Learning status constants
const (
	LearningStatusActive   = "active"
	LearningStatusObsolete = "obsolete"
	LearningStatusArchived = "archived"
)

// MarkLearningObsolete marks a learning as obsolete with a reason.
func (m *Memory) MarkLearningObsolete(id, reason string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := m.db.Exec(`
		UPDATE learnings
		SET status = ?, obsolete_reason = ?, last_used = ?
		WHERE id = ?
	`, LearningStatusObsolete, reason, now, id)
	if err != nil {
		return fmt.Errorf("mark learning obsolete: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("learning not found: %s", id)
	}
	return nil
}

// ArchiveOldLearnings archives learnings that are unused and low confidence.
func (m *Memory) ArchiveOldLearnings(unusedDays int, maxConfidence float64) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -unusedDays).Format(time.RFC3339)
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := m.db.Exec(`
		UPDATE learnings
		SET status = ?, archived_at = ?
		WHERE last_used < ? AND confidence <= ? AND status = ?
	`, LearningStatusArchived, now, cutoff, maxConfidence, LearningStatusActive)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetLearningsByStatus retrieves learnings by lifecycle status.
func (m *Memory) GetLearningsByStatus(status string, limit int) ([]Learning, error) {
	query := `
		SELECT id, session_id, scope, scope_path, content, confidence, source, created_at, last_used, use_count
		FROM learnings
		WHERE status = ?
		ORDER BY last_used DESC
	`
	args := []interface{}{status}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query learnings by status: %w", err)
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

// ============================================================================
// Decision-Learning Links (Outcome Feedback)
// ============================================================================

// LinkLearningToDecision creates a relationship between a learning and decision.
// When the decision's outcome is recorded, the linked learning's confidence is updated.
func (m *Memory) LinkLearningToDecision(decisionID, learningID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := m.db.Exec(`
		INSERT OR IGNORE INTO decision_learnings (decision_id, learning_id, created_at)
		VALUES (?, ?, ?)
	`, decisionID, learningID, now)
	if err != nil {
		return fmt.Errorf("link learning to decision: %w", err)
	}
	return nil
}

// UnlinkLearningFromDecision removes the relationship between a learning and decision.
func (m *Memory) UnlinkLearningFromDecision(decisionID, learningID string) error {
	_, err := m.db.Exec(`
		DELETE FROM decision_learnings WHERE decision_id = ? AND learning_id = ?
	`, decisionID, learningID)
	return err
}

// GetLearningsForDecision returns learnings linked to a decision.
func (m *Memory) GetLearningsForDecision(decisionID string) ([]Learning, error) {
	rows, err := m.db.Query(`
		SELECT l.id, l.session_id, l.scope, l.scope_path, l.content, l.confidence, l.source, l.created_at, l.last_used, l.use_count
		FROM learnings l
		JOIN decision_learnings dl ON l.id = dl.learning_id
		WHERE dl.decision_id = ?
		ORDER BY l.confidence DESC
	`, decisionID)
	if err != nil {
		return nil, fmt.Errorf("query learnings for decision: %w", err)
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

// GetDecisionsForLearning returns decisions linked to a learning.
func (m *Memory) GetDecisionsForLearning(learningID string) ([]Decision, error) {
	rows, err := m.db.Query(`
		SELECT d.id, d.content, d.rationale, d.context, d.status, d.outcome, d.outcome_note, d.outcome_at, d.scope, d.scope_path, d.session_id, d.source, d.created_at, d.updated_at
		FROM decisions d
		JOIN decision_learnings dl ON d.id = dl.decision_id
		WHERE dl.learning_id = ?
		ORDER BY d.created_at DESC
	`, learningID)
	if err != nil {
		return nil, fmt.Errorf("query decisions for learning: %w", err)
	}
	defer rows.Close()

	var decisions []Decision
	for rows.Next() {
		var d Decision
		var createdAt, updatedAt, outcomeAt string
		if err := rows.Scan(&d.ID, &d.Content, &d.Rationale, &d.Context, &d.Status, &d.Outcome, &d.OutcomeNote, &outcomeAt, &d.Scope, &d.ScopePath, &d.SessionID, &d.Source, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan decision: %w", err)
		}
		d.CreatedAt = parseTimeOrZero(createdAt)
		d.UpdatedAt = parseTimeOrZero(updatedAt)
		d.OutcomeAt = parseTimeOrZero(outcomeAt)
		decisions = append(decisions, d)
	}
	return decisions, nil
}
