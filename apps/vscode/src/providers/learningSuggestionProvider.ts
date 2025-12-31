import * as vscode from 'vscode';
import { PalaceBridge, SemanticSearchResult } from '../bridge';

/**
 * Cached learning suggestions for a file
 */
interface CachedSuggestions {
    suggestions: LearningSuggestion[];
    timestamp: number;
}

/**
 * Learning suggestion with metadata
 */
export interface LearningSuggestion {
    id: string;
    content: string;
    similarity: number;
    kind: string;
    createdAt: string;
}

/**
 * LearningSuggestionProvider provides contextual learning suggestions
 * based on the current file content using semantic search.
 */
export class LearningSuggestionProvider implements vscode.CodeLensProvider {
    private bridge?: PalaceBridge;
    private cache = new Map<string, CachedSuggestions>();
    private cacheTTL = 120000; // 2 minute TTL
    private debounceTimer: NodeJS.Timeout | undefined;
    private debounceDelay = 1000; // 1 second debounce

    private _onDidChangeCodeLenses = new vscode.EventEmitter<void>();
    public readonly onDidChangeCodeLenses = this._onDidChangeCodeLenses.event;

    constructor() {}

    /**
     * Set the bridge for MCP communication
     */
    setBridge(bridge: PalaceBridge): void {
        this.bridge = bridge;
    }

    /**
     * Refresh suggestions for all documents
     */
    refresh(): void {
        this._onDidChangeCodeLenses.fire();
    }

    /**
     * Invalidate cache and refresh
     */
    invalidateCache(path?: string): void {
        if (path) {
            this.cache.delete(path);
        } else {
            this.cache.clear();
        }
        this.refresh();
    }

    /**
     * Schedule a debounced refresh
     */
    scheduleRefresh(): void {
        if (this.debounceTimer) {
            clearTimeout(this.debounceTimer);
        }
        this.debounceTimer = setTimeout(() => {
            this.refresh();
        }, this.debounceDelay);
    }

    /**
     * Provide CodeLens for learning suggestions
     */
    async provideCodeLenses(
        document: vscode.TextDocument,
        token: vscode.CancellationToken
    ): Promise<vscode.CodeLens[]> {
        if (!this.bridge || !this.bridge.isMCPConnected) {
            return [];
        }

        // Only show for code files
        if (!this.isCodeFile(document)) {
            return [];
        }

        const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
        if (!workspaceRoot) {
            return [];
        }

        const filePath = document.uri.fsPath;
        if (!filePath.startsWith(workspaceRoot)) {
            return [];
        }

        try {
            const suggestions = await this.getSuggestions(document);

            if (!suggestions || suggestions.length === 0) {
                return [];
            }

            return this.createCodeLenses(document, suggestions);
        } catch {
            return [];
        }
    }

    /**
     * Resolve CodeLens command
     */
    resolveCodeLens(
        codeLens: vscode.CodeLens,
        token: vscode.CancellationToken
    ): vscode.CodeLens {
        return codeLens;
    }

    /**
     * Check if the document is a code file we should show suggestions for
     */
    private isCodeFile(document: vscode.TextDocument): boolean {
        const supportedLanguages = [
            'typescript', 'javascript', 'typescriptreact', 'javascriptreact',
            'python', 'go', 'rust', 'java', 'csharp', 'cpp', 'c',
            'ruby', 'php', 'swift', 'kotlin', 'scala', 'dart',
            'lua', 'sql', 'markdown', 'json', 'yaml'
        ];
        return supportedLanguages.includes(document.languageId);
    }

    /**
     * Get suggestions with caching
     */
    private async getSuggestions(document: vscode.TextDocument): Promise<LearningSuggestion[]> {
        const path = document.uri.fsPath;
        const cached = this.cache.get(path);
        if (cached && (Date.now() - cached.timestamp) < this.cacheTTL) {
            return cached.suggestions;
        }

        if (!this.bridge) return [];

        try {
            // Extract context from the document for semantic search
            const context = this.extractContext(document);
            if (!context) {
                return [];
            }

            // Perform semantic search
            const results = await this.bridge.semanticSearch(context, {
                kinds: ['learning'],
                limit: 5,
                minSimilarity: 0.3,
            });

            const suggestions: LearningSuggestion[] = results.map(r => ({
                id: r.id,
                content: r.content,
                similarity: r.similarity,
                kind: r.kind,
                createdAt: r.createdAt,
            }));

            this.cache.set(path, {
                suggestions,
                timestamp: Date.now(),
            });

            return suggestions;
        } catch {
            return [];
        }
    }

    /**
     * Extract meaningful context from the document for semantic search
     */
    private extractContext(document: vscode.TextDocument): string {
        const text = document.getText();
        const lines = text.split('\n');

        // Extract key elements: class/function names, comments, imports
        const contextParts: string[] = [];
        const fileName = document.fileName.split('/').pop() || '';
        contextParts.push(fileName);

        // Look for class/function definitions and comments
        for (let i = 0; i < Math.min(lines.length, 100); i++) {
            const line = lines[i].trim();

            // Skip empty lines
            if (!line) continue;

            // Include comments (often contain intent/documentation)
            if (line.startsWith('//') || line.startsWith('#') || line.startsWith('/*') || line.startsWith('*')) {
                contextParts.push(line.replace(/^[\/\*#\s]+/, '').trim());
                continue;
            }

            // Include function/class/type definitions
            if (/^(function|class|type|interface|const|let|var|def|func|fn|pub|async|export)\s/.test(line)) {
                // Extract just the name
                const match = line.match(/(?:function|class|type|interface|const|let|var|def|func|fn)\s+(\w+)/);
                if (match) {
                    contextParts.push(match[1]);
                }
            }
        }

        // Combine and limit length
        const context = contextParts.slice(0, 20).join(' ');
        return context.slice(0, 500);
    }

    /**
     * Create CodeLens items for learning suggestions
     */
    private createCodeLenses(
        document: vscode.TextDocument,
        suggestions: LearningSuggestion[]
    ): vscode.CodeLens[] {
        const lenses: vscode.CodeLens[] = [];
        const range = new vscode.Range(0, 0, 0, 0);

        // Main suggestion indicator
        const mainTitle = `$(book) ${suggestions.length} relevant learning${suggestions.length > 1 ? 's' : ''}`;
        lenses.push(new vscode.CodeLens(range, {
            title: mainTitle,
            command: 'mindPalace.showLearningSuggestions',
            arguments: [document.uri.fsPath, suggestions],
            tooltip: 'Click to view relevant learnings for this file',
        }));

        // Show top suggestion inline if highly relevant
        if (suggestions.length > 0 && suggestions[0].similarity > 0.5) {
            const top = suggestions[0];
            const shortContent = this.truncate(top.content, 60);
            lenses.push(new vscode.CodeLens(range, {
                title: `$(lightbulb) ${shortContent}`,
                command: 'mindPalace.showLearningDetail',
                arguments: [top],
                tooltip: `${Math.round(top.similarity * 100)}% relevant: ${top.content}`,
            }));
        }

        return lenses;
    }

    /**
     * Truncate text with ellipsis
     */
    private truncate(text: string, maxLength: number): string {
        if (text.length <= maxLength) return text;
        return text.slice(0, maxLength - 3) + '...';
    }

    /**
     * Dispose resources
     */
    dispose(): void {
        if (this.debounceTimer) {
            clearTimeout(this.debounceTimer);
        }
        this._onDidChangeCodeLenses.dispose();
    }
}
