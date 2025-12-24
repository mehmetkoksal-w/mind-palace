import { Routes } from '@angular/router';

export const routes: Routes = [
  { path: '', redirectTo: '/overview', pathMatch: 'full' },
  {
    path: 'overview',
    loadComponent: () => import('./features/overview/overview.component').then(m => m.OverviewComponent)
  },
  {
    path: 'rooms',
    loadComponent: () => import('./features/rooms/rooms.component').then(m => m.RoomsComponent)
  },
  {
    path: 'sessions',
    loadComponent: () => import('./features/sessions/sessions.component').then(m => m.SessionsComponent)
  },
  {
    path: 'learnings',
    loadComponent: () => import('./features/learnings/learnings.component').then(m => m.LearningsComponent)
  },
  {
    path: 'intel',
    loadComponent: () => import('./features/intel/intel.component').then(m => m.IntelComponent)
  },
  {
    path: 'graph',
    loadComponent: () => import('./features/graph/graph.component').then(m => m.GraphComponent)
  },
  {
    path: 'corridors',
    loadComponent: () => import('./features/corridors/corridors.component').then(m => m.CorridorsComponent)
  }
];
