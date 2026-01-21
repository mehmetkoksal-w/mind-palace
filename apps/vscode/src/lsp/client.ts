import * as vscode from "vscode";
import * as path from "path";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
  State,
  ErrorAction,
  CloseAction,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;
let restartCount = 0;
const MAX_RESTART_ATTEMPTS = 3;
const RESTART_DELAY_MS = 2000;
let restartTimeout: NodeJS.Timeout | undefined;

/**
 * Activates the Mind Palace LSP client.
 * Starts the palace lsp server and connects the VS Code language client.
 */
export async function activateLspClient(
  context: vscode.ExtensionContext,
): Promise<LanguageClient | undefined> {
  const config = vscode.workspace.getConfiguration("mindPalace");
  const lspEnabled = config.get<boolean>("lsp.enabled", true);

  if (!lspEnabled) {
    return undefined;
  }

  const binaryPath = config.get<string>("binaryPath", "palace");
  const workspaceRoot = getWorkspaceRoot();

  if (!workspaceRoot) {
    vscode.window.showWarningMessage(
      "Mind Palace LSP: No workspace folder open",
    );
    return undefined;
  }

  // Server options - start palace lsp
  const serverOptions: ServerOptions = {
    command: binaryPath,
    args: ["lsp", "--root", workspaceRoot],
    transport: TransportKind.stdio,
  };

  // Languages to activate LSP for
  const documentSelector = [
    { scheme: "file", language: "typescript" },
    { scheme: "file", language: "typescriptreact" },
    { scheme: "file", language: "javascript" },
    { scheme: "file", language: "javascriptreact" },
    { scheme: "file", language: "go" },
    { scheme: "file", language: "python" },
    { scheme: "file", language: "rust" },
    { scheme: "file", language: "java" },
    { scheme: "file", language: "csharp" },
  ];

  // Client options
  const clientOptions: LanguageClientOptions = {
    documentSelector,
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher("**/*"),
    },
    outputChannelName: "Mind Palace LSP",
    initializationOptions: {
      workspaceRoot,
    },
    errorHandler: {
      error: (error, message, count) => {
        console.error(
          `Mind Palace LSP error (count: ${count}):`,
          error,
          message,
        );
        // Continue on transient errors
        if (count && count < 3) {
          return { action: ErrorAction.Continue };
        }
        return { action: ErrorAction.Shutdown };
      },
      closed: () => {
        restartCount++;
        if (restartCount <= MAX_RESTART_ATTEMPTS) {
          console.log(
            `Mind Palace LSP crashed, attempting restart ${restartCount}/${MAX_RESTART_ATTEMPTS}...`,
          );
          return { action: CloseAction.Restart };
        }
        console.error(
          `Mind Palace LSP crashed ${MAX_RESTART_ATTEMPTS} times, giving up`,
        );
        vscode.window.showErrorMessage(
          `Mind Palace LSP server crashed ${MAX_RESTART_ATTEMPTS} times. Please check the output for errors.`,
        );
        return { action: CloseAction.DoNotRestart };
      },
    },
  };

  // Create the language client
  client = new LanguageClient(
    "mindPalaceLsp",
    "Mind Palace LSP",
    serverOptions,
    clientOptions,
  );

  // Reset restart count when client successfully starts
  client.onDidChangeState((event) => {
    if (event.newState === State.Running) {
      restartCount = 0;
      console.log("Mind Palace LSP client is running, restart count reset");
    } else if (event.newState === State.Stopped) {
      console.log("Mind Palace LSP client stopped");
    }
  });

  // Register commands for LSP actions
  registerLspCommands(context);

  // Start the client
  try {
    await client.start();
    console.log("Mind Palace LSP client started");
  } catch (error) {
    console.error("Failed to start Mind Palace LSP:", error);
    vscode.window.showErrorMessage(`Mind Palace LSP failed to start: ${error}`);
    return undefined;
  }

  return client;
}

/**
 * Deactivates the LSP client.
 */
export async function deactivateLspClient(): Promise<void> {
  if (client) {
    await client.stop();
    client = undefined;
  }
}

/**
 * Gets the LSP client instance.
 */
export function getLspClient(): LanguageClient | undefined {
  return client;
}

/**
 * Registers commands for LSP code actions.
 */
function registerLspCommands(context: vscode.ExtensionContext): void {
  // Approve pattern
  context.subscriptions.push(
    vscode.commands.registerCommand(
      "mindPalace.approvePattern",
      async (patternId: string) => {
        const binaryPath = vscode.workspace
          .getConfiguration("mindPalace")
          .get<string>("binaryPath", "palace");
        const workspaceRoot = getWorkspaceRoot();

        if (!workspaceRoot) {
          return;
        }

        try {
          const terminal = vscode.window.createTerminal({
            name: "Mind Palace",
            cwd: workspaceRoot,
          });
          terminal.sendText(`${binaryPath} patterns approve ${patternId}`);
          terminal.show();
        } catch (error) {
          vscode.window.showErrorMessage(`Failed to approve pattern: ${error}`);
        }
      },
    ),
  );

  // Ignore pattern
  context.subscriptions.push(
    vscode.commands.registerCommand(
      "mindPalace.ignorePattern",
      async (patternId: string) => {
        const binaryPath = vscode.workspace
          .getConfiguration("mindPalace")
          .get<string>("binaryPath", "palace");
        const workspaceRoot = getWorkspaceRoot();

        if (!workspaceRoot) {
          return;
        }

        try {
          const terminal = vscode.window.createTerminal({
            name: "Mind Palace",
            cwd: workspaceRoot,
          });
          terminal.sendText(`${binaryPath} patterns ignore ${patternId}`);
          terminal.show();
        } catch (error) {
          vscode.window.showErrorMessage(`Failed to ignore pattern: ${error}`);
        }
      },
    ),
  );

  // Show pattern details
  context.subscriptions.push(
    vscode.commands.registerCommand(
      "mindPalace.showPattern",
      async (patternId: string) => {
        const binaryPath = vscode.workspace
          .getConfiguration("mindPalace")
          .get<string>("binaryPath", "palace");
        const workspaceRoot = getWorkspaceRoot();

        if (!workspaceRoot) {
          return;
        }

        try {
          const terminal = vscode.window.createTerminal({
            name: "Mind Palace",
            cwd: workspaceRoot,
          });
          terminal.sendText(`${binaryPath} patterns show ${patternId}`);
          terminal.show();
        } catch (error) {
          vscode.window.showErrorMessage(`Failed to show pattern: ${error}`);
        }
      },
    ),
  );

  // Show patterns list for file
  context.subscriptions.push(
    vscode.commands.registerCommand(
      "mindPalace.showPatterns",
      async (uri: string) => {
        const binaryPath = vscode.workspace
          .getConfiguration("mindPalace")
          .get<string>("binaryPath", "palace");
        const workspaceRoot = getWorkspaceRoot();

        if (!workspaceRoot) {
          return;
        }

        try {
          const terminal = vscode.window.createTerminal({
            name: "Mind Palace",
            cwd: workspaceRoot,
          });
          terminal.sendText(`${binaryPath} patterns list`);
          terminal.show();
        } catch (error) {
          vscode.window.showErrorMessage(`Failed to list patterns: ${error}`);
        }
      },
    ),
  );

  // Verify contract
  context.subscriptions.push(
    vscode.commands.registerCommand(
      "mindPalace.verifyContract",
      async (contractId: string) => {
        const binaryPath = vscode.workspace
          .getConfiguration("mindPalace")
          .get<string>("binaryPath", "palace");
        const workspaceRoot = getWorkspaceRoot();

        if (!workspaceRoot) {
          return;
        }

        try {
          const terminal = vscode.window.createTerminal({
            name: "Mind Palace",
            cwd: workspaceRoot,
          });
          terminal.sendText(`${binaryPath} contracts verify ${contractId}`);
          terminal.show();
        } catch (error) {
          vscode.window.showErrorMessage(`Failed to verify contract: ${error}`);
        }
      },
    ),
  );

  // Ignore contract
  context.subscriptions.push(
    vscode.commands.registerCommand(
      "mindPalace.ignoreContract",
      async (contractId: string) => {
        const binaryPath = vscode.workspace
          .getConfiguration("mindPalace")
          .get<string>("binaryPath", "palace");
        const workspaceRoot = getWorkspaceRoot();

        if (!workspaceRoot) {
          return;
        }

        try {
          const terminal = vscode.window.createTerminal({
            name: "Mind Palace",
            cwd: workspaceRoot,
          });
          terminal.sendText(`${binaryPath} contracts ignore ${contractId}`);
          terminal.show();
        } catch (error) {
          vscode.window.showErrorMessage(`Failed to ignore contract: ${error}`);
        }
      },
    ),
  );

  // Show contract details
  context.subscriptions.push(
    vscode.commands.registerCommand(
      "mindPalace.showContract",
      async (contractId: string) => {
        const binaryPath = vscode.workspace
          .getConfiguration("mindPalace")
          .get<string>("binaryPath", "palace");
        const workspaceRoot = getWorkspaceRoot();

        if (!workspaceRoot) {
          return;
        }

        try {
          const terminal = vscode.window.createTerminal({
            name: "Mind Palace",
            cwd: workspaceRoot,
          });
          terminal.sendText(`${binaryPath} contracts show ${contractId}`);
          terminal.show();
        } catch (error) {
          vscode.window.showErrorMessage(`Failed to show contract: ${error}`);
        }
      },
    ),
  );

  // Show contracts list
  context.subscriptions.push(
    vscode.commands.registerCommand(
      "mindPalace.showContracts",
      async (uri: string) => {
        const binaryPath = vscode.workspace
          .getConfiguration("mindPalace")
          .get<string>("binaryPath", "palace");
        const workspaceRoot = getWorkspaceRoot();

        if (!workspaceRoot) {
          return;
        }

        try {
          const terminal = vscode.window.createTerminal({
            name: "Mind Palace",
            cwd: workspaceRoot,
          });
          terminal.sendText(`${binaryPath} contracts list`);
          terminal.show();
        } catch (error) {
          vscode.window.showErrorMessage(`Failed to list contracts: ${error}`);
        }
      },
    ),
  );

  // Restart LSP server manually
  context.subscriptions.push(
    vscode.commands.registerCommand("mindPalace.restartLsp", async () => {
      if (!client) {
        vscode.window.showWarningMessage("Mind Palace LSP is not running");
        return;
      }

      try {
        vscode.window.showInformationMessage("Restarting Mind Palace LSP...");
        restartCount = 0; // Reset count for manual restart
        await client.restart();
        vscode.window.showInformationMessage(
          "Mind Palace LSP restarted successfully",
        );
      } catch (error) {
        vscode.window.showErrorMessage(
          `Failed to restart Mind Palace LSP: ${error}`,
        );
      }
    }),
  );
}

/**
 * Gets the workspace root path.
 */
function getWorkspaceRoot(): string | undefined {
  const workspaceFolders = vscode.workspace.workspaceFolders;
  if (workspaceFolders && workspaceFolders.length > 0) {
    return workspaceFolders[0].uri.fsPath;
  }
  return undefined;
}
