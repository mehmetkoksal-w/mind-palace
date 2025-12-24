# Mind Palace Test Implementation Plan

## Overview

Comprehensive plan to increase test coverage from ~55% to ~95% package coverage. Tests are prioritized by risk and impact on contributors breaking functionality.

**Current State:**
- 11/20 packages have tests (55%)
- ~150+ test functions needed
- Critical business logic largely untested

**Goals:**
- Protect core functionality from regressions
- Enable confident contributions from community
- Document expected behavior through tests
- Catch errors early in CI pipeline

---

## Phase 1: Core Analysis Engine

**Goal:** Protect the language parsing foundation that everything depends on.

**Package:** `internal/analysis`

### 1.1 Create `analysis/parser_test.go`

```go
// Test cases needed:

func TestDetectLanguage(t *testing.T)
// - All 25+ file extensions (.go, .ts, .py, .rs, etc.)
// - Special filenames (Dockerfile, Makefile, BUILD, Jenkinsfile)
// - Dockerfile variants (Dockerfile.prod, dockerfile, Containerfile)
// - Unknown extensions return LangUnknown
// - Case sensitivity handling

func TestIsAnalyzable(t *testing.T)
// - Known analyzable files return true
// - Unknown files return false
// - Edge cases: empty path, path with no extension

func TestSupportedExtensions(t *testing.T)
// - Returns non-empty list
// - Contains expected extensions
// - No duplicates

func TestParserRegistry(t *testing.T)
// - Register() adds parser
// - GetParser() retrieves correct parser
// - GetParser() returns nil for unknown language
// - Parse() delegates to correct parser

func TestAnalyze(t *testing.T)
// - Returns FileAnalysis for supported file
// - Returns error for unsupported file
// - Handles empty file content
// - Handles malformed code gracefully
```

### 1.2 Create `analysis/languages_test.go`

```go
// Test each language parser with minimal valid code:

func TestGoParser(t *testing.T)
// - Parse package, func, type, const, var
// - Extract imports and relationships
// - Handle syntax errors gracefully

func TestTypeScriptParser(t *testing.T)
// - Parse class, function, interface, type
// - Handle JSX/TSX variants

func TestPythonParser(t *testing.T)
// - Parse class, def, async def
// - Handle decorators

// ... similar for each supported language
```

**Files to create:**
- `internal/analysis/parser_test.go`
- `internal/analysis/languages_test.go`

**Estimated test count:** 40-50 tests

---

## Phase 2: Data Collection Pipeline

**Goal:** Protect the main orchestration that builds context packs.

**Package:** `internal/collect`

### 2.1 Create `collect/collect_test.go`

```go
func TestRun(t *testing.T)
// - Full workflow with temp directory setup
// - Creates valid context pack
// - Handles missing palace.jsonc
// - Handles missing index database
// - Handles stale file detection
// - Respects scope parameter (full, diff, signal)

func TestRunWithCorridors(t *testing.T)
// - Fetches neighbor context when configured
// - Handles neighbor fetch failures gracefully
// - Merges corridor context into pack

func TestCollectEntryPoints(t *testing.T)
// - Loads entry points from room files
// - Handles missing room files
// - Handles malformed JSONC in rooms
// - Returns empty list for no rooms

func TestMergeOrderedUnique(t *testing.T)
// - Merges two lists preserving order
// - Removes duplicates
// - Handles empty lists
// - Handles nil inputs

func TestPrioritizeHits(t *testing.T)
// - Moves changed files to front
// - Preserves relative order otherwise
// - Handles empty changed list
// - Handles no matching hits

func TestFilterExisting(t *testing.T)
// - Filters paths against metadata
// - Returns only paths that exist in metadata
// - Handles empty inputs
```

**Files to create:**
- `internal/collect/collect_test.go`

**Estimated test count:** 25-30 tests

---

## Phase 3: HTTP Dashboard API

**Goal:** Protect all 18 API endpoints from breaking changes.

**Package:** `internal/dashboard`

### 3.1 Create `dashboard/handlers_test.go`

```go
// Test infrastructure
func setupTestServer(t *testing.T) (*Server, func())
// - Creates server with mock butler, memory, corridor
// - Returns cleanup function

// Health & Status
func TestHandleHealth(t *testing.T)
// - Returns 200 with status info
// - Works when butler is nil

func TestHandleStats(t *testing.T)
// - Returns aggregated statistics
// - Handles nil memory gracefully

// Room & Session endpoints
func TestHandleRooms(t *testing.T)
// - Returns room list as JSON array
// - Returns empty array (not null) when no rooms
// - Returns 503 when butler unavailable

func TestHandleSessions(t *testing.T)
// - Returns session list
// - Respects limit parameter
// - Validates limit is positive
// - Returns empty array when no sessions

func TestHandleSessionDetail(t *testing.T)
// - Returns session by ID
// - Returns 404 for unknown session
// - Parses session ID from URL path

// Activity & Learning endpoints
func TestHandleActivity(t *testing.T)
func TestHandleLearnings(t *testing.T)
func TestHandleFileIntel(t *testing.T)

// Corridor endpoints
func TestHandleCorridors(t *testing.T)
func TestHandleCorridorPersonal(t *testing.T)
func TestHandleCorridorLinks(t *testing.T)

// Search & Graph endpoints
func TestHandleSearch(t *testing.T)
// - Requires query parameter
// - Returns 400 for empty query
// - Respects limit parameter
// - Returns combined results from all sources

func TestHandleGraph(t *testing.T)
func TestHandleHotspots(t *testing.T)
func TestHandleAgents(t *testing.T)
func TestHandleBrief(t *testing.T)

// Workspace management
func TestHandleWorkspaces(t *testing.T)
func TestHandleWorkspaceSwitch(t *testing.T)
// - Validates workspace path exists
// - Returns 400 for invalid path
// - Properly switches resources
// - Handles switch failures gracefully

// HTTP method validation
func TestMethodNotAllowed(t *testing.T)
// - All GET endpoints reject POST/PUT/DELETE
// - Returns 405 Method Not Allowed

// Concurrent access
func TestConcurrentAccess(t *testing.T)
// - Multiple concurrent requests don't race
// - Workspace switch doesn't break active requests
```

### 3.2 Create `dashboard/server_test.go`

```go
func TestCorsMiddleware(t *testing.T)
// - Adds CORS headers
// - Handles OPTIONS preflight

func TestWriteJSON(t *testing.T)
// - Sets Content-Type header
// - Encodes data as JSON

func TestWriteError(t *testing.T)
// - Sets error status code
// - Returns JSON error body
```

**Files to create:**
- `internal/dashboard/handlers_test.go`
- `internal/dashboard/server_test.go`

**Estimated test count:** 35-40 tests

---

## Phase 4: Index & Scan Operations

**Goal:** Protect file indexing and change detection.

**Packages:** `internal/scan`, `internal/stale`, `internal/index`

### 4.1 Create `scan/scan_test.go`

```go
func TestRun(t *testing.T)
// - Full scan creates database
// - Indexes all files matching guardrails
// - Creates scan.json artifact
// - Handles invalid root path

func TestRunIncremental(t *testing.T)
// - Returns error when no index exists
// - Detects added files
// - Detects modified files
// - Detects deleted files
// - Calculates FilesUnchanged correctly
// - Faster than full scan for few changes
```

### 4.2 Create `stale/stale_test.go`

```go
func TestDetect(t *testing.T)
// - Fast mode: checks file existence only
// - Strict mode: checks content hashes
// - Detects new files
// - Detects modified files
// - Detects deleted files
// - Respects guardrails
// - includeMissing parameter behavior

func TestDetectWithGuardrails(t *testing.T)
// - Ignores files matching exclude patterns
// - Only checks files matching include patterns
```

### 4.3 Extend `index/index_test.go`

```go
func TestDetectChanges(t *testing.T)
// - Detects added files
// - Detects modified files (hash change)
// - Detects deleted files
// - Returns empty for no changes

func TestIncrementalScan(t *testing.T)
// - Processes only changed files
// - Updates database correctly
// - Handles errors in single file gracefully

func TestDeleteFileFromIndex(t *testing.T)
// - Removes file record
// - Removes associated chunks
// - Removes associated symbols
// - Cleans up FTS tables
```

**Files to create:**
- `internal/scan/scan_test.go`
- `internal/stale/stale_test.go`

**Files to extend:**
- `internal/index/index_test.go`

**Estimated test count:** 30-35 tests

---

## Phase 5: Project Detection & Validation

**Goal:** Protect language detection and artifact validation.

**Packages:** `internal/project`, `internal/lint`, `internal/validate`

### 5.1 Create `project/profile_test.go`

```go
func TestBuildProfile(t *testing.T)
// - Returns valid ProjectProfile
// - Detects languages from marker files

func TestDetectLanguages(t *testing.T)
// - go.mod -> Go
// - package.json -> JavaScript/TypeScript
// - Cargo.toml -> Rust
// - pyproject.toml/requirements.txt -> Python
// - pom.xml/build.gradle -> Java
// - Multiple languages detected
// - Returns "unknown" when no markers

func TestDefaultCommands(t *testing.T)
// - defaultGraphCommand for each language
// - defaultTestCommand for each language
// - defaultLintCommand for each language
// - Unknown language returns empty string
```

### 5.2 Create `lint/lint_test.go`

```go
func TestRun(t *testing.T)
// - Passes for valid palace structure
// - Reports missing palace.jsonc
// - Reports invalid room files
// - Reports invalid playbook files
// - Aggregates multiple errors
```

### 5.3 Create `validate/validate_test.go`

```go
func TestJSONC(t *testing.T)
// - Validates correct JSONC files
// - Reports schema violations
// - Handles missing files
// - Handles invalid JSONC syntax

func TestJSON(t *testing.T)
// - Validates correct JSON files
// - Reports schema violations
```

**Files to create:**
- `internal/project/profile_test.go`
- `internal/lint/lint_test.go`
- `internal/validate/validate_test.go`

**Estimated test count:** 25-30 tests

---

## Phase 6: Extend Existing Tests

**Goal:** Fill gaps in packages that already have some tests.

### 6.1 Extend `butler/butler_test.go`

```go
func TestNew(t *testing.T)
// - Creates butler with valid database
// - Handles missing database

func TestSearch(t *testing.T)
// - Returns results for matching query
// - Returns empty for no matches
// - Respects limit parameter
// - Combines symbol and chunk results

func TestListRooms(t *testing.T)
// - Returns all room definitions
// - Returns empty array for no rooms

func TestGetIncomingCalls(t *testing.T)
func TestGetOutgoingCalls(t *testing.T)
// - Returns call relationships
// - Handles unknown symbols
```

### 6.2 Extend `memory/memory_test.go`

```go
func TestLearningOperations(t *testing.T)
// - RecordLearning creates learning
// - GetLearnings retrieves by scope
// - UpdateLearningConfidence modifies confidence
// - DecayUnusedLearnings reduces old learnings

func TestFileIntel(t *testing.T)
// - RecordFileIntel stores intel
// - GetFileIntel retrieves by path
// - GetRecentFileIntel respects limit

func TestAgentTracking(t *testing.T)
// - RegisterAgent creates agent
// - UpdateAgentHeartbeat updates timestamp
// - GetActiveAgents returns non-stale agents
// - CleanupStaleAgents removes old agents

func TestConflictDetection(t *testing.T)
// - DetectConflict finds overlapping edits
// - Returns nil for no conflict
```

### 6.3 Extend `corridor/corridor_test.go`

```go
func TestFetchNeighbors(t *testing.T)
// - Fetches from local path
// - Fetches from URL (mock server)
// - Uses cache when within TTL
// - Falls back to cache on network error

func TestApplyAuth(t *testing.T)
// - Bearer token authentication
// - Basic authentication
// - Custom header authentication
// - Environment variable expansion

func TestCacheOperations(t *testing.T)
// - cacheContextPack writes file
// - cacheRooms writes room files
// - loadFromCache reads cached data
// - Cache expires after TTL
```

### 6.4 Extend `config/config_test.go`

```go
func TestLoadConfig(t *testing.T)
// - Loads valid palace.jsonc
// - Returns error for missing file
// - Returns error for invalid JSONC

func TestEnsureLayout(t *testing.T)
// - Creates .palace directory structure
// - Handles permission errors
// - Idempotent on existing layout
```

### 6.5 Extend `fsutil/fsutil_test.go`

```go
func TestListFiles(t *testing.T)
// - Lists all files in directory
// - Respects guardrail excludes
// - Handles permission errors
// - Returns relative paths

func TestHashFile(t *testing.T)
// - Returns consistent hash
// - Handles missing file
// - Handles permission error

func TestChunkContent(t *testing.T)
// - Splits content into chunks
// - Respects line limits
// - Respects byte limits
// - Handles empty content
```

**Files to extend:**
- `internal/butler/butler_test.go`
- `internal/memory/memory_test.go`
- `internal/corridor/corridor_test.go`
- `internal/config/config_test.go`
- `internal/fsutil/fsutil_test.go`

**Estimated test count:** 40-50 tests

---

## Phase 7: Utility Packages

**Goal:** Cover remaining utility packages.

### 7.1 Create `jsonc/jsonc_test.go`

```go
func TestDecodeFile(t *testing.T)
// - Decodes valid JSONC with comments
// - Decodes JSONC with trailing commas
// - Returns error for missing file
// - Returns error for invalid syntax
// - Returns error for type mismatch

func TestClean(t *testing.T)
// - Removes // comments
// - Preserves URLs in strings
// - Removes trailing commas
// - Handles nested structures
```

### 7.2 Create `update/update_test.go`

```go
func TestCompareVersions(t *testing.T)
// - v1.0.0 < v1.0.1
// - v1.0.0 < v1.1.0
// - v1.0.0 < v2.0.0
// - v1.0.0-beta < v1.0.0
// - Equal versions return 0

func TestParseVersion(t *testing.T)
// - Parses major.minor.patch
// - Handles pre-release suffix
// - Handles v prefix

func TestBuildAssetName(t *testing.T)
// - Correct name for darwin/amd64
// - Correct name for darwin/arm64
// - Correct name for linux/amd64
// - Correct name for windows/amd64

func TestCacheOperations(t *testing.T)
// - saveCache writes cache file
// - loadCache reads cache file
// - Cache expires after TTL
// - Handles missing cache file
```

**Files to create:**
- `internal/jsonc/jsonc_test.go`
- `internal/update/update_test.go`

**Estimated test count:** 20-25 tests

---

## Implementation Guidelines

### Test File Structure

```go
package pkgname

import (
    "testing"
    // other imports
)

// setupTest creates test fixtures and returns cleanup function
func setupTest(t *testing.T) (fixture, func()) {
    t.Helper()
    // setup code
    return fixture, func() {
        // cleanup code
    }
}

// Table-driven tests for multiple cases
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected OutputType
        wantErr  bool
    }{
        {"valid input", validInput, expectedOutput, false},
        {"empty input", "", zero, true},
        // more cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.expected {
                t.Errorf("got %v, want %v", got, tt.expected)
            }
        })
    }
}
```

### Test Naming Conventions

- `TestFunctionName` - Tests the happy path
- `TestFunctionName_ErrorCase` - Tests specific error condition
- `TestFunctionName_EdgeCase` - Tests boundary conditions

### Mock & Fixture Guidelines

1. Use `t.TempDir()` for filesystem tests
2. Use `httptest.NewServer()` for HTTP tests
3. Use interfaces for dependency injection where practical
4. Avoid global state in tests

### CI Integration

Add to `.github/workflows/test.yml`:

```yaml
- name: Run tests with coverage
  run: go test -v -race -coverprofile=coverage.out ./...

- name: Check coverage threshold
  run: |
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    if (( $(echo "$COVERAGE < 70" | bc -l) )); then
      echo "Coverage $COVERAGE% is below 70% threshold"
      exit 1
    fi
```

---

## Summary

| Phase | Focus | New Files | Tests | Priority |
|-------|-------|-----------|-------|----------|
| 1 | Analysis Engine | 2 | 40-50 | Critical |
| 2 | Collection Pipeline | 1 | 25-30 | Critical |
| 3 | Dashboard API | 2 | 35-40 | Critical |
| 4 | Index & Scan | 2 | 30-35 | High |
| 5 | Project & Validation | 3 | 25-30 | High |
| 6 | Extend Existing | 0 | 40-50 | Medium |
| 7 | Utilities | 2 | 20-25 | Medium |
| **Total** | | **12** | **~220** | |

---

## Success Criteria

- [ ] All packages have test files
- [ ] Coverage > 70% for critical packages (analysis, collect, dashboard)
- [ ] Coverage > 60% overall
- [ ] All tests pass in CI
- [ ] No race conditions (`go test -race`)
- [ ] Tests run in < 60 seconds total
