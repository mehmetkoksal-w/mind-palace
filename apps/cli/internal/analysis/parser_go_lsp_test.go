//go:build integration

package analysis

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// These are integration tests that require gopls to be installed
// Run with: go test -tags=integration ./internal/analysis
// Note: testGoCode is defined in test_data.go and shared with unit tests

func TestGoLSPParser_IsAvailable(t *testing.T) {
	parser := NewGoLSPParser("")

	// Check if gopls is in PATH
	_, err := exec.LookPath("gopls")
	expected := err == nil

	if parser.IsAvailable() != expected {
		t.Errorf("IsAvailable() = %v, want %v", parser.IsAvailable(), expected)
	}

	if parser.IsAvailable() {
		t.Log("gopls is available")
	} else {
		t.Skip("gopls not available, skipping LSP tests")
	}
}

func TestGoLSPParser_Language(t *testing.T) {
	parser := NewGoLSPParser("")
	if parser.Language() != LangGo {
		t.Errorf("Language() = %v, want %v", parser.Language(), LangGo)
	}
}

func TestGoLSPParser_Parse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	parser := NewGoLSPParser("")

	if !parser.IsAvailable() {
		t.Skip("gopls not available")
	}

	// Create a temporary directory and file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	if err := os.WriteFile(testFile, []byte(testGoCode), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Parse the file
	analysis, err := parser.Parse([]byte(testGoCode), testFile)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify basic structure
	if analysis.Path != testFile {
		t.Errorf("Path = %v, want %v", analysis.Path, testFile)
	}
	if analysis.Language != string(LangGo) {
		t.Errorf("Language = %v, want %v", analysis.Language, string(LangGo))
	}

	// Verify we found symbols
	if len(analysis.Symbols) == 0 {
		t.Error("No symbols found")
	}

	t.Logf("Found %d symbols", len(analysis.Symbols))
	for _, sym := range analysis.Symbols {
		t.Logf("  %s: %s (line %d-%d)", sym.Kind, sym.Name, sym.LineStart, sym.LineEnd)
	}

	// Verify specific symbols exist
	symbolNames := make(map[string]SymbolKind)
	for _, sym := range analysis.Symbols {
		symbolNames[sym.Name] = sym.Kind
	}

	// Check for expected symbols
	expectedSymbols := map[string]SymbolKind{
		"Person":        KindClass,
		"NewPerson":     KindFunction,
		"main":          KindFunction,
		"MaxAge":        KindConstant,
		"defaultPerson": KindVariable,
	}

	for name, expectedKind := range expectedSymbols {
		kind, found := symbolNames[name]
		if !found {
			t.Errorf("Expected symbol %s not found", name)
		} else if kind != expectedKind {
			t.Errorf("Symbol %s: got kind %s, want %s", name, kind, expectedKind)
		}
	}

	// Check for methods on Person struct
	var personStruct *Symbol
	for i := range analysis.Symbols {
		if analysis.Symbols[i].Name == "Person" {
			personStruct = &analysis.Symbols[i]
			break
		}
	}

	if personStruct != nil {
		t.Logf("Person struct has %d children", len(personStruct.Children))

		// gopls may include methods or fields as children
		// The exact structure depends on gopls version, so we're lenient here
		if len(personStruct.Children) > 0 {
			for _, child := range personStruct.Children {
				t.Logf("  Person.%s (%s)", child.Name, child.Kind)
			}
		}
	}

	// Verify relationships (imports)
	if len(analysis.Relationships) == 0 {
		t.Error("No relationships found")
	}

	importCount := 0
	callCount := 0
	for _, rel := range analysis.Relationships {
		if rel.Kind == RelImport {
			importCount++
			t.Logf("Import: %s (line %d)", rel.TargetFile, rel.Line)
		} else if rel.Kind == RelCall {
			callCount++
		}
	}

	if importCount == 0 {
		t.Error("No imports found")
	}

	t.Logf("Found %d imports, %d calls", importCount, callCount)
}

func TestGoLSPParser_ParseWithErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	parser := NewGoLSPParser("")

	if !parser.IsAvailable() {
		t.Skip("gopls not available")
	}

	invalidCode := `package main

func broken( {
	// Invalid syntax
}
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "broken.go")

	if err := os.WriteFile(testFile, []byte(invalidCode), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Parse should still work, but may have limited symbols
	analysis, err := parser.Parse([]byte(invalidCode), testFile)

	// gopls should handle errors gracefully
	if err != nil {
		t.Logf("Parse returned error (expected for invalid code): %v", err)
	}

	if analysis != nil {
		t.Logf("Analysis returned %d symbols despite errors", len(analysis.Symbols))
	}
}

func TestGoLSPParser_ExportedSymbols(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	parser := NewGoLSPParser("")

	if !parser.IsAvailable() {
		t.Skip("gopls not available")
	}

	code := `package test

type PublicStruct struct {
	PublicField  string
	privateField int
}

func PublicFunc() {}
func privateFunc() {}

const PublicConst = 1
const privateConst = 2

var PublicVar = "public"
var privateVar = "private"
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "exports.go")

	if err := os.WriteFile(testFile, []byte(code), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	analysis, err := parser.Parse([]byte(code), testFile)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check exported flags
	for _, sym := range analysis.Symbols {
		expectedExported := len(sym.Name) > 0 && sym.Name[0] >= 'A' && sym.Name[0] <= 'Z'
		if sym.Exported != expectedExported {
			t.Errorf("Symbol %s: Exported = %v, want %v", sym.Name, sym.Exported, expectedExported)
		}
	}
}

func TestGoLSPParser_ComplexStructures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	parser := NewGoLSPParser("")

	if !parser.IsAvailable() {
		t.Skip("gopls not available")
	}

	code := `package test

type Shape interface {
	Area() float64
	Perimeter() float64
}

type Rectangle struct {
	Width  float64
	Height float64
}

func (r Rectangle) Area() float64 {
	return r.Width * r.Height
}

func (r *Rectangle) Scale(factor float64) {
	r.Width *= factor
	r.Height *= factor
}

type Circle struct {
	Radius float64
}

func (c Circle) Area() float64 {
	return 3.14159 * c.Radius * c.Radius
}
`

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "shapes.go")

	if err := os.WriteFile(testFile, []byte(code), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	analysis, err := parser.Parse([]byte(code), testFile)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Find interface
	hasInterface := false
	hasStruct := false
	hasMethod := false

	for _, sym := range analysis.Symbols {
		if sym.Name == "Shape" && sym.Kind == KindInterface {
			hasInterface = true
			t.Logf("Found interface: %s", sym.Name)
		}
		if (sym.Name == "Rectangle" || sym.Name == "Circle") && sym.Kind == KindClass {
			hasStruct = true
			t.Logf("Found struct: %s with %d children", sym.Name, len(sym.Children))
		}
		if sym.Name == "Area" && sym.Kind == KindMethod {
			hasMethod = true
			t.Logf("Found method: %s", sym.Name)
		}
	}

	if !hasInterface {
		t.Error("Interface not found or not recognized")
	}
	if !hasStruct {
		t.Error("Structs not found")
	}
	if !hasMethod {
		t.Error("Methods not found")
	}
}

func BenchmarkGoLSPParser_Parse(b *testing.B) {
	parser := NewGoLSPParser("")

	if !parser.IsAvailable() {
		b.Skip("gopls not available")
	}

	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "bench.go")

	if err := os.WriteFile(testFile, []byte(testGoCode), 0o644); err != nil {
		b.Fatalf("Failed to write test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse([]byte(testGoCode), testFile)
		if err != nil {
			b.Fatalf("Parse() error = %v", err)
		}
	}
}

// TestGoLSPParser_CompareWithTreeSitter compares LSP and tree-sitter results
func TestGoLSPParser_CompareWithTreeSitter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	lspParser := NewGoLSPParser("")
	tsParser := NewGoParser()

	if !lspParser.IsAvailable() {
		t.Skip("gopls not available")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "compare.go")

	if err := os.WriteFile(testFile, []byte(testGoCode), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Parse with both
	lspResult, err := lspParser.Parse([]byte(testGoCode), testFile)
	if err != nil {
		t.Fatalf("LSP Parse() error = %v", err)
	}

	tsResult, err := tsParser.Parse([]byte(testGoCode), testFile)
	if err != nil {
		t.Fatalf("Tree-sitter Parse() error = %v", err)
	}

	t.Logf("LSP found %d symbols", len(lspResult.Symbols))
	t.Logf("Tree-sitter found %d symbols", len(tsResult.Symbols))

	// Create maps for comparison
	lspSymbols := make(map[string]SymbolKind)
	tsSymbols := make(map[string]SymbolKind)

	for _, sym := range lspResult.Symbols {
		lspSymbols[sym.Name] = sym.Kind
	}
	for _, sym := range tsResult.Symbols {
		tsSymbols[sym.Name] = sym.Kind
	}

	// Compare common symbols
	for name, lspKind := range lspSymbols {
		if tsKind, found := tsSymbols[name]; found {
			if lspKind != tsKind {
				t.Logf("Symbol %s: LSP=%s, TreeSitter=%s", name, lspKind, tsKind)
			}
		} else {
			t.Logf("Symbol %s: found by LSP but not Tree-sitter", name)
		}
	}

	for name := range tsSymbols {
		if _, found := lspSymbols[name]; !found {
			t.Logf("Symbol %s: found by Tree-sitter but not LSP", name)
		}
	}

	// Both should find at least some common symbols
	if len(lspSymbols) == 0 && len(tsSymbols) == 0 {
		t.Error("Neither parser found any symbols")
	}
}
