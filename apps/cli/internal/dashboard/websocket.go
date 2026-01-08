package dashboard

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Event types for WebSocket broadcasts
const (
	EventSessionStarted        = "session_started"
	EventSessionEnded          = "session_ended"
	EventLearningAdded         = "learning_added"
	EventDecisionAdded         = "decision_added"
	EventIdeaAdded             = "idea_added"
	EventScanStarted           = "scan_started"
	EventScanCompleted         = "scan_completed"
	EventConflictDetected      = "conflict_detected"
	EventContradictionDetected = "contradiction_detected"
	EventWorkspaceChanged      = "workspace_changed"
	EventActivityLogged        = "activity_logged"
	EventHeartbeat             = "heartbeat"
)

// Event represents a WebSocket event message
type Event struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// WSClient represents a connected WebSocket client
type WSClient struct {
	hub  *WSHub
	conn *websocket.Conn
	send chan []byte
}

// WSHub manages all WebSocket connections
type WSHub struct {
	clients    map[*WSClient]bool
	broadcast  chan Event
	register   chan *WSClient
	unregister chan *WSClient
	mu         sync.RWMutex
}

// NewWSHub creates a new WebSocket hub
func NewWSHub() *WSHub {
	return &WSHub{
		clients:    make(map[*WSClient]bool),
		broadcast:  make(chan Event, 256),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
	}
}

// Run starts the hub's main event loop
func (h *WSHub) Run() {
	// Heartbeat ticker to keep connections alive
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WS] Client connected (total: %d)", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("[WS] Client disconnected (total: %d)", len(h.clients))

		case event := <-h.broadcast:
			h.broadcastEvent(event)

		case <-heartbeat.C:
			h.broadcastEvent(Event{
				Type:      EventHeartbeat,
				Timestamp: time.Now(),
			})
		}
	}
}

// broadcastEvent sends an event to all connected clients
func (h *WSHub) broadcastEvent(event Event) {
	event.Timestamp = time.Now()
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[WS] Error marshaling event: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- data:
		default:
			// Client's send buffer is full, close connection
			close(client.send)
			delete(h.clients, client)
		}
	}
}

// Broadcast sends an event to all connected clients
func (h *WSHub) Broadcast(eventType string, payload interface{}) {
	h.broadcast <- Event{
		Type:    eventType,
		Payload: payload,
	}
}

// ClientCount returns the number of connected clients
func (h *WSHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// createUpgrader creates a WebSocket upgrader with origin checking.
func createUpgrader(allowedOrigins []string) websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")

			// Allow requests with no Origin header (e.g., non-browser clients, tests)
			// This is safe because non-browser clients are not subject to CORS
			if origin == "" {
				return true
			}

			// Check if origin is in allowed list
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					return true
				}
			}
			return false
		},
	}
}

// writePump handles writing messages to the WebSocket connection
func (c *WSClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current write
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump handles reading messages from the WebSocket connection
func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS] Error: %v", err)
			}
			break
		}
		// Currently we don't process incoming messages from clients
		// This could be extended for bidirectional communication
	}
}

// ServeWS handles WebSocket upgrade requests
func ServeWS(hub *WSHub, allowedOrigins []string, w http.ResponseWriter, r *http.Request) {
	upgrader := createUpgrader(allowedOrigins)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade error: %v", err)
		return
	}

	client := &WSClient{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	hub.register <- client

	// Start read and write pumps in separate goroutines
	go client.writePump()
	go client.readPump()
}

// Event payload types for type safety

// SessionEventPayload is the payload for session events
type SessionEventPayload struct {
	SessionID string `json:"sessionId"`
	AgentType string `json:"agentType,omitempty"`
	AgentID   string `json:"agentId,omitempty"`
	Goal      string `json:"goal,omitempty"`
	Outcome   string `json:"outcome,omitempty"`
	Summary   string `json:"summary,omitempty"`
}

// LearningEventPayload is the payload for learning events
type LearningEventPayload struct {
	ID         string  `json:"id"`
	Content    string  `json:"content"`
	Confidence float64 `json:"confidence"`
	Scope      string  `json:"scope"`
	ScopePath  string  `json:"scopePath,omitempty"`
}

// DecisionEventPayload is the payload for decision events
type DecisionEventPayload struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Status    string `json:"status"`
	Scope     string `json:"scope"`
	ScopePath string `json:"scopePath,omitempty"`
}

// IdeaEventPayload is the payload for idea events
type IdeaEventPayload struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Status    string `json:"status"`
	Scope     string `json:"scope"`
	ScopePath string `json:"scopePath,omitempty"`
}

// ScanEventPayload is the payload for scan events
type ScanEventPayload struct {
	ScanID   string `json:"scanId,omitempty"`
	ScanHash string `json:"scanHash,omitempty"`
	Files    int    `json:"files,omitempty"`
	Symbols  int    `json:"symbols,omitempty"`
	Duration string `json:"duration,omitempty"`
}

// ConflictEventPayload is the payload for conflict events
type ConflictEventPayload struct {
	Path       string `json:"path"`
	AgentType  string `json:"agentType"`
	AgentID    string `json:"agentId,omitempty"`
	SessionID  string `json:"sessionId"`
	ConflictID string `json:"conflictId,omitempty"`
}

// WorkspaceEventPayload is the payload for workspace change events
type WorkspaceEventPayload struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ActivityEventPayload is the payload for activity log events
type ActivityEventPayload struct {
	SessionID string `json:"sessionId"`
	Kind      string `json:"kind"`
	Target    string `json:"target"`
	Outcome   string `json:"outcome,omitempty"`
}

// ContradictionEventPayload is the payload for contradiction detection events
type ContradictionEventPayload struct {
	RecordID       string                `json:"recordId"`
	RecordKind     string                `json:"recordKind"`
	RecordContent  string                `json:"recordContent"`
	Contradictions []ContradictionDetail `json:"contradictions"`
}

// ContradictionDetail contains details about a single contradiction
type ContradictionDetail struct {
	ConflictingID      string  `json:"conflictingId"`
	ConflictingKind    string  `json:"conflictingKind"`
	ConflictingContent string  `json:"conflictingContent"`
	Confidence         float64 `json:"confidence"`
	Type               string  `json:"type"` // "direct", "implicit", "temporal"
	Explanation        string  `json:"explanation"`
	AutoLinked         bool    `json:"autoLinked"`
}
