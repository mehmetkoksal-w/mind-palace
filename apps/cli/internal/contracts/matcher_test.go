package contracts

import (
	"testing"
)

func TestMatcher_ExactMatch(t *testing.T) {
	m := NewMatcher()
	m.AddEndpoint("GET", "/api/users")
	m.AddEndpoint("POST", "/api/users")
	m.AddEndpoint("GET", "/api/posts")

	// Test exact match
	match := m.Match("GET", "/api/users")
	if match == nil {
		t.Fatal("expected match for GET /api/users")
	}
	if match.BackendEndpoint != "/api/users" {
		t.Errorf("expected /api/users, got %s", match.BackendEndpoint)
	}
	if match.Method != "GET" {
		t.Errorf("expected GET, got %s", match.Method)
	}
	if match.Confidence < 0.9 {
		t.Errorf("expected high confidence for exact match, got %f", match.Confidence)
	}
}

func TestMatcher_PathParams(t *testing.T) {
	m := NewMatcher()
	m.AddEndpoint("GET", "/api/users/:id")
	m.AddEndpoint("GET", "/api/users/:userId/posts/:postId")

	// Test single param
	match := m.Match("GET", "/api/users/123")
	if match == nil {
		t.Fatal("expected match for GET /api/users/123")
	}
	if match.BackendEndpoint != "/api/users/:id" {
		t.Errorf("expected /api/users/:id, got %s", match.BackendEndpoint)
	}
	if match.PathParams["id"] != "123" {
		t.Errorf("expected param id=123, got %v", match.PathParams)
	}

	// Test multiple params
	match = m.Match("GET", "/api/users/456/posts/789")
	if match == nil {
		t.Fatal("expected match for GET /api/users/456/posts/789")
	}
	if match.PathParams["userId"] != "456" {
		t.Errorf("expected param userId=456, got %v", match.PathParams)
	}
	if match.PathParams["postId"] != "789" {
		t.Errorf("expected param postId=789, got %v", match.PathParams)
	}
}

func TestMatcher_CurlyBraceParams(t *testing.T) {
	m := NewMatcher()
	m.AddEndpoint("GET", "/api/users/{id}")

	match := m.Match("GET", "/api/users/123")
	if match == nil {
		t.Fatal("expected match for GET /api/users/123")
	}
	if match.PathParams["id"] != "123" {
		t.Errorf("expected param id=123, got %v", match.PathParams)
	}
}

func TestMatcher_NoMatch(t *testing.T) {
	m := NewMatcher()
	m.AddEndpoint("GET", "/api/users")

	// Wrong path
	match := m.Match("GET", "/api/posts")
	if match != nil {
		t.Errorf("expected no match for /api/posts, got %+v", match)
	}

	// Wrong method
	match = m.Match("POST", "/api/users")
	if match != nil {
		t.Errorf("expected no match for POST /api/users, got %+v", match)
	}
}

func TestMatcher_AnyMethod(t *testing.T) {
	m := NewMatcher()
	m.AddEndpoint("ANY", "/api/wildcard")

	// ANY should match any method
	for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH"} {
		match := m.Match(method, "/api/wildcard")
		if match == nil {
			t.Errorf("expected ANY endpoint to match %s", method)
		}
	}
}

func TestMatcher_MatchAll(t *testing.T) {
	m := NewMatcher()
	m.AddEndpoint("GET", "/api/users")
	m.AddEndpoint("GET", "/api/users/:id")

	// Match all should return both for overlapping patterns
	// (though /api/users won't match /api/users/123)
	matches := m.MatchAll("GET", "/api/users")
	if len(matches) != 1 {
		t.Errorf("expected 1 match for /api/users, got %d", len(matches))
	}

	matches = m.MatchAll("GET", "/api/users/123")
	if len(matches) != 1 {
		t.Errorf("expected 1 match for /api/users/123, got %d", len(matches))
	}
}

func TestMatcher_Confidence(t *testing.T) {
	m := NewMatcher()
	m.AddEndpoint("GET", "/api/users")
	m.AddEndpoint("GET", "/api/users/:id")

	// Exact match should have higher confidence
	exactMatch := m.Match("GET", "/api/users")
	paramMatch := m.Match("GET", "/api/users/123")

	if exactMatch == nil {
		t.Fatal("expected exact match to succeed")
	}
	if paramMatch == nil {
		t.Fatal("expected param match to succeed")
	}

	if exactMatch.Confidence < paramMatch.Confidence {
		t.Errorf("expected exact match to have higher or equal confidence: %f vs %f",
			exactMatch.Confidence, paramMatch.Confidence)
	}
}

func TestMatcher_NormalizePath(t *testing.T) {
	m := NewMatcher()
	m.AddEndpoint("GET", "api/users/") // No leading slash, trailing slash

	// Should still match normalized path
	match := m.Match("GET", "/api/users")
	if match == nil {
		t.Fatal("expected match with path normalization")
	}
}

func TestMatcher_CaseInsensitiveMethod(t *testing.T) {
	m := NewMatcher()
	m.AddEndpoint("get", "/api/users") // lowercase method

	match := m.Match("GET", "/api/users")
	if match == nil {
		t.Fatal("expected match with case-insensitive method")
	}

	match = m.Match("get", "/api/users")
	if match == nil {
		t.Fatal("expected match with lowercase method")
	}
}

func TestMatcher_FindUnmatchedEndpoints(t *testing.T) {
	m := NewMatcher()
	m.AddEndpoint("GET", "/api/users")
	m.AddEndpoint("POST", "/api/users")
	m.AddEndpoint("GET", "/api/posts")

	calls := []struct{ Method, URL string }{
		{"GET", "/api/users"},
	}

	unmatched := m.FindUnmatchedEndpoints(calls)
	if len(unmatched) != 2 {
		t.Errorf("expected 2 unmatched endpoints, got %d", len(unmatched))
	}
}

func TestMatcher_FindUnmatchedCalls(t *testing.T) {
	m := NewMatcher()
	m.AddEndpoint("GET", "/api/users")

	calls := []struct{ Method, URL string }{
		{"GET", "/api/users"},
		{"GET", "/api/unknown"},
		{"POST", "/api/other"},
	}

	unmatched := m.FindUnmatchedCalls(calls)
	if len(unmatched) != 2 {
		t.Errorf("expected 2 unmatched calls, got %d", len(unmatched))
	}
}
