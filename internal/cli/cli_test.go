package cli

import (
	"testing"

	"github.com/koksalmehmet/mind-palace/internal/verify"
)

func TestResolveVerifyModeConflicts(t *testing.T) {
	var fast, strict boolFlag
	_ = fast.Set("true")
	_ = strict.Set("true")
	if _, err := resolveVerifyMode(fast, strict); err == nil {
		t.Fatalf("expected conflict when both true")
	}

	var fastFalse, strictTrue boolFlag
	_ = fastFalse.Set("false")
	_ = strictTrue.Set("true")
	if mode, err := resolveVerifyMode(fastFalse, strictTrue); err != nil || mode != verify.ModeStrict {
		t.Fatalf("expected strict when fast=false strict=true, got mode %v err %v", mode, err)
	}

	var fastFalseOnly boolFlag
	_ = fastFalseOnly.Set("false")
	if _, err := resolveVerifyMode(fastFalseOnly, boolFlag{}); err == nil {
		t.Fatalf("expected error when fast=false without strict=true")
	}

	mode, err := resolveVerifyMode(boolFlag{}, boolFlag{})
	if err != nil || mode != verify.ModeFast {
		t.Fatalf("expected default fast mode, got %v err %v", mode, err)
	}
}
