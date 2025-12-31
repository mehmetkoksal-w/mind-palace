export type NodeType = 'room' | 'symbol' | 'idea' | 'decision' | 'learning';
export type LinkType = 'contains' | 'references' | 'supports' | 'contradicts' | 'refines' | 'implements' | 'depends';

export interface NeuralNode {
  id: string;
  type: NodeType;
  label: string;
  data: any;
  x?: number;
  y?: number;
  fx?: number | null;
  fy?: number | null;
  radius: number;
  color: string;
  group: string;
  connectionCount: number;
  importance: number;
  // Visualization enhancements
  confidence?: number;           // 0-1, affects opacity
  contradictionCount?: number;   // Red pulse if > 0
  decayRisk?: boolean;          // Yellow indicator if at risk
  lastActivity?: string;        // For timeline filtering
}

export interface NeuralLink {
  id: string;
  source: string | NeuralNode;
  target: string | NeuralNode;
  type: LinkType;
  strength: number;
}

export interface NeuralMapData {
  nodes: NeuralNode[];
  links: NeuralLink[];
}

export const NODE_COLORS: Record<NodeType, string> = {
  room: '#9d4edd',
  symbol: '#60a5fa',
  idea: '#fbbf24',
  decision: '#4ade80',
  learning: '#00b4d8'
};

export const LINK_COLORS: Record<LinkType, string> = {
  contains: '#4a5568',
  references: '#60a5fa',
  supports: '#4ade80',
  contradicts: '#f87171',
  refines: '#fbbf24',
  implements: '#9d4edd',
  depends: '#f472b6'  // Pink for dependencies
};

export const NODE_RADII: Record<NodeType, number> = {
  room: 28,
  symbol: 10,
  idea: 14,
  decision: 16,
  learning: 12
};
