package analysis

import (
	"fmt"
	"path/filepath"
)

// Parser Priority Strategy (IMPLEMENTED):
//
// 1. LSP-First: Use language servers when available (most accurate, semantic understanding)
//    - Go: gopls (requires Go 1.20+, tested with gopls v0.13.0+)
//    - TypeScript/JavaScript: typescript-language-server (planned)
//    - Python: pyright, pylsp (planned)
//    - Rust: rust-analyzer (planned)
//    - Java: jdtls (planned)
//    - C/C++: clangd (planned)
//
// 2. Tree-Sitter: Fallback when LSP unavailable (good AST parsing, requires CGO)
//    - Current implementation for 30+ languages
//    - Requires C compiler (gcc/MinGW on Windows)
//
// 3. Regex: Last resort for basic symbol extraction (works everywhere)
//    - Currently: Dart, CUE
//    - Good for simple languages or when no better option exists

// Parser is the interface implemented by all language-specific parsers.
type Parser interface {
	Parse(content []byte, filePath string) (*FileAnalysis, error)
	Language() Language
}

// ParserPriority defines the priority order of parsers
type ParserPriority int

// Priorities for parser selection.
const (
	PriorityLSP        ParserPriority = 1
	PriorityTreeSitter ParserPriority = 2
	PriorityRegex      ParserPriority = 3
)

// LSPParser extends Parser with availability check
type LSPParser interface {
	Parser
	IsAvailable() bool
}

// parserEntry holds a parser with its priority
type parserEntry struct {
	parser   Parser
	priority ParserPriority
}

// ParserRegistry manages all registered parsers and their priorities.
type ParserRegistry struct {
	parsers   map[Language][]parserEntry
	rootPath  string
	enableLSP bool
	debugMode bool
}

// NewParserRegistry creates a new registry with default parsers.
func NewParserRegistry() *ParserRegistry {
	reg := &ParserRegistry{
		parsers:   make(map[Language][]parserEntry),
		enableLSP: true,
		debugMode: false,
	}
	reg.registerDefaults()
	return reg
}

// NewParserRegistryWithPath creates a registry with a root path for LSP parsers
func NewParserRegistryWithPath(rootPath string) *ParserRegistry {
	reg := &ParserRegistry{
		parsers:   make(map[Language][]parserEntry),
		rootPath:  rootPath,
		enableLSP: true,
		debugMode: false,
	}
	reg.registerDefaults()
	return reg
}

// SetDebugMode enables debug logging
func (r *ParserRegistry) SetDebugMode(enabled bool) {
	r.debugMode = enabled
}

// SetEnableLSP enables or disables LSP parsers
func (r *ParserRegistry) SetEnableLSP(enabled bool) {
	r.enableLSP = enabled
}

func (r *ParserRegistry) registerDefaults() {
	// LSP parsers - Priority 1 (when available)
	r.RegisterWithPriority(NewGoLSPParser(r.rootPath), PriorityLSP)

	// Tree-sitter parsers - Priority 2
	r.RegisterWithPriority(NewGoParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewJavaScriptParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewTypeScriptParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewPythonParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewRustParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewJavaParser(), PriorityTreeSitter)

	// C family
	r.RegisterWithPriority(NewCParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewCPPParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewCSharpParser(), PriorityTreeSitter)

	// Backend languages
	r.RegisterWithPriority(NewRubyParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewPHPParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewKotlinParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewScalaParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewSwiftParser(), PriorityTreeSitter)

	// Infrastructure/scripting
	r.RegisterWithPriority(NewBashParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewSQLParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewDockerfileParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewHCLParser(), PriorityTreeSitter)

	// Config/web
	r.RegisterWithPriority(NewHTMLParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewCSSParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewYAMLParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewTOMLParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewJSONParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewMarkdownParser(), PriorityTreeSitter)

	// Other languages
	r.RegisterWithPriority(NewElixirParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewLuaParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewGroovyParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewSvelteParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewOCamlParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewElmParser(), PriorityTreeSitter)
	r.RegisterWithPriority(NewProtobufParser(), PriorityTreeSitter)

	// Regex-based parsers - Priority 3
	r.RegisterWithPriority(NewDartParser(), PriorityRegex)
	r.RegisterWithPriority(NewCUEParser(), PriorityRegex)
}

// Register adds a parser to the registry with default Tree-sitter priority.
func (r *ParserRegistry) Register(p Parser) {
	r.RegisterWithPriority(p, PriorityTreeSitter)
}

// RegisterWithPriority registers a parser with a specific priority
func (r *ParserRegistry) RegisterWithPriority(p Parser, priority ParserPriority) {
	lang := p.Language()
	r.parsers[lang] = append(r.parsers[lang], parserEntry{
		parser:   p,
		priority: priority,
	})

	// Sort by priority (lower number = higher priority)
	entries := r.parsers[lang]
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].priority < entries[i].priority {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	r.parsers[lang] = entries
}

// GetParser returns the highest priority parser available for the given language.
func (r *ParserRegistry) GetParser(lang Language) (Parser, bool) {
	entries, ok := r.parsers[lang]
	if !ok || len(entries) == 0 {
		return nil, false
	}

	// Try parsers in priority order
	for _, entry := range entries {
		// Skip LSP parsers if disabled
		if entry.priority == PriorityLSP && !r.enableLSP {
			continue
		}

		// Check if LSP parser is available
		lspParser, ok := entry.parser.(LSPParser)
		if ok { //nolint:nestif // acceptable complexity for parser selection logic
			if !lspParser.IsAvailable() {
				if r.debugMode {
					fmt.Printf("[DEBUG] LSP parser for %s not available, trying fallback\n", lang)
				}
				continue
			}
			if r.debugMode {
				fmt.Printf("[DEBUG] Using LSP parser for %s\n", lang)
			}
		} else if r.debugMode {
			fmt.Printf("[DEBUG] Using %s parser (priority %d) for %s\n",
				r.getPriorityName(entry.priority), entry.priority, lang)
		}

		return entry.parser, true
	}

	// No available parser found
	return nil, false
}

func (r *ParserRegistry) getPriorityName(priority ParserPriority) string {
	switch priority {
	case PriorityLSP:
		return "LSP"
	case PriorityTreeSitter:
		return "Tree-sitter"
	case PriorityRegex:
		return "Regex"
	default:
		return "Unknown"
	}
}

// Parse analyzes the content of a file and returns the symbol extraction results.
func (r *ParserRegistry) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	lang := DetectLanguage(filePath)
	if lang == LangUnknown {
		return &FileAnalysis{
			Path:     filePath,
			Language: string(LangUnknown),
		}, nil
	}

	parser, ok := r.GetParser(lang)
	if !ok {
		return &FileAnalysis{
			Path:     filePath,
			Language: string(lang),
		}, nil
	}

	// Try to parse with selected parser
	analysis, err := parser.Parse(content, filePath)

	// If LSP parser failed, try fallback
	if err != nil { //nolint:nestif // acceptable complexity for fallback logic
		lspParser, ok := parser.(LSPParser)
		if ok && lspParser.IsAvailable() {
			if r.debugMode {
				fmt.Printf("[DEBUG] LSP parser failed for %s: %v, trying fallback\n", lang, err)
			}

			// Try next parser in priority order
			entries := r.parsers[lang]
			for i, entry := range entries {
				if entry.parser == parser && i+1 < len(entries) {
					fallbackParser := entries[i+1].parser
					if r.debugMode {
						fmt.Printf("[DEBUG] Falling back to %s parser\n",
							r.getPriorityName(entries[i+1].priority))
					}
					return fallbackParser.Parse(content, filePath)
				}
			}
		}
	}

	return analysis, err
}

var defaultRegistry *ParserRegistry

func init() {
	defaultRegistry = NewParserRegistryWithPath("")
}

// Analyze is a convenience function that uses the default registry to analyze a file.
func Analyze(content []byte, filePath string) (*FileAnalysis, error) {
	// Update root path if analyzing a real file
	if defaultRegistry.rootPath == "" && filePath != "" {
		defaultRegistry.rootPath = filepath.Dir(filePath)
	}
	return defaultRegistry.Parse(content, filePath)
}
