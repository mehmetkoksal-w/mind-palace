import { Component, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService, AutoInjectedContext, PrioritizedLearning, Decision } from '../../core/services/api.service';

@Component({
  selector: 'app-context-preview',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="context-preview-page">
      <div class="page-header">
        <h2>AI Context Preview</h2>
        <p class="subtitle">Preview what context AI agents will receive when working with files</p>
      </div>

      <!-- Input Section -->
      <div class="input-section">
        <div class="input-row">
          <input
            type="text"
            [(ngModel)]="filePath"
            placeholder="Enter file path (e.g., src/auth/jwt.go)"
            (keyup.enter)="previewContext()"
          />
          <button (click)="previewContext()" [disabled]="loading() || !filePath">
            @if (loading()) {
              <span class="spinner-inline"></span>
            } @else {
              Preview
            }
          </button>
        </div>

        <!-- Options -->
        <div class="options-row">
          <label>
            <input type="checkbox" [(ngModel)]="includeLearnings" />
            Learnings
          </label>
          <label>
            <input type="checkbox" [(ngModel)]="includeDecisions" />
            Decisions
          </label>
          <label>
            <input type="checkbox" [(ngModel)]="includeFailures" />
            Failures
          </label>
          <div class="max-tokens">
            <span>Max Tokens:</span>
            <input type="number" [(ngModel)]="maxTokens" min="500" max="10000" step="500" />
          </div>
        </div>
      </div>

      <!-- Loading State -->
      @if (loading()) {
        <div class="loading">
          <div class="spinner"></div>
          <span>Analyzing context...</span>
        </div>
      }

      <!-- Error State -->
      @if (error()) {
        <div class="error-message">
          {{ error() }}
        </div>
      }

      <!-- Results -->
      @if (context() && !loading()) {
        <div class="results">
          <!-- Token Budget Bar -->
          <div class="token-budget">
            <div class="budget-header">
              <span>Token Budget</span>
              <span class="token-count">{{ context()!.totalTokens | number }} / {{ maxTokens | number }}</span>
            </div>
            <div class="budget-bar">
              <div class="budget-fill"
                   [style.width.%]="getTokenPercentage()"
                   [class.over-budget]="context()!.totalTokens > maxTokens">
              </div>
            </div>
            <div class="budget-legend">
              <span class="legend-item learnings">Learnings ({{ context()!.learnings.length }})</span>
              <span class="legend-item decisions">Decisions ({{ context()!.decisions.length }})</span>
              <span class="legend-item failures">Failures ({{ context()!.failures.length }})</span>
            </div>
          </div>

          <!-- Room Info -->
          @if (context()!.room) {
            <div class="room-info">
              <span class="label">Resolved Room:</span>
              <span class="value">{{ context()!.room }}</span>
            </div>
          }

          <!-- Warnings -->
          @if (context()!.warnings.length > 0) {
            <div class="warnings-section">
              <h3>Warnings</h3>
              @for (warning of context()!.warnings; track warning.recordId) {
                <div class="warning-item" [class]="'warning-' + warning.type">
                  <span class="warning-icon">
                    @if (warning.type === 'contradiction') { !! }
                    @else if (warning.type === 'decay') { * }
                    @else { ! }
                  </span>
                  <span class="warning-message">{{ warning.message }}</span>
                </div>
              }
            </div>
          }

          <!-- Learnings Section -->
          @if (context()!.learnings.length > 0) {
            <div class="section">
              <div class="section-header" (click)="toggleSection('learnings')">
                <h3>Learnings ({{ context()!.learnings.length }})</h3>
                <span class="toggle">{{ expandedSections.learnings ? '-' : '+' }}</span>
              </div>
              @if (expandedSections.learnings) {
                <div class="section-content">
                  @for (item of context()!.learnings; track item.learning.id) {
                    <div class="learning-card">
                      <div class="card-header">
                        <span class="priority-badge" [class]="getPriorityClass(item.priority)">
                          {{ (item.priority * 100).toFixed(0) }}%
                        </span>
                        <span class="scope">{{ item.learning.scope }}</span>
                      </div>
                      <p class="content">{{ item.learning.content }}</p>
                      <div class="card-footer">
                        <span class="reason">{{ item.reason }}</span>
                        <span class="confidence">Confidence: {{ (item.learning.confidence * 100).toFixed(0) }}%</span>
                      </div>
                    </div>
                  }
                </div>
              }
            </div>
          }

          <!-- Decisions Section -->
          @if (context()!.decisions.length > 0) {
            <div class="section">
              <div class="section-header" (click)="toggleSection('decisions')">
                <h3>Decisions ({{ context()!.decisions.length }})</h3>
                <span class="toggle">{{ expandedSections.decisions ? '-' : '+' }}</span>
              </div>
              @if (expandedSections.decisions) {
                <div class="section-content">
                  @for (decision of context()!.decisions; track decision.id) {
                    <div class="decision-card" [class]="'status-' + decision.status">
                      <div class="card-header">
                        <span class="status-badge" [class]="'badge-' + decision.status">{{ decision.status }}</span>
                        <span class="scope">{{ decision.scope }}</span>
                      </div>
                      <p class="content">{{ decision.content }}</p>
                      @if (decision.rationale) {
                        <div class="rationale">{{ decision.rationale }}</div>
                      }
                    </div>
                  }
                </div>
              }
            </div>
          }

          <!-- Failures Section -->
          @if (context()!.failures.length > 0) {
            <div class="section">
              <div class="section-header" (click)="toggleSection('failures')">
                <h3>Failures ({{ context()!.failures.length }})</h3>
                <span class="toggle">{{ expandedSections.failures ? '-' : '+' }}</span>
              </div>
              @if (expandedSections.failures) {
                <div class="section-content">
                  @for (failure of context()!.failures; track failure.path) {
                    <div class="failure-card">
                      <span class="failure-count">{{ failure.failureCount }}x</span>
                      <span class="failure-path">{{ failure.path }}</span>
                      <span class="failure-date">Last: {{ formatDate(failure.lastFailure) }}</span>
                    </div>
                  }
                </div>
              }
            </div>
          }

          <!-- Empty Sections -->
          @if (context()!.learnings.length === 0 && context()!.decisions.length === 0 && context()!.failures.length === 0) {
            <div class="empty-state">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <circle cx="12" cy="12" r="10"/>
                <path d="M12 6v6l4 2"/>
              </svg>
              <h3>No context found</h3>
              <p>No learnings, decisions, or failures are associated with this file path</p>
            </div>
          }
        </div>
      }

      <!-- Initial State -->
      @if (!context() && !loading() && !error()) {
        <div class="empty-state">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M9 12h6M12 9v6"/>
            <path d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
          </svg>
          <h3>Preview AI Context</h3>
          <p>Enter a file path to see what knowledge AI agents will receive</p>
        </div>
      }
    </div>
  `,
  styles: [`
    .context-preview-page {
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

    .input-section {
      background: #1a1a2e;
      border-radius: 8px;
      padding: 1.25rem;
      margin-bottom: 1.5rem;
      border: 1px solid #2d2d44;
    }

    .input-row {
      display: flex;
      gap: 0.75rem;
    }

    .input-row input[type="text"] {
      flex: 1;
      background: #2d2d44;
      border: 1px solid #3d3d54;
      border-radius: 6px;
      padding: 0.75rem 1rem;
      color: #e2e8f0;
      font-size: 0.95rem;
    }

    .input-row input[type="text"]:focus {
      outline: none;
      border-color: #9d4edd;
    }

    .input-row button {
      background: #9d4edd;
      border: none;
      border-radius: 6px;
      padding: 0.75rem 1.5rem;
      color: white;
      font-weight: 500;
      cursor: pointer;
      transition: background 0.2s;
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }

    .input-row button:hover:not(:disabled) {
      background: #b06edd;
    }

    .input-row button:disabled {
      background: #4a4a60;
      cursor: not-allowed;
    }

    .spinner-inline {
      width: 16px;
      height: 16px;
      border: 2px solid rgba(255,255,255,0.3);
      border-top-color: white;
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
    }

    .options-row {
      display: flex;
      align-items: center;
      gap: 1.5rem;
      margin-top: 1rem;
      padding-top: 1rem;
      border-top: 1px solid #2d2d44;
    }

    .options-row label {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      color: #94a3b8;
      font-size: 0.875rem;
      cursor: pointer;
    }

    .options-row input[type="checkbox"] {
      accent-color: #9d4edd;
    }

    .max-tokens {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      margin-left: auto;
      color: #94a3b8;
      font-size: 0.875rem;
    }

    .max-tokens input[type="number"] {
      width: 80px;
      background: #2d2d44;
      border: 1px solid #3d3d54;
      border-radius: 4px;
      padding: 0.35rem 0.5rem;
      color: #e2e8f0;
      text-align: center;
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
      width: 24px;
      height: 24px;
      border: 2px solid #3d3d54;
      border-top-color: #9d4edd;
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
      to { transform: rotate(360deg); }
    }

    .error-message {
      background: rgba(239, 68, 68, 0.1);
      border: 1px solid rgba(239, 68, 68, 0.3);
      border-radius: 8px;
      padding: 1rem;
      color: #ef4444;
      margin-bottom: 1rem;
    }

    .results {
      display: flex;
      flex-direction: column;
      gap: 1rem;
    }

    .token-budget {
      background: #1a1a2e;
      border-radius: 8px;
      padding: 1rem;
      border: 1px solid #2d2d44;
    }

    .budget-header {
      display: flex;
      justify-content: space-between;
      margin-bottom: 0.5rem;
      font-size: 0.875rem;
      color: #94a3b8;
    }

    .token-count {
      color: #e2e8f0;
      font-weight: 500;
    }

    .budget-bar {
      height: 8px;
      background: #2d2d44;
      border-radius: 4px;
      overflow: hidden;
    }

    .budget-fill {
      height: 100%;
      background: linear-gradient(90deg, #9d4edd, #b06edd);
      border-radius: 4px;
      transition: width 0.3s;
    }

    .budget-fill.over-budget {
      background: linear-gradient(90deg, #ef4444, #f97316);
    }

    .budget-legend {
      display: flex;
      gap: 1rem;
      margin-top: 0.75rem;
      font-size: 0.75rem;
    }

    .legend-item {
      display: flex;
      align-items: center;
      gap: 0.25rem;
      color: #64748b;
    }

    .legend-item::before {
      content: '';
      width: 8px;
      height: 8px;
      border-radius: 2px;
    }

    .legend-item.learnings::before { background: #22c55e; }
    .legend-item.decisions::before { background: #3b82f6; }
    .legend-item.failures::before { background: #ef4444; }

    .room-info {
      background: #1a1a2e;
      border-radius: 8px;
      padding: 0.75rem 1rem;
      border: 1px solid #2d2d44;
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }

    .room-info .label {
      color: #64748b;
      font-size: 0.875rem;
    }

    .room-info .value {
      color: #e2e8f0;
      font-weight: 500;
    }

    .warnings-section {
      background: rgba(245, 158, 11, 0.1);
      border: 1px solid rgba(245, 158, 11, 0.3);
      border-radius: 8px;
      padding: 1rem;
    }

    .warnings-section h3 {
      color: #f59e0b;
      font-size: 0.875rem;
      margin: 0 0 0.75rem 0;
    }

    .warning-item {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.5rem;
      font-size: 0.85rem;
      color: #f59e0b;
    }

    .warning-icon {
      font-weight: bold;
    }

    .warning-contradiction { color: #ef4444; }
    .warning-decay { color: #f59e0b; }

    .section {
      background: #1a1a2e;
      border-radius: 8px;
      border: 1px solid #2d2d44;
      overflow: hidden;
    }

    .section-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 1rem;
      cursor: pointer;
      user-select: none;
    }

    .section-header:hover {
      background: rgba(45, 45, 68, 0.5);
    }

    .section-header h3 {
      font-size: 0.95rem;
      font-weight: 600;
      color: #e2e8f0;
      margin: 0;
    }

    .toggle {
      font-size: 1.25rem;
      color: #64748b;
    }

    .section-content {
      padding: 0 1rem 1rem;
      display: flex;
      flex-direction: column;
      gap: 0.75rem;
    }

    .learning-card, .decision-card, .failure-card {
      background: #2d2d44;
      border-radius: 6px;
      padding: 0.75rem;
    }

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 0.5rem;
    }

    .priority-badge {
      font-size: 0.7rem;
      font-weight: 600;
      padding: 0.2rem 0.5rem;
      border-radius: 4px;
    }

    .priority-high {
      background: rgba(34, 197, 94, 0.15);
      color: #22c55e;
    }

    .priority-medium {
      background: rgba(245, 158, 11, 0.15);
      color: #f59e0b;
    }

    .priority-low {
      background: rgba(100, 116, 139, 0.15);
      color: #64748b;
    }

    .scope {
      font-size: 0.75rem;
      color: #64748b;
    }

    .content {
      color: #e2e8f0;
      font-size: 0.875rem;
      line-height: 1.5;
      margin: 0 0 0.5rem 0;
    }

    .card-footer {
      display: flex;
      justify-content: space-between;
      font-size: 0.75rem;
    }

    .reason {
      color: #9d4edd;
      font-style: italic;
    }

    .confidence {
      color: #64748b;
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

    .rationale {
      font-size: 0.8rem;
      color: #94a3b8;
      background: rgba(30, 30, 46, 0.5);
      padding: 0.5rem;
      border-radius: 4px;
      margin-top: 0.5rem;
    }

    .failure-card {
      display: flex;
      align-items: center;
      gap: 1rem;
    }

    .failure-count {
      background: rgba(239, 68, 68, 0.15);
      color: #ef4444;
      font-weight: 600;
      padding: 0.25rem 0.5rem;
      border-radius: 4px;
      font-size: 0.8rem;
    }

    .failure-path {
      flex: 1;
      color: #e2e8f0;
      font-family: monospace;
      font-size: 0.85rem;
    }

    .failure-date {
      color: #64748b;
      font-size: 0.75rem;
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
export class ContextPreviewComponent {
  private api = inject(ApiService);

  filePath = '';
  maxTokens = 2000;
  includeLearnings = true;
  includeDecisions = true;
  includeFailures = true;

  loading = signal(false);
  error = signal<string | null>(null);
  context = signal<AutoInjectedContext | null>(null);

  expandedSections = {
    learnings: true,
    decisions: true,
    failures: true
  };

  previewContext() {
    if (!this.filePath) return;

    this.loading.set(true);
    this.error.set(null);

    this.api.getContextPreview(this.filePath, {
      maxTokens: this.maxTokens,
      includeLearnings: this.includeLearnings,
      includeDecisions: this.includeDecisions,
      includeFailures: this.includeFailures
    }).subscribe({
      next: (res) => {
        this.context.set(res);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to preview context');
        this.context.set(null);
        this.loading.set(false);
      }
    });
  }

  getTokenPercentage(): number {
    const ctx = this.context();
    if (!ctx) return 0;
    return Math.min((ctx.totalTokens / this.maxTokens) * 100, 100);
  }

  getPriorityClass(priority: number): string {
    if (priority >= 0.7) return 'priority-high';
    if (priority >= 0.4) return 'priority-medium';
    return 'priority-low';
  }

  toggleSection(section: 'learnings' | 'decisions' | 'failures') {
    this.expandedSections[section] = !this.expandedSections[section];
  }

  formatDate(dateStr: string): string {
    if (!dateStr) return '';
    const date = new Date(dateStr);
    return date.toLocaleDateString();
  }
}
