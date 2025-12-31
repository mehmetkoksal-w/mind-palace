package dashboard

import (
	"io"
	"strings"
	"testing"
)

func TestEmbeddedAngularDashboard(t *testing.T) {
	f, err := embeddedAssets.Open("index.html")
	if err != nil {
		t.Fatalf("Open(index.html) error = %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	content := string(data)

	// Verify it's an Angular app
	if !strings.Contains(content, "<app-root>") {
		t.Error("expected Angular app-root element")
	}

	// Verify it has the main script
	if !strings.Contains(content, "main-") && !strings.Contains(content, ".js") {
		t.Error("expected JavaScript bundle references")
	}
}
