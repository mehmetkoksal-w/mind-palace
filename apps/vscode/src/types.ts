export type RoomCapability =
    | 'search.text'
    | 'read.file'
    | 'graph.deps'
    | 'tests.run'
    | 'lint.run'
    | 'symbols.lookup';

export interface RoomArtifact {
    name: string;
    description?: string;
    pathHint?: string;
}

export interface RoomStep {
    name: string;
    description?: string;
    capability?: string;
    evidence?: string;
}

export interface Provenance {
    createdBy: string;
    createdAt: string;
    updatedBy?: string;
    updatedAt?: string;
    generator?: string;
    generatorVersion?: string;
}

export interface Room {
    schemaVersion: '1.0.0';
    kind: 'palace/room';
    name: string;
    summary: string;
    entryPoints: string[];
    artifacts?: RoomArtifact[];
    capabilities?: RoomCapability[];
    steps?: RoomStep[];
    provenance: Provenance;
}

export interface Guardrails {
    doNotTouchGlobs?: string[];
    readOnlyGlobs?: string[];
}

export interface ProjectInfo {
    name: string;
    description?: string;
    language?: string;
    repository?: string;
}

export interface NeighborAuth {
    type: 'none' | 'bearer' | 'basic' | 'header';
    token?: string;
    user?: string;
    pass?: string;
    header?: string;
    value?: string;
}

export interface Neighbor {
    url?: string;
    localPath?: string;
    auth?: NeighborAuth;
    ttl?: string;
    enabled?: boolean;
}

export interface VSCodeConfig {
    autoSync?: boolean;
    autoSyncDelay?: number;
    waitForCleanWorkspace?: boolean;
    decorations?: {
        enabled?: boolean;
        style?: 'gutter' | 'inline' | 'both';
    };
    statusBar?: {
        position?: 'left' | 'right';
        priority?: number;
    };
    sidebar?: {
        defaultView?: 'tree' | 'graph';
        graphLayout?: 'cose' | 'circle' | 'grid' | 'breadthfirst';
    };
}

export interface PalaceConfig {
    schemaVersion: '1.0.0';
    kind: 'palace/config';
    project: ProjectInfo;
    defaultRoom?: string;
    guardrails?: Guardrails;
    neighbors?: Record<string, Neighbor>;
    provenance: Provenance;
    vscode?: VSCodeConfig;
}

export type HUDStatus = 'fresh' | 'stale' | 'scanning' | 'pending';

export interface GraphNode {
    data: {
        id: string;
        label: string;
        type: 'room' | 'file' | 'ghost';
        parent?: string;
        fullPath?: string;
        description?: string;
        snippet?: string;
        lineNumber?: number;
    };
}

export interface GraphEdge {
    data: {
        id: string;
        source: string;
        target: string;
    };
}

export interface GraphData {
    nodes: GraphNode[];
    edges: GraphEdge[];
}

export interface SearchMatch {
    filePath: string;
    snippet: string;
    lineNumber?: number;
    score?: number;
}

export interface RoomSearchResult {
    roomName: string;
    matches: SearchMatch[];
}

export interface SearchResults {
    query: string;
    results: RoomSearchResult[];
    totalMatches: number;
}
