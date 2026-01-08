package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Postmortem represents a failure record with structured analysis.
type Postmortem struct {
	ID              string    `json:"id"`                        // Prefix: "pm_"
	Title           string    `json:"title"`                     // Brief failure description
	WhatHappened    string    `json:"whatHappened"`              // Detailed description
	RootCause       string    `json:"rootCause,omitempty"`       // Why it failed
	LessonsLearned  []string  `json:"lessonsLearned,omitempty"`  // Key takeaways
	PreventionSteps []string  `json:"preventionSteps,omitempty"` // How to prevent recurrence
	Severity        string    `json:"severity"`                  // low, medium, high, critical
	Status          string    `json:"status"`                    // open, resolved, recurring
	AffectedFiles   []string  `json:"affectedFiles,omitempty"`   // Files involved
	RelatedDecision string    `json:"relatedDecision,omitempty"` // Decision that led to failure
	RelatedSession  string    `json:"relatedSession,omitempty"`  // Session where failure occurred
	CreatedAt       time.Time `json:"createdAt"`
	ResolvedAt      time.Time `json:"resolvedAt,omitempty"`
}

// PostmortemInput is the input for creating a postmortem.
type PostmortemInput struct {
	Title           string   `json:"title"`
	WhatHappened    string   `json:"whatHappened"`
	RootCause       string   `json:"rootCause,omitempty"`
	LessonsLearned  []string `json:"lessonsLearned,omitempty"`
	PreventionSteps []string `json:"preventionSteps,omitempty"`
	Severity        string   `json:"severity,omitempty"`
	AffectedFiles   []string `json:"affectedFiles,omitempty"`
	RelatedDecision string   `json:"relatedDecision,omitempty"`
	RelatedSession  string   `json:"relatedSession,omitempty"`
}

// PostmortemStats contains aggregated postmortem statistics.
type PostmortemStats struct {
	Total             int            `json:"total"`
	Open              int            `json:"open"`
	Resolved          int            `json:"resolved"`
	Recurring         int            `json:"recurring"`
	BySeverity        map[string]int `json:"bySeverity"`
	RecentPostmortems []Postmortem   `json:"recentPostmortems,omitempty"`
}

// StorePostmortem creates a new postmortem record.
func (m *Memory) StorePostmortem(input PostmortemInput) (*Postmortem, error) {
	id := "pm_" + uuid.New().String()[:8]
	now := time.Now()

	severity := input.Severity
	if severity == "" {
		severity = "medium"
	}

	lessonsJSON, err := json.Marshal(input.LessonsLearned)
	if err != nil {
		lessonsJSON = []byte("[]")
	}

	preventionJSON, err := json.Marshal(input.PreventionSteps)
	if err != nil {
		preventionJSON = []byte("[]")
	}

	filesJSON, err := json.Marshal(input.AffectedFiles)
	if err != nil {
		filesJSON = []byte("[]")
	}

	_, err = m.db.ExecContext(context.Background(), `
		INSERT INTO postmortems (
			id, title, what_happened, root_cause, lessons_learned,
			prevention_steps, severity, status, affected_files,
			related_decision, related_session, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'open', ?, ?, ?, ?)`,
		id, input.Title, input.WhatHappened, input.RootCause, string(lessonsJSON),
		string(preventionJSON), severity, string(filesJSON),
		input.RelatedDecision, input.RelatedSession, now.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("store postmortem: %w", err)
	}

	return &Postmortem{
		ID:              id,
		Title:           input.Title,
		WhatHappened:    input.WhatHappened,
		RootCause:       input.RootCause,
		LessonsLearned:  input.LessonsLearned,
		PreventionSteps: input.PreventionSteps,
		Severity:        severity,
		Status:          "open",
		AffectedFiles:   input.AffectedFiles,
		RelatedDecision: input.RelatedDecision,
		RelatedSession:  input.RelatedSession,
		CreatedAt:       now,
	}, nil
}

// GetPostmortem retrieves a postmortem by ID.
func (m *Memory) GetPostmortem(id string) (*Postmortem, error) {
	row := m.db.QueryRowContext(context.Background(), `
		SELECT id, title, what_happened, root_cause, lessons_learned,
			   prevention_steps, severity, status, affected_files,
			   related_decision, related_session, created_at, resolved_at
		FROM postmortems WHERE id = ?`, id)

	pm := &Postmortem{}
	var lessonsJSON, preventionJSON, filesJSON string
	var createdAt string
	var resolvedAt sql.NullString

	err := row.Scan(
		&pm.ID, &pm.Title, &pm.WhatHappened, &pm.RootCause, &lessonsJSON,
		&preventionJSON, &pm.Severity, &pm.Status, &filesJSON,
		&pm.RelatedDecision, &pm.RelatedSession, &createdAt, &resolvedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get postmortem: %w", err)
	}

	json.Unmarshal([]byte(lessonsJSON), &pm.LessonsLearned)
	json.Unmarshal([]byte(preventionJSON), &pm.PreventionSteps)
	json.Unmarshal([]byte(filesJSON), &pm.AffectedFiles)

	pm.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if resolvedAt.Valid {
		pm.ResolvedAt, _ = time.Parse(time.RFC3339, resolvedAt.String)
	}

	return pm, nil
}

// GetPostmortems retrieves postmortems with optional filters.
func (m *Memory) GetPostmortems(status, severity string, limit int) ([]Postmortem, error) {
	query := `
		SELECT id, title, what_happened, root_cause, lessons_learned,
			   prevention_steps, severity, status, affected_files,
			   related_decision, related_session, created_at, resolved_at
		FROM postmortems WHERE 1=1`
	args := []any{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if severity != "" {
		query += " AND severity = ?"
		args = append(args, severity)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("get postmortems: %w", err)
	}
	defer rows.Close()

	var postmortems []Postmortem
	for rows.Next() {
		pm := Postmortem{}
		var lessonsJSON, preventionJSON, filesJSON string
		var createdAt string
		var resolvedAt sql.NullString

		err := rows.Scan(
			&pm.ID, &pm.Title, &pm.WhatHappened, &pm.RootCause, &lessonsJSON,
			&preventionJSON, &pm.Severity, &pm.Status, &filesJSON,
			&pm.RelatedDecision, &pm.RelatedSession, &createdAt, &resolvedAt,
		)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(lessonsJSON), &pm.LessonsLearned)
		json.Unmarshal([]byte(preventionJSON), &pm.PreventionSteps)
		json.Unmarshal([]byte(filesJSON), &pm.AffectedFiles)

		pm.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if resolvedAt.Valid {
			pm.ResolvedAt, _ = time.Parse(time.RFC3339, resolvedAt.String)
		}

		postmortems = append(postmortems, pm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate postmortems: %w", err)
	}

	return postmortems, nil
}

// GetPostmortemsForFile retrieves postmortems affecting a specific file.
func (m *Memory) GetPostmortemsForFile(filePath string, limit int) ([]Postmortem, error) {
	// Use JSON array contains check
	query := `
		SELECT id, title, what_happened, root_cause, lessons_learned,
			   prevention_steps, severity, status, affected_files,
			   related_decision, related_session, created_at, resolved_at
		FROM postmortems
		WHERE affected_files LIKE ?
		ORDER BY created_at DESC LIMIT ?`

	// Pattern to match file in JSON array
	pattern := fmt.Sprintf("%%%q%%", filePath)

	rows, err := m.db.QueryContext(context.Background(), query, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("get postmortems for file: %w", err)
	}
	defer rows.Close()

	var postmortems []Postmortem
	for rows.Next() {
		pm := Postmortem{}
		var lessonsJSON, preventionJSON, filesJSON string
		var createdAt string
		var resolvedAt sql.NullString

		err := rows.Scan(
			&pm.ID, &pm.Title, &pm.WhatHappened, &pm.RootCause, &lessonsJSON,
			&preventionJSON, &pm.Severity, &pm.Status, &filesJSON,
			&pm.RelatedDecision, &pm.RelatedSession, &createdAt, &resolvedAt,
		)
		if err != nil {
			continue
		}

		json.Unmarshal([]byte(lessonsJSON), &pm.LessonsLearned)
		json.Unmarshal([]byte(preventionJSON), &pm.PreventionSteps)
		json.Unmarshal([]byte(filesJSON), &pm.AffectedFiles)

		pm.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if resolvedAt.Valid {
			pm.ResolvedAt, _ = time.Parse(time.RFC3339, resolvedAt.String)
		}

		postmortems = append(postmortems, pm)
	}

	return postmortems, nil
}

// ResolvePostmortem marks a postmortem as resolved.
func (m *Memory) ResolvePostmortem(id string) error {
	now := time.Now()
	result, err := m.db.ExecContext(context.Background(), `
		UPDATE postmortems
		SET status = 'resolved', resolved_at = ?
		WHERE id = ?`,
		now.Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("resolve postmortem: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("postmortem not found: %s", id)
	}
	return nil
}

// MarkPostmortemRecurring marks a postmortem as recurring.
func (m *Memory) MarkPostmortemRecurring(id string) error {
	result, err := m.db.ExecContext(context.Background(), `
		UPDATE postmortems
		SET status = 'recurring', resolved_at = NULL
		WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("mark postmortem recurring: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("postmortem not found: %s", id)
	}
	return nil
}

// UpdatePostmortem updates a postmortem.
func (m *Memory) UpdatePostmortem(id string, input PostmortemInput) error {
	lessonsJSON, _ := json.Marshal(input.LessonsLearned)
	preventionJSON, _ := json.Marshal(input.PreventionSteps)
	filesJSON, _ := json.Marshal(input.AffectedFiles)

	result, err := m.db.ExecContext(context.Background(), `
		UPDATE postmortems
		SET title = ?, what_happened = ?, root_cause = ?,
		    lessons_learned = ?, prevention_steps = ?,
		    severity = ?, affected_files = ?,
		    related_decision = ?, related_session = ?
		WHERE id = ?`,
		input.Title, input.WhatHappened, input.RootCause,
		string(lessonsJSON), string(preventionJSON),
		input.Severity, string(filesJSON),
		input.RelatedDecision, input.RelatedSession, id,
	)
	if err != nil {
		return fmt.Errorf("update postmortem: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("postmortem not found: %s", id)
	}
	return nil
}

// DeletePostmortem deletes a postmortem.
func (m *Memory) DeletePostmortem(id string) error {
	result, err := m.db.ExecContext(context.Background(), `DELETE FROM postmortems WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete postmortem: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("postmortem not found: %s", id)
	}
	return nil
}

// GetPostmortemStats returns aggregated postmortem statistics.
func (m *Memory) GetPostmortemStats() (*PostmortemStats, error) {
	stats := &PostmortemStats{
		BySeverity: make(map[string]int),
	}

	// Get total counts
	row := m.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM postmortems`)
	row.Scan(&stats.Total)

	row = m.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM postmortems WHERE status = 'open'`)
	row.Scan(&stats.Open)

	row = m.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM postmortems WHERE status = 'resolved'`)
	row.Scan(&stats.Resolved)

	row = m.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM postmortems WHERE status = 'recurring'`)
	row.Scan(&stats.Recurring)

	// Get counts by severity
	rows, err := m.db.QueryContext(context.Background(), `SELECT severity, COUNT(*) FROM postmortems GROUP BY severity`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var severity string
			var count int
			if rows.Scan(&severity, &count) == nil {
				stats.BySeverity[severity] = count
			}
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate severity counts: %w", err)
		}
	}

	// Get recent postmortems
	stats.RecentPostmortems, _ = m.GetPostmortems("", "", 5)

	return stats, nil
}

// ConvertPostmortemToLearning creates a learning from a postmortem's lessons.
func (m *Memory) ConvertPostmortemToLearning(postmortemID string) ([]string, error) {
	pm, err := m.GetPostmortem(postmortemID)
	if err != nil {
		return nil, err
	}
	if pm == nil {
		return nil, fmt.Errorf("postmortem not found: %s", postmortemID)
	}

	var learningIDs []string
	for _, lesson := range pm.LessonsLearned {
		id, err := m.AddLearning(Learning{
			Content:    lesson,
			Scope:      "palace",
			Confidence: 0.8,
			Source:     "postmortem:" + postmortemID,
		})
		if err != nil {
			continue
		}
		learningIDs = append(learningIDs, id)

		// Link learning to postmortem
		m.AddLink(Link{
			SourceID:   postmortemID,
			SourceKind: "postmortem",
			TargetID:   id,
			TargetKind: "learning",
			Relation:   RelationInspiredBy,
		})
	}

	return learningIDs, nil
}
