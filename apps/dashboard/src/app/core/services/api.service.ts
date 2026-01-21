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

export interface Idea {
  id: string;
  sessionId: string;
  scope: string;
  scopePath: string;
  content: string;
  status: string; // active, exploring, implemented, dropped
  createdAt: string;
  updatedAt: string;
  tags: string[];
}

export interface Decision {
  id: string;
  sessionId: string;
  scope: string;
  scopePath: string;
  content: string;
  status: string; // active, superseded, reversed
  rationale: string;
  createdAt: string;
  updatedAt: string;
  tags: string[];
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

// Phase 11: Context Preview
export interface PrioritizedLearning {
  learning: Learning;
  priority: number;
  reason: string;
}

export interface FileFailure {
  path: string;
  failureCount: number;
  lastFailure: string;
}

export interface ContextWarning {
  type: string;
  message: string;
  recordId: string;
}

export interface AutoInjectedContext {
  filePath: string;
  room: string;
  learnings: PrioritizedLearning[];
  decisions: Decision[];
  failures: FileFailure[];
  warnings: ContextWarning[];
  totalTokens: number;
  generatedAt: string;
}

// Phase 12: Decision Timeline
export interface ChainedDecision {
  decision: Decision;
  relation: string;
  linkReason: string;
}

export interface DecisionChain {
  current: Decision;
  predecessors: ChainedDecision[];
  successors: ChainedDecision[];
  linkedLearnings: Learning[];
}

export interface TimelineDecision extends Decision {
  outcomeColor: string;
}

// Phase 13: Postmortems
export interface Postmortem {
  id: string;
  title: string;
  whatHappened: string;
  rootCause: string;
  lessonsLearned: string[];
  preventionSteps: string[];
  severity: 'low' | 'medium' | 'high' | 'critical';
  status: 'open' | 'resolved' | 'recurring';
  affectedFiles: string[];
  relatedDecision: string;
  relatedSession: string;
  createdAt: string;
  resolvedAt: string;
}

export interface PostmortemStats {
  total: number;
  open: number;
  resolved: number;
  recurring: number;
  bySeverity: Record<string, number>;
}

// Phase 14: Scope Explorer
export interface ScopeLevel {
  scope: string;
  path: string;
  recordCount: number;
  active: boolean;
}

export interface ScopeExplanation {
  filePath: string;
  resolvedRoom: string;
  inheritanceChain: ScopeLevel[];
  totalRecords: Record<string, number>;
}

export interface ScopeHierarchy {
  levels: ScopeLevelDetail[];
}

export interface ScopeLevelDetail {
  scope: string;
  learnings: Learning[];
  decisions: Decision[];
  ideas: Idea[];
}

// Patterns (Sprint 1)
export interface Pattern {
  id: string;
  category: string;
  subcategory: string;
  name: string;
  description: string;
  detectorId: string;
  confidence: number;
  frequencyScore: number;
  consistencyScore: number;
  spreadScore: number;
  ageScore: number;
  status: 'discovered' | 'approved' | 'ignored';
  authority: string;
  learningId: string;
  locations: PatternLocation[];
  outliers: PatternLocation[];
  metadata: Record<string, any>;
  firstSeen: string;
  lastSeen: string;
  createdAt: string;
  updatedAt: string;
}

export interface PatternLocation {
  id: string;
  patternId: string;
  filePath: string;
  lineStart: number;
  lineEnd: number;
  snippet: string;
  isOutlier: boolean;
  outlierReason: string;
  createdAt: string;
}

export interface PatternStats {
  total: number;
  discovered: number;
  approved: number;
  ignored: number;
  byCategory: Record<string, number>;
  averageConfidence: number;
}

// Contracts (Sprint 2)
export interface Contract {
  id: string;
  method: string;
  endpoint: string;
  endpointPattern: string;
  backend: BackendEndpoint;
  frontendCalls: FrontendCall[];
  mismatches: FieldMismatch[];
  status: 'discovered' | 'verified' | 'mismatch' | 'ignored';
  authority: string;
  confidence: number;
  firstSeen: string;
  lastSeen: string;
  createdAt: string;
  updatedAt: string;
}

export interface BackendEndpoint {
  file: string;
  line: number;
  framework: string;
  handler: string;
  requestSchema?: TypeSchema;
  responseSchema?: TypeSchema;
}

export interface FrontendCall {
  id: string;
  contractId: string;
  file: string;
  line: number;
  callType: string;
  expectedSchema?: TypeSchema;
  createdAt: string;
}

export interface FieldMismatch {
  id: string;
  fieldPath: string;
  type: 'missing_in_frontend' | 'missing_in_backend' | 'type_mismatch' | 'optionality_mismatch' | 'nullability_mismatch';
  severity: 'error' | 'warning' | 'info';
  description: string;
  backendType?: string;
  frontendType?: string;
}

export interface TypeSchema {
  type: string;
  properties?: Record<string, TypeSchema>;
  items?: TypeSchema;
  required?: string[];
  nullable?: boolean;
  format?: string;
}

export interface ContractStats {
  total: number;
  discovered: number;
  verified: number;
  mismatch: number;
  ignored: number;
  byMethod: Record<string, number>;
  totalCalls: number;
  totalErrors: number;
  totalWarnings: number;
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

  // Ideas
  getIdeas(status = '', scope = '', limit = 50): Observable<{ ideas: Idea[]; count: number }> {
    const params: any = { limit: limit.toString() };
    if (status) params.status = status;
    if (scope) params.scope = scope;
    return this.http.get<any>(`${this.baseUrl}/ideas`, { params });
  }

  // Decisions
  getDecisions(status = '', scope = '', limit = 50): Observable<{ decisions: Decision[]; count: number }> {
    const params: any = { limit: limit.toString() };
    if (status) params.status = status;
    if (scope) params.scope = scope;
    return this.http.get<any>(`${this.baseUrl}/decisions`, { params });
  }

  // Conversations
  getConversations(params: {
    sessionId?: string;
    agentType?: string;
    query?: string;
    timeline?: boolean;
    limit?: number;
  } = {}): Observable<any> {
    const queryParams: any = { limit: (params.limit || 50).toString() };
    if (params.sessionId) queryParams.sessionId = params.sessionId;
    if (params.agentType) queryParams.agentType = params.agentType;
    if (params.query) queryParams.q = params.query;
    if (params.timeline) queryParams.timeline = 'true';
    return this.http.get<any>(`${this.baseUrl}/conversations`, { params: queryParams });
  }

  getConversation(id: string): Observable<any> {
    return this.http.get<any>(`${this.baseUrl}/conversations/${id}`);
  }

  getConversationTimeline(id: string): Observable<any> {
    return this.http.get<any>(`${this.baseUrl}/conversations/${id}/timeline`);
  }

  searchConversations(query: string, limit = 50): Observable<any> {
    return this.http.get<any>(`${this.baseUrl}/conversations`, {
      params: { q: query, limit: limit.toString() }
    });
  }

  // Phase 11: Context Preview
  getContextPreview(filePath: string, options?: {
    maxTokens?: number;
    includeLearnings?: boolean;
    includeDecisions?: boolean;
    includeFailures?: boolean;
  }): Observable<AutoInjectedContext> {
    return this.http.post<AutoInjectedContext>(`${this.baseUrl}/context/preview`, {
      filePath,
      ...options
    });
  }

  // Phase 12: Decision Timeline & Chain
  getDecisionTimeline(scope?: string, limit = 100): Observable<{ decisions: TimelineDecision[]; count: number }> {
    const params: any = { limit: limit.toString() };
    if (scope) params.scope = scope;
    return this.http.get<any>(`${this.baseUrl}/decisions/timeline`, { params });
  }

  getDecisionChain(id: string): Observable<DecisionChain> {
    return this.http.get<DecisionChain>(`${this.baseUrl}/decisions/${id}/chain`);
  }

  // Phase 13: Postmortems
  getPostmortems(status?: string, severity?: string, limit = 50): Observable<{ postmortems: Postmortem[]; count: number }> {
    const params: any = { limit: limit.toString() };
    if (status) params.status = status;
    if (severity) params.severity = severity;
    return this.http.get<any>(`${this.baseUrl}/postmortems`, { params });
  }

  getPostmortem(id: string): Observable<Postmortem> {
    return this.http.get<Postmortem>(`${this.baseUrl}/postmortems/${id}`);
  }

  createPostmortem(data: Partial<Postmortem>): Observable<Postmortem> {
    return this.http.post<Postmortem>(`${this.baseUrl}/postmortems`, data);
  }

  updatePostmortem(id: string, data: Partial<Postmortem>): Observable<Postmortem> {
    return this.http.put<Postmortem>(`${this.baseUrl}/postmortems/${id}`, data);
  }

  deletePostmortem(id: string): Observable<{ deleted: boolean; id: string }> {
    return this.http.delete<any>(`${this.baseUrl}/postmortems/${id}`);
  }

  resolvePostmortem(id: string): Observable<Postmortem> {
    return this.http.post<Postmortem>(`${this.baseUrl}/postmortems/${id}/resolve`, {});
  }

  convertPostmortemToLearnings(id: string): Observable<{ created: number; learningIds: string[] }> {
    return this.http.post<any>(`${this.baseUrl}/postmortems/${id}/learnings`, {});
  }

  getPostmortemStats(): Observable<PostmortemStats> {
    return this.http.get<PostmortemStats>(`${this.baseUrl}/postmortems/stats`);
  }

  // Phase 14: Scope Explorer
  getScopeExplanation(filePath: string): Observable<ScopeExplanation> {
    return this.http.post<ScopeExplanation>(`${this.baseUrl}/scope/explain`, { filePath });
  }

  getScopeHierarchy(): Observable<ScopeHierarchy> {
    return this.http.get<ScopeHierarchy>(`${this.baseUrl}/scope/hierarchy`);
  }

  // Decay (Phase 6)
  getDecayStats(): Observable<any> {
    return this.http.get<any>(`${this.baseUrl}/decay/stats`);
  }

  getDecayPreview(): Observable<any> {
    return this.http.get<any>(`${this.baseUrl}/decay/preview`);
  }

  // Smart Briefing (Phase 7)
  getSmartBriefing(context: string, contextPath: string, style = 'summary'): Observable<any> {
    return this.http.post<any>(`${this.baseUrl}/briefings/smart`, {
      context,
      contextPath,
      style
    });
  }

  // Patterns (Sprint 1)
  getPatterns(params: {
    category?: string;
    status?: string;
    minConfidence?: number;
    limit?: number;
  } = {}): Observable<{ patterns: Pattern[]; count: number }> {
    const queryParams: any = { limit: (params.limit || 50).toString() };
    if (params.category) queryParams.category = params.category;
    if (params.status) queryParams.status = params.status;
    if (params.minConfidence) queryParams.minConfidence = params.minConfidence.toString();
    return this.http.get<any>(`${this.baseUrl}/patterns`, { params: queryParams });
  }

  getPattern(id: string): Observable<Pattern> {
    return this.http.get<Pattern>(`${this.baseUrl}/patterns/${id}`);
  }

  approvePattern(id: string, withLearning = false): Observable<Pattern | { pattern: Pattern; learningId: string }> {
    return this.http.post<any>(`${this.baseUrl}/patterns/${id}/approve`, { withLearning });
  }

  ignorePattern(id: string): Observable<Pattern> {
    return this.http.post<Pattern>(`${this.baseUrl}/patterns/${id}/ignore`, {});
  }

  bulkApprovePatterns(ids: string[], withLearning = false): Observable<{ approved: number; patterns: Pattern[]; learningIds?: string[] }> {
    return this.http.post<any>(`${this.baseUrl}/patterns/bulk-approve`, { ids, withLearning });
  }

  getPatternStats(): Observable<PatternStats> {
    return this.http.get<PatternStats>(`${this.baseUrl}/patterns/stats`);
  }

  // Contracts (Sprint 2)
  getContracts(params: {
    method?: string;
    status?: string;
    endpoint?: string;
    hasMismatches?: boolean;
    limit?: number;
  } = {}): Observable<{ contracts: Contract[]; count: number }> {
    const queryParams: any = { limit: (params.limit || 50).toString() };
    if (params.method) queryParams.method = params.method;
    if (params.status) queryParams.status = params.status;
    if (params.endpoint) queryParams.endpoint = params.endpoint;
    if (params.hasMismatches) queryParams.mismatches = 'true';
    return this.http.get<any>(`${this.baseUrl}/contracts`, { params: queryParams });
  }

  getContract(id: string): Observable<Contract> {
    return this.http.get<Contract>(`${this.baseUrl}/contracts/${id}`);
  }

  verifyContract(id: string): Observable<Contract> {
    return this.http.post<Contract>(`${this.baseUrl}/contracts/${id}/verify`, {});
  }

  ignoreContract(id: string): Observable<Contract> {
    return this.http.post<Contract>(`${this.baseUrl}/contracts/${id}/ignore`, {});
  }

  getContractStats(): Observable<ContractStats> {
    return this.http.get<ContractStats>(`${this.baseUrl}/contracts/stats`);
  }
}
