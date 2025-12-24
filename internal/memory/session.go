package memory

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// StartSession creates a new session and returns its ID.
func (m *Memory) StartSession(agentType, agentID, goal string) (*Session, error) {
	id := generateID("ses")
	now := time.Now().UTC()

	_, err := m.db.Exec(`
		INSERT INTO sessions (id, agent_type, agent_id, goal, started_at, last_activity, state)
		VALUES (?, ?, ?, ?, ?, ?, 'active')
	`, id, agentType, agentID, goal, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}

	return &Session{
		ID:           id,
		AgentType:    agentType,
		AgentID:      agentID,
		Goal:         goal,
		StartedAt:    now,
		LastActivity: now,
		State:        "active",
	}, nil
}

// GetSession retrieves a session by ID.
func (m *Memory) GetSession(id string) (*Session, error) {
	row := m.db.QueryRow(`
		SELECT id, agent_type, agent_id, goal, started_at, last_activity, state, summary
		FROM sessions WHERE id = ?
	`, id)

	var s Session
	var startedAt, lastActivity string
	err := row.Scan(&s.ID, &s.AgentType, &s.AgentID, &s.Goal, &startedAt, &lastActivity, &s.State, &s.Summary)
	if err != nil {
		return nil, fmt.Errorf("scan session: %w", err)
	}

	s.StartedAt = parseTimeOrZero(startedAt)
	s.LastActivity = parseTimeOrZero(lastActivity)
	return &s, nil
}

// ListSessions returns sessions matching the given filters.
func (m *Memory) ListSessions(activeOnly bool, limit int) ([]Session, error) {
	query := `SELECT id, agent_type, agent_id, goal, started_at, last_activity, state, summary FROM sessions`
	args := []interface{}{}

	if activeOnly {
		query += ` WHERE state = 'active'`
	}
	query += ` ORDER BY last_activity DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		var startedAt, lastActivity string
		if err := rows.Scan(&s.ID, &s.AgentType, &s.AgentID, &s.Goal, &startedAt, &lastActivity, &s.State, &s.Summary); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		s.StartedAt = parseTimeOrZero(startedAt)
		s.LastActivity = parseTimeOrZero(lastActivity)
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// UpdateSessionActivity updates the last activity time for a session.
func (m *Memory) UpdateSessionActivity(sessionID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := m.db.Exec(`UPDATE sessions SET last_activity = ? WHERE id = ?`, now, sessionID)
	return err
}

// EndSession marks a session as completed or abandoned.
func (m *Memory) EndSession(sessionID, state, summary string) error {
	if state == "" {
		state = "completed"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := m.db.Exec(`
		UPDATE sessions SET state = ?, summary = ?, last_activity = ? WHERE id = ?
	`, state, summary, now, sessionID)
	return err
}

// LogActivity records an activity within a session.
func (m *Memory) LogActivity(sessionID string, act Activity) error {
	if act.ID == "" {
		act.ID = generateID("act")
	}
	if act.Timestamp.IsZero() {
		act.Timestamp = time.Now().UTC()
	}
	if act.Outcome == "" {
		act.Outcome = "unknown"
	}
	if act.Details == "" {
		act.Details = "{}"
	}

	_, err := m.db.Exec(`
		INSERT INTO activities (id, session_id, kind, target, details, timestamp, outcome)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, act.ID, sessionID, act.Kind, act.Target, act.Details, act.Timestamp.Format(time.RFC3339), act.Outcome)
	if err != nil {
		return fmt.Errorf("insert activity: %w", err)
	}

	// Update session last activity
	return m.UpdateSessionActivity(sessionID)
}

// GetActivities retrieves activities for a session or file.
func (m *Memory) GetActivities(sessionID, filePath string, limit int) ([]Activity, error) {
	query := `SELECT id, session_id, kind, target, details, timestamp, outcome FROM activities WHERE 1=1`
	args := []interface{}{}

	if sessionID != "" {
		query += ` AND session_id = ?`
		args = append(args, sessionID)
	}
	if filePath != "" {
		query += ` AND target = ?`
		args = append(args, filePath)
	}
	query += ` ORDER BY timestamp DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query activities: %w", err)
	}
	defer rows.Close()

	var activities []Activity
	for rows.Next() {
		var a Activity
		var ts string
		if err := rows.Scan(&a.ID, &a.SessionID, &a.Kind, &a.Target, &a.Details, &ts, &a.Outcome); err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}
		a.Timestamp = parseTimeOrZero(ts)
		activities = append(activities, a)
	}
	return activities, nil
}

// RecordOutcome records the outcome of a session.
func (m *Memory) RecordOutcome(sessionID, outcome, summary string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	// Log as activity
	act := Activity{
		ID:        generateID("act"),
		SessionID: sessionID,
		Kind:      "outcome",
		Target:    "",
		Details:   fmt.Sprintf(`{"outcome":"%s"}`, outcome),
		Outcome:   outcome,
	}
	if err := m.LogActivity(sessionID, act); err != nil {
		return err
	}

	// Update session
	_, err := m.db.Exec(`
		UPDATE sessions SET summary = ?, last_activity = ? WHERE id = ?
	`, summary, now, sessionID)
	return err
}

// CleanupAbandonedSessions marks old active sessions as abandoned.
func (m *Memory) CleanupAbandonedSessions(maxAge time.Duration) (int64, error) {
	cutoff := time.Now().Add(-maxAge).Format(time.RFC3339)
	result, err := m.db.Exec(`
		UPDATE sessions SET state = 'abandoned'
		WHERE state = 'active' AND last_activity < ?
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// PurgeOldSessions deletes sessions older than maxAge.
func (m *Memory) PurgeOldSessions(maxAge time.Duration) (int64, error) {
	cutoff := time.Now().Add(-maxAge).Format(time.RFC3339)
	result, err := m.db.Exec(`DELETE FROM sessions WHERE started_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// generateID creates a random ID with the given prefix.
func generateID(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if random fails
		return prefix + "_" + fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(b)
}
