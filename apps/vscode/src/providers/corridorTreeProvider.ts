import * as vscode from 'vscode';
import { PalaceBridge, CorridorLearning, LinkedWorkspace } from '../bridge';

/**
 * Tree item types for the corridor tree
 */
type CorridorTreeItemType = 'category' | 'learning' | 'workspace';

/**
 * CorridorTreeItem represents a node in the corridor tree view.
 */
export class CorridorTreeItem extends vscode.TreeItem {
    constructor(
        public readonly itemType: CorridorTreeItemType,
        label: string,
        public readonly learning?: CorridorLearning,
        public readonly workspace?: LinkedWorkspace,
        collapsibleState?: vscode.TreeItemCollapsibleState
    ) {
        super(label, collapsibleState ?? vscode.TreeItemCollapsibleState.None);
        this.contextValue = itemType;

        if (itemType === 'learning' && learning) {
            this.setupLearning(learning);
        } else if (itemType === 'workspace' && workspace) {
            this.setupWorkspace(workspace);
        }
    }

    private setupLearning(learning: CorridorLearning): void {
        // Icon based on confidence
        if (learning.confidence >= 0.8) {
            this.iconPath = new vscode.ThemeIcon('star-full', new vscode.ThemeColor('charts.yellow'));
        } else if (learning.confidence >= 0.5) {
            this.iconPath = new vscode.ThemeIcon('star-half', new vscode.ThemeColor('charts.yellow'));
        } else {
            this.iconPath = new vscode.ThemeIcon('star-empty');
        }

        // Description: origin workspace and confidence
        const parts: string[] = [];
        if (learning.originWorkspace) {
            parts.push(learning.originWorkspace);
        }
        parts.push(`${(learning.confidence * 100).toFixed(0)}%`);
        this.description = parts.join(' - ');

        // Tooltip with full details
        const tooltipLines = [
            `**Content:** ${learning.content}`,
            `**Confidence:** ${(learning.confidence * 100).toFixed(0)}%`,
            `**Origin:** ${learning.originWorkspace || 'Unknown'}`,
            `**Used:** ${learning.useCount} times`,
        ];
        if (learning.tags && learning.tags.length > 0) {
            tooltipLines.push(`**Tags:** ${learning.tags.join(', ')}`);
        }
        this.tooltip = new vscode.MarkdownString(tooltipLines.join('\n\n'));

        // Command to show learning details
        this.command = {
            command: 'mindPalace.showCorridorLearningDetail',
            title: 'Show Learning Details',
            arguments: [learning],
        };
    }

    private setupWorkspace(workspace: LinkedWorkspace): void {
        this.iconPath = new vscode.ThemeIcon('folder-library');
        this.description = workspace.path;

        const tooltipLines = [
            `**Name:** ${workspace.name}`,
            `**Path:** ${workspace.path}`,
        ];
        if (workspace.lastAccessed) {
            tooltipLines.push(`**Last Accessed:** ${new Date(workspace.lastAccessed).toLocaleString()}`);
        }
        this.tooltip = new vscode.MarkdownString(tooltipLines.join('\n\n'));
    }
}

/**
 * Cached corridor data
 */
interface CachedCorridor {
    learnings: CorridorLearning[];
    workspaces: LinkedWorkspace[];
    timestamp: number;
}

/**
 * CorridorTreeProvider provides the tree data for the Corridor view.
 */
export class CorridorTreeProvider implements vscode.TreeDataProvider<CorridorTreeItem> {
    private bridge?: PalaceBridge;
    private learnings: CorridorLearning[] = [];
    private workspaces: LinkedWorkspace[] = [];

    private _onDidChangeTreeData = new vscode.EventEmitter<CorridorTreeItem | undefined | null | void>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

    // Cache settings
    private cache?: CachedCorridor;
    private cacheTTL = 60000; // 1 minute TTL

    // Request deduplication
    private pendingRefresh?: Promise<void>;

    constructor() {}

    /**
     * Set the bridge for MCP communication
     */
    setBridge(bridge: PalaceBridge): void {
        this.bridge = bridge;
    }

    /**
     * Refresh the tree view
     * @param forceRefresh If true, bypasses cache and fetches fresh data
     */
    async refresh(forceRefresh: boolean = true): Promise<void> {
        if (!this.bridge) {
            this._onDidChangeTreeData.fire();
            return;
        }

        // Check cache validity (only if not forcing refresh)
        if (!forceRefresh && this.cache && (Date.now() - this.cache.timestamp) < this.cacheTTL) {
            this.learnings = this.cache.learnings;
            this.workspaces = this.cache.workspaces;
            this._onDidChangeTreeData.fire();
            return;
        }

        // Deduplicate concurrent refresh requests
        if (this.pendingRefresh) {
            return this.pendingRefresh;
        }

        this.pendingRefresh = this.doRefresh();
        try {
            await this.pendingRefresh;
        } finally {
            this.pendingRefresh = undefined;
        }
    }

    private async doRefresh(): Promise<void> {
        try {
            const [learnings, workspaces] = await Promise.all([
                this.bridge!.getCorridorLearnings({ limit: 50 }),
                this.bridge!.getCorridorLinks(),
            ]);
            this.learnings = learnings;
            this.workspaces = workspaces;

            // Update cache
            this.cache = {
                learnings: this.learnings,
                workspaces: this.workspaces,
                timestamp: Date.now(),
            };
        } catch {
            this.learnings = [];
            this.workspaces = [];
        }

        this._onDidChangeTreeData.fire();
    }

    /**
     * Clear the cache
     */
    clearCache(): void {
        this.cache = undefined;
    }

    /**
     * Get tree item for display
     */
    getTreeItem(element: CorridorTreeItem): vscode.TreeItem {
        return element;
    }

    /**
     * Get children of a tree item
     */
    async getChildren(element?: CorridorTreeItem): Promise<CorridorTreeItem[]> {
        if (!this.bridge) {
            return [];
        }

        // Root level - show categories
        if (!element) {
            // Use cache if available and valid, otherwise fetch
            if (this.learnings.length === 0 && this.workspaces.length === 0) {
                if (this.cache && (Date.now() - this.cache.timestamp) < this.cacheTTL) {
                    this.learnings = this.cache.learnings;
                    this.workspaces = this.cache.workspaces;
                } else {
                    try {
                        const [learnings, workspaces] = await Promise.all([
                            this.bridge.getCorridorLearnings({ limit: 50 }),
                            this.bridge.getCorridorLinks(),
                        ]);
                        this.learnings = learnings;
                        this.workspaces = workspaces;
                        this.cache = {
                            learnings: this.learnings,
                            workspaces: this.workspaces,
                            timestamp: Date.now(),
                        };
                    } catch {
                        this.learnings = [];
                        this.workspaces = [];
                    }
                }
            }

            const items: CorridorTreeItem[] = [];

            // Personal Learnings category
            const learningsItem = new CorridorTreeItem(
                'category',
                `Personal Learnings (${this.learnings.length})`,
                undefined,
                undefined,
                this.learnings.length > 0
                    ? vscode.TreeItemCollapsibleState.Expanded
                    : vscode.TreeItemCollapsibleState.Collapsed
            );
            learningsItem.iconPath = new vscode.ThemeIcon('book');
            learningsItem.contextValue = 'category_learnings';
            items.push(learningsItem);

            // Linked Workspaces category
            const workspacesItem = new CorridorTreeItem(
                'category',
                `Linked Workspaces (${this.workspaces.length})`,
                undefined,
                undefined,
                this.workspaces.length > 0
                    ? vscode.TreeItemCollapsibleState.Collapsed
                    : vscode.TreeItemCollapsibleState.None
            );
            workspacesItem.iconPath = new vscode.ThemeIcon('folder-library');
            workspacesItem.contextValue = 'category_workspaces';
            items.push(workspacesItem);

            return items;
        }

        // Category level - show items in that category
        if (element.itemType === 'category') {
            if (element.contextValue === 'category_learnings') {
                if (this.learnings.length === 0) {
                    const emptyItem = new CorridorTreeItem('category', 'No personal learnings yet');
                    emptyItem.iconPath = new vscode.ThemeIcon('info');
                    emptyItem.description = 'Promote learnings from workspaces';
                    return [emptyItem];
                }

                return this.learnings.map(learning => {
                    const truncatedContent = learning.content.length > 50
                        ? learning.content.substring(0, 50) + '...'
                        : learning.content;
                    return new CorridorTreeItem('learning', truncatedContent, learning);
                });
            }

            if (element.contextValue === 'category_workspaces') {
                if (this.workspaces.length === 0) {
                    const emptyItem = new CorridorTreeItem('category', 'No workspaces linked');
                    emptyItem.iconPath = new vscode.ThemeIcon('info');
                    emptyItem.description = 'Use `palace corridor link`';
                    return [emptyItem];
                }

                return this.workspaces.map(workspace =>
                    new CorridorTreeItem('workspace', workspace.name, undefined, workspace)
                );
            }
        }

        return [];
    }
}
