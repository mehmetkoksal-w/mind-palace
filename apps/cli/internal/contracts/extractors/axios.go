package extractors

import (
	"context"
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/analysis"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/contracts"
)

// AxiosExtractor extracts API calls using axios.
type AxiosExtractor struct {
	parser        *sitter.Parser
	typeExtractor *TSTypeExtractor
}

// NewAxiosExtractor creates a new axios API call extractor.
func NewAxiosExtractor() *AxiosExtractor {
	p := sitter.NewParser()
	p.SetLanguage(typescript.GetLanguage())
	return &AxiosExtractor{
		parser:        p,
		typeExtractor: NewTSTypeExtractor(),
	}
}

// ID returns the unique identifier for this extractor.
func (e *AxiosExtractor) ID() string {
	return "axios"
}

// CallType returns the type of calls extracted.
func (e *AxiosExtractor) CallType() string {
	return "axios"
}

// Languages returns the languages this extractor supports.
func (e *AxiosExtractor) Languages() []string {
	return []string{"typescript", "javascript"}
}

// CanExtract returns true if this extractor can handle the given file.
func (e *AxiosExtractor) CanExtract(file *analysis.FileAnalysis) bool {
	return file.Language == "typescript" || file.Language == "javascript"
}

// ExtractCalls extracts axios API calls from source code.
func (e *AxiosExtractor) ExtractCalls(file *analysis.FileAnalysis) ([]ExtractedCall, error) {
	return e.ExtractCallsFromContent([]byte{}, file.Path)
}

// ExtractCallsFromContent extracts axios calls directly from source content.
func (e *AxiosExtractor) ExtractCallsFromContent(content []byte, filePath string) ([]ExtractedCall, error) {
	tree, err := e.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	var calls []ExtractedCall
	e.walkTree(tree.RootNode(), content, filePath, &calls)
	return calls, nil
}

func (e *AxiosExtractor) walkTree(node *sitter.Node, content []byte, filePath string, calls *[]ExtractedCall) {
	nodeType := node.Type()

	// Look for call expressions
	if nodeType == "call_expression" {
		if call := e.parseCallExpression(node, content, filePath); call != nil {
			*calls = append(*calls, *call)
		}
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			e.walkTree(child, content, filePath, calls)
		}
	}
}

func (e *AxiosExtractor) parseCallExpression(node *sitter.Node, content []byte, filePath string) *ExtractedCall {
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return nil
	}

	argsNode := node.ChildByFieldName("arguments")
	if argsNode == nil {
		return nil
	}

	funcContent := funcNode.Content(content)

	// Check for axios() direct call
	if funcContent == "axios" {
		return e.parseAxiosDirectCall(argsNode, content, filePath, node)
	}

	// Check for member expression patterns
	if funcNode.Type() == "member_expression" {
		objectNode := funcNode.ChildByFieldName("object")
		propertyNode := funcNode.ChildByFieldName("property")

		if objectNode == nil || propertyNode == nil {
			return nil
		}

		objectName := objectNode.Content(content)
		methodName := propertyNode.Content(content)

		// Check for axios.get, axios.post, etc.
		if objectName == "axios" || strings.HasSuffix(objectName, ".axios") || objectName == "api" || objectName == "http" {
			return e.parseAxiosMethodCall(methodName, argsNode, content, filePath, node)
		}

		// Check for instance methods: instance.get, client.post
		if e.isHTTPMethod(methodName) {
			return e.parseAxiosMethodCall(methodName, argsNode, content, filePath, node)
		}
	}

	return nil
}

func (e *AxiosExtractor) parseAxiosDirectCall(argsNode *sitter.Node, content []byte, filePath string, callNode *sitter.Node) *ExtractedCall {
	// axios(config) or axios(url, config?)
	args := e.extractArguments(argsNode, content)
	if len(args) < 1 {
		return nil
	}

	firstArg := args[0]

	// Check if first arg is a URL string
	if strings.HasPrefix(firstArg, "\"") || strings.HasPrefix(firstArg, "'") || strings.HasPrefix(firstArg, "`") {
		url, isDynamic, variables := e.parseURL(firstArg)
		method := "GET"

		// Check for config in second argument
		if len(args) > 1 {
			method = e.extractMethodFromConfig(args[1])
		}

		return &ExtractedCall{
			Method:    method,
			URL:       url,
			File:      filePath,
			Line:      int(callNode.StartPoint().Row) + 1,
			IsDynamic: isDynamic,
			Variables: variables,
		}
	}

	// First arg is a config object
	url, method := e.extractFromConfigObject(firstArg)
	if url != "" {
		_, isDynamic, variables := e.parseURL("\"" + url + "\"")
		return &ExtractedCall{
			Method:    method,
			URL:       url,
			File:      filePath,
			Line:      int(callNode.StartPoint().Row) + 1,
			IsDynamic: isDynamic,
			Variables: variables,
		}
	}

	return nil
}

func (e *AxiosExtractor) parseAxiosMethodCall(methodName string, argsNode *sitter.Node, content []byte, filePath string, callNode *sitter.Node) *ExtractedCall {
	// axios.get(url, config?) or axios.post(url, data?, config?)
	method := e.normalizeMethod(methodName)
	if method == "" {
		return nil
	}

	args := e.extractArguments(argsNode, content)
	if len(args) < 1 {
		return nil
	}

	url, isDynamic, variables := e.parseURL(args[0])

	return &ExtractedCall{
		Method:    method,
		URL:       url,
		File:      filePath,
		Line:      int(callNode.StartPoint().Row) + 1,
		IsDynamic: isDynamic,
		Variables: variables,
	}
}

func (e *AxiosExtractor) isHTTPMethod(name string) bool {
	methods := map[string]bool{
		"get":     true,
		"post":    true,
		"put":     true,
		"delete":  true,
		"patch":   true,
		"head":    true,
		"options": true,
		"request": true,
	}
	return methods[strings.ToLower(name)]
}

func (e *AxiosExtractor) normalizeMethod(name string) string {
	methods := map[string]string{
		"get":     "GET",
		"post":    "POST",
		"put":     "PUT",
		"delete":  "DELETE",
		"patch":   "PATCH",
		"head":    "HEAD",
		"options": "OPTIONS",
		"request": "GET", // Default for request()
	}
	if method, ok := methods[strings.ToLower(name)]; ok {
		return method
	}
	return ""
}

func (e *AxiosExtractor) parseURL(urlArg string) (url string, isDynamic bool, variables []string) {
	urlArg = strings.TrimSpace(urlArg)

	// Template literal
	if strings.HasPrefix(urlArg, "`") && strings.HasSuffix(urlArg, "`") {
		content := urlArg[1 : len(urlArg)-1]
		isDynamic = strings.Contains(content, "${")

		if isDynamic {
			varRe := regexp.MustCompile(`\$\{([^}]+)\}`)
			matches := varRe.FindAllStringSubmatch(content, -1)
			for _, match := range matches {
				if len(match) > 1 {
					variables = append(variables, match[1])
				}
			}
		}

		normalizedRe := regexp.MustCompile(`\$\{[^}]+\}`)
		url = normalizedRe.ReplaceAllString(content, ":param")
		return
	}

	// String literal
	if (strings.HasPrefix(urlArg, "\"") && strings.HasSuffix(urlArg, "\"")) ||
		(strings.HasPrefix(urlArg, "'") && strings.HasSuffix(urlArg, "'")) {
		if len(urlArg) >= 2 {
			url = urlArg[1 : len(urlArg)-1]
		}
		isDynamic = false
		return
	}

	// Variable or concatenation
	url = urlArg
	isDynamic = true
	variables = []string{urlArg}
	return
}

func (e *AxiosExtractor) extractMethodFromConfig(configArg string) string {
	methodRe := regexp.MustCompile(`method\s*:\s*["'](\w+)["']`)
	if matches := methodRe.FindStringSubmatch(configArg); len(matches) > 1 {
		return strings.ToUpper(matches[1])
	}
	return "GET"
}

func (e *AxiosExtractor) extractFromConfigObject(configArg string) (url string, method string) {
	method = "GET"

	// Extract url
	urlRe := regexp.MustCompile(`url\s*:\s*["']([^"']+)["']`)
	if matches := urlRe.FindStringSubmatch(configArg); len(matches) > 1 {
		url = matches[1]
	}

	// Extract method
	methodRe := regexp.MustCompile(`method\s*:\s*["'](\w+)["']`)
	if matches := methodRe.FindStringSubmatch(configArg); len(matches) > 1 {
		method = strings.ToUpper(matches[1])
	}

	return
}

func (e *AxiosExtractor) extractArguments(argsNode *sitter.Node, content []byte) []string {
	var args []string
	depth := 0

	for i := 0; i < int(argsNode.ChildCount()); i++ {
		child := argsNode.Child(i)
		if child == nil {
			continue
		}
		childType := child.Type()

		// Skip parentheses
		if childType == "(" || childType == ")" {
			continue
		}

		// Track depth for nested structures
		if childType == "{" || childType == "[" {
			depth++
			continue
		}
		if childType == "}" || childType == "]" {
			depth--
			continue
		}

		// Skip commas at depth 0
		if childType == "," && depth == 0 {
			continue
		}

		args = append(args, child.Content(content))
	}
	return args
}

// ExtractTypeFromGeneric attempts to extract the expected response type from
// axios generic type parameters like axios.get<User>('/users').
func (e *AxiosExtractor) ExtractTypeFromGeneric(content []byte, callLine int) *contracts.TypeSchema {
	// This is a simplified implementation
	// Full implementation would parse the generic type parameter
	return nil
}

func init() {
	RegisterCallExtractor(NewAxiosExtractor())
}
