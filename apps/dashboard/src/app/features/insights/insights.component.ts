import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet, RouterLink, RouterLinkActive } from '@angular/router';

@Component({
  selector: 'app-insights',
  standalone: true,
  imports: [CommonModule, RouterOutlet, RouterLink, RouterLinkActive],
  template: `
    <div class="insights-container">
      <!-- Sub Navigation -->
      <nav class="sub-nav">
        <a routerLink="sessions" routerLinkActive="active">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <path d="M12 6v6l4 2"/>
          </svg>
          Sessions
        </a>
        <a routerLink="learnings" routerLinkActive="active">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M2 3h6a4 4 0 014 4v14a3 3 0 00-3-3H2z"/>
            <path d="M22 3h-6a4 4 0 00-4 4v14a3 3 0 013-3h7z"/>
          </svg>
          Learnings
        </a>
        <a routerLink="ideas" routerLinkActive="active">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"/>
          </svg>
          Ideas
        </a>
        <a routerLink="decisions" routerLinkActive="active">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M9 11l3 3L22 4"/>
            <path d="M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11"/>
          </svg>
          Decisions
        </a>
        <a routerLink="corridors" routerLinkActive="active">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2"/>
            <circle cx="9" cy="7" r="4"/>
            <path d="M23 21v-2a4 4 0 00-3-3.87"/>
            <path d="M16 3.13a4 4 0 010 7.75"/>
          </svg>
          Corridors
        </a>
        <a routerLink="conversations" routerLinkActive="active">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z"/>
          </svg>
          Conversations
        </a>
        <a routerLink="contradictions" routerLinkActive="active">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
          </svg>
          Contradictions
        </a>
        <a routerLink="postmortems" routerLinkActive="active">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z"/>
          </svg>
          Postmortems
        </a>
        <a routerLink="decision-timeline" routerLinkActive="active">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="3" y1="12" x2="21" y2="12"/>
            <circle cx="6" cy="12" r="2"/>
            <circle cx="12" cy="12" r="2"/>
            <circle cx="18" cy="12" r="2"/>
          </svg>
          Timeline
        </a>
        <a routerLink="context-preview" routerLinkActive="active">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <circle cx="12" cy="12" r="6"/>
            <circle cx="12" cy="12" r="2"/>
          </svg>
          Context
        </a>
        <a routerLink="scope-explorer" routerLinkActive="active">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/>
          </svg>
          Scope
        </a>
      </nav>

      <!-- Content Area -->
      <div class="sub-content">
        <router-outlet></router-outlet>
      </div>
    </div>
  `,
  styles: [`
    .insights-container {
      display: flex;
      flex-direction: column;
      height: 100%;
    }

    .sub-nav {
      display: flex;
      gap: 0.5rem;
      padding-bottom: 1rem;
      margin-bottom: 1rem;
      border-bottom: 1px solid #2d2d44;
    }

    .sub-nav a {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.5rem 1rem;
      color: #94a3b8;
      text-decoration: none;
      border-radius: 6px;
      font-size: 0.875rem;
      font-weight: 500;
      transition: all 0.2s ease;
    }

    .sub-nav a:hover {
      background: rgba(157, 78, 221, 0.1);
      color: #e2e8f0;
    }

    .sub-nav a.active {
      background: rgba(157, 78, 221, 0.15);
      color: #9d4edd;
    }

    .nav-icon {
      width: 16px;
      height: 16px;
    }

    .sub-content {
      flex: 1;
      overflow-y: auto;
    }

    @media (max-width: 768px) {
      .sub-nav {
        overflow-x: auto;
        padding-bottom: 0.75rem;
        -webkit-overflow-scrolling: touch;
      }

      .sub-nav a {
        white-space: nowrap;
        padding: 0.5rem 0.75rem;
      }
    }
  `]
})
export class InsightsComponent {}
