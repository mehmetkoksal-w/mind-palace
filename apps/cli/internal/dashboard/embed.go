// Package dashboard provides an HTTP server for the Mind Palace web dashboard.
// This file handles embedding the built Angular dashboard assets.
package dashboard

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// embeddedAssets provides the embedded dashboard files.
var embeddedAssets fs.FS

func init() {
	// Use dist/browser/ subdirectory (Angular 17+ output structure)
	sub, err := fs.Sub(distFS, "dist/browser")
	if err != nil {
		// Try legacy dist/ structure for backwards compatibility
		sub, err = fs.Sub(distFS, "dist")
		if err != nil {
			panic("dashboard: dist directory not found - run 'npm run build' in apps/dashboard first")
		}
	}

	// Verify index.html exists
	if _, err := sub.Open("index.html"); err != nil {
		panic("dashboard: index.html not found in dist - run 'npm run build' in apps/dashboard first")
	}

	embeddedAssets = sub
}
