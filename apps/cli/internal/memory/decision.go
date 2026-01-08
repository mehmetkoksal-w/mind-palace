package memory

import (
	"context"
	"fmt"
	"time"
)

// Decision represents a commitment with rationale and outcome tracking.
type Decision struct {
	ID          string    `json:"id"`                    // Prefix: "d_"
	Content     string    `json:"content"`               // What was decided
	Rationale   string    `json:"rationale,omitempty"`   // Why this decision was made
	Context     string    `json:"context,omitempty"`     // Surrounding context
	Status      string    `json:"status"`                // "active", "superseded", "reversed"
	Outcome     string    `json:"outcome"`               // "unknown", "successful", "failed", "mixed"
	OutcomeNote string    `json:"outcomeNote,omitempty"` // Details about the outcome
	OutcomeAt   time.Time `json:"outcomeAt,omitempty"`   // When outcome was recorded
	Scope       string    `json:"scope"`                 // "file", "room", "palace"
	ScopePath   string    `json:"scopePath,omitempty"`   // e.g., "auth/login.go" or "auth"
	SessionID   string    `json:"sessionId,omitempty"`   // Optional session link
	Source      string    `json:"source"`                // "cli", "api", "auto-extract"
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
}

// DecisionStatus constants
const (
	DecisionStatusActive     = "active"
	DecisionStatusSuperseded = "superseded"
	DecisionStatusReversed   = "reversed"
)

// DecisionOutcome constants
const (
	DecisionOutcomeUnknown    = "unknown"
	DecisionOutcomeSuccessful = "successful"
	DecisionOutcomeFailed     = "failed"
	DecisionOutcomeMixed      = "mixed"
)

// AddDecision stores a new decision in the database.
func (m *Memory) AddDecision(dec Decision) (string, error) {
	if dec.ID == "" {
		dec.ID = generateID("d")
	}
	if dec.Status == "" {
		dec.Status = DecisionStatusActive
	}
	if dec.Outcome == "" {
		dec.Outcome = DecisionOutcomeUnknown
	}
	if dec.Scope == "" {
		dec.Scope = "palace"
	}
	if dec.Source == "" {
		dec.Source = "cli"
	}
	now := time.Now().UTC()
	if dec.CreatedAt.IsZero() {
		dec.CreatedAt = now
	}
	if dec.UpdatedAt.IsZero() {
		dec.UpdatedAt = now
	}

	outcomeAt := ""
	if !dec.OutcomeAt.IsZero() {
		outcomeAt = dec.OutcomeAt.Format(time.RFC3339)
	}

	_, err := m.db.ExecContext(context.Background(), `
		INSERT INTO decisions (id, content, rationale, context, status, outcome, outcome_note, outcome_at, scope, scope_path, session_id, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, dec.ID, dec.Content, dec.Rationale, dec.Context, dec.Status, dec.Outcome, dec.OutcomeNote, outcomeAt,
		dec.Scope, dec.ScopePath, dec.SessionID, dec.Source,
		dec.CreatedAt.Format(time.RFC3339), dec.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return "", fmt.Errorf("insert decision: %w", err)
	}

	// Enqueue embedding generation (non-blocking)
	m.enqueueEmbedding(dec.ID, "decision", dec.Content)

	return dec.ID, nil
}

// GetDecision retrieves a decision by ID.
func (m *Memory) GetDecision(id string) (*Decision, error) {
	row := m.db.QueryRowContext(context.Background(), `
		SELECT id, content, rationale, context, status, outcome, outcome_note, outcome_at, scope, scope_path, session_id, source, created_at, updated_at
		FROM decisions WHERE id = ?
	`, id)

	var dec Decision
	var createdAt, updatedAt, outcomeAt string
	err := row.Scan(&dec.ID, &dec.Content, &dec.Rationale, &dec.Context, &dec.Status, &dec.Outcome,
		&dec.OutcomeNote, &outcomeAt, &dec.Scope, &dec.ScopePath, &dec.SessionID, &dec.Source, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan decision: %w", err)
	}

	dec.CreatedAt = parseTimeOrZero(createdAt)
	dec.UpdatedAt = parseTimeOrZero(updatedAt)
	dec.OutcomeAt = parseTimeOrZero(outcomeAt)
	return &dec, nil
}

// GetDecisions retrieves decisions matching the given criteria.
func (m *Memory) GetDecisions(status, outcome, scope, scopePath string, limit int) ([]Decision, error) {
	query := `SELECT id, content, rationale, context, status, outcome, outcome_note, outcome_at, scope, scope_path, session_id, source, created_at, updated_at FROM decisions WHERE 1=1`
	args := []interface{}{}

	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	if outcome != "" {
		query += ` AND outcome = ?`
		args = append(args, outcome)
	}
	if scope != "" {
		query += ` AND scope = ?`
		args = append(args, scope)
	}
	if scopePath != "" {
		query += ` AND scope_path = ?`
		args = append(args, scopePath)
	}
	query += ` ORDER BY created_at DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query decisions: %w", err)
	}
	defer rows.Close()

	var decisions []Decision
	for rows.Next() {
		var dec Decision
		var createdAt, updatedAt, outcomeAt string
		if err := rows.Scan(&dec.ID, &dec.Content, &dec.Rationale, &dec.Context, &dec.Status, &dec.Outcome,
			&dec.OutcomeNote, &outcomeAt, &dec.Scope, &dec.ScopePath, &dec.SessionID, &dec.Source, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan decision: %w", err)
		}
		dec.CreatedAt = parseTimeOrZero(createdAt)
		dec.UpdatedAt = parseTimeOrZero(updatedAt)
		dec.OutcomeAt = parseTimeOrZero(outcomeAt)
		decisions = append(decisions, dec)
	}
	return decisions, nil
}

// SearchDecisions searches decisions by content using FTS5.
func (m *Memory) SearchDecisions(query string, limit int) ([]Decision, error) {
	sqlQuery := `
		SELECT d.id, d.content, d.rationale, d.context, d.status, d.outcome, d.outcome_note, d.outcome_at, d.scope, d.scope_path, d.session_id, d.source, d.created_at, d.updated_at
		FROM decisions d
		JOIN decisions_fts fts ON d.rowid = fts.rowid
		WHERE decisions_fts MATCH ?
		ORDER BY rank
	`
	args := []interface{}{query}
	if limit > 0 {
		sqlQuery += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.QueryContext(context.Background(), sqlQuery, args...)
	if err != nil {
		// Fall back to LIKE search if FTS fails
		return m.searchDecisionsLike(query, limit)
	}
	defer rows.Close()

	var decisions []Decision
	for rows.Next() {
		var dec Decision
		var createdAt, updatedAt, outcomeAt string
		if err := rows.Scan(&dec.ID, &dec.Content, &dec.Rationale, &dec.Context, &dec.Status, &dec.Outcome,
			&dec.OutcomeNote, &outcomeAt, &dec.Scope, &dec.ScopePath, &dec.SessionID, &dec.Source, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan decision: %w", err)
		}
		dec.CreatedAt = parseTimeOrZero(createdAt)
		dec.UpdatedAt = parseTimeOrZero(updatedAt)
		dec.OutcomeAt = parseTimeOrZero(outcomeAt)
		decisions = append(decisions, dec)
	}
	return decisions, nil
}

// searchDecisionsLike is a fallback search using LIKE.
func (m *Memory) searchDecisionsLike(query string, limit int) ([]Decision, error) {
	sqlQuery := `
		SELECT id, content, rationale, context, status, outcome, outcome_note, outcome_at, scope, scope_path, session_id, source, created_at, updated_at
		FROM decisions
		WHERE content LIKE ? OR rationale LIKE ? OR context LIKE ?
		ORDER BY created_at DESC
	`
	pattern := "%" + query + "%"
	args := []interface{}{pattern, pattern, pattern}
	if limit > 0 {
		sqlQuery += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.QueryContext(context.Background(), sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search decisions: %w", err)
	}
	defer rows.Close()

	var decisions []Decision
	for rows.Next() {
		var dec Decision
		var createdAt, updatedAt, outcomeAt string
		if err := rows.Scan(&dec.ID, &dec.Content, &dec.Rationale, &dec.Context, &dec.Status, &dec.Outcome,
			&dec.OutcomeNote, &outcomeAt, &dec.Scope, &dec.ScopePath, &dec.SessionID, &dec.Source, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan decision: %w", err)
		}
		dec.CreatedAt = parseTimeOrZero(createdAt)
		dec.UpdatedAt = parseTimeOrZero(updatedAt)
		dec.OutcomeAt = parseTimeOrZero(outcomeAt)
		decisions = append(decisions, dec)
	}
	return decisions, nil
}

// RecordDecisionOutcome records the outcome of a decision.
func (m *Memory) RecordDecisionOutcome(id, outcome, note string) error {
	if outcome != DecisionOutcomeSuccessful && outcome != DecisionOutcomeFailed && outcome != DecisionOutcomeMixed {
		return fmt.Errorf("invalid outcome: %s (must be 'successful', 'failed', or 'mixed')", outcome)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	result, err := m.db.ExecContext(context.Background(), `
		UPDATE decisions
		SET outcome = ?, outcome_note = ?, outcome_at = ?, updated_at = ?
		WHERE id = ?
	`, outcome, note, now, now, id)
	if err != nil {
		return fmt.Errorf("record decision outcome: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("decision not found: %s", id)
	}

	// Feedback to linked learnings - adjust their confidence based on outcome
	learnings, err := m.GetLearningsForDecision(id)
	if err == nil && len(learnings) > 0 {
		for i := range learnings {
			l := &learnings[i]
			switch outcome {
			case DecisionOutcomeSuccessful:
				// Successful outcome reinforces the learning
				_ = m.ReinforceLearning(l.ID)
			case DecisionOutcomeFailed:
				// Failed outcome weakens the learning
				_ = m.WeakenLearning(l.ID)
				// Mixed outcome: no change to confidence
			}
		}
	}

	return nil
}

// UpdateDecisionStatus updates the status of a decision.
func (m *Memory) UpdateDecisionStatus(id, status string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := m.db.ExecContext(context.Background(), `UPDATE decisions SET status = ?, updated_at = ? WHERE id = ?`, status, now, id)
	if err != nil {
		return fmt.Errorf("update decision status: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("decision not found: %s", id)
	}
	return nil
}

// UpdateDecision updates a decision's content, rationale, and contextInfo.
func (m *Memory) UpdateDecision(id, content, rationale, contextInfo string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := m.db.ExecContext(context.Background(), `
		UPDATE decisions
		SET content = ?, rationale = ?, context = ?, updated_at = ?
		WHERE id = ?
	`, content, rationale, contextInfo, now, id)
	if err != nil {
		return fmt.Errorf("update decision: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("decision not found: %s", id)
	}
	return nil
}

// DeleteDecision removes a decision and its associated data from the database.
func (m *Memory) DeleteDecision(id string) error {
	// Delete associated links first (both as source and target)
	m.DeleteLinksForRecord(id)
	// Delete associated tags
	m.DeleteTagsForRecord(id, "decision")
	// Delete associated embedding
	m.DeleteEmbedding(id)
	// Delete the decision
	_, err := m.db.ExecContext(context.Background(), `DELETE FROM decisions WHERE id = ?`, id)
	return err
}

// CountDecisions returns the total number of decisions, optionally filtered.
func (m *Memory) CountDecisions(status, outcome string) (int, error) {
	query := "SELECT COUNT(*) FROM decisions WHERE 1=1"
	args := []interface{}{}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if outcome != "" {
		query += " AND outcome = ?"
		args = append(args, outcome)
	}
	var count int
	err := m.db.QueryRowContext(context.Background(), query, args...).Scan(&count)
	return count, err
}

// GetDecisionsAwaitingReview returns decisions older than the given age with unknown outcome.
func (m *Memory) GetDecisionsAwaitingReview(olderThanDays, limit int) ([]Decision, error) {
	cutoff := time.Now().AddDate(0, 0, -olderThanDays).Format(time.RFC3339)
	query := `
		SELECT id, content, rationale, context, status, outcome, outcome_note, outcome_at, scope, scope_path, session_id, source, created_at, updated_at
		FROM decisions
		WHERE outcome = 'unknown' AND status = 'active' AND created_at < ?
		ORDER BY created_at ASC
	`
	args := []interface{}{cutoff}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query decisions awaiting review: %w", err)
	}
	defer rows.Close()

	var decisions []Decision
	for rows.Next() {
		var dec Decision
		var createdAt, updatedAt, outcomeAt string
		if err := rows.Scan(&dec.ID, &dec.Content, &dec.Rationale, &dec.Context, &dec.Status, &dec.Outcome,
			&dec.OutcomeNote, &outcomeAt, &dec.Scope, &dec.ScopePath, &dec.SessionID, &dec.Source, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan decision: %w", err)
		}
		dec.CreatedAt = parseTimeOrZero(createdAt)
		dec.UpdatedAt = parseTimeOrZero(updatedAt)
		dec.OutcomeAt = parseTimeOrZero(outcomeAt)
		decisions = append(decisions, dec)
	}
	return decisions, nil
}

// GetDecisionsSince returns decisions created after the given time.
func (m *Memory) GetDecisionsSince(since time.Time, limit int) ([]Decision, error) {
	query := `
		SELECT id, content, rationale, context, status, outcome, outcome_note, outcome_at, scope, scope_path, session_id, source, created_at, updated_at
		FROM decisions
		WHERE outcome = 'unknown' AND status = 'active' AND created_at >= ?
		ORDER BY created_at ASC
	`
	args := []interface{}{since.Format(time.RFC3339)}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query decisions since: %w", err)
	}
	defer rows.Close()

	var decisions []Decision
	for rows.Next() {
		var dec Decision
		var createdAt, updatedAt, outcomeAt string
		if err := rows.Scan(&dec.ID, &dec.Content, &dec.Rationale, &dec.Context, &dec.Status, &dec.Outcome,
			&dec.OutcomeNote, &outcomeAt, &dec.Scope, &dec.ScopePath, &dec.SessionID, &dec.Source, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan decision: %w", err)
		}
		dec.CreatedAt = parseTimeOrZero(createdAt)
		dec.UpdatedAt = parseTimeOrZero(updatedAt)
		dec.OutcomeAt = parseTimeOrZero(outcomeAt)
		decisions = append(decisions, dec)
	}
	return decisions, nil
}

// DecisionConflict represents a potential conflict with an existing decision.
type DecisionConflict struct {
	ConflictingID string   `json:"conflictingId"`
	ConflictType  string   `json:"conflictType"` // "contradicts", "superseded_failed", "similar_failed"
	Reason        string   `json:"reason"`
	Decision      Decision `json:"decision"`
}

// CheckDecisionConflicts checks if a decision has any conflicts with existing decisions.
func (m *Memory) CheckDecisionConflicts(decisionID string) ([]DecisionConflict, error) {
	var conflicts []DecisionConflict

	// 1. Check for contradicting links
	links, err := m.GetAllLinksFor(decisionID)
	if err == nil {
		for i := range links {
			link := &links[i]
			if link.Relation == RelationContradicts {
				// Get the contradicting decision
				otherID := link.TargetID
				if link.TargetID == decisionID {
					otherID = link.SourceID
				}
				if otherDecision, err := m.GetDecision(otherID); err == nil {
					conflicts = append(conflicts, DecisionConflict{
						ConflictingID: otherID,
						ConflictType:  "contradicts",
						Reason:        "This decision has a 'contradicts' relationship",
						Decision:      *otherDecision,
					})
				}
			}
		}
	}

	// 2. Check if this decision superseded another (and that one had bad outcome)
	for i := range links {
		link := &links[i]
		if link.Relation == RelationSupersedes && link.SourceID == decisionID {
			if superseded, err := m.GetDecision(link.TargetID); err == nil {
				if superseded.Outcome == "failed" {
					conflicts = append(conflicts, DecisionConflict{
						ConflictingID: link.TargetID,
						ConflictType:  "superseded_failed",
						Reason:        "This decision supersedes a failed decision - ensure it addresses the failure",
						Decision:      *superseded,
					})
				}
			}
		}
	}

	return conflicts, nil
}

// FindSimilarDecisions finds decisions with similar content that may conflict.
func (m *Memory) FindSimilarDecisions(content, excludeID string, limit int) ([]Decision, error) {
	// Use FTS to find similar decisions
	if limit <= 0 {
		limit = 5
	}

	decisions, err := m.SearchDecisions(content, limit+1) // +1 to account for potential self-match
	if err != nil {
		return nil, err
	}

	// Filter out the excluded ID and return
	var filtered []Decision
	for i := range decisions {
		d := &decisions[i]
		if d.ID != excludeID {
			filtered = append(filtered, *d)
		}
		if len(filtered) >= limit {
			break
		}
	}

	return filtered, nil
}

// GetFailedDecisions returns decisions with failed outcomes.
func (m *Memory) GetFailedDecisions(limit int) ([]Decision, error) {
	return m.GetDecisions("", "failed", "", "", limit)
}

// DecisionChain represents the evolution chain of a decision.
type DecisionChain struct {
	Current         Decision          `json:"current"`
	Predecessors    []ChainedDecision `json:"predecessors"`    // Decisions this one superseded
	Successors      []ChainedDecision `json:"successors"`      // Decisions that supersede this
	LinkedLearnings []Learning        `json:"linkedLearnings"` // Learnings that informed this decision
}

// ChainedDecision wraps a decision with link information.
type ChainedDecision struct {
	Decision   Decision `json:"decision"`
	Relation   string   `json:"relation"`   // "supersedes", "contradicts"
	LinkReason string   `json:"linkReason"` // Why the link was made
}

// GetDecisionChain returns the full evolution chain for a decision.
func (m *Memory) GetDecisionChain(id string) (*DecisionChain, error) {
	// Get the current decision
	decision, err := m.GetDecision(id)
	if err != nil {
		return nil, fmt.Errorf("get decision: %w", err)
	}

	chain := &DecisionChain{
		Current:         *decision,
		Predecessors:    []ChainedDecision{},
		Successors:      []ChainedDecision{},
		LinkedLearnings: []Learning{},
	}

	// Get all links for this decision
	links, err := m.GetAllLinksFor(id)
	if err == nil {
		for i := range links {
			link := &links[i]
			// Skip non-decision relations
			if link.Relation != RelationSupersedes && link.Relation != RelationContradicts {
				continue
			}

			// Determine if this is a predecessor or successor
			if link.SourceID == id {
				// This decision supersedes/contradicts another (predecessor)
				if predecessor, err := m.GetDecision(link.TargetID); err == nil {
					chain.Predecessors = append(chain.Predecessors, ChainedDecision{
						Decision: *predecessor,
						Relation: link.Relation,
					})
				}
			} else if link.TargetID == id {
				// Another decision supersedes/contradicts this one (successor)
				if successor, err := m.GetDecision(link.SourceID); err == nil {
					chain.Successors = append(chain.Successors, ChainedDecision{
						Decision: *successor,
						Relation: link.Relation,
					})
				}
			}
		}
	}

	// Get linked learnings
	learnings, err := m.GetLearningsForDecision(id)
	if err == nil {
		chain.LinkedLearnings = learnings
	}

	return chain, nil
}

// GetDecisionTimeline returns all decisions ordered by creation date for timeline visualization.
func (m *Memory) GetDecisionTimeline(scope, scopePath string, limit int) ([]Decision, error) {
	query := `
		SELECT id, content, rationale, context, status, outcome, outcome_note, outcome_at, scope, scope_path, session_id, source, created_at, updated_at
		FROM decisions
		WHERE 1=1
	`
	args := []interface{}{}

	if scope != "" {
		query += ` AND scope = ?`
		args = append(args, scope)
	}
	if scopePath != "" {
		query += ` AND scope_path = ?`
		args = append(args, scopePath)
	}

	query += ` ORDER BY created_at ASC`

	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query decision timeline: %w", err)
	}
	defer rows.Close()

	var decisions []Decision
	for rows.Next() {
		var dec Decision
		var createdAt, updatedAt, outcomeAt string
		if err := rows.Scan(&dec.ID, &dec.Content, &dec.Rationale, &dec.Context, &dec.Status, &dec.Outcome,
			&dec.OutcomeNote, &outcomeAt, &dec.Scope, &dec.ScopePath, &dec.SessionID, &dec.Source, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan decision: %w", err)
		}
		dec.CreatedAt = parseTimeOrZero(createdAt)
		dec.UpdatedAt = parseTimeOrZero(updatedAt)
		dec.OutcomeAt = parseTimeOrZero(outcomeAt)
		decisions = append(decisions, dec)
	}
	return decisions, nil
}
