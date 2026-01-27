import * as vscode from "vscode";
import * as fs from "fs";
import * as path from "path";
import * as jsonc from "jsonc-parser";

/**
 * VS Code extension configuration from .palace/palace.jsonc
 */
export interface PalaceVSCodeConfig {
  statusBar?: {
    position?: "left" | "right";
    priority?: number;
  };
}

/**
 * Merged configuration from both .palace/palace.jsonc and VS Code settings.
 * Project config (.palace/palace.jsonc) takes precedence.
 */
export interface MergedConfig {
  binaryPath: string;
  showStatusBarItem: boolean;
  lsp: {
    enabled: boolean;
    diagnostics: {
      patterns: boolean;
      contracts: boolean;
    };
    codeLens: {
      enabled: boolean;
    };
  };
  statusBar: {
    position: "left" | "right";
    priority: number;
  };
}

/**
 * Default configuration values
 */
const DEFAULTS: Omit<MergedConfig, "binaryPath"> = {
  showStatusBarItem: true,
  lsp: {
    enabled: true,
    diagnostics: {
      patterns: true,
      contracts: true,
    },
    codeLens: {
      enabled: true,
    },
  },
  statusBar: {
    position: "left",
    priority: 100,
  },
};

// Adapter for filesystem calls so tests can stub without touching core module
export const fsAdapter = {
  existsSync: (p: string) => fs.existsSync(p),
  readFileSync: (p: string, enc: BufferEncoding) => fs.readFileSync(p, enc),
};

/**
 * Reads and merges configuration from .palace/palace.jsonc and VS Code settings.
 * Project config takes precedence over VS Code settings.
 */
export function getConfig(): MergedConfig {
  const vsCodeConfig = vscode.workspace.getConfiguration("mindPalace");
  const projectConfig = readProjectConfig();

  return {
    // Binary path only comes from VS Code settings
    binaryPath: vsCodeConfig.get<string>("binaryPath") || "palace",

    showStatusBarItem:
      vsCodeConfig.get<boolean>("showStatusBarItem") ?? DEFAULTS.showStatusBarItem,

    lsp: {
      enabled: vsCodeConfig.get<boolean>("lsp.enabled") ?? DEFAULTS.lsp.enabled,
      diagnostics: {
        patterns:
          vsCodeConfig.get<boolean>("lsp.diagnostics.patterns") ??
          DEFAULTS.lsp.diagnostics.patterns,
        contracts:
          vsCodeConfig.get<boolean>("lsp.diagnostics.contracts") ??
          DEFAULTS.lsp.diagnostics.contracts,
      },
      codeLens: {
        enabled:
          vsCodeConfig.get<boolean>("lsp.codeLens.enabled") ??
          DEFAULTS.lsp.codeLens.enabled,
      },
    },

    statusBar: {
      position:
        projectConfig?.statusBar?.position ?? DEFAULTS.statusBar.position,
      priority:
        projectConfig?.statusBar?.priority ?? DEFAULTS.statusBar.priority,
    },
  };
}

/**
 * Reads the vscode section from .palace/palace.jsonc
 */
export function readProjectConfig(): PalaceVSCodeConfig | null {
  const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
  if (!workspaceFolder) {
    return null;
  }

  const configPath = path.join(
    workspaceFolder.uri.fsPath,
    ".palace",
    "palace.jsonc"
  );

  try {
    if (!fsAdapter.existsSync(configPath)) {
      return null;
    }

    const content = fsAdapter.readFileSync(configPath, "utf-8");
    const errors: jsonc.ParseError[] = [];
    const parsed = jsonc.parse(content, errors);

    if (errors.length > 0) {
      console.warn("Errors parsing palace.jsonc:", errors);
      return null;
    }

    return parsed?.vscode ?? null;
  } catch (error) {
    console.error("Error reading palace.jsonc:", error);
    return null;
  }
}

/**
 * Creates a file watcher for .palace/palace.jsonc
 * Returns a disposable that should be added to extension subscriptions
 */
export function watchProjectConfig(
  onConfigChange: () => void
): vscode.Disposable {
  const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
  if (!workspaceFolder) {
    // No workspace open: return a no-op disposable to avoid errors in tests/runtime
    return { dispose: () => {} } as vscode.Disposable;
  }

  // Use string base path for broader compatibility in tests and runtime
  const basePath = workspaceFolder.uri.fsPath;
  const pattern = new vscode.RelativePattern(basePath, ".palace/palace.jsonc");

  const watcher = vscode.workspace.createFileSystemWatcher(pattern);

  watcher.onDidChange(onConfigChange);
  watcher.onDidCreate(onConfigChange);
  watcher.onDidDelete(onConfigChange);

  return watcher;
}
