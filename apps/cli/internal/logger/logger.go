// Package logger provides a simple verbose logging utility for the CLI.
package logger

import (
	"fmt"
	"os"
	"time"
)

// Level represents the logging level
type Level int

const (
	// LevelOff disables all logging
	LevelOff Level = iota
	// LevelInfo shows basic progress information
	LevelInfo
	// LevelDebug shows detailed debugging information
	LevelDebug
)

var (
	currentLevel = LevelOff
	startTime    = time.Now()
)

// SetLevel sets the global logging level
func SetLevel(level Level) {
	currentLevel = level
	startTime = time.Now()
}

// GetLevel returns the current logging level
func GetLevel() Level {
	return currentLevel
}

// IsVerbose returns true if verbose logging is enabled
func IsVerbose() bool {
	return currentLevel >= LevelInfo
}

// IsDebug returns true if debug logging is enabled
func IsDebug() bool {
	return currentLevel >= LevelDebug
}

// Info logs an informational message (shown with --verbose)
func Info(format string, args ...interface{}) {
	if currentLevel >= LevelInfo {
		elapsed := time.Since(startTime).Round(time.Millisecond)
		prefix := fmt.Sprintf("[%s] ", elapsed)
		fmt.Fprintf(os.Stderr, prefix+format+"\n", args...)
	}
}

// Debug logs a debug message (shown with --debug)
func Debug(format string, args ...interface{}) {
	if currentLevel >= LevelDebug {
		elapsed := time.Since(startTime).Round(time.Millisecond)
		prefix := fmt.Sprintf("[%s] [DEBUG] ", elapsed)
		fmt.Fprintf(os.Stderr, prefix+format+"\n", args...)
	}
}

// Error logs an error message (always shown when verbose is on)
func Error(format string, args ...interface{}) {
	if currentLevel >= LevelInfo {
		elapsed := time.Since(startTime).Round(time.Millisecond)
		prefix := fmt.Sprintf("[%s] [ERROR] ", elapsed)
		fmt.Fprintf(os.Stderr, prefix+format+"\n", args...)
	}
}
