package contracts

import (
	"regexp"
	"strings"
)

// EndpointMatch represents a match between a frontend call and backend endpoint.
type EndpointMatch struct {
	BackendEndpoint string  // Normalized backend endpoint path
	FrontendURL     string  // Frontend URL/pattern
	Method          string  // HTTP method
	Confidence      float64 // Match confidence (0.0 - 1.0)
	PathParams      map[string]string // Matched path parameters
}

// Matcher matches frontend API calls to backend endpoints.
type Matcher struct {
	endpoints []MatchableEndpoint
}

type MatchableEndpoint struct {
	Method  string
	Path    string
	Pattern *regexp.Regexp
	Params  []string
}

// NewMatcher creates a new endpoint matcher.
func NewMatcher() *Matcher {
	return &Matcher{
		endpoints: make([]MatchableEndpoint, 0),
	}
}

// AddEndpoint adds a backend endpoint to the matcher.
func (m *Matcher) AddEndpoint(method, path string) {
	// Normalize path before creating pattern
	normalizedPath := NormalizePath(path)
	pattern := PathToPattern(normalizedPath)
	re, err := regexp.Compile(pattern)
	if err != nil {
		return
	}

	m.endpoints = append(m.endpoints, MatchableEndpoint{
		Method:  strings.ToUpper(method),
		Path:    normalizedPath,
		Pattern: re,
		Params:  ExtractPathParams(normalizedPath),
	})
}

// Match attempts to match a frontend URL to a backend endpoint.
func (m *Matcher) Match(method, url string) *EndpointMatch {
	method = strings.ToUpper(method)
	url = NormalizePath(url)

	var bestMatch *EndpointMatch
	var bestConfidence float64

	for _, ep := range m.endpoints {
		// Check method match
		methodMatches := m.methodMatches(method, ep.Method)
		if !methodMatches {
			continue
		}

		// Check path match
		if ep.Pattern.MatchString(url) {
			confidence := m.calculateConfidence(url, ep)
			if confidence > bestConfidence {
				bestConfidence = confidence
				bestMatch = &EndpointMatch{
					BackendEndpoint: ep.Path,
					FrontendURL:     url,
					Method:          method,
					Confidence:      confidence,
					PathParams:      m.extractMatchedParams(url, ep),
				}
			}
		}
	}

	return bestMatch
}

// MatchAll returns all matching endpoints for a frontend URL.
func (m *Matcher) MatchAll(method, url string) []*EndpointMatch {
	method = strings.ToUpper(method)
	url = NormalizePath(url)

	var matches []*EndpointMatch

	for _, ep := range m.endpoints {
		if !m.methodMatches(method, ep.Method) {
			continue
		}

		if ep.Pattern.MatchString(url) {
			matches = append(matches, &EndpointMatch{
				BackendEndpoint: ep.Path,
				FrontendURL:     url,
				Method:          method,
				Confidence:      m.calculateConfidence(url, ep),
				PathParams:      m.extractMatchedParams(url, ep),
			})
		}
	}

	return matches
}

func (m *Matcher) methodMatches(frontendMethod, backendMethod string) bool {
	// Exact match
	if frontendMethod == backendMethod {
		return true
	}

	// ANY matches everything
	if frontendMethod == "ANY" || backendMethod == "ANY" {
		return true
	}

	// USE (express mount) matches everything
	if backendMethod == "USE" {
		return true
	}

	return false
}

func (m *Matcher) calculateConfidence(url string, ep MatchableEndpoint) float64 {
	// Base confidence from pattern match
	confidence := 0.8

	// Exact match (no params) gets higher confidence
	if url == ep.Path {
		confidence = 1.0
	}

	// More path segments matched = higher confidence
	urlSegments := strings.Split(url, "/")
	pathSegments := strings.Split(ep.Path, "/")

	// Penalize for different segment counts
	if len(urlSegments) != len(pathSegments) {
		confidence -= 0.1
	}

	// Bonus for matching static segments
	matchedStatic := 0
	totalStatic := 0
	for i, seg := range pathSegments {
		if !strings.HasPrefix(seg, ":") && !strings.HasPrefix(seg, "{") {
			totalStatic++
			if i < len(urlSegments) && urlSegments[i] == seg {
				matchedStatic++
			}
		}
	}
	if totalStatic > 0 {
		confidence += 0.1 * float64(matchedStatic) / float64(totalStatic)
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

func (m *Matcher) extractMatchedParams(url string, ep MatchableEndpoint) map[string]string {
	params := make(map[string]string)

	urlSegments := strings.Split(url, "/")
	pathSegments := strings.Split(ep.Path, "/")

	paramIndex := 0
	for i, seg := range pathSegments {
		if strings.HasPrefix(seg, ":") || strings.HasPrefix(seg, "{") {
			if i < len(urlSegments) && paramIndex < len(ep.Params) {
				params[ep.Params[paramIndex]] = urlSegments[i]
				paramIndex++
			}
		}
	}

	return params
}

// FindUnmatchedEndpoints returns backend endpoints with no matching frontend calls.
func (m *Matcher) FindUnmatchedEndpoints(calls []struct{ Method, URL string }) []MatchableEndpoint {
	matchedPaths := make(map[string]bool)

	for _, call := range calls {
		match := m.Match(call.Method, call.URL)
		if match != nil {
			key := match.Method + ":" + match.BackendEndpoint
			matchedPaths[key] = true
		}
	}

	var unmatched []MatchableEndpoint
	for _, ep := range m.endpoints {
		key := ep.Method + ":" + ep.Path
		if !matchedPaths[key] {
			unmatched = append(unmatched, ep)
		}
	}

	return unmatched
}

// FindUnmatchedCalls returns frontend calls with no matching backend endpoint.
func (m *Matcher) FindUnmatchedCalls(calls []struct{ Method, URL string }) []struct{ Method, URL string } {
	var unmatched []struct{ Method, URL string }

	for _, call := range calls {
		match := m.Match(call.Method, call.URL)
		if match == nil {
			unmatched = append(unmatched, call)
		}
	}

	return unmatched
}
