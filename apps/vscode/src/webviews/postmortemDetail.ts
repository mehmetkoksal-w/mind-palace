import * as vscode from "vscode";
import { PalaceBridge } from "../bridge";
import { logger } from "../logger";

/**
 * Postmortem detail webview panel
 * Displays comprehensive information about a postmortem
 */
export class PostmortemDetailPanel {
  public static currentPanel: PostmortemDetailPanel | undefined;
  private static readonly viewType = "mindPalacePostmortemDetail";

  private readonly _panel: vscode.WebviewPanel;
  private readonly _extensionUri: vscode.Uri;
  private _disposables: vscode.Disposable[] = [];
  private _postmortem: any;
  private _bridge: PalaceBridge;

  /**
   * Create or show the postmortem detail panel
   */
  public static async createOrShow(
    extensionUri: vscode.Uri,
    bridge: PalaceBridge,
    postmortemId: string
  ): Promise<void> {
    const column = vscode.window.activeTextEditor
      ? vscode.window.activeTextEditor.viewColumn
      : undefined;

    // If we already have a panel, show it
    if (PostmortemDetailPanel.currentPanel) {
      PostmortemDetailPanel.currentPanel._panel.reveal(column);
      await PostmortemDetailPanel.currentPanel.loadPostmortem(postmortemId);
      return;
    }

    // Otherwise, create a new panel
    const panel = vscode.window.createWebviewPanel(
      PostmortemDetailPanel.viewType,
      "Postmortem",
      column || vscode.ViewColumn.One,
      {
        enableScripts: true,
        retainContextWhenHidden: true,
      }
    );

    PostmortemDetailPanel.currentPanel = new PostmortemDetailPanel(
      panel,
      extensionUri,
      bridge
    );
    await PostmortemDetailPanel.currentPanel.loadPostmortem(postmortemId);
  }

  private constructor(
    panel: vscode.WebviewPanel,
    extensionUri: vscode.Uri,
    bridge: PalaceBridge
  ) {
    this._panel = panel;
    this._extensionUri = extensionUri;
    this._bridge = bridge;

    // Set initial content
    this._panel.webview.html = this._getLoadingHtml();

    // Listen for when the panel is disposed
    this._panel.onDidDispose(() => this.dispose(), null, this._disposables);

    // Handle messages from the webview
    this._panel.webview.onDidReceiveMessage(
      async (message) => {
        switch (message.command) {
          case "resolve":
            await this.resolvePostmortem();
            break;
          case "convertToLearnings":
            await this.convertToLearnings();
            break;
          case "openFile":
            await this.openFile(message.path);
            break;
        }
      },
      null,
      this._disposables
    );
  }

  /**
   * Load postmortem data from backend
   */
  private async loadPostmortem(postmortemId: string): Promise<void> {
    try {
      logger.info("Loading postmortem detail", "PostmortemDetailPanel", {
        postmortemId,
      });

      const text = await this._bridge.getPostmortem(postmortemId);

      // For now, we'll parse the text response
      // In a production system, you might want the backend to return structured JSON
      this._postmortem = this._parsePostmortemResponse(text, postmortemId);

      // Update panel title and content
      this._panel.title = `Postmortem: ${this._postmortem.title}`;
      this._panel.webview.html = this._getHtmlForWebview();

      logger.info("Postmortem loaded successfully", "PostmortemDetailPanel");
    } catch (error: any) {
      logger.error("Failed to load postmortem", error, "PostmortemDetailPanel");
      this._panel.webview.html = this._getErrorHtml(error.message);
    }
  }

  /**
   * Parse the text response from get_postmortem
   * This is a temporary solution - ideally backend would return JSON
   */
  private _parsePostmortemResponse(text: string, id: string): any {
    // Basic parsing - extract key fields
    const lines = text.split("\n");
    const postmortem: any = {
      id,
      title: "",
      whatHappened: "",
      rootCause: "",
      severity: "medium",
      status: "open",
      lessonsLearned: [],
      preventionSteps: [],
      affectedFiles: [],
      createdAt: new Date().toISOString(),
    };

    let currentSection = "";
    for (const line of lines) {
      if (line.startsWith("Title:")) {
        postmortem.title = line.substring(6).trim();
      } else if (line.startsWith("Severity:")) {
        postmortem.severity = line.substring(9).trim().toLowerCase();
      } else if (line.startsWith("Status:")) {
        postmortem.status = line.substring(7).trim().toLowerCase();
      } else if (line.startsWith("Created:")) {
        postmortem.createdAt = line.substring(8).trim();
      } else if (line.includes("What happened:")) {
        currentSection = "whatHappened";
      } else if (line.includes("Root cause:")) {
        currentSection = "rootCause";
      } else if (line.includes("Lessons learned:")) {
        currentSection = "lessons";
      } else if (line.includes("Prevention steps:")) {
        currentSection = "prevention";
      } else if (line.startsWith("  - ")) {
        const content = line.substring(4).trim();
        if (currentSection === "lessons") {
          postmortem.lessonsLearned.push(content);
        } else if (currentSection === "prevention") {
          postmortem.preventionSteps.push(content);
        }
      } else if (line.trim() && currentSection === "whatHappened") {
        postmortem.whatHappened += line.trim() + " ";
      } else if (line.trim() && currentSection === "rootCause") {
        postmortem.rootCause += line.trim() + " ";
      }
    }

    return postmortem;
  }

  /**
   * Resolve the postmortem
   */
  private async resolvePostmortem(): Promise<void> {
    try {
      await this._bridge.resolvePostmortem(this._postmortem.id);
      this._postmortem.status = "resolved";
      this._panel.webview.html = this._getHtmlForWebview();
      vscode.window.showInformationMessage("Postmortem marked as resolved");
      vscode.commands.executeCommand("mindPalace.refreshKnowledge");
    } catch (error: any) {
      vscode.window.showErrorMessage(`Failed to resolve: ${error.message}`);
    }
  }

  /**
   * Convert postmortem lessons to learnings
   */
  private async convertToLearnings(): Promise<void> {
    try {
      await this._bridge.postmortemToLearnings(this._postmortem.id);
      vscode.window.showInformationMessage("Lessons converted to learnings");
      vscode.commands.executeCommand("mindPalace.refreshKnowledge");
    } catch (error: any) {
      vscode.window.showErrorMessage(`Failed to convert: ${error.message}`);
    }
  }

  /**
   * Open a file from affected files list
   */
  private async openFile(path: string): Promise<void> {
    try {
      const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
      if (!workspaceRoot) {
        return;
      }

      const fullPath = vscode.Uri.file(`${workspaceRoot}/${path}`);
      const doc = await vscode.workspace.openTextDocument(fullPath);
      await vscode.window.showTextDocument(doc);
    } catch (error: any) {
      vscode.window.showErrorMessage(`Could not open file: ${error.message}`);
    }
  }

  /**
   * Generate HTML for the webview
   */
  private _getHtmlForWebview(): string {
    const pm = this._postmortem;
    const severityColor = this._getSeverityColor(pm.severity);
    const severityIcon = this._getSeverityIcon(pm.severity);
    const statusIcon =
      pm.status === "resolved" ? "✅" : pm.status === "recurring" ? "🔄" : "🔴";

    const lessonsHtml =
      pm.lessonsLearned.length > 0
        ? pm.lessonsLearned
            .map((lesson: string) => `<li>${this._escapeHtml(lesson)}</li>`)
            .join("")
        : '<li class="empty">No lessons recorded</li>';

    const preventionHtml =
      pm.preventionSteps.length > 0
        ? pm.preventionSteps
            .map((step: string) => `<li>${this._escapeHtml(step)}</li>`)
            .join("")
        : '<li class="empty">No prevention steps recorded</li>';

    const affectedFilesHtml =
      pm.affectedFiles.length > 0
        ? pm.affectedFiles
            .map(
              (file: string) =>
                `<li><a href="#" class="file-link" data-path="${this._escapeHtml(
                  file
                )}">${this._escapeHtml(file)}</a></li>`
            )
            .join("")
        : '<li class="empty">No affected files recorded</li>';

    const resolveButton =
      pm.status !== "resolved"
        ? `<button class="button primary" onclick="resolvePostmortem()">Mark as Resolved</button>`
        : '<span class="resolved-badge">✅ Resolved</span>';

    return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Postmortem: ${this._escapeHtml(pm.title)}</title>
    <style>
        * {
            box-sizing: border-box;
        }
        body {
            font-family: var(--vscode-font-family);
            font-size: var(--vscode-font-size);
            color: var(--vscode-foreground);
            background-color: var(--vscode-editor-background);
            padding: 0;
            margin: 0;
            line-height: 1.6;
        }
        .container {
            max-width: 900px;
            margin: 0 auto;
            padding: 24px;
        }
        .header {
            border-bottom: 2px solid var(--vscode-panel-border);
            padding-bottom: 16px;
            margin-bottom: 24px;
        }
        .title {
            font-size: 28px;
            font-weight: 600;
            margin: 0 0 12px 0;
            display: flex;
            align-items: center;
            gap: 12px;
        }
        .severity-badge {
            display: inline-flex;
            align-items: center;
            gap: 6px;
            padding: 4px 12px;
            border-radius: 4px;
            font-size: 14px;
            font-weight: 500;
            background-color: ${severityColor}33;
            color: ${severityColor};
            border: 1px solid ${severityColor}66;
        }
        .metadata {
            display: flex;
            gap: 24px;
            color: var(--vscode-descriptionForeground);
            font-size: 13px;
            margin-top: 8px;
        }
        .metadata-item {
            display: flex;
            align-items: center;
            gap: 6px;
        }
        .section {
            margin: 24px 0;
        }
        .section-title {
            font-size: 18px;
            font-weight: 600;
            margin-bottom: 12px;
            color: var(--vscode-foreground);
        }
        .section-content {
            padding: 16px;
            background-color: var(--vscode-editor-inactiveSelectionBackground);
            border-radius: 6px;
            border: 1px solid var(--vscode-panel-border);
        }
        ul {
            margin: 0;
            padding-left: 24px;
        }
        li {
            margin: 8px 0;
        }
        li.empty {
            color: var(--vscode-descriptionForeground);
            font-style: italic;
            list-style: none;
            margin-left: -24px;
        }
        .file-link {
            color: var(--vscode-textLink-foreground);
            text-decoration: none;
            cursor: pointer;
        }
        .file-link:hover {
            text-decoration: underline;
        }
        .actions {
            margin-top: 32px;
            padding-top: 24px;
            border-top: 1px solid var(--vscode-panel-border);
            display: flex;
            gap: 12px;
            align-items: center;
        }
        .button {
            padding: 8px 16px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-family: var(--vscode-font-family);
            font-size: 13px;
            font-weight: 500;
            transition: opacity 0.2s;
        }
        .button:hover {
            opacity: 0.8;
        }
        .button.primary {
            background-color: var(--vscode-button-background);
            color: var(--vscode-button-foreground);
        }
        .button.secondary {
            background-color: var(--vscode-button-secondaryBackground);
            color: var(--vscode-button-secondaryForeground);
        }
        .resolved-badge {
            padding: 8px 16px;
            background-color: var(--vscode-testing-iconPassed);
            color: white;
            border-radius: 4px;
            font-weight: 500;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1 class="title">
                ${statusIcon} ${this._escapeHtml(pm.title)}
            </h1>
            <div>
                <span class="severity-badge">${severityIcon} ${pm.severity.toUpperCase()}</span>
            </div>
            <div class="metadata">
                <div class="metadata-item">
                    <span>📅</span>
                    <span>${new Date(pm.createdAt).toLocaleString()}</span>
                </div>
                <div class="metadata-item">
                    <span>🆔</span>
                    <code>${pm.id}</code>
                </div>
                <div class="metadata-item">
                    <span>📊</span>
                    <span>${
                      pm.status.charAt(0).toUpperCase() + pm.status.slice(1)
                    }</span>
                </div>
            </div>
        </div>

        <div class="section">
            <h2 class="section-title">What Happened</h2>
            <div class="section-content">
                ${this._escapeHtml(pm.whatHappened)}
            </div>
        </div>

        ${
          pm.rootCause
            ? `
        <div class="section">
            <h2 class="section-title">Root Cause</h2>
            <div class="section-content">
                ${this._escapeHtml(pm.rootCause)}
            </div>
        </div>
        `
            : ""
        }

        <div class="section">
            <h2 class="section-title">Lessons Learned</h2>
            <div class="section-content">
                <ul>${lessonsHtml}</ul>
            </div>
        </div>

        <div class="section">
            <h2 class="section-title">Prevention Steps</h2>
            <div class="section-content">
                <ul>${preventionHtml}</ul>
            </div>
        </div>

        ${
          pm.affectedFiles.length > 0
            ? `
        <div class="section">
            <h2 class="section-title">Affected Files</h2>
            <div class="section-content">
                <ul>${affectedFilesHtml}</ul>
            </div>
        </div>
        `
            : ""
        }

        <div class="actions">
            ${resolveButton}
            <button class="button secondary" onclick="convertToLearnings()">
                Convert Lessons to Learnings
            </button>
        </div>
    </div>

    <script>
        const vscode = acquireVsCodeApi();

        function resolvePostmortem() {
            vscode.postMessage({ command: 'resolve' });
        }

        function convertToLearnings() {
            vscode.postMessage({ command: 'convertToLearnings' });
        }

        // Handle file link clicks
        document.querySelectorAll('.file-link').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const path = e.target.getAttribute('data-path');
                vscode.postMessage({ command: 'openFile', path });
            });
        });
    </script>
</body>
</html>`;
  }

  private _getLoadingHtml(): string {
    return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <style>
        body {
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            font-family: var(--vscode-font-family);
            color: var(--vscode-foreground);
        }
    </style>
</head>
<body>
    <div>Loading postmortem...</div>
</body>
</html>`;
  }

  private _getErrorHtml(message: string): string {
    return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <style>
        body {
            font-family: var(--vscode-font-family);
            color: var(--vscode-errorForeground);
            padding: 20px;
        }
    </style>
</head>
<body>
    <h2>Error Loading Postmortem</h2>
    <p>${this._escapeHtml(message)}</p>
</body>
</html>`;
  }

  private _getSeverityColor(severity: string): string {
    switch (severity) {
      case "critical":
        return "#dc3545";
      case "high":
        return "#fd7e14";
      case "medium":
        return "#ffc107";
      case "low":
        return "#28a745";
      default:
        return "#6c757d";
    }
  }

  private _getSeverityIcon(severity: string): string {
    switch (severity) {
      case "critical":
        return "🔥";
      case "high":
        return "⚠️";
      case "medium":
        return "⚡";
      case "low":
        return "ℹ️";
      default:
        return "📌";
    }
  }

  private _escapeHtml(text: string): string {
    return text
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#039;");
  }

  public dispose(): void {
    PostmortemDetailPanel.currentPanel = undefined;

    this._panel.dispose();

    while (this._disposables.length) {
      const disposable = this._disposables.pop();
      if (disposable) {
        disposable.dispose();
      }
    }
  }
}
