import { Component, inject, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService, Postmortem, PostmortemStats } from '../../core/services/api.service';

@Component({
  selector: 'app-postmortems',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="postmortems-page">
      <div class="page-header">
        <h2>Failure Postmortems</h2>
        <p class="subtitle">Learn from failures to prevent recurring issues</p>
      </div>

      <!-- Stats Summary -->
      @if (stats()) {
        <div class="stats-bar">
          <div class="stat">
            <span class="stat-value">{{ stats()!.total }}</span>
            <span class="stat-label">Total</span>
          </div>
          <div class="stat stat-open">
            <span class="stat-value">{{ stats()!.open }}</span>
            <span class="stat-label">Open</span>
          </div>
          <div class="stat stat-resolved">
            <span class="stat-value">{{ stats()!.resolved }}</span>
            <span class="stat-label">Resolved</span>
          </div>
          <div class="stat stat-recurring">
            <span class="stat-value">{{ stats()!.recurring }}</span>
            <span class="stat-label">Recurring</span>
          </div>
        </div>
      }

      <div class="content-layout">
        <!-- Left: List -->
        <div class="list-panel">
          <!-- Filters -->
          <div class="filters">
            <select [(ngModel)]="statusFilter" (change)="loadPostmortems()">
              <option value="">All Status</option>
              <option value="open">Open</option>
              <option value="resolved">Resolved</option>
              <option value="recurring">Recurring</option>
            </select>
            <select [(ngModel)]="severityFilter" (change)="loadPostmortems()">
              <option value="">All Severity</option>
              <option value="critical">Critical</option>
              <option value="high">High</option>
              <option value="medium">Medium</option>
              <option value="low">Low</option>
            </select>
          </div>

          <!-- Loading -->
          @if (loading()) {
            <div class="loading">
              <div class="spinner"></div>
              <span>Loading...</span>
            </div>
          }

          <!-- List -->
          @if (!loading()) {
            <div class="list">
              @for (pm of postmortems(); track pm.id) {
                <div
                  class="list-item"
                  [class.selected]="selectedId() === pm.id"
                  [class]="'severity-' + pm.severity + ' status-' + pm.status"
                  (click)="selectPostmortem(pm.id)"
                >
                  <div class="item-header">
                    <span class="severity-dot" [class]="'dot-' + pm.severity"></span>
                    <span class="status-badge" [class]="'badge-' + pm.status">{{ pm.status }}</span>
                  </div>
                  <h4 class="item-title">{{ pm.title }}</h4>
                  <span class="item-date">{{ formatDate(pm.createdAt) }}</span>
                </div>
              } @empty {
                <div class="empty-list">
                  <p>No postmortems found</p>
                </div>
              }
            </div>
          }
        </div>

        <!-- Right: Detail -->
        <div class="detail-panel">
          @if (selectedPostmortem()) {
            <div class="detail-content">
              <div class="detail-header">
                <h3>{{ selectedPostmortem()!.title }}</h3>
                <div class="header-badges">
                  <span class="severity-badge" [class]="'severity-' + selectedPostmortem()!.severity">
                    {{ selectedPostmortem()!.severity | uppercase }}
                  </span>
                  <span class="status-badge" [class]="'badge-' + selectedPostmortem()!.status">
                    {{ selectedPostmortem()!.status }}
                  </span>
                </div>
              </div>

              <div class="detail-meta">
                <span>Created: {{ formatDate(selectedPostmortem()!.createdAt) }}</span>
                @if (selectedPostmortem()!.resolvedAt) {
                  <span>Resolved: {{ formatDate(selectedPostmortem()!.resolvedAt) }}</span>
                }
              </div>

              <div class="detail-section">
                <h4>What Happened</h4>
                <p>{{ selectedPostmortem()!.whatHappened }}</p>
              </div>

              @if (selectedPostmortem()!.rootCause) {
                <div class="detail-section">
                  <h4>Root Cause</h4>
                  <p>{{ selectedPostmortem()!.rootCause }}</p>
                </div>
              }

              @if (selectedPostmortem()!.lessonsLearned?.length) {
                <div class="detail-section">
                  <h4>Lessons Learned</h4>
                  <ul class="lessons-list">
                    @for (lesson of selectedPostmortem()!.lessonsLearned; track lesson) {
                      <li>{{ lesson }}</li>
                    }
                  </ul>
                </div>
              }

              @if (selectedPostmortem()!.preventionSteps?.length) {
                <div class="detail-section">
                  <h4>Prevention Steps</h4>
                  <ul class="prevention-list">
                    @for (step of selectedPostmortem()!.preventionSteps; track step) {
                      <li>{{ step }}</li>
                    }
                  </ul>
                </div>
              }

              @if (selectedPostmortem()!.affectedFiles?.length) {
                <div class="detail-section">
                  <h4>Affected Files</h4>
                  <div class="file-list">
                    @for (file of selectedPostmortem()!.affectedFiles; track file) {
                      <span class="file-tag">{{ file }}</span>
                    }
                  </div>
                </div>
              }

              @if (selectedPostmortem()!.relatedDecision) {
                <div class="detail-section">
                  <h4>Related Decision</h4>
                  <span class="relation-link">{{ selectedPostmortem()!.relatedDecision }}</span>
                </div>
              }

              <!-- Actions -->
              <div class="detail-actions">
                @if (selectedPostmortem()!.status === 'open') {
                  <button class="btn-resolve" (click)="resolvePostmortem()" [disabled]="actionLoading()">
                    @if (actionLoading()) {
                      <span class="spinner-inline"></span>
                    }
                    Mark as Resolved
                  </button>
                }
                @if (selectedPostmortem()!.lessonsLearned?.length) {
                  <button class="btn-convert" (click)="convertToLearnings()" [disabled]="actionLoading()">
                    @if (actionLoading()) {
                      <span class="spinner-inline"></span>
                    }
                    Convert to Learnings
                  </button>
                }
              </div>

              @if (actionMessage()) {
                <div class="action-message" [class.success]="actionSuccess()">
                  {{ actionMessage() }}
                </div>
              }
            </div>
          } @else {
            <div class="empty-detail">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <path d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
              </svg>
              <h3>Select a Postmortem</h3>
              <p>Click on a postmortem from the list to view details</p>
            </div>
          }
        </div>
      </div>
    </div>
  `,
  styles: [`
    .postmortems-page {
      height: 100%;
      display: flex;
      flex-direction: column;
    }

    .page-header {
      margin-bottom: 1rem;
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

    .stats-bar {
      display: flex;
      gap: 1rem;
      margin-bottom: 1rem;
    }

    .stat {
      background: #1a1a2e;
      border-radius: 8px;
      padding: 0.75rem 1.25rem;
      border: 1px solid #2d2d44;
      display: flex;
      flex-direction: column;
      align-items: center;
    }

    .stat-value {
      font-size: 1.5rem;
      font-weight: 600;
      color: #e2e8f0;
    }

    .stat-label {
      font-size: 0.75rem;
      color: #64748b;
      text-transform: uppercase;
    }

    .stat-open .stat-value { color: #ef4444; }
    .stat-resolved .stat-value { color: #22c55e; }
    .stat-recurring .stat-value { color: #f59e0b; }

    .content-layout {
      display: grid;
      grid-template-columns: 320px 1fr;
      gap: 1rem;
      flex: 1;
      min-height: 0;
    }

    .list-panel {
      background: #1a1a2e;
      border-radius: 8px;
      border: 1px solid #2d2d44;
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }

    .filters {
      display: flex;
      gap: 0.5rem;
      padding: 0.75rem;
      border-bottom: 1px solid #2d2d44;
    }

    .filters select {
      flex: 1;
      background: #2d2d44;
      border: 1px solid #3d3d54;
      border-radius: 6px;
      padding: 0.5rem;
      color: #e2e8f0;
      font-size: 0.8rem;
    }

    .loading {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 0.5rem;
      padding: 2rem;
      color: #64748b;
    }

    .spinner {
      width: 20px;
      height: 20px;
      border: 2px solid #3d3d54;
      border-top-color: #9d4edd;
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
      to { transform: rotate(360deg); }
    }

    .list {
      flex: 1;
      overflow-y: auto;
    }

    .list-item {
      padding: 0.75rem 1rem;
      border-bottom: 1px solid #2d2d44;
      cursor: pointer;
      transition: background 0.15s;
    }

    .list-item:hover {
      background: rgba(45, 45, 68, 0.5);
    }

    .list-item.selected {
      background: rgba(157, 78, 221, 0.15);
      border-left: 3px solid #9d4edd;
    }

    .item-header {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      margin-bottom: 0.35rem;
    }

    .severity-dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
    }

    .dot-critical { background: #ef4444; }
    .dot-high { background: #f97316; }
    .dot-medium { background: #eab308; }
    .dot-low { background: #64748b; }

    .status-badge {
      font-size: 0.6rem;
      font-weight: 600;
      text-transform: uppercase;
      padding: 0.15rem 0.4rem;
      border-radius: 4px;
    }

    .badge-open {
      background: rgba(239, 68, 68, 0.15);
      color: #ef4444;
    }

    .badge-resolved {
      background: rgba(34, 197, 94, 0.15);
      color: #22c55e;
    }

    .badge-recurring {
      background: rgba(245, 158, 11, 0.15);
      color: #f59e0b;
    }

    .item-title {
      font-size: 0.875rem;
      color: #e2e8f0;
      margin: 0 0 0.25rem 0;
      font-weight: 500;
    }

    .item-date {
      font-size: 0.7rem;
      color: #64748b;
    }

    .empty-list {
      padding: 2rem;
      text-align: center;
      color: #64748b;
    }

    .detail-panel {
      background: #1a1a2e;
      border-radius: 8px;
      border: 1px solid #2d2d44;
      overflow-y: auto;
    }

    .detail-content {
      padding: 1.5rem;
    }

    .detail-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      margin-bottom: 1rem;
    }

    .detail-header h3 {
      font-size: 1.25rem;
      color: #e2e8f0;
      margin: 0;
      flex: 1;
    }

    .header-badges {
      display: flex;
      gap: 0.5rem;
      flex-shrink: 0;
    }

    .severity-badge {
      font-size: 0.65rem;
      font-weight: 700;
      padding: 0.25rem 0.5rem;
      border-radius: 4px;
    }

    .severity-critical {
      background: rgba(239, 68, 68, 0.2);
      color: #ef4444;
    }

    .severity-high {
      background: rgba(249, 115, 22, 0.2);
      color: #f97316;
    }

    .severity-medium {
      background: rgba(234, 179, 8, 0.2);
      color: #eab308;
    }

    .severity-low {
      background: rgba(100, 116, 139, 0.2);
      color: #64748b;
    }

    .detail-meta {
      display: flex;
      gap: 1.5rem;
      font-size: 0.8rem;
      color: #64748b;
      margin-bottom: 1.5rem;
      padding-bottom: 1rem;
      border-bottom: 1px solid #2d2d44;
    }

    .detail-section {
      margin-bottom: 1.5rem;
    }

    .detail-section h4 {
      font-size: 0.8rem;
      font-weight: 600;
      color: #94a3b8;
      text-transform: uppercase;
      margin: 0 0 0.5rem 0;
    }

    .detail-section p {
      color: #e2e8f0;
      line-height: 1.6;
      margin: 0;
      font-size: 0.95rem;
    }

    .lessons-list, .prevention-list {
      margin: 0;
      padding-left: 1.25rem;
      color: #e2e8f0;
      line-height: 1.8;
    }

    .lessons-list li::marker {
      color: #22c55e;
    }

    .prevention-list li::marker {
      color: #3b82f6;
    }

    .file-list {
      display: flex;
      flex-wrap: wrap;
      gap: 0.5rem;
    }

    .file-tag {
      background: #2d2d44;
      padding: 0.25rem 0.5rem;
      border-radius: 4px;
      font-family: monospace;
      font-size: 0.8rem;
      color: #94a3b8;
    }

    .relation-link {
      color: #9d4edd;
      font-family: monospace;
      font-size: 0.85rem;
    }

    .detail-actions {
      display: flex;
      gap: 0.75rem;
      margin-top: 2rem;
      padding-top: 1rem;
      border-top: 1px solid #2d2d44;
    }

    .btn-resolve, .btn-convert {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.6rem 1rem;
      border-radius: 6px;
      font-size: 0.85rem;
      font-weight: 500;
      cursor: pointer;
      border: none;
      transition: background 0.2s;
    }

    .btn-resolve {
      background: #22c55e;
      color: white;
    }

    .btn-resolve:hover:not(:disabled) {
      background: #16a34a;
    }

    .btn-convert {
      background: #3b82f6;
      color: white;
    }

    .btn-convert:hover:not(:disabled) {
      background: #2563eb;
    }

    .btn-resolve:disabled, .btn-convert:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }

    .spinner-inline {
      width: 14px;
      height: 14px;
      border: 2px solid rgba(255,255,255,0.3);
      border-top-color: white;
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
    }

    .action-message {
      margin-top: 1rem;
      padding: 0.75rem;
      border-radius: 6px;
      font-size: 0.85rem;
      background: rgba(239, 68, 68, 0.1);
      color: #ef4444;
    }

    .action-message.success {
      background: rgba(34, 197, 94, 0.1);
      color: #22c55e;
    }

    .empty-detail {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      height: 100%;
      padding: 2rem;
      text-align: center;
      color: #64748b;
    }

    .empty-detail svg {
      width: 64px;
      height: 64px;
      margin-bottom: 1rem;
      opacity: 0.5;
    }

    .empty-detail h3 {
      color: #94a3b8;
      margin: 0 0 0.5rem 0;
    }

    .empty-detail p {
      margin: 0;
      font-size: 0.875rem;
    }
  `]
})
export class PostmortemsComponent implements OnInit {
  private api = inject(ApiService);

  postmortems = signal<Postmortem[]>([]);
  selectedPostmortem = signal<Postmortem | null>(null);
  selectedId = signal<string | null>(null);
  stats = signal<PostmortemStats | null>(null);
  loading = signal(true);
  actionLoading = signal(false);
  actionMessage = signal<string | null>(null);
  actionSuccess = signal(false);

  statusFilter = '';
  severityFilter = '';

  ngOnInit() {
    this.loadPostmortems();
    this.loadStats();
  }

  loadPostmortems() {
    this.loading.set(true);
    this.api.getPostmortems(this.statusFilter, this.severityFilter, 100).subscribe({
      next: (res) => {
        this.postmortems.set(res.postmortems || []);
        this.loading.set(false);
        // Re-select if still in list
        if (this.selectedId() && res.postmortems) {
          const found = res.postmortems.find(p => p.id === this.selectedId());
          this.selectedPostmortem.set(found || null);
        }
      },
      error: () => {
        this.postmortems.set([]);
        this.loading.set(false);
      }
    });
  }

  loadStats() {
    this.api.getPostmortemStats().subscribe({
      next: (res) => this.stats.set(res),
      error: () => this.stats.set(null)
    });
  }

  selectPostmortem(id: string) {
    this.selectedId.set(id);
    this.actionMessage.set(null);
    this.api.getPostmortem(id).subscribe({
      next: (pm) => this.selectedPostmortem.set(pm),
      error: () => this.selectedPostmortem.set(null)
    });
  }

  resolvePostmortem() {
    const pm = this.selectedPostmortem();
    if (!pm) return;

    this.actionLoading.set(true);
    this.actionMessage.set(null);

    this.api.resolvePostmortem(pm.id).subscribe({
      next: (updated) => {
        this.selectedPostmortem.set(updated);
        this.actionMessage.set('Postmortem marked as resolved');
        this.actionSuccess.set(true);
        this.actionLoading.set(false);
        this.loadPostmortems();
        this.loadStats();
      },
      error: (err) => {
        this.actionMessage.set(err.error?.error || 'Failed to resolve');
        this.actionSuccess.set(false);
        this.actionLoading.set(false);
      }
    });
  }

  convertToLearnings() {
    const pm = this.selectedPostmortem();
    if (!pm) return;

    this.actionLoading.set(true);
    this.actionMessage.set(null);

    this.api.convertPostmortemToLearnings(pm.id).subscribe({
      next: (res) => {
        this.actionMessage.set(`Created ${res.created} learning(s)`);
        this.actionSuccess.set(true);
        this.actionLoading.set(false);
      },
      error: (err) => {
        this.actionMessage.set(err.error?.error || 'Failed to convert');
        this.actionSuccess.set(false);
        this.actionLoading.set(false);
      }
    });
  }

  formatDate(dateStr: string): string {
    if (!dateStr) return '';
    const date = new Date(dateStr);
    return date.toLocaleDateString();
  }
}
