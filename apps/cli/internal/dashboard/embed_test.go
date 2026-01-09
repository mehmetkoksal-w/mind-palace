package dashboard

import (
	"io"
	"strings"
	"testing"
)

func TestEmbeddedAngularDashboard(t *testing.T) {
	f, err := embeddedAssets.Open("index.html")
	if err != nil {
		t.Skipf("dashboard assets not embedded (index.html missing): %v — build apps/dashboard to enable this test", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	content := string(data)

	// Check for placeholder (go install support) - this is valid
	if strings.Contains(content, "go install") {
		t.Log("using placeholder dashboard (for go install support)")
		return
	}

	// Verify it's an Angular app (full build)
	if !strings.Contains(content, "<app-root>") {
		t.Error("expected Angular app-root element")
	}

	// Verify it has the main script
	if !strings.Contains(content, "main-") && !strings.Contains(content, ".js") {
		t.Error("expected JavaScript bundle references")
	}
}
