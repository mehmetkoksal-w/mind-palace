# Sprint 3: LSP Implementation

> **Goal**: Implement a full Language Server Protocol server for real-time pattern and contract feedback in editors

## Sprint Overview

| Attribute | Value |
|-----------|-------|
| Sprint Number | 3 |
| Status | ✅ Complete |
| Depends On | Sprint 1 (Patterns), Sprint 2 (Contracts) |
| Blocks | None (final sprint in this cycle) |

---

## Objectives

### Primary
- [ ] Implement LSP server core with JSON-RPC 2.0
- [ ] Diagnostics provider for pattern violations and contract mismatches
- [ ] Hover provider for pattern and contract information
- [ ] Code action provider for quick fixes

### Secondary
- [ ] Code lens for pattern counts and contract status
- [ ] Go to definition for pattern sources
- [ ] Document symbols for contracts
- [ ] VS Code extension integration

---

## Scope

### In Scope
- LSP server implementation in Go
- Diagnostics for pattern violations (outliers)
- Diagnostics for contract mismatches
- Hover information for patterns and contracts
- Code actions: approve pattern, ignore pattern, verify contract
- Code lens showing pattern/contract counts per file
- Integration with existing VS Code extension
- `palace lsp` command to start server

### Out of Scope
- Workspace-wide rename (complex, future)
- Semantic tokens (syntax highlighting via patterns)
- Signature help
- Completion provider (not relevant for patterns)

---

## Architecture

### New Packages

```
apps/cli/internal/
├── lsp/
│   ├── server.go             # LSP server main entry
│   ├── protocol.go           # LSP protocol types
│   ├── handler.go            # Request dispatcher
│   ├── initialize.go         # Initialize/shutdown handlers
│   ├── text_document.go      # Document sync handlers
│   ├── diagnostics.go        # Diagnostic provider
│   ├── hover.go              # Hover provider
│   ├── code_action.go        # Code action provider
│   ├── code_lens.go          # Code lens provider
│   ├── definition.go         # Go to definition
│   └── document_symbol.go    # Document symbols
```

### LSP Server Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        EDITOR (VS Code)                          │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │  VS Code Extension (existing)                               │ │
│  │  - Starts LSP server via palace lsp                        │ │
│  │  - Communicates via stdio                                  │ │
│  └─────────────────────────────────────────────────────────────┘ │
└───────────────────────────┬─────────────────────────────────────┘
                            │ JSON-RPC 2.0 (stdio)
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                     LSP SERVER (Go)                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   Handler    │  │  Dispatcher  │  │   Protocol   │          │
│  │  (requests)  │◄─┤  (routing)   │◄─┤   (codec)    │          │
│  └──────┬───────┘  └──────────────┘  └──────────────┘          │
│         │                                                        │
│         ▼                                                        │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    PROVIDERS                              │   │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐           │   │
│  │  │Diagnostics │ │   Hover    │ │Code Actions│           │   │
│  │  └─────┬──────┘ └─────┬──────┘ └─────┬──────┘           │   │
│  │        │              │              │                    │   │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐           │   │
│  │  │ Code Lens  │ │ Definition │ │Doc Symbols │           │   │
│  │  └─────┬──────┘ └─────┬──────┘ └─────┬──────┘           │   │
│  └────────┼──────────────┼──────────────┼───────────────────┘   │
│           │              │              │                        │
│           ▼              ▼              ▼                        │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                  MIND PALACE CORE                         │   │
│  │  ├── Butler (search, MCP tools)                          │   │
│  │  ├── Pattern Store (from Sprint 1)                       │   │
│  │  ├── Contract Store (from Sprint 2)                      │   │
│  │  └── Index (code index)                                  │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Core Types

```go
// internal/lsp/server.go

type Server struct {
    conn       *jsonrpc2.Conn
    butler     *butler.Butler
    patterns   *patterns.Store
    contracts  *contracts.Store
    index      *index.Index

    // Document state
    documents  map[string]*TextDocument
    docMu      sync.RWMutex

    // Server state
    initialized bool
    shutdown    bool
    rootURI     string
}

type TextDocument struct {
    URI        string
    LanguageID string
    Version    int
    Content    string

    // Cached analysis
    Patterns   []PatternLocation
    Contracts  []ContractInfo
    Diagnostics []Diagnostic
}

func NewServer(butler *butler.Butler, patterns *patterns.Store,
               contracts *contracts.Store, index *index.Index) *Server
func (s *Server) Run(ctx context.Context, stream io.ReadWriteCloser) error
```

```go
// internal/lsp/protocol.go

// LSP Protocol types (subset)

type InitializeParams struct {
    ProcessID    int          `json:"processId"`
    RootURI      string       `json:"rootUri"`
    Capabilities ClientCapabilities `json:"capabilities"`
}

type InitializeResult struct {
    Capabilities ServerCapabilities `json:"capabilities"`
}

type ServerCapabilities struct {
    TextDocumentSync   *TextDocumentSyncOptions `json:"textDocumentSync,omitempty"`
    DiagnosticProvider *DiagnosticOptions       `json:"diagnosticProvider,omitempty"`
    HoverProvider      bool                     `json:"hoverProvider,omitempty"`
    CodeActionProvider *CodeActionOptions       `json:"codeActionProvider,omitempty"`
    CodeLensProvider   *CodeLensOptions         `json:"codeLensProvider,omitempty"`
    DefinitionProvider bool                     `json:"definitionProvider,omitempty"`
    DocumentSymbolProvider bool                 `json:"documentSymbolProvider,omitempty"`
}

type Diagnostic struct {
    Range    Range              `json:"range"`
    Severity DiagnosticSeverity `json:"severity"`
    Code     string             `json:"code,omitempty"`
    Source   string             `json:"source"`
    Message  string             `json:"message"`
    Data     any                `json:"data,omitempty"`
}

type DiagnosticSeverity int

const (
    DiagnosticSeverityError       DiagnosticSeverity = 1
    DiagnosticSeverityWarning     DiagnosticSeverity = 2
    DiagnosticSeverityInformation DiagnosticSeverity = 3
    DiagnosticSeverityHint        DiagnosticSeverity = 4
)
```

```go
// internal/lsp/diagnostics.go

type DiagnosticsProvider struct {
    patterns  *patterns.Store
    contracts *contracts.Store
}

func (p *DiagnosticsProvider) GetDiagnostics(uri string, content string) []Diagnostic {
    var diagnostics []Diagnostic

    // 1. Pattern violations (outliers in this file)
    outliers := p.patterns.GetOutliersForFile(uri)
    for _, outlier := range outliers {
        diagnostics = append(diagnostics, Diagnostic{
            Range:    toRange(outlier.LineStart, outlier.LineEnd),
            Severity: DiagnosticSeverityWarning,
            Code:     outlier.PatternID,
            Source:   "mind-palace",
            Message:  fmt.Sprintf("Deviates from pattern: %s", outlier.PatternName),
            Data:     outlier,
        })
    }

    // 2. Contract mismatches (if this file has API calls or endpoints)
    mismatches := p.contracts.GetMismatchesForFile(uri)
    for _, mismatch := range mismatches {
        diagnostics = append(diagnostics, Diagnostic{
            Range:    toRange(mismatch.Line, mismatch.Line),
            Severity: toSeverity(mismatch.Severity),
            Code:     string(mismatch.Type),
            Source:   "mind-palace",
            Message:  mismatch.Description,
            Data:     mismatch,
        })
    }

    return diagnostics
}
```

---

## Tasks

### Week 1: LSP Server Core ✅

#### 1.1 JSON-RPC 2.0 Foundation
- [x] Implement JSON-RPC 2.0 codec (request/response/notification)
- [x] Create message dispatcher for routing
- [x] Handle batch requests
- [x] Implement error responses
- [x] Write codec tests

#### 1.2 Server Lifecycle
- [x] Implement `initialize` handler
- [x] Implement `initialized` notification handler
- [x] Implement `shutdown` handler
- [x] Implement `exit` notification handler
- [x] Server capabilities declaration
- [x] Write lifecycle tests

#### 1.3 Document Synchronization
- [x] Implement `textDocument/didOpen`
- [x] Implement `textDocument/didChange` (full sync)
- [x] Implement `textDocument/didClose`
- [x] Implement `textDocument/didSave`
- [x] Document state management
- [x] Write sync tests

### Week 2: Diagnostics Provider ✅

#### 2.1 Pattern Diagnostics
- [x] Query pattern outliers for document
- [x] Convert outliers to LSP diagnostics
- [x] Map pattern severity to LSP severity
- [x] Include pattern ID in diagnostic code
- [x] Write pattern diagnostic tests

#### 2.2 Contract Diagnostics
- [x] Query contract mismatches for document
- [x] Convert mismatches to LSP diagnostics
- [x] Different severity for different mismatch types
- [x] Include contract ID in diagnostic data
- [x] Write contract diagnostic tests

#### 2.3 Diagnostic Delivery
- [x] Implement push diagnostics (publish on change)
- [x] Clear diagnostics on document close
- [x] Butler adapter for Memory/Contracts integration
- [x] Write delivery tests

### Week 3: Hover & Code Actions ✅

#### 3.1 Hover Provider
- [x] Implement `textDocument/hover`
- [x] Pattern hover: show pattern name, description, confidence
- [x] Contract hover: show endpoint, method, mismatches
- [x] Format hover content as Markdown
- [x] Write hover tests

#### 3.2 Code Action Provider
- [x] Implement `textDocument/codeAction`
- [x] Pattern actions: Approve, Ignore, View Details
- [x] Contract actions: Verify, Ignore, Show Mismatches
- [x] Quick fix: "Apply pattern" (if applicable)
- [x] Write code action tests

#### 3.3 Code Action Resolution
- [x] Implement `codeAction/resolve` for deferred actions
- [x] Execute approve/ignore via Butler
- [x] Refresh diagnostics after action
- [x] Write resolution tests

### Week 4: Code Lens & Navigation ✅

#### 4.1 Code Lens Provider
- [x] Implement `textDocument/codeLens`
- [x] Show pattern count at file top
- [x] Show contract count at file top
- [x] Show "X patterns detected" on first line
- [x] Write code lens tests

#### 4.2 Code Lens Resolution
- [x] Implement `codeLens/resolve`
- [x] Click to open patterns list
- [x] Click to open contracts list
- [x] Write resolution tests

#### 4.3 Go to Definition
- [x] Implement `textDocument/definition`
- [x] Navigate to pattern definition file
- [x] Navigate to contract backend endpoint
- [x] Navigate to related files
- [x] Write definition tests

#### 4.4 Document Symbols
- [x] Implement `textDocument/documentSymbol`
- [x] List contracts defined in file (for backend files)
- [x] List API calls in file (for frontend files)
- [x] Write document symbol tests

### Week 5: VS Code Integration

#### 5.1 CLI Command ✅
- [x] Implement `palace lsp` command
- [x] Support stdio transport
- [x] Support socket transport (optional - deferred)
- [x] Logging to file (not stdout)
- [x] Graceful shutdown

#### 5.2 VS Code Extension Updates ✅
- [x] Add LSP client to extension
- [x] Configure language client for TypeScript, Go, Python
- [x] Start LSP server on extension activation
- [x] Handle server crashes with restart
- [ ] Write extension integration tests (deferred to Week 6)

#### 5.3 Extension Settings ✅
- [x] `mindPalace.lsp.enabled` - Enable/disable LSP
- [x] `mindPalace.lsp.diagnostics.patterns` - Show pattern diagnostics
- [x] `mindPalace.lsp.diagnostics.contracts` - Show contract diagnostics
- [x] `mindPalace.lsp.codeLens.enabled` - Show code lens

### Week 6: Integration & Polish ✅

#### 6.1 Performance Optimization ✅
- [x] Cache diagnostic results
- [x] Debounce document change handlers (300ms default)
- [x] Background recomputation on scan
- [x] Profile and optimize hot paths

#### 6.2 Error Handling ✅
- [x] Graceful degradation when stores unavailable
- [x] Clear error messages for initialization failures
- [x] Recovery from parse errors

#### 6.3 Testing & Documentation
- [x] Test all LSP methods (17 tests passing)
- [ ] Integration tests with mock editor (future)
- [x] Update VS Code extension README
- [x] Add LSP section to main docs
- [x] Update CHANGELOG

---

## Definition of Done

- [x] LSP server starts via `palace lsp`
- [x] Initialize/shutdown lifecycle working
- [x] Document sync (open/change/close) working
- [x] Diagnostics show pattern violations
- [x] Diagnostics show contract mismatches
- [x] Hover shows pattern/contract info
- [x] Code actions: approve, ignore, verify
- [x] Code lens shows counts
- [x] Go to definition works
- [x] VS Code extension uses LSP
- [x] All existing tests passing
- [x] New tests for LSP package (17 tests)
- [x] Documentation updated

---

## LSP Methods Reference

| Method | Priority | Description |
|--------|----------|-------------|
| `initialize` | P0 | Server initialization |
| `initialized` | P0 | Initialization complete |
| `shutdown` | P0 | Server shutdown |
| `exit` | P0 | Server exit |
| `textDocument/didOpen` | P0 | Document opened |
| `textDocument/didChange` | P0 | Document changed |
| `textDocument/didClose` | P0 | Document closed |
| `textDocument/didSave` | P1 | Document saved |
| `textDocument/diagnostic` | P0 | Pull diagnostics |
| `textDocument/publishDiagnostics` | P0 | Push diagnostics |
| `textDocument/hover` | P1 | Hover information |
| `textDocument/codeAction` | P1 | Code actions |
| `codeAction/resolve` | P1 | Resolve code action |
| `textDocument/codeLens` | P2 | Code lens |
| `codeLens/resolve` | P2 | Resolve code lens |
| `textDocument/definition` | P2 | Go to definition |
| `textDocument/documentSymbol` | P2 | Document symbols |

---

## Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| LSP protocol complexity | High | Medium | Use existing Go LSP libraries where possible |
| Performance with large files | Medium | Medium | Debouncing, caching, incremental updates |
| VS Code extension compatibility | Medium | Low | Follow VS Code LSP client best practices |
| Cross-platform stdio issues | Medium | Medium | Test on Windows, macOS, Linux |

---

## Dependencies

### From Sprint 1
- Pattern store and queries
- Outlier detection

### From Sprint 2
- Contract store and queries
- Mismatch detection

### Internal
- `internal/butler` - For executing actions
- `internal/index` - For file lookups

### External
- Consider: `github.com/sourcegraph/jsonrpc2` for JSON-RPC
- Consider: `golang.org/x/tools/gopls` patterns for reference

---

## Notes

- LSP runs as separate process, communicates via stdio
- Server must be fast - diagnostics on every keystroke
- Consider lazy loading of stores on first diagnostic request
- VS Code extension already exists - extend rather than replace
- Test with both VS Code and Neovim/other LSP clients
