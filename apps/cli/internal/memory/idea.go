package memory

import (
	"fmt"
	"time"
)

// Idea represents an exploratory thought that may become a decision or learning.
type Idea struct {
	ID        string    `json:"id"`                  // Prefix: "i_"
	Content   string    `json:"content"`             // The idea itself
	Context   string    `json:"context,omitempty"`   // Surrounding context
	Status    string    `json:"status"`              // "active", "exploring", "implemented", "dropped"
	Scope     string    `json:"scope"`               // "file", "room", "palace"
	ScopePath string    `json:"scopePath,omitempty"` // e.g., "auth/login.go" or "auth"
	SessionID string    `json:"sessionId,omitempty"` // Optional session link
	Source    string    `json:"source"`              // "cli", "api", "auto-extract"
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

// IdeaStatus constants
const (
	IdeaStatusActive      = "active"
	IdeaStatusExploring   = "exploring"
	IdeaStatusImplemented = "implemented"
	IdeaStatusDropped     = "dropped"
)

// AddIdea stores a new idea in the database.
func (m *Memory) AddIdea(idea Idea) (string, error) {
	if idea.ID == "" {
		idea.ID = generateID("i")
	}
	if idea.Status == "" {
		idea.Status = IdeaStatusActive
	}
	if idea.Scope == "" {
		idea.Scope = "palace"
	}
	if idea.Source == "" {
		idea.Source = "cli"
	}
	now := time.Now().UTC()
	if idea.CreatedAt.IsZero() {
		idea.CreatedAt = now
	}
	if idea.UpdatedAt.IsZero() {
		idea.UpdatedAt = now
	}

	_, err := m.db.Exec(`
		INSERT INTO ideas (id, content, context, status, scope, scope_path, session_id, source, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, idea.ID, idea.Content, idea.Context, idea.Status, idea.Scope, idea.ScopePath, idea.SessionID, idea.Source,
		idea.CreatedAt.Format(time.RFC3339), idea.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return "", fmt.Errorf("insert idea: %w", err)
	}

	// Enqueue embedding generation (non-blocking)
	m.enqueueEmbedding(idea.ID, "idea", idea.Content)

	return idea.ID, nil
}

// GetIdea retrieves an idea by ID.
func (m *Memory) GetIdea(id string) (*Idea, error) {
	row := m.db.QueryRow(`
		SELECT id, content, context, status, scope, scope_path, session_id, source, created_at, updated_at
		FROM ideas WHERE id = ?
	`, id)

	var idea Idea
	var createdAt, updatedAt string
	err := row.Scan(&idea.ID, &idea.Content, &idea.Context, &idea.Status, &idea.Scope, &idea.ScopePath,
		&idea.SessionID, &idea.Source, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan idea: %w", err)
	}

	idea.CreatedAt = parseTimeOrZero(createdAt)
	idea.UpdatedAt = parseTimeOrZero(updatedAt)
	return &idea, nil
}

// GetIdeas retrieves ideas matching the given criteria.
func (m *Memory) GetIdeas(status, scope, scopePath string, limit int) ([]Idea, error) {
	query := `SELECT id, content, context, status, scope, scope_path, session_id, source, created_at, updated_at FROM ideas WHERE 1=1`
	args := []interface{}{}

	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
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

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query ideas: %w", err)
	}
	defer rows.Close()

	var ideas []Idea
	for rows.Next() {
		var idea Idea
		var createdAt, updatedAt string
		if err := rows.Scan(&idea.ID, &idea.Content, &idea.Context, &idea.Status, &idea.Scope, &idea.ScopePath,
			&idea.SessionID, &idea.Source, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan idea: %w", err)
		}
		idea.CreatedAt = parseTimeOrZero(createdAt)
		idea.UpdatedAt = parseTimeOrZero(updatedAt)
		ideas = append(ideas, idea)
	}
	return ideas, nil
}

// SearchIdeas searches ideas by content using FTS5.
func (m *Memory) SearchIdeas(query string, limit int) ([]Idea, error) {
	sqlQuery := `
		SELECT i.id, i.content, i.context, i.status, i.scope, i.scope_path, i.session_id, i.source, i.created_at, i.updated_at
		FROM ideas i
		JOIN ideas_fts fts ON i.rowid = fts.rowid
		WHERE ideas_fts MATCH ?
		ORDER BY rank
	`
	args := []interface{}{query}
	if limit > 0 {
		sqlQuery += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.Query(sqlQuery, args...)
	if err != nil {
		// Fall back to LIKE search if FTS fails
		return m.searchIdeasLike(query, limit)
	}
	defer rows.Close()

	var ideas []Idea
	for rows.Next() {
		var idea Idea
		var createdAt, updatedAt string
		if err := rows.Scan(&idea.ID, &idea.Content, &idea.Context, &idea.Status, &idea.Scope, &idea.ScopePath,
			&idea.SessionID, &idea.Source, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan idea: %w", err)
		}
		idea.CreatedAt = parseTimeOrZero(createdAt)
		idea.UpdatedAt = parseTimeOrZero(updatedAt)
		ideas = append(ideas, idea)
	}
	return ideas, nil
}

// searchIdeasLike is a fallback search using LIKE.
func (m *Memory) searchIdeasLike(query string, limit int) ([]Idea, error) {
	sqlQuery := `
		SELECT id, content, context, status, scope, scope_path, session_id, source, created_at, updated_at
		FROM ideas
		WHERE content LIKE ? OR context LIKE ?
		ORDER BY created_at DESC
	`
	pattern := "%" + query + "%"
	args := []interface{}{pattern, pattern}
	if limit > 0 {
		sqlQuery += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search ideas: %w", err)
	}
	defer rows.Close()

	var ideas []Idea
	for rows.Next() {
		var idea Idea
		var createdAt, updatedAt string
		if err := rows.Scan(&idea.ID, &idea.Content, &idea.Context, &idea.Status, &idea.Scope, &idea.ScopePath,
			&idea.SessionID, &idea.Source, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan idea: %w", err)
		}
		idea.CreatedAt = parseTimeOrZero(createdAt)
		idea.UpdatedAt = parseTimeOrZero(updatedAt)
		ideas = append(ideas, idea)
	}
	return ideas, nil
}

// UpdateIdeaStatus updates the status of an idea.
func (m *Memory) UpdateIdeaStatus(id, status string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := m.db.Exec(`UPDATE ideas SET status = ?, updated_at = ? WHERE id = ?`, status, now, id)
	if err != nil {
		return fmt.Errorf("update idea status: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("idea not found: %s", id)
	}
	return nil
}

// UpdateIdea updates an idea's content and context.
func (m *Memory) UpdateIdea(id, content, context string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := m.db.Exec(`UPDATE ideas SET content = ?, context = ?, updated_at = ? WHERE id = ?`,
		content, context, now, id)
	if err != nil {
		return fmt.Errorf("update idea: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("idea not found: %s", id)
	}
	return nil
}

// DeleteIdea removes an idea and its associated data from the database.
func (m *Memory) DeleteIdea(id string) error {
	// Delete associated links first (both as source and target)
	m.DeleteLinksForRecord(id)
	// Delete associated tags
	m.DeleteTagsForRecord(id, "idea")
	// Delete associated embedding
	m.DeleteEmbedding(id)
	// Delete the idea
	_, err := m.db.Exec(`DELETE FROM ideas WHERE id = ?`, id)
	return err
}

// CountIdeas returns the total number of ideas, optionally filtered by status.
func (m *Memory) CountIdeas(status string) (int, error) {
	query := "SELECT COUNT(*) FROM ideas"
	args := []interface{}{}
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	var count int
	err := m.db.QueryRow(query, args...).Scan(&count)
	return count, err
}
