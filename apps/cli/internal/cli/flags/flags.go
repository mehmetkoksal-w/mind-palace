// Package flags provides common flag types and validators for the CLI.
package flags

import (
	"fmt"
	"strings"
)

// BoolFlag is a boolean flag that tracks whether it was explicitly set.
// This is useful for differentiating between an unset flag and a flag explicitly set to false.
type BoolFlag struct {
	Value  bool
	WasSet bool
}

// Set parses and sets the boolean value.
func (b *BoolFlag) Set(s string) error {
	if s == "" {
		b.Value = true
		b.WasSet = true
		return nil
	}
	switch strings.ToLower(s) {
	case "true", "1":
		b.Value = true
	case "false", "0":
		b.Value = false
	default:
		return fmt.Errorf("invalid boolean %q", s)
	}
	b.WasSet = true
	return nil
}

// String returns the string representation of the boolean value.
func (b *BoolFlag) String() string {
	if b.Value {
		return "true"
	}
	return "false"
}

// IsBoolFlag returns true, indicating this is a boolean flag that doesn't require a value.
func (b *BoolFlag) IsBoolFlag() bool { return true }
