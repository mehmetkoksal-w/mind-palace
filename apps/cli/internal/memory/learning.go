package memory

import (
	"context"
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
	if l.Authority == "" {
		l.Authority = string(AuthorityProposed)
	}
	now := time.Now().UTC()
	if l.CreatedAt.IsZero() {
		l.CreatedAt = now
	}
	if l.LastUsed.IsZero() {
		l.LastUsed = now
	}

	_, err := m.db.ExecContext(context.Background(), `
		INSERT INTO learnings (id, session_id, scope, scope_path, content, confidence, source, authority, promoted_from_proposal_id, created_at, last_used, use_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, l.ID, l.SessionID, l.Scope, l.ScopePath, l.Content, l.Confidence, l.Source, l.Authority, l.PromotedFromProposalID,
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
	row := m.db.QueryRowContext(context.Background(), `
		SELECT id, session_id, scope, scope_path, content, confidence, source, authority, promoted_from_proposal_id, created_at, last_used, use_count
		FROM learnings WHERE id = ?
	`, id)

	var l Learning
	var createdAt, lastUsed string
	err := row.Scan(&l.ID, &l.SessionID, &l.Scope, &l.ScopePath, &l.Content, &l.Confidence, &l.Source, &l.Authority, &l.PromotedFromProposalID, &createdAt, &lastUsed, &l.UseCount)
	if err != nil {
		return nil, fmt.Errorf("scan learning: %w", err)
	}

	l.CreatedAt = parseTimeOrZero(createdAt)
	l.LastUsed = parseTimeOrZero(lastUsed)
	return &l, nil
}

// GetLearnings retrieves learnings matching the given criteria.
// By default, only returns authoritative records (approved or legacy_approved).
func (m *Memory) GetLearnings(scope, scopePath string, limit int) ([]Learning, error) {
	return m.GetLearningsWithAuthority(scope, scopePath, limit, true)
}

// GetLearningsWithAuthority retrieves learnings with explicit authority filtering.
func (m *Memory) GetLearningsWithAuthority(scope, scopePath string, limit int, authoritativeOnly bool) ([]Learning, error) {
	query := `SELECT id, session_id, scope, scope_path, content, confidence, source, authority, promoted_from_proposal_id, created_at, last_used, use_count FROM learnings WHERE 1=1`
	args := []interface{}{}

	if authoritativeOnly {
		authVals := AuthoritativeValuesStrings()
		query += ` AND authority IN (` + SQLPlaceholders(len(authVals)) + `)`
		for _, v := range authVals {
			args = append(args, v)
		}
	}
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

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query learnings: %w", err)
	}
	defer rows.Close()

	var learnings []Learning
	for rows.Next() {
		var l Learning
		var createdAt, lastUsed string
		if err := rows.Scan(&l.ID, &l.SessionID, &l.Scope, &l.ScopePath, &l.Content, &l.Confidence, &l.Source, &l.Authority, &l.PromotedFromProposalID, &createdAt, &lastUsed, &l.UseCount); err != nil {
			return nil, fmt.Errorf("scan learning: %w", err)
		}
		l.CreatedAt = parseTimeOrZero(createdAt)
		l.LastUsed = parseTimeOrZero(lastUsed)
		learnings = append(learnings, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate learnings: %w", err)
	}
	return learnings, nil
}

// SearchLearnings searches learnings by content.
// By default, only returns authoritative records.
func (m *Memory) SearchLearnings(query string, limit int) ([]Learning, error) {
	return m.SearchLearningsWithAuthority(query, limit, true)
}

// SearchLearningsWithAuthority searches learnings with explicit authority filtering.
func (m *Memory) SearchLearningsWithAuthority(query string, limit int, authoritativeOnly bool) ([]Learning, error) {
	// Build authority filter
	authFilter := ""
	authArgs := []interface{}{}
	if authoritativeOnly {
		authVals := AuthoritativeValuesStrings()
		authFilter = ` AND authority IN (` + SQLPlaceholders(len(authVals)) + `)`
		for _, v := range authVals {
			authArgs = append(authArgs, v)
		}
	}

	sqlQuery := `
		SELECT id, session_id, scope, scope_path, content, confidence, source, authority, promoted_from_proposal_id, created_at, last_used, use_count
		FROM learnings
		WHERE content LIKE ?` + authFilter + `
		ORDER BY confidence DESC, use_count DESC
	`
	args := append([]interface{}{"%" + query + "%"}, authArgs...)
	if limit > 0 {
		sqlQuery += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.QueryContext(context.Background(), sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search learnings: %w", err)
	}
	defer rows.Close()

	var learnings []Learning
	for rows.Next() {
		var l Learning
		var createdAt, lastUsed string
		if err := rows.Scan(&l.ID, &l.SessionID, &l.Scope, &l.ScopePath, &l.Content, &l.Confidence, &l.Source, &l.Authority, &l.PromotedFromProposalID, &createdAt, &lastUsed, &l.UseCount); err != nil {
			return nil, fmt.Errorf("scan learning: %w", err)
		}
		l.CreatedAt = parseTimeOrZero(createdAt)
		l.LastUsed = parseTimeOrZero(lastUsed)
		learnings = append(learnings, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate learnings: %w", err)
	}
	return learnings, nil
}

// ReinforceLearning increases confidence and use count of a learning.
func (m *Memory) ReinforceLearning(id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := m.db.ExecContext(context.Background(), `
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
	_, err := m.db.ExecContext(context.Background(), `
		UPDATE learnings
		SET confidence = MAX(0.0, confidence - 0.1),
		    last_used = ?
		WHERE id = ?
	`, now, id)
	return err
}

// DeleteLearning removes a learning from the database.
func (m *Memory) DeleteLearning(id string) error {
	_, err := m.db.ExecContext(context.Background(), `DELETE FROM learnings WHERE id = ?`, id)
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
		for i := range allLearnings {
			l := &allLearnings[i]
			seen[l.ID] = true
		}
		for i := range searchResults {
			l := &searchResults[i]
			if !seen[l.ID] {
				allLearnings = append(allLearnings, *l)
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
// Only returns authoritative records.
func (m *Memory) GetHighConfidenceLearnings(minConfidence float64, minUseCount int) ([]Learning, error) {
	// Build authority filter
	authVals := AuthoritativeValuesStrings()
	authPlaceholders := SQLPlaceholders(len(authVals))

	query := `
		SELECT id, session_id, scope, scope_path, content, confidence, source, authority, promoted_from_proposal_id, created_at, last_used, use_count
		FROM learnings
		WHERE confidence >= ? AND use_count >= ? AND authority IN (` + authPlaceholders + `)
		ORDER BY confidence DESC, use_count DESC
	`
	args := []interface{}{minConfidence, minUseCount}
	for _, v := range authVals {
		args = append(args, v)
	}

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query high confidence learnings: %w", err)
	}
	defer rows.Close()

	var learnings []Learning
	for rows.Next() {
		var l Learning
		var createdAt, lastUsed string
		if err := rows.Scan(&l.ID, &l.SessionID, &l.Scope, &l.ScopePath, &l.Content, &l.Confidence, &l.Source, &l.Authority, &l.PromotedFromProposalID, &createdAt, &lastUsed, &l.UseCount); err != nil {
			return nil, fmt.Errorf("scan learning: %w", err)
		}
		l.CreatedAt = parseTimeOrZero(createdAt)
		l.LastUsed = parseTimeOrZero(lastUsed)
		learnings = append(learnings, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate learnings: %w", err)
	}
	return learnings, nil
}

// DecayUnusedLearnings reduces confidence of learnings not used recently.
func (m *Memory) DecayUnusedLearnings(unusedDays int, decayAmount float64) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -unusedDays).Format(time.RFC3339)
	result, err := m.db.ExecContext(context.Background(), `
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
	result, err := m.db.ExecContext(context.Background(), `DELETE FROM learnings WHERE confidence < ?`, minConfidence)
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
	result, err := m.db.ExecContext(context.Background(), `
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
	result, err := m.db.ExecContext(context.Background(), `
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
// Only returns authoritative records.
func (m *Memory) GetLearningsByStatus(status string, limit int) ([]Learning, error) {
	// Build authority filter
	authVals := AuthoritativeValuesStrings()
	authPlaceholders := SQLPlaceholders(len(authVals))

	query := `
		SELECT id, session_id, scope, scope_path, content, confidence, source, authority, promoted_from_proposal_id, created_at, last_used, use_count
		FROM learnings
		WHERE status = ? AND authority IN (` + authPlaceholders + `)
		ORDER BY last_used DESC
	`
	args := []interface{}{status}
	for _, v := range authVals {
		args = append(args, v)
	}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query learnings by status: %w", err)
	}
	defer rows.Close()

	var learnings []Learning
	for rows.Next() {
		var l Learning
		var createdAt, lastUsed string
		if err := rows.Scan(&l.ID, &l.SessionID, &l.Scope, &l.ScopePath, &l.Content, &l.Confidence, &l.Source, &l.Authority, &l.PromotedFromProposalID, &createdAt, &lastUsed, &l.UseCount); err != nil {
			return nil, fmt.Errorf("scan learning: %w", err)
		}
		l.CreatedAt = parseTimeOrZero(createdAt)
		l.LastUsed = parseTimeOrZero(lastUsed)
		learnings = append(learnings, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate learnings: %w", err)
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
	_, err := m.db.ExecContext(context.Background(), `
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
	_, err := m.db.ExecContext(context.Background(), `
		DELETE FROM decision_learnings WHERE decision_id = ? AND learning_id = ?
	`, decisionID, learningID)
	return err
}

// GetLearningsForDecision returns learnings linked to a decision.
// Only returns authoritative records.
func (m *Memory) GetLearningsForDecision(decisionID string) ([]Learning, error) {
	// Build authority filter
	authVals := AuthoritativeValuesStrings()
	authPlaceholders := SQLPlaceholders(len(authVals))

	query := `
		SELECT l.id, l.session_id, l.scope, l.scope_path, l.content, l.confidence, l.source, l.authority, l.promoted_from_proposal_id, l.created_at, l.last_used, l.use_count
		FROM learnings l
		JOIN decision_learnings dl ON l.id = dl.learning_id
		WHERE dl.decision_id = ? AND l.authority IN (` + authPlaceholders + `)
		ORDER BY l.confidence DESC
	`
	args := []interface{}{decisionID}
	for _, v := range authVals {
		args = append(args, v)
	}

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query learnings for decision: %w", err)
	}
	defer rows.Close()

	var learnings []Learning
	for rows.Next() {
		var l Learning
		var createdAt, lastUsed string
		if err := rows.Scan(&l.ID, &l.SessionID, &l.Scope, &l.ScopePath, &l.Content, &l.Confidence, &l.Source, &l.Authority, &l.PromotedFromProposalID, &createdAt, &lastUsed, &l.UseCount); err != nil {
			return nil, fmt.Errorf("scan learning: %w", err)
		}
		l.CreatedAt = parseTimeOrZero(createdAt)
		l.LastUsed = parseTimeOrZero(lastUsed)
		learnings = append(learnings, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate learnings: %w", err)
	}
	return learnings, nil
}

// GetDecisionsForLearning returns decisions linked to a learning.
// Only returns authoritative records.
func (m *Memory) GetDecisionsForLearning(learningID string) ([]Decision, error) {
	// Build authority filter
	authVals := AuthoritativeValuesStrings()
	authPlaceholders := SQLPlaceholders(len(authVals))

	query := `
		SELECT d.id, d.content, d.rationale, d.context, d.status, d.outcome, d.outcome_note, d.outcome_at, d.scope, d.scope_path, d.session_id, d.source, d.authority, d.promoted_from_proposal_id, d.created_at, d.updated_at
		FROM decisions d
		JOIN decision_learnings dl ON d.id = dl.decision_id
		WHERE dl.learning_id = ? AND d.authority IN (` + authPlaceholders + `)
		ORDER BY d.created_at DESC
	`
	args := []interface{}{learningID}
	for _, v := range authVals {
		args = append(args, v)
	}

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query decisions for learning: %w", err)
	}
	defer rows.Close()

	var decisions []Decision
	for rows.Next() {
		var d Decision
		var createdAt, updatedAt, outcomeAt string
		if err := rows.Scan(&d.ID, &d.Content, &d.Rationale, &d.Context, &d.Status, &d.Outcome, &d.OutcomeNote, &outcomeAt, &d.Scope, &d.ScopePath, &d.SessionID, &d.Source, &d.Authority, &d.PromotedFromProposalID, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan decision: %w", err)
		}
		d.CreatedAt = parseTimeOrZero(createdAt)
		d.UpdatedAt = parseTimeOrZero(updatedAt)
		d.OutcomeAt = parseTimeOrZero(outcomeAt)
		decisions = append(decisions, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate decisions: %w", err)
	}
	return decisions, nil
}
