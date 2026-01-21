package contracts

import (
	"testing"
)

func TestTypeSchemaString(t *testing.T) {
	tests := []struct {
		name     string
		schema   *TypeSchema
		expected string
	}{
		{
			name:     "nil schema",
			schema:   nil,
			expected: "unknown",
		},
		{
			name:     "string type",
			schema:   NewPrimitiveSchema(SchemaTypeString),
			expected: "string",
		},
		{
			name:     "nullable string",
			schema:   &TypeSchema{Type: SchemaTypeString, Nullable: true},
			expected: "string?",
		},
		{
			name:     "string with format",
			schema:   &TypeSchema{Type: SchemaTypeString, Format: "date-time"},
			expected: "string(date-time)",
		},
		{
			name:     "array of strings",
			schema:   NewArraySchema(NewPrimitiveSchema(SchemaTypeString)),
			expected: "array<string>",
		},
		{
			name:     "empty object",
			schema:   NewObjectSchema(),
			expected: "object",
		},
		{
			name: "object with properties",
			schema: func() *TypeSchema {
				s := NewObjectSchema()
				s.AddProperty("name", NewPrimitiveSchema(SchemaTypeString), true)
				s.AddProperty("age", NewPrimitiveSchema(SchemaTypeInteger), false)
				return s
			}(),
			expected: "object{age,name}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.schema.String()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestTypeSchemaCompare(t *testing.T) {
	tests := []struct {
		name           string
		backend        *TypeSchema
		frontend       *TypeSchema
		expectedCount  int
		expectedTypes  []MismatchType
	}{
		{
			name:          "identical primitives",
			backend:       NewPrimitiveSchema(SchemaTypeString),
			frontend:      NewPrimitiveSchema(SchemaTypeString),
			expectedCount: 0,
		},
		{
			name:          "type mismatch",
			backend:       NewPrimitiveSchema(SchemaTypeString),
			frontend:      NewPrimitiveSchema(SchemaTypeNumber),
			expectedCount: 1,
			expectedTypes: []MismatchType{MismatchTypeMismatch},
		},
		{
			name:          "number and integer are compatible",
			backend:       NewPrimitiveSchema(SchemaTypeNumber),
			frontend:      NewPrimitiveSchema(SchemaTypeInteger),
			expectedCount: 0,
		},
		{
			name:          "nullability mismatch",
			backend:       &TypeSchema{Type: SchemaTypeString, Nullable: true},
			frontend:      &TypeSchema{Type: SchemaTypeString, Nullable: false},
			expectedCount: 1,
			expectedTypes: []MismatchType{MismatchNullabilityMismatch},
		},
		{
			name:          "any type matches everything",
			backend:       NewPrimitiveSchema(SchemaTypeAny),
			frontend:      NewPrimitiveSchema(SchemaTypeString),
			expectedCount: 0,
		},
		{
			name: "missing in frontend",
			backend: func() *TypeSchema {
				s := NewObjectSchema()
				s.AddProperty("id", NewPrimitiveSchema(SchemaTypeString), true)
				s.AddProperty("name", NewPrimitiveSchema(SchemaTypeString), true)
				return s
			}(),
			frontend: func() *TypeSchema {
				s := NewObjectSchema()
				s.AddProperty("id", NewPrimitiveSchema(SchemaTypeString), true)
				return s
			}(),
			expectedCount: 1,
			expectedTypes: []MismatchType{MismatchMissingInFrontend},
		},
		{
			name: "missing in backend",
			backend: func() *TypeSchema {
				s := NewObjectSchema()
				s.AddProperty("id", NewPrimitiveSchema(SchemaTypeString), true)
				return s
			}(),
			frontend: func() *TypeSchema {
				s := NewObjectSchema()
				s.AddProperty("id", NewPrimitiveSchema(SchemaTypeString), true)
				s.AddProperty("email", NewPrimitiveSchema(SchemaTypeString), true)
				return s
			}(),
			expectedCount: 1,
			expectedTypes: []MismatchType{MismatchMissingInBackend},
		},
		{
			name: "optionality mismatch",
			backend: func() *TypeSchema {
				s := NewObjectSchema()
				s.AddProperty("id", NewPrimitiveSchema(SchemaTypeString), false) // optional
				return s
			}(),
			frontend: func() *TypeSchema {
				s := NewObjectSchema()
				s.AddProperty("id", NewPrimitiveSchema(SchemaTypeString), true) // required
				return s
			}(),
			expectedCount: 1,
			expectedTypes: []MismatchType{MismatchOptionalityMismatch},
		},
		{
			name: "nested object mismatch",
			backend: func() *TypeSchema {
				inner := NewObjectSchema()
				inner.AddProperty("email", NewPrimitiveSchema(SchemaTypeString), true)
				s := NewObjectSchema()
				s.AddProperty("user", inner, true)
				return s
			}(),
			frontend: func() *TypeSchema {
				inner := NewObjectSchema()
				inner.AddProperty("email", NewPrimitiveSchema(SchemaTypeNumber), true) // wrong type
				s := NewObjectSchema()
				s.AddProperty("user", inner, true)
				return s
			}(),
			expectedCount: 1,
			expectedTypes: []MismatchType{MismatchTypeMismatch},
		},
		{
			name: "array item mismatch",
			backend:       NewArraySchema(NewPrimitiveSchema(SchemaTypeString)),
			frontend:      NewArraySchema(NewPrimitiveSchema(SchemaTypeNumber)),
			expectedCount: 1,
			expectedTypes: []MismatchType{MismatchTypeMismatch},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mismatches := tt.backend.Compare(tt.frontend, "")

			if len(mismatches) != tt.expectedCount {
				t.Errorf("expected %d mismatches, got %d", tt.expectedCount, len(mismatches))
				for _, m := range mismatches {
					t.Logf("  - %s: %s (%s)", m.FieldPath, m.Type, m.Description)
				}
				return
			}

			if tt.expectedTypes != nil {
				for i, expectedType := range tt.expectedTypes {
					if i >= len(mismatches) {
						break
					}
					if mismatches[i].Type != expectedType {
						t.Errorf("mismatch %d: expected type %q, got %q", i, expectedType, mismatches[i].Type)
					}
				}
			}
		})
	}
}

func TestGoTypeToSchema(t *testing.T) {
	tests := []struct {
		goType       string
		expectedType SchemaType
		nullable     bool
		isArray      bool
	}{
		{"string", SchemaTypeString, false, false},
		{"int", SchemaTypeInteger, false, false},
		{"int64", SchemaTypeInteger, false, false},
		{"float64", SchemaTypeNumber, false, false},
		{"bool", SchemaTypeBoolean, false, false},
		{"*string", SchemaTypeString, true, false},
		{"[]string", SchemaTypeArray, false, true},
		{"time.Time", SchemaTypeString, false, false}, // with date-time format
		{"interface{}", SchemaTypeAny, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.goType, func(t *testing.T) {
			schema := GoTypeToSchema(tt.goType)

			if tt.isArray {
				if schema.Type != SchemaTypeArray {
					t.Errorf("expected array type, got %s", schema.Type)
				}
			} else {
				if schema.Type != tt.expectedType {
					t.Errorf("expected type %s, got %s", tt.expectedType, schema.Type)
				}
				if schema.Nullable != tt.nullable {
					t.Errorf("expected nullable=%v, got %v", tt.nullable, schema.Nullable)
				}
			}
		})
	}
}

func TestTSTypeToSchema(t *testing.T) {
	tests := []struct {
		tsType       string
		expectedType SchemaType
		nullable     bool
		isArray      bool
	}{
		{"string", SchemaTypeString, false, false},
		{"number", SchemaTypeNumber, false, false},
		{"boolean", SchemaTypeBoolean, false, false},
		{"any", SchemaTypeAny, false, false},
		{"string | null", SchemaTypeString, true, false},
		{"string[]", SchemaTypeArray, false, true},
		{"Array<string>", SchemaTypeArray, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.tsType, func(t *testing.T) {
			schema := TSTypeToSchema(tt.tsType)

			if tt.isArray {
				if schema.Type != SchemaTypeArray {
					t.Errorf("expected array type, got %s", schema.Type)
				}
			} else {
				if schema.Type != tt.expectedType {
					t.Errorf("expected type %s, got %s", tt.expectedType, schema.Type)
				}
				if schema.Nullable != tt.nullable {
					t.Errorf("expected nullable=%v, got %v", tt.nullable, schema.Nullable)
				}
			}
		})
	}
}

func TestTypeSchemaClone(t *testing.T) {
	original := NewObjectSchema()
	original.AddProperty("name", NewPrimitiveSchema(SchemaTypeString), true)
	original.AddProperty("tags", NewArraySchema(NewPrimitiveSchema(SchemaTypeString)), false)

	clone := original.Clone()

	// Modify original
	original.Properties["name"].Nullable = true
	original.AddProperty("new", NewPrimitiveSchema(SchemaTypeNumber), false)

	// Clone should be unchanged
	if clone.Properties["name"].Nullable {
		t.Error("clone should not be affected by original modification")
	}
	if _, exists := clone.Properties["new"]; exists {
		t.Error("clone should not have new property added to original")
	}
}
