import { Component, OnInit, inject, signal, computed } from "@angular/core";
import { FormsModule } from "@angular/forms";
import {
  ApiService,
  Pattern,
  PatternStats,
} from "../../core/services/api.service";
import { LoggerService } from "../../core/services/logger.service";

@Component({
  selector: "app-patterns",
  imports: [FormsModule],
  template: `
    <div class="patterns">
      <h2>Patterns</h2>

      <!-- Stats Summary -->
      <div class="stats-bar">
        @if (stats()) {
          <div class="stat">
            <span class="value">{{ stats()!.total }}</span>
            <span class="label">Total</span>
          </div>
          <div class="stat discovered">
            <span class="value">{{ stats()!.discovered }}</span>
            <span class="label">Discovered</span>
          </div>
          <div class="stat approved">
            <span class="value">{{ stats()!.approved }}</span>
            <span class="label">Approved</span>
          </div>
          <div class="stat ignored">
            <span class="value">{{ stats()!.ignored }}</span>
            <span class="label">Ignored</span>
          </div>
          <div class="stat">
            <span class="value"
              >{{ (stats()!.averageConfidence * 100).toFixed(0) }}%</span
            >
            <span class="label">Avg Confidence</span>
          </div>
        }
      </div>

      <!-- Filters -->
      <div class="filters">
        <select [(ngModel)]="statusFilter" (change)="loadPatterns()">
          <option value="">All Status</option>
          <option value="discovered">Discovered</option>
          <option value="approved">Approved</option>
          <option value="ignored">Ignored</option>
        </select>

        <select [(ngModel)]="categoryFilter" (change)="loadPatterns()">
          <option value="">All Categories</option>
          <option value="api">API</option>
          <option value="errors">Errors</option>
          <option value="naming">Naming</option>
          <option value="structural">Structural</option>
          <option value="testing">Testing</option>
          <option value="logging">Logging</option>
          <option value="config">Config</option>
          <option value="documentation">Documentation</option>
          <option value="complexity">Complexity</option>
        </select>

        <input
          type="number"
          placeholder="Min confidence %"
          [(ngModel)]="minConfidenceFilter"
          (change)="loadPatterns()"
          min="0"
          max="100"
        />

        <label class="learning-toggle">
          <input type="checkbox" [(ngModel)]="withLearning" />
          <span>Create Learning</span>
        </label>

        @if (selectedPatterns().length > 0) {
          <button class="bulk-approve" (click)="bulkApprove()">
            Approve Selected ({{ selectedPatterns().length }})
          </button>
        }
      </div>

      <!-- Patterns List -->
      <div class="patterns-list">
        @for (pattern of patterns(); track pattern.id) {
          <div class="pattern-card" [class.selected]="isSelected(pattern.id)">
            <div class="pattern-header">
              <input
                type="checkbox"
                [checked]="isSelected(pattern.id)"
                (change)="toggleSelection(pattern.id)"
                [disabled]="pattern.status !== 'discovered'"
              />
              <span
                class="category-badge"
                [attr.data-category]="pattern.category"
              >
                {{ pattern.category }}
              </span>
              <span class="pattern-name">{{ pattern.name }}</span>
              <span class="status-badge" [attr.data-status]="pattern.status">
                {{ pattern.status }}
              </span>
            </div>

            <div class="confidence-section">
              <div class="confidence-bar">
                <div
                  class="fill"
                  [style.width.%]="pattern.confidence * 100"
                  [class.high]="pattern.confidence >= 0.85"
                  [class.medium]="
                    pattern.confidence >= 0.7 && pattern.confidence < 0.85
                  "
                  [class.low]="pattern.confidence < 0.7"
                ></div>
              </div>
              <span class="confidence-text"
                >{{ (pattern.confidence * 100).toFixed(0) }}%</span
              >
            </div>

            <div class="pattern-description">{{ pattern.description }}</div>

            <div class="pattern-meta">
              <span class="locations"
                >{{ pattern.locations.length || 0 }} locations</span
              >
              <span
                class="outliers"
                [class.has-outliers]="(pattern.outliers.length || 0) > 0"
              >
                {{ pattern.outliers.length || 0 }} outliers
              </span>
              <span class="detector">{{ pattern.detectorId }}</span>
            </div>

            <div class="confidence-factors">
              <span title="Frequency"
                >F: {{ (pattern.frequencyScore * 100).toFixed(0) }}%</span
              >
              <span title="Consistency"
                >C: {{ (pattern.consistencyScore * 100).toFixed(0) }}%</span
              >
              <span title="Spread"
                >S: {{ (pattern.spreadScore * 100).toFixed(0) }}%</span
              >
              <span title="Age"
                >A: {{ (pattern.ageScore * 100).toFixed(0) }}%</span
              >
            </div>

            @if (pattern.status === "discovered") {
              <div class="pattern-actions">
                <button class="approve" (click)="approve(pattern.id)">
                  Approve
                </button>
                <button class="ignore" (click)="ignore(pattern.id)">
                  Ignore
                </button>
              </div>
            }
          </div>
        }
        @if (patterns().length === 0) {
          <div class="empty">
            No patterns found. Run <code>palace patterns scan</code> to detect
            patterns.
          </div>
        }
      </div>
    </div>
  `,
  styles: [
    `
      .patterns h2 {
        color: #9d4edd;
        margin-bottom: 1.5rem;
      }

      .stats-bar {
        display: flex;
        gap: 1.5rem;
        margin-bottom: 1.5rem;
        padding: 1rem;
        background: #16213e;
        border-radius: 8px;
      }

      .stat {
        display: flex;
        flex-direction: column;
        align-items: center;
      }

      .stat .value {
        font-size: 1.5rem;
        font-weight: bold;
        color: #eee;
      }

      .stat .label {
        font-size: 0.75rem;
        color: #718096;
        text-transform: uppercase;
      }

      .stat.discovered .value {
        color: #00b4d8;
      }

      .stat.approved .value {
        color: #00d26a;
      }

      .stat.ignored .value {
        color: #718096;
      }

      .filters {
        display: flex;
        gap: 0.75rem;
        margin-bottom: 1.5rem;
        flex-wrap: wrap;
      }

      .filters select,
      .filters input {
        padding: 0.5rem 0.75rem;
        background: #16213e;
        border: 1px solid #2d3748;
        border-radius: 6px;
        color: #eee;
        font-size: 0.875rem;
      }

      .filters input[type="number"] {
        width: 140px;
      }

      .bulk-approve {
        padding: 0.5rem 1rem;
        background: #00d26a;
        border: none;
        border-radius: 6px;
        color: #fff;
        cursor: pointer;
        font-weight: 500;
      }

      .bulk-approve:hover {
        background: #00b85c;
      }

      .learning-toggle {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.5rem 0.75rem;
        background: #16213e;
        border: 1px solid #2d3748;
        border-radius: 6px;
        color: #eee;
        font-size: 0.875rem;
        cursor: pointer;
      }

      .learning-toggle:hover {
        border-color: #9d4edd;
      }

      .learning-toggle input[type="checkbox"] {
        width: 16px;
        height: 16px;
        cursor: pointer;
      }

      .patterns-list {
        display: flex;
        flex-direction: column;
        gap: 1rem;
      }

      .pattern-card {
        background: #16213e;
        border-radius: 12px;
        padding: 1.25rem;
        border: 2px solid transparent;
        transition: border-color 0.2s;
      }

      .pattern-card.selected {
        border-color: #9d4edd;
      }

      .pattern-header {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        margin-bottom: 0.75rem;
      }

      .pattern-header input[type="checkbox"] {
        width: 18px;
        height: 18px;
        cursor: pointer;
      }

      .category-badge {
        padding: 0.25rem 0.5rem;
        border-radius: 4px;
        font-size: 0.75rem;
        font-weight: 500;
        text-transform: uppercase;
        background: #2d3748;
        color: #eee;
      }

      .category-badge[data-category="api"] {
        background: #3182ce;
      }

      .category-badge[data-category="errors"] {
        background: #e53e3e;
      }

      .category-badge[data-category="naming"] {
        background: #805ad5;
      }

      .category-badge[data-category="structural"] {
        background: #dd6b20;
      }

      .category-badge[data-category="testing"] {
        background: #38a169;
      }

      .category-badge[data-category="logging"] {
        background: #d69e2e;
      }

      .pattern-name {
        flex: 1;
        font-weight: 600;
        color: #eee;
      }

      .status-badge {
        padding: 0.25rem 0.5rem;
        border-radius: 4px;
        font-size: 0.75rem;
        font-weight: 500;
      }

      .status-badge[data-status="discovered"] {
        background: #00b4d8;
        color: #fff;
      }

      .status-badge[data-status="approved"] {
        background: #00d26a;
        color: #fff;
      }

      .status-badge[data-status="ignored"] {
        background: #718096;
        color: #fff;
      }

      .confidence-section {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        margin-bottom: 0.75rem;
      }

      .confidence-bar {
        flex: 1;
        height: 8px;
        background: #2d3748;
        border-radius: 4px;
        overflow: hidden;
      }

      .confidence-bar .fill {
        height: 100%;
        transition: width 0.3s;
      }

      .confidence-bar .fill.high {
        background: linear-gradient(90deg, #00d26a, #00b4d8);
      }

      .confidence-bar .fill.medium {
        background: linear-gradient(90deg, #d69e2e, #dd6b20);
      }

      .confidence-bar .fill.low {
        background: linear-gradient(90deg, #e53e3e, #dd6b20);
      }

      .confidence-text {
        color: #00d26a;
        font-weight: bold;
        min-width: 40px;
      }

      .pattern-description {
        color: #a0aec0;
        font-size: 0.875rem;
        line-height: 1.5;
        margin-bottom: 0.75rem;
      }

      .pattern-meta {
        display: flex;
        gap: 1rem;
        color: #718096;
        font-size: 0.8rem;
        margin-bottom: 0.5rem;
      }

      .outliers.has-outliers {
        color: #e53e3e;
      }

      .confidence-factors {
        display: flex;
        gap: 1rem;
        color: #718096;
        font-size: 0.75rem;
        margin-bottom: 0.75rem;
      }

      .pattern-actions {
        display: flex;
        gap: 0.5rem;
        margin-top: 0.75rem;
      }

      .pattern-actions button {
        padding: 0.5rem 1rem;
        border: none;
        border-radius: 6px;
        cursor: pointer;
        font-weight: 500;
        transition: background 0.2s;
      }

      .pattern-actions .approve {
        background: #00d26a;
        color: #fff;
      }

      .pattern-actions .approve:hover {
        background: #00b85c;
      }

      .pattern-actions .ignore {
        background: #4a5568;
        color: #fff;
      }

      .pattern-actions .ignore:hover {
        background: #2d3748;
      }

      .empty {
        text-align: center;
        color: #718096;
        padding: 2rem;
      }

      .empty code {
        background: #2d3748;
        padding: 0.25rem 0.5rem;
        border-radius: 4px;
        color: #00b4d8;
      }
    `,
  ],
})
export class PatternsComponent implements OnInit {
  private readonly api = inject(ApiService);
  private readonly logger =
    inject(LoggerService).forContext("PatternsComponent");

  patterns = signal<Pattern[]>([]);
  stats = signal<PatternStats | null>(null);
  selectedIds = signal<Set<string>>(new Set());

  statusFilter = "";
  categoryFilter = "";
  minConfidenceFilter: number | null = null;
  withLearning = false;

  selectedPatterns = computed(() => {
    return this.patterns().filter((p) => this.selectedIds().has(p.id));
  });

  ngOnInit() {
    this.loadPatterns();
    this.loadStats();
  }

  loadPatterns() {
    const params: any = {};
    if (this.statusFilter) params.status = this.statusFilter;
    if (this.categoryFilter) params.category = this.categoryFilter;
    if (this.minConfidenceFilter)
      params.minConfidence = this.minConfidenceFilter / 100;

    this.api.getPatterns(params).subscribe({
      next: (data) => this.patterns.set(data.patterns || []),
      error: (err) =>
        this.logger.error("Failed to load patterns", err, {
          endpoint: "/api/patterns",
        }),
    });
  }

  loadStats() {
    this.api.getPatternStats().subscribe({
      next: (data) => this.stats.set(data),
      error: (err) =>
        this.logger.error("Failed to load pattern stats", err, {
          endpoint: "/api/patterns/stats",
        }),
    });
  }

  isSelected(id: string): boolean {
    return this.selectedIds().has(id);
  }

  toggleSelection(id: string) {
    const current = new Set(this.selectedIds());
    if (current.has(id)) {
      current.delete(id);
    } else {
      current.add(id);
    }
    this.selectedIds.set(current);
  }

  approve(id: string) {
    this.api.approvePattern(id, this.withLearning).subscribe({
      next: (result: any) => {
        if (this.withLearning && result?.learningId) {
          this.logger.info("Pattern approved with learning created", {
            patternId: id,
            learningId: result.learningId,
          });
        }
        this.loadPatterns();
        this.loadStats();
      },
      error: (err) =>
        this.logger.error("Failed to approve pattern", err, { patternId: id }),
    });
  }

  ignore(id: string) {
    this.api.ignorePattern(id).subscribe({
      next: () => {
        this.loadPatterns();
        this.loadStats();
      },
      error: (err) =>
        this.logger.error("Failed to ignore pattern", err, { patternId: id }),
    });
  }

  bulkApprove() {
    const ids = Array.from(this.selectedIds());
    this.api.bulkApprovePatterns(ids, this.withLearning).subscribe({
      next: (result) => {
        if (this.withLearning && result?.learningIds?.length) {
          this.logger.info("Patterns approved with learnings created", {
            approved: result.approved,
            learningsCreated: result.learningIds.length,
          });
        }
        this.selectedIds.set(new Set());
        this.loadPatterns();
        this.loadStats();
      },
      error: (err) =>
        this.logger.error("Failed to bulk approve patterns", err, {
          count: ids.length,
        }),
    });
  }
}
