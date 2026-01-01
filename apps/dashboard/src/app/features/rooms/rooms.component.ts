import { Component, OnInit, inject, signal } from '@angular/core';

import { FormsModule } from '@angular/forms';
import { ApiService, Room } from '../../core/services/api.service';

@Component({
    selector: 'app-rooms',
    imports: [FormsModule],
    template: `
    <div class="rooms-container">
      <header class="page-header">
        <h1>Rooms Explorer</h1>
        <p>Explore the logical organization of your codebase</p>
      </header>

      @if (loading()) {
        <div class="loading">Loading rooms...</div>
      } @else if (error()) {
        <div class="error">{{ error() }}</div>
      } @else {
        <div class="rooms-stats">
          <div class="stat">
            <span class="stat-value">{{ rooms().length }}</span>
            <span class="stat-label">Total Rooms</span>
          </div>
          <div class="stat">
            <span class="stat-value">{{ totalFiles() }}</span>
            <span class="stat-label">Total Files</span>
          </div>
          <div class="stat">
            <span class="stat-value">{{ totalEntryPoints() }}</span>
            <span class="stat-label">Entry Points</span>
          </div>
        </div>

        <div class="search-bar">
          <input
            type="text"
            [(ngModel)]="searchTerm"
            (ngModelChange)="filterRooms()"
            placeholder="Filter rooms..."
            class="search-input"
          />
        </div>

        <div class="rooms-grid">
          @for (room of filteredRooms(); track room.name) {
            <div class="room-card" [class.expanded]="expandedRoom() === room.name" (click)="toggleRoom(room.name)">
              <div class="room-header">
                <div class="room-icon">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                  </svg>
                </div>
                <div class="room-info">
                  <h3>{{ room.name }}</h3>
                  <span class="room-path">{{ room.path }}</span>
                </div>
                <div class="room-badges">
                  <span class="badge files">{{ room.files?.length || 0 }} files</span>
                  @if (room.entryPoints?.length) {
                    <span class="badge entries">{{ room.entryPoints.length }} entry</span>
                  }
                </div>
              </div>

              @if (expandedRoom() === room.name) {
                <div class="room-details">
                  @if (room.description) {
                    <p class="room-description">{{ room.description }}</p>
                  }

                  @if (room.entryPoints?.length) {
                    <div class="section">
                      <h4>Entry Points</h4>
                      <ul class="entry-list">
                        @for (entry of room.entryPoints; track entry) {
                          <li class="entry-point">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                              <polygon points="5 3 19 12 5 21 5 3"/>
                            </svg>
                            {{ entry }}
                          </li>
                        }
                      </ul>
                    </div>
                  }

                  @if (room.files?.length) {
                    <div class="section">
                      <h4>Files ({{ room.files.length }})</h4>
                      <ul class="file-list">
                        @for (file of room.files.slice(0, showAllFiles() ? undefined : 10); track file) {
                          <li class="file-item">
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                              <polyline points="14 2 14 8 20 8"/>
                            </svg>
                            {{ file }}
                          </li>
                        }
                      </ul>
                      @if (room.files.length > 10) {
                        <button class="show-more" (click)="toggleShowAll($event)">
                          {{ showAllFiles() ? 'Show less' : 'Show all ' + room.files.length + ' files' }}
                        </button>
                      }
                    </div>
                  }
                </div>
              }
            </div>
          }

          @if (filteredRooms().length === 0) {
            <div class="no-results">
              @if (rooms().length === 0) {
                <p>No rooms found. Run <code>palace scan</code> to index your codebase.</p>
              } @else {
                <p>No rooms match your search.</p>
              }
            </div>
          }
        </div>
      }
    </div>
  `,
    styles: [`
    .rooms-container {
      padding: 24px;
      max-width: 1400px;
      margin: 0 auto;
    }

    .page-header {
      margin-bottom: 24px;
    }

    .page-header h1 {
      margin: 0 0 8px 0;
      font-size: 28px;
      color: #fff;
    }

    .page-header p {
      margin: 0;
      color: #888;
    }

    .rooms-stats {
      display: flex;
      gap: 24px;
      margin-bottom: 24px;
    }

    .stat {
      background: #16213e;
      padding: 16px 24px;
      border-radius: 8px;
      display: flex;
      flex-direction: column;
    }

    .stat-value {
      font-size: 28px;
      font-weight: 600;
      color: #fff;
    }

    .stat-label {
      font-size: 12px;
      color: #888;
      text-transform: uppercase;
    }

    .search-bar {
      margin-bottom: 24px;
    }

    .search-input {
      width: 100%;
      max-width: 400px;
      padding: 12px 16px;
      background: #16213e;
      border: 1px solid #2d3748;
      border-radius: 8px;
      color: #fff;
      font-size: 14px;
    }

    .search-input:focus {
      outline: none;
      border-color: #4ade80;
    }

    .rooms-grid {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .room-card {
      background: #16213e;
      border-radius: 8px;
      cursor: pointer;
      transition: all 0.2s;
      border: 1px solid transparent;
    }

    .room-card:hover {
      border-color: #2d3748;
    }

    .room-card.expanded {
      border-color: #4ade80;
    }

    .room-header {
      padding: 16px 20px;
      display: flex;
      align-items: center;
      gap: 16px;
    }

    .room-icon {
      color: #4ade80;
      flex-shrink: 0;
    }

    .room-info {
      flex: 1;
      min-width: 0;
    }

    .room-info h3 {
      margin: 0 0 4px 0;
      font-size: 16px;
      color: #fff;
    }

    .room-path {
      font-size: 12px;
      color: #888;
      font-family: monospace;
    }

    .room-badges {
      display: flex;
      gap: 8px;
    }

    .badge {
      padding: 4px 8px;
      border-radius: 4px;
      font-size: 11px;
      font-weight: 500;
    }

    .badge.files {
      background: rgba(74, 222, 128, 0.15);
      color: #4ade80;
    }

    .badge.entries {
      background: rgba(251, 191, 36, 0.15);
      color: #fbbf24;
    }

    .room-details {
      padding: 0 20px 20px;
      border-top: 1px solid #2d3748;
      margin-top: 0;
    }

    .room-description {
      color: #aaa;
      font-size: 14px;
      margin: 16px 0;
    }

    .section {
      margin-top: 16px;
    }

    .section h4 {
      margin: 0 0 12px 0;
      font-size: 13px;
      color: #888;
      text-transform: uppercase;
    }

    .entry-list, .file-list {
      list-style: none;
      padding: 0;
      margin: 0;
    }

    .entry-point, .file-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 6px 0;
      font-family: monospace;
      font-size: 13px;
      color: #ccc;
    }

    .entry-point svg {
      color: #fbbf24;
    }

    .file-item svg {
      color: #60a5fa;
    }

    .show-more {
      background: none;
      border: none;
      color: #4ade80;
      cursor: pointer;
      padding: 8px 0;
      font-size: 13px;
    }

    .show-more:hover {
      text-decoration: underline;
    }

    .loading, .error, .no-results {
      text-align: center;
      padding: 48px;
      color: #888;
    }

    .error {
      color: #f87171;
    }

    .no-results code {
      background: #2d3748;
      padding: 2px 6px;
      border-radius: 4px;
      font-size: 13px;
    }
  `]
})
export class RoomsComponent implements OnInit {
  private readonly api = inject(ApiService);

  rooms = signal<Room[]>([]);
  filteredRooms = signal<Room[]>([]);
  loading = signal(true);
  error = signal<string | null>(null);
  expandedRoom = signal<string | null>(null);
  showAllFiles = signal(false);
  searchTerm = '';

  ngOnInit() {
    this.loadRooms();
  }

  loadRooms() {
    this.loading.set(true);
    this.error.set(null);

    this.api.getRooms().subscribe({
      next: (data) => {
        this.rooms.set(data.rooms || []);
        this.filteredRooms.set(data.rooms || []);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.message || 'Failed to load rooms');
        this.loading.set(false);
      }
    });
  }

  filterRooms() {
    const term = this.searchTerm.toLowerCase();
    if (!term) {
      this.filteredRooms.set(this.rooms());
      return;
    }

    this.filteredRooms.set(
      this.rooms().filter(room =>
        room.name.toLowerCase().includes(term) ||
        room.path.toLowerCase().includes(term)
      )
    );
  }

  toggleRoom(name: string) {
    if (this.expandedRoom() === name) {
      this.expandedRoom.set(null);
      this.showAllFiles.set(false);
    } else {
      this.expandedRoom.set(name);
      this.showAllFiles.set(false);
    }
  }

  toggleShowAll(event: Event) {
    event.stopPropagation();
    this.showAllFiles.set(!this.showAllFiles());
  }

  totalFiles(): number {
    return this.rooms().reduce((sum, room) => sum + (room.files?.length || 0), 0);
  }

  totalEntryPoints(): number {
    return this.rooms().reduce((sum, room) => sum + (room.entryPoints?.length || 0), 0);
  }
}
