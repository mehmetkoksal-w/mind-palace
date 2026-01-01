import { Component, OnInit, inject, signal } from '@angular/core';

import { ApiService, Session } from '../../core/services/api.service';

@Component({
    selector: 'app-sessions',
    imports: [],
    template: `
    <div class="sessions">
      <h2>Sessions</h2>

      <div class="filters">
        <label>
          <input type="checkbox" [checked]="activeOnly()" (change)="toggleActive()">
          Active only
        </label>
      </div>

      <div class="sessions-list">
        @for (session of sessions(); track session.id) {
          <div class="session-card" [class.active]="session.state === 'active'">
            <div class="session-header">
              <span class="agent-type">{{ session.agentType }}</span>
              <span class="state" [class]="session.state">{{ session.state }}</span>
            </div>
            <div class="session-goal">{{ session.goal || 'No goal specified' }}</div>
            <div class="session-meta">
              <span>Started: {{ formatDate(session.startedAt) }}</span>
              <span>Last activity: {{ formatDate(session.lastActivity) }}</span>
            </div>
            @if (session.summary) {
              <div class="session-summary">{{ session.summary }}</div>
            }
          </div>
        }

        @if (sessions().length === 0) {
          <div class="empty">No sessions found</div>
        }
      </div>
    </div>
  `,
    styles: [`
    .sessions h2 {
      color: #9d4edd;
      margin-bottom: 1.5rem;
    }

    .filters {
      margin-bottom: 1rem;
    }

    .filters label {
      color: #a0aec0;
      cursor: pointer;
    }

    .sessions-list {
      display: flex;
      flex-direction: column;
      gap: 1rem;
    }

    .session-card {
      background: #16213e;
      border-radius: 12px;
      padding: 1.25rem;
      border-left: 4px solid #718096;
    }

    .session-card.active {
      border-left-color: #00d26a;
    }

    .session-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 0.5rem;
    }

    .agent-type {
      font-weight: bold;
      color: #00b4d8;
    }

    .state {
      padding: 0.25rem 0.75rem;
      border-radius: 12px;
      font-size: 0.75rem;
      text-transform: uppercase;
    }

    .state.active {
      background: #00d26a33;
      color: #00d26a;
    }

    .state.completed {
      background: #9d4edd33;
      color: #9d4edd;
    }

    .state.abandoned {
      background: #ff6b6b33;
      color: #ff6b6b;
    }

    .session-goal {
      color: #eee;
      margin-bottom: 0.5rem;
    }

    .session-meta {
      display: flex;
      gap: 1.5rem;
      color: #718096;
      font-size: 0.875rem;
    }

    .session-summary {
      margin-top: 0.75rem;
      padding-top: 0.75rem;
      border-top: 1px solid #2d3748;
      color: #a0aec0;
      font-size: 0.875rem;
    }

    .empty {
      text-align: center;
      color: #718096;
      padding: 2rem;
    }
  `]
})
export class SessionsComponent implements OnInit {
  private readonly api = inject(ApiService);

  sessions = signal<Session[]>([]);
  activeOnly = signal(false);

  ngOnInit() {
    this.loadSessions();
  }

  loadSessions() {
    this.api.getSessions(this.activeOnly()).subscribe({
      next: (data) => this.sessions.set(data.sessions || []),
      error: (err) => console.error('Failed to load sessions:', err)
    });
  }

  toggleActive() {
    this.activeOnly.update(v => !v);
    this.loadSessions();
  }

  formatDate(timestamp: string): string {
    if (!timestamp) return 'Unknown';
    const date = new Date(timestamp);
    return date.toLocaleString();
  }
}
