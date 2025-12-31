package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunInitInvalidFlag(t *testing.T) {
	err := RunInit([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

func TestExecuteInitCreatesLayout(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{
		Root:        root,
		Force:       false,
		WithOutputs: false,
		Detect:      false,
	})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	expected := []string{
		filepath.Join(root, ".palace", "palace.jsonc"),
		filepath.Join(root, ".palace", "rooms", "project-overview.jsonc"),
		filepath.Join(root, ".palace", "playbooks", "default.jsonc"),
		filepath.Join(root, ".palace", "project-profile.json"),
	}
	for _, path := range expected {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s to exist: %v", path, err)
		}
	}
}

func TestExecuteInitWithOutputs(t *testing.T) {
	root := t.TempDir()

	err := ExecuteInit(InitOptions{
		Root:        root,
		Force:       false,
		WithOutputs: true,
		Detect:      false,
	})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	cpPath := filepath.Join(root, ".palace", "outputs", "context-pack.json")
	if _, err := os.Stat(cpPath); err != nil {
		t.Fatalf("expected context-pack.json to exist: %v", err)
	}
}

func TestExecuteInitWithDetect(t *testing.T) {
	root := t.TempDir()

	// Create a Go file to detect
	goFile := filepath.Join(root, "main.go")
	os.WriteFile(goFile, []byte("package main\n"), 0644)

	err := ExecuteInit(InitOptions{
		Root:   root,
		Detect: true,
	})
	if err != nil {
		t.Fatalf("ExecuteInit() error: %v", err)
	}

	profilePath := filepath.Join(root, ".palace", "project-profile.json")
	if _, err := os.Stat(profilePath); err != nil {
		t.Fatalf("expected project-profile.json to exist: %v", err)
	}
}

func TestExecuteInitForceOverwrite(t *testing.T) {
	root := t.TempDir()

	// First init
	err := ExecuteInit(InitOptions{Root: root})
	if err != nil {
		t.Fatalf("First ExecuteInit() error: %v", err)
	}

	// Second init with force
	err = ExecuteInit(InitOptions{Root: root, Force: true})
	if err != nil {
		t.Fatalf("Second ExecuteInit() error: %v", err)
	}
}
