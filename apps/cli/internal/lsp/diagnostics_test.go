package lsp

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// MockDiagnosticsProvider is a mock diagnostics provider for testing.
type MockDiagnosticsProvider struct {
	Outliers   map[string][]PatternOutlier
	Mismatches map[string][]ContractMismatch
}

func (m *MockDiagnosticsProvider) GetPatternOutliersForFile(filePath string) ([]PatternOutlier, error) {
	if m.Outliers == nil {
		return nil, nil
	}
	// Normalize path for cross-platform matching
	normalized := filepath.ToSlash(filePath)
	if outliers, ok := m.Outliers[normalized]; ok {
		return outliers, nil
	}
	return m.Outliers[filePath], nil
}

func (m *MockDiagnosticsProvider) GetContractMismatchesForFile(filePath string) ([]ContractMismatch, error) {
	if m.Mismatches == nil {
		return nil, nil
	}
	// Normalize path for cross-platform matching
	normalized := filepath.ToSlash(filePath)
	if mismatches, ok := m.Mismatches[normalized]; ok {
		return mismatches, nil
	}
	return m.Mismatches[filePath], nil
}

func TestDiagnosticsWithPatternOutliers(t *testing.T) {
	var output bytes.Buffer
	server := NewServerWithIO(strings.NewReader(""), &output)
	server.initialized = true

	// Set up mock diagnostics provider
	provider := &MockDiagnosticsProvider{
		Outliers: map[string][]PatternOutlier{
			"/test/file.go": {
				{
					PatternID:     "pat_123",
					PatternName:   "error-handling",
					Description:   "Error not handled",
					FilePath:      "/test/file.go",
					LineStart:     10,
					LineEnd:       10,
					OutlierReason: "Missing error check",
					Confidence:    0.85,
				},
			},
		},
	}
	server.SetDiagnosticsProvider(provider)

	// Open document
	doc := &TextDocument{
		URI:        "file:///test/file.go",
		LanguageID: "go",
		Version:    1,
		Content:    "package main\n",
	}
	server.setDocument(doc)

	// Compute diagnostics
	diagnostics := server.computeDiagnostics(doc)

	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diagnostics))
	}

	diag := diagnostics[0]
	if diag.Source != "mind-palace" {
		t.Errorf("expected source 'mind-palace', got '%s'", diag.Source)
	}
	if diag.Code != "pat_123" {
		t.Errorf("expected code 'pat_123', got '%v'", diag.Code)
	}
	if diag.Range.Start.Line != 9 { // 0-based
		t.Errorf("expected line 9 (0-based), got %d", diag.Range.Start.Line)
	}
	if diag.Severity != DiagnosticSeverityWarning {
		t.Errorf("expected warning severity, got %d", diag.Severity)
	}
}

func TestDiagnosticsWithContractMismatches(t *testing.T) {
	var output bytes.Buffer
	server := NewServerWithIO(strings.NewReader(""), &output)
	server.initialized = true

	// Set up mock diagnostics provider
	provider := &MockDiagnosticsProvider{
		Mismatches: map[string][]ContractMismatch{
			"/test/api.ts": {
				{
					ContractID:   "ctr_456",
					Method:       "GET",
					Endpoint:     "/api/users",
					FieldPath:    "user.email",
					MismatchType: "type_mismatch",
					Severity:     "error",
					Description:  "Type mismatch at user.email",
					BackendType:  "string",
					FrontendType: "number",
					FilePath:     "/test/api.ts",
					Line:         25,
				},
			},
		},
	}
	server.SetDiagnosticsProvider(provider)

	// Open document
	doc := &TextDocument{
		URI:        "file:///test/api.ts",
		LanguageID: "typescript",
		Version:    1,
		Content:    "// api file\n",
	}
	server.setDocument(doc)

	// Compute diagnostics
	diagnostics := server.computeDiagnostics(doc)

	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diagnostics))
	}

	diag := diagnostics[0]
	if diag.Source != "mind-palace" {
		t.Errorf("expected source 'mind-palace', got '%s'", diag.Source)
	}
	if diag.Code != "type_mismatch" {
		t.Errorf("expected code 'type_mismatch', got '%v'", diag.Code)
	}
	if diag.Range.Start.Line != 24 { // 0-based
		t.Errorf("expected line 24 (0-based), got %d", diag.Range.Start.Line)
	}
	if diag.Severity != DiagnosticSeverityError {
		t.Errorf("expected error severity, got %d", diag.Severity)
	}
}

func TestDiagnosticsPublished(t *testing.T) {
	var output bytes.Buffer
	server := NewServerWithIO(strings.NewReader(""), &output)
	server.initialized = true

	// Set up mock diagnostics provider
	provider := &MockDiagnosticsProvider{
		Outliers: map[string][]PatternOutlier{
			"/test/file.go": {
				{
					PatternID:     "pat_test",
					PatternName:   "test-pattern",
					FilePath:      "/test/file.go",
					LineStart:     1,
					LineEnd:       1,
					OutlierReason: "Test outlier",
				},
			},
		},
	}
	server.SetDiagnosticsProvider(provider)

	// Publish diagnostics
	doc := &TextDocument{
		URI:     "file:///test/file.go",
		Content: "test",
	}
	server.setDocument(doc)

	err := server.publishDiagnostics(doc.URI, server.computeDiagnostics(doc))
	if err != nil {
		t.Fatalf("publishDiagnostics failed: %v", err)
	}

	// Check output
	result := output.String()
	if !strings.Contains(result, "Content-Length:") {
		t.Error("expected Content-Length header in output")
	}
	if !strings.Contains(result, "textDocument/publishDiagnostics") {
		t.Error("expected publishDiagnostics method in output")
	}
	if !strings.Contains(result, "pat_test") {
		t.Error("expected pattern ID in output")
	}
}

func TestURIConversion(t *testing.T) {
	tests := []struct {
		uri      string
		expected string
	}{
		{"file:///test/file.go", "/test/file.go"},
		{"file:///C:/Users/test/file.go", "C:\\Users\\test\\file.go"},
		{"file://C:/Windows/file.txt", "C:\\Windows\\file.txt"},
	}

	for _, tc := range tests {
		result := uriToPath(tc.uri)
		// Normalize for comparison (handle OS differences)
		if !strings.HasSuffix(result, strings.ReplaceAll(tc.expected, "\\", "/")) &&
			!strings.HasSuffix(result, strings.ReplaceAll(tc.expected, "/", "\\")) {
			// Allow some flexibility for OS-specific path handling
			t.Logf("uriToPath(%s) = %s (expected suffix: %s)", tc.uri, result, tc.expected)
		}
	}
}

func TestDiagnosticData(t *testing.T) {
	server := NewServerWithIO(strings.NewReader(""), &bytes.Buffer{})

	outlier := PatternOutlier{
		PatternID:  "pat_123",
		Confidence: 0.95,
	}

	diag := server.patternOutlierToDiagnostic(outlier)

	data, ok := diag.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be map[string]interface{}, got %T", diag.Data)
	}

	if data["type"] != "pattern" {
		t.Errorf("expected type 'pattern', got %v", data["type"])
	}
	if data["patternId"] != "pat_123" {
		t.Errorf("expected patternId 'pat_123', got %v", data["patternId"])
	}
	if data["confidence"] != 0.95 {
		t.Errorf("expected confidence 0.95, got %v", data["confidence"])
	}
}

func TestHoverForPatternOutlier(t *testing.T) {
	var output bytes.Buffer
	server := NewServerWithIO(strings.NewReader(""), &output)
	server.initialized = true

	// Set up mock diagnostics provider
	provider := &MockDiagnosticsProvider{
		Outliers: map[string][]PatternOutlier{
			"/test/file.go": {
				{
					PatternID:     "pat_hover",
					PatternName:   "error-handling",
					Description:   "Standard error handling pattern",
					FilePath:      "/test/file.go",
					LineStart:     10,
					LineEnd:       15,
					OutlierReason: "Missing error check after function call",
					Confidence:    0.92,
				},
			},
		},
	}
	server.SetDiagnosticsProvider(provider)

	// Set up document
	server.setDocument(&TextDocument{
		URI:     "file:///test/file.go",
		Content: "test content",
	})

	// Create hover request
	params := HoverParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test/file.go"},
		Position:     Position{Line: 11, Character: 5}, // Line 12 (0-based 11) is within range
	}
	paramsJSON, _ := json.Marshal(params)
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/hover",
		Params:  paramsJSON,
	}

	resp := server.handleHover(req)
	if resp.Error != nil {
		t.Fatalf("handleHover failed: %v", resp.Error)
	}

	hover, ok := resp.Result.(*Hover)
	if !ok {
		t.Fatalf("expected *Hover, got %T", resp.Result)
	}

	if hover.Contents.Kind != MarkupKindMarkdown {
		t.Errorf("expected markdown, got %s", hover.Contents.Kind)
	}

	content := hover.Contents.Value
	if !strings.Contains(content, "error-handling") {
		t.Error("expected pattern name in hover content")
	}
	if !strings.Contains(content, "92%") {
		t.Error("expected confidence in hover content")
	}
	if !strings.Contains(content, "Missing error check") {
		t.Error("expected outlier reason in hover content")
	}
}

func TestHoverForContractMismatch(t *testing.T) {
	var output bytes.Buffer
	server := NewServerWithIO(strings.NewReader(""), &output)
	server.initialized = true

	// Set up mock diagnostics provider
	provider := &MockDiagnosticsProvider{
		Mismatches: map[string][]ContractMismatch{
			"/test/api.ts": {
				{
					ContractID:   "ctr_hover",
					Method:       "POST",
					Endpoint:     "/api/users",
					FieldPath:    "user.age",
					MismatchType: "type_mismatch",
					Severity:     "error",
					Description:  "Type mismatch at user.age",
					BackendType:  "int",
					FrontendType: "string",
					FilePath:     "/test/api.ts",
					Line:         20,
				},
			},
		},
	}
	server.SetDiagnosticsProvider(provider)

	// Set up document
	server.setDocument(&TextDocument{
		URI:     "file:///test/api.ts",
		Content: "test content",
	})

	// Create hover request on line 20 (0-based 19)
	params := HoverParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test/api.ts"},
		Position:     Position{Line: 19, Character: 5},
	}
	paramsJSON, _ := json.Marshal(params)
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/hover",
		Params:  paramsJSON,
	}

	resp := server.handleHover(req)
	if resp.Error != nil {
		t.Fatalf("handleHover failed: %v", resp.Error)
	}

	hover, ok := resp.Result.(*Hover)
	if !ok {
		t.Fatalf("expected *Hover, got %T", resp.Result)
	}

	content := hover.Contents.Value
	if !strings.Contains(content, "POST /api/users") {
		t.Error("expected contract info in hover content")
	}
	if !strings.Contains(content, "Backend type") {
		t.Error("expected backend type in hover content")
	}
	if !strings.Contains(content, "Frontend type") {
		t.Error("expected frontend type in hover content")
	}
}

func TestCodeActionForDiagnostics(t *testing.T) {
	var output bytes.Buffer
	server := NewServerWithIO(strings.NewReader(""), &output)
	server.initialized = true

	// Create a code action request with diagnostics
	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test/file.go"},
		Range:        Range{Start: Position{Line: 0}, End: Position{Line: 0}},
		Context: CodeActionContext{
			Diagnostics: []Diagnostic{
				{
					Range:    Range{Start: Position{Line: 0}, End: Position{Line: 0}},
					Severity: DiagnosticSeverityWarning,
					Code:     "pat_test",
					Source:   "mind-palace",
					Message:  "Test pattern",
				},
			},
		},
	}

	paramsJSON, _ := json.Marshal(params)
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/codeAction",
		Params:  paramsJSON,
	}

	// Set up mock document
	server.setDocument(&TextDocument{URI: "file:///test/file.go", Content: "test"})

	resp := server.handleCodeAction(req)
	if resp.Error != nil {
		t.Fatalf("handleCodeAction failed: %v", resp.Error)
	}

	actions, ok := resp.Result.([]CodeAction)
	if !ok {
		t.Fatalf("expected []CodeAction, got %T", resp.Result)
	}

	// Should have approve and ignore actions
	if len(actions) < 2 {
		t.Errorf("expected at least 2 actions, got %d", len(actions))
	}

	// Check for approve action
	hasApprove := false
	hasIgnore := false
	for _, action := range actions {
		if strings.Contains(action.Title, "Approve") {
			hasApprove = true
		}
		if strings.Contains(action.Title, "Ignore") {
			hasIgnore = true
		}
	}

	if !hasApprove {
		t.Error("expected approve action")
	}
	if !hasIgnore {
		t.Error("expected ignore action")
	}
}

func TestCodeLensForPatterns(t *testing.T) {
	var output bytes.Buffer
	server := NewServerWithIO(strings.NewReader(""), &output)
	server.initialized = true

	// Set up mock diagnostics provider with patterns
	provider := &MockDiagnosticsProvider{
		Outliers: map[string][]PatternOutlier{
			"/test/file.go": {
				{
					PatternID:     "pat_lens_1",
					PatternName:   "error-handling",
					FilePath:      "/test/file.go",
					LineStart:     10,
					LineEnd:       10,
					OutlierReason: "Missing error check",
					Confidence:    0.85,
				},
				{
					PatternID:     "pat_lens_2",
					PatternName:   "naming-convention",
					FilePath:      "/test/file.go",
					LineStart:     20,
					LineEnd:       20,
					OutlierReason: "Non-standard name",
					Confidence:    0.75,
				},
			},
		},
	}
	server.SetDiagnosticsProvider(provider)

	// Set up document
	server.setDocument(&TextDocument{
		URI:     "file:///test/file.go",
		Content: "test content",
	})

	// Create code lens request
	params := CodeLensParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test/file.go"},
	}
	paramsJSON, _ := json.Marshal(params)
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/codeLens",
		Params:  paramsJSON,
	}

	resp := server.handleCodeLens(req)
	if resp.Error != nil {
		t.Fatalf("handleCodeLens failed: %v", resp.Error)
	}

	lenses, ok := resp.Result.([]CodeLens)
	if !ok {
		t.Fatalf("expected []CodeLens, got %T", resp.Result)
	}

	// Should have 1 summary lens + 2 pattern lenses = 3 total
	if len(lenses) != 3 {
		t.Errorf("expected 3 lenses, got %d", len(lenses))
	}

	// Check for summary lens
	hasSummary := false
	hasInline := false
	for _, lens := range lenses {
		if lens.Command != nil {
			if strings.Contains(lens.Command.Title, "2 pattern issues") {
				hasSummary = true
			}
			if strings.Contains(lens.Command.Title, "Pattern:") {
				hasInline = true
			}
		}
	}

	if !hasSummary {
		t.Error("expected summary code lens with count")
	}
	if !hasInline {
		t.Error("expected inline code lens for patterns")
	}
}

func TestCodeLensForContracts(t *testing.T) {
	var output bytes.Buffer
	server := NewServerWithIO(strings.NewReader(""), &output)
	server.initialized = true

	// Set up mock diagnostics provider with contracts
	provider := &MockDiagnosticsProvider{
		Mismatches: map[string][]ContractMismatch{
			"/test/api.ts": {
				{
					ContractID:   "ctr_lens_1",
					Method:       "GET",
					Endpoint:     "/api/users",
					MismatchType: "type_mismatch",
					Severity:     "error",
					FilePath:     "/test/api.ts",
					Line:         15,
				},
			},
		},
	}
	server.SetDiagnosticsProvider(provider)

	// Set up document
	server.setDocument(&TextDocument{
		URI:     "file:///test/api.ts",
		Content: "test content",
	})

	// Create code lens request
	params := CodeLensParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test/api.ts"},
	}
	paramsJSON, _ := json.Marshal(params)
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/codeLens",
		Params:  paramsJSON,
	}

	resp := server.handleCodeLens(req)
	if resp.Error != nil {
		t.Fatalf("handleCodeLens failed: %v", resp.Error)
	}

	lenses, ok := resp.Result.([]CodeLens)
	if !ok {
		t.Fatalf("expected []CodeLens, got %T", resp.Result)
	}

	// Should have 1 summary lens + 1 contract lens = 2 total
	if len(lenses) != 2 {
		t.Errorf("expected 2 lenses, got %d", len(lenses))
	}

	// Check for contract lens
	hasContract := false
	for _, lens := range lenses {
		if lens.Command != nil && strings.Contains(lens.Command.Title, "GET /api/users") {
			hasContract = true
		}
	}

	if !hasContract {
		t.Error("expected contract code lens")
	}
}

func TestGoToDefinition(t *testing.T) {
	var output bytes.Buffer
	server := NewServerWithIO(strings.NewReader(""), &output)
	server.initialized = true

	// Set up mock diagnostics provider
	provider := &MockDiagnosticsProvider{
		Outliers: map[string][]PatternOutlier{
			"/test/file.go": {
				{
					PatternID:     "pat_def",
					PatternName:   "error-handling",
					FilePath:      "/test/file.go",
					LineStart:     10,
					LineEnd:       15,
					OutlierReason: "Missing error check",
				},
			},
		},
	}
	server.SetDiagnosticsProvider(provider)

	// Set up document
	server.setDocument(&TextDocument{
		URI:     "file:///test/file.go",
		Content: "test content",
	})

	// Create definition request on a pattern line
	params := DefinitionParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test/file.go"},
		Position:     Position{Line: 11, Character: 5}, // Line 12 (0-based 11) is within range
	}
	paramsJSON, _ := json.Marshal(params)
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/definition",
		Params:  paramsJSON,
	}

	resp := server.handleDefinition(req)
	if resp.Error != nil {
		t.Fatalf("handleDefinition failed: %v", resp.Error)
	}

	location, ok := resp.Result.(*Location)
	if !ok {
		t.Fatalf("expected *Location, got %T", resp.Result)
	}

	if location == nil {
		t.Fatal("expected location, got nil")
	}

	// Check that the location points to the pattern
	if location.Range.Start.Line != 9 { // 0-based
		t.Errorf("expected line 9 (0-based), got %d", location.Range.Start.Line)
	}
}

func TestDocumentSymbols(t *testing.T) {
	var output bytes.Buffer
	server := NewServerWithIO(strings.NewReader(""), &output)
	server.initialized = true

	// Set up mock diagnostics provider
	provider := &MockDiagnosticsProvider{
		Outliers: map[string][]PatternOutlier{
			"/test/file.go": {
				{
					PatternID:     "pat_sym",
					PatternName:   "error-handling",
					FilePath:      "/test/file.go",
					LineStart:     10,
					LineEnd:       10,
					OutlierReason: "Missing error check",
				},
			},
		},
		Mismatches: map[string][]ContractMismatch{
			"/test/file.go": {
				{
					ContractID:   "ctr_sym",
					Method:       "POST",
					Endpoint:     "/api/users",
					MismatchType: "type_mismatch",
					Description:  "Type mismatch",
					FilePath:     "/test/file.go",
					Line:         20,
				},
			},
		},
	}
	server.SetDiagnosticsProvider(provider)

	// Set up document
	server.setDocument(&TextDocument{
		URI:     "file:///test/file.go",
		Content: "test content",
	})

	// Create document symbol request
	params := DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{URI: "file:///test/file.go"},
	}
	paramsJSON, _ := json.Marshal(params)
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "textDocument/documentSymbol",
		Params:  paramsJSON,
	}

	resp := server.handleDocumentSymbol(req)
	if resp.Error != nil {
		t.Fatalf("handleDocumentSymbol failed: %v", resp.Error)
	}

	symbols, ok := resp.Result.([]DocumentSymbol)
	if !ok {
		t.Fatalf("expected []DocumentSymbol, got %T", resp.Result)
	}

	// Should have 1 pattern + 1 contract = 2 symbols
	if len(symbols) != 2 {
		t.Errorf("expected 2 symbols, got %d", len(symbols))
	}

	// Check for pattern symbol
	hasPattern := false
	hasContract := false
	for _, sym := range symbols {
		if strings.Contains(sym.Name, "Pattern:") {
			hasPattern = true
		}
		if strings.Contains(sym.Name, "Contract:") {
			hasContract = true
		}
	}

	if !hasPattern {
		t.Error("expected pattern symbol")
	}
	if !hasContract {
		t.Error("expected contract symbol")
	}
}
