// Package dashboard provides an HTTP server for the Mind Palace web dashboard.
package dashboard

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/koksalmehmet/mind-palace/apps/cli/internal/butler"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/corridor"
	"github.com/koksalmehmet/mind-palace/apps/cli/internal/memory"
)

// Server provides the dashboard HTTP server.
type Server struct {
	butler   *butler.Butler
	memory   *memory.Memory
	corridor *corridor.GlobalCorridor
	port     int
	root     string
	mu       sync.RWMutex // Protects butler, memory, and root during workspace switch
}

// Config holds dashboard server configuration.
type Config struct {
	Butler   *butler.Butler
	Memory   *memory.Memory
	Corridor *corridor.GlobalCorridor
	Port     int
	Root     string
}

// New creates a new dashboard server.
func New(cfg Config) *Server {
	return &Server{
		butler:   cfg.Butler,
		memory:   cfg.Memory,
		corridor: cfg.Corridor,
		port:     cfg.Port,
		root:     cfg.Root,
	}
}

// Start starts the dashboard server.
func (s *Server) Start(openBrowser bool) error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/rooms", s.handleRooms)
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/sessions/", s.handleSessionDetail)
	mux.HandleFunc("/api/activity", s.handleActivity)
	mux.HandleFunc("/api/learnings", s.handleLearnings)
	mux.HandleFunc("/api/file-intel", s.handleFileIntel)
	mux.HandleFunc("/api/corridors", s.handleCorridors)
	mux.HandleFunc("/api/corridors/personal", s.handleCorridorPersonal)
	mux.HandleFunc("/api/corridors/links", s.handleCorridorLinks)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/graph/", s.handleGraph)
	mux.HandleFunc("/api/hotspots", s.handleHotspots)
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/brief", s.handleBrief)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/workspaces", s.handleWorkspaces)
	mux.HandleFunc("/api/workspace/switch", s.handleWorkspaceSwitch)

	// Static files (embedded dashboard or fallback)
	mux.Handle("/", http.FileServer(http.FS(embeddedAssets)))

	// Find available port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	addr := listener.Addr().(*net.TCPAddr)
	url := fmt.Sprintf("http://localhost:%d", addr.Port)

	fmt.Printf("Dashboard server starting at %s\n", url)

	if openBrowser {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openURL(url)
		}()
	}

	server := &http.Server{
		Handler:      corsMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return server.Serve(listener)
}

// corsMiddleware adds CORS headers for development.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// openURL opens a URL in the default browser.
func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return
	}
	// Run in goroutine to avoid blocking and properly clean up process
	go func() {
		_ = cmd.Run()
	}()
}
