package butler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// toolStoreDirect stores a record directly, bypassing the proposal system.
// This tool is only available in human mode.
// Creates an audit log entry for accountability.
func (s *MCPServer) toolStoreDirect(id any, args map[string]interface{}) jsonRPCResponse {
	content, _ := args["content"].(string)
	if content == "" {
		return s.toolError(id, "content is required")
	}

	kindStr, _ := args["as"].(string)
	if kindStr == "" {
		return s.toolError(id, "as is required for direct write (decision or learning)")
	}

	scope, _ := args["scope"].(string)
	if scope == "" {
		scope = "palace"
	}
	scopePath, _ := args["scopePath"].(string)
	contextStr, _ := args["context"].(string)
	rationale, _ := args["rationale"].(string)
	actorID, _ := args["actorId"].(string)

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	var recordID string
	var err error
	var targetKind string

	switch kindStr {
	case "decision":
		targetKind = "decision"
		dec := memory.Decision{
			Content:   content,
			Rationale: rationale,
			Context:   contextStr,
			Status:    memory.DecisionStatusActive,
			Outcome:   memory.DecisionOutcomeUnknown,
			Scope:     scope,
			ScopePath: scopePath,
			Source:    "human",
			Authority: string(memory.AuthorityApproved),
		}
		recordID, err = mem.AddDecision(dec)

	case "learning":
		targetKind = "learning"
		confidence := 0.7 // Default confidence for human-created learnings
		if c, ok := args["confidence"].(float64); ok && c > 0 && c <= 1.0 {
			confidence = c
		}
		learning := memory.Learning{
			Content:    content,
			Scope:      scope,
			ScopePath:  scopePath,
			Source:     "human",
			Confidence: confidence,
			Authority:  string(memory.AuthorityApproved),
		}
		recordID, err = mem.AddLearning(learning)

	default:
		return s.toolError(id, fmt.Sprintf("invalid record type %q; must be 'decision' or 'learning'", kindStr))
	}

	if err != nil {
		return s.toolError(id, fmt.Sprintf("store direct failed: %v", err))
	}

	// Create audit log entry
	details := map[string]string{
		"scope":        scope,
		"scope_path":   scopePath,
		"content_hash": hashContent(content),
	}
	detailsJSON, _ := json.Marshal(details)

	_, auditErr := mem.AddAuditLog(memory.AuditLogEntry{
		Action:     memory.AuditActionDirectWrite,
		ActorType:  memory.AuditActorHuman,
		ActorID:    actorID,
		TargetID:   recordID,
		TargetKind: targetKind,
		Details:    string(detailsJSON),
	})
	if auditErr != nil {
		// Log the error but don't fail the operation
		// The record was created successfully
		fmt.Printf("WARNING: failed to create audit log: %v\n", auditErr)
	}

	var output strings.Builder
	output.WriteString("# Direct Write Successful\n\n")
	fmt.Fprintf(&output, "**ID:** `%s`\n", recordID)
	fmt.Fprintf(&output, "**Type:** %s\n", kindStr)
	fmt.Fprintf(&output, "**Authority:** approved (direct)\n")
	fmt.Fprintf(&output, "**Scope:** %s", scope)
	if scopePath != "" {
		fmt.Fprintf(&output, " (%s)", scopePath)
	}
	output.WriteString("\n")
	fmt.Fprintf(&output, "**Content:** %s\n\n", content)
	output.WriteString("---\n")
	output.WriteString("This record was created directly (bypassing proposals).\n")
	output.WriteString("An audit entry has been recorded.\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolApprove approves a pending proposal, creating the corresponding record.
// This tool is only available in human mode.
func (s *MCPServer) toolApprove(id any, args map[string]interface{}) jsonRPCResponse {
	proposalID, _ := args["proposalId"].(string)
	if proposalID == "" {
		return s.toolError(id, "proposalId is required")
	}

	reviewedBy, _ := args["by"].(string)
	reviewNote, _ := args["note"].(string)

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	// Get the proposal first to show details in response
	proposal, err := mem.GetProposal(proposalID)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get proposal failed: %v", err))
	}

	// Approve the proposal
	promotedID, err := mem.ApproveProposal(proposalID, reviewedBy, reviewNote)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("approve failed: %v", err))
	}

	// Create audit log entry
	details := map[string]string{
		"proposal_id": proposalID,
		"promoted_to": promotedID,
		"proposed_as": proposal.ProposedAs,
	}
	if reviewNote != "" {
		details["note"] = reviewNote
	}
	detailsJSON, _ := json.Marshal(details)

	_, auditErr := mem.AddAuditLog(memory.AuditLogEntry{
		Action:     memory.AuditActionApprove,
		ActorType:  memory.AuditActorHuman,
		ActorID:    reviewedBy,
		TargetID:   proposalID,
		TargetKind: "proposal",
		Details:    string(detailsJSON),
	})
	if auditErr != nil {
		fmt.Printf("WARNING: failed to create audit log: %v\n", auditErr)
	}

	var output strings.Builder
	output.WriteString("# Proposal Approved\n\n")
	fmt.Fprintf(&output, "**Proposal:** `%s`\n", proposalID)
	fmt.Fprintf(&output, "**Type:** %s\n", proposal.ProposedAs)
	fmt.Fprintf(&output, "**Promoted To:** `%s`\n", promotedID)
	if reviewedBy != "" {
		fmt.Fprintf(&output, "**Approved By:** %s\n", reviewedBy)
	}
	if reviewNote != "" {
		fmt.Fprintf(&output, "**Note:** %s\n", reviewNote)
	}
	output.WriteString("\n")
	fmt.Fprintf(&output, "**Content:** %s\n\n", proposal.Content)
	output.WriteString("---\n")
	fmt.Fprintf(&output, "The %s has been created with `authority = approved`.\n", proposal.ProposedAs)

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolReject rejects a pending proposal with a reason.
// This tool is only available in human mode.
func (s *MCPServer) toolReject(id any, args map[string]interface{}) jsonRPCResponse {
	proposalID, _ := args["proposalId"].(string)
	if proposalID == "" {
		return s.toolError(id, "proposalId is required")
	}

	reviewedBy, _ := args["by"].(string)
	reviewNote, _ := args["note"].(string)

	mem := s.butler.Memory()
	if mem == nil {
		return s.toolError(id, "memory not initialized")
	}

	// Get the proposal first to show details in response
	proposal, err := mem.GetProposal(proposalID)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("get proposal failed: %v", err))
	}

	// Reject the proposal
	err = mem.RejectProposal(proposalID, reviewedBy, reviewNote)
	if err != nil {
		return s.toolError(id, fmt.Sprintf("reject failed: %v", err))
	}

	// Create audit log entry
	details := map[string]string{
		"proposal_id": proposalID,
		"proposed_as": proposal.ProposedAs,
	}
	if reviewNote != "" {
		details["note"] = reviewNote
	}
	detailsJSON, _ := json.Marshal(details)

	_, auditErr := mem.AddAuditLog(memory.AuditLogEntry{
		Action:     memory.AuditActionReject,
		ActorType:  memory.AuditActorHuman,
		ActorID:    reviewedBy,
		TargetID:   proposalID,
		TargetKind: "proposal",
		Details:    string(detailsJSON),
	})
	if auditErr != nil {
		fmt.Printf("WARNING: failed to create audit log: %v\n", auditErr)
	}

	var output strings.Builder
	output.WriteString("# Proposal Rejected\n\n")
	fmt.Fprintf(&output, "**Proposal:** `%s`\n", proposalID)
	fmt.Fprintf(&output, "**Type:** %s\n", proposal.ProposedAs)
	if reviewedBy != "" {
		fmt.Fprintf(&output, "**Rejected By:** %s\n", reviewedBy)
	}
	if reviewNote != "" {
		fmt.Fprintf(&output, "**Reason:** %s\n", reviewNote)
	}
	output.WriteString("\n")
	fmt.Fprintf(&output, "**Content:** %s\n\n", proposal.Content)
	output.WriteString("---\n")
	output.WriteString("The proposal has been rejected and will not be promoted.\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// hashContent computes SHA-256 hash of content for audit trails.
func hashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}
