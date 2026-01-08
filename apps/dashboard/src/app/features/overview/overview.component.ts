import { Component, OnInit, inject, signal } from "@angular/core";

import {
  ApiService,
  Stats,
  ActiveAgent,
} from "../../core/services/api.service";
import { LoggerService } from "../../core/services/logger.service";
import { NeuralMapComponent } from "./neural-map/neural-map.component";

@Component({
  selector: "app-overview",
  imports: [NeuralMapComponent],
  template: `
    <div class="overview">
      <h2>Overview</h2>

      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-value">{{ stats()?.sessions?.total || 0 }}</div>
          <div class="stat-label">Total Sessions</div>
          <div class="stat-sub">
            {{ stats()?.sessions?.active || 0 }} active
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-value">{{ stats()?.learnings || 0 }}</div>
          <div class="stat-label">Learnings</div>
        </div>

        <div class="stat-card">
          <div class="stat-value">{{ stats()?.filesTracked || 0 }}</div>
          <div class="stat-label">Files Tracked</div>
        </div>

        <div class="stat-card">
          <div class="stat-value">{{ stats()?.rooms || 0 }}</div>
          <div class="stat-label">Rooms</div>
        </div>
      </div>

      <!-- Neural Map Section -->
      <div class="section neural-map-section">
        <div class="section-header">
          <h3>Neural Map</h3>
          <span class="section-subtitle"
            >Relationships between code and knowledge</span
          >
        </div>
        <app-neural-map [height]="420" />
      </div>

      @if (agents().length > 0) {
      <div class="section">
        <h3>Active Agents</h3>
        <div class="agents-list">
          @for (agent of agents(); track agent.agentId) {
          <div class="agent-card">
            <div class="agent-type">{{ agent.agentType }}</div>
            <div class="agent-file">
              {{ agent.currentFile || "No active file" }}
            </div>
            <div class="agent-heartbeat">
              Last seen: {{ formatTime(agent.heartbeat) }}
            </div>
          </div>
          }
        </div>
      </div>
      } @if (stats()?.corridor) {
      <div class="section">
        <h3>Personal Corridor</h3>
        <div class="corridor-stats">
          <div class="stat-item">
            <span class="label">Learnings:</span>
            <span class="value">{{
              stats()?.corridor?.learningCount || 0
            }}</span>
          </div>
          <div class="stat-item">
            <span class="label">Linked Workspaces:</span>
            <span class="value">{{
              stats()?.corridor?.linkedWorkspaces || 0
            }}</span>
          </div>
        </div>
      </div>
      }
    </div>
  `,
  styles: [
    `
      .overview h2 {
        color: #9d4edd;
        margin-bottom: 1.5rem;
      }

      .stats-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
        gap: 1rem;
        margin-bottom: 2rem;
      }

      .stat-card {
        background: #16213e;
        border-radius: 12px;
        padding: 1.5rem;
        text-align: center;
      }

      .stat-value {
        font-size: 2.5rem;
        font-weight: bold;
        color: #00d26a;
      }

      .stat-label {
        color: #a0aec0;
        margin-top: 0.5rem;
      }

      .stat-sub {
        color: #718096;
        font-size: 0.875rem;
        margin-top: 0.25rem;
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

      .agents-list {
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
      }

      .agent-card {
        background: #1a1a2e;
        border-radius: 8px;
        padding: 1rem;
      }

      .agent-type {
        font-weight: bold;
        color: #00b4d8;
      }

      .agent-file {
        color: #a0aec0;
        font-size: 0.875rem;
        margin-top: 0.25rem;
      }

      .agent-heartbeat {
        color: #718096;
        font-size: 0.75rem;
        margin-top: 0.25rem;
      }

      .corridor-stats {
        display: flex;
        gap: 2rem;
      }

      .stat-item .label {
        color: #a0aec0;
      }

      .stat-item .value {
        color: #00d26a;
        font-weight: bold;
        margin-left: 0.5rem;
      }

      .neural-map-section {
        margin-bottom: 1.5rem;
      }

      .section-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 1rem;
      }

      .section-header h3 {
        margin: 0;
      }

      .section-subtitle {
        font-size: 0.8rem;
        color: #64748b;
      }
    `,
  ],
})
export class OverviewComponent implements OnInit {
  private readonly api = inject(ApiService);
  private readonly logger =
    inject(LoggerService).forContext("OverviewComponent");

  stats = signal<Stats | null>(null);
  agents = signal<ActiveAgent[]>([]);

  ngOnInit() {
    this.loadStats();
    this.loadAgents();
  }

  loadStats() {
    this.api.getStats().subscribe({
      next: (data) => this.stats.set(data),
      error: (err) =>
        this.logger.error("Failed to load stats", err, {
          endpoint: "/api/stats",
        }),
    });
  }

  loadAgents() {
    this.api.getActiveAgents().subscribe({
      next: (data) => this.agents.set(data.agents || []),
      error: (err) =>
        this.logger.error("Failed to load active agents", err, {
          endpoint: "/api/agents/active",
        }),
    });
  }

  formatTime(timestamp: string): string {
    if (!timestamp) return "Unknown";
    const date = new Date(timestamp);
    return date.toLocaleTimeString();
  }
}
