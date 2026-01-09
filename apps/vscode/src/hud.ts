import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
import { parse as parseJSONC } from 'jsonc-parser';
import { Room, HUDStatus } from './types';

export class PalaceHUD {
    private statusBarItem: vscode.StatusBarItem;
    private currentStatus: HUDStatus = 'fresh';
    private pendingSeconds: number = 0;

    constructor() {
        this.statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
        this.statusBarItem.command = 'mindPalace.showMenu';
        this.statusBarItem.tooltip = "Click to manage Mind Palace";
        this.showFresh();
        this.statusBarItem.show();
    }

    showFresh() {
        this.currentStatus = 'fresh';
        this.pendingSeconds = 0;
        this.statusBarItem.text = "$(check) Palace: Synced";
        this.statusBarItem.backgroundColor = undefined;
        this.statusBarItem.color = undefined;
        this.updateRoomInfo();
    }

    showStale() {
        this.currentStatus = 'stale';
        this.pendingSeconds = 0;
        this.statusBarItem.text = "$(alert) Palace: Stale";
        this.statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.errorBackground');
        this.statusBarItem.color = undefined;
        this.updateRoomInfo();
    }

    showPending(secondsRemaining: number) {
        this.currentStatus = 'pending';
        this.pendingSeconds = secondsRemaining;
        const seconds = Math.ceil(secondsRemaining);
        this.statusBarItem.text = `$(clock) Palace: Healing in ${seconds}s`;
        this.statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.warningBackground');
        this.statusBarItem.color = undefined;
    }

    showScanning() {
        this.currentStatus = 'scanning';
        this.pendingSeconds = 0;
        this.statusBarItem.text = "$(sync~spin) Palace: Scanning...";
        this.statusBarItem.backgroundColor = undefined;
        this.statusBarItem.color = undefined;
    }

    refresh() {
        switch (this.currentStatus) {
            case 'fresh':
                this.statusBarItem.text = "$(check) Palace: Synced";
                this.statusBarItem.backgroundColor = undefined;
                break;
            case 'stale':
                this.statusBarItem.text = "$(alert) Palace: Stale";
                this.statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.errorBackground');
                break;
            case 'pending':
                const seconds = Math.ceil(this.pendingSeconds);
                this.statusBarItem.text = `$(clock) Palace: Healing in ${seconds}s`;
                this.statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.warningBackground');
                break;
            case 'scanning':
                this.statusBarItem.text = "$(sync~spin) Palace: Scanning...";
                this.statusBarItem.backgroundColor = undefined;
                break;
        }
        this.statusBarItem.color = undefined;
        this.updateRoomInfo();
    }

    getStatus(): HUDStatus {
        return this.currentStatus;
    }

    async updateRoomInfo() {
        const editor = vscode.window.activeTextEditor;
        if (!editor) {
            return;
        }

        const currentFilePath = vscode.workspace.asRelativePath(editor.document.fileName);
        const roomName = await this.findRoomForFile(currentFilePath);

        if (roomName) {
            const { icon, label } = this.getStatusDisplay();
            this.statusBarItem.text = `${icon} Palace: ${label} | Room: ${roomName}`;
        }
    }

    private getStatusDisplay(): { icon: string; label: string } {
        switch (this.currentStatus) {
            case 'fresh':
                return { icon: '$(check)', label: 'Synced' };
            case 'stale':
                return { icon: '$(alert)', label: 'Stale' };
            case 'pending':
                return { icon: '$(clock)', label: `Healing in ${Math.ceil(this.pendingSeconds)}s` };
            case 'scanning':
                return { icon: '$(sync~spin)', label: 'Scanning...' };
        }
    }

    private async findRoomForFile(filePath: string): Promise<string | undefined> {
        const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
        if (!workspaceFolder) return undefined;

        const roomFiles = await vscode.workspace.findFiles('.palace/rooms/*.jsonc');

        for (const uri of roomFiles) {
            try {
                const content = fs.readFileSync(uri.fsPath, 'utf8');
                const room: Room = parseJSONC(content);

                if (room.entryPoints?.includes(filePath)) {
                    return room.name || path.basename(uri.fsPath, '.jsonc');
                }
            } catch {
                continue;
            }
        }
        return undefined;
    }

    /**
     * Dispose of the status bar item
     */
    dispose() {
        this.statusBarItem.dispose();
    }
}
