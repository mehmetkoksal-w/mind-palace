package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMemoryBasics(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test Open
	mem, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	defer mem.Close()

	// Verify database file was created
	dbPath := filepath.Join(tmpDir, ".palace", "memory.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestSessionLifecycle(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mem, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	defer mem.Close()

	// Start session
	session, err := mem.StartSession("claude-code", "test-instance-1", "Test the session system")
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}
	if session.AgentType != "claude-code" {
		t.Errorf("Expected agent type 'claude-code', got '%s'", session.AgentType)
	}
	if session.Goal != "Test the session system" {
		t.Errorf("Expected goal 'Test the session system', got '%s'", session.Goal)
	}
	if session.State != "active" {
		t.Errorf("Expected state 'active', got '%s'", session.State)
	}

	// List sessions
	sessions, err := mem.ListSessions(true, 10)
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("Expected 1 active session, got %d", len(sessions))
	}

	// Get session by ID
	retrieved, err := mem.GetSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Retrieved session is nil")
	}
	if retrieved.ID != session.ID {
		t.Errorf("Expected session ID '%s', got '%s'", session.ID, retrieved.ID)
	}

	// End session
	err = mem.EndSession(session.ID, "completed", "Test completed successfully")
	if err != nil {
		t.Fatalf("Failed to end session: %v", err)
	}

	// Verify session ended
	retrieved, err = mem.GetSession(session.ID)
	if err != nil {
		t.Fatalf("Failed to get ended session: %v", err)
	}
	if retrieved.State != "completed" {
		t.Errorf("Expected state 'completed', got '%s'", retrieved.State)
	}

	// Active sessions should be 0
	sessions, err = mem.ListSessions(true, 10)
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("Expected 0 active sessions, got %d", len(sessions))
	}
}

func TestActivityLogging(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mem, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	defer mem.Close()

	// Start session
	session, err := mem.StartSession("cursor", "cursor-1", "Edit files")
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	// Log activities
	activities := []Activity{
		{Kind: "file_read", Target: "main.go", Outcome: "success"},
		{Kind: "file_edit", Target: "main.go", Outcome: "success"},
		{Kind: "command", Target: "go build", Outcome: "success"},
		{Kind: "file_edit", Target: "utils.go", Outcome: "failure"},
	}

	for _, act := range activities {
		err := mem.LogActivity(session.ID, act)
		if err != nil {
			t.Fatalf("Failed to log activity: %v", err)
		}
	}

	// Get activities for session
	retrieved, err := mem.GetActivities(session.ID, "", 10)
	if err != nil {
		t.Fatalf("Failed to get activities: %v", err)
	}
	if len(retrieved) != 4 {
		t.Errorf("Expected 4 activities, got %d", len(retrieved))
	}

	// Get activities for specific file
	fileActs, err := mem.GetActivities("", "main.go", 10)
	if err != nil {
		t.Fatalf("Failed to get file activities: %v", err)
	}
	if len(fileActs) != 2 {
		t.Errorf("Expected 2 activities for main.go, got %d", len(fileActs))
	}
}

func TestLearnings(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mem, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	defer mem.Close()

	// Add learnings
	learnings := []Learning{
		{Scope: "palace", Content: "Always run tests before committing", Confidence: 0.8, Source: "user"},
		{Scope: "file", ScopePath: "auth/login.go", Content: "This file requires special handling", Confidence: 0.7, Source: "agent"},
		{Scope: "room", ScopePath: "authentication", Content: "Use bcrypt for password hashing", Confidence: 0.9, Source: "user"},
	}

	for _, l := range learnings {
		_, err := mem.AddLearning(l)
		if err != nil {
			t.Fatalf("Failed to add learning: %v", err)
		}
	}

	// Get all learnings
	all, err := mem.GetLearnings("", "", 10)
	if err != nil {
		t.Fatalf("Failed to get learnings: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("Expected 3 learnings, got %d", len(all))
	}

	// Get learnings by scope
	palaceLearnings, err := mem.GetLearnings("palace", "", 10)
	if err != nil {
		t.Fatalf("Failed to get palace learnings: %v", err)
	}
	if len(palaceLearnings) != 1 {
		t.Errorf("Expected 1 palace learning, got %d", len(palaceLearnings))
	}

	// Search learnings
	searchResults, err := mem.SearchLearnings("password", 10)
	if err != nil {
		t.Fatalf("Failed to search learnings: %v", err)
	}
	if len(searchResults) != 1 {
		t.Errorf("Expected 1 search result for 'password', got %d", len(searchResults))
	}
}

func TestFileIntel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mem, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	defer mem.Close()

	// Record file edits directly (file intel is tracked separately from activities)
	for i := 0; i < 5; i++ {
		err := mem.RecordFileEdit("src/main.go", "test-agent")
		if err != nil {
			t.Fatalf("Failed to record file edit: %v", err)
		}
	}

	// Record some failures
	for i := 0; i < 2; i++ {
		err := mem.RecordFileFailure("src/main.go")
		if err != nil {
			t.Fatalf("Failed to record file failure: %v", err)
		}
	}

	// Get file intel
	intel, err := mem.GetFileIntel("src/main.go")
	if err != nil {
		t.Fatalf("Failed to get file intel: %v", err)
	}
	if intel == nil {
		t.Fatal("File intel is nil")
	}
	if intel.EditCount != 5 {
		t.Errorf("Expected 5 edits, got %d", intel.EditCount)
	}
	if intel.FailureCount != 2 {
		t.Errorf("Expected 2 failures, got %d", intel.FailureCount)
	}

	// Get hotspots
	hotspots, err := mem.GetFileHotspots(10)
	if err != nil {
		t.Fatalf("Failed to get hotspots: %v", err)
	}
	if len(hotspots) != 1 {
		t.Errorf("Expected 1 hotspot, got %d", len(hotspots))
	}
}

func TestAgentRegistry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mem, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	defer mem.Close()

	// Start sessions and register agents
	session1, _ := mem.StartSession("claude-code", "claude-1", "Work on feature A")
	mem.RegisterAgent("claude-code", "claude-1", session1.ID)

	session2, _ := mem.StartSession("cursor", "cursor-1", "Work on feature B")
	mem.RegisterAgent("cursor", "cursor-1", session2.ID)

	// Get active agents
	agents, err := mem.GetActiveAgents(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to get active agents: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("Expected 2 active agents, got %d", len(agents))
	}

	// Test unregister
	err = mem.UnregisterAgent("claude-1")
	if err != nil {
		t.Fatalf("Failed to unregister agent: %v", err)
	}

	agents, _ = mem.GetActiveAgents(5 * time.Minute)
	if len(agents) != 1 {
		t.Errorf("Expected 1 active agent after unregister, got %d", len(agents))
	}
}

func TestConflictDetection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mem, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	defer mem.Close()

	// Start two sessions
	session1, _ := mem.StartSession("claude-code", "claude-1", "Edit main.go")
	session2, _ := mem.StartSession("cursor", "cursor-1", "Also edit main.go")

	// Session 1 works on main.go
	mem.LogActivity(session1.ID, Activity{
		Kind:    "file_edit",
		Target:  "main.go",
		Outcome: "success",
	})

	// Session 2 checks for conflicts
	conflict, err := mem.CheckConflict(session2.ID, "main.go")
	if err != nil {
		t.Fatalf("Failed to check conflict: %v", err)
	}
	if conflict == nil {
		t.Error("Expected conflict but got nil")
	} else if conflict.Path != "main.go" {
		t.Errorf("Expected conflict path 'main.go', got '%s'", conflict.Path)
	}
}

func TestRelevantLearnings(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "memory-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mem, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open memory: %v", err)
	}
	defer mem.Close()

	// Add learnings with different scopes
	mem.AddLearning(Learning{Scope: "file", ScopePath: "auth/login.go", Content: "Specific to login.go", Confidence: 0.9})
	mem.AddLearning(Learning{Scope: "room", ScopePath: "auth", Content: "Auth room learning", Confidence: 0.8})
	mem.AddLearning(Learning{Scope: "palace", Content: "Global learning", Confidence: 0.7})

	// Get relevant learnings for auth/login.go
	relevant, err := mem.GetRelevantLearnings("auth/login.go", "auth", 10)
	if err != nil {
		t.Fatalf("Failed to get relevant learnings: %v", err)
	}

	// Should include file-specific, room, and palace learnings
	if len(relevant) != 3 {
		t.Errorf("Expected 3 relevant learnings, got %d", len(relevant))
	}
}

func TestReinforceAndWeakenLearning(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "memory-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddLearning(Learning{Content: "test", Confidence: 0.5})

	// Reinforce
	mem.ReinforceLearning(id)
	l, _ := mem.GetLearning(id)
	if l.Confidence <= 0.5 {
		t.Errorf("Expected confidence > 0.5, got %f", l.Confidence)
	}
	if l.UseCount != 1 {
		t.Errorf("Expected use count 1, got %d", l.UseCount)
	}

	// Weaken
	mem.WeakenLearning(id)
	l2, _ := mem.GetLearning(id)
	if l2.Confidence >= l.Confidence {
		t.Errorf("Expected confidence < %f, got %f", l.Confidence, l2.Confidence)
	}
}

func TestDeleteLearning(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "memory-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddLearning(Learning{Content: "test"})
	mem.DeleteLearning(id)

	l, _ := mem.GetLearning(id)
	if l != nil {
		t.Error("Expected learning to be deleted")
	}
}

func TestAgentHeartbeatAndCurrentFile(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "memory-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// MUST create a session because active_agents references sessions(id)
	sess, err := mem.StartSession("bot", "test-agent", "testing")
	if err != nil {
		t.Fatal(err)
	}

	agentID := "test-agent"
	err = mem.RegisterAgent("bot", agentID, sess.ID)
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	// Set current file
	err = mem.SetCurrentFile(agentID, "main.go")
	if err != nil {
		t.Fatalf("SetCurrentFile failed: %v", err)
	}

	agent, err := mem.GetAgentForFile("main.go")
	if err != nil {
		t.Fatalf("GetAgentForFile failed: %v", err)
	}
	if agent == nil {
		t.Fatalf("Expected agent working on main.go, got nil")
	}
	if agent.AgentID != agentID {
		t.Errorf("Expected agent %s, got %s", agentID, agent.AgentID)
	}

	// Heartbeat
	oldHeartbeat := agent.Heartbeat
	newTime := time.Now().UTC().Add(1 * time.Minute)
	_, err = mem.db.ExecContext(context.Background(), "UPDATE active_agents SET last_heartbeat = ? WHERE agent_id = ?", newTime.Format(time.RFC3339), agentID)
	if err != nil {
		t.Fatalf("Manual heartbeat update failed: %v", err)
	}

	agents, err := mem.GetActiveAgents(5 * time.Minute)
	if err != nil {
		t.Fatalf("GetActiveAgents failed: %v", err)
	}
	if len(agents) == 0 || !agents[0].Heartbeat.After(oldHeartbeat) {
		t.Errorf("Expected heartbeat to be updated: old=%v, new=%v", oldHeartbeat, agents[0].Heartbeat)
	}
}

func TestCleanupStaleAgents(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "memory-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	sess, _ := mem.StartSession("bot", "stale", "clean")
	err := mem.RegisterAgent("bot", "stale", sess.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Manually backdate the heartbeat in DB - MUST BE UTC for string comparison
	staleTime := time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339)
	mem.db.ExecContext(context.Background(), "UPDATE active_agents SET last_heartbeat = ?", staleTime)

	count, _ := mem.CleanupStaleAgents(5 * time.Minute)
	if count != 1 {
		t.Errorf("Expected 1 agent cleaned up, got %d", count)
	}
}

func TestFragileFilesAndHotspots(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "memory-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	mem.RecordFileEdit("good.go", "bot")
	mem.RecordFileEdit("bad.go", "bot")
	mem.RecordFileFailure("bad.go")

	hotspots, _ := mem.GetFileHotspots(10)
	if len(hotspots) != 2 {
		t.Errorf("Expected 2 hotspots, got %d", len(hotspots))
	}

	fragile, _ := mem.GetFragileFiles(10)
	if len(fragile) != 1 || fragile[0].Path != "bad.go" {
		t.Errorf("Expected 'bad.go' as fragile, got %v", fragile)
	}
}

func TestHighConfidenceLearningsAndDecay(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "memory-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	mem.AddLearning(Learning{Content: "high", Confidence: 0.95, UseCount: 10})
	mem.AddLearning(Learning{Content: "low", Confidence: 0.4, UseCount: 1})

	high, _ := mem.GetHighConfidenceLearnings(0.9, 5)
	if len(high) != 1 || high[0].Content != "high" {
		t.Errorf("Expected 1 high confidence learning, got %d", len(high))
	}

	// Decay
	// Backdate last_used
	oldTime := time.Now().UTC().Add(-10 * 24 * time.Hour).Format(time.RFC3339)
	mem.db.ExecContext(context.Background(), "UPDATE learnings SET last_used = ?", oldTime)

	decayed, _ := mem.DecayUnusedLearnings(5, 0.1)
	if decayed != 2 {
		t.Errorf("Expected 2 learnings decayed, got %d", decayed)
	}

	// Prune
	pruned, _ := mem.PruneLowConfidenceLearnings(0.5)
	if pruned < 1 {
		t.Errorf("Expected at least 1 learning pruned, got %d", pruned)
	}
}

func TestAssociateLearningWithFile(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "memory-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	lID, _ := mem.AddLearning(Learning{Content: "tips"})
	mem.AssociateLearningWithFile("target.go", lID)

	intel, _ := mem.GetFileIntel("target.go")
	if len(intel.Learnings) != 1 || intel.Learnings[0] != lID {
		t.Error("Expected learning associated with file intel")
	}

	fileLearnings, _ := mem.GetFileLearnings("target.go")
	if len(fileLearnings) != 1 || fileLearnings[0].ID != lID {
		t.Error("Expected learning retrieved via GetFileLearnings")
	}
}

func TestCounters(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "memory-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	mem.StartSession("bot", "1", "goal")
	mem.AddLearning(Learning{Content: "test"})
	mem.RecordFileEdit("file.go", "bot")

	if c, _ := mem.CountSessions(false); c != 1 {
		t.Errorf("Expected 1 session, got %d", c)
	}
	if c, _ := mem.CountLearnings(); c != 1 {
		t.Errorf("Expected 1 learning, got %d", c)
	}
	if c, _ := mem.CountFilesTracked(); c != 1 {
		t.Errorf("Expected 1 file tracked, got %d", c)
	}
}
