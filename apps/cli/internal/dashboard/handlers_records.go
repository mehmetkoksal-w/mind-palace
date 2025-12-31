package dashboard

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// handleIdeas returns ideas.
// GET /api/ideas?status=active&scope=palace&scopePath=...&limit=20
func (s *Server) handleIdeas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "memory not available")
		return
	}

	status := r.URL.Query().Get("status")
	scope := r.URL.Query().Get("scope")
	scopePath := r.URL.Query().Get("scopePath")

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			limit = n
		}
	}

	ideas, err := mem.GetIdeas(status, scope, scopePath, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if ideas == nil {
		ideas = []memory.Idea{}
	}

	writeJSON(w, map[string]any{
		"ideas": ideas,
		"count": len(ideas),
	})
}

// handleDecisions returns decisions.
// GET /api/decisions?status=active&scope=palace&scopePath=...&limit=20
func (s *Server) handleDecisions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "memory not available")
		return
	}

	status := r.URL.Query().Get("status")
	scope := r.URL.Query().Get("scope")
	scopePath := r.URL.Query().Get("scopePath")

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			limit = n
		}
	}

	decisions, err := mem.GetDecisions(status, "", scope, scopePath, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if decisions == nil {
		decisions = []memory.Decision{}
	}

	writeJSON(w, map[string]any{
		"decisions": decisions,
		"count":     len(decisions),
	})
}

// ConversationTimelineItem is a conversation with additional timeline metadata.
type ConversationTimelineItem struct {
	ID           string `json:"id"`
	SessionID    string `json:"sessionId,omitempty"`
	AgentType    string `json:"agentType"`
	Summary      string `json:"summary"`
	MessageCount int    `json:"messageCount"`
	Duration     string `json:"duration,omitempty"` // Human readable duration
	CreatedAt    string `json:"createdAt"`
}

// handleConversations returns conversations.
// GET /api/conversations?sessionId=...&agentType=...&limit=20&offset=0
func (s *Server) handleConversations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "memory not available")
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	agentType := r.URL.Query().Get("agentType")
	query := r.URL.Query().Get("q")
	timeline := r.URL.Query().Get("timeline") // If set, returns timeline format

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			limit = n
		}
	}

	var conversations []memory.Conversation
	var err error

	if query != "" {
		conversations, err = mem.SearchConversations(query, limit)
	} else {
		conversations, err = mem.GetConversations(sessionID, agentType, limit)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if conversations == nil {
		conversations = []memory.Conversation{}
	}

	// Return timeline format if requested
	if timeline == "true" || timeline == "1" {
		total, _ := mem.CountConversations()
		timelineItems := make([]ConversationTimelineItem, len(conversations))
		for i, c := range conversations {
			duration := ""
			if len(c.Messages) > 1 {
				first := c.Messages[0].Timestamp
				last := c.Messages[len(c.Messages)-1].Timestamp
				d := last.Sub(first)
				if d.Hours() >= 1 {
					duration = strconv.Itoa(int(d.Hours())) + "h"
				} else if d.Minutes() >= 1 {
					duration = strconv.Itoa(int(d.Minutes())) + "m"
				} else {
					duration = strconv.Itoa(int(d.Seconds())) + "s"
				}
			}
			timelineItems[i] = ConversationTimelineItem{
				ID:           c.ID,
				SessionID:    c.SessionID,
				AgentType:    c.AgentType,
				Summary:      c.Summary,
				MessageCount: len(c.Messages),
				Duration:     duration,
				CreatedAt:    c.CreatedAt.Format("2006-01-02T15:04:05Z"),
			}
		}
		writeJSON(w, map[string]any{
			"conversations": timelineItems,
			"total":         total,
			"hasMore":       len(conversations) == limit,
		})
		return
	}

	writeJSON(w, map[string]any{
		"conversations": conversations,
		"count":         len(conversations),
	})
}

// handleConversationDetail returns a single conversation with full messages and extracted records.
// GET /api/conversations/{id}
// GET /api/conversations/{id}/timeline - returns timeline events
func (s *Server) handleConversationDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "memory not available")
		return
	}

	// Extract ID from path: /api/conversations/{id} or /api/conversations/{id}/timeline
	path := r.URL.Path[len("/api/conversations/"):]
	parts := strings.Split(path, "/")
	id := parts[0]
	if id == "" {
		writeError(w, http.StatusBadRequest, "conversation ID required")
		return
	}

	conversation, err := mem.GetConversation(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if conversation == nil {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}

	// Check if timeline view requested
	if len(parts) > 1 && parts[1] == "timeline" {
		s.handleConversationTimeline(w, conversation, mem)
		return
	}

	// Build enhanced response with extracted records
	result := map[string]any{
		"id":        conversation.ID,
		"agentType": conversation.AgentType,
		"summary":   conversation.Summary,
		"messages":  conversation.Messages,
		"sessionId": conversation.SessionID,
		"createdAt": conversation.CreatedAt,
	}

	// Fetch full extracted records
	if len(conversation.Extracted) > 0 {
		extracted := []map[string]any{}
		for _, recordID := range conversation.Extracted {
			if record := s.fetchRecord(mem, recordID); record != nil {
				extracted = append(extracted, record)
			}
		}
		result["extracted"] = extracted
	}

	// Calculate duration
	if len(conversation.Messages) > 1 {
		first := conversation.Messages[0].Timestamp
		last := conversation.Messages[len(conversation.Messages)-1].Timestamp
		result["duration"] = last.Sub(first).String()
	}

	writeJSON(w, result)
}

// handleConversationTimeline returns a timeline view of the conversation.
func (s *Server) handleConversationTimeline(w http.ResponseWriter, conv *memory.Conversation, mem *memory.Memory) {
	type TimelineEvent struct {
		Timestamp string `json:"timestamp"`
		Type      string `json:"type"`      // "message", "extraction", "summary"
		Role      string `json:"role,omitempty"`
		Content   string `json:"content"`
		RecordID  string `json:"recordId,omitempty"`
		RecordKind string `json:"recordKind,omitempty"`
	}

	events := []TimelineEvent{}

	// Add message events
	for _, msg := range conv.Messages {
		events = append(events, TimelineEvent{
			Timestamp: msg.Timestamp.Format(time.RFC3339),
			Type:      "message",
			Role:      msg.Role,
			Content:   truncateContent(msg.Content, 500),
		})
	}

	// Add extraction events (at conversation end time or last message time)
	extractionTime := conv.CreatedAt
	if len(conv.Messages) > 0 {
		extractionTime = conv.Messages[len(conv.Messages)-1].Timestamp
	}

	for _, recordID := range conv.Extracted {
		if record := s.fetchRecord(mem, recordID); record != nil {
			kind, _ := record["kind"].(string)
			content, _ := record["content"].(string)
			events = append(events, TimelineEvent{
				Timestamp:  extractionTime.Format(time.RFC3339),
				Type:       "extraction",
				Content:    truncateContent(content, 200),
				RecordID:   recordID,
				RecordKind: kind,
			})
		}
	}

	// Sort events by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp < events[j].Timestamp
	})

	writeJSON(w, map[string]any{
		"conversationId": conv.ID,
		"summary":        conv.Summary,
		"events":         events,
		"totalMessages":  len(conv.Messages),
		"totalExtracted": len(conv.Extracted),
	})
}

// fetchRecord retrieves a record by ID and returns it as a map.
func (s *Server) fetchRecord(mem *memory.Memory, id string) map[string]any {
	if len(id) < 2 {
		return nil
	}

	prefix := id[:2]
	switch prefix {
	case "i_":
		if idea, err := mem.GetIdea(id); err == nil && idea != nil {
			return map[string]any{
				"id":      idea.ID,
				"kind":    "idea",
				"content": idea.Content,
				"status":  idea.Status,
				"scope":   idea.Scope,
			}
		}
	case "d_":
		if decision, err := mem.GetDecision(id); err == nil && decision != nil {
			return map[string]any{
				"id":      decision.ID,
				"kind":    "decision",
				"content": decision.Content,
				"status":  decision.Status,
				"scope":   decision.Scope,
			}
		}
	case "l_":
		if learning, err := mem.GetLearning(id); err == nil && learning != nil {
			return map[string]any{
				"id":         learning.ID,
				"kind":       "learning",
				"content":    learning.Content,
				"confidence": learning.Confidence,
				"scope":      learning.Scope,
			}
		}
	}
	return nil
}

// truncateContent truncates content for display.
func truncateContent(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// handleDecisionDetail returns a single decision or its chain.
// GET /api/decisions/{id} - returns single decision
// GET /api/decisions/{id}/chain - returns decision evolution chain
func (s *Server) handleDecisionDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "memory not available")
		return
	}

	// Extract ID from path: /api/decisions/{id} or /api/decisions/{id}/chain
	path := r.URL.Path[len("/api/decisions/"):]
	parts := strings.Split(path, "/")
	id := parts[0]
	if id == "" {
		writeError(w, http.StatusBadRequest, "decision ID required")
		return
	}

	// Check if chain view requested
	if len(parts) > 1 && parts[1] == "chain" {
		chain, err := mem.GetDecisionChain(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if chain == nil {
			writeError(w, http.StatusNotFound, "decision not found")
			return
		}
		writeJSON(w, chain)
		return
	}

	// Return single decision
	decision, err := mem.GetDecision(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if decision == nil {
		writeError(w, http.StatusNotFound, "decision not found")
		return
	}

	writeJSON(w, decision)
}

// handleDecisionTimeline returns decisions as a timeline.
// GET /api/decisions/timeline?scope=palace&scopePath=...&limit=50
func (s *Server) handleDecisionTimeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "memory not available")
		return
	}

	scope := r.URL.Query().Get("scope")
	scopePath := r.URL.Query().Get("scopePath")

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			limit = n
		}
	}

	decisions, err := mem.GetDecisionTimeline(scope, scopePath, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if decisions == nil {
		decisions = []memory.Decision{}
	}

	// Build timeline with outcome color coding
	type TimelineDecision struct {
		memory.Decision
		OutcomeColor string `json:"outcomeColor"` // green, red, yellow, gray
	}

	timeline := make([]TimelineDecision, len(decisions))
	for i, d := range decisions {
		color := "gray" // unknown
		switch d.Outcome {
		case "success":
			color = "green"
		case "failed":
			color = "red"
		case "mixed":
			color = "yellow"
		}
		timeline[i] = TimelineDecision{
			Decision:     d,
			OutcomeColor: color,
		}
	}

	writeJSON(w, map[string]any{
		"decisions": timeline,
		"count":     len(timeline),
	})
}

// handleLinks returns links for a record or all links.
// GET /api/links?recordId=...&relation=...&limit=20
func (s *Server) handleLinks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	s.mu.RLock()
	mem := s.memory
	s.mu.RUnlock()

	if mem == nil {
		writeError(w, http.StatusServiceUnavailable, "memory not available")
		return
	}

	recordID := r.URL.Query().Get("recordId")
	relation := r.URL.Query().Get("relation")

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil {
			limit = n
		}
	}

	var links []memory.Link
	var err error

	if recordID != "" {
		links, err = mem.GetAllLinksFor(recordID)
	} else if relation != "" {
		links, err = mem.GetLinksByRelation(relation, limit)
	} else {
		// Return stale links by default if no filter
		links, err = mem.GetStaleLinks()
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if links == nil {
		links = []memory.Link{}
	}

	writeJSON(w, map[string]any{
		"links": links,
		"count": len(links),
	})
}
