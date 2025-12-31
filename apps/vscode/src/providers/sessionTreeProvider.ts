import * as vscode from 'vscode';
import { PalaceBridge, SessionInfo } from '../bridge';

/**
 * Tree item types for the session tree
 */
type SessionTreeItemType = 'category' | 'session';

/**
 * SessionTreeItem represents a node in the session tree view.
 */
export class SessionTreeItem extends vscode.TreeItem {
    constructor(
        public readonly itemType: SessionTreeItemType,
        label: string,
        public readonly session?: SessionInfo,
        collapsibleState?: vscode.TreeItemCollapsibleState
    ) {
        super(label, collapsibleState ?? vscode.TreeItemCollapsibleState.None);
        this.contextValue = itemType;

        if (itemType === 'session' && session) {
            this.setupSession(session);
        }
    }

    private setupSession(session: SessionInfo): void {
        // Icon based on state
        switch (session.state) {
            case 'active':
                this.iconPath = new vscode.ThemeIcon('debug-start', new vscode.ThemeColor('debugIcon.startForeground'));
                break;
            case 'completed':
                this.iconPath = new vscode.ThemeIcon('check', new vscode.ThemeColor('testing.iconPassed'));
                break;
            case 'abandoned':
                this.iconPath = new vscode.ThemeIcon('close', new vscode.ThemeColor('testing.iconFailed'));
                break;
        }

        // Description: agent type and goal
        const parts: string[] = [session.agentType];
        if (session.goal) {
            const truncatedGoal = session.goal.length > 30
                ? session.goal.substring(0, 30) + '...'
                : session.goal;
            parts.push(truncatedGoal);
        }
        this.description = parts.join(' - ');

        // Tooltip with full details
        const tooltipLines = [
            `**Session ID:** ${session.id}`,
            `**Agent:** ${session.agentType}`,
            `**State:** ${session.state}`,
            `**Started:** ${new Date(session.startedAt).toLocaleString()}`,
        ];
        if (session.goal) {
            tooltipLines.push(`**Goal:** ${session.goal}`);
        }
        if (session.summary) {
            tooltipLines.push(`**Summary:** ${session.summary}`);
        }
        this.tooltip = new vscode.MarkdownString(tooltipLines.join('\n\n'));

        // Command to show session details
        this.command = {
            command: 'mindPalace.showSessionDetail',
            title: 'Show Session Details',
            arguments: [session],
        };
    }
}

/**
 * Cached session data
 */
interface CachedSessions {
    sessions: SessionInfo[];
    timestamp: number;
}

/**
 * SessionTreeProvider provides the tree data for the Sessions view.
 */
export class SessionTreeProvider implements vscode.TreeDataProvider<SessionTreeItem> {
    private bridge?: PalaceBridge;
    private sessions: SessionInfo[] = [];

    private _onDidChangeTreeData = new vscode.EventEmitter<SessionTreeItem | undefined | null | void>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

    // Cache settings
    private cache?: CachedSessions;
    private cacheTTL = 30000; // 30 seconds TTL (sessions change more frequently)

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
            this.sessions = this.cache.sessions;
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
            this.sessions = await this.bridge!.listSessions({ limit: 50 });

            // Update cache
            this.cache = {
                sessions: this.sessions,
                timestamp: Date.now(),
            };
        } catch {
            this.sessions = [];
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
    getTreeItem(element: SessionTreeItem): vscode.TreeItem {
        return element;
    }

    /**
     * Get children of a tree item
     */
    async getChildren(element?: SessionTreeItem): Promise<SessionTreeItem[]> {
        if (!this.bridge) {
            return [];
        }

        // Root level - show categories
        if (!element) {
            // Use cache if available and valid, otherwise fetch
            if (this.sessions.length === 0) {
                if (this.cache && (Date.now() - this.cache.timestamp) < this.cacheTTL) {
                    this.sessions = this.cache.sessions;
                } else {
                    try {
                        this.sessions = await this.bridge.listSessions({ limit: 50 });
                        this.cache = {
                            sessions: this.sessions,
                            timestamp: Date.now(),
                        };
                    } catch {
                        this.sessions = [];
                    }
                }
            }

            const activeSessions = this.sessions.filter(s => s.state === 'active');
            const completedSessions = this.sessions.filter(s => s.state === 'completed');
            const abandonedSessions = this.sessions.filter(s => s.state === 'abandoned');

            const items: SessionTreeItem[] = [];

            // Active sessions category
            if (activeSessions.length > 0) {
                const activeItem = new SessionTreeItem(
                    'category',
                    `Active (${activeSessions.length})`,
                    undefined,
                    vscode.TreeItemCollapsibleState.Expanded
                );
                activeItem.iconPath = new vscode.ThemeIcon('debug-start');
                activeItem.contextValue = 'category_active';
                items.push(activeItem);
            }

            // Completed sessions category
            if (completedSessions.length > 0) {
                const completedItem = new SessionTreeItem(
                    'category',
                    `Completed (${completedSessions.length})`,
                    undefined,
                    vscode.TreeItemCollapsibleState.Collapsed
                );
                completedItem.iconPath = new vscode.ThemeIcon('check');
                completedItem.contextValue = 'category_completed';
                items.push(completedItem);
            }

            // Abandoned sessions category
            if (abandonedSessions.length > 0) {
                const abandonedItem = new SessionTreeItem(
                    'category',
                    `Abandoned (${abandonedSessions.length})`,
                    undefined,
                    vscode.TreeItemCollapsibleState.Collapsed
                );
                abandonedItem.iconPath = new vscode.ThemeIcon('close');
                abandonedItem.contextValue = 'category_abandoned';
                items.push(abandonedItem);
            }

            // If no sessions at all, show a placeholder
            if (items.length === 0) {
                const emptyItem = new SessionTreeItem('category', 'No sessions yet');
                emptyItem.iconPath = new vscode.ThemeIcon('info');
                emptyItem.description = 'Start a session to track agent activities';
                return [emptyItem];
            }

            return items;
        }

        // Category level - show sessions in that category
        if (element.itemType === 'category') {
            let filteredSessions: SessionInfo[] = [];

            if (element.contextValue === 'category_active') {
                filteredSessions = this.sessions.filter(s => s.state === 'active');
            } else if (element.contextValue === 'category_completed') {
                filteredSessions = this.sessions.filter(s => s.state === 'completed');
            } else if (element.contextValue === 'category_abandoned') {
                filteredSessions = this.sessions.filter(s => s.state === 'abandoned');
            }

            return filteredSessions.map(session => {
                const displayId = session.id.length > 12 ? session.id.substring(0, 12) + '...' : session.id;
                return new SessionTreeItem('session', displayId, session);
            });
        }

        return [];
    }
}
