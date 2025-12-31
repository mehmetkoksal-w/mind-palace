import { Component, inject, OnInit, signal, ElementRef, ViewChild } from '@angular/core';
import { Router, RouterOutlet, RouterLink, RouterLinkActive } from '@angular/router';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { WebSocketService } from './core/services/websocket.service';
import { ApiService } from './core/services/api.service';
import { ContradictionAlertComponent } from './core/components/contradiction-alert.component';
import { Subject, debounceTime, distinctUntilChanged, switchMap, of } from 'rxjs';

interface SearchResult {
  symbols: Array<{ name: string; file: string; line: number; kind: string }>;
  learnings: Array<{ id: string; content: string; scope: string; confidence: number }>;
  corridor: Array<{ id: string; content: string; source: string }>;
}

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterOutlet, RouterLink, RouterLinkActive, ContradictionAlertComponent],
  template: `
    <div class="app-container">
      <!-- Top Header -->
      <header class="top-header">
        <div class="header-left">
          <div class="logo" routerLink="/overview">
            <svg viewBox="0 0 512 512" xmlns="http://www.w3.org/2000/svg" class="logo-icon">
              <g fill="currentColor">
                <path d="M499.51,335.772l-46.048-130.234C445.439,90.702,349.802,0,232.917,0C110.768,0,11.759,99.019,11.759,221.158v69.684C11.759,412.982,110.768,512,232.917,512c100.571,0,185.406-67.154,212.256-159.054h42.186c4.181,0,8.104-2.032,10.518-5.45C500.291,344.088,500.895,339.712,499.51,335.772z M328.82,214.59c2.511,14.166-2.495,37.128-47.051,37.128c-21.355,0-51.382,0-61.731,0c0,33.737-50.903,25.819-68.779,25.819c-17.911,0-20.832-19.19-17.565-24.178c-55.846-0.417-63.701-58.749-49.988-84.196C89.573,99.96,159.585,50.77,229.242,50.77c91.661,0,85.59,30.009,95.805,30.861c25.03,2.023,59.219,31.269,59.219,65.415C384.267,181.244,370.074,214.59,328.82,214.59z"/>
              </g>
            </svg>
            <span class="logo-text">Mind Palace</span>
          </div>
        </div>

        <!-- Main Navigation -->
        <nav class="main-nav">
          <a routerLink="/overview" routerLinkActive="active" [routerLinkActiveOptions]="{exact: true}">
            <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="7" height="7" rx="1"/>
              <rect x="14" y="3" width="7" height="7" rx="1"/>
              <rect x="3" y="14" width="7" height="7" rx="1"/>
              <rect x="14" y="14" width="7" height="7" rx="1"/>
            </svg>
            Overview
          </a>
          <a routerLink="/explore" routerLinkActive="active">
            <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"/>
              <path d="M21 21l-4.35-4.35"/>
              <path d="M11 8v6M8 11h6"/>
            </svg>
            Explore
          </a>
          <a routerLink="/insights" routerLinkActive="active">
            <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 2L2 7l10 5 10-5-10-5z"/>
              <path d="M2 17l10 5 10-5"/>
              <path d="M2 12l10 5 10-5"/>
            </svg>
            Insights
          </a>
        </nav>

        <!-- Header Right -->
        <div class="header-right">
          <!-- Global Search -->
          <div class="search-container" (click)="openSearch()">
            <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"/>
              <path d="M21 21l-4.35-4.35"/>
            </svg>
            <span class="search-placeholder">Search... (⌘K)</span>
          </div>

          <!-- Connection Status -->
          <div class="status-indicator" [class.connected]="wsConnected()" [title]="wsConnected() ? 'Connected' : 'Disconnected'">
            <span class="status-dot"></span>
          </div>
        </div>
      </header>

      <!-- Main Content -->
      <main class="content">
        <router-outlet></router-outlet>
      </main>

      <!-- Search Modal -->
      <!-- Contradiction Alerts -->
      <app-contradiction-alert></app-contradiction-alert>

      @if (searchOpen()) {
        <div class="search-modal-overlay" (click)="closeSearch()">
          <div class="search-modal" (click)="$event.stopPropagation()">
            <div class="search-input-wrapper">
              <svg class="search-modal-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"/>
                <path d="M21 21l-4.35-4.35"/>
              </svg>
              <input
                #searchInput
                type="text"
                class="search-modal-input"
                placeholder="Search symbols, learnings, corridors..."
                [ngModel]="searchQuery"
                (ngModelChange)="onSearchInput($event)"
                (keyup.escape)="closeSearch()"
                (keyup.enter)="navigateToFirstResult()"
                (keydown.arrowDown)="selectNext()"
                (keydown.arrowUp)="selectPrev()"
              />
              @if (searching()) {
                <div class="search-spinner"></div>
              }
            </div>

            <div class="search-results" (keydown)="onResultsKeydown($event)">
              <!-- No query hint -->
              @if (!searchQuery) {
                <div class="search-hint">
                  <p>Type to search across your codebase and knowledge</p>
                  <div class="search-shortcuts">
                    <span><kbd>↵</kbd> to select</span>
                    <span><kbd>↑↓</kbd> to navigate</span>
                    <span><kbd>esc</kbd> to close</span>
                  </div>
                </div>
              }

              <!-- Loading -->
              @if (searchQuery && searching()) {
                <div class="search-loading">Searching...</div>
              }

              <!-- Results -->
              @if (searchQuery && !searching() && hasResults()) {
                <!-- Symbols -->
                @if (results().symbols?.length) {
                  <div class="result-section">
                    <div class="result-section-header">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
                        <path d="M14 2v6h6"/>
                      </svg>
                      Code Symbols
                      <span class="result-count">{{ results().symbols.length }}</span>
                    </div>
                    @for (symbol of results().symbols.slice(0, 5); track symbol.name + symbol.file; let i = $index) {
                      <div
                        class="result-item"
                        [class.selected]="selectedIndex() === i"
                        (click)="navigateToSymbol(symbol)"
                        (mouseenter)="selectedIndex.set(i)"
                      >
                        <div class="result-icon symbol-icon">
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M16 18l6-6-6-6M8 6l-6 6 6 6"/>
                          </svg>
                        </div>
                        <div class="result-content">
                          <span class="result-title">{{ symbol.name }}</span>
                          <span class="result-meta">{{ getFileName(symbol.file) }}:{{ symbol.line }}</span>
                        </div>
                        <span class="result-badge">{{ symbol.kind }}</span>
                      </div>
                    }
                  </div>
                }

                <!-- Learnings -->
                @if (results().learnings?.length) {
                  <div class="result-section">
                    <div class="result-section-header">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M2 3h6a4 4 0 014 4v14a3 3 0 00-3-3H2z"/>
                        <path d="M22 3h-6a4 4 0 00-4 4v14a3 3 0 013-3h7z"/>
                      </svg>
                      Learnings
                      <span class="result-count">{{ results().learnings.length }}</span>
                    </div>
                    @for (learning of results().learnings.slice(0, 5); track learning.id; let i = $index) {
                      <div
                        class="result-item"
                        [class.selected]="selectedIndex() === (results().symbols?.length || 0) + i"
                        (click)="navigateToLearning(learning)"
                        (mouseenter)="selectedIndex.set((results().symbols?.length || 0) + i)"
                      >
                        <div class="result-icon learning-icon">
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <circle cx="12" cy="12" r="10"/>
                            <path d="M12 16v-4M12 8h.01"/>
                          </svg>
                        </div>
                        <div class="result-content">
                          <span class="result-title">{{ truncate(learning.content, 60) }}</span>
                          <span class="result-meta">{{ learning.scope || 'Global' }}</span>
                        </div>
                        <span class="result-badge confidence">{{ (learning.confidence * 100).toFixed(0) }}%</span>
                      </div>
                    }
                  </div>
                }

                <!-- Corridor (Personal Learnings) -->
                @if (results().corridor?.length) {
                  <div class="result-section">
                    <div class="result-section-header">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2"/>
                        <circle cx="9" cy="7" r="4"/>
                        <path d="M23 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75"/>
                      </svg>
                      Personal Knowledge
                      <span class="result-count">{{ results().corridor.length }}</span>
                    </div>
                    @for (item of results().corridor.slice(0, 5); track item.id; let i = $index) {
                      <div
                        class="result-item"
                        [class.selected]="selectedIndex() === (results().symbols?.length || 0) + (results().learnings?.length || 0) + i"
                        (click)="navigateToCorridor()"
                        (mouseenter)="selectedIndex.set((results().symbols?.length || 0) + (results().learnings?.length || 0) + i)"
                      >
                        <div class="result-icon corridor-icon">
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M12 2L2 7l10 5 10-5-10-5z"/>
                            <path d="M2 17l10 5 10-5"/>
                          </svg>
                        </div>
                        <div class="result-content">
                          <span class="result-title">{{ truncate(item.content, 60) }}</span>
                          <span class="result-meta">{{ item.source || 'Personal' }}</span>
                        </div>
                      </div>
                    }
                  </div>
                }
              }

              <!-- No results -->
              @if (searchQuery && !searching() && !hasResults()) {
                <div class="no-results">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <circle cx="11" cy="11" r="8"/>
                    <path d="M21 21l-4.35-4.35"/>
                    <path d="M8 8l6 6M14 8l-6 6"/>
                  </svg>
                  <p>No results found for "{{ searchQuery }}"</p>
                </div>
              }
            </div>
          </div>
        </div>
      }
    </div>
  `,
  styles: [`
    :host {
      display: block;
      height: 100vh;
    }

    .app-container {
      display: flex;
      flex-direction: column;
      min-height: 100vh;
      background: #0f0f1a;
      color: #e2e8f0;
    }

    /* Top Header */
    .top-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0 1.5rem;
      height: 64px;
      background: #1a1a2e;
      border-bottom: 1px solid #2d2d44;
      position: sticky;
      top: 0;
      z-index: 100;
    }

    .header-left {
      display: flex;
      align-items: center;
    }

    /* Logo */
    .logo {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      cursor: pointer;
      text-decoration: none;
      color: inherit;
    }

    .logo-icon {
      width: 32px;
      height: 32px;
      color: #9d4edd;
    }

    .logo-text {
      font-size: 1.25rem;
      font-weight: 600;
      color: #fff;
    }

    /* Main Navigation */
    .main-nav {
      display: flex;
      gap: 0.5rem;
    }

    .main-nav a {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.625rem 1rem;
      color: #94a3b8;
      text-decoration: none;
      border-radius: 8px;
      font-size: 0.9rem;
      font-weight: 500;
      transition: all 0.2s ease;
    }

    .main-nav a:hover {
      background: rgba(157, 78, 221, 0.1);
      color: #e2e8f0;
    }

    .main-nav a.active {
      background: rgba(157, 78, 221, 0.2);
      color: #9d4edd;
    }

    .nav-icon {
      width: 18px;
      height: 18px;
    }

    /* Header Right */
    .header-right {
      display: flex;
      align-items: center;
      gap: 1rem;
    }

    /* Search */
    .search-container {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      background: #2d2d44;
      border-radius: 8px;
      padding: 0.5rem 1rem;
      cursor: pointer;
      transition: all 0.2s ease;
    }

    .search-container:hover {
      background: #3d3d54;
    }

    .search-icon {
      width: 16px;
      height: 16px;
      color: #64748b;
    }

    .search-placeholder {
      color: #64748b;
      font-size: 0.875rem;
    }

    /* Connection Status */
    .status-indicator {
      display: flex;
      align-items: center;
      padding: 0.5rem;
    }

    .status-dot {
      width: 10px;
      height: 10px;
      border-radius: 50%;
      background: #ef4444;
      transition: background 0.3s ease;
    }

    .status-indicator.connected .status-dot {
      background: #22c55e;
      box-shadow: 0 0 8px rgba(34, 197, 94, 0.5);
    }

    /* Main Content */
    .content {
      flex: 1;
      padding: 1.5rem;
      overflow-y: auto;
    }

    /* Search Modal */
    .search-modal-overlay {
      position: fixed;
      inset: 0;
      background: rgba(0, 0, 0, 0.7);
      display: flex;
      align-items: flex-start;
      justify-content: center;
      padding-top: 12vh;
      z-index: 1000;
      backdrop-filter: blur(4px);
    }

    .search-modal {
      background: #1a1a2e;
      border-radius: 12px;
      width: 100%;
      max-width: 640px;
      border: 1px solid #2d2d44;
      box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
      overflow: hidden;
    }

    .search-input-wrapper {
      display: flex;
      align-items: center;
      padding: 0 1rem;
      border-bottom: 1px solid #2d2d44;
    }

    .search-modal-icon {
      width: 20px;
      height: 20px;
      color: #64748b;
      flex-shrink: 0;
    }

    .search-modal-input {
      flex: 1;
      padding: 1rem 0.75rem;
      background: transparent;
      border: none;
      color: #e2e8f0;
      font-size: 1rem;
      outline: none;
    }

    .search-modal-input::placeholder {
      color: #64748b;
    }

    .search-spinner {
      width: 18px;
      height: 18px;
      border: 2px solid #3d3d54;
      border-top-color: #9d4edd;
      border-radius: 50%;
      animation: spin 0.7s linear infinite;
    }

    @keyframes spin {
      to { transform: rotate(360deg); }
    }

    .search-results {
      max-height: 400px;
      overflow-y: auto;
    }

    .search-hint {
      padding: 2rem;
      text-align: center;
    }

    .search-hint p {
      color: #64748b;
      margin: 0 0 1rem 0;
    }

    .search-shortcuts {
      display: flex;
      justify-content: center;
      gap: 1.5rem;
      font-size: 0.75rem;
      color: #4a5568;
    }

    .search-shortcuts kbd {
      background: #2d2d44;
      padding: 0.2rem 0.4rem;
      border-radius: 4px;
      font-family: inherit;
      margin-right: 0.25rem;
    }

    .search-loading {
      padding: 2rem;
      text-align: center;
      color: #64748b;
    }

    .result-section {
      padding: 0.5rem 0;
    }

    .result-section-header {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.5rem 1rem;
      color: #64748b;
      font-size: 0.75rem;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }

    .result-section-header svg {
      width: 14px;
      height: 14px;
    }

    .result-count {
      background: #2d2d44;
      padding: 0.1rem 0.4rem;
      border-radius: 4px;
      margin-left: auto;
    }

    .result-item {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      padding: 0.625rem 1rem;
      cursor: pointer;
      transition: background 0.15s ease;
    }

    .result-item:hover,
    .result-item.selected {
      background: rgba(157, 78, 221, 0.1);
    }

    .result-icon {
      width: 32px;
      height: 32px;
      border-radius: 6px;
      display: flex;
      align-items: center;
      justify-content: center;
      flex-shrink: 0;
    }

    .result-icon svg {
      width: 16px;
      height: 16px;
    }

    .symbol-icon {
      background: rgba(59, 130, 246, 0.15);
      color: #3b82f6;
    }

    .learning-icon {
      background: rgba(34, 197, 94, 0.15);
      color: #22c55e;
    }

    .corridor-icon {
      background: rgba(157, 78, 221, 0.15);
      color: #9d4edd;
    }

    .result-content {
      flex: 1;
      min-width: 0;
    }

    .result-title {
      display: block;
      color: #e2e8f0;
      font-size: 0.9rem;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .result-meta {
      display: block;
      color: #64748b;
      font-size: 0.75rem;
      margin-top: 0.125rem;
    }

    .result-badge {
      font-size: 0.65rem;
      padding: 0.2rem 0.5rem;
      border-radius: 4px;
      background: #2d2d44;
      color: #94a3b8;
      text-transform: uppercase;
      flex-shrink: 0;
    }

    .result-badge.confidence {
      background: rgba(34, 197, 94, 0.15);
      color: #22c55e;
    }

    .no-results {
      padding: 3rem 2rem;
      text-align: center;
      color: #64748b;
    }

    .no-results svg {
      width: 48px;
      height: 48px;
      margin-bottom: 1rem;
      opacity: 0.5;
    }

    .no-results p {
      margin: 0;
    }

    /* Responsive */
    @media (max-width: 768px) {
      .top-header {
        padding: 0 1rem;
      }

      .logo-text {
        display: none;
      }

      .main-nav a span {
        display: none;
      }

      .search-placeholder {
        display: none;
      }

      .search-modal {
        margin: 0 1rem;
      }
    }

    @media (max-width: 480px) {
      .search-container {
        padding: 0.5rem;
      }
    }
  `]
})
export class AppComponent implements OnInit {
  @ViewChild('searchInput') searchInputRef!: ElementRef<HTMLInputElement>;

  private ws = inject(WebSocketService);
  private api = inject(ApiService);
  private router = inject(Router);

  wsConnected = this.ws.connected;
  searchOpen = signal(false);
  searchQuery = '';
  searching = signal(false);
  results = signal<SearchResult>({ symbols: [], learnings: [], corridor: [] });
  selectedIndex = signal(0);

  private searchSubject = new Subject<string>();

  ngOnInit() {
    this.ws.connect();

    // Listen for Cmd+K / Ctrl+K
    document.addEventListener('keydown', (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        this.openSearch();
      }
    });

    // Setup debounced search
    this.searchSubject.pipe(
      debounceTime(300),
      distinctUntilChanged(),
      switchMap(query => {
        if (!query || query.length < 2) {
          return of({ symbols: [], learnings: [], corridor: [] });
        }
        this.searching.set(true);
        return this.api.search(query, 20);
      })
    ).subscribe({
      next: (res) => {
        this.results.set(res);
        this.searching.set(false);
        this.selectedIndex.set(0);
      },
      error: () => {
        this.results.set({ symbols: [], learnings: [], corridor: [] });
        this.searching.set(false);
      }
    });
  }

  openSearch() {
    this.searchOpen.set(true);
    setTimeout(() => {
      this.searchInputRef?.nativeElement?.focus();
    }, 50);
  }

  closeSearch() {
    this.searchOpen.set(false);
    this.searchQuery = '';
    this.results.set({ symbols: [], learnings: [], corridor: [] });
    this.selectedIndex.set(0);
  }

  onSearchInput(query: string) {
    this.searchQuery = query;
    this.searchSubject.next(query);
  }

  hasResults(): boolean {
    const r = this.results();
    return (r.symbols?.length || 0) + (r.learnings?.length || 0) + (r.corridor?.length || 0) > 0;
  }

  getTotalResults(): number {
    const r = this.results();
    return Math.min(5, r.symbols?.length || 0) +
           Math.min(5, r.learnings?.length || 0) +
           Math.min(5, r.corridor?.length || 0);
  }

  selectNext() {
    const total = this.getTotalResults();
    if (total > 0) {
      this.selectedIndex.set((this.selectedIndex() + 1) % total);
    }
  }

  selectPrev() {
    const total = this.getTotalResults();
    if (total > 0) {
      this.selectedIndex.set((this.selectedIndex() - 1 + total) % total);
    }
  }

  navigateToFirstResult() {
    const r = this.results();
    const idx = this.selectedIndex();
    const symbolsCount = Math.min(5, r.symbols?.length || 0);
    const learningsCount = Math.min(5, r.learnings?.length || 0);

    if (idx < symbolsCount && r.symbols?.[idx]) {
      this.navigateToSymbol(r.symbols[idx]);
    } else if (idx < symbolsCount + learningsCount && r.learnings?.[idx - symbolsCount]) {
      this.navigateToLearning(r.learnings[idx - symbolsCount]);
    } else if (r.corridor?.length) {
      this.navigateToCorridor();
    }
  }

  navigateToSymbol(symbol: { name: string; file: string; line: number }) {
    this.closeSearch();
    this.router.navigate(['/explore/graph'], { queryParams: { symbol: symbol.name } });
  }

  navigateToLearning(learning: { id: string }) {
    this.closeSearch();
    this.router.navigate(['/insights/learnings'], { queryParams: { id: learning.id } });
  }

  navigateToCorridor() {
    this.closeSearch();
    this.router.navigate(['/insights/corridors']);
  }

  onResultsKeydown(event: KeyboardEvent) {
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      this.selectNext();
    } else if (event.key === 'ArrowUp') {
      event.preventDefault();
      this.selectPrev();
    }
  }

  getFileName(path: string): string {
    return path?.split('/').pop() || path;
  }

  truncate(text: string, maxLength: number): string {
    if (!text) return '';
    return text.length > maxLength ? text.substring(0, maxLength) + '...' : text;
  }
}
