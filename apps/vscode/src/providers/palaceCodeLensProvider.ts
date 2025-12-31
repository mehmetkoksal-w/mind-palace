import * as vscode from 'vscode';
import { PalaceBridge, FileIntelResult } from '../bridge';

/**
 * Cached file intel for CodeLens
 */
interface CachedIntel {
    data: FileIntelResult;
    timestamp: number;
}

/**
 * PalaceCodeLensProvider shows Mind Palace intelligence as CodeLens
 * at the top of files, showing learning counts, decision counts, and file status.
 */
export class PalaceCodeLensProvider implements vscode.CodeLensProvider {
    private bridge?: PalaceBridge;
    private cache = new Map<string, CachedIntel>();
    private cacheTTL = 60000; // 1 minute TTL

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
     * Refresh CodeLens for all documents
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
     * Provide CodeLens for the document
     */
    async provideCodeLenses(
        document: vscode.TextDocument,
        token: vscode.CancellationToken
    ): Promise<vscode.CodeLens[]> {
        if (!this.bridge) {
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

        const relativePath = filePath.substring(workspaceRoot.length + 1);

        try {
            const intel = await this.getFileIntel(relativePath);

            if (!intel) {
                return [];
            }

            return this.createCodeLenses(document, intel);
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
        // CodeLens is already resolved in provideCodeLenses
        return codeLens;
    }

    /**
     * Get file intel with caching
     */
    private async getFileIntel(path: string): Promise<FileIntelResult | null> {
        const cached = this.cache.get(path);
        if (cached && (Date.now() - cached.timestamp) < this.cacheTTL) {
            return cached.data;
        }

        if (!this.bridge) return null;

        try {
            const intel = await this.bridge.getFileIntel(path);

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
     * Create CodeLens items for the document
     */
    private createCodeLenses(
        document: vscode.TextDocument,
        intel: FileIntelResult
    ): vscode.CodeLens[] {
        const lenses: vscode.CodeLens[] = [];

        // Only show CodeLens if there's something interesting
        const hasData = intel.editCount > 0 ||
                       intel.failureCount > 0 ||
                       (intel.learnings && intel.learnings.length > 0);

        if (!hasData) {
            return [];
        }

        // Build title parts
        const parts: string[] = [];

        if (intel.learnings && intel.learnings.length > 0) {
            parts.push(`$(lightbulb) ${intel.learnings.length} learning${intel.learnings.length > 1 ? 's' : ''}`);
        }

        if (intel.editCount > 5) {
            const status = intel.editCount > 10 ? 'Hot' : 'Active';
            parts.push(`$(flame) ${status} (${intel.editCount} edits)`);
        }

        if (intel.failureCount > 0) {
            parts.push(`$(warning) ${intel.failureCount} failure${intel.failureCount > 1 ? 's' : ''}`);
        }

        if (parts.length === 0) {
            return [];
        }

        // Create CodeLens at line 0
        const range = new vscode.Range(0, 0, 0, 0);

        lenses.push(new vscode.CodeLens(range, {
            title: parts.join(' | '),
            command: 'mindPalace.showFileIntel',
            arguments: [document.uri.fsPath],
            tooltip: 'Click to view Mind Palace intelligence for this file',
        }));

        return lenses;
    }
}
