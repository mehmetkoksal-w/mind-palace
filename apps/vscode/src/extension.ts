import * as path from 'path';
import * as vscode from 'vscode';
import { PalaceBridge, CorridorLearning, Conversation, RecordLink } from './bridge';
import { registerStoreCommands } from './commands/store';
import { getConfig, watchProjectConfig } from './config';
import { PalaceDecorator } from './decorator';
import { PalaceHUD } from './hud';
import { CallGraphHoverProvider } from './providers/callGraphHoverProvider';
import { ConflictDetectionProvider } from './providers/conflictDetectionProvider';
import { CorridorTreeProvider } from './providers/corridorTreeProvider';
import { FileIntelligenceProvider } from './providers/fileIntelligenceProvider';
import { InlineLearningDecorator } from './providers/inlineLearningDecorator';
import { KnowledgeTreeProvider } from './providers/knowledgeTreeProvider';
import { LearningSuggestionProvider, LearningSuggestion } from './providers/learningSuggestionProvider';
import { PalaceCodeLensProvider } from './providers/palaceCodeLensProvider';
import { SessionTreeProvider } from './providers/sessionTreeProvider';
import { PalaceSidebarProvider } from './sidebar';
import { warnIfIncompatible } from './version';
import { KnowledgeGraphPanel } from './webviews/knowledgeGraph/knowledgeGraphPanel';

export function activate(context: vscode.ExtensionContext) {
    warnIfIncompatible();

    const bridge = new PalaceBridge();
    const hud = new PalaceHUD();
    const decorator = new PalaceDecorator();

    const configWatcher = watchProjectConfig(() => checkStatus());
    context.subscriptions.push(configWatcher);

    const sidebarProvider = new PalaceSidebarProvider(context.extensionUri);
    sidebarProvider.setBridge(bridge);

    context.subscriptions.push(
        vscode.window.registerWebviewViewProvider(PalaceSidebarProvider.viewType, sidebarProvider)
    );

    // Knowledge tree view
    const knowledgeProvider = new KnowledgeTreeProvider();
    knowledgeProvider.setBridge(bridge);

    context.subscriptions.push(
        vscode.window.registerTreeDataProvider('mindPalace.knowledgeView', knowledgeProvider)
    );

    // Sessions tree view
    const sessionProvider = new SessionTreeProvider();
    sessionProvider.setBridge(bridge);

    context.subscriptions.push(
        vscode.window.registerTreeDataProvider('mindPalace.sessionsView', sessionProvider)
    );

    // Corridor tree view (personal cross-workspace learnings)
    const corridorProvider = new CorridorTreeProvider();
    corridorProvider.setBridge(bridge);

    context.subscriptions.push(
        vscode.window.registerTreeDataProvider('mindPalace.corridorView', corridorProvider)
    );

    // Store commands (storeIdea, storeDecision, storeLearning, quickStore)
    registerStoreCommands(context, bridge);

    // File Intelligence Provider (gutter decorations + status bar)
    const fileIntelProvider = new FileIntelligenceProvider();
    fileIntelProvider.setBridge(bridge);
    context.subscriptions.push(fileIntelProvider);

    // CodeLens Provider (shows learning/decision counts at file top)
    const codeLensProvider = new PalaceCodeLensProvider();
    codeLensProvider.setBridge(bridge);
    context.subscriptions.push(
        vscode.languages.registerCodeLensProvider({ scheme: 'file' }, codeLensProvider)
    );

    // Call Graph Hover Provider (shows callers/callees on hover)
    const callGraphProvider = new CallGraphHoverProvider();
    callGraphProvider.setBridge(bridge);
    context.subscriptions.push(
        vscode.languages.registerHoverProvider({ scheme: 'file' }, callGraphProvider)
    );

    // Conflict Detection Provider (warns when another agent is working on the same file)
    const conflictProvider = new ConflictDetectionProvider();
    conflictProvider.setBridge(bridge);
    context.subscriptions.push(conflictProvider);

    // Learning Suggestion Provider (contextual learning suggestions via semantic search)
    const learningSuggestionProvider = new LearningSuggestionProvider();
    learningSuggestionProvider.setBridge(bridge);
    context.subscriptions.push(
        vscode.languages.registerCodeLensProvider({ scheme: 'file' }, learningSuggestionProvider)
    );

    // Inline Learning Decorator (shows learnings as inline decorations)
    const inlineLearningDecorator = new InlineLearningDecorator();
    inlineLearningDecorator.setBridge(bridge);
    inlineLearningDecorator.activate(context);
    context.subscriptions.push(inlineLearningDecorator);

    context.subscriptions.push({
        dispose: () => {
            bridge.dispose();
            hud.dispose();
        }
    });

    let debounceTimer: NodeJS.Timeout | undefined;
    let countdownInterval: NodeJS.Timeout | undefined;

    checkStatus();

    const disposableHeal = vscode.commands.registerCommand('mindPalace.heal', () => performHeal(false));
    const disposableCheckStatus = vscode.commands.registerCommand('mindPalace.checkStatus', () => checkStatus());
    const disposableOpenBlueprint = vscode.commands.registerCommand('mindPalace.openBlueprint', () => {
        vscode.commands.executeCommand('mindPalace.blueprintView.focus');
    });

    // Knowledge panel commands
    const disposableRefreshKnowledge = vscode.commands.registerCommand('mindPalace.refreshKnowledge', () => {
        knowledgeProvider.refresh();
    });

    // Session panel commands
    const disposableRefreshSessions = vscode.commands.registerCommand('mindPalace.refreshSessions', () => {
        sessionProvider.refresh();
    });

    const disposableStartSession = vscode.commands.registerCommand('mindPalace.startSession', async () => {
        const agentType = await vscode.window.showInputBox({
            prompt: 'Enter agent type',
            placeHolder: 'claude-code, cursor, aider, etc.',
            value: 'claude-code',
        });

        if (!agentType) return;

        const goal = await vscode.window.showInputBox({
            prompt: 'What is the goal of this session? (optional)',
            placeHolder: 'e.g., Fix authentication bug',
        });

        try {
            const result = await bridge.startSession(agentType, goal);
            vscode.window.showInformationMessage(`Session started: ${result.sessionId}`);
            sessionProvider.refresh();
        } catch (err: any) {
            vscode.window.showErrorMessage(`Failed to start session: ${err.message}`);
        }
    });

    const disposableEndSession = vscode.commands.registerCommand('mindPalace.endSession', async (sessionInfo?: any) => {
        let sessionId = sessionInfo?.id;

        if (!sessionId) {
            sessionId = await vscode.window.showInputBox({
                prompt: 'Enter session ID to end',
                placeHolder: 'ses_...',
            });
        }

        if (!sessionId) return;

        const outcome = await vscode.window.showQuickPick(
            ['success', 'failure', 'partial'],
            { placeHolder: 'Select outcome' }
        ) as 'success' | 'failure' | 'partial' | undefined;

        const summary = await vscode.window.showInputBox({
            prompt: 'Brief summary of what was accomplished (optional)',
        });

        try {
            await bridge.endSession(sessionId, outcome, summary);
            vscode.window.showInformationMessage(`Session ended: ${sessionId}`);
            sessionProvider.refresh();
        } catch (err: any) {
            vscode.window.showErrorMessage(`Failed to end session: ${err.message}`);
        }
    });

    const disposableShowSessionDetail = vscode.commands.registerCommand('mindPalace.showSessionDetail', async (session: any) => {
        if (!session) return;

        const panel = vscode.window.createWebviewPanel(
            'mindPalaceSessionDetail',
            `Session: ${session.id.substring(0, 12)}...`,
            vscode.ViewColumn.One,
            { enableScripts: false }
        );

        const stateIcon = session.state === 'active' ? '🟢' : session.state === 'completed' ? '✅' : '❌';
        const goalInfo = session.goal ? `<p><strong>Goal:</strong> ${session.goal}</p>` : '';
        const summaryInfo = session.summary ? `<p><strong>Summary:</strong> ${session.summary}</p>` : '';

        panel.webview.html = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Session Detail</title>
    <style>
        body {
            font-family: var(--vscode-font-family);
            padding: 20px;
            color: var(--vscode-foreground);
            background-color: var(--vscode-editor-background);
        }
        h1 {
            font-size: 1.5em;
            margin-bottom: 16px;
        }
        .meta {
            font-size: 0.9em;
            color: var(--vscode-descriptionForeground);
        }
        .meta p {
            margin: 8px 0;
        }
        .meta strong {
            color: var(--vscode-foreground);
        }
        .status {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 4px;
            background-color: var(--vscode-badge-background);
            color: var(--vscode-badge-foreground);
        }
    </style>
</head>
<body>
    <h1>${stateIcon} Session</h1>
    <div class="meta">
        <p><strong>ID:</strong> <code>${session.id}</code></p>
        <p><strong>Agent:</strong> ${session.agentType}</p>
        <p><strong>State:</strong> <span class="status">${session.state}</span></p>
        <p><strong>Started:</strong> ${new Date(session.startedAt).toLocaleString()}</p>
        ${goalInfo}
        ${summaryInfo}
    </div>
</body>
</html>`;
    });

    // Show Conflict Info command
    const disposableShowConflictInfo = vscode.commands.registerCommand('mindPalace.showConflictInfo', async () => {
        const editor = vscode.window.activeTextEditor;
        if (!editor) {
            vscode.window.showInformationMessage('No file open');
            return;
        }

        const filePath = editor.document.uri.fsPath;
        const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
        if (!workspaceRoot) return;

        const relativePath = filePath.startsWith(workspaceRoot)
            ? filePath.substring(workspaceRoot.length + 1)
            : filePath;

        try {
            const result = await bridge.checkConflict(relativePath);

            if (result.conflict) {
                const action = await vscode.window.showWarningMessage(
                    `${result.agent || 'Another agent'} is also working on this file. Consider coordinating changes.`,
                    'View Sessions',
                    'OK'
                );

                if (action === 'View Sessions') {
                    vscode.commands.executeCommand('mindPalace.sessionsView.focus');
                }
            } else {
                vscode.window.showInformationMessage('No conflicts detected on this file.');
            }
        } catch (err: any) {
            vscode.window.showErrorMessage(`Failed to check conflict: ${err.message}`);
        }
    });

    // File Intelligence command
    const disposableShowFileIntel = vscode.commands.registerCommand('mindPalace.showFileIntel', async (filePath?: string) => {
        const targetPath = filePath || vscode.window.activeTextEditor?.document.uri.fsPath;
        if (!targetPath) {
            vscode.window.showWarningMessage('No file selected');
            return;
        }

        const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
        if (!workspaceRoot) return;

        const relativePath = targetPath.startsWith(workspaceRoot)
            ? targetPath.substring(workspaceRoot.length + 1)
            : targetPath;

        try {
            const intel = await bridge.getFileIntel(relativePath);

            // Show in a quick pick or notification
            const items: vscode.QuickPickItem[] = [];

            items.push({
                label: `$(file) ${relativePath}`,
                description: 'File Intelligence',
                detail: `Edits: ${intel.editCount} | Failures: ${intel.failureCount}`,
            });

            if (intel.learnings && intel.learnings.length > 0) {
                items.push({ label: '', kind: vscode.QuickPickItemKind.Separator });
                items.push({
                    label: `$(lightbulb) Learnings (${intel.learnings.length})`,
                    description: '',
                });
                intel.learnings.forEach(l => {
                    items.push({
                        label: `    ${l.content.substring(0, 60)}${l.content.length > 60 ? '...' : ''}`,
                        description: `${Math.round((l.confidence ?? 0.5) * 100)}%`,
                    });
                });
            }

            if (intel.lastEdited) {
                items.push({ label: '', kind: vscode.QuickPickItemKind.Separator });
                items.push({
                    label: `$(calendar) Last edited: ${intel.lastEdited}`,
                    description: '',
                });
            }

            vscode.window.showQuickPick(items, {
                title: 'Mind Palace - File Intelligence',
                placeHolder: relativePath,
            });
        } catch (err: any) {
            vscode.window.showErrorMessage(`Failed to get file intel: ${err.message}`);
        }
    });

    // Show Call Graph command
    const disposableShowCallGraph = vscode.commands.registerCommand('mindPalace.showCallGraph', async (args?: { symbol: string; file: string }) => {
        if (!args) {
            vscode.window.showWarningMessage('No symbol selected');
            return;
        }

        const { symbol, file } = args;
        const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
        if (!workspaceRoot) return;

        try {
            const [callers, callees] = await Promise.all([
                bridge.getCallers(symbol).catch(() => ({ symbol, callers: [], callees: [] })),
                bridge.getCallees(symbol, file).catch(() => ({ symbol, callers: [], callees: [] })),
            ]);

            const items: vscode.QuickPickItem[] = [];

            // Callers section
            if (callers.callers && callers.callers.length > 0) {
                items.push({ label: 'Callers', kind: vscode.QuickPickItemKind.Separator });
                callers.callers.forEach((c: any) => {
                    items.push({
                        label: `$(arrow-left) ${c.symbol}`,
                        description: `${c.file}:${c.line}`,
                        detail: 'Calls this function',
                    });
                });
            }

            // Callees section
            if (callees.callees && callees.callees.length > 0) {
                items.push({ label: 'Callees', kind: vscode.QuickPickItemKind.Separator });
                callees.callees.forEach((c: any) => {
                    items.push({
                        label: `$(arrow-right) ${c.symbol}`,
                        description: `${c.file}:${c.line}`,
                        detail: 'Called by this function',
                    });
                });
            }

            if (items.length === 0) {
                vscode.window.showInformationMessage(`No call graph data found for ${symbol}`);
                return;
            }

            const selection = await vscode.window.showQuickPick(items, {
                title: `Call Graph: ${symbol}`,
                placeHolder: 'Select to navigate to definition',
            });

            if (selection && selection.description) {
                const [filePath, lineStr] = selection.description.split(':');
                const line = parseInt(lineStr, 10) - 1;
                const fullPath = filePath.startsWith('/') ? filePath : path.join(workspaceRoot, filePath);
                const uri = vscode.Uri.file(fullPath);
                const doc = await vscode.workspace.openTextDocument(uri);
                const editor = await vscode.window.showTextDocument(doc);
                const pos = new vscode.Position(line, 0);
                editor.selection = new vscode.Selection(pos, pos);
                editor.revealRange(new vscode.Range(pos, pos), vscode.TextEditorRevealType.InCenter);
            }
        } catch (err: any) {
            vscode.window.showErrorMessage(`Failed to get call graph: ${err.message}`);
        }
    });

    const disposableShowKnowledgeDetail = vscode.commands.registerCommand('mindPalace.showKnowledgeDetail', async (item: { type: string; data: any }) => {
        if (!item) return;

        const { type, data } = item;
        const panel = vscode.window.createWebviewPanel(
            'mindPalaceKnowledgeDetail',
            `${type.charAt(0).toUpperCase() + type.slice(1)}: ${data.content?.substring(0, 30) ?? 'Detail'}...`,
            vscode.ViewColumn.One,
            { enableScripts: false }
        );

        const scopeInfo = data.scopePath ? `<p><strong>Scope Path:</strong> ${data.scopePath}</p>` : '';
        const outcomeInfo = data.outcome ? `<p><strong>Outcome:</strong> ${data.outcome}${data.outcomeNote ? ` - ${data.outcomeNote}` : ''}</p>` : '';
        const confidenceInfo = type === 'learning' && data.confidence !== undefined
            ? `<p><strong>Confidence:</strong> ${Math.round(data.confidence * 100)}%</p>`
            : '';
        const statusInfo = data.status ? `<p><strong>Status:</strong> ${data.status}</p>` : '';
        const tagsInfo = data.tags?.length ? `<p><strong>Tags:</strong> ${data.tags.join(', ')}</p>` : '';

        panel.webview.html = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>${type} Detail</title>
    <style>
        body {
            font-family: var(--vscode-font-family);
            padding: 20px;
            color: var(--vscode-foreground);
            background-color: var(--vscode-editor-background);
        }
        h1 {
            font-size: 1.5em;
            margin-bottom: 16px;
            color: var(--vscode-foreground);
        }
        .content {
            font-size: 1.1em;
            line-height: 1.6;
            margin-bottom: 24px;
            padding: 16px;
            background-color: var(--vscode-textBlockQuote-background);
            border-left: 4px solid var(--vscode-textLink-activeForeground);
            border-radius: 4px;
        }
        .meta {
            font-size: 0.9em;
            color: var(--vscode-descriptionForeground);
        }
        .meta p {
            margin: 8px 0;
        }
        .meta strong {
            color: var(--vscode-foreground);
        }
    </style>
</head>
<body>
    <h1>${type.charAt(0).toUpperCase() + type.slice(1)}</h1>
    <div class="content">${data.content}</div>
    <div class="meta">
        <p><strong>ID:</strong> ${data.id}</p>
        <p><strong>Scope:</strong> ${data.scope || 'palace'}</p>
        ${scopeInfo}
        ${statusInfo}
        ${confidenceInfo}
        ${outcomeInfo}
        ${tagsInfo}
    </div>
</body>
</html>`;
    });

    // Command: Show Menu
    let disposableShowMenu = vscode.commands.registerCommand('mindPalace.showMenu', async () => {
        const items: vscode.QuickPickItem[] = [
            { label: '$(heart) Heal Context', description: 'Run palace scan && collect' },
            { label: '$(search) Search Palace', description: 'Focus the search input in Blueprint' },
            { label: '$(layout-sidebar-left) Focus Blueprint', description: 'Show the Blueprint Sidebar' },
            { label: '$(file-code) Open Context Pack', description: 'View the generated context-pack.json' },
            { label: '$(settings-gear) Settings', description: 'Configure Mind Palace extension' }
        ];

        const selection = await vscode.window.showQuickPick(items, {
            placeHolder: 'Mind Palace Actions'
        });

        if (!selection) return;

        if (selection.label === '$(heart) Heal Context') {
            performHeal(false);
        } else if (selection.label === '$(search) Search Palace') {
            // Focus the Blueprint view which contains the search
            await vscode.commands.executeCommand('mindPalace.blueprintView.focus');
        } else if (selection.label === '$(layout-sidebar-left) Focus Blueprint') {
            vscode.commands.executeCommand('mindPalace.blueprintView.focus');
        } else if (selection.label === '$(file-code) Open Context Pack') {
            if (vscode.workspace.workspaceFolders?.[0]) {
                const uri = vscode.Uri.file(path.join(
                    vscode.workspace.workspaceFolders[0].uri.fsPath,
                    '.palace', 'outputs', 'context-pack.json'
                ));
                try {
                    const doc = await vscode.workspace.openTextDocument(uri);
                    await vscode.window.showTextDocument(doc);
                } catch (e) {
                    vscode.window.showErrorMessage("Could not open context-pack.json. Has it been generated?");
                }
            }
        } else if (selection.label === '$(settings-gear) Settings') {
            vscode.commands.executeCommand('workbench.action.openSettings', 'mindPalace');
        }
    });

    const disposableSave = vscode.workspace.onDidSaveTextDocument(async (doc) => {
        try {
            const workspaceFolder = vscode.workspace.getWorkspaceFolder(doc.uri);
            if (!workspaceFolder) {
                return;
            }

            const config = getConfig();
            const { waitForCleanWorkspace: waitForClean, autoSync, autoSyncDelay } = config;

            if (waitForClean) {
                const hasDirty = vscode.workspace.textDocuments.some(d => d.isDirty);
                if (hasDirty) {
                    return;
                }
            }

            if (debounceTimer) {
                clearTimeout(debounceTimer);
            }
            if (countdownInterval) {
                clearInterval(countdownInterval);
            }

            const delaySeconds = autoSyncDelay / 1000;
            let remaining = delaySeconds;

            hud.showPending(remaining);

            countdownInterval = setInterval(() => {
                remaining -= 0.5;
                if (remaining > 0) {
                    hud.showPending(remaining);
                }
            }, 500);

            debounceTimer = setTimeout(() => {
                if (countdownInterval) {
                    clearInterval(countdownInterval);
                    countdownInterval = undefined;
                }
                if (autoSync) {
                    performHeal(true);
                } else {
                    checkStatus();
                }
            }, autoSyncDelay);
        } catch {
            hud.showStale();
        }
    });

    const disposableEditorChange = vscode.window.onDidChangeActiveTextEditor(editor => {
        if (editor) {
            hud.updateRoomInfo();
            decorator.updateDecorations(editor);
            fileIntelProvider.updateDecorations(editor);
        }
    });

    // Corridor commands
    const disposableRefreshCorridor = vscode.commands.registerCommand('mindPalace.refreshCorridor', () => {
        corridorProvider.refresh();
    });

    const disposableShowCorridorLearningDetail = vscode.commands.registerCommand('mindPalace.showCorridorLearningDetail', async (learning: CorridorLearning) => {
        if (!learning) return;

        const panel = vscode.window.createWebviewPanel(
            'mindPalaceCorridorLearning',
            `Learning: ${learning.content.substring(0, 30)}...`,
            vscode.ViewColumn.One,
            { enableScripts: false }
        );

        const tagsHtml = learning.tags && learning.tags.length > 0
            ? `<p><strong>Tags:</strong> ${learning.tags.map(t => `<span class="tag">${t}</span>`).join(' ')}</p>`
            : '';

        panel.webview.html = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Corridor Learning</title>
    <style>
        body {
            font-family: var(--vscode-font-family);
            padding: 20px;
            color: var(--vscode-foreground);
            background-color: var(--vscode-editor-background);
        }
        h1 { font-size: 1.5em; margin-bottom: 16px; }
        .meta { font-size: 0.9em; color: var(--vscode-descriptionForeground); }
        .meta p { margin: 8px 0; }
        .meta strong { color: var(--vscode-foreground); }
        .content {
            padding: 16px;
            background-color: var(--vscode-textCodeBlock-background);
            border-radius: 4px;
            margin: 16px 0;
        }
        .tag {
            display: inline-block;
            padding: 2px 8px;
            margin: 2px;
            border-radius: 12px;
            background-color: var(--vscode-badge-background);
            color: var(--vscode-badge-foreground);
            font-size: 0.85em;
        }
        .confidence {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 4px;
            background-color: var(--vscode-badge-background);
            color: var(--vscode-badge-foreground);
        }
    </style>
</head>
<body>
    <h1>Personal Learning</h1>
    <div class="content">${learning.content}</div>
    <div class="meta">
        <p><strong>ID:</strong> ${learning.id}</p>
        <p><strong>Confidence:</strong> <span class="confidence">${Math.round(learning.confidence * 100)}%</span></p>
        <p><strong>Origin:</strong> ${learning.originWorkspace || 'Unknown'}</p>
        <p><strong>Source:</strong> ${learning.source || 'manual'}</p>
        <p><strong>Used:</strong> ${learning.useCount} times</p>
        <p><strong>Created:</strong> ${new Date(learning.createdAt).toLocaleString()}</p>
        <p><strong>Last Used:</strong> ${new Date(learning.lastUsed).toLocaleString()}</p>
        ${tagsHtml}
    </div>
</body>
</html>`;
    });

    const disposableReinforceLearning = vscode.commands.registerCommand('mindPalace.reinforceCorridorLearning', async (learning?: CorridorLearning) => {
        if (!learning) return;

        try {
            await bridge.reinforceCorridorLearning(learning.id);
            vscode.window.showInformationMessage(`Learning reinforced: confidence increased`);
            corridorProvider.refresh();
        } catch (err: any) {
            vscode.window.showErrorMessage(`Failed to reinforce: ${err.message}`);
        }
    });

    // Conversation commands
    const disposableSearchConversations = vscode.commands.registerCommand('mindPalace.searchConversations', async () => {
        const query = await vscode.window.showInputBox({
            prompt: 'Search past conversations',
            placeHolder: 'Enter search query (leave empty to list recent)',
        });

        try {
            const conversations = await bridge.searchConversations({ query, limit: 20 });

            if (conversations.length === 0) {
                vscode.window.showInformationMessage('No conversations found');
                return;
            }

            const items = conversations.map(c => ({
                label: `$(comment-discussion) ${c.summary}`,
                description: `${c.agentType} - ${c.messages.length} messages`,
                detail: new Date(c.createdAt).toLocaleString(),
                conversation: c,
            }));

            const selection = await vscode.window.showQuickPick(items, {
                placeHolder: 'Select a conversation to view',
                title: 'Past Conversations',
            });

            if (selection) {
                vscode.commands.executeCommand('mindPalace.showConversationDetail', selection.conversation);
            }
        } catch (err: any) {
            vscode.window.showErrorMessage(`Failed to search: ${err.message}`);
        }
    });

    const disposableShowConversationDetail = vscode.commands.registerCommand('mindPalace.showConversationDetail', async (conversation: Conversation) => {
        if (!conversation) return;

        const panel = vscode.window.createWebviewPanel(
            'mindPalaceConversation',
            `Conversation: ${conversation.summary.substring(0, 30)}...`,
            vscode.ViewColumn.One,
            { enableScripts: false }
        );

        const messagesHtml = conversation.messages.map(m => {
            const roleClass = m.role === 'user' ? 'user' : m.role === 'assistant' ? 'assistant' : 'system';
            const roleLabel = m.role.charAt(0).toUpperCase() + m.role.slice(1);
            return `<div class="message ${roleClass}">
                <div class="role">${roleLabel}</div>
                <div class="content">${escapeHtml(m.content)}</div>
            </div>`;
        }).join('\n');

        panel.webview.html = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Conversation</title>
    <style>
        body {
            font-family: var(--vscode-font-family);
            padding: 20px;
            color: var(--vscode-foreground);
            background-color: var(--vscode-editor-background);
        }
        h1 { font-size: 1.3em; margin-bottom: 8px; }
        .meta { font-size: 0.85em; color: var(--vscode-descriptionForeground); margin-bottom: 16px; }
        .message { margin: 12px 0; padding: 12px; border-radius: 8px; }
        .message.user { background-color: var(--vscode-textBlockQuote-background); border-left: 3px solid var(--vscode-textLink-foreground); }
        .message.assistant { background-color: var(--vscode-editor-inactiveSelectionBackground); border-left: 3px solid var(--vscode-debugIcon-startForeground); }
        .message.system { background-color: var(--vscode-editorWarning-background); border-left: 3px solid var(--vscode-editorWarning-foreground); font-style: italic; }
        .role { font-weight: bold; font-size: 0.85em; margin-bottom: 4px; text-transform: uppercase; }
        .content { white-space: pre-wrap; word-wrap: break-word; }
    </style>
</head>
<body>
    <h1>${escapeHtml(conversation.summary)}</h1>
    <div class="meta">
        <strong>ID:</strong> ${conversation.id} |
        <strong>Agent:</strong> ${conversation.agentType} |
        <strong>Messages:</strong> ${conversation.messages.length} |
        <strong>Created:</strong> ${new Date(conversation.createdAt).toLocaleString()}
    </div>
    <div class="messages">${messagesHtml}</div>
</body>
</html>`;
    });

    // Links & Tags commands
    const disposableShowLinks = vscode.commands.registerCommand('mindPalace.showLinks', async (item?: { type: string; data: { id: string } }) => {
        if (!item?.data?.id) {
            vscode.window.showWarningMessage('No record selected');
            return;
        }

        try {
            const links = await bridge.getLinks(item.data.id);

            if (links.length === 0) {
                vscode.window.showInformationMessage(`No links found for ${item.data.id}`);
                return;
            }

            const items = links.map(link => ({
                label: `$(link) ${link.relation}`,
                description: link.sourceId === item.data.id
                    ? `-> ${link.targetId}`
                    : `<- ${link.sourceId}`,
                detail: `${link.sourceKind} ${link.relation} ${link.targetKind}`,
                link,
            }));

            const selection = await vscode.window.showQuickPick(items, {
                placeHolder: 'Links for this record',
                title: `Links: ${item.data.id}`,
            });

            if (selection) {
                // Show link details or navigate to related record
                vscode.window.showInformationMessage(
                    `Link: ${selection.link.sourceId} ${selection.link.relation} ${selection.link.targetId}`
                );
            }
        } catch (err: any) {
            vscode.window.showErrorMessage(`Failed to get links: ${err.message}`);
        }
    });

    const disposableCreateLink = vscode.commands.registerCommand('mindPalace.createLink', async (item?: { type: string; data: { id: string } }) => {
        if (!item?.data?.id) {
            vscode.window.showWarningMessage('No record selected');
            return;
        }

        const targetId = await vscode.window.showInputBox({
            prompt: 'Enter the target record ID',
            placeHolder: 'e.g., d_abc123, i_def456, l_ghi789',
        });

        if (!targetId) return;

        const relation = await vscode.window.showQuickPick(
            [
                { label: 'supports', description: 'This record supports the target' },
                { label: 'contradicts', description: 'This record contradicts the target' },
                { label: 'implements', description: 'This record implements the target' },
                { label: 'supersedes', description: 'This record supersedes the target' },
                { label: 'inspired_by', description: 'This record is inspired by the target' },
                { label: 'related', description: 'This record is related to the target' },
            ],
            { placeHolder: 'Select relationship type' }
        );

        if (!relation) return;

        try {
            const linkId = await bridge.createLink(item.data.id, targetId, relation.label as RecordLink['relation']);
            vscode.window.showInformationMessage(`Link created: ${linkId}`);
            knowledgeProvider.refresh();
        } catch (err: any) {
            vscode.window.showErrorMessage(`Failed to create link: ${err.message}`);
        }
    });

    // Show learning suggestions command
    const disposableShowLearningSuggestions = vscode.commands.registerCommand('mindPalace.showLearningSuggestions', async (filePath: string, suggestions: LearningSuggestion[]) => {
        if (!suggestions || suggestions.length === 0) {
            vscode.window.showInformationMessage('No relevant learnings found for this file');
            return;
        }

        const items = suggestions.map((s, i) => ({
            label: `${i + 1}. ${s.content.substring(0, 60)}${s.content.length > 60 ? '...' : ''}`,
            description: `${Math.round(s.similarity * 100)}% relevant`,
            detail: `Created: ${new Date(s.createdAt).toLocaleDateString()}`,
            suggestion: s,
        }));

        const selection = await vscode.window.showQuickPick(items, {
            placeHolder: `${suggestions.length} relevant learnings for this file`,
            title: 'Learning Suggestions',
        });

        if (selection) {
            vscode.commands.executeCommand('mindPalace.showLearningDetail', selection.suggestion);
        }
    });

    // Show learning detail command
    const disposableShowLearningDetail = vscode.commands.registerCommand('mindPalace.showLearningDetail', async (learning: LearningSuggestion | any) => {
        if (!learning) return;

        const panel = vscode.window.createWebviewPanel(
            'mindPalaceLearningDetail',
            `Learning: ${learning.content?.substring(0, 30) ?? 'Detail'}...`,
            vscode.ViewColumn.One,
            { enableScripts: false }
        );

        const confidenceInfo = learning.confidence !== undefined || learning.similarity !== undefined
            ? `<p><strong>Confidence:</strong> ${Math.round((learning.confidence ?? learning.similarity ?? 0.5) * 100)}%</p>`
            : '';

        panel.webview.html = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Learning Detail</title>
    <style>
        body {
            font-family: var(--vscode-font-family);
            padding: 20px;
            color: var(--vscode-foreground);
            background-color: var(--vscode-editor-background);
        }
        h1 { font-size: 1.5em; margin-bottom: 16px; }
        .content {
            font-size: 1.1em;
            line-height: 1.6;
            margin-bottom: 24px;
            padding: 16px;
            background-color: var(--vscode-textBlockQuote-background);
            border-left: 4px solid var(--vscode-charts-green);
            border-radius: 4px;
        }
        .meta {
            font-size: 0.9em;
            color: var(--vscode-descriptionForeground);
        }
        .meta p { margin: 8px 0; }
        .meta strong { color: var(--vscode-foreground); }
    </style>
</head>
<body>
    <h1>Learning</h1>
    <div class="content">${escapeHtml(learning.content || '')}</div>
    <div class="meta">
        ${learning.id ? `<p><strong>ID:</strong> ${learning.id}</p>` : ''}
        ${confidenceInfo}
        ${learning.createdAt ? `<p><strong>Created:</strong> ${new Date(learning.createdAt).toLocaleString()}</p>` : ''}
    </div>
</body>
</html>`;
    });

    // Show knowledge graph command
    const disposableShowKnowledgeGraph = vscode.commands.registerCommand('mindPalace.showKnowledgeGraph', async () => {
        const currentFile = vscode.window.activeTextEditor?.document.uri.fsPath;
        KnowledgeGraphPanel.createOrShow(context.extensionUri, bridge, currentFile);
    });

    // Semantic search command
    const disposableSemanticSearch = vscode.commands.registerCommand('mindPalace.semanticSearch', async () => {
        const query = await vscode.window.showInputBox({
            prompt: 'Semantic search for knowledge',
            placeHolder: 'Enter natural language query (e.g., "error handling patterns")',
        });

        if (!query) return;

        const kindOptions = await vscode.window.showQuickPick(
            [
                { label: 'All', description: 'Search all record types', picked: true, kinds: [] },
                { label: 'Ideas', description: 'Search ideas only', kinds: ['idea'] },
                { label: 'Decisions', description: 'Search decisions only', kinds: ['decision'] },
                { label: 'Learnings', description: 'Search learnings only', kinds: ['learning'] },
            ],
            { placeHolder: 'Filter by record type' }
        );

        if (!kindOptions) return;

        try {
            const results = await bridge.hybridSearch(query, {
                kinds: kindOptions.kinds as string[],
                limit: 20,
            });

            if (results.length === 0) {
                vscode.window.showInformationMessage('No matching records found');
                return;
            }

            const items = results.map((r, i) => ({
                label: `${i + 1}. [${r.kind}] ${r.content.substring(0, 50)}${r.content.length > 50 ? '...' : ''}`,
                description: r.matchType === 'both' ? 'keyword + semantic' : r.matchType,
                detail: r.similarity ? `Similarity: ${(r.similarity * 100).toFixed(0)}%` : undefined,
                result: r,
            }));

            const selection = await vscode.window.showQuickPick(items, {
                placeHolder: `Found ${results.length} results`,
                title: 'Semantic Search Results',
            });

            if (selection) {
                vscode.commands.executeCommand('mindPalace.showKnowledgeDetail', {
                    type: selection.result.kind,
                    data: selection.result,
                });
            }
        } catch (err: any) {
            if (err.message?.includes('embedding')) {
                vscode.window.showWarningMessage(
                    'Semantic search requires embeddings to be configured. Using keyword search only.'
                );
            } else {
                vscode.window.showErrorMessage(`Search failed: ${err.message}`);
            }
        }
    });

    context.subscriptions.push(
        disposableHeal,
        disposableCheckStatus,
        disposableOpenBlueprint,
        disposableRefreshKnowledge,
        disposableRefreshSessions,
        disposableStartSession,
        disposableEndSession,
        disposableShowSessionDetail,
        disposableShowConflictInfo,
        disposableShowFileIntel,
        disposableShowCallGraph,
        disposableShowKnowledgeDetail,
        disposableShowMenu,
        disposableRefreshCorridor,
        disposableShowCorridorLearningDetail,
        disposableReinforceLearning,
        disposableSearchConversations,
        disposableShowConversationDetail,
        disposableShowLinks,
        disposableCreateLink,
        disposableShowLearningSuggestions,
        disposableShowLearningDetail,
        disposableShowKnowledgeGraph,
        disposableSemanticSearch,
        disposableSave,
        disposableEditorChange
    );

    // Apply decorations to the initially active editor
    if (vscode.window.activeTextEditor) {
        decorator.updateDecorations(vscode.window.activeTextEditor);
    }

    async function performHeal(silent: boolean = false) {
        const config = getConfig();

        if (config.waitForCleanWorkspace) {
            const hasDirty = vscode.workspace.textDocuments.some(d => d.isDirty);
            if (hasDirty) {
                if (!silent) {
                    vscode.window.showWarningMessage("Mind Palace: Heal aborted. Please save all files first.");
                }
                return;
            }
        }

        hud.showScanning();
        try {
            await bridge.runHeal();
            hud.showFresh();
            if (!silent) {
                vscode.window.showInformationMessage('Mind Palace healed successfully.');
            }
            await checkStatus();
        } catch (err: any) {
            vscode.window.showErrorMessage(`Mind Palace heal failed: ${err.message}`);
            hud.showStale();
        }
    }

    async function checkStatus() {
        try {
            const isSynced = await bridge.runVerify();
            if (isSynced) {
                hud.showFresh();
            } else {
                hud.showStale();
            }

            if (vscode.window.activeTextEditor) {
                decorator.updateDecorations(vscode.window.activeTextEditor);
            }
            sidebarProvider.refresh();
            knowledgeProvider.refresh();
            sessionProvider.refresh();
        } catch (error: any) {
            if (error.message === 'Palace binary not found') {
                vscode.window.showErrorMessage("Palace binary not found. Please configure 'mindPalace.binaryPath'.");
            }
            hud.showStale();
        }
    }
}

export function deactivate() { }

/**
 * Escape HTML special characters
 */
function escapeHtml(text: string): string {
    return text
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}