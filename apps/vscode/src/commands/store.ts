import * as vscode from 'vscode';
import * as path from 'path';
import { PalaceBridge, StoreResult } from '../bridge';

/**
 * Store command types
 */
export type StoreType = 'idea' | 'decision' | 'learning';

/**
 * Determine scope from a file path
 */
function determineScope(filePath: string | undefined, workspaceRoot: string | undefined): { scope: 'palace' | 'room' | 'file'; scopePath?: string } {
    if (!filePath || !workspaceRoot) {
        return { scope: 'palace' };
    }

    // Get relative path from workspace root
    const relativePath = path.relative(workspaceRoot, filePath);

    // Check if inside a room (.palace/rooms/*.jsonc defines rooms)
    // For now, use file scope for specific files
    if (relativePath) {
        return {
            scope: 'file',
            scopePath: relativePath,
        };
    }

    return { scope: 'palace' };
}

/**
 * Get selected text or prompt for input
 */
async function getContent(preselectedText?: string): Promise<string | undefined> {
    if (preselectedText) {
        // If text is selected, optionally allow editing it
        const result = await vscode.window.showInputBox({
            prompt: 'Edit content (or leave as-is)',
            value: preselectedText,
            placeHolder: 'Enter your thought...',
            validateInput: (value) => {
                if (!value.trim()) {
                    return 'Content cannot be empty';
                }
                return undefined;
            },
        });
        return result;
    }

    // No pre-selected text, prompt for input
    const result = await vscode.window.showInputBox({
        prompt: 'What do you want to remember?',
        placeHolder: 'Enter your thought...',
        validateInput: (value) => {
            if (!value.trim()) {
                return 'Content cannot be empty';
            }
            return undefined;
        },
    });
    return result;
}

/**
 * Show success notification with stored item info
 */
function showSuccessNotification(type: StoreType, result: StoreResult): void {
    const typeLabels: Record<StoreType, string> = {
        'idea': 'Idea',
        'decision': 'Decision',
        'learning': 'Learning',
    };

    const icons: Record<StoreType, string> = {
        'idea': '$(lightbulb)',
        'decision': '$(law)',
        'learning': '$(book)',
    };

    const message = `${icons[type]} ${typeLabels[type]} stored successfully`;

    // Show with view action
    vscode.window.showInformationMessage(
        message,
        'View in Knowledge Panel'
    ).then(selection => {
        if (selection === 'View in Knowledge Panel') {
            vscode.commands.executeCommand('mindPalace.knowledgeView.focus');
            vscode.commands.executeCommand('mindPalace.refreshKnowledge');
        }
    });

    // Check for contradictions and show warning
    if (result.contradictions && result.contradictions.length > 0) {
        showContradictionWarning(result);
    }
}

/**
 * Show contradiction warning when storing conflicts with existing knowledge
 */
async function showContradictionWarning(result: StoreResult): Promise<void> {
    const contradictions = result.contradictions!;
    const count = contradictions.length;

    const firstContradiction = contradictions[0];
    const confidence = Math.round(firstContradiction.confidence * 100);

    const message = count === 1
        ? `$(warning) Contradiction detected (${confidence}% confidence): ${firstContradiction.explanation?.substring(0, 80) || 'This may conflict with existing knowledge'}...`
        : `$(warning) ${count} contradictions detected with existing knowledge`;

    const selection = await vscode.window.showWarningMessage(
        message,
        { modal: false },
        'View Details',
        'Dismiss'
    );

    if (selection === 'View Details') {
        showContradictionDetails(result.id, contradictions);
    }
}

/**
 * Show detailed contradiction information in a webview
 */
function showContradictionDetails(recordId: string, contradictions: StoreResult['contradictions']): void {
    if (!contradictions || contradictions.length === 0) return;

    const panel = vscode.window.createWebviewPanel(
        'mindPalaceContradictions',
        `Contradictions: ${recordId.substring(0, 12)}...`,
        vscode.ViewColumn.Two,
        { enableScripts: false }
    );

    const contradictionItems = contradictions.map((c, i) => `
        <div class="contradiction">
            <div class="contradiction-header">
                <span class="contradiction-number">#${i + 1}</span>
                <span class="contradiction-type">${c.type}</span>
                <span class="contradiction-confidence">${Math.round(c.confidence * 100)}% confidence</span>
            </div>
            <div class="conflicting-record">
                <div class="record-kind">${c.conflictingKind}</div>
                <div class="record-content">${escapeHtml(c.conflictingContent)}</div>
                <div class="record-id">ID: ${c.conflictingId}</div>
            </div>
            ${c.explanation ? `<div class="explanation">${escapeHtml(c.explanation)}</div>` : ''}
            ${c.autoLinked ? '<div class="auto-linked">Auto-linked as contradiction</div>' : ''}
        </div>
    `).join('\n');

    panel.webview.html = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Contradictions</title>
    <style>
        body {
            font-family: var(--vscode-font-family);
            padding: 20px;
            color: var(--vscode-foreground);
            background-color: var(--vscode-editor-background);
        }
        h1 {
            font-size: 1.3em;
            margin-bottom: 16px;
            color: var(--vscode-editorWarning-foreground);
        }
        .intro {
            margin-bottom: 24px;
            padding: 12px;
            background-color: var(--vscode-editorWarning-background);
            border-radius: 6px;
            font-size: 0.9em;
        }
        .contradiction {
            margin: 16px 0;
            padding: 16px;
            background-color: var(--vscode-editor-inactiveSelectionBackground);
            border-left: 4px solid var(--vscode-editorWarning-foreground);
            border-radius: 4px;
        }
        .contradiction-header {
            display: flex;
            gap: 12px;
            margin-bottom: 12px;
            font-size: 0.85em;
        }
        .contradiction-number {
            font-weight: bold;
            color: var(--vscode-editorWarning-foreground);
        }
        .contradiction-type {
            padding: 2px 8px;
            background-color: var(--vscode-badge-background);
            border-radius: 4px;
            text-transform: uppercase;
            font-size: 0.8em;
        }
        .contradiction-confidence {
            color: var(--vscode-descriptionForeground);
        }
        .conflicting-record {
            padding: 12px;
            background-color: var(--vscode-textBlockQuote-background);
            border-radius: 4px;
            margin-bottom: 12px;
        }
        .record-kind {
            font-size: 0.75em;
            text-transform: uppercase;
            color: var(--vscode-descriptionForeground);
            margin-bottom: 4px;
        }
        .record-content {
            margin-bottom: 8px;
            line-height: 1.5;
        }
        .record-id {
            font-size: 0.8em;
            color: var(--vscode-descriptionForeground);
            font-family: monospace;
        }
        .explanation {
            padding: 8px 12px;
            background-color: var(--vscode-textPreformat-background);
            border-radius: 4px;
            font-style: italic;
            font-size: 0.9em;
        }
        .auto-linked {
            margin-top: 8px;
            font-size: 0.8em;
            color: var(--vscode-debugIcon-startForeground);
        }
    </style>
</head>
<body>
    <h1>$(warning) Contradictions Detected</h1>
    <div class="intro">
        The knowledge you just stored may conflict with existing records. Review the contradictions below
        and consider updating or archiving outdated information.
    </div>
    <div class="contradictions">
        ${contradictionItems}
    </div>
</body>
</html>`;
}

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

/**
 * Store content as a specific type
 */
export async function storeAs(
    bridge: PalaceBridge,
    type: StoreType,
    content?: string
): Promise<void> {
    try {
        // Get content from selection or prompt
        const editor = vscode.window.activeTextEditor;
        const selectedText = editor?.document.getText(editor.selection);

        const finalContent = await getContent(content || selectedText);
        if (!finalContent) {
            return; // User cancelled
        }

        // Determine scope from current file
        const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
        const filePath = editor?.document.uri.fsPath;
        const { scope, scopePath } = determineScope(filePath, workspaceRoot);

        // Store via bridge
        const result = await bridge.store(finalContent, {
            as: type,
            scope,
            scopePath,
        });

        // Show success
        showSuccessNotification(type, result);

        // Refresh knowledge panel
        vscode.commands.executeCommand('mindPalace.refreshKnowledge');

    } catch (error: any) {
        vscode.window.showErrorMessage(`Failed to store: ${error.message}`);
    }
}

/**
 * Quick store with type picker
 */
export async function quickStore(bridge: PalaceBridge): Promise<void> {
    // Get selected text first
    const editor = vscode.window.activeTextEditor;
    const selectedText = editor?.document.getText(editor.selection);

    // Show type picker
    const items: vscode.QuickPickItem[] = [
        {
            label: '$(lightbulb) Idea',
            description: 'A potential improvement or feature to explore',
            detail: selectedText ? `"${selectedText.substring(0, 50)}${selectedText.length > 50 ? '...' : ''}"` : undefined,
        },
        {
            label: '$(law) Decision',
            description: 'An architectural or design choice made',
            detail: selectedText ? `"${selectedText.substring(0, 50)}${selectedText.length > 50 ? '...' : ''}"` : undefined,
        },
        {
            label: '$(book) Learning',
            description: 'Something learned about the codebase',
            detail: selectedText ? `"${selectedText.substring(0, 50)}${selectedText.length > 50 ? '...' : ''}"` : undefined,
        },
    ];

    const selection = await vscode.window.showQuickPick(items, {
        placeHolder: 'What type of knowledge is this?',
        title: 'Mind Palace: Quick Store',
    });

    if (!selection) {
        return; // User cancelled
    }

    // Map selection to type
    let type: StoreType;
    if (selection.label.includes('Idea')) {
        type = 'idea';
    } else if (selection.label.includes('Decision')) {
        type = 'decision';
    } else {
        type = 'learning';
    }

    await storeAs(bridge, type, selectedText);
}

/**
 * Register all store commands
 */
export function registerStoreCommands(
    context: vscode.ExtensionContext,
    bridge: PalaceBridge
): void {
    // Store as Idea
    context.subscriptions.push(
        vscode.commands.registerCommand('mindPalace.storeIdea', () => {
            storeAs(bridge, 'idea');
        })
    );

    // Store as Decision
    context.subscriptions.push(
        vscode.commands.registerCommand('mindPalace.storeDecision', () => {
            storeAs(bridge, 'decision');
        })
    );

    // Store as Learning
    context.subscriptions.push(
        vscode.commands.registerCommand('mindPalace.storeLearning', () => {
            storeAs(bridge, 'learning');
        })
    );

    // Quick Store (with type picker)
    context.subscriptions.push(
        vscode.commands.registerCommand('mindPalace.quickStore', () => {
            quickStore(bridge);
        })
    );
}
