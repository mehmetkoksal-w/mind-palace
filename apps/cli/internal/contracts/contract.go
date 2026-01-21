package contracts

import (
	"regexp"
	"strings"
	"time"
)

// Contract represents a FE-BE API contract.
type Contract struct {
	ID              string          `json:"id"`
	Method          string          `json:"method"`           // GET, POST, PUT, DELETE, PATCH
	Endpoint        string          `json:"endpoint"`         // Normalized path: /api/users/:id
	EndpointPattern string          `json:"endpoint_pattern"` // Regex for matching: /api/users/[^/]+

	Backend       BackendEndpoint `json:"backend"`
	FrontendCalls []FrontendCall  `json:"frontend_calls"`
	Mismatches    []FieldMismatch `json:"mismatches"`

	Status     ContractStatus `json:"status"`
	Authority  string         `json:"authority"`
	Confidence float64        `json:"confidence"`

	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BackendEndpoint represents an API endpoint extracted from backend code.
type BackendEndpoint struct {
	File           string      `json:"file"`
	Line           int         `json:"line"`
	Framework      string      `json:"framework"` // go-http, gin, echo, express, fastapi
	Handler        string      `json:"handler"`
	RequestSchema  *TypeSchema `json:"request_schema,omitempty"`
	ResponseSchema *TypeSchema `json:"response_schema,omitempty"`
}

// FrontendCall represents an API call from frontend code.
type FrontendCall struct {
	ID             string      `json:"id"`
	ContractID     string      `json:"contract_id"`
	File           string      `json:"file"`
	Line           int         `json:"line"`
	CallType       string      `json:"call_type"` // fetch, axios, custom
	ExpectedSchema *TypeSchema `json:"expected_schema,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
}

// ContractStatus represents the status of a contract.
type ContractStatus string

const (
	ContractDiscovered ContractStatus = "discovered"
	ContractVerified   ContractStatus = "verified"
	ContractMismatch   ContractStatus = "mismatch"
	ContractIgnored    ContractStatus = "ignored"
)

// HTTPMethod constants
const (
	MethodGET    = "GET"
	MethodPOST   = "POST"
	MethodPUT    = "PUT"
	MethodPATCH  = "PATCH"
	MethodDELETE = "DELETE"
)

// ValidMethods returns all valid HTTP methods.
func ValidMethods() []string {
	return []string{MethodGET, MethodPOST, MethodPUT, MethodPATCH, MethodDELETE}
}

// IsValidMethod checks if a method is valid.
func IsValidMethod(method string) bool {
	method = strings.ToUpper(method)
	for _, m := range ValidMethods() {
		if m == method {
			return true
		}
	}
	return false
}

// NormalizePath normalizes a path by converting path parameters to a standard format.
// Examples:
//   - /api/users/:id -> /api/users/:id (Express/Go style)
//   - /api/users/{id} -> /api/users/:id (FastAPI/OpenAPI style)
//   - /api/users/<id> -> /api/users/:id (Flask style)
func NormalizePath(path string) string {
	// Convert {param} to :param
	curlyRe := regexp.MustCompile(`\{([^}]+)\}`)
	path = curlyRe.ReplaceAllString(path, ":$1")

	// Convert <param> to :param
	angleRe := regexp.MustCompile(`<([^>]+)>`)
	path = angleRe.ReplaceAllString(path, ":$1")

	// Ensure leading slash
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Remove trailing slash (except for root)
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	return path
}

// PathToPattern converts a normalized path to a regex pattern.
// Supports both :id (gin/echo) and {id} (gorilla/mux) formats.
// Example: /api/users/:id -> ^/api/users/[^/]+$
// Example: /api/users/{id} -> ^/api/users/[^/]+$
func PathToPattern(path string) string {
	// First, replace :param and {param} placeholders with a marker
	// that won't be affected by QuoteMeta
	const colonPlaceholder = "<<<PARAM>>>"
	const bracePlaceholder = "<<<BRACE_PARAM>>>"

	// Replace :param (colon is not a special regex char, but we need to handle it first)
	colonRe := regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
	pattern := colonRe.ReplaceAllString(path, colonPlaceholder)

	// Replace {param}
	braceRe := regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)
	pattern = braceRe.ReplaceAllString(pattern, bracePlaceholder)

	// Escape any remaining regex special characters
	pattern = regexp.QuoteMeta(pattern)

	// Replace placeholders with [^/]+ pattern
	pattern = strings.ReplaceAll(pattern, colonPlaceholder, `[^/]+`)
	pattern = strings.ReplaceAll(pattern, bracePlaceholder, `[^/]+`)

	return "^" + pattern + "$"
}

// ExtractPathParams extracts parameter names from a path.
// Supports both :id (gin/echo) and {id} (gorilla/mux) formats.
// Example: /api/users/:id/posts/:postId -> ["id", "postId"]
// Example: /api/users/{id}/posts/{postId} -> ["id", "postId"]
func ExtractPathParams(path string) []string {
	// Match both :param and {param} formats
	colonRe := regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
	braceRe := regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

	var params []string

	// Extract :param style
	colonMatches := colonRe.FindAllStringSubmatch(path, -1)
	for _, match := range colonMatches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	// Extract {param} style
	braceMatches := braceRe.FindAllStringSubmatch(path, -1)
	for _, match := range braceMatches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	return params
}

// MatchPath checks if a URL matches a path pattern.
func MatchPath(url, pattern string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(url)
}

// HasMismatches returns true if the contract has any mismatches.
func (c *Contract) HasMismatches() bool {
	return len(c.Mismatches) > 0
}

// MismatchCount returns the number of mismatches.
func (c *Contract) MismatchCount() int {
	return len(c.Mismatches)
}

// FrontendCallCount returns the number of frontend calls.
func (c *Contract) FrontendCallCount() int {
	return len(c.FrontendCalls)
}

// ErrorCount returns the number of error-severity mismatches.
func (c *Contract) ErrorCount() int {
	count := 0
	for _, m := range c.Mismatches {
		if m.Severity == SeverityError {
			count++
		}
	}
	return count
}

// WarningCount returns the number of warning-severity mismatches.
func (c *Contract) WarningCount() int {
	count := 0
	for _, m := range c.Mismatches {
		if m.Severity == SeverityWarning {
			count++
		}
	}
	return count
}

// UpdateMismatches updates the contract's mismatches based on comparing
// the backend response schema with frontend expected schemas.
func (c *Contract) UpdateMismatches() {
	c.Mismatches = nil

	if c.Backend.ResponseSchema == nil {
		return
	}

	// Compare with each frontend call's expected schema
	for _, call := range c.FrontendCalls {
		if call.ExpectedSchema == nil {
			continue
		}

		mismatches := c.Backend.ResponseSchema.Compare(call.ExpectedSchema, "$")
		for i := range mismatches {
			mismatches[i].ID = GenerateID("mis")
		}
		c.Mismatches = append(c.Mismatches, mismatches...)
	}

	// Update status based on mismatches
	if len(c.Mismatches) > 0 {
		c.Status = ContractMismatch
	}
}

// ContractFilters contains filter options for querying contracts.
type ContractFilters struct {
	Method        string
	Status        string
	HasMismatches *bool
	Endpoint      string // Partial match
	Limit         int
	Offset        int
}

// ContractStats contains aggregate statistics about contracts.
type ContractStats struct {
	Total         int            `json:"total"`
	Discovered    int            `json:"discovered"`
	Verified      int            `json:"verified"`
	Mismatch      int            `json:"mismatch"`
	Ignored       int            `json:"ignored"`
	ByMethod      map[string]int `json:"by_method"`
	TotalCalls    int            `json:"total_calls"`
	TotalErrors   int            `json:"total_errors"`
	TotalWarnings int            `json:"total_warnings"`
}
