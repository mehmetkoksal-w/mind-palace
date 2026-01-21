package extractors

import (
	"testing"
)

func TestFetchExtractor_BasicCalls(t *testing.T) {
	code := []byte(`
async function fetchUsers() {
    const response = await fetch('/api/users');
    return response.json();
}

async function createUser(user) {
    const response = await fetch('/api/users', {
        method: 'POST',
        body: JSON.stringify(user)
    });
    return response.json();
}
`)

	extractor := NewFetchExtractor()
	calls, err := extractor.ExtractCallsFromContent(code, "api.ts")
	if err != nil {
		t.Fatalf("failed to extract calls: %v", err)
	}

	if len(calls) != 2 {
		t.Errorf("expected 2 calls, got %d", len(calls))
		for _, c := range calls {
			t.Logf("  - %s %s", c.Method, c.URL)
		}
		return
	}

	// Check first call (GET)
	if calls[0].Method != "GET" {
		t.Errorf("expected first call method GET, got %s", calls[0].Method)
	}
	if calls[0].URL != "/api/users" {
		t.Errorf("expected first call URL /api/users, got %s", calls[0].URL)
	}

	// Check second call (POST)
	if calls[1].Method != "POST" {
		t.Errorf("expected second call method POST, got %s", calls[1].Method)
	}
}

func TestFetchExtractor_AllMethods(t *testing.T) {
	code := []byte(`
fetch('/api/data', { method: 'GET' });
fetch('/api/data', { method: 'POST' });
fetch('/api/data', { method: 'PUT' });
fetch('/api/data', { method: 'DELETE' });
fetch('/api/data', { method: 'PATCH' });
`)

	extractor := NewFetchExtractor()
	calls, err := extractor.ExtractCallsFromContent(code, "api.ts")
	if err != nil {
		t.Fatalf("failed to extract calls: %v", err)
	}

	if len(calls) != 5 {
		t.Errorf("expected 5 calls, got %d", len(calls))
		return
	}

	expectedMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for i, expected := range expectedMethods {
		if calls[i].Method != expected {
			t.Errorf("call %d: expected method %s, got %s", i, expected, calls[i].Method)
		}
	}
}

func TestFetchExtractor_TemplateLiteral(t *testing.T) {
	code := []byte("fetch(`/api/users/${userId}`);")

	extractor := NewFetchExtractor()
	calls, err := extractor.ExtractCallsFromContent(code, "api.ts")
	if err != nil {
		t.Fatalf("failed to extract calls: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}

	call := calls[0]
	if !call.IsDynamic {
		t.Error("expected call to be dynamic")
	}
	if len(call.Variables) == 0 || call.Variables[0] != "userId" {
		t.Errorf("expected variable 'userId', got %v", call.Variables)
	}
}

func TestFetchExtractor_TemplateLiteralMultipleVars(t *testing.T) {
	code := []byte("fetch(`${baseUrl}/users/${userId}/posts/${postId}`);")

	extractor := NewFetchExtractor()
	calls, err := extractor.ExtractCallsFromContent(code, "api.ts")
	if err != nil {
		t.Fatalf("failed to extract calls: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}

	call := calls[0]
	if !call.IsDynamic {
		t.Error("expected call to be dynamic")
	}
	if len(call.Variables) != 3 {
		t.Errorf("expected 3 variables, got %d: %v", len(call.Variables), call.Variables)
	}
}

func TestFetchExtractor_StringLiterals(t *testing.T) {
	code := []byte(`
fetch("/api/users");
fetch('/api/posts');
`)

	extractor := NewFetchExtractor()
	calls, err := extractor.ExtractCallsFromContent(code, "api.ts")
	if err != nil {
		t.Fatalf("failed to extract calls: %v", err)
	}

	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}

	if calls[0].URL != "/api/users" {
		t.Errorf("expected /api/users, got %s", calls[0].URL)
	}
	if calls[1].URL != "/api/posts" {
		t.Errorf("expected /api/posts, got %s", calls[1].URL)
	}
}

func TestFetchExtractor_DefaultMethod(t *testing.T) {
	code := []byte(`fetch('/api/data');`)

	extractor := NewFetchExtractor()
	calls, err := extractor.ExtractCallsFromContent(code, "api.ts")
	if err != nil {
		t.Fatalf("failed to extract calls: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}

	if calls[0].Method != "GET" {
		t.Errorf("expected default method GET, got %s", calls[0].Method)
	}
}

func TestFetchExtractor_LineNumbers(t *testing.T) {
	code := []byte(`const a = 1;

fetch('/first');

const b = 2;

fetch('/second');
`)

	extractor := NewFetchExtractor()
	calls, err := extractor.ExtractCallsFromContent(code, "api.ts")
	if err != nil {
		t.Fatalf("failed to extract calls: %v", err)
	}

	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}

	// Line numbers should be different and > 0
	if calls[0].Line <= 0 {
		t.Errorf("expected positive line number, got %d", calls[0].Line)
	}
	if calls[1].Line <= calls[0].Line {
		t.Errorf("expected second call on later line: %d vs %d", calls[0].Line, calls[1].Line)
	}
}

func TestFetchExtractor_AsyncAwait(t *testing.T) {
	code := []byte(`
async function getData() {
    const response = await fetch('/api/data');
    const json = await response.json();
    return json;
}
`)

	extractor := NewFetchExtractor()
	calls, err := extractor.ExtractCallsFromContent(code, "api.ts")
	if err != nil {
		t.Fatalf("failed to extract calls: %v", err)
	}

	if len(calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(calls))
	}
}

func TestFetchExtractor_PromiseThen(t *testing.T) {
	code := []byte(`
fetch('/api/data')
    .then(response => response.json())
    .then(data => console.log(data));
`)

	extractor := NewFetchExtractor()
	calls, err := extractor.ExtractCallsFromContent(code, "api.ts")
	if err != nil {
		t.Fatalf("failed to extract calls: %v", err)
	}

	if len(calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(calls))
	}
}

func TestFetchExtractor_WrapperFunction(t *testing.T) {
	code := []byte(`
api.fetch('/api/users');
this.fetch('/api/posts');
http.fetch('/api/data');
`)

	extractor := NewFetchExtractor()
	calls, err := extractor.ExtractCallsFromContent(code, "api.ts")
	if err != nil {
		t.Fatalf("failed to extract calls: %v", err)
	}

	if len(calls) != 3 {
		t.Errorf("expected 3 calls, got %d", len(calls))
		for _, c := range calls {
			t.Logf("  - %s %s", c.Method, c.URL)
		}
	}
}

func TestFetchExtractor_ComplexOptions(t *testing.T) {
	code := []byte(`
fetch('/api/users', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer token'
    },
    body: JSON.stringify({ name: 'test' })
});
`)

	extractor := NewFetchExtractor()
	calls, err := extractor.ExtractCallsFromContent(code, "api.ts")
	if err != nil {
		t.Fatalf("failed to extract calls: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}

	if calls[0].Method != "POST" {
		t.Errorf("expected POST method, got %s", calls[0].Method)
	}
}

func TestFetchExtractor_MethodLowerCase(t *testing.T) {
	code := []byte(`fetch('/api/data', { method: 'post' });`)

	extractor := NewFetchExtractor()
	calls, err := extractor.ExtractCallsFromContent(code, "api.ts")
	if err != nil {
		t.Fatalf("failed to extract calls: %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}

	if calls[0].Method != "POST" {
		t.Errorf("expected POST (uppercase), got %s", calls[0].Method)
	}
}
