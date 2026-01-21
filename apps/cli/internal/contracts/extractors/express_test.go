package extractors

import (
	"testing"
)

func TestExpressExtractor_BasicRoutes(t *testing.T) {
	code := []byte(`
import express from 'express';

const app = express();

app.get('/api/users', listUsers);
app.post('/api/users', createUser);
app.get('/api/users/:id', getUser);
app.put('/api/users/:id', updateUser);
app.delete('/api/users/:id', deleteUser);

function listUsers(req, res) {}
function createUser(req, res) {}
function getUser(req, res) {}
function updateUser(req, res) {}
function deleteUser(req, res) {}
`)

	extractor := NewExpressExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "routes.ts")
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

func TestExpressExtractor_Router(t *testing.T) {
	code := []byte(`
import { Router } from 'express';

const router = Router();

router.get('/', listPosts);
router.get('/:id', getPost);
router.post('/', createPost);

export default router;
`)

	extractor := NewExpressExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "posts.ts")
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
}

func TestExpressExtractor_PathParams(t *testing.T) {
	code := []byte(`
const app = express();
app.get('/api/users/:userId/posts/:postId', getPost);
`)

	extractor := NewExpressExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "routes.ts")
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

	if ep.PathParams[0] != "userId" {
		t.Errorf("expected first param 'userId', got %s", ep.PathParams[0])
	}
	if ep.PathParams[1] != "postId" {
		t.Errorf("expected second param 'postId', got %s", ep.PathParams[1])
	}
}

func TestExpressExtractor_InlineHandler(t *testing.T) {
	code := []byte(`
const app = express();

app.get('/api/health', (req, res) => {
    res.json({ status: 'ok' });
});

app.post('/api/data', async (req, res) => {
    const data = await processData(req.body);
    res.json(data);
});
`)

	extractor := NewExpressExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "app.ts")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) != 2 {
		t.Errorf("expected 2 endpoints, got %d", len(endpoints))
		for _, ep := range endpoints {
			t.Logf("  - %s %s", ep.Method, ep.Path)
		}
	}
}

func TestExpressExtractor_Middleware(t *testing.T) {
	code := []byte(`
const app = express();

// Route with middleware
app.get('/api/protected', authMiddleware, protectedHandler);
app.post('/api/upload', uploadMiddleware, processUpload, handleUpload);
`)

	extractor := NewExpressExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "routes.ts")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) != 2 {
		t.Errorf("expected 2 endpoints, got %d", len(endpoints))
		for _, ep := range endpoints {
			t.Logf("  - %s %s (handler: %s)", ep.Method, ep.Path, ep.Handler)
		}
	}
}

func TestExpressExtractor_UseMount(t *testing.T) {
	code := []byte(`
const app = express();
const apiRouter = require('./api');
const userRouter = require('./users');

app.use('/api', apiRouter);
app.use('/api/users', userRouter);
`)

	extractor := NewExpressExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "app.ts")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) != 2 {
		t.Errorf("expected 2 mount points, got %d", len(endpoints))
		for _, ep := range endpoints {
			t.Logf("  - %s %s (handler: %s)", ep.Method, ep.Path, ep.Handler)
		}
		return
	}

	// Check that use() is detected as USE method
	for _, ep := range endpoints {
		if ep.Method != "USE" {
			t.Errorf("expected USE method for mount point, got %s", ep.Method)
		}
	}
}

func TestExpressExtractor_AllMethod(t *testing.T) {
	code := []byte(`
const app = express();
app.all('/api/*', corsHandler);
`)

	extractor := NewExpressExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "app.ts")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) != 1 {
		t.Errorf("expected 1 endpoint, got %d", len(endpoints))
		return
	}

	if endpoints[0].Method != "ANY" {
		t.Errorf("expected ANY method for all(), got %s", endpoints[0].Method)
	}
}

func TestExpressExtractor_Framework(t *testing.T) {
	code := []byte(`
const app = express();
app.get('/test', handler);
`)

	extractor := NewExpressExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "app.ts")
	if err != nil {
		t.Fatalf("failed to extract endpoints: %v", err)
	}

	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}

	if endpoints[0].Framework != "express" {
		t.Errorf("expected framework 'express', got %s", endpoints[0].Framework)
	}
}

func TestExpressExtractor_LineNumbers(t *testing.T) {
	code := []byte(`const app = express();

app.get('/first', handler1);

app.get('/second', handler2);
`)

	extractor := NewExpressExtractor()
	endpoints, err := extractor.ExtractEndpointsFromContent(code, "app.ts")
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
