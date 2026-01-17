package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/jsonc"
	"github.com/koksalmehmet/mind-palace/apps/cli/schemas"
	"github.com/koksalmehmet/mind-palace/apps/cli/starter"
)

type Guardrails struct {
	DoNotTouchGlobs []string `json:"doNotTouchGlobs,omitempty"`
	ReadOnlyGlobs   []string `json:"readOnlyGlobs,omitempty"`
}

type NeighborConfig struct {
	URL       string      `json:"url,omitempty"`
	LocalPath string      `json:"localPath,omitempty"`
	Auth      *AuthConfig `json:"auth,omitempty"`
	TTL       string      `json:"ttl,omitempty"`
	Enabled   *bool       `json:"enabled,omitempty"`
}

type AuthConfig struct {
	Type   string `json:"type"`
	Token  string `json:"token,omitempty"`
	User   string `json:"user,omitempty"`
	Pass   string `json:"pass,omitempty"`
	Header string `json:"header,omitempty"`
	Value  string `json:"value,omitempty"`
}

type DashboardConfig struct {
	CORS *CORSConfig `json:"cors,omitempty"`
}

type CORSConfig struct {
	AllowedOrigins []string `json:"allowedOrigins,omitempty"`
}

type PalaceConfig struct {
	SchemaVersion string `json:"schemaVersion"`
	Kind          string `json:"kind"`
	Project       struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Language    string `json:"language"`
		Repository  string `json:"repository"`
	} `json:"project"`
	DefaultRoom string                    `json:"defaultRoom"`
	Guardrails  Guardrails                `json:"guardrails"`
	Neighbors   map[string]NeighborConfig `json:"neighbors,omitempty"`
	Provenance  any                       `json:"provenance"`
	Dashboard   *DashboardConfig          `json:"dashboard,omitempty"`

	// Embedding configuration for semantic search
	EmbeddingBackend string `json:"embeddingBackend,omitempty"` // "ollama", "openai", or "disabled"
	EmbeddingModel   string `json:"embeddingModel,omitempty"`   // e.g., "nomic-embed-text", "text-embedding-3-small"
	EmbeddingURL     string `json:"embeddingUrl,omitempty"`     // Base URL for Ollama API
	EmbeddingAPIKey  string `json:"embeddingApiKey,omitempty"`  // API key for OpenAI

	// LLM configuration for text generation (extraction, contradiction detection)
	LLMBackend string `json:"llmBackend,omitempty"` // "ollama", "openai", "anthropic", or "disabled"
	LLMModel   string `json:"llmModel,omitempty"`   // e.g., "llama3.2", "gpt-4o-mini", "claude-3-haiku-20240307"
	LLMURL     string `json:"llmUrl,omitempty"`     // Base URL for Ollama API
	LLMAPIKey  string `json:"llmApiKey,omitempty"`  // API key for cloud providers

	// Auto-extraction configuration
	AutoExtract bool `json:"autoExtract,omitempty"` // Enable auto-extraction on session end

	// Contradiction detection configuration
	ContradictionAutoLink      bool    `json:"contradictionAutoLink,omitempty"`      // Auto-create contradiction links
	ContradictionMinConfidence float64 `json:"contradictionMinConfidence,omitempty"` // Minimum confidence for auto-linking (default 0.8)
	ContradictionAutoCheck     bool    `json:"contradictionAutoCheck,omitempty"`     // Auto-check for contradictions on store

	// Confidence decay configuration
	ConfidenceDecay *DecayConfig `json:"confidenceDecay,omitempty"`

	// Auto-injection configuration for AI agents
	AutoInjection *AutoInjectionConfig `json:"autoInjection,omitempty"`

	// Scope configuration for inheritance rules
	Scope *ScopeConfig `json:"scope,omitempty"`
}

// DecayConfig holds configuration for confidence decay of learnings.
type DecayConfig struct {
	Enabled       bool    `json:"enabled"`
	DecayDays     int     `json:"decayDays"`     // Days before decay starts (default: 30)
	DecayRate     float64 `json:"decayRate"`     // Decay per period (default: 0.05)
	DecayInterval int     `json:"decayInterval"` // Days between decay (default: 7)
	MinConfidence float64 `json:"minConfidence"` // Floor (default: 0.1)
}

// DefaultDecayConfig returns the default decay configuration.
func DefaultDecayConfig() *DecayConfig {
	return &DecayConfig{
		Enabled:       false,
		DecayDays:     30,
		DecayRate:     0.05,
		DecayInterval: 7,
		MinConfidence: 0.1,
	}
}

// AutoInjectionConfig holds configuration for automatic context injection.
type AutoInjectionConfig struct {
	Enabled          bool    `json:"enabled"`
	MaxTokens        int     `json:"maxTokens"`        // Maximum tokens for context (default: 2000)
	IncludeLearnings bool    `json:"includeLearnings"` // Include learnings in context
	IncludeDecisions bool    `json:"includeDecisions"` // Include decisions in context
	IncludeFailures  bool    `json:"includeFailures"`  // Include failure info in context
	MinConfidence    float64 `json:"minConfidence"`    // Minimum learning confidence (default: 0.5)
	PrioritizeRecent bool    `json:"prioritizeRecent"` // Prioritize recent over old
	ScopeInheritance bool    `json:"scopeInheritance"` // Include room+palace scope
}

// DefaultAutoInjectionConfig returns the default auto-injection configuration.
func DefaultAutoInjectionConfig() *AutoInjectionConfig {
	return &AutoInjectionConfig{
		Enabled:          true,
		MaxTokens:        2000,
		IncludeLearnings: true,
		IncludeDecisions: true,
		IncludeFailures:  true,
		MinConfidence:    0.5,
		PrioritizeRecent: true,
		ScopeInheritance: true,
	}
}

// ScopeConfig holds configuration for scope inheritance rules.
type ScopeConfig struct {
	InheritFromRoom     bool   `json:"inheritFromRoom"`     // Include room-level knowledge
	InheritFromPalace   bool   `json:"inheritFromPalace"`   // Include palace-level knowledge
	InheritFromCorridor bool   `json:"inheritFromCorridor"` // Include corridor knowledge (opt-in)
	RoomDetection       string `json:"roomDetection"`       // "first_dir", "manifest", or "custom"
}

// DefaultScopeConfig returns the default scope configuration.
func DefaultScopeConfig() *ScopeConfig {
	return &ScopeConfig{
		InheritFromRoom:     true,
		InheritFromPalace:   true,
		InheritFromCorridor: false,
		RoomDetection:       "first_dir",
	}
}

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
		filepath.Join(palaceDir, "cache"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return "", fmt.Errorf("create %s: %w", d, err)
		}
	}
	return palaceDir, nil
}

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
	if err := os.WriteFile(destPath, []byte(contents), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", destPath, err)
	}
	return nil
}

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
			// Version control & IDE
			".git/**",
			".palace/**",
			".idea/**",
			"**/.idea/**",
			".vscode/**",
			"**/.DS_Store",

			// Package managers & dependencies
			"node_modules/**",
			"vendor/**",
			"**/Pods/**",        // iOS CocoaPods
			"**/.symlinks/**",   // Flutter iOS symlinks
			"**/DerivedData/**", // Xcode derived data

			// Build outputs
			"dist/**",
			"build/**",
			"**/build/**", // Nested build directories
			"coverage/**",
			"target/**", // Rust/Maven
			"out/**",

			// Flutter/Dart specific
			".dart_tool/**",
			"**/.dart_tool/**",
			"**/*.dill", // Dart compiled files
			"**/*.dill.track.dill",
			"**/test_cache/**",

			// JavaScript/TypeScript
			".next/**",
			".turbo/**",
			".nx/**",
			".nuxt/**",
			".output/**",

			// Mobile
			".gradle/**",
			"**/.gradle/**",
			"**/android/.gradle/**",
			"**/android/build/**",
			"**/ios/build/**",
			"**/*.apk",
			"**/*.aab",
			"**/*.ipa",

			// Generated/minified files
			"**/*.min.*",
			"**/*.lock",
			"**/*.generated.*",
			"**/*.g.dart",       // Dart generated files
			"**/*.freezed.dart", // Freezed generated
			"**/*.gr.dart",      // Auto-route generated
			"**/*.mocks.dart",   // Mockito generated
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
			if bytes.Equal(existing, data) {
				continue // already canonical
			}
		}
		if err := os.WriteFile(dest, data, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}
	}
	return nil
}

func loadEmbeddedSchemas() (map[string][]byte, error) {
	return schemas.List()
}

func WriteJSON(path string, data any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
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
