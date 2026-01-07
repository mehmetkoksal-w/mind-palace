# Parser Architecture

## Overview

Mind Palace uses a **three-tier fallback strategy** for code parsing, prioritizing accuracy while ensuring broad compatibility.

## Priority Chain

### 1. LSP (Language Server Protocol) - **Preferred** ✅

**Best for**: Semantic analysis, production accuracy

- **Most accurate** - Uses official language tools with full type information
- **Semantic understanding** - Not just syntax, but types, references, implementations
- **Auto-updated** - Maintained by language communities
- **Call hierarchy** - Accurate caller/callee relationships (future)

**Implemented Language Servers**:

- **Go**: `gopls` (official) ✅ **IMPLEMENTED**

**Planned Language Servers**:

- **TypeScript/JavaScript**: `typescript-language-server`
- **Python**: `pyright`, `pylsp`
- **Rust**: `rust-analyzer`
- **Java**: `jdtls`
- **C/C++**: `clangd`
- **C#**: `omnisharp`
- **Ruby**: `solargraph`
- **PHP**: `intelephense`
- [Full list](https://langserver.org)

**Implementation Files**:

- `lsp_client.go` - Generic LSP protocol client
- `parser_go_lsp.go` - Go language LSP parser
- `dart_lsp.go` - Dart LSP client (reference)

**Status**: ✅ **ACTIVE** - Generic LSP client + gopls integration complete

---

### 2. Tree-Sitter - **Current Default**

**Best for**: Fast AST parsing when LSP unavailable

- **Good accuracy** - Proper AST parsing with node types
- **Fast** - Compiled C parsers via CGO
- **30+ languages** - Wide language coverage
- **Requires CGO** - Needs C compiler (gcc/clang/MinGW)

**CI Support**:

- ✅ macOS: Clang/LLVM (built-in)
- ✅ Linux: GCC (built-in)
- ✅ Windows: MinGW (installed via msys2 in CI)

**Local Development**:

- macOS/Linux: Works out of box
- Windows: Requires MinGW/TDM-GCC installation
  ```powershell
  choco install mingw
  # or download from: https://jmeubank.github.io/tdm-gcc/
  ```

**Status**: ✅ **Active** - All 31 parsers available in CI builds

**Supported Languages**:

- Tree-sitter parsers: Go, TypeScript, JavaScript, Python, Rust, Java, C, C++, C#
- Backend: Ruby, PHP, Kotlin, Scala, Swift
- Infrastructure: Bash, SQL, Dockerfile, HCL
- Config/Web: HTML, CSS, YAML, TOML, JSON, Markdown
- Other: Elixir, Lua, Groovy, Svelte, OCaml, Elm, Protobuf

---

### 3. Regex - **Fallback**

**Best for**: Simple symbol extraction, guaranteed availability

- **Always works** - Pure Go, no dependencies
- **Basic extraction** - Classes, functions, imports
- **Simple patterns** - No AST, just regex matching
- **Limited accuracy** - Can miss nested/complex structures

**Status**: ✅ **Active** - Dart, CUE

**Use Cases**:

- Languages without LSP/tree-sitter support
- Environments without C compiler
- Quick symbol extraction where full AST not needed

---

## LSP Infrastructure ✅ IMPLEMENTED

### Generic LSP Client (`lsp_client.go`)

A reusable LSP protocol client that can communicate with any language server via stdio.

**Features**:

- **Generic**: Works with any LSP-compliant language server
- **Stdio communication**: Standard LSP transport
- **Document symbols**: Retrieves hierarchical symbol information
- **Timeout handling**: Configurable request timeouts (default: 5s)
- **Error handling**: Graceful handling of LSP errors
- **Type conversion**: Converts LSP symbols to Mind Palace format

**Key Types**:

```go
type LSPClient struct {
    cmd       *exec.Cmd
    stdin     io.WriteCloser
    stdout    io.ReadCloser
    // ... internal state
}

type LSPClientConfig struct {
    ServerCmd  string        // e.g., "gopls"
    ServerArgs []string      // e.g., ["-mode=stdio"]
    RootPath   string        // Workspace root
    LanguageID string        // e.g., "go"
    Timeout    time.Duration // Request timeout
}
```

**Usage**:

```go
client, err := NewLSPClient(LSPClientConfig{
    ServerCmd:  "gopls",
    ServerArgs: []string{},
    RootPath:   "/path/to/project",
    LanguageID: "go",
})
defer client.Close()

symbols, err := client.DocumentSymbols(uri, content)
```

### Go LSP Parser (`parser_go_lsp.go`)

**Features**:

- **gopls integration**: Uses official Go language server
- **Availability check**: Detects if gopls is installed
- **Symbol extraction**: Functions, methods, structs, interfaces, variables, constants
- **Export detection**: Correctly identifies exported symbols
- **Relationship extraction**: Imports and function calls
- **Fallback-ready**: Implements `LSPParser` interface for automatic fallback

**Example Symbols Extracted**:

- **Functions**: `func NewPerson(name string, age int) *Person`
- **Methods**: `func (p *Person) Greet() string`
- **Structs**: `type Person struct { ... }`
- **Interfaces**: `type Reader interface { ... }`
- **Constants**: `const MaxAge = 120`
- **Variables**: `var defaultPerson = Person{...}`

### Fallback Logic (`parser.go`) ✅ IMPLEMENTED

The parser registry now implements automatic fallback with priority-based selection:

```go
type ParserPriority int

const (
    PriorityLSP        ParserPriority = 1  // Highest priority
    PriorityTreeSitter ParserPriority = 2  // Medium priority
    PriorityRegex      ParserPriority = 3  // Lowest priority
)

// GetParser automatically selects best available parser
func (r *ParserRegistry) GetParser(lang Language) (Parser, bool) {
    // 1. Try LSP if available and enabled
    if lspParser.IsAvailable() {
        return lspParser
    }

    // 2. Fall back to tree-sitter
    if tsParser available {
        return tsParser
    }

    // 3. Fall back to regex
    return regexParser
}
```

**Features**:

- **Automatic fallback**: If LSP parser fails, tries tree-sitter, then regex
- **Availability detection**: Checks if language server is installed
- **Enable/disable LSP**: Can disable LSP parsers via `SetEnableLSP(false)`
- **Debug mode**: `SetDebugMode(true)` logs which parser is used
- **Per-language priorities**: Each language can have multiple parsers

**Example Flow for Go**:

1. **Try gopls**: If installed → uses LSP parser → most accurate
2. **Fall back to tree-sitter**: If gopls unavailable/fails → uses tree-sitter
3. **No regex fallback for Go**: Tree-sitter is sufficient

---

## Implementation Details

### File Structure

```
apps/cli/internal/analysis/
├── lsp_client.go          ✅ Generic LSP client (new - 545 lines)
├── parser_go_lsp.go       ✅ Go LSP parser (new - 300 lines)
├── parser_go_lsp_test.go  ✅ Go LSP tests (new - 350 lines)
├── parser.go              ✅ Updated with fallback logic
├── parser_go.go           ✅ Existing tree-sitter Go parser
├── dart_lsp.go            ✅ Dart LSP reference
├── types.go               ✅ Shared types
└── language.go            ✅ Language detection
```

### Testing

**Test Files Created**:

- `parser_go_lsp_test.go` - Comprehensive test suite

**Test Coverage**:

- ✅ Availability detection
- ✅ Basic symbol parsing (functions, methods, structs)
- ✅ Export detection (uppercase = exported)
- ✅ Complex structures (interfaces, nested types)
- ✅ Error handling (malformed code)
- ✅ Comparison with tree-sitter output
- ✅ Performance benchmarks

**Running Tests** (requires gopls):

```powershell
# Install gopls
go install golang.org/x/tools/gopls@latest

# Run tests
cd apps/cli
go test ./internal/analysis/... -v -run TestGoLSP
```

**Sample Test Output**:

```
=== RUN   TestGoLSPParser_Parse
Found 7 symbols
  function: main (line 31-35)
  function: NewPerson (line 14-18)
  method: Greet (line 21-23)
  method: IsAdult (line 26-28)
  class: Person (line 8-11)
  constant: MaxAge (line 30)
  variable: defaultPerson (line 32)
--- PASS: TestGoLSPParser_Parse
```

---

## Current Status

| Parser Type     | Languages      | Status        | Requires        |
| --------------- | -------------- | ------------- | --------------- |
| **LSP**         | Go (gopls)     | ✅ **ACTIVE** | gopls binary    |
| **LSP**         | Dart (example) | ✅ Working    | dart SDK        |
| **Tree-Sitter** | 31 languages   | ✅ CI builds  | gcc/clang/MinGW |
| **Regex**       | Dart, CUE      | ✅ Working    | Nothing         |

---

## Developer Notes

### Adding a New LSP Parser

1. **Create parser file**: `parser_<lang>_lsp.go`

   ```go
   type <Lang>LSPParser struct {
       available bool
       rootPath  string
   }

   func New<Lang>LSPParser(rootPath string) *<Lang>LSPParser {
       _, err := exec.LookPath("language-server-binary")
       return &<Lang>LSPParser{
           available: err == nil,
           rootPath:  rootPath,
       }
   }

   func (p *<Lang>LSPParser) IsAvailable() bool {
       return p.available
   }

   func (p *<Lang>LSPParser) Language() Language {
       return Lang<Lang>
   }

   func (p *<Lang>LSPParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
       client, err := NewLSPClient(LSPClientConfig{
           ServerCmd:  "language-server-binary",
           ServerArgs: []string{},
           RootPath:   p.rootPath,
           LanguageID: "language-id",
       })
       defer client.Close()

       // Get and convert symbols
       // ...
   }
   ```

2. **Register in `parser.go`**:

   ```go
   r.RegisterWithPriority(New<Lang>LSPParser(r.rootPath), PriorityLSP)
   ```

3. **Create tests**: `parser_<lang>_lsp_test.go`

4. **Document language server requirement** in README

---

## Benefits of LSP-First Approach

1. **Most Accurate**: Official language tools, semantic analysis
2. **Auto-Updated**: Language servers updated by communities
3. **Rich Features**: Call hierarchy, type information, cross-references
4. **Developer-Friendly**: Tools developers already have installed
5. **Graceful Degradation**: Falls back to tree-sitter if LSP unavailable
6. **No CGO Required**: LSP parsers work without C compiler
7. **Future-Proof**: Easy to add new languages as LSP servers become available

---

## Changelog

### 2026-01-07 - LSP Infrastructure Implemented ✅

**Added**:

- `lsp_client.go` - Generic LSP protocol client (545 lines)
- `parser_go_lsp.go` - Go language LSP parser (300 lines)
- `parser_go_lsp_test.go` - Comprehensive test suite (350 lines)

**Modified**:

- `parser.go` - Added priority-based fallback logic
  - New types: `ParserPriority`, `LSPParser`, `parserEntry`
  - New methods: `RegisterWithPriority`, `SetDebugMode`, `SetEnableLSP`
  - Automatic fallback: LSP → Tree-sitter → Regex

**Features**:

- ✅ Generic LSP client supporting any language server
- ✅ gopls integration for Go parsing
- ✅ Automatic fallback to tree-sitter if LSP unavailable
- ✅ Priority-based parser selection
- ✅ Debug mode for parser selection logging
- ✅ Comprehensive test coverage

**Next Steps**:

- Implement TypeScript/JavaScript LSP parser
- Add call hierarchy support
- Optimize with persistent LSP server
