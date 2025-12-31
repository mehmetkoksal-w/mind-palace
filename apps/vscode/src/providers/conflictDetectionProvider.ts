import * as vscode from 'vscode';
import { PalaceBridge } from '../bridge';

/**
 * ConflictDetectionProvider monitors open files for conflicts with other agents.
 * It shows warnings when another agent is actively working on the same file.
 */
export class ConflictDetectionProvider implements vscode.Disposable {
    private bridge?: PalaceBridge;
    private statusBarItem: vscode.StatusBarItem;
    private decorationType: vscode.TextEditorDecorationType;
    private disposables: vscode.Disposable[] = [];
    private conflictCache = new Map<string, { conflict: boolean; agent?: string; timestamp: number }>();
    private cacheTTL = 30000; // 30 seconds
    private checkInterval?: NodeJS.Timeout;

    constructor() {
        // Status bar item for showing conflict status
        this.statusBarItem = vscode.window.createStatusBarItem(
            vscode.StatusBarAlignment.Right,
            98
        );
        this.statusBarItem.command = 'mindPalace.showConflictInfo';

        // Decoration for conflict warning
        this.decorationType = vscode.window.createTextEditorDecorationType({
            backgroundColor: new vscode.ThemeColor('editorWarning.background'),
            isWholeLine: true,
            overviewRulerColor: new vscode.ThemeColor('editorWarning.foreground'),
            overviewRulerLane: vscode.OverviewRulerLane.Right,
        });

        // Listen for editor changes
        this.disposables.push(
            vscode.window.onDidChangeActiveTextEditor(editor => {
                if (editor) {
                    this.checkConflict(editor);
                }
            })
        );

        // Listen for file saves to check conflicts
        this.disposables.push(
            vscode.workspace.onDidSaveTextDocument(doc => {
                const editor = vscode.window.activeTextEditor;
                if (editor && editor.document === doc) {
                    this.invalidateCache(doc.uri.fsPath);
                    this.checkConflict(editor);
                }
            })
        );

        // Periodic conflict check
        this.checkInterval = setInterval(() => {
            const editor = vscode.window.activeTextEditor;
            if (editor) {
                this.checkConflict(editor);
            }
        }, 60000); // Check every minute
    }

    /**
     * Set the bridge for MCP communication
     */
    setBridge(bridge: PalaceBridge): void {
        this.bridge = bridge;
    }

    /**
     * Check for conflicts on the current file
     */
    async checkConflict(editor: vscode.TextEditor): Promise<void> {
        if (!this.bridge) {
            this.hideConflictWarning();
            return;
        }

        const filePath = editor.document.uri.fsPath;
        const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;

        if (!workspaceRoot || !filePath.startsWith(workspaceRoot)) {
            this.hideConflictWarning();
            return;
        }

        const relativePath = filePath.substring(workspaceRoot.length + 1);

        // Check cache first
        const cached = this.conflictCache.get(relativePath);
        if (cached && (Date.now() - cached.timestamp) < this.cacheTTL) {
            if (cached.conflict) {
                this.showConflictWarning(editor, cached.agent);
            } else {
                this.hideConflictWarning();
            }
            return;
        }

        try {
            const result = await this.bridge.checkConflict(relativePath);

            // Cache the result
            this.conflictCache.set(relativePath, {
                conflict: result.conflict,
                agent: result.agent,
                timestamp: Date.now(),
            });

            if (result.conflict) {
                this.showConflictWarning(editor, result.agent);
            } else {
                this.hideConflictWarning();
            }
        } catch {
            this.hideConflictWarning();
        }
    }

    /**
     * Show conflict warning in status bar and decorations
     */
    private showConflictWarning(editor: vscode.TextEditor, agent?: string): void {
        const agentName = agent || 'another agent';

        // Status bar
        this.statusBarItem.text = `$(warning) Conflict: ${agentName}`;
        this.statusBarItem.tooltip = new vscode.MarkdownString(
            `**Mind Palace - File Conflict**\n\n` +
            `${agentName} is also working on this file.\n\n` +
            `Consider coordinating changes to avoid conflicts.`
        );
        this.statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.warningBackground');
        this.statusBarItem.show();

        // First line decoration
        const firstLine = editor.document.lineAt(0);
        const decoration: vscode.DecorationOptions = {
            range: firstLine.range,
            hoverMessage: new vscode.MarkdownString(
                `**$(warning) File Conflict**\n\n` +
                `${agentName} is also working on this file.`
            ),
        };
        editor.setDecorations(this.decorationType, [decoration]);
    }

    /**
     * Hide conflict warning
     */
    private hideConflictWarning(): void {
        this.statusBarItem.hide();

        const editor = vscode.window.activeTextEditor;
        if (editor) {
            editor.setDecorations(this.decorationType, []);
        }
    }

    /**
     * Invalidate cache for a specific path
     */
    invalidateCache(path: string): void {
        const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
        if (workspaceRoot && path.startsWith(workspaceRoot)) {
            const relativePath = path.substring(workspaceRoot.length + 1);
            this.conflictCache.delete(relativePath);
        }
    }

    /**
     * Clear all cache
     */
    clearCache(): void {
        this.conflictCache.clear();
    }

    /**
     * Dispose resources
     */
    dispose(): void {
        if (this.checkInterval) {
            clearInterval(this.checkInterval);
        }
        this.statusBarItem.dispose();
        this.decorationType.dispose();
        this.disposables.forEach(d => d.dispose());
    }
}
