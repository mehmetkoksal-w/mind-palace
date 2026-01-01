import { Component, OnInit, inject, signal } from '@angular/core';

import { FormsModule } from '@angular/forms';
import { ApiService, Learning } from '../../core/services/api.service';

@Component({
    selector: 'app-learnings',
    imports: [FormsModule],
    template: `
    <div class="learnings">
      <h2>Learnings</h2>

      <div class="search-bar">
        <input
          type="text"
          placeholder="Search learnings..."
          [(ngModel)]="searchQuery"
          (keyup.enter)="search()">
        <button (click)="search()">Search</button>
      </div>

      <div class="learnings-list">
        @for (learning of learnings(); track learning.id) {
          <div class="learning-card">
            <div class="learning-header">
              <div class="confidence-bar">
                <div class="fill" [style.width.%]="learning.confidence * 100"></div>
              </div>
              <span class="confidence-text">{{ (learning.confidence * 100).toFixed(0) }}%</span>
            </div>
            <div class="learning-content">{{ learning.content }}</div>
            <div class="learning-meta">
              <span class="scope">{{ learning.scope }}{{ learning.scopePath ? ':' + learning.scopePath : '' }}</span>
              <span class="source">{{ learning.source }}</span>
              <span class="used">Used {{ learning.useCount }} times</span>
            </div>
          </div>
        }

        @if (learnings().length === 0) {
          <div class="empty">No learnings found</div>
        }
      </div>
    </div>
  `,
    styles: [`
    .learnings h2 {
      color: #9d4edd;
      margin-bottom: 1.5rem;
    }

    .search-bar {
      display: flex;
      gap: 0.5rem;
      margin-bottom: 1.5rem;
    }

    .search-bar input {
      flex: 1;
      padding: 0.75rem 1rem;
      background: #16213e;
      border: 1px solid #2d3748;
      border-radius: 8px;
      color: #eee;
      font-size: 1rem;
    }

    .search-bar button {
      padding: 0.75rem 1.5rem;
      background: #9d4edd;
      border: none;
      border-radius: 8px;
      color: #fff;
      cursor: pointer;
      transition: background 0.2s;
    }

    .search-bar button:hover {
      background: #7b2cbf;
    }

    .learnings-list {
      display: flex;
      flex-direction: column;
      gap: 1rem;
    }

    .learning-card {
      background: #16213e;
      border-radius: 12px;
      padding: 1.25rem;
    }

    .learning-header {
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
      background: linear-gradient(90deg, #00d26a, #00b4d8);
      transition: width 0.3s;
    }

    .confidence-text {
      color: #00d26a;
      font-weight: bold;
      min-width: 40px;
    }

    .learning-content {
      color: #eee;
      line-height: 1.5;
      margin-bottom: 0.75rem;
    }

    .learning-meta {
      display: flex;
      gap: 1rem;
      color: #718096;
      font-size: 0.875rem;
    }

    .scope {
      color: #00b4d8;
    }

    .empty {
      text-align: center;
      color: #718096;
      padding: 2rem;
    }
  `]
})
export class LearningsComponent implements OnInit {
  private readonly api = inject(ApiService);

  learnings = signal<Learning[]>([]);
  searchQuery = '';

  ngOnInit() {
    this.loadLearnings();
  }

  loadLearnings() {
    this.api.getLearnings('', this.searchQuery).subscribe({
      next: (data) => this.learnings.set(data.learnings || []),
      error: (err) => console.error('Failed to load learnings:', err)
    });
  }

  search() {
    this.loadLearnings();
  }
}
