package dashboard

import (
	"testing"
)

func TestNew(t *testing.T) {
	cfg := Config{
		Port: 8080,
		Root: "/test/root",
	}

	server := New(cfg)

	if server == nil {
		t.Fatal("New() returned nil")
	}

	if server.port != 8080 {
		t.Errorf("expected port 8080, got %d", server.port)
	}

	if server.root != "/test/root" {
		t.Errorf("expected root '/test/root', got %q", server.root)
	}

	if server.butler != nil {
		t.Error("expected butler to be nil")
	}

	if server.memory != nil {
		t.Error("expected memory to be nil")
	}

	if server.corridor != nil {
		t.Error("expected corridor to be nil")
	}
}

func TestConfig(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		cfg := Config{}

		if cfg.Port != 0 {
			t.Errorf("expected default port 0, got %d", cfg.Port)
		}

		if cfg.Root != "" {
			t.Errorf("expected empty root, got %q", cfg.Root)
		}
	})

	t.Run("with values", func(t *testing.T) {
		cfg := Config{
			Port: 3000,
			Root: "/workspace",
		}

		if cfg.Port != 3000 {
			t.Errorf("expected port 3000, got %d", cfg.Port)
		}

		if cfg.Root != "/workspace" {
			t.Errorf("expected root '/workspace', got %q", cfg.Root)
		}
	})
}

func TestServerFields(t *testing.T) {
	server := &Server{
		port: 9000,
		root: "/my/workspace",
	}

	if server.port != 9000 {
		t.Errorf("expected port 9000, got %d", server.port)
	}

	if server.root != "/my/workspace" {
		t.Errorf("expected root '/my/workspace', got %q", server.root)
	}
}

func TestWorkspaceInfo(t *testing.T) {
	server := &Server{
		root: "/home/user/project",
	}

	info := server.getWorkspaceInfo()

	if info == nil {
		t.Fatal("getWorkspaceInfo() returned nil")
	}

	if info["path"] != "/home/user/project" {
		t.Errorf("expected path '/home/user/project', got %v", info["path"])
	}

	if info["name"] != "project" {
		t.Errorf("expected name 'project', got %v", info["name"])
	}
}

func TestWorkspaceInfoSafe(t *testing.T) {
	server := &Server{}

	info := server.getWorkspaceInfoSafe("/test/path", nil)

	if info == nil {
		t.Fatal("getWorkspaceInfoSafe() returned nil")
	}

	if info["path"] != "/test/path" {
		t.Errorf("expected path '/test/path', got %v", info["path"])
	}

	if info["name"] != "path" {
		t.Errorf("expected name 'path', got %v", info["name"])
	}
}

func TestSwitchWorkspace(t *testing.T) {
	t.Run("switch to temp directory", func(t *testing.T) {
		server := &Server{
			root: "/original",
		}

		// Switch to /tmp which exists
		err := server.switchWorkspace("/tmp")
		if err != nil {
			t.Errorf("switchWorkspace failed: %v", err)
		}

		if server.root != "/tmp" {
			t.Errorf("expected root to be '/tmp', got %q", server.root)
		}
	})

	t.Run("handles missing palace directory", func(t *testing.T) {
		server := &Server{
			root: "/original",
		}

		// Switch to /tmp - should succeed even without .palace
		err := server.switchWorkspace("/tmp")
		if err != nil {
			t.Errorf("switchWorkspace failed: %v", err)
		}

		// Butler and memory may be nil (no palace setup)
		// That's expected behavior
	})
}
