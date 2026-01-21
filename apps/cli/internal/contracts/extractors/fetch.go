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

// FetchExtractor extracts API calls using the fetch() API.
type FetchExtractor struct {
	parser        *sitter.Parser
	typeExtractor *TSTypeExtractor
}

// NewFetchExtractor creates a new Fetch API call extractor.
func NewFetchExtractor() *FetchExtractor {
	p := sitter.NewParser()
	p.SetLanguage(typescript.GetLanguage())
	return &FetchExtractor{
		parser:        p,
		typeExtractor: NewTSTypeExtractor(),
	}
}

// ID returns the unique identifier for this extractor.
func (e *FetchExtractor) ID() string {
	return "fetch"
}

// CallType returns the type of calls extracted.
func (e *FetchExtractor) CallType() string {
	return "fetch"
}

// Languages returns the languages this extractor supports.
func (e *FetchExtractor) Languages() []string {
	return []string{"typescript", "javascript"}
}

// CanExtract returns true if this extractor can handle the given file.
func (e *FetchExtractor) CanExtract(file *analysis.FileAnalysis) bool {
	return file.Language == "typescript" || file.Language == "javascript"
}

// ExtractCalls extracts fetch API calls from source code.
func (e *FetchExtractor) ExtractCalls(file *analysis.FileAnalysis) ([]ExtractedCall, error) {
	return e.ExtractCallsFromContent([]byte{}, file.Path)
}

// ExtractCallsFromContent extracts fetch calls directly from source content.
func (e *FetchExtractor) ExtractCallsFromContent(content []byte, filePath string) ([]ExtractedCall, error) {
	tree, err := e.parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	var calls []ExtractedCall
	e.walkTree(tree.RootNode(), content, filePath, &calls)
	return calls, nil
}

func (e *FetchExtractor) walkTree(node *sitter.Node, content []byte, filePath string, calls *[]ExtractedCall) {
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

func (e *FetchExtractor) parseCallExpression(node *sitter.Node, content []byte, filePath string) *ExtractedCall {
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return nil
	}

	argsNode := node.ChildByFieldName("arguments")
	if argsNode == nil {
		return nil
	}

	funcName := funcNode.Content(content)

	// Check for direct fetch() call
	if funcName == "fetch" {
		return e.parseFetchCall(argsNode, content, filePath, node)
	}

	// Check for wrapper patterns like api.fetch, this.fetch, http.fetch
	if funcNode.Type() == "member_expression" {
		propertyNode := funcNode.ChildByFieldName("property")
		if propertyNode != nil && propertyNode.Content(content) == "fetch" {
			return e.parseFetchCall(argsNode, content, filePath, node)
		}
	}

	return nil
}

func (e *FetchExtractor) parseFetchCall(argsNode *sitter.Node, content []byte, filePath string, callNode *sitter.Node) *ExtractedCall {
	// fetch(url, options?) or fetch(url, { method: 'POST', ... })
	args := e.extractArguments(argsNode, content)
	if len(args) < 1 {
		return nil
	}

	// First argument is URL
	urlArg := args[0]
	url, isDynamic, variables := e.parseURL(urlArg)

	// Default method is GET
	method := "GET"

	// Check for options object
	if len(args) > 1 {
		method = e.extractMethodFromOptions(args[1], content)
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

func (e *FetchExtractor) parseURL(urlArg string) (url string, isDynamic bool, variables []string) {
	urlArg = strings.TrimSpace(urlArg)

	// Check for template literal: `${baseUrl}/users/${id}`
	if strings.HasPrefix(urlArg, "`") && strings.HasSuffix(urlArg, "`") {
		content := urlArg[1 : len(urlArg)-1]
		isDynamic = strings.Contains(content, "${")

		// Extract variable names
		if isDynamic {
			varRe := regexp.MustCompile(`\$\{([^}]+)\}`)
			matches := varRe.FindAllStringSubmatch(content, -1)
			for _, match := range matches {
				if len(match) > 1 {
					variables = append(variables, match[1])
				}
			}
		}

		// Normalize URL by replacing ${...} with placeholders
		normalizedRe := regexp.MustCompile(`\$\{[^}]+\}`)
		url = normalizedRe.ReplaceAllString(content, ":param")
		return
	}

	// Check for string literal: "/api/users" or '/api/users'
	if (strings.HasPrefix(urlArg, "\"") && strings.HasSuffix(urlArg, "\"")) ||
		(strings.HasPrefix(urlArg, "'") && strings.HasSuffix(urlArg, "'")) {
		url = urlArg[1 : len(urlArg)-1]
		isDynamic = false
		return
	}

	// Check for string concatenation or variable
	if strings.Contains(urlArg, "+") {
		isDynamic = true
		// Try to extract literal parts
		literalRe := regexp.MustCompile(`["']([^"']+)["']`)
		matches := literalRe.FindAllStringSubmatch(urlArg, -1)
		var parts []string
		for _, match := range matches {
			if len(match) > 1 {
				parts = append(parts, match[1])
			}
		}
		url = strings.Join(parts, ":param")
		return
	}

	// Variable reference
	url = urlArg
	isDynamic = true
	variables = []string{urlArg}
	return
}

func (e *FetchExtractor) extractMethodFromOptions(optionsArg string, _ []byte) string {
	// Look for method property in object literal
	// { method: 'POST' } or { method: "POST" }
	methodRe := regexp.MustCompile(`method\s*:\s*["'](\w+)["']`)
	if matches := methodRe.FindStringSubmatch(optionsArg); len(matches) > 1 {
		return strings.ToUpper(matches[1])
	}
	return "GET"
}

func (e *FetchExtractor) extractArguments(argsNode *sitter.Node, content []byte) []string {
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

		// Handle nested structures
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

		// Capture the argument content
		args = append(args, child.Content(content))
	}
	return args
}

// ExtractTypeFromThen attempts to extract the expected response type from
// a .then() chain or await expression.
func (e *FetchExtractor) ExtractTypeFromThen(_ []byte, _ int) *contracts.TypeSchema {
	// This is a simplified implementation
	// Full implementation would track the call chain and type annotations
	// Example: fetch(url).then(r => r.json()).then((data: User) => ...)
	// or: const data: User = await fetch(url).then(r => r.json())
	return nil
}

func init() {
	RegisterCallExtractor(NewFetchExtractor())
}
