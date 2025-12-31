package commands

import (
	"testing"
)

func TestRunUpdateInvalidFlag(t *testing.T) {
	err := RunUpdate([]string{"--invalid-flag"})
	if err == nil {
		t.Error("expected error for invalid flag")
	}
}

// Note: ExecuteUpdate is not easily testable without mocking the update package
// which would require dependency injection. For now, we test the flag parsing.
