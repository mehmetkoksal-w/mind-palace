package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Conversation stores full AI conversation transcripts for context replay.
type Conversation struct {
	ID        string    `json:"id"`                  // Prefix: "c_"
	AgentType string    `json:"agentType"`           // "claude-code", "cursor", etc.
	Summary   string    `json:"summary"`             // AI-generated summary
	Messages  []Message `json:"messages"`            // Full transcript
	Extracted []string  `json:"extracted,omitempty"` // IDs of records extracted
	SessionID string    `json:"sessionId,omitempty"` // Link to session
	CreatedAt time.Time `json:"createdAt"`
}

// Message represents a single message in a conversation.
type Message struct {
	Role      string    `json:"role"`      // "user", "assistant", "system"
	Content   string    `json:"content"`   // Message content
	Timestamp time.Time `json:"timestamp"` // When the message was sent
}

// AddConversation stores a new conversation.
func (m *Memory) AddConversation(c Conversation) (string, error) {
	if c.ID == "" {
		c.ID = "c_" + uuid.New().String()[:8]
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}

	// Serialize messages to JSON
	messagesJSON, err := json.Marshal(c.Messages)
	if err != nil {
		return "", err
	}

	// Serialize extracted IDs to JSON
	extractedJSON, err := json.Marshal(c.Extracted)
	if err != nil {
		return "", err
	}

	_, err = m.db.ExecContext(context.Background(), `
		INSERT INTO conversations (id, agent_type, summary, messages, extracted, session_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.AgentType, c.Summary, string(messagesJSON), string(extractedJSON), c.SessionID, c.CreatedAt.Format(time.RFC3339))

	return c.ID, err
}

// GetConversation retrieves a conversation by ID.
func (m *Memory) GetConversation(id string) (*Conversation, error) {
	row := m.db.QueryRowContext(context.Background(), `
		SELECT id, agent_type, summary, messages, extracted, session_id, created_at
		FROM conversations WHERE id = ?`, id)

	var c Conversation
	var messagesJSON, extractedJSON, sessionID sql.NullString
	var createdAt string

	err := row.Scan(&c.ID, &c.AgentType, &c.Summary, &messagesJSON, &extractedJSON, &sessionID, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Parse messages
	if messagesJSON.Valid && messagesJSON.String != "" {
		json.Unmarshal([]byte(messagesJSON.String), &c.Messages)
	}

	// Parse extracted IDs
	if extractedJSON.Valid && extractedJSON.String != "" {
		json.Unmarshal([]byte(extractedJSON.String), &c.Extracted)
	}

	if sessionID.Valid {
		c.SessionID = sessionID.String
	}

	c.CreatedAt = parseTimeOrZero(createdAt)

	return &c, nil
}

// GetConversations retrieves conversations with optional filters.
func (m *Memory) GetConversations(sessionID, agentType string, limit int) ([]Conversation, error) {
	query := `SELECT id, agent_type, summary, messages, extracted, session_id, created_at FROM conversations WHERE 1=1`
	args := []interface{}{}

	if sessionID != "" {
		query += ` AND session_id = ?`
		args = append(args, sessionID)
	}
	if agentType != "" {
		query += ` AND agent_type = ?`
		args = append(args, agentType)
	}

	query += ` ORDER BY created_at DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var c Conversation
		var messagesJSON, extractedJSON, sid sql.NullString
		var createdAt string

		err := rows.Scan(&c.ID, &c.AgentType, &c.Summary, &messagesJSON, &extractedJSON, &sid, &createdAt)
		if err != nil {
			return nil, err
		}

		if messagesJSON.Valid && messagesJSON.String != "" {
			json.Unmarshal([]byte(messagesJSON.String), &c.Messages)
		}
		if extractedJSON.Valid && extractedJSON.String != "" {
			json.Unmarshal([]byte(extractedJSON.String), &c.Extracted)
		}
		if sid.Valid {
			c.SessionID = sid.String
		}
		c.CreatedAt = parseTimeOrZero(createdAt)

		conversations = append(conversations, c)
	}

	return conversations, nil
}

// SearchConversations searches conversations by summary using FTS5.
func (m *Memory) SearchConversations(query string, limit int) ([]Conversation, error) {
	if limit <= 0 {
		limit = 10
	}

	sqlQuery := `
		SELECT c.id, c.agent_type, c.summary, c.messages, c.extracted, c.session_id, c.created_at
		FROM conversations c
		JOIN conversations_fts fts ON c.rowid = fts.rowid
		WHERE conversations_fts MATCH ?
		ORDER BY rank
		LIMIT ?`

	rows, err := m.db.QueryContext(context.Background(), sqlQuery, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var c Conversation
		var messagesJSON, extractedJSON, sessionID sql.NullString
		var createdAt string

		err := rows.Scan(&c.ID, &c.AgentType, &c.Summary, &messagesJSON, &extractedJSON, &sessionID, &createdAt)
		if err != nil {
			return nil, err
		}

		if messagesJSON.Valid && messagesJSON.String != "" {
			json.Unmarshal([]byte(messagesJSON.String), &c.Messages)
		}
		if extractedJSON.Valid && extractedJSON.String != "" {
			json.Unmarshal([]byte(extractedJSON.String), &c.Extracted)
		}
		if sessionID.Valid {
			c.SessionID = sessionID.String
		}
		c.CreatedAt = parseTimeOrZero(createdAt)

		conversations = append(conversations, c)
	}

	return conversations, nil
}

// DeleteConversation removes a conversation.
func (m *Memory) DeleteConversation(id string) error {
	result, err := m.db.ExecContext(context.Background(), `DELETE FROM conversations WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetConversationForSession retrieves the conversation for a specific session.
func (m *Memory) GetConversationForSession(sessionID string) (*Conversation, error) {
	row := m.db.QueryRowContext(context.Background(), `
		SELECT id, agent_type, summary, messages, extracted, session_id, created_at
		FROM conversations WHERE session_id = ?`, sessionID)

	var c Conversation
	var messagesJSON, extractedJSON, sid sql.NullString
	var createdAt string

	err := row.Scan(&c.ID, &c.AgentType, &c.Summary, &messagesJSON, &extractedJSON, &sid, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if messagesJSON.Valid && messagesJSON.String != "" {
		json.Unmarshal([]byte(messagesJSON.String), &c.Messages)
	}
	if extractedJSON.Valid && extractedJSON.String != "" {
		json.Unmarshal([]byte(extractedJSON.String), &c.Extracted)
	}
	if sid.Valid {
		c.SessionID = sid.String
	}
	c.CreatedAt = parseTimeOrZero(createdAt)

	return &c, nil
}

// UpdateConversationExtracted updates the extracted IDs for a conversation.
func (m *Memory) UpdateConversationExtracted(id string, extracted []string) error {
	extractedJSON, err := json.Marshal(extracted)
	if err != nil {
		return err
	}
	_, err = m.db.ExecContext(context.Background(), `UPDATE conversations SET extracted = ? WHERE id = ?`, string(extractedJSON), id)
	return err
}

// EndSessionWithConversation ends a session and stores the conversation.
func (m *Memory) EndSessionWithConversation(sessionID, summary string, messages []Message, agentType string) error {
	// End the session (existing functionality)
	if err := m.EndSession(sessionID, "completed", summary); err != nil {
		return err
	}

	// Store conversation
	conv := Conversation{
		AgentType: agentType,
		Summary:   summary,
		Messages:  messages,
		SessionID: sessionID,
	}

	_, err := m.AddConversation(conv)
	return err
}

// CountConversations returns the total number of conversations.
func (m *Memory) CountConversations() (int, error) {
	var count int
	err := m.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM conversations`).Scan(&count)
	return count, err
}

// GetRecentConversations returns the most recent conversations.
func (m *Memory) GetRecentConversations(limit int) ([]Conversation, error) {
	if limit <= 0 {
		limit = 10
	}
	return m.GetConversations("", "", limit)
}
