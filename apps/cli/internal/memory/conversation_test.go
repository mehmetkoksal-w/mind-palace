package memory

import (
	"os"
	"testing"
	"time"
)

func TestAddConversation(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "conv-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	conv := Conversation{
		AgentType: "claude-code",
		Summary:   "Discussion about authentication implementation",
		Messages: []Message{
			{Role: "user", Content: "How should I implement JWT auth?", Timestamp: time.Now()},
			{Role: "assistant", Content: "Use the jwt-go library...", Timestamp: time.Now()},
		},
	}

	id, err := mem.AddConversation(conv)
	if err != nil {
		t.Fatalf("Failed to add conversation: %v", err)
	}

	if id == "" {
		t.Error("Expected non-empty ID")
	}
	if id[:2] != "c_" {
		t.Errorf("Expected ID to start with 'c_', got %s", id)
	}
}

func TestGetConversation(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "conv-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	conv := Conversation{
		AgentType: "claude-code",
		Summary:   "Test conversation",
		Messages: []Message{
			{Role: "user", Content: "Hello", Timestamp: time.Now()},
			{Role: "assistant", Content: "Hi there!", Timestamp: time.Now()},
		},
		Extracted: []string{"i_abc123", "d_xyz789"},
	}

	id, _ := mem.AddConversation(conv)

	retrieved, err := mem.GetConversation(id)
	if err != nil {
		t.Fatalf("Failed to get conversation: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected conversation, got nil")
	}

	if retrieved.AgentType != "claude-code" {
		t.Errorf("Expected agent type 'claude-code', got %s", retrieved.AgentType)
	}
	if retrieved.Summary != "Test conversation" {
		t.Errorf("Expected summary 'Test conversation', got %s", retrieved.Summary)
	}
	if len(retrieved.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(retrieved.Messages))
	}
	if len(retrieved.Extracted) != 2 {
		t.Errorf("Expected 2 extracted IDs, got %d", len(retrieved.Extracted))
	}
}

func TestGetConversationNotFound(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "conv-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	retrieved, err := mem.GetConversation("nonexistent")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if retrieved != nil {
		t.Error("Expected nil for non-existent conversation")
	}
}

func TestGetConversations(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "conv-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add multiple conversations
	mem.AddConversation(Conversation{
		AgentType: "claude-code",
		Summary:   "First conversation",
		SessionID: "session1",
	})
	mem.AddConversation(Conversation{
		AgentType: "cursor",
		Summary:   "Second conversation",
		SessionID: "session2",
	})
	mem.AddConversation(Conversation{
		AgentType: "claude-code",
		Summary:   "Third conversation",
	})

	// Get all conversations
	all, err := mem.GetConversations("", "", 10)
	if err != nil {
		t.Fatalf("Failed to get conversations: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("Expected 3 conversations, got %d", len(all))
	}

	// Filter by agent type
	claudeConvs, _ := mem.GetConversations("", "claude-code", 10)
	if len(claudeConvs) != 2 {
		t.Errorf("Expected 2 claude-code conversations, got %d", len(claudeConvs))
	}

	// Filter by session ID
	session1Convs, _ := mem.GetConversations("session1", "", 10)
	if len(session1Convs) != 1 {
		t.Errorf("Expected 1 conversation for session1, got %d", len(session1Convs))
	}
}

func TestSearchConversations(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "conv-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add conversations with different summaries
	mem.AddConversation(Conversation{
		AgentType: "claude-code",
		Summary:   "Discussion about authentication and JWT tokens",
	})
	mem.AddConversation(Conversation{
		AgentType: "cursor",
		Summary:   "Implementing database migrations",
	})
	mem.AddConversation(Conversation{
		AgentType: "claude-code",
		Summary:   "Fixing authentication bugs in login flow",
	})

	// Search for "authentication"
	results, err := mem.SearchConversations("authentication", 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'authentication', got %d", len(results))
	}

	// Search for "database"
	dbResults, _ := mem.SearchConversations("database", 10)
	if len(dbResults) != 1 {
		t.Errorf("Expected 1 result for 'database', got %d", len(dbResults))
	}
}

func TestDeleteConversation(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "conv-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddConversation(Conversation{
		Summary: "To be deleted",
	})

	// Delete
	err := mem.DeleteConversation(id)
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Verify deleted
	retrieved, _ := mem.GetConversation(id)
	if retrieved != nil {
		t.Error("Expected conversation to be deleted")
	}

	// Delete non-existent
	err = mem.DeleteConversation("nonexistent")
	if err == nil {
		t.Error("Expected error deleting non-existent conversation")
	}
}

func TestGetConversationForSession(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "conv-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	sessionID := "sess_12345"
	mem.AddConversation(Conversation{
		Summary:   "Session conversation",
		SessionID: sessionID,
	})

	conv, err := mem.GetConversationForSession(sessionID)
	if err != nil {
		t.Fatalf("Failed to get conversation for session: %v", err)
	}

	if conv == nil {
		t.Fatal("Expected conversation, got nil")
	}
	if conv.SessionID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, conv.SessionID)
	}
}

func TestUpdateConversationExtracted(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "conv-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	id, _ := mem.AddConversation(Conversation{
		Summary:   "Test",
		Extracted: []string{},
	})

	// Update extracted IDs
	extracted := []string{"i_new1", "d_new2", "l_new3"}
	err := mem.UpdateConversationExtracted(id, extracted)
	if err != nil {
		t.Fatalf("Failed to update extracted: %v", err)
	}

	// Verify
	conv, _ := mem.GetConversation(id)
	if len(conv.Extracted) != 3 {
		t.Errorf("Expected 3 extracted IDs, got %d", len(conv.Extracted))
	}
}

func TestEndSessionWithConversation(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "conv-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Start a session first
	session, _ := mem.StartSession("test_agent", "agent1", "fix a bug")
	sessionID := session.ID

	// End with conversation
	messages := []Message{
		{Role: "user", Content: "Fix the bug", Timestamp: time.Now()},
		{Role: "assistant", Content: "Fixed!", Timestamp: time.Now()},
	}
	err := mem.EndSessionWithConversation(sessionID, "Bug fix session", messages, "claude-code")
	if err != nil {
		t.Fatalf("Failed to end session with conversation: %v", err)
	}

	// Verify session is ended
	endedSession, _ := mem.GetSession(sessionID)
	if endedSession.State != "completed" {
		t.Errorf("Expected session state 'completed', got %s", endedSession.State)
	}

	// Verify conversation was stored
	conv, _ := mem.GetConversationForSession(sessionID)
	if conv == nil {
		t.Fatal("Expected conversation to be stored")
	}
	if conv.Summary != "Bug fix session" {
		t.Errorf("Expected summary 'Bug fix session', got %s", conv.Summary)
	}
	if len(conv.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(conv.Messages))
	}
}

func TestCountConversations(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "conv-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	count, _ := mem.CountConversations()
	if count != 0 {
		t.Errorf("Expected 0 conversations, got %d", count)
	}

	mem.AddConversation(Conversation{Summary: "One"})
	mem.AddConversation(Conversation{Summary: "Two"})

	count, _ = mem.CountConversations()
	if count != 2 {
		t.Errorf("Expected 2 conversations, got %d", count)
	}
}

func TestGetRecentConversations(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "conv-test-*")
	defer os.RemoveAll(tmpDir)
	mem, _ := Open(tmpDir)
	defer mem.Close()

	// Add multiple conversations
	for i := 0; i < 5; i++ {
		mem.AddConversation(Conversation{Summary: "Conversation"})
	}

	// Get recent with limit
	recent, _ := mem.GetRecentConversations(3)
	if len(recent) != 3 {
		t.Errorf("Expected 3 recent conversations, got %d", len(recent))
	}
}
