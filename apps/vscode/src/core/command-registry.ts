import * as vscode from "vscode";
import {
  PalaceBridge,
  CorridorLearning,
  Conversation,
  RecordLink,
} from "../bridge";
import { PalaceHUD } from "../hud";
import { PalaceDecorator } from "../decorator";
import { getConfig } from "../config";
import { ViewRegistry } from "./view-registry";

export interface CommandContext {
  bridge: PalaceBridge;
  hud: PalaceHUD;
  decorator: PalaceDecorator;
  extensionContext: vscode.ExtensionContext;
  views: ViewRegistry;
}

export class CommandRegistry {
  private context: CommandContext;
  private commands: vscode.Disposable[] = [];
  private debounceTimer: NodeJS.Timeout | undefined;
  private countdownInterval: NodeJS.Timeout | undefined;

  constructor(context: CommandContext) {
    this.context = context;
  }

  registerAll(): vscode.Disposable[] {
    this.registerCoreCommands();
    this.registerStoreCommands();
    this.registerSessionCommands();
    this.registerKnowledgeCommands();
    this.registerPostmortemCommands();
    this.registerCorridorCommands();
    this.registerConversationCommands();
    this.registerLinksAndTagsCommands();
    this.registerSemanticCommands();
    this.registerGraphCommands();
    this.registerViewRefreshCommands();
    return this.commands;
  }

  private registerCoreCommands(): void {
    this.register("mindPalace.heal", (silent?: boolean) =>
      this.performHeal(Boolean(silent))
    );
    this.register("mindPalace.checkStatus", () => this.checkStatus());
    this.register("mindPalace.openBlueprint", () => {
      vscode.commands.executeCommand("mindPalace.blueprintView.focus");
    });
    this.register("mindPalace.showMenu", async () => this.showMenu());
    this.register("mindPalace.showFileIntel", async (filePath?: string) =>
      this.showFileIntel(filePath)
    );
    this.register(
      "mindPalace.showCallGraph",
      async (args?: { symbol: string; file: string }) =>
        this.showCallGraph(args)
    );
  }

  private registerStoreCommands(): void {
    const { registerStoreCommands } = require("../commands/store");
    const disposables: vscode.Disposable[] = registerStoreCommands(
      this.context.extensionContext,
      this.context.bridge
    );
    this.commands.push(...disposables);
  }

  private registerSessionCommands(): void {
    this.register("mindPalace.refreshSessions", () =>
      this.context.views.sessionProvider?.refresh()
    );
    this.register("mindPalace.startSession", async () => this.startSession());
    this.register("mindPalace.endSession", async (sessionInfo?: any) =>
      this.endSession(sessionInfo)
    );
    this.register("mindPalace.showSessionDetail", async (session: any) =>
      this.showSessionDetail(session)
    );
    this.register("mindPalace.showConflictInfo", async () =>
      this.showConflictInfo()
    );
  }

  private registerKnowledgeCommands(): void {
    this.register("mindPalace.refreshKnowledge", () =>
      this.context.views.knowledgeProvider?.refresh()
    );
    this.register(
      "mindPalace.showKnowledgeDetail",
      async (item: { type: string; data: any }) =>
        this.showKnowledgeDetail(item)
    );
    this.register(
      "mindPalace.showLearningSuggestions",
      async (filePath: string, suggestions: any[]) =>
        this.showLearningSuggestions(filePath, suggestions)
    );
    this.register("mindPalace.showLearningDetail", async (learning: any) =>
      this.showLearningDetail(learning)
    );
  }

  private registerPostmortemCommands(): void {
    const {
      addPostmortem,
      quickPostmortem,
    } = require("../commands/postmortem");
    this.register("mindPalace.addPostmortem", async () =>
      addPostmortem(this.context.bridge)
    );
    this.register("mindPalace.quickPostmortem", async () =>
      quickPostmortem(this.context.bridge)
    );

    this.register(
      "mindPalace.showPostmortemDetail",
      async (arg: { id: string }) => {
        const {
          PostmortemDetailPanel,
        } = require("../webviews/postmortemDetail");
        await PostmortemDetailPanel.createOrShow(
          this.context.extensionContext.extensionUri,
          this.context.bridge,
          arg.id
        );
      }
    );
  }

  private registerCorridorCommands(): void {
    this.register("mindPalace.refreshCorridor", () =>
      this.context.views.corridorProvider?.refresh()
    );
    this.register(
      "mindPalace.showCorridorLearningDetail",
      async (learning: CorridorLearning) =>
        this.showCorridorLearningDetail(learning)
    );
    this.register(
      "mindPalace.reinforceCorridorLearning",
      async (learning?: CorridorLearning) => this.reinforceLearning(learning)
    );
  }

  private registerConversationCommands(): void {
    this.register("mindPalace.searchConversations", async () =>
      this.searchConversations()
    );
    this.register(
      "mindPalace.showConversationDetail",
      async (conversation: Conversation) =>
        this.showConversationDetail(conversation)
    );
  }

  private registerLinksAndTagsCommands(): void {
    this.register(
      "mindPalace.showLinks",
      async (item?: { type: string; data: { id: string } }) =>
        this.showLinks(item)
    );
    this.register(
      "mindPalace.createLink",
      async (item?: { type: string; data: { id: string } }) =>
        this.createLink(item)
    );
  }

  private registerSemanticCommands(): void {
    this.register("mindPalace.semanticSearch", async () =>
      this.performSemanticSearch()
    );
  }

  private async performSemanticSearch(): Promise<void> {
    const query = await vscode.window.showInputBox({
      prompt: "Enter search query",
      placeHolder: "Search for learnings, decisions, ideas...",
    });

    if (!query) return;

    try {
      const results = await this.context.bridge.semanticSearch(query, {
        limit: 20,
      });

      if (results.length === 0) {
        vscode.window.showInformationMessage("No results found");
        return;
      }

      const items = results.map((r) => ({
        label: `$(${
          r.kind === "learning"
            ? "book"
            : r.kind === "decision"
            ? "law"
            : "lightbulb"
        }) ${r.content.substring(0, 60)}${r.content.length > 60 ? "..." : ""}`,
        description: `${Math.round(r.similarity * 100)}% similarity`,
        detail: new Date(r.createdAt).toLocaleDateString(),
        result: r,
      }));

      const selection = await vscode.window.showQuickPick(items, {
        placeHolder: `${results.length} results for "${query}"`,
        title: "Semantic Search Results",
      });

      if (selection) {
        vscode.commands.executeCommand("mindPalace.showKnowledgeDetail", {
          type: selection.result.kind,
          data: selection.result,
        });
      }
    } catch (err: any) {
      vscode.window.showErrorMessage(`Semantic search failed: ${err.message}`);
    }
  }

  private registerGraphCommands(): void {
    this.register("mindPalace.showKnowledgeGraph", async () => {
      const currentFile = vscode.window.activeTextEditor?.document.uri.fsPath;
      const {
        KnowledgeGraphPanel,
      } = require("../webviews/knowledgeGraph/knowledgeGraphPanel");
      KnowledgeGraphPanel.createOrShow(
        this.context.extensionContext.extensionUri,
        this.context.bridge,
        currentFile
      );
    });
  }

  private registerViewRefreshCommands(): void {
    // Focus blueprint
    this.register("mindPalace.openBlueprint", () => {
      vscode.commands.executeCommand("mindPalace.blueprintView.focus");
    });
  }

  private register(command: string, handler: (...args: any[]) => any): void {
    const disposable = vscode.commands.registerCommand(command, handler);
    this.commands.push(disposable);
  }

  async performHeal(silent: boolean = false): Promise<void> {
    const config = getConfig();

    if (config.waitForCleanWorkspace) {
      const hasDirty = vscode.workspace.textDocuments.some((d) => d.isDirty);
      if (hasDirty) {
        if (!silent) {
          vscode.window.showWarningMessage(
            "Mind Palace: Heal aborted. Please save all files first."
          );
        }
        return;
      }
    }

    this.context.hud.showScanning();
    try {
      await this.context.bridge.runHeal();
      this.context.hud.showFresh();
      if (!silent) {
        vscode.window.showInformationMessage(
          "Mind Palace healed successfully."
        );
      }
      await this.checkStatus();
    } catch (err: any) {
      vscode.window.showErrorMessage(`Mind Palace heal failed: ${err.message}`);
      this.context.hud.showStale();
    }
  }

  async checkStatus(): Promise<void> {
    try {
      const isSynced = await this.context.bridge.runVerify();
      if (isSynced) {
        this.context.hud.showFresh();
      } else {
        this.context.hud.showStale();
      }

      if (vscode.window.activeTextEditor) {
        this.context.decorator.updateDecorations(
          vscode.window.activeTextEditor
        );
      }

      this.context.views.sidebarProvider?.refresh();
      this.context.views.knowledgeProvider?.refresh();
      this.context.views.sessionProvider?.refresh();
    } catch (error: any) {
      if (error.message === "Palace binary not found") {
        vscode.window.showErrorMessage(
          "Palace binary not found. Please configure 'mindPalace.binaryPath'."
        );
      }
      this.context.hud.showStale();
    }
  }

  private async showMenu(): Promise<void> {
    const items: vscode.QuickPickItem[] = [
      {
        label: "$(heart) Heal Context",
        description: "Run palace scan && collect",
      },
      {
        label: "$(search) Search Palace",
        description: "Focus the search input in Blueprint",
      },
      {
        label: "$(layout-sidebar-left) Focus Blueprint",
        description: "Show the Blueprint Sidebar",
      },
      {
        label: "$(file-code) Open Context Pack",
        description: "View the generated context-pack.json",
      },
      {
        label: "$(settings-gear) Settings",
        description: "Configure Mind Palace extension",
      },
    ];

    const selection = await vscode.window.showQuickPick(items, {
      placeHolder: "Mind Palace Actions",
    });

    if (!selection) return;

    if (selection.label === "$(heart) Heal Context") {
      this.performHeal(false);
    } else if (selection.label === "$(search) Search Palace") {
      await vscode.commands.executeCommand("mindPalace.blueprintView.focus");
    } else if (selection.label === "$(layout-sidebar-left) Focus Blueprint") {
      vscode.commands.executeCommand("mindPalace.blueprintView.focus");
    } else if (selection.label === "$(file-code) Open Context Pack") {
      if (vscode.workspace.workspaceFolders?.[0]) {
        const uri = vscode.Uri.file(
          pathJoin(
            vscode.workspace.workspaceFolders[0].uri.fsPath,
            ".palace",
            "outputs",
            "context-pack.json"
          )
        );
        try {
          const doc = await vscode.workspace.openTextDocument(uri);
          await vscode.window.showTextDocument(doc);
        } catch (e) {
          vscode.window.showErrorMessage(
            "Could not open context-pack.json. Has it been generated?"
          );
        }
      }
    } else if (selection.label === "$(settings-gear) Settings") {
      vscode.commands.executeCommand(
        "workbench.action.openSettings",
        "mindPalace"
      );
    }
  }

  private async showFileIntel(filePath?: string): Promise<void> {
    const targetPath =
      filePath || vscode.window.activeTextEditor?.document.uri.fsPath;
    if (!targetPath) {
      vscode.window.showWarningMessage("No file selected");
      return;
    }

    const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
    if (!workspaceRoot) return;

    const relativePath = targetPath.startsWith(workspaceRoot)
      ? targetPath.substring(workspaceRoot.length + 1)
      : targetPath;

    try {
      const intel = await this.context.bridge.getFileIntel(relativePath);
      const items: vscode.QuickPickItem[] = [];

      items.push({
        label: `$(file) ${relativePath}`,
        description: "File Intelligence",
        detail: `Edits: ${intel.editCount} | Failures: ${intel.failureCount}`,
      });

      if (intel.learnings && intel.learnings.length > 0) {
        items.push({ label: "", kind: vscode.QuickPickItemKind.Separator });
        items.push({
          label: `$(lightbulb) Learnings (${intel.learnings.length})`,
          description: "",
        });
        intel.learnings.forEach((l: any) => {
          items.push({
            label: `    ${l.content.substring(0, 60)}${
              l.content.length > 60 ? "..." : ""
            }`,
            description: `${Math.round((l.confidence ?? 0.5) * 100)}%`,
          });
        });
      }

      if (intel.lastEdited) {
        items.push({ label: "", kind: vscode.QuickPickItemKind.Separator });
        items.push({
          label: `$(calendar) Last edited: ${intel.lastEdited}`,
          description: "",
        });
      }

      vscode.window.showQuickPick(items, {
        title: "Mind Palace - File Intelligence",
        placeHolder: relativePath,
      });
    } catch (err: any) {
      vscode.window.showErrorMessage(
        `Failed to get file intel: ${err.message}`
      );
    }
  }

  private async showCallGraph(args?: {
    symbol: string;
    file: string;
  }): Promise<void> {
    if (!args) {
      vscode.window.showWarningMessage("No symbol selected");
      return;
    }

    const { symbol, file } = args;
    const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
    if (!workspaceRoot) return;

    try {
      const [callers, callees] = await Promise.all([
        this.context.bridge
          .getCallers(symbol)
          .catch(() => ({ symbol, callers: [], callees: [] })),
        this.context.bridge
          .getCallees(symbol, file)
          .catch(() => ({ symbol, callers: [], callees: [] })),
      ]);

      const items: vscode.QuickPickItem[] = [];

      if (callers.callers && callers.callers.length > 0) {
        items.push({
          label: "Callers",
          kind: vscode.QuickPickItemKind.Separator,
        });
        callers.callers.forEach((c: any) => {
          items.push({
            label: `$(arrow-left) ${c.symbol}`,
            description: `${c.file}:${c.line}`,
            detail: "Calls this function",
          });
        });
      }

      if (callees.callees && callees.callees.length > 0) {
        items.push({
          label: "Callees",
          kind: vscode.QuickPickItemKind.Separator,
        });
        callees.callees.forEach((c: any) => {
          items.push({
            label: `$(arrow-right) ${c.symbol}`,
            description: `${c.file}:${c.line}`,
            detail: "Called by this function",
          });
        });
      }

      if (items.length === 0) {
        vscode.window.showInformationMessage(
          `No call graph data found for ${symbol}`
        );
        return;
      }

      const selection = await vscode.window.showQuickPick(items, {
        title: `Call Graph: ${symbol}`,
        placeHolder: "Select to navigate to definition",
      });

      if (selection && selection.description) {
        const [filePath, lineStr] = selection.description.split(":");
        const line = parseInt(lineStr, 10) - 1;
        const fullPath = filePath.startsWith("/")
          ? filePath
          : pathJoin(workspaceRoot, filePath);
        const uri = vscode.Uri.file(fullPath);
        const doc = await vscode.workspace.openTextDocument(uri);
        const editor = await vscode.window.showTextDocument(doc);
        const pos = new vscode.Position(line, 0);
        editor.selection = new vscode.Selection(pos, pos);
        editor.revealRange(
          new vscode.Range(pos, pos),
          vscode.TextEditorRevealType.InCenter
        );
      }
    } catch (err: any) {
      vscode.window.showErrorMessage(
        `Failed to get call graph: ${err.message}`
      );
    }
  }

  private async showKnowledgeDetail(item: {
    type: string;
    data: any;
  }): Promise<void> {
    if (!item) return;

    const { type, data } = item;
    const panel = vscode.window.createWebviewPanel(
      "mindPalaceKnowledgeDetail",
      `${type.charAt(0).toUpperCase() + type.slice(1)}: ${
        data.content?.substring(0, 30) ?? "Detail"
      }...`,
      vscode.ViewColumn.One,
      { enableScripts: false }
    );

    const scopeInfo = data.scopePath
      ? `<p><strong>Scope Path:</strong> ${data.scopePath}</p>`
      : "";
    const outcomeInfo = data.outcome
      ? `<p><strong>Outcome:</strong> ${data.outcome}${
          data.outcomeNote ? ` - ${data.outcomeNote}` : ""
        }</p>`
      : "";
    const confidenceInfo =
      type === "learning" && data.confidence !== undefined
        ? `<p><strong>Confidence:</strong> ${Math.round(
            data.confidence * 100
          )}%</p>`
        : "";
    const statusInfo = data.status
      ? `<p><strong>Status:</strong> ${data.status}</p>`
      : "";
    const tagsInfo = data.tags?.length
      ? `<p><strong>Tags:</strong> ${data.tags.join(", ")}</p>`
      : "";

    panel.webview.html = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>${type} Detail</title>
    <style>
        body { font-family: var(--vscode-font-family); padding: 20px; color: var(--vscode-foreground); background-color: var(--vscode-editor-background); }
        h1 { font-size: 1.5em; margin-bottom: 16px; color: var(--vscode-foreground); }
        .content { font-size: 1.1em; line-height: 1.6; margin-bottom: 24px; padding: 16px; background-color: var(--vscode-textBlockQuote-background); border-left: 4px solid var(--vscode-textLink-activeForeground); border-radius: 4px; }
        .meta { font-size: 0.9em; color: var(--vscode-descriptionForeground); }
        .meta p { margin: 8px 0; }
        .meta strong { color: var(--vscode-foreground); }
    </style>
</head>
<body>
    <h1>${type.charAt(0).toUpperCase() + type.slice(1)}</h1>
    <div class="content">${data.content}</div>
    <div class="meta">
        <p><strong>ID:</strong> ${data.id}</p>
        <p><strong>Scope:</strong> ${data.scope || "palace"}</p>
        ${scopeInfo}
        ${statusInfo}
        ${confidenceInfo}
        ${outcomeInfo}
        ${tagsInfo}
    </div>
</body>
</html>`;
  }

  private async startSession(): Promise<void> {
    const agentType = await vscode.window.showInputBox({
      prompt: "Enter agent type",
      placeHolder: "claude-code, cursor, aider, etc.",
      value: "claude-code",
    });

    if (!agentType) return;

    const goal = await vscode.window.showInputBox({
      prompt: "What is the goal of this session? (optional)",
      placeHolder: "e.g., Fix authentication bug",
    });

    try {
      const result = await this.context.bridge.startSession(agentType, goal);
      vscode.window.showInformationMessage(
        `Session started: ${result.sessionId}`
      );
      this.context.views.sessionProvider?.refresh();
    } catch (err: any) {
      vscode.window.showErrorMessage(`Failed to start session: ${err.message}`);
    }
  }

  private async endSession(sessionInfo?: any): Promise<void> {
    let sessionId = sessionInfo?.id;

    if (!sessionId) {
      sessionId = await vscode.window.showInputBox({
        prompt: "Enter session ID to end",
        placeHolder: "ses_...",
      });
    }

    if (!sessionId) return;

    const outcome = (await vscode.window.showQuickPick(
      ["success", "failure", "partial"],
      { placeHolder: "Select outcome" }
    )) as "success" | "failure" | "partial" | undefined;

    const summary = await vscode.window.showInputBox({
      prompt: "Brief summary of what was accomplished (optional)",
    });

    try {
      await this.context.bridge.endSession(sessionId, outcome, summary);
      vscode.window.showInformationMessage(`Session ended: ${sessionId}`);
      this.context.views.sessionProvider?.refresh();
    } catch (err: any) {
      vscode.window.showErrorMessage(`Failed to end session: ${err.message}`);
    }
  }

  private async showSessionDetail(session: any): Promise<void> {
    if (!session) return;

    const panel = vscode.window.createWebviewPanel(
      "mindPalaceSessionDetail",
      `Session: ${session.id.substring(0, 12)}...`,
      vscode.ViewColumn.One,
      { enableScripts: false }
    );

    const stateIcon =
      session.state === "active"
        ? "🟢"
        : session.state === "completed"
        ? "✅"
        : "❌";
    const goalInfo = session.goal
      ? `<p><strong>Goal:</strong> ${session.goal}</p>`
      : "";
    const summaryInfo = session.summary
      ? `<p><strong>Summary:</strong> ${session.summary}</p>`
      : "";

    panel.webview.html = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Session Detail</title>
    <style>
        body { font-family: var(--vscode-font-family); padding: 20px; color: var(--vscode-foreground); background-color: var(--vscode-editor-background); }
        h1 { font-size: 1.5em; margin-bottom: 16px; }
        .meta { font-size: 0.9em; color: var(--vscode-descriptionForeground); }
        .meta p { margin: 8px 0; }
        .meta strong { color: var(--vscode-foreground); }
        .status { display: inline-block; padding: 4px 8px; border-radius: 4px; background-color: var(--vscode-badge-background); color: var(--vscode-badge-foreground); }
    </style>
</head>
<body>
    <h1>${stateIcon} Session</h1>
    <div class="meta">
        <p><strong>ID:</strong> <code>${session.id}</code></p>
        <p><strong>Agent:</strong> ${session.agentType}</p>
        <p><strong>State:</strong> <span class="status">${
          session.state
        }</span></p>
        <p><strong>Started:</strong> ${new Date(
          session.startedAt
        ).toLocaleString()}</p>
        ${goalInfo}
        ${summaryInfo}
    </div>
</body>
</html>`;
  }

  private async showConflictInfo(): Promise<void> {
    const editor = vscode.window.activeTextEditor;
    if (!editor) {
      vscode.window.showInformationMessage("No file open");
      return;
    }

    const filePath = editor.document.uri.fsPath;
    const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
    if (!workspaceRoot) return;

    const relativePath = filePath.startsWith(workspaceRoot)
      ? filePath.substring(workspaceRoot.length + 1)
      : filePath;

    try {
      const result = await this.context.bridge.checkConflict(relativePath);

      if (result.conflict) {
        const action = await vscode.window.showWarningMessage(
          `${
            result.agent || "Another agent"
          } is also working on this file. Consider coordinating changes.`,
          "View Sessions",
          "OK"
        );

        if (action === "View Sessions") {
          vscode.commands.executeCommand("mindPalace.sessionsView.focus");
        }
      } else {
        vscode.window.showInformationMessage(
          "No conflicts detected on this file."
        );
      }
    } catch (err: any) {
      vscode.window.showErrorMessage(
        `Failed to check conflict: ${err.message}`
      );
    }
  }

  private async showCorridorLearningDetail(
    learning: CorridorLearning
  ): Promise<void> {
    if (!learning) return;

    const panel = vscode.window.createWebviewPanel(
      "mindPalaceCorridorLearning",
      `Learning: ${learning.content.substring(0, 30)}...`,
      vscode.ViewColumn.One,
      { enableScripts: false }
    );

    const tagsHtml =
      learning.tags && learning.tags.length > 0
        ? `<p><strong>Tags:</strong> ${learning.tags
            .map((t) => `<span class="tag">${t}</span>`)
            .join(" ")}</p>`
        : "";

    panel.webview.html = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Corridor Learning</title>
    <style>
        body { font-family: var(--vscode-font-family); padding: 20px; color: var(--vscode-foreground); background-color: var(--vscode-editor-background); }
        h1 { font-size: 1.5em; margin-bottom: 16px; }
        .meta { font-size: 0.9em; color: var(--vscode-descriptionForeground); }
        .meta p { margin: 8px 0; }
        .meta strong { color: var(--vscode-foreground); }
        .content { padding: 16px; background-color: var(--vscode-textCodeBlock-background); border-radius: 4px; margin: 16px 0; }
        .tag { display: inline-block; padding: 2px 8px; margin: 2px; border-radius: 12px; background-color: var(--vscode-badge-background); color: var(--vscode-badge-foreground); font-size: 0.85em; }
        .confidence { display: inline-block; padding: 4px 8px; border-radius: 4px; background-color: var(--vscode-badge-background); color: var(--vscode-badge-foreground); }
    </style>
</head>
<body>
    <h1>Personal Learning</h1>
    <div class="content">${learning.content}</div>
    <div class="meta">
        <p><strong>ID:</strong> ${learning.id}</p>
        <p><strong>Confidence:</strong> <span class="confidence">${Math.round(
          learning.confidence * 100
        )}%</span></p>
        <p><strong>Origin:</strong> ${learning.originWorkspace || "Unknown"}</p>
        <p><strong>Source:</strong> ${learning.source || "manual"}</p>
        <p><strong>Used:</strong> ${learning.useCount} times</p>
        <p><strong>Created:</strong> ${new Date(
          learning.createdAt
        ).toLocaleString()}</p>
        <p><strong>Last Used:</strong> ${new Date(
          learning.lastUsed
        ).toLocaleString()}</p>
        ${tagsHtml}
    </div>
</body>
</html>`;
  }

  private async reinforceLearning(learning?: CorridorLearning): Promise<void> {
    if (!learning) return;

    try {
      await this.context.bridge.reinforceCorridorLearning(learning.id);
      vscode.window.showInformationMessage(
        "Learning reinforced: confidence increased"
      );
      this.context.views.corridorProvider?.refresh();
    } catch (err: any) {
      vscode.window.showErrorMessage(`Failed to reinforce: ${err.message}`);
    }
  }

  private async searchConversations(): Promise<void> {
    const query = await vscode.window.showInputBox({
      prompt: "Search past conversations",
      placeHolder: "Enter search query (leave empty to list recent)",
    });

    try {
      const conversations = await this.context.bridge.searchConversations({
        query,
        limit: 20,
      });

      if (conversations.length === 0) {
        vscode.window.showInformationMessage("No conversations found");
        return;
      }

      const items = conversations.map((c) => ({
        label: `$(comment-discussion) ${c.summary}`,
        description: `${c.agentType} - ${c.messages.length} messages`,
        detail: new Date(c.createdAt).toLocaleString(),
        conversation: c,
      }));

      const selection = await vscode.window.showQuickPick(items, {
        placeHolder: "Select a conversation to view",
        title: "Past Conversations",
      });

      if (selection) {
        vscode.commands.executeCommand(
          "mindPalace.showConversationDetail",
          selection.conversation
        );
      }
    } catch (err: any) {
      vscode.window.showErrorMessage(`Failed to search: ${err.message}`);
    }
  }

  private async showConversationDetail(
    conversation: Conversation
  ): Promise<void> {
    if (!conversation) return;

    const panel = vscode.window.createWebviewPanel(
      "mindPalaceConversation",
      `Conversation: ${conversation.summary.substring(0, 30)}...`,
      vscode.ViewColumn.One,
      { enableScripts: false }
    );

    const messagesHtml = conversation.messages
      .map((m) => {
        const roleClass =
          m.role === "user"
            ? "user"
            : m.role === "assistant"
            ? "assistant"
            : "system";
        const roleLabel = m.role.charAt(0).toUpperCase() + m.role.slice(1);
        return `<div class="message ${roleClass}"><div class="role">${roleLabel}</div><div class="content">${escapeHtml(
          m.content
        )}</div></div>`;
      })
      .join("\n");

    panel.webview.html = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Conversation</title>
    <style>
        body { font-family: var(--vscode-font-family); padding: 20px; color: var(--vscode-foreground); background-color: var(--vscode-editor-background); }
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
    <div class="meta"><strong>ID:</strong> ${
      conversation.id
    } | <strong>Agent:</strong> ${
      conversation.agentType
    } | <strong>Messages:</strong> ${
      conversation.messages.length
    } | <strong>Created:</strong> ${new Date(
      conversation.createdAt
    ).toLocaleString()}</div>
    <div class="messages">${messagesHtml}</div>
</body>
</html>`;
  }

  private async showLinks(item?: {
    type: string;
    data: { id: string };
  }): Promise<void> {
    if (!item?.data?.id) {
      vscode.window.showWarningMessage("No record selected");
      return;
    }

    try {
      const links = await this.context.bridge.getLinks(item.data.id);

      if (links.length === 0) {
        vscode.window.showInformationMessage(
          `No links found for ${item.data.id}`
        );
        return;
      }

      const items = links.map((link: RecordLink) => ({
        label: `$(link) ${link.relation}`,
        description:
          link.sourceId === item.data.id
            ? `-> ${link.targetId}`
            : `<- ${link.sourceId}`,
        detail: `${link.sourceKind} ${link.relation} ${link.targetKind}`,
        link,
      }));

      const selection = await vscode.window.showQuickPick(items, {
        placeHolder: "Links for this record",
        title: `Links: ${item.data.id}`,
      });

      if (selection) {
        vscode.window.showInformationMessage(
          `Link: ${selection.link.sourceId} ${selection.link.relation} ${selection.link.targetId}`
        );
      }
    } catch (err: any) {
      vscode.window.showErrorMessage(`Failed to get links: ${err.message}`);
    }
  }

  private async createLink(item?: {
    type: string;
    data: { id: string };
  }): Promise<void> {
    if (!item?.data?.id) {
      vscode.window.showWarningMessage("No record selected");
      return;
    }

    const targetId = await vscode.window.showInputBox({
      prompt: "Enter the target record ID",
      placeHolder: "e.g., d_abc123, i_def456, l_ghi789",
    });

    if (!targetId) return;

    const relation = await vscode.window.showQuickPick(
      [
        { label: "supports", description: "This record supports the target" },
        {
          label: "contradicts",
          description: "This record contradicts the target",
        },
        {
          label: "implements",
          description: "This record implements the target",
        },
        {
          label: "supersedes",
          description: "This record supersedes the target",
        },
        {
          label: "inspired_by",
          description: "This record is inspired by the target",
        },
        {
          label: "related",
          description: "This record is related to the target",
        },
      ],
      { placeHolder: "Select relationship type" }
    );

    if (!relation) return;

    try {
      const linkId = await this.context.bridge.createLink(
        item.data.id,
        targetId,
        relation.label as RecordLink["relation"]
      );
      vscode.window.showInformationMessage(`Link created: ${linkId}`);
      this.context.views.knowledgeProvider?.refresh();
    } catch (err: any) {
      vscode.window.showErrorMessage(`Failed to create link: ${err.message}`);
    }
  }

  private async showLearningSuggestions(
    _filePath: string,
    suggestions: any[]
  ): Promise<void> {
    if (!suggestions || suggestions.length === 0) {
      vscode.window.showInformationMessage(
        "No relevant learnings found for this file"
      );
      return;
    }

    const items = suggestions.map((s: any, i: number) => ({
      label: `${i + 1}. ${s.content.substring(0, 60)}${
        s.content.length > 60 ? "..." : ""
      }`,
      description: `${Math.round(s.similarity * 100)}% relevant`,
      detail: `Created: ${new Date(s.createdAt).toLocaleDateString()}`,
      suggestion: s,
    }));

    const selection = await vscode.window.showQuickPick(items, {
      placeHolder: `${suggestions.length} relevant learnings for this file`,
      title: "Learning Suggestions",
    });

    if (selection) {
      vscode.commands.executeCommand(
        "mindPalace.showLearningDetail",
        selection.suggestion
      );
    }
  }

  private async showLearningDetail(learning: any): Promise<void> {
    if (!learning) return;

    const panel = vscode.window.createWebviewPanel(
      "mindPalaceLearningDetail",
      `Learning: ${learning.content?.substring(0, 30) ?? "Detail"}...`,
      vscode.ViewColumn.One,
      { enableScripts: false }
    );

    const confidenceInfo =
      learning.confidence !== undefined || learning.similarity !== undefined
        ? `<p><strong>Confidence:</strong> ${Math.round(
            (learning.confidence ?? learning.similarity ?? 0.5) * 100
          )}%</p>`
        : "";

    panel.webview.html = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Learning Detail</title>
    <style>
        body { font-family: var(--vscode-font-family); padding: 20px; color: var(--vscode-foreground); background-color: var(--vscode-editor-background); }
        h1 { font-size: 1.5em; margin-bottom: 16px; }
        .content { font-size: 1.1em; line-height: 1.6; margin-bottom: 24px; padding: 16px; background-color: var(--vscode-textBlockQuote-background); border-left: 4px solid var(--vscode-charts-green); border-radius: 4px; }
        .meta { font-size: 0.9em; color: var(--vscode-descriptionForeground); }
        .meta p { margin: 8px 0; }
        .meta strong { color: var(--vscode-foreground); }
    </style>
</head>
<body>
    <h1>Learning</h1>
    <div class="content">${escapeHtml(learning.content || "")}</div>
    <div class="meta">
        ${learning.id ? `<p><strong>ID:</strong> ${learning.id}</p>` : ""}
        ${confidenceInfo}
        ${
          learning.createdAt
            ? `<p><strong>Created:</strong> ${new Date(
                learning.createdAt
              ).toLocaleString()}</p>`
            : ""
        }
    </div>
</body>
</html>`;
  }

  dispose(): void {
    this.commands.forEach((cmd) => cmd.dispose());
    if (this.debounceTimer) clearTimeout(this.debounceTimer);
    if (this.countdownInterval) clearInterval(this.countdownInterval);
  }
}

function pathJoin(...parts: string[]): string {
  // Avoid importing 'path' directly in this module to keep it light
  return parts.join(pathSeparator());
}

function pathSeparator(): string {
  return process.platform === "win32" ? "\\" : "/";
}

function escapeHtml(text: string): string {
  return String(text)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}
