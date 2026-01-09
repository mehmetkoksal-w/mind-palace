// Package gitutil provides utilities for interacting with git repositories.
package gitutil

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// IsGitRepo checks if the given path is inside a git repository.
func IsGitRepo(root string) bool {
	cmd := exec.Command("git", "-C", root, "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}

// GetHeadCommit returns the current HEAD commit hash.
func GetHeadCommit(root string) (string, error) {
	cmd := exec.Command("git", "-C", root, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// IsValidCommit checks if a commit hash exists in the repository.
func IsValidCommit(root, commit string) bool {
	cmd := exec.Command("git", "-C", root, "cat-file", "-t", commit)
	err := cmd.Run()
	return err == nil
}

// GetChangedFiles returns files that have changed between two commits.
// If baseCommit is empty, returns all tracked files.
// Returns relative paths from the repository root.
func GetChangedFiles(root, baseCommit, headCommit string) (added, modified, deleted []string, err error) {
	if baseCommit == "" {
		// No base commit - return all tracked files as "added"
		return getAllTrackedFiles(root)
	}

	// Check if baseCommit still exists (handles rebases)
	if !IsValidCommit(root, baseCommit) {
		// Base commit is no longer valid (possibly rebased)
		// Fall back to returning all tracked files
		return getAllTrackedFiles(root)
	}

	// Get diff between commits
	cmd := exec.Command("git", "-C", root, "diff", "--name-status", baseCommit, headCommit)
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		path := parts[1]

		// Handle renamed files (R100 old new)
		if strings.HasPrefix(status, "R") && len(parts) >= 3 {
			deleted = append(deleted, parts[1])
			added = append(added, parts[2])
			continue
		}

		switch status[0] {
		case 'A':
			added = append(added, path)
		case 'M':
			modified = append(modified, path)
		case 'D':
			deleted = append(deleted, path)
		}
	}

	return added, modified, deleted, nil
}

// GetChangedFilesSinceCommit returns files that have changed since a specific commit,
// including both committed and uncommitted changes.
func GetChangedFilesSinceCommit(root, baseCommit string) (added, modified, deleted []string, err error) {
	// Get changes between base commit and HEAD
	headCommit, err := GetHeadCommit(root)
	if err != nil {
		return nil, nil, nil, err
	}

	added, modified, deleted, err = GetChangedFiles(root, baseCommit, headCommit)
	if err != nil {
		return nil, nil, nil, err
	}

	// Also get uncommitted changes (both staged and unstaged)
	uncommittedAdded, uncommittedModified, uncommittedDeleted, err := getUncommittedChanges(root)
	if err != nil {
		// Ignore errors for uncommitted changes
		return added, modified, deleted, nil
	}

	// Merge uncommitted changes (using a set to avoid duplicates)
	addedSet := make(map[string]bool)
	modifiedSet := make(map[string]bool)
	deletedSet := make(map[string]bool)

	for _, f := range added {
		addedSet[f] = true
	}
	for _, f := range modified {
		modifiedSet[f] = true
	}
	for _, f := range deleted {
		deletedSet[f] = true
	}

	// Add uncommitted changes
	for _, f := range uncommittedAdded {
		if !addedSet[f] && !modifiedSet[f] {
			addedSet[f] = true
		}
	}
	for _, f := range uncommittedModified {
		if !addedSet[f] && !modifiedSet[f] {
			modifiedSet[f] = true
		}
	}
	for _, f := range uncommittedDeleted {
		if !deletedSet[f] {
			deletedSet[f] = true
			// Remove from added/modified if now deleted
			delete(addedSet, f)
			delete(modifiedSet, f)
		}
	}

	// Convert back to slices
	added = make([]string, 0, len(addedSet))
	for f := range addedSet {
		added = append(added, f)
	}
	modified = make([]string, 0, len(modifiedSet))
	for f := range modifiedSet {
		modified = append(modified, f)
	}
	deleted = make([]string, 0, len(deletedSet))
	for f := range deletedSet {
		deleted = append(deleted, f)
	}

	return added, modified, deleted, nil
}

// getUncommittedChanges returns files with uncommitted changes (staged + unstaged).
func getUncommittedChanges(root string) (added, modified, deleted []string, err error) {
	// Get status of all changed files
	cmd := exec.Command("git", "-C", root, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		status := line[:2]
		path := strings.TrimSpace(line[3:])

		// Handle paths with -> for renames
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			if len(parts) == 2 {
				deleted = append(deleted, parts[0])
				added = append(added, parts[1])
				continue
			}
		}

		// First char is index status, second is working tree status
		indexStatus := status[0]
		worktreeStatus := status[1]

		// Determine the effective status
		if indexStatus == '?' || worktreeStatus == '?' {
			// Untracked file
			added = append(added, path)
		} else if indexStatus == 'D' || worktreeStatus == 'D' {
			deleted = append(deleted, path)
		} else if indexStatus == 'A' {
			added = append(added, path)
		} else if indexStatus == 'M' || worktreeStatus == 'M' {
			modified = append(modified, path)
		}
	}

	return added, modified, deleted, nil
}

// getAllTrackedFiles returns all files currently tracked by git.
func getAllTrackedFiles(root string) (added, modified, deleted []string, err error) {
	cmd := exec.Command("git", "-C", root, "ls-files")
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line != "" {
			added = append(added, line)
		}
	}

	return added, nil, nil, nil
}

// GetRepoRoot returns the root directory of the git repository.
func GetRepoRoot(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("git", "-C", absPath, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

// IsDirtyWorkingTree checks if there are uncommitted changes.
func IsDirtyWorkingTree(root string) bool {
	cmd := exec.Command("git", "-C", root, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}
