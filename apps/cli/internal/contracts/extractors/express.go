package extractors

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/analysis"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/contracts"
)

// ExpressExtractor extracts API endpoints from Express.js applications.
type ExpressExtractor struct {
	parser        *sitter.Parser
	typeExtractor *TSTypeExtractor
}

// NewExpressExtractor creates a new Express endpoint extractor.
func NewExpressExtractor() *ExpressExtractor {
	p := sitter.NewParser()
	p.SetLanguage(typescript.GetLanguage())
	return &ExpressExtractor{
		parser:        p,
		typeExtractor: NewTSTypeExtractor(),
	}
}

// ID returns the unique identifier for this extractor.
func (e *ExpressExtractor) ID() string {
	return "express"
}

// Framework returns the framework name.
func (e *ExpressExtractor) Framework() string {
	return "express"
}

// Languages returns the languages this extractor supports.
func (e *ExpressExtractor) Languages() []string {
	return []string{"typescript", "javascript"}
}

// CanExtract returns true if this extractor can handle the given file.
func (e *ExpressExtractor) CanExtract(file *analysis.FileAnalysis) bool {
	return file.Language == "typescript" || file.Language == "javascript"
}

// ExtractEndpoints extracts API endpoints from Express source code.
func (e *ExpressExtractor) ExtractEndpoints(file *analysis.FileAnalysis) ([]ExtractedEndpoint, error) {
	return e.ExtractEndpointsFromContent([]byte{}, file.Path)
}

// ExtractEndpointsFromContent extracts endpoints directly from source content.
func (e *ExpressExtractor) ExtractEndpointsFromContent(content []byte, filePath string) ([]ExtractedEndpoint, error) {
	tree, err := e.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	var endpoints []ExtractedEndpoint
	e.walkTree(tree.RootNode(), content, filePath, &endpoints)
	return endpoints, nil
}

func (e *ExpressExtractor) walkTree(node *sitter.Node, content []byte, filePath string, endpoints *[]ExtractedEndpoint) {
	nodeType := node.Type()

	// Look for call expressions that might be Express route registrations
	if nodeType == "call_expression" {
		if endpoint := e.parseCallExpression(node, content, filePath); endpoint != nil {
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

func (e *ExpressExtractor) parseCallExpression(node *sitter.Node, content []byte, filePath string) *ExtractedEndpoint {
	// Get the function being called
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return nil
	}

	argsNode := node.ChildByFieldName("arguments")
	if argsNode == nil {
		return nil
	}

	funcType := funcNode.Type()

	// Check for member_expression: app.get, router.post, etc.
	if funcType == "member_expression" {
		return e.parseMemberExpression(funcNode, argsNode, content, filePath, node)
	}

	return nil
}

func (e *ExpressExtractor) parseMemberExpression(funcNode, argsNode *sitter.Node, content []byte, filePath string, callNode *sitter.Node) *ExtractedEndpoint {
	// Get object.method
	objectNode := funcNode.ChildByFieldName("object")
	propertyNode := funcNode.ChildByFieldName("property")

	if objectNode == nil || propertyNode == nil {
		return nil
	}

	objectName := objectNode.Content(content)
	methodName := propertyNode.Content(content)

	// Check for HTTP method patterns
	method := e.detectHTTPMethod(methodName)
	if method == "" {
		// Check for use() which might define sub-routes
		if methodName == "use" {
			return e.parseUseRoute(argsNode, content, filePath, callNode, objectName)
		}
		return nil
	}

	// Extract route path and handler
	return e.parseRouteCall(method, argsNode, content, filePath, callNode, objectName)
}

func (e *ExpressExtractor) detectHTTPMethod(name string) string {
	methods := map[string]string{
		"get":     "GET",
		"post":    "POST",
		"put":     "PUT",
		"delete":  "DELETE",
		"patch":   "PATCH",
		"head":    "HEAD",
		"options": "OPTIONS",
		"all":     "ANY",
	}

	if method, ok := methods[strings.ToLower(name)]; ok {
		return method
	}
	return ""
}

func (e *ExpressExtractor) parseRouteCall(method string, argsNode *sitter.Node, content []byte, filePath string, callNode *sitter.Node, receiver string) *ExtractedEndpoint {
	// Express routes: app.get('/path', handler) or app.get('/path', middleware, handler)
	args := e.extractArguments(argsNode, content)
	if len(args) < 2 {
		return nil
	}

	path := e.cleanStringLiteral(args[0])
	if path == "" {
		return nil
	}

	// Last argument is usually the handler
	handler := args[len(args)-1]
	pathParams := contracts.ExtractPathParams(path)

	return &ExtractedEndpoint{
		Method:     method,
		Path:       path,
		PathParams: pathParams,
		Handler:    handler,
		File:       filePath,
		Line:       int(callNode.StartPoint().Row) + 1,
		Framework:  "express",
	}
}

func (e *ExpressExtractor) parseUseRoute(argsNode *sitter.Node, content []byte, filePath string, callNode *sitter.Node, receiver string) *ExtractedEndpoint {
	// Express use: app.use('/api', apiRouter) - mounts a router
	args := e.extractArguments(argsNode, content)
	if len(args) < 1 {
		return nil
	}

	// First arg might be path or just middleware
	firstArg := args[0]
	if strings.HasPrefix(firstArg, "\"") || strings.HasPrefix(firstArg, "'") || strings.HasPrefix(firstArg, "`") {
		path := e.cleanStringLiteral(firstArg)
		if path != "" && strings.HasPrefix(path, "/") {
			// This is a path-based use(), which mounts a router
			handler := ""
			if len(args) > 1 {
				handler = args[1]
			}

			return &ExtractedEndpoint{
				Method:     "USE", // Special method to indicate router mounting
				Path:       path,
				PathParams: contracts.ExtractPathParams(path),
				Handler:    handler,
				File:       filePath,
				Line:       int(callNode.StartPoint().Row) + 1,
				Framework:  "express",
			}
		}
	}

	return nil
}

func (e *ExpressExtractor) extractArguments(argsNode *sitter.Node, content []byte) []string {
	var args []string
	for i := 0; i < int(argsNode.ChildCount()); i++ {
		child := argsNode.Child(i)
		if child == nil {
			continue
		}
		childType := child.Type()
		// Skip parentheses and commas
		if childType == "(" || childType == ")" || childType == "," {
			continue
		}
		args = append(args, child.Content(content))
	}
	return args
}

func (e *ExpressExtractor) cleanStringLiteral(s string) string {
	s = strings.TrimSpace(s)
	// Handle double quotes
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") && len(s) >= 2 {
		return s[1 : len(s)-1]
	}
	// Handle single quotes
	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") && len(s) >= 2 {
		return s[1 : len(s)-1]
	}
	// Handle template literals
	if strings.HasPrefix(s, "`") && strings.HasSuffix(s, "`") && len(s) >= 2 {
		return s[1 : len(s)-1]
	}
	return ""
}

func init() {
	RegisterEndpointExtractor(NewExpressExtractor())
}
