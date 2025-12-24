package analysis

type Parser interface {
	Parse(content []byte, filePath string) (*FileAnalysis, error)
	Language() Language
}

type ParserRegistry struct {
	parsers map[Language]Parser
}

func NewParserRegistry() *ParserRegistry {
	reg := &ParserRegistry{
		parsers: make(map[Language]Parser),
	}
	reg.registerDefaults()
	return reg
}

func (r *ParserRegistry) registerDefaults() {
	// Tree-sitter parsers - existing
	r.Register(NewGoParser())
	r.Register(NewJavaScriptParser())
	r.Register(NewTypeScriptParser())
	r.Register(NewPythonParser())
	r.Register(NewRustParser())
	r.Register(NewJavaParser())

	// C family
	r.Register(NewCParser())
	r.Register(NewCPPParser())
	r.Register(NewCSharpParser())

	// Backend languages
	r.Register(NewRubyParser())
	r.Register(NewPHPParser())
	r.Register(NewKotlinParser())
	r.Register(NewScalaParser())
	r.Register(NewSwiftParser())

	// Infrastructure/scripting
	r.Register(NewBashParser())
	r.Register(NewSQLParser())
	r.Register(NewDockerfileParser())
	r.Register(NewHCLParser())

	// Config/web
	r.Register(NewHTMLParser())
	r.Register(NewCSSParser())
	r.Register(NewYAMLParser())
	r.Register(NewTOMLParser())
	r.Register(NewJSONParser())
	r.Register(NewMarkdownParser())

	// Other languages
	r.Register(NewElixirParser())
	r.Register(NewLuaParser())
	r.Register(NewGroovyParser())
	r.Register(NewSvelteParser())
	r.Register(NewOCamlParser())
	r.Register(NewElmParser())
	r.Register(NewProtobufParser())

	// Regex-based parsers
	r.Register(NewDartParser())
	r.Register(NewCUEParser())
}

func (r *ParserRegistry) Register(p Parser) {
	r.parsers[p.Language()] = p
}

func (r *ParserRegistry) GetParser(lang Language) (Parser, bool) {
	p, ok := r.parsers[lang]
	return p, ok
}

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

	return parser.Parse(content, filePath)
}

var defaultRegistry *ParserRegistry

func init() {
	defaultRegistry = NewParserRegistry()
}

func Analyze(content []byte, filePath string) (*FileAnalysis, error) {
	return defaultRegistry.Parse(content, filePath)
}
