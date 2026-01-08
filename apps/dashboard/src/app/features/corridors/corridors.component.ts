import { Component, OnInit, inject, signal } from "@angular/core";

import { ApiService } from "../../core/services/api.service";
import { LoggerService } from "../../core/services/logger.service";

interface PersonalLearning {
  id: string;
  originWorkspace: string;
  content: string;
  confidence: number;
  useCount: number;
}

interface LinkedWorkspace {
  name: string;
  path: string;
  addedAt: string;
  lastAccessed: string;
}

@Component({
  selector: "app-corridors",
  imports: [],
  template: `
    <div class="corridors">
      <h2>Corridors</h2>

      <div class="section">
        <h3>Personal Corridor</h3>
        <div class="stats-row">
          <div class="stat">
            <span class="value">{{ stats()?.learningCount || 0 }}</span>
            <span class="label">Learnings</span>
          </div>
          <div class="stat">
            <span class="value">{{
              formatConfidence(stats()?.averageConfidence)
            }}</span>
            <span class="label">Avg Confidence</span>
          </div>
        </div>

        <div class="learnings-preview">
          @for (learning of personalLearnings(); track learning.id) {
          <div class="learning-item">
            <div class="confidence-badge">
              {{ (learning.confidence * 100).toFixed(0) }}%
            </div>
            <div class="content">{{ learning.content }}</div>
            @if (learning.originWorkspace) {
            <div class="origin">from: {{ learning.originWorkspace }}</div>
            }
          </div>
          } @if (personalLearnings().length === 0) {
          <div class="empty">No personal learnings yet</div>
          }
        </div>
      </div>

      <div class="section">
        <h3>Linked Workspaces</h3>
        <div class="links-list">
          @for (link of links(); track link.name) {
          <div class="link-card">
            <div class="link-name">{{ link.name }}</div>
            <div class="link-path">{{ link.path }}</div>
            <div class="link-meta">
              Added: {{ formatDate(link.addedAt) }} | Last accessed:
              {{ link.lastAccessed ? formatDate(link.lastAccessed) : "Never" }}
            </div>
          </div>
          } @if (links().length === 0) {
          <div class="empty">
            No linked workspaces. Use
            <code>palace corridor link &lt;name&gt; &lt;path&gt;</code> to link
            one.
          </div>
          }
        </div>
      </div>
    </div>
  `,
  styles: [
    `
      .corridors h2 {
        color: #9d4edd;
        margin-bottom: 1.5rem;
      }

      .section {
        background: #16213e;
        border-radius: 12px;
        padding: 1.5rem;
        margin-bottom: 1.5rem;
      }

      .section h3 {
        color: #9d4edd;
        margin: 0 0 1rem 0;
      }

      .stats-row {
        display: flex;
        gap: 2rem;
        margin-bottom: 1.5rem;
      }

      .stat {
        display: flex;
        flex-direction: column;
      }

      .stat .value {
        font-size: 1.5rem;
        font-weight: bold;
        color: #00d26a;
      }

      .stat .label {
        color: #718096;
        font-size: 0.875rem;
      }

      .learnings-preview {
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
      }

      .learning-item {
        background: #1a1a2e;
        border-radius: 8px;
        padding: 1rem;
        display: flex;
        gap: 1rem;
        align-items: flex-start;
      }

      .confidence-badge {
        background: #00d26a33;
        color: #00d26a;
        padding: 0.25rem 0.5rem;
        border-radius: 4px;
        font-size: 0.75rem;
        font-weight: bold;
      }

      .content {
        flex: 1;
        color: #eee;
      }

      .origin {
        color: #718096;
        font-size: 0.75rem;
      }

      .links-list {
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
      }

      .link-card {
        background: #1a1a2e;
        border-radius: 8px;
        padding: 1rem;
      }

      .link-name {
        font-weight: bold;
        color: #00b4d8;
      }

      .link-path {
        font-family: monospace;
        color: #a0aec0;
        font-size: 0.875rem;
        margin: 0.25rem 0;
      }

      .link-meta {
        color: #718096;
        font-size: 0.75rem;
      }

      .empty {
        text-align: center;
        color: #718096;
        padding: 1rem;
      }

      .empty code {
        background: #1a1a2e;
        padding: 0.25rem 0.5rem;
        border-radius: 4px;
      }
    `,
  ],
})
export class CorridorsComponent implements OnInit {
  private readonly api = inject(ApiService);
  private readonly logger =
    inject(LoggerService).forContext("CorridorsComponent");

  stats = signal<any>(null);
  links = signal<LinkedWorkspace[]>([]);
  personalLearnings = signal<PersonalLearning[]>([]);

  ngOnInit() {
    this.loadData();
  }

  loadData() {
    this.api.getCorridors().subscribe({
      next: (data) => {
        this.stats.set(data.stats);
        this.links.set(data.links || []);
      },
      error: (err) =>
        this.logger.error("Failed to load corridors", err, {
          endpoint: "/api/corridors",
        }),
    });

    this.api.getPersonalLearnings("", 10).subscribe({
      next: (data) => this.personalLearnings.set(data.learnings || []),
      error: (err) =>
        this.logger.error("Failed to load personal learnings", err, {
          endpoint: "/api/learnings/personal",
          limit: 10,
        }),
    });
  }

  formatConfidence(value: number | undefined): string {
    if (!value) return "0%";
    return (value * 100).toFixed(0) + "%";
  }

  formatDate(timestamp: string): string {
    if (!timestamp) return "Unknown";
    const date = new Date(timestamp);
    return date.toLocaleDateString();
  }
}
