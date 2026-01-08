// Package dashboard provides an HTTP server for the Mind Palace web dashboard.
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
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
	butler         *butler.Butler
	memory         *memory.Memory
	corridor       *corridor.GlobalCorridor
	port           int
	root           string
	allowedOrigins []string     // CORS allowed origins
	wsHub          *WSHub       // WebSocket hub for real-time updates
	mu             sync.RWMutex // Protects butler, memory, and root during workspace switch
}

// Config holds dashboard server configuration.
type Config struct {
	Butler         *butler.Butler
	Memory         *memory.Memory
	Corridor       *corridor.GlobalCorridor
	Port           int
	Root           string
	AllowedOrigins []string // CORS allowed origins
}

// New creates a new dashboard server.
func New(cfg Config) *Server {
	// Default to localhost origins if none specified (development mode)
	allowedOrigins := cfg.AllowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{
			"http://localhost:4200",
			"http://localhost:3000",
			"http://127.0.0.1:4200",
			"http://127.0.0.1:3000",
		}
	}

	return &Server{
		butler:         cfg.Butler,
		memory:         cfg.Memory,
		corridor:       cfg.Corridor,
		port:           cfg.Port,
		root:           cfg.Root,
		allowedOrigins: allowedOrigins,
		wsHub:          NewWSHub(),
	}
}

// Start starts the dashboard server.
func (s *Server) Start(openBrowser bool) error {
	mux := http.NewServeMux()

	// Start WebSocket hub
	go s.wsHub.Run()

	// WebSocket endpoint
	mux.HandleFunc("/api/ws", s.handleWebSocket)

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
	mux.HandleFunc("/api/search/semantic", s.handleSemanticSearch)
	mux.HandleFunc("/api/graph/", s.handleGraph)
	mux.HandleFunc("/api/hotspots", s.handleHotspots)
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/brief", s.handleBrief)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/workspaces", s.handleWorkspaces)
	mux.HandleFunc("/api/workspace/switch", s.handleWorkspaceSwitch)

	// Brain API endpoints (ideas, decisions, conversations, context)
	mux.HandleFunc("/api/remember", s.handleRemember)
	mux.HandleFunc("/api/brain/search", s.handleBrainSearch)
	mux.HandleFunc("/api/brain/context", s.handleBrainContext)
	mux.HandleFunc("/api/ideas", s.handleIdeas)
	mux.HandleFunc("/api/decisions", s.handleDecisions)
	mux.HandleFunc("/api/decisions/timeline", s.handleDecisionTimeline)
	mux.HandleFunc("/api/decisions/", s.handleDecisionDetail)
	mux.HandleFunc("/api/conversations", s.handleConversations)
	mux.HandleFunc("/api/conversations/", s.handleConversationDetail)
	mux.HandleFunc("/api/links", s.handleLinks)
	mux.HandleFunc("/api/contradictions", s.handleContradictions)
	mux.HandleFunc("/api/decay/stats", s.handleDecayStats)
	mux.HandleFunc("/api/decay/preview", s.handleDecayPreview)

	// Postmortem API endpoints
	mux.HandleFunc("/api/postmortems/stats", s.handlePostmortemStats)
	mux.HandleFunc("/api/postmortems/", s.handlePostmortemDetail)
	mux.HandleFunc("/api/postmortems", s.handlePostmortems)
	mux.HandleFunc("/api/briefings/smart", s.handleSmartBriefing)

	// Context & Scope API endpoints
	mux.HandleFunc("/api/context/preview", s.handleContextPreview)
	mux.HandleFunc("/api/scope/explain", s.handleScopeExplain)
	mux.HandleFunc("/api/scope/hierarchy", s.handleScopeHierarchy)

	// Favicon handler (serves inline SVG regardless of which dashboard is used)
	mux.HandleFunc("/favicon.ico", s.handleFavicon)
	mux.HandleFunc("/favicon.svg", s.handleFavicon)

	// Static files with SPA fallback (serve index.html for client-side routes)
	mux.Handle("/", spaHandler(embeddedAssets))

	// Find available port
	lc := net.ListenConfig{}
	listener, err := lc.Listen(context.Background(), "tcp", fmt.Sprintf(":%d", s.port))
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
		Handler:      s.configureCORS(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return server.Serve(listener)
}

// configureCORS returns a CORS middleware with origin checking.
func (s *Server) configureCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range s.allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

		// Set CORS headers only for allowed origins
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			if allowed {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
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
		cmd = exec.CommandContext(context.Background(), "open", url)
	case "linux":
		cmd = exec.CommandContext(context.Background(), "xdg-open", url)
	case "windows":
		cmd = exec.CommandContext(context.Background(), "cmd", "/c", "start", url)
	default:
		return
	}
	// Run in goroutine to avoid blocking and properly clean up process
	go func() {
		_ = cmd.Run()
	}()
}

// handleWebSocket upgrades HTTP connections to WebSocket
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ServeWS(s.wsHub, s.allowedOrigins, w, r)
}

// Broadcast sends an event to all connected WebSocket clients
func (s *Server) Broadcast(eventType string, payload interface{}) {
	if s.wsHub != nil {
		s.wsHub.Broadcast(eventType, payload)
	}
}

// WSHub returns the WebSocket hub for external use
func (s *Server) WSHub() *WSHub {
	return s.wsHub
}

// WSClientCount returns the number of connected WebSocket clients
func (s *Server) WSClientCount() int {
	if s.wsHub != nil {
		return s.wsHub.ClientCount()
	}
	return 0
}

// spaHandler creates an HTTP handler that serves static files from the given
// filesystem, with SPA fallback to index.html for routes that don't match files.
func spaHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Try to serve the file directly
		f, err := fsys.Open(path[1:]) // Remove leading slash
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// File not found - serve index.html for SPA routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
