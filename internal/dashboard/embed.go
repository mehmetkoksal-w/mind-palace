package dashboard

import (
	"embed"
	"io"
	"io/fs"
	"time"
)

//go:embed all:dist
var distFS embed.FS

// embeddedAssets provides the embedded dashboard files.
// Falls back to a simple placeholder if dist/ doesn't exist.
var embeddedAssets fs.FS

func init() {
	// Try to use dist/ subdirectory
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// No dist directory - use fallback
		embeddedAssets = &fallbackFS{}
		return
	}

	// Check if index.html exists in dist - if not, use fallback
	// This handles the case where dist/ exists but only has .gitkeep
	if _, err := sub.Open("index.html"); err != nil {
		embeddedAssets = &fallbackFS{}
		return
	}

	embeddedAssets = sub
}

// fallbackFS provides a minimal fallback when no dashboard is built.
type fallbackFS struct{}

func (f *fallbackFS) Open(name string) (fs.File, error) {
	if name == "." {
		return &fallbackDir{}, nil
	}
	if name == "index.html" {
		return &fallbackFile{}, nil
	}
	return nil, fs.ErrNotExist
}

// fallbackDir represents the root directory for http.FileServer
type fallbackDir struct{}

func (d *fallbackDir) Read(b []byte) (int, error) {
	return 0, fs.ErrInvalid
}

func (d *fallbackDir) Stat() (fs.FileInfo, error) {
	return &fallbackDirInfo{}, nil
}

func (d *fallbackDir) Close() error {
	return nil
}

func (d *fallbackDir) ReadDir(n int) ([]fs.DirEntry, error) {
	// Return index.html as the only entry
	if n <= 0 {
		return []fs.DirEntry{&fallbackDirEntry{}}, nil
	}
	return []fs.DirEntry{&fallbackDirEntry{}}, nil
}

type fallbackDirInfo struct{}

func (di *fallbackDirInfo) Name() string       { return "." }
func (di *fallbackDirInfo) Size() int64        { return 0 }
func (di *fallbackDirInfo) Mode() fs.FileMode  { return fs.ModeDir | 0555 }
func (di *fallbackDirInfo) ModTime() time.Time { return time.Now() }
func (di *fallbackDirInfo) IsDir() bool        { return true }
func (di *fallbackDirInfo) Sys() any           { return nil }

type fallbackDirEntry struct{}

func (de *fallbackDirEntry) Name() string               { return "index.html" }
func (de *fallbackDirEntry) IsDir() bool                { return false }
func (de *fallbackDirEntry) Type() fs.FileMode          { return 0 }
func (de *fallbackDirEntry) Info() (fs.FileInfo, error) { return &fallbackFileInfo{}, nil }

type fallbackFile struct {
	offset int
}

var fallbackHTML = []byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Mind Palace Dashboard</title>
    <style>
        :root {
            --palace-purple: #6B5B95;
            --palace-purple-light: #8B7BB5;
            --palace-purple-dark: #4B3B75;
            --memory-blue: #4A90D9;
            --memory-blue-light: #6AB0F9;
            --archive-gray: #2D3748;
            --archive-gray-light: #4A5568;
            --parchment: #F7F5F2;
            --fresh: #10B981;
            --stale: #EF4444;
            --scanning: #F59E0B;
            --bg-dark: #1a1b2e;
            --bg-card: #252640;
            --bg-card-hover: #2d2e4a;
            --text-primary: #F7F5F2;
            --text-secondary: #A0AEC0;
            --text-muted: #718096;
        }

        * { box-sizing: border-box; margin: 0; padding: 0; }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif;
            background: var(--bg-dark);
            color: var(--text-primary);
            min-height: 100vh;
            overflow-x: hidden;
        }

        .dashboard {
            max-width: 1400px;
            margin: 0 auto;
            padding: 2rem;
        }

        /* Header */
        .header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 2rem;
            padding-bottom: 1.5rem;
            border-bottom: 1px solid var(--archive-gray-light);
        }

        .logo-section {
            display: flex;
            align-items: center;
            gap: 1rem;
        }

        .logo {
            width: 48px;
            height: 48px;
            background: var(--palace-purple);
            border-radius: 12px;
            display: flex;
            align-items: center;
            justify-content: center;
        }

        .logo svg {
            width: 32px;
            height: 32px;
            color: white;
        }

        .title {
            font-size: 1.75rem;
            font-weight: 700;
            color: var(--palace-purple-light);
        }

        .subtitle {
            font-size: 0.875rem;
            color: var(--text-secondary);
            margin-top: 0.25rem;
        }

        .status-badge {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            padding: 0.5rem 1rem;
            background: rgba(16, 185, 129, 0.15);
            border: 1px solid var(--fresh);
            border-radius: 2rem;
            font-size: 0.875rem;
            color: var(--fresh);
        }

        .status-dot {
            width: 8px;
            height: 8px;
            background: var(--fresh);
            border-radius: 50%;
            animation: pulse 2s infinite;
        }

        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.4; }
        }

        /* Workspace Switcher */
        .workspace-switcher {
            position: relative;
        }

        .workspace-current {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            padding: 0.5rem 1rem;
            background: var(--bg-card);
            border: 1px solid var(--archive-gray-light);
            border-radius: 8px;
            cursor: pointer;
            transition: border-color 0.2s, background 0.2s;
            min-width: 180px;
        }

        .workspace-current:hover {
            border-color: var(--palace-purple);
            background: var(--bg-card-hover);
        }

        .workspace-current-icon {
            width: 24px;
            height: 24px;
            background: var(--palace-purple);
            border-radius: 4px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 0.75rem;
            color: white;
            flex-shrink: 0;
        }

        .workspace-current-name {
            font-size: 0.85rem;
            font-weight: 500;
            flex: 1;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .workspace-current-arrow {
            font-size: 0.6rem;
            color: var(--text-muted);
            transition: transform 0.2s;
        }

        .workspace-switcher.open .workspace-current-arrow {
            transform: rotate(180deg);
        }

        .workspace-dropdown {
            position: absolute;
            top: calc(100% + 4px);
            right: 0;
            min-width: 280px;
            background: var(--bg-card);
            border: 1px solid var(--archive-gray-light);
            border-radius: 8px;
            box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
            z-index: 100;
            display: none;
            max-height: 320px;
            overflow-y: auto;
        }

        .workspace-switcher.open .workspace-dropdown {
            display: block;
        }

        .workspace-dropdown-header {
            padding: 0.75rem 1rem;
            border-bottom: 1px solid var(--archive-gray-light);
            font-size: 0.7rem;
            font-weight: 600;
            color: var(--text-muted);
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        .workspace-option {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            padding: 0.75rem 1rem;
            cursor: pointer;
            transition: background 0.2s;
            border-bottom: 1px solid var(--archive-gray);
        }

        .workspace-option:last-child {
            border-bottom: none;
        }

        .workspace-option:hover {
            background: var(--bg-card-hover);
        }

        .workspace-option.active {
            background: rgba(107, 91, 149, 0.15);
        }

        .workspace-option-icon {
            width: 28px;
            height: 28px;
            background: var(--memory-blue);
            border-radius: 4px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 0.7rem;
            color: white;
            flex-shrink: 0;
        }

        .workspace-option.active .workspace-option-icon {
            background: var(--palace-purple);
        }

        .workspace-option-info {
            flex: 1;
            min-width: 0;
        }

        .workspace-option-name {
            font-size: 0.85rem;
            font-weight: 500;
            margin-bottom: 0.15rem;
        }

        .workspace-option-path {
            font-size: 0.7rem;
            color: var(--text-muted);
            font-family: monospace;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .workspace-option-status {
            font-size: 0.65rem;
            padding: 0.15rem 0.4rem;
            border-radius: 3px;
            flex-shrink: 0;
        }

        .workspace-option-status.current {
            background: rgba(107, 91, 149, 0.2);
            color: var(--palace-purple-light);
        }

        .workspace-option-status.no-palace {
            background: rgba(239, 68, 68, 0.2);
            color: var(--stale);
        }

        /* Stats Grid */
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
            gap: 1rem;
            margin-bottom: 2rem;
        }

        .stat-card {
            background: var(--bg-card);
            border-radius: 12px;
            padding: 1.25rem;
            border: 1px solid var(--archive-gray-light);
            transition: border-color 0.2s, transform 0.2s;
        }

        .stat-card:hover {
            border-color: var(--palace-purple);
            transform: translateY(-2px);
        }

        .stat-icon {
            width: 36px;
            height: 36px;
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            margin-bottom: 0.75rem;
            font-size: 1.1rem;
        }

        .stat-icon.rooms { background: rgba(107, 91, 149, 0.2); }
        .stat-icon.sessions { background: rgba(74, 144, 217, 0.2); }
        .stat-icon.learnings { background: rgba(16, 185, 129, 0.2); }
        .stat-icon.files { background: rgba(245, 158, 11, 0.2); }
        .stat-icon.corridors { background: rgba(139, 123, 181, 0.2); }

        .stat-value {
            font-size: 2rem;
            font-weight: 700;
            color: var(--text-primary);
            line-height: 1;
            margin-bottom: 0.25rem;
        }

        .stat-label {
            font-size: 0.75rem;
            color: var(--text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        /* Panels */
        .panel {
            background: var(--bg-card);
            border-radius: 12px;
            border: 1px solid var(--archive-gray-light);
            overflow: hidden;
            margin-bottom: 1.5rem;
        }

        .panel-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 1rem 1.25rem;
            border-bottom: 1px solid var(--archive-gray-light);
            background: rgba(0, 0, 0, 0.15);
        }

        .panel-title {
            font-size: 0.9rem;
            font-weight: 600;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .panel-controls {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            font-size: 0.75rem;
            color: var(--text-muted);
        }

        .panel-body {
            padding: 1.25rem;
        }

        /* Neural Map */
        .neural-panel {
            height: 450px;
        }

        .neural-panel .panel-body {
            padding: 0;
            height: calc(100% - 52px);
            position: relative;
            overflow: hidden;
            cursor: grab;
        }

        .neural-panel .panel-body:active {
            cursor: grabbing;
        }

        #neural-canvas {
            display: block;
            position: absolute;
            top: 0;
            left: 0;
        }

        .zoom-controls {
            position: absolute;
            bottom: 1rem;
            right: 1rem;
            display: flex;
            gap: 0.5rem;
            z-index: 10;
        }

        .zoom-btn {
            width: 32px;
            height: 32px;
            border: 1px solid var(--archive-gray-light);
            background: var(--bg-card);
            color: var(--text-primary);
            border-radius: 6px;
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 1.1rem;
            transition: background 0.2s, border-color 0.2s;
        }

        .zoom-btn:hover {
            background: var(--bg-card-hover);
            border-color: var(--palace-purple);
        }

        .zoom-info {
            position: absolute;
            bottom: 1rem;
            left: 1rem;
            font-size: 0.7rem;
            color: var(--text-muted);
            z-index: 10;
        }

        .neural-legend {
            position: absolute;
            top: 0.75rem;
            left: 0.75rem;
            background: rgba(26, 27, 46, 0.9);
            border: 1px solid var(--archive-gray-light);
            border-radius: 6px;
            padding: 0.5rem 0.75rem;
            z-index: 10;
            font-size: 0.7rem;
        }

        .neural-legend-title {
            font-weight: 600;
            color: var(--text-secondary);
            margin-bottom: 0.4rem;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        .neural-legend-item {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            margin-bottom: 0.25rem;
            color: var(--text-secondary);
        }

        .neural-legend-item:last-child {
            margin-bottom: 0;
        }

        .neural-legend-dot {
            width: 10px;
            height: 10px;
            border-radius: 50%;
            flex-shrink: 0;
        }

        .neural-legend-dot.palace { background: var(--palace-purple); }
        .neural-legend-dot.room { background: var(--memory-blue); }
        .neural-legend-dot.file { background: var(--parchment); }
        .neural-legend-dot.folder { background: #5DADE2; }
        .neural-legend-dot.learning { background: var(--fresh); }
        .neural-legend-dot.corridor { background: var(--scanning); }

        /* Content Grid */
        .content-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 1.5rem;
        }

        @media (max-width: 900px) {
            .content-grid { grid-template-columns: 1fr; }
        }

        .content-grid .panel-body {
            max-height: 350px;
            overflow-y: auto;
        }

        .content-grid .panel-body::-webkit-scrollbar {
            width: 5px;
        }

        .content-grid .panel-body::-webkit-scrollbar-track {
            background: var(--archive-gray);
        }

        .content-grid .panel-body::-webkit-scrollbar-thumb {
            background: var(--palace-purple);
            border-radius: 3px;
        }

        /* Session Items */
        .session-item {
            display: flex;
            align-items: flex-start;
            gap: 0.75rem;
            padding: 0.875rem;
            background: var(--bg-dark);
            border-radius: 8px;
            margin-bottom: 0.5rem;
            border: 1px solid transparent;
            transition: border-color 0.2s;
        }

        .session-item:hover {
            border-color: var(--archive-gray-light);
        }

        .session-icon {
            width: 32px;
            height: 32px;
            border-radius: 6px;
            background: var(--palace-purple);
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 0.8rem;
            flex-shrink: 0;
        }

        .session-content { flex: 1; min-width: 0; }

        .session-agent {
            font-weight: 600;
            font-size: 0.85rem;
            margin-bottom: 0.2rem;
        }

        .session-goal {
            font-size: 0.75rem;
            color: var(--text-secondary);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .session-state {
            display: inline-block;
            margin-top: 0.4rem;
            padding: 0.15rem 0.4rem;
            border-radius: 3px;
            font-size: 0.65rem;
            font-weight: 600;
            text-transform: uppercase;
        }

        .session-state.active { background: rgba(16, 185, 129, 0.2); color: var(--fresh); }
        .session-state.completed { background: rgba(74, 144, 217, 0.2); color: var(--memory-blue); }

        /* Learning Items */
        .learning-item {
            padding: 0.875rem;
            background: var(--bg-dark);
            border-radius: 8px;
            margin-bottom: 0.5rem;
            border-left: 3px solid var(--palace-purple);
        }

        .learning-content {
            font-size: 0.85rem;
            line-height: 1.4;
            margin-bottom: 0.5rem;
        }

        .learning-meta {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            font-size: 0.7rem;
            color: var(--text-muted);
        }

        .learning-scope {
            padding: 0.15rem 0.4rem;
            background: rgba(107, 91, 149, 0.2);
            color: var(--palace-purple-light);
            border-radius: 3px;
            font-weight: 500;
        }

        .confidence-bar {
            width: 50px;
            height: 3px;
            background: var(--archive-gray);
            border-radius: 2px;
            overflow: hidden;
        }

        .confidence-fill {
            height: 100%;
            background: var(--palace-purple);
            border-radius: 2px;
        }

        /* Empty States */
        .empty-state {
            text-align: center;
            padding: 2rem;
            color: var(--text-muted);
        }

        .empty-state-icon {
            font-size: 2rem;
            margin-bottom: 0.75rem;
            opacity: 0.5;
        }

        /* Loading */
        .loading {
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 2rem;
        }

        .loading-spinner {
            width: 24px;
            height: 24px;
            border: 2px solid var(--archive-gray-light);
            border-top-color: var(--palace-purple);
            border-radius: 50%;
            animation: spin 0.8s linear infinite;
        }

        @keyframes spin { to { transform: rotate(360deg); } }

        /* Workspace Card */
        .workspace-card {
            background: var(--bg-card);
            border-radius: 12px;
            border: 1px solid var(--archive-gray-light);
            padding: 1.25rem;
            margin-bottom: 1.5rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            flex-wrap: wrap;
            gap: 1rem;
        }

        .workspace-main {
            display: flex;
            align-items: center;
            gap: 1rem;
        }

        .workspace-icon {
            width: 48px;
            height: 48px;
            background: var(--palace-purple);
            border-radius: 10px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 1.5rem;
        }

        .workspace-name {
            font-size: 1.25rem;
            font-weight: 700;
            color: var(--text-primary);
        }

        .workspace-path {
            font-size: 0.75rem;
            color: var(--text-muted);
            font-family: monospace;
            max-width: 400px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }

        .workspace-meta {
            display: flex;
            gap: 2rem;
        }

        .meta-item {
            text-align: center;
        }

        .meta-label {
            display: block;
            font-size: 0.65rem;
            color: var(--text-muted);
            text-transform: uppercase;
            letter-spacing: 0.05em;
            margin-bottom: 0.25rem;
        }

        .meta-value {
            font-size: 0.9rem;
            font-weight: 600;
            color: var(--text-primary);
        }

        .status-fresh { color: var(--fresh); }
        .status-stale { color: var(--stale); }
        .status-scanning { color: var(--scanning); }

        /* Corridor Items */
        .corridor-item {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            padding: 0.875rem;
            background: var(--bg-dark);
            border-radius: 8px;
            margin-bottom: 0.5rem;
            border: 1px solid transparent;
            transition: border-color 0.2s;
        }

        .corridor-item:hover {
            border-color: var(--archive-gray-light);
        }

        .corridor-icon {
            width: 32px;
            height: 32px;
            border-radius: 6px;
            background: var(--scanning);
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 0.8rem;
            flex-shrink: 0;
        }

        .corridor-content { flex: 1; min-width: 0; }

        .corridor-name {
            font-weight: 600;
            font-size: 0.85rem;
            margin-bottom: 0.2rem;
        }

        .corridor-path {
            font-size: 0.7rem;
            color: var(--text-muted);
            font-family: monospace;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .corridor-meta {
            font-size: 0.7rem;
            color: var(--text-secondary);
        }

        /* Hotspot Items */
        .hotspot-item {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            padding: 0.875rem;
            background: var(--bg-dark);
            border-radius: 8px;
            margin-bottom: 0.5rem;
        }

        .hotspot-icon {
            width: 32px;
            height: 32px;
            border-radius: 6px;
            background: var(--stale);
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 0.8rem;
            flex-shrink: 0;
        }

        .hotspot-icon.warm { background: var(--scanning); }
        .hotspot-icon.hot { background: var(--stale); }

        .hotspot-content { flex: 1; min-width: 0; }

        .hotspot-path {
            font-weight: 500;
            font-size: 0.8rem;
            font-family: monospace;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .hotspot-meta {
            display: flex;
            gap: 1rem;
            font-size: 0.7rem;
            color: var(--text-muted);
            margin-top: 0.25rem;
        }

        /* Three column grid for bottom panels */
        .bottom-grid {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 1.5rem;
        }

        @media (max-width: 1100px) {
            .bottom-grid { grid-template-columns: 1fr 1fr; }
        }

        @media (max-width: 700px) {
            .bottom-grid { grid-template-columns: 1fr; }
            .workspace-meta { flex-wrap: wrap; gap: 1rem; }
        }

        /* Footer */
        .footer {
            text-align: center;
            padding: 1.5rem;
            color: var(--text-muted);
            font-size: 0.8rem;
        }

        .footer a {
            color: var(--palace-purple-light);
            text-decoration: none;
        }

        .footer a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="dashboard">
        <!-- Header -->
        <header class="header">
            <div class="logo-section">
                <div class="logo">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                        <circle cx="12" cy="12" r="9"/>
                        <circle cx="12" cy="12" r="3"/>
                        <line x1="12" y1="3" x2="12" y2="6"/>
                        <line x1="12" y1="18" x2="12" y2="21"/>
                        <line x1="3" y1="12" x2="6" y2="12"/>
                        <line x1="18" y1="12" x2="21" y2="12"/>
                    </svg>
                </div>
                <div>
                    <h1 class="title">Mind Palace</h1>
                    <p class="subtitle">Workspace Intelligence Dashboard</p>
                </div>
            </div>
            <div class="workspace-switcher" id="workspace-switcher">
                <div class="workspace-current" id="workspace-current">
                    <div class="workspace-current-icon">W</div>
                    <span class="workspace-current-name" id="workspace-switcher-name">Loading...</span>
                    <span class="workspace-current-arrow">&#9660;</span>
                </div>
                <div class="workspace-dropdown" id="workspace-dropdown">
                    <div class="workspace-dropdown-header">Switch Workspace</div>
                    <div id="workspace-options"></div>
                </div>
            </div>
        </header>

        <!-- Workspace Identity -->
        <div class="workspace-card" id="workspace-card">
            <div class="workspace-main">
                <div class="workspace-icon">W</div>
                <div class="workspace-info">
                    <div class="workspace-name" id="workspace-name">Loading...</div>
                    <div class="workspace-path" id="workspace-path"></div>
                </div>
            </div>
            <div class="workspace-meta">
                <div class="meta-item">
                    <span class="meta-label">Last Scan</span>
                    <span class="meta-value" id="last-scan">-</span>
                </div>
                <div class="meta-item">
                    <span class="meta-label">Files Indexed</span>
                    <span class="meta-value" id="files-indexed">-</span>
                </div>
                <div class="meta-item">
                    <span class="meta-label">Status</span>
                    <span class="meta-value status-fresh" id="index-status">-</span>
                </div>
            </div>
        </div>

        <!-- Stats Grid -->
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-icon rooms">R</div>
                <div class="stat-value" id="stat-rooms">-</div>
                <div class="stat-label">Rooms</div>
            </div>
            <div class="stat-card">
                <div class="stat-icon sessions">S</div>
                <div class="stat-value" id="stat-sessions">-</div>
                <div class="stat-label">Sessions</div>
            </div>
            <div class="stat-card">
                <div class="stat-icon learnings">L</div>
                <div class="stat-value" id="stat-learnings">-</div>
                <div class="stat-label">Learnings</div>
            </div>
            <div class="stat-card">
                <div class="stat-icon files">F</div>
                <div class="stat-value" id="stat-files">-</div>
                <div class="stat-label">Files</div>
            </div>
            <div class="stat-card">
                <div class="stat-icon corridors">C</div>
                <div class="stat-value" id="stat-corridors">-</div>
                <div class="stat-label">Corridors</div>
            </div>
        </div>

        <!-- Neural Map -->
        <div class="panel neural-panel">
            <div class="panel-header">
                <span class="panel-title">Neural Map</span>
                <span class="panel-controls">Scroll to zoom &middot; Drag to pan</span>
            </div>
            <div class="panel-body" id="neural-container">
                <canvas id="neural-canvas"></canvas>
                <div class="neural-legend">
                    <div class="neural-legend-title">Legend</div>
                    <div class="neural-legend-item"><span class="neural-legend-dot palace"></span> Palace</div>
                    <div class="neural-legend-item"><span class="neural-legend-dot room"></span> Rooms</div>
                    <div class="neural-legend-item"><span class="neural-legend-dot file"></span> Entry Files</div>
                    <div class="neural-legend-item"><span class="neural-legend-dot folder"></span> Entry Folders</div>
                    <div class="neural-legend-item"><span class="neural-legend-dot learning"></span> Learnings</div>
                    <div class="neural-legend-item"><span class="neural-legend-dot corridor"></span> Corridors</div>
                </div>
                <div class="zoom-controls">
                    <button class="zoom-btn" id="zoom-in" title="Zoom In">+</button>
                    <button class="zoom-btn" id="zoom-out" title="Zoom Out">-</button>
                    <button class="zoom-btn" id="zoom-reset" title="Reset View">&#8634;</button>
                </div>
                <div class="zoom-info" id="zoom-info">100%</div>
            </div>
        </div>

        <!-- Bottom Grid: Sessions, Learnings, Corridors -->
        <div class="bottom-grid">
            <div class="panel">
                <div class="panel-header">
                    <span class="panel-title">Recent Sessions</span>
                </div>
                <div class="panel-body" id="sessions-list">
                    <div class="loading"><div class="loading-spinner"></div></div>
                </div>
            </div>
            <div class="panel">
                <div class="panel-header">
                    <span class="panel-title">Knowledge Learnings</span>
                </div>
                <div class="panel-body" id="learnings-list">
                    <div class="loading"><div class="loading-spinner"></div></div>
                </div>
            </div>
            <div class="panel">
                <div class="panel-header">
                    <span class="panel-title">Linked Corridors</span>
                </div>
                <div class="panel-body" id="corridors-list">
                    <div class="loading"><div class="loading-spinner"></div></div>
                </div>
            </div>
        </div>

        <!-- File Hotspots -->
        <div class="panel" style="margin-top: 1.5rem;">
            <div class="panel-header">
                <span class="panel-title">File Hotspots</span>
                <span class="panel-controls">Frequently edited files</span>
            </div>
            <div class="panel-body" id="hotspots-list">
                <div class="loading"><div class="loading-spinner"></div></div>
            </div>
        </div>

        <!-- Footer -->
        <footer class="footer">
            Mind Palace &middot;
            <a href="/api/health">Health</a> &middot;
            <a href="/api/stats">Stats</a> &middot;
            <a href="/api/sessions">Sessions</a> &middot;
            <a href="/api/learnings">Learnings</a>
        </footer>
    </div>

    <script>
    (function() {
        'use strict';

        // Advanced Neural Map with physics, interactions, and animations
        class NeuralMap {
            constructor(container, canvas) {
                this.container = container;
                this.canvas = canvas;
                this.ctx = canvas.getContext('2d');

                // Transform state
                this.scale = 1;
                this.offsetX = 0;
                this.offsetY = 0;
                this.minScale = 0.2;
                this.maxScale = 5;

                // Interaction state
                this.isDragging = false;
                this.lastX = 0;
                this.lastY = 0;
                this.hoveredNode = null;
                this.selectedNode = null;
                this.mouseX = 0;
                this.mouseY = 0;

                // Data
                this.nodes = [];
                this.edges = [];
                this.particles = [];

                // Physics settings
                this.physics = {
                    enabled: true,
                    repulsion: 800,
                    attraction: 0.03,
                    damping: 0.85,
                    minVelocity: 0.1,
                    centerGravity: 0.01
                };

                // Animation
                this.animationId = null;
                this.lastTime = 0;
                this.particleSpeed = 0.0004; // Much slower for visibility

                // Device pixel ratio
                this.dpr = window.devicePixelRatio || 1;

                this.setupCanvas();
                this.bindEvents();
                this.startAnimation();
            }

            setupCanvas() {
                const rect = this.container.getBoundingClientRect();
                this.width = rect.width;
                this.height = rect.height;

                this.canvas.width = this.width * this.dpr;
                this.canvas.height = this.height * this.dpr;
                this.canvas.style.width = this.width + 'px';
                this.canvas.style.height = this.height + 'px';

                this.ctx.setTransform(this.dpr, 0, 0, this.dpr, 0, 0);
            }

            bindEvents() {
                // Resize
                const resizeObserver = new ResizeObserver(() => {
                    this.setupCanvas();
                });
                resizeObserver.observe(this.container);

                // Mouse wheel zoom
                this.container.addEventListener('wheel', (e) => {
                    e.preventDefault();
                    const rect = this.canvas.getBoundingClientRect();
                    const mouseX = e.clientX - rect.left;
                    const mouseY = e.clientY - rect.top;
                    const delta = e.deltaY > 0 ? 0.9 : 1.1;
                    this.zoomAt(mouseX, mouseY, delta);
                }, { passive: false });

                // Mouse interactions
                this.container.addEventListener('mousedown', (e) => {
                    const node = this.getNodeAtPosition(e.clientX, e.clientY);
                    if (node) {
                        this.selectedNode = node;
                        node.fixed = true;
                    } else {
                        this.isDragging = true;
                    }
                    this.lastX = e.clientX;
                    this.lastY = e.clientY;
                });

                this.container.addEventListener('mousemove', (e) => {
                    const rect = this.canvas.getBoundingClientRect();
                    this.mouseX = e.clientX - rect.left;
                    this.mouseY = e.clientY - rect.top;

                    if (this.selectedNode) {
                        // Drag node
                        const worldPos = this.screenToWorld(this.mouseX, this.mouseY);
                        this.selectedNode.x = worldPos.x;
                        this.selectedNode.y = worldPos.y;
                        this.selectedNode.vx = 0;
                        this.selectedNode.vy = 0;
                    } else if (this.isDragging) {
                        // Pan canvas
                        const dx = e.clientX - this.lastX;
                        const dy = e.clientY - this.lastY;
                        this.offsetX += dx;
                        this.offsetY += dy;
                        this.lastX = e.clientX;
                        this.lastY = e.clientY;
                    } else {
                        // Hover detection
                        this.hoveredNode = this.getNodeAtPosition(e.clientX, e.clientY);
                        this.container.style.cursor = this.hoveredNode ? 'pointer' : 'grab';
                    }
                });

                window.addEventListener('mouseup', () => {
                    if (this.selectedNode) {
                        this.selectedNode.fixed = false;
                        this.selectedNode = null;
                    }
                    this.isDragging = false;
                });

                // Click to focus
                this.container.addEventListener('click', (e) => {
                    const node = this.getNodeAtPosition(e.clientX, e.clientY);
                    if (node && !this.isDragging) {
                        this.focusOnNode(node);
                    }
                });

                // Touch support
                let lastTouchDist = 0;
                this.container.addEventListener('touchstart', (e) => {
                    if (e.touches.length === 1) {
                        const touch = e.touches[0];
                        const node = this.getNodeAtPosition(touch.clientX, touch.clientY);
                        if (node) {
                            this.selectedNode = node;
                            node.fixed = true;
                        } else {
                            this.isDragging = true;
                        }
                        this.lastX = touch.clientX;
                        this.lastY = touch.clientY;
                    } else if (e.touches.length === 2) {
                        lastTouchDist = Math.hypot(
                            e.touches[0].clientX - e.touches[1].clientX,
                            e.touches[0].clientY - e.touches[1].clientY
                        );
                    }
                }, { passive: true });

                this.container.addEventListener('touchmove', (e) => {
                    if (e.touches.length === 1) {
                        const touch = e.touches[0];
                        const rect = this.canvas.getBoundingClientRect();
                        this.mouseX = touch.clientX - rect.left;
                        this.mouseY = touch.clientY - rect.top;

                        if (this.selectedNode) {
                            const worldPos = this.screenToWorld(this.mouseX, this.mouseY);
                            this.selectedNode.x = worldPos.x;
                            this.selectedNode.y = worldPos.y;
                        } else if (this.isDragging) {
                            this.offsetX += touch.clientX - this.lastX;
                            this.offsetY += touch.clientY - this.lastY;
                            this.lastX = touch.clientX;
                            this.lastY = touch.clientY;
                        }
                    } else if (e.touches.length === 2) {
                        const dist = Math.hypot(
                            e.touches[0].clientX - e.touches[1].clientX,
                            e.touches[0].clientY - e.touches[1].clientY
                        );
                        if (lastTouchDist > 0) {
                            const centerX = (e.touches[0].clientX + e.touches[1].clientX) / 2;
                            const centerY = (e.touches[0].clientY + e.touches[1].clientY) / 2;
                            const rect = this.canvas.getBoundingClientRect();
                            this.zoomAt(centerX - rect.left, centerY - rect.top, dist / lastTouchDist);
                        }
                        lastTouchDist = dist;
                    }
                }, { passive: true });

                this.container.addEventListener('touchend', () => {
                    if (this.selectedNode) {
                        this.selectedNode.fixed = false;
                        this.selectedNode = null;
                    }
                    this.isDragging = false;
                    lastTouchDist = 0;
                });

                // Zoom buttons
                document.getElementById('zoom-in').addEventListener('click', () => {
                    this.zoomAt(this.width / 2, this.height / 2, 1.3);
                });
                document.getElementById('zoom-out').addEventListener('click', () => {
                    this.zoomAt(this.width / 2, this.height / 2, 0.7);
                });
                document.getElementById('zoom-reset').addEventListener('click', () => {
                    this.resetView();
                });
            }

            screenToWorld(sx, sy) {
                return {
                    x: (sx - this.offsetX) / this.scale,
                    y: (sy - this.offsetY) / this.scale
                };
            }

            worldToScreen(wx, wy) {
                return {
                    x: wx * this.scale + this.offsetX,
                    y: wy * this.scale + this.offsetY
                };
            }

            getNodeAtPosition(clientX, clientY) {
                const rect = this.canvas.getBoundingClientRect();
                const screenX = clientX - rect.left;
                const screenY = clientY - rect.top;
                const world = this.screenToWorld(screenX, screenY);

                // Check nodes in reverse order (top-most first)
                for (let i = this.nodes.length - 1; i >= 0; i--) {
                    const node = this.nodes[i];
                    const dx = world.x - node.x;
                    const dy = world.y - node.y;
                    const dist = Math.sqrt(dx * dx + dy * dy);
                    if (dist <= node.radius + 5) {
                        return node;
                    }
                }
                return null;
            }

            zoomAt(x, y, factor) {
                const newScale = Math.max(this.minScale, Math.min(this.maxScale, this.scale * factor));
                if (newScale === this.scale) return;

                const worldX = (x - this.offsetX) / this.scale;
                const worldY = (y - this.offsetY) / this.scale;

                this.scale = newScale;
                this.offsetX = x - worldX * this.scale;
                this.offsetY = y - worldY * this.scale;

                document.getElementById('zoom-info').textContent = Math.round(this.scale * 100) + '%';
            }

            resetView() {
                this.scale = 1;
                this.offsetX = 0;
                this.offsetY = 0;
                document.getElementById('zoom-info').textContent = '100%';
            }

            focusOnNode(node) {
                // Animate zoom to node
                const targetScale = 1.5;
                const targetOffsetX = this.width / 2 - node.x * targetScale;
                const targetOffsetY = this.height / 2 - node.y * targetScale;

                const startScale = this.scale;
                const startOffsetX = this.offsetX;
                const startOffsetY = this.offsetY;
                const duration = 300;
                const startTime = performance.now();

                const animate = (time) => {
                    const elapsed = time - startTime;
                    const t = Math.min(1, elapsed / duration);
                    const ease = 1 - Math.pow(1 - t, 3); // Ease out cubic

                    this.scale = startScale + (targetScale - startScale) * ease;
                    this.offsetX = startOffsetX + (targetOffsetX - startOffsetX) * ease;
                    this.offsetY = startOffsetY + (targetOffsetY - startOffsetY) * ease;

                    document.getElementById('zoom-info').textContent = Math.round(this.scale * 100) + '%';

                    if (t < 1) {
                        requestAnimationFrame(animate);
                    }
                };
                requestAnimationFrame(animate);
            }

            setData(stats, rooms, learnings) {
                this.nodes = [];
                this.edges = [];
                this.particles = [];

                const centerX = this.width / 2;
                const centerY = this.height / 2;

                // Palace center node (largest, fixed position initially)
                const palaceNode = {
                    id: 'palace',
                    x: centerX,
                    y: centerY,
                    vx: 0,
                    vy: 0,
                    radius: 32,
                    baseRadius: 32,
                    label: 'Palace',
                    color: '#6B5B95',
                    type: 'palace',
                    fixed: false,
                    connections: 0
                };
                this.nodes.push(palaceNode);

                // Room nodes - use actual rooms array, fallback to stats count
                const roomData = rooms || [];
                const roomCount = roomData.length > 0 ? roomData.length : (stats.rooms || 0);
                const roomRadius = Math.min(this.width, this.height) * 0.25;
                const learningsPerRoom = {};

                // Count learnings per room
                const learningData = learnings || [];
                learningData.forEach((l, i) => {
                    const roomIdx = i % Math.max(1, Math.min(roomCount, 12));
                    learningsPerRoom[roomIdx] = (learningsPerRoom[roomIdx] || 0) + 1;
                });

                // Track room nodes for file attachment
                const roomNodes = [];

                for (let i = 0; i < Math.min(roomCount, 12); i++) {
                    const room = roomData[i] || {};
                    const roomName = room.name || ('Room ' + (i + 1));
                    const entryPoints = room.entryPoints || [];

                    const angle = (i / Math.min(roomCount, 12)) * Math.PI * 2 - Math.PI / 2;
                    const roomLearnings = learningsPerRoom[i] || 0;
                    const sizeBonus = Math.min(roomLearnings * 2, 10);
                    const baseRadius = 16 + sizeBonus;

                    const roomId = 'room_' + i;
                    const roomNode = {
                        id: roomId,
                        x: centerX + Math.cos(angle) * roomRadius + (Math.random() - 0.5) * 20,
                        y: centerY + Math.sin(angle) * roomRadius + (Math.random() - 0.5) * 20,
                        vx: 0,
                        vy: 0,
                        radius: baseRadius,
                        baseRadius: baseRadius,
                        label: roomName,
                        color: '#4A90D9',
                        type: 'room',
                        fixed: false,
                        connections: roomLearnings + 1 + entryPoints.length
                    };
                    this.nodes.push(roomNode);
                    roomNodes.push({ node: roomNode, entryPoints: entryPoints });

                    // Edge strength based on room activity
                    const strength = 0.5 + Math.min(roomLearnings * 0.1, 0.5);
                    this.edges.push({
                        from: 'palace',
                        to: roomId,
                        strength: strength,
                        width: 1.5 + strength * 2
                    });
                    palaceNode.connections++;
                }

                // Entry point nodes - connected to their rooms (files vs folders)
                roomNodes.forEach((roomInfo, roomIdx) => {
                    const parent = roomInfo.node;
                    const entryPoints = roomInfo.entryPoints;

                    entryPoints.forEach((filePath, fileIdx) => {
                        const isFolder = filePath.endsWith('/');
                        const pathParts = filePath.split('/').filter(p => p);
                        const displayName = isFolder ? pathParts[pathParts.length - 1] + '/' : pathParts[pathParts.length - 1];
                        const angle = (fileIdx / Math.max(entryPoints.length, 1)) * Math.PI * 2 + Math.PI / 4;
                        const dist = parent.radius + 25 + (fileIdx % 2) * 10;

                        const nodeId = (isFolder ? 'folder_' : 'file_') + roomIdx + '_' + fileIdx;
                        this.nodes.push({
                            id: nodeId,
                            x: parent.x + Math.cos(angle) * dist + (Math.random() - 0.5) * 10,
                            y: parent.y + Math.sin(angle) * dist + (Math.random() - 0.5) * 10,
                            vx: 0,
                            vy: 0,
                            radius: isFolder ? 10 : 8,
                            baseRadius: isFolder ? 10 : 8,
                            label: displayName,
                            fullPath: filePath,
                            color: isFolder ? '#5DADE2' : '#F7F5F2',
                            type: isFolder ? 'folder' : 'file',
                            fixed: false,
                            connections: 1
                        });
                        this.edges.push({
                            from: parent.id,
                            to: nodeId,
                            strength: 0.6,
                            width: 1.2
                        });
                    });
                });

                // Learning nodes - size based on confidence
                const learningCount = Math.min(learningData.length, 30);
                for (let i = 0; i < learningCount; i++) {
                    const learning = learningData[i];
                    const parentIdx = 1 + (i % Math.max(1, Math.min(roomCount, 12)));
                    const parent = this.nodes[parentIdx] || this.nodes[0];
                    const angle = (i * 137.5 * Math.PI / 180); // Golden angle for distribution
                    const dist = 50 + (i % 4) * 15;

                    const confidence = learning.confidence || 0.5;
                    const baseRadius = 4 + confidence * 6;

                    const learnId = 'learn_' + i;
                    this.nodes.push({
                        id: learnId,
                        x: parent.x + Math.cos(angle) * dist + (Math.random() - 0.5) * 30,
                        y: parent.y + Math.sin(angle) * dist + (Math.random() - 0.5) * 30,
                        vx: 0,
                        vy: 0,
                        radius: baseRadius,
                        baseRadius: baseRadius,
                        color: '#10B981',
                        type: 'learning',
                        confidence: confidence,
                        fixed: false,
                        connections: 1
                    });
                    this.edges.push({
                        from: parent.id,
                        to: learnId,
                        strength: confidence,
                        width: 0.5 + confidence
                    });
                }

                // Corridor connections
                const corridorCount = stats.corridor?.linkedWorkspaces || 0;
                for (let i = 0; i < Math.min(corridorCount, 6); i++) {
                    const angle = Math.PI + (i - (corridorCount - 1) / 2) * 0.5;
                    const dist = roomRadius * 1.3;
                    const corrId = 'corridor_' + i;

                    this.nodes.push({
                        id: corrId,
                        x: centerX + Math.cos(angle) * dist + (Math.random() - 0.5) * 20,
                        y: centerY + Math.sin(angle) * dist + (Math.random() - 0.5) * 20,
                        vx: 0,
                        vy: 0,
                        radius: 12,
                        baseRadius: 12,
                        label: 'Link ' + (i + 1),
                        color: '#F59E0B',
                        type: 'corridor',
                        fixed: false,
                        connections: 1
                    });
                    this.edges.push({
                        from: 'palace',
                        to: corrId,
                        strength: 0.3,
                        width: 1.5,
                        dashed: true
                    });
                    palaceNode.connections++;
                }

                // Build node map for O(1) lookups in render()
                this.nodeMap = new Map();
                this.nodes.forEach(n => this.nodeMap.set(n.id, n));

                // Initialize particles on edges
                this.initParticles();

                // Center the view
                this.offsetX = 0;
                this.offsetY = 0;
            }

            initParticles() {
                this.particles = [];
                this.edges.forEach((edge, idx) => {
                    // Fewer, more visible particles
                    const particleCount = Math.max(1, Math.ceil(edge.strength * 2));
                    for (let i = 0; i < particleCount; i++) {
                        this.particles.push({
                            edgeIndex: idx,
                            progress: i / particleCount, // Evenly spaced
                            speed: this.particleSpeed * (0.9 + Math.random() * 0.2),
                            size: 2 + edge.strength * 1.5
                        });
                    }
                });
            }

            startAnimation() {
                const animate = (time) => {
                    const deltaTime = time - this.lastTime;
                    this.lastTime = time;

                    if (this.physics.enabled && deltaTime < 100) {
                        this.updatePhysics(deltaTime);
                    }
                    this.updateParticles(deltaTime);
                    this.render();

                    this.animationId = requestAnimationFrame(animate);
                };
                this.animationId = requestAnimationFrame(animate);
            }

            updatePhysics(dt) {
                const nodes = this.nodes;
                const centerX = this.width / 2;
                const centerY = this.height / 2;

                // Apply forces
                for (let i = 0; i < nodes.length; i++) {
                    const node = nodes[i];
                    if (node.fixed) continue;

                    let fx = 0, fy = 0;

                    // Center gravity (stronger for palace)
                    const gravityStrength = node.type === 'palace' ? this.physics.centerGravity * 3 : this.physics.centerGravity;
                    fx += (centerX - node.x) * gravityStrength;
                    fy += (centerY - node.y) * gravityStrength;

                    // Repulsion from other nodes
                    for (let j = 0; j < nodes.length; j++) {
                        if (i === j) continue;
                        const other = nodes[j];
                        const dx = node.x - other.x;
                        const dy = node.y - other.y;
                        const distSq = dx * dx + dy * dy;
                        const minDist = (node.radius + other.radius) * 2;

                        if (distSq < minDist * minDist * 4) {
                            const dist = Math.sqrt(distSq) || 1;
                            const force = this.physics.repulsion / distSq;
                            fx += (dx / dist) * force;
                            fy += (dy / dist) * force;
                        }
                    }

                    // Attraction along edges (use nodeMap for O(1) lookups)
                    this.edges.forEach(edge => {
                        let other = null;
                        if (edge.from === node.id) {
                            other = this.nodeMap.get(edge.to);
                        } else if (edge.to === node.id) {
                            other = this.nodeMap.get(edge.from);
                        }
                        if (other) {
                            const dx = other.x - node.x;
                            const dy = other.y - node.y;
                            const dist = Math.sqrt(dx * dx + dy * dy) || 1;
                            const targetDist = (node.radius + other.radius) * 3;
                            const force = (dist - targetDist) * this.physics.attraction * edge.strength;
                            fx += (dx / dist) * force;
                            fy += (dy / dist) * force;
                        }
                    });

                    // Update velocity with damping
                    node.vx = (node.vx + fx) * this.physics.damping;
                    node.vy = (node.vy + fy) * this.physics.damping;

                    // Stop if very slow
                    if (Math.abs(node.vx) < this.physics.minVelocity) node.vx = 0;
                    if (Math.abs(node.vy) < this.physics.minVelocity) node.vy = 0;

                    // Update position
                    node.x += node.vx;
                    node.y += node.vy;
                }
            }

            updateParticles(dt) {
                this.particles.forEach(p => {
                    p.progress += p.speed * (dt || 16);
                    if (p.progress > 1) {
                        p.progress -= 1;
                    }
                });
            }

            render() {
                const ctx = this.ctx;
                ctx.save();

                // Clear
                ctx.setTransform(this.dpr, 0, 0, this.dpr, 0, 0);
                ctx.fillStyle = '#1a1b2e';
                ctx.fillRect(0, 0, this.width, this.height);

                // Apply transform
                ctx.translate(this.offsetX, this.offsetY);
                ctx.scale(this.scale, this.scale);

                // Find connected nodes for hover highlighting
                const connectedIds = new Set();
                if (this.hoveredNode) {
                    connectedIds.add(this.hoveredNode.id);
                    this.edges.forEach(e => {
                        if (e.from === this.hoveredNode.id) connectedIds.add(e.to);
                        if (e.to === this.hoveredNode.id) connectedIds.add(e.from);
                    });
                }

                // Draw edges with bezier curves
                ctx.lineCap = 'round';
                this.edges.forEach((edge, idx) => {
                    const from = this.nodeMap.get(edge.from);
                    const to = this.nodeMap.get(edge.to);
                    if (!from || !to) return;

                    const isHighlighted = this.hoveredNode &&
                        (edge.from === this.hoveredNode.id || edge.to === this.hoveredNode.id);
                    const isDimmed = this.hoveredNode && !isHighlighted;

                    // Calculate control point for bezier curve
                    const midX = (from.x + to.x) / 2;
                    const midY = (from.y + to.y) / 2;
                    const dx = to.x - from.x;
                    const dy = to.y - from.y;
                    const dist = Math.sqrt(dx * dx + dy * dy);

                    // Perpendicular offset for curve
                    const curvature = 0.15;
                    const cpX = midX + (-dy / dist) * dist * curvature;
                    const cpY = midY + (dx / dist) * dist * curvature;

                    // Draw edge
                    ctx.beginPath();
                    ctx.moveTo(from.x, from.y);
                    ctx.quadraticCurveTo(cpX, cpY, to.x, to.y);

                    let alpha = isDimmed ? 0.1 : (isHighlighted ? 0.6 : 0.25);
                    ctx.strokeStyle = 'rgba(107, 91, 149, ' + alpha + ')';
                    ctx.lineWidth = edge.width * (isHighlighted ? 1.5 : 1);

                    if (edge.dashed) {
                        ctx.setLineDash([6, 4]);
                    } else {
                        ctx.setLineDash([]);
                    }
                    ctx.stroke();
                    ctx.setLineDash([]);

                    // Store curve info for particles
                    edge._curve = { from, to, cpX, cpY };
                });

                // Draw particles flowing along edges with comet tail for direction
                if (!this.hoveredNode) {
                    this.particles.forEach(p => {
                        const edge = this.edges[p.edgeIndex];
                        if (!edge || !edge._curve) return;

                        const { from, to, cpX, cpY } = edge._curve;

                        // Draw comet tail (multiple fading circles behind)
                        const tailLength = 5;
                        for (let i = tailLength; i >= 0; i--) {
                            const tailT = Math.max(0, p.progress - i * 0.015);
                            const tx = (1-tailT)*(1-tailT)*from.x + 2*(1-tailT)*tailT*cpX + tailT*tailT*to.x;
                            const ty = (1-tailT)*(1-tailT)*from.y + 2*(1-tailT)*tailT*cpY + tailT*tailT*to.y;

                            const alpha = (1 - i / tailLength) * (0.3 + edge.strength * 0.4);
                            const size = p.size * (1 - i * 0.12);

                            ctx.beginPath();
                            ctx.arc(tx, ty, Math.max(0.5, size), 0, Math.PI * 2);
                            ctx.fillStyle = 'rgba(139, 123, 181, ' + alpha + ')';
                            ctx.fill();
                        }

                        // Draw bright head
                        const t = p.progress;
                        const x = (1-t)*(1-t)*from.x + 2*(1-t)*t*cpX + t*t*to.x;
                        const y = (1-t)*(1-t)*from.y + 2*(1-t)*t*cpY + t*t*to.y;

                        ctx.beginPath();
                        ctx.arc(x, y, p.size * 1.2, 0, Math.PI * 2);
                        ctx.fillStyle = 'rgba(167, 155, 201, ' + (0.6 + edge.strength * 0.3) + ')';
                        ctx.fill();
                    });
                }

                // Draw nodes
                this.nodes.forEach(node => {
                    const isHovered = this.hoveredNode === node;
                    const isConnected = connectedIds.has(node.id);
                    const isDimmed = this.hoveredNode && !isConnected;

                    // Calculate display radius (pulse effect for hovered)
                    let displayRadius = node.radius;
                    if (isHovered) {
                        displayRadius = node.radius * 1.15;
                    }

                    // Outer glow for important nodes
                    if ((node.type === 'palace' || isHovered) && !isDimmed) {
                        ctx.beginPath();
                        ctx.arc(node.x, node.y, displayRadius + 8, 0, Math.PI * 2);
                        ctx.fillStyle = 'rgba(' + this.hexToRgb(node.color) + ', 0.15)';
                        ctx.fill();
                    }

                    // Node circle
                    ctx.beginPath();
                    ctx.arc(node.x, node.y, displayRadius, 0, Math.PI * 2);
                    ctx.fillStyle = isDimmed ? this.dimColor(node.color) : node.color;
                    ctx.fill();

                    // Border
                    ctx.strokeStyle = isDimmed ? 'rgba(255, 255, 255, 0.05)' :
                                      isHovered ? 'rgba(255, 255, 255, 0.5)' : 'rgba(255, 255, 255, 0.15)';
                    ctx.lineWidth = isHovered ? 2.5 : 1.5;
                    ctx.stroke();

                    // Inner highlight
                    if (!isDimmed && node.type !== 'learning') {
                        ctx.beginPath();
                        ctx.arc(node.x - displayRadius * 0.3, node.y - displayRadius * 0.3, displayRadius * 0.25, 0, Math.PI * 2);
                        ctx.fillStyle = 'rgba(255, 255, 255, 0.2)';
                        ctx.fill();
                    }

                    // Label
                    if (node.label && this.scale > 0.4 && !isDimmed) {
                        ctx.fillStyle = isHovered ? '#FFFFFF' : '#F7F5F2';
                        ctx.font = (node.type === 'palace' ? 'bold 12px' : '10px') + ' system-ui, sans-serif';
                        ctx.textAlign = 'center';
                        ctx.textBaseline = 'top';
                        ctx.fillText(node.label, node.x, node.y + displayRadius + 5);
                    }
                });

                // Draw tooltip for hovered node
                if (this.hoveredNode && this.scale > 0.3) {
                    this.drawTooltip(ctx, this.hoveredNode);
                }

                ctx.restore();

                // Draw minimap
                this.drawMinimap();
            }

            drawTooltip(ctx, node) {
                const padding = 8;
                const lineHeight = 14;
                let lines = [node.label || node.id];

                if (node.type === 'file' && node.fullPath) {
                    lines.push(node.fullPath);
                }
                if (node.type === 'folder' && node.fullPath) {
                    lines.push('Folder: ' + node.fullPath);
                }
                if (node.type === 'learning' && node.confidence !== undefined) {
                    lines.push('Confidence: ' + Math.round(node.confidence * 100) + '%');
                }
                if (node.type === 'room') {
                    lines.push('Entry points: ' + (node.connections - 1));
                }
                if (node.connections > 1 && node.type !== 'room') {
                    lines.push('Connections: ' + node.connections);
                }

                const maxWidth = Math.max(...lines.map(l => ctx.measureText(l).width));
                const boxWidth = maxWidth + padding * 2;
                const boxHeight = lines.length * lineHeight + padding * 2;

                const tooltipX = node.x + node.radius + 10;
                const tooltipY = node.y - boxHeight / 2;

                // Background
                ctx.fillStyle = 'rgba(37, 38, 64, 0.95)';
                ctx.beginPath();
                ctx.roundRect(tooltipX, tooltipY, boxWidth, boxHeight, 4);
                ctx.fill();

                // Border
                ctx.strokeStyle = 'rgba(107, 91, 149, 0.5)';
                ctx.lineWidth = 1;
                ctx.stroke();

                // Text
                ctx.fillStyle = '#F7F5F2';
                ctx.font = '10px system-ui, sans-serif';
                ctx.textAlign = 'left';
                ctx.textBaseline = 'top';
                lines.forEach((line, i) => {
                    ctx.fillText(line, tooltipX + padding, tooltipY + padding + i * lineHeight);
                });
            }

            drawMinimap() {
                const ctx = this.ctx;
                const mapSize = 100;
                const mapPadding = 10;
                const mapX = this.width - mapSize - mapPadding;
                const mapY = this.height - mapSize - mapPadding;

                // Find bounds of all nodes
                if (this.nodes.length === 0) return;

                let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
                this.nodes.forEach(n => {
                    minX = Math.min(minX, n.x - n.radius);
                    minY = Math.min(minY, n.y - n.radius);
                    maxX = Math.max(maxX, n.x + n.radius);
                    maxY = Math.max(maxY, n.y + n.radius);
                });

                const graphWidth = maxX - minX || 1;
                const graphHeight = maxY - minY || 1;
                const mapScale = Math.min((mapSize - 10) / graphWidth, (mapSize - 10) / graphHeight);

                // Background
                ctx.fillStyle = 'rgba(26, 27, 46, 0.9)';
                ctx.fillRect(mapX, mapY, mapSize, mapSize);
                ctx.strokeStyle = 'rgba(107, 91, 149, 0.3)';
                ctx.lineWidth = 1;
                ctx.strokeRect(mapX, mapY, mapSize, mapSize);

                // Draw nodes on minimap
                this.nodes.forEach(node => {
                    const mx = mapX + 5 + (node.x - minX) * mapScale;
                    const my = mapY + 5 + (node.y - minY) * mapScale;
                    const mr = Math.max(2, node.radius * mapScale * 0.5);

                    ctx.beginPath();
                    ctx.arc(mx, my, mr, 0, Math.PI * 2);
                    ctx.fillStyle = node.color;
                    ctx.fill();
                });

                // Draw viewport rectangle
                const viewLeft = -this.offsetX / this.scale;
                const viewTop = -this.offsetY / this.scale;
                const viewWidth = this.width / this.scale;
                const viewHeight = this.height / this.scale;

                const vx = mapX + 5 + (viewLeft - minX) * mapScale;
                const vy = mapY + 5 + (viewTop - minY) * mapScale;
                const vw = viewWidth * mapScale;
                const vh = viewHeight * mapScale;

                ctx.strokeStyle = 'rgba(247, 245, 242, 0.6)';
                ctx.lineWidth = 1;
                ctx.strokeRect(
                    Math.max(mapX, Math.min(mapX + mapSize - vw, vx)),
                    Math.max(mapY, Math.min(mapY + mapSize - vh, vy)),
                    Math.min(vw, mapSize),
                    Math.min(vh, mapSize)
                );
            }

            hexToRgb(hex) {
                const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
                return result ?
                    parseInt(result[1], 16) + ', ' + parseInt(result[2], 16) + ', ' + parseInt(result[3], 16) :
                    '107, 91, 149';
            }

            dimColor(hex) {
                const rgb = this.hexToRgb(hex);
                return 'rgba(' + rgb + ', 0.2)';
            }

            destroy() {
                if (this.animationId) {
                    cancelAnimationFrame(this.animationId);
                }
            }
        }

        // Dashboard
        class Dashboard {
            constructor() {
                const container = document.getElementById('neural-container');
                const canvas = document.getElementById('neural-canvas');
                this.neuralMap = new NeuralMap(container, canvas);
                this.initWorkspaceSwitcher();
            }

            initWorkspaceSwitcher() {
                const switcher = document.getElementById('workspace-switcher');
                const current = document.getElementById('workspace-current');

                // Toggle dropdown
                current.addEventListener('click', (e) => {
                    e.stopPropagation();
                    switcher.classList.toggle('open');
                });

                // Close dropdown when clicking outside
                document.addEventListener('click', () => {
                    switcher.classList.remove('open');
                });

                // Prevent dropdown clicks from closing
                document.getElementById('workspace-dropdown').addEventListener('click', (e) => {
                    e.stopPropagation();
                });
            }

            async loadWorkspaces() {
                try {
                    const res = await fetch('/api/workspaces');
                    const data = await res.json();
                    const workspaces = data.workspaces || [];

                    // Update current workspace name in switcher
                    const current = workspaces.find(w => w.isCurrent);
                    if (current) {
                        document.getElementById('workspace-switcher-name').textContent = current.name;
                    }

                    // Render workspace options
                    const container = document.getElementById('workspace-options');
                    container.innerHTML = workspaces.map(w => {
                        const activeClass = w.isCurrent ? ' active' : '';
                        const statusHtml = w.isCurrent ?
                            '<span class="workspace-option-status current">Current</span>' :
                            (!w.hasPalace ? '<span class="workspace-option-status no-palace">No Palace</span>' : '');

                        return '<div class="workspace-option' + activeClass + '" data-path="' + this.escAttr(w.path) + '">' +
                            '<div class="workspace-option-icon">' + w.name.charAt(0).toUpperCase() + '</div>' +
                            '<div class="workspace-option-info">' +
                                '<div class="workspace-option-name">' + this.esc(w.name) + '</div>' +
                                '<div class="workspace-option-path">' + this.esc(w.path) + '</div>' +
                            '</div>' +
                            statusHtml +
                        '</div>';
                    }).join('');

                    // Add click handlers
                    container.querySelectorAll('.workspace-option').forEach(opt => {
                        opt.addEventListener('click', () => {
                            const path = opt.dataset.path;
                            if (path && !opt.classList.contains('active')) {
                                this.switchWorkspace(path);
                            }
                        });
                    });
                } catch (err) {
                    console.error('Failed to load workspaces:', err);
                }
            }

            async switchWorkspace(path) {
                const switcher = document.getElementById('workspace-switcher');
                switcher.classList.remove('open');

                // Show loading state
                document.getElementById('workspace-switcher-name').textContent = 'Switching...';

                try {
                    const res = await fetch('/api/workspace/switch', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ path: path })
                    });

                    const data = await res.json();

                    if (data.success) {
                        // Reload all data for the new workspace
                        await this.load();
                        await this.loadWorkspaces();
                    } else {
                        alert('Failed to switch workspace: ' + (data.error || 'Unknown error'));
                        await this.loadWorkspaces(); // Reset name
                    }
                } catch (err) {
                    console.error('Failed to switch workspace:', err);
                    alert('Failed to switch workspace: ' + err.message);
                    await this.loadWorkspaces();
                }
            }

            async load() {
                try {
                    const [statsRes, sessionsRes, learningsRes, corridorsRes, hotspotsRes, roomsRes] = await Promise.all([
                        fetch('/api/stats'),
                        fetch('/api/sessions'),
                        fetch('/api/learnings'),
                        fetch('/api/corridors'),
                        fetch('/api/hotspots'),
                        fetch('/api/rooms')
                    ]);

                    const stats = await statsRes.json();
                    const sessionsData = await sessionsRes.json();
                    const learningsData = await learningsRes.json();
                    const corridorsData = await corridorsRes.json();
                    const hotspotsData = await hotspotsRes.json();
                    const roomsData = await roomsRes.json();

                    const sessions = sessionsData.sessions || [];
                    const learnings = learningsData.learnings || [];
                    const corridors = corridorsData.links || [];
                    const hotspots = hotspotsData.hotspots || [];
                    const rooms = roomsData.rooms || [];

                    this.updateWorkspace(stats.workspace);
                    this.updateStats(stats);
                    this.updateSessions(sessions);
                    this.updateLearnings(learnings);
                    this.updateCorridors(corridors, stats.workspace?.name);
                    this.updateHotspots(hotspots);
                    this.neuralMap.setData(stats, rooms, learnings);
                } catch (err) {
                    console.error('Failed to load:', err);
                    document.getElementById('sessions-list').innerHTML = '<div class="empty-state"><p>Failed to load</p></div>';
                    document.getElementById('learnings-list').innerHTML = '<div class="empty-state"><p>Failed to load</p></div>';
                    document.getElementById('corridors-list').innerHTML = '<div class="empty-state"><p>Failed to load</p></div>';
                    document.getElementById('hotspots-list').innerHTML = '<div class="empty-state"><p>Failed to load</p></div>';
                }
            }

            escAttr(text) {
                return (text || '').replace(/"/g, '&quot;').replace(/'/g, '&#39;');
            }

            updateWorkspace(workspace) {
                if (!workspace) {
                    document.getElementById('workspace-name').textContent = 'Unknown Workspace';
                    return;
                }

                document.getElementById('workspace-name').textContent = workspace.name || 'Workspace';
                document.getElementById('workspace-path').textContent = workspace.path || '';

                // Last scan
                if (workspace.lastScan) {
                    const d = new Date(workspace.lastScan);
                    const ago = this.timeAgo(d);
                    document.getElementById('last-scan').textContent = ago;
                } else {
                    document.getElementById('last-scan').textContent = 'Never';
                }

                // Files indexed
                document.getElementById('files-indexed').textContent = workspace.fileCount || 0;

                // Status
                const statusEl = document.getElementById('index-status');
                const status = workspace.status || 'unknown';
                statusEl.textContent = status.charAt(0).toUpperCase() + status.slice(1);
                statusEl.className = 'meta-value status-' + status;
            }

            updateStats(stats) {
                document.getElementById('stat-rooms').textContent = stats.rooms || 0;
                document.getElementById('stat-sessions').textContent = stats.sessions?.total || 0;
                document.getElementById('stat-learnings').textContent = stats.learnings || 0;
                document.getElementById('stat-files').textContent = stats.filesTracked || 0;
                document.getElementById('stat-corridors').textContent = stats.corridor?.linkedWorkspaces || 0;
            }

            updateSessions(sessions) {
                const container = document.getElementById('sessions-list');
                if (!sessions.length) {
                    container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">S</div><p>No sessions yet</p><p style="font-size:0.75rem;margin-top:0.5rem;color:#718096">palace session start</p></div>';
                    return;
                }

                container.innerHTML = sessions.slice(0, 5).map(s => {
                    const agentType = s.agentType || 'unknown';
                    const icon = agentType.includes('claude') ? 'C' : 'A';
                    return '<div class="session-item">' +
                        '<div class="session-icon">' + icon + '</div>' +
                        '<div class="session-content">' +
                            '<div class="session-agent">' + this.esc(agentType) + '</div>' +
                            '<div class="session-goal">' + this.esc(s.goal || 'No goal') + '</div>' +
                            '<span class="session-state ' + s.state + '">' + s.state + '</span>' +
                        '</div>' +
                    '</div>';
                }).join('');
            }

            updateLearnings(learnings) {
                const container = document.getElementById('learnings-list');
                if (!learnings.length) {
                    container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">L</div><p>No learnings yet</p><p style="font-size:0.75rem;margin-top:0.5rem;color:#718096">palace learn "insight"</p></div>';
                    return;
                }

                container.innerHTML = learnings.slice(0, 6).map(l => {
                    const conf = Math.round((l.confidence || 0.5) * 100);
                    return '<div class="learning-item">' +
                        '<div class="learning-content">' + this.esc(l.content) + '</div>' +
                        '<div class="learning-meta">' +
                            '<span class="learning-scope">' + (l.scope || 'palace') + '</span>' +
                            '<div class="confidence-bar"><div class="confidence-fill" style="width:' + conf + '%"></div></div>' +
                            '<span>' + conf + '%</span>' +
                        '</div>' +
                    '</div>';
                }).join('');
            }

            updateCorridors(corridors, currentWorkspace) {
                const container = document.getElementById('corridors-list');

                // Add current workspace as first item
                let items = [];
                if (currentWorkspace) {
                    items.push('<div class="corridor-item" style="border: 1px solid var(--palace-purple);">' +
                        '<div class="corridor-icon" style="background: var(--palace-purple);">P</div>' +
                        '<div class="corridor-content">' +
                            '<div class="corridor-name">' + this.esc(currentWorkspace) + ' (Current)</div>' +
                            '<div class="corridor-path">This workspace</div>' +
                        '</div>' +
                    '</div>');
                }

                if (!corridors.length && !currentWorkspace) {
                    container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">C</div><p>No corridors linked</p><p style="font-size:0.75rem;margin-top:0.5rem;color:#718096">palace corridor link name path</p></div>';
                    return;
                }

                corridors.forEach(c => {
                    items.push('<div class="corridor-item">' +
                        '<div class="corridor-icon">C</div>' +
                        '<div class="corridor-content">' +
                            '<div class="corridor-name">' + this.esc(c.name) + '</div>' +
                            '<div class="corridor-path">' + this.esc(c.path) + '</div>' +
                        '</div>' +
                    '</div>');
                });

                container.innerHTML = items.join('');
            }

            updateHotspots(hotspots) {
                const container = document.getElementById('hotspots-list');
                if (!hotspots || !hotspots.length) {
                    container.innerHTML = '<div class="empty-state"><div class="empty-state-icon">H</div><p>No file activity yet</p><p style="font-size:0.75rem;margin-top:0.5rem;color:#718096">Edit files to track hotspots</p></div>';
                    return;
                }

                container.innerHTML = hotspots.slice(0, 8).map(h => {
                    const editCount = h.editCount || 0;
                    const heatClass = editCount > 10 ? 'hot' : (editCount > 5 ? 'warm' : '');
                    const lastEdit = h.lastEdited ? this.timeAgo(new Date(h.lastEdited)) : 'Never';
                    const fileName = h.path.split('/').pop();
                    const dir = h.path.substring(0, h.path.length - fileName.length - 1) || '/';

                    return '<div class="hotspot-item">' +
                        '<div class="hotspot-icon ' + heatClass + '">F</div>' +
                        '<div class="hotspot-content">' +
                            '<div class="hotspot-path">' + this.esc(fileName) + '</div>' +
                            '<div class="hotspot-meta">' +
                                '<span>' + this.esc(dir) + '</span>' +
                                '<span>' + editCount + ' edits</span>' +
                                '<span>' + lastEdit + '</span>' +
                            '</div>' +
                        '</div>' +
                    '</div>';
                }).join('');
            }

            timeAgo(date) {
                const seconds = Math.floor((new Date() - date) / 1000);
                if (seconds < 60) return 'Just now';
                if (seconds < 3600) return Math.floor(seconds / 60) + 'm ago';
                if (seconds < 86400) return Math.floor(seconds / 3600) + 'h ago';
                if (seconds < 604800) return Math.floor(seconds / 86400) + 'd ago';
                return date.toLocaleDateString();
            }

            esc(text) {
                const d = document.createElement('div');
                d.textContent = text || '';
                return d.innerHTML;
            }
        }

        // Init
        document.addEventListener('DOMContentLoaded', () => {
            const dashboard = new Dashboard();
            dashboard.load();
            dashboard.loadWorkspaces();
            setInterval(() => dashboard.load(), 30000);
        });
    })();
    </script>
</body>
</html>
`)

func (f *fallbackFile) Read(b []byte) (int, error) {
	if f.offset >= len(fallbackHTML) {
		return 0, io.EOF
	}
	n := copy(b, fallbackHTML[f.offset:])
	f.offset += n
	return n, nil
}

func (f *fallbackFile) Stat() (fs.FileInfo, error) {
	return &fallbackFileInfo{}, nil
}

func (f *fallbackFile) Close() error {
	return nil
}

type fallbackFileInfo struct{}

func (fi *fallbackFileInfo) Name() string       { return "index.html" }
func (fi *fallbackFileInfo) Size() int64        { return int64(len(fallbackHTML)) }
func (fi *fallbackFileInfo) Mode() fs.FileMode  { return 0444 }
func (fi *fallbackFileInfo) ModTime() time.Time { return time.Now() }
func (fi *fallbackFileInfo) IsDir() bool        { return false }
func (fi *fallbackFileInfo) Sys() any           { return nil }
