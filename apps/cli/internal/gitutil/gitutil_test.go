package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIsGitRepo(t *testing.T) {
	// Create a temp dir without git
	noGitDir := t.TempDir()
	if IsGitRepo(noGitDir) {
		t.Error("expected non-git dir to return false")
	}

	// Create a temp dir with git
	gitDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = gitDir
	if err := cmd.Run(); err != nil {
		t.Skip("git not available")
	}
	if !IsGitRepo(gitDir) {
		t.Error("expected git dir to return true")
	}
}

func TestGetHeadCommit(t *testing.T) {
	dir := t.TempDir()

	// Init repo
	if err := exec.Command("git", "-C", dir, "init").Run(); err != nil {
		t.Skip("git not available")
	}

	// Configure git user for commits
	exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test").Run()

	// No commits yet should fail
	_, err := GetHeadCommit(dir)
	if err == nil {
		t.Error("expected error for repo with no commits")
	}

	// Create a file and commit
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "initial").Run()

	// Now should succeed
	hash, err := GetHeadCommit(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hash) != 40 {
		t.Errorf("expected 40-char hash, got %d chars", len(hash))
	}
}

func TestIsValidCommit(t *testing.T) {
	dir := t.TempDir()

	// Init repo
	if err := exec.Command("git", "-C", dir, "init").Run(); err != nil {
		t.Skip("git not available")
	}

	// Configure git user
	exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test").Run()

	// Create commit
	testFile := filepath.Join(dir, "test.txt")
	os.WriteFile(testFile, []byte("hello"), 0o644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "initial").Run()

	hash, _ := GetHeadCommit(dir)

	// Valid commit should return true
	if !IsValidCommit(dir, hash) {
		t.Error("expected valid commit to return true")
	}

	// Invalid commit should return false
	if IsValidCommit(dir, "0000000000000000000000000000000000000000") {
		t.Error("expected invalid commit to return false")
	}
}

func TestGetChangedFiles(t *testing.T) {
	dir := t.TempDir()

	// Init repo
	if err := exec.Command("git", "-C", dir, "init").Run(); err != nil {
		t.Skip("git not available")
	}

	// Configure git user
	exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test").Run()

	// Create initial commit
	file1 := filepath.Join(dir, "file1.txt")
	os.WriteFile(file1, []byte("hello"), 0o644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "initial").Run()
	baseCommit, _ := GetHeadCommit(dir)

	// Add a new file
	file2 := filepath.Join(dir, "file2.txt")
	os.WriteFile(file2, []byte("world"), 0o644)

	// Modify existing file
	os.WriteFile(file1, []byte("hello modified"), 0o644)

	// Commit changes
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "second").Run()
	headCommit, _ := GetHeadCommit(dir)

	added, modified, deleted, err := GetChangedFiles(dir, baseCommit, headCommit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(added) != 1 || added[0] != "file2.txt" {
		t.Errorf("expected file2.txt added, got %v", added)
	}

	if len(modified) != 1 || modified[0] != "file1.txt" {
		t.Errorf("expected file1.txt modified, got %v", modified)
	}

	if len(deleted) != 0 {
		t.Errorf("expected no deleted files, got %v", deleted)
	}
}

func TestGetChangedFiles_EmptyBase(t *testing.T) {
	dir := t.TempDir()

	// Init repo
	if err := exec.Command("git", "-C", dir, "init").Run(); err != nil {
		t.Skip("git not available")
	}

	// Configure git user
	exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test").Run()

	// Create files
	file1 := filepath.Join(dir, "file1.txt")
	file2 := filepath.Join(dir, "file2.txt")
	os.WriteFile(file1, []byte("hello"), 0o644)
	os.WriteFile(file2, []byte("world"), 0o644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "initial").Run()
	headCommit, _ := GetHeadCommit(dir)

	// Empty base should return all files as added
	added, modified, deleted, err := GetChangedFiles(dir, "", headCommit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(added) != 2 {
		t.Errorf("expected 2 added files, got %d", len(added))
	}
	if len(modified) != 0 {
		t.Errorf("expected no modified files, got %d", len(modified))
	}
	if len(deleted) != 0 {
		t.Errorf("expected no deleted files, got %d", len(deleted))
	}
}

func TestIsDirtyWorkingTree(t *testing.T) {
	dir := t.TempDir()

	// Init repo
	if err := exec.Command("git", "-C", dir, "init").Run(); err != nil {
		t.Skip("git not available")
	}

	// Configure git user
	exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "Test").Run()

	// Create initial commit
	file1 := filepath.Join(dir, "file1.txt")
	os.WriteFile(file1, []byte("hello"), 0o644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "initial").Run()

	// Clean working tree
	if IsDirtyWorkingTree(dir) {
		t.Error("expected clean working tree")
	}

	// Make a change
	os.WriteFile(file1, []byte("hello modified"), 0o644)

	// Dirty working tree
	if !IsDirtyWorkingTree(dir) {
		t.Error("expected dirty working tree")
	}
}
