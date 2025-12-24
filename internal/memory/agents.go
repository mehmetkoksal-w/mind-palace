package memory

import (
	"database/sql"
	"fmt"
	"time"
)

// ActiveAgent represents an agent currently working in the workspace.
type ActiveAgent struct {
	AgentID     string    `json:"agentId"`
	AgentType   string    `json:"agentType"`
	SessionID   string    `json:"sessionId"`
	Heartbeat   time.Time `json:"heartbeat"`
	CurrentFile string    `json:"currentFile"`
}

// Conflict represents a potential conflict between agents.
type Conflict struct {
	Path         string    `json:"path"`
	OtherSession string    `json:"otherSession"`
	OtherAgent   string    `json:"otherAgent"`
	LastTouched  time.Time `json:"lastTouched"`
	Severity     string    `json:"severity"` // "warning", "critical"
}

// RegisterAgent registers an agent as active in the workspace.
func (m *Memory) RegisterAgent(agentType, agentID, sessionID string) error {
	// Cleanup stale agents before registering (non-blocking, ignore errors)
	_, _ = m.CleanupStaleAgents(5 * time.Minute)

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := m.db.Exec(`
		INSERT INTO active_agents (agent_id, agent_type, session_id, last_heartbeat, current_file)
		VALUES (?, ?, ?, ?, '')
		ON CONFLICT(agent_id) DO UPDATE SET
			agent_type = excluded.agent_type,
			session_id = excluded.session_id,
			last_heartbeat = excluded.last_heartbeat
	`, agentID, agentType, sessionID, now)
	return err
}

// UnregisterAgent removes an agent from the active registry.
func (m *Memory) UnregisterAgent(agentID string) error {
	_, err := m.db.Exec(`DELETE FROM active_agents WHERE agent_id = ?`, agentID)
	return err
}

// Heartbeat updates the last heartbeat time for an agent.
func (m *Memory) Heartbeat(agentID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := m.db.Exec(`
		UPDATE active_agents SET last_heartbeat = ? WHERE agent_id = ?
	`, now, agentID)
	return err
}

// SetCurrentFile updates the file an agent is currently working on.
func (m *Memory) SetCurrentFile(agentID, filePath string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := m.db.Exec(`
		UPDATE active_agents SET current_file = ?, last_heartbeat = ? WHERE agent_id = ?
	`, filePath, now, agentID)
	return err
}

// GetActiveAgents returns all agents that have sent a heartbeat recently.
func (m *Memory) GetActiveAgents(staleThreshold time.Duration) ([]ActiveAgent, error) {
	cutoff := time.Now().UTC().Add(-staleThreshold).Format(time.RFC3339)

	rows, err := m.db.Query(`
		SELECT agent_id, agent_type, session_id, last_heartbeat, current_file
		FROM active_agents
		WHERE last_heartbeat > ?
	`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("query active agents: %w", err)
	}
	defer rows.Close()

	var agents []ActiveAgent
	for rows.Next() {
		var a ActiveAgent
		var heartbeat string
		if err := rows.Scan(&a.AgentID, &a.AgentType, &a.SessionID, &heartbeat, &a.CurrentFile); err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}
		a.Heartbeat = parseTimeOrZero(heartbeat)
		agents = append(agents, a)
	}
	return agents, nil
}

// GetAgentForFile returns the agent currently working on a file, if any.
func (m *Memory) GetAgentForFile(path string) (*ActiveAgent, error) {
	staleThreshold := 5 * time.Minute
	cutoff := time.Now().UTC().Add(-staleThreshold).Format(time.RFC3339)

	row := m.db.QueryRow(`
		SELECT agent_id, agent_type, session_id, last_heartbeat, current_file
		FROM active_agents
		WHERE current_file = ? AND last_heartbeat > ?
	`, path, cutoff)

	var a ActiveAgent
	var heartbeat string
	err := row.Scan(&a.AgentID, &a.AgentType, &a.SessionID, &heartbeat, &a.CurrentFile)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan agent: %w", err)
	}
	a.Heartbeat = parseTimeOrZero(heartbeat)
	return &a, nil
}

// CheckConflict checks if another agent is working on a file.
func (m *Memory) CheckConflict(sessionID, path string) (*Conflict, error) {
	// Check if another active agent is working on this file
	agent, err := m.GetAgentForFile(path)
	if err != nil {
		return nil, err
	}
	if agent != nil && agent.SessionID != sessionID {
		return &Conflict{
			Path:         path,
			OtherSession: agent.SessionID,
			OtherAgent:   agent.AgentType,
			LastTouched:  agent.Heartbeat,
			Severity:     "critical",
		}, nil
	}

	// Check recent activity on this file from other sessions
	rows, err := m.db.Query(`
		SELECT session_id, timestamp
		FROM activities
		WHERE target = ? AND session_id != ? AND kind IN ('file_edit', 'file_read')
		ORDER BY timestamp DESC
		LIMIT 1
	`, path, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query recent activity: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		var otherSession, ts string
		if err := rows.Scan(&otherSession, &ts); err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}
		lastTouched := parseTimeOrZero(ts)

		// Only report as warning if touched within last 10 minutes
		if time.Since(lastTouched) < 10*time.Minute {
			// Get agent type for the session (non-critical, ok if fails)
			var agentType string
			_ = m.db.QueryRow(`SELECT agent_type FROM sessions WHERE id = ?`, otherSession).Scan(&agentType)

			return &Conflict{
				Path:         path,
				OtherSession: otherSession,
				OtherAgent:   agentType,
				LastTouched:  lastTouched,
				Severity:     "warning",
			}, nil
		}
	}

	return nil, nil
}

// CleanupStaleAgents removes agents that haven't sent a heartbeat recently.
func (m *Memory) CleanupStaleAgents(staleThreshold time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-staleThreshold).Format(time.RFC3339)
	result, err := m.db.Exec(`DELETE FROM active_agents WHERE last_heartbeat < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
