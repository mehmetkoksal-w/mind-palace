// Package watch provides file system watching with debouncing for the Mind Palace index.
package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/config"
)

// FileChange represents a detected file change.
type FileChange struct {
	Path   string // Relative path from root
	Action string // "create", "modify", "delete", "rename"
}

// OnChangeFunc is the callback type for handling batched file changes.
type OnChangeFunc func(changes []FileChange) error

// WatcherConfig contains configuration for the file watcher.
type WatcherConfig struct {
	// Debounce is the delay before processing accumulated changes.
	// Default: 500ms
	Debounce time.Duration

	// IgnorePatterns are glob patterns for paths to ignore.
	// Default: common patterns like .git, node_modules, etc.
	IgnorePatterns []string

	// Recursive enables recursive directory watching.
	// Default: true
	Recursive bool
}

// DefaultConfig returns the default watcher configuration.
func DefaultConfig() WatcherConfig {
	return WatcherConfig{
		Debounce:  500 * time.Millisecond,
		Recursive: true,
		IgnorePatterns: []string{
			".git",
			".palace/index",
			".palace/cache",
			".palace/outputs",
			".palace/sessions",
			"node_modules",
			"vendor",
			".venv",
			"__pycache__",
			"*.pyc",
			".DS_Store",
			"Thumbs.db",
		},
	}
}

// Watcher watches a directory tree for file changes with debouncing.
type Watcher struct {
	root       string
	config     WatcherConfig
	guardrails config.Guardrails
	onChange   OnChangeFunc
	watcher    *fsnotify.Watcher

	// Debounce state
	mu            sync.Mutex
	pending       map[string]FileChange
	debounceTimer *time.Timer

	// Control
	done chan struct{}
	wg   sync.WaitGroup
}

// New creates a new file watcher for the given root directory.
func New(root string, guardrails config.Guardrails, onChange OnChangeFunc, cfg WatcherConfig) (*Watcher, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root path: %w", err)
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify watcher: %w", err)
	}

	w := &Watcher{
		root:       absRoot,
		config:     cfg,
		guardrails: guardrails,
		onChange:   onChange,
		watcher:    fsWatcher,
		pending:    make(map[string]FileChange),
		done:       make(chan struct{}),
	}

	return w, nil
}

// Start begins watching for file changes. This blocks until Stop is called or ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	// Add root directory and all subdirectories
	if err := w.addWatchRecursive(w.root); err != nil {
		return fmt.Errorf("add watch paths: %w", err)
	}

	// Start the event processing goroutine
	w.wg.Add(1)
	go w.processEvents(ctx)

	// Wait for either context cancellation or Stop()
	select {
	case <-ctx.Done():
		w.Stop()
		return ctx.Err()
	case <-w.done:
		return nil
	}
}

// Stop gracefully stops the watcher.
func (w *Watcher) Stop() {
	// Signal done
	select {
	case <-w.done:
		// Already stopped
		return
	default:
		close(w.done)
	}

	// Wait for goroutines
	w.wg.Wait()

	// Close the fsnotify watcher
	w.watcher.Close()

	// Cancel any pending debounce timer
	w.mu.Lock()
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}
	w.mu.Unlock()
}

// addWatchRecursive adds a directory and all its subdirectories to the watcher.
func (w *Watcher) addWatchRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible paths
		}

		if !info.IsDir() {
			return nil // Only watch directories
		}

		// Check if path should be ignored
		relPath, err := filepath.Rel(w.root, path)
		if err != nil {
			return nil
		}

		if w.shouldIgnore(relPath) {
			return filepath.SkipDir
		}

		if err := w.watcher.Add(path); err != nil {
			// Log but don't fail on individual watch errors
			return nil
		}

		return nil
	})
}

// shouldIgnore returns true if the path should be ignored based on patterns.
func (w *Watcher) shouldIgnore(relPath string) bool {
	// Normalize path separators
	relPath = filepath.ToSlash(relPath)

	// Check ignore patterns
	for _, pattern := range w.config.IgnorePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(relPath)); matched {
			return true
		}
		if strings.HasPrefix(relPath, pattern+"/") || relPath == pattern {
			return true
		}
	}

	// Check guardrails
	for _, glob := range w.guardrails.DoNotTouchGlobs {
		// Simple prefix matching for directories
		glob = strings.TrimSuffix(glob, "/**")
		glob = strings.TrimSuffix(glob, "/*")
		if strings.HasPrefix(relPath, glob+"/") || relPath == glob {
			return true
		}
	}

	return false
}

// processEvents handles fsnotify events and triggers onChange with debouncing.
func (w *Watcher) processEvents(ctx context.Context) {
	defer w.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			// Log error but continue watching
			fmt.Fprintf(os.Stderr, "watch error: %v\n", err)
		}
	}
}

// handleEvent processes a single fsnotify event.
func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Get relative path
	relPath, err := filepath.Rel(w.root, event.Name)
	if err != nil {
		return
	}

	// Normalize path
	relPath = filepath.ToSlash(relPath)

	// Ignore paths that match patterns
	if w.shouldIgnore(relPath) {
		return
	}

	// Determine action
	var action string
	switch {
	case event.Op&fsnotify.Create != 0:
		action = "create"
		// If a directory was created, add it to the watch list
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			w.addWatchRecursive(event.Name)
		}
	case event.Op&fsnotify.Write != 0:
		action = "modify"
	case event.Op&fsnotify.Remove != 0:
		action = "delete"
	case event.Op&fsnotify.Rename != 0:
		action = "rename"
	case event.Op&fsnotify.Chmod != 0:
		// Ignore chmod events
		return
	default:
		return
	}

	// Add to pending changes with debounce
	w.mu.Lock()
	defer w.mu.Unlock()

	// Use the most recent action for this path
	// Priority: delete > create > modify (if file deleted then created, treat as modify)
	existing, exists := w.pending[relPath]
	if exists {
		if existing.Action == "delete" && action == "create" {
			action = "modify"
		} else if action == "delete" {
			// Delete supersedes everything
		}
	}

	w.pending[relPath] = FileChange{
		Path:   relPath,
		Action: action,
	}

	// Reset or start debounce timer
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}
	w.debounceTimer = time.AfterFunc(w.config.Debounce, w.flushPending)
}

// flushPending processes all pending changes.
func (w *Watcher) flushPending() {
	w.mu.Lock()

	if len(w.pending) == 0 {
		w.mu.Unlock()
		return
	}

	// Collect changes
	changes := make([]FileChange, 0, len(w.pending))
	for _, change := range w.pending {
		changes = append(changes, change)
	}

	// Clear pending
	w.pending = make(map[string]FileChange)
	w.mu.Unlock()

	// Call the onChange handler
	if w.onChange != nil {
		if err := w.onChange(changes); err != nil {
			fmt.Fprintf(os.Stderr, "onChange error: %v\n", err)
		}
	}
}

// Stats returns current watcher statistics.
func (w *Watcher) Stats() WatcherStats {
	w.mu.Lock()
	defer w.mu.Unlock()

	return WatcherStats{
		WatchedDirs:    len(w.watcher.WatchList()),
		PendingChanges: len(w.pending),
	}
}

// WatcherStats contains watcher statistics.
type WatcherStats struct {
	WatchedDirs    int
	PendingChanges int
}
