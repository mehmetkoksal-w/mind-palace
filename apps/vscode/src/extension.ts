import * as vscode from "vscode";
import { PalaceBridge } from "./bridge";
import { watchProjectConfig } from "./config";
import { PalaceDecorator } from "./decorator";
import { PalaceHUD } from "./hud";
import { CommandRegistry } from "./core/command-registry";
import { ProviderRegistry } from "./core/provider-registry";
import { ViewRegistry } from "./core/view-registry";
import { EventBus } from "./core/event-bus";
import { warnIfIncompatible } from "./version";

export function activate(context: vscode.ExtensionContext) {
  warnIfIncompatible();

  const bridge = new PalaceBridge();
  const hud = new PalaceHUD();
  const decorator = new PalaceDecorator();
  decorator.activate(context);

  // Initialize registries
  const viewRegistry = new ViewRegistry(bridge, context.extensionUri);
  const providerRegistry = new ProviderRegistry(bridge, context);
  const commandRegistry = new CommandRegistry({
    bridge,
    hud,
    decorator,
    extensionContext: context,
    views: viewRegistry,
  });
  const eventBus = new EventBus();

  // Watch project config
  const configWatcher = watchProjectConfig(() => commandRegistry.checkStatus());
  context.subscriptions.push(configWatcher);

  // Cleanup bridge and HUD on deactivation
  context.subscriptions.push({
    dispose: () => {
      bridge.dispose();
      hud.dispose();
    },
  });

  // Register all views (sidebar, trees)
  context.subscriptions.push(...viewRegistry.registerAll());

  // Register all providers (CodeLens, Hover, FileIntel, Conflict, Learning, Inline decorators)
  context.subscriptions.push(...providerRegistry.registerAll());

  // Register all commands (heal, sessions, knowledge, corridor, conversations, links, etc.)
  context.subscriptions.push(...commandRegistry.registerAll());

  // Register all event listeners (save, editor change, workspace changes, config changes)
  context.subscriptions.push(...eventBus.registerAll(context));

  // Perform initial status check
  commandRegistry.checkStatus();
}

export function deactivate() {}
