package butler

import (
	"fmt"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// ============================================================================
// Brain Methods (Ideas & Decisions)
// ============================================================================

// AddIdea adds a new idea.
func (b *Butler) AddIdea(i memory.Idea) (string, error) {
	if b.memory == nil {
		return "", fmt.Errorf("session memory not available")
	}
	return b.memory.AddIdea(i)
}

// AddDecision adds a new decision.
func (b *Butler) AddDecision(d memory.Decision) (string, error) {
	if b.memory == nil {
		return "", fmt.Errorf("session memory not available")
	}
	return b.memory.AddDecision(d)
}

// GetDecisions retrieves decisions with optional filtering.
func (b *Butler) GetDecisions(status, scope, scopePath string, limit int) ([]memory.Decision, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetDecisions(status, "", scope, scopePath, limit)
}

// SearchDecisions searches decisions by content.
func (b *Butler) SearchDecisions(query string, limit int) ([]memory.Decision, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.SearchDecisions(query, limit)
}

// GetIdeas retrieves ideas with optional filtering.
func (b *Butler) GetIdeas(status, scope, scopePath string, limit int) ([]memory.Idea, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetIdeas(status, scope, scopePath, limit)
}

// SearchIdeas searches ideas by content.
func (b *Butler) SearchIdeas(query string, limit int) ([]memory.Idea, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.SearchIdeas(query, limit)
}

// RecordDecisionOutcome records the outcome of a decision.
func (b *Butler) RecordDecisionOutcome(id, outcome, note string) error {
	if b.memory == nil {
		return fmt.Errorf("session memory not available")
	}
	return b.memory.RecordDecisionOutcome(id, outcome, note)
}

// SetTags sets tags for a record.
func (b *Butler) SetTags(recordID, recordKind string, tags []string) error {
	if b.memory == nil {
		return fmt.Errorf("session memory not available")
	}
	return b.memory.SetTags(recordID, recordKind, tags)
}

// AddLink creates a link between records.
func (b *Butler) AddLink(link memory.Link) (string, error) {
	if b.memory == nil {
		return "", fmt.Errorf("session memory not available")
	}
	return b.memory.AddLink(link)
}

// GetLink retrieves a link by ID.
func (b *Butler) GetLink(id string) (*memory.Link, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetLink(id)
}

// GetLinksForRecord retrieves all links for a record (as source or target).
func (b *Butler) GetLinksForRecord(recordID string) ([]memory.Link, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetAllLinksFor(recordID)
}

// DeleteLink deletes a link by ID.
func (b *Butler) DeleteLink(id string) error {
	if b.memory == nil {
		return fmt.Errorf("session memory not available")
	}
	return b.memory.DeleteLink(id)
}

// ============================================================================
// Conversation Methods
// ============================================================================

// AddConversation stores a new conversation.
func (b *Butler) AddConversation(c memory.Conversation) (string, error) {
	if b.memory == nil {
		return "", fmt.Errorf("session memory not available")
	}
	return b.memory.AddConversation(c)
}

// GetConversation retrieves a conversation by ID.
func (b *Butler) GetConversation(id string) (*memory.Conversation, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetConversation(id)
}

// GetConversations retrieves conversations with optional filters.
func (b *Butler) GetConversations(sessionID, agentType string, limit int) ([]memory.Conversation, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetConversations(sessionID, agentType, limit)
}

// SearchConversations searches conversations by summary.
func (b *Butler) SearchConversations(query string, limit int) ([]memory.Conversation, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.SearchConversations(query, limit)
}

// GetConversationForSession retrieves the conversation for a specific session.
func (b *Butler) GetConversationForSession(sessionID string) (*memory.Conversation, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetConversationForSession(sessionID)
}
