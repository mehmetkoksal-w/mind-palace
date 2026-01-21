# Sprint 2: Contract Detection

> **Goal**: Implement FE↔BE contract detection with type mismatch analysis and API endpoint extraction

## Sprint Overview

| Attribute | Value |
|-----------|-------|
| Sprint Number | 2 |
| Status | ✅ Complete |
| Depends On | Sprint 1 (Pattern Detection Foundation) |
| Blocks | Sprint 3 (LSP - uses contracts for diagnostics) |

---

## Objectives

### Primary
- [x] Extract API endpoints from backend code (Go, Express, FastAPI)
- [x] Detect frontend API calls (fetch, axios, custom clients)
- [x] Match backend endpoints to frontend calls
- [x] Detect type mismatches between FE and BE

### Secondary
- [x] Contract storage model with governance integration
- [x] Dashboard contract visualization
- [x] MCP tools for contract queries
- [x] CLI commands for contract management

---

## Scope

### In Scope
- Backend endpoint extractors for Go (net/http, gin, echo), TypeScript (Express), Python (FastAPI)
- Frontend API call detection for fetch, axios, and common patterns
- Type schema extraction from Go structs and TypeScript interfaces
- 5 mismatch types: missing_in_frontend, missing_in_backend, type_mismatch, optionality_mismatch, nullability_mismatch
- Contract storage in memory.db
- CLI: `palace contracts scan`, `palace contracts list`, `palace contracts verify`
- Dashboard contracts view

### Out of Scope
- GraphQL contracts (future sprint)
- gRPC contracts (future sprint)
- OpenAPI/Swagger import (future sprint)
- Automatic fix generation (Sprint 3 via LSP)

---

## Architecture

### New Packages

```
apps/cli/internal/
├── contracts/
│   ├── contract.go           # Contract model and types
│   ├── store.go              # Contract persistence
│   ├── matcher.go            # Match BE endpoints to FE calls
│   ├── analyzer.go           # Mismatch detection engine
│   ├── types.go              # TypeSchema and comparison
│   └── extractors/           # Language-specific extractors
│       ├── extractor.go      # Extractor interface
│       ├── go_http.go        # Go net/http, gin, echo
│       ├── express.go        # Express.js routes
│       ├── fastapi.go        # FastAPI routes
│       ├── fetch.go          # Frontend fetch() calls
│       └── axios.go          # Frontend axios calls
```

### Database Schema

```sql
-- Add to memory.db migrations (v9)

CREATE TABLE contracts (
    id TEXT PRIMARY KEY,
    method TEXT NOT NULL,              -- GET, POST, PUT, DELETE, PATCH
    endpoint TEXT NOT NULL,            -- Normalized path: /api/users/:id
    endpoint_pattern TEXT,             -- Regex for matching: /api/users/[^/]+

    -- Backend info
    backend_file TEXT,
    backend_line INTEGER,
    backend_framework TEXT,            -- go-http, gin, echo, express, fastapi
    backend_handler TEXT,              -- Handler function name
    backend_request_schema TEXT,       -- JSON TypeSchema
    backend_response_schema TEXT,      -- JSON TypeSchema

    -- Aggregated frontend info
    frontend_call_count INTEGER DEFAULT 0,

    -- Status and governance
    status TEXT DEFAULT 'discovered',  -- discovered, verified, mismatch, ignored
    authority TEXT DEFAULT 'proposed',
    confidence REAL DEFAULT 0.0,

    -- Timestamps
    first_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE contract_frontend_calls (
    id TEXT PRIMARY KEY,
    contract_id TEXT REFERENCES contracts(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    line_number INTEGER NOT NULL,
    call_type TEXT,                    -- fetch, axios, custom
    expected_schema TEXT,              -- JSON TypeSchema (what FE expects)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE contract_mismatches (
    id TEXT PRIMARY KEY,
    contract_id TEXT REFERENCES contracts(id) ON DELETE CASCADE,
    field_path TEXT NOT NULL,          -- e.g., "user.email" or "data[].id"
    mismatch_type TEXT NOT NULL,       -- missing_in_frontend, missing_in_backend, etc.
    severity TEXT DEFAULT 'warning',   -- error, warning, info
    description TEXT,
    backend_type TEXT,                 -- Type on backend side
    frontend_type TEXT,                -- Type on frontend side
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_contracts_endpoint ON contracts(endpoint);
CREATE INDEX idx_contracts_method ON contracts(method);
CREATE INDEX idx_contracts_status ON contracts(status);
CREATE INDEX idx_contract_calls_contract ON contract_frontend_calls(contract_id);
CREATE INDEX idx_contract_mismatches_contract ON contract_mismatches(contract_id);
CREATE INDEX idx_contract_mismatches_type ON contract_mismatches(mismatch_type);
```

### Core Types

```go
// internal/contracts/contract.go

type Contract struct {
    ID                   string           `json:"id"`
    Method               string           `json:"method"`
    Endpoint             string           `json:"endpoint"`
    EndpointPattern      string           `json:"endpoint_pattern"`

    Backend              BackendEndpoint  `json:"backend"`
    FrontendCalls        []FrontendCall   `json:"frontend_calls"`
    Mismatches           []FieldMismatch  `json:"mismatches"`

    Status               ContractStatus   `json:"status"`
    Authority            string           `json:"authority"`
    Confidence           float64          `json:"confidence"`

    FirstSeen            time.Time        `json:"first_seen"`
    LastSeen             time.Time        `json:"last_seen"`
}

type BackendEndpoint struct {
    File            string       `json:"file"`
    Line            int          `json:"line"`
    Framework       string       `json:"framework"`
    Handler         string       `json:"handler"`
    RequestSchema   *TypeSchema  `json:"request_schema,omitempty"`
    ResponseSchema  *TypeSchema  `json:"response_schema,omitempty"`
}

type FrontendCall struct {
    ID              string       `json:"id"`
    File            string       `json:"file"`
    Line            int          `json:"line"`
    CallType        string       `json:"call_type"`  // fetch, axios, custom
    ExpectedSchema  *TypeSchema  `json:"expected_schema,omitempty"`
}

type ContractStatus string

const (
    ContractDiscovered ContractStatus = "discovered"
    ContractVerified   ContractStatus = "verified"
    ContractMismatch   ContractStatus = "mismatch"
    ContractIgnored    ContractStatus = "ignored"
)
```

```go
// internal/contracts/types.go

type TypeSchema struct {
    Type       string                 `json:"type"`       // object, array, string, number, boolean, null, any
    Properties map[string]*TypeSchema `json:"properties,omitempty"`
    Items      *TypeSchema            `json:"items,omitempty"`
    Required   []string               `json:"required,omitempty"`
    Nullable   bool                   `json:"nullable,omitempty"`
    Enum       []string               `json:"enum,omitempty"`
}

type FieldMismatch struct {
    ID           string        `json:"id"`
    FieldPath    string        `json:"field_path"`    // e.g., "user.profile.email"
    Type         MismatchType  `json:"type"`
    Severity     string        `json:"severity"`      // error, warning, info
    Description  string        `json:"description"`
    BackendType  string        `json:"backend_type,omitempty"`
    FrontendType string        `json:"frontend_type,omitempty"`
}

type MismatchType string

const (
    MismatchMissingInFrontend  MismatchType = "missing_in_frontend"
    MismatchMissingInBackend   MismatchType = "missing_in_backend"
    MismatchTypeMismatch       MismatchType = "type_mismatch"
    MismatchOptionalityMismatch MismatchType = "optionality_mismatch"
    MismatchNullabilityMismatch MismatchType = "nullability_mismatch"
)
```

```go
// internal/contracts/extractors/extractor.go

type EndpointExtractor interface {
    // Metadata
    ID() string
    Framework() string
    Languages() []string

    // Extraction
    ExtractEndpoints(file *analysis.ParsedFile) ([]ExtractedEndpoint, error)
}

type APICallExtractor interface {
    // Metadata
    ID() string
    CallType() string  // fetch, axios, etc.
    Languages() []string

    // Extraction
    ExtractCalls(file *analysis.ParsedFile) ([]ExtractedCall, error)
}

type ExtractedEndpoint struct {
    Method          string
    Path            string
    PathParams      []string
    Handler         string
    File            string
    Line            int
    RequestSchema   *TypeSchema
    ResponseSchema  *TypeSchema
}

type ExtractedCall struct {
    Method          string      // May be dynamic
    URL             string      // May contain variables
    File            string
    Line            int
    ExpectedSchema  *TypeSchema
}
```

---

## Tasks

### Week 1: Type System & Schema Extraction ✅

#### 1.1 TypeSchema Implementation
- [x] Define TypeSchema struct with recursive properties
- [x] Implement schema comparison algorithm
- [x] Handle nested objects and arrays
- [x] Write comparison tests with edge cases

#### 1.2 Go Type Extractor
- [x] Extract struct definitions from Go files
- [x] Map Go types to TypeSchema (string, int, bool, etc.)
- [x] Handle pointer types (nullable)
- [x] Handle struct tags (json:"name,omitempty")
- [x] Write Go type extraction tests

#### 1.3 TypeScript Type Extractor
- [x] Extract interface definitions
- [x] Extract type aliases
- [x] Handle optional properties (?)
- [x] Handle union types (basic support)
- [x] Write TypeScript type extraction tests

### Week 2: Backend Endpoint Extractors ✅

#### 2.1 Go HTTP Extractor
- [x] Detect `http.HandleFunc` patterns
- [x] Detect `mux.HandleFunc` (gorilla/mux)
- [x] Detect gin router patterns (`r.GET`, `r.POST`)
- [x] Detect echo router patterns (`e.GET`, `e.POST`)
- [x] Extract path parameters from routes
- [x] Link handlers to response types
- [x] Write extractor tests

#### 2.2 Express Extractor
- [x] Detect `app.get`, `app.post`, etc.
- [x] Detect `router.get`, `router.post`, etc.
- [x] Handle route parameters (`:id`)
- [x] Extract handler function references
- [x] Write extractor tests

#### 2.3 FastAPI Extractor
- [x] Detect `@app.get`, `@app.post` decorators
- [x] Detect `@router.get`, `@router.post` decorators
- [x] Extract path parameters (`{user_id}`)
- [x] Extract Pydantic models for request/response
- [x] Write extractor tests

### Week 3: Frontend Call Detection ✅

#### 3.1 Fetch Extractor
- [x] Detect `fetch()` calls
- [x] Extract URL (literal and template strings)
- [x] Detect HTTP method from options
- [x] Handle `.then()` chains for response typing
- [x] Write extractor tests

#### 3.2 Axios Extractor
- [x] Detect `axios.get`, `axios.post`, etc.
- [x] Detect `axios()` with config object
- [x] Extract URL and method
- [x] Handle generic type parameters `axios.get<User>`
- [x] Write extractor tests

#### 3.3 Custom Client Detection
- [x] Detect common patterns: `api.users.get()`
- [x] Configurable client patterns in palace.jsonc
- [x] Write detection tests

### Week 4: Matching & Mismatch Detection ✅

#### 4.1 Endpoint Matcher
- [x] Normalize endpoint paths (/api/users/:id → /api/users/[^/]+)
- [x] Match frontend URLs to backend endpoints
- [x] Handle path parameters
- [x] Calculate match confidence
- [x] Write matcher tests

#### 4.2 Mismatch Detection Engine
- [x] Compare backend response schema to frontend expected schema
- [x] Detect missing fields in frontend
- [x] Detect missing fields in backend
- [x] Detect type mismatches
- [x] Detect optionality mismatches (required vs optional)
- [x] Detect nullability mismatches
- [x] Generate human-readable descriptions
- [x] Write mismatch detection tests

#### 4.3 Contract Store
- [x] CRUD operations for contracts
- [x] Mismatch storage and retrieval
- [x] Frontend call tracking
- [x] Status transitions
- [x] Write store tests

### Week 5: CLI & Dashboard ✅

#### 5.1 CLI Commands
- [x] `palace contracts scan` - Scan for contracts
- [x] `palace contracts list` - List all contracts
- [x] `palace contracts list --status mismatch` - Filter by status
- [x] `palace contracts show <id>` - Show contract details
- [x] `palace contracts verify <id>` - Mark as verified
- [x] `palace contracts ignore <id>` - Ignore contract
- [x] JSON output option for all commands

#### 5.2 Dashboard Integration
- [x] Add Contracts view to Angular dashboard
- [x] Contract list with filters (method, status, mismatches)
- [x] Contract detail view with:
  - Backend endpoint info
  - Frontend calls list
  - Mismatches with severity indicators
  - Schema comparison visualization
- [x] Verify/Ignore actions

#### 5.3 MCP Tools
- [x] `contracts_get` - Query contracts with filters
- [x] `contract_show` - Show contract details
- [x] `contract_mismatches` - Get contracts with mismatches
- [x] `contract_verify` - Mark contract as verified
- [x] `contract_ignore` - Mark contract as ignored
- [x] `contract_stats` - Get contract statistics

### Week 6: Integration & Polish ✅

#### 6.1 Governance Integration
- [x] Contracts with mismatches create Issues (new knowledge type?)
- [x] Verified contracts can create Learnings about API patterns
- [x] Link contracts to relevant patterns from Sprint 1

#### 6.2 Incremental Detection
- [x] Only scan changed files
- [x] Update existing contracts on rescan
- [x] Remove stale contracts when endpoints deleted

#### 6.3 Testing & Documentation
- [x] Integration tests for full contract flow
- [x] Test with real-world project structures
- [x] Update CLI help text
- [x] Add contracts section to docs
- [x] Update CHANGELOG

---

## Definition of Done

- [x] Backend extractors working for Go, Express, FastAPI
- [x] Frontend extractors working for fetch, axios
- [x] Type schema extraction for Go and TypeScript
- [x] All 5 mismatch types detected correctly
- [x] Contract storage in memory.db
- [x] CLI commands: scan, list, show, verify, ignore
- [x] Dashboard contracts view functional
- [x] MCP tools exposed and working
- [x] All existing tests passing
- [x] New tests for contracts package (>70% coverage)
- [x] Documentation updated

---

## Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Dynamic URLs hard to match | High | High | Focus on static URLs first, pattern config for dynamic |
| Type inference accuracy | Medium | Medium | Conservative matching, flag uncertain inferences |
| Framework variations | Medium | High | Start with most common patterns, extensible extractors |
| Large API surfaces | Medium | Medium | Pagination in dashboard, filtering in CLI |

---

## Dependencies

### From Sprint 1
- Pattern detection engine (for API pattern integration)
- Confidence scoring (reuse for contract confidence)
- Database migration infrastructure

### Internal
- `internal/analysis` - Tree-sitter parsing
- `internal/index` - Code index for cross-file lookups

### External
- None new

---

## Notes

- Start with most common frameworks: Go stdlib, gin, Express, FastAPI
- Focus on JSON APIs (most common)
- Type extraction is best-effort - flag low confidence
- Consider OpenAPI import as future enhancement
- Contract violations could feed into LSP diagnostics (Sprint 3)
