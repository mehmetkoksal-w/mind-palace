package analysis

// SymbolKind represents the type of a symbol (e.g., class, function).
type SymbolKind string

// Predefined symbol kinds.
const (
	KindClass       SymbolKind = "class"
	KindInterface   SymbolKind = "interface"
	KindFunction    SymbolKind = "function"
	KindMethod      SymbolKind = "method"
	KindVariable    SymbolKind = "variable"
	KindConstant    SymbolKind = "constant"
	KindType        SymbolKind = "type"
	KindEnum        SymbolKind = "enum"
	KindProperty    SymbolKind = "property"
	KindConstructor SymbolKind = "constructor"
)

// RelationshipKind represents the type of relationship between symbols.
type RelationshipKind string

// Predefined relationship kinds.
const (
	RelImport     RelationshipKind = "import"
	RelCall       RelationshipKind = "call"
	RelReference  RelationshipKind = "reference"
	RelExtends    RelationshipKind = "extends"
	RelImplements RelationshipKind = "implements"
	RelUses       RelationshipKind = "uses"
)

// Symbol represents a programming construct found in a file.
type Symbol struct {
	Name       string
	Kind       SymbolKind
	LineStart  int
	LineEnd    int
	Signature  string
	DocComment string
	Exported   bool
	Children   []Symbol
}

// Relationship represents a semantic link between symbols.
type Relationship struct {
	SourceSymbol string
	TargetFile   string
	TargetSymbol string
	Kind         RelationshipKind
	Line         int
	Column       int
}

// FileAnalysis stores the results of analyzing a single file.
type FileAnalysis struct {
	Path          string
	Language      string
	Symbols       []Symbol
	Relationships []Relationship
}

// Language represents a programming or markup language.
type Language string

// Supported languages.
const (
	LangGo         Language = "go"
	LangJavaScript Language = "javascript"
	LangTypeScript Language = "typescript"
	LangPython     Language = "python"
	LangRust       Language = "rust"
	LangJava       Language = "java"
	LangDart       Language = "dart"
	LangC          Language = "c"
	LangCPP        Language = "cpp"
	LangCSharp     Language = "csharp"
	LangRuby       Language = "ruby"
	LangSwift      Language = "swift"
	LangKotlin     Language = "kotlin"
	LangScala      Language = "scala"
	LangPHP        Language = "php"
	LangBash       Language = "bash"
	LangSQL        Language = "sql"
	LangHTML       Language = "html"
	LangCSS        Language = "css"
	LangYAML       Language = "yaml"
	LangTOML       Language = "toml"
	LangJSON       Language = "json"
	LangMarkdown   Language = "markdown"
	LangDockerfile Language = "dockerfile"
	LangHCL        Language = "hcl"
	LangProtobuf   Language = "protobuf"
	LangLua        Language = "lua"
	LangElixir     Language = "elixir"
	LangGroovy     Language = "groovy"
	LangSvelte     Language = "svelte"
	LangOCaml      Language = "ocaml"
	LangElm        Language = "elm"
	LangCUE        Language = "cue"
	LangUnknown    Language = "unknown"
)
