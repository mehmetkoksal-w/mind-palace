import * as vscode from "vscode";
import { PalaceBridge, RecallResult, RecordLink } from "../../bridge";

/**
 * Graph node for visualization
 */
interface GraphNode {
  id: string;
  kind: "idea" | "decision" | "learning";
  content: string;
  confidence?: number;
  status?: string;
  links: number;
}

/**
 * Graph link for visualization
 */
interface GraphLink {
  source: string;
  target: string;
  relation: string;
}

/**
 * KnowledgeGraphPanel provides a D3.js visualization of the knowledge graph
 * showing relationships between ideas, decisions, and learnings.
 */
export class KnowledgeGraphPanel {
  public static currentPanel: KnowledgeGraphPanel | undefined;
  private static readonly viewType = "mindPalace.knowledgeGraph";

  private readonly panel: vscode.WebviewPanel;
  private readonly extensionUri: vscode.Uri;
  private bridge?: PalaceBridge;
  private disposables: vscode.Disposable[] = [];
  private currentFile?: string;

  private constructor(panel: vscode.WebviewPanel, extensionUri: vscode.Uri) {
    this.panel = panel;
    this.extensionUri = extensionUri;

    // Handle panel disposal
    this.panel.onDidDispose(() => this.dispose(), null, this.disposables);

    // Handle messages from the webview
    this.panel.webview.onDidReceiveMessage(
      (message) => this.handleMessage(message),
      null,
      this.disposables
    );

    // Update content
    this.updateContent();
  }

  /**
   * Create or show the knowledge graph panel
   */
  public static createOrShow(
    extensionUri: vscode.Uri,
    bridge: PalaceBridge,
    file?: string
  ): void {
    const column = vscode.window.activeTextEditor
      ? vscode.window.activeTextEditor.viewColumn
      : undefined;

    // If we already have a panel, show it
    if (KnowledgeGraphPanel.currentPanel) {
      KnowledgeGraphPanel.currentPanel.panel.reveal(column);
      KnowledgeGraphPanel.currentPanel.bridge = bridge;
      KnowledgeGraphPanel.currentPanel.currentFile = file;
      KnowledgeGraphPanel.currentPanel.loadData();
      return;
    }

    // Create a new panel
    const panel = vscode.window.createWebviewPanel(
      KnowledgeGraphPanel.viewType,
      "Knowledge Graph",
      column || vscode.ViewColumn.Two,
      {
        enableScripts: true,
        retainContextWhenHidden: true,
        localResourceRoots: [extensionUri],
      }
    );

    KnowledgeGraphPanel.currentPanel = new KnowledgeGraphPanel(
      panel,
      extensionUri
    );
    KnowledgeGraphPanel.currentPanel.bridge = bridge;
    KnowledgeGraphPanel.currentPanel.currentFile = file;
    KnowledgeGraphPanel.currentPanel.loadData();
  }

  /**
   * Set the bridge for MCP communication
   */
  setBridge(bridge: PalaceBridge): void {
    this.bridge = bridge;
  }

  /**
   * Load knowledge data and send to webview
   */
  async loadData(): Promise<void> {
    if (!this.bridge) {
      this.sendError("Bridge not connected");
      return;
    }

    try {
      const nodes: GraphNode[] = [];
      const links: GraphLink[] = [];
      const nodeMap = new Map<string, GraphNode>();

      // Get learnings
      const learningsResult = await this.bridge.recallLearnings({ limit: 50 });
      if (learningsResult.learnings) {
        for (const l of learningsResult.learnings) {
          const node: GraphNode = {
            id: l.id,
            kind: "learning",
            content: l.content,
            confidence: l.confidence,
            links: 0,
          };
          nodes.push(node);
          nodeMap.set(l.id, node);
        }
      }

      // Get decisions
      const decisionsResult = await this.bridge.recallDecisions({ limit: 50 });
      if (decisionsResult.decisions) {
        for (const d of decisionsResult.decisions) {
          const node: GraphNode = {
            id: d.id,
            kind: "decision",
            content: d.content,
            status: d.status,
            links: 0,
          };
          nodes.push(node);
          nodeMap.set(d.id, node);
        }
      }

      // Get ideas
      const ideasResult = await this.bridge.recallIdeas({ limit: 50 });
      if (ideasResult.ideas) {
        for (const i of ideasResult.ideas) {
          const node: GraphNode = {
            id: i.id,
            kind: "idea",
            content: i.content,
            status: i.status,
            links: 0,
          };
          nodes.push(node);
          nodeMap.set(i.id, node);
        }
      }

      // Get links for all nodes
      for (const node of nodes) {
        try {
          const nodeLinks = await this.bridge.getLinks(node.id);
          node.links = nodeLinks.length;

          for (const link of nodeLinks) {
            // Only add links where both nodes exist
            if (nodeMap.has(link.sourceId) && nodeMap.has(link.targetId)) {
              // Avoid duplicates
              const exists = links.some(
                (l) =>
                  (l.source === link.sourceId && l.target === link.targetId) ||
                  (l.source === link.targetId && l.target === link.sourceId)
              );
              if (!exists) {
                links.push({
                  source: link.sourceId,
                  target: link.targetId,
                  relation: link.relation,
                });
              }
            }
          }
        } catch {
          // Ignore link errors for individual nodes
        }
      }

      // Send data to webview
      this.panel.webview.postMessage({
        type: "data",
        nodes,
        links,
        currentFile: this.currentFile,
      });
    } catch (error: any) {
      this.sendError(error.message || "Failed to load knowledge data");
    }
  }

  /**
   * Send error message to webview
   */
  private sendError(message: string): void {
    this.panel.webview.postMessage({
      type: "error",
      message,
    });
  }

  /**
   * Handle messages from the webview
   */
  private async handleMessage(message: any): Promise<void> {
    switch (message.type) {
      case "refresh":
        await this.loadData();
        break;
      case "showDetail":
        vscode.commands.executeCommand(
          "mindPalace.showLearningDetail",
          message.node
        );
        break;
      case "reinforce":
        if (this.bridge && message.nodeId) {
          try {
            await this.bridge.reinforceCorridorLearning(message.nodeId);
            vscode.window.showInformationMessage("Learning reinforced");
            await this.loadData();
          } catch (error: any) {
            vscode.window.showErrorMessage(
              `Failed to reinforce: ${error.message}`
            );
          }
        }
        break;
    }
  }

  /**
   * Update the webview content
   */
  private updateContent(): void {
    this.panel.webview.html = this.getHtmlContent();
  }

  /**
   * Get the HTML content for the webview
   */
  private getHtmlContent(): string {
    const nonce = getNonce();

    // Get bundled script URI
    const scriptUri = this.panel.webview.asWebviewUri(
      vscode.Uri.joinPath(
        this.extensionUri,
        "out",
        "webviews",
        "knowledge-graph.js"
      )
    );

    return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'nonce-${nonce}'; style-src 'unsafe-inline'; img-src data:;">
    <title>Knowledge Graph</title>
    <style>
        body {
            margin: 0;
            padding: 0;
            overflow: hidden;
            background: var(--vscode-editor-background);
            color: var(--vscode-foreground);
            font-family: var(--vscode-font-family);
        }

        #container {
            width: 100vw;
            height: 100vh;
            display: flex;
            flex-direction: column;
        }

        #toolbar {
            padding: 8px 16px;
            background: var(--vscode-titleBar-activeBackground);
            border-bottom: 1px solid var(--vscode-panel-border);
            display: flex;
            gap: 8px;
            align-items: center;
        }

        #toolbar button {
            background: var(--vscode-button-background);
            color: var(--vscode-button-foreground);
            border: none;
            padding: 4px 12px;
            border-radius: 4px;
            cursor: pointer;
        }

        #toolbar button:hover {
            background: var(--vscode-button-hoverBackground);
        }

        #legend {
            display: flex;
            gap: 16px;
            margin-left: auto;
            font-size: 12px;
        }

        .legend-item {
            display: flex;
            align-items: center;
            gap: 4px;
        }

        .legend-dot {
            width: 12px;
            height: 12px;
            border-radius: 50%;
        }

        .legend-dot.idea { background: #f59e0b; }
        .legend-dot.decision { background: #3b82f6; }
        .legend-dot.learning { background: #10b981; }

        #graph {
            flex: 1;
            overflow: hidden;
        }

        svg {
            width: 100%;
            height: 100%;
        }

        .node {
            cursor: pointer;
        }

        .node circle {
            stroke-width: 2px;
        }

        .node.idea circle { fill: #f59e0b; stroke: #d97706; }
        .node.decision circle { fill: #3b82f6; stroke: #2563eb; }
        .node.learning circle { fill: #10b981; stroke: #059669; }

        .node text {
            font-size: 10px;
            fill: var(--vscode-foreground);
            pointer-events: none;
        }

        .link {
            stroke: var(--vscode-editorWidget-border);
            stroke-opacity: 0.6;
            fill: none;
        }

        .link.supports { stroke: #10b981; }
        .link.contradicts { stroke: #ef4444; stroke-dasharray: 4,2; }
        .link.implements { stroke: #3b82f6; }
        .link.related { stroke: #8b5cf6; }

        #tooltip {
            position: absolute;
            padding: 8px 12px;
            background: var(--vscode-editorWidget-background);
            border: 1px solid var(--vscode-editorWidget-border);
            border-radius: 4px;
            font-size: 12px;
            max-width: 300px;
            pointer-events: none;
            display: none;
            z-index: 1000;
        }

        #tooltip h4 {
            margin: 0 0 4px 0;
            font-size: 11px;
            text-transform: uppercase;
            color: var(--vscode-descriptionForeground);
        }

        #tooltip p {
            margin: 0;
            word-wrap: break-word;
        }

        .loading {
            display: flex;
            align-items: center;
            justify-content: center;
            height: 100%;
            font-size: 14px;
            color: var(--vscode-descriptionForeground);
        }

        .error {
            color: var(--vscode-errorForeground);
            padding: 20px;
            text-align: center;
        }
    </style>
</head>
<body>
    <div id="container">
        <div id="toolbar">
            <button id="refresh">Refresh</button>
            <button id="zoomIn">Zoom In</button>
            <button id="zoomOut">Zoom Out</button>
            <button id="reset">Reset View</button>
            <div id="legend">
                <div class="legend-item">
                    <div class="legend-dot idea"></div>
                    <span>Idea</span>
                </div>
                <div class="legend-item">
                    <div class="legend-dot decision"></div>
                    <span>Decision</span>
                </div>
                <div class="legend-item">
                    <div class="legend-dot learning"></div>
                    <span>Learning</span>
                </div>
            </div>
        </div>
        <div id="graph">
            <div class="loading">Loading knowledge graph...</div>
        </div>
        <div id="tooltip"></div>
    </div>

    <!-- Bundled D3.js script -->
    <script nonce="${nonce}" src="${scriptUri}"></script>
</body>
</html>`;
  }

  /**
   * Dispose the panel
   */
  dispose(): void {
    KnowledgeGraphPanel.currentPanel = undefined;

    this.panel.dispose();

    while (this.disposables.length) {
      const disposable = this.disposables.pop();
      if (disposable) {
        disposable.dispose();
      }
    }
  }
}

/**
 * Generate a nonce for CSP
 */
function getNonce(): string {
  let text = "";
  const possible =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  for (let i = 0; i < 32; i++) {
    text += possible.charAt(Math.floor(Math.random() * possible.length));
  }
  return text;
}
