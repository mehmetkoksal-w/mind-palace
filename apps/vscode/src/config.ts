import * as vscode from "vscode";
import * as fs from "fs";
import * as path from "path";
import * as jsonc from "jsonc-parser";
import { logger } from "./services/logger";

/**
 * VS Code extension configuration from .palace/palace.jsonc
 */
export interface PalaceVSCodeConfig {
  autoSync?: boolean;
  autoSyncDelay?: number;
  waitForCleanWorkspace?: boolean;
  decorations?: {
    enabled?: boolean;
    style?: "gutter" | "inline" | "both";
  };
  statusBar?: {
    position?: "left" | "right";
    priority?: number;
  };
  sidebar?: {
    defaultView?: "tree" | "graph";
    graphLayout?: "cose" | "circle" | "grid" | "breadthfirst";
  };
}

/**
 * Merged configuration from both .palace/palace.jsonc and VS Code settings.
 * Project config (.palace/palace.jsonc) takes precedence.
 */
export interface MergedConfig {
  binaryPath: string;
  autoSync: boolean;
  autoSyncDelay: number;
  waitForCleanWorkspace: boolean;
  decorations: {
    enabled: boolean;
    style: "gutter" | "inline" | "both";
  };
  statusBar: {
    position: "left" | "right";
    priority: number;
  };
  sidebar: {
    defaultView: "tree" | "graph";
    graphLayout: "cose" | "circle" | "grid" | "breadthfirst";
  };
}

/**
 * Default configuration values
 */
const DEFAULTS: Omit<MergedConfig, "binaryPath"> = {
  autoSync: true,
  autoSyncDelay: 1500,
  waitForCleanWorkspace: false,
  decorations: {
    enabled: true,
    style: "gutter",
  },
  statusBar: {
    position: "left",
    priority: 100,
  },
  sidebar: {
    defaultView: "tree",
    graphLayout: "cose",
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

    // Merged settings (project config > VS Code settings > defaults)
    autoSync:
      projectConfig?.autoSync ??
      vsCodeConfig.get<boolean>("autoSync") ??
      DEFAULTS.autoSync,
    autoSyncDelay:
      projectConfig?.autoSyncDelay ??
      vsCodeConfig.get<number>("autoSyncDelay") ??
      DEFAULTS.autoSyncDelay,
    waitForCleanWorkspace:
      projectConfig?.waitForCleanWorkspace ??
      vsCodeConfig.get<boolean>("waitForCleanWorkspace") ??
      DEFAULTS.waitForCleanWorkspace,

    decorations: {
      enabled:
        projectConfig?.decorations?.enabled ?? DEFAULTS.decorations.enabled,
      style: projectConfig?.decorations?.style ?? DEFAULTS.decorations.style,
    },

    statusBar: {
      position:
        projectConfig?.statusBar?.position ?? DEFAULTS.statusBar.position,
      priority:
        projectConfig?.statusBar?.priority ?? DEFAULTS.statusBar.priority,
    },

    sidebar: {
      defaultView:
        projectConfig?.sidebar?.defaultView ?? DEFAULTS.sidebar.defaultView,
      graphLayout:
        projectConfig?.sidebar?.graphLayout ?? DEFAULTS.sidebar.graphLayout,
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
      logger.warn("Errors parsing palace.jsonc", "Config", { errors });
      return null;
    }

    return parsed?.vscode ?? null;
  } catch (error) {
    logger.error("Error reading palace.jsonc", error, "Config");
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
