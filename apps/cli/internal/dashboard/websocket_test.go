package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewWSHub(t *testing.T) {
	hub := NewWSHub()
	if hub == nil {
		t.Fatal("NewWSHub returned nil")
	}
	if hub.clients == nil {
		t.Error("clients map not initialized")
	}
	if hub.broadcast == nil {
		t.Error("broadcast channel not initialized")
	}
	if hub.register == nil {
		t.Error("register channel not initialized")
	}
	if hub.unregister == nil {
		t.Error("unregister channel not initialized")
	}
}

func TestWSHubClientCount(t *testing.T) {
	hub := NewWSHub()
	if hub.ClientCount() != 0 {
		t.Errorf("Expected 0 clients, got %d", hub.ClientCount())
	}
}

func TestEventTypes(t *testing.T) {
	// Verify event type constants are defined
	events := []string{
		EventSessionStarted,
		EventSessionEnded,
		EventLearningAdded,
		EventDecisionAdded,
		EventIdeaAdded,
		EventScanStarted,
		EventScanCompleted,
		EventConflictDetected,
		EventWorkspaceChanged,
		EventActivityLogged,
		EventHeartbeat,
	}

	for _, e := range events {
		if e == "" {
			t.Error("Event type constant is empty")
		}
	}
}

func TestEventMarshal(t *testing.T) {
	event := Event{
		Type: EventLearningAdded,
		Payload: LearningEventPayload{
			ID:         "lrn_123",
			Content:    "Test learning",
			Confidence: 0.8,
			Scope:      "palace",
		},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if decoded.Type != EventLearningAdded {
		t.Errorf("Expected type %s, got %s", EventLearningAdded, decoded.Type)
	}
}

func TestSessionEventPayload(t *testing.T) {
	payload := SessionEventPayload{
		SessionID: "ses_123",
		AgentType: "claude-code",
		Goal:      "Test goal",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded SessionEventPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.SessionID != "ses_123" {
		t.Errorf("Expected sessionId ses_123, got %s", decoded.SessionID)
	}
	if decoded.AgentType != "claude-code" {
		t.Errorf("Expected agentType claude-code, got %s", decoded.AgentType)
	}
}

func TestWebSocketConnection(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigins := []string{"http://localhost"}
		ServeWS(hub, allowedOrigins, w, r)
	}))
	defer server.Close()

	// Connect WebSocket client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Wait for registration
	time.Sleep(100 * time.Millisecond)

	// Check client count
	if hub.ClientCount() != 1 {
		t.Errorf("Expected 1 client, got %d", hub.ClientCount())
	}
}

func TestWebSocketBroadcast(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigins := []string{"http://localhost"}
		ServeWS(hub, allowedOrigins, w, r)
	}))
	defer server.Close()

	// Connect WebSocket client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Wait for registration
	time.Sleep(100 * time.Millisecond)

	// Broadcast an event
	hub.Broadcast(EventLearningAdded, LearningEventPayload{
		ID:         "lrn_test",
		Content:    "Test broadcast",
		Confidence: 0.9,
		Scope:      "palace",
	})

	// Read the message
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	var event Event
	if err := json.Unmarshal(message, &event); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if event.Type != EventLearningAdded {
		t.Errorf("Expected event type %s, got %s", EventLearningAdded, event.Type)
	}
}

func TestWebSocketDisconnect(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigins := []string{"http://localhost"}
		ServeWS(hub, allowedOrigins, w, r)
	}))
	defer server.Close()

	// Connect WebSocket client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for registration
	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Errorf("Expected 1 client, got %d", hub.ClientCount())
	}

	// Close connection
	conn.Close()

	// Wait for unregistration
	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("Expected 0 clients after disconnect, got %d", hub.ClientCount())
	}
}

func TestMultipleClients(t *testing.T) {
	hub := NewWSHub()
	go hub.Run()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigins := []string{"http://localhost"}
		ServeWS(hub, allowedOrigins, w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Connect multiple clients
	var conns []*websocket.Conn
	for i := 0; i < 3; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect client %d: %v", i, err)
		}
		conns = append(conns, conn)
	}
	defer func() {
		for _, conn := range conns {
			conn.Close()
		}
	}()

	// Wait for registrations
	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 3 {
		t.Errorf("Expected 3 clients, got %d", hub.ClientCount())
	}

	// Broadcast event
	hub.Broadcast(EventHeartbeat, nil)

	// Verify all clients receive the message
	for i, conn := range conns {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, message, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("Client %d failed to read: %v", i, err)
			continue
		}

		var event Event
		if err := json.Unmarshal(message, &event); err != nil {
			t.Errorf("Client %d failed to unmarshal: %v", i, err)
			continue
		}

		if event.Type != EventHeartbeat {
			t.Errorf("Client %d: expected %s, got %s", i, EventHeartbeat, event.Type)
		}
	}
}

func TestAllPayloadTypes(t *testing.T) {
	tests := []struct {
		name    string
		payload interface{}
	}{
		{"SessionEventPayload", SessionEventPayload{SessionID: "ses_1", AgentType: "test"}},
		{"LearningEventPayload", LearningEventPayload{ID: "lrn_1", Content: "test", Confidence: 0.5, Scope: "palace"}},
		{"DecisionEventPayload", DecisionEventPayload{ID: "d_1", Content: "test", Status: "active", Scope: "palace"}},
		{"IdeaEventPayload", IdeaEventPayload{ID: "i_1", Content: "test", Status: "active", Scope: "palace"}},
		{"ScanEventPayload", ScanEventPayload{ScanID: "scan_1", Files: 10, Symbols: 100}},
		{"ConflictEventPayload", ConflictEventPayload{Path: "test.go", AgentType: "claude", SessionID: "ses_1"}},
		{"WorkspaceEventPayload", WorkspaceEventPayload{Name: "test", Path: "/test"}},
		{"ActivityEventPayload", ActivityEventPayload{SessionID: "ses_1", Kind: "file_edit", Target: "test.go"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.payload)
			if err != nil {
				t.Errorf("Failed to marshal %s: %v", tt.name, err)
			}
			if len(data) == 0 {
				t.Errorf("%s marshaled to empty data", tt.name)
			}
		})
	}
}
