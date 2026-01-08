// Package util provides utility functions for the CLI.
package util

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/model"
)

// MustAbs returns the absolute path, or the original path if resolution fails.
func MustAbs(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

// ScopeFileCount returns the file count from a context pack's scope.
func ScopeFileCount(cp model.ContextPack) int {
	if cp.Scope == nil {
		return 0
	}
	return cp.Scope.FileCount
}

// TruncateLine truncates a string to maxLen characters, appending "..." if truncated.
func TruncateLine(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// PrintScope prints scope information for a command.
func PrintScope(cmd string, fullScope bool, source, diffRange string, fileCount int, rootPath string) {
	mode := "diff"
	if fullScope {
		mode = "full"
	}
	if source == "" {
		if fullScope {
			source = "full-scan"
		} else {
			source = "git-diff/change-signal"
		}
	}
	fmt.Printf("Scope (%s):\n", cmd)
	fmt.Printf("  root: %s\n", rootPath)
	fmt.Printf("  mode: %s\n", mode)
	fmt.Printf("  source: %s\n", source)
	fmt.Printf("  fileCount: %d\n", fileCount)
	if !fullScope {
		fmt.Printf("  diffRange: %s\n", strings.TrimSpace(diffRange))
	}
}
