import * as vscode from "vscode";
import { PalaceBridge } from "../bridge";
import { CallGraphHoverProvider } from "../providers/callGraphHoverProvider";
import { ConflictDetectionProvider } from "../providers/conflictDetectionProvider";
import { FileIntelligenceProvider } from "../providers/fileIntelligenceProvider";
import { InlineLearningDecorator } from "../providers/inlineLearningDecorator";
import { LearningSuggestionProvider } from "../providers/learningSuggestionProvider";
import { PalaceCodeLensProvider } from "../providers/palaceCodeLensProvider";

export class ProviderRegistry {
  private providers: vscode.Disposable[] = [];
  private bridge: PalaceBridge;
  private context: vscode.ExtensionContext;

  // Expose instances needed by other components
  public fileIntelProvider?: FileIntelligenceProvider;
  public inlineDecorator?: InlineLearningDecorator;

  constructor(bridge: PalaceBridge, context: vscode.ExtensionContext) {
    this.bridge = bridge;
    this.context = context;
  }

  registerAll(): vscode.Disposable[] {
    this.registerLanguageProviders();
    this.registerDecorators();
    return this.providers;
  }

  private registerLanguageProviders(): void {
    const codeLensProvider = new PalaceCodeLensProvider();
    codeLensProvider.setBridge(this.bridge);
    this.providers.push(
      vscode.languages.registerCodeLensProvider(
        { scheme: "file" },
        codeLensProvider
      )
    );

    const hoverProvider = new CallGraphHoverProvider();
    hoverProvider.setBridge(this.bridge);
    this.providers.push(
      vscode.languages.registerHoverProvider({ scheme: "file" }, hoverProvider)
    );

    const suggestionProvider = new LearningSuggestionProvider();
    suggestionProvider.setBridge(this.bridge);
    this.providers.push(
      vscode.languages.registerCodeLensProvider(
        { scheme: "file" },
        suggestionProvider
      )
    );
  }

  private registerDecorators(): void {
    this.fileIntelProvider = new FileIntelligenceProvider();
    this.fileIntelProvider.setBridge(this.bridge);
    this.providers.push(this.fileIntelProvider);

    const conflictProvider = new ConflictDetectionProvider();
    conflictProvider.setBridge(this.bridge);
    this.providers.push(conflictProvider);

    this.inlineDecorator = new InlineLearningDecorator();
    this.inlineDecorator.setBridge(this.bridge);
    this.inlineDecorator.activate(this.context);
    this.providers.push(this.inlineDecorator);
  }

  dispose(): void {
    this.providers.forEach((p) => p.dispose());
  }
}
