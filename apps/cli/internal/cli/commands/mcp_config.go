package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func init() {
	Register(&Command{
		Name:        "mcp-config",
		Description: "Generate MCP configuration for AI tools",
		Run:         RunMCPConfig,
	})
}

// MCPConfigOptions contains the configuration for the mcp-config command.
type MCPConfigOptions struct {
	For     string
	Root    string
	Install bool
}

// MCPServerConfig represents an MCP server entry in the config.
type MCPServerConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// MCPConfig represents the full MCP configuration structure (for mcpServers format).
type MCPConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// ToolInfo contains metadata about a supported tool.
type ToolInfo struct {
	Name        string
	Description string
	ConfigKey   string // The key used for MCP servers in config (mcpServers, servers, mcp, context_servers)
	Format      string // json, toml
}

// supportedTools lists all supported AI tools with their configuration details.
var supportedTools = map[string]ToolInfo{
	"claude-code": {
		Name:        "Claude Code",
		Description: "Anthropic's Claude Code CLI",
		ConfigKey:   "mcpServers",
		Format:      "json",
	},
	"claude-desktop": {
		Name:        "Claude Desktop",
		Description: "Anthropic's Claude Desktop app",
		ConfigKey:   "mcpServers",
		Format:      "json",
	},
	"cursor": {
		Name:        "Cursor",
		Description: "Cursor AI editor",
		ConfigKey:   "mcpServers",
		Format:      "json",
	},
	"vscode": {
		Name:        "VS Code Copilot",
		Description: "GitHub Copilot in VS Code",
		ConfigKey:   "servers",
		Format:      "json",
	},
	"windsurf": {
		Name:        "Windsurf",
		Description: "Codeium's Windsurf IDE",
		ConfigKey:   "mcpServers",
		Format:      "json",
	},
	"cline": {
		Name:        "Cline",
		Description: "Cline VS Code extension",
		ConfigKey:   "mcpServers",
		Format:      "json",
	},
	"zed": {
		Name:        "Zed",
		Description: "Zed editor",
		ConfigKey:   "context_servers",
		Format:      "json",
	},
	"codex": {
		Name:        "OpenAI Codex",
		Description: "OpenAI's Codex CLI",
		ConfigKey:   "mcp_servers",
		Format:      "toml",
	},
	"antigravity": {
		Name:        "Antigravity",
		Description: "Google's Antigravity IDE",
		ConfigKey:   "mcpServers",
		Format:      "json",
	},
	"opencode": {
		Name:        "OpenCode",
		Description: "OpenCode terminal AI assistant",
		ConfigKey:   "mcp",
		Format:      "json",
	},
	"jetbrains": {
		Name:        "JetBrains",
		Description: "JetBrains IDEs (IntelliJ, PyCharm, etc.)",
		ConfigKey:   "mcpServers",
		Format:      "json",
	},
	"gemini-cli": {
		Name:        "Gemini CLI",
		Description: "Google's Gemini CLI",
		ConfigKey:   "mcpServers",
		Format:      "json",
	},
}

// RunMCPConfig executes the mcp-config command with parsed arguments.
func RunMCPConfig(args []string) error {
	fs := flag.NewFlagSet("mcp-config", flag.ContinueOnError)
	forTarget := fs.String("for", "", "target tool (see 'palace help mcp-config' for list)")
	root := fs.String("root", ".", "workspace root")
	install := fs.Bool("install", false, "install config to target's config file")
	list := fs.Bool("list", false, "list all supported tools")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *list {
		return listSupportedTools()
	}

	if *forTarget == "" {
		return fmt.Errorf("--for flag is required. Use --list to see supported tools")
	}

	return ExecuteMCPConfig(MCPConfigOptions{
		For:     *forTarget,
		Root:    *root,
		Install: *install,
	})
}

// listSupportedTools prints all supported tools.
func listSupportedTools() error {
	fmt.Println("Supported AI tools:")
	fmt.Println()
	for key, tool := range supportedTools {
		fmt.Printf("  %-15s %s\n", key, tool.Description)
	}
	fmt.Println()
	fmt.Println("Usage: palace mcp-config --for <tool> [--install]")
	return nil
}

// ExecuteMCPConfig generates or installs MCP configuration.
func ExecuteMCPConfig(opts MCPConfigOptions) error {
	// Validate target
	tool, ok := supportedTools[opts.For]
	if !ok {
		return fmt.Errorf("unknown tool %q. Use --list to see supported tools", opts.For)
	}

	// Get absolute workspace root
	rootPath, err := filepath.Abs(opts.Root)
	if err != nil {
		return fmt.Errorf("resolve workspace root: %w", err)
	}

	// Auto-detect palace binary location
	palacePath, err := findPalaceBinary()
	if err != nil {
		return fmt.Errorf("detect palace binary: %w", err)
	}

	if opts.Install {
		return installConfigForTool(opts.For, tool, palacePath, rootPath)
	}

	// Print the config for the tool
	return printConfigForTool(opts.For, tool, palacePath, rootPath)
}

// findPalaceBinary locates the palace binary.
func findPalaceBinary() (string, error) {
	// First try: get the current executable path
	execPath, err := os.Executable()
	if err == nil {
		resolved, err := filepath.EvalSymlinks(execPath)
		if err == nil {
			return resolved, nil
		}
		return execPath, nil
	}

	// Second try: look in PATH
	path, err := exec.LookPath("palace")
	if err == nil {
		return path, nil
	}

	// Third try: check common go install locations
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			gopath = filepath.Join(home, "go")
		}
	}
	if gopath != "" {
		goBinPath := filepath.Join(gopath, "bin", "palace")
		if runtime.GOOS == "windows" {
			goBinPath += ".exe"
		}
		if _, err := os.Stat(goBinPath); err == nil {
			return goBinPath, nil
		}
	}

	return "", fmt.Errorf("palace binary not found. Install with: go install github.com/koksalmehmet/mind-palace/apps/cli@latest")
}

// printConfigForTool outputs the configuration for a specific tool.
func printConfigForTool(target string, tool ToolInfo, palacePath, rootPath string) error {
	switch tool.Format {
	case "toml":
		return printTOMLConfig(target, palacePath, rootPath)
	default:
		return printJSONConfig(target, tool, palacePath, rootPath)
	}
}

// printJSONConfig outputs JSON configuration for the tool.
func printJSONConfig(target string, tool ToolInfo, palacePath, rootPath string) error {
	config := generateJSONConfig(target, tool, palacePath, rootPath)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(config)
}

// printTOMLConfig outputs TOML configuration for OpenAI Codex.
func printTOMLConfig(_, palacePath, rootPath string) error {
	toml := generateTOMLConfig(palacePath, rootPath)
	fmt.Println(toml)
	return nil
}

// generateJSONConfig creates the appropriate JSON config structure for the tool.
func generateJSONConfig(target string, tool ToolInfo, palacePath, rootPath string) map[string]interface{} {
	serverConfig := map[string]interface{}{
		"command": palacePath,
		"args":    []string{"serve", "--root", rootPath},
	}

	// VS Code uses a different structure with "type": "stdio"
	if target == "vscode" {
		serverConfig["type"] = "stdio"
	}

	// Zed uses "source": "custom" instead of type
	if target == "zed" {
		serverConfig = map[string]interface{}{
			"source":  "custom",
			"command": palacePath,
			"args":    []string{"serve", "--root", rootPath},
		}
	}

	// OpenCode uses a different structure
	if target == "opencode" {
		serverConfig = map[string]interface{}{
			"type":    "local",
			"command": []string{palacePath, "serve", "--root", rootPath},
			"enabled": true,
		}
	}

	return map[string]interface{}{
		tool.ConfigKey: map[string]interface{}{
			"mind-palace": serverConfig,
		},
	}
}

// generateTOMLConfig creates TOML configuration for OpenAI Codex.
func generateTOMLConfig(palacePath, rootPath string) string {
	return fmt.Sprintf(`[mcp_servers.mind-palace]
command = %q
args = ["serve", "--root", %q]
`, palacePath, rootPath)
}

// installConfigForTool writes the configuration to the appropriate file.
func installConfigForTool(target string, tool ToolInfo, palacePath, rootPath string) error {
	configPath, err := getConfigPath(target, rootPath)
	if err != nil {
		return err
	}

	switch tool.Format {
	case "toml":
		return installTOMLConfig(configPath, palacePath, rootPath)
	default:
		return installJSONConfig(target, tool, configPath, palacePath, rootPath)
	}
}

// installJSONConfig installs JSON configuration.
func installJSONConfig(target string, tool ToolInfo, configPath, palacePath, rootPath string) error {
	// Read existing config if it exists
	existingConfig := make(map[string]interface{})
	data, err := os.ReadFile(configPath)
	if err == nil {
		if err := json.Unmarshal(data, &existingConfig); err != nil {
			return fmt.Errorf("parse existing config %s: %w", configPath, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read config %s: %w", configPath, err)
	}

	// Generate the server config
	newConfig := generateJSONConfig(target, tool, palacePath, rootPath)
	newServers, ok := newConfig[tool.ConfigKey].(map[string]interface{})
	if !ok {
		return fmt.Errorf("internal error: invalid config structure for %s", target)
	}

	// Merge servers
	existingServers, ok := existingConfig[tool.ConfigKey].(map[string]interface{})
	if !ok {
		existingServers = make(map[string]interface{})
	}
	for k, v := range newServers {
		existingServers[k] = v
	}
	existingConfig[tool.ConfigKey] = existingServers

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Write updated config
	output, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, output, 0o644); err != nil {
		return fmt.Errorf("write config %s: %w", configPath, err)
	}

	fmt.Fprintf(os.Stderr, "Installed mind-palace MCP server to %s\n", configPath)
	return nil
}

// installTOMLConfig installs TOML configuration for OpenAI Codex.
func installTOMLConfig(configPath, palacePath, rootPath string) error {
	// Read existing config if it exists
	existingContent := ""
	data, err := os.ReadFile(configPath)
	if err == nil {
		existingContent = string(data)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read config %s: %w", configPath, err)
	}

	// Check if mind-palace config already exists
	if strings.Contains(existingContent, "[mcp_servers.mind-palace]") {
		// Update existing config by replacing the section
		// This is a simple approach - a proper TOML parser would be better
		lines := strings.Split(existingContent, "\n")
		var result []string
		skip := false
		for _, line := range lines {
			if strings.HasPrefix(line, "[mcp_servers.mind-palace]") {
				skip = true
				continue
			}
			if skip && strings.HasPrefix(line, "[") {
				skip = false
			}
			if !skip {
				result = append(result, line)
			}
		}
		existingContent = strings.Join(result, "\n")
	}

	// Append our config
	newConfig := generateTOMLConfig(palacePath, rootPath)
	finalContent := strings.TrimRight(existingContent, "\n") + "\n\n" + newConfig

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, []byte(finalContent), 0o644); err != nil {
		return fmt.Errorf("write config %s: %w", configPath, err)
	}

	fmt.Fprintf(os.Stderr, "Installed mind-palace MCP server to %s\n", configPath)
	return nil
}

// getConfigPath returns the configuration file path for the given target.
func getConfigPath(target, rootPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	switch target {
	case "claude-code":
		// Claude Code uses .mcp.json in workspace root
		return filepath.Join(rootPath, ".mcp.json"), nil

	case "claude-desktop":
		// Claude Desktop config location varies by OS
		switch runtime.GOOS {
		case "darwin":
			return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), nil
		case "windows":
			return filepath.Join(home, "AppData", "Roaming", "Claude", "claude_desktop_config.json"), nil
		default: // Linux and others
			return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json"), nil
		}

	case "cursor":
		return filepath.Join(home, ".cursor", "mcp.json"), nil

	case "vscode":
		// VS Code uses .vscode/mcp.json in workspace
		return filepath.Join(rootPath, ".vscode", "mcp.json"), nil

	case "windsurf":
		return filepath.Join(home, ".codeium", "windsurf", "mcp_config.json"), nil

	case "cline":
		// Cline stores config in VS Code's globalStorage
		switch runtime.GOOS {
		case "darwin":
			return filepath.Join(home, "Library", "Application Support", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"), nil
		case "windows":
			return filepath.Join(home, "AppData", "Roaming", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"), nil
		default:
			return filepath.Join(home, ".config", "Code", "User", "globalStorage", "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"), nil
		}

	case "zed":
		// Zed uses settings.json
		switch runtime.GOOS {
		case "darwin":
			return filepath.Join(home, "Library", "Application Support", "Zed", "settings.json"), nil
		case "windows":
			return filepath.Join(home, "AppData", "Roaming", "Zed", "settings.json"), nil
		default:
			return filepath.Join(home, ".config", "zed", "settings.json"), nil
		}

	case "codex":
		// OpenAI Codex uses TOML config
		return filepath.Join(home, ".codex", "config.toml"), nil

	case "antigravity":
		// Antigravity uses mcp_config.json (location may vary)
		switch runtime.GOOS {
		case "darwin":
			return filepath.Join(home, "Library", "Application Support", "Antigravity", "mcp_config.json"), nil
		case "windows":
			return filepath.Join(home, "AppData", "Roaming", "Antigravity", "mcp_config.json"), nil
		default:
			return filepath.Join(home, ".config", "antigravity", "mcp_config.json"), nil
		}

	case "opencode":
		// OpenCode uses opencode.json in workspace or ~/.config/opencode/
		// Default to global config
		return filepath.Join(home, ".config", "opencode", "opencode.json"), nil

	case "jetbrains":
		// JetBrains IDEs typically manage MCP through the UI
		// But we can provide a config file location for manual setup
		switch runtime.GOOS {
		case "darwin":
			return filepath.Join(home, "Library", "Application Support", "JetBrains", "mcp_config.json"), nil
		case "windows":
			return filepath.Join(home, "AppData", "Roaming", "JetBrains", "mcp_config.json"), nil
		default:
			return filepath.Join(home, ".config", "JetBrains", "mcp_config.json"), nil
		}

	case "gemini-cli":
		// Gemini CLI config location
		switch runtime.GOOS {
		case "darwin":
			return filepath.Join(home, "Library", "Application Support", "gemini", "config.json"), nil
		case "windows":
			return filepath.Join(home, "AppData", "Roaming", "gemini", "config.json"), nil
		default:
			return filepath.Join(home, ".config", "gemini", "config.json"), nil
		}

	default:
		return "", fmt.Errorf("unknown target: %s", target)
	}
}

// Legacy functions for backward compatibility (used by tests)

// generateMCPConfig creates the MCP server configuration (legacy).
func generateMCPConfig(palacePath, rootPath string) MCPConfig {
	return MCPConfig{
		MCPServers: map[string]MCPServerConfig{
			"mind-palace": {
				Command: palacePath,
				Args:    []string{"serve", "--root", rootPath},
			},
		},
	}
}

// installConfig writes the configuration to the appropriate file (legacy).
func installConfig(target string, _ MCPConfig, rootPath string) error {
	tool, ok := supportedTools[target]
	if !ok {
		return fmt.Errorf("unknown target: %s", target)
	}

	palacePath, _ := findPalaceBinary()
	return installConfigForTool(target, tool, palacePath, rootPath)
}
