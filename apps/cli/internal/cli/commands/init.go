package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/cli/flags"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/config"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/project"
)

func init() {
	Register(&Command{
		Name:        "init",
		Aliases:     []string{"build", "enter"},
		Description: "Initialize a new Mind Palace in the current directory",
		Run:         RunInit,
	})
}

// InitOptions contains the configuration for the init command.
type InitOptions struct {
	Root        string
	Force       bool
	WithOutputs bool
	SkipDetect  bool
	NoScan      bool   // Skip automatic scan after init
	WithAgents  string // Comma-separated agent names or "auto" for auto-detect
	NoGitignore bool   // Skip .gitignore updates
	NoHooks     bool   // Skip git hooks installation
	NoVSCode    bool   // Skip VS Code integration
}

// DetectedAgent represents an auto-detected AI tool in the environment
type DetectedAgent struct {
	Name       string
	Key        string // The key used in supportedTools map
	Confidence string // "high", "medium", "low"
	Indicator  string // What we detected
}

// RunInit initializes a new Mind Palace in the specified directory.
func RunInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	root := flags.AddRootFlag(fs)
	force := flags.AddForceFlag(fs)
	withOutputs := fs.Bool("with-outputs", false, "also create generated outputs (context-pack)")
	skipDetect := fs.Bool("skip-detect", false, "skip auto-detection of project type")
	noScan := fs.Bool("no-scan", false, "skip automatic scan after init (not recommended)")
	withAgents := fs.String("with-agents", "", "install agent configs: 'auto' for auto-detect, or comma-separated list (claude-code,cursor,vscode)")
	noGitignore := fs.Bool("no-gitignore", false, "skip .gitignore updates")
	noHooks := fs.Bool("no-hooks", false, "skip git hooks installation")
	noVSCode := fs.Bool("no-vscode", false, "skip VS Code integration (extensions.json)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	opts := InitOptions{
		Root:        *root,
		Force:       *force,
		WithOutputs: *withOutputs,
		SkipDetect:  *skipDetect,
		NoScan:      *noScan,
		WithAgents:  *withAgents,
		NoGitignore: *noGitignore,
		NoHooks:     *noHooks,
		NoVSCode:    *noVSCode,
	}

	return ExecuteInit(opts)
}

// ExecuteInit performs the initialization with the given options.
// This is separated for easier testing.
func ExecuteInit(opts InitOptions) error {
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return err
	}
	if _, err := config.EnsureLayout(rootPath); err != nil {
		return err
	}
	if err := config.CopySchemas(rootPath, opts.Force); err != nil {
		return err
	}

	// Auto-detect project type by default
	language := "unknown"
	var monorepoInfo *project.MonorepoInfo
	if !opts.SkipDetect {
		profile := project.BuildProfile(rootPath)
		profilePath := filepath.Join(rootPath, ".palace", "project-profile.json")
		if err := config.WriteJSON(profilePath, profile); err != nil {
			return err
		}
		if len(profile.Languages) > 0 {
			language = profile.Languages[0]
		}
		fmt.Printf("detected project type: %s\n", language)

		// Detect monorepo structure using the new comprehensive detection
		monorepoInfo = project.DetectMonorepo(rootPath, nil)
		if monorepoInfo.IsMonorepo {
			managerStr := ""
			if monorepoInfo.Manager != "" {
				managerStr = fmt.Sprintf(" (%s)", monorepoInfo.Manager)
			}
			fmt.Printf("detected monorepo structure%s with %d subprojects\n", managerStr, len(monorepoInfo.Projects))

			// Write monorepo info to file for reference
			monorepoPath := filepath.Join(rootPath, ".palace", "monorepo.json")
			if err := config.WriteJSON(monorepoPath, monorepoInfo); err != nil {
				fmt.Printf("warning: could not write monorepo.json: %v\n", err)
			}
		}
	}

	replacements := map[string]string{
		"projectName": filepath.Base(rootPath),
		"language":    language,
	}
	if err := config.WriteTemplate(filepath.Join(rootPath, ".palace", "palace.jsonc"), "palace.jsonc", replacements, opts.Force); err != nil {
		return err
	}
	if err := config.WriteTemplate(filepath.Join(rootPath, ".palace", "rooms", "project-overview.jsonc"), "rooms/project-overview.jsonc", map[string]string{}, opts.Force); err != nil {
		return err
	}

	// Write auto-detected monorepo rooms
	if monorepoInfo != nil && monorepoInfo.IsMonorepo {
		for _, proj := range monorepoInfo.Projects {
			room := MonorepoRoom{
				Name:        proj.Name,
				RelPath:     proj.Path,
				Category:    proj.Category,
				EntryPoints: proj.EntryPoints,
				DependsOn:   proj.DependsOn,
			}
			roomPath := filepath.Join(rootPath, ".palace", "rooms", room.Name+".jsonc")
			if _, err := os.Stat(roomPath); err == nil && !opts.Force {
				continue // Don't overwrite existing rooms
			}
			if err := writeMonorepoRoom(roomPath, room); err != nil {
				fmt.Printf("warning: could not create room %s: %v\n", room.Name, err)
			} else {
				depsInfo := ""
				if len(room.DependsOn) > 0 {
					depsInfo = fmt.Sprintf(" (depends on: %s)", strings.Join(room.DependsOn, ", "))
				}
				fmt.Printf("  created room: %s (%s)%s\n", room.Name, room.RelPath, depsInfo)
			}
		}
	}

	if err := config.WriteTemplate(filepath.Join(rootPath, ".palace", "playbooks", "default.jsonc"), "playbooks/default.jsonc", map[string]string{}, opts.Force); err != nil {
		return err
	}
	// Only write default project-profile template if skipping detection (detection writes its own profile)
	if opts.SkipDetect {
		if err := config.WriteTemplate(filepath.Join(rootPath, ".palace", "project-profile.json"), "project-profile.json", map[string]string{}, opts.Force); err != nil {
			return err
		}
	}

	if opts.WithOutputs {
		cpPath := filepath.Join(rootPath, ".palace", "outputs", "context-pack.json")
		if _, err := os.Stat(cpPath); os.IsNotExist(err) || opts.Force {
			cpReplacements := map[string]string{
				"goal":      "unspecified",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			if err := config.WriteTemplate(cpPath, "outputs/context-pack.json", cpReplacements, opts.Force); err != nil {
				return err
			}
		}
	}

	fmt.Printf("initialized palace in %s\n", filepath.Join(rootPath, ".palace"))

	// Update .gitignore unless disabled
	if !opts.NoGitignore {
		if err := updateGitignore(rootPath, opts.Force); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not update .gitignore: %v\n", err)
		}
	}

	// Install VS Code integration unless disabled
	if !opts.NoVSCode {
		if err := installVSCodeIntegration(rootPath, opts.Force); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not set up VS Code integration: %v\n", err)
		}
	}

	// Install git hooks unless disabled
	if !opts.NoHooks {
		if err := installGitHooks(rootPath, opts.Force); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not install git hooks: %v\n", err)
		}
	}

	// Install agent configs if requested
	if opts.WithAgents != "" {
		if err := installAgentConfigs(rootPath, opts.WithAgents, opts.Force); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not install agent configs: %v\n", err)
		}
	}

	// Auto-scan unless explicitly disabled
	if !opts.NoScan {
		fmt.Printf("\nbuilding code index...\n")
		scanErr := ExecuteScan(ScanOptions{
			Root:    rootPath,
			Verbose: false,
		})

		if scanErr != nil {
			// Scan failed - provide helpful guidance based on error type
			fmt.Fprintf(os.Stderr, "\n⚠️  Warning: Initial scan failed: %v\n", scanErr)
			fmt.Fprintf(os.Stderr, "\nPossible causes:\n")
			fmt.Fprintf(os.Stderr, "  • Very large workspace (>100k files) - scanning may take a while\n")
			fmt.Fprintf(os.Stderr, "  • Complex/unconventional project structure\n")
			fmt.Fprintf(os.Stderr, "  • Insufficient disk space or permissions\n")
			fmt.Fprintf(os.Stderr, "  • Unsupported language/framework for deep analysis\n")
			fmt.Fprintf(os.Stderr, "\nYou can:\n")
			fmt.Fprintf(os.Stderr, "  1. Run 'palace scan --verbose' to see detailed progress\n")
			fmt.Fprintf(os.Stderr, "  2. Check .palace/guardrails.jsonc to exclude problematic directories\n")
			fmt.Fprintf(os.Stderr, "  3. Use Mind Palace without index (limited functionality)\n")
			fmt.Fprintf(os.Stderr, "\nMind Palace will still work for knowledge storage (store, recall, sessions),\n")
			fmt.Fprintf(os.Stderr, "but codebase exploration features (explore, symbols) won't be available.\n")

			// Don't fail init if scan fails - allow partial functionality
			// Return nil so user can still use Mind Palace for non-index features
			return nil
		}

		fmt.Printf("✓ Code index built successfully\n")
	}

	return nil
}

// MonorepoRoom represents a detected subproject in a monorepo
type MonorepoRoom struct {
	Name        string   // Room name (e.g., "driver-app")
	RelPath     string   // Relative path (e.g., "apps/driver_app")
	Category    string   // Category (e.g., "apps", "packages")
	EntryPoints []string // Detected entry points
	DependsOn   []string // Names of projects this depends on
}

// Note: detectMonorepoRooms was removed - use project.DetectMonorepo instead

// RoomConfig represents the JSON structure for a room file
type RoomConfig struct {
	SchemaVersion string     `json:"schemaVersion"`
	Kind          string     `json:"kind"`
	Name          string     `json:"name"`
	Summary       string     `json:"summary"`
	EntryPoints   []string   `json:"entryPoints"`
	Capabilities  []string   `json:"capabilities"`
	DependsOn     []string   `json:"dependsOn,omitempty"`
	Provenance    Provenance `json:"provenance"`
}

// Provenance tracks who/when created a config
type Provenance struct {
	CreatedBy string `json:"createdBy"`
	CreatedAt string `json:"createdAt"`
}

// writeMonorepoRoom creates a room configuration file for a detected monorepo subproject
func writeMonorepoRoom(path string, room MonorepoRoom) error {
	// Prefix entry points with the relative path
	var entryPoints []string
	for _, ep := range room.EntryPoints {
		entryPoints = append(entryPoints, filepath.Join(room.RelPath, ep))
	}

	// If no entry points detected, use the root of the subproject
	if len(entryPoints) == 0 {
		entryPoints = []string{room.RelPath}
	}

	roomConfig := RoomConfig{
		SchemaVersion: "1.0.0",
		Kind:          "palace/room",
		Name:          room.Name,
		Summary:       fmt.Sprintf("Auto-detected %s at %s", room.Category, room.RelPath),
		EntryPoints:   entryPoints,
		Capabilities:  []string{"read.file", "search.text"},
		DependsOn:     room.DependsOn,
		Provenance: Provenance{
			CreatedBy: "palace init --detect",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}

	data, err := json.MarshalIndent(roomConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// updateGitignore adds Mind Palace entries to .gitignore
func updateGitignore(root string, _ bool) error {
	gitignorePath := filepath.Join(root, ".gitignore")

	// Entries to add
	entries := []string{
		"",
		"# Mind Palace",
		".palace/scan/",
		".palace/outputs/",
		".palace/cache/",
		".palace/sessions/",
	}

	// Check if .gitignore exists
	content, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading .gitignore: %w", err)
	}

	existingContent := string(content)

	// Check if already configured
	if strings.Contains(existingContent, "# Mind Palace") {
		return nil // Already configured
	}

	// Append entries
	newContent := existingContent
	if len(existingContent) > 0 && !strings.HasSuffix(existingContent, "\n") {
		newContent += "\n"
	}
	newContent += strings.Join(entries, "\n") + "\n"

	if err := os.WriteFile(gitignorePath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("writing .gitignore: %w", err)
	}

	fmt.Println("✓ Updated .gitignore with Mind Palace entries")
	return nil
}

// installVSCodeIntegration sets up VS Code extensions.json and settings
func installVSCodeIntegration(root string, force bool) error {
	vscodeDir := filepath.Join(root, ".vscode")

	// Only set up if .vscode directory exists (VS Code project)
	if _, err := os.Stat(vscodeDir); os.IsNotExist(err) {
		return nil // Not a VS Code project, skip silently
	}

	extensionsPath := filepath.Join(vscodeDir, "extensions.json")

	// Check if extensions.json already exists
	if _, err := os.Stat(extensionsPath); err == nil && !force {
		// File exists, try to merge
		return mergeVSCodeExtensions(extensionsPath)
	}

	// Create new extensions.json
	extensions := map[string]interface{}{
		"recommendations": []string{
			"mind-palace.vscode-mind-palace",
		},
	}

	data, err := json.MarshalIndent(extensions, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling extensions.json: %w", err)
	}

	if err := os.WriteFile(extensionsPath, data, 0o644); err != nil {
		return fmt.Errorf("writing extensions.json: %w", err)
	}

	fmt.Println("✓ Created .vscode/extensions.json with Mind Palace extension")
	return nil
}

// mergeVSCodeExtensions adds Mind Palace extension to existing extensions.json
func mergeVSCodeExtensions(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var extensions map[string]interface{}
	if err := json.Unmarshal(content, &extensions); err != nil {
		return err
	}

	// Get or create recommendations array
	recommendations, ok := extensions["recommendations"].([]interface{})
	if !ok {
		recommendations = []interface{}{}
	}

	// Check if already present
	for _, rec := range recommendations {
		if rec == "mind-palace.vscode-mind-palace" {
			return nil // Already present
		}
	}

	// Add Mind Palace extension
	recommendations = append(recommendations, "mind-palace.vscode-mind-palace")
	extensions["recommendations"] = recommendations

	data, err := json.MarshalIndent(extensions, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}

	fmt.Println("✓ Added Mind Palace to .vscode/extensions.json recommendations")
	return nil
}

// installGitHooks sets up git hooks for Mind Palace
func installGitHooks(root string, force bool) error {
	gitDir := filepath.Join(root, ".git")

	// Check if this is a git repository
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil // Not a git repo, skip silently
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}

	// Install post-commit hook to update scan on significant changes
	postCommitPath := filepath.Join(hooksDir, "post-commit")

	// Check if hook already exists
	if _, err := os.Stat(postCommitPath); err == nil && !force {
		// Check if it already contains our hook
		content, err := os.ReadFile(postCommitPath)
		if err == nil && strings.Contains(string(content), "mind-palace") {
			return nil // Already installed
		}
		// Append to existing hook
		return appendToGitHook(postCommitPath, getPalaceHookContent())
	}

	// Create new hook
	hookContent := `#!/bin/sh
# Mind Palace post-commit hook
# Auto-refreshes the code index after commits

` + getPalaceHookContent()

	if err := os.WriteFile(postCommitPath, []byte(hookContent), 0o755); err != nil {
		return fmt.Errorf("writing post-commit hook: %w", err)
	}

	fmt.Println("✓ Installed git post-commit hook for auto-index refresh")
	return nil
}

// getPalaceHookContent returns the Mind Palace hook content
func getPalaceHookContent() string {
	return `# Mind Palace: Refresh index on commit (runs in background)
if command -v palace >/dev/null 2>&1; then
  palace scan --quiet &
fi
`
}

// appendToGitHook appends content to an existing git hook
func appendToGitHook(path, content string) error {
	existing, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	newContent := string(existing)
	if !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	newContent += "\n" + content

	if err := os.WriteFile(path, []byte(newContent), 0o755); err != nil {
		return err
	}

	fmt.Println("✓ Added Mind Palace to existing post-commit hook")
	return nil
}

// installAgentConfigs installs agent configurations based on the --with-agents flag
func installAgentConfigs(root string, agentsArg string, _ bool) error {
	var agents []string

	if agentsArg == "auto" {
		// Auto-detect installed agents
		detected := detectInstalledAgents(root)
		if len(detected) == 0 {
			fmt.Println("ℹ No AI coding agents detected - skipping agent config installation")
			fmt.Println("  Run 'palace mcp-config install --list' to see supported agents")
			return nil
		}

		for _, d := range detected {
			agents = append(agents, d.Key)
		}
		fmt.Printf("✓ Detected %d AI coding agent(s): %s\n", len(agents), strings.Join(agents, ", "))
	} else {
		// Parse comma-separated list
		for _, agent := range strings.Split(agentsArg, ",") {
			agent = strings.TrimSpace(agent)
			if agent != "" {
				agents = append(agents, agent)
			}
		}
	}

	if len(agents) == 0 {
		return nil
	}

	// Install configs for each agent
	fmt.Printf("\nInstalling agent configurations...\n")
	for _, agent := range agents {
		opts := MCPConfigOptions{
			For:     agent,
			Root:    root,
			Install: true,
		}
		if err := ExecuteMCPConfig(opts); err != nil {
			fmt.Fprintf(os.Stderr, "  warning: failed to install config for %s: %v\n", agent, err)
		}
	}

	return nil
}

// detectInstalledAgents detects which AI coding agents are installed
func detectInstalledAgents(root string) []DetectedAgent {
	var detected []DetectedAgent

	// Agent detection rules
	detectionRules := []struct {
		Key        string
		Name       string
		Indicators []string // Files/directories that indicate this agent
	}{
		{
			Key:  "cursor",
			Name: "Cursor",
			Indicators: []string{
				".cursor",
				".cursorrules",
			},
		},
		{
			Key:  "vscode",
			Name: "VS Code",
			Indicators: []string{
				".vscode",
			},
		},
		{
			Key:  "windsurf",
			Name: "Windsurf",
			Indicators: []string{
				".windsurf",
				".windsurfrules",
			},
		},
		{
			Key:  "claude-code",
			Name: "Claude Code",
			Indicators: []string{
				"CLAUDE.md",
				".claude",
			},
		},
		{
			Key:  "zed",
			Name: "Zed",
			Indicators: []string{
				".zed",
			},
		},
		{
			Key:  "cline",
			Name: "Cline",
			Indicators: []string{
				".clinerules",
				".cline",
			},
		},
	}

	for _, rule := range detectionRules {
		for _, indicator := range rule.Indicators {
			indicatorPath := filepath.Join(root, indicator)
			if _, err := os.Stat(indicatorPath); err == nil {
				detected = append(detected, DetectedAgent{
					Key:        rule.Key,
					Name:       rule.Name,
					Indicator:  indicator,
					Confidence: "high",
				})
				break // Only add once per agent
			}
		}
	}

	// Also check for global installations (home directory)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalIndicators := []struct {
			Key       string
			Name      string
			Path      string
			Subfolder string
		}{
			{Key: "claude-desktop", Name: "Claude Desktop", Path: ".config/claude", Subfolder: ""},
			{Key: "cursor", Name: "Cursor", Path: ".cursor", Subfolder: ""},
			{Key: "vscode", Name: "VS Code", Path: ".vscode", Subfolder: ""},
		}

		for _, gi := range globalIndicators {
			globalPath := filepath.Join(homeDir, gi.Path)
			if gi.Subfolder != "" {
				globalPath = filepath.Join(globalPath, gi.Subfolder)
			}
			if _, err := os.Stat(globalPath); err == nil {
				// Check if already detected
				found := false
				for _, d := range detected {
					if d.Key == gi.Key {
						found = true
						break
					}
				}
				if !found {
					detected = append(detected, DetectedAgent{
						Key:        gi.Key,
						Name:       gi.Name,
						Indicator:  gi.Path + " (global)",
						Confidence: "medium", // Lower confidence for global detection
					})
				}
			}
		}
	}

	return detected
}
