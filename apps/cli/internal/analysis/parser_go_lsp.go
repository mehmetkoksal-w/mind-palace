package analysis

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

// GoLSPParser uses gopls (Go language server) to parse Go files
type GoLSPParser struct {
	available bool
	rootPath  string
}

// NewGoLSPParser creates a new Go LSP parser
func NewGoLSPParser(rootPath string) *GoLSPParser {
	// Check if gopls is available
	_, err := exec.LookPath("gopls")
	return &GoLSPParser{
		available: err == nil,
		rootPath:  rootPath,
	}
}

// IsAvailable returns whether gopls is available
func (p *GoLSPParser) IsAvailable() bool {
	return p.available
}

// Language returns the language this parser handles
func (p *GoLSPParser) Language() Language {
	return LangGo
}

// Parse parses a Go file using gopls
func (p *GoLSPParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	if !p.available {
		return nil, fmt.Errorf("gopls not available")
	}

	analysis := &FileAnalysis{
		Path:     filePath,
		Language: string(LangGo),
	}

	// Determine root path (use provided or current directory)
	rootPath := p.rootPath
	if rootPath == "" {
		rootPath = filepath.Dir(filePath)
	}

	// Create LSP client
	client, err := NewLSPClient(LSPClientConfig{
		ServerCmd:  "gopls",
		ServerArgs: []string{},
		RootPath:   rootPath,
		LanguageID: "go",
	})
	if err != nil {
		return nil, fmt.Errorf("create LSP client: %w", err)
	}
	defer client.Close()

	// Get absolute path and convert to URI
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}
	uri := pathToURI(absPath)

	// Request document symbols
	lspSymbols, err := client.DocumentSymbols(uri, string(content))
	if err != nil {
		return nil, fmt.Errorf("get document symbols: %w", err)
	}

	// Convert LSP symbols to our format
	analysis.Symbols = p.convertSymbols(lspSymbols, content)

	// Extract relationships (imports, calls)
	p.extractRelationships(content, analysis)

	return analysis, nil
}

// convertSymbols recursively converts LSP document symbols to our Symbol format
func (p *GoLSPParser) convertSymbols(lspSymbols []LSPDocumentSymbol, content []byte) []Symbol {
	symbols := make([]Symbol, 0, len(lspSymbols))

	for i := range lspSymbols {
		symbol := p.convertSymbol(lspSymbols[i], content)
		symbols = append(symbols, symbol)
	}

	return symbols
}

// convertSymbol converts a single LSP symbol (with children) to our format
func (p *GoLSPParser) convertSymbol(lspSym LSPDocumentSymbol, content []byte) Symbol {
	symbol := Symbol{
		Name:      lspSym.Name,
		Kind:      ConvertLSPSymbolKind(lspSym.Kind),
		LineStart: lspSym.Range.Start.Line + 1, // LSP uses 0-based lines
		LineEnd:   lspSym.Range.End.Line + 1,
		Signature: p.extractSignature(lspSym, content),
		Exported:  isExported(lspSym.Name),
	}

	// Handle special cases for Go
	symbol.Kind = p.refineSymbolKind(lspSym, symbol.Kind)

	// Convert children recursively
	if len(lspSym.Children) > 0 {
		symbol.Children = make([]Symbol, len(lspSym.Children))
		for i := range lspSym.Children {
			symbol.Children[i] = p.convertSymbol(lspSym.Children[i], content)
		}
	}

	return symbol
}

// refineSymbolKind refines the symbol kind based on Go-specific patterns
func (p *GoLSPParser) refineSymbolKind(lspSym LSPDocumentSymbol, kind SymbolKind) SymbolKind {
	// gopls reports structs as Class
	if lspSym.Kind == LSPSymbolKindClass || lspSym.Kind == LSPSymbolKindStruct {
		return KindClass
	}

	// gopls reports type aliases as Class sometimes, refine based on detail
	if lspSym.Kind == LSPSymbolKindClass && strings.Contains(lspSym.Detail, "type") {
		return KindType
	}

	// Methods have receivers in their detail
	if lspSym.Kind == LSPSymbolKindMethod {
		return KindMethod
	}

	// Functions vs methods
	if lspSym.Kind == LSPSymbolKindFunction {
		// If it has a receiver (indicated in detail), it's a method
		if strings.Contains(lspSym.Detail, ")") && strings.Contains(lspSym.Detail, "(") {
			// Check if there's a receiver pattern like "(r *Receiver)"
			detail := strings.TrimSpace(lspSym.Detail)
			if strings.HasPrefix(detail, "(") {
				return KindMethod
			}
		}
		return KindFunction
	}

	return kind
}

// extractSignature extracts function/method signature from LSP symbol
func (p *GoLSPParser) extractSignature(lspSym LSPDocumentSymbol, content []byte) string {
	// For functions and methods, use the detail field if available
	if lspSym.Detail != "" {
		// gopls provides signatures in the detail field
		// Format: "func(params) returns" or "(receiver) func(params) returns"
		return lspSym.Detail
	}

	// Fallback: extract from source code
	if lspSym.Range.Start.Line < lspSym.Range.End.Line {
		// Multi-line symbol, extract first line
		lines := strings.Split(string(content), "\n")
		if lspSym.Range.Start.Line < len(lines) {
			line := lines[lspSym.Range.Start.Line]
			return strings.TrimSpace(line)
		}
	}

	return lspSym.Name
}

// extractRelationships extracts imports and basic call relationships
func (p *GoLSPParser) extractRelationships(content []byte, analysis *FileAnalysis) {
	lines := strings.Split(string(content), "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Extract imports
		if strings.HasPrefix(line, "import ") {
			// Single import: import "path"
			if start := strings.Index(line, `"`); start != -1 {
				if end := strings.Index(line[start+1:], `"`); end != -1 {
					importPath := line[start+1 : start+1+end]
					analysis.Relationships = append(analysis.Relationships, Relationship{
						TargetFile: importPath,
						Kind:       RelImport,
						Line:       i + 1,
					})
				}
			}
		} else if line == "import (" {
			// Multi-line imports
			for j := i + 1; j < len(lines); j++ {
				importLine := strings.TrimSpace(lines[j])
				if importLine == ")" {
					break
				}
				if start := strings.Index(importLine, `"`); start != -1 {
					if end := strings.Index(importLine[start+1:], `"`); end != -1 {
						importPath := importLine[start+1 : start+1+end]
						analysis.Relationships = append(analysis.Relationships, Relationship{
							TargetFile: importPath,
							Kind:       RelImport,
							Line:       j + 1,
						})
					}
				}
			}
		}

		// Extract basic function calls (simple heuristic)
		// Look for patterns like: functionName() or obj.Method()
		p.extractCallsFromLine(line, i+1, analysis)
	}
}

// extractCallsFromLine extracts function/method calls from a line
func (p *GoLSPParser) extractCallsFromLine(line string, lineNum int, analysis *FileAnalysis) {
	// Skip comments, imports, and declarations
	if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") ||
		strings.HasPrefix(line, "import") || strings.HasPrefix(line, "package") ||
		strings.HasPrefix(line, "func ") || strings.HasPrefix(line, "type ") ||
		strings.HasPrefix(line, "var ") || strings.HasPrefix(line, "const ") {
		return
	}

	// Find function calls: identifier followed by '('
	// This is a simple heuristic and won't catch all cases
	i := 0
	for i < len(line) {
		// Skip strings
		if line[i] == '"' || line[i] == '`' {
			i++
			for i < len(line) && line[i] != '"' && line[i] != '`' {
				if line[i] == '\\' {
					i++ // Skip escaped character
				}
				i++
			}
			i++
			continue
		}

		// Look for identifier
		if unicode.IsLetter(rune(line[i])) || line[i] == '_' {
			start := i
			for i < len(line) && (unicode.IsLetter(rune(line[i])) || unicode.IsDigit(rune(line[i])) || line[i] == '_') {
				i++
			}
			identifier := line[start:i]

			// Check for method call (obj.Method)
			if i < len(line) && line[i] == '.' {
				i++ // Skip '.'
				if i < len(line) && (unicode.IsLetter(rune(line[i])) || line[i] == '_') {
					methodStart := i
					for i < len(line) && (unicode.IsLetter(rune(line[i])) || unicode.IsDigit(rune(line[i])) || line[i] == '_') {
						i++
					}
					method := line[methodStart:i]

					// Check if followed by '('
					for i < len(line) && line[i] == ' ' {
						i++
					}
					if i < len(line) && line[i] == '(' {
						// Found method call
						analysis.Relationships = append(analysis.Relationships, Relationship{
							TargetSymbol: identifier + "." + method,
							Kind:         RelCall,
							Line:         lineNum,
						})
					}
				}
			} else {
				// Check for simple function call
				for i < len(line) && line[i] == ' ' {
					i++
				}
				if i < len(line) && line[i] == '(' {
					// Skip common keywords
					if identifier != "if" && identifier != "for" && identifier != "switch" &&
						identifier != "return" && identifier != "defer" && identifier != "go" &&
						identifier != "make" && identifier != "new" && identifier != "len" &&
						identifier != "cap" && identifier != "append" && identifier != "copy" &&
						identifier != "delete" && identifier != "panic" && identifier != "recover" {
						// Found function call
						analysis.Relationships = append(analysis.Relationships, Relationship{
							TargetSymbol: identifier,
							Kind:         RelCall,
							Line:         lineNum,
						})
					}
				}
			}
		}
		i++
	}
}
