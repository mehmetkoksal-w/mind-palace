import { Component, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ApiService, ScopeExplanation, ScopeLevel, Learning, Decision, Idea } from '../../core/services/api.service';

@Component({
  selector: 'app-scope-explorer',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="scope-explorer-page">
      <div class="page-header">
        <h2>Scope Explorer</h2>
        <p class="subtitle">Understand knowledge inheritance for any file path</p>
      </div>

      <!-- Input Section -->
      <div class="input-section">
        <div class="input-row">
          <input
            type="text"
            [(ngModel)]="filePath"
            placeholder="Enter file path (e.g., src/auth/jwt.go)"
            (keyup.enter)="exploreScope()"
          />
          <button (click)="exploreScope()" [disabled]="loading() || !filePath">
            @if (loading()) {
              <span class="spinner-inline"></span>
            } @else {
              Explore
            }
          </button>
        </div>
      </div>

      <!-- Loading State -->
      @if (loading()) {
        <div class="loading">
          <div class="spinner"></div>
          <span>Analyzing scope...</span>
        </div>
      }

      <!-- Error State -->
      @if (error()) {
        <div class="error-message">
          {{ error() }}
        </div>
      }

      <!-- Results -->
      @if (scopeData() && !loading()) {
        <!-- Room Info -->
        <div class="room-info">
          <span class="label">Resolved Room:</span>
          <span class="value">{{ scopeData()!.resolvedRoom || 'None' }}</span>
        </div>

        <!-- Scope Hierarchy Visualization -->
        <div class="scope-hierarchy">
          @for (level of scopeData()!.inheritanceChain; track level.scope; let i = $index) {
            <div
              class="scope-level"
              [class]="'level-' + level.scope"
              [class.active]="level.active"
              [class.expanded]="expandedScope() === level.scope"
              [style.margin-left.px]="i * 24"
            >
              <div class="level-header" (click)="toggleScopeLevel(level.scope)">
                <span class="level-icon">{{ getScopeIcon(level.scope) }}</span>
                <span class="level-name">{{ getScopeName(level.scope) }}</span>
                @if (level.path) {
                  <span class="level-path">{{ level.path }}</span>
                }
                <span class="level-count">{{ level.recordCount }} records</span>
                <span class="level-status" [class.inherited]="level.active">
                  {{ level.active ? 'Inherited' : 'OFF' }}
                </span>
                <span class="toggle-icon">{{ expandedScope() === level.scope ? '−' : '+' }}</span>
              </div>

              @if (expandedScope() === level.scope) {
                <div class="level-content">
                  @if (levelRecords().length > 0) {
                    @for (record of levelRecords(); track record.id) {
                      <div class="record-item" [class]="'record-' + record.type">
                        <span class="record-type">{{ record.type }}</span>
                        <span class="record-content">{{ record.content }}</span>
                        @if (record.confidence) {
                          <span class="record-confidence">{{ (record.confidence * 100).toFixed(0) }}%</span>
                        }
                      </div>
                    }
                  } @else {
                    <div class="no-records">No records at this level</div>
                  }
                </div>
              }
            </div>
          }
        </div>

        <!-- Total Summary -->
        <div class="totals-summary">
          <h4>Effective Knowledge</h4>
          <div class="totals-grid">
            <div class="total-item">
              <span class="total-value">{{ scopeData()!.totalRecords['learnings'] || 0 }}</span>
              <span class="total-label">Learnings</span>
            </div>
            <div class="total-item">
              <span class="total-value">{{ scopeData()!.totalRecords['decisions'] || 0 }}</span>
              <span class="total-label">Decisions</span>
            </div>
            <div class="total-item">
              <span class="total-value">{{ scopeData()!.totalRecords['ideas'] || 0 }}</span>
              <span class="total-label">Ideas</span>
            </div>
          </div>
          <p class="totals-note">
            Total knowledge accessible from <code>{{ scopeData()!.filePath }}</code>
          </p>
        </div>

        <!-- Inheritance Flow Diagram -->
        <div class="inheritance-diagram">
          <h4>Inheritance Flow</h4>
          <div class="flow-container">
            <div class="flow-node corridor">
              <span class="node-icon">🌐</span>
              <span class="node-label">Corridor</span>
              <span class="node-status">{{ getInheritanceStatus('corridor') }}</span>
            </div>
            <div class="flow-arrow">↓</div>
            <div class="flow-node palace">
              <span class="node-icon">🏛</span>
              <span class="node-label">Palace</span>
              <span class="node-status">{{ getInheritanceStatus('palace') }}</span>
            </div>
            <div class="flow-arrow">↓</div>
            <div class="flow-node room">
              <span class="node-icon">🏠</span>
              <span class="node-label">Room</span>
              <span class="node-status">{{ getInheritanceStatus('room') }}</span>
            </div>
            <div class="flow-arrow">↓</div>
            <div class="flow-node file active">
              <span class="node-icon">📄</span>
              <span class="node-label">File</span>
              <span class="node-status">Target</span>
            </div>
          </div>
        </div>
      }

      <!-- Initial State -->
      @if (!scopeData() && !loading() && !error()) {
        <div class="empty-state">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/>
          </svg>
          <h3>Explore Scope Hierarchy</h3>
          <p>Enter a file path to see how knowledge flows through the scope hierarchy</p>
          <div class="hint-list">
            <div class="hint-item">
              <span class="hint-icon">📄</span>
              <span><strong>File</strong> - Knowledge specific to one file</span>
            </div>
            <div class="hint-item">
              <span class="hint-icon">🏠</span>
              <span><strong>Room</strong> - Shared within a directory/module</span>
            </div>
            <div class="hint-item">
              <span class="hint-icon">🏛</span>
              <span><strong>Palace</strong> - Project-wide knowledge</span>
            </div>
            <div class="hint-item">
              <span class="hint-icon">🌐</span>
              <span><strong>Corridor</strong> - Cross-workspace knowledge</span>
            </div>
          </div>
        </div>
      }
    </div>
  `,
  styles: [`
    .scope-explorer-page {
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
      font-family: monospace;
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

    .room-info {
      background: #1a1a2e;
      border-radius: 8px;
      padding: 0.75rem 1rem;
      border: 1px solid #2d2d44;
      display: flex;
      align-items: center;
      gap: 0.5rem;
      margin-bottom: 1rem;
    }

    .room-info .label {
      color: #64748b;
      font-size: 0.875rem;
    }

    .room-info .value {
      color: #e2e8f0;
      font-weight: 500;
      font-family: monospace;
    }

    .scope-hierarchy {
      background: #1a1a2e;
      border-radius: 8px;
      padding: 1rem;
      border: 1px solid #2d2d44;
      margin-bottom: 1rem;
    }

    .scope-level {
      border-radius: 8px;
      margin-bottom: 0.5rem;
      overflow: hidden;
      border: 1px solid transparent;
    }

    .scope-level.active {
      border-color: rgba(157, 78, 221, 0.3);
    }

    .level-header {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      padding: 0.75rem 1rem;
      cursor: pointer;
      background: #2d2d44;
      transition: background 0.15s;
    }

    .level-header:hover {
      background: #3d3d54;
    }

    .scope-level.expanded .level-header {
      border-bottom: 1px solid #3d3d54;
    }

    .level-icon {
      font-size: 1.1rem;
    }

    .level-name {
      font-weight: 600;
      color: #e2e8f0;
      min-width: 80px;
    }

    .level-path {
      font-family: monospace;
      font-size: 0.8rem;
      color: #94a3b8;
      flex: 1;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .level-count {
      font-size: 0.8rem;
      color: #64748b;
    }

    .level-status {
      font-size: 0.7rem;
      padding: 0.2rem 0.5rem;
      border-radius: 4px;
      background: rgba(100, 116, 139, 0.2);
      color: #64748b;
    }

    .level-status.inherited {
      background: rgba(34, 197, 94, 0.15);
      color: #22c55e;
    }

    .toggle-icon {
      color: #64748b;
      font-size: 1rem;
    }

    .level-content {
      padding: 0.75rem 1rem;
      background: rgba(30, 30, 46, 0.5);
      max-height: 300px;
      overflow-y: auto;
    }

    .record-item {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      padding: 0.5rem;
      border-radius: 4px;
      margin-bottom: 0.5rem;
      background: #2d2d44;
    }

    .record-type {
      font-size: 0.65rem;
      font-weight: 600;
      text-transform: uppercase;
      padding: 0.15rem 0.4rem;
      border-radius: 4px;
      min-width: 60px;
      text-align: center;
    }

    .record-learning .record-type {
      background: rgba(34, 197, 94, 0.15);
      color: #22c55e;
    }

    .record-decision .record-type {
      background: rgba(59, 130, 246, 0.15);
      color: #3b82f6;
    }

    .record-idea .record-type {
      background: rgba(245, 158, 11, 0.15);
      color: #f59e0b;
    }

    .record-content {
      flex: 1;
      color: #e2e8f0;
      font-size: 0.85rem;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .record-confidence {
      font-size: 0.75rem;
      color: #22c55e;
      font-weight: 500;
    }

    .no-records {
      text-align: center;
      color: #64748b;
      padding: 1rem;
      font-size: 0.875rem;
    }

    .totals-summary {
      background: #1a1a2e;
      border-radius: 8px;
      padding: 1.25rem;
      border: 1px solid #2d2d44;
      margin-bottom: 1rem;
    }

    .totals-summary h4 {
      font-size: 0.9rem;
      color: #94a3b8;
      margin: 0 0 1rem 0;
      text-transform: uppercase;
    }

    .totals-grid {
      display: flex;
      gap: 2rem;
      margin-bottom: 1rem;
    }

    .total-item {
      display: flex;
      flex-direction: column;
      align-items: center;
    }

    .total-value {
      font-size: 2rem;
      font-weight: 600;
      color: #e2e8f0;
    }

    .total-label {
      font-size: 0.75rem;
      color: #64748b;
      text-transform: uppercase;
    }

    .totals-note {
      font-size: 0.8rem;
      color: #64748b;
      margin: 0;
    }

    .totals-note code {
      background: #2d2d44;
      padding: 0.15rem 0.35rem;
      border-radius: 4px;
      font-family: monospace;
      color: #94a3b8;
    }

    .inheritance-diagram {
      background: #1a1a2e;
      border-radius: 8px;
      padding: 1.25rem;
      border: 1px solid #2d2d44;
    }

    .inheritance-diagram h4 {
      font-size: 0.9rem;
      color: #94a3b8;
      margin: 0 0 1rem 0;
      text-transform: uppercase;
    }

    .flow-container {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 0.25rem;
    }

    .flow-node {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      background: #2d2d44;
      border-radius: 8px;
      padding: 0.75rem 1.5rem;
      min-width: 200px;
      border: 1px solid #3d3d54;
    }

    .flow-node.active {
      border-color: #9d4edd;
      background: rgba(157, 78, 221, 0.1);
    }

    .node-icon {
      font-size: 1.25rem;
    }

    .node-label {
      font-weight: 500;
      color: #e2e8f0;
      flex: 1;
    }

    .node-status {
      font-size: 0.7rem;
      padding: 0.2rem 0.5rem;
      border-radius: 4px;
      background: rgba(100, 116, 139, 0.2);
      color: #64748b;
    }

    .flow-node.active .node-status {
      background: rgba(157, 78, 221, 0.2);
      color: #9d4edd;
    }

    .flow-arrow {
      color: #64748b;
      font-size: 1rem;
    }

    .empty-state {
      text-align: center;
      padding: 3rem 2rem;
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
      margin: 0 0 1.5rem 0;
      font-size: 0.875rem;
    }

    .hint-list {
      display: flex;
      flex-direction: column;
      gap: 0.75rem;
      text-align: left;
      max-width: 400px;
      margin: 0 auto;
    }

    .hint-item {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      background: #1a1a2e;
      padding: 0.75rem 1rem;
      border-radius: 8px;
      border: 1px solid #2d2d44;
    }

    .hint-icon {
      font-size: 1.25rem;
    }

    .hint-item span {
      color: #94a3b8;
      font-size: 0.85rem;
    }

    .hint-item strong {
      color: #e2e8f0;
    }
  `]
})
export class ScopeExplorerComponent {
  private api = inject(ApiService);

  filePath = '';
  loading = signal(false);
  error = signal<string | null>(null);
  scopeData = signal<ScopeExplanation | null>(null);
  expandedScope = signal<string | null>(null);
  levelRecords = signal<Array<{id: string; type: string; content: string; confidence?: number}>>([]);

  exploreScope() {
    if (!this.filePath) return;

    this.loading.set(true);
    this.error.set(null);
    this.expandedScope.set(null);

    this.api.getScopeExplanation(this.filePath).subscribe({
      next: (res) => {
        this.scopeData.set(res);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error || 'Failed to analyze scope');
        this.scopeData.set(null);
        this.loading.set(false);
      }
    });
  }

  toggleScopeLevel(scope: string) {
    if (this.expandedScope() === scope) {
      this.expandedScope.set(null);
      this.levelRecords.set([]);
    } else {
      this.expandedScope.set(scope);
      this.loadLevelRecords(scope);
    }
  }

  loadLevelRecords(scope: string) {
    // For now, show placeholder. In a full implementation,
    // you'd fetch actual records for this scope level
    const data = this.scopeData();
    if (!data) return;

    // Mock data based on scope level counts
    const level = data.inheritanceChain.find(l => l.scope === scope);
    if (!level || level.recordCount === 0) {
      this.levelRecords.set([]);
      return;
    }

    // In a real implementation, you'd call an API to get records
    // For now, show a placeholder
    this.levelRecords.set([
      { id: '1', type: 'learning', content: 'Sample learning at ' + scope + ' scope', confidence: 0.85 },
      { id: '2', type: 'decision', content: 'Sample decision at ' + scope + ' scope' },
    ]);
  }

  getScopeIcon(scope: string): string {
    const icons: Record<string, string> = {
      'corridor': '🌐',
      'palace': '🏛',
      'room': '🏠',
      'file': '📄'
    };
    return icons[scope] || '📁';
  }

  getScopeName(scope: string): string {
    const names: Record<string, string> = {
      'corridor': 'Corridor',
      'palace': 'Palace',
      'room': 'Room',
      'file': 'File'
    };
    return names[scope] || scope;
  }

  getInheritanceStatus(scope: string): string {
    const data = this.scopeData();
    if (!data) return 'OFF';
    const level = data.inheritanceChain.find(l => l.scope === scope);
    if (!level) return 'OFF';
    return level.active ? `${level.recordCount}` : 'OFF';
  }
}
