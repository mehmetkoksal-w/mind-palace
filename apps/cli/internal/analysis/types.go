package analysis

type SymbolKind string

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

type RelationshipKind string

const (
	RelImport     RelationshipKind = "import"
	RelCall       RelationshipKind = "call"
	RelReference  RelationshipKind = "reference"
	RelExtends    RelationshipKind = "extends"
	RelImplements RelationshipKind = "implements"
	RelUses       RelationshipKind = "uses"
)

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

type Relationship struct {
	SourceSymbol string
	TargetFile   string
	TargetSymbol string
	Kind         RelationshipKind
	Line         int
	Column       int
}

type FileAnalysis struct {
	Path          string
	Language      string
	Symbols       []Symbol
	Relationships []Relationship
}

type Language string

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
