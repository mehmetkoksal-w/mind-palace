import { Component, inject, OnInit, signal } from "@angular/core";

import { FormsModule } from "@angular/forms";
import { ApiService, Idea } from "../../core/services/api.service";

@Component({
  selector: "app-ideas",
  imports: [FormsModule],
  template: `
    <div class="ideas-page">
      <div class="page-header">
        <h2>Ideas</h2>
        <p class="subtitle">
          Track exploration and concepts across your codebase
        </p>
      </div>

      <!-- Filters -->
      <div class="filters">
        <select [(ngModel)]="statusFilter" (change)="loadIdeas()">
          <option value="">All Status</option>
          <option value="active">Active</option>
          <option value="exploring">Exploring</option>
          <option value="implemented">Implemented</option>
          <option value="dropped">Dropped</option>
        </select>
        <span class="count">{{ ideas().length }} ideas</span>
      </div>

      <!-- Loading -->
      @if (loading()) {
      <div class="loading">
        <div class="spinner"></div>
        <span>Loading ideas...</span>
      </div>
      }

      <!-- Ideas List -->
      @if (!loading() && ideas().length > 0) {
      <div class="ideas-grid">
        @for (idea of ideas(); track idea.id) {
        <div class="idea-card" [class]="'status-' + idea.status">
          <div class="idea-header">
            <span class="status-badge">{{ idea.status }}</span>
            <span class="scope" [title]="idea.scopePath">{{
              idea.scope || "Global"
            }}</span>
          </div>
          <p class="idea-content">{{ idea.content }}</p>
          <div class="idea-footer">
            <span class="date">{{ formatDate(idea.createdAt) }}</span>
            @if (idea.tags.length) {
            <div class="tags">
              @for (tag of idea.tags.slice(0, 3); track tag) {
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
      @if (!loading() && ideas().length === 0) {
      <div class="empty-state">
        <svg
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
        >
          <path
            d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
          />
        </svg>
        <h3>No ideas yet</h3>
        <p>Ideas will appear here as you explore and work on your codebase</p>
      </div>
      }
    </div>
  `,
  styles: [
    `
      .ideas-page {
        max-width: 1200px;
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
        to {
          transform: rotate(360deg);
        }
      }

      .ideas-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
        gap: 1rem;
      }

      .idea-card {
        background: #1a1a2e;
        border-radius: 8px;
        padding: 1rem;
        border: 1px solid #2d2d44;
        transition: border-color 0.2s;
      }

      .idea-card:hover {
        border-color: #3d3d54;
      }

      .idea-card.status-active {
        border-left: 3px solid #22c55e;
      }
      .idea-card.status-exploring {
        border-left: 3px solid #3b82f6;
      }
      .idea-card.status-implemented {
        border-left: 3px solid #9d4edd;
      }
      .idea-card.status-dropped {
        border-left: 3px solid #64748b;
      }

      .idea-header {
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
        background: rgba(157, 78, 221, 0.15);
        color: #9d4edd;
      }

      .scope {
        font-size: 0.75rem;
        color: #64748b;
        max-width: 150px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .idea-content {
        color: #e2e8f0;
        font-size: 0.9rem;
        line-height: 1.5;
        margin: 0 0 0.75rem 0;
        display: -webkit-box;
        -webkit-line-clamp: 3;
        -webkit-box-orient: vertical;
        overflow: hidden;
      }

      .idea-footer {
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
    `,
  ],
})
export class IdeasComponent implements OnInit {
  private api = inject(ApiService);

  ideas = signal<Idea[]>([]);
  loading = signal(true);
  statusFilter = "";

  ngOnInit() {
    this.loadIdeas();
  }

  loadIdeas() {
    this.loading.set(true);
    this.api.getIdeas(this.statusFilter, "", 100).subscribe({
      next: (res) => {
        this.ideas.set(res.ideas || []);
        this.loading.set(false);
      },
      error: () => {
        this.ideas.set([]);
        this.loading.set(false);
      },
    });
  }

  formatDate(dateStr: string): string {
    if (!dateStr) return "";
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (days === 0) return "Today";
    if (days === 1) return "Yesterday";
    if (days < 7) return `${days} days ago`;
    return date.toLocaleDateString();
  }
}
