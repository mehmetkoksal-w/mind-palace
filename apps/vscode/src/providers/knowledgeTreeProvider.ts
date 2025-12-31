import * as vscode from 'vscode';
import { PalaceBridge } from '../bridge';

/**
 * Tree item types for the Knowledge panel
 */
export enum KnowledgeItemType {
    Category = 'category',
    StatusGroup = 'status_group',
    ScopeGroup = 'scope_group',
    Idea = 'idea',
    Decision = 'decision',
    Learning = 'learning',
}

/**
 * Status icons for different states
 */
const STATUS_ICONS: Record<string, string> = {
    // Ideas
    'active': '$(lightbulb)',
    'exploring': '$(beaker)',
    'implemented': '$(check)',
    'dropped': '$(x)',
    // Decisions
    'pending': '$(tools)',
    'superseded': '$(history)',
    'reversed': '$(arrow-left)',
    'has_outcome': '$(graph)',
    // Generic
    'success': '$(pass)',
    'failure': '$(error)',
    'mixed': '$(warning)',
};

/**
 * Scope icons
 */
const SCOPE_ICONS: Record<string, string> = {
    'palace': '$(home)',
    'room': '$(folder)',
    'file': '$(file)',
};

/**
 * Knowledge tree item representing items in the sidebar
 */
export class KnowledgeTreeItem extends vscode.TreeItem {
    constructor(
        public readonly label: string,
        public readonly itemType: KnowledgeItemType,
        public readonly collapsibleState: vscode.TreeItemCollapsibleState,
        public readonly data?: any,
        public readonly parent?: KnowledgeTreeItem,
    ) {
        super(label, collapsibleState);
        this.contextValue = itemType;
        this.setupItem();
    }

    private setupItem(): void {
        switch (this.itemType) {
            case KnowledgeItemType.Category:
                this.setupCategory();
                break;
            case KnowledgeItemType.StatusGroup:
                this.setupStatusGroup();
                break;
            case KnowledgeItemType.ScopeGroup:
                this.setupScopeGroup();
                break;
            case KnowledgeItemType.Idea:
                this.setupIdea();
                break;
            case KnowledgeItemType.Decision:
                this.setupDecision();
                break;
            case KnowledgeItemType.Learning:
                this.setupLearning();
                break;
        }
    }

    private setupCategory(): void {
        const count = this.data?.count ?? 0;
        this.description = `(${count})`;

        switch (this.label) {
            case 'Ideas':
                this.iconPath = new vscode.ThemeIcon('lightbulb');
                break;
            case 'Decisions':
                this.iconPath = new vscode.ThemeIcon('law');
                break;
            case 'Learnings':
                this.iconPath = new vscode.ThemeIcon('book');
                break;
        }
    }

    private setupStatusGroup(): void {
        const count = this.data?.count ?? 0;
        this.description = `(${count})`;

        const status = this.data?.status;
        const iconName = STATUS_ICONS[status] ?? '$(circle)';
        // Extract the icon name from $(iconName) format
        const iconId = iconName.replace('$(', '').replace(')', '');
        this.iconPath = new vscode.ThemeIcon(iconId);
    }

    private setupScopeGroup(): void {
        const count = this.data?.count ?? 0;
        this.description = `(${count})`;

        const scope = this.data?.scope;
        const iconName = SCOPE_ICONS[scope] ?? '$(circle)';
        const iconId = iconName.replace('$(', '').replace(')', '');
        this.iconPath = new vscode.ThemeIcon(iconId);
    }

    private setupIdea(): void {
        const idea = this.data;
        this.tooltip = new vscode.MarkdownString();
        this.tooltip.appendMarkdown(`**Idea**: ${idea.content}\n\n`);
        this.tooltip.appendMarkdown(`**Status**: ${idea.status}\n\n`);
        this.tooltip.appendMarkdown(`**Scope**: ${idea.scope}`);
        if (idea.scopePath) {
            this.tooltip.appendMarkdown(` (${idea.scopePath})`);
        }

        this.description = idea.scopePath ? `[${this.getScopeLabel(idea.scope, idea.scopePath)}]` : '';

        const status = idea.status || 'active';
        const iconId = (STATUS_ICONS[status] ?? '$(lightbulb)').replace('$(', '').replace(')', '');
        this.iconPath = new vscode.ThemeIcon(iconId);

        // Click to show detail
        this.command = {
            command: 'mindPalace.showKnowledgeDetail',
            title: 'Show Detail',
            arguments: [{ type: 'idea', data: idea }],
        };
    }

    private setupDecision(): void {
        const decision = this.data;
        this.tooltip = new vscode.MarkdownString();
        this.tooltip.appendMarkdown(`**Decision**: ${decision.content}\n\n`);
        this.tooltip.appendMarkdown(`**Status**: ${decision.status}\n\n`);
        this.tooltip.appendMarkdown(`**Scope**: ${decision.scope}`);
        if (decision.scopePath) {
            this.tooltip.appendMarkdown(` (${decision.scopePath})`);
        }
        if (decision.outcome) {
            this.tooltip.appendMarkdown(`\n\n**Outcome**: ${decision.outcome}`);
        }

        this.description = decision.scopePath ? `[${this.getScopeLabel(decision.scope, decision.scopePath)}]` : '';

        const status = decision.outcome ? 'has_outcome' : (decision.status || 'pending');
        const iconId = (STATUS_ICONS[status] ?? '$(law)').replace('$(', '').replace(')', '');
        this.iconPath = new vscode.ThemeIcon(iconId);

        this.command = {
            command: 'mindPalace.showKnowledgeDetail',
            title: 'Show Detail',
            arguments: [{ type: 'decision', data: decision }],
        };
    }

    private setupLearning(): void {
        const learning = this.data;
        this.tooltip = new vscode.MarkdownString();
        this.tooltip.appendMarkdown(`**Learning**: ${learning.content}\n\n`);
        this.tooltip.appendMarkdown(`**Confidence**: ${Math.round((learning.confidence ?? 0.5) * 100)}%\n\n`);
        this.tooltip.appendMarkdown(`**Scope**: ${learning.scope}`);
        if (learning.scopePath) {
            this.tooltip.appendMarkdown(` (${learning.scopePath})`);
        }

        this.description = learning.confidence ? `${Math.round(learning.confidence * 100)}%` : '';

        // Icon based on confidence
        const confidence = learning.confidence ?? 0.5;
        let iconId = 'book';
        if (confidence >= 0.8) {
            iconId = 'verified';
        } else if (confidence < 0.5) {
            iconId = 'question';
        }
        this.iconPath = new vscode.ThemeIcon(iconId);

        this.command = {
            command: 'mindPalace.showKnowledgeDetail',
            title: 'Show Detail',
            arguments: [{ type: 'learning', data: learning }],
        };
    }

    private getScopeLabel(scope: string, scopePath: string): string {
        if (scope === 'file') {
            // Just filename
            return scopePath.split('/').pop() ?? scopePath;
        }
        if (scope === 'room') {
            return scopePath;
        }
        return '';
    }
}

/**
 * Cached knowledge data
 */
interface CachedKnowledge {
    ideas: any[];
    decisions: any[];
    learnings: any[];
    timestamp: number;
}

/**
 * Tree data provider for the Knowledge panel
 */
export class KnowledgeTreeProvider implements vscode.TreeDataProvider<KnowledgeTreeItem> {
    private _onDidChangeTreeData = new vscode.EventEmitter<KnowledgeTreeItem | undefined | null | void>();
    readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

    private bridge?: PalaceBridge;
    private ideas: any[] = [];
    private decisions: any[] = [];
    private learnings: any[] = [];
    private isLoading = false;
    private lastError?: string;

    // Cache settings
    private cache?: CachedKnowledge;
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
     * Refresh the tree data
     * @param forceRefresh If true, bypasses cache and fetches fresh data
     */
    async refresh(forceRefresh: boolean = true): Promise<void> {
        if (!this.bridge) {
            this._onDidChangeTreeData.fire();
            return;
        }

        // Check cache validity (only if not forcing refresh)
        if (!forceRefresh && this.cache && (Date.now() - this.cache.timestamp) < this.cacheTTL) {
            this.ideas = this.cache.ideas;
            this.decisions = this.cache.decisions;
            this.learnings = this.cache.learnings;
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
        this.isLoading = true;
        this._onDidChangeTreeData.fire();

        try {
            // Fetch all data in parallel
            const [ideasResult, decisionsResult, learningsResult] = await Promise.all([
                this.bridge!.recallIdeas({ limit: 100 }).catch(() => ({ ideas: [] })),
                this.bridge!.recallDecisions({ limit: 100 }).catch(() => ({ decisions: [] })),
                this.bridge!.recallLearnings({ limit: 100 }).catch(() => ({ learnings: [] })),
            ]);

            this.ideas = ideasResult.ideas ?? [];
            this.decisions = decisionsResult.decisions ?? [];
            this.learnings = learningsResult.learnings ?? [];
            this.lastError = undefined;

            // Update cache
            this.cache = {
                ideas: this.ideas,
                decisions: this.decisions,
                learnings: this.learnings,
                timestamp: Date.now(),
            };
        } catch (error: any) {
            this.lastError = error.message;
            this.ideas = [];
            this.decisions = [];
            this.learnings = [];
        }

        this.isLoading = false;
        this._onDidChangeTreeData.fire();
    }

    /**
     * Clear the cache
     */
    clearCache(): void {
        this.cache = undefined;
    }

    getTreeItem(element: KnowledgeTreeItem): vscode.TreeItem {
        return element;
    }

    async getChildren(element?: KnowledgeTreeItem): Promise<KnowledgeTreeItem[]> {
        if (this.isLoading) {
            return [new KnowledgeTreeItem(
                'Loading...',
                KnowledgeItemType.Category,
                vscode.TreeItemCollapsibleState.None,
            )];
        }

        if (this.lastError) {
            const item = new KnowledgeTreeItem(
                `Error: ${this.lastError}`,
                KnowledgeItemType.Category,
                vscode.TreeItemCollapsibleState.None,
            );
            item.iconPath = new vscode.ThemeIcon('error');
            return [item];
        }

        if (!element) {
            // Root level - show categories
            return this.getRootItems();
        }

        // Handle children based on parent type
        switch (element.itemType) {
            case KnowledgeItemType.Category:
                return this.getCategoryChildren(element);
            case KnowledgeItemType.StatusGroup:
                return this.getStatusGroupChildren(element);
            case KnowledgeItemType.ScopeGroup:
                return this.getScopeGroupChildren(element);
            default:
                return [];
        }
    }

    private getRootItems(): KnowledgeTreeItem[] {
        const items: KnowledgeTreeItem[] = [];

        // Ideas category
        items.push(new KnowledgeTreeItem(
            'Ideas',
            KnowledgeItemType.Category,
            this.ideas.length > 0
                ? vscode.TreeItemCollapsibleState.Expanded
                : vscode.TreeItemCollapsibleState.Collapsed,
            { count: this.ideas.length, categoryType: 'ideas' },
        ));

        // Decisions category
        items.push(new KnowledgeTreeItem(
            'Decisions',
            KnowledgeItemType.Category,
            this.decisions.length > 0
                ? vscode.TreeItemCollapsibleState.Expanded
                : vscode.TreeItemCollapsibleState.Collapsed,
            { count: this.decisions.length, categoryType: 'decisions' },
        ));

        // Learnings category
        items.push(new KnowledgeTreeItem(
            'Learnings',
            KnowledgeItemType.Category,
            this.learnings.length > 0
                ? vscode.TreeItemCollapsibleState.Expanded
                : vscode.TreeItemCollapsibleState.Collapsed,
            { count: this.learnings.length, categoryType: 'learnings' },
        ));

        return items;
    }

    private getCategoryChildren(category: KnowledgeTreeItem): KnowledgeTreeItem[] {
        const categoryType = category.data?.categoryType;

        switch (categoryType) {
            case 'ideas':
                return this.getIdeasGroupedByStatus(category);
            case 'decisions':
                return this.getDecisionsGroupedByStatus(category);
            case 'learnings':
                return this.getLearningsGroupedByScope(category);
            default:
                return [];
        }
    }

    private getIdeasGroupedByStatus(parent: KnowledgeTreeItem): KnowledgeTreeItem[] {
        const grouped = this.groupByStatus(this.ideas, ['active', 'exploring', 'implemented', 'dropped']);
        const items: KnowledgeTreeItem[] = [];

        for (const [status, statusIdeas] of Object.entries(grouped)) {
            if (statusIdeas.length === 0) continue;

            const statusLabels: Record<string, string> = {
                'active': 'Active',
                'exploring': 'Exploring',
                'implemented': 'Implemented',
                'dropped': 'Dropped',
            };

            items.push(new KnowledgeTreeItem(
                statusLabels[status] ?? status,
                KnowledgeItemType.StatusGroup,
                vscode.TreeItemCollapsibleState.Expanded,
                { status, count: statusIdeas.length, items: statusIdeas, itemType: 'idea' },
                parent,
            ));
        }

        return items;
    }

    private getDecisionsGroupedByStatus(parent: KnowledgeTreeItem): KnowledgeTreeItem[] {
        const grouped = this.groupByStatus(this.decisions, ['pending', 'active', 'superseded', 'reversed']);
        const items: KnowledgeTreeItem[] = [];

        // Separate decisions with outcomes
        const withOutcome = this.decisions.filter(d => d.outcome);
        const withoutOutcome = this.decisions.filter(d => !d.outcome);

        for (const [status, statusDecisions] of Object.entries(grouped)) {
            const decisionsInStatus = statusDecisions.filter(d => !d.outcome);
            if (decisionsInStatus.length === 0) continue;

            const statusLabels: Record<string, string> = {
                'pending': 'Pending',
                'active': 'Active',
                'superseded': 'Superseded',
                'reversed': 'Reversed',
            };

            items.push(new KnowledgeTreeItem(
                statusLabels[status] ?? status,
                KnowledgeItemType.StatusGroup,
                vscode.TreeItemCollapsibleState.Expanded,
                { status, count: decisionsInStatus.length, items: decisionsInStatus, itemType: 'decision' },
                parent,
            ));
        }

        // Add "Has Outcome" group if there are any
        if (withOutcome.length > 0) {
            items.push(new KnowledgeTreeItem(
                'Has Outcome',
                KnowledgeItemType.StatusGroup,
                vscode.TreeItemCollapsibleState.Expanded,
                { status: 'has_outcome', count: withOutcome.length, items: withOutcome, itemType: 'decision' },
                parent,
            ));
        }

        return items;
    }

    private getLearningsGroupedByScope(parent: KnowledgeTreeItem): KnowledgeTreeItem[] {
        const grouped: Record<string, any[]> = {
            'palace': [],
            'room': [],
            'file': [],
        };

        for (const learning of this.learnings) {
            const scope = learning.scope || 'palace';
            if (grouped[scope]) {
                grouped[scope].push(learning);
            }
        }

        const items: KnowledgeTreeItem[] = [];
        const scopeLabels: Record<string, string> = {
            'palace': 'Palace',
            'room': 'Room',
            'file': 'File',
        };

        for (const [scope, scopeLearnings] of Object.entries(grouped)) {
            if (scopeLearnings.length === 0) continue;

            items.push(new KnowledgeTreeItem(
                scopeLabels[scope] ?? scope,
                KnowledgeItemType.ScopeGroup,
                vscode.TreeItemCollapsibleState.Expanded,
                { scope, count: scopeLearnings.length, items: scopeLearnings },
                parent,
            ));
        }

        return items;
    }

    private getStatusGroupChildren(group: KnowledgeTreeItem): KnowledgeTreeItem[] {
        const items = group.data?.items ?? [];
        const itemType = group.data?.itemType;

        return items.map((item: any) => {
            const truncatedLabel = this.truncateLabel(item.content, 50);
            switch (itemType) {
                case 'idea':
                    return new KnowledgeTreeItem(
                        truncatedLabel,
                        KnowledgeItemType.Idea,
                        vscode.TreeItemCollapsibleState.None,
                        item,
                        group,
                    );
                case 'decision':
                    return new KnowledgeTreeItem(
                        truncatedLabel,
                        KnowledgeItemType.Decision,
                        vscode.TreeItemCollapsibleState.None,
                        item,
                        group,
                    );
                default:
                    return new KnowledgeTreeItem(
                        truncatedLabel,
                        KnowledgeItemType.Learning,
                        vscode.TreeItemCollapsibleState.None,
                        item,
                        group,
                    );
            }
        });
    }

    private getScopeGroupChildren(group: KnowledgeTreeItem): KnowledgeTreeItem[] {
        const items = group.data?.items ?? [];

        return items.map((item: any) => new KnowledgeTreeItem(
            this.truncateLabel(item.content, 50),
            KnowledgeItemType.Learning,
            vscode.TreeItemCollapsibleState.None,
            item,
            group,
        ));
    }

    private truncateLabel(text: string, maxLength: number): string {
        if (!text) return '';
        if (text.length <= maxLength) return text;
        return text.substring(0, maxLength - 3) + '...';
    }

    private groupByStatus(items: any[], statuses: string[]): Record<string, any[]> {
        const grouped: Record<string, any[]> = {};
        for (const status of statuses) {
            grouped[status] = [];
        }

        for (const item of items) {
            const status = item.status || statuses[0];
            if (grouped[status]) {
                grouped[status].push(item);
            }
        }

        return grouped;
    }
}
