package butler

import (
	"path/filepath"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// truncateSnippet truncates a string to maxLen and adds an indicator if truncated.
func truncateSnippet(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}

// sanitizePath cleans a file path and prevents path traversal attacks.
// Returns empty string if the path is invalid or attempts to escape workspace.
func sanitizePath(path string) string {
	// Reject Unix-style absolute paths before cleaning (Windows converts / to \)
	if strings.HasPrefix(path, "/") {
		return ""
	}

	// Clean the path to normalize . and .. elements
	clean := filepath.Clean(path)

	// Reject Windows-style absolute paths
	if filepath.IsAbs(clean) {
		return ""
	}

	// Reject paths that try to escape (start with ..)
	if strings.HasPrefix(clean, "..") {
		return ""
	}

	// Reject paths containing .. anywhere (even after clean)
	if strings.Contains(clean, "..") {
		return ""
	}

	return clean
}

// inferKindFromID infers the record kind from its ID prefix
func inferKindFromID(id string) string {
	if strings.HasPrefix(id, "i_") {
		return memory.TargetKindIdea
	}
	if strings.HasPrefix(id, "d_") {
		return memory.TargetKindDecision
	}
	if strings.HasPrefix(id, "l_") {
		return memory.TargetKindLearning
	}
	if strings.HasPrefix(id, "http://") || strings.HasPrefix(id, "https://") {
		return memory.TargetKindURL
	}
	// Assume code if it looks like a file path
	if strings.Contains(id, "/") || strings.Contains(id, ".") {
		return memory.TargetKindCode
	}
	return "unknown"
}
