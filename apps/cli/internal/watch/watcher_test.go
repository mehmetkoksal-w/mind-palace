package watch

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/config"
)

func TestNewWatcher(t *testing.T) {
	root := t.TempDir()

	watcher, err := New(root, config.Guardrails{}, nil, DefaultConfig())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer watcher.Stop()

	if watcher.root != root {
		t.Errorf("expected root %s, got %s", root, watcher.root)
	}
}

func TestWatcherDetectsFileCreation(t *testing.T) {
	root := t.TempDir()

	var changes []FileChange
	var mu sync.Mutex
	changeReceived := make(chan struct{}, 1)

	cfg := DefaultConfig()
	cfg.Debounce = 100 * time.Millisecond

	watcher, err := New(root, config.Guardrails{}, func(c []FileChange) error {
		mu.Lock()
		changes = append(changes, c...)
		mu.Unlock()
		select {
		case changeReceived <- struct{}{}:
		default:
		}
		return nil
	}, cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start watcher in background
	go func() {
		watcher.Start(ctx)
	}()

	// Give watcher time to initialize
	time.Sleep(200 * time.Millisecond)

	// Create a file
	testFile := filepath.Join(root, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for change notification
	select {
	case <-changeReceived:
		// Success
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for file change notification")
	}

	mu.Lock()
	defer mu.Unlock()

	if len(changes) == 0 {
		t.Fatal("expected at least one change")
	}

	// Check the change
	found := false
	for _, c := range changes {
		if c.Path == "test.txt" && (c.Action == "create" || c.Action == "modify") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected change for test.txt, got: %+v", changes)
	}
}

func TestWatcherDetectsFileModification(t *testing.T) {
	root := t.TempDir()

	// Create a file first
	testFile := filepath.Join(root, "existing.txt")
	if err := os.WriteFile(testFile, []byte("initial"), 0o644); err != nil {
		t.Fatal(err)
	}

	var changes []FileChange
	var mu sync.Mutex
	changeReceived := make(chan struct{}, 1)

	cfg := DefaultConfig()
	cfg.Debounce = 100 * time.Millisecond

	watcher, err := New(root, config.Guardrails{}, func(c []FileChange) error {
		mu.Lock()
		changes = append(changes, c...)
		mu.Unlock()
		select {
		case changeReceived <- struct{}{}:
		default:
		}
		return nil
	}, cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start watcher in background
	go func() {
		watcher.Start(ctx)
	}()

	// Give watcher time to initialize
	time.Sleep(200 * time.Millisecond)

	// Modify the file
	if err := os.WriteFile(testFile, []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for change notification
	select {
	case <-changeReceived:
		// Success
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for file change notification")
	}

	mu.Lock()
	defer mu.Unlock()

	if len(changes) == 0 {
		t.Fatal("expected at least one change")
	}

	// Check the change
	found := false
	for _, c := range changes {
		if c.Path == "existing.txt" && c.Action == "modify" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected modify change for existing.txt, got: %+v", changes)
	}
}

func TestWatcherIgnoresPatterns(t *testing.T) {
	root := t.TempDir()

	// Create directories that should be ignored
	gitDir := filepath.Join(root, ".git")
	os.MkdirAll(gitDir, 0o755)
	nodeModules := filepath.Join(root, "node_modules")
	os.MkdirAll(nodeModules, 0o755)

	var changeCount int32

	cfg := DefaultConfig()
	cfg.Debounce = 50 * time.Millisecond

	watcher, err := New(root, config.Guardrails{}, func(c []FileChange) error {
		atomic.AddInt32(&changeCount, int32(len(c)))
		return nil
	}, cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Start watcher in background
	go func() {
		watcher.Start(ctx)
	}()

	// Give watcher time to initialize
	time.Sleep(200 * time.Millisecond)

	// Create files in ignored directories
	os.WriteFile(filepath.Join(gitDir, "config"), []byte("test"), 0o644)
	os.WriteFile(filepath.Join(nodeModules, "pkg.js"), []byte("test"), 0o644)

	// Wait for potential events
	time.Sleep(300 * time.Millisecond)

	// Should not have received any changes
	if atomic.LoadInt32(&changeCount) > 0 {
		t.Errorf("expected no changes for ignored paths, got %d", changeCount)
	}
}

func TestWatcherDebouncing(t *testing.T) {
	root := t.TempDir()

	var batchCount int32
	var totalChanges int32
	batchReceived := make(chan struct{}, 10)

	cfg := DefaultConfig()
	cfg.Debounce = 200 * time.Millisecond

	watcher, err := New(root, config.Guardrails{}, func(c []FileChange) error {
		atomic.AddInt32(&batchCount, 1)
		atomic.AddInt32(&totalChanges, int32(len(c)))
		batchReceived <- struct{}{}
		return nil
	}, cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start watcher in background
	go func() {
		watcher.Start(ctx)
	}()

	// Give watcher time to initialize
	time.Sleep(100 * time.Millisecond)

	// Create multiple files rapidly (should be debounced into one batch)
	for i := 0; i < 5; i++ {
		testFile := filepath.Join(root, "file"+string(rune('a'+i))+".txt")
		os.WriteFile(testFile, []byte("content"), 0o644)
		time.Sleep(20 * time.Millisecond) // Faster than debounce
	}

	// Wait for debounced notification
	select {
	case <-batchReceived:
		// First batch received
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for debounced notification")
	}

	// Wait a bit more to ensure no more batches
	time.Sleep(500 * time.Millisecond)

	batches := atomic.LoadInt32(&batchCount)
	changes := atomic.LoadInt32(&totalChanges)

	// Should have received 1-2 batches (depending on timing)
	if batches > 2 {
		t.Errorf("expected 1-2 batches due to debouncing, got %d", batches)
	}

	// Should have all changes
	if changes < 5 {
		t.Errorf("expected at least 5 changes, got %d", changes)
	}
}

func TestWatcherStats(t *testing.T) {
	root := t.TempDir()

	// Create subdirectories
	os.MkdirAll(filepath.Join(root, "src"), 0o755)
	os.MkdirAll(filepath.Join(root, "lib"), 0o755)

	watcher, err := New(root, config.Guardrails{}, nil, DefaultConfig())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Add watches manually to test stats
	watcher.addWatchRecursive(root)

	stats := watcher.Stats()

	// Should be watching at least 3 directories (root, src, lib)
	if stats.WatchedDirs < 3 {
		t.Errorf("expected at least 3 watched dirs, got %d", stats.WatchedDirs)
	}

	watcher.Stop()
}

func TestWatcherGracefulStop(t *testing.T) {
	root := t.TempDir()

	watcher, err := New(root, config.Guardrails{}, nil, DefaultConfig())
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start watcher in background
	done := make(chan struct{})
	go func() {
		watcher.Start(ctx)
		close(done)
	}()

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Should exit cleanly within timeout
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not stop within timeout")
	}
}

func TestWatcherHandlesNewDirectories(t *testing.T) {
	root := t.TempDir()

	var changes []FileChange
	var mu sync.Mutex
	changeReceived := make(chan struct{}, 1)

	cfg := DefaultConfig()
	cfg.Debounce = 100 * time.Millisecond

	watcher, err := New(root, config.Guardrails{}, func(c []FileChange) error {
		mu.Lock()
		changes = append(changes, c...)
		mu.Unlock()
		select {
		case changeReceived <- struct{}{}:
		default:
		}
		return nil
	}, cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start watcher in background
	go func() {
		watcher.Start(ctx)
	}()

	// Give watcher time to initialize
	time.Sleep(200 * time.Millisecond)

	// Create a new directory and file
	newDir := filepath.Join(root, "newdir")
	os.MkdirAll(newDir, 0o755)
	time.Sleep(100 * time.Millisecond) // Let watcher add the new dir

	testFile := filepath.Join(newDir, "file.txt")
	os.WriteFile(testFile, []byte("content"), 0o644)

	// Wait for change notification
	select {
	case <-changeReceived:
		// Success
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for file change in new directory")
	}

	mu.Lock()
	defer mu.Unlock()

	// Should have detected changes for the new directory or file
	if len(changes) == 0 {
		t.Fatal("expected at least one change")
	}
}
