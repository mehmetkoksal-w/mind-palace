package extractors

import (
	"context"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/contracts"
)

// GoTypeExtractor extracts type schemas from Go source files.
type GoTypeExtractor struct {
	parser *sitter.Parser
}

// NewGoTypeExtractor creates a new Go type extractor.
func NewGoTypeExtractor() *GoTypeExtractor {
	p := sitter.NewParser()
	p.SetLanguage(golang.GetLanguage())
	return &GoTypeExtractor{parser: p}
}

// ExtractStructSchema extracts a TypeSchema for a Go struct by name.
// It parses the file and finds the struct definition.
func (e *GoTypeExtractor) ExtractStructSchema(content []byte, structName string) (*contracts.TypeSchema, error) {
	tree, err := e.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	return e.findStructSchema(tree.RootNode(), content, structName), nil
}

// ExtractAllStructSchemas extracts TypeSchemas for all exported structs in a file.
func (e *GoTypeExtractor) ExtractAllStructSchemas(content []byte) (map[string]*contracts.TypeSchema, error) {
	tree, err := e.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	schemas := make(map[string]*contracts.TypeSchema)
	e.extractTypeDeclarations(tree.RootNode(), content, schemas)
	return schemas, nil
}

func (e *GoTypeExtractor) findStructSchema(node *sitter.Node, content []byte, structName string) *contracts.TypeSchema {
	if node.Type() == "type_spec" {
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil && nameNode.Content(content) == structName {
			typeNode := node.ChildByFieldName("type")
			if typeNode != nil && typeNode.Type() == "struct_type" {
				return e.parseStructType(typeNode, content)
			}
		}
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			if result := e.findStructSchema(child, content, structName); result != nil {
				return result
			}
		}
	}

	return nil
}

func (e *GoTypeExtractor) extractTypeDeclarations(node *sitter.Node, content []byte, schemas map[string]*contracts.TypeSchema) {
	if node.Type() == "type_spec" { //nolint:nestif // type extraction requires nested conditions
		nameNode := node.ChildByFieldName("name")
		typeNode := node.ChildByFieldName("type")

		if nameNode != nil && typeNode != nil {
			name := nameNode.Content(content)
			// Only extract exported types
			if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
				if typeNode.Type() == "struct_type" {
					schemas[name] = e.parseStructType(typeNode, content)
				}
			}
		}
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			e.extractTypeDeclarations(child, content, schemas)
		}
	}
}

func (e *GoTypeExtractor) parseStructType(node *sitter.Node, content []byte) *contracts.TypeSchema {
	schema := contracts.NewObjectSchema()

	// Find field_declaration_list
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "field_declaration_list" {
			e.parseFieldList(child, content, schema)
		}
	}

	return schema
}

func (e *GoTypeExtractor) parseFieldList(node *sitter.Node, content []byte, schema *contracts.TypeSchema) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil || child.Type() != "field_declaration" {
			continue
		}

		e.parseFieldDeclaration(child, content, schema)
	}
}

func (e *GoTypeExtractor) parseFieldDeclaration(node *sitter.Node, content []byte, schema *contracts.TypeSchema) {
	// Get field name(s), type, and tag
	var names []string
	var typeNode *sitter.Node
	var tagNode *sitter.Node

	// Parse Go field declaration: field_identifier(s) type_identifier/type_node tag?
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		childType := child.Type()

		switch childType {
		case "field_identifier", "identifier":
			// Field name
			names = append(names, child.Content(content))
		case "raw_string_literal", "interpreted_string_literal":
			// Struct tag
			tagNode = child
		case "type_identifier", "pointer_type", "slice_type", "array_type",
			"map_type", "struct_type", "interface_type", "qualified_type":
			// Type node
			if typeNode == nil {
				typeNode = child
			}
		}
	}

	// Also check for the field name using the field API
	if len(names) == 0 {
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			names = append(names, nameNode.Content(content))
		}
	}

	// Get the type node using field API if not found
	if typeNode == nil {
		typeNode = node.ChildByFieldName("type")
	}

	if len(names) == 0 || typeNode == nil {
		return
	}

	// Parse the type
	fieldSchema := e.parseTypeNode(typeNode, content)

	// Parse struct tag for JSON field name and omitempty
	jsonName := ""
	omitempty := false
	if tagNode != nil {
		tag := tagNode.Content(content)
		jsonName, omitempty = parseJSONTag(tag)
	}

	// Add each field to the schema
	for _, name := range names {
		// Skip unexported fields
		if len(name) == 0 || name[0] < 'A' || name[0] > 'Z' {
			continue
		}

		// Use JSON tag name if available, otherwise lowercase the field name
		fieldName := jsonName
		if fieldName == "" {
			fieldName = strings.ToLower(name[:1]) + name[1:]
		}
		if fieldName == "-" {
			continue // Field is ignored in JSON
		}

		// Clone the schema for this field
		fieldSchemaCopy := fieldSchema.Clone()

		// If omitempty, the field is optional (not required)
		required := !omitempty
		schema.AddProperty(fieldName, fieldSchemaCopy, required)
	}
}

func (e *GoTypeExtractor) parseTypeNode(node *sitter.Node, content []byte) *contracts.TypeSchema {
	if node == nil {
		return contracts.NewPrimitiveSchema(contracts.SchemaTypeAny)
	}

	nodeType := node.Type()

	switch nodeType {
	case "type_identifier", "identifier", "field_identifier":
		typeName := node.Content(content)
		return contracts.GoTypeToSchema(typeName)

	case "pointer_type":
		// *T -> T with nullable=true
		inner := e.parseTypeNode(node.ChildByFieldName("type"), content)
		if inner == nil {
			// Try first child
			for i := 0; i < int(node.ChildCount()); i++ {
				child := node.Child(i)
				if child != nil && child.Type() != "*" {
					inner = e.parseTypeNode(child, content)
					break
				}
			}
		}
		if inner != nil {
			inner.Nullable = true
		}
		return inner

	case "slice_type", "array_type":
		// []T or [N]T
		var elemNode *sitter.Node
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() != "[" && child.Type() != "]" {
				// Skip array length for fixed-size arrays
				if child.Type() != "int_literal" {
					elemNode = child
				}
			}
		}
		if elemNode == nil {
			elemNode = node.ChildByFieldName("element")
		}
		elemSchema := e.parseTypeNode(elemNode, content)
		return contracts.NewArraySchema(elemSchema)

	case "map_type":
		// map[K]V -> object (simplified)
		return contracts.NewObjectSchema()

	case "struct_type":
		// Inline struct
		return e.parseStructType(node, content)

	case "interface_type":
		// interface{} -> any
		return contracts.NewPrimitiveSchema(contracts.SchemaTypeAny)

	case "qualified_type":
		// package.Type
		return contracts.GoTypeToSchema(node.Content(content))

	default:
		// Try to get the type from the content
		return contracts.GoTypeToSchema(node.Content(content))
	}
}

// parseJSONTag parses a struct tag and extracts the JSON field name and omitempty flag.
func parseJSONTag(tag string) (name string, omitempty bool) {
	// Remove backticks
	tag = strings.Trim(tag, "`")

	// Find json tag
	jsonRe := regexp.MustCompile(`json:"([^"]*)"`)
	matches := jsonRe.FindStringSubmatch(tag)
	if len(matches) < 2 {
		return "", false
	}

	jsonValue := matches[1]
	parts := strings.Split(jsonValue, ",")

	if len(parts) > 0 {
		name = parts[0]
	}

	for _, part := range parts[1:] {
		if part == "omitempty" {
			omitempty = true
			break
		}
	}

	return name, omitempty
}
