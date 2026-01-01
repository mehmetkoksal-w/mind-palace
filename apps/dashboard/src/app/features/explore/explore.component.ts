import { Component } from '@angular/core';

import { RouterOutlet, RouterLink, RouterLinkActive } from '@angular/router';

@Component({
    selector: 'app-explore',
    imports: [RouterOutlet, RouterLink, RouterLinkActive],
    template: `
    <div class="explore-container">
      <!-- Sub Navigation -->
      <nav class="sub-nav">
        <a routerLink="rooms" routerLinkActive="active">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 9l9-7 9 7v11a2 2 0 01-2 2H5a2 2 0 01-2-2V9z"/>
            <path d="M9 22V12h6v10"/>
          </svg>
          Rooms
        </a>
        <a routerLink="graph" routerLinkActive="active">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="5" r="3"/>
            <circle cx="5" cy="19" r="3"/>
            <circle cx="19" cy="19" r="3"/>
            <line x1="12" y1="8" x2="5" y2="16"/>
            <line x1="12" y1="8" x2="19" y2="16"/>
          </svg>
          Call Graph
        </a>
        <a routerLink="intel" routerLinkActive="active">
          <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/>
            <path d="M14 2v6h6"/>
            <path d="M16 13H8"/>
            <path d="M16 17H8"/>
            <path d="M10 9H8"/>
          </svg>
          File Intel
        </a>
      </nav>

      <!-- Content Area -->
      <div class="sub-content">
        <router-outlet></router-outlet>
      </div>
    </div>
  `,
    styles: [`
    .explore-container {
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

    @media (max-width: 600px) {
      .sub-nav {
        overflow-x: auto;
        padding-bottom: 0.75rem;
      }

      .sub-nav a {
        white-space: nowrap;
      }
    }
  `]
})
export class ExploreComponent {}
