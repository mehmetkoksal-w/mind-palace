// Package index provides the core database functionality for indexing project files and symbols.
package index

import (
	"context"
	"database/sql"
	"path/filepath"
)

// ExpandedFile represents a file with its expansion context
type ExpandedFile struct {
	Path        string   `json:"path"`
	Depth       int      `json:"depth"`       // 0 = seed file, 1+ = dependency levels
	ExpandedVia string   `json:"expandedVia"` // Relationship that brought this file in (empty for seed)
	Imports     []string `json:"imports"`     // Direct imports from this file
	ImportedBy  []string `json:"importedBy"`  // Files that import this file
}

// ExpandOptions configures dependency expansion behavior
type ExpandOptions struct {
	MaxDepth        int      // Max levels to traverse (default: 2)
	IncludeKinds    []string // Relationship kinds to follow (default: ["import"])
	ExcludePatterns []string // Glob patterns to skip
	MaxFiles        int      // Cap on total expanded files (default: 50)
	BothDirections  bool     // Expand both imports and importers (default: false, imports only)
}

// DefaultExpandOptions returns sensible defaults for expansion
func DefaultExpandOptions() *ExpandOptions {
	return &ExpandOptions{
		MaxDepth:        2,
		IncludeKinds:    []string{"import"},
		ExcludePatterns: DefaultExcludePatterns,
		MaxFiles:        50,
		BothDirections:  false,
	}
}

// ExpandWithDependencies recursively expands files with their dependencies.
// Given a set of seed files, it follows import relationships to find related files.
func ExpandWithDependencies(db *sql.DB, seedFiles []string, opts *ExpandOptions) ([]ExpandedFile, error) {
	if opts == nil {
		opts = DefaultExpandOptions()
	}
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 2
	}
	if opts.MaxFiles <= 0 {
		opts.MaxFiles = 50
	}
	if len(opts.IncludeKinds) == 0 {
		opts.IncludeKinds = []string{"import"}
	}

	visited := make(map[string]*ExpandedFile)

	var expand func(file string, depth int, via string) error
	expand = func(file string, depth int, via string) error {
		// Stop conditions
		if depth > opts.MaxDepth || len(visited) >= opts.MaxFiles {
			return nil
		}
		if visited[file] != nil {
			return nil
		}
		if shouldExcludeFile(file, opts.ExcludePatterns) {
			return nil
		}

		ef := &ExpandedFile{
			Path:        file,
			Depth:       depth,
			ExpandedVia: via,
			Imports:     []string{},
			ImportedBy:  []string{},
		}
		visited[file] = ef

		// Get files this file imports
		imports, err := getImportsForExpansion(db, file, opts.IncludeKinds)
		if err != nil {
			return err
		}

		for _, imp := range imports {
			if imp.TargetFile == "" {
				continue
			}
			ef.Imports = append(ef.Imports, imp.TargetFile)
			if err := expand(imp.TargetFile, depth+1, "imported-by:"+file); err != nil {
				return err
			}
		}

		// Optionally get files that import this file
		if opts.BothDirections {
			importers, err := getImportersForExpansion(db, file, opts.IncludeKinds)
			if err != nil {
				return err
			}
			for _, imp := range importers {
				ef.ImportedBy = append(ef.ImportedBy, imp.SourceFile)
				if err := expand(imp.SourceFile, depth+1, "imports:"+file); err != nil {
					return err
				}
			}
		}

		return nil
	}

	// Expand from each seed file
	for _, f := range seedFiles {
		if err := expand(f, 0, ""); err != nil {
			return nil, err
		}
	}

	// Convert map to slice, sorted by depth then path
	result := make([]ExpandedFile, 0, len(visited))
	for _, ef := range visited {
		result = append(result, *ef)
	}

	// Sort by depth (closer = first), then alphabetically
	sortExpandedFiles(result)

	return result, nil
}

// getImportsForExpansion returns imports for a file (filtered by relationship kinds)
func getImportsForExpansion(db *sql.DB, path string, kinds []string) ([]ImportInfo, error) {
	if len(kinds) == 0 {
		kinds = []string{"import"}
	}

	// Build placeholders for IN clause
	placeholders := make([]interface{}, 0, len(kinds)+1)
	placeholders = append(placeholders, path)
	kindPlaceholders := ""
	for i, k := range kinds {
		if i > 0 {
			kindPlaceholders += ","
		}
		kindPlaceholders += "?"
		placeholders = append(placeholders, k)
	}

	query := `
		SELECT source_file, target_file, target_symbol, kind, line
		FROM relationships
		WHERE source_file = ? AND kind IN (` + kindPlaceholders + `)
		ORDER BY line
	`

	rows, err := db.QueryContext(context.Background(), query, placeholders...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var imports []ImportInfo
	for rows.Next() {
		var imp ImportInfo
		var targetFile, targetSymbol sql.NullString
		if err := rows.Scan(&imp.SourceFile, &targetFile, &targetSymbol, &imp.Kind, &imp.Line); err != nil {
			return nil, err
		}
		if targetFile.Valid {
			imp.TargetFile = targetFile.String
		}
		if targetSymbol.Valid {
			imp.TargetSymbol = targetSymbol.String
		}
		imports = append(imports, imp)
	}
	return imports, rows.Err()
}

// getImportersForExpansion returns files that import the given file
func getImportersForExpansion(db *sql.DB, path string, kinds []string) ([]ImportInfo, error) {
	if len(kinds) == 0 {
		kinds = []string{"import"}
	}

	// Build placeholders for IN clause
	placeholders := make([]interface{}, 0, len(kinds)+1)
	placeholders = append(placeholders, path)
	kindPlaceholders := ""
	for i, k := range kinds {
		if i > 0 {
			kindPlaceholders += ","
		}
		kindPlaceholders += "?"
		placeholders = append(placeholders, k)
	}

	query := `
		SELECT source_file, target_file, target_symbol, kind, line
		FROM relationships
		WHERE target_file = ? AND kind IN (` + kindPlaceholders + `)
		ORDER BY source_file
	`

	rows, err := db.QueryContext(context.Background(), query, placeholders...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var imports []ImportInfo
	for rows.Next() {
		var imp ImportInfo
		var targetFile, targetSymbol sql.NullString
		if err := rows.Scan(&imp.SourceFile, &targetFile, &targetSymbol, &imp.Kind, &imp.Line); err != nil {
			return nil, err
		}
		if targetFile.Valid {
			imp.TargetFile = targetFile.String
		}
		if targetSymbol.Valid {
			imp.TargetSymbol = targetSymbol.String
		}
		imports = append(imports, imp)
	}
	return imports, rows.Err()
}

// sortExpandedFiles sorts by depth (ascending) then path (alphabetical)
func sortExpandedFiles(files []ExpandedFile) {
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			swap := false
			if files[i].Depth > files[j].Depth {
				swap = true
			} else if files[i].Depth == files[j].Depth && files[i].Path > files[j].Path {
				swap = true
			}
			if swap {
				files[i], files[j] = files[j], files[i]
			}
		}
	}
}

// GetImportGraph returns the full import graph for a set of files.
// Unlike ExpandWithDependencies, this doesn't follow recursively but returns
// direct imports and importers for each file.
func GetImportGraph(db *sql.DB, files []string) (map[string]*ExpandedFile, error) {
	result := make(map[string]*ExpandedFile)

	for _, file := range files {
		if shouldExcludeFile(file, DefaultExcludePatterns) {
			continue
		}

		ef := &ExpandedFile{
			Path:       file,
			Depth:      0,
			Imports:    []string{},
			ImportedBy: []string{},
		}

		// Get imports
		imports, err := getImportsForExpansion(db, file, []string{"import"})
		if err != nil {
			continue
		}
		for _, imp := range imports {
			if imp.TargetFile != "" {
				ef.Imports = append(ef.Imports, imp.TargetFile)
			}
		}

		// Get importers
		importers, err := getImportersForExpansion(db, file, []string{"import"})
		if err != nil {
			continue
		}
		for _, imp := range importers {
			ef.ImportedBy = append(ef.ImportedBy, imp.SourceFile)
		}

		result[file] = ef
	}

	return result, nil
}

// GetRelatedFilesBySymbol finds files related to a symbol through calls and references.
// This complements import-based expansion with semantic relationships.
func GetRelatedFilesBySymbol(db *sql.DB, symbolName string, maxFiles int) ([]ExpandedFile, error) {
	if maxFiles <= 0 {
		maxFiles = 20
	}

	seen := make(map[string]bool)
	var result []ExpandedFile

	// Get files that call this symbol
	callerRows, err := db.QueryContext(context.Background(), `
		SELECT DISTINCT source_file
		FROM relationships
		WHERE kind = 'call' 
		AND (target_symbol = ? OR target_symbol LIKE ? OR target_symbol LIKE ?)
		LIMIT ?
	`, symbolName, "%."+symbolName, "%::"+symbolName, maxFiles)
	if err != nil {
		return nil, err
	}
	defer callerRows.Close()

	for callerRows.Next() && len(result) < maxFiles {
		var path string
		if err := callerRows.Scan(&path); err != nil {
			continue
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		result = append(result, ExpandedFile{
			Path:        path,
			Depth:       1,
			ExpandedVia: "calls:" + symbolName,
		})
	}

	// Get files that reference this symbol
	refRows, err := db.QueryContext(context.Background(), `
		SELECT DISTINCT source_file
		FROM relationships
		WHERE kind = 'reference'
		AND (target_symbol = ? OR target_symbol LIKE ? OR target_symbol LIKE ?)
		LIMIT ?
	`, symbolName, "%."+symbolName, "%::"+symbolName, maxFiles-len(result))
	if err != nil {
		return nil, err
	}
	defer refRows.Close()

	for refRows.Next() && len(result) < maxFiles {
		var path string
		if err := refRows.Scan(&path); err != nil {
			continue
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		result = append(result, ExpandedFile{
			Path:        path,
			Depth:       1,
			ExpandedVia: "references:" + symbolName,
		})
	}

	return result, nil
}

// NormalizePath ensures consistent path format for comparisons
func NormalizePath(path string) string {
	// Convert to forward slashes and clean
	path = filepath.ToSlash(path)
	return filepath.Clean(path)
}
