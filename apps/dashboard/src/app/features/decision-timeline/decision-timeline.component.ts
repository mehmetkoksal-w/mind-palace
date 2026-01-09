import { Component, inject, OnInit, signal } from "@angular/core";
import { CommonModule } from "@angular/common";
import { FormsModule } from "@angular/forms";
import {
  ApiService,
  TimelineDecision,
  DecisionChain,
  Learning,
} from "../../core/services/api.service";

@Component({
  selector: "app-decision-timeline",
  imports: [CommonModule, FormsModule],
  template: `
    <div class="decision-timeline-page">
      <div class="page-header">
        <h2>Decision Timeline</h2>
        <p class="subtitle">
          Track the evolution of architectural decisions over time
        </p>
      </div>

      <!-- Filters -->
      <div class="filters">
        <select [(ngModel)]="scopeFilter" (change)="loadTimeline()">
          <option value="">All Scopes</option>
          <option value="file">File</option>
          <option value="room">Room</option>
          <option value="palace">Palace</option>
        </select>
        <span class="count">{{ decisions().length }} decisions</span>
      </div>

      <!-- Loading -->
      @if (loading()) {
      <div class="loading">
        <div class="spinner"></div>
        <span>Loading timeline...</span>
      </div>
      }

      <!-- Timeline View -->
      @if (!loading() && decisions().length > 0) {
      <!-- Timeline Track -->
      <div class="timeline-track">
        <div class="track-line"></div>
        <div class="track-dots">
          @for (decision of decisions(); track decision.id; let i = $index) {
          <div
            class="track-dot"
            [class]="'dot-' + decision.outcomeColor"
            [class.selected]="selectedId() === decision.id"
            [style.left.%]="getPosition(i)"
            (click)="selectDecision(decision)"
            [title]="decision.content | slice : 0 : 50"
          ></div>
          }
        </div>
        <div class="track-dates">
          @for (date of timelineDates(); track date) {
          <span>{{ date }}</span>
          }
        </div>
      </div>

      <!-- Legend -->
      <div class="legend">
        <span class="legend-item"
          ><span class="dot dot-green"></span> Success</span
        >
        <span class="legend-item"
          ><span class="dot dot-red"></span> Failed</span
        >
        <span class="legend-item"
          ><span class="dot dot-yellow"></span> Mixed</span
        >
        <span class="legend-item"
          ><span class="dot dot-gray"></span> Unknown</span
        >
      </div>

      <!-- Decision Cards -->
      <div class="decisions-list">
        @for (decision of decisions(); track decision.id) {
        <div
          class="decision-card"
          [class.selected]="selectedId() === decision.id"
          [class]="'outcome-' + decision.outcomeColor"
          (click)="selectDecision(decision)"
        >
          <div class="card-header">
            <span
              class="outcome-dot"
              [class]="'dot-' + decision.outcomeColor"
            ></span>
            <span class="status-badge" [class]="'badge-' + decision.status">{{
              decision.status
            }}</span>
            <span class="scope">{{ decision.scope }}</span>
            <span class="date">{{ formatDate(decision.createdAt) }}</span>
          </div>
          <p class="content">{{ decision.content }}</p>
          @if (decision.rationale) {
          <div class="rationale">
            <strong>Rationale:</strong> {{ decision.rationale }}
          </div>
          } @if (decision.tags.length) {
          <div class="tags">
            @for (tag of decision.tags.slice(0, 4); track tag) {
            <span class="tag">{{ tag }}</span>
            }
          </div>
          } @if (isOldAndUnknown(decision)) {
          <div class="review-warning">
            <span class="warning-icon">!</span>
            Review needed: outcome unknown after 90+ days
          </div>
          }
          <button
            class="btn-chain"
            (click)="loadChain(decision.id); $event.stopPropagation()"
          >
            View Chain
          </button>
        </div>
        }
      </div>
      }

      <!-- Empty State -->
      @if (!loading() && decisions().length === 0) {
      <div class="empty-state">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
        >
          <path d="M9 11l3 3L22 4" />
          <path d="M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11" />
        </svg>
        <h3>No decisions recorded</h3>
        <p>Decisions will appear here as you make architectural choices</p>
      </div>
      }

      <!-- Chain Modal -->
      @if (showChainModal() && chain()) {
      <div class="modal-overlay" (click)="closeChain()">
        <div class="modal-content" (click)="$event.stopPropagation()">
          <div class="modal-header">
            <h3>Decision Chain</h3>
            <button class="close-btn" (click)="closeChain()">&times;</button>
          </div>

          <div class="chain-view">
            <!-- Predecessors -->
            @if (chain()!.predecessors.length > 0) {
            <div class="chain-section">
              <h4>Superseded Decisions</h4>
              @for (pred of chain()!.predecessors; track pred.decision.id) {
              <div class="chain-item predecessor">
                <span class="chain-relation">supersedes</span>
                <div class="chain-decision">
                  <span class="chain-title">{{ pred.decision.content }}</span>
                  <span class="chain-date">{{
                    formatDate(pred.decision.createdAt)
                  }}</span>
                </div>
              </div>
              }
              <div class="chain-arrow">↓</div>
            </div>
            }

            <!-- Current -->
            <div class="chain-current">
              <span class="current-label">Current Decision</span>
              <div class="current-decision">
                <p>{{ chain()!.current.content }}</p>
                @if (chain()!.current.rationale) {
                <div class="current-rationale">
                  {{ chain()!.current.rationale }}
                </div>
                }
              </div>
            </div>

            <!-- Successors -->
            @if (chain()!.successors.length > 0) {
            <div class="chain-section">
              <div class="chain-arrow">↓</div>
              <h4>Superseded By</h4>
              @for (succ of chain()!.successors; track succ.decision.id) {
              <div class="chain-item successor">
                <span class="chain-relation">superseded by</span>
                <div class="chain-decision">
                  <span class="chain-title">{{ succ.decision.content }}</span>
                  <span class="chain-date">{{
                    formatDate(succ.decision.createdAt)
                  }}</span>
                </div>
              </div>
              }
            </div>
            }

            <!-- Linked Learnings -->
            @if (chain()!.linkedLearnings.length > 0) {
            <div class="chain-learnings">
              <h4>Linked Learnings ({{ chain()!.linkedLearnings.length }})</h4>
              @for (learning of chain()!.linkedLearnings; track learning.id) {
              <div class="learning-item">
                <span class="learning-content">{{ learning.content }}</span>
                <span class="learning-confidence"
                  >{{ (learning.confidence * 100).toFixed(0) }}%</span
                >
              </div>
              }
            </div>
            }
          </div>
        </div>
      </div>
      }
    </div>
  `,
  styles: [
    `
      .decision-timeline-page {
        max-width: 1000px;
      }

      .page-header {
        margin-bottom: 1.5rem;
      }

      .page-header h2 {
        font-size: 1.5rem;
        font-weight: 600;
        color: #e2e8f0;
        margin: 0 0 0.25rem 0;
      }

      .subtitle {
        color: #64748b;
        margin: 0;
        font-size: 0.875rem;
      }

      .filters {
        display: flex;
        align-items: center;
        gap: 1rem;
        margin-bottom: 1.5rem;
      }

      .filters select {
        background: #2d2d44;
        border: 1px solid #3d3d54;
        border-radius: 6px;
        padding: 0.5rem 1rem;
        color: #e2e8f0;
        font-size: 0.875rem;
      }

      .count {
        color: #64748b;
        font-size: 0.875rem;
      }

      .loading {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 0.75rem;
        padding: 3rem;
        color: #64748b;
      }

      /* spinner styles in global styles.scss */

      .timeline-track {
        background: #1a1a2e;
        border-radius: 8px;
        padding: 1.5rem;
        margin-bottom: 1rem;
        border: 1px solid #2d2d44;
        position: relative;
      }

      .track-line {
        height: 4px;
        background: linear-gradient(90deg, #3d3d54, #9d4edd);
        border-radius: 2px;
        margin: 1rem 2rem;
      }

      .track-dots {
        position: relative;
        height: 30px;
        margin: 0 2rem;
      }

      .track-dot {
        position: absolute;
        width: 14px;
        height: 14px;
        border-radius: 50%;
        top: 50%;
        transform: translate(-50%, -50%);
        cursor: pointer;
        transition: transform 0.2s, box-shadow 0.2s;
        border: 2px solid #1a1a2e;
      }

      .track-dot:hover {
        transform: translate(-50%, -50%) scale(1.3);
        z-index: 10;
      }

      .track-dot.selected {
        transform: translate(-50%, -50%) scale(1.5);
        box-shadow: 0 0 0 4px rgba(157, 78, 221, 0.3);
        z-index: 10;
      }

      .dot-green {
        background: #22c55e;
      }
      .dot-red {
        background: #ef4444;
      }
      .dot-yellow {
        background: #f59e0b;
      }
      .dot-gray {
        background: #64748b;
      }

      .track-dates {
        display: flex;
        justify-content: space-between;
        margin: 0.5rem 1rem 0;
        font-size: 0.7rem;
        color: #64748b;
      }

      .legend {
        display: flex;
        justify-content: center;
        gap: 1.5rem;
        margin-bottom: 1.5rem;
      }

      .legend-item {
        display: flex;
        align-items: center;
        gap: 0.35rem;
        font-size: 0.75rem;
        color: #94a3b8;
      }

      .dot {
        width: 10px;
        height: 10px;
        border-radius: 50%;
      }

      .decisions-list {
        display: flex;
        flex-direction: column;
        gap: 1rem;
      }

      .decision-card {
        background: #1a1a2e;
        border-radius: 8px;
        padding: 1.25rem;
        border: 1px solid #2d2d44;
        cursor: pointer;
        transition: border-color 0.2s, background 0.2s;
        position: relative;
      }

      .decision-card:hover {
        border-color: #3d3d54;
      }

      .decision-card.selected {
        border-color: #9d4edd;
        background: rgba(157, 78, 221, 0.05);
      }

      .decision-card.outcome-green {
        border-left: 4px solid #22c55e;
      }
      .decision-card.outcome-red {
        border-left: 4px solid #ef4444;
      }
      .decision-card.outcome-yellow {
        border-left: 4px solid #f59e0b;
      }
      .decision-card.outcome-gray {
        border-left: 4px solid #64748b;
      }

      .card-header {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        margin-bottom: 0.75rem;
      }

      .outcome-dot {
        width: 10px;
        height: 10px;
        border-radius: 50%;
      }

      .status-badge {
        font-size: 0.65rem;
        font-weight: 600;
        text-transform: uppercase;
        padding: 0.15rem 0.4rem;
        border-radius: 4px;
      }

      .badge-active {
        background: rgba(34, 197, 94, 0.15);
        color: #22c55e;
      }

      .badge-superseded {
        background: rgba(245, 158, 11, 0.15);
        color: #f59e0b;
      }

      .badge-reversed {
        background: rgba(239, 68, 68, 0.15);
        color: #ef4444;
      }

      .scope {
        font-size: 0.75rem;
        color: #64748b;
      }

      .date {
        font-size: 0.75rem;
        color: #64748b;
        margin-left: auto;
      }

      .content {
        color: #e2e8f0;
        font-size: 0.95rem;
        line-height: 1.5;
        margin: 0 0 0.75rem 0;
      }

      .rationale {
        font-size: 0.85rem;
        color: #94a3b8;
        background: rgba(45, 45, 68, 0.5);
        padding: 0.75rem;
        border-radius: 6px;
        margin-bottom: 0.75rem;
        line-height: 1.4;
      }

      .rationale strong {
        color: #e2e8f0;
      }

      .tags {
        display: flex;
        gap: 0.25rem;
        margin-bottom: 0.75rem;
      }

      .tag {
        font-size: 0.65rem;
        padding: 0.15rem 0.4rem;
        background: #2d2d44;
        border-radius: 4px;
        color: #94a3b8;
      }

      .review-warning {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        background: rgba(245, 158, 11, 0.1);
        border: 1px solid rgba(245, 158, 11, 0.3);
        border-radius: 6px;
        padding: 0.5rem 0.75rem;
        font-size: 0.8rem;
        color: #f59e0b;
        margin-bottom: 0.75rem;
      }

      .warning-icon {
        font-weight: bold;
        background: #f59e0b;
        color: #1a1a2e;
        width: 18px;
        height: 18px;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 0.75rem;
      }

      .btn-chain {
        background: rgba(157, 78, 221, 0.15);
        border: 1px solid rgba(157, 78, 221, 0.3);
        color: #9d4edd;
        padding: 0.4rem 0.75rem;
        border-radius: 4px;
        font-size: 0.8rem;
        cursor: pointer;
        transition: background 0.2s;
      }

      .btn-chain:hover {
        background: rgba(157, 78, 221, 0.25);
      }

      /* empty-state styles in global styles.scss */

      /* Modal */
      .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.7);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
      }

      .modal-content {
        background: #1a1a2e;
        border-radius: 12px;
        border: 1px solid #2d2d44;
        width: 90%;
        max-width: 600px;
        max-height: 80vh;
        overflow-y: auto;
      }

      .modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 1rem 1.25rem;
        border-bottom: 1px solid #2d2d44;
      }

      .modal-header h3 {
        margin: 0;
        color: #e2e8f0;
        font-size: 1.1rem;
      }

      .close-btn {
        background: none;
        border: none;
        color: #64748b;
        font-size: 1.5rem;
        cursor: pointer;
        padding: 0;
        line-height: 1;
      }

      .close-btn:hover {
        color: #e2e8f0;
      }

      .chain-view {
        padding: 1.25rem;
      }

      .chain-section h4 {
        font-size: 0.8rem;
        color: #64748b;
        text-transform: uppercase;
        margin: 0 0 0.75rem 0;
      }

      .chain-item {
        background: #2d2d44;
        border-radius: 6px;
        padding: 0.75rem;
        margin-bottom: 0.5rem;
      }

      .chain-relation {
        font-size: 0.7rem;
        color: #9d4edd;
        text-transform: uppercase;
      }

      .chain-decision {
        margin-top: 0.25rem;
      }

      .chain-title {
        display: block;
        color: #e2e8f0;
        font-size: 0.9rem;
      }

      .chain-date {
        font-size: 0.75rem;
        color: #64748b;
      }

      .chain-arrow {
        text-align: center;
        color: #64748b;
        font-size: 1.25rem;
        padding: 0.5rem;
      }

      .chain-current {
        background: rgba(157, 78, 221, 0.1);
        border: 1px solid rgba(157, 78, 221, 0.3);
        border-radius: 8px;
        padding: 1rem;
        margin: 1rem 0;
      }

      .current-label {
        font-size: 0.7rem;
        color: #9d4edd;
        text-transform: uppercase;
        font-weight: 600;
      }

      .current-decision p {
        color: #e2e8f0;
        margin: 0.5rem 0;
        font-size: 0.95rem;
      }

      .current-rationale {
        font-size: 0.85rem;
        color: #94a3b8;
        background: rgba(45, 45, 68, 0.5);
        padding: 0.5rem;
        border-radius: 4px;
        margin-top: 0.5rem;
      }

      .chain-learnings {
        margin-top: 1.5rem;
        padding-top: 1rem;
        border-top: 1px solid #2d2d44;
      }

      .chain-learnings h4 {
        font-size: 0.8rem;
        color: #64748b;
        text-transform: uppercase;
        margin: 0 0 0.75rem 0;
      }

      .learning-item {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        padding: 0.5rem;
        background: #2d2d44;
        border-radius: 4px;
        margin-bottom: 0.5rem;
      }

      .learning-content {
        flex: 1;
        color: #e2e8f0;
        font-size: 0.85rem;
      }

      .learning-confidence {
        color: #22c55e;
        font-size: 0.75rem;
        font-weight: 600;
        margin-left: 0.5rem;
      }
    `,
  ],
})
export class DecisionTimelineComponent implements OnInit {
  private api = inject(ApiService);

  decisions = signal<TimelineDecision[]>([]);
  loading = signal(true);
  selectedId = signal<string | null>(null);
  scopeFilter = "";

  showChainModal = signal(false);
  chain = signal<DecisionChain | null>(null);

  timelineDates = signal<string[]>([]);

  ngOnInit() {
    this.loadTimeline();
  }

  loadTimeline() {
    this.loading.set(true);
    this.api.getDecisionTimeline(this.scopeFilter, 100).subscribe({
      next: (res) => {
        this.decisions.set(res.decisions || []);
        this.calculateTimelineDates(res.decisions || []);
        this.loading.set(false);
      },
      error: () => {
        this.decisions.set([]);
        this.loading.set(false);
      },
    });
  }

  calculateTimelineDates(decisions: TimelineDecision[]) {
    if (decisions.length === 0) {
      this.timelineDates.set([]);
      return;
    }

    const dates = decisions.map((d) => new Date(d.createdAt).getTime());
    const minDate = new Date(Math.min(...dates));
    const maxDate = new Date(Math.max(...dates));

    const months: string[] = [];
    const current = new Date(minDate);
    while (current <= maxDate) {
      months.push(
        current.toLocaleDateString("en-US", { month: "short", year: "2-digit" })
      );
      current.setMonth(current.getMonth() + 1);
    }

    // Limit to 6 evenly spaced labels
    if (months.length > 6) {
      const step = Math.ceil(months.length / 6);
      const filtered = months.filter((_, i) => i % step === 0);
      this.timelineDates.set(filtered.slice(0, 6));
    } else {
      this.timelineDates.set(months);
    }
  }

  getPosition(index: number): number {
    const total = this.decisions().length;
    if (total <= 1) return 50;
    return (index / (total - 1)) * 100;
  }

  selectDecision(decision: TimelineDecision) {
    this.selectedId.set(decision.id);
  }

  loadChain(id: string) {
    this.api.getDecisionChain(id).subscribe({
      next: (res) => {
        this.chain.set(res);
        this.showChainModal.set(true);
      },
      error: () => {
        this.chain.set(null);
      },
    });
  }

  closeChain() {
    this.showChainModal.set(false);
  }

  isOldAndUnknown(decision: TimelineDecision): boolean {
    if (decision.outcomeColor !== "gray") return false;
    const created = new Date(decision.createdAt);
    const daysSince = (Date.now() - created.getTime()) / (1000 * 60 * 60 * 24);
    return daysSince > 90;
  }

  formatDate(dateStr: string): string {
    if (!dateStr) return "";
    const date = new Date(dateStr);
    return date.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
    });
  }
}
