package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AuditAction represents the type of audited action.
type AuditAction string

const (
	// AuditActionDirectWrite is logged when a record is created bypassing proposals.
	AuditActionDirectWrite AuditAction = "direct_write"

	// AuditActionApprove is logged when a proposal is approved.
	AuditActionApprove AuditAction = "approve"

	// AuditActionReject is logged when a proposal is rejected.
	AuditActionReject AuditAction = "reject"
)

// AuditActorType represents who performed the action.
type AuditActorType string

const (
	// AuditActorHuman indicates action was performed by a human.
	AuditActorHuman AuditActorType = "human"

	// AuditActorAgent indicates action was performed by an AI agent.
	// Note: Agents should not be able to perform audited actions in practice,
	// but this type exists for completeness and debugging.
	AuditActorAgent AuditActorType = "agent"
)

// AuditLog represents an entry in the audit log.
type AuditLog struct {
	ID         string         `json:"id"`
	Action     AuditAction    `json:"action"`
	ActorType  AuditActorType `json:"actor_type"`
	ActorID    string         `json:"actor_id,omitempty"`
	TargetID   string         `json:"target_id"`
	TargetKind string         `json:"target_kind"`
	Details    string         `json:"details,omitempty"` // JSON details
	CreatedAt  time.Time      `json:"created_at"`
}

// AuditLogEntry contains the parameters for creating an audit log entry.
type AuditLogEntry struct {
	Action     AuditAction
	ActorType  AuditActorType
	ActorID    string // Optional identifier for the actor (e.g., username, session ID)
	TargetID   string // ID of the affected record (decision, learning, proposal, etc.)
	TargetKind string // Type of target: "decision", "learning", "proposal"
	Details    string // JSON details about the action
}

// AddAuditLog creates a new audit log entry.
func (m *Memory) AddAuditLog(entry AuditLogEntry) (string, error) {
	id := "audit_" + uuid.New().String()[:8]
	now := time.Now().UTC().Format(time.RFC3339)

	query := `
		INSERT INTO audit_log (id, action, actor_type, actor_id, target_id, target_kind, details, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := m.db.ExecContext(context.Background(), query,
		id,
		string(entry.Action),
		string(entry.ActorType),
		entry.ActorID,
		entry.TargetID,
		entry.TargetKind,
		entry.Details,
		now,
	)
	if err != nil {
		return "", fmt.Errorf("add audit log: %w", err)
	}

	return id, nil
}

// GetAuditLogs retrieves audit logs with optional filtering.
func (m *Memory) GetAuditLogs(action string, targetID string, limit int) ([]AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, action, actor_type, actor_id, target_id, target_kind, details, created_at
		FROM audit_log
		WHERE 1=1
	`
	args := []interface{}{}

	if action != "" {
		query += " AND action = ?"
		args = append(args, action)
	}
	if targetID != "" {
		query += " AND target_id = ?"
		args = append(args, targetID)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("get audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		var createdAt string
		if err := rows.Scan(
			&log.ID,
			&log.Action,
			&log.ActorType,
			&log.ActorID,
			&log.TargetID,
			&log.TargetKind,
			&log.Details,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		log.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// GetAuditLogByID retrieves a single audit log entry by ID.
func (m *Memory) GetAuditLogByID(id string) (*AuditLog, error) {
	query := `
		SELECT id, action, actor_type, actor_id, target_id, target_kind, details, created_at
		FROM audit_log
		WHERE id = ?
	`
	var log AuditLog
	var createdAt string
	err := m.db.QueryRowContext(context.Background(), query, id).Scan(
		&log.ID,
		&log.Action,
		&log.ActorType,
		&log.ActorID,
		&log.TargetID,
		&log.TargetKind,
		&log.Details,
		&createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get audit log: %w", err)
	}
	log.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &log, nil
}

// GetAuditLogsForTarget retrieves all audit logs for a specific target.
func (m *Memory) GetAuditLogsForTarget(targetID string) ([]AuditLog, error) {
	return m.GetAuditLogs("", targetID, 100)
}

// CountAuditLogs returns the total count of audit logs, optionally filtered by action.
func (m *Memory) CountAuditLogs(action string) (int, error) {
	query := "SELECT COUNT(*) FROM audit_log"
	args := []interface{}{}

	if action != "" {
		query += " WHERE action = ?"
		args = append(args, action)
	}

	var count int
	err := m.db.QueryRowContext(context.Background(), query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count audit logs: %w", err)
	}

	return count, nil
}
