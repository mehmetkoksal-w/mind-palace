package extractors

import (
	"testing"
)

func TestFastAPIExtractor_BasicRoutes(t *testing.T) {
	code := []byte(`
from fastapi import FastAPI

app = FastAPI()

@app.get("/api/users")
def list_users():
    return []

@app.post("/api/users")
def create_user(user: UserCreate):
    return user

@app.get("/api/users/{user_id}")
def get_user(user_id: int):
    return {"id": user_id}

@app.put("/api/users/{user_id}")
def update_user(user_id: int, user: UserUpdate):
    return user

@app.delete("/api/users/{user_id}")
def delete_user(user_id: int):
    return {"deleted": True}
`)

	extractor := NewFastAPIExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "main.py")
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

func TestFastAPIExtractor_Router(t *testing.T) {
	code := []byte(`
from fastapi import APIRouter

router = APIRouter()

@router.get("/")
def list_items():
    return []

@router.get("/{item_id}")
def get_item(item_id: int):
    return {}

@router.post("/")
def create_item(item: ItemCreate):
    return item
`)

	extractor := NewFastAPIExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "items.py")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) != 3 {
		t.Errorf("expected 3 endpoints, got %d", len(endpoints))
		for _, ep := range endpoints {
			t.Logf("  - %s %s (handler: %s)", ep.Method, ep.Path, ep.Handler)
		}
	}
}

func TestFastAPIExtractor_PathParams(t *testing.T) {
	code := []byte(`
from fastapi import FastAPI

app = FastAPI()

@app.get("/users/{user_id}/posts/{post_id}")
def get_user_post(user_id: int, post_id: int):
    return {}
`)

	extractor := NewFastAPIExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "main.py")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	ep := endpoints[0]
	if len(ep.PathParams) != 2 {
		t.Errorf("expected 2 path params, got %d: %v", len(ep.PathParams), ep.PathParams)
		return
	}

	if ep.PathParams[0] != "user_id" {
		t.Errorf("expected first param 'user_id', got %s", ep.PathParams[0])
	}
	if ep.PathParams[1] != "post_id" {
		t.Errorf("expected second param 'post_id', got %s", ep.PathParams[1])
	}
}

func TestFastAPIExtractor_TypedPathParams(t *testing.T) {
	code := []byte(`
from fastapi import FastAPI

app = FastAPI()

@app.get("/items/{item_id:int}")
def get_item(item_id: int):
    return {}

@app.get("/files/{file_path:path}")
def get_file(file_path: str):
    return {}
`)

	extractor := NewFastAPIExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "main.py")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) != 2 {
		t.Errorf("expected 2 endpoints, got %d", len(endpoints))
		return
	}

	// Check that typed params are extracted correctly
	for _, ep := range endpoints {
		if len(ep.PathParams) != 1 {
			t.Errorf("expected 1 path param for %s, got %d", ep.Path, len(ep.PathParams))
		}
	}
}

func TestFastAPIExtractor_AsyncRoutes(t *testing.T) {
	code := []byte(`
from fastapi import FastAPI

app = FastAPI()

@app.get("/async")
async def async_endpoint():
    return {"async": True}

@app.post("/async")
async def async_post():
    return {}
`)

	extractor := NewFastAPIExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "main.py")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) != 2 {
		t.Errorf("expected 2 endpoints, got %d", len(endpoints))
	}
}

func TestFastAPIExtractor_NamedPathArg(t *testing.T) {
	code := []byte(`
from fastapi import FastAPI

app = FastAPI()

@app.get(path="/explicit")
def explicit_path():
    return {}
`)

	extractor := NewFastAPIExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "main.py")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	if endpoints[0].Path != "/explicit" {
		t.Errorf("expected path '/explicit', got %s", endpoints[0].Path)
	}
}

func TestFastAPIExtractor_Framework(t *testing.T) {
	code := []byte(`
from fastapi import FastAPI

app = FastAPI()

@app.get("/test")
def test():
    return {}
`)

	extractor := NewFastAPIExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "main.py")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	if endpoints[0].Framework != "fastapi" {
		t.Errorf("expected framework 'fastapi', got %s", endpoints[0].Framework)
	}
}

func TestFastAPIExtractor_LineNumbers(t *testing.T) {
	code := []byte(`from fastapi import FastAPI

app = FastAPI()

@app.get("/first")
def first():
    return {}

@app.get("/second")
def second():
    return {}
`)

	extractor := NewFastAPIExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "main.py")
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

func TestFastAPIExtractor_HandlerNames(t *testing.T) {
	code := []byte(`
from fastapi import FastAPI

app = FastAPI()

@app.get("/users")
def list_users():
    return []

@app.post("/users")
def create_user():
    return {}
`)

	extractor := NewFastAPIExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "main.py")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(endpoints))
	}

	handlers := make(map[string]bool)
	for _, ep := range endpoints {
		handlers[ep.Handler] = true
	}

	if !handlers["list_users"] {
		t.Error("expected handler 'list_users' not found")
	}
	if !handlers["create_user"] {
		t.Error("expected handler 'create_user' not found")
	}
}
