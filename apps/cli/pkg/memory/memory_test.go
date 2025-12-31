package memory

import (
	"testing"
	"time"
)

func TestMemoryLifecycle(t *testing.T) {
	root := t.TempDir()
	mem, err := Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	sess, err := mem.StartSession("agent", "id-1", "ship it")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}

	if err := mem.LogActivity(sess.ID, Activity{Kind: ActivityFileEdit, Target: "main.go", Outcome: OutcomeSuccess}); err != nil {
		t.Fatalf("LogActivity() error = %v", err)
	}

	activities, err := mem.GetActivities(sess.ID, "", 10)
	if err != nil {
		t.Fatalf("GetActivities() error = %v", err)
	}
	if len(activities) == 0 {
		t.Fatalf("expected activities")
	}

	learningID, err := mem.AddLearning(Learning{
		Scope:      ScopePalace,
		Content:    "Remember to test",
		Confidence: 0.6,
		Source:     "test",
		UseCount:   1,
	})
	if err != nil {
		t.Fatalf("AddLearning() error = %v", err)
	}

	if err := mem.ReinforceLearning(learningID); err != nil {
		t.Fatalf("ReinforceLearning() error = %v", err)
	}

	learnings, err := mem.GetLearnings(ScopePalace, "", 10)
	if err != nil || len(learnings) == 0 {
		t.Fatalf("GetLearnings() = %v, err = %v", learnings, err)
	}

	found, err := mem.SearchLearnings("test", 10)
	if err != nil || len(found) == 0 {
		t.Fatalf("SearchLearnings() = %v, err = %v", found, err)
	}

	if _, err := mem.AddLearning(Learning{Scope: ScopeFile, ScopePath: "src/main.go", Content: "file note"}); err != nil {
		t.Fatalf("AddLearning(file) error = %v", err)
	}
	if _, err := mem.AddLearning(Learning{Scope: ScopeRoom, ScopePath: "src", Content: "room note"}); err != nil {
		t.Fatalf("AddLearning(room) error = %v", err)
	}

	relevant, err := mem.GetRelevantLearnings("src/main.go", "", 10)
	if err != nil || len(relevant) == 0 {
		t.Fatalf("GetRelevantLearnings() = %v, err = %v", relevant, err)
	}

	if err := mem.RecordFileEdit("src/main.go", "agent"); err != nil {
		t.Fatalf("RecordFileEdit() error = %v", err)
	}
	if err := mem.RecordFileFailure("src/main.go"); err != nil {
		t.Fatalf("RecordFileFailure() error = %v", err)
	}

	intel, err := mem.GetFileIntel("src/main.go")
	if err != nil || intel == nil || intel.EditCount == 0 {
		t.Fatalf("GetFileIntel() = %+v, err = %v", intel, err)
	}

	hotspots, err := mem.GetFileHotspots(5)
	if err != nil || len(hotspots) == 0 {
		t.Fatalf("GetFileHotspots() = %v, err = %v", hotspots, err)
	}

	if err := mem.EndSession(sess.ID, SessionCompleted, "done"); err != nil {
		t.Fatalf("EndSession() error = %v", err)
	}
}

func TestCheckConflict(t *testing.T) {
	root := t.TempDir()
	mem, err := Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	sessA, err := mem.StartSession("agent-a", "id-a", "goal a")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	sessB, err := mem.StartSession("agent-b", "id-b", "goal b")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}

	if err := mem.LogActivity(sessA.ID, Activity{
		Kind:      ActivityFileEdit,
		Target:    "conflict.go",
		Outcome:   OutcomeSuccess,
		Details:   "{}",
		Timestamp: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("LogActivity() error = %v", err)
	}

	conflict, err := mem.CheckConflict(sessB.ID, "conflict.go")
	if err != nil {
		t.Fatalf("CheckConflict() error = %v", err)
	}
	if conflict == nil {
		t.Fatalf("expected conflict for active file")
	}
}

func TestGetSession(t *testing.T) {
	root := t.TempDir()
	mem, err := Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	sess, err := mem.StartSession("agent", "id-1", "test goal")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}

	// Get the session by ID
	retrieved, err := mem.GetSession(sess.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}

	if retrieved == nil {
		t.Fatal("GetSession() returned nil session")
	}

	if retrieved.ID != sess.ID {
		t.Errorf("GetSession() ID = %q, want %q", retrieved.ID, sess.ID)
	}

	if retrieved.AgentType != "agent" {
		t.Errorf("GetSession() AgentType = %q, want %q", retrieved.AgentType, "agent")
	}

	if retrieved.Goal != "test goal" {
		t.Errorf("GetSession() Goal = %q, want %q", retrieved.Goal, "test goal")
	}
}

func TestGetSessionNotFound(t *testing.T) {
	root := t.TempDir()
	mem, err := Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	// Get a non-existent session
	_, err = mem.GetSession("nonexistent-session-id")
	if err == nil {
		t.Error("GetSession() should return error for non-existent session")
	}
}

func TestListSessions(t *testing.T) {
	root := t.TempDir()
	mem, err := Open(root)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = mem.Close() })

	// Create multiple sessions
	sess1, err := mem.StartSession("agent1", "id-1", "goal 1")
	if err != nil {
		t.Fatalf("StartSession(1) error = %v", err)
	}
	sess2, err := mem.StartSession("agent2", "id-2", "goal 2")
	if err != nil {
		t.Fatalf("StartSession(2) error = %v", err)
	}

	// End one session
	if err := mem.EndSession(sess1.ID, SessionCompleted, "done"); err != nil {
		t.Fatalf("EndSession() error = %v", err)
	}

	// List all sessions
	sessions, err := mem.ListSessions(false, 10)
	if err != nil {
		t.Fatalf("ListSessions(all) error = %v", err)
	}

	if len(sessions) < 2 {
		t.Errorf("ListSessions(all) = %d sessions, want >= 2", len(sessions))
	}

	// List only active sessions
	activeSessions, err := mem.ListSessions(true, 10)
	if err != nil {
		t.Fatalf("ListSessions(active) error = %v", err)
	}

	// Should have sess2 as active
	foundActive := false
	for _, s := range activeSessions {
		if s.ID == sess2.ID && s.State == SessionActive {
			foundActive = true
			break
		}
	}
	if !foundActive {
		t.Error("ListSessions(active) should include active session")
	}
}
