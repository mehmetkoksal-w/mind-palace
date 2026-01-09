import * as vscode from 'vscode';
import { PalaceBridge } from '../bridge';

/**
 * Learning with line information
 */
interface InlineLearning {
    id: string;
    content: string;
    confidence: number;
    line?: number;
    scope: string;
}

/**
 * InlineLearningDecorator shows learnings as subtle inline decorations
 * with opacity based on confidence level.
 */
export class InlineLearningDecorator {
    private bridge?: PalaceBridge;
    private disposables: vscode.Disposable[] = [];

    // Decoration types for different confidence levels
    private highConfidenceType: vscode.TextEditorDecorationType;
    private mediumConfidenceType: vscode.TextEditorDecorationType;
    private lowConfidenceType: vscode.TextEditorDecorationType;

    // Cache for file learnings
    private cache = new Map<string, { learnings: InlineLearning[]; timestamp: number }>();
    private cacheTTL = 60000; // 1 minute

    constructor() {
        // High confidence (>= 0.7) - more visible
        this.highConfidenceType = vscode.window.createTextEditorDecorationType({
            after: {
                margin: '0 0 0 1.5em',
                color: new vscode.ThemeColor('editorInfo.foreground'),
                fontStyle: 'italic',
            },
            overviewRulerColor: new vscode.ThemeColor('editorInfo.foreground'),
            overviewRulerLane: vscode.OverviewRulerLane.Right,
        });

        // Medium confidence (0.4 - 0.7) - subtle
        this.mediumConfidenceType = vscode.window.createTextEditorDecorationType({
            after: {
                margin: '0 0 0 1.5em',
                color: new vscode.ThemeColor('editorHint.foreground'),
                fontStyle: 'italic',
            },
            overviewRulerColor: new vscode.ThemeColor('editorHint.foreground'),
            overviewRulerLane: vscode.OverviewRulerLane.Right,
        });

        // Low confidence (< 0.4) - very subtle
        this.lowConfidenceType = vscode.window.createTextEditorDecorationType({
            after: {
                margin: '0 0 0 1.5em',
                color: new vscode.ThemeColor('descriptionForeground'),
                fontStyle: 'italic',
            },
        });
    }

    /**
     * Set the bridge for MCP communication
     */
    setBridge(bridge: PalaceBridge): void {
        this.bridge = bridge;
    }

    /**
     * Activate the decorator
     */
    activate(context: vscode.ExtensionContext): void {
        // Update decorations when active editor changes
        this.disposables.push(
            vscode.window.onDidChangeActiveTextEditor(editor => {
                if (editor) {
                    this.updateDecorations(editor);
                }
            })
        );

        // Update decorations when document changes (debounced)
        let debounceTimer: NodeJS.Timeout | undefined;
        this.disposables.push(
            vscode.workspace.onDidChangeTextDocument(event => {
                const editor = vscode.window.activeTextEditor;
                if (editor && event.document === editor.document) {
                    if (debounceTimer) {
                        clearTimeout(debounceTimer);
                    }
                    debounceTimer = setTimeout(() => {
                        this.updateDecorations(editor);
                    }, 500);
                }
            })
        );

        // Update for current editor
        if (vscode.window.activeTextEditor) {
            this.updateDecorations(vscode.window.activeTextEditor);
        }

        context.subscriptions.push(...this.disposables);
    }

    /**
     * Update decorations for an editor
     */
    async updateDecorations(editor: vscode.TextEditor): Promise<void> {
        if (!this.bridge || !this.bridge.isMCPConnected) {
            this.clearDecorations(editor);
            return;
        }

        const config = vscode.workspace.getConfiguration('mindPalace');
        if (!config.get<boolean>('showInlineLearnings', true)) {
            this.clearDecorations(editor);
            return;
        }

        const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
        if (!workspaceRoot) {
            return;
        }

        const filePath = editor.document.uri.fsPath;
        if (!filePath.startsWith(workspaceRoot)) {
            return;
        }

        const relativePath = filePath.substring(workspaceRoot.length + 1);

        try {
            const learnings = await this.getFileLearnings(relativePath);

            if (!learnings || learnings.length === 0) {
                this.clearDecorations(editor);
                return;
            }

            this.applyDecorations(editor, learnings);
        } catch {
            this.clearDecorations(editor);
        }
    }

    /**
     * Get learnings for a file with caching
     */
    private async getFileLearnings(path: string): Promise<InlineLearning[]> {
        const cached = this.cache.get(path);
        if (cached && (Date.now() - cached.timestamp) < this.cacheTTL) {
            return cached.learnings;
        }

        if (!this.bridge) return [];

        try {
            const intel = await this.bridge.getFileIntel(path);

            if (!intel || !intel.learnings) {
                return [];
            }

            const learnings: InlineLearning[] = intel.learnings.map(l => ({
                id: l.id,
                content: l.content,
                confidence: l.confidence,
                scope: 'file',
            }));

            this.cache.set(path, {
                learnings,
                timestamp: Date.now(),
            });

            return learnings;
        } catch {
            return [];
        }
    }

    /**
     * Apply decorations to the editor
     */
    private applyDecorations(editor: vscode.TextEditor, learnings: InlineLearning[]): void {
        const highDecorations: vscode.DecorationOptions[] = [];
        const mediumDecorations: vscode.DecorationOptions[] = [];
        const lowDecorations: vscode.DecorationOptions[] = [];

        // Find relevant lines for each learning
        for (const learning of learnings) {
            const line = this.findRelevantLine(editor.document, learning);
            if (line < 0) continue;

            const range = new vscode.Range(line, 0, line, 0);
            const shortContent = this.truncate(learning.content, 50);
            const confidencePercent = Math.round(learning.confidence * 100);

            const decoration: vscode.DecorationOptions = {
                range,
                hoverMessage: new vscode.MarkdownString(
                    `**Learning** (${confidencePercent}% confidence)\n\n${learning.content}\n\n[View details](command:mindPalace.showLearningDetail?${encodeURIComponent(JSON.stringify(learning))})`
                ),
                renderOptions: {
                    after: {
                        contentText: `💡 ${shortContent}`,
                    },
                },
            };

            if (learning.confidence >= 0.7) {
                highDecorations.push(decoration);
            } else if (learning.confidence >= 0.4) {
                mediumDecorations.push(decoration);
            } else {
                lowDecorations.push(decoration);
            }
        }

        editor.setDecorations(this.highConfidenceType, highDecorations);
        editor.setDecorations(this.mediumConfidenceType, mediumDecorations);
        editor.setDecorations(this.lowConfidenceType, lowDecorations);
    }

    /**
     * Find a relevant line for a learning based on content matching
     */
    private findRelevantLine(document: vscode.TextDocument, learning: InlineLearning): number {
        // If the learning has a specific line, use it
        if (learning.line !== undefined && learning.line >= 0) {
            return Math.min(learning.line, document.lineCount - 1);
        }

        // Try to find a matching line based on content keywords
        const keywords = this.extractKeywords(learning.content);
        if (keywords.length === 0) {
            // Default to showing at the top
            return 0;
        }

        let bestLine = 0;
        let bestScore = 0;

        for (let i = 0; i < document.lineCount; i++) {
            const line = document.lineAt(i).text.toLowerCase();
            let score = 0;

            for (const keyword of keywords) {
                if (line.includes(keyword.toLowerCase())) {
                    score++;
                }
            }

            if (score > bestScore) {
                bestScore = score;
                bestLine = i;
            }
        }

        return bestLine;
    }

    /**
     * Extract keywords from learning content
     */
    private extractKeywords(content: string): string[] {
        // Remove common words and extract significant terms
        const stopWords = new Set([
            'the', 'a', 'an', 'is', 'are', 'was', 'were', 'be', 'been', 'being',
            'have', 'has', 'had', 'do', 'does', 'did', 'will', 'would', 'could',
            'should', 'may', 'might', 'must', 'can', 'this', 'that', 'these',
            'those', 'it', 'its', 'of', 'in', 'to', 'for', 'with', 'on', 'at',
            'by', 'from', 'or', 'and', 'but', 'if', 'then', 'else', 'when',
            'where', 'why', 'how', 'all', 'each', 'every', 'both', 'few',
            'more', 'most', 'other', 'some', 'such', 'no', 'nor', 'not',
            'only', 'own', 'same', 'so', 'than', 'too', 'very', 'just',
        ]);

        const words = content
            .toLowerCase()
            .replace(/[^\w\s]/g, ' ')
            .split(/\s+/)
            .filter(w => w.length > 2 && !stopWords.has(w));

        // Return unique keywords
        return [...new Set(words)].slice(0, 5);
    }

    /**
     * Truncate text with ellipsis
     */
    private truncate(text: string, maxLength: number): string {
        if (text.length <= maxLength) return text;
        return text.slice(0, maxLength - 3) + '...';
    }

    /**
     * Clear all decorations from an editor
     */
    private clearDecorations(editor: vscode.TextEditor): void {
        editor.setDecorations(this.highConfidenceType, []);
        editor.setDecorations(this.mediumConfidenceType, []);
        editor.setDecorations(this.lowConfidenceType, []);
    }

    /**
     * Invalidate cache
     */
    invalidateCache(path?: string): void {
        if (path) {
            this.cache.delete(path);
        } else {
            this.cache.clear();
        }

        // Refresh current editor
        if (vscode.window.activeTextEditor) {
            this.updateDecorations(vscode.window.activeTextEditor);
        }
    }

    /**
     * Dispose resources
     */
    dispose(): void {
        this.highConfidenceType.dispose();
        this.mediumConfidenceType.dispose();
        this.lowConfidenceType.dispose();
        this.disposables.forEach(d => d.dispose());
    }
}
