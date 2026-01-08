package butler

import (
	"context"
	"fmt"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/index"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/llm"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// HasMemory returns true if session memory is available.
func (b *Butler) HasMemory() bool {
	return b.memory != nil
}

// StartSession creates a new session.
func (b *Butler) StartSession(agentType, agentID, goal string) (*memory.Session, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.StartSession(agentType, agentID, goal)
}

// EndSession ends a session.
func (b *Butler) EndSession(sessionID, state, summary string) error {
	if b.memory == nil {
		return fmt.Errorf("session memory not available")
	}

	// End the session first
	if err := b.memory.EndSession(sessionID, state, summary); err != nil {
		return err
	}

	// Auto-extract if enabled and LLM is configured
	if b.config != nil && b.config.AutoExtract {
		if llmClient, err := b.GetLLMClient(); err == nil && llmClient != nil {
			// Find conversation for this session and extract
			go b.autoExtractForSession(sessionID, llmClient)
		}
	}

	return nil
}

// autoExtractForSession finds the conversation for a session and extracts records.
func (b *Butler) autoExtractForSession(sessionID string, llmClient llm.Client) {
	if b.memory == nil {
		return
	}

	// Get conversations for this session
	conversations, err := b.memory.GetConversations(sessionID, "", 1)
	if err != nil || len(conversations) == 0 {
		return
	}

	// Extract from the most recent conversation
	conv := conversations[0]
	extractor := memory.NewLLMExtractor(llmClient, b.memory)
	_, _ = extractor.ExtractFromConversation(conv)
}

// GetSession retrieves a session by ID.
func (b *Butler) GetSession(sessionID string) (*memory.Session, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetSession(sessionID)
}

// ListSessions lists sessions.
func (b *Butler) ListSessions(activeOnly bool, limit int) ([]memory.Session, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.ListSessions(activeOnly, limit)
}

// LogActivity logs an activity.
func (b *Butler) LogActivity(sessionID string, act memory.Activity) error {
	if b.memory == nil {
		return fmt.Errorf("session memory not available")
	}
	return b.memory.LogActivity(sessionID, act)
}

// RecordOutcome records session outcome.
func (b *Butler) RecordOutcome(sessionID, outcome, summary string) error {
	if b.memory == nil {
		return fmt.Errorf("session memory not available")
	}
	return b.memory.RecordOutcome(sessionID, outcome, summary)
}

// AddLearning adds a new learning.
func (b *Butler) AddLearning(l memory.Learning) (string, error) {
	if b.memory == nil {
		return "", fmt.Errorf("session memory not available")
	}
	return b.memory.AddLearning(l)
}

// GetLearnings retrieves learnings.
func (b *Butler) GetLearnings(scope, scopePath string, limit int) ([]memory.Learning, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetLearnings(scope, scopePath, limit)
}

// SearchLearnings searches learnings by content.
func (b *Butler) SearchLearnings(query string, limit int) ([]memory.Learning, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.SearchLearnings(query, limit)
}

// ReinforceLearning increases learning confidence.
func (b *Butler) ReinforceLearning(id string) error {
	if b.memory == nil {
		return fmt.Errorf("session memory not available")
	}
	return b.memory.ReinforceLearning(id)
}

// GetFileIntel gets file intelligence.
func (b *Butler) GetFileIntel(path string) (*memory.FileIntel, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetFileIntel(path)
}

// RecordFileEdit records a file edit.
func (b *Butler) RecordFileEdit(path, agentType string) error {
	if b.memory == nil {
		return fmt.Errorf("session memory not available")
	}
	return b.memory.RecordFileEdit(path, agentType)
}

// GetActivities retrieves activities.
func (b *Butler) GetActivities(sessionID, filePath string, limit int) ([]memory.Activity, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetActivities(sessionID, filePath, limit)
}

// GetRelevantLearnings gets relevant learnings for a context.
func (b *Butler) GetRelevantLearnings(filePath, query string, limit int) ([]memory.Learning, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetRelevantLearnings(filePath, query, limit)
}

// GetActiveAgents returns currently active agents.
func (b *Butler) GetActiveAgents() ([]memory.ActiveAgent, error) {
	if b.memory == nil {
		return nil, fmt.Errorf("session memory not available")
	}
	return b.memory.GetActiveAgents(5 * time.Minute)
}

// CheckConflict checks if another agent is working on a file.
func (b *Butler) CheckConflict(sessionID, path string) (*memory.Conflict, error) {
	if b.memory == nil {
		return nil, nil // No conflict if memory not available
	}
	return b.memory.CheckConflict(sessionID, path)
}

// GetBrief returns a comprehensive briefing.
func (b *Butler) GetBrief(filePath string) (*BriefingResult, error) {
	if b.memory == nil {
		return &BriefingResult{}, nil
	}

	result := &BriefingResult{
		FilePath: filePath,
	}

	// Get active agents
	agents, err := b.memory.GetActiveAgents(5 * time.Minute)
	if err == nil {
		result.ActiveAgents = agents
	}

	// Check conflict if file specified
	if filePath != "" {
		conflict, err := b.memory.CheckConflict("", filePath)
		if err == nil && conflict != nil {
			result.Conflict = conflict
		}

		intel, err := b.memory.GetFileIntel(filePath)
		if err == nil {
			result.FileIntel = intel
		}
	}

	// Get relevant learnings
	learnings, err := b.memory.GetRelevantLearnings(filePath, "", 5)
	if err == nil {
		result.Learnings = learnings
	}

	// Get hotspots
	hotspots, err := b.memory.GetFileHotspots(5)
	if err == nil {
		result.Hotspots = hotspots
	}

	// Get recent brain ideas (active ones)
	ideas, err := b.memory.GetIdeas("active", "", "", 5)
	if err == nil {
		result.BrainIdeas = ideas
	}

	// Get recent brain decisions (active, with outcomes shown)
	decisions, err := b.memory.GetDecisions("active", "", "", "", 5)
	if err == nil {
		result.BrainDecisions = decisions
	}

	return result, nil
}

// GetIndexInfo returns information about the code index.
func (b *Butler) GetIndexInfo() *IndexInfo {
	if b.db == nil {
		return nil
	}

	info := &IndexInfo{
		Status: "fresh",
	}

	// Count files in index
	row := b.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM files")
	if err := row.Scan(&info.FileCount); err != nil {
		// If count fails, FileCount remains 0 which is acceptable
		info.FileCount = 0
	}

	// Get last scan time from scans table
	scan, err := index.LatestScan(b.db)
	if err == nil && !scan.CompletedAt.IsZero() {
		info.LastScan = scan.CompletedAt
		// Check if stale (more than 1 hour old)
		if time.Since(scan.CompletedAt) > time.Hour {
			info.Status = "stale"
		}
	}

	return info
}
