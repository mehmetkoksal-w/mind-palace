import * as cp from 'child_process';
import * as vscode from 'vscode';
import * as util from 'util';
import * as readline from 'readline';

const exec = util.promisify(cp.exec);

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
     * Calls the `search_mind_palace` tool via MCP
     */
    async search(query: string): Promise<SearchResults> {
        this.channel.appendLine(`[Search] Query: "${query}"`);

        try {
            // MCP tool call format
            const result = await this.mcpClient.request<any>('tools/call', {
                name: 'search_mind_palace',
                arguments: { query },
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
     * Runs `palace verify --fast`.
     * Returns true if synced, false if stale.
     * Throws error if binary is missing or fails unexpectedly.
     */
    async runVerify(): Promise<boolean> {
        const bin = this.getBinaryPath();
        try {
            await exec(`${bin} verify --fast`, { cwd: vscode.workspace.workspaceFolders?.[0]?.uri.fsPath });
            return true;
        } catch (error: any) {
            if (error.code === 127 || error.message.includes('command not found')) {
                throw new Error('Palace binary not found');
            }
            // 'palace verify' returns non-zero exit code if verification fails (stale)
            return false;
        }
    }

    /**
     * Runs `palace scan && palace collect`.
     */
    async runHeal(): Promise<void> {
        const bin = this.getBinaryPath();
        try {
            this.channel.appendLine(`Running ${bin} scan && ${bin} collect...`);
            const { stdout, stderr } = await exec(`${bin} scan && ${bin} collect`, {
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

    /**
     * Dispose resources
     */
    dispose(): void {
        this.mcpClient.disconnect();
        this._onConnectionChange.dispose();
        this.channel.dispose();
    }
}
