package extractors

import (
	"testing"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/contracts"
)

func TestGoTypeExtractor_BasicStruct(t *testing.T) {
	code := []byte(`
package main

type User struct {
	ID        string ` + "`json:\"id\"`" + `
	Name      string ` + "`json:\"name\"`" + `
	Email     string ` + "`json:\"email\"`" + `
	Age       int    ` + "`json:\"age\"`" + `
	IsActive  bool   ` + "`json:\"isActive\"`" + `
}
`)

	extractor := NewGoTypeExtractor()
	schema, err := extractor.ExtractStructSchema(code, "User")
	if err != nil {
		t.Fatalf("failed to extract schema: %v", err)
	}

	if schema == nil {
		t.Fatal("schema is nil")
	}

	if schema.Type != contracts.SchemaTypeObject {
		t.Errorf("expected object type, got %s", schema.Type)
	}

	// Check properties
	expectedProps := map[string]contracts.SchemaType{
		"id":       contracts.SchemaTypeString,
		"name":     contracts.SchemaTypeString,
		"email":    contracts.SchemaTypeString,
		"age":      contracts.SchemaTypeInteger,
		"isActive": contracts.SchemaTypeBoolean,
	}

	for name, expectedType := range expectedProps {
		prop, exists := schema.Properties[name]
		if !exists {
			t.Errorf("missing property %q", name)
			continue
		}
		if prop.Type != expectedType {
			t.Errorf("property %q: expected type %s, got %s", name, expectedType, prop.Type)
		}
	}
}

func TestGoTypeExtractor_PointerFields(t *testing.T) {
	code := []byte(`
package main

type User struct {
	Name     string  ` + "`json:\"name\"`" + `
	Nickname *string ` + "`json:\"nickname\"`" + `
}
`)

	extractor := NewGoTypeExtractor()
	schema, err := extractor.ExtractStructSchema(code, "User")
	if err != nil {
		t.Fatalf("failed to extract schema: %v", err)
	}

	// Name should not be nullable
	nameProp := schema.Properties["name"]
	if nameProp == nil {
		t.Fatal("missing name property")
	}
	if nameProp.Nullable {
		t.Error("name should not be nullable")
	}

	// Nickname should be nullable (pointer)
	nicknameProp := schema.Properties["nickname"]
	if nicknameProp == nil {
		t.Fatal("missing nickname property")
	}
	if !nicknameProp.Nullable {
		t.Error("nickname should be nullable (pointer type)")
	}
}

func TestGoTypeExtractor_OmitemptyTag(t *testing.T) {
	code := []byte(`
package main

type User struct {
	ID       string ` + "`json:\"id\"`" + `
	Email    string ` + "`json:\"email,omitempty\"`" + `
}
`)

	extractor := NewGoTypeExtractor()
	schema, err := extractor.ExtractStructSchema(code, "User")
	if err != nil {
		t.Fatalf("failed to extract schema: %v", err)
	}

	// ID should be required (no omitempty)
	if !schema.IsRequired("id") {
		t.Error("id should be required")
	}

	// Email should not be required (has omitempty)
	if schema.IsRequired("email") {
		t.Error("email should not be required (has omitempty)")
	}
}

func TestGoTypeExtractor_IgnoredField(t *testing.T) {
	code := []byte(`
package main

type User struct {
	ID       string ` + "`json:\"id\"`" + `
	Password string ` + "`json:\"-\"`" + `
}
`)

	extractor := NewGoTypeExtractor()
	schema, err := extractor.ExtractStructSchema(code, "User")
	if err != nil {
		t.Fatalf("failed to extract schema: %v", err)
	}

	// Password should be ignored
	if _, exists := schema.Properties["password"]; exists {
		t.Error("password should be ignored (json:\"-\")")
	}
	if _, exists := schema.Properties["-"]; exists {
		t.Error("should not have a field named '-'")
	}
}

func TestGoTypeExtractor_ArrayField(t *testing.T) {
	code := []byte(`
package main

type User struct {
	Tags []string ` + "`json:\"tags\"`" + `
}
`)

	extractor := NewGoTypeExtractor()
	schema, err := extractor.ExtractStructSchema(code, "User")
	if err != nil {
		t.Fatalf("failed to extract schema: %v", err)
	}

	tagsProp := schema.Properties["tags"]
	if tagsProp == nil {
		t.Fatal("missing tags property")
	}
	if tagsProp.Type != contracts.SchemaTypeArray {
		t.Errorf("tags should be array, got %s", tagsProp.Type)
	}
	if tagsProp.Items == nil {
		t.Fatal("tags items is nil")
	}
	if tagsProp.Items.Type != contracts.SchemaTypeString {
		t.Errorf("tags items should be string, got %s", tagsProp.Items.Type)
	}
}

func TestGoTypeExtractor_ExtractAllStructs(t *testing.T) {
	code := []byte(`
package main

type User struct {
	ID   string ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

type Post struct {
	ID      string ` + "`json:\"id\"`" + `
	Title   string ` + "`json:\"title\"`" + `
	Content string ` + "`json:\"content\"`" + `
}

// Unexported struct - should not be extracted
type internal struct {
	data string
}
`)

	extractor := NewGoTypeExtractor()
	schemas, err := extractor.ExtractAllStructSchemas(code)
	if err != nil {
		t.Fatalf("failed to extract schemas: %v", err)
	}

	if len(schemas) != 2 {
		t.Errorf("expected 2 schemas, got %d", len(schemas))
	}

	if _, exists := schemas["User"]; !exists {
		t.Error("missing User schema")
	}
	if _, exists := schemas["Post"]; !exists {
		t.Error("missing Post schema")
	}
	if _, exists := schemas["internal"]; exists {
		t.Error("internal struct should not be extracted (unexported)")
	}
}

func TestParseJSONTag(t *testing.T) {
	tests := []struct {
		tag           string
		expectedName  string
		expectedOmit  bool
	}{
		{"`json:\"id\"`", "id", false},
		{"`json:\"name,omitempty\"`", "name", true},
		{"`json:\"-\"`", "-", false},
		{"`json:\"user_id,omitempty,string\"`", "user_id", true},
		{"`db:\"id\" json:\"userId\"`", "userId", false},
		{"`xml:\"id\"`", "", false}, // No json tag
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			name, omit := parseJSONTag(tt.tag)
			if name != tt.expectedName {
				t.Errorf("expected name %q, got %q", tt.expectedName, name)
			}
			if omit != tt.expectedOmit {
				t.Errorf("expected omitempty=%v, got %v", tt.expectedOmit, omit)
			}
		})
	}
}
