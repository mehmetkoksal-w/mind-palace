package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunMCPConfig_MissingFor(t *testing.T) {
	err := RunMCPConfig([]string{})
	if err == nil {
		t.Fatal("expected error for missing --for flag")
	}
	if !strings.Contains(err.Error(), "--for flag is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunMCPConfig_InvalidTarget(t *testing.T) {
	err := RunMCPConfig([]string{"--for", "invalid"})
	if err == nil {
		t.Fatal("expected error for invalid target")
	}
	if !strings.Contains(err.Error(), "invalid target") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGenerateMCPConfig(t *testing.T) {
	config := generateMCPConfig("/usr/local/bin/palace", "/home/user/project")

	if len(config.MCPServers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(config.MCPServers))
	}

	server, ok := config.MCPServers["mind-palace"]
	if !ok {
		t.Fatal("expected mind-palace server")
	}

	if server.Command != "/usr/local/bin/palace" {
		t.Errorf("unexpected command: %s", server.Command)
	}

	if len(server.Args) != 3 || server.Args[0] != "serve" || server.Args[1] != "--root" || server.Args[2] != "/home/user/project" {
		t.Errorf("unexpected args: %v", server.Args)
	}
}

func TestGetConfigPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home directory")
	}

	tests := []struct {
		target   string
		rootPath string
		want     string
	}{
		{
			target:   "claude-code",
			rootPath: "/workspace",
			want:     "/workspace/.mcp.json",
		},
		{
			target: "cursor",
			want:   filepath.Join(home, ".cursor", "mcp.json"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			root := tt.rootPath
			if root == "" {
				root = home
			}
			got, err := getConfigPath(tt.target, root)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}

	// Test claude-desktop (platform-specific)
	t.Run("claude-desktop", func(t *testing.T) {
		got, err := getConfigPath("claude-desktop", home)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var expectedSuffix string
		switch runtime.GOOS {
		case "darwin":
			expectedSuffix = filepath.Join("Library", "Application Support", "Claude", "claude_desktop_config.json")
		case "windows":
			expectedSuffix = filepath.Join("AppData", "Roaming", "Claude", "claude_desktop_config.json")
		default:
			expectedSuffix = filepath.Join(".config", "Claude", "claude_desktop_config.json")
		}

		if !strings.HasSuffix(got, expectedSuffix) {
			t.Errorf("got %q, expected suffix %q", got, expectedSuffix)
		}
	})
}

func TestInstallConfig(t *testing.T) {
	tmpDir := t.TempDir()

	config := generateMCPConfig("/usr/local/bin/palace", tmpDir)

	// Test installing to a new file
	configPath := filepath.Join(tmpDir, ".mcp.json")
	err := installConfig("claude-code", config, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the file was created
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	mcpServers, ok := result["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("expected mcpServers in config")
	}

	if _, ok := mcpServers["mind-palace"]; !ok {
		t.Fatal("expected mind-palace server in config")
	}
}

func TestInstallConfig_MergeExisting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing config with another server
	existingConfig := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"other-server": map[string]interface{}{
				"command": "/usr/bin/other",
				"args":    []string{"serve"},
			},
		},
		"otherConfig": "value",
	}

	configPath := filepath.Join(tmpDir, ".mcp.json")
	data, _ := json.MarshalIndent(existingConfig, "", "  ")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("failed to write existing config: %v", err)
	}

	// Install our config
	config := generateMCPConfig("/usr/local/bin/palace", tmpDir)
	err := installConfig("claude-code", config, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify both servers exist
	data, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	mcpServers, ok := result["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("expected mcpServers in config")
	}

	if _, ok := mcpServers["mind-palace"]; !ok {
		t.Fatal("expected mind-palace server in config")
	}

	if _, ok := mcpServers["other-server"]; !ok {
		t.Fatal("expected other-server to be preserved")
	}

	// Verify other config is preserved
	if result["otherConfig"] != "value" {
		t.Fatal("expected otherConfig to be preserved")
	}
}
