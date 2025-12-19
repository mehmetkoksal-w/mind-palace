package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mind-palace/internal/jsonc"
	"mind-palace/schemas"
	"mind-palace/starter"
)

// Guardrails defines file patterns the CLI should avoid touching.
type Guardrails struct {
	DoNotTouchGlobs []string `json:"doNotTouchGlobs,omitempty"`
	ReadOnlyGlobs   []string `json:"readOnlyGlobs,omitempty"`
}

// PalaceConfig mirrors the curated palace.jsonc structure.
type PalaceConfig struct {
	SchemaVersion string `json:"schemaVersion"`
	Kind          string `json:"kind"`
	Project       struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Language    string `json:"language"`
		Repository  string `json:"repository"`
	} `json:"project"`
	DefaultRoom string     `json:"defaultRoom"`
	Guardrails  Guardrails `json:"guardrails"`
	Provenance  any        `json:"provenance"`
}

// EnsureLayout creates the .palace directory hierarchy.
func EnsureLayout(root string) (string, error) {
	palaceDir := filepath.Join(root, ".palace")
	dirs := []string{
		palaceDir,
		filepath.Join(palaceDir, "rooms"),
		filepath.Join(palaceDir, "playbooks"),
		filepath.Join(palaceDir, "outputs"),
		filepath.Join(palaceDir, "schemas"),
		filepath.Join(palaceDir, "maps"),
		filepath.Join(palaceDir, "index"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return "", fmt.Errorf("create %s: %w", d, err)
		}
	}
	return palaceDir, nil
}

// WriteTemplate writes a starter template into the palace directory.
func WriteTemplate(destPath, templateName string, replacements map[string]string, allowOverwrite bool) error {
	if _, err := os.Stat(destPath); err == nil && !allowOverwrite {
		return nil
	}
	tpl, err := starter.Get(templateName)
	if err != nil {
		return fmt.Errorf("load template %s: %w", templateName, err)
	}
	if replacements == nil {
		replacements = map[string]string{}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	replacements["createdAt"] = replaceZero(replacements["createdAt"], now)
	contents := starter.Apply(tpl, replacements)
	if err := os.WriteFile(destPath, []byte(contents), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", destPath, err)
	}
	return nil
}

// LoadPalaceConfig parses the curated palace.jsonc file if present.
func LoadPalaceConfig(root string) (*PalaceConfig, error) {
	path := filepath.Join(root, ".palace", "palace.jsonc")
	var cfg PalaceConfig
	if err := jsonc.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// LoadGuardrails returns guardrails from palace.jsonc if available; otherwise defaults.
func LoadGuardrails(root string) Guardrails {
	cfg, err := LoadPalaceConfig(root)
	if err != nil {
		return defaultGuardrails()
	}
	def := defaultGuardrails()
	return Guardrails{
		DoNotTouchGlobs: mergeGlobs(def.DoNotTouchGlobs, cfg.Guardrails.DoNotTouchGlobs),
		ReadOnlyGlobs:   mergeGlobs(def.ReadOnlyGlobs, cfg.Guardrails.ReadOnlyGlobs),
	}
}

func defaultGuardrails() Guardrails {
	return Guardrails{
		DoNotTouchGlobs: []string{
			".git/**",
			".palace/**",
			"node_modules/**",
			"vendor/**",
			"dist/**",
			"build/**",
			"coverage/**",
			"target/**",
			".dart_tool/**",
			".next/**",
			".turbo/**",
			".nx/**",
			".gradle/**",
			".idea/**",
			".vscode/**",
			"**/*.min.*",
			"**/*.lock",
			"**/*.generated.*",
			"**/*.g.*",
		},
	}
}

func mergeGlobs(defaults, user []string) []string {
	seen := make(map[string]struct{})
	var merged []string
	appendIfMissing := func(globs []string) {
		for _, g := range globs {
			norm := normalizeGlob(g)
			if norm == "" {
				continue
			}
			if _, ok := seen[norm]; ok {
				continue
			}
			seen[norm] = struct{}{}
			merged = append(merged, norm)
		}
	}
	appendIfMissing(defaults)
	appendIfMissing(user)
	return merged
}

func normalizeGlob(g string) string {
	trimmed := strings.TrimSpace(g)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	for strings.Contains(trimmed, "//") {
		trimmed = strings.ReplaceAll(trimmed, "//", "/")
	}
	return filepath.ToSlash(trimmed)
}

// CopySchemas exports embedded schema files into the workspace at .palace/schemas for transparency.
// The embedded schemas under /schemas remain the canonical source for validation.
func CopySchemas(root string, allowOverwrite bool) error {
	_ = allowOverwrite // schemas are always refreshed to match embedded versions
	schemaDir := filepath.Join(root, ".palace", "schemas")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		return fmt.Errorf("ensure schema dir: %w", err)
	}

	schemaMap, err := loadEmbeddedSchemas()
	if err != nil {
		return err
	}
	for name, data := range schemaMap {
		dest := filepath.Join(schemaDir, fmt.Sprintf("%s.schema.json", name))
		if existing, err := os.ReadFile(dest); err == nil && len(existing) > 0 {
			if string(existing) == string(data) {
				continue // already canonical
			}
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}
	}
	return nil
}

func loadEmbeddedSchemas() (map[string][]byte, error) {
	return schemas.List()
}

// WriteJSON writes JSON (not JSONC) with indentation.
func WriteJSON(path string, data any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func replaceZero(current, fallback string) string {
	if strings.TrimSpace(current) == "" {
		return fallback
	}
	return current
}
