import { Component } from '@angular/core';
import { RouterOutlet, RouterLink, RouterLinkActive } from '@angular/router';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet, RouterLink, RouterLinkActive],
  template: `
    <div class="app-container">
      <nav class="sidebar">
        <div class="logo">
          <h1>Mind Palace</h1>
        </div>
        <ul class="nav-links">
          <li>
            <a routerLink="/overview" routerLinkActive="active">Overview</a>
          </li>
          <li>
            <a routerLink="/rooms" routerLinkActive="active">Rooms</a>
          </li>
          <li>
            <a routerLink="/graph" routerLinkActive="active">Call Graph</a>
          </li>
          <li>
            <a routerLink="/sessions" routerLinkActive="active">Sessions</a>
          </li>
          <li>
            <a routerLink="/learnings" routerLinkActive="active">Learnings</a>
          </li>
          <li>
            <a routerLink="/intel" routerLinkActive="active">File Intel</a>
          </li>
          <li>
            <a routerLink="/corridors" routerLinkActive="active">Corridors</a>
          </li>
        </ul>
      </nav>
      <main class="content">
        <router-outlet></router-outlet>
      </main>
    </div>
  `,
  styles: [`
    .app-container {
      display: flex;
      min-height: 100vh;
      background: #1a1a2e;
      color: #eee;
    }

    .sidebar {
      width: 220px;
      background: #16213e;
      padding: 1rem;
      border-right: 1px solid #2d3748;
    }

    .logo h1 {
      color: #9d4edd;
      font-size: 1.5rem;
      margin: 0 0 2rem 0;
      padding: 0.5rem;
    }

    .nav-links {
      list-style: none;
      padding: 0;
      margin: 0;
    }

    .nav-links li {
      margin-bottom: 0.5rem;
    }

    .nav-links a {
      display: block;
      padding: 0.75rem 1rem;
      color: #a0aec0;
      text-decoration: none;
      border-radius: 8px;
      transition: all 0.2s;
    }

    .nav-links a:hover {
      background: #2d3748;
      color: #fff;
    }

    .nav-links a.active {
      background: #9d4edd;
      color: #fff;
    }

    .content {
      flex: 1;
      padding: 2rem;
      overflow-y: auto;
    }
  `]
})
export class AppComponent {
  title = 'Mind Palace Dashboard';
}
