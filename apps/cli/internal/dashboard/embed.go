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
	var sub fs.FS
	var err error

	// Try different structures based on how the dashboard was built/packaged:
	// 1. Local Angular 17+ build: dist/browser/index.html
	// 2. CI flattened build: dist/index.html
	structures := []string{
		"dist/browser", // Local Angular 17+ output
		"dist",         // CI flattened or legacy structure
	}

	for _, path := range structures {
		sub, err = fs.Sub(distFS, path)
		if err != nil {
			continue
		}
		// Check if index.html exists at this level
		if _, openErr := sub.Open("index.html"); openErr == nil {
			embeddedAssets = sub
			return
		}
	}

	panic("dashboard: index.html not found in embedded assets - run 'npm run build' in apps/dashboard first")
}
