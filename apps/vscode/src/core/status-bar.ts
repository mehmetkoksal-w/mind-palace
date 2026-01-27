import * as vscode from "vscode";
import { PalaceBridge } from "../bridge";

/**
 * StatusBar shows Mind Palace status in the VS Code status bar.
 * Displays: index freshness, knowledge counts, and provides quick access to commands.
 */
export class StatusBar implements vscode.Disposable {
  private statusBarItem: vscode.StatusBarItem;
  private bridge: PalaceBridge;

  constructor(bridge: PalaceBridge) {
    this.bridge = bridge;
    this.statusBarItem = vscode.window.createStatusBarItem(
      vscode.StatusBarAlignment.Left,
      100
    );
    this.statusBarItem.command = "mindPalace.checkStatus";
    this.statusBarItem.show();
    this.refresh();
  }

  async refresh(): Promise<void> {
    try {
      const status = await this.bridge.getStatus();
      
      if (status.error) {
        this.statusBarItem.text = "$(warning) Palace: Error";
        this.statusBarItem.tooltip = status.error;
        this.statusBarItem.backgroundColor = new vscode.ThemeColor("statusBarItem.errorBackground");
        return;
      }

      if (!status.initialized) {
        this.statusBarItem.text = "$(info) Palace: Not initialized";
        this.statusBarItem.tooltip = "Run 'palace init' to initialize";
        this.statusBarItem.backgroundColor = undefined;
        return;
      }

      const freshness = status.fresh ? "$(check)" : "$(warning)";
      const counts = [];
      if (status.decisions > 0) counts.push(`${status.decisions}D`);
      if (status.ideas > 0) counts.push(`${status.ideas}I`);
      if (status.learnings > 0) counts.push(`${status.learnings}L`);
      
      const countStr = counts.length > 0 ? ` ${counts.join("/")}` : "";
      
      this.statusBarItem.text = `${freshness} Palace${countStr}`;
      this.statusBarItem.tooltip = status.fresh 
        ? `Index fresh (${status.rooms} rooms)`
        : "Index stale - run 'palace index scan'";
      this.statusBarItem.backgroundColor = status.fresh 
        ? undefined 
        : new vscode.ThemeColor("statusBarItem.warningBackground");
        
    } catch (error) {
      this.statusBarItem.text = "$(error) Palace";
      this.statusBarItem.tooltip = `Error: ${error}`;
      this.statusBarItem.backgroundColor = new vscode.ThemeColor("statusBarItem.errorBackground");
    }
  }

  dispose(): void {
    this.statusBarItem.dispose();
  }
}
