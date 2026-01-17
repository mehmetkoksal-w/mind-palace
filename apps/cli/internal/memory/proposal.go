package memory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// ProposalStatus constants
const (
	ProposalStatusPending  = "pending"
	ProposalStatusApproved = "approved"
	ProposalStatusRejected = "rejected"
	ProposalStatusExpired  = "expired"
)

// ProposedAs constants - what type of record this proposal would become
const (
	ProposedAsDecision = "decision"
	ProposedAsLearning = "learning"
)

// Proposal represents a proposed decision or learning awaiting human approval.
type Proposal struct {
	ID                       string    `json:"id"`
	ProposedAs               string    `json:"proposedAs"`               // "decision" or "learning"
	Content                  string    `json:"content"`                  // The proposed content
	Context                  string    `json:"context,omitempty"`        // Additional context
	Rationale                string    `json:"rationale,omitempty"`      // For decisions
	Scope                    string    `json:"scope"`                    // "file", "room", "palace"
	ScopePath                string    `json:"scopePath,omitempty"`      // e.g., "auth/login.go" or "auth"
	Source                   string    `json:"source"`                   // "agent", "auto-extract", etc.
	SessionID                string    `json:"sessionId,omitempty"`      // Optional session link
	AgentType                string    `json:"agentType,omitempty"`      // Type of agent that created it
	EvidenceRefs             string    `json:"evidenceRefs,omitempty"`   // JSON with evidence references
	ClassificationConfidence float64   `json:"classificationConfidence"` // Auto-classification confidence
	ClassificationSignals    string    `json:"classificationSignals"`    // JSON array of signals
	DedupeKey                string    `json:"dedupeKey,omitempty"`      // For duplicate detection
	Status                   string    `json:"status"`                   // pending, approved, rejected, expired
	ReviewedBy               string    `json:"reviewedBy,omitempty"`     // Who reviewed it
	ReviewedAt               time.Time `json:"reviewedAt,omitempty"`     // When it was reviewed
	ReviewNote               string    `json:"reviewNote,omitempty"`     // Note from reviewer
	PromotedToID             string    `json:"promotedToId,omitempty"`   // ID of created decision/learning
	CreatedAt                time.Time `json:"createdAt"`
	ExpiresAt                time.Time `json:"expiresAt,omitempty"` // When proposal expires
	ArchivedAt               time.Time `json:"archivedAt,omitempty"`
}

// EvidenceRef represents evidence supporting a proposal.
type EvidenceRef struct {
	SessionID      string  `json:"sessionId,omitempty"`
	ConversationID string  `json:"conversationId,omitempty"`
	Extractor      string  `json:"extractor,omitempty"`
	SourceRecord   string  `json:"sourceRecord,omitempty"`
	TargetRecord   string  `json:"targetRecord,omitempty"`
	Confidence     float64 `json:"confidence,omitempty"`
	Explanation    string  `json:"explanation,omitempty"`
}

// GenerateDedupeKey creates a deterministic key for duplicate detection.
func GenerateDedupeKey(proposedAs, content, scope, scopePath string) string {
	data := fmt.Sprintf("%s:%s:%s:%s", proposedAs, content, scope, scopePath)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes (32 hex chars)
}

// AddProposal stores a new proposal in the database.
func (m *Memory) AddProposal(p Proposal) (string, error) {
	if p.ID == "" {
		p.ID = generateID("prop")
	}
	if p.ProposedAs == "" {
		return "", fmt.Errorf("proposedAs is required")
	}
	if p.Content == "" {
		return "", fmt.Errorf("content is required")
	}
	if p.Source == "" {
		p.Source = "agent"
	}
	if p.Scope == "" {
		p.Scope = "palace"
	}
	if p.Status == "" {
		p.Status = ProposalStatusPending
	}
	if p.EvidenceRefs == "" {
		p.EvidenceRefs = "{}"
	}
	if p.ClassificationSignals == "" {
		p.ClassificationSignals = "[]"
	}

	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}

	// Generate dedupe key if not provided
	if p.DedupeKey == "" {
		p.DedupeKey = GenerateDedupeKey(p.ProposedAs, p.Content, p.Scope, p.ScopePath)
	}

	// Format timestamps
	expiresAt := ""
	if !p.ExpiresAt.IsZero() {
		expiresAt = p.ExpiresAt.Format(time.RFC3339)
	}
	reviewedAt := ""
	if !p.ReviewedAt.IsZero() {
		reviewedAt = p.ReviewedAt.Format(time.RFC3339)
	}
	archivedAt := ""
	if !p.ArchivedAt.IsZero() {
		archivedAt = p.ArchivedAt.Format(time.RFC3339)
	}

	_, err := m.db.ExecContext(context.Background(), `
		INSERT INTO proposals (id, proposed_as, content, context, rationale, scope, scope_path, source, session_id, agent_type, evidence_refs, classification_confidence, classification_signals, dedupe_key, status, reviewed_by, reviewed_at, review_note, promoted_to_id, created_at, expires_at, archived_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.ProposedAs, p.Content, p.Context, p.Rationale, p.Scope, p.ScopePath, p.Source, p.SessionID, p.AgentType, p.EvidenceRefs, p.ClassificationConfidence, p.ClassificationSignals, p.DedupeKey, p.Status, p.ReviewedBy, reviewedAt, p.ReviewNote, p.PromotedToID,
		p.CreatedAt.Format(time.RFC3339), expiresAt, archivedAt)
	if err != nil {
		return "", fmt.Errorf("insert proposal: %w", err)
	}

	return p.ID, nil
}

// GetProposal retrieves a proposal by ID.
func (m *Memory) GetProposal(id string) (*Proposal, error) {
	row := m.db.QueryRowContext(context.Background(), `
		SELECT id, proposed_as, content, context, rationale, scope, scope_path, source, session_id, agent_type, evidence_refs, classification_confidence, classification_signals, dedupe_key, status, reviewed_by, reviewed_at, review_note, promoted_to_id, created_at, expires_at, archived_at
		FROM proposals WHERE id = ?
	`, id)

	var p Proposal
	var createdAt, expiresAt, reviewedAt, archivedAt string
	err := row.Scan(&p.ID, &p.ProposedAs, &p.Content, &p.Context, &p.Rationale, &p.Scope, &p.ScopePath, &p.Source, &p.SessionID, &p.AgentType, &p.EvidenceRefs, &p.ClassificationConfidence, &p.ClassificationSignals, &p.DedupeKey, &p.Status, &p.ReviewedBy, &reviewedAt, &p.ReviewNote, &p.PromotedToID, &createdAt, &expiresAt, &archivedAt)
	if err != nil {
		return nil, fmt.Errorf("scan proposal: %w", err)
	}

	p.CreatedAt = parseTimeOrZero(createdAt)
	p.ExpiresAt = parseTimeOrZero(expiresAt)
	p.ReviewedAt = parseTimeOrZero(reviewedAt)
	p.ArchivedAt = parseTimeOrZero(archivedAt)
	return &p, nil
}

// GetProposals retrieves proposals matching the given criteria.
func (m *Memory) GetProposals(status, proposedAs string, limit int) ([]Proposal, error) {
	query := `SELECT id, proposed_as, content, context, rationale, scope, scope_path, source, session_id, agent_type, evidence_refs, classification_confidence, classification_signals, dedupe_key, status, reviewed_by, reviewed_at, review_note, promoted_to_id, created_at, expires_at, archived_at FROM proposals WHERE 1=1`
	args := []interface{}{}

	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	if proposedAs != "" {
		query += ` AND proposed_as = ?`
		args = append(args, proposedAs)
	}
	query += ` ORDER BY created_at DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := m.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("query proposals: %w", err)
	}
	defer rows.Close()

	var proposals []Proposal
	for rows.Next() {
		var p Proposal
		var createdAt, expiresAt, reviewedAt, archivedAt string
		if err := rows.Scan(&p.ID, &p.ProposedAs, &p.Content, &p.Context, &p.Rationale, &p.Scope, &p.ScopePath, &p.Source, &p.SessionID, &p.AgentType, &p.EvidenceRefs, &p.ClassificationConfidence, &p.ClassificationSignals, &p.DedupeKey, &p.Status, &p.ReviewedBy, &reviewedAt, &p.ReviewNote, &p.PromotedToID, &createdAt, &expiresAt, &archivedAt); err != nil {
			return nil, fmt.Errorf("scan proposal: %w", err)
		}
		p.CreatedAt = parseTimeOrZero(createdAt)
		p.ExpiresAt = parseTimeOrZero(expiresAt)
		p.ReviewedAt = parseTimeOrZero(reviewedAt)
		p.ArchivedAt = parseTimeOrZero(archivedAt)
		proposals = append(proposals, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proposals: %w", err)
	}
	return proposals, nil
}

// SearchProposals searches proposals by content using FTS5.
func (m *Memory) SearchProposals(query string, limit int) ([]Proposal, error) {
	sqlQuery := `
		SELECT p.id, p.proposed_as, p.content, p.context, p.rationale, p.scope, p.scope_path, p.source, p.session_id, p.agent_type, p.evidence_refs, p.classification_confidence, p.classification_signals, p.dedupe_key, p.status, p.reviewed_by, p.reviewed_at, p.review_note, p.promoted_to_id, p.created_at, p.expires_at, p.archived_at
		FROM proposals p
		JOIN proposals_fts fts ON p.rowid = fts.rowid
		WHERE proposals_fts MATCH ?
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
		return m.searchProposalsLike(query, limit)
	}
	defer rows.Close()

	var proposals []Proposal
	for rows.Next() {
		var p Proposal
		var createdAt, expiresAt, reviewedAt, archivedAt string
		if err := rows.Scan(&p.ID, &p.ProposedAs, &p.Content, &p.Context, &p.Rationale, &p.Scope, &p.ScopePath, &p.Source, &p.SessionID, &p.AgentType, &p.EvidenceRefs, &p.ClassificationConfidence, &p.ClassificationSignals, &p.DedupeKey, &p.Status, &p.ReviewedBy, &reviewedAt, &p.ReviewNote, &p.PromotedToID, &createdAt, &expiresAt, &archivedAt); err != nil {
			return nil, fmt.Errorf("scan proposal: %w", err)
		}
		p.CreatedAt = parseTimeOrZero(createdAt)
		p.ExpiresAt = parseTimeOrZero(expiresAt)
		p.ReviewedAt = parseTimeOrZero(reviewedAt)
		p.ArchivedAt = parseTimeOrZero(archivedAt)
		proposals = append(proposals, p)
	}
	return proposals, nil
}

// searchProposalsLike is a fallback search using LIKE.
func (m *Memory) searchProposalsLike(query string, limit int) ([]Proposal, error) {
	sqlQuery := `
		SELECT id, proposed_as, content, context, rationale, scope, scope_path, source, session_id, agent_type, evidence_refs, classification_confidence, classification_signals, dedupe_key, status, reviewed_by, reviewed_at, review_note, promoted_to_id, created_at, expires_at, archived_at
		FROM proposals
		WHERE (content LIKE ? OR context LIKE ? OR rationale LIKE ?)
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
		return nil, fmt.Errorf("search proposals: %w", err)
	}
	defer rows.Close()

	var proposals []Proposal
	for rows.Next() {
		var p Proposal
		var createdAt, expiresAt, reviewedAt, archivedAt string
		if err := rows.Scan(&p.ID, &p.ProposedAs, &p.Content, &p.Context, &p.Rationale, &p.Scope, &p.ScopePath, &p.Source, &p.SessionID, &p.AgentType, &p.EvidenceRefs, &p.ClassificationConfidence, &p.ClassificationSignals, &p.DedupeKey, &p.Status, &p.ReviewedBy, &reviewedAt, &p.ReviewNote, &p.PromotedToID, &createdAt, &expiresAt, &archivedAt); err != nil {
			return nil, fmt.Errorf("scan proposal: %w", err)
		}
		p.CreatedAt = parseTimeOrZero(createdAt)
		p.ExpiresAt = parseTimeOrZero(expiresAt)
		p.ReviewedAt = parseTimeOrZero(reviewedAt)
		p.ArchivedAt = parseTimeOrZero(archivedAt)
		proposals = append(proposals, p)
	}
	return proposals, nil
}

// CheckDuplicateProposal checks if a proposal with the same dedupe key already exists.
func (m *Memory) CheckDuplicateProposal(dedupeKey string) (*Proposal, error) {
	if dedupeKey == "" {
		return nil, nil
	}

	row := m.db.QueryRowContext(context.Background(), `
		SELECT id, proposed_as, content, context, rationale, scope, scope_path, source, session_id, agent_type, evidence_refs, classification_confidence, classification_signals, dedupe_key, status, reviewed_by, reviewed_at, review_note, promoted_to_id, created_at, expires_at, archived_at
		FROM proposals WHERE dedupe_key = ? AND status = 'pending'
	`, dedupeKey)

	var p Proposal
	var createdAt, expiresAt, reviewedAt, archivedAt string
	err := row.Scan(&p.ID, &p.ProposedAs, &p.Content, &p.Context, &p.Rationale, &p.Scope, &p.ScopePath, &p.Source, &p.SessionID, &p.AgentType, &p.EvidenceRefs, &p.ClassificationConfidence, &p.ClassificationSignals, &p.DedupeKey, &p.Status, &p.ReviewedBy, &reviewedAt, &p.ReviewNote, &p.PromotedToID, &createdAt, &expiresAt, &archivedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	p.CreatedAt = parseTimeOrZero(createdAt)
	p.ExpiresAt = parseTimeOrZero(expiresAt)
	p.ReviewedAt = parseTimeOrZero(reviewedAt)
	p.ArchivedAt = parseTimeOrZero(archivedAt)
	return &p, nil
}

// CountProposals returns the count of proposals by status.
func (m *Memory) CountProposals(status string) (int, error) {
	query := "SELECT COUNT(*) FROM proposals"
	args := []interface{}{}
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	var count int
	err := m.db.QueryRowContext(context.Background(), query, args...).Scan(&count)
	return count, err
}

// ApproveProposal approves a proposal and creates the corresponding decision/learning.
// Returns the ID of the promoted record.
func (m *Memory) ApproveProposal(proposalID, reviewedBy, reviewNote string) (string, error) {
	// Get the proposal
	proposal, err := m.GetProposal(proposalID)
	if err != nil {
		return "", fmt.Errorf("get proposal: %w", err)
	}

	// Check if already processed
	if proposal.Status != ProposalStatusPending {
		return "", fmt.Errorf("proposal %s is already %s", proposalID, proposal.Status)
	}

	// Begin transaction to ensure atomicity
	ctx := context.Background()
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on any error

	now := time.Now().UTC()
	var promotedID string

	// Create the corresponding record based on proposedAs
	switch proposal.ProposedAs {
	case ProposedAsDecision:
		dec := Decision{
			Content:                content(proposal),
			Rationale:              proposal.Rationale,
			Context:                proposal.Context,
			Status:                 DecisionStatusActive,
			Outcome:                DecisionOutcomeUnknown,
			Scope:                  proposal.Scope,
			ScopePath:              proposal.ScopePath,
			SessionID:              proposal.SessionID,
			Source:                 proposal.Source,
			Authority:              string(AuthorityApproved),
			PromotedFromProposalID: proposalID,
			CreatedAt:              now,
		}
		promotedID, err = m.AddDecision(dec)
		if err != nil {
			return "", fmt.Errorf("create decision: %w", err)
		}

	case ProposedAsLearning:
		// Parse confidence from evidence if available
		confidence := 0.5
		if proposal.ClassificationConfidence > 0 {
			confidence = proposal.ClassificationConfidence
		}

		learning := Learning{
			Content:                proposal.Content,
			Scope:                  proposal.Scope,
			ScopePath:              proposal.ScopePath,
			SessionID:              proposal.SessionID,
			Source:                 proposal.Source,
			Confidence:             confidence,
			Authority:              string(AuthorityApproved),
			PromotedFromProposalID: proposalID,
			CreatedAt:              now,
		}
		promotedID, err = m.AddLearning(learning)
		if err != nil {
			return "", fmt.Errorf("create learning: %w", err)
		}

	default:
		return "", fmt.Errorf("unknown proposedAs: %s", proposal.ProposedAs)
	}

	// Update the proposal to approved status
	_, err = tx.ExecContext(ctx, `
		UPDATE proposals
		SET status = ?, reviewed_by = ?, reviewed_at = ?, review_note = ?, promoted_to_id = ?
		WHERE id = ?
	`, ProposalStatusApproved, reviewedBy, now.Format(time.RFC3339), reviewNote, promotedID, proposalID)
	if err != nil {
		return "", fmt.Errorf("update proposal status: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("commit transaction: %w", err)
	}

	return promotedID, nil
}

// content is a helper to get the content from a proposal
func content(p *Proposal) string {
	return p.Content
}

// RejectProposal rejects a proposal with a reason.
func (m *Memory) RejectProposal(proposalID, reviewedBy, reviewNote string) error {
	// Get the proposal first to validate it exists and is pending
	proposal, err := m.GetProposal(proposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	if proposal.Status != ProposalStatusPending {
		return fmt.Errorf("proposal %s is already %s", proposalID, proposal.Status)
	}

	now := time.Now().UTC()
	_, err = m.db.ExecContext(context.Background(), `
		UPDATE proposals
		SET status = ?, reviewed_by = ?, reviewed_at = ?, review_note = ?
		WHERE id = ?
	`, ProposalStatusRejected, reviewedBy, now.Format(time.RFC3339), reviewNote, proposalID)
	if err != nil {
		return fmt.Errorf("update proposal status: %w", err)
	}

	return nil
}

// ExpireProposal marks a proposal as expired.
func (m *Memory) ExpireProposal(proposalID string) error {
	now := time.Now().UTC()
	result, err := m.db.ExecContext(context.Background(), `
		UPDATE proposals
		SET status = ?, archived_at = ?
		WHERE id = ? AND status = ?
	`, ProposalStatusExpired, now.Format(time.RFC3339), proposalID, ProposalStatusPending)
	if err != nil {
		return fmt.Errorf("expire proposal: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("proposal %s not found or not pending", proposalID)
	}
	return nil
}

// ExpireOldProposals marks proposals as expired if they're past their expiration date.
func (m *Memory) ExpireOldProposals() (int64, error) {
	now := time.Now().UTC()
	result, err := m.db.ExecContext(context.Background(), `
		UPDATE proposals
		SET status = ?, archived_at = ?
		WHERE status = ? AND expires_at != '' AND expires_at < ?
	`, ProposalStatusExpired, now.Format(time.RFC3339), ProposalStatusPending, now.Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("expire old proposals: %w", err)
	}
	return result.RowsAffected()
}

// DeleteProposal removes a proposal from the database (admin only).
func (m *Memory) DeleteProposal(id string) error {
	_, err := m.db.ExecContext(context.Background(), `DELETE FROM proposals WHERE id = ?`, id)
	return err
}

// SetEvidenceRefs sets the evidence references for a proposal as JSON.
func (m *Memory) SetEvidenceRefs(proposalID string, refs EvidenceRef) error {
	data, err := json.Marshal(refs)
	if err != nil {
		return fmt.Errorf("marshal evidence refs: %w", err)
	}

	_, err = m.db.ExecContext(context.Background(), `
		UPDATE proposals SET evidence_refs = ? WHERE id = ?
	`, string(data), proposalID)
	return err
}

// GetPendingProposalsCount returns the count of pending proposals.
func (m *Memory) GetPendingProposalsCount() (int, error) {
	return m.CountProposals(ProposalStatusPending)
}
