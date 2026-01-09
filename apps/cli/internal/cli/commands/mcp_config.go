package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// MCPConfig represents the full MCP configuration structure.
type MCPConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// RunMCPConfig executes the mcp-config command with parsed arguments.
func RunMCPConfig(args []string) error {
	fs := flag.NewFlagSet("mcp-config", flag.ContinueOnError)
	forTarget := fs.String("for", "", "target tool: claude-code, claude-desktop, cursor")
	root := fs.String("root", ".", "workspace root")
	install := fs.Bool("install", false, "install config to target's config file")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *forTarget == "" {
		return fmt.Errorf("--for flag is required. Use: claude-code, claude-desktop, or cursor")
	}

	return ExecuteMCPConfig(MCPConfigOptions{
		For:     *forTarget,
		Root:    *root,
		Install: *install,
	})
}

// ExecuteMCPConfig generates or installs MCP configuration.
func ExecuteMCPConfig(opts MCPConfigOptions) error {
	// Validate target
	validTargets := map[string]bool{
		"claude-code":    true,
		"claude-desktop": true,
		"cursor":         true,
	}
	if !validTargets[opts.For] {
		return fmt.Errorf("invalid target %q. Use: claude-code, claude-desktop, or cursor", opts.For)
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

	// Generate the config
	config := generateMCPConfig(palacePath, rootPath)

	if opts.Install {
		return installConfig(opts.For, config, rootPath)
	}

	// Print the config as JSON
	return printConfig(config)
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

// generateMCPConfig creates the MCP server configuration.
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

// printConfig outputs the configuration as formatted JSON.
func printConfig(config MCPConfig) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(config)
}

// installConfig writes the configuration to the appropriate file.
func installConfig(target string, config MCPConfig, rootPath string) error {
	configPath, err := getConfigPath(target, rootPath)
	if err != nil {
		return err
	}

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

	// Merge mcpServers
	mcpServers, ok := existingConfig["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}
	mcpServers["mind-palace"] = config.MCPServers["mind-palace"]
	existingConfig["mcpServers"] = mcpServers

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

// getConfigPath returns the configuration file path for the given target.
func getConfigPath(target, rootPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	switch target {
	case "claude-code":
		// Claude Code uses .mcp.json in workspace root or ~/.claude/claude_desktop_config.json
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
	default:
		return "", fmt.Errorf("unknown target: %s", target)
	}
}
