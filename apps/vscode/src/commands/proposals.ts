import * as vscode from "vscode";
import * as cp from "child_process";
import * as util from "util";
import { PalaceBridge } from "../bridge";

const exec = util.promisify(cp.exec);

/**
 * Proposal interface matching backend
 */
export interface Proposal {
  id: string;
  type: string;
  content: string;
  scope: string;
  status: "proposed" | "approved" | "rejected";
  created_at: number;
  updated_at: number;
  reviewed_by?: string;
  reviewed_at?: number;
  evidence?: string;
  target_id?: string;
}

/**
 * Register proposal-related commands
 */
export function registerProposalCommands(
  context: vscode.ExtensionContext,
  bridge: PalaceBridge
): void {
  // List proposals command
  context.subscriptions.push(
    vscode.commands.registerCommand(
      "palace.proposals.list",
      async () => await listProposals(bridge)
    )
  );

  // Approve proposal command
  context.subscriptions.push(
    vscode.commands.registerCommand(
      "palace.proposals.approve",
      async () => await approveProposal(bridge)
    )
  );

  // Reject proposal command
  context.subscriptions.push(
    vscode.commands.registerCommand(
      "palace.proposals.reject",
      async () => await rejectProposal(bridge)
    )
  );

  // Review pending proposals command
  context.subscriptions.push(
    vscode.commands.registerCommand(
      "palace.proposals.reviewPending",
      async () => await reviewPendingProposals(bridge)
    )
  );
}

/**
 * List proposals with filtering
 */
async function listProposals(bridge: PalaceBridge): Promise<void> {
  try {
    // Ask for filter
    const statusFilter = await vscode.window.showQuickPick(
      [
        { label: "All Proposals", value: "" },
        { label: "Pending Review", value: "proposed" },
        { label: "Approved", value: "approved" },
        { label: "Rejected", value: "rejected" },
      ],
      {
        placeHolder: "Filter by status",
      }
    );

    if (!statusFilter) {
      return;
    }

    // Fetch proposals (this would call the CLI or use the bridge)
    const proposals = await fetchProposals(bridge, statusFilter.value);

    if (proposals.length === 0) {
      vscode.window.showInformationMessage(
        `No ${statusFilter.label.toLowerCase()} found.`
      );
      return;
    }

    // Show proposals in quick pick
    const selected = await vscode.window.showQuickPick(
      proposals.map((p) => ({
        label: `$(${getIconForType(p.type)}) ${p.content.substring(0, 60)}${
          p.content.length > 60 ? "..." : ""
        }`,
        description: `${p.type} • ${p.scope} • ${formatStatus(p.status)}`,
        detail: `Created ${formatDate(p.created_at)} • ID: ${p.id}`,
        proposal: p,
      })),
      {
        placeHolder: "Select a proposal to view details",
      }
    );

    if (selected) {
      await showProposalDetails(selected.proposal);
    }
  } catch (error) {
    vscode.window.showErrorMessage(
      `Failed to list proposals: ${
        error instanceof Error ? error.message : String(error)
      }`
    );
  }
}

/**
 * Approve a proposal
 */
async function approveProposal(bridge: PalaceBridge): Promise<void> {
  try {
    // Fetch pending proposals
    const proposals = await fetchProposals(bridge, "proposed");

    if (proposals.length === 0) {
      vscode.window.showInformationMessage("No pending proposals to approve.");
      return;
    }

    // Show proposals for selection
    const selected = await vscode.window.showQuickPick(
      proposals.map((p) => ({
        label: `$(${getIconForType(p.type)}) ${p.content.substring(0, 60)}${
          p.content.length > 60 ? "..." : ""
        }`,
        description: `${p.type} • ${p.scope}`,
        detail: `Created ${formatDate(p.created_at)} • ID: ${p.id}`,
        proposal: p,
      })),
      {
        placeHolder: "Select a proposal to approve",
      }
    );

    if (!selected) {
      return;
    }

    // Confirm approval
    const confirm = await vscode.window.showWarningMessage(
      `Approve this ${selected.proposal.type}?\n\n"${selected.proposal.content}"\n\nIt will become authoritative knowledge.`,
      { modal: true },
      "Approve"
    );

    if (confirm !== "Approve") {
      return;
    }

    // Call CLI to approve
    await execCLI(["approve", selected.proposal.id]);

    vscode.window.showInformationMessage(
      `✓ Proposal approved: ${selected.proposal.type}`
    );
  } catch (error) {
    vscode.window.showErrorMessage(
      `Failed to approve proposal: ${
        error instanceof Error ? error.message : String(error)
      }`
    );
  }
}

/**
 * Reject a proposal
 */
async function rejectProposal(bridge: PalaceBridge): Promise<void> {
  try {
    // Fetch pending proposals
    const proposals = await fetchProposals(bridge, "proposed");

    if (proposals.length === 0) {
      vscode.window.showInformationMessage("No pending proposals to reject.");
      return;
    }

    // Show proposals for selection
    const selected = await vscode.window.showQuickPick(
      proposals.map((p) => ({
        label: `$(${getIconForType(p.type)}) ${p.content.substring(0, 60)}${
          p.content.length > 60 ? "..." : ""
        }`,
        description: `${p.type} • ${p.scope}`,
        detail: `Created ${formatDate(p.created_at)} • ID: ${p.id}`,
        proposal: p,
      })),
      {
        placeHolder: "Select a proposal to reject",
      }
    );

    if (!selected) {
      return;
    }

    // Confirm rejection
    const confirm = await vscode.window.showWarningMessage(
      `Reject this ${selected.proposal.type}?\n\n"${selected.proposal.content}"\n\nThis action cannot be undone.`,
      { modal: true },
      "Reject"
    );

    if (confirm !== "Reject") {
      return;
    }

    // Call CLI to reject
    await execCLI(["reject", selected.proposal.id]);

    vscode.window.showInformationMessage(
      `✗ Proposal rejected: ${selected.proposal.type}`
    );
  } catch (error) {
    vscode.window.showErrorMessage(
      `Failed to reject proposal: ${
        error instanceof Error ? error.message : String(error)
      }`
    );
  }
}

/**
 * Review pending proposals one by one
 */
async function reviewPendingProposals(bridge: PalaceBridge): Promise<void> {
  try {
    // Fetch pending proposals
    let proposals = await fetchProposals(bridge, "proposed");

    if (proposals.length === 0) {
      vscode.window.showInformationMessage("No pending proposals to review.");
      return;
    }

    vscode.window.showInformationMessage(
      `${proposals.length} pending proposal(s) to review.`
    );

    // Review each proposal
    for (const proposal of proposals) {
      const action = await vscode.window.showInformationMessage(
        `Review ${proposal.type} (${proposals.indexOf(proposal) + 1}/${
          proposals.length
        }):\n\n"${proposal.content}"\n\nScope: ${
          proposal.scope
        }\nCreated: ${formatDate(proposal.created_at)}`,
        { modal: true },
        "Approve",
        "Reject",
        "Skip",
        "Cancel"
      );

      if (action === "Approve") {
        await execCLI(["approve", proposal.id]);
        vscode.window.showInformationMessage("✓ Approved");
      } else if (action === "Reject") {
        await execCLI(["reject", proposal.id]);
        vscode.window.showInformationMessage("✗ Rejected");
      } else if (action === "Cancel") {
        break;
      }
      // Skip continues to next
    }

    vscode.window.showInformationMessage("Review complete.");
  } catch (error) {
    vscode.window.showErrorMessage(
      `Failed to review proposals: ${
        error instanceof Error ? error.message : String(error)
      }`
    );
  }
}

/**
 * Show detailed proposal information
 */
async function showProposalDetails(proposal: Proposal): Promise<void> {
  const panel = vscode.window.createWebviewPanel(
    "proposalDetails",
    `Proposal: ${proposal.type}`,
    vscode.ViewColumn.One,
    { enableScripts: true }
  );

  panel.webview.html = getProposalDetailsHTML(proposal);
}

/**
 * Fetch proposals from CLI
 */
async function fetchProposals(
  bridge: PalaceBridge,
  status: string
): Promise<Proposal[]> {
  try {
    const args = ["proposals", "--json"];
    if (status) {
      args.push("--status", status);
    }

    const result = await execCLI(args);

    // Parse JSON output
    try {
      const data = JSON.parse(result);
      return data.proposals || [];
    } catch {
      // Fallback: return empty array if parsing fails
      return [];
    }
  } catch (error) {
    console.error("Failed to fetch proposals:", error);
    return [];
  }
}

/**
 * Execute CLI command
 */
async function execCLI(args: string[]): Promise<string> {
  const config = vscode.workspace.getConfiguration("mindPalace");
  const bin = config.get<string>("binaryPath") || "palace";
  const cwd = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;

  const command = `${bin} ${args.join(" ")}`;
  const { stdout } = await exec(command, { cwd });
  return stdout;
}

/**
 * Get icon for proposal type
 */
function getIconForType(type: string): string {
  switch (type) {
    case "decision":
      return "law";
    case "learning":
      return "lightbulb";
    case "fragment":
      return "code";
    case "postmortem":
      return "archive";
    default:
      return "file";
  }
}

/**
 * Format status with emoji
 */
function formatStatus(status: string): string {
  switch (status) {
    case "proposed":
      return "⏳ Pending";
    case "approved":
      return "✓ Approved";
    case "rejected":
      return "✗ Rejected";
    default:
      return status;
  }
}

/**
 * Format timestamp
 */
function formatDate(timestamp: number): string {
  const date = new Date(timestamp * 1000);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);

  if (diffMins < 1) return "just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffMins < 1440) return `${Math.floor(diffMins / 60)}h ago`;
  if (diffMins < 10080) return `${Math.floor(diffMins / 1440)}d ago`;

  return date.toLocaleDateString();
}

/**
 * Generate HTML for proposal details webview
 */
function getProposalDetailsHTML(proposal: Proposal): string {
  return `<!DOCTYPE html>
<html>
<head>
    <style>
        body {
            font-family: var(--vscode-font-family);
            padding: 20px;
            color: var(--vscode-foreground);
            background: var(--vscode-editor-background);
        }
        .header {
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 1px solid var(--vscode-panel-border);
        }
        .badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: 600;
            margin-right: 8px;
        }
        .badge-type {
            background: var(--vscode-button-background);
            color: var(--vscode-button-foreground);
        }
        .badge-status {
            background: var(--vscode-editorWarning-foreground);
        }
        .content {
            margin: 20px 0;
            padding: 15px;
            background: var(--vscode-editor-background);
            border-left: 3px solid var(--vscode-focusBorder);
        }
        .metadata {
            display: grid;
            grid-template-columns: 120px 1fr;
            gap: 10px;
            margin: 20px 0;
            font-size: 13px;
        }
        .metadata dt {
            font-weight: 600;
            color: var(--vscode-descriptionForeground);
        }
        .evidence {
            margin-top: 20px;
            padding: 15px;
            background: var(--vscode-textCodeBlock-background);
            border-radius: 4px;
        }
        pre {
            margin: 0;
            white-space: pre-wrap;
        }
    </style>
</head>
<body>
    <div class="header">
        <h2>Proposal Details</h2>
        <div>
            <span class="badge badge-type">${proposal.type}</span>
            <span class="badge badge-status">${formatStatus(
              proposal.status
            )}</span>
        </div>
    </div>
    
    <div class="content">
        <p>${proposal.content}</p>
    </div>
    
    <dl class="metadata">
        <dt>ID:</dt>
        <dd>${proposal.id}</dd>
        
        <dt>Scope:</dt>
        <dd>${proposal.scope}</dd>
        
        <dt>Created:</dt>
        <dd>${new Date(proposal.created_at * 1000).toLocaleString()}</dd>
        
        <dt>Updated:</dt>
        <dd>${new Date(proposal.updated_at * 1000).toLocaleString()}</dd>
        
        ${
          proposal.reviewed_by
            ? `
        <dt>Reviewed By:</dt>
        <dd>${proposal.reviewed_by}</dd>
        
        <dt>Reviewed At:</dt>
        <dd>${new Date(
          (proposal.reviewed_at || 0) * 1000
        ).toLocaleString()}</dd>
        `
            : ""
        }
        
        ${
          proposal.target_id
            ? `
        <dt>Target ID:</dt>
        <dd>${proposal.target_id}</dd>
        `
            : ""
        }
    </dl>
    
    ${
      proposal.evidence
        ? `
    <div class="evidence">
        <strong>Evidence:</strong>
        <pre>${proposal.evidence}</pre>
    </div>
    `
        : ""
    }
</body>
</html>`;
}
