import * as vscode from 'vscode';
import { PalaceBridge, CallGraphResult } from '../bridge';

/**
 * Cached call graph data
 */
interface CachedCallGraph {
    data: CallGraphResult;
    timestamp: number;
}

/**
 * CallGraphHoverProvider shows callers and callees when hovering over function names.
 * It uses Mind Palace's call graph data to provide navigation information.
 */
export class CallGraphHoverProvider implements vscode.HoverProvider {
    private bridge?: PalaceBridge;
    private cache = new Map<string, CachedCallGraph>();
    private cacheTTL = 120000; // 2 minutes TTL

    constructor() {}

    /**
     * Set the bridge for MCP communication
     */
    setBridge(bridge: PalaceBridge): void {
        this.bridge = bridge;
    }

    /**
     * Provide hover information for the given position
     */
    async provideHover(
        document: vscode.TextDocument,
        position: vscode.Position,
        token: vscode.CancellationToken
    ): Promise<vscode.Hover | null> {
        if (!this.bridge) {
            return null;
        }

        const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
        if (!workspaceRoot) {
            return null;
        }

        const filePath = document.uri.fsPath;
        if (!filePath.startsWith(workspaceRoot)) {
            return null;
        }

        // Get the word at the current position
        const wordRange = document.getWordRangeAtPosition(position);
        if (!wordRange) {
            return null;
        }

        const word = document.getText(wordRange);
        if (!word || word.length < 2) {
            return null;
        }

        // Check if this looks like a function definition or call
        if (!this.isFunctionContext(document, position, word)) {
            return null;
        }

        const relativePath = filePath.substring(workspaceRoot.length + 1);

        try {
            const callGraph = await this.getCallGraph(word, relativePath);

            if (!callGraph) {
                return null;
            }

            // Only show hover if there are callers or callees
            if (callGraph.callers.length === 0 && callGraph.callees.length === 0) {
                return null;
            }

            const markdown = this.buildHoverMarkdown(callGraph, word, relativePath);
            return new vscode.Hover(markdown, wordRange);
        } catch {
            return null;
        }
    }

    /**
     * Check if the position looks like a function context
     */
    private isFunctionContext(
        document: vscode.TextDocument,
        position: vscode.Position,
        word: string
    ): boolean {
        const line = document.lineAt(position.line).text;

        // Common function definition patterns
        const functionPatterns = [
            /function\s+\w+/,           // function foo
            /func\s+\w+/,               // func foo (Go)
            /def\s+\w+/,                // def foo (Python)
            /fn\s+\w+/,                 // fn foo (Rust)
            /\w+\s*\([^)]*\)\s*{/,      // foo() { (JS methods)
            /\w+\s*\([^)]*\)\s*=>/,     // foo() => (arrow functions)
            /async\s+\w+/,              // async foo
            /public\s+\w+/,             // public foo
            /private\s+\w+/,            // private foo
            /protected\s+\w+/,          // protected foo
            /static\s+\w+/,             // static foo
            /\w+\s*:\s*function/,       // foo: function
            /\w+\s*=\s*function/,       // foo = function
            /\w+\s*=\s*\([^)]*\)\s*=>/, // foo = () =>
        ];

        // Check if line matches any function pattern
        for (const pattern of functionPatterns) {
            if (pattern.test(line)) {
                return true;
            }
        }

        // Also check for function calls: word followed by parenthesis
        const afterWord = line.substring(line.indexOf(word) + word.length);
        if (afterWord.trimStart().startsWith('(')) {
            return true;
        }

        return false;
    }

    /**
     * Get call graph with caching
     */
    private async getCallGraph(symbol: string, file: string): Promise<CallGraphResult | null> {
        const cacheKey = `${file}:${symbol}`;
        const cached = this.cache.get(cacheKey);
        if (cached && (Date.now() - cached.timestamp) < this.cacheTTL) {
            return cached.data;
        }

        if (!this.bridge) return null;

        try {
            // Get callers and callees
            const [callers, callees] = await Promise.all([
                this.bridge.getCallers(symbol).catch(() => ({ symbol, callers: [], callees: [] })),
                this.bridge.getCallees(symbol, file).catch(() => ({ symbol, callers: [], callees: [] })),
            ]);

            const result: CallGraphResult = {
                symbol,
                callers: callers.callers || [],
                callees: callees.callees || [],
            };

            this.cache.set(cacheKey, {
                data: result,
                timestamp: Date.now(),
            });

            return result;
        } catch {
            return null;
        }
    }

    /**
     * Build markdown content for hover
     */
    private buildHoverMarkdown(
        callGraph: CallGraphResult,
        symbol: string,
        currentFile: string
    ): vscode.MarkdownString {
        const md = new vscode.MarkdownString();
        md.isTrusted = true;
        md.supportHtml = true;

        md.appendMarkdown(`### Mind Palace - Call Graph\n\n`);
        md.appendMarkdown(`**${symbol}**\n\n`);

        // Show callers (who calls this function)
        if (callGraph.callers.length > 0) {
            md.appendMarkdown(`#### $(arrow-left) Called by (${callGraph.callers.length})\n\n`);

            const displayCallers = callGraph.callers.slice(0, 5);
            for (const caller of displayCallers) {
                const fileLink = this.createFileLink(caller.file, caller.line);
                md.appendMarkdown(`- \`${caller.symbol}\` in ${fileLink}\n`);
            }

            if (callGraph.callers.length > 5) {
                md.appendMarkdown(`- *... and ${callGraph.callers.length - 5} more*\n`);
            }
            md.appendMarkdown('\n');
        }

        // Show callees (what this function calls)
        if (callGraph.callees.length > 0) {
            md.appendMarkdown(`#### $(arrow-right) Calls (${callGraph.callees.length})\n\n`);

            const displayCallees = callGraph.callees.slice(0, 5);
            for (const callee of displayCallees) {
                const fileLink = this.createFileLink(callee.file, callee.line);
                md.appendMarkdown(`- \`${callee.symbol}\` in ${fileLink}\n`);
            }

            if (callGraph.callees.length > 5) {
                md.appendMarkdown(`- *... and ${callGraph.callees.length - 5} more*\n`);
            }
        }

        // Add action links
        md.appendMarkdown('\n---\n');
        md.appendMarkdown(`[View Full Graph](command:mindPalace.showCallGraph?${encodeURIComponent(JSON.stringify({ symbol, file: currentFile }))})`);

        return md;
    }

    /**
     * Create a clickable file link
     */
    private createFileLink(file: string, line: number): string {
        const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath || '';
        const fullPath = file.startsWith('/') ? file : `${workspaceRoot}/${file}`;
        const uri = vscode.Uri.file(fullPath).with({ fragment: `L${line}` });

        // Use command URI to open file at line
        const args = encodeURIComponent(JSON.stringify([uri.toString(), { selection: { startLine: line - 1, startColumn: 0 } }]));
        return `[${file}:${line}](command:vscode.open?${args})`;
    }

    /**
     * Clear the cache
     */
    clearCache(): void {
        this.cache.clear();
    }
}
