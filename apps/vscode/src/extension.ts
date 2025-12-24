import * as path from 'path';
import * as vscode from 'vscode';
import { PalaceBridge } from './bridge';
import { getConfig, watchProjectConfig } from './config';
import { PalaceDecorator } from './decorator';
import { PalaceHUD } from './hud';
import { PalaceSidebarProvider } from './sidebar';
import { warnIfIncompatible } from './version';

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
        }
    });

    context.subscriptions.push(
        disposableHeal,
        disposableCheckStatus,
        disposableOpenBlueprint,
        disposableShowMenu,
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
        } catch (error: any) {
            if (error.message === 'Palace binary not found') {
                vscode.window.showErrorMessage("Palace binary not found. Please configure 'mindPalace.binaryPath'.");
            }
            hud.showStale();
        }
    }
}

export function deactivate() { }