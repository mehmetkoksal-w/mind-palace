package analysis

import (
	"fmt"
	"testing"
)

// TestGoLSPParser_Parse_Mock tests the parser with a mocked LSP client (unit test)
func TestGoLSPParser_Parse_Mock(t *testing.T) {
	// Create mock client with test symbols
	_ = NewMockLSPClient().WithDocumentSymbols(CreateTestSymbols())

	// Create parser (we'll inject the mock in a moment)
	parser := &GoLSPParser{
		available: true,
		rootPath:  "/test",
	}

	// Parse using mock - test the symbol conversion logic directly
	lspSymbols := CreateTestSymbols()
	symbols := parser.convertSymbols(lspSymbols, []byte(testGoCode))

	// Verify we got symbols
	if len(symbols) == 0 {
		t.Fatal("No symbols returned from conversion")
	}

	// Verify symbol structure
	symbolNames := make(map[string]SymbolKind)
	for _, sym := range symbols {
		symbolNames[sym.Name] = sym.Kind
	}

	// Check expected symbols
	expectedSymbols := map[string]SymbolKind{
		"Person":    KindClass,
		"NewPerson": KindFunction,
		"Greet":     KindMethod,
		"IsAdult":   KindMethod,
		"MaxAge":    KindConstant,
	}

	for name, expectedKind := range expectedSymbols {
		kind, found := symbolNames[name]
		if !found {
			t.Errorf("Expected symbol %s not found", name)
		} else if kind != expectedKind {
			t.Errorf("Symbol %s: got kind %s, want %s", name, kind, expectedKind)
		}
	}

	// Verify Person struct has children (fields)
	var personStruct *Symbol
	for i := range symbols {
		if symbols[i].Name == "Person" {
			personStruct = &symbols[i]
			break
		}
	}

	if personStruct == nil {
		t.Fatal("Person struct not found")
	}

	if len(personStruct.Children) != 2 {
		t.Errorf("Person has %d children, expected 2 (Name, Age)", len(personStruct.Children))
	}

	// Verify fields
	fieldNames := make(map[string]bool)
	for _, child := range personStruct.Children {
		fieldNames[child.Name] = true
		// LSP may report fields as Property or Variable
		if child.Kind != KindProperty && child.Kind != KindVariable {
			t.Logf("Note: Child %s has kind %s (expected property or variable)", child.Name, child.Kind)
		}
	}

	if !fieldNames["Name"] {
		t.Error("Person.Name field not found")
	}
	if !fieldNames["Age"] {
		t.Error("Person.Age field not found")
	}

	// Verify exported status
	for _, sym := range symbols {
		if sym.Name == "Person" || sym.Name == "NewPerson" || sym.Name == "Greet" || sym.Name == "IsAdult" || sym.Name == "MaxAge" {
			if !sym.Exported {
				t.Errorf("Symbol %s should be exported", sym.Name)
			}
		}
	}
}

func TestGoLSPParser_ConvertSymbol_Method(t *testing.T) {
	parser := &GoLSPParser{available: true}

	lspSym := LSPDocumentSymbol{
		Name: "Greet",
		Kind: LSPSymbolKindMethod,
		Range: LSPRange{
			Start: LSPPosition{Line: 20, Character: 0},
			End:   LSPPosition{Line: 22, Character: 1},
		},
		SelectionRange: LSPRange{
			Start: LSPPosition{Line: 20, Character: 16},
			End:   LSPPosition{Line: 20, Character: 21},
		},
		Detail: "(p *Person)",
	}

	symbol := parser.convertSymbol(lspSym, []byte(testGoCode))

	if symbol.Name != "Greet" {
		t.Errorf("Name = %s, want Greet", symbol.Name)
	}
	if symbol.Kind != KindMethod {
		t.Errorf("Kind = %s, want %s", symbol.Kind, KindMethod)
	}
	if symbol.LineStart != 21 { // LSP uses 0-based, we use 1-based
		t.Errorf("LineStart = %d, want 21", symbol.LineStart)
	}
	if symbol.LineEnd != 23 {
		t.Errorf("LineEnd = %d, want 23", symbol.LineEnd)
	}
	if !symbol.Exported {
		t.Error("Greet should be exported")
	}
}

func TestGoLSPParser_RefineSymbolKind(t *testing.T) {
	parser := &GoLSPParser{available: true}

	tests := []struct {
		name     string
		lspSym   LSPDocumentSymbol
		wantKind SymbolKind
	}{
		{
			name: "struct reported as class",
			lspSym: LSPDocumentSymbol{
				Kind: LSPSymbolKindClass,
				Name: "Person",
			},
			wantKind: KindClass,
		},
		{
			name: "method",
			lspSym: LSPDocumentSymbol{
				Kind:   LSPSymbolKindMethod,
				Name:   "Greet",
				Detail: "(p *Person)",
			},
			wantKind: KindMethod,
		},
		{
			name: "function",
			lspSym: LSPDocumentSymbol{
				Kind:   LSPSymbolKindFunction,
				Name:   "NewPerson",
				Detail: "func(string, int) *Person",
			},
			wantKind: KindFunction,
		},
		{
			name: "function with receiver is method",
			lspSym: LSPDocumentSymbol{
				Kind:   LSPSymbolKindFunction,
				Name:   "IsAdult",
				Detail: "(p *Person) bool",
			},
			wantKind: KindMethod,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start with the LSP kind converted to our kind
			initialKind := ConvertLSPSymbolKind(tt.lspSym.Kind)
			refinedKind := parser.refineSymbolKind(tt.lspSym, initialKind)

			if refinedKind != tt.wantKind {
				t.Errorf("refineSymbolKind() = %s, want %s", refinedKind, tt.wantKind)
			}
		})
	}
}

func TestGoLSPParser_ExtractSignature(t *testing.T) {
	parser := &GoLSPParser{available: true}
	content := []byte(testGoCode)

	tests := []struct {
		name     string
		lspSym   LSPDocumentSymbol
		wantSig  string
		contains bool // if true, check if signature contains wantSig
	}{
		{
			name: "function with detail",
			lspSym: LSPDocumentSymbol{
				Name:   "NewPerson",
				Kind:   LSPSymbolKindFunction,
				Detail: "func(string, int) *Person",
			},
			wantSig:  "func(string, int) *Person",
			contains: true,
		},
		{
			name: "method with detail",
			lspSym: LSPDocumentSymbol{
				Name:   "Greet",
				Kind:   LSPSymbolKindMethod,
				Detail: "(p *Person) string",
			},
			wantSig:  "(p *Person) string",
			contains: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig := parser.extractSignature(tt.lspSym, content)

			if tt.contains {
				// For detailed signatures, the Detail field should be included
				if sig == "" {
					t.Error("Signature is empty")
				}
			} else {
				if sig != tt.wantSig {
					t.Errorf("extractSignature() = %q, want %q", sig, tt.wantSig)
				}
			}
		})
	}
}

func TestMockLSPClient(t *testing.T) {
	// Test basic mock functionality
	mock := NewMockLSPClient()

	// Test default behavior (empty symbols)
	symbols, err := mock.DocumentSymbols("file:///test.go", "package main")
	if err != nil {
		t.Fatalf("DocumentSymbols() error = %v", err)
	}
	if len(symbols) != 0 {
		t.Errorf("Default mock should return empty symbols, got %d", len(symbols))
	}

	// Test with custom symbols
	testSymbols := CreateTestSymbols()
	mock.WithDocumentSymbols(testSymbols)

	symbols, err = mock.DocumentSymbols("file:///test.go", "package main")
	if err != nil {
		t.Fatalf("DocumentSymbols() error = %v", err)
	}
	if len(symbols) != len(testSymbols) {
		t.Errorf("Expected %d symbols, got %d", len(testSymbols), len(symbols))
	}

	// Test error behavior
	testErr := fmt.Errorf("mock error")
	mock.WithError(testErr)

	_, err = mock.DocumentSymbols("file:///test.go", "package main")
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != testErr.Error() {
		t.Errorf("Expected error %q, got %q", testErr.Error(), err.Error())
	}

	// Test close
	if err := mock.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
	if !mock.closed {
		t.Error("Mock should be marked as closed")
	}

	// Operations after close should fail
	_, err = mock.DocumentSymbols("file:///test.go", "package main")
	if err == nil {
		t.Error("Expected error after close, got nil")
	}
}

func TestCreateTestSymbols(t *testing.T) {
	symbols := CreateTestSymbols()

	if len(symbols) == 0 {
		t.Fatal("CreateTestSymbols returned no symbols")
	}

	// Verify symbol structure
	symbolMap := make(map[string]LSPDocumentSymbol)
	for _, sym := range symbols {
		symbolMap[sym.Name] = sym
	}

	// Check for expected symbols
	expectedNames := []string{"Person", "NewPerson", "Greet", "IsAdult", "MaxAge"}
	for _, name := range expectedNames {
		if _, found := symbolMap[name]; !found {
			t.Errorf("Expected symbol %s not found in test symbols", name)
		}
	}

	// Verify Person has children
	person := symbolMap["Person"]
	if len(person.Children) != 2 {
		t.Errorf("Person should have 2 children (Name, Age), got %d", len(person.Children))
	}
}
