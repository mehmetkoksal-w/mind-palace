import * as fs from 'fs';
import { parse as parseJSONC } from 'jsonc-parser';
import * as path from 'path';
import * as vscode from 'vscode';
import { PalaceBridge } from './bridge';

interface RoomConfig {
    name: string;
    summary?: string;
    description?: string;
    entryPoints?: string[];
    doNotTouchGlob?: string[];
}

interface GraphNode {
    data: {
        id: string;
        label: string;
        type: 'room' | 'file' | 'ghost';
        parent?: string;
        fullPath?: string;
        description?: string;
        snippet?: string;
        lineNumber?: number;
    };
}

interface GraphEdge {
    data: {
        id: string;
        source: string;
        target: string;
    };
}

interface GraphData {
    nodes: GraphNode[];
    edges: GraphEdge[];
}

// Tree View data structures
interface TreeRoom {
    name: string;
    description?: string;
    files: TreeFile[];
    expanded: boolean;
}

interface TreeFile {
    name: string;
    fullPath: string;
    snippet?: string;
    lineNumber?: number;
    isMatch?: boolean;
}

export class PalaceSidebarProvider implements vscode.WebviewViewProvider {
    public static readonly viewType = 'mindPalace.blueprintView';

    private _view?: vscode.WebviewView;
    private _bridge?: PalaceBridge;
    private _searchDebounceTimer?: NodeJS.Timeout;
    private _lastSearchQuery = '';
    private _isSearchMode = false;

    constructor(private readonly _extensionUri: vscode.Uri) { }

    /**
     * Set the bridge for MCP communication
     */
    public setBridge(bridge: PalaceBridge): void {
        this._bridge = bridge;

        // Update UI when connection status changes
        bridge.onConnectionChange((connected) => {
            this._view?.webview.postMessage({
                command: 'connectionStatus',
                connected
            });
        });
    }

    public resolveWebviewView(
        webviewView: vscode.WebviewView,
        _context: vscode.WebviewViewResolveContext,
        _token: vscode.CancellationToken
    ) {
        this._view = webviewView;

        webviewView.webview.options = {
            enableScripts: true,
            localResourceRoots: [this._extensionUri],
        };

        webviewView.webview.html = this._getHtmlForWebview(webviewView.webview);

        webviewView.webview.onDidReceiveMessage(async (data) => {
            switch (data.command) {
                case 'openFile':
                    await this._openFile(data.filePath, data.lineNumber);
                    break;
                case 'refresh':
                    await this.refresh();
                    break;
                case 'ready':
                    await this.refresh();
                    // Try to connect to MCP on startup
                    this._tryConnectMCP();
                    break;
                case 'search':
                    await this._handleSearch(data.query);
                    break;
                case 'clearSearch':
                    this._clearSearch();
                    break;
            }
        });
    }

    private async _tryConnectMCP(): Promise<void> {
        if (!this._bridge) return;

        try {
            await this._bridge.connectMCP();
            this._view?.webview.postMessage({
                command: 'connectionStatus',
                connected: true
            });
        } catch (error) {
            console.error('Failed to connect to MCP:', error);
            this._view?.webview.postMessage({
                command: 'connectionStatus',
                connected: false
            });
        }
    }

    private async _handleSearch(query: string): Promise<void> {
        // Debounce search
        if (this._searchDebounceTimer) {
            clearTimeout(this._searchDebounceTimer);
        }

        // If empty query, clear search
        if (!query.trim()) {
            this._clearSearch();
            return;
        }

        this._searchDebounceTimer = setTimeout(async () => {
            await this._performSearch(query.trim());
        }, 300); // 300ms debounce
    }

    private async _performSearch(query: string): Promise<void> {
        if (!this._bridge) {
            vscode.window.showErrorMessage('Search not available: Bridge not initialized');
            return;
        }

        this._lastSearchQuery = query;
        this._isSearchMode = true;

        // Show searching state
        this._view?.webview.postMessage({
            command: 'searchState',
            state: 'searching',
            query
        });

        try {
            const results = await this._bridge.search(query);

            // Send results to webview
            this._view?.webview.postMessage({
                command: 'searchResults',
                results
            });
        } catch (error: any) {
            console.error('Search failed:', error);
            this._view?.webview.postMessage({
                command: 'searchState',
                state: 'error',
                error: error.message
            });
        }
    }

    private async _clearSearch(): Promise<void> {
        this._lastSearchQuery = '';
        this._isSearchMode = false;

        // Notify webview to exit search mode
        this._view?.webview.postMessage({
            command: 'clearSearchResults'
        });

        // Refresh the graph to normal state
        await this.refresh();
    }

    public async refresh(): Promise<void> {
        if (this._view) {
            const data = await this._buildGraphData();
            this._view.webview.postMessage({ command: 'updateGraph', data });
        }
    }

    private async _openFile(filePath: string, lineNumber?: number): Promise<void> {
        const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
        if (!workspaceRoot) {
            vscode.window.showErrorMessage('No workspace folder open.');
            return;
        }

        const absolutePath = path.join(workspaceRoot, filePath);
        try {
            const uri = vscode.Uri.file(absolutePath);
            const doc = await vscode.workspace.openTextDocument(uri);
            const editor = await vscode.window.showTextDocument(doc);

            // If we have a line number, scroll to it
            if (lineNumber && lineNumber > 0) {
                const line = lineNumber - 1; // Convert to 0-indexed
                const range = new vscode.Range(line, 0, line, 0);
                editor.selection = new vscode.Selection(range.start, range.end);
                editor.revealRange(range, vscode.TextEditorRevealType.InCenter);
            }
        } catch {
            vscode.window.showErrorMessage(`Could not open: ${filePath}`);
        }
    }

    private async _buildGraphData(): Promise<GraphData> {
        const nodes: GraphNode[] = [];
        const edges: GraphEdge[] = [];

        const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
        if (!workspaceRoot) {
            return { nodes, edges };
        }

        const roomsDir = path.join(workspaceRoot, '.palace', 'rooms');
        if (!fs.existsSync(roomsDir)) {
            return { nodes, edges };
        }

        const roomFiles = await vscode.workspace.findFiles('.palace/rooms/*.jsonc');

        for (const uri of roomFiles) {
            try {
                const content = fs.readFileSync(uri.fsPath, 'utf8');
                const config: RoomConfig = parseJSONC(content);
                const roomId = `room-${config.name || path.basename(uri.fsPath, '.jsonc')}`;
                const roomLabel = config.name || path.basename(uri.fsPath, '.jsonc');

                // Create Room (Parent) Node
                nodes.push({
                    data: {
                        id: roomId,
                        label: roomLabel,
                        type: 'room',
                        description: config.summary || config.description,
                    },
                });

                if (config.entryPoints && config.entryPoints.length > 0) {
                    // Create File (Child) Nodes for each entry point
                    for (const entryPoint of config.entryPoints) {
                        const fileId = `file-${roomId}-${entryPoint}`;
                        nodes.push({
                            data: {
                                id: fileId,
                                label: path.basename(entryPoint),
                                type: 'file',
                                parent: roomId,
                                fullPath: entryPoint,
                            },
                        });
                    }
                } else {
                    // Create Ghost (Placeholder) Node for empty rooms
                    const ghostId = `ghost-${roomId}`;
                    nodes.push({
                        data: {
                            id: ghostId,
                            label: 'empty',
                            type: 'ghost',
                            parent: roomId,
                        },
                    });
                }
            } catch (e) {
                console.error(`Error parsing room file: ${uri.fsPath}`, e);
            }
        }

        return { nodes, edges };
    }

    private _getHtmlForWebview(_webview: vscode.Webview): string {
        const nonce = this._getNonce();

        return /*html*/ `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; script-src 'nonce-${nonce}' https://cdnjs.cloudflare.com; connect-src https://cdnjs.cloudflare.com;">
    <title>Mind Palace Blueprint</title>
    <style>
        /* ===== CSS Reset & Base ===== */
        *, *::before, *::after {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        html, body {
            height: 100%;
            width: 100%;
            overflow: hidden;
            font-family: var(--vscode-font-family, system-ui, -apple-system, sans-serif);
            font-size: var(--vscode-font-size, 13px);
            background-color: var(--vscode-sideBar-background, #1e1e1e);
            color: var(--vscode-sideBar-foreground, #cccccc);
        }

        /* ===== Layout Container ===== */
        .container {
            display: flex;
            flex-direction: column;
            height: 100vh;
            width: 100%;
        }

        /* ===== Header Bar ===== */
        .header {
            display: flex;
            flex-direction: column;
            padding: 6px 10px;
            background-color: var(--vscode-sideBarSectionHeader-background, #252526);
            border-bottom: 1px solid var(--vscode-sideBarSectionHeader-border, rgba(128,128,128,0.2));
            flex-shrink: 0;
            gap: 8px;
        }

        .header-top {
            display: flex;
            align-items: center;
            justify-content: space-between;
        }

        .header-title {
            display: flex;
            align-items: center;
            gap: 6px;
            font-size: 11px;
            font-weight: 500;
            text-transform: uppercase;
            letter-spacing: 1px;
            color: var(--vscode-sideBarTitle-foreground, #9d9d9d);
        }

        .header-indicator {
            width: 6px;
            height: 6px;
            background-color: var(--vscode-charts-green, #89d185);
            border-radius: 50%;
            box-shadow: 0 0 6px var(--vscode-charts-green, #89d185);
            transition: all 0.3s ease;
        }

        .header-indicator.disconnected {
            background-color: var(--vscode-charts-orange, #d19a66);
            box-shadow: 0 0 6px var(--vscode-charts-orange, #d19a66);
        }

        .header-indicator.searching {
            background-color: var(--vscode-charts-blue, #4fc1ff);
            box-shadow: 0 0 8px var(--vscode-charts-blue, #4fc1ff);
            animation: pulse 1.5s ease-in-out infinite;
        }

        .header-indicator.search-active {
            background-color: var(--vscode-charts-purple, #c586c0);
            box-shadow: 0 0 10px var(--vscode-charts-purple, #c586c0);
        }

        @keyframes pulse {
            0%, 100% { opacity: 1; transform: scale(1); }
            50% { opacity: 0.6; transform: scale(1.2); }
        }

        .header-actions {
            display: flex;
            gap: 2px;
        }

        .icon-button {
            display: flex;
            align-items: center;
            justify-content: center;
            width: 24px;
            height: 24px;
            border: none;
            border-radius: 4px;
            background-color: transparent;
            color: var(--vscode-icon-foreground, #c5c5c5);
            cursor: pointer;
            opacity: 0.7;
            transition: opacity 0.15s ease, background-color 0.15s ease;
        }

        .icon-button:hover {
            opacity: 1;
            background-color: var(--vscode-toolbar-hoverBackground, rgba(90,93,94,0.31));
        }

        .icon-button:active {
            background-color: var(--vscode-toolbar-activeBackground, rgba(99,102,103,0.31));
        }

        .icon-button.active {
            opacity: 1;
            background-color: var(--vscode-toolbar-activeBackground, rgba(99,102,103,0.31));
            color: var(--vscode-focusBorder, #007fd4);
        }

        .icon-button svg {
            width: 14px;
            height: 14px;
            fill: currentColor;
        }

        /* ===== View Toggle ===== */
        .view-toggle {
            display: flex;
            gap: 2px;
            padding: 2px;
            background-color: var(--vscode-input-background, #3c3c3c);
            border-radius: 6px;
            margin-right: 8px;
        }

        .view-toggle .icon-button {
            width: 28px;
            height: 22px;
            border-radius: 4px;
        }

        .view-toggle .icon-button.active {
            background-color: var(--vscode-button-background, #0e639c);
            color: var(--vscode-button-foreground, #ffffff);
            opacity: 1;
        }

        /* ===== Search Bar ===== */
        .search-container {
            position: relative;
            display: flex;
            align-items: center;
        }

        .search-icon {
            position: absolute;
            left: 10px;
            width: 14px;
            height: 14px;
            fill: var(--vscode-input-placeholderForeground, #6b6b6b);
            pointer-events: none;
            transition: fill 0.2s ease;
        }

        .search-input {
            width: 100%;
            padding: 7px 32px 7px 32px;
            background-color: var(--vscode-input-background, #3c3c3c);
            border: 1px solid var(--vscode-input-border, transparent);
            border-radius: 6px;
            color: var(--vscode-input-foreground, #cccccc);
            font-size: 12px;
            font-family: inherit;
            outline: none;
            transition: all 0.2s ease;
        }

        .search-input::placeholder {
            color: var(--vscode-input-placeholderForeground, #6b6b6b);
        }

        .search-input:focus {
            border-color: var(--vscode-focusBorder, #007fd4);
            box-shadow: 0 0 0 1px var(--vscode-focusBorder, #007fd4) inset;
        }

        .search-input:focus + .search-icon {
            fill: var(--vscode-focusBorder, #007fd4);
        }

        .search-clear {
            position: absolute;
            right: 6px;
            width: 20px;
            height: 20px;
            padding: 0;
            background: none;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            display: none;
            align-items: center;
            justify-content: center;
            color: var(--vscode-input-placeholderForeground, #6b6b6b);
            transition: color 0.15s ease, background-color 0.15s ease;
        }

        .search-clear.visible {
            display: flex;
        }

        .search-clear:hover {
            color: var(--vscode-foreground, #cccccc);
            background-color: var(--vscode-toolbar-hoverBackground, rgba(90,93,94,0.31));
        }

        .search-clear svg {
            width: 12px;
            height: 12px;
            fill: currentColor;
        }

        /* ===== Search Status ===== */
        .search-status {
            display: none;
            align-items: center;
            gap: 6px;
            padding: 4px 8px;
            font-size: 10px;
            color: var(--vscode-descriptionForeground, #8b8b8b);
            background: var(--vscode-editor-background, #1e1e1e);
            border-radius: 4px;
            margin-top: 4px;
        }

        .search-status.visible {
            display: flex;
        }

        .search-status-badge {
            padding: 2px 6px;
            background: var(--vscode-charts-purple, #c586c0);
            color: white;
            border-radius: 10px;
            font-weight: 600;
            font-size: 9px;
        }

        /* ===== Main Content Area ===== */
        .content-area {
            flex: 1;
            position: relative;
            overflow: hidden;
        }

        /* ===== TREE VIEW ===== */
        .tree-view {
            display: none;
            flex-direction: column;
            height: 100%;
            overflow-y: auto;
            overflow-x: hidden;
            padding: 4px 0;
        }

        .tree-view.active {
            display: flex;
        }

        .tree-view::-webkit-scrollbar {
            width: 10px;
        }

        .tree-view::-webkit-scrollbar-track {
            background: transparent;
        }

        .tree-view::-webkit-scrollbar-thumb {
            background: var(--vscode-scrollbarSlider-background, rgba(121, 121, 121, 0.4));
            border-radius: 5px;
        }

        .tree-view::-webkit-scrollbar-thumb:hover {
            background: var(--vscode-scrollbarSlider-hoverBackground, rgba(100, 100, 100, 0.7));
        }

        /* Tree Room */
        .tree-room {
            margin-bottom: 2px;
        }

        .tree-room-header {
            display: flex;
            align-items: center;
            padding: 4px 8px 4px 4px;
            cursor: pointer;
            border-radius: 4px;
            margin: 0 4px;
            transition: background-color 0.1s ease;
            user-select: none;
        }

        .tree-room-header:hover {
            background-color: var(--vscode-list-hoverBackground, rgba(90, 93, 94, 0.31));
        }

        .tree-room-header.search-match {
            background-color: rgba(197, 134, 192, 0.15);
        }

        .tree-room-header.search-match:hover {
            background-color: rgba(197, 134, 192, 0.25);
        }

        .tree-room-header.ghost-mode {
            opacity: 0.35;
        }

        .tree-chevron {
            width: 16px;
            height: 16px;
            display: flex;
            align-items: center;
            justify-content: center;
            flex-shrink: 0;
            transition: transform 0.15s ease;
        }

        .tree-chevron svg {
            width: 10px;
            height: 10px;
            fill: var(--vscode-foreground, #cccccc);
            opacity: 0.6;
        }

        .tree-room.expanded .tree-chevron {
            transform: rotate(90deg);
        }

        .tree-room-icon {
            width: 16px;
            height: 16px;
            display: flex;
            align-items: center;
            justify-content: center;
            flex-shrink: 0;
            margin-right: 6px;
        }

        .tree-room-icon svg {
            width: 14px;
            height: 14px;
            fill: var(--vscode-charts-orange, #d19a66);
            opacity: 0.9;
        }

        .tree-room-header.search-match .tree-room-icon svg {
            fill: var(--vscode-charts-purple, #c586c0);
        }

        .tree-room-label {
            flex: 1;
            font-size: 12px;
            font-weight: 500;
            color: var(--vscode-foreground, #cccccc);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .tree-room-count {
            font-size: 10px;
            color: var(--vscode-descriptionForeground, #8b8b8b);
            padding: 1px 5px;
            background: var(--vscode-badge-background, #4d4d4d);
            border-radius: 8px;
            margin-left: 8px;
            flex-shrink: 0;
        }

        .tree-room-header.search-match .tree-room-count {
            background: var(--vscode-charts-purple, #c586c0);
            color: white;
        }

        /* Tree Files Container */
        .tree-files {
            display: none;
            padding-left: 20px;
        }

        .tree-room.expanded .tree-files {
            display: block;
        }

        /* Tree File */
        .tree-file {
            display: flex;
            align-items: center;
            padding: 3px 8px 3px 4px;
            cursor: pointer;
            border-radius: 4px;
            margin: 1px 4px 1px 0;
            transition: background-color 0.1s ease;
        }

        .tree-file:hover {
            background-color: var(--vscode-list-hoverBackground, rgba(90, 93, 94, 0.31));
        }

        .tree-file.search-match {
            background-color: rgba(197, 134, 192, 0.2);
        }

        .tree-file.search-match:hover {
            background-color: rgba(197, 134, 192, 0.3);
        }

        .tree-file.ghost-mode {
            opacity: 0.3;
        }

        .tree-file-icon {
            width: 16px;
            height: 16px;
            display: flex;
            align-items: center;
            justify-content: center;
            flex-shrink: 0;
            margin-right: 6px;
        }

        .tree-file-icon svg {
            width: 14px;
            height: 14px;
            fill: var(--vscode-charts-blue, #4fc1ff);
            opacity: 0.8;
        }

        .tree-file.search-match .tree-file-icon svg {
            fill: var(--vscode-charts-purple, #c586c0);
        }

        .tree-file-label {
            flex: 1;
            font-size: 12px;
            font-family: var(--vscode-editor-font-family, monospace);
            color: var(--vscode-foreground, #cccccc);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .tree-file.search-match .tree-file-label {
            color: var(--vscode-charts-purple, #c586c0);
            font-weight: 500;
        }

        .tree-file-line {
            font-size: 10px;
            color: var(--vscode-descriptionForeground, #8b8b8b);
            margin-left: 8px;
            flex-shrink: 0;
        }

        .tree-file.search-match .tree-file-line {
            color: var(--vscode-charts-purple, #c586c0);
            opacity: 0.8;
        }

        /* Tree Empty State */
        .tree-empty {
            display: flex;
            align-items: center;
            padding: 3px 8px 3px 4px;
            margin: 1px 4px 1px 0;
            opacity: 0.4;
            font-style: italic;
        }

        .tree-empty-icon {
            width: 16px;
            height: 16px;
            display: flex;
            align-items: center;
            justify-content: center;
            flex-shrink: 0;
            margin-right: 6px;
        }

        .tree-empty-icon svg {
            width: 12px;
            height: 12px;
            fill: var(--vscode-foreground, #cccccc);
            opacity: 0.5;
        }

        .tree-empty-label {
            font-size: 11px;
            color: var(--vscode-descriptionForeground, #8b8b8b);
        }

        /* ===== MAP VIEW (Cytoscape) ===== */
        .map-view {
            display: none;
            position: relative;
            height: 100%;
            width: 100%;
        }

        .map-view.active {
            display: block;
        }

        .canvas-wrapper {
            height: 100%;
            width: 100%;
            position: relative;
            overflow: hidden;
            background-color: var(--vscode-editor-background, #1e1e1e);
        }

        /* Subtle dot grid pattern - normal mode */
        .canvas-wrapper::before {
            content: '';
            position: absolute;
            inset: 0;
            pointer-events: none;
            opacity: 0.35;
            background-image: radial-gradient(
                var(--vscode-editorLineNumber-foreground, #5a5a5a) 1px,
                transparent 1px
            );
            background-size: 24px 24px;
            transition: all 0.4s ease;
        }

        /* ===== TACTICAL HUD MODE ===== */
        .canvas-wrapper.search-mode::before {
            opacity: 0.2;
            background-image:
                linear-gradient(to right, var(--vscode-charts-purple, #c586c0) 1px, transparent 1px),
                linear-gradient(to bottom, var(--vscode-charts-purple, #c586c0) 1px, transparent 1px);
            background-size: 40px 40px;
        }

        .canvas-wrapper.search-mode::after {
            content: '';
            position: absolute;
            inset: 0;
            pointer-events: none;
            background: radial-gradient(
                ellipse at center,
                transparent 0%,
                transparent 50%,
                rgba(100, 0, 150, 0.05) 100%
            );
            animation: hudPulse 3s ease-in-out infinite;
        }

        @keyframes hudPulse {
            0%, 100% { opacity: 0.3; }
            50% { opacity: 0.6; }
        }

        /* Scanline effect */
        .scanline {
            display: none;
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            height: 2px;
            background: linear-gradient(
                to right,
                transparent 0%,
                var(--vscode-charts-purple, #c586c0) 50%,
                transparent 100%
            );
            opacity: 0.4;
            pointer-events: none;
            z-index: 5;
        }

        .canvas-wrapper.search-mode .scanline {
            display: block;
            animation: scanlineMove 4s linear infinite;
        }

        @keyframes scanlineMove {
            0% { transform: translateY(-100%); }
            100% { transform: translateY(calc(100vh + 100%)); }
        }

        #cy {
            width: 100%;
            height: 100%;
            position: relative;
            z-index: 1;
        }

        /* ===== Empty State ===== */
        .empty-state {
            display: none;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            height: 100%;
            padding: 32px;
            text-align: center;
            color: var(--vscode-descriptionForeground, #8b8b8b);
        }

        .empty-state.visible {
            display: flex;
        }

        .empty-state-icon {
            width: 40px;
            height: 40px;
            margin-bottom: 16px;
            opacity: 0.4;
        }

        .empty-state-title {
            font-size: 13px;
            font-weight: 500;
            margin-bottom: 6px;
            color: var(--vscode-foreground, #cccccc);
        }

        .empty-state-text {
            font-size: 11px;
            line-height: 1.6;
            max-width: 180px;
            opacity: 0.8;
        }

        .empty-state-path {
            margin-top: 14px;
            padding: 5px 10px;
            background-color: var(--vscode-textCodeBlock-background, rgba(30,30,30,0.5));
            border: 1px solid var(--vscode-widget-border, rgba(128,128,128,0.2));
            border-radius: 4px;
            font-family: var(--vscode-editor-font-family, 'SF Mono', Consolas, monospace);
            font-size: 10px;
            color: var(--vscode-textPreformat-foreground, #d7ba7d);
        }

        /* ===== No Results State ===== */
        .no-results {
            display: none;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            padding: 40px 20px;
            text-align: center;
            height: 100%;
        }

        .no-results.visible {
            display: flex;
        }

        .no-results-icon {
            width: 48px;
            height: 48px;
            margin-bottom: 12px;
            opacity: 0.3;
            fill: var(--vscode-charts-purple, #c586c0);
        }

        .no-results-title {
            font-size: 13px;
            font-weight: 500;
            color: var(--vscode-foreground, #cccccc);
            margin-bottom: 4px;
        }

        .no-results-text {
            font-size: 11px;
            color: var(--vscode-descriptionForeground, #8b8b8b);
        }

        /* ===== Loading State ===== */
        .loading-overlay {
            display: none;
            position: absolute;
            inset: 0;
            background-color: var(--vscode-editor-background, #1e1e1e);
            align-items: center;
            justify-content: center;
            z-index: 10;
        }

        .loading-overlay.visible {
            display: flex;
        }

        .spinner {
            width: 20px;
            height: 20px;
            border: 2px solid var(--vscode-progressBar-background, #0e70c0);
            border-top-color: transparent;
            border-radius: 50%;
            animation: spin 0.7s linear infinite;
        }

        @keyframes spin {
            to { transform: rotate(360deg); }
        }

        /* ===== Tooltip ===== */
        .tooltip {
            position: fixed;
            display: none;
            padding: 10px 14px;
            background-color: var(--vscode-editorHoverWidget-background, #252526);
            border: 1px solid var(--vscode-editorHoverWidget-border, #454545);
            border-radius: 8px;
            font-size: 12px;
            color: var(--vscode-editorHoverWidget-foreground, #cccccc);
            pointer-events: none;
            z-index: 1000;
            max-width: 320px;
            box-shadow: 0 6px 24px rgba(0, 0, 0, 0.5);
        }

        .tooltip.visible {
            display: block;
        }

        .tooltip.search-match {
            border-color: var(--vscode-charts-purple, #c586c0);
            box-shadow: 0 0 12px rgba(197, 134, 192, 0.3), 0 6px 24px rgba(0, 0, 0, 0.5);
        }

        .tooltip-title {
            font-weight: 600;
            font-size: 12px;
            margin-bottom: 2px;
            color: var(--vscode-foreground, #e0e0e0);
        }

        .tooltip-description {
            font-size: 11px;
            opacity: 0.8;
            line-height: 1.4;
            margin-top: 4px;
        }

        .tooltip-path {
            margin-top: 8px;
            padding-top: 8px;
            border-top: 1px solid var(--vscode-editorHoverWidget-border, rgba(128,128,128,0.3));
            font-family: var(--vscode-editor-font-family, monospace);
            font-size: 10px;
            color: var(--vscode-textPreformat-foreground, #d7ba7d);
            word-break: break-all;
            opacity: 0.9;
        }

        /* ===== Snippet in Tooltip ===== */
        .tooltip-snippet {
            margin-top: 10px;
            padding: 8px 10px;
            background-color: var(--vscode-textCodeBlock-background, rgba(0,0,0,0.3));
            border-radius: 4px;
            font-family: var(--vscode-editor-font-family, monospace);
            font-size: 10px;
            line-height: 1.5;
            color: var(--vscode-editor-foreground, #d4d4d4);
            white-space: pre-wrap;
            overflow: hidden;
            max-height: 100px;
        }

        .tooltip-snippet-header {
            font-size: 9px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            color: var(--vscode-charts-purple, #c586c0);
            margin-bottom: 6px;
            font-weight: 600;
        }

        .tooltip-line-number {
            color: var(--vscode-editorLineNumber-foreground, #5a5a5a);
            margin-right: 8px;
            user-select: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <header class="header">
            <div class="header-top">
                <div class="header-title">
                    <span class="header-indicator" id="indicator"></span>
                    Blueprint
                </div>
                <div class="header-actions">
                    <!-- View Toggle -->
                    <div class="view-toggle">
                        <button class="icon-button active" id="btn-list-view" title="List View">
                            <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                                <path d="M2 3h3v3H2V3zm5 0h7v1H7V3zm0 2h4v1H7V5zM2 8h3v3H2V8zm5 0h7v1H7V8zm0 2h4v1H7v-1z"/>
                            </svg>
                        </button>
                        <button class="icon-button" id="btn-map-view" title="Map View">
                            <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                                <path d="M14 2.5l-4 1.3V12.5l4-1.3V2.5zM9 4l-3-1.2v9l3 1.2V4zM5 3L1 4.7v9L5 12V3z"/>
                            </svg>
                        </button>
                    </div>
                    <button class="icon-button" id="btn-refresh" title="Refresh">
                        <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                            <path d="M4.681 8c0-1.826 1.495-3.298 3.34-3.298.695 0 1.342.21 1.878.567l.99-.99A4.8 4.8 0 008.021 3.2C5.378 3.2 3.22 5.344 3.22 8c0 .768.183 1.495.505 2.137l1.066-1.066A3.27 3.27 0 014.68 8zm6.638 0c0 1.826-1.495 3.298-3.34 3.298a3.34 3.34 0 01-1.878-.567l-.99.99a4.8 4.8 0 002.868 1.079c2.643 0 4.801-2.144 4.801-4.8 0-.768-.183-1.495-.505-2.137l-1.066 1.066c.072.345.11.704.11 1.071z"/>
                            <path d="M8 1v3l2-1.5L8 1zm0 14v-3l-2 1.5L8 15z"/>
                        </svg>
                    </button>
                    <button class="icon-button" id="btn-fit" title="Fit to View">
                        <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                            <path d="M2 2v4h1.5V3.5H6V2H2zm8 0v1.5h2.5V6H14V2h-4zM3.5 10H2v4h4v-1.5H3.5V10zM14 10h-1.5v2.5H10V14h4v-4z"/>
                        </svg>
                    </button>
                    <button class="icon-button" id="btn-expand-all" title="Expand All">
                        <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                            <path d="M5 7l3 3 3-3H5z"/>
                        </svg>
                    </button>
                    <button class="icon-button" id="btn-collapse-all" title="Collapse All">
                        <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                            <path d="M11 9L8 6 5 9h6z"/>
                        </svg>
                    </button>
                </div>
            </div>

            <!-- Search Bar -->
            <div class="search-container">
                <input
                    type="text"
                    class="search-input"
                    id="search-input"
                    placeholder="Search palace... (e.g., auth logic)"
                    autocomplete="off"
                    spellcheck="false"
                />
                <svg class="search-icon" viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                    <path d="M11.742 10.344a6.5 6.5 0 1 0-1.397 1.398h-.001c.03.04.062.078.098.115l3.85 3.85a1 1 0 0 0 1.415-1.414l-3.85-3.85a1.007 1.007 0 0 0-.115-.1zM12 6.5a5.5 5.5 0 1 1-11 0 5.5 5.5 0 0 1 11 0z"/>
                </svg>
                <button class="search-clear" id="search-clear" title="Clear search">
                    <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                        <path d="M8 8.707l3.646 3.647.708-.707L8.707 8l3.647-3.646-.707-.708L8 7.293 4.354 3.646l-.708.708L7.293 8l-3.647 3.646.708.708L8 8.707z"/>
                    </svg>
                </button>
            </div>

            <!-- Search Status Bar -->
            <div class="search-status" id="search-status">
                <span class="search-status-badge" id="match-count">0</span>
                <span id="search-status-text">matches found</span>
            </div>
        </header>

        <div class="content-area" id="content-area">
            <!-- Tree View -->
            <div class="tree-view active" id="tree-view">
                <!-- Populated dynamically -->
            </div>

            <!-- Map View (Cytoscape) -->
            <div class="map-view" id="map-view">
                <div class="canvas-wrapper" id="canvas-wrapper">
                    <div class="scanline"></div>
                    <div id="cy"></div>
                </div>
            </div>

            <!-- Empty State -->
            <div class="empty-state" id="empty-state">
                <svg class="empty-state-icon" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M3 3h8v8H3V3zm2 2v4h4V5H5zm8-2h8v8h-8V3zm2 2v4h4V5h-4zM3 13h8v8H3v-8zm2 2v4h4v-4H5zm13 0v4h-2v-4h2zm0-2h2v6a2 2 0 01-2 2h-4v-2h4v-6z"/>
                </svg>
                <div class="empty-state-title">No Rooms Defined</div>
                <div class="empty-state-text">
                    Add room configurations to visualize your palace architecture.
                </div>
                <code class="empty-state-path">.palace/rooms/*.jsonc</code>
            </div>

            <!-- No Results State -->
            <div class="no-results" id="no-results">
                <svg class="no-results-icon" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M15.5 14h-.79l-.28-.27A6.471 6.471 0 0016 9.5 6.5 6.5 0 109.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/>
                </svg>
                <div class="no-results-title">No Matches Found</div>
                <div class="no-results-text" id="no-results-text">Try a different search term</div>
            </div>

            <!-- Loading Overlay -->
            <div class="loading-overlay" id="loading">
                <div class="spinner"></div>
            </div>
        </div>

        <div class="tooltip" id="tooltip">
            <div class="tooltip-title" id="tooltip-title"></div>
            <div class="tooltip-description" id="tooltip-description"></div>
            <div class="tooltip-path" id="tooltip-path"></div>
            <div class="tooltip-snippet" id="tooltip-snippet" style="display: none;">
                <div class="tooltip-snippet-header">Code Snippet</div>
                <div id="tooltip-snippet-content"></div>
            </div>
        </div>
    </div>

    <script src="https://cdnjs.cloudflare.com/ajax/libs/cytoscape/3.28.1/cytoscape.min.js"></script>
    <script nonce="${nonce}">
        (function() {
            const vscode = acquireVsCodeApi();

            // ══════════════════════════════════════════════════════════════════
            // STATE
            // ══════════════════════════════════════════════════════════════════

            let currentView = 'list'; // 'list' or 'map'
            let cy = null;
            let isSearchMode = false;
            let currentSearchResults = null;
            let originalGraphData = null;
            let treeData = []; // Array of room objects with files

            // ══════════════════════════════════════════════════════════════════
            // DOM ELEMENTS
            // ══════════════════════════════════════════════════════════════════

            const contentArea = document.getElementById('content-area');
            const treeView = document.getElementById('tree-view');
            const mapView = document.getElementById('map-view');
            const cyContainer = document.getElementById('cy');
            const canvasWrapper = document.getElementById('canvas-wrapper');
            const emptyState = document.getElementById('empty-state');
            const noResults = document.getElementById('no-results');
            const noResultsText = document.getElementById('no-results-text');
            const loadingOverlay = document.getElementById('loading');
            const tooltip = document.getElementById('tooltip');
            const tooltipTitle = document.getElementById('tooltip-title');
            const tooltipDescription = document.getElementById('tooltip-description');
            const tooltipPath = document.getElementById('tooltip-path');
            const tooltipSnippet = document.getElementById('tooltip-snippet');
            const tooltipSnippetContent = document.getElementById('tooltip-snippet-content');
            const btnRefresh = document.getElementById('btn-refresh');
            const btnFit = document.getElementById('btn-fit');
            const btnExpandAll = document.getElementById('btn-expand-all');
            const btnCollapseAll = document.getElementById('btn-collapse-all');
            const btnListView = document.getElementById('btn-list-view');
            const btnMapView = document.getElementById('btn-map-view');
            const searchInput = document.getElementById('search-input');
            const searchClear = document.getElementById('search-clear');
            const searchStatus = document.getElementById('search-status');
            const matchCount = document.getElementById('match-count');
            const searchStatusText = document.getElementById('search-status-text');
            const indicator = document.getElementById('indicator');

            // ══════════════════════════════════════════════════════════════════
            // VIEW SWITCHING
            // ══════════════════════════════════════════════════════════════════

            function switchView(view) {
                currentView = view;

                if (view === 'list') {
                    treeView.classList.add('active');
                    mapView.classList.remove('active');
                    btnListView.classList.add('active');
                    btnMapView.classList.remove('active');
                    btnExpandAll.style.display = '';
                    btnCollapseAll.style.display = '';
                    btnFit.style.display = 'none';
                } else {
                    treeView.classList.remove('active');
                    mapView.classList.add('active');
                    btnListView.classList.remove('active');
                    btnMapView.classList.add('active');
                    btnExpandAll.style.display = 'none';
                    btnCollapseAll.style.display = 'none';
                    btnFit.style.display = '';

                    // Initialize Cytoscape if needed
                    if (originalGraphData && !cy) {
                        initCytoscape({
                            nodes: originalGraphData.nodes,
                            edges: originalGraphData.edges || []
                        });
                    }

                    // Apply search state to map if in search mode
                    if (isSearchMode && currentSearchResults) {
                        applySearchResultsToMap(currentSearchResults);
                    }
                }
            }

            // ══════════════════════════════════════════════════════════════════
            // TREE VIEW RENDERING
            // ══════════════════════════════════════════════════════════════════

            function buildTreeData(graphData) {
                const rooms = [];
                const roomMap = new Map();

                // First pass: create rooms
                for (const node of graphData.nodes) {
                    if (node.data.type === 'room') {
                        const room = {
                            id: node.data.id,
                            name: node.data.label,
                            description: node.data.description,
                            files: [],
                            expanded: true, // Default expanded
                            isMatch: false
                        };
                        rooms.push(room);
                        roomMap.set(node.data.id, room);
                    }
                }

                // Second pass: add files to rooms
                for (const node of graphData.nodes) {
                    if (node.data.type === 'file' && node.data.parent) {
                        const room = roomMap.get(node.data.parent);
                        if (room) {
                            room.files.push({
                                id: node.data.id,
                                name: node.data.label,
                                fullPath: node.data.fullPath,
                                snippet: node.data.snippet,
                                lineNumber: node.data.lineNumber,
                                isMatch: false
                            });
                        }
                    }
                }

                // Sort rooms alphabetically
                rooms.sort((a, b) => a.name.localeCompare(b.name));

                return rooms;
            }

            function renderTreeView(searchResults = null) {
                treeView.innerHTML = '';

                if (treeData.length === 0) {
                    return;
                }

                // Build matching sets for search mode
                const matchingRooms = new Set();
                const matchingFiles = new Map();

                if (searchResults && searchResults.results) {
                    for (const roomResult of searchResults.results) {
                        matchingRooms.add(roomResult.roomName);
                        for (const match of roomResult.matches) {
                            matchingFiles.set(match.filePath, match);
                        }
                    }
                }

                for (const room of treeData) {
                    const roomEl = document.createElement('div');
                    roomEl.className = 'tree-room' + (room.expanded ? ' expanded' : '');
                    roomEl.dataset.roomId = room.id;

                    // Check if room matches search
                    const roomMatches = searchResults && (
                        matchingRooms.has(room.name) ||
                        matchingRooms.has(room.id.replace('room-', ''))
                    );
                    const hasMatchingFiles = room.files.some(f => matchingFiles.has(f.fullPath));
                    const roomIsRelevant = roomMatches || hasMatchingFiles;

                    // Room Header
                    const headerEl = document.createElement('div');
                    headerEl.className = 'tree-room-header';
                    if (searchResults) {
                        headerEl.classList.add(roomIsRelevant ? 'search-match' : 'ghost-mode');
                    }

                    // Count matching files
                    const matchingFileCount = room.files.filter(f => matchingFiles.has(f.fullPath)).length;

                    headerEl.innerHTML = \`
                        <div class="tree-chevron">
                            <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                                <path d="M6 4l4 4-4 4V4z"/>
                            </svg>
                        </div>
                        <div class="tree-room-icon">
                            <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                                <path d="M14.5 3H7.71l-.85-.85L6.51 2h-5l-.5.5v11l.5.5h13l.5-.5v-10L14.5 3zm-.51 8.49V13h-12V3h4.29l.85.85.36.15H14v7.49z"/>
                            </svg>
                        </div>
                        <span class="tree-room-label">\${escapeHtml(room.name)}</span>
                        <span class="tree-room-count">\${searchResults && roomIsRelevant ? matchingFileCount : room.files.length}</span>
                    \`;

                    headerEl.addEventListener('click', () => {
                        room.expanded = !room.expanded;
                        roomEl.classList.toggle('expanded', room.expanded);
                    });

                    roomEl.appendChild(headerEl);

                    // Files Container
                    const filesEl = document.createElement('div');
                    filesEl.className = 'tree-files';

                    if (room.files.length === 0) {
                        // Empty room placeholder
                        const emptyEl = document.createElement('div');
                        emptyEl.className = 'tree-empty';
                        emptyEl.innerHTML = \`
                            <div class="tree-empty-icon">
                                <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                                    <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2,2"/>
                                </svg>
                            </div>
                            <span class="tree-empty-label">No entry point</span>
                        \`;
                        filesEl.appendChild(emptyEl);
                    } else {
                        for (const file of room.files) {
                            const matchData = matchingFiles.get(file.fullPath);
                            const fileMatches = !!matchData;

                            const fileEl = document.createElement('div');
                            fileEl.className = 'tree-file';
                            if (searchResults) {
                                fileEl.classList.add(fileMatches ? 'search-match' : 'ghost-mode');
                            }
                            fileEl.dataset.filePath = file.fullPath;

                            let lineHtml = '';
                            if (fileMatches && matchData.lineNumber) {
                                lineHtml = \`<span class="tree-file-line">L\${matchData.lineNumber}</span>\`;
                            }

                            fileEl.innerHTML = \`
                                <div class="tree-file-icon">
                                    <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                                        <path d="M13.85 4.44l-3.28-3.3-.35-.14H3.5l-.5.5v13l.5.5h10l.5-.5V4.8l-.15-.36zM10.5 2l2.65 2.65H10.5V2zM13 14H4V2h5.5v3.5l.5.5H13v8z"/>
                                    </svg>
                                </div>
                                <span class="tree-file-label">\${escapeHtml(file.name)}</span>
                                \${lineHtml}
                            \`;

                            fileEl.addEventListener('click', () => {
                                const lineNum = matchData ? matchData.lineNumber : file.lineNumber;
                                vscode.postMessage({
                                    command: 'openFile',
                                    filePath: file.fullPath,
                                    lineNumber: lineNum
                                });
                            });

                            // Tooltip on hover for search matches
                            if (fileMatches && matchData.snippet) {
                                fileEl.addEventListener('mouseenter', (e) => {
                                    tooltipTitle.textContent = file.name;
                                    tooltipDescription.style.display = 'none';
                                    tooltipPath.style.display = 'block';
                                    tooltipPath.textContent = file.fullPath;

                                    tooltipSnippet.style.display = 'block';
                                    let snippetHtml = '';
                                    if (matchData.lineNumber) {
                                        snippetHtml = '<span class="tooltip-line-number">L' + matchData.lineNumber + '</span>';
                                    }
                                    snippetHtml += escapeHtml(matchData.snippet);
                                    tooltipSnippetContent.innerHTML = snippetHtml;
                                    tooltip.classList.add('search-match');
                                    tooltip.classList.add('visible');

                                    positionTooltip(e);
                                });

                                fileEl.addEventListener('mousemove', positionTooltip);

                                fileEl.addEventListener('mouseleave', () => {
                                    tooltip.classList.remove('visible');
                                    tooltip.classList.remove('search-match');
                                });
                            }

                            filesEl.appendChild(fileEl);
                        }
                    }

                    roomEl.appendChild(filesEl);
                    treeView.appendChild(roomEl);
                }

                // Auto-expand rooms with matches in search mode
                if (searchResults) {
                    for (const room of treeData) {
                        const roomEl = treeView.querySelector(\`[data-room-id="\${room.id}"]\`);
                        if (roomEl) {
                            const hasMatches = room.files.some(f => matchingFiles.has(f.fullPath));
                            if (hasMatches) {
                                room.expanded = true;
                                roomEl.classList.add('expanded');
                            }
                        }
                    }
                }
            }

            function positionTooltip(e) {
                const padding = 14;
                const x = e.clientX + padding;
                const y = e.clientY + padding;

                const rect = tooltip.getBoundingClientRect();
                const maxX = window.innerWidth - rect.width - 8;
                const maxY = window.innerHeight - rect.height - 8;

                tooltip.style.left = Math.min(x, maxX) + 'px';
                tooltip.style.top = Math.min(y, maxY) + 'px';
            }

            function expandAllRooms() {
                for (const room of treeData) {
                    room.expanded = true;
                }
                const roomEls = treeView.querySelectorAll('.tree-room');
                roomEls.forEach(el => el.classList.add('expanded'));
            }

            function collapseAllRooms() {
                for (const room of treeData) {
                    room.expanded = false;
                }
                const roomEls = treeView.querySelectorAll('.tree-room');
                roomEls.forEach(el => el.classList.remove('expanded'));
            }

            // ══════════════════════════════════════════════════════════════════
            // CYTOSCAPE HELPERS
            // ══════════════════════════════════════════════════════════════════

            function getCssVar(name, fallback) {
                const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
                return value || fallback;
            }

            function getCytoscapeStyle() {
                const fg = getCssVar('--vscode-editor-foreground', '#d4d4d4');
                const fgDim = getCssVar('--vscode-descriptionForeground', '#808080');
                const fgMuted = getCssVar('--vscode-editorLineNumber-foreground', '#5a5a5a');
                const accent = getCssVar('--vscode-focusBorder', '#007fd4');
                const accentSoft = getCssVar('--vscode-charts-blue', '#4fc1ff');
                const green = getCssVar('--vscode-charts-green', '#89d185');
                const purple = getCssVar('--vscode-charts-purple', '#c586c0');
                const orange = getCssVar('--vscode-charts-orange', '#d19a66');

                return [
                    // ROOM: Normal State
                    {
                        selector: 'node[type="room"]',
                        style: {
                            'shape': 'round-rectangle',
                            'corner-radius': 8,
                            'background-color': fgMuted,
                            'background-opacity': 0.04,
                            'border-width': 1,
                            'border-style': 'solid',
                            'border-color': fgMuted,
                            'border-opacity': 0.25,
                            'label': 'data(label)',
                            'text-valign': 'top',
                            'text-halign': 'center',
                            'text-margin-y': 12,
                            'font-size': '9px',
                            'font-weight': '600',
                            'text-transform': 'uppercase',
                            'color': fgDim,
                            'text-opacity': 0.7,
                            'padding': '40px',
                            'compound-sizing-wrt-labels': 'include',
                            'min-width': '120px',
                            'min-height': '70px',
                        }
                    },
                    // ROOM: Search Match
                    {
                        selector: 'node[type="room"].search-match',
                        style: {
                            'background-color': purple,
                            'background-opacity': 0.15,
                            'border-color': purple,
                            'border-opacity': 0.8,
                            'border-width': 2,
                            'color': purple,
                            'text-opacity': 1,
                            'shadow-blur': 20,
                            'shadow-color': purple,
                            'shadow-opacity': 0.4,
                            'shadow-offset-x': 0,
                            'shadow-offset-y': 0,
                        }
                    },
                    // ROOM: Ghost Mode
                    {
                        selector: 'node[type="room"].ghost-mode',
                        style: {
                            'background-opacity': 0.01,
                            'border-opacity': 0.08,
                            'text-opacity': 0.2,
                        }
                    },
                    // FILE: Normal State
                    {
                        selector: 'node[type="file"]',
                        style: {
                            'shape': 'round-rectangle',
                            'corner-radius': 4,
                            'background-color': accentSoft,
                            'background-opacity': 0.12,
                            'border-width': 1,
                            'border-style': 'solid',
                            'border-color': accentSoft,
                            'border-opacity': 0.5,
                            'label': 'data(label)',
                            'text-valign': 'center',
                            'text-halign': 'center',
                            'font-size': '11px',
                            'font-family': 'var(--vscode-editor-font-family, monospace)',
                            'font-weight': '500',
                            'color': fg,
                            'width': 'label',
                            'height': '26px',
                            'padding': '12px',
                        }
                    },
                    // FILE: Search Match
                    {
                        selector: 'node[type="file"].search-match',
                        style: {
                            'background-color': purple,
                            'background-opacity': 0.35,
                            'border-color': '#ffffff',
                            'border-opacity': 0.9,
                            'border-width': 2,
                            'color': '#ffffff',
                            'font-weight': '600',
                            'shadow-blur': 25,
                            'shadow-color': purple,
                            'shadow-opacity': 0.6,
                            'shadow-offset-x': 0,
                            'shadow-offset-y': 0,
                            'z-index': 100,
                        }
                    },
                    // FILE: Ghost Mode
                    {
                        selector: 'node[type="file"].ghost-mode',
                        style: {
                            'background-opacity': 0.03,
                            'border-opacity': 0.1,
                            'text-opacity': 0.15,
                        }
                    },
                    // FILE: Hover in Search Mode
                    {
                        selector: 'node[type="file"].search-match:active',
                        style: {
                            'background-opacity': 0.5,
                            'shadow-opacity': 0.8,
                        }
                    },
                    // GHOST: Empty Placeholder
                    {
                        selector: 'node[type="ghost"]',
                        style: {
                            'shape': 'round-rectangle',
                            'corner-radius': 3,
                            'background-color': fgMuted,
                            'background-opacity': 0.02,
                            'border-width': 1,
                            'border-style': 'dotted',
                            'border-color': fgMuted,
                            'border-opacity': 0.15,
                            'label': 'data(label)',
                            'text-valign': 'center',
                            'text-halign': 'center',
                            'font-size': '9px',
                            'font-style': 'italic',
                            'color': fgMuted,
                            'text-opacity': 0.4,
                            'width': '60px',
                            'height': '20px',
                            'padding': '4px',
                        }
                    },
                    // Ghost in ghost mode
                    {
                        selector: 'node[type="ghost"].ghost-mode',
                        style: {
                            'background-opacity': 0.005,
                            'border-opacity': 0.05,
                            'text-opacity': 0.1,
                        }
                    },
                    // SELECTION STATE
                    {
                        selector: ':selected',
                        style: {
                            'border-width': 2,
                            'border-color': green,
                            'border-opacity': 0.9,
                        }
                    },
                    // EDGES
                    {
                        selector: 'edge',
                        style: {
                            'width': 1,
                            'line-color': fgMuted,
                            'line-opacity': 0.4,
                            'line-style': 'solid',
                            'curve-style': 'bezier',
                            'target-arrow-shape': 'triangle',
                            'target-arrow-color': fgMuted,
                            'arrow-scale': 0.6,
                        }
                    },
                    {
                        selector: 'edge.ghost-mode',
                        style: {
                            'line-opacity': 0.1,
                        }
                    },
                ];
            }

            function getLayoutConfig() {
                return {
                    name: 'cose',
                    animate: false,
                    fit: true,
                    padding: 40,
                    nodeDimensionsIncludeLabels: true,
                    nodeRepulsion: function(node) {
                        return node.data('type') === 'room' ? 20000 : 8000;
                    },
                    idealEdgeLength: 120,
                    componentSpacing: 80,
                    nestingFactor: 5,
                    gravity: 0.15,
                    numIter: 500,
                    initialTemp: 300,
                    coolingFactor: 0.95,
                    minTemp: 1.0,
                };
            }

            function initCytoscape(elements) {
                if (cy) {
                    cy.destroy();
                }

                cy = cytoscape({
                    container: cyContainer,
                    elements: elements,
                    style: getCytoscapeStyle(),
                    layout: getLayoutConfig(),
                    wheelSensitivity: 0.25,
                    minZoom: 0.25,
                    maxZoom: 4,
                    boxSelectionEnabled: false,
                    selectionType: 'single',
                });

                // File Click Handler
                cy.on('tap', 'node[type="file"]', function(evt) {
                    const node = evt.target;
                    const filePath = node.data('fullPath');
                    const lineNumber = node.data('lineNumber');
                    if (filePath) {
                        vscode.postMessage({
                            command: 'openFile',
                            filePath: filePath,
                            lineNumber: lineNumber
                        });
                    }
                });

                // Tooltip: Show on hover
                cy.on('mouseover', 'node', function(evt) {
                    const node = evt.target;
                    const nodeType = node.data('type');

                    if (nodeType === 'ghost') return;

                    tooltipTitle.textContent = node.data('label');

                    const description = node.data('description');
                    tooltipDescription.style.display = description ? 'block' : 'none';
                    tooltipDescription.textContent = description || '';

                    const fullPath = node.data('fullPath');
                    tooltipPath.style.display = fullPath ? 'block' : 'none';
                    tooltipPath.textContent = fullPath || '';

                    // Show snippet for search matches
                    const snippet = node.data('snippet');
                    const lineNum = node.data('lineNumber');
                    if (snippet && isSearchMode) {
                        tooltipSnippet.style.display = 'block';
                        let snippetHtml = '';
                        if (lineNum) {
                            snippetHtml = '<span class="tooltip-line-number">L' + lineNum + '</span>';
                        }
                        snippetHtml += escapeHtml(snippet);
                        tooltipSnippetContent.innerHTML = snippetHtml;
                        tooltip.classList.add('search-match');
                    } else {
                        tooltipSnippet.style.display = 'none';
                        tooltip.classList.remove('search-match');
                    }

                    tooltip.classList.add('visible');
                });

                // Tooltip: Hide
                cy.on('mouseout', 'node', function() {
                    tooltip.classList.remove('visible');
                });

                // Tooltip: Follow cursor
                cy.on('mousemove', function(evt) {
                    if (tooltip.classList.contains('visible') && evt.originalEvent) {
                        const padding = 14;
                        const x = evt.originalEvent.clientX + padding;
                        const y = evt.originalEvent.clientY + padding;

                        const rect = tooltip.getBoundingClientRect();
                        const maxX = window.innerWidth - rect.width - 8;
                        const maxY = window.innerHeight - rect.height - 8;

                        tooltip.style.left = Math.min(x, maxX) + 'px';
                        tooltip.style.top = Math.min(y, maxY) + 'px';
                    }
                });

                // Cursor: Pointer on clickable nodes
                cy.on('mouseover', 'node[type="file"]', function() {
                    cyContainer.style.cursor = 'pointer';
                });
                cy.on('mouseout', 'node[type="file"]', function() {
                    cyContainer.style.cursor = 'default';
                });
            }

            // ══════════════════════════════════════════════════════════════════
            // GRAPH UPDATE
            // ══════════════════════════════════════════════════════════════════

            function updateGraph(data) {
                loadingOverlay.classList.remove('visible');
                noResults.classList.remove('visible');

                if (!data || !data.nodes || data.nodes.length === 0) {
                    emptyState.classList.add('visible');
                    treeView.style.display = 'none';
                    mapView.style.display = 'none';
                    return;
                }

                emptyState.classList.remove('visible');

                // Store original data
                originalGraphData = data;

                // Build tree data
                treeData = buildTreeData(data);

                // Render appropriate view
                if (currentView === 'list') {
                    treeView.style.display = '';
                    mapView.style.display = 'none';
                    renderTreeView();
                } else {
                    treeView.style.display = 'none';
                    mapView.style.display = '';
                    initCytoscape({
                        nodes: data.nodes,
                        edges: data.edges || []
                    });
                }
            }

            // ══════════════════════════════════════════════════════════════════
            // SEARCH MODE
            // ══════════════════════════════════════════════════════════════════

            function enterSearchMode() {
                isSearchMode = true;
                canvasWrapper.classList.add('search-mode');
                indicator.classList.add('search-active');
            }

            function exitSearchMode() {
                isSearchMode = false;
                currentSearchResults = null;
                canvasWrapper.classList.remove('search-mode');
                indicator.classList.remove('search-active');
                indicator.classList.remove('searching');
                searchStatus.classList.remove('visible');
                noResults.classList.remove('visible');

                // Re-render tree without search highlights
                if (currentView === 'list') {
                    renderTreeView();
                }

                // Remove all search classes from map nodes
                if (cy) {
                    cy.nodes().removeClass('search-match ghost-mode');
                    cy.edges().removeClass('ghost-mode');
                }
            }

            function applySearchResults(results) {
                currentSearchResults = results;
                enterSearchMode();

                // Update match count
                matchCount.textContent = results.totalMatches;
                searchStatusText.textContent = results.totalMatches === 1 ? 'match found' : 'matches found';
                searchStatus.classList.add('visible');

                if (results.totalMatches === 0) {
                    noResults.classList.add('visible');
                    noResultsText.textContent = 'No matches for "' + results.query + '"';
                    if (currentView === 'list') {
                        treeView.style.display = 'none';
                    } else {
                        cyContainer.style.display = 'none';
                    }
                    return;
                }

                noResults.classList.remove('visible');

                if (currentView === 'list') {
                    treeView.style.display = '';
                    renderTreeView(results);
                } else {
                    cyContainer.style.display = '';
                    applySearchResultsToMap(results);
                }
            }

            function applySearchResultsToMap(results) {
                if (!cy || !results) return;

                // Get matching file paths and room names
                const matchingRooms = new Set();
                const matchingFiles = new Map();

                for (const roomResult of results.results) {
                    matchingRooms.add(roomResult.roomName);
                    for (const match of roomResult.matches) {
                        matchingFiles.set(match.filePath, match);
                    }
                }

                // Apply classes to nodes
                cy.nodes().forEach(node => {
                    const nodeType = node.data('type');
                    const nodeId = node.data('id');
                    const fullPath = node.data('fullPath');
                    const label = node.data('label');

                    if (nodeType === 'room') {
                        const roomName = label;
                        const roomId = nodeId.replace('room-', '');

                        if (matchingRooms.has(roomName) || matchingRooms.has(roomId)) {
                            node.removeClass('ghost-mode');
                            node.addClass('search-match');
                        } else {
                            node.removeClass('search-match');
                            node.addClass('ghost-mode');
                        }
                    } else if (nodeType === 'file') {
                        const match = matchingFiles.get(fullPath);
                        if (match) {
                            node.removeClass('ghost-mode');
                            node.addClass('search-match');
                            node.data('snippet', match.snippet);
                            node.data('lineNumber', match.lineNumber);
                        } else {
                            node.removeClass('search-match');
                            node.addClass('ghost-mode');
                            node.data('snippet', null);
                            node.data('lineNumber', null);
                        }
                    } else {
                        node.addClass('ghost-mode');
                    }
                });

                cy.edges().addClass('ghost-mode');

                // Fit camera to matching nodes
                const matchingNodes = cy.nodes('.search-match');
                if (matchingNodes.length > 0) {
                    cy.animate({
                        fit: {
                            eles: matchingNodes,
                            padding: 50
                        },
                        duration: 300,
                        easing: 'ease-out-cubic'
                    });
                }
            }

            function escapeHtml(text) {
                const div = document.createElement('div');
                div.textContent = text;
                return div.innerHTML;
            }

            // ══════════════════════════════════════════════════════════════════
            // EVENT HANDLERS
            // ══════════════════════════════════════════════════════════════════

            // View toggle buttons
            btnListView.addEventListener('click', () => switchView('list'));
            btnMapView.addEventListener('click', () => switchView('map'));

            // Expand/Collapse buttons
            btnExpandAll.addEventListener('click', expandAllRooms);
            btnCollapseAll.addEventListener('click', collapseAllRooms);

            // Search input handler
            searchInput.addEventListener('input', function(e) {
                const query = e.target.value;

                if (query.length > 0) {
                    searchClear.classList.add('visible');
                } else {
                    searchClear.classList.remove('visible');
                }

                vscode.postMessage({ command: 'search', query: query });
            });

            // Search input keyboard handlers
            searchInput.addEventListener('keydown', function(e) {
                if (e.key === 'Escape') {
                    searchInput.value = '';
                    searchClear.classList.remove('visible');
                    vscode.postMessage({ command: 'clearSearch' });
                }
            });

            // Clear button
            searchClear.addEventListener('click', function() {
                searchInput.value = '';
                searchClear.classList.remove('visible');
                searchInput.focus();
                vscode.postMessage({ command: 'clearSearch' });
            });

            // Refresh button
            btnRefresh.addEventListener('click', function() {
                loadingOverlay.classList.add('visible');
                vscode.postMessage({ command: 'refresh' });
            });

            // Fit button (map view only)
            btnFit.addEventListener('click', function() {
                if (cy) {
                    const targetNodes = isSearchMode ? cy.nodes('.search-match') : cy.nodes();
                    if (targetNodes.length > 0) {
                        cy.animate({
                            fit: {
                                eles: targetNodes.length > 0 ? targetNodes : cy.elements(),
                                padding: 40
                            },
                            duration: 200,
                            easing: 'ease-out-cubic'
                        });
                    }
                }
            });

            // ══════════════════════════════════════════════════════════════════
            // MESSAGE HANDLER
            // ══════════════════════════════════════════════════════════════════

            window.addEventListener('message', function(event) {
                const message = event.data;

                switch (message.command) {
                    case 'updateGraph':
                        updateGraph(message.data);
                        break;

                    case 'searchState':
                        if (message.state === 'searching') {
                            indicator.classList.add('searching');
                            indicator.classList.remove('search-active');
                        } else if (message.state === 'error') {
                            indicator.classList.remove('searching');
                            indicator.classList.add('disconnected');
                            searchStatus.classList.add('visible');
                            matchCount.textContent = '!';
                            searchStatusText.textContent = message.error || 'Search failed';
                        }
                        break;

                    case 'searchResults':
                        indicator.classList.remove('searching');
                        applySearchResults(message.results);
                        break;

                    case 'clearSearchResults':
                        exitSearchMode();
                        // Restore original graph
                        if (originalGraphData) {
                            updateGraph(originalGraphData);
                        }
                        break;

                    case 'connectionStatus':
                        if (message.connected) {
                            indicator.classList.remove('disconnected');
                        } else {
                            indicator.classList.add('disconnected');
                        }
                        break;
                }
            });

            // ══════════════════════════════════════════════════════════════════
            // INITIAL LOAD
            // ══════════════════════════════════════════════════════════════════

            // Set initial button visibility
            btnFit.style.display = 'none'; // Hidden in list view by default

            loadingOverlay.classList.add('visible');
            vscode.postMessage({ command: 'ready' });
        })();
    </script>
</body>
</html>`;
    }

    private _getNonce(): string {
        let text = '';
        const possible = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
        for (let i = 0; i < 32; i++) {
            text += possible.charAt(Math.floor(Math.random() * possible.length));
        }
        return text;
    }
}
