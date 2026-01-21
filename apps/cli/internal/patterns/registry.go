package patterns

import (
	"fmt"
	"sort"
	"sync"
)

// Registry manages all registered pattern detectors.
type Registry struct {
	detectors map[string]Detector
	mu        sync.RWMutex
}

// NewRegistry creates a new detector registry.
func NewRegistry() *Registry {
	return &Registry{
		detectors: make(map[string]Detector),
	}
}

// Register adds a detector to the registry.
// It returns an error if a detector with the same ID already exists.
func (r *Registry) Register(d Detector) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.detectors[d.ID()]; exists {
		return fmt.Errorf("detector already registered: %s", d.ID())
	}

	r.detectors[d.ID()] = d
	return nil
}

// MustRegister adds a detector to the registry and panics on error.
// Use this for built-in detectors that should always succeed.
func (r *Registry) MustRegister(d Detector) {
	if err := r.Register(d); err != nil {
		panic(err)
	}
}

// Get returns a detector by ID, or nil if not found.
func (r *Registry) Get(id string) Detector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.detectors[id]
}

// Has checks if a detector with the given ID exists.
func (r *Registry) Has(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.detectors[id]
	return exists
}

// All returns all registered detectors.
func (r *Registry) All() []Detector {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Detector, 0, len(r.detectors))
	for _, d := range r.detectors {
		result = append(result, d)
	}

	// Sort by ID for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID() < result[j].ID()
	})

	return result
}

// ByCategory returns all detectors in a given category.
func (r *Registry) ByCategory(category PatternCategory) []Detector {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Detector
	for _, d := range r.detectors {
		if d.Category() == category {
			result = append(result, d)
		}
	}

	// Sort by ID for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID() < result[j].ID()
	})

	return result
}

// ByLanguage returns all detectors that support the given language.
func (r *Registry) ByLanguage(language string) []Detector {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Detector
	for _, d := range r.detectors {
		langs := d.Languages()
		// Empty languages means all languages supported
		if len(langs) == 0 {
			result = append(result, d)
			continue
		}
		for _, l := range langs {
			if l == language {
				result = append(result, d)
				break
			}
		}
	}

	// Sort by ID for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID() < result[j].ID()
	})

	return result
}

// Categories returns all categories that have at least one registered detector.
func (r *Registry) Categories() []PatternCategory {
	r.mu.RLock()
	defer r.mu.RUnlock()

	categorySet := make(map[PatternCategory]struct{})
	for _, d := range r.detectors {
		categorySet[d.Category()] = struct{}{}
	}

	result := make([]PatternCategory, 0, len(categorySet))
	for cat := range categorySet {
		result = append(result, cat)
	}

	// Sort for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	return result
}

// Count returns the number of registered detectors.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.detectors)
}

// IDs returns all registered detector IDs.
func (r *Registry) IDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.detectors))
	for id := range r.detectors {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// DefaultRegistry is the global registry for pattern detectors.
var DefaultRegistry = NewRegistry()

// Register registers a detector with the default registry.
func Register(d Detector) error {
	return DefaultRegistry.Register(d)
}

// MustRegister registers a detector with the default registry and panics on error.
func MustRegister(d Detector) {
	DefaultRegistry.MustRegister(d)
}
