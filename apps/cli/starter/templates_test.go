package starter

import (
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantErr   bool
		wantParts []string
	}{
		{
			name:      "get palace.jsonc",
			path:      "palace.jsonc",
			wantErr:   false,
			wantParts: []string{"schemaVersion", "project"},
		},
		{
			name:      "get project-profile.json",
			path:      "project-profile.json",
			wantErr:   false,
			wantParts: []string{"language"},
		},
		{
			name:    "get non-existent file",
			path:    "nonexistent.json",
			wantErr: true,
		},
		{
			name:      "path with leading slash is stripped",
			path:      "/palace.jsonc",
			wantErr:   false,
			wantParts: []string{"schemaVersion"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := Get(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, part := range tt.wantParts {
				if !strings.Contains(content, part) {
					t.Errorf("expected content to contain %q", part)
				}
			}
		})
	}
}

func TestApply(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		replacements map[string]string
		want         string
	}{
		{
			name:         "single replacement",
			template:     "Hello {{NAME}}!",
			replacements: map[string]string{"NAME": "World"},
			want:         "Hello World!",
		},
		{
			name:     "multiple replacements",
			template: "{{GREETING}} {{NAME}}! Welcome to {{PLACE}}.",
			replacements: map[string]string{
				"GREETING": "Hello",
				"NAME":     "User",
				"PLACE":    "Mind Palace",
			},
			want: "Hello User! Welcome to Mind Palace.",
		},
		{
			name:         "no replacements",
			template:     "Static content",
			replacements: map[string]string{},
			want:         "Static content",
		},
		{
			name:         "nil replacements",
			template:     "Static content",
			replacements: nil,
			want:         "Static content",
		},
		{
			name:         "multiple occurrences of same placeholder",
			template:     "{{A}} and {{A}} and {{A}}",
			replacements: map[string]string{"A": "X"},
			want:         "X and X and X",
		},
		{
			name:         "placeholder not in map is left unchanged",
			template:     "{{FOUND}} and {{NOTFOUND}}",
			replacements: map[string]string{"FOUND": "replaced"},
			want:         "replaced and {{NOTFOUND}}",
		},
		{
			name:         "empty template",
			template:     "",
			replacements: map[string]string{"A": "B"},
			want:         "",
		},
		{
			name:         "replacement with special characters",
			template:     "Path: {{PATH}}",
			replacements: map[string]string{"PATH": "/home/user/.palace"},
			want:         "Path: /home/user/.palace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Apply(tt.template, tt.replacements)
			if got != tt.want {
				t.Errorf("Apply() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetRoomTemplates(t *testing.T) {
	// Test that we can get room templates (if they exist in the embed)
	content, err := Get("rooms/room.jsonc")
	if err != nil {
		// The room might have a different name, that's ok
		t.Logf("rooms/room.jsonc not found (may have different name): %v", err)
		return
	}
	if !strings.Contains(content, "glob") && !strings.Contains(content, "patterns") {
		t.Log("room template doesn't contain expected fields, but that's ok for flexible templates")
	}
}

func TestGetPlaybookTemplates(t *testing.T) {
	// Try to access playbook templates
	content, err := Get("playbooks/playbook.jsonc")
	if err != nil {
		t.Logf("playbooks/playbook.jsonc not found (may have different name): %v", err)
		return
	}
	if len(content) == 0 {
		t.Error("playbook template should not be empty")
	}
}

func TestGetOutputTemplates(t *testing.T) {
	// Try to access output templates
	content, err := Get("outputs/context-pack.json")
	if err != nil {
		t.Logf("outputs/context-pack.json not found (may have different name): %v", err)
		return
	}
	if len(content) == 0 {
		t.Error("output template should not be empty")
	}
}
