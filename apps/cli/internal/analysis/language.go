package analysis

import (
	"path/filepath"
	"strings"
)

var extensionToLanguage = map[string]Language{
	// Go
	".go": LangGo,
	// JavaScript
	".js":  LangJavaScript,
	".mjs": LangJavaScript,
	".cjs": LangJavaScript,
	".jsx": LangJavaScript,
	// TypeScript
	".ts":  LangTypeScript,
	".tsx": LangTypeScript,
	".mts": LangTypeScript,
	".cts": LangTypeScript,
	// Python
	".py":  LangPython,
	".pyw": LangPython,
	".pyi": LangPython,
	// Rust
	".rs": LangRust,
	// Java
	".java": LangJava,
	// Dart
	".dart": LangDart,
	// C
	".c": LangC,
	".h": LangC,
	// C++
	".cpp": LangCPP,
	".cc":  LangCPP,
	".cxx": LangCPP,
	".hpp": LangCPP,
	".hxx": LangCPP,
	".hh":  LangCPP,
	// C#
	".cs": LangCSharp,
	// Ruby
	".rb":   LangRuby,
	".rake": LangRuby,
	// Swift
	".swift": LangSwift,
	// Kotlin
	".kt":  LangKotlin,
	".kts": LangKotlin,
	// Scala
	".scala": LangScala,
	".sc":    LangScala,
	// PHP
	".php":   LangPHP,
	".phtml": LangPHP,
	// Bash/Shell
	".sh":   LangBash,
	".bash": LangBash,
	".zsh":  LangBash,
	// SQL
	".sql": LangSQL,
	// HTML
	".html": LangHTML,
	".htm":  LangHTML,
	// CSS
	".css":  LangCSS,
	".scss": LangCSS,
	".less": LangCSS,
	// YAML
	".yaml": LangYAML,
	".yml":  LangYAML,
	// TOML
	".toml": LangTOML,
	// JSON
	".json":  LangJSON,
	".jsonc": LangJSON,
	// Markdown
	".md":       LangMarkdown,
	".markdown": LangMarkdown,
	// HCL (Terraform)
	".tf":     LangHCL,
	".tfvars": LangHCL,
	".hcl":    LangHCL,
	// Protobuf
	".proto": LangProtobuf,
	// Lua
	".lua": LangLua,
	// Elixir
	".ex":  LangElixir,
	".exs": LangElixir,
	// Groovy
	".groovy": LangGroovy,
	".gradle": LangGroovy,
	// Svelte
	".svelte": LangSvelte,
	// OCaml
	".ml":  LangOCaml,
	".mli": LangOCaml,
	// Elm
	".elm": LangElm,
	// CUE
	".cue": LangCUE,
}

// filenameToLanguage maps specific filenames (without extensions) to languages
var filenameToLanguage = map[string]Language{
	"Dockerfile":      LangDockerfile,
	"dockerfile":      LangDockerfile,
	"Makefile":        LangBash,
	"makefile":        LangBash,
	"GNUmakefile":     LangBash,
	"Jenkinsfile":     LangGroovy,
	"BUILD":           LangPython,
	"BUILD.bazel":     LangPython,
	"WORKSPACE":       LangPython,
	"WORKSPACE.bazel": LangPython,
}

// DetectLanguage returns the programming language of a file based on its extension or filename.
func DetectLanguage(filePath string) Language {
	// First check extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if lang, ok := extensionToLanguage[ext]; ok {
		return lang
	}

	// Check filename (for Dockerfile, Makefile, etc.)
	filename := filepath.Base(filePath)
	if lang, ok := filenameToLanguage[filename]; ok {
		return lang
	}

	// Check for Dockerfile.* pattern
	if strings.HasPrefix(filename, "Dockerfile.") || strings.HasPrefix(filename, "dockerfile.") {
		return LangDockerfile
	}

	return LangUnknown
}

// IsAnalyzable returns true if the file's language is supported for analysis.
func IsAnalyzable(filePath string) bool {
	return DetectLanguage(filePath) != LangUnknown
}

// SupportedExtensions returns a list of all file extensions supported by the system.
func SupportedExtensions() []string {
	exts := make([]string, 0, len(extensionToLanguage))
	for ext := range extensionToLanguage {
		exts = append(exts, ext)
	}
	return exts
}
