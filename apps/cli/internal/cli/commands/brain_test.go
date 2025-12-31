package commands

import (
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

func TestRunStoreMissingContent(t *testing.T) {
	err := RunStore([]string{})
	if err == nil {
		t.Error("expected error for missing content")
	}
}

func TestRunStoreInvalidScope(t *testing.T) {
	err := RunStore([]string{"--scope", "invalid", "test content"})
	if err == nil {
		t.Error("expected error for invalid scope")
	}
}

func TestRunRecallUpdateInvalidFlag(t *testing.T) {
	err := RunRecall([]string{"update", "--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestRunRecallUpdateMissingArgs(t *testing.T) {
	err := RunRecall([]string{"update"})
	if err == nil {
		t.Error("expected error for missing arguments")
	}

	err = RunRecall([]string{"update", "d_123"})
	if err == nil {
		t.Error("expected error for missing outcome")
	}
}

func TestRunRecallUpdateInvalidOutcome(t *testing.T) {
	err := RunRecall([]string{"update", "d_123", "invalid"})
	if err == nil {
		t.Error("expected error for invalid outcome")
	}
}

func TestRunRecallInvalidLimit(t *testing.T) {
	err := RunRecall([]string{"--limit", "-1"})
	if err == nil {
		t.Error("expected error for negative limit")
	}
}

func TestRunRecallLinkInvalidFlag(t *testing.T) {
	err := RunRecall([]string{"link", "--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestRunRecallLinkMissingSourceID(t *testing.T) {
	err := RunRecall([]string{"link"})
	if err == nil {
		t.Error("expected error for missing source ID")
	}
}

func TestRunRecallLinkMissingRelation(t *testing.T) {
	err := RunRecall([]string{"link", "d_123"})
	if err == nil {
		t.Error("expected error for missing relation")
	}
}

func TestExecuteStoreSuccess(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	err = ExecuteStore(StoreOptions{
		Root:    root,
		Content: "Let's use JWT for authentication",
		Scope:   "palace",
	})
	if err != nil {
		t.Fatalf("ExecuteStore() error: %v", err)
	}
}

func TestExecuteStoreAsDecision(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	err = ExecuteStore(StoreOptions{
		Root:    root,
		Content: "We will use PostgreSQL",
		Scope:   "palace",
		AsType:  "decision",
	})
	if err != nil {
		t.Fatalf("ExecuteStore(decision) error: %v", err)
	}
}

func TestExecuteStoreAsIdea(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	err = ExecuteStore(StoreOptions{
		Root:    root,
		Content: "What if we add caching?",
		Scope:   "palace",
		AsType:  "idea",
	})
	if err != nil {
		t.Fatalf("ExecuteStore(idea) error: %v", err)
	}
}

func TestExecuteStoreAsLearning(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	err = ExecuteStore(StoreOptions{
		Root:    root,
		Content: "Always run tests before committing",
		Scope:   "palace",
		AsType:  "learning",
	})
	if err != nil {
		t.Fatalf("ExecuteStore(learning) error: %v", err)
	}
}

func TestRunRecallPendingNoDecisions(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	// Test recall --pending via RunRecall
	err = RunRecall([]string{"--root", root, "--pending"})
	if err != nil {
		t.Fatalf("RunRecall(--pending) error: %v", err)
	}
}

func TestInferTargetKind(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"d_123", "decision"},
		{"i_123", "idea"},
		{"l_123", "learning"},
		{"main.go", "code"},
		{"src/auth.go:15-20", "code"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		got := inferTargetKind(tt.id)
		if got != tt.want {
			t.Errorf("inferTargetKind(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestRunRecallLinkSuccess(t *testing.T) {
	root := t.TempDir()
	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error: %v", err)
	}
	d1ID, err := mem.AddDecision(memory.Decision{Content: "D1", Status: "proposed", Source: "user", Scope: "palace"})
	if err != nil {
		t.Fatalf("AddDecision() error: %v", err)
	}
	d2ID, err := mem.AddDecision(memory.Decision{Content: "D2", Status: "proposed", Source: "user", Scope: "palace"})
	if err != nil {
		t.Fatalf("AddDecision() error: %v", err)
	}
	mem.Close()

	err = RunRecall([]string{"link", "--root", root, "--supersedes", d1ID, d2ID})
	if err != nil {
		t.Fatalf("RunRecall(link) error: %v", err)
	}
}
