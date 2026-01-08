import * as vscode from "vscode";
import { PalaceBridge } from "../bridge";
import { KnowledgeTreeProvider } from "../providers/knowledgeTreeProvider";
import { SessionTreeProvider } from "../providers/sessionTreeProvider";
import { CorridorTreeProvider } from "../providers/corridorTreeProvider";
import { PalaceSidebarProvider } from "../sidebar";

export class ViewRegistry {
  private views: vscode.Disposable[] = [];
  private bridge: PalaceBridge;
  private extensionUri: vscode.Uri;

  public knowledgeProvider?: KnowledgeTreeProvider;
  public sessionProvider?: SessionTreeProvider;
  public corridorProvider?: CorridorTreeProvider;
  public sidebarProvider?: PalaceSidebarProvider;

  constructor(bridge: PalaceBridge, extensionUri: vscode.Uri) {
    this.bridge = bridge;
    this.extensionUri = extensionUri;
  }

  registerAll(): vscode.Disposable[] {
    this.registerSidebar();
    this.registerTreeViews();
    return this.views;
  }

  private registerSidebar(): void {
    this.sidebarProvider = new PalaceSidebarProvider(this.extensionUri);
    this.sidebarProvider.setBridge(this.bridge);
    this.views.push(
      vscode.window.registerWebviewViewProvider(
        PalaceSidebarProvider.viewType,
        this.sidebarProvider
      )
    );
  }

  private registerTreeViews(): void {
    this.knowledgeProvider = new KnowledgeTreeProvider();
    this.knowledgeProvider.setBridge(this.bridge);
    this.views.push(
      vscode.window.registerTreeDataProvider(
        "mindPalace.knowledgeView",
        this.knowledgeProvider
      )
    );

    this.sessionProvider = new SessionTreeProvider();
    this.sessionProvider.setBridge(this.bridge);
    this.views.push(
      vscode.window.registerTreeDataProvider(
        "mindPalace.sessionsView",
        this.sessionProvider
      )
    );

    this.corridorProvider = new CorridorTreeProvider();
    this.corridorProvider.setBridge(this.bridge);
    this.views.push(
      vscode.window.registerTreeDataProvider(
        "mindPalace.corridorView",
        this.corridorProvider
      )
    );
  }

  dispose(): void {
    this.views.forEach((v) => v.dispose());
  }
}
