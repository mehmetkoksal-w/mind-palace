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

	// Skip if dashboard is not built (placeholder index.html)
	if !strings.Contains(content, "<app-root>") {
		t.Skip("dashboard not built (placeholder index.html) — run 'cd apps/dashboard && ng build' to enable this test")
	}

	// Verify it has the main script
	if !strings.Contains(content, "main-") && !strings.Contains(content, ".js") {
		t.Error("expected JavaScript bundle references")
	}
}
