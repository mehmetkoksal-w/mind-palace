package commands

import (
	"testing"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

func TestRunBriefInvalidFlag(t *testing.T) {
	err := RunBrief([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestRunBriefSuccess(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	// Test via RunBrief with --root flag
	err = RunBrief([]string{"--root", root})
	if err != nil {
		t.Fatalf("RunBrief() error: %v", err)
	}

	err = ExecuteBrief(BriefOptions{
		Root: root,
	})
	if err != nil {
		t.Fatalf("ExecuteBrief() error: %v", err)
	}
}

func TestRunBriefWithFileArg(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	// Test with file argument
	err = RunBrief([]string{"--root", root, "main.go"})
	if err != nil {
		t.Fatalf("RunBrief(file) error: %v", err)
	}
}

func TestRunBriefWithFile(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	err = ExecuteBrief(BriefOptions{
		Root:     root,
		FilePath: "main.go",
	})
	if err != nil {
		t.Fatalf("ExecuteBrief(file) error: %v", err)
	}
}

func TestRunBriefWithData(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	mem, err := memory.Open(root)
	if err != nil {
		t.Fatalf("memory.Open() error: %v", err)
	}
	defer mem.Close()

	// Add an idea
	if _, err := mem.AddIdea(memory.Idea{
		Content: "test idea",
		Scope:   "palace",
		Source:  "user",
	}); err != nil {
		t.Fatalf("AddIdea() error: %v", err)
	}

	// Add a decision (active)
	if _, err := mem.AddDecision(memory.Decision{
		Content: "active decision",
		Status:  "proposed",
		Scope:   "palace",
		Source:  "user",
	}); err != nil {
		t.Fatalf("AddDecision() error: %v", err)
	}

	// Add a hotspot
	for i := 0; i < 5; i++ {
		if err := mem.RecordFileEdit("main.go", "cli"); err != nil {
			t.Fatalf("RecordFileEdit() error: %v", err)
		}
	}

	err = ExecuteBrief(BriefOptions{
		Root: root,
	})
	if err != nil {
		t.Fatalf("ExecuteBrief() error: %v", err)
	}
}
