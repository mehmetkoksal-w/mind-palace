package analysis

import (
	"sort"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected Language
	}{
		// Go
		{"go file", "main.go", LangGo},
		{"go test file", "main_test.go", LangGo},
		{"go in subdir", "pkg/handler/router.go", LangGo},

		// JavaScript variants
		{"js file", "app.js", LangJavaScript},
		{"mjs module", "module.mjs", LangJavaScript},
		{"cjs module", "config.cjs", LangJavaScript},
		{"jsx file", "Component.jsx", LangJavaScript},

		// TypeScript variants
		{"ts file", "app.ts", LangTypeScript},
		{"tsx file", "Component.tsx", LangTypeScript},
		{"mts module", "module.mts", LangTypeScript},
		{"cts module", "config.cts", LangTypeScript},

		// Python variants
		{"py file", "script.py", LangPython},
		{"pyw file", "gui.pyw", LangPython},
		{"pyi stub", "types.pyi", LangPython},

		// Rust
		{"rust file", "main.rs", LangRust},

		// Java
		{"java file", "Main.java", LangJava},

		// Dart
		{"dart file", "main.dart", LangDart},

		// C
		{"c file", "main.c", LangC},
		{"c header", "header.h", LangC},

		// C++
		{"cpp file", "main.cpp", LangCPP},
		{"cc file", "main.cc", LangCPP},
		{"cxx file", "main.cxx", LangCPP},
		{"hpp header", "header.hpp", LangCPP},
		{"hxx header", "header.hxx", LangCPP},
		{"hh header", "header.hh", LangCPP},

		// C#
		{"csharp file", "Program.cs", LangCSharp},

		// Ruby
		{"ruby file", "app.rb", LangRuby},
		{"rake file", "Rakefile.rake", LangRuby},

		// Swift
		{"swift file", "main.swift", LangSwift},

		// Kotlin
		{"kotlin file", "Main.kt", LangKotlin},
		{"kotlin script", "build.gradle.kts", LangKotlin},

		// Scala
		{"scala file", "Main.scala", LangScala},
		{"sc file", "script.sc", LangScala},

		// PHP
		{"php file", "index.php", LangPHP},
		{"phtml file", "template.phtml", LangPHP},

		// Bash/Shell
		{"sh file", "script.sh", LangBash},
		{"bash file", "script.bash", LangBash},
		{"zsh file", "script.zsh", LangBash},

		// SQL
		{"sql file", "query.sql", LangSQL},

		// HTML
		{"html file", "index.html", LangHTML},
		{"htm file", "page.htm", LangHTML},

		// CSS
		{"css file", "style.css", LangCSS},
		{"scss file", "style.scss", LangCSS},
		{"less file", "style.less", LangCSS},

		// YAML
		{"yaml file", "config.yaml", LangYAML},
		{"yml file", "config.yml", LangYAML},

		// TOML
		{"toml file", "config.toml", LangTOML},

		// JSON
		{"json file", "data.json", LangJSON},
		{"jsonc file", "tsconfig.jsonc", LangJSON},

		// Markdown
		{"md file", "README.md", LangMarkdown},
		{"markdown file", "docs.markdown", LangMarkdown},

		// HCL/Terraform
		{"tf file", "main.tf", LangHCL},
		{"tfvars file", "vars.tfvars", LangHCL},
		{"hcl file", "config.hcl", LangHCL},

		// Protobuf
		{"proto file", "service.proto", LangProtobuf},

		// Lua
		{"lua file", "script.lua", LangLua},

		// Elixir
		{"ex file", "app.ex", LangElixir},
		{"exs file", "test.exs", LangElixir},

		// Groovy
		{"groovy file", "script.groovy", LangGroovy},
		{"gradle file", "build.gradle", LangGroovy},

		// Svelte
		{"svelte file", "Component.svelte", LangSvelte},

		// OCaml
		{"ml file", "main.ml", LangOCaml},
		{"mli file", "types.mli", LangOCaml},

		// Elm
		{"elm file", "Main.elm", LangElm},

		// CUE
		{"cue file", "schema.cue", LangCUE},

		// Special filenames
		{"Dockerfile", "Dockerfile", LangDockerfile},
		{"dockerfile lowercase", "dockerfile", LangDockerfile},
		{"Dockerfile.prod", "Dockerfile.prod", LangDockerfile},
		{"dockerfile.dev", "dockerfile.dev", LangDockerfile},
		{"Makefile", "Makefile", LangBash},
		{"makefile lowercase", "makefile", LangBash},
		{"GNUmakefile", "GNUmakefile", LangBash},
		{"Jenkinsfile", "Jenkinsfile", LangGroovy},
		{"BUILD", "BUILD", LangPython},
		{"BUILD.bazel", "BUILD.bazel", LangPython},
		{"WORKSPACE", "WORKSPACE", LangPython},
		{"WORKSPACE.bazel", "WORKSPACE.bazel", LangPython},

		// Unknown
		{"unknown extension", "file.xyz", LangUnknown},
		{"no extension", "LICENSE", LangUnknown},
		{"binary file", "program.exe", LangUnknown},

		// Case sensitivity for extensions
		{"uppercase GO", "main.GO", LangGo},
		{"mixed case Py", "script.Py", LangPython},

		// Paths with directories
		{"nested path go", "src/pkg/handler/router.go", LangGo},
		{"nested Dockerfile", "docker/Dockerfile.prod", LangDockerfile},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguage(tt.filePath)
			if got != tt.expected {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tt.filePath, got, tt.expected)
			}
		})
	}
}

func TestIsAnalyzable(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{"go file is analyzable", "main.go", true},
		{"ts file is analyzable", "app.ts", true},
		{"py file is analyzable", "script.py", true},
		{"Dockerfile is analyzable", "Dockerfile", true},
		{"unknown is not analyzable", "file.xyz", false},
		{"no extension not analyzable", "LICENSE", false},
		{"empty path not analyzable", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAnalyzable(tt.filePath)
			if got != tt.expected {
				t.Errorf("IsAnalyzable(%q) = %v, want %v", tt.filePath, got, tt.expected)
			}
		})
	}
}

func TestSupportedExtensions(t *testing.T) {
	exts := SupportedExtensions()

	// Should return non-empty list
	if len(exts) == 0 {
		t.Error("SupportedExtensions() returned empty list")
	}

	// Should contain expected extensions
	expectedExts := []string{".go", ".ts", ".py", ".rs", ".java", ".js"}
	extMap := make(map[string]bool)
	for _, ext := range exts {
		extMap[ext] = true
	}

	for _, expected := range expectedExts {
		if !extMap[expected] {
			t.Errorf("SupportedExtensions() missing %q", expected)
		}
	}

	// Should not contain duplicates
	seen := make(map[string]bool)
	for _, ext := range exts {
		if seen[ext] {
			t.Errorf("SupportedExtensions() contains duplicate %q", ext)
		}
		seen[ext] = true
	}

	// All extensions should start with a dot
	for _, ext := range exts {
		if ext == "" || ext[0] != '.' {
			t.Errorf("SupportedExtensions() contains invalid extension %q", ext)
		}
	}
}

func TestParserRegistry(t *testing.T) {
	t.Run("NewParserRegistry creates registry with default parsers", func(t *testing.T) {
		reg := NewParserRegistry()
		if reg == nil {
			t.Fatal("NewParserRegistry() returned nil")
		}

		// Should have Go parser
		parser, ok := reg.GetParser(LangGo)
		if !ok {
			t.Error("Registry missing Go parser")
		}
		if parser == nil {
			t.Error("Go parser is nil")
		}
		if parser.Language() != LangGo {
			t.Errorf("Go parser.Language() = %q, want %q", parser.Language(), LangGo)
		}
	})

	t.Run("GetParser returns false for unknown language", func(t *testing.T) {
		reg := NewParserRegistry()
		parser, ok := reg.GetParser(LangUnknown)
		if ok {
			t.Error("GetParser(LangUnknown) should return false")
		}
		if parser != nil {
			t.Error("GetParser(LangUnknown) should return nil parser")
		}
	})

	t.Run("Register adds custom parser", func(t *testing.T) {
		reg := NewParserRegistry()

		// Create a mock parser
		mock := &mockParser{lang: "testlang"}
		reg.Register(mock)

		parser, ok := reg.GetParser("testlang")
		if !ok {
			t.Error("Custom parser not found after Register")
		}
		if parser != mock {
			t.Error("GetParser returned different parser than registered")
		}
	})

	t.Run("Parse returns FileAnalysis for unknown language", func(t *testing.T) {
		reg := NewParserRegistry()
		result, err := reg.Parse([]byte("unknown content"), "file.xyz")
		if err != nil {
			t.Errorf("Parse returned error for unknown language: %v", err)
		}
		if result == nil {
			t.Fatal("Parse returned nil for unknown language")
		}
		if result.Path != "file.xyz" {
			t.Errorf("result.Path = %q, want %q", result.Path, "file.xyz")
		}
		if result.Language != string(LangUnknown) {
			t.Errorf("result.Language = %q, want %q", result.Language, LangUnknown)
		}
	})

	t.Run("Parse returns FileAnalysis for language without parser", func(t *testing.T) {
		// Create empty registry (no default parsers)
		reg := &ParserRegistry{parsers: make(map[Language]Parser)}

		// Add mapping but no parser
		result, err := reg.Parse([]byte("content"), "main.go")
		if err != nil {
			t.Errorf("Parse returned error: %v", err)
		}
		if result == nil {
			t.Fatal("Parse returned nil")
		}
		if result.Language != string(LangGo) {
			t.Errorf("result.Language = %q, want %q", result.Language, LangGo)
		}
	})

	t.Run("all registered languages have parsers", func(t *testing.T) {
		reg := NewParserRegistry()

		// Languages that should have parsers
		expectedLanguages := []Language{
			LangGo, LangJavaScript, LangTypeScript, LangPython, LangRust,
			LangJava, LangC, LangCPP, LangCSharp, LangRuby, LangPHP,
			LangKotlin, LangScala, LangSwift, LangBash, LangSQL,
			LangDockerfile, LangHCL, LangHTML, LangCSS, LangYAML,
			LangTOML, LangJSON, LangMarkdown, LangElixir, LangLua,
			LangGroovy, LangSvelte, LangOCaml, LangElm, LangProtobuf,
			LangDart, LangCUE,
		}

		for _, lang := range expectedLanguages {
			parser, ok := reg.GetParser(lang)
			if !ok {
				t.Errorf("Missing parser for %s", lang)
				continue
			}
			if parser.Language() != lang {
				t.Errorf("Parser for %s returns Language() = %s", lang, parser.Language())
			}
		}
	})
}

func TestAnalyze(t *testing.T) {
	t.Run("analyze Go code extracts symbols", func(t *testing.T) {
		code := `package main

import "fmt"

// Hello prints a greeting
func Hello(name string) string {
	return "Hello, " + name
}

type Greeter struct {
	Prefix string
}

func (g *Greeter) Greet(name string) string {
	return g.Prefix + name
}

var DefaultGreeting = "Hi"

const MaxLength = 100
`
		result, err := Analyze([]byte(code), "main.go")
		if err != nil {
			t.Fatalf("Analyze returned error: %v", err)
		}

		if result.Language != "go" {
			t.Errorf("Language = %q, want %q", result.Language, "go")
		}

		// Check we found the function
		foundHello := false
		foundGreeter := false
		foundGreet := false
		foundVar := false
		foundConst := false

		for _, sym := range result.Symbols {
			switch sym.Name {
			case "Hello":
				foundHello = true
				if sym.Kind != KindFunction {
					t.Errorf("Hello.Kind = %q, want %q", sym.Kind, KindFunction)
				}
				if !sym.Exported {
					t.Error("Hello should be exported")
				}
			case "Greeter":
				foundGreeter = true
				if sym.Kind != KindClass {
					t.Errorf("Greeter.Kind = %q, want %q", sym.Kind, KindClass)
				}
			case "Greet":
				foundGreet = true
				if sym.Kind != KindMethod {
					t.Errorf("Greet.Kind = %q, want %q", sym.Kind, KindMethod)
				}
			case "DefaultGreeting":
				foundVar = true
				if sym.Kind != KindVariable {
					t.Errorf("DefaultGreeting.Kind = %q, want %q", sym.Kind, KindVariable)
				}
			case "MaxLength":
				foundConst = true
				if sym.Kind != KindConstant {
					t.Errorf("MaxLength.Kind = %q, want %q", sym.Kind, KindConstant)
				}
			}
		}

		if !foundHello {
			t.Error("Did not find Hello function")
		}
		if !foundGreeter {
			t.Error("Did not find Greeter struct")
		}
		if !foundGreet {
			t.Error("Did not find Greet method")
		}
		if !foundVar {
			t.Error("Did not find DefaultGreeting variable")
		}
		if !foundConst {
			t.Error("Did not find MaxLength constant")
		}

		// Check import relationship
		foundImport := false
		for _, rel := range result.Relationships {
			if rel.Kind == RelImport && rel.TargetFile == "fmt" {
				foundImport = true
				break
			}
		}
		if !foundImport {
			t.Error("Did not find fmt import relationship")
		}
	})

	t.Run("analyze unknown language returns empty analysis", func(t *testing.T) {
		result, err := Analyze([]byte("unknown content"), "file.xyz")
		if err != nil {
			t.Fatalf("Analyze returned error: %v", err)
		}

		if result.Language != string(LangUnknown) {
			t.Errorf("Language = %q, want %q", result.Language, LangUnknown)
		}
		if len(result.Symbols) != 0 {
			t.Errorf("Expected 0 symbols, got %d", len(result.Symbols))
		}
	})

	t.Run("analyze empty file", func(t *testing.T) {
		result, err := Analyze([]byte(""), "main.go")
		if err != nil {
			t.Fatalf("Analyze returned error: %v", err)
		}

		if result == nil {
			t.Fatal("Analyze returned nil for empty file")
		}
	})

	t.Run("analyze TypeScript code", func(t *testing.T) {
		code := `
interface User {
	id: number;
	name: string;
}

class UserService {
	private users: User[] = [];

	addUser(user: User): void {
		this.users.push(user);
	}

	getUser(id: number): User | undefined {
		return this.users.find(u => u.id === id);
	}
}

export function createUser(name: string): User {
	return { id: Date.now(), name };
}
`
		result, err := Analyze([]byte(code), "user.ts")
		if err != nil {
			t.Fatalf("Analyze returned error: %v", err)
		}

		if result.Language != "typescript" {
			t.Errorf("Language = %q, want %q", result.Language, "typescript")
		}

		// Should find interface, class, and function
		symbolNames := make(map[string]bool)
		for _, sym := range result.Symbols {
			symbolNames[sym.Name] = true
		}

		expected := []string{"User", "UserService", "createUser"}
		for _, name := range expected {
			if !symbolNames[name] {
				t.Errorf("Did not find symbol %q", name)
			}
		}
	})

	t.Run("analyze Python code", func(t *testing.T) {
		code := `
import os
from typing import List

class Calculator:
    """A simple calculator class"""

    def __init__(self):
        self.history: List[float] = []

    def add(self, a: float, b: float) -> float:
        result = a + b
        self.history.append(result)
        return result

def main():
    calc = Calculator()
    print(calc.add(1, 2))

if __name__ == "__main__":
    main()
`
		result, err := Analyze([]byte(code), "calc.py")
		if err != nil {
			t.Fatalf("Analyze returned error: %v", err)
		}

		if result.Language != "python" {
			t.Errorf("Language = %q, want %q", result.Language, "python")
		}

		// Should find class and functions
		symbolNames := make(map[string]bool)
		for _, sym := range result.Symbols {
			symbolNames[sym.Name] = true
		}

		if !symbolNames["Calculator"] {
			t.Error("Did not find Calculator class")
		}
		if !symbolNames["main"] {
			t.Error("Did not find main function")
		}
	})
}

func TestAnalyzeCallRelationships(t *testing.T) {
	code := `package main

import "fmt"

func greet(name string) {
	fmt.Println("Hello", name)
}

func main() {
	greet("World")
}
`
	result, err := Analyze([]byte(code), "main.go")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	// Check for call relationships
	var calls []string
	for _, rel := range result.Relationships {
		if rel.Kind == RelCall {
			calls = append(calls, rel.TargetSymbol)
		}
	}

	// Should find greet and fmt.Println calls
	sort.Strings(calls)
	if len(calls) < 2 {
		t.Errorf("Expected at least 2 call relationships, got %d: %v", len(calls), calls)
	}
}

func TestAnalyzeExportedSymbols(t *testing.T) {
	code := `package example

func PublicFunc() {}
func privateFunc() {}

type PublicType struct{}
type privateType struct{}

var PublicVar = 1
var privateVar = 2

const PublicConst = "pub"
const privateConst = "priv"
`
	result, err := Analyze([]byte(code), "example.go")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	exported := make(map[string]bool)
	for _, sym := range result.Symbols {
		exported[sym.Name] = sym.Exported
	}

	tests := []struct {
		name     string
		expected bool
	}{
		{"PublicFunc", true},
		{"privateFunc", false},
		{"PublicType", true},
		{"privateType", false},
		{"PublicVar", true},
		{"privateVar", false},
		{"PublicConst", true},
		{"privateConst", false},
	}

	for _, tt := range tests {
		got, found := exported[tt.name]
		if !found {
			t.Errorf("Symbol %q not found", tt.name)
			continue
		}
		if got != tt.expected {
			t.Errorf("Symbol %q exported = %v, want %v", tt.name, got, tt.expected)
		}
	}
}

// mockParser is a simple mock for testing custom parser registration
type mockParser struct {
	lang Language
}

func (m *mockParser) Parse(content []byte, filePath string) (*FileAnalysis, error) {
	return &FileAnalysis{
		Path:     filePath,
		Language: string(m.lang),
	}, nil
}

func (m *mockParser) Language() Language {
	return m.lang
}
