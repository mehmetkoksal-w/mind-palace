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
	if fileExists(filepath.Join(root, "go.mod")) {
		langs = append(langs, "go")
	}
	if fileExists(filepath.Join(root, "package.json")) {
		langs = append(langs, "javascript")
	}
	if len(langs) == 0 {
		langs = append(langs, "unknown")
	}
	return langs
}

func defaultGraphCommand(langs []string) string {
	for _, l := range langs {
		switch l {
		case "go":
			return "go list -deps ./..."
		case "javascript":
			return "npm ls"
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
		}
	}
	return "echo lint not configured"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
