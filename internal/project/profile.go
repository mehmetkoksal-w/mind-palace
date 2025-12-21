package project

import (
	"os"
	"path/filepath"
	"time"

	"github.com/koksalmehmet/mind-palace/internal/config"
	"github.com/koksalmehmet/mind-palace/internal/model"
)

func BuildProfile(root string) model.ProjectProfile {
	languages := detectLanguages(root)
	guardrails := config.LoadGuardrails(root)
	now := time.Now().UTC().Format(time.RFC3339)

	capabilities := map[string]model.Capability{
		"search.text": {
			Command:     "rg --no-heading --line-number --color never \"{{query}}\" {{paths}}",
			Description: "Text search via ripgrep",
		},
		"read.file": {
			Command:     "cat {{path}}",
			Description: "Read a single file",
		},
		"graph.deps": {
			Command:     defaultGraphCommand(languages),
			Description: "List project dependencies",
		},
		"tests.run": {
			Command:     defaultTestCommand(languages),
			Description: "Run project tests",
		},
		"lint.run": {
			Command:     defaultLintCommand(languages),
			Description: "Run project lint",
		},
		"symbols.lookup": {
			Command:     "echo symbols lookup not configured",
			Description: "Symbol lookup is not configured",
		},
	}

	return model.ProjectProfile{
		SchemaVersion: "1.0.0",
		Kind:          "palace/project-profile",
		ProjectRoot:   ".",
		Languages:     languages,
		Capabilities:  capabilities,
		Guardrails:    guardrails,
		Provenance: map[string]string{
			"createdBy": "palace detect",
			"createdAt": now,
		},
	}
}

func detectLanguages(root string) []string {
	var langs []string

	// Go
	if fileExists(filepath.Join(root, "go.mod")) {
		langs = append(langs, "go")
	}

	// JavaScript/TypeScript (Node.js)
	if fileExists(filepath.Join(root, "package.json")) {
		langs = append(langs, "javascript")
	}

	// Dart/Flutter
	if fileExists(filepath.Join(root, "pubspec.yaml")) {
		langs = append(langs, "dart")
	}

	// Rust
	if fileExists(filepath.Join(root, "Cargo.toml")) {
		langs = append(langs, "rust")
	}

	// Python
	if fileExists(filepath.Join(root, "pyproject.toml")) ||
		fileExists(filepath.Join(root, "setup.py")) ||
		fileExists(filepath.Join(root, "requirements.txt")) {
		langs = append(langs, "python")
	}

	// Ruby
	if fileExists(filepath.Join(root, "Gemfile")) {
		langs = append(langs, "ruby")
	}

	// Java/Kotlin (Maven/Gradle)
	if fileExists(filepath.Join(root, "pom.xml")) ||
		fileExists(filepath.Join(root, "build.gradle")) ||
		fileExists(filepath.Join(root, "build.gradle.kts")) {
		langs = append(langs, "java")
	}

	// C#/.NET
	if hasFileWithExtension(root, ".csproj") || hasFileWithExtension(root, ".sln") {
		langs = append(langs, "csharp")
	}

	// Swift
	if fileExists(filepath.Join(root, "Package.swift")) {
		langs = append(langs, "swift")
	}

	// PHP (Composer)
	if fileExists(filepath.Join(root, "composer.json")) {
		langs = append(langs, "php")
	}

	if len(langs) == 0 {
		langs = append(langs, "unknown")
	}
	return langs
}

// hasFileWithExtension checks if any file with the given extension exists in the root directory
func hasFileWithExtension(root string, ext string) bool {
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ext {
			return true
		}
	}
	return false
}

func defaultGraphCommand(langs []string) string {
	for _, l := range langs {
		switch l {
		case "go":
			return "go list -deps ./..."
		case "javascript":
			return "npm ls"
		case "dart":
			return "dart pub deps"
		case "rust":
			return "cargo tree"
		case "python":
			return "pip list"
		case "ruby":
			return "bundle list"
		case "java":
			return "gradle dependencies || mvn dependency:tree"
		case "csharp":
			return "dotnet list package"
		case "swift":
			return "swift package show-dependencies"
		case "php":
			return "composer show"
		}
	}
	return "echo graph deps not configured"
}

func defaultTestCommand(langs []string) string {
	for _, l := range langs {
		switch l {
		case "go":
			return "go test ./..."
		case "javascript":
			return "npm test"
		case "dart":
			return "dart test"
		case "rust":
			return "cargo test"
		case "python":
			return "pytest"
		case "ruby":
			return "bundle exec rspec"
		case "java":
			return "gradle test || mvn test"
		case "csharp":
			return "dotnet test"
		case "swift":
			return "swift test"
		case "php":
			return "composer test || ./vendor/bin/phpunit"
		}
	}
	return "echo tests not configured"
}

func defaultLintCommand(langs []string) string {
	for _, l := range langs {
		switch l {
		case "go":
			return "go vet ./..."
		case "javascript":
			return "npm run lint"
		case "dart":
			return "dart analyze"
		case "rust":
			return "cargo clippy"
		case "python":
			return "ruff check . || flake8"
		case "ruby":
			return "bundle exec rubocop"
		case "java":
			return "gradle check || mvn verify"
		case "csharp":
			return "dotnet format --verify-no-changes"
		case "swift":
			return "swiftlint"
		case "php":
			return "composer lint || ./vendor/bin/phpstan analyse"
		}
	}
	return "echo lint not configured"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
