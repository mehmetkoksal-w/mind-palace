import * as vscode from "vscode";
import * as fs from "fs";
import * as path from "path";
import { logger } from "./services/logger";

interface Finding {
  file: string;
  detail: string;
}

interface ContextPack {
  findings: Finding[];
}

export class PalaceDecorator {
  private decorationType: vscode.TextEditorDecorationType;
  private disposables: vscode.Disposable[] = [];

  constructor() {
    this.decorationType = vscode.window.createTextEditorDecorationType({
      borderWidth: "1px",
      borderStyle: "solid",
      borderColor: "blue",
      overviewRulerColor: "blue",
      overviewRulerLane: vscode.OverviewRulerLane.Right,
    });
  }

  activate(context: vscode.ExtensionContext) {
    const updateActive = (editor?: vscode.TextEditor) => {
      if (editor) {
        this.updateDecorations(editor);
      }
    };

    this.disposables.push(
      vscode.window.onDidChangeActiveTextEditor(updateActive)
    );

    let debounceTimer: NodeJS.Timeout | undefined;
    this.disposables.push(
      vscode.workspace.onDidChangeTextDocument((event) => {
        const editor = vscode.window.activeTextEditor;
        if (editor && event.document === editor.document) {
          if (debounceTimer) clearTimeout(debounceTimer);
          debounceTimer = setTimeout(() => this.updateDecorations(editor), 300);
        }
      })
    );

    if (vscode.window.activeTextEditor) {
      this.updateDecorations(vscode.window.activeTextEditor);
    }

    context.subscriptions.push(this);
    context.subscriptions.push(...this.disposables);
  }

  updateDecorations(editor: vscode.TextEditor) {
    if (!editor || !vscode.workspace.rootPath) {
      return;
    }

    const contextPackPath = path.join(
      vscode.workspace.rootPath,
      ".palace",
      "outputs",
      "context-pack.json"
    );

    if (!fs.existsSync(contextPackPath)) {
      return; // No context pack found
    }

    try {
      const content = fs.readFileSync(contextPackPath, "utf8");
      const data: ContextPack = JSON.parse(content);
      const relativePath = vscode.workspace.asRelativePath(
        editor.document.fileName
      );

      const relevantFindings = data.findings.filter(
        (f) => f.file === relativePath || relativePath.endsWith(f.file)
      );

      const ranges: vscode.Range[] = [];

      for (const finding of relevantFindings) {
        const lineMatch = finding.detail.match(/lines? (\d+)-(\d+)/i);
        if (lineMatch) {
          const startLine = parseInt(lineMatch[1]) - 1; // 0-indexed
          const endLine = parseInt(lineMatch[2]) - 1;

          // Basic validation
          if (startLine >= 0 && endLine >= startLine) {
            const range = new vscode.Range(
              startLine,
              0,
              endLine,
              Number.MAX_VALUE
            );
            ranges.push(range);
          }
        }
      }

      editor.setDecorations(this.decorationType, ranges);
    } catch (e) {
      logger.error("Failed to parse context-pack.json", e, "PalaceDecorator");
    }
  }

  dispose(): void {
    this.decorationType.dispose();
    this.disposables.forEach((d) => d.dispose());
  }
}
