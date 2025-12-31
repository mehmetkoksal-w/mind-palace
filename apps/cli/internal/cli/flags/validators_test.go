package flags

import (
	"testing"
)

func TestValidateConfidence(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		wantErr bool
	}{
		{"zero is valid", 0.0, false},
		{"one is valid", 1.0, false},
		{"mid-range is valid", 0.5, false},
		{"negative is invalid", -0.1, true},
		{"above one is invalid", 1.1, true},
		{"large negative is invalid", -10.0, true},
		{"large positive is invalid", 10.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfidence(tt.value)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateLimit(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"zero is valid", 0, false},
		{"positive is valid", 10, false},
		{"large positive is valid", 1000, false},
		{"negative is invalid", -1, true},
		{"large negative is invalid", -100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLimit(tt.value)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateScope(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"file is valid", "file", false},
		{"room is valid", "room", false},
		{"palace is valid", "palace", false},
		{"invalid scope", "invalid", true},
		{"empty is invalid", "", true},
		{"uppercase file is invalid", "FILE", true},
		{"folder is invalid", "folder", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateScope(tt.value)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"port 1 is valid", 1, false},
		{"port 80 is valid", 80, false},
		{"port 443 is valid", 443, false},
		{"port 8080 is valid", 8080, false},
		{"port 65535 is valid", 65535, false},
		{"port 0 is invalid", 0, true},
		{"negative port is invalid", -1, true},
		{"port above 65535 is invalid", 65536, true},
		{"large port is invalid", 100000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePort(tt.value)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
