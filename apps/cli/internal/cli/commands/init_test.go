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
		SkipDetect:  true,
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
		SkipDetect:  true,
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
	os.WriteFile(goFile, []byte("package main\n"), 0o644)
	goMod := filepath.Join(root, "go.mod")
	os.WriteFile(goMod, []byte("module test\n"), 0o644)

	// Detection is now the default behavior (SkipDetect: false)
	err := ExecuteInit(InitOptions{
		Root: root,
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

func TestUpdateGitignore(t *testing.T) {
	root := t.TempDir()

	// Test creating new .gitignore
	t.Run("creates new gitignore", func(t *testing.T) {
		err := updateGitignore(root, false)
		if err != nil {
			t.Fatalf("updateGitignore() error: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(root, ".gitignore"))
		if err != nil {
			t.Fatalf("reading .gitignore: %v", err)
		}

		expected := []string{
			"# Mind Palace",
			".palace/scan/",
			".palace/outputs/",
			".palace/cache/",
			".palace/sessions/",
		}

		for _, entry := range expected {
			if !contains(string(content), entry) {
				t.Errorf("expected .gitignore to contain %q", entry)
			}
		}
	})

	// Test skipping if already configured
	t.Run("skips if already configured", func(t *testing.T) {
		// updateGitignore was already called in previous subtest
		originalContent, _ := os.ReadFile(filepath.Join(root, ".gitignore"))

		err := updateGitignore(root, false)
		if err != nil {
			t.Fatalf("updateGitignore() error: %v", err)
		}

		newContent, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
		if string(originalContent) != string(newContent) {
			t.Error("expected .gitignore to be unchanged when already configured")
		}
	})
}

func TestUpdateGitignoreAppendsToExisting(t *testing.T) {
	root := t.TempDir()

	// Create existing .gitignore
	existingContent := "node_modules/\n.env\n"
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(existingContent), 0o644); err != nil {
		t.Fatal(err)
	}

	err := updateGitignore(root, false)
	if err != nil {
		t.Fatalf("updateGitignore() error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	contentStr := string(content)

	// Should preserve existing content
	if !contains(contentStr, "node_modules/") {
		t.Error("expected .gitignore to preserve node_modules/")
	}
	if !contains(contentStr, ".env") {
		t.Error("expected .gitignore to preserve .env")
	}

	// Should add Mind Palace entries
	if !contains(contentStr, "# Mind Palace") {
		t.Error("expected .gitignore to contain Mind Palace header")
	}
}

func TestInstallVSCodeIntegration(t *testing.T) {
	root := t.TempDir()

	// Test skipping when no .vscode directory
	t.Run("skips when no .vscode", func(t *testing.T) {
		err := installVSCodeIntegration(root, false)
		if err != nil {
			t.Fatalf("installVSCodeIntegration() error: %v", err)
		}

		// extensions.json should not be created
		if _, err := os.Stat(filepath.Join(root, ".vscode", "extensions.json")); !os.IsNotExist(err) {
			t.Error("expected extensions.json to not be created when .vscode doesn't exist")
		}
	})

	// Test creating extensions.json when .vscode exists
	t.Run("creates extensions.json when .vscode exists", func(t *testing.T) {
		vscodeDir := filepath.Join(root, ".vscode")
		os.MkdirAll(vscodeDir, 0o755)

		err := installVSCodeIntegration(root, false)
		if err != nil {
			t.Fatalf("installVSCodeIntegration() error: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(vscodeDir, "extensions.json"))
		if err != nil {
			t.Fatalf("reading extensions.json: %v", err)
		}

		if !contains(string(content), "mind-palace.vscode-mind-palace") {
			t.Error("expected extensions.json to contain mind-palace extension")
		}
	})
}

func TestInstallGitHooks(t *testing.T) {
	root := t.TempDir()

	// Test skipping when not a git repo
	t.Run("skips when not git repo", func(t *testing.T) {
		err := installGitHooks(root, false)
		if err != nil {
			t.Fatalf("installGitHooks() error: %v", err)
		}

		// hooks should not be created
		if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "post-commit")); !os.IsNotExist(err) {
			t.Error("expected hook to not be created when .git doesn't exist")
		}
	})

	// Test creating hook when .git exists
	t.Run("creates hook when git repo", func(t *testing.T) {
		gitDir := filepath.Join(root, ".git")
		os.MkdirAll(gitDir, 0o755)

		err := installGitHooks(root, false)
		if err != nil {
			t.Fatalf("installGitHooks() error: %v", err)
		}

		hookPath := filepath.Join(gitDir, "hooks", "post-commit")
		content, err := os.ReadFile(hookPath)
		if err != nil {
			t.Fatalf("reading post-commit hook: %v", err)
		}

		if !contains(string(content), "palace scan") {
			t.Error("expected hook to contain palace scan command")
		}

		// Check hook is executable (on Unix)
		// On Windows, we just check it exists
		if _, err := os.Stat(hookPath); err != nil {
			t.Error("expected hook file to exist")
		}
	})
}

func TestDetectInstalledAgents(t *testing.T) {
	root := t.TempDir()

	// Test detection with no agents
	t.Run("detects nothing when empty", func(t *testing.T) {
		detected := detectInstalledAgents(root)
		// Should be empty for local detection (global detection may find some)
		for _, d := range detected {
			if d.Confidence == "high" {
				t.Errorf("unexpected high-confidence detection: %s", d.Key)
			}
		}
	})

	// Test detection with .vscode
	t.Run("detects vscode", func(t *testing.T) {
		vscodeDir := filepath.Join(root, ".vscode")
		os.MkdirAll(vscodeDir, 0o755)

		detected := detectInstalledAgents(root)
		found := false
		for _, d := range detected {
			if d.Key == "vscode" && d.Confidence == "high" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected to detect vscode with high confidence")
		}
	})

	// Test detection with CLAUDE.md
	t.Run("detects claude-code", func(t *testing.T) {
		claudeFile := filepath.Join(root, "CLAUDE.md")
		os.WriteFile(claudeFile, []byte("# Claude Instructions\n"), 0o644)

		detected := detectInstalledAgents(root)
		found := false
		for _, d := range detected {
			if d.Key == "claude-code" && d.Confidence == "high" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected to detect claude-code with high confidence")
		}
	})

	// Test detection with .cursorrules
	t.Run("detects cursor", func(t *testing.T) {
		cursorFile := filepath.Join(root, ".cursorrules")
		os.WriteFile(cursorFile, []byte("# Cursor Rules\n"), 0o644)

		detected := detectInstalledAgents(root)
		found := false
		for _, d := range detected {
			if d.Key == "cursor" && d.Confidence == "high" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected to detect cursor with high confidence")
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
