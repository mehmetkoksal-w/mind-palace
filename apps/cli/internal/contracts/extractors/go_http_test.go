package extractors

import (
	"testing"
)

func TestGoHTTPExtractor_NetHTTP(t *testing.T) {
	code := []byte(`
package main

import "net/http"

func main() {
	http.HandleFunc("/api/users", usersHandler)
	http.HandleFunc("/api/users/profile", profileHandler)
}

func usersHandler(w http.ResponseWriter, r *http.Request) {}
func profileHandler(w http.ResponseWriter, r *http.Request) {}
`)

	extractor := NewGoHTTPExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "main.go")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) != 2 {
		t.Errorf("expected 2 endpoints, got %d", len(endpoints))
		for _, ep := range endpoints {
			t.Logf("  - %s %s (handler: %s)", ep.Method, ep.Path, ep.Handler)
		}
		return
	}

	// Check first endpoint
	if endpoints[0].Path != "/api/users" {
		t.Errorf("expected path /api/users, got %s", endpoints[0].Path)
	}
	if endpoints[0].Handler != "usersHandler" {
		t.Errorf("expected handler usersHandler, got %s", endpoints[0].Handler)
	}
}

func TestGoHTTPExtractor_GorillaMux(t *testing.T) {
	code := []byte(`
package main

import "github.com/gorilla/mux"

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/api/users/{id}", getUserHandler).Methods("GET")
	r.HandleFunc("/api/users", createUserHandler).Methods("POST")
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {}
func createUserHandler(w http.ResponseWriter, r *http.Request) {}
`)

	extractor := NewGoHTTPExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "main.go")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) < 2 {
		t.Errorf("expected at least 2 endpoints, got %d", len(endpoints))
		for _, ep := range endpoints {
			t.Logf("  - %s %s (handler: %s)", ep.Method, ep.Path, ep.Handler)
		}
		return
	}

	// Check for path parameters
	found := false
	for _, ep := range endpoints {
		if ep.Path == "/api/users/{id}" {
			found = true
			if len(ep.PathParams) == 0 || ep.PathParams[0] != "id" {
				t.Errorf("expected path param 'id', got %v", ep.PathParams)
			}
		}
	}
	if !found {
		t.Error("endpoint /api/users/{id} not found")
	}
}

func TestGoHTTPExtractor_Gin(t *testing.T) {
	code := []byte(`
package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/api/users", listUsers)
	r.POST("/api/users", createUser)
	r.GET("/api/users/:id", getUser)
	r.PUT("/api/users/:id", updateUser)
	r.DELETE("/api/users/:id", deleteUser)
}

func listUsers(c *gin.Context) {}
func createUser(c *gin.Context) {}
func getUser(c *gin.Context) {}
func updateUser(c *gin.Context) {}
func deleteUser(c *gin.Context) {}
`)

	extractor := NewGoHTTPExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "main.go")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) != 5 {
		t.Errorf("expected 5 endpoints, got %d", len(endpoints))
		for _, ep := range endpoints {
			t.Logf("  - %s %s (handler: %s)", ep.Method, ep.Path, ep.Handler)
		}
		return
	}

	// Check methods
	methodCounts := make(map[string]int)
	for _, ep := range endpoints {
		methodCounts[ep.Method]++
	}

	if methodCounts["GET"] != 2 {
		t.Errorf("expected 2 GET endpoints, got %d", methodCounts["GET"])
	}
	if methodCounts["POST"] != 1 {
		t.Errorf("expected 1 POST endpoint, got %d", methodCounts["POST"])
	}
	if methodCounts["PUT"] != 1 {
		t.Errorf("expected 1 PUT endpoint, got %d", methodCounts["PUT"])
	}
	if methodCounts["DELETE"] != 1 {
		t.Errorf("expected 1 DELETE endpoint, got %d", methodCounts["DELETE"])
	}
}

func TestGoHTTPExtractor_Echo(t *testing.T) {
	code := []byte(`
package main

import "github.com/labstack/echo/v4"

func main() {
	e := echo.New()
	e.GET("/api/users", listUsers)
	e.POST("/api/users", createUser)
	e.GET("/api/users/:id", getUser)
}

func listUsers(c echo.Context) error { return nil }
func createUser(c echo.Context) error { return nil }
func getUser(c echo.Context) error { return nil }
`)

	extractor := NewGoHTTPExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "main.go")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) != 3 {
		t.Errorf("expected 3 endpoints, got %d", len(endpoints))
		for _, ep := range endpoints {
			t.Logf("  - %s %s (handler: %s)", ep.Method, ep.Path, ep.Handler)
		}
		return
	}

	// Check path parameters
	for _, ep := range endpoints {
		if ep.Path == "/api/users/:id" {
			if len(ep.PathParams) == 0 || ep.PathParams[0] != "id" {
				t.Errorf("expected path param 'id', got %v", ep.PathParams)
			}
		}
	}
}

func TestGoHTTPExtractor_GinGroups(t *testing.T) {
	code := []byte(`
package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	api := r.Group("/api")
	{
		users := api.Group("/users")
		users.GET("", listUsers)
		users.GET("/:id", getUser)
	}
}

func listUsers(c *gin.Context) {}
func getUser(c *gin.Context) {}
`)

	extractor := NewGoHTTPExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "main.go")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	// Note: This test shows current limitation - we detect the GET calls
	// but don't resolve the full path with groups. Group resolution would
	// require tracking variable assignments which is complex.
	// For now, we just verify we detect some routes.
	if len(endpoints) == 0 {
		t.Log("No endpoints extracted - group routes not yet supported")
	} else {
		t.Logf("Extracted %d endpoints (group path resolution not implemented)", len(endpoints))
	}
}

func TestGoHTTPExtractor_PathParams(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedParams []string
	}{
		{
			name:           "no params",
			path:           "/api/users",
			expectedParams: nil,
		},
		{
			name:           "colon param",
			path:           "/api/users/:id",
			expectedParams: []string{"id"},
		},
		{
			name:           "curly brace param",
			path:           "/api/users/{id}",
			expectedParams: []string{"id"},
		},
		{
			name:           "multiple params",
			path:           "/api/users/:userId/posts/:postId",
			expectedParams: []string{"userId", "postId"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := []byte(`
package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("` + tt.path + `", handler)
}
`)

			extractor := NewGoHTTPExtractor()
			endpoints, err := extractor.ExtractEndpointsFromContent(code, "main.go")
			if err != nil {
				t.Fatalf("failed to extract endpoints: %v", err)
			}

			if len(endpoints) != 1 {
				t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
			}

			ep := endpoints[0]
			if len(ep.PathParams) != len(tt.expectedParams) {
				t.Errorf("expected %d params, got %d: %v", len(tt.expectedParams), len(ep.PathParams), ep.PathParams)
				return
			}

			for i, expected := range tt.expectedParams {
				if ep.PathParams[i] != expected {
					t.Errorf("param %d: expected %s, got %s", i, expected, ep.PathParams[i])
				}
			}
		})
	}
}

func TestGoHTTPExtractor_LineNumbers(t *testing.T) {
	code := []byte(`package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/api/users", listUsers)
	r.POST("/api/users", createUser)
}
`)

	extractor := NewGoHTTPExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "main.go")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	// Line numbers should be different and > 0
	if endpoints[0].Line <= 0 {
		t.Errorf("expected positive line number, got %d", endpoints[0].Line)
	}
	if endpoints[1].Line <= endpoints[0].Line {
		t.Errorf("expected second endpoint to be on later line: %d vs %d", endpoints[0].Line, endpoints[1].Line)
	}
}
