package butler

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Handoff represents a task handoff between agents.
type Handoff struct {
	ID            string    `json:"id"`
	FromAgent     string    `json:"fromAgent"`     // Agent creating the handoff
	FromSessionID string    `json:"fromSessionId"` // Session creating the handoff
	ToAgentType   string    `json:"toAgentType"`   // Target agent type (or "any")
	Task          string    `json:"task"`          // Task description
	Context       string    `json:"context"`       // Relevant context
	PendingWork   []string  `json:"pendingWork"`   // List of pending items
	PinnedRecords []string  `json:"pinnedRecords"` // Record IDs to include
	Priority      string    `json:"priority"`      // "low", "normal", "high", "urgent"
	Status        string    `json:"status"`        // "pending", "accepted", "completed", "expired"
	AcceptedBy    string    `json:"acceptedBy"`    // Session that accepted
	Summary       string    `json:"summary"`       // Completion summary
	CreatedAt     time.Time `json:"createdAt"`
	ExpiresAt     time.Time `json:"expiresAt"`
}

// In-memory handoff storage (simple implementation - can be migrated to DB later)
var (
	handoffStore = make(map[string]*Handoff)
	handoffMu    sync.RWMutex
)

// toolHandoffCreate creates a handoff request for another agent.
func (s *MCPServer) toolHandoffCreate(id any, args map[string]interface{}) jsonRPCResponse {
	task, _ := args["task"].(string)
	if task == "" {
		return s.toolError(id, "task is required")
	}

	toAgentType, _ := args["to"].(string)
	if toAgentType == "" {
		toAgentType = "any" // Any agent can pick up
	}

	context, _ := args["context"].(string)
	priority, _ := args["priority"].(string)
	if priority == "" {
		priority = "normal"
	}

	// Parse pending work items
	var pendingWork []string
	if pending, ok := args["pending"].([]interface{}); ok {
		for _, p := range pending {
			if item, ok := p.(string); ok && item != "" {
				pendingWork = append(pendingWork, item)
			}
		}
	}

	// Parse pinned record IDs
	var pinnedRecords []string
	if pinned, ok := args["pin"].([]interface{}); ok {
		for _, p := range pinned {
			if pid, ok := p.(string); ok && pid != "" {
				pinnedRecords = append(pinnedRecords, pid)
			}
		}
	}

	// Include current context pins
	pinnedRecords = append(pinnedRecords, s.contextPriorityUp...)

	// Generate handoff ID
	handoffID := fmt.Sprintf("hoff_%d", time.Now().UnixNano()%100000000)

	// Determine expiration (default 24 hours)
	expiresIn := 24 * time.Hour
	if exp, ok := args["expires_in_hours"].(float64); ok && exp > 0 {
		expiresIn = time.Duration(exp) * time.Hour
	}

	// Get current agent type
	fromAgent := "unknown"
	if s.currentSessionID != "" {
		if session, err := s.butler.Memory().GetSession(s.currentSessionID); err == nil && session != nil {
			fromAgent = session.AgentType
		}
	}

	handoff := &Handoff{
		ID:            handoffID,
		FromAgent:     fromAgent,
		FromSessionID: s.currentSessionID,
		ToAgentType:   toAgentType,
		Task:          task,
		Context:       context,
		PendingWork:   pendingWork,
		PinnedRecords: pinnedRecords,
		Priority:      priority,
		Status:        "pending",
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(expiresIn),
	}

	// Store handoff
	handoffMu.Lock()
	handoffStore[handoffID] = handoff
	handoffMu.Unlock()

	var output strings.Builder
	output.WriteString("# Handoff Created\n\n")
	fmt.Fprintf(&output, "**Handoff ID:** `%s`\n", handoffID)
	fmt.Fprintf(&output, "**To:** %s\n", toAgentType)
	fmt.Fprintf(&output, "**Priority:** %s\n", priority)
	fmt.Fprintf(&output, "**Expires:** %s\n\n", handoff.ExpiresAt.Format(time.RFC3339))
	fmt.Fprintf(&output, "**Task:** %s\n", task)

	if context != "" {
		fmt.Fprintf(&output, "\n**Context:** %s\n", context)
	}

	if len(pendingWork) > 0 {
		output.WriteString("\n**Pending Work:**\n")
		for _, item := range pendingWork {
			fmt.Fprintf(&output, "- [ ] %s\n", item)
		}
	}

	if len(pinnedRecords) > 0 {
		fmt.Fprintf(&output, "\n**Pinned Records:** %d items\n", len(pinnedRecords))
	}

	output.WriteString("\n---\n")
	output.WriteString("The handoff has been created and is waiting to be picked up.\n")
	output.WriteString("Another agent can use `handoff_accept` to take over this task.\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolHandoffList lists available handoffs.
func (s *MCPServer) toolHandoffList(id any, args map[string]interface{}) jsonRPCResponse {
	status, _ := args["status"].(string)
	if status == "" {
		status = "pending"
	}

	handoffMu.RLock()
	var handoffs []*Handoff
	now := time.Now()
	for _, h := range handoffStore {
		// Skip expired
		if h.ExpiresAt.Before(now) && h.Status == "pending" {
			continue
		}
		if status == "" || h.Status == status {
			handoffs = append(handoffs, h)
		}
	}
	handoffMu.RUnlock()

	var output strings.Builder
	output.WriteString("# Available Handoffs\n\n")

	if len(handoffs) == 0 {
		output.WriteString("No handoffs available.\n")
	} else {
		for _, h := range handoffs {
			priorityIcon := "🔵"
			switch h.Priority {
			case "high":
				priorityIcon = "🟠"
			case "urgent":
				priorityIcon = "🔴"
			}

			fmt.Fprintf(&output, "## %s `%s` - %s\n\n", priorityIcon, h.ID, truncateString(h.Task, 60))
			fmt.Fprintf(&output, "- **From:** %s\n", h.FromAgent)
			fmt.Fprintf(&output, "- **To:** %s\n", h.ToAgentType)
			fmt.Fprintf(&output, "- **Priority:** %s\n", h.Priority)
			fmt.Fprintf(&output, "- **Created:** %s\n", h.CreatedAt.Format(time.RFC3339))
			if len(h.PendingWork) > 0 {
				fmt.Fprintf(&output, "- **Pending items:** %d\n", len(h.PendingWork))
			}
			output.WriteString("\n")
		}
	}

	output.WriteString("---\n")
	output.WriteString("Use `handoff_accept({id: '...'})` to accept a handoff.\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// toolHandoffAccept accepts a handoff and provides full context.
func (s *MCPServer) toolHandoffAccept(id any, args map[string]interface{}) jsonRPCResponse {
	handoffID, _ := args["id"].(string)
	if handoffID == "" {
		return s.toolError(id, "handoff id is required")
	}

	handoffMu.Lock()
	handoff, exists := handoffStore[handoffID]
	if !exists {
		handoffMu.Unlock()
		return s.toolError(id, "handoff not found")
	}

	if handoff.Status != "pending" {
		handoffMu.Unlock()
		return s.toolError(id, fmt.Sprintf("handoff is already %s", handoff.Status))
	}

	// Mark as accepted
	handoff.Status = "accepted"
	handoff.AcceptedBy = s.currentSessionID
	handoffMu.Unlock()

	// Set context focus from handoff
	s.currentTaskFocus = handoff.Task
	s.focusKeywords = extractKeywords(handoff.Task)
	s.contextPriorityUp = handoff.PinnedRecords

	var output strings.Builder
	output.WriteString("# Handoff Accepted\n\n")
	fmt.Fprintf(&output, "**Handoff ID:** `%s`\n", handoffID)
	fmt.Fprintf(&output, "**From:** %s (session: %s)\n", handoff.FromAgent, handoff.FromSessionID)
	fmt.Fprintf(&output, "**Priority:** %s\n\n", handoff.Priority)

	output.WriteString("## Task\n\n")
	fmt.Fprintf(&output, "%s\n", handoff.Task)

	if handoff.Context != "" {
		output.WriteString("\n## Context\n\n")
		fmt.Fprintf(&output, "%s\n", handoff.Context)
	}

	if len(handoff.PendingWork) > 0 {
		output.WriteString("\n## Pending Work\n\n")
		for _, item := range handoff.PendingWork {
			fmt.Fprintf(&output, "- [ ] %s\n", item)
		}
	}

	// Get pinned learnings
	mem := s.butler.Memory()
	if len(handoff.PinnedRecords) > 0 && mem != nil {
		output.WriteString("\n## Pinned Knowledge\n\n")
		for _, pid := range handoff.PinnedRecords {
			if strings.HasPrefix(pid, "l_") {
				if l, err := mem.GetLearning(pid); err == nil && l != nil {
					fmt.Fprintf(&output, "- `%s`: %s\n", l.ID, truncateString(l.Content, 80))
				}
			} else if strings.HasPrefix(pid, "d_") {
				if d, err := mem.GetDecision(pid); err == nil && d != nil {
					fmt.Fprintf(&output, "- `%s`: %s\n", d.ID, truncateString(d.Content, 80))
				}
			}
		}
	}

	output.WriteString("\n---\n")
	output.WriteString("You have accepted this handoff. Your context focus has been set to the task.\n")
	output.WriteString("When complete, use `handoff_complete({id: '...'})` to mark it done.\n")

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: output.String()}},
		},
	}
}

// getPendingHandoffsForAgent returns pending handoffs that match the given agent type.
func getPendingHandoffsForAgent(agentType string) []*Handoff {
	handoffMu.RLock()
	defer handoffMu.RUnlock()

	var handoffs []*Handoff
	now := time.Now()

	for _, h := range handoffStore {
		// Skip non-pending or expired
		if h.Status != "pending" || h.ExpiresAt.Before(now) {
			continue
		}
		// Match if target is "any" or matches the agent type
		if h.ToAgentType == "any" || h.ToAgentType == agentType {
			handoffs = append(handoffs, h)
		}
	}

	// Sort by priority (urgent > high > normal > low)
	priorityOrder := map[string]int{"urgent": 0, "high": 1, "normal": 2, "low": 3}
	for i := 0; i < len(handoffs)-1; i++ {
		for j := i + 1; j < len(handoffs); j++ {
			if priorityOrder[handoffs[i].Priority] > priorityOrder[handoffs[j].Priority] {
				handoffs[i], handoffs[j] = handoffs[j], handoffs[i]
			}
		}
	}

	return handoffs
}

// toolHandoffComplete marks a handoff as completed.
func (s *MCPServer) toolHandoffComplete(id any, args map[string]interface{}) jsonRPCResponse {
	handoffID, _ := args["id"].(string)
	if handoffID == "" {
		return s.toolError(id, "handoff id is required")
	}

	summary, _ := args["summary"].(string)

	handoffMu.Lock()
	handoff, exists := handoffStore[handoffID]
	if !exists {
		handoffMu.Unlock()
		return s.toolError(id, "handoff not found")
	}

	handoff.Status = "completed"
	handoff.Summary = summary
	handoffMu.Unlock()

	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("Handoff `%s` marked as completed.", handoffID)}},
		},
	}
}
