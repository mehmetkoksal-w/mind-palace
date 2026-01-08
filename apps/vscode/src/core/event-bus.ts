import * as vscode from "vscode";
import { cacheRegistry } from "../services/cache";
import { logger } from "../services/logger";

export class EventBus {
  private disposables: vscode.Disposable[] = [];
  private debounceTimer: NodeJS.Timeout | undefined;
  private countdownInterval: NodeJS.Timeout | undefined;

  constructor() {}

  registerAll(context: vscode.ExtensionContext): vscode.Disposable[] {
    this.registerWorkspaceEvents();
    this.registerConfigurationEvents();
    this.registerFileEvents();
    this.registerEditorEvents();
    return this.disposables;
  }

  private registerWorkspaceEvents(): void {
    this.disposables.push(
      vscode.workspace.onDidChangeWorkspaceFolders(() => {
        logger.info("Workspace folders changed", "EventBus");
        cacheRegistry.clearAll();
        vscode.commands.executeCommand("mindPalace.checkStatus");
      })
    );
  }

  private registerConfigurationEvents(): void {
    this.disposables.push(
      vscode.workspace.onDidChangeConfiguration((e) => {
        if (e.affectsConfiguration("mindPalace")) {
          logger.info("Configuration changed", "EventBus");
          cacheRegistry.clearAll();
        }
      })
    );
  }

  private registerFileEvents(): void {
    this.disposables.push(
      vscode.workspace.onDidSaveTextDocument(async () => {
        try {
          const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
          if (!workspaceFolder) {
            return;
          }

          const vsConfig = vscode.workspace.getConfiguration("mindPalace");
          const waitForClean =
            vsConfig.get<boolean>("waitForCleanWorkspace") ?? false;
          const autoSync = vsConfig.get<boolean>("autoSync") ?? true;
          const autoSyncDelay = vsConfig.get<number>("autoSyncDelay") ?? 1500;

          if (waitForClean) {
            const hasDirty = vscode.workspace.textDocuments.some(
              (d) => d.isDirty
            );
            if (hasDirty) {
              return;
            }
          }

          if (this.debounceTimer) {
            clearTimeout(this.debounceTimer);
          }
          if (this.countdownInterval) {
            clearInterval(this.countdownInterval);
          }

          const delaySeconds = autoSyncDelay / 1000;
          let remaining = delaySeconds;
          vscode.commands.executeCommand("mindPalace.checkStatus");

          this.countdownInterval = setInterval(() => {
            remaining -= 0.5;
            // Optionally could post status via HUD, but avoid tight coupling here
          }, 500);

          this.debounceTimer = setTimeout(() => {
            if (this.countdownInterval) {
              clearInterval(this.countdownInterval);
              this.countdownInterval = undefined;
            }
            if (autoSync) {
              vscode.commands.executeCommand("mindPalace.heal", true);
            } else {
              vscode.commands.executeCommand("mindPalace.checkStatus");
            }
          }, autoSyncDelay);
        } catch {
          // Swallow errors to avoid disrupting save flow
        }
      })
    );
  }

  private registerEditorEvents(): void {
    this.disposables.push(
      vscode.window.onDidChangeActiveTextEditor(async () => {
        // Update status and decorations when editor focus changes
        vscode.commands.executeCommand("mindPalace.checkStatus");
      })
    );
  }

  dispose(): void {
    this.disposables.forEach((d) => d.dispose());
    if (this.debounceTimer) clearTimeout(this.debounceTimer);
    if (this.countdownInterval) clearInterval(this.countdownInterval);
  }
}
