// Package memory provides a public API for Mind Palace session memory.
// External tools can import this package to interact with workspace memory.
//
// Example usage:
//
//	mem, err := memory.Open("/path/to/workspace")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer mem.Close()
//
//	session, err := mem.StartSession("my-agent", "instance-1", "Implement feature X")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Log activities as you work
//	mem.LogActivity(session.ID, memory.Activity{
//	    Kind:    memory.ActivityFileEdit,
//	    Target:  "main.go",
//	    Outcome: memory.OutcomeSuccess,
//	})
//
//	// End session when done
//	mem.EndSession(session.ID, memory.SessionCompleted, "Feature implemented")
package memory

import (
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
	"github.com/koksalmehmet/mind-palace/apps/cli/pkg/types"
)

// Re-export types for convenience
type (
	Session   = types.Session
	Activity  = types.Activity
	Learning  = types.Learning
	FileIntel = types.FileIntel
	Conflict  = types.Conflict
)

// Re-export constants
const (
	ActivityFileRead  = types.ActivityFileRead
	ActivityFileEdit  = types.ActivityFileEdit
	ActivitySearch    = types.ActivitySearch
	ActivityCommand   = types.ActivityCommand
	SessionActive     = types.SessionActive
	SessionCompleted  = types.SessionCompleted
	SessionAbandoned  = types.SessionAbandoned
	ScopeFile         = types.ScopeFile
	ScopeRoom         = types.ScopeRoom
	ScopePalace       = types.ScopePalace
	OutcomeSuccess    = types.OutcomeSuccess
	OutcomeFailure    = types.OutcomeFailure
	OutcomeUnknown    = types.OutcomeUnknown
)

// Memory provides workspace session memory management.
// Use Open() to create a new instance.
type Memory struct {
	internal *memory.Memory
}

// Open opens or creates the memory database at the given workspace root.
// The database is stored at <root>/.palace/memory.db
func Open(workspaceRoot string) (*Memory, error) {
	m, err := memory.Open(workspaceRoot)
	if err != nil {
		return nil, err
	}
	return &Memory{internal: m}, nil
}

// Close closes the memory database.
func (m *Memory) Close() error {
	return m.internal.Close()
}

// StartSession begins a new agent work session.
// Returns the created session with a unique ID.
func (m *Memory) StartSession(agentType, agentID, goal string) (*Session, error) {
	s, err := m.internal.StartSession(agentType, agentID, goal)
	if err != nil {
		return nil, err
	}
	return convertSession(s), nil
}

// EndSession ends a session with the given state and summary.
// State should be one of: SessionCompleted, SessionAbandoned
func (m *Memory) EndSession(sessionID, state, summary string) error {
	return m.internal.EndSession(sessionID, state, summary)
}

// GetSession retrieves a session by ID.
func (m *Memory) GetSession(sessionID string) (*Session, error) {
	s, err := m.internal.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	return convertSession(s), nil
}

// ListSessions lists sessions, optionally filtering to active only.
func (m *Memory) ListSessions(activeOnly bool, limit int) ([]Session, error) {
	sessions, err := m.internal.ListSessions(activeOnly, limit)
	if err != nil {
		return nil, err
	}
	return convertSessions(sessions), nil
}

// LogActivity records an activity for a session.
func (m *Memory) LogActivity(sessionID string, act Activity) error {
	return m.internal.LogActivity(sessionID, memory.Activity{
		Kind:    act.Kind,
		Target:  act.Target,
		Details: act.Details,
		Outcome: act.Outcome,
	})
}

// GetActivities retrieves activities, optionally filtered by session or file.
func (m *Memory) GetActivities(sessionID, filePath string, limit int) ([]Activity, error) {
	activities, err := m.internal.GetActivities(sessionID, filePath, limit)
	if err != nil {
		return nil, err
	}
	return convertActivities(activities), nil
}

// AddLearning adds a new learning to the workspace.
func (m *Memory) AddLearning(l Learning) (string, error) {
	return m.internal.AddLearning(memory.Learning{
		SessionID:  l.SessionID,
		Scope:      l.Scope,
		ScopePath:  l.ScopePath,
		Content:    l.Content,
		Confidence: l.Confidence,
		Source:     l.Source,
	})
}

// GetLearnings retrieves learnings filtered by scope and path.
func (m *Memory) GetLearnings(scope, scopePath string, limit int) ([]Learning, error) {
	learnings, err := m.internal.GetLearnings(scope, scopePath, limit)
	if err != nil {
		return nil, err
	}
	return convertLearnings(learnings), nil
}

// SearchLearnings searches learnings by content.
func (m *Memory) SearchLearnings(query string, limit int) ([]Learning, error) {
	learnings, err := m.internal.SearchLearnings(query, limit)
	if err != nil {
		return nil, err
	}
	return convertLearnings(learnings), nil
}

// GetRelevantLearnings gets learnings relevant to a file path and room.
func (m *Memory) GetRelevantLearnings(filePath, room string, limit int) ([]Learning, error) {
	learnings, err := m.internal.GetRelevantLearnings(filePath, room, limit)
	if err != nil {
		return nil, err
	}
	return convertLearnings(learnings), nil
}

// ReinforceLearning increases the confidence of a learning.
func (m *Memory) ReinforceLearning(learningID string) error {
	return m.internal.ReinforceLearning(learningID)
}

// GetFileIntel retrieves intelligence about a specific file.
func (m *Memory) GetFileIntel(path string) (*FileIntel, error) {
	fi, err := m.internal.GetFileIntel(path)
	if err != nil {
		return nil, err
	}
	return convertFileIntel(fi), nil
}

// RecordFileEdit records a file edit for intel tracking.
func (m *Memory) RecordFileEdit(path, agentType string) error {
	return m.internal.RecordFileEdit(path, agentType)
}

// RecordFileFailure records a failure related to a file.
func (m *Memory) RecordFileFailure(path string) error {
	return m.internal.RecordFileFailure(path)
}

// GetFileHotspots returns files ordered by edit frequency.
func (m *Memory) GetFileHotspots(limit int) ([]FileIntel, error) {
	hotspots, err := m.internal.GetFileHotspots(limit)
	if err != nil {
		return nil, err
	}
	return convertFileIntels(hotspots), nil
}

// CheckConflict checks if another session is working on the same file.
func (m *Memory) CheckConflict(sessionID, path string) (*Conflict, error) {
	c, err := m.internal.CheckConflict(sessionID, path)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	return &Conflict{
		Path:         c.Path,
		OtherSession: c.OtherSession,
		OtherAgent:   c.OtherAgent,
		LastTouched:  c.LastTouched,
		Severity:     c.Severity,
	}, nil
}

// Conversion helpers
func convertSession(s *memory.Session) *Session {
	if s == nil {
		return nil
	}
	return &Session{
		ID:           s.ID,
		AgentType:    s.AgentType,
		AgentID:      s.AgentID,
		Goal:         s.Goal,
		StartedAt:    s.StartedAt,
		LastActivity: s.LastActivity,
		State:        s.State,
		Summary:      s.Summary,
	}
}

func convertSessions(sessions []memory.Session) []Session {
	result := make([]Session, len(sessions))
	for i, s := range sessions {
		result[i] = Session{
			ID:           s.ID,
			AgentType:    s.AgentType,
			AgentID:      s.AgentID,
			Goal:         s.Goal,
			StartedAt:    s.StartedAt,
			LastActivity: s.LastActivity,
			State:        s.State,
			Summary:      s.Summary,
		}
	}
	return result
}

func convertActivities(activities []memory.Activity) []Activity {
	result := make([]Activity, len(activities))
	for i, a := range activities {
		result[i] = Activity{
			ID:        a.ID,
			SessionID: a.SessionID,
			Kind:      a.Kind,
			Target:    a.Target,
			Details:   a.Details,
			Timestamp: a.Timestamp,
			Outcome:   a.Outcome,
		}
	}
	return result
}

func convertLearnings(learnings []memory.Learning) []Learning {
	result := make([]Learning, len(learnings))
	for i, l := range learnings {
		result[i] = Learning{
			ID:         l.ID,
			SessionID:  l.SessionID,
			Scope:      l.Scope,
			ScopePath:  l.ScopePath,
			Content:    l.Content,
			Confidence: l.Confidence,
			Source:     l.Source,
			CreatedAt:  l.CreatedAt,
			LastUsed:   l.LastUsed,
			UseCount:   l.UseCount,
		}
	}
	return result
}

func convertFileIntel(fi *memory.FileIntel) *FileIntel {
	if fi == nil {
		return nil
	}
	return &FileIntel{
		Path:         fi.Path,
		EditCount:    fi.EditCount,
		LastEdited:   fi.LastEdited,
		LastEditor:   fi.LastEditor,
		FailureCount: fi.FailureCount,
		Learnings:    fi.Learnings,
	}
}

func convertFileIntels(fis []memory.FileIntel) []FileIntel {
	result := make([]FileIntel, len(fis))
	for i, fi := range fis {
		result[i] = FileIntel{
			Path:         fi.Path,
			EditCount:    fi.EditCount,
			LastEdited:   fi.LastEdited,
			LastEditor:   fi.LastEditor,
			FailureCount: fi.FailureCount,
			Learnings:    fi.Learnings,
		}
	}
	return result
}
