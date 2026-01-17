import { Component, inject, OnInit, signal } from "@angular/core";
import { FormsModule } from "@angular/forms";
import { ApiService } from "../../core/services/api.service";
import { CommonModule } from "@angular/common";

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

@Component({
  selector: "app-proposals",
  imports: [FormsModule, CommonModule],
  template: `
    <div class="proposals-page">
      <div class="page-header">
        <h2>Knowledge Proposals</h2>
        <p class="subtitle">
          Review and approve AI-generated knowledge for the palace
        </p>
      </div>

      <!-- Stats -->
      <div class="stats-row">
        <div class="stat-card proposed">
          <div class="stat-value">{{ pendingCount() }}</div>
          <div class="stat-label">Pending Review</div>
        </div>
        <div class="stat-card approved">
          <div class="stat-value">{{ approvedCount() }}</div>
          <div class="stat-label">Approved</div>
        </div>
        <div class="stat-card rejected">
          <div class="stat-value">{{ rejectedCount() }}</div>
          <div class="stat-label">Rejected</div>
        </div>
      </div>

      <!-- Filters -->
      <div class="filters">
        <select [(ngModel)]="statusFilter" (change)="loadProposals()">
          <option value="">All Status</option>
          <option value="proposed">Pending Review</option>
          <option value="approved">Approved</option>
          <option value="rejected">Rejected</option>
        </select>

        <select [(ngModel)]="typeFilter" (change)="loadProposals()">
          <option value="">All Types</option>
          <option value="decision">Decisions</option>
          <option value="learning">Learnings</option>
          <option value="fragment">Fragments</option>
          <option value="postmortem">Postmortems</option>
        </select>

        <span class="count">{{ proposals().length }} proposals</span>
      </div>

      <!-- Loading -->
      @if (loading()) {
      <div class="loading">
        <div class="spinner"></div>
        <span>Loading proposals...</span>
      </div>
      }

      <!-- Proposals List -->
      @if (!loading() && proposals().length > 0) {
      <div class="proposals-list">
        @for (proposal of proposals(); track proposal.id) {
        <div
          class="proposal-card"
          [class]="'status-' + proposal.status + ' type-' + proposal.type"
        >
          <div class="proposal-header">
            <div class="header-left">
              <span class="type-badge" [class]="'badge-' + proposal.type">{{
                proposal.type
              }}</span>
              <span class="status-badge" [class]="'badge-' + proposal.status">{{
                proposal.status
              }}</span>
            </div>
            <div class="header-right">
              <span class="scope">{{ proposal.scope || "Global" }}</span>
              <span class="timestamp">{{
                formatTimestamp(proposal.created_at)
              }}</span>
            </div>
          </div>

          <p class="proposal-content">{{ proposal.content }}</p>

          @if (proposal.evidence) {
          <div class="evidence">
            <strong>Evidence:</strong>
            <pre>{{ formatEvidence(proposal.evidence) }}</pre>
          </div>
          } @if (proposal.reviewed_by) {
          <div class="review-info">
            <span>Reviewed by {{ proposal.reviewed_by }}</span>
            <span>{{ formatTimestamp(proposal.reviewed_at!) }}</span>
          </div>
          } @if (proposal.target_id) {
          <div class="target-link">
            <a [href]="getTargetLink(proposal)">→ View Created Record</a>
          </div>
          }

          <!-- Actions for pending proposals -->
          @if (proposal.status === 'proposed') {
          <div class="proposal-actions">
            <button
              class="btn-approve"
              (click)="approveProposal(proposal.id)"
              [disabled]="processing()"
            >
              ✓ Approve
            </button>
            <button
              class="btn-reject"
              (click)="rejectProposal(proposal.id)"
              [disabled]="processing()"
            >
              ✗ Reject
            </button>
          </div>
          }
        </div>
        }
      </div>
      }

      <!-- Empty State -->
      @if (!loading() && proposals().length === 0) {
      <div class="empty-state">
        <div class="empty-icon">📋</div>
        <h3>No Proposals</h3>
        <p>
          @if (statusFilter === 'proposed') { No pending proposals to review. }
          @else if (statusFilter === 'approved') { No approved proposals yet. }
          @else if (statusFilter === 'rejected') { No rejected proposals yet. }
          @else { No proposals found. AI agents will create proposals as they
          learn. }
        </p>
      </div>
      }

      <!-- Error -->
      @if (error()) {
      <div class="error"><strong>Error:</strong> {{ error() }}</div>
      }
    </div>
  `,
  styles: [
    `
      .proposals-page {
        padding: 2rem;
        max-width: 1200px;
        margin: 0 auto;
      }

      .page-header {
        margin-bottom: 2rem;
      }

      .page-header h2 {
        margin: 0 0 0.5rem 0;
        font-size: 2rem;
        color: var(--text-primary, #1a1a1a);
      }

      .subtitle {
        margin: 0;
        color: var(--text-secondary, #666);
      }

      .stats-row {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 1rem;
        margin-bottom: 2rem;
      }

      .stat-card {
        padding: 1.5rem;
        border-radius: 8px;
        text-align: center;
        border: 2px solid;
      }

      .stat-card.proposed {
        background: #fff9e6;
        border-color: #f5c518;
      }

      .stat-card.approved {
        background: #e8f5e9;
        border-color: #4caf50;
      }

      .stat-card.rejected {
        background: #ffebee;
        border-color: #f44336;
      }

      .stat-value {
        font-size: 2.5rem;
        font-weight: bold;
        margin-bottom: 0.5rem;
      }

      .stat-label {
        color: var(--text-secondary, #666);
        font-size: 0.875rem;
      }

      .filters {
        display: flex;
        gap: 1rem;
        margin-bottom: 2rem;
        align-items: center;
      }

      .filters select {
        padding: 0.5rem 1rem;
        border: 1px solid var(--border-color, #ddd);
        border-radius: 4px;
        background: white;
        font-size: 0.875rem;
      }

      .count {
        margin-left: auto;
        color: var(--text-secondary, #666);
        font-size: 0.875rem;
      }

      .loading,
      .empty-state {
        text-align: center;
        padding: 4rem 2rem;
      }

      .spinner {
        width: 48px;
        height: 48px;
        border: 4px solid var(--border-color, #ddd);
        border-top-color: var(--primary-color, #007bff);
        border-radius: 50%;
        animation: spin 1s linear infinite;
        margin: 0 auto 1rem;
      }

      @keyframes spin {
        to {
          transform: rotate(360deg);
        }
      }

      .empty-icon {
        font-size: 4rem;
        margin-bottom: 1rem;
      }

      .proposals-list {
        display: flex;
        flex-direction: column;
        gap: 1rem;
      }

      .proposal-card {
        background: white;
        border: 2px solid var(--border-color, #ddd);
        border-radius: 8px;
        padding: 1.5rem;
        transition: all 0.2s;
      }

      .proposal-card:hover {
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
      }

      .proposal-card.status-proposed {
        border-left: 4px solid #f5c518;
      }

      .proposal-card.status-approved {
        border-left: 4px solid #4caf50;
      }

      .proposal-card.status-rejected {
        border-left: 4px solid #f44336;
        opacity: 0.7;
      }

      .proposal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 1rem;
      }

      .header-left,
      .header-right {
        display: flex;
        gap: 0.5rem;
        align-items: center;
      }

      .type-badge,
      .status-badge {
        padding: 0.25rem 0.75rem;
        border-radius: 4px;
        font-size: 0.75rem;
        font-weight: 600;
        text-transform: uppercase;
      }

      .type-badge {
        background: var(--bg-secondary, #f5f5f5);
        color: var(--text-primary, #333);
      }

      .status-badge.badge-proposed {
        background: #fff9e6;
        color: #856404;
      }

      .status-badge.badge-approved {
        background: #e8f5e9;
        color: #2e7d32;
      }

      .status-badge.badge-rejected {
        background: #ffebee;
        color: #c62828;
      }

      .scope,
      .timestamp {
        font-size: 0.75rem;
        color: var(--text-secondary, #666);
      }

      .proposal-content {
        margin: 1rem 0;
        line-height: 1.6;
        color: var(--text-primary, #1a1a1a);
      }

      .evidence {
        margin: 1rem 0;
        padding: 1rem;
        background: var(--bg-secondary, #f5f5f5);
        border-radius: 4px;
        font-size: 0.875rem;
      }

      .evidence pre {
        margin: 0.5rem 0 0 0;
        white-space: pre-wrap;
        font-family: monospace;
        font-size: 0.8125rem;
      }

      .review-info {
        display: flex;
        justify-content: space-between;
        margin-top: 1rem;
        padding-top: 1rem;
        border-top: 1px solid var(--border-color, #ddd);
        font-size: 0.875rem;
        color: var(--text-secondary, #666);
      }

      .target-link {
        margin-top: 1rem;
      }

      .target-link a {
        color: var(--primary-color, #007bff);
        text-decoration: none;
        font-size: 0.875rem;
      }

      .target-link a:hover {
        text-decoration: underline;
      }

      .proposal-actions {
        display: flex;
        gap: 1rem;
        margin-top: 1rem;
        padding-top: 1rem;
        border-top: 1px solid var(--border-color, #ddd);
      }

      .proposal-actions button {
        flex: 1;
        padding: 0.75rem 1.5rem;
        border: none;
        border-radius: 4px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.2s;
      }

      .proposal-actions button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }

      .btn-approve {
        background: #4caf50;
        color: white;
      }

      .btn-approve:hover:not(:disabled) {
        background: #45a049;
      }

      .btn-reject {
        background: #f44336;
        color: white;
      }

      .btn-reject:hover:not(:disabled) {
        background: #da190b;
      }

      .error {
        padding: 1rem;
        background: #ffebee;
        border: 1px solid #f44336;
        border-radius: 4px;
        color: #c62828;
        margin-top: 1rem;
      }
    `,
  ],
})
export class ProposalsComponent implements OnInit {
  private api = inject(ApiService);

  proposals = signal<Proposal[]>([]);
  loading = signal(false);
  processing = signal(false);
  error = signal<string | null>(null);

  statusFilter = "";
  typeFilter = "";

  ngOnInit() {
    this.loadProposals();
  }

  pendingCount = signal(0);
  approvedCount = signal(0);
  rejectedCount = signal(0);

  async loadProposals() {
    this.loading.set(true);
    this.error.set(null);

    try {
      // Call API to get proposals
      const params = new URLSearchParams();
      if (this.statusFilter) params.set("status", this.statusFilter);
      if (this.typeFilter) params.set("type", this.typeFilter);

      const response = await fetch(`/api/proposals?${params.toString()}`);

      if (!response.ok) {
        throw new Error(`Failed to load proposals: ${response.statusText}`);
      }

      const data = await response.json();
      this.proposals.set(data.proposals || []);

      // Update counts
      const allProposals = this.statusFilter
        ? await this.fetchAllProposals()
        : data.proposals || [];

      this.pendingCount.set(
        allProposals.filter((p: Proposal) => p.status === "proposed").length
      );
      this.approvedCount.set(
        allProposals.filter((p: Proposal) => p.status === "approved").length
      );
      this.rejectedCount.set(
        allProposals.filter((p: Proposal) => p.status === "rejected").length
      );
    } catch (err) {
      this.error.set(
        err instanceof Error ? err.message : "Failed to load proposals"
      );
    } finally {
      this.loading.set(false);
    }
  }

  private async fetchAllProposals(): Promise<Proposal[]> {
    try {
      const response = await fetch("/api/proposals");
      if (!response.ok) return [];
      const data = await response.json();
      return data.proposals || [];
    } catch {
      return [];
    }
  }

  async approveProposal(id: string) {
    if (
      !confirm("Approve this proposal? It will become authoritative knowledge.")
    ) {
      return;
    }

    this.processing.set(true);
    this.error.set(null);

    try {
      const response = await fetch(`/api/proposals/${id}/approve`, {
        method: "POST",
      });

      if (!response.ok) {
        throw new Error(`Failed to approve: ${response.statusText}`);
      }

      // Reload proposals
      await this.loadProposals();
    } catch (err) {
      this.error.set(
        err instanceof Error ? err.message : "Failed to approve proposal"
      );
    } finally {
      this.processing.set(false);
    }
  }

  async rejectProposal(id: string) {
    if (!confirm("Reject this proposal? This action cannot be undone.")) {
      return;
    }

    this.processing.set(true);
    this.error.set(null);

    try {
      const response = await fetch(`/api/proposals/${id}/reject`, {
        method: "POST",
      });

      if (!response.ok) {
        throw new Error(`Failed to reject: ${response.statusText}`);
      }

      // Reload proposals
      await this.loadProposals();
    } catch (err) {
      this.error.set(
        err instanceof Error ? err.message : "Failed to reject proposal"
      );
    } finally {
      this.processing.set(false);
    }
  }

  formatTimestamp(timestamp: number): string {
    const date = new Date(timestamp * 1000);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return "Just now";
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffMins < 1440) return `${Math.floor(diffMins / 60)}h ago`;
    if (diffMins < 10080) return `${Math.floor(diffMins / 1440)}d ago`;

    return date.toLocaleDateString();
  }

  formatEvidence(evidence: string): string {
    try {
      const parsed = JSON.parse(evidence);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return evidence;
    }
  }

  getTargetLink(proposal: Proposal): string {
    if (!proposal.target_id) return "#";

    switch (proposal.type) {
      case "decision":
        return `/decisions#${proposal.target_id}`;
      case "learning":
        return `/learnings#${proposal.target_id}`;
      case "fragment":
        return `/intel#${proposal.target_id}`;
      case "postmortem":
        return `/postmortems#${proposal.target_id}`;
      default:
        return "#";
    }
  }
}
