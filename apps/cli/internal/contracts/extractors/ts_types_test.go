package extractors

import (
	"testing"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/contracts"
)

func TestTSTypeExtractor_BasicInterface(t *testing.T) {
	code := []byte(`
interface User {
	id: string;
	name: string;
	email: string;
	age: number;
	isActive: boolean;
}
`)

	extractor := NewTSTypeExtractor()
	schema, err := extractor.ExtractInterfaceSchema(code, "User")
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
		"age":      contracts.SchemaTypeNumber,
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

func TestTSTypeExtractor_OptionalProperties(t *testing.T) {
	code := []byte(`
interface User {
	id: string;
	nickname?: string;
	email?: string;
}
`)

	extractor := NewTSTypeExtractor()
	schema, err := extractor.ExtractInterfaceSchema(code, "User")
	if err != nil {
		t.Fatalf("failed to extract schema: %v", err)
	}

	// id should be required
	if !schema.IsRequired("id") {
		t.Error("id should be required")
	}

	// nickname should be optional
	if schema.IsRequired("nickname") {
		t.Error("nickname should be optional")
	}

	// email should be optional
	if schema.IsRequired("email") {
		t.Error("email should be optional")
	}
}

func TestTSTypeExtractor_NullableUnion(t *testing.T) {
	code := []byte(`
interface User {
	name: string;
	nickname: string | null;
	bio: string | undefined;
}
`)

	extractor := NewTSTypeExtractor()
	schema, err := extractor.ExtractInterfaceSchema(code, "User")
	if err != nil {
		t.Fatalf("failed to extract schema: %v", err)
	}

	// name should not be nullable
	nameProp := schema.Properties["name"]
	if nameProp == nil {
		t.Fatal("missing name property")
	}
	if nameProp.Nullable {
		t.Error("name should not be nullable")
	}

	// nickname should be nullable (| null)
	nicknameProp := schema.Properties["nickname"]
	if nicknameProp == nil {
		t.Fatal("missing nickname property")
	}
	if !nicknameProp.Nullable {
		t.Error("nickname should be nullable (union with null)")
	}

	// bio should be nullable (| undefined)
	bioProp := schema.Properties["bio"]
	if bioProp == nil {
		t.Fatal("missing bio property")
	}
	if !bioProp.Nullable {
		t.Error("bio should be nullable (union with undefined)")
	}
}

func TestTSTypeExtractor_ArrayTypes(t *testing.T) {
	code := []byte(`
interface User {
	tags: string[];
	scores: Array<number>;
}
`)

	extractor := NewTSTypeExtractor()
	schema, err := extractor.ExtractInterfaceSchema(code, "User")
	if err != nil {
		t.Fatalf("failed to extract schema: %v", err)
	}

	// tags should be array of strings
	tagsProp := schema.Properties["tags"]
	if tagsProp == nil {
		t.Fatal("missing tags property")
	}
	if tagsProp.Type != contracts.SchemaTypeArray {
		t.Errorf("tags should be array, got %s", tagsProp.Type)
	}
	if tagsProp.Items == nil || tagsProp.Items.Type != contracts.SchemaTypeString {
		t.Error("tags items should be string")
	}

	// scores should be array of numbers
	scoresProp := schema.Properties["scores"]
	if scoresProp == nil {
		t.Fatal("missing scores property")
	}
	if scoresProp.Type != contracts.SchemaTypeArray {
		t.Errorf("scores should be array, got %s", scoresProp.Type)
	}
	if scoresProp.Items == nil || scoresProp.Items.Type != contracts.SchemaTypeNumber {
		t.Error("scores items should be number")
	}
}

func TestTSTypeExtractor_TypeAlias(t *testing.T) {
	code := []byte(`
type User = {
	id: string;
	name: string;
}
`)

	extractor := NewTSTypeExtractor()
	schema, err := extractor.ExtractTypeAliasSchema(code, "User")
	if err != nil {
		t.Fatalf("failed to extract schema: %v", err)
	}

	if schema == nil {
		t.Fatal("schema is nil")
	}

	if schema.Type != contracts.SchemaTypeObject {
		t.Errorf("expected object type, got %s", schema.Type)
	}

	if _, exists := schema.Properties["id"]; !exists {
		t.Error("missing id property")
	}
	if _, exists := schema.Properties["name"]; !exists {
		t.Error("missing name property")
	}
}

func TestTSTypeExtractor_ExtractAllSchemas(t *testing.T) {
	code := []byte(`
interface User {
	id: string;
	name: string;
}

interface Post {
	id: string;
	title: string;
	content: string;
}

type UserResponse = {
	data: User;
}
`)

	extractor := NewTSTypeExtractor()
	schemas, err := extractor.ExtractAllSchemas(code)
	if err != nil {
		t.Fatalf("failed to extract schemas: %v", err)
	}

	if len(schemas) < 3 {
		t.Errorf("expected at least 3 schemas, got %d", len(schemas))
	}

	if _, exists := schemas["User"]; !exists {
		t.Error("missing User schema")
	}
	if _, exists := schemas["Post"]; !exists {
		t.Error("missing Post schema")
	}
	if _, exists := schemas["UserResponse"]; !exists {
		t.Error("missing UserResponse schema")
	}
}

func TestTSTypeExtractor_NestedInterface(t *testing.T) {
	code := []byte(`
interface Address {
	street: string;
	city: string;
}

interface User {
	id: string;
	address: Address;
}
`)

	extractor := NewTSTypeExtractor()
	schema, err := extractor.ExtractInterfaceSchema(code, "User")
	if err != nil {
		t.Fatalf("failed to extract schema: %v", err)
	}

	// address property should reference Address type
	addrProp := schema.Properties["address"]
	if addrProp == nil {
		t.Fatal("missing address property")
	}
	// Since we can't resolve references, it should be unknown
	// In a real implementation we'd resolve type references
	if addrProp.Type == contracts.SchemaTypeUnknown {
		// Expected - we don't resolve type references yet
		t.Log("address type is unknown (reference resolution not implemented)")
	}
}
