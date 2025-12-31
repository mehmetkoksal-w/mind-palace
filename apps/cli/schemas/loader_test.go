package schemas

import (
	"testing"
)

func TestCompile(t *testing.T) {
	tests := []struct {
		name       string
		schemaName string
		wantErr    bool
	}{
		{
			name:       "compile palace schema",
			schemaName: Palace,
			wantErr:    false,
		},
		{
			name:       "compile room schema",
			schemaName: Room,
			wantErr:    false,
		},
		{
			name:       "compile playbook schema",
			schemaName: Playbook,
			wantErr:    false,
		},
		{
			name:       "compile context-pack schema",
			schemaName: ContextPack,
			wantErr:    false,
		},
		{
			name:       "compile change-signal schema",
			schemaName: ChangeSignal,
			wantErr:    false,
		},
		{
			name:       "compile project-profile schema",
			schemaName: ProjectProfile,
			wantErr:    false,
		},
		{
			name:       "compile scan schema",
			schemaName: ScanSummary,
			wantErr:    false,
		},
		{
			name:       "compile non-existent schema",
			schemaName: "nonexistent",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := Compile(tt.schemaName)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if schema == nil {
				t.Error("expected non-nil schema")
			}
		})
	}
}

func TestList(t *testing.T) {
	schemas, err := List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	expectedSchemas := []string{Palace, Room, Playbook, ContextPack, ChangeSignal, ProjectProfile, ScanSummary}
	for _, name := range expectedSchemas {
		data, ok := schemas[name]
		if !ok {
			t.Errorf("schema %q not found in List() result", name)
			continue
		}
		if len(data) == 0 {
			t.Errorf("schema %q has empty content", name)
		}
	}

	// Verify the number of schemas
	if len(schemas) != len(expectedSchemas) {
		t.Errorf("List() returned %d schemas, want %d", len(schemas), len(expectedSchemas))
	}
}

func TestSchemaPath(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "palace", want: "palace.schema.json"},
		{name: "room", want: "room.schema.json"},
		{name: "test", want: "test.schema.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := schemaPath(tt.name)
			if got != tt.want {
				t.Errorf("schemaPath(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestSchemaURL(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "palace", want: "mem://schemas/palace.schema.json"},
		{name: "room", want: "mem://schemas/room.schema.json"},
		{name: "test", want: "mem://schemas/test.schema.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := schemaURL(tt.name)
			if got != tt.want {
				t.Errorf("schemaURL(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestGetCompiler(t *testing.T) {
	// Test that getCompiler returns a valid compiler
	compiler, err := getCompiler()
	if err != nil {
		t.Fatalf("getCompiler() error: %v", err)
	}
	if compiler == nil {
		t.Error("expected non-nil compiler")
	}

	// Calling again should return the same compiler (singleton)
	compiler2, err := getCompiler()
	if err != nil {
		t.Fatalf("getCompiler() second call error: %v", err)
	}
	if compiler != compiler2 {
		t.Error("getCompiler() should return the same instance")
	}
}

func TestCompileMultipleTimes(t *testing.T) {
	// Compiling the same schema multiple times should work
	for i := 0; i < 3; i++ {
		schema, err := Compile(Palace)
		if err != nil {
			t.Fatalf("Compile(Palace) iteration %d error: %v", i, err)
		}
		if schema == nil {
			t.Errorf("Compile(Palace) iteration %d returned nil", i)
		}
	}
}

func TestSchemaConstants(t *testing.T) {
	// Ensure constants have expected values
	if Palace != "palace" {
		t.Errorf("Palace = %q, want %q", Palace, "palace")
	}
	if Room != "room" {
		t.Errorf("Room = %q, want %q", Room, "room")
	}
	if Playbook != "playbook" {
		t.Errorf("Playbook = %q, want %q", Playbook, "playbook")
	}
	if ContextPack != "context-pack" {
		t.Errorf("ContextPack = %q, want %q", ContextPack, "context-pack")
	}
	if ChangeSignal != "change-signal" {
		t.Errorf("ChangeSignal = %q, want %q", ChangeSignal, "change-signal")
	}
	if ProjectProfile != "project-profile" {
		t.Errorf("ProjectProfile = %q, want %q", ProjectProfile, "project-profile")
	}
	if ScanSummary != "scan" {
		t.Errorf("ScanSummary = %q, want %q", ScanSummary, "scan")
	}
}
