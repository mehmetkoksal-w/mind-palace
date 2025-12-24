// Package corridor provides a public API for Mind Palace global corridors.
// Corridors enable cross-workspace learning by storing personal learnings
// in a global location (~/.palace/corridors/) and linking to other workspaces.
//
// Example usage:
//
//	cor, err := corridor.OpenGlobal()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cor.Close()
//
//	// Add a personal learning
//	cor.AddPersonalLearning(corridor.PersonalLearning{
//	    Content:    "Always validate user input before processing",
//	    Confidence: 0.9,
//	    Source:     "promoted",
//	})
//
//	// Link another workspace
//	cor.Link("my-api", "/path/to/api-project")
//
//	// Get learnings from all linked workspaces
//	learnings, _ := cor.GetAllLinkedLearnings(10)
package corridor

import (
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/corridor"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
	"github.com/koksalmehmet/mind-palace/apps/cli/pkg/types"
)

// Re-export types for convenience
type (
	PersonalLearning = types.PersonalLearning
	LinkedWorkspace  = types.LinkedWorkspace
	Learning         = types.Learning
)

// GlobalCorridor manages cross-workspace learnings stored in ~/.palace/corridors/
type GlobalCorridor struct {
	internal *corridor.GlobalCorridor
}

// GlobalPath returns the path to the global palace directory (~/.palace).
func GlobalPath() (string, error) {
	return corridor.GlobalPath()
}

// EnsureGlobalLayout creates the global palace directory structure if needed.
func EnsureGlobalLayout() (string, error) {
	return corridor.EnsureGlobalLayout()
}

// OpenGlobal opens the global corridor database.
// Creates the database if it doesn't exist.
func OpenGlobal() (*GlobalCorridor, error) {
	g, err := corridor.OpenGlobal()
	if err != nil {
		return nil, err
	}
	return &GlobalCorridor{internal: g}, nil
}

// Close closes the corridor database.
func (g *GlobalCorridor) Close() error {
	return g.internal.Close()
}

// AddPersonalLearning adds a learning to the personal corridor.
// These learnings are available across all your workspaces.
func (g *GlobalCorridor) AddPersonalLearning(l PersonalLearning) error {
	return g.internal.AddPersonalLearning(corridor.PersonalLearning{
		ID:              l.ID,
		OriginWorkspace: l.OriginWorkspace,
		Content:         l.Content,
		Confidence:      l.Confidence,
		Source:          l.Source,
		CreatedAt:       l.CreatedAt,
		LastUsed:        l.LastUsed,
		UseCount:        l.UseCount,
		Tags:            l.Tags,
	})
}

// GetPersonalLearnings retrieves personal learnings, optionally filtered by query.
func (g *GlobalCorridor) GetPersonalLearnings(query string, limit int) ([]PersonalLearning, error) {
	learnings, err := g.internal.GetPersonalLearnings(query, limit)
	if err != nil {
		return nil, err
	}
	return convertPersonalLearnings(learnings), nil
}

// ReinforceLearning increases confidence for a learning.
// Call this when a learning proves useful.
func (g *GlobalCorridor) ReinforceLearning(id string) error {
	return g.internal.ReinforceLearning(id)
}

// DeleteLearning removes a learning from the personal corridor.
func (g *GlobalCorridor) DeleteLearning(id string) error {
	return g.internal.DeleteLearning(id)
}

// Link connects a workspace to the global corridor.
// The workspace must have a .palace directory.
//
// Example:
//
//	cor.Link("my-api", "/Users/me/code/api-service")
func (g *GlobalCorridor) Link(name, localPath string) error {
	return g.internal.Link(name, localPath)
}

// Unlink removes a workspace link.
func (g *GlobalCorridor) Unlink(name string) error {
	return g.internal.Unlink(name)
}

// GetLinks returns all linked workspaces.
func (g *GlobalCorridor) GetLinks() ([]LinkedWorkspace, error) {
	links, err := g.internal.GetLinks()
	if err != nil {
		return nil, err
	}
	return convertLinkedWorkspaces(links), nil
}

// GetLinkedLearnings retrieves learnings from a specific linked workspace.
func (g *GlobalCorridor) GetLinkedLearnings(name string, limit int) ([]Learning, error) {
	learnings, err := g.internal.GetLinkedLearnings(name, limit)
	if err != nil {
		return nil, err
	}
	return convertInternalLearnings(learnings), nil
}

// GetAllLinkedLearnings retrieves learnings from all linked workspaces.
func (g *GlobalCorridor) GetAllLinkedLearnings(limit int) ([]Learning, error) {
	learnings, err := g.internal.GetAllLinkedLearnings(limit)
	if err != nil {
		return nil, err
	}
	return convertInternalLearnings(learnings), nil
}

// Stats returns statistics about the personal corridor.
func (g *GlobalCorridor) Stats() (map[string]any, error) {
	return g.internal.Stats()
}

// Conversion helpers
func convertPersonalLearnings(learnings []corridor.PersonalLearning) []PersonalLearning {
	result := make([]PersonalLearning, len(learnings))
	for i, l := range learnings {
		result[i] = PersonalLearning{
			ID:              l.ID,
			OriginWorkspace: l.OriginWorkspace,
			Content:         l.Content,
			Confidence:      l.Confidence,
			Source:          l.Source,
			CreatedAt:       l.CreatedAt,
			LastUsed:        l.LastUsed,
			UseCount:        l.UseCount,
			Tags:            l.Tags,
		}
	}
	return result
}

func convertLinkedWorkspaces(links []corridor.LinkedWorkspace) []LinkedWorkspace {
	result := make([]LinkedWorkspace, len(links))
	for i, l := range links {
		result[i] = LinkedWorkspace{
			Name:         l.Name,
			Path:         l.Path,
			AddedAt:      l.AddedAt,
			LastAccessed: l.LastAccessed,
		}
	}
	return result
}

func convertInternalLearnings(learnings []memory.Learning) []Learning {
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
