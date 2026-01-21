package extractors

import (
	"context"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/analysis"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/contracts"
)

// GoHTTPExtractor extracts API endpoints from Go HTTP handler code.
// Supports: net/http, gorilla/mux, gin, echo
type GoHTTPExtractor struct {
	parser        *sitter.Parser
	typeExtractor *GoTypeExtractor
}

// NewGoHTTPExtractor creates a new Go HTTP endpoint extractor.
func NewGoHTTPExtractor() *GoHTTPExtractor {
	p := sitter.NewParser()
	p.SetLanguage(golang.GetLanguage())
	return &GoHTTPExtractor{
		parser:        p,
		typeExtractor: NewGoTypeExtractor(),
	}
}

// ID returns the unique identifier for this extractor.
func (e *GoHTTPExtractor) ID() string {
	return "go-http"
}

// Framework returns the framework name.
func (e *GoHTTPExtractor) Framework() string {
	return "go-http"
}

// Languages returns the languages this extractor supports.
func (e *GoHTTPExtractor) Languages() []string {
	return []string{"go"}
}

// CanExtract returns true if this extractor can handle the given file.
func (e *GoHTTPExtractor) CanExtract(file *analysis.FileAnalysis) bool {
	return file.Language == "go"
}

// ExtractEndpoints extracts API endpoints from Go source code.
func (e *GoHTTPExtractor) ExtractEndpoints(file *analysis.FileAnalysis) ([]ExtractedEndpoint, error) {
	return e.ExtractEndpointsFromContent([]byte{}, file.Path)
}

// ExtractEndpointsFromContent extracts endpoints directly from source content.
func (e *GoHTTPExtractor) ExtractEndpointsFromContent(content []byte, filePath string) ([]ExtractedEndpoint, error) {
	tree, err := e.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	var endpoints []ExtractedEndpoint
	e.walkTree(tree.RootNode(), content, filePath, &endpoints)
	return endpoints, nil
}

func (e *GoHTTPExtractor) walkTree(node *sitter.Node, content []byte, filePath string, endpoints *[]ExtractedEndpoint) {
	nodeType := node.Type()

	// Look for call expressions that might be HTTP route registrations
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

func (e *GoHTTPExtractor) parseCallExpression(node *sitter.Node, content []byte, filePath string) *ExtractedEndpoint {
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
	funcContent := funcNode.Content(content)

	// Check for different patterns
	switch funcType {
	case "selector_expression":
		return e.parseSelectorCall(funcNode, argsNode, content, filePath, node)
	case "identifier":
		// Direct function call like HandleFunc
		if funcContent == "HandleFunc" {
			return e.parseHandleFuncCall(argsNode, content, filePath, node, "")
		}
	}

	return nil
}

func (e *GoHTTPExtractor) parseSelectorCall(funcNode, argsNode *sitter.Node, content []byte, filePath string, callNode *sitter.Node) *ExtractedEndpoint {
	// Get object.method
	var objectName, methodName string

	for i := 0; i < int(funcNode.ChildCount()); i++ {
		child := funcNode.Child(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "identifier", "field_identifier":
			if objectName == "" {
				objectName = child.Content(content)
			} else {
				methodName = child.Content(content)
			}
		case "selector_expression":
			// Nested selector, get the last identifier as method
			methodName = e.getLastIdentifier(child, content)
			objectName = e.getFirstIdentifier(child, content)
		}
	}

	// Check for HTTP method registration patterns
	method := e.detectHTTPMethod(methodName)
	if method != "" {
		// Gin/Echo style: r.GET("/path", handler)
		return e.parseGinEchoRoute(method, argsNode, content, filePath, callNode, objectName)
	}

	// Check for HandleFunc/Handle patterns
	if methodName == "HandleFunc" || methodName == "Handle" {
		return e.parseHandleFuncCall(argsNode, content, filePath, callNode, objectName)
	}

	// Check for gorilla/mux Methods().Path().HandlerFunc() pattern
	if methodName == "HandlerFunc" || methodName == "Handler" {
		// Look for chained Methods() and Path() calls
		return e.parseMuxChainedRoute(funcNode, argsNode, content, filePath, callNode)
	}

	return nil
}

func (e *GoHTTPExtractor) detectHTTPMethod(name string) string {
	methods := map[string]string{
		"GET":     "GET",
		"POST":    "POST",
		"PUT":     "PUT",
		"DELETE":  "DELETE",
		"PATCH":   "PATCH",
		"HEAD":    "HEAD",
		"OPTIONS": "OPTIONS",
		"Any":     "ANY", // Gin's Any() method
		"Handle":  "",    // Need to check arguments
	}

	if method, ok := methods[name]; ok {
		return method
	}
	return ""
}

func (e *GoHTTPExtractor) parseHandleFuncCall(argsNode *sitter.Node, content []byte, filePath string, callNode *sitter.Node, receiver string) *ExtractedEndpoint {
	// http.HandleFunc("/path", handler) or mux.HandleFunc("/path", handler)
	args := e.extractArguments(argsNode, content)
	if len(args) < 2 {
		return nil
	}

	path := e.cleanStringLiteral(args[0])
	if path == "" {
		return nil
	}

	handler := args[1]
	pathParams := contracts.ExtractPathParams(path)

	framework := "net/http"
	if receiver == "mux" || strings.Contains(receiver, "Router") || strings.Contains(receiver, "Mux") {
		framework = "gorilla/mux"
	}

	return &ExtractedEndpoint{
		Method:     "ANY", // HandleFunc accepts all methods
		Path:       path,
		PathParams: pathParams,
		Handler:    handler,
		File:       filePath,
		Line:       int(callNode.StartPoint().Row) + 1,
		Framework:  framework,
	}
}

func (e *GoHTTPExtractor) parseGinEchoRoute(method string, argsNode *sitter.Node, content []byte, filePath string, callNode *sitter.Node, receiver string) *ExtractedEndpoint {
	// r.GET("/path", handler) or e.GET("/path", handler)
	args := e.extractArguments(argsNode, content)
	if len(args) < 2 {
		return nil
	}

	path := e.cleanStringLiteral(args[0])
	if path == "" {
		return nil
	}

	handler := args[1]
	pathParams := contracts.ExtractPathParams(path)

	// Try to determine framework from receiver naming convention
	framework := "gin" // Default assumption
	if strings.HasPrefix(receiver, "e") || strings.Contains(strings.ToLower(receiver), "echo") {
		framework = "echo"
	}

	return &ExtractedEndpoint{
		Method:     method,
		Path:       path,
		PathParams: pathParams,
		Handler:    handler,
		File:       filePath,
		Line:       int(callNode.StartPoint().Row) + 1,
		Framework:  framework,
	}
}

func (e *GoHTTPExtractor) parseMuxChainedRoute(funcNode, argsNode *sitter.Node, content []byte, filePath string, callNode *sitter.Node) *ExtractedEndpoint {
	// mux.NewRouter().Methods("GET").Path("/users/{id}").HandlerFunc(handler)
	// We need to walk up the call chain to find Methods() and Path()

	// For now, implement a simpler pattern detection
	// Look for the pattern in the full expression
	fullExpr := funcNode.Content(content)

	// Extract method from Methods("GET", "POST")
	methodRe := regexp.MustCompile(`Methods\s*\(\s*"([^"]+)"`)
	methodMatches := methodRe.FindStringSubmatch(fullExpr)
	method := "ANY"
	if len(methodMatches) > 1 {
		method = methodMatches[1]
	}

	// Extract path from Path("/users/{id}")
	pathRe := regexp.MustCompile(`Path\s*\(\s*"([^"]+)"`)
	pathMatches := pathRe.FindStringSubmatch(fullExpr)
	if len(pathMatches) < 2 {
		return nil
	}
	path := pathMatches[1]

	// Extract handler
	args := e.extractArguments(argsNode, content)
	handler := ""
	if len(args) > 0 {
		handler = args[0]
	}

	pathParams := contracts.ExtractPathParams(path)

	return &ExtractedEndpoint{
		Method:     method,
		Path:       path,
		PathParams: pathParams,
		Handler:    handler,
		File:       filePath,
		Line:       int(callNode.StartPoint().Row) + 1,
		Framework:  "gorilla/mux",
	}
}

func (e *GoHTTPExtractor) extractArguments(argsNode *sitter.Node, content []byte) []string {
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

func (e *GoHTTPExtractor) cleanStringLiteral(s string) string {
	// Remove quotes from string literals
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return s[1 : len(s)-1]
	}
	if strings.HasPrefix(s, "`") && strings.HasSuffix(s, "`") {
		return s[1 : len(s)-1]
	}
	return ""
}

func (e *GoHTTPExtractor) getLastIdentifier(node *sitter.Node, content []byte) string {
	var last string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && (child.Type() == "identifier" || child.Type() == "field_identifier") {
			last = child.Content(content)
		}
	}
	return last
}

func (e *GoHTTPExtractor) getFirstIdentifier(node *sitter.Node, content []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			if child.Type() == "identifier" || child.Type() == "field_identifier" {
				return child.Content(content)
			}
			if child.Type() == "selector_expression" {
				return e.getFirstIdentifier(child, content)
			}
		}
	}
	return ""
}

func init() {
	RegisterEndpointExtractor(NewGoHTTPExtractor())
}
