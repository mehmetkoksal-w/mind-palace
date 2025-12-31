import { Injectable, inject } from '@angular/core';
import { forkJoin, map, Observable, of } from 'rxjs';
import { catchError } from 'rxjs/operators';
import { ApiService, Room, Idea, Decision, Learning } from '../../../core/services/api.service';
import { NeuralNode, NeuralLink, NeuralMapData, NodeType, LinkType, NODE_COLORS, NODE_RADII } from './neural-map.types';

const MAX_NODES = 150;
const MAX_LINKS = 300;

@Injectable({
  providedIn: 'root'
})
export class NeuralMapService {
  private readonly api = inject(ApiService);

  fetchMapData(): Observable<NeuralMapData> {
    return forkJoin({
      rooms: this.api.getRooms().pipe(catchError(() => of({ rooms: [], count: 0 }))),
      ideas: this.api.getIdeas('', '', 50).pipe(catchError(() => of({ ideas: [], count: 0 }))),
      decisions: this.api.getDecisions('', '', 50).pipe(catchError(() => of({ decisions: [], count: 0 }))),
      learnings: this.api.getLearnings('', '', 50).pipe(catchError(() => of({ learnings: [], count: 0 })))
    }).pipe(
      map(data => this.transformToGraph(data))
    );
  }

  private transformToGraph(data: {
    rooms: { rooms: Room[]; count: number };
    ideas: { ideas: Idea[]; count: number };
    decisions: { decisions: Decision[]; count: number };
    learnings: { learnings: Learning[]; count: number };
  }): NeuralMapData {
    const nodes: NeuralNode[] = [];
    const links: NeuralLink[] = [];
    const roomNodes = new Map<string, NeuralNode>();

    // Create room nodes
    (data.rooms.rooms || []).forEach(room => {
      const node = this.createNode('room', room.name, room.name, room, 'global');
      nodes.push(node);
      roomNodes.set(room.name, node);
    });

    // Create inter-room dependency links based on project structure
    this.createRoomDependencyLinks(data.rooms.rooms || [], roomNodes, links);

    // Create idea nodes and link to rooms
    (data.ideas.ideas || []).forEach(idea => {
      const roomName = this.findRoomForScope(idea.scopePath, data.rooms.rooms || []);
      const node = this.createNode('idea', idea.id, this.truncate(idea.content, 40), idea, roomName);
      nodes.push(node);

      if (roomName !== 'global' && roomNodes.has(roomName)) {
        links.push(this.createLink(node.id, roomNodes.get(roomName)!.id, 'references'));
      }
    });

    // Create decision nodes and link to rooms
    (data.decisions.decisions || []).forEach(decision => {
      const roomName = this.findRoomForScope(decision.scopePath, data.rooms.rooms || []);
      const node = this.createNode('decision', decision.id, this.truncate(decision.content, 40), decision, roomName);
      nodes.push(node);

      if (roomName !== 'global' && roomNodes.has(roomName)) {
        links.push(this.createLink(node.id, roomNodes.get(roomName)!.id, 'references'));
      }
    });

    // Create learning nodes and link to rooms
    (data.learnings.learnings || []).forEach(learning => {
      const roomName = this.findRoomForScope(learning.scopePath, data.rooms.rooms || []);
      const node = this.createNode('learning', learning.id, this.truncate(learning.content, 40), learning, roomName);
      nodes.push(node);

      if (roomName !== 'global' && roomNodes.has(roomName)) {
        links.push(this.createLink(node.id, roomNodes.get(roomName)!.id, 'references'));
      }
    });

    // Link global knowledge items to all rooms (weak connections)
    this.linkGlobalKnowledge(nodes, links, roomNodes);

    // Create knowledge item inter-links based on matching tags/scopes
    this.createKnowledgeLinks(nodes, links, data);

    // Calculate connection counts and importance
    this.calculateMetrics(nodes, links);

    // Limit nodes and links if necessary
    return this.limitGraph(nodes, links);
  }

  private createNode(type: NodeType, id: string, label: string, data: any, group: string): NeuralNode {
    return {
      id: `${type}-${id}`,
      type,
      label,
      data,
      radius: NODE_RADII[type],
      color: NODE_COLORS[type],
      group,
      connectionCount: 0,
      importance: type === 'room' ? 1 : 0.5
    };
  }

  private createLink(sourceId: string, targetId: string, type: LinkType): NeuralLink {
    return {
      id: `${sourceId}-${targetId}`,
      source: sourceId,
      target: targetId,
      type,
      strength: type === 'contains' ? 0.8 : type === 'references' ? 0.5 : 0.3
    };
  }

  private createRoomDependencyLinks(rooms: Room[], roomNodes: Map<string, NeuralNode>, links: NeuralLink[]): void {
    // Categorize rooms into apps and packages based on entry point paths
    const apps: Room[] = [];
    const packages: Room[] = [];
    const corePackages: Room[] = [];

    rooms.forEach(room => {
      const entryPoint = room.entryPoints?.[0] || '';
      if (entryPoint.includes('/apps/') || entryPoint.startsWith('apps/')) {
        apps.push(room);
      } else if (entryPoint.includes('/packages/') || entryPoint.startsWith('packages/')) {
        // Core packages are typically named with 'core' or are foundational
        if (room.name.includes('core') || room.name.includes('shared') || room.name.includes('common')) {
          corePackages.push(room);
        } else {
          packages.push(room);
        }
      }
    });

    // Apps depend on core packages (strong dependency)
    apps.forEach(app => {
      const appNode = roomNodes.get(app.name);
      if (!appNode) return;

      corePackages.forEach(pkg => {
        const pkgNode = roomNodes.get(pkg.name);
        if (pkgNode) {
          links.push({
            id: `dep-${app.name}-${pkg.name}`,
            source: appNode.id,
            target: pkgNode.id,
            type: 'depends',
            strength: 0.7
          });
        }
      });
    });

    // Apps depend on other packages (localization, widgets, etc.)
    apps.forEach(app => {
      const appNode = roomNodes.get(app.name);
      if (!appNode) return;

      packages.forEach(pkg => {
        const pkgNode = roomNodes.get(pkg.name);
        if (pkgNode) {
          // Check for likely dependencies based on naming
          const appName = app.name.toLowerCase();
          const pkgName = pkg.name.toLowerCase();

          // Localization is used by most apps
          if (pkgName.includes('localization') || pkgName.includes('l10n') || pkgName.includes('i18n')) {
            links.push({
              id: `dep-${app.name}-${pkg.name}`,
              source: appNode.id,
              target: pkgNode.id,
              type: 'depends',
              strength: 0.6
            });
          }
          // Widgets/UI packages are commonly shared
          else if (pkgName.includes('widget') || pkgName.includes('ui') || pkgName.includes('component')) {
            links.push({
              id: `dep-${app.name}-${pkg.name}`,
              source: appNode.id,
              target: pkgNode.id,
              type: 'depends',
              strength: 0.6
            });
          }
          // App-specific packages (e.g., driver app uses printer package)
          else if (appName.includes('pos') && (pkgName.includes('printer') || pkgName.includes('payment'))) {
            links.push({
              id: `dep-${app.name}-${pkg.name}`,
              source: appNode.id,
              target: pkgNode.id,
              type: 'depends',
              strength: 0.6
            });
          }
        }
      });
    });

    // Packages may depend on core packages
    packages.forEach(pkg => {
      const pkgNode = roomNodes.get(pkg.name);
      if (!pkgNode) return;

      corePackages.forEach(corePkg => {
        const coreNode = roomNodes.get(corePkg.name);
        if (coreNode) {
          links.push({
            id: `dep-${pkg.name}-${corePkg.name}`,
            source: pkgNode.id,
            target: coreNode.id,
            type: 'depends',
            strength: 0.5
          });
        }
      });
    });
  }

  private linkGlobalKnowledge(nodes: NeuralNode[], links: NeuralLink[], roomNodes: Map<string, NeuralNode>): void {
    // Find global knowledge items (those with group === 'global')
    const globalItems = nodes.filter(n => n.group === 'global' && n.type !== 'room');
    const rooms = Array.from(roomNodes.values());

    // Link global decisions to a few central rooms
    globalItems.filter(n => n.type === 'decision').forEach(decision => {
      // Link to up to 3 rooms
      rooms.slice(0, 3).forEach(room => {
        links.push({
          id: `${decision.id}-${room.id}`,
          source: decision.id,
          target: room.id,
          type: 'implements',
          strength: 0.2
        });
      });
    });

    // Link global learnings to one central room
    globalItems.filter(n => n.type === 'learning').forEach(learning => {
      if (rooms.length > 0) {
        const centralRoom = rooms[Math.floor(rooms.length / 2)];
        links.push({
          id: `${learning.id}-${centralRoom.id}`,
          source: learning.id,
          target: centralRoom.id,
          type: 'references',
          strength: 0.15
        });
      }
    });
  }

  private findRoomForScope(scopePath: string, rooms: Room[]): string {
    if (!scopePath) return 'global';

    for (const room of rooms) {
      // Check if scopePath matches any entry point
      if (room.entryPoints?.some(ep => scopePath.includes(ep) || ep.includes(scopePath))) {
        return room.name;
      }
      // Check if scopePath is in the same directory as entry points
      const scopeDir = scopePath.split('/').slice(0, -1).join('/');
      if (room.entryPoints?.some(ep => {
        const epDir = ep.split('/').slice(0, -1).join('/');
        return scopeDir.startsWith(epDir) || epDir.startsWith(scopeDir);
      })) {
        return room.name;
      }
      // Check against files array if present
      if (room.files?.some(f => scopePath.includes(f) || f.includes(scopePath))) {
        return room.name;
      }
    }
    return 'global';
  }

  private createKnowledgeLinks(nodes: NeuralNode[], links: NeuralLink[], data: any): void {
    const ideasByTag = new Map<string, NeuralNode[]>();
    const decisionsByTag = new Map<string, NeuralNode[]>();

    // Group by tags
    nodes.filter(n => n.type === 'idea').forEach(node => {
      const tags = node.data.tags || [];
      tags.forEach((tag: string) => {
        if (!ideasByTag.has(tag)) ideasByTag.set(tag, []);
        ideasByTag.get(tag)!.push(node);
      });
    });

    nodes.filter(n => n.type === 'decision').forEach(node => {
      const tags = node.data.tags || [];
      tags.forEach((tag: string) => {
        if (!decisionsByTag.has(tag)) decisionsByTag.set(tag, []);
        decisionsByTag.get(tag)!.push(node);
      });
    });

    // Link ideas that share tags (supports relationship)
    for (const [, taggedNodes] of ideasByTag) {
      if (taggedNodes.length > 1) {
        for (let i = 0; i < Math.min(taggedNodes.length - 1, 3); i++) {
          links.push(this.createLink(taggedNodes[i].id, taggedNodes[i + 1].id, 'supports'));
        }
      }
    }

    // Link decisions to related ideas
    for (const [tag, decisions] of decisionsByTag) {
      const relatedIdeas = ideasByTag.get(tag) || [];
      for (const decision of decisions.slice(0, 2)) {
        for (const idea of relatedIdeas.slice(0, 2)) {
          links.push(this.createLink(decision.id, idea.id, 'implements'));
        }
      }
    }
  }

  private calculateMetrics(nodes: NeuralNode[], links: NeuralLink[]): void {
    const connectionCounts = new Map<string, number>();

    links.forEach(link => {
      const sourceId = typeof link.source === 'string' ? link.source : link.source.id;
      const targetId = typeof link.target === 'string' ? link.target : link.target.id;

      connectionCounts.set(sourceId, (connectionCounts.get(sourceId) || 0) + 1);
      connectionCounts.set(targetId, (connectionCounts.get(targetId) || 0) + 1);
    });

    const maxConnections = Math.max(...connectionCounts.values(), 1);

    nodes.forEach(node => {
      node.connectionCount = connectionCounts.get(node.id) || 0;
      // Importance based on connections and type
      const typeWeight = node.type === 'room' ? 1.5 : node.type === 'decision' ? 1.2 : 1;
      node.importance = (node.connectionCount / maxConnections) * typeWeight;
      // Adjust radius based on importance
      node.radius = NODE_RADII[node.type] * (1 + node.importance * 0.3);
    });
  }

  private limitGraph(nodes: NeuralNode[], links: NeuralLink[]): NeuralMapData {
    // Sort by importance and limit
    if (nodes.length > MAX_NODES) {
      nodes.sort((a, b) => b.importance - a.importance);
      nodes.length = MAX_NODES;
    }

    const nodeIds = new Set(nodes.map(n => n.id));

    // Filter links to only include existing nodes
    let filteredLinks = links.filter(link => {
      const sourceId = typeof link.source === 'string' ? link.source : link.source.id;
      const targetId = typeof link.target === 'string' ? link.target : link.target.id;
      return nodeIds.has(sourceId) && nodeIds.has(targetId);
    });

    // Limit links
    if (filteredLinks.length > MAX_LINKS) {
      filteredLinks.sort((a, b) => b.strength - a.strength);
      filteredLinks.length = MAX_LINKS;
    }

    return { nodes, links: filteredLinks };
  }

  private truncate(text: string, maxLength: number): string {
    if (!text) return '';
    return text.length > maxLength ? text.slice(0, maxLength) + '...' : text;
  }
}
