import * as cp from "child_process";
import * as vscode from "vscode";
import * as util from "util";

const exec = util.promisify(cp.exec);

/**
 * Status result from palace status command
 */
export interface StatusResult {
  initialized: boolean;
  fresh: boolean;
  decisions: number;
  ideas: number;
  learnings: number;
  rooms: number;
  error?: string;
}

/**
 * PalaceBridge - minimal bridge to CLI for status bar functionality
 */
export class PalaceBridge {
  private workspacePath: string | undefined;
  private binaryPath: string;

  constructor() {
    this.workspacePath = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
    const config = vscode.workspace.getConfiguration("mindPalace");
    this.binaryPath = config.get<string>("binaryPath") || "palace";
  }

  /**
   * Execute a palace CLI command
   */
  private async exec(args: string): Promise<{ stdout: string; stderr: string }> {
    if (!this.workspacePath) {
      throw new Error("No workspace folder open");
    }
    return exec(`${this.binaryPath} ${args}`, {
      cwd: this.workspacePath,
    });
  }

  /**
   * Get workspace status for status bar display
   */
  async getStatus(): Promise<StatusResult> {
    if (!this.workspacePath) {
      return {
        initialized: false,
        fresh: false,
        decisions: 0,
        ideas: 0,
        learnings: 0,
        rooms: 0,
        error: "No workspace open",
      };
    }

    try {
      const { stdout } = await this.exec("status --json");
      const status = JSON.parse(stdout);
      
      return {
        initialized: true,
        fresh: status.fresh ?? true,
        decisions: status.decisionCount ?? 0,
        ideas: status.ideaCount ?? 0,
        learnings: status.learningCount ?? 0,
        rooms: status.roomCount ?? 0,
      };
    } catch (error: any) {
      // Check if not initialized
      if (error.message?.includes("not initialized") || 
          error.stderr?.includes("not initialized")) {
        return {
          initialized: false,
          fresh: false,
          decisions: 0,
          ideas: 0,
          learnings: 0,
          rooms: 0,
        };
      }
      
      return {
        initialized: false,
        fresh: false,
        decisions: 0,
        ideas: 0,
        learnings: 0,
        rooms: 0,
        error: error.message || "Unknown error",
      };
    }
  }

  /**
   * Check if Palace is initialized in the workspace
   */
  async isInitialized(): Promise<boolean> {
    const status = await this.getStatus();
    return status.initialized;
  }
}
