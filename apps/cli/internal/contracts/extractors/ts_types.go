package extractors

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/contracts"
)

// TSTypeExtractor extracts type schemas from TypeScript source files.
type TSTypeExtractor struct {
	parser *sitter.Parser
}

// NewTSTypeExtractor creates a new TypeScript type extractor.
func NewTSTypeExtractor() *TSTypeExtractor {
	p := sitter.NewParser()
	p.SetLanguage(typescript.GetLanguage())
	return &TSTypeExtractor{parser: p}
}

// ExtractInterfaceSchema extracts a TypeSchema for a TypeScript interface by name.
func (e *TSTypeExtractor) ExtractInterfaceSchema(content []byte, interfaceName string) (*contracts.TypeSchema, error) {
	tree, err := e.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	return e.findInterfaceSchema(tree.RootNode(), content, interfaceName), nil
}

// ExtractTypeAliasSchema extracts a TypeSchema for a TypeScript type alias by name.
func (e *TSTypeExtractor) ExtractTypeAliasSchema(content []byte, typeName string) (*contracts.TypeSchema, error) {
	tree, err := e.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	return e.findTypeAliasSchema(tree.RootNode(), content, typeName), nil
}

// ExtractAllSchemas extracts TypeSchemas for all exported interfaces and type aliases.
func (e *TSTypeExtractor) ExtractAllSchemas(content []byte) (map[string]*contracts.TypeSchema, error) {
	tree, err := e.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	schemas := make(map[string]*contracts.TypeSchema)
	e.extractTypeDeclarations(tree.RootNode(), content, schemas)
	return schemas, nil
}

func (e *TSTypeExtractor) findInterfaceSchema(node *sitter.Node, content []byte, interfaceName string) *contracts.TypeSchema {
	if node.Type() == "interface_declaration" {
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil && nameNode.Content(content) == interfaceName {
			bodyNode := node.ChildByFieldName("body")
			if bodyNode != nil {
				return e.parseObjectType(bodyNode, content)
			}
		}
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			if result := e.findInterfaceSchema(child, content, interfaceName); result != nil {
				return result
			}
		}
	}

	return nil
}

func (e *TSTypeExtractor) findTypeAliasSchema(node *sitter.Node, content []byte, typeName string) *contracts.TypeSchema {
	if node.Type() == "type_alias_declaration" {
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil && nameNode.Content(content) == typeName {
			valueNode := node.ChildByFieldName("value")
			if valueNode != nil {
				return e.parseTypeNode(valueNode, content)
			}
		}
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			if result := e.findTypeAliasSchema(child, content, typeName); result != nil {
				return result
			}
		}
	}

	return nil
}

//nolint:gocognit,nestif // TypeScript AST traversal is inherently complex
func (e *TSTypeExtractor) extractTypeDeclarations(node *sitter.Node, content []byte, schemas map[string]*contracts.TypeSchema) {
	nodeType := node.Type()

	// Check for export statement wrapper
	if nodeType == "export_statement" {
		// Process the declaration inside
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil {
				e.extractTypeDeclarations(child, content, schemas)
			}
		}
		return
	}

	// Also consider declarations at top level as potentially exported
	if nodeType == "interface_declaration" || nodeType == "type_alias_declaration" {
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			name := nameNode.Content(content)
			// In TS, check if it starts with capital letter (convention for types)
			if len(name) > 0 {
				var schema *contracts.TypeSchema
				if nodeType == "interface_declaration" {
					bodyNode := node.ChildByFieldName("body")
					if bodyNode != nil {
						schema = e.parseObjectType(bodyNode, content)
					}
				} else {
					valueNode := node.ChildByFieldName("value")
					if valueNode != nil {
						schema = e.parseTypeNode(valueNode, content)
					}
				}
				if schema != nil {
					schemas[name] = schema
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

func (e *TSTypeExtractor) parseObjectType(node *sitter.Node, content []byte) *contracts.TypeSchema {
	schema := contracts.NewObjectSchema()

	// Parse properties from object_type or interface body
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		childType := child.Type()
		if childType == "property_signature" || childType == "public_field_definition" {
			e.parsePropertySignature(child, content, schema)
		}
	}

	return schema
}

func (e *TSTypeExtractor) parsePropertySignature(node *sitter.Node, content []byte, schema *contracts.TypeSchema) {
	var name string
	var typeNode *sitter.Node
	optional := false

	// Parse property signature: name?: type
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		childType := child.Type()
		switch childType {
		case "property_identifier", "identifier":
			name = child.Content(content)
		case "?":
			optional = true
		case "type_annotation":
			// The actual type is inside the type_annotation
			for j := 0; j < int(child.ChildCount()); j++ {
				typeChild := child.Child(j)
				if typeChild != nil && typeChild.Type() != ":" {
					typeNode = typeChild
				}
			}
		}
	}

	// Also try field API
	if name == "" {
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil {
			name = nameNode.Content(content)
		}
	}
	if typeNode == nil {
		typeNode = node.ChildByFieldName("type")
	}

	if name == "" {
		return
	}

	var propSchema *contracts.TypeSchema
	if typeNode != nil {
		propSchema = e.parseTypeNode(typeNode, content)
	} else {
		propSchema = contracts.NewPrimitiveSchema(contracts.SchemaTypeAny)
	}

	// Required if not optional
	schema.AddProperty(name, propSchema, !optional)
}

//nolint:gocognit,gocyclo // TypeScript type parsing requires handling many node types
func (e *TSTypeExtractor) parseTypeNode(node *sitter.Node, content []byte) *contracts.TypeSchema {
	if node == nil {
		return contracts.NewPrimitiveSchema(contracts.SchemaTypeAny)
	}

	nodeType := node.Type()

	switch nodeType {
	case "predefined_type", "type_identifier", "identifier":
		typeName := node.Content(content)
		return contracts.TSTypeToSchema(typeName)

	case "object_type":
		return e.parseObjectType(node, content)

	case "array_type":
		// T[]
		var elemNode *sitter.Node
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() != "[" && child.Type() != "]" {
				elemNode = child
				break
			}
		}
		elemSchema := e.parseTypeNode(elemNode, content)
		return contracts.NewArraySchema(elemSchema)

	case "generic_type":
		// Array<T>, Promise<T>, Record<K,V>, etc.
		var nameNode *sitter.Node
		var argsNode *sitter.Node
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child == nil {
				continue
			}
			switch child.Type() {
			case "type_identifier", "identifier":
				nameNode = child
			case "type_arguments":
				argsNode = child
			}
		}

		if nameNode != nil { //nolint:nestif // complex type resolution logic
			name := nameNode.Content(content)
			switch name {
			case "Array":
				// Array<T> -> array
				if argsNode != nil {
					for i := 0; i < int(argsNode.ChildCount()); i++ {
						arg := argsNode.Child(i)
						if arg != nil && arg.Type() != "<" && arg.Type() != ">" && arg.Type() != "," {
							return contracts.NewArraySchema(e.parseTypeNode(arg, content))
						}
					}
				}
				return contracts.NewArraySchema(contracts.NewPrimitiveSchema(contracts.SchemaTypeAny))
			case "Promise":
				// Promise<T> -> T (unwrap)
				if argsNode != nil {
					for i := 0; i < int(argsNode.ChildCount()); i++ {
						arg := argsNode.Child(i)
						if arg != nil && arg.Type() != "<" && arg.Type() != ">" && arg.Type() != "," {
							return e.parseTypeNode(arg, content)
						}
					}
				}
				return contracts.NewPrimitiveSchema(contracts.SchemaTypeAny)
			case "Record":
				// Record<K, V> -> object
				return contracts.NewObjectSchema()
			default:
				// Unknown generic, treat as unknown
				return contracts.NewPrimitiveSchema(contracts.SchemaTypeUnknown)
			}
		}

	case "union_type":
		// T | null | undefined
		nullable := false
		var mainType *contracts.TypeSchema
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child == nil || child.Type() == "|" {
				continue
			}
			typeName := child.Content(content)
			if typeName == "null" || typeName == "undefined" {
				nullable = true
			} else if mainType == nil {
				mainType = e.parseTypeNode(child, content)
			}
		}
		if mainType == nil {
			mainType = contracts.NewPrimitiveSchema(contracts.SchemaTypeAny)
		}
		mainType.Nullable = nullable
		return mainType

	case "intersection_type":
		// T & U -> treat as object, merge properties (simplified)
		schema := contracts.NewObjectSchema()
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child == nil || child.Type() == "&" {
				continue
			}
			childSchema := e.parseTypeNode(child, content)
			if childSchema != nil && childSchema.Properties != nil {
				for k, v := range childSchema.Properties {
					schema.Properties[k] = v
				}
			}
		}
		return schema

	case "literal_type":
		// "value" | 123 | true
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil {
				switch child.Type() {
				case "string", "template_string":
					return contracts.NewPrimitiveSchema(contracts.SchemaTypeString)
				case "number":
					return contracts.NewPrimitiveSchema(contracts.SchemaTypeNumber)
				case "true", "false":
					return contracts.NewPrimitiveSchema(contracts.SchemaTypeBoolean)
				case "null":
					return &contracts.TypeSchema{Type: contracts.SchemaTypeNull, Nullable: true}
				}
			}
		}
		return contracts.NewPrimitiveSchema(contracts.SchemaTypeAny)

	case "parenthesized_type":
		// (T)
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() != "(" && child.Type() != ")" {
				return e.parseTypeNode(child, content)
			}
		}
	}

	// Try to get type from content
	typeName := strings.TrimSpace(node.Content(content))
	return contracts.TSTypeToSchema(typeName)
}
