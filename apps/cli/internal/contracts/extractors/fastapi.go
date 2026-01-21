package extractors

import (
	"context"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/analysis"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/contracts"
)

// FastAPIExtractor extracts API endpoints from FastAPI applications.
type FastAPIExtractor struct {
	parser *sitter.Parser
}

// NewFastAPIExtractor creates a new FastAPI endpoint extractor.
func NewFastAPIExtractor() *FastAPIExtractor {
	p := sitter.NewParser()
	p.SetLanguage(python.GetLanguage())
	return &FastAPIExtractor{
		parser: p,
	}
}

// ID returns the unique identifier for this extractor.
func (e *FastAPIExtractor) ID() string {
	return "fastapi"
}

// Framework returns the framework name.
func (e *FastAPIExtractor) Framework() string {
	return "fastapi"
}

// Languages returns the languages this extractor supports.
func (e *FastAPIExtractor) Languages() []string {
	return []string{"python"}
}

// CanExtract returns true if this extractor can handle the given file.
func (e *FastAPIExtractor) CanExtract(file *analysis.FileAnalysis) bool {
	return file.Language == "python"
}

// ExtractEndpoints extracts API endpoints from FastAPI source code.
func (e *FastAPIExtractor) ExtractEndpoints(file *analysis.FileAnalysis) ([]ExtractedEndpoint, error) {
	return e.ExtractEndpointsFromContent([]byte{}, file.Path)
}

// ExtractEndpointsFromContent extracts endpoints directly from source content.
func (e *FastAPIExtractor) ExtractEndpointsFromContent(content []byte, filePath string) ([]ExtractedEndpoint, error) {
	tree, err := e.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	var endpoints []ExtractedEndpoint
	e.walkTree(tree.RootNode(), content, filePath, &endpoints)
	return endpoints, nil
}

func (e *FastAPIExtractor) walkTree(node *sitter.Node, content []byte, filePath string, endpoints *[]ExtractedEndpoint) {
	nodeType := node.Type()

	// Look for decorated definitions
	if nodeType == "decorated_definition" {
		if endpoint := e.parseDecoratedDefinition(node, content, filePath); endpoint != nil {
			*endpoints = append(*endpoints, *endpoint)
		}
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			e.walkTree(child, content, filePath, endpoints)
		}
	}
}

func (e *FastAPIExtractor) parseDecoratedDefinition(node *sitter.Node, content []byte, filePath string) *ExtractedEndpoint {
	// Find decorator(s) and function definition
	var decorator *sitter.Node
	var funcDef *sitter.Node

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "decorator":
			decorator = child
		case "function_definition":
			funcDef = child
		}
	}

	if decorator == nil || funcDef == nil {
		return nil
	}

	// Parse decorator to get route info
	endpoint := e.parseDecorator(decorator, content, filePath, node)
	if endpoint == nil {
		return nil
	}

	// Get function name as handler
	nameNode := funcDef.ChildByFieldName("name")
	if nameNode != nil {
		endpoint.Handler = nameNode.Content(content)
	}

	return endpoint
}

func (e *FastAPIExtractor) parseDecorator(node *sitter.Node, content []byte, filePath string, parentNode *sitter.Node) *ExtractedEndpoint {
	// Decorator format: @app.get("/path") or @router.post("/path")
	decoratorContent := node.Content(content)

	// Check if it's a FastAPI route decorator
	method := e.detectHTTPMethod(decoratorContent)
	if method == "" {
		return nil
	}

	// Extract path from decorator
	path := e.extractPath(decoratorContent)
	if path == "" {
		return nil
	}

	// Convert FastAPI path params {param} to standard format
	pathParams := e.extractFastAPIPathParams(path)

	return &ExtractedEndpoint{
		Method:     method,
		Path:       path,
		PathParams: pathParams,
		File:       filePath,
		Line:       int(parentNode.StartPoint().Row) + 1,
		Framework:  "fastapi",
	}
}

func (e *FastAPIExtractor) detectHTTPMethod(decorator string) string {
	// Match patterns like @app.get, @router.post, @api.delete
	patterns := map[string]string{
		`\.get\s*\(`:     "GET",
		`\.post\s*\(`:    "POST",
		`\.put\s*\(`:     "PUT",
		`\.delete\s*\(`:  "DELETE",
		`\.patch\s*\(`:   "PATCH",
		`\.head\s*\(`:    "HEAD",
		`\.options\s*\(`: "OPTIONS",
		`\.api_route\s*\(`: "ANY", // api_route can accept any method
	}

	for pattern, method := range patterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(decorator) {
			return method
		}
	}

	return ""
}

func (e *FastAPIExtractor) extractPath(decorator string) string {
	// Extract path from decorator like @app.get("/users/{id}")
	// or @app.get(path="/users/{id}")

	// Try positional argument first: @app.get("/path")
	positionalRe := regexp.MustCompile(`\.\w+\s*\(\s*["']([^"']+)["']`)
	if matches := positionalRe.FindStringSubmatch(decorator); len(matches) > 1 {
		return matches[1]
	}

	// Try named argument: @app.get(path="/path")
	namedRe := regexp.MustCompile(`path\s*=\s*["']([^"']+)["']`)
	if matches := namedRe.FindStringSubmatch(decorator); len(matches) > 1 {
		return matches[1]
	}

	return ""
}

func (e *FastAPIExtractor) extractFastAPIPathParams(path string) []string {
	// FastAPI uses {param} format, optionally with type hints like {user_id:int}
	paramRe := regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)(?::[^}]+)?\}`)
	matches := paramRe.FindAllStringSubmatch(path, -1)

	params := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}
	return params
}

// NormalizePath normalizes a FastAPI path for comparison.
// Converts {param:type} to {param}.
func (e *FastAPIExtractor) NormalizePath(path string) string {
	// Remove type hints from path params: {id:int} -> {id}
	paramRe := regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*):[^}]+\}`)
	return paramRe.ReplaceAllString(path, "{$1}")
}

// ExtractPydanticModels extracts Pydantic model schemas from content.
// This is a simplified implementation that extracts class definitions
// inheriting from BaseModel.
func (e *FastAPIExtractor) ExtractPydanticModels(content []byte) (map[string]*contracts.TypeSchema, error) {
	tree, err := e.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	models := make(map[string]*contracts.TypeSchema)
	e.findPydanticModels(tree.RootNode(), content, models)
	return models, nil
}

func (e *FastAPIExtractor) findPydanticModels(node *sitter.Node, content []byte, models map[string]*contracts.TypeSchema) {
	if node.Type() == "class_definition" { //nolint:nestif // complex model extraction logic
		// Check if it inherits from BaseModel
		if e.isPydanticModel(node, content) {
			className := ""
			nameNode := node.ChildByFieldName("name")
			if nameNode != nil {
				className = nameNode.Content(content)
			}
			if className != "" {
				schema := e.parsePydanticModel(node, content)
				if schema != nil {
					models[className] = schema
				}
			}
		}
	}

	// Recurse
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			e.findPydanticModels(child, content, models)
		}
	}
}

func (e *FastAPIExtractor) isPydanticModel(node *sitter.Node, content []byte) bool {
	// Check superclasses for BaseModel
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "argument_list" {
			// Check each argument for BaseModel
			argContent := child.Content(content)
			if strings.Contains(argContent, "BaseModel") {
				return true
			}
		}
	}
	return false
}

func (e *FastAPIExtractor) parsePydanticModel(node *sitter.Node, content []byte) *contracts.TypeSchema {
	schema := contracts.NewObjectSchema()

	// Find class body
	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return schema
	}

	// Parse typed assignments in class body
	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		child := bodyNode.Child(i)
		if child == nil {
			continue
		}

		// Look for typed_assignment or expression_statement containing assignment
		if child.Type() == "expression_statement" {
			// Check for annotated assignment: field: type = default
			for j := 0; j < int(child.ChildCount()); j++ {
				assign := child.Child(j)
				if assign != nil && assign.Type() == "assignment" {
					e.parseFieldAssignment(assign, content, schema)
				}
			}
		}
	}

	return schema
}

func (e *FastAPIExtractor) parseFieldAssignment(node *sitter.Node, content []byte, schema *contracts.TypeSchema) {
	// Pydantic field: name: Type = default or name: Type
	leftNode := node.ChildByFieldName("left")
	if leftNode == nil {
		return
	}

	// Check if this is an annotated name
	if leftNode.Type() != "identifier" {
		return
	}

	fieldName := leftNode.Content(content)

	// Get the type annotation if available
	// This is simplified - full type parsing would be more complex
	nodeContent := node.Content(content)

	// Try to extract type from annotation like "field: str"
	typeRe := regexp.MustCompile(`:\s*([A-Za-z][A-Za-z0-9_\[\], |]*)`)
	if matches := typeRe.FindStringSubmatch(nodeContent); len(matches) > 1 {
		typeName := strings.TrimSpace(matches[1])
		fieldSchema := e.pythonTypeToSchema(typeName)
		schema.AddProperty(fieldName, fieldSchema, true) // Pydantic fields are required by default
	}
}

func (e *FastAPIExtractor) pythonTypeToSchema(typeName string) *contracts.TypeSchema {
	// Map Python types to TypeSchema
	typeName = strings.TrimSpace(typeName)

	// Handle Optional[T]
	if strings.HasPrefix(typeName, "Optional[") {
		inner := typeName[9 : len(typeName)-1]
		schema := e.pythonTypeToSchema(inner)
		schema.Nullable = true
		return schema
	}

	// Handle List[T]
	if strings.HasPrefix(typeName, "List[") || strings.HasPrefix(typeName, "list[") {
		inner := typeName[5 : len(typeName)-1]
		return contracts.NewArraySchema(e.pythonTypeToSchema(inner))
	}

	// Handle basic types
	switch strings.ToLower(typeName) {
	case "str", "string":
		return contracts.NewPrimitiveSchema(contracts.SchemaTypeString)
	case "int", "integer":
		return contracts.NewPrimitiveSchema(contracts.SchemaTypeInteger)
	case "float", "double", "decimal":
		return contracts.NewPrimitiveSchema(contracts.SchemaTypeNumber)
	case "bool", "boolean":
		return contracts.NewPrimitiveSchema(contracts.SchemaTypeBoolean)
	case "dict", "dict[str, any]":
		return contracts.NewObjectSchema()
	case "any":
		return contracts.NewPrimitiveSchema(contracts.SchemaTypeAny)
	default:
		// Unknown type - could be a reference to another model
		return contracts.NewPrimitiveSchema(contracts.SchemaTypeUnknown)
	}
}

func init() {
	RegisterEndpointExtractor(NewFastAPIExtractor())
}
