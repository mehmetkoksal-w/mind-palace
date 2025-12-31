package flags

import (
	"testing"
)

func TestBoolFlag(t *testing.T) {
	var f BoolFlag

	// Test default state
	if f.Value != false || f.WasSet != false {
		t.Fatalf("expected default false/unset, got value=%v set=%v", f.Value, f.WasSet)
	}

	// Test setting to true
	if err := f.Set("true"); err != nil {
		t.Fatalf("unexpected error setting true: %v", err)
	}
	if !f.Value || !f.WasSet {
		t.Fatalf("expected true/set after Set(true), got value=%v set=%v", f.Value, f.WasSet)
	}

	// Test setting to false
	f = BoolFlag{}
	if err := f.Set("false"); err != nil {
		t.Fatalf("unexpected error setting false: %v", err)
	}
	if f.Value || !f.WasSet {
		t.Fatalf("expected false/set after Set(false), got value=%v set=%v", f.Value, f.WasSet)
	}

	// Test empty string (boolean flag without value)
	f = BoolFlag{}
	if err := f.Set(""); err != nil {
		t.Fatalf("unexpected error setting empty: %v", err)
	}
	if !f.Value || !f.WasSet {
		t.Fatalf("expected true/set after Set(\"\"), got value=%v set=%v", f.Value, f.WasSet)
	}

	// Test invalid value
	f = BoolFlag{}
	if err := f.Set("invalid"); err == nil {
		t.Fatal("expected error for invalid value")
	}

	// Test String()
	f = BoolFlag{Value: true}
	if f.String() != "true" {
		t.Fatalf("expected String()=\"true\", got %q", f.String())
	}
	f = BoolFlag{Value: false}
	if f.String() != "false" {
		t.Fatalf("expected String()=\"false\", got %q", f.String())
	}

	// Test IsBoolFlag
	if !f.IsBoolFlag() {
		t.Fatal("expected IsBoolFlag() to return true")
	}
}

func TestBoolFlagAdditionalCases(t *testing.T) {
	// Test "1" value
	f := BoolFlag{}
	if err := f.Set("1"); err != nil {
		t.Errorf("Set(\"1\") error: %v", err)
	}
	if !f.Value {
		t.Error("Set(\"1\") should set value to true")
	}

	// Test "0" value
	f = BoolFlag{}
	if err := f.Set("0"); err != nil {
		t.Errorf("Set(\"0\") error: %v", err)
	}
	if f.Value {
		t.Error("Set(\"0\") should set value to false")
	}

	// Test "TRUE" (uppercase)
	f = BoolFlag{}
	if err := f.Set("TRUE"); err != nil {
		t.Errorf("Set(\"TRUE\") error: %v", err)
	}
	if !f.Value {
		t.Error("Set(\"TRUE\") should set value to true")
	}

	// Test "FALSE" (uppercase)
	f = BoolFlag{}
	if err := f.Set("FALSE"); err != nil {
		t.Errorf("Set(\"FALSE\") error: %v", err)
	}
	if f.Value {
		t.Error("Set(\"FALSE\") should set value to false")
	}
}
