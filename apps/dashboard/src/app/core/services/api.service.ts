import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface Session {
  id: string;
  agentType: string;
  agentId: string;
  goal: string;
  startedAt: string;
  lastActivity: string;
  state: string;
  summary: string;
}

export interface Learning {
  id: string;
  sessionId: string;
  scope: string;
  scopePath: string;
  content: string;
  confidence: number;
  source: string;
  createdAt: string;
  lastUsed: string;
  useCount: number;
}

export interface FileIntel {
  path: string;
  editCount: number;
  failureCount: number;
  lastEdited: string;
  lastEditor: string;
}

export interface ActiveAgent {
  agentId: string;
  agentType: string;
  sessionId: string;
  heartbeat: string;
  currentFile: string;
}

export interface Stats {
  sessions: { total: number; active: number };
  learnings: number;
  filesTracked: number;
  rooms: number;
  corridor?: {
    learningCount: number;
    linkedWorkspaces: number;
    averageConfidence: number;
  };
}

export interface Room {
  name: string;
  path: string;
  files: string[];
  entryPoints: string[];
  description?: string;
}

export interface Activity {
  id: string;
  sessionId: string;
  kind: string;
  target: string;
  details: string;
  timestamp: string;
  outcome: string;
}

export interface GraphNode {
  id: string;
  name: string;
  file: string;
  kind: string;
}

export interface GraphLink {
  source: string;
  target: string;
  type: string;
}

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = '/api';

  // Health & Stats
  getHealth(): Observable<{ status: string; timestamp: string }> {
    return this.http.get<any>(`${this.baseUrl}/health`);
  }

  getStats(): Observable<Stats> {
    return this.http.get<Stats>(`${this.baseUrl}/stats`);
  }

  // Sessions
  getSessions(activeOnly = false, limit = 50): Observable<{ sessions: Session[]; count: number }> {
    return this.http.get<any>(`${this.baseUrl}/sessions`, {
      params: { active: activeOnly.toString(), limit: limit.toString() }
    });
  }

  getSession(id: string): Observable<{ session: Session; activities: any[] }> {
    return this.http.get<any>(`${this.baseUrl}/sessions/${id}`);
  }

  // Learnings
  getLearnings(scope = '', query = '', limit = 50): Observable<{ learnings: Learning[]; count: number }> {
    const params: any = { limit: limit.toString() };
    if (scope) params.scope = scope;
    if (query) params.query = query;
    return this.http.get<any>(`${this.baseUrl}/learnings`, { params });
  }

  // File Intel
  getFileIntel(path: string): Observable<{ intel: FileIntel; learnings: Learning[] }> {
    return this.http.get<any>(`${this.baseUrl}/file-intel`, { params: { path } });
  }

  getHotspots(limit = 20): Observable<{ hotspots: FileIntel[]; fragile: FileIntel[] }> {
    return this.http.get<any>(`${this.baseUrl}/hotspots`, { params: { limit: limit.toString() } });
  }

  // Agents
  getActiveAgents(): Observable<{ agents: ActiveAgent[]; count: number }> {
    return this.http.get<any>(`${this.baseUrl}/agents`);
  }

  // Corridors
  getCorridors(): Observable<{ stats: any; links: any[] }> {
    return this.http.get<any>(`${this.baseUrl}/corridors`);
  }

  getPersonalLearnings(query = '', limit = 50): Observable<{ learnings: any[]; count: number }> {
    const params: any = { limit: limit.toString() };
    if (query) params.query = query;
    return this.http.get<any>(`${this.baseUrl}/corridors/personal`, { params });
  }

  // Search
  search(query: string, limit = 20): Observable<any> {
    return this.http.get<any>(`${this.baseUrl}/search`, {
      params: { q: query, limit: limit.toString() }
    });
  }

  // Brief
  getBrief(path = ''): Observable<any> {
    const params: any = {};
    if (path) params.path = path;
    return this.http.get<any>(`${this.baseUrl}/brief`, { params });
  }

  // Rooms
  getRooms(): Observable<{ rooms: Room[]; count: number }> {
    return this.http.get<any>(`${this.baseUrl}/rooms`);
  }

  // Graph
  getGraph(symbol: string, file = ''): Observable<{ symbol: string; callers: any[]; callees: any[] }> {
    const params: any = {};
    if (file) params.file = file;
    return this.http.get<any>(`${this.baseUrl}/graph/${encodeURIComponent(symbol)}`, { params });
  }

  // Activity
  getActivity(sessionId = '', path = '', limit = 50): Observable<{ activities: Activity[]; count: number }> {
    const params: any = { limit: limit.toString() };
    if (sessionId) params.sessionId = sessionId;
    if (path) params.path = path;
    return this.http.get<any>(`${this.baseUrl}/activity`, { params });
  }
}
