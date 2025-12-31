import * as cp from 'child_process';
import * as vscode from 'vscode';
import * as util from 'util';
import * as readline from 'readline';

const exec = util.promisify(cp.exec);

/**
 * MCP Tool Names - aligned with CLI commands
 * See: apps/cli/internal/butler/mcp_tools_list.go
 */
export const MCP_TOOLS = {
    // Explore tools
    EXPLORE: 'explore',
    EXPLORE_ROOMS: 'explore_rooms',
    EXPLORE_CONTEXT: 'explore_context',
    EXPLORE_IMPACT: 'explore_impact',
    EXPLORE_SYMBOLS: 'explore_symbols',
    EXPLORE_SYMBOL: 'explore_symbol',
    EXPLORE_FILE: 'explore_file',
    EXPLORE_DEPS: 'explore_deps',
    EXPLORE_CALLERS: 'explore_callers',
    EXPLORE_CALLEES: 'explore_callees',
    EXPLORE_GRAPH: 'explore_graph',
    // Store tools
    STORE: 'store',
    // Recall tools
    RECALL: 'recall',
    RECALL_DECISIONS: 'recall_decisions',
    RECALL_IDEAS: 'recall_ideas',
    RECALL_OUTCOME: 'recall_outcome',
    RECALL_LINK: 'recall_link',
    RECALL_LINKS: 'recall_links',
    RECALL_UNLINK: 'recall_unlink',
    // Brief tools
    BRIEF: 'brief',
    BRIEF_FILE: 'brief_file',
    // Session tools
    SESSION_START: 'session_start',
    SESSION_LOG: 'session_log',
    SESSION_END: 'session_end',
    SESSION_CONFLICT: 'session_conflict',
    SESSION_LIST: 'session_list',
    // Conversation tools
    CONVERSATION_STORE: 'conversation_store',
    CONVERSATION_SEARCH: 'conversation_search',
    // Corridor tools
    CORRIDOR_LEARNINGS: 'corridor_learnings',
    CORRIDOR_LINKS: 'corridor_links',
    CORRIDOR_STATS: 'corridor_stats',
    CORRIDOR_PROMOTE: 'corridor_promote',
    CORRIDOR_REINFORCE: 'corridor_reinforce',
    // Semantic search tools
    SEARCH_SEMANTIC: 'search_semantic',
    SEARCH_HYBRID: 'search_hybrid',
    SEARCH_SIMILAR: 'search_similar',
} as const;

/**
 * Search result types from the MCP server
 */
export interface SearchMatch {
    filePath: string;
    snippet: string;
    lineNumber?: number;
    score?: number;
}

export interface RoomSearchResult {
    roomName: string;
    matches: SearchMatch[];
}

export interface SearchResults {
    query: string;
    results: RoomSearchResult[];
    totalMatches: number;
}

/**
 * MCP Tool Response Types
 */
export interface MCPToolResponse<T = any> {
    content?: Array<{ type: string; text: string }>;
    result?: T;
}

export interface ContradictionDetail {
    conflictingId: string;
    conflictingKind: string;
    conflictingContent: string;
    confidence: number;
    type: string;
    explanation: string;
    autoLinked: boolean;
}

export interface StoreResult {
    id: string;
    kind: 'idea' | 'decision' | 'learning';
    confidence: number;
    signals: string[];
    contradictions?: ContradictionDetail[];
}

export interface BriefResult {
    agents: Array<{ agentType: string; sessionId: string; goal: string }>;
    learnings: Array<{ id: string; content: string; confidence: number }>;
    hotspots: Array<{ path: string; editCount: number }>;
    conflict?: { path: string; agentType: string };
}

export interface FileIntelResult {
    path: string;
    editCount: number;
    failureCount: number;
    lastEdited?: string;
    learnings: Array<{ id: string; content: string; confidence: number }>;
}

export interface SessionStartResult {
    sessionId: string;
}

export interface SessionInfo {
    id: string;
    agentType: string;
    agentId?: string;
    goal?: string;
    startedAt: string;
    lastActivity?: string;
    state: 'active' | 'completed' | 'abandoned';
    summary?: string;
}

export interface SessionListResult {
    sessions: SessionInfo[];
}

export interface RecallResult {
    learnings?: Array<{ id: string; content: string; confidence: number; scope: string }>;
    decisions?: Array<{ id: string; content: string; status: string; scope: string }>;
    ideas?: Array<{ id: string; content: string; status: string; scope: string }>;
}

export interface ExploreContextResult {
    files: Array<{ path: string; symbols: string[] }>;
    learnings: Array<{ id: string; content: string }>;
    decisions: Array<{ id: string; content: string }>;
    ideas: Array<{ id: string; content: string }>;
}

export interface CallGraphResult {
    symbol: string;
    callers: Array<{ symbol: string; file: string; line: number }>;
    callees: Array<{ symbol: string; file: string; line: number }>;
}

export interface CorridorLearning {
    id: string;
    originWorkspace: string;
    content: string;
    confidence: number;
    source: string;
    createdAt: string;
    lastUsed: string;
    useCount: number;
    tags: string[];
}

export interface LinkedWorkspace {
    name: string;
    path: string;
    addedAt: string;
    lastAccessed?: string;
}

export interface CorridorStats {
    learningCount: number;
    linkedWorkspaces: number;
    averageConfidence: number;
}

/**
 * Semantic Search Types
 */
export interface SemanticSearchResult {
    id: string;
    kind: 'idea' | 'decision' | 'learning';
    content: string;
    similarity: number;
    createdAt: string;
}

export interface HybridSearchResult extends SemanticSearchResult {
    matchType: 'keyword' | 'semantic' | 'both';
    ftsScore?: number;
}

export interface ConversationMessage {
    role: 'user' | 'assistant' | 'system';
    content: string;
}

export interface Conversation {
    id: string;
    agentType: string;
    summary: string;
    messages: ConversationMessage[];
    sessionId?: string;
    createdAt: string;
}

export interface RecordLink {
    id: string;
    sourceId: string;
    sourceKind: string;
    targetId: string;
    targetKind: string;
    relation: 'supports' | 'contradicts' | 'implements' | 'supersedes' | 'inspired_by' | 'related';
    createdAt: string;
}

/**
 * JSON-RPC message types
 */
interface JsonRpcRequest {
    jsonrpc: '2.0';
    id: number;
    method: string;
    params?: any;
}

interface JsonRpcResponse {
    jsonrpc: '2.0';
    id: number;
    result?: any;
    error?: {
        code: number;
        message: string;
        data?: any;
    };
}

/**
 * MCP Client manages a persistent connection to `palace serve`
 */
class MCPClient {
    private process: cp.ChildProcessWithoutNullStreams | null = null;
    private requestId = 0;
    private pendingRequests = new Map<number, {
        resolve: (value: any) => void;
        reject: (error: Error) => void;
        timeout: NodeJS.Timeout;
    }>();
    private channel: vscode.OutputChannel;
    private isConnected = false;
    private restartAttempts = 0;
    private maxRestartAttempts = 3;
    private restartDelay = 1000;
    private onConnectionChange?: (connected: boolean) => void;
    private lineBuffer = '';

    constructor(channel: vscode.OutputChannel, onConnectionChange?: (connected: boolean) => void) {
        this.channel = channel;
        this.onConnectionChange = onConnectionChange;
    }

    private getBinaryPath(): string {
        const config = vscode.workspace.getConfiguration('mindPalace');
        return config.get<string>('binaryPath') || 'palace';
    }

    /**
     * Start the MCP server process
     */
    async connect(): Promise<void> {
        if (this.isConnected && this.process) {
            return;
        }

        const bin = this.getBinaryPath();
        const cwd = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;

        if (!cwd) {
            throw new Error('No workspace folder open');
        }

        this.channel.appendLine(`[MCP] Starting palace serve...`);
        this.channel.appendLine(`[MCP] Binary: ${bin}, CWD: ${cwd}`);

        try {
            this.process = cp.spawn(bin, ['serve'], {
                cwd,
                stdio: ['pipe', 'pipe', 'pipe'],
                shell: false,
            });

            this.process.on('error', (err) => {
                this.channel.appendLine(`[MCP] Process error: ${err.message}`);
                this.handleDisconnect();
            });

            this.process.on('exit', (code, signal) => {
                this.channel.appendLine(`[MCP] Process exited with code ${code}, signal ${signal}`);
                this.handleDisconnect();
            });

            // Handle stderr for debugging
            this.process.stderr?.on('data', (data) => {
                this.channel.appendLine(`[MCP STDERR] ${data.toString().trim()}`);
            });

            // Handle stdout line by line for JSON-RPC responses
            this.process.stdout?.on('data', (data) => {
                this.lineBuffer += data.toString();
                this.processLineBuffer();
            });

            // Wait for the server to be ready (initialization)
            await this.waitForReady();

            this.isConnected = true;
            this.restartAttempts = 0;
            this.onConnectionChange?.(true);
            this.channel.appendLine(`[MCP] Connected successfully`);
        } catch (error: any) {
            this.channel.appendLine(`[MCP] Failed to start: ${error.message}`);
            throw error;
        }
    }

    private processLineBuffer(): void {
        const lines = this.lineBuffer.split('\n');
        // Keep the last incomplete line in the buffer
        this.lineBuffer = lines.pop() || '';

        for (const line of lines) {
            const trimmed = line.trim();
            if (!trimmed) continue;

            try {
                const response = JSON.parse(trimmed) as JsonRpcResponse;
                this.handleResponse(response);
            } catch (e) {
                // Not JSON, might be log output
                this.channel.appendLine(`[MCP RAW] ${trimmed}`);
            }
        }
    }

    private handleResponse(response: JsonRpcResponse): void {
        const pending = this.pendingRequests.get(response.id);
        if (!pending) {
            this.channel.appendLine(`[MCP] Received response for unknown request ID: ${response.id}`);
            return;
        }

        clearTimeout(pending.timeout);
        this.pendingRequests.delete(response.id);

        if (response.error) {
            pending.reject(new Error(response.error.message));
        } else {
            pending.resolve(response.result);
        }
    }

    private async waitForReady(): Promise<void> {
        // Send initialize request per MCP protocol
        return new Promise((resolve, reject) => {
            const timeout = setTimeout(() => {
                reject(new Error('MCP server initialization timeout'));
            }, 10000);

            // For now, just wait a bit for the server to start
            // In a full implementation, we'd do the MCP initialize handshake
            setTimeout(() => {
                clearTimeout(timeout);
                resolve();
            }, 500);
        });
    }

    private handleDisconnect(): void {
        this.isConnected = false;
        this.process = null;
        this.onConnectionChange?.(false);

        // Reject all pending requests
        for (const [id, pending] of this.pendingRequests) {
            clearTimeout(pending.timeout);
            pending.reject(new Error('MCP connection lost'));
        }
        this.pendingRequests.clear();

        // Attempt restart
        if (this.restartAttempts < this.maxRestartAttempts) {
            this.restartAttempts++;
            this.channel.appendLine(`[MCP] Attempting restart (${this.restartAttempts}/${this.maxRestartAttempts})...`);
            setTimeout(() => {
                this.connect().catch((err) => {
                    this.channel.appendLine(`[MCP] Restart failed: ${err.message}`);
                });
            }, this.restartDelay * this.restartAttempts);
        }
    }

    /**
     * Send a JSON-RPC request to the MCP server
     */
    async request<T>(method: string, params?: any): Promise<T> {
        if (!this.isConnected || !this.process) {
            await this.connect();
        }

        const id = ++this.requestId;
        const request: JsonRpcRequest = {
            jsonrpc: '2.0',
            id,
            method,
            params,
        };

        return new Promise((resolve, reject) => {
            const timeout = setTimeout(() => {
                this.pendingRequests.delete(id);
                reject(new Error(`Request timeout for method: ${method}`));
            }, 30000);

            this.pendingRequests.set(id, { resolve, reject, timeout });

            const requestStr = JSON.stringify(request) + '\n';
            this.channel.appendLine(`[MCP TX] ${requestStr.trim()}`);

            this.process?.stdin?.write(requestStr, (err) => {
                if (err) {
                    clearTimeout(timeout);
                    this.pendingRequests.delete(id);
                    reject(err);
                }
            });
        });
    }

    /**
     * Disconnect from the MCP server
     */
    disconnect(): void {
        if (this.process) {
            this.channel.appendLine('[MCP] Disconnecting...');
            this.process.kill();
            this.process = null;
        }
        this.isConnected = false;
        this.onConnectionChange?.(false);
    }

    get connected(): boolean {
        return this.isConnected;
    }
}

export class PalaceBridge {
    private channel: vscode.OutputChannel;
    private mcpClient: MCPClient;
    private _onConnectionChange = new vscode.EventEmitter<boolean>();
    public readonly onConnectionChange = this._onConnectionChange.event;

    constructor() {
        this.channel = vscode.window.createOutputChannel("Mind Palace");
        this.mcpClient = new MCPClient(this.channel, (connected) => {
            this._onConnectionChange.fire(connected);
        });
    }

    private getBinaryPath(): string {
        const config = vscode.workspace.getConfiguration('mindPalace');
        return config.get<string>('binaryPath') || 'palace';
    }

    /**
     * Connect to the MCP server (palace serve)
     */
    async connectMCP(): Promise<void> {
        await this.mcpClient.connect();
    }

    /**
     * Disconnect from the MCP server
     */
    disconnectMCP(): void {
        this.mcpClient.disconnect();
    }

    /**
     * Check if connected to MCP server
     */
    get isMCPConnected(): boolean {
        return this.mcpClient.connected;
    }

    /**
     * Search the Mind Palace using Butler
     * Calls the `explore` tool via MCP
     */
    async search(query: string, options?: { limit?: number; room?: string; fuzzy?: boolean }): Promise<SearchResults> {
        this.channel.appendLine(`[Search] Query: "${query}"`);

        try {
            // MCP tool call format - use 'explore' tool (was 'search_mind_palace')
            const result = await this.mcpClient.request<any>('tools/call', {
                name: MCP_TOOLS.EXPLORE,
                arguments: {
                    query,
                    limit: options?.limit ?? 10,
                    room: options?.room,
                    fuzzy: options?.fuzzy ?? false,
                },
            });

            this.channel.appendLine(`[Search] Raw result: ${JSON.stringify(result)}`);

            // Parse the result into our expected format
            return this.parseSearchResults(query, result);
        } catch (error: any) {
            this.channel.appendLine(`[Search] Error: ${error.message}`);
            throw error;
        }
    }

    private parseSearchResults(query: string, rawResult: any): SearchResults {
        // Handle different possible response structures
        const results: RoomSearchResult[] = [];
        let totalMatches = 0;

        // Expected structure: { content: [{ type: "text", text: "..." }] }
        // or direct object structure
        let data = rawResult;

        if (rawResult?.content) {
            // MCP tool response format
            const textContent = rawResult.content.find((c: any) => c.type === 'text');
            if (textContent?.text) {
                try {
                    data = JSON.parse(textContent.text);
                } catch {
                    data = { rooms: [] };
                }
            }
        }

        // Parse room-grouped results
        if (data?.rooms || Array.isArray(data)) {
            const rooms = data.rooms || data;
            for (const room of rooms) {
                const matches: SearchMatch[] = [];

                if (room.matches || room.files) {
                    for (const match of (room.matches || room.files)) {
                        matches.push({
                            filePath: match.filePath || match.path || match.file,
                            snippet: match.snippet || match.content || '',
                            lineNumber: match.lineNumber || match.line,
                            score: match.score,
                        });
                        totalMatches++;
                    }
                }

                if (matches.length > 0) {
                    results.push({
                        roomName: room.name || room.roomName || 'Unknown',
                        matches,
                    });
                }
            }
        }

        return {
            query,
            results,
            totalMatches,
        };
    }

    /**
     * Runs `palace check` to verify index freshness.
     * Returns true if synced, false if stale.
     * Throws error if binary is missing or fails unexpectedly.
     */
    async runVerify(): Promise<boolean> {
        const bin = this.getBinaryPath();
        try {
            await exec(`${bin} check`, { cwd: vscode.workspace.workspaceFolders?.[0]?.uri.fsPath });
            return true;
        } catch (error: any) {
            if (error.code === 127 || error.message.includes('command not found')) {
                throw new Error('Palace binary not found');
            }
            // 'palace check' returns non-zero exit code if verification fails (stale)
            return false;
        }
    }

    /**
     * Runs `palace scan && palace check --collect` to heal the index and generate context pack.
     */
    async runHeal(): Promise<void> {
        const bin = this.getBinaryPath();
        try {
            this.channel.appendLine(`Running ${bin} scan && ${bin} check --collect...`);
            const { stdout, stderr } = await exec(`${bin} scan && ${bin} check --collect`, {
                cwd: vscode.workspace.workspaceFolders?.[0]?.uri.fsPath
            });
            if (stdout) this.channel.append(stdout);
            if (stderr) this.channel.append(stderr);
            this.channel.appendLine("Heal complete.");
        } catch (error: any) {
            if (error.code === 127 || error.message.includes('command not found')) {
                throw new Error('Palace binary not found');
            }
            this.channel.appendLine(`Error running heal: ${error.message}`);
            throw error;
        }
    }

    // ========================================================================
    // MCP Tool Methods - Expose all MCP tools as typed methods
    // ========================================================================

    /**
     * Store a thought with auto-classification (idea, decision, or learning)
     */
    async store(content: string, options?: {
        as?: 'idea' | 'decision' | 'learning';
        scope?: 'palace' | 'room' | 'file';
        scopePath?: string;
        tags?: string[];
        confidence?: number;
    }): Promise<StoreResult> {
        const result = await this.callTool<StoreResult>(MCP_TOOLS.STORE, {
            content,
            as: options?.as,
            scope: options?.scope ?? 'palace',
            scopePath: options?.scopePath,
            tags: options?.tags,
            confidence: options?.confidence ?? 0.5,
        });
        return result;
    }

    /**
     * Get learnings, optionally filtered by scope or query
     */
    async recallLearnings(options?: {
        query?: string;
        scope?: 'palace' | 'room' | 'file';
        scopePath?: string;
        limit?: number;
    }): Promise<RecallResult> {
        return this.callTool<RecallResult>(MCP_TOOLS.RECALL, options);
    }

    /**
     * Get decisions, optionally filtered by status, scope, or query
     */
    async recallDecisions(options?: {
        query?: string;
        status?: 'active' | 'superseded' | 'reversed';
        scope?: 'palace' | 'room' | 'file';
        scopePath?: string;
        limit?: number;
    }): Promise<RecallResult> {
        return this.callTool<RecallResult>(MCP_TOOLS.RECALL_DECISIONS, options);
    }

    /**
     * Get ideas, optionally filtered by status, scope, or query
     */
    async recallIdeas(options?: {
        query?: string;
        status?: 'active' | 'exploring' | 'implemented' | 'dropped';
        scope?: 'palace' | 'room' | 'file';
        scopePath?: string;
        limit?: number;
    }): Promise<RecallResult> {
        return this.callTool<RecallResult>(MCP_TOOLS.RECALL_IDEAS, options);
    }

    /**
     * Record the outcome of a decision
     */
    async recordOutcome(decisionId: string, outcome: 'success' | 'failed' | 'mixed', note?: string): Promise<void> {
        await this.callTool(MCP_TOOLS.RECALL_OUTCOME, { decisionId, outcome, note });
    }

    /**
     * Get a workspace or file briefing
     */
    async getBrief(file?: string): Promise<BriefResult> {
        return this.callTool<BriefResult>(MCP_TOOLS.BRIEF, { file });
    }

    /**
     * Get file intelligence (edit history, failure rate, learnings)
     */
    async getFileIntel(path: string): Promise<FileIntelResult> {
        return this.callTool<FileIntelResult>(MCP_TOOLS.BRIEF_FILE, { path });
    }

    /**
     * Get complete context for a task
     */
    async getContext(task: string, options?: {
        limit?: number;
        maxTokens?: number;
        includeTests?: boolean;
        includeLearnings?: boolean;
        includeIdeas?: boolean;
        includeDecisions?: boolean;
    }): Promise<ExploreContextResult> {
        return this.callTool<ExploreContextResult>(MCP_TOOLS.EXPLORE_CONTEXT, {
            task,
            ...options,
        });
    }

    /**
     * Find all callers of a function/method
     */
    async getCallers(symbol: string): Promise<CallGraphResult> {
        return this.callTool<CallGraphResult>(MCP_TOOLS.EXPLORE_CALLERS, { symbol });
    }

    /**
     * Find all functions/methods called by a symbol
     */
    async getCallees(symbol: string, file: string): Promise<CallGraphResult> {
        return this.callTool<CallGraphResult>(MCP_TOOLS.EXPLORE_CALLEES, { symbol, file });
    }

    /**
     * Get complete call graph for a file
     */
    async getCallGraph(file: string): Promise<CallGraphResult> {
        return this.callTool<CallGraphResult>(MCP_TOOLS.EXPLORE_GRAPH, { file });
    }

    /**
     * List all rooms in the Mind Palace
     */
    async listRooms(): Promise<any> {
        return this.callTool(MCP_TOOLS.EXPLORE_ROOMS, {});
    }

    /**
     * Start a new agent session
     */
    async startSession(agentType: string, goal?: string, agentId?: string): Promise<SessionStartResult> {
        return this.callTool<SessionStartResult>(MCP_TOOLS.SESSION_START, {
            agentType,
            goal,
            agentId,
        });
    }

    /**
     * End an agent session
     */
    async endSession(sessionId: string, outcome?: 'success' | 'failure' | 'partial', summary?: string): Promise<void> {
        await this.callTool(MCP_TOOLS.SESSION_END, { sessionId, outcome, summary });
    }

    /**
     * Log an activity within a session
     */
    async logActivity(sessionId: string, kind: 'file_read' | 'file_edit' | 'search' | 'command', target: string, outcome?: 'success' | 'failure' | 'unknown'): Promise<void> {
        await this.callTool(MCP_TOOLS.SESSION_LOG, { sessionId, kind, target, outcome });
    }

    /**
     * Check if another agent is working on a file
     */
    async checkConflict(path: string, sessionId?: string): Promise<{ conflict: boolean; agent?: string }> {
        return this.callTool(MCP_TOOLS.SESSION_CONFLICT, { path, sessionId });
    }

    /**
     * List all sessions
     */
    async listSessions(options?: { active?: boolean; limit?: number }): Promise<SessionInfo[]> {
        const result = await this.callTool<SessionListResult>(MCP_TOOLS.SESSION_LIST, {
            active: options?.active ?? false,
            limit: options?.limit ?? 20,
        });
        return result.sessions || [];
    }

    // ========================================================================
    // Corridor Methods - Personal cross-workspace learnings
    // ========================================================================

    /**
     * Get personal learnings from the global corridor
     */
    async getCorridorLearnings(options?: { query?: string; limit?: number }): Promise<CorridorLearning[]> {
        const result = await this.callTool<{ learnings: CorridorLearning[]; count: number }>(
            MCP_TOOLS.CORRIDOR_LEARNINGS,
            options || {}
        );
        return result.learnings || [];
    }

    /**
     * Get linked workspaces
     */
    async getCorridorLinks(): Promise<LinkedWorkspace[]> {
        const result = await this.callTool<{ links: LinkedWorkspace[]; count: number }>(
            MCP_TOOLS.CORRIDOR_LINKS,
            {}
        );
        return result.links || [];
    }

    /**
     * Get corridor statistics
     */
    async getCorridorStats(): Promise<CorridorStats> {
        return this.callTool<CorridorStats>(MCP_TOOLS.CORRIDOR_STATS, {});
    }

    /**
     * Promote a learning to the personal corridor
     */
    async promoteToCorrdior(learningId: string): Promise<void> {
        await this.callTool(MCP_TOOLS.CORRIDOR_PROMOTE, { learningId });
    }

    /**
     * Reinforce a corridor learning (increase confidence)
     */
    async reinforceCorridorLearning(learningId: string): Promise<void> {
        await this.callTool(MCP_TOOLS.CORRIDOR_REINFORCE, { learningId });
    }

    // ========================================================================
    // Semantic Search Methods
    // ========================================================================

    /**
     * Perform semantic search using AI embeddings
     */
    async semanticSearch(query: string, options?: {
        kinds?: string[];
        limit?: number;
        minSimilarity?: number;
        scope?: string;
        scopePath?: string;
    }): Promise<SemanticSearchResult[]> {
        const result = await this.callTool<SemanticSearchResult[]>(
            MCP_TOOLS.SEARCH_SEMANTIC,
            { query, ...options }
        );
        return result || [];
    }

    /**
     * Perform hybrid (keyword + semantic) search
     */
    async hybridSearch(query: string, options?: {
        kinds?: string[];
        limit?: number;
    }): Promise<HybridSearchResult[]> {
        const result = await this.callTool<HybridSearchResult[]>(
            MCP_TOOLS.SEARCH_HYBRID,
            { query, ...options }
        );
        return result || [];
    }

    /**
     * Find records similar to a given record
     */
    async findSimilar(recordId: string, options?: {
        limit?: number;
        minSimilarity?: number;
    }): Promise<SemanticSearchResult[]> {
        const result = await this.callTool<SemanticSearchResult[]>(
            MCP_TOOLS.SEARCH_SIMILAR,
            { recordId, ...options }
        );
        return result || [];
    }

    // ========================================================================
    // Conversation Methods - Store and search conversations
    // ========================================================================

    /**
     * Store a conversation for future reference
     */
    async storeConversation(
        summary: string,
        messages: ConversationMessage[],
        options?: { agentType?: string; sessionId?: string }
    ): Promise<string> {
        const result = await this.callTool<{ id: string }>(MCP_TOOLS.CONVERSATION_STORE, {
            summary,
            messages,
            agentType: options?.agentType ?? 'claude-code',
            sessionId: options?.sessionId,
        });
        return result.id;
    }

    /**
     * Search past conversations
     */
    async searchConversations(options?: {
        query?: string;
        sessionId?: string;
        limit?: number;
    }): Promise<Conversation[]> {
        const result = await this.callTool<{ conversations: Conversation[] }>(
            MCP_TOOLS.CONVERSATION_SEARCH,
            options || {}
        );
        return result.conversations || [];
    }

    // ========================================================================
    // Link Methods - Manage relationships between records
    // ========================================================================

    /**
     * Create a link between two records
     */
    async createLink(
        sourceId: string,
        targetId: string,
        relation: RecordLink['relation']
    ): Promise<string> {
        const result = await this.callTool<{ id: string }>(MCP_TOOLS.RECALL_LINK, {
            sourceId,
            targetId,
            relation,
        });
        return result.id;
    }

    /**
     * Get links for a record
     */
    async getLinks(recordId: string, direction?: 'from' | 'to' | 'all'): Promise<RecordLink[]> {
        const result = await this.callTool<{ links: RecordLink[] }>(MCP_TOOLS.RECALL_LINKS, {
            recordId,
            direction: direction || 'all',
        });
        return result.links || [];
    }

    /**
     * Remove a link
     */
    async removeLink(linkId: string): Promise<void> {
        await this.callTool(MCP_TOOLS.RECALL_UNLINK, { linkId });
    }

    /**
     * Generic MCP tool call helper
     */
    private async callTool<T>(toolName: string, args: any): Promise<T> {
        this.channel.appendLine(`[MCP] Calling tool: ${toolName}`);

        try {
            const result = await this.mcpClient.request<any>('tools/call', {
                name: toolName,
                arguments: args,
            });

            // Parse MCP response format
            if (result?.content) {
                const textContent = result.content.find((c: any) => c.type === 'text');
                if (textContent?.text) {
                    try {
                        return JSON.parse(textContent.text) as T;
                    } catch {
                        return textContent.text as T;
                    }
                }
            }

            return result as T;
        } catch (error: any) {
            this.channel.appendLine(`[MCP] Tool error (${toolName}): ${error.message}`);
            throw error;
        }
    }

    /**
     * Dispose resources
     */
    dispose(): void {
        this.mcpClient.disconnect();
        this._onConnectionChange.dispose();
        this.channel.dispose();
    }
}
