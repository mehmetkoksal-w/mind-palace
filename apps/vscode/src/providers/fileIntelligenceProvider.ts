import * as vscode from 'vscode';
import { PalaceBridge, FileIntelResult } from '../bridge';

/**
 * File intelligence decoration types
 */
interface FileIntelDecorations {
    hot: vscode.TextEditorDecorationType;
    fragile: vscode.TextEditorDecorationType;
    hasLearnings: vscode.TextEditorDecorationType;
    hasDecisions: vscode.TextEditorDecorationType;
}

/**
 * Cached file intelligence data
 */
interface CachedIntel {
    data: FileIntelResult;
    timestamp: number;
}

/**
 * FileIntelligenceProvider adds gutter decorations and status bar info
 * for files based on Mind Palace intelligence data.
 */
export class FileIntelligenceProvider implements vscode.Disposable {
    private bridge?: PalaceBridge;
    private decorations: FileIntelDecorations;
    private cache = new Map<string, CachedIntel>();
    private cacheTTL = 60000; // 1 minute TTL
    private disposables: vscode.Disposable[] = [];
    private statusBarItem: vscode.StatusBarItem;

    constructor() {
        // Create decoration types with gutter icons
        this.decorations = {
            hot: vscode.window.createTextEditorDecorationType({
                gutterIconPath: this.getIconPath('hot'),
                gutterIconSize: 'contain',
                overviewRulerColor: new vscode.ThemeColor('editorWarning.foreground'),
                overviewRulerLane: vscode.OverviewRulerLane.Right,
            }),
            fragile: vscode.window.createTextEditorDecorationType({
                gutterIconPath: this.getIconPath('fragile'),
                gutterIconSize: 'contain',
                overviewRulerColor: new vscode.ThemeColor('editorError.foreground'),
                overviewRulerLane: vscode.OverviewRulerLane.Right,
            }),
            hasLearnings: vscode.window.createTextEditorDecorationType({
                gutterIconPath: this.getIconPath('learning'),
                gutterIconSize: 'contain',
                overviewRulerColor: new vscode.ThemeColor('editorInfo.foreground'),
                overviewRulerLane: vscode.OverviewRulerLane.Left,
            }),
            hasDecisions: vscode.window.createTextEditorDecorationType({
                gutterIconPath: this.getIconPath('decision'),
                gutterIconSize: 'contain',
                overviewRulerColor: new vscode.ThemeColor('editorHint.foreground'),
                overviewRulerLane: vscode.OverviewRulerLane.Left,
            }),
        };

        // Create status bar item
        this.statusBarItem = vscode.window.createStatusBarItem(
            vscode.StatusBarAlignment.Right,
            99
        );
        this.statusBarItem.command = 'mindPalace.showFileIntel';

        // Subscribe to editor changes
        this.disposables.push(
            vscode.window.onDidChangeActiveTextEditor(editor => {
                if (editor) {
                    this.updateDecorations(editor);
                }
            })
        );

        // Subscribe to document saves to refresh cache
        this.disposables.push(
            vscode.workspace.onDidSaveTextDocument(doc => {
                // Invalidate cache for saved file
                this.cache.delete(doc.uri.fsPath);
                const editor = vscode.window.activeTextEditor;
                if (editor && editor.document === doc) {
                    this.updateDecorations(editor);
                }
            })
        );
    }

    /**
     * Set the bridge for MCP communication
     */
    setBridge(bridge: PalaceBridge): void {
        this.bridge = bridge;
    }

    /**
     * Get icon path for decoration type
     * Uses ThemeIcon for simplicity (VS Code built-in icons)
     */
    private getIconPath(_type: string): vscode.Uri | undefined {
        // For now, we'll use text-based decorations instead of custom icons
        // Custom SVG icons could be added in resources/ folder
        return undefined;
    }

    /**
     * Update decorations for the given editor
     */
    async updateDecorations(editor: vscode.TextEditor): Promise<void> {
        if (!this.bridge) {
            this.clearDecorations(editor);
            this.statusBarItem.hide();
            return;
        }

        const filePath = editor.document.uri.fsPath;
        const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;

        if (!workspaceRoot || !filePath.startsWith(workspaceRoot)) {
            this.clearDecorations(editor);
            this.statusBarItem.hide();
            return;
        }

        // Get relative path
        const relativePath = filePath.substring(workspaceRoot.length + 1);

        try {
            const intel = await this.getFileIntel(relativePath);

            if (!intel) {
                this.clearDecorations(editor);
                this.statusBarItem.hide();
                return;
            }

            // Apply decorations based on intel
            this.applyDecorations(editor, intel);
            this.updateStatusBar(intel, relativePath);

        } catch (error) {
            console.error('Failed to get file intel:', error);
            this.clearDecorations(editor);
            this.statusBarItem.hide();
        }
    }

    /**
     * Get file intelligence with caching
     */
    private async getFileIntel(path: string): Promise<FileIntelResult | null> {
        // Check cache
        const cached = this.cache.get(path);
        if (cached && (Date.now() - cached.timestamp) < this.cacheTTL) {
            return cached.data;
        }

        // Fetch from MCP
        if (!this.bridge) return null;

        try {
            const intel = await this.bridge.getFileIntel(path);

            // Cache result
            this.cache.set(path, {
                data: intel,
                timestamp: Date.now(),
            });

            return intel;
        } catch {
            return null;
        }
    }

    /**
     * Apply decorations to editor based on intel
     */
    private applyDecorations(editor: vscode.TextEditor, intel: FileIntelResult): void {
        const decorationsMap: Map<vscode.TextEditorDecorationType, vscode.DecorationOptions[]> = new Map();

        // Initialize empty arrays
        decorationsMap.set(this.decorations.hot, []);
        decorationsMap.set(this.decorations.fragile, []);
        decorationsMap.set(this.decorations.hasLearnings, []);
        decorationsMap.set(this.decorations.hasDecisions, []);

        // First line decoration for file-level indicators
        const firstLineRange = new vscode.Range(0, 0, 0, 0);

        // Hot file indicator (edit count > 10)
        if (intel.editCount > 10) {
            decorationsMap.get(this.decorations.hot)!.push({
                range: firstLineRange,
                hoverMessage: new vscode.MarkdownString(`**Hot File** - ${intel.editCount} edits`),
            });
        }

        // Fragile file indicator (failure count > 2)
        if (intel.failureCount > 2) {
            decorationsMap.get(this.decorations.fragile)!.push({
                range: firstLineRange,
                hoverMessage: new vscode.MarkdownString(`**Fragile File** - ${intel.failureCount} failures`),
            });
        }

        // Has learnings indicator
        if (intel.learnings && intel.learnings.length > 0) {
            const learningsList = intel.learnings
                .slice(0, 3)
                .map(l => `- ${l.content.substring(0, 50)}...`)
                .join('\n');

            decorationsMap.get(this.decorations.hasLearnings)!.push({
                range: firstLineRange,
                hoverMessage: new vscode.MarkdownString(
                    `**${intel.learnings.length} Learning(s)**\n\n${learningsList}`
                ),
            });
        }

        // Apply all decorations
        for (const [type, options] of decorationsMap) {
            editor.setDecorations(type, options);
        }
    }

    /**
     * Clear all decorations from editor
     */
    private clearDecorations(editor: vscode.TextEditor): void {
        editor.setDecorations(this.decorations.hot, []);
        editor.setDecorations(this.decorations.fragile, []);
        editor.setDecorations(this.decorations.hasLearnings, []);
        editor.setDecorations(this.decorations.hasDecisions, []);
    }

    /**
     * Update status bar with file intel summary
     */
    private updateStatusBar(intel: FileIntelResult, path: string): void {
        const parts: string[] = [];

        if (intel.editCount > 10) {
            parts.push(`$(flame) ${intel.editCount}`);
        }

        if (intel.failureCount > 2) {
            parts.push(`$(warning) ${intel.failureCount}`);
        }

        if (intel.learnings && intel.learnings.length > 0) {
            parts.push(`$(lightbulb) ${intel.learnings.length}`);
        }

        if (parts.length > 0) {
            this.statusBarItem.text = `$(brain) ${parts.join(' ')}`;
            this.statusBarItem.tooltip = new vscode.MarkdownString(
                `**Mind Palace - ${path}**\n\n` +
                `- Edits: ${intel.editCount}\n` +
                `- Failures: ${intel.failureCount}\n` +
                `- Learnings: ${intel.learnings?.length ?? 0}\n` +
                (intel.lastEdited ? `- Last edited: ${intel.lastEdited}` : '')
            );
            this.statusBarItem.show();
        } else {
            this.statusBarItem.hide();
        }
    }

    /**
     * Invalidate cache for a specific file
     */
    invalidateCache(path: string): void {
        this.cache.delete(path);
    }

    /**
     * Clear entire cache
     */
    clearCache(): void {
        this.cache.clear();
    }

    /**
     * Dispose resources
     */
    dispose(): void {
        this.decorations.hot.dispose();
        this.decorations.fragile.dispose();
        this.decorations.hasLearnings.dispose();
        this.decorations.hasDecisions.dispose();
        this.statusBarItem.dispose();
        this.disposables.forEach(d => d.dispose());
    }
}
