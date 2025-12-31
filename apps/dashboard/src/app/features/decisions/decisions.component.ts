import { Component, inject, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService, Decision } from '../../core/services/api.service';

@Component({
  selector: 'app-decisions',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="decisions-page">
      <div class="page-header">
        <h2>Decisions</h2>
        <p class="subtitle">Track architectural decisions and their outcomes</p>
      </div>

      <!-- Filters -->
      <div class="filters">
        <select [(ngModel)]="statusFilter" (change)="loadDecisions()">
          <option value="">All Status</option>
          <option value="active">Active</option>
          <option value="superseded">Superseded</option>
          <option value="reversed">Reversed</option>
        </select>
        <span class="count">{{ decisions().length }} decisions</span>
      </div>

      <!-- Loading -->
      @if (loading()) {
        <div class="loading">
          <div class="spinner"></div>
          <span>Loading decisions...</span>
        </div>
      }

      <!-- Decisions List -->
      @if (!loading() && decisions().length > 0) {
        <div class="decisions-list">
          @for (decision of decisions(); track decision.id) {
            <div class="decision-card" [class]="'status-' + decision.status">
              <div class="decision-header">
                <span class="status-badge" [class]="'badge-' + decision.status">{{ decision.status }}</span>
                <span class="scope" [title]="decision.scopePath">{{ decision.scope || 'Global' }}</span>
              </div>
              <p class="decision-content">{{ decision.content }}</p>
              @if (decision.rationale) {
                <div class="rationale">
                  <strong>Rationale:</strong> {{ decision.rationale }}
                </div>
              }
              <div class="decision-footer">
                <span class="date">{{ formatDate(decision.createdAt) }}</span>
                @if (decision.tags?.length) {
                  <div class="tags">
                    @for (tag of decision.tags.slice(0, 3); track tag) {
                      <span class="tag">{{ tag }}</span>
                    }
                  </div>
                }
              </div>
            </div>
          }
        </div>
      }

      <!-- Empty State -->
      @if (!loading() && decisions().length === 0) {
        <div class="empty-state">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M9 11l3 3L22 4"/>
            <path d="M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11"/>
          </svg>
          <h3>No decisions recorded</h3>
          <p>Decisions will appear here as you make architectural choices</p>
        </div>
      }
    </div>
  `,
  styles: [`
    .decisions-page {
      max-width: 900px;
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
      transition: border-color 0.2s;
    }

    .decision-card:hover {
      border-color: #3d3d54;
    }

    .decision-card.status-active { border-left: 3px solid #22c55e; }
    .decision-card.status-superseded { border-left: 3px solid #f59e0b; }
    .decision-card.status-reversed { border-left: 3px solid #ef4444; }

    .decision-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 0.75rem;
    }

    .status-badge {
      font-size: 0.7rem;
      font-weight: 600;
      text-transform: uppercase;
      padding: 0.2rem 0.5rem;
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
      max-width: 200px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .decision-content {
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

    .decision-footer {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .date {
      font-size: 0.75rem;
      color: #64748b;
    }

    .tags {
      display: flex;
      gap: 0.25rem;
    }

    .tag {
      font-size: 0.65rem;
      padding: 0.15rem 0.4rem;
      background: #2d2d44;
      border-radius: 4px;
      color: #94a3b8;
    }

    .empty-state {
      text-align: center;
      padding: 4rem 2rem;
      color: #64748b;
    }

    .empty-state svg {
      width: 64px;
      height: 64px;
      margin-bottom: 1rem;
      opacity: 0.5;
    }

    .empty-state h3 {
      color: #94a3b8;
      margin: 0 0 0.5rem 0;
    }

    .empty-state p {
      margin: 0;
      font-size: 0.875rem;
    }
  `]
})
export class DecisionsComponent implements OnInit {
  private api = inject(ApiService);

  decisions = signal<Decision[]>([]);
  loading = signal(true);
  statusFilter = '';

  ngOnInit() {
    this.loadDecisions();
  }

  loadDecisions() {
    this.loading.set(true);
    this.api.getDecisions(this.statusFilter, '', 100).subscribe({
      next: (res) => {
        this.decisions.set(res.decisions || []);
        this.loading.set(false);
      },
      error: () => {
        this.decisions.set([]);
        this.loading.set(false);
      }
    });
  }

  formatDate(dateStr: string): string {
    if (!dateStr) return '';
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (days === 0) return 'Today';
    if (days === 1) return 'Yesterday';
    if (days < 7) return `${days} days ago`;
    return date.toLocaleDateString();
  }
}
