package memory

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestAddDecision(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "decision-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mem, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	defer mem.Close()

	// Test adding a basic decision
	dec := Decision{
		Content:   "Use JWT for authentication",
		Rationale: "Industry standard, stateless, works well with microservices",
	}
	id, err := mem.AddDecision(dec)
	if err != nil {
		t.Fatalf("Failed to add decision: %v", err)
	}
	if !strings.HasPrefix(id, "d_") {
		t.Errorf("Expected ID to start with 'd_', got %s", id)
	}

	// Verify defaults
	retrieved, err := mem.GetDecision(id)
	if err != nil {
		t.Fatalf("Failed to get decision: %v", err)
	}
	if retrieved.Status != DecisionStatusActive {
		t.Errorf("Expected status 'active', got '%s'", retrieved.Status)
	}
	if retrieved.Outcome != DecisionOutcomeUnknown {
		t.Errorf("Expected outcome 'unknown', got '%s'", retrieved.Outcome)
	}
	if retrieved.Scope != "palace" {
		t.Errorf("Expected scope 'palace', got '%s'", retrieved.Scope)
	}
	if retrieved.Source != "cli" {
		t.Errorf("Expected source 'cli', got '%s'", retrieved.Source)
	}
}

func TestAddDecisionWithScope(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "decision-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Test decision with room scope
	dec := Decision{
		Content:   "Use bcrypt for password hashing",
		Rationale: "Secure, proven, adjustable work factor",
		Scope:     "room",
		ScopePath: "auth",
	}
	id, err := mem.AddDecision(dec)
	if err != nil {
		t.Fatalf("Failed to add decision: %v", err)
	}

	retrieved, _ := mem.GetDecision(id)
	if retrieved.Scope != "room" {
		t.Errorf("Expected scope 'room', got '%s'", retrieved.Scope)
	}
	if retrieved.ScopePath != "auth" {
		t.Errorf("Expected scope_path 'auth', got '%s'", retrieved.ScopePath)
	}
}

func TestGetDecisions(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "decision-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add multiple decisions
	decisions := []Decision{
		{Content: "Decision 1", Status: DecisionStatusActive, Outcome: DecisionOutcomeUnknown},
		{Content: "Decision 2", Status: DecisionStatusActive, Outcome: DecisionOutcomeSuccessful},
		{Content: "Decision 3", Status: DecisionStatusSuperseded, Outcome: DecisionOutcomeUnknown},
		{Content: "Decision 4", Status: DecisionStatusActive, Outcome: DecisionOutcomeFailed, Scope: "room", ScopePath: "api"},
	}
	for _, dec := range decisions {
		_, err := mem.AddDecision(dec)
		if err != nil {
			t.Fatalf("Failed to add decision: %v", err)
		}
	}

	// Get all decisions
	all, err := mem.GetDecisions("", "", "", "", 10)
	if err != nil {
		t.Fatalf("Failed to get decisions: %v", err)
	}
	if len(all) != 4 {
		t.Errorf("Expected 4 decisions, got %d", len(all))
	}

	// Filter by status
	active, _ := mem.GetDecisions(DecisionStatusActive, "", "", "", 10)
	if len(active) != 3 {
		t.Errorf("Expected 3 active decisions, got %d", len(active))
	}

	// Filter by outcome
	unknown, _ := mem.GetDecisions("", DecisionOutcomeUnknown, "", "", 10)
	if len(unknown) != 2 {
		t.Errorf("Expected 2 unknown outcome decisions, got %d", len(unknown))
	}

	// Filter by scope
	roomDecisions, _ := mem.GetDecisions("", "", "room", "", 10)
	if len(roomDecisions) != 1 {
		t.Errorf("Expected 1 room decision, got %d", len(roomDecisions))
	}

	// Test limit
	limited, _ := mem.GetDecisions("", "", "", "", 2)
	if len(limited) != 2 {
		t.Errorf("Expected 2 decisions with limit, got %d", len(limited))
	}
}

func TestSearchDecisions(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "decision-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add decisions with different content
	decisions := []Decision{
		{Content: "Use PostgreSQL for the database", Rationale: "ACID compliance needed"},
		{Content: "Use Redis for caching", Rationale: "Fast in-memory store"},
		{Content: "Implement rate limiting", Context: "API security"},
	}
	for _, dec := range decisions {
		mem.AddDecision(dec)
	}

	// Search by content
	results, err := mem.SearchDecisions("database", 10)
	if err != nil {
		t.Fatalf("Failed to search decisions: %v", err)
	}
	if len(results) < 1 {
		t.Errorf("Expected at least 1 result for 'database', got %d", len(results))
	}

	// Search by rationale
	rationaleResults, _ := mem.SearchDecisions("ACID", 10)
	if len(rationaleResults) < 1 {
		t.Errorf("Expected at least 1 result for 'ACID', got %d", len(rationaleResults))
	}
}

func TestRecordDecisionOutcome(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "decision-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddDecision(Decision{Content: "Test decision"})

	// Record successful outcome
	err := mem.RecordDecisionOutcome(id, DecisionOutcomeSuccessful, "Reduced latency by 40%")
	if err != nil {
		t.Fatalf("Failed to record outcome: %v", err)
	}

	dec, _ := mem.GetDecision(id)
	if dec.Outcome != DecisionOutcomeSuccessful {
		t.Errorf("Expected outcome 'successful', got '%s'", dec.Outcome)
	}
	if dec.OutcomeNote != "Reduced latency by 40%" {
		t.Errorf("Expected outcome note 'Reduced latency by 40%%', got '%s'", dec.OutcomeNote)
	}
	if dec.OutcomeAt.IsZero() {
		t.Error("Expected outcome_at to be set")
	}

	// Test invalid outcome
	err = mem.RecordDecisionOutcome(id, "invalid", "note")
	if err == nil {
		t.Error("Expected error for invalid outcome")
	}

	// Test non-existent decision
	err = mem.RecordDecisionOutcome("nonexistent", DecisionOutcomeSuccessful, "note")
	if err == nil {
		t.Error("Expected error for non-existent decision")
	}
}

func TestUpdateDecisionStatus(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "decision-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddDecision(Decision{Content: "Test decision"})

	// Update status to superseded
	err := mem.UpdateDecisionStatus(id, DecisionStatusSuperseded)
	if err != nil {
		t.Fatalf("Failed to update decision status: %v", err)
	}

	dec, _ := mem.GetDecision(id)
	if dec.Status != DecisionStatusSuperseded {
		t.Errorf("Expected status 'superseded', got '%s'", dec.Status)
	}

	// Update non-existent decision
	err = mem.UpdateDecisionStatus("nonexistent", DecisionStatusActive)
	if err == nil {
		t.Error("Expected error for non-existent decision")
	}
}

func TestUpdateDecision(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "decision-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddDecision(Decision{
		Content:   "Original content",
		Rationale: "Original rationale",
		Context:   "Original context",
	})

	// Update content, rationale, and context
	err := mem.UpdateDecision(id, "Updated content", "Updated rationale", "Updated context")
	if err != nil {
		t.Fatalf("Failed to update decision: %v", err)
	}

	dec, _ := mem.GetDecision(id)
	if dec.Content != "Updated content" {
		t.Errorf("Expected content 'Updated content', got '%s'", dec.Content)
	}
	if dec.Rationale != "Updated rationale" {
		t.Errorf("Expected rationale 'Updated rationale', got '%s'", dec.Rationale)
	}
	if dec.Context != "Updated context" {
		t.Errorf("Expected context 'Updated context', got '%s'", dec.Context)
	}

	// Update non-existent decision
	err = mem.UpdateDecision("nonexistent", "content", "rationale", "context")
	if err == nil {
		t.Error("Expected error for non-existent decision")
	}
}

func TestDeleteDecision(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "decision-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddDecision(Decision{Content: "To be deleted"})

	// Delete the decision
	err := mem.DeleteDecision(id)
	if err != nil {
		t.Fatalf("Failed to delete decision: %v", err)
	}

	// Verify it's gone
	_, err = mem.GetDecision(id)
	if err == nil {
		t.Error("Expected error getting deleted decision")
	}
}

func TestCountDecisions(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "decision-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add decisions with different statuses and outcomes
	mem.AddDecision(Decision{Content: "1", Status: DecisionStatusActive, Outcome: DecisionOutcomeUnknown})
	mem.AddDecision(Decision{Content: "2", Status: DecisionStatusActive, Outcome: DecisionOutcomeSuccessful})
	mem.AddDecision(Decision{Content: "3", Status: DecisionStatusSuperseded, Outcome: DecisionOutcomeUnknown})

	// Count all
	total, err := mem.CountDecisions("", "")
	if err != nil {
		t.Fatalf("Failed to count decisions: %v", err)
	}
	if total != 3 {
		t.Errorf("Expected 3 total decisions, got %d", total)
	}

	// Count by status
	active, _ := mem.CountDecisions(DecisionStatusActive, "")
	if active != 2 {
		t.Errorf("Expected 2 active decisions, got %d", active)
	}

	// Count by outcome
	unknown, _ := mem.CountDecisions("", DecisionOutcomeUnknown)
	if unknown != 2 {
		t.Errorf("Expected 2 unknown outcome decisions, got %d", unknown)
	}

	// Count by both
	activeUnknown, _ := mem.CountDecisions(DecisionStatusActive, DecisionOutcomeUnknown)
	if activeUnknown != 1 {
		t.Errorf("Expected 1 active+unknown decision, got %d", activeUnknown)
	}
}

func TestGetDecisionsAwaitingReview(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "decision-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add a decision and manually backdate it
	id, _ := mem.AddDecision(Decision{Content: "Old decision"})
	oldDate := time.Now().AddDate(0, 0, -45).Format(time.RFC3339)
	mem.db.Exec("UPDATE decisions SET created_at = ? WHERE id = ?", oldDate, id)

	// Add a recent decision
	mem.AddDecision(Decision{Content: "Recent decision"})

	// Get decisions awaiting review (older than 30 days)
	awaiting, err := mem.GetDecisionsAwaitingReview(30, 10)
	if err != nil {
		t.Fatalf("Failed to get decisions awaiting review: %v", err)
	}
	if len(awaiting) != 1 {
		t.Errorf("Expected 1 decision awaiting review, got %d", len(awaiting))
	}
	if len(awaiting) > 0 && awaiting[0].Content != "Old decision" {
		t.Errorf("Expected 'Old decision', got '%s'", awaiting[0].Content)
	}
}

func TestGetDecisionsSince(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "decision-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Set cutoff time in the past (use UTC to match how decisions are stored)
	cutoffTime := time.Now().UTC().Add(-1 * time.Hour)

	// Add decisions (all will be after the cutoff since they use time.Now().UTC())
	mem.AddDecision(Decision{Content: "Decision 1"})
	mem.AddDecision(Decision{Content: "Decision 2"})

	// Get decisions since cutoff (should find all)
	since, err := mem.GetDecisionsSince(cutoffTime, 10)
	if err != nil {
		t.Fatalf("Failed to get decisions since: %v", err)
	}
	if len(since) != 2 {
		t.Errorf("Expected 2 decisions since cutoff, got %d", len(since))
	}

	// Test with future cutoff (should find none)
	futureCutoff := time.Now().UTC().Add(1 * time.Hour)
	noneSince, err := mem.GetDecisionsSince(futureCutoff, 10)
	if err != nil {
		t.Fatalf("Failed to get decisions since future: %v", err)
	}
	if len(noneSince) != 0 {
		t.Errorf("Expected 0 decisions since future cutoff, got %d", len(noneSince))
	}
}

func TestDecisionWithSession(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "decision-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Create a session first
	session, _ := mem.StartSession("claude-code", "test-agent", "Testing decisions")

	// Add decision linked to session
	dec := Decision{
		Content:   "Session-linked decision",
		SessionID: session.ID,
	}
	id, _ := mem.AddDecision(dec)

	retrieved, _ := mem.GetDecision(id)
	if retrieved.SessionID != session.ID {
		t.Errorf("Expected session ID '%s', got '%s'", session.ID, retrieved.SessionID)
	}
}

func TestDecisionFTSIntegration(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "decision-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add decision
	id, _ := mem.AddDecision(Decision{
		Content:   "Use MongoDB for the document store",
		Rationale: "Schema flexibility needed",
	})

	// Update decision (should update FTS via trigger)
	err := mem.UpdateDecision(id, "Use PostgreSQL JSONB for document storage", "Better querying with SQL", "Changed approach")
	if err != nil {
		t.Fatalf("Failed to update decision: %v", err)
	}

	// Search should find updated content
	results, _ := mem.SearchDecisions("PostgreSQL", 10)
	found := false
	for _, r := range results {
		if r.ID == id {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find updated decision via FTS search")
	}

	// Old content should not be found
	oldResults, _ := mem.SearchDecisions("MongoDB", 10)
	for _, r := range oldResults {
		if r.ID == id {
			t.Error("Found old content in FTS after update - trigger may not be working")
		}
	}
}

func TestDecisionOutcomeValidation(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "decision-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddDecision(Decision{Content: "Test"})

	// Test all valid outcomes
	validOutcomes := []string{DecisionOutcomeSuccessful, DecisionOutcomeFailed, DecisionOutcomeMixed}
	for _, outcome := range validOutcomes {
		err := mem.RecordDecisionOutcome(id, outcome, "test")
		if err != nil {
			t.Errorf("Expected no error for valid outcome '%s', got %v", outcome, err)
		}
	}

	// Test invalid outcomes
	invalidOutcomes := []string{"unknown", "partial", "pending", ""}
	for _, outcome := range invalidOutcomes {
		err := mem.RecordDecisionOutcome(id, outcome, "test")
		if err == nil {
			t.Errorf("Expected error for invalid outcome '%s'", outcome)
		}
	}
}

func TestSearchDecisionsLikeFallback(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "decision-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add decisions
	mem.AddDecision(Decision{Content: "Use PostgreSQL for database", Rationale: "ACID compliance needed"})
	mem.AddDecision(Decision{Content: "Implement caching with Redis", Context: "Performance improvement"})

	// Test the searchDecisionsLike function directly
	results, err := mem.searchDecisionsLike("PostgreSQL", 10)
	if err != nil {
		t.Fatalf("searchDecisionsLike failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	// Search in rationale
	rationaleResults, _ := mem.searchDecisionsLike("ACID", 10)
	if len(rationaleResults) != 1 {
		t.Errorf("Expected 1 result for rationale search, got %d", len(rationaleResults))
	}

	// Search in context
	contextResults, _ := mem.searchDecisionsLike("Performance", 10)
	if len(contextResults) != 1 {
		t.Errorf("Expected 1 result for context search, got %d", len(contextResults))
	}

	// No results
	noResults, _ := mem.searchDecisionsLike("nonexistent", 10)
	if len(noResults) != 0 {
		t.Errorf("Expected 0 results, got %d", len(noResults))
	}
}

func TestGetSchemaVersion(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "decision-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// After opening, schema version should be 3 (v0, v1, v2, and v3 for postmortems)
	version, err := mem.GetSchemaVersion()
	if err != nil {
		t.Fatalf("GetSchemaVersion failed: %v", err)
	}
	if version != 3 {
		t.Errorf("Expected schema version 3, got %d", version)
	}
}
