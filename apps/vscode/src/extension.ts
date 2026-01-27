import * as vscode from "vscode";
import { PalaceBridge } from "./bridge";
import { watchProjectConfig } from "./config";
import { StatusBar } from "./core/status-bar";
import { warnIfIncompatible } from "./version";
import { activateLspClient, deactivateLspClient } from "./lsp";

export async function activate(context: vscode.ExtensionContext) {
  warnIfIncompatible();

  const bridge = new PalaceBridge();
  const statusBar = new StatusBar(bridge);

  // Watch project config
  const configWatcher = watchProjectConfig(() => statusBar.refresh());
  context.subscriptions.push(configWatcher);

  // Register status bar
  context.subscriptions.push(statusBar);

  // Register commands
  context.subscriptions.push(
    vscode.commands.registerCommand("mindPalace.checkStatus", () => statusBar.refresh()),
    vscode.commands.registerCommand("mindPalace.restartLsp", async () => {
      await deactivateLspClient();
      await activateLspClient(context);
      vscode.window.showInformationMessage("Mind Palace LSP server restarted");
    })
  );

  // Start LSP client for real-time pattern and contract diagnostics
  try {
    await activateLspClient(context);
  } catch (error) {
    console.error("Failed to activate LSP client:", error);
  }

  // Perform initial status check
  statusBar.refresh();
}

export async function deactivate() {
  await deactivateLspClient();
}
