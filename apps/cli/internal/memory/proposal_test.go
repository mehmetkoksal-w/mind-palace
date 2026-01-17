package memory

import (
	"os"
	"strings"
	"testing"
)

func TestAddProposal(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "proposal-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mem, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	defer mem.Close()

	// Test adding a basic proposal
	prop := Proposal{
		ProposedAs: ProposedAsDecision,
		Content:    "Use JWT for authentication",
		Rationale:  "Industry standard, stateless",
		Source:     "agent",
	}
	id, err := mem.AddProposal(prop)
	if err != nil {
		t.Fatalf("Failed to add proposal: %v", err)
	}
	if !strings.HasPrefix(id, "prop_") {
		t.Errorf("Expected ID to start with 'prop_', got %s", id)
	}

	// Verify defaults
	retrieved, err := mem.GetProposal(id)
	if err != nil {
		t.Fatalf("Failed to get proposal: %v", err)
	}
	if retrieved.Status != ProposalStatusPending {
		t.Errorf("Expected status 'pending', got '%s'", retrieved.Status)
	}
	if retrieved.Scope != "palace" {
		t.Errorf("Expected scope 'palace', got '%s'", retrieved.Scope)
	}
	if retrieved.DedupeKey == "" {
		t.Error("Expected dedupe_key to be generated")
	}
}

func TestAddProposalValidation(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "proposal-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Test missing proposedAs
	_, err := mem.AddProposal(Proposal{Content: "test"})
	if err == nil {
		t.Error("Expected error for missing proposedAs")
	}

	// Test missing content
	_, err = mem.AddProposal(Proposal{ProposedAs: ProposedAsDecision})
	if err == nil {
		t.Error("Expected error for missing content")
	}
}

func TestGetProposals(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "proposal-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add multiple proposals
	proposals := []Proposal{
		{ProposedAs: ProposedAsDecision, Content: "Decision 1", Source: "agent"},
		{ProposedAs: ProposedAsLearning, Content: "Learning 1", Source: "agent"},
		{ProposedAs: ProposedAsDecision, Content: "Decision 2", Source: "auto-extract"},
	}
	for _, p := range proposals {
		_, err := mem.AddProposal(p)
		if err != nil {
			t.Fatalf("Failed to add proposal: %v", err)
		}
	}

	// Get all pending proposals
	all, err := mem.GetProposals(ProposalStatusPending, "", 10)
	if err != nil {
		t.Fatalf("Failed to get proposals: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("Expected 3 proposals, got %d", len(all))
	}

	// Filter by proposedAs
	decisions, _ := mem.GetProposals("", ProposedAsDecision, 10)
	if len(decisions) != 2 {
		t.Errorf("Expected 2 decision proposals, got %d", len(decisions))
	}

	learnings, _ := mem.GetProposals("", ProposedAsLearning, 10)
	if len(learnings) != 1 {
		t.Errorf("Expected 1 learning proposal, got %d", len(learnings))
	}

	// Test limit
	limited, _ := mem.GetProposals("", "", 2)
	if len(limited) != 2 {
		t.Errorf("Expected 2 proposals with limit, got %d", len(limited))
	}
}

func TestSearchProposals(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "proposal-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add proposals
	proposals := []Proposal{
		{ProposedAs: ProposedAsDecision, Content: "Use PostgreSQL for database", Rationale: "ACID compliance", Source: "agent"},
		{ProposedAs: ProposedAsLearning, Content: "Redis caching improves performance", Source: "agent"},
	}
	for _, p := range proposals {
		mem.AddProposal(p)
	}

	// Search by content
	results, err := mem.SearchProposals("database", 10)
	if err != nil {
		t.Fatalf("Failed to search proposals: %v", err)
	}
	if len(results) < 1 {
		t.Errorf("Expected at least 1 result for 'database', got %d", len(results))
	}

	// Search by rationale
	rationaleResults, _ := mem.SearchProposals("ACID", 10)
	if len(rationaleResults) < 1 {
		t.Errorf("Expected at least 1 result for 'ACID', got %d", len(rationaleResults))
	}
}

func TestCheckDuplicateProposal(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "proposal-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add a proposal
	prop := Proposal{
		ProposedAs: ProposedAsDecision,
		Content:    "Use JWT for authentication",
		Source:     "agent",
		Scope:      "palace",
	}
	id, _ := mem.AddProposal(prop)

	// Generate the same dedupe key
	dedupeKey := GenerateDedupeKey(prop.ProposedAs, prop.Content, prop.Scope, prop.ScopePath)

	// Check for duplicate
	existing, err := mem.CheckDuplicateProposal(dedupeKey)
	if err != nil {
		t.Fatalf("CheckDuplicateProposal failed: %v", err)
	}
	if existing == nil {
		t.Error("Expected to find duplicate proposal")
	}
	if existing != nil && existing.ID != id {
		t.Errorf("Expected duplicate ID '%s', got '%s'", id, existing.ID)
	}

	// Check for non-existent
	noExisting, _ := mem.CheckDuplicateProposal("nonexistent")
	if noExisting != nil {
		t.Error("Expected no duplicate for nonexistent key")
	}
}

func TestApproveProposal(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "proposal-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add a decision proposal
	prop := Proposal{
		ProposedAs: ProposedAsDecision,
		Content:    "Use JWT for authentication",
		Rationale:  "Industry standard",
		Source:     "agent",
		Scope:      "room",
		ScopePath:  "auth",
	}
	propID, _ := mem.AddProposal(prop)

	// Approve it
	promotedID, err := mem.ApproveProposal(propID, "test-reviewer", "Looks good")
	if err != nil {
		t.Fatalf("ApproveProposal failed: %v", err)
	}
	if !strings.HasPrefix(promotedID, "d_") {
		t.Errorf("Expected promoted ID to start with 'd_', got %s", promotedID)
	}

	// Verify proposal status updated
	updatedProp, _ := mem.GetProposal(propID)
	if updatedProp.Status != ProposalStatusApproved {
		t.Errorf("Expected proposal status 'approved', got '%s'", updatedProp.Status)
	}
	if updatedProp.PromotedToID != promotedID {
		t.Errorf("Expected promoted_to_id '%s', got '%s'", promotedID, updatedProp.PromotedToID)
	}
	if updatedProp.ReviewedBy != "test-reviewer" {
		t.Errorf("Expected reviewed_by 'test-reviewer', got '%s'", updatedProp.ReviewedBy)
	}
	if updatedProp.ReviewNote != "Looks good" {
		t.Errorf("Expected review_note 'Looks good', got '%s'", updatedProp.ReviewNote)
	}

	// Verify decision was created correctly
	decision, err := mem.GetDecision(promotedID)
	if err != nil {
		t.Fatalf("Failed to get promoted decision: %v", err)
	}
	if decision.Content != prop.Content {
		t.Errorf("Expected decision content '%s', got '%s'", prop.Content, decision.Content)
	}
	if decision.Authority != string(AuthorityApproved) {
		t.Errorf("Expected decision authority 'approved', got '%s'", decision.Authority)
	}
	if decision.PromotedFromProposalID != propID {
		t.Errorf("Expected promoted_from_proposal_id '%s', got '%s'", propID, decision.PromotedFromProposalID)
	}
	if decision.Scope != prop.Scope {
		t.Errorf("Expected decision scope '%s', got '%s'", prop.Scope, decision.Scope)
	}
	if decision.ScopePath != prop.ScopePath {
		t.Errorf("Expected decision scope_path '%s', got '%s'", prop.ScopePath, decision.ScopePath)
	}
}

func TestApproveLearningProposal(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "proposal-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add a learning proposal
	prop := Proposal{
		ProposedAs:               ProposedAsLearning,
		Content:                  "Always run tests before committing",
		Source:                   "agent",
		ClassificationConfidence: 0.85,
	}
	propID, _ := mem.AddProposal(prop)

	// Approve it
	promotedID, err := mem.ApproveProposal(propID, "reviewer", "Good practice")
	if err != nil {
		t.Fatalf("ApproveProposal failed: %v", err)
	}
	if !strings.HasPrefix(promotedID, "lrn_") {
		t.Errorf("Expected promoted ID to start with 'lrn_', got %s", promotedID)
	}

	// Verify learning was created correctly
	learning, err := mem.GetLearning(promotedID)
	if err != nil {
		t.Fatalf("Failed to get promoted learning: %v", err)
	}
	if learning.Content != prop.Content {
		t.Errorf("Expected learning content '%s', got '%s'", prop.Content, learning.Content)
	}
	if learning.Authority != string(AuthorityApproved) {
		t.Errorf("Expected learning authority 'approved', got '%s'", learning.Authority)
	}
	if learning.PromotedFromProposalID != propID {
		t.Errorf("Expected promoted_from_proposal_id '%s', got '%s'", propID, learning.PromotedFromProposalID)
	}
	if learning.Confidence != prop.ClassificationConfidence {
		t.Errorf("Expected learning confidence %f, got %f", prop.ClassificationConfidence, learning.Confidence)
	}
}

func TestApproveProposalAlreadyProcessed(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "proposal-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add and approve a proposal
	propID, _ := mem.AddProposal(Proposal{
		ProposedAs: ProposedAsDecision,
		Content:    "Test",
		Source:     "agent",
	})
	mem.ApproveProposal(propID, "reviewer", "")

	// Try to approve again
	_, err := mem.ApproveProposal(propID, "reviewer", "")
	if err == nil {
		t.Error("Expected error when approving already-approved proposal")
	}
	if !strings.Contains(err.Error(), "already approved") {
		t.Errorf("Expected 'already approved' in error, got: %v", err)
	}
}

func TestRejectProposal(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "proposal-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add a proposal
	propID, _ := mem.AddProposal(Proposal{
		ProposedAs: ProposedAsDecision,
		Content:    "Questionable decision",
		Source:     "agent",
	})

	// Reject it
	err := mem.RejectProposal(propID, "reviewer", "Not accurate")
	if err != nil {
		t.Fatalf("RejectProposal failed: %v", err)
	}

	// Verify status
	prop, _ := mem.GetProposal(propID)
	if prop.Status != ProposalStatusRejected {
		t.Errorf("Expected status 'rejected', got '%s'", prop.Status)
	}
	if prop.ReviewNote != "Not accurate" {
		t.Errorf("Expected review_note 'Not accurate', got '%s'", prop.ReviewNote)
	}
}

func TestRejectProposalAlreadyProcessed(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "proposal-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add and reject a proposal
	propID, _ := mem.AddProposal(Proposal{
		ProposedAs: ProposedAsDecision,
		Content:    "Test",
		Source:     "agent",
	})
	mem.RejectProposal(propID, "reviewer", "No")

	// Try to reject again
	err := mem.RejectProposal(propID, "reviewer", "")
	if err == nil {
		t.Error("Expected error when rejecting already-rejected proposal")
	}
}

func TestExpireProposal(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "proposal-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add a proposal
	propID, _ := mem.AddProposal(Proposal{
		ProposedAs: ProposedAsDecision,
		Content:    "Old proposal",
		Source:     "agent",
	})

	// Expire it
	err := mem.ExpireProposal(propID)
	if err != nil {
		t.Fatalf("ExpireProposal failed: %v", err)
	}

	// Verify status
	prop, _ := mem.GetProposal(propID)
	if prop.Status != ProposalStatusExpired {
		t.Errorf("Expected status 'expired', got '%s'", prop.Status)
	}
}

func TestCountProposals(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "proposal-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add proposals
	mem.AddProposal(Proposal{ProposedAs: ProposedAsDecision, Content: "1", Source: "agent"})
	mem.AddProposal(Proposal{ProposedAs: ProposedAsDecision, Content: "2", Source: "agent"})

	// Count all
	total, err := mem.CountProposals("")
	if err != nil {
		t.Fatalf("CountProposals failed: %v", err)
	}
	if total != 2 {
		t.Errorf("Expected 2 proposals, got %d", total)
	}

	// Count pending
	pending, _ := mem.CountProposals(ProposalStatusPending)
	if pending != 2 {
		t.Errorf("Expected 2 pending proposals, got %d", pending)
	}

	// Approve one
	props, _ := mem.GetProposals("", "", 1)
	mem.ApproveProposal(props[0].ID, "reviewer", "")

	// Count again
	pending, _ = mem.CountProposals(ProposalStatusPending)
	if pending != 1 {
		t.Errorf("Expected 1 pending proposal after approval, got %d", pending)
	}

	approved, _ := mem.CountProposals(ProposalStatusApproved)
	if approved != 1 {
		t.Errorf("Expected 1 approved proposal, got %d", approved)
	}
}

func TestGenerateDedupeKey(t *testing.T) {
	// Same inputs should generate same key
	key1 := GenerateDedupeKey(ProposedAsDecision, "content", "palace", "")
	key2 := GenerateDedupeKey(ProposedAsDecision, "content", "palace", "")
	if key1 != key2 {
		t.Error("Same inputs should generate same dedupe key")
	}

	// Different inputs should generate different keys
	key3 := GenerateDedupeKey(ProposedAsLearning, "content", "palace", "")
	if key1 == key3 {
		t.Error("Different proposedAs should generate different dedupe key")
	}

	key4 := GenerateDedupeKey(ProposedAsDecision, "different content", "palace", "")
	if key1 == key4 {
		t.Error("Different content should generate different dedupe key")
	}

	key5 := GenerateDedupeKey(ProposedAsDecision, "content", "room", "auth")
	if key1 == key5 {
		t.Error("Different scope should generate different dedupe key")
	}
}

func TestDeleteProposal(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "proposal-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add and delete a proposal
	propID, _ := mem.AddProposal(Proposal{
		ProposedAs: ProposedAsDecision,
		Content:    "To be deleted",
		Source:     "agent",
	})

	err := mem.DeleteProposal(propID)
	if err != nil {
		t.Fatalf("DeleteProposal failed: %v", err)
	}

	// Verify it's gone
	_, err = mem.GetProposal(propID)
	if err == nil {
		t.Error("Expected error getting deleted proposal")
	}
}

func TestProposalFTSSearch(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "proposal-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add proposals
	mem.AddProposal(Proposal{
		ProposedAs: ProposedAsDecision,
		Content:    "Use PostgreSQL for the main database",
		Rationale:  "ACID compliance needed for transactions",
		Source:     "agent",
	})

	// Search by content
	results, _ := mem.SearchProposals("PostgreSQL", 10)
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'PostgreSQL', got %d", len(results))
	}

	// Search by rationale
	results, _ = mem.SearchProposals("ACID", 10)
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'ACID', got %d", len(results))
	}
}
