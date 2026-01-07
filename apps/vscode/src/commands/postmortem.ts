import * as vscode from "vscode";
import { PalaceBridge } from "../bridge";
import { logger } from "../logger";

/**
 * Postmortem data structure
 */
export interface Postmortem {
  id: string;
  title: string;
  whatHappened: string;
  rootCause?: string;
  lessonsLearned: string[];
  preventionSteps: string[];
  severity: "low" | "medium" | "high" | "critical";
  status: "open" | "resolved" | "recurring";
  affectedFiles: string[];
  relatedDecision?: string;
  relatedSession?: string;
  createdAt: string;
}

/**
 * Add a new postmortem record
 */
export async function addPostmortem(bridge: PalaceBridge): Promise<void> {
  logger.info("Starting postmortem capture", "PostmortemCommand");

  // Step 1: Get title
  const title = await vscode.window.showInputBox({
    prompt: "Postmortem title",
    placeHolder: "e.g., Authentication Bug Resolution, Build Failure in CI/CD",
    validateInput: (value) => {
      if (!value || value.trim().length === 0) {
        return "Title is required";
      }
      if (value.trim().length < 5) {
        return "Title must be at least 5 characters";
      }
      return null;
    },
  });

  if (!title) {
    logger.info("Postmortem cancelled: no title provided", "PostmortemCommand");
    return;
  }

  // Step 2: Get what happened
  const whatHappened = await vscode.window.showInputBox({
    prompt: "What went wrong? (Describe the failure)",
    placeHolder:
      "e.g., JWT tokens were being validated incorrectly, allowing unauthorized access",
    validateInput: (value) => {
      if (!value || value.trim().length < 10) {
        return "Please provide a detailed description (min 10 characters)";
      }
      return null;
    },
  });

  if (!whatHappened) {
    logger.info(
      "Postmortem cancelled: no description provided",
      "PostmortemCommand"
    );
    return;
  }

  // Step 3: Get root cause (optional but recommended)
  const rootCause = await vscode.window.showInputBox({
    prompt: "Root cause analysis (optional)",
    placeHolder: "e.g., Incorrect algorithm used in token verification logic",
  });

  // Step 4: Get severity
  const severity = await vscode.window.showQuickPick(
    [
      {
        label: "$(warning) Low",
        value: "low",
        description: "Minor issue, limited impact",
      },
      {
        label: "$(alert) Medium",
        value: "medium",
        description: "Moderate impact, needs attention",
      },
      {
        label: "$(flame) High",
        value: "high",
        description: "Significant impact, urgent fix needed",
      },
      {
        label: "$(error) Critical",
        value: "critical",
        description: "Severe impact, immediate action required",
      },
    ],
    { placeHolder: "Select severity level" }
  );

  if (!severity) {
    logger.info(
      "Postmortem cancelled: no severity selected",
      "PostmortemCommand"
    );
    return;
  }

  // Step 5: Gather lessons learned (multi-step input)
  const lessonsLearned: string[] = [];
  let addMoreLessons = true;

  while (addMoreLessons && lessonsLearned.length < 5) {
    const lesson = await vscode.window.showInputBox({
      prompt: `Lesson learned #${
        lessonsLearned.length + 1
      } (leave empty to skip)`,
      placeHolder:
        "e.g., Always validate token expiry before checking signature",
    });

    if (lesson && lesson.trim()) {
      lessonsLearned.push(lesson.trim());

      if (lessonsLearned.length < 5) {
        const addMore = await vscode.window.showQuickPick(
          ["Add another lesson", "Continue to prevention steps"],
          { placeHolder: "Add more lessons?" }
        );
        addMoreLessons = addMore === "Add another lesson";
      } else {
        addMoreLessons = false;
      }
    } else {
      addMoreLessons = false;
    }
  }

  // Step 6: Gather prevention steps
  const preventionSteps: string[] = [];
  let addMoreSteps = true;

  while (addMoreSteps && preventionSteps.length < 5) {
    const step = await vscode.window.showInputBox({
      prompt: `Prevention step #${
        preventionSteps.length + 1
      } (leave empty to skip)`,
      placeHolder: "e.g., Add unit tests for token validation edge cases",
    });

    if (step && step.trim()) {
      preventionSteps.push(step.trim());

      if (preventionSteps.length < 5) {
        const addMore = await vscode.window.showQuickPick(
          ["Add another step", "Finish and save"],
          { placeHolder: "Add more prevention steps?" }
        );
        addMoreSteps = addMore === "Add another step";
      } else {
        addMoreSteps = false;
      }
    } else {
      addMoreSteps = false;
    }
  }

  // Step 7: Gather context automatically
  const context = await gatherPostmortemContext();

  // Step 8: Store the postmortem
  try {
    logger.info("Storing postmortem via MCP", "PostmortemCommand", { title });

    const notes = [
      `What Happened: ${whatHappened.trim()}`,
      `Root Cause: ${rootCause?.trim() || "Unknown"}`,
      `Lessons Learned:`,
      ...lessonsLearned.map((l) => `- ${l}`),
      `Prevention Steps:`,
      ...preventionSteps.map((s) => `- ${s}`),
    ].join("\\n");

    const result = await bridge.addPostmortem(title.trim(), notes, {
      severity: severity.value,
      affectedFiles: context.affectedFiles,
      relatedSession: context.relatedSession,
    });

    const postmortemId = result.id;

    vscode.window
      .showInformationMessage(
        `✅ Postmortem captured: ${title}`,
        "View Details",
        "View All Postmortems"
      )
      .then((selection) => {
        if (selection === "View Details") {
          vscode.commands.executeCommand("mindPalace.showPostmortemDetail", {
            id: postmortemId,
          });
        } else if (selection === "View All Postmortems") {
          vscode.commands.executeCommand("mindPalace.refreshKnowledge");
          vscode.commands.executeCommand("mindPalace.knowledgeView.focus");
        }
      });

    // Refresh knowledge view to show new postmortem
    vscode.commands.executeCommand("mindPalace.refreshKnowledge");

    logger.info("Postmortem stored successfully", "PostmortemCommand", {
      title,
      severity: severity.value,
      lessonsCount: lessonsLearned.length,
      preventionCount: preventionSteps.length,
    });
  } catch (error: any) {
    logger.error("Failed to store postmortem", error, "PostmortemCommand");
    vscode.window.showErrorMessage(
      `Failed to capture postmortem: ${error.message}`
    );
  }
}

/**
 * Gather contextual information for the postmortem
 */
async function gatherPostmortemContext(): Promise<{
  affectedFiles: string[];
  relatedSession?: string;
  workspace: string | undefined;
}> {
  const affectedFiles: string[] = [];
  const workspaceName = vscode.workspace.name;

  // Get currently open/active file
  const activeFile = vscode.window.activeTextEditor?.document.uri.fsPath;
  const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;

  if (activeFile && workspaceRoot) {
    const relativePath = activeFile.startsWith(workspaceRoot)
      ? activeFile.substring(workspaceRoot.length + 1).replace(/\\/g, "/")
      : activeFile;
    affectedFiles.push(relativePath);
  }

  // Get recently changed files from git (if available)
  try {
    const recentFiles = await getRecentlyEditedFiles();
    affectedFiles.push(...recentFiles.slice(0, 5)); // Max 5 recent files
  } catch (error) {
    logger.debug("Could not get recent files from git", "PostmortemCommand");
  }

  // Try to get current session ID if one is active
  let relatedSession: string | undefined;
  try {
    // This would require a bridge method to get current session
    // For now, we'll leave it undefined
  } catch (error) {
    logger.debug("Could not get current session", "PostmortemCommand");
  }

  return {
    affectedFiles: [...new Set(affectedFiles)], // Remove duplicates
    relatedSession,
    workspace: workspaceName,
  };
}

/**
 * Get recently edited files from git status
 */
async function getRecentlyEditedFiles(): Promise<string[]> {
  const files: string[] = [];

  try {
    const gitExtension = vscode.extensions.getExtension("vscode.git")?.exports;
    if (!gitExtension) {
      return files;
    }

    const git = gitExtension.getAPI(1);
    const repo = git.repositories[0];

    if (!repo) {
      return files;
    }

    const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
    if (!workspaceRoot) {
      return files;
    }

    // Get modified and staged files
    const changes = repo.state.workingTreeChanges.concat(
      repo.state.indexChanges
    );

    for (const change of changes) {
      if (change.uri) {
        const relativePath = change.uri.fsPath.startsWith(workspaceRoot)
          ? change.uri.fsPath
              .substring(workspaceRoot.length + 1)
              .replace(/\\/g, "/")
          : change.uri.fsPath;
        files.push(relativePath);
      }
    }
  } catch (error) {
    logger.debug("Error getting git status", "PostmortemCommand");
  }

  return files;
}

/**
 * Quick postmortem - simplified workflow for rapid capture
 */
export async function quickPostmortem(bridge: PalaceBridge): Promise<void> {
  logger.info("Starting quick postmortem capture", "PostmortemCommand");

  const title = await vscode.window.showInputBox({
    prompt: "Quick Postmortem: What went wrong?",
    placeHolder: "Brief description of the failure",
    validateInput: (value) => {
      if (!value || value.trim().length < 5) {
        return "Description must be at least 5 characters";
      }
      return null;
    },
  });

  if (!title) {
    return;
  }

  const context = await gatherPostmortemContext();

  try {
    await bridge.addPostmortem(title.trim(), title.trim(), {
      severity: "medium",
      affectedFiles: context.affectedFiles,
    });

    vscode.window.showInformationMessage(`✅ Quick postmortem saved: ${title}`);
    vscode.commands.executeCommand("mindPalace.refreshKnowledge");

    logger.info("Quick postmortem stored", "PostmortemCommand", { title });
  } catch (error: any) {
    logger.error(
      "Failed to store quick postmortem",
      error,
      "PostmortemCommand"
    );
    vscode.window.showErrorMessage(`Failed: ${error.message}`);
  }
}
