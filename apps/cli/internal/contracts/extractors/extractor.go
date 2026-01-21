// Package extractors provides language-specific endpoint and API call extractors.
package extractors

import (
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/analysis"
	"github.com/mehmetkoksal-w/mind-palace/apps/cli/internal/contracts"
)

// EndpointExtractor extracts backend API endpoints from source files.
type EndpointExtractor interface {
	// ID returns the unique identifier for this extractor.
	ID() string

	// Framework returns the framework name (e.g., "gin", "express", "fastapi").
	Framework() string

	// Languages returns the languages this extractor supports.
	Languages() []string

	// CanExtract returns true if this extractor can handle the given file.
	CanExtract(file *analysis.FileAnalysis) bool

	// ExtractEndpoints extracts API endpoints from a parsed file.
	ExtractEndpoints(file *analysis.FileAnalysis) ([]ExtractedEndpoint, error)
}

// APICallExtractor extracts frontend API calls from source files.
type APICallExtractor interface {
	// ID returns the unique identifier for this extractor.
	ID() string

	// CallType returns the type of calls extracted (e.g., "fetch", "axios").
	CallType() string

	// Languages returns the languages this extractor supports.
	Languages() []string

	// CanExtract returns true if this extractor can handle the given file.
	CanExtract(file *analysis.FileAnalysis) bool

	// ExtractCalls extracts API calls from a parsed file.
	ExtractCalls(file *analysis.FileAnalysis) ([]ExtractedCall, error)
}

// ExtractedEndpoint represents an API endpoint extracted from backend code.
type ExtractedEndpoint struct {
	Method         string              // HTTP method: GET, POST, etc.
	Path           string              // Route path: /api/users/:id
	PathParams     []string            // Path parameters: ["id"]
	Handler        string              // Handler function name
	File           string              // Source file path
	Line           int                 // Line number
	Framework      string              // Framework: net/http, gin, echo, express, fastapi
	RequestSchema  *contracts.TypeSchema // Request body schema (for POST, PUT, PATCH)
	ResponseSchema *contracts.TypeSchema // Response body schema
}

// ExtractedCall represents an API call extracted from frontend code.
type ExtractedCall struct {
	Method         string              // HTTP method (may be dynamic)
	URL            string              // URL or URL pattern
	File           string              // Source file path
	Line           int                 // Line number
	ExpectedSchema *contracts.TypeSchema // Expected response type (if inferrable)
	IsDynamic      bool                // True if URL contains variables
	Variables      []string            // Variable names in URL
}

// Registry holds registered extractors.
type Registry struct {
	endpointExtractors []EndpointExtractor
	callExtractors     []APICallExtractor
}

// NewRegistry creates a new extractor registry.
func NewRegistry() *Registry {
	return &Registry{
		endpointExtractors: make([]EndpointExtractor, 0),
		callExtractors:     make([]APICallExtractor, 0),
	}
}

// RegisterEndpointExtractor registers an endpoint extractor.
func (r *Registry) RegisterEndpointExtractor(e EndpointExtractor) {
	r.endpointExtractors = append(r.endpointExtractors, e)
}

// RegisterCallExtractor registers an API call extractor.
func (r *Registry) RegisterCallExtractor(e APICallExtractor) {
	r.callExtractors = append(r.callExtractors, e)
}

// GetEndpointExtractors returns all endpoint extractors that can handle a file.
func (r *Registry) GetEndpointExtractors(file *analysis.FileAnalysis) []EndpointExtractor {
	var result []EndpointExtractor
	for _, e := range r.endpointExtractors {
		if e.CanExtract(file) {
			result = append(result, e)
		}
	}
	return result
}

// GetCallExtractors returns all call extractors that can handle a file.
func (r *Registry) GetCallExtractors(file *analysis.FileAnalysis) []APICallExtractor {
	var result []APICallExtractor
	for _, e := range r.callExtractors {
		if e.CanExtract(file) {
			result = append(result, e)
		}
	}
	return result
}

// AllEndpointExtractors returns all registered endpoint extractors.
func (r *Registry) AllEndpointExtractors() []EndpointExtractor {
	return r.endpointExtractors
}

// AllCallExtractors returns all registered call extractors.
func (r *Registry) AllCallExtractors() []APICallExtractor {
	return r.callExtractors
}

// DefaultRegistry is the global default registry.
var DefaultRegistry = NewRegistry()

// RegisterEndpointExtractor registers an extractor to the default registry.
func RegisterEndpointExtractor(e EndpointExtractor) {
	DefaultRegistry.RegisterEndpointExtractor(e)
}

// RegisterCallExtractor registers a call extractor to the default registry.
func RegisterCallExtractor(e APICallExtractor) {
	DefaultRegistry.RegisterCallExtractor(e)
}
