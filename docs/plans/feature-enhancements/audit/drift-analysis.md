# Drift Feature Analysis

> Analysis of Drift's implementation for features to port to Mind Palace

## 1. Pattern Detection Automation

### How Drift Does It

**Architecture:**
```
┌─────────────────────────────────────────────────────────┐
│                   DETECTOR REGISTRY                      │
│  101 detectors across 15 categories                     │
├─────────────────────────────────────────────────────────┤
│  Categories:                                            │
│  api, auth, security, errors, logging, data-access,    │
│  config, testing, performance, components, styling,     │
│  structural, types, accessibility, documentation        │
└─────────────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────┐
│                  DETECTION ENGINE                        │
│  ├── AST-based (Tree-sitter)                           │
│  ├── Regex-based (naming conventions)                  │
│  ├── Semantic-based (code meaning)                     │
│  └── Custom (detector-specific logic)                  │
└─────────────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────┐
│               CONFIDENCE SCORING                         │
│  Weighted factors:                                       │
│  - Frequency: How often pattern appears                 │
│  - Consistency: Uniformity of implementations           │
│  - Spread: Number of files using it                     │
│  - Age: How long pattern has existed                    │
└─────────────────────────────────────────────────────────┘
```

**Key Files in Drift:**
- `packages/detectors/src/index.ts` - 101 detector registrations
- `packages/core/src/matcher/types.ts` - Confidence scoring types
- `packages/core/src/store/pattern-store.ts` - Pattern persistence

**Detector Structure:**
```typescript
interface Detector {
  id: string;                    // e.g., "api/response-envelope"
  category: PatternCategory;
  name: string;
  description: string;
  detect: (context: DetectionContext) => DetectionResult;
  confidence: ConfidenceConfig;
}

interface DetectionResult {
  patterns: Pattern[];
  outliers: Location[];
  confidence: number;  // 0.0-1.0
}
```

**Confidence Thresholds:**
- High: ≥ 0.85
- Medium: 0.70-0.84
- Low: 0.50-0.69
- Uncertain: < 0.50

### Port to Mind Palace

**Proposed Go Structure:**
```go
// internal/patterns/detector.go
type Detector interface {
    ID() string
    Category() string
    Detect(ctx *DetectionContext) (*DetectionResult, error)
}

type DetectorRegistry struct {
    detectors map[string]Detector
    mu        sync.RWMutex
}

// internal/patterns/confidence.go
type ConfidenceScore struct {
    Value       float64   // 0.0-1.0
    Frequency   float64   // Weight: 0.3
    Consistency float64   // Weight: 0.3
    Spread      float64   // Weight: 0.25
    Age         float64   // Weight: 0.15
}

func (c *ConfidenceScore) Calculate() float64 {
    return c.Frequency*0.3 + c.Consistency*0.3 +
           c.Spread*0.25 + c.Age*0.15
}
```

**Integration Point:** Detected patterns become Learnings with:
- `authority: "proposed"` (goes through governance)
- `scope: "palace"` (project-wide by default)
- `confidence: <calculated_score>`

---

## 2. Bulk Approvals with Confidence Levels

### How Drift Does It

**Quick Review Feature:**
```typescript
// CLI: drift approve --quick-review
// Approves all patterns with confidence >= 0.95

interface QuickReviewOptions {
  threshold: number;        // Default: 0.95
  categories?: string[];    // Optional: filter by category
  dryRun?: boolean;         // Preview what would be approved
}
```

**Dashboard Bulk UI:**
- Multi-select checkboxes
- Filter by confidence level (High/Medium/Low)
- Batch approve/ignore actions
- Preview of affected patterns

### Port to Mind Palace

**CLI Commands:**
```bash
# Bulk approve high-confidence patterns
palace patterns approve --bulk --min-confidence 0.95

# Bulk approve by category
palace patterns approve --bulk --category api

# Dry run to preview
palace patterns approve --bulk --min-confidence 0.85 --dry-run

# Interactive bulk review
palace patterns review
```

**Dashboard Integration:**
- Add to existing Knowledge view
- Filter patterns by confidence
- Multi-select with shift-click
- Batch actions dropdown

---

## 3. Contract Detection (FE↔BE)

### How Drift Does It

**Architecture:**
```
┌─────────────────┐     ┌─────────────────┐
│    BACKEND      │     │    FRONTEND     │
│  Endpoint Scan  │     │   API Call Scan │
└────────┬────────┘     └────────┬────────┘
         │                       │
         ▼                       ▼
┌─────────────────────────────────────────┐
│           CONTRACT MATCHER               │
│  1. Match endpoints to API calls        │
│  2. Compare request/response schemas    │
│  3. Detect mismatches                   │
└─────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│          MISMATCH TYPES                  │
│  - missing_in_frontend                  │
│  - missing_in_backend                   │
│  - type_mismatch                        │
│  - optionality_mismatch                 │
│  - nullability_mismatch                 │
└─────────────────────────────────────────┘
```

**Key Files in Drift:**
- `packages/core/src/store/contract-store.ts` - Contract persistence
- `packages/core/src/analysis/contract-analyzer.ts` - Detection logic

**Contract Structure:**
```typescript
interface Contract {
  id: string;
  method: HttpMethod;
  endpoint: string;                // Normalized: /api/users/:id
  backend: {
    file: string;
    line: number;
    responseSchema: TypeSchema;
    requestSchema?: TypeSchema;
  };
  frontend: {
    calls: ApiCall[];              // All places this endpoint is called
    expectedSchema: TypeSchema;
  };
  mismatches: FieldMismatch[];
  status: 'discovered' | 'verified' | 'mismatch' | 'ignored';
  confidence: number;
}
```

### Port to Mind Palace

**New Knowledge Type - Contracts:**
```go
// internal/model/contract.go
type Contract struct {
    ID          string            `json:"id"`
    Method      string            `json:"method"`
    Endpoint    string            `json:"endpoint"`
    Backend     BackendEndpoint   `json:"backend"`
    Frontend    []FrontendCall    `json:"frontend"`
    Mismatches  []FieldMismatch   `json:"mismatches"`
    Status      ContractStatus    `json:"status"`
    Authority   string            `json:"authority"`
    Confidence  float64           `json:"confidence"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

type FieldMismatch struct {
    FieldPath    string         `json:"field_path"`
    Type         MismatchType   `json:"type"`
    Description  string         `json:"description"`
    Severity     string         `json:"severity"`
    BackendType  string         `json:"backend_type,omitempty"`
    FrontendType string         `json:"frontend_type,omitempty"`
}
```

**Database Schema:**
```sql
CREATE TABLE contracts (
    id TEXT PRIMARY KEY,
    method TEXT NOT NULL,
    endpoint TEXT NOT NULL,
    backend_file TEXT,
    backend_line INTEGER,
    frontend_files TEXT,  -- JSON array
    status TEXT DEFAULT 'discovered',
    authority TEXT DEFAULT 'proposed',
    confidence REAL DEFAULT 0.0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE contract_mismatches (
    id TEXT PRIMARY KEY,
    contract_id TEXT REFERENCES contracts(id),
    field_path TEXT NOT NULL,
    mismatch_type TEXT NOT NULL,
    description TEXT,
    severity TEXT DEFAULT 'warning',
    backend_type TEXT,
    frontend_type TEXT
);
```

---

## 4. Type Mismatch Detection

### How Drift Does It

**Type Schema Extraction:**
```typescript
// From TypeScript
interface TypeSchema {
  type: 'object' | 'array' | 'string' | 'number' | 'boolean' | 'null';
  properties?: Record<string, TypeSchema>;
  items?: TypeSchema;
  required?: string[];
  nullable?: boolean;
}

// Extraction from TS interfaces, type aliases, and inline types
function extractTypeSchema(node: TSNode): TypeSchema
```

**Comparison Algorithm:**
```typescript
function compareSchemas(
  backend: TypeSchema,
  frontend: TypeSchema,
  path: string = ''
): FieldMismatch[] {
  const mismatches: FieldMismatch[] = [];

  // 1. Check type equality
  if (backend.type !== frontend.type) {
    mismatches.push({
      fieldPath: path,
      type: 'type_mismatch',
      backendType: backend.type,
      frontendType: frontend.type
    });
  }

  // 2. Check required fields (backend required, frontend optional)
  // 3. Check nullable differences
  // 4. Recurse into nested objects
  // 5. Compare array item types

  return mismatches;
}
```

### Port to Mind Palace

**Go Implementation:**
```go
// internal/contracts/types.go
type TypeSchema struct {
    Type       string                 `json:"type"`
    Properties map[string]*TypeSchema `json:"properties,omitempty"`
    Items      *TypeSchema            `json:"items,omitempty"`
    Required   []string               `json:"required,omitempty"`
    Nullable   bool                   `json:"nullable,omitempty"`
}

// internal/contracts/compare.go
func CompareSchemas(backend, frontend *TypeSchema, path string) []FieldMismatch {
    // Port Drift's comparison logic
}
```

**Leverage Existing:**
- Mind Palace already has Tree-sitter for 20+ languages
- Can extract Go struct types, TypeScript interfaces
- Extend symbol extraction to include type information

---

## 5. API Endpoint Analysis

### How Drift Does It

**Backend Framework Detection:**
```typescript
// Express.js
app.get('/users/:id', handler)
router.post('/auth/login', controller.login)

// FastAPI (Python)
@app.get("/users/{user_id}")
@router.post("/auth/login")

// Detected patterns:
// - Route path with parameters
// - HTTP method
// - Handler function reference
// - Middleware chain
```

**Frontend API Call Detection:**
```typescript
// fetch
fetch('/api/users/' + id)
fetch(`/api/users/${id}`)

// axios
axios.get('/api/users', { params: { id } })
axios.post('/api/auth/login', credentials)

// Custom clients
api.users.get(id)
```

### Port to Mind Palace

**Go Endpoint Extractors:**
```go
// internal/contracts/extractors/go_http.go
type GoHTTPExtractor struct{}

func (e *GoHTTPExtractor) Extract(file *analysis.ParsedFile) []Endpoint {
    // Detect: http.HandleFunc, mux.HandleFunc, gin routes, echo routes
}

// internal/contracts/extractors/express.go
type ExpressExtractor struct{}

func (e *ExpressExtractor) Extract(file *analysis.ParsedFile) []Endpoint {
    // Detect: app.get, app.post, router.get, router.post
}
```

**Unified Endpoint Model:**
```go
type Endpoint struct {
    Method      string            // GET, POST, PUT, DELETE, PATCH
    Path        string            // /api/users/:id
    PathParams  []string          // [id]
    QueryParams []string          // Detected query params
    BodyType    *TypeSchema       // Request body schema
    ResponseType *TypeSchema      // Response schema
    Handler     string            // Handler function name
    File        string
    Line        int
    Framework   string            // express, fastapi, gin, echo, etc.
}
```

---

## 6. LSP Implementation

### How Drift Does It

**LSP Server Features:**
```typescript
// packages/lsp/src/server.ts
connection.onInitialize(() => ({
  capabilities: {
    textDocumentSync: TextDocumentSyncKind.Incremental,
    diagnosticProvider: { interFileDependencies: true },
    hoverProvider: true,
    codeActionProvider: true,
    codeLensProvider: { resolveProvider: true }
  }
}));

// Diagnostics: Pattern violations
// Hover: Pattern information
// Code Actions: Quick fixes
// Code Lens: Pattern counts per file
```

### Port to Mind Palace

**Go LSP Server:**
```go
// internal/lsp/server.go
type Server struct {
    conn    *jsonrpc2.Conn
    butler  *butler.Butler
    index   *index.Index
    memory  *memory.Store
}

func (s *Server) Initialize(ctx context.Context, params *InitializeParams) (*InitializeResult, error) {
    return &InitializeResult{
        Capabilities: ServerCapabilities{
            TextDocumentSync: &TextDocumentSyncOptions{
                OpenClose: true,
                Change:    TextDocumentSyncKindIncremental,
            },
            DiagnosticProvider: &DiagnosticOptions{},
            HoverProvider:      true,
            CodeActionProvider: true,
        },
    }, nil
}
```

**Key LSP Methods to Implement:**
1. `textDocument/diagnostic` - Return pattern violations
2. `textDocument/hover` - Show pattern info on hover
3. `textDocument/codeAction` - Quick fixes for violations
4. `textDocument/codeLens` - Show pattern counts

**Integration:**
- LSP server runs alongside MCP server
- Shares Butler and Index instances
- Real-time diagnostics as files change

---

## Summary: Priority & Dependencies

```
┌─────────────────────────────────────────────────────────────┐
│  SPRINT 1: Pattern Detection Foundation                     │
│  ├── Detector Registry (Go)                                │
│  ├── Confidence Scoring Algorithm                          │
│  ├── Pattern Storage (extend memory.db)                    │
│  ├── Integration with Governance (patterns → learnings)    │
│  └── Bulk Approval (CLI + Dashboard)                       │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  SPRINT 2: Contract Detection                               │
│  ├── Backend Endpoint Extractors (Go, TS, Python)          │
│  ├── Frontend API Call Detection                           │
│  ├── Type Schema Extraction                                │
│  ├── Mismatch Detection Engine                             │
│  ├── Contract Storage Model                                │
│  └── Dashboard Visualization                               │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  SPRINT 3: LSP Implementation                               │
│  ├── LSP Server Core (JSON-RPC 2.0)                        │
│  ├── Diagnostics Provider (pattern violations)             │
│  ├── Hover Provider (pattern info)                         │
│  ├── Code Action Provider (quick fixes)                    │
│  └── VS Code Extension Integration                         │
└─────────────────────────────────────────────────────────────┘
```

---

## Key Differences to Maintain

| Aspect | Drift | Mind Palace (Target) |
|--------|-------|---------------------|
| Storage | JSON files | SQLite (existing) |
| Patterns | Standalone | Become Learnings |
| Approval | Simple approve/ignore | 5-phase governance |
| Confidence | Static | Decay over time |
| Scope | Project only | File→Room→Palace→Corridor |
| Philosophy | Pattern enforcement | Knowledge capture |
