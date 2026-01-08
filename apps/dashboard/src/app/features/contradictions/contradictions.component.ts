import { Component, OnInit, inject, signal } from "@angular/core";

import { HttpClient } from "@angular/common/http";

interface ContradictionLink {
  id: string;
  sourceId: string;
  sourceKind: string;
  targetId: string;
  targetKind: string;
  confidence: number;
  explanation: string;
  createdAt: string;
}

interface ContradictionSummary {
  totalContradictionLinks: number;
  topContradictions: ContradictionLink[];
  recordsWithContradictions: number;
}

interface RecordDetail {
  id: string;
  kind: string;
  content: string;
  status?: string;
  confidence?: number;
}

@Component({
  selector: "app-contradictions",
  imports: [],
  template: `
    <div class="contradictions-container">
      <header class="page-header">
        <h1>Contradictions</h1>
        <p class="subtitle">Manage conflicting knowledge in your Mind Palace</p>
      </header>

      @if (loading()) {
      <div class="loading">
        <div class="spinner"></div>
        <span>Loading contradictions...</span>
      </div>
      } @else if (error()) {
      <div class="error">
        <span>{{ error() }}</span>
        <button (click)="loadContradictions()">Retry</button>
      </div>
      } @else {
      <!-- Stats -->
      <div class="stats-row">
        <div class="stat-card">
          <div class="stat-value warning">
            {{ summary()?.totalContradictionLinks || 0 }}
          </div>
          <div class="stat-label">Active Contradictions</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">
            {{ summary()?.recordsWithContradictions || 0 }}
          </div>
          <div class="stat-label">Records Affected</div>
        </div>
      </div>

      @if (!summary()?.topContradictions?.length) {
      <div class="empty-state">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
        >
          <path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <h3>No Contradictions Found</h3>
        <p>Your knowledge base is consistent. Great job!</p>
      </div>
      } @else {
      <div class="contradictions-list">
        @for (contradiction of summary()?.topContradictions; track
        contradiction.id) {
        <div
          class="contradiction-card"
          [class.expanded]="expandedId() === contradiction.id"
        >
          <div class="card-header" (click)="toggleExpand(contradiction.id)">
            <div class="conflict-icon">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <path
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                />
              </svg>
            </div>
            <div class="conflict-summary">
              <div class="types">
                <span class="type-badge" [class]="contradiction.sourceKind">{{
                  contradiction.sourceKind
                }}</span>
                <svg
                  class="conflict-arrow"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path d="M8 12h8m-4-4l4 4-4 4" />
                </svg>
                <span class="type-badge" [class]="contradiction.targetKind">{{
                  contradiction.targetKind
                }}</span>
              </div>
              <div class="confidence">
                Confidence: {{ (contradiction.confidence * 100).toFixed(0) }}%
              </div>
            </div>
            <div
              class="expand-icon"
              [class.expanded]="expandedId() === contradiction.id"
            >
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <path d="M19 9l-7 7-7-7" />
              </svg>
            </div>
          </div>

          @if (expandedId() === contradiction.id) {
          <div class="card-content">
            <div class="explanation">
              <strong>Why they conflict:</strong>
              <p>
                {{
                  contradiction.explanation ||
                    "These records contain contradictory information."
                }}
              </p>
            </div>

            <div class="records-comparison">
              <div class="record-side source">
                <div class="record-header">
                  <span class="type-badge" [class]="contradiction.sourceKind">{{
                    contradiction.sourceKind
                  }}</span>
                  <code class="record-id">{{ contradiction.sourceId }}</code>
                </div>
                @if (recordDetails()[contradiction.sourceId]) {
                <div class="record-content">
                  {{ recordDetails()[contradiction.sourceId].content }}
                </div>
                } @else {
                <button
                  class="load-btn"
                  (click)="loadRecord(contradiction.sourceId)"
                >
                  Load content
                </button>
                }
              </div>

              <div class="vs-divider">VS</div>

              <div class="record-side target">
                <div class="record-header">
                  <span class="type-badge" [class]="contradiction.targetKind">{{
                    contradiction.targetKind
                  }}</span>
                  <code class="record-id">{{ contradiction.targetId }}</code>
                </div>
                @if (recordDetails()[contradiction.targetId]) {
                <div class="record-content">
                  {{ recordDetails()[contradiction.targetId].content }}
                </div>
                } @else {
                <button
                  class="load-btn"
                  (click)="loadRecord(contradiction.targetId)"
                >
                  Load content
                </button>
                }
              </div>
            </div>

            <div class="actions">
              <button
                class="action-btn"
                (click)="resolveContradiction(contradiction, 'source')"
              >
                Keep {{ contradiction.sourceKind }}
              </button>
              <button
                class="action-btn"
                (click)="resolveContradiction(contradiction, 'target')"
              >
                Keep {{ contradiction.targetKind }}
              </button>
              <button
                class="action-btn secondary"
                (click)="markResolved(contradiction)"
              >
                Mark as Resolved
              </button>
            </div>

            <div class="meta">
              Detected: {{ formatDate(contradiction.createdAt) }}
            </div>
          </div>
          }
        </div>
        }
      </div>
      } }
    </div>
  `,
  styles: [
    `
      .contradictions-container {
        padding: 1.5rem;
        max-width: 900px;
        margin: 0 auto;
      }

      .page-header {
        margin-bottom: 1.5rem;
      }

      .page-header h1 {
        margin: 0 0 0.25rem 0;
        font-size: 1.5rem;
        color: #e2e8f0;
      }

      .subtitle {
        margin: 0;
        color: #64748b;
        font-size: 0.875rem;
      }

      .loading,
      .error {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 3rem;
        gap: 1rem;
        color: #64748b;
      }

      .spinner {
        width: 32px;
        height: 32px;
        border: 3px solid #2d2d44;
        border-top-color: #9d4edd;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
      }

      @keyframes spin {
        to {
          transform: rotate(360deg);
        }
      }

      .error button {
        padding: 0.5rem 1rem;
        background: #9d4edd;
        color: white;
        border: none;
        border-radius: 4px;
        cursor: pointer;
      }

      .stats-row {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 1rem;
        margin-bottom: 1.5rem;
      }

      .stat-card {
        background: #1a1a2e;
        border: 1px solid #2d2d44;
        border-radius: 8px;
        padding: 1.25rem;
        text-align: center;
      }

      .stat-value {
        font-size: 2rem;
        font-weight: 600;
        color: #e2e8f0;
      }

      .stat-value.warning {
        color: #f87171;
      }

      .stat-label {
        color: #64748b;
        font-size: 0.75rem;
        margin-top: 0.25rem;
        text-transform: uppercase;
      }

      .empty-state {
        text-align: center;
        padding: 3rem;
        color: #64748b;
      }

      .empty-state svg {
        width: 64px;
        height: 64px;
        margin-bottom: 1rem;
        color: #4ade80;
      }

      .empty-state h3 {
        margin: 0 0 0.5rem 0;
        color: #e2e8f0;
      }

      .empty-state p {
        margin: 0;
      }

      .contradictions-list {
        display: flex;
        flex-direction: column;
        gap: 1rem;
      }

      .contradiction-card {
        background: #1a1a2e;
        border: 1px solid #2d2d44;
        border-radius: 8px;
        overflow: hidden;
        transition: all 0.2s;
      }

      .contradiction-card:hover {
        border-color: rgba(248, 113, 113, 0.3);
      }

      .contradiction-card.expanded {
        border-color: rgba(248, 113, 113, 0.5);
      }

      .card-header {
        display: flex;
        align-items: center;
        gap: 1rem;
        padding: 1rem;
        cursor: pointer;
      }

      .conflict-icon {
        width: 32px;
        height: 32px;
        border-radius: 50%;
        background: rgba(248, 113, 113, 0.1);
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
      }

      .conflict-icon svg {
        width: 18px;
        height: 18px;
        color: #f87171;
      }

      .conflict-summary {
        flex: 1;
      }

      .types {
        display: flex;
        align-items: center;
        gap: 0.5rem;
      }

      .conflict-arrow {
        width: 16px;
        height: 16px;
        color: #f87171;
      }

      .type-badge {
        font-size: 0.65rem;
        font-weight: 600;
        text-transform: uppercase;
        padding: 2px 6px;
        border-radius: 3px;
      }

      .type-badge.idea {
        background: rgba(251, 191, 36, 0.2);
        color: #fbbf24;
      }
      .type-badge.decision {
        background: rgba(74, 222, 128, 0.2);
        color: #4ade80;
      }
      .type-badge.learning {
        background: rgba(0, 180, 216, 0.2);
        color: #00b4d8;
      }

      .confidence {
        font-size: 0.75rem;
        color: #64748b;
        margin-top: 0.25rem;
      }

      .expand-icon {
        width: 24px;
        height: 24px;
        color: #64748b;
        transition: transform 0.2s;
      }

      .expand-icon.expanded {
        transform: rotate(180deg);
      }

      .expand-icon svg {
        width: 100%;
        height: 100%;
      }

      .card-content {
        padding: 0 1rem 1rem 1rem;
        border-top: 1px solid #2d2d44;
      }

      .explanation {
        padding: 1rem 0;
        color: #94a3b8;
        font-size: 0.875rem;
      }

      .explanation strong {
        color: #e2e8f0;
        display: block;
        margin-bottom: 0.5rem;
      }

      .explanation p {
        margin: 0;
      }

      .records-comparison {
        display: grid;
        grid-template-columns: 1fr auto 1fr;
        gap: 1rem;
        align-items: stretch;
        margin-bottom: 1rem;
      }

      .record-side {
        background: #0f172a;
        border-radius: 6px;
        padding: 0.75rem;
      }

      .record-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 0.5rem;
      }

      .record-id {
        font-size: 0.65rem;
        color: #64748b;
        background: #1a1a2e;
        padding: 2px 6px;
        border-radius: 3px;
      }

      .record-content {
        font-size: 0.8rem;
        color: #94a3b8;
        line-height: 1.4;
      }

      .load-btn {
        font-size: 0.75rem;
        padding: 0.25rem 0.5rem;
        background: transparent;
        border: 1px solid #2d2d44;
        color: #9d4edd;
        border-radius: 4px;
        cursor: pointer;
      }

      .load-btn:hover {
        background: rgba(157, 78, 221, 0.1);
      }

      .vs-divider {
        display: flex;
        align-items: center;
        font-size: 0.75rem;
        font-weight: 600;
        color: #f87171;
      }

      .actions {
        display: flex;
        gap: 0.5rem;
        flex-wrap: wrap;
      }

      .action-btn {
        padding: 0.5rem 1rem;
        font-size: 0.8rem;
        border: none;
        border-radius: 4px;
        cursor: pointer;
        transition: all 0.2s;
      }

      .action-btn:not(.secondary) {
        background: #4ade80;
        color: #0f172a;
      }

      .action-btn:not(.secondary):hover {
        background: #22c55e;
      }

      .action-btn.secondary {
        background: transparent;
        border: 1px solid #2d2d44;
        color: #94a3b8;
      }

      .action-btn.secondary:hover {
        background: rgba(255, 255, 255, 0.05);
        color: #e2e8f0;
      }

      .meta {
        margin-top: 1rem;
        font-size: 0.7rem;
        color: #64748b;
      }

      @media (max-width: 640px) {
        .records-comparison {
          grid-template-columns: 1fr;
        }

        .vs-divider {
          justify-content: center;
          padding: 0.5rem 0;
        }
      }
    `,
  ],
})
export class ContradictionsComponent implements OnInit {
  private readonly http = inject(HttpClient);

  loading = signal(true);
  error = signal<string | null>(null);
  summary = signal<ContradictionSummary | null>(null);
  expandedId = signal<string | null>(null);
  recordDetails = signal<Record<string, RecordDetail>>({});

  ngOnInit() {
    this.loadContradictions();
  }

  loadContradictions() {
    this.loading.set(true);
    this.error.set(null);

    this.http
      .get<ContradictionSummary>("/api/contradictions?limit=50")
      .subscribe({
        next: (data) => {
          this.summary.set(data);
          this.loading.set(false);
        },
        error: (err) => {
          this.error.set(err.message || "Failed to load contradictions");
          this.loading.set(false);
        },
      });
  }

  toggleExpand(id: string) {
    if (this.expandedId() === id) {
      this.expandedId.set(null);
    } else {
      this.expandedId.set(id);
    }
  }

  loadRecord(id: string) {
    // Determine kind from ID prefix
    let endpoint = "";
    if (id.startsWith("i_")) {
      endpoint = "/api/ideas";
    } else if (id.startsWith("d_")) {
      endpoint = "/api/decisions";
    } else if (id.startsWith("l_")) {
      endpoint = "/api/learnings";
    } else {
      return;
    }

    this.http.get<any>(endpoint + "?limit=1000").subscribe({
      next: (data) => {
        const records = data.ideas || data.decisions || data.learnings || [];
        const record = records.find((r: any) => r.id === id);
        if (record) {
          const current = this.recordDetails();
          this.recordDetails.set({
            ...current,
            [id]: {
              id: record.id,
              kind: id.startsWith("i_")
                ? "idea"
                : id.startsWith("d_")
                ? "decision"
                : "learning",
              content: record.content,
              status: record.status,
              confidence: record.confidence,
            },
          });
        }
      },
    });
  }

  resolveContradiction(
    contradiction: ContradictionLink,
    keep: "source" | "target"
  ) {
    // Archive the other record
    const archiveId =
      keep === "source" ? contradiction.targetId : contradiction.sourceId;
    // In a real implementation, this would call an API to archive the record
    alert(`This would archive ${archiveId} and resolve the contradiction.`);
  }

  markResolved(contradiction: ContradictionLink) {
    // In a real implementation, this would call an API to remove the contradiction link
    alert(
      `This would mark the contradiction as resolved without archiving either record.`
    );
  }

  formatDate(dateStr: string): string {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (days === 0) return "Today";
    if (days === 1) return "Yesterday";
    if (days < 7) return `${days} days ago`;

    return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
  }
}
