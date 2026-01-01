import { Component, OnInit, OnDestroy, AfterViewInit, inject, signal, input, output, ElementRef, ViewChild } from '@angular/core';

import { FormsModule } from '@angular/forms';
import * as d3 from 'd3';
import { NeuralMapService } from './neural-map.service';
import { NeuralNode, NeuralLink, NeuralMapData, LINK_COLORS, NodeType } from './neural-map.types';
import { NeuralMapLegendComponent } from './neural-map-legend.component';

@Component({
    selector: 'app-neural-map',
    imports: [FormsModule, NeuralMapLegendComponent],
    template: `
    <div class="neural-map-container" #container [style.height.px]="height()">
      @if (loading()) {
        <div class="loading-overlay">
          <div class="spinner"></div>
          <span>Building neural map...</span>
        </div>
      }

      <svg #mapSvg class="neural-map-svg"></svg>

      <div class="controls-overlay">
        <div class="zoom-controls">
          <button (click)="zoomIn()" title="Zoom In">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"/><path d="M21 21l-4.35-4.35M11 8v6M8 11h6"/>
            </svg>
          </button>
          <button (click)="zoomOut()" title="Zoom Out">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"/><path d="M21 21l-4.35-4.35M8 11h6"/>
            </svg>
          </button>
          <button (click)="fitToScreen()" title="Fit to Screen">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M8 3H5a2 2 0 00-2 2v3m18 0V5a2 2 0 00-2-2h-3m0 18h3a2 2 0 002-2v-3M3 16v3a2 2 0 002 2h3"/>
            </svg>
          </button>
        </div>

        <div class="filter-controls">
          <label>
            <input type="checkbox" [(ngModel)]="showRooms" (change)="updateFilters()">
            Rooms
          </label>
          <label>
            <input type="checkbox" [(ngModel)]="showIdeas" (change)="updateFilters()">
            Ideas
          </label>
          <label>
            <input type="checkbox" [(ngModel)]="showDecisions" (change)="updateFilters()">
            Decisions
          </label>
          <label>
            <input type="checkbox" [(ngModel)]="showLearnings" (change)="updateFilters()">
            Learnings
          </label>
        </div>
      </div>

      <app-neural-map-legend class="legend-overlay" />

      @if (hoveredNode()) {
        <div class="tooltip" [style.left.px]="tooltipPos().x" [style.top.px]="tooltipPos().y">
          <div class="tooltip-header">
            <span class="type-badge" [class]="hoveredNode()!.type">{{ hoveredNode()!.type }}</span>
          </div>
          <div class="tooltip-label">{{ hoveredNode()!.label }}</div>
          <div class="tooltip-meta">{{ hoveredNode()!.connectionCount }} connections</div>
        </div>
      }

      @if (selectedNode()) {
        <div class="detail-panel">
          <div class="panel-header">
            <span class="type-badge" [class]="selectedNode()!.type">{{ selectedNode()!.type }}</span>
            <button class="close-btn" (click)="deselectAll()">x</button>
          </div>
          <h4>{{ selectedNode()!.label }}</h4>
          <div class="panel-content">
            @if (selectedNode()!.type === 'room') {
              <p class="detail-text">{{ selectedNode()!.data.description || 'Code module' }}</p>
              <div class="stats-row">
                <span>{{ selectedNode()!.data.files?.length || 0 }} files</span>
                <span>{{ selectedNode()!.data.entryPoints?.length || 0 }} entry points</span>
              </div>
            } @else {
              <p class="detail-text">{{ getFullContent() }}</p>
              @if (selectedNode()!.data.scope) {
                <span class="scope-badge">{{ selectedNode()!.data.scope }}</span>
              }
            }
          </div>
        </div>
      }

      @if (!loading() && nodes.length === 0) {
        <div class="empty-state">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="12" cy="12" r="10"/>
            <path d="M12 6v6l4 2"/>
          </svg>
          <p>No data to visualize yet</p>
          <span>Add rooms, ideas, or learnings to see them here</span>
        </div>
      }
    </div>
  `,
    styles: [`
    .neural-map-container {
      position: relative;
      width: 100%;
      height: 100%;
      min-height: 400px;
      background: #0f172a;
      border-radius: 8px;
      overflow: hidden;
    }

    .neural-map-svg {
      width: 100%;
      height: 100%;
      cursor: grab;
    }

    .neural-map-svg:active {
      cursor: grabbing;
    }

    .loading-overlay {
      position: absolute;
      inset: 0;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      background: rgba(15, 23, 42, 0.9);
      gap: 1rem;
      color: #64748b;
      z-index: 10;
    }

    .spinner {
      width: 32px;
      height: 32px;
      border: 3px solid #2d2d44;
      border-top-color: #9d4edd;
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
      to { transform: rotate(360deg); }
    }

    .controls-overlay {
      position: absolute;
      top: 12px;
      right: 12px;
      display: flex;
      flex-direction: column;
      gap: 8px;
      z-index: 5;
    }

    .zoom-controls {
      display: flex;
      flex-direction: column;
      background: #1a1a2e;
      border-radius: 6px;
      overflow: hidden;
      border: 1px solid #2d2d44;
    }

    .zoom-controls button {
      width: 36px;
      height: 36px;
      background: transparent;
      border: none;
      color: #94a3b8;
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: center;
      transition: all 0.2s;
    }

    .zoom-controls button:hover {
      background: rgba(157, 78, 221, 0.2);
      color: #9d4edd;
    }

    .zoom-controls button svg {
      width: 18px;
      height: 18px;
    }

    .filter-controls {
      display: flex;
      flex-direction: column;
      background: #1a1a2e;
      border-radius: 6px;
      padding: 8px;
      border: 1px solid #2d2d44;
      gap: 4px;
    }

    .filter-controls label {
      display: flex;
      align-items: center;
      gap: 6px;
      color: #94a3b8;
      font-size: 0.75rem;
      cursor: pointer;
    }

    .filter-controls input[type="checkbox"] {
      accent-color: #9d4edd;
    }

    .legend-overlay {
      position: absolute;
      bottom: 12px;
      left: 12px;
      z-index: 5;
    }

    .tooltip {
      position: absolute;
      background: rgba(26, 26, 46, 0.95);
      border: 1px solid #2d2d44;
      border-radius: 6px;
      padding: 8px 12px;
      pointer-events: none;
      z-index: 20;
      max-width: 250px;
    }

    .tooltip-header {
      margin-bottom: 4px;
    }

    .tooltip-label {
      color: #e2e8f0;
      font-size: 0.85rem;
      line-height: 1.3;
    }

    .tooltip-meta {
      color: #64748b;
      font-size: 0.7rem;
      margin-top: 4px;
    }

    .type-badge {
      font-size: 0.6rem;
      font-weight: 600;
      text-transform: uppercase;
      padding: 2px 6px;
      border-radius: 3px;
    }

    .type-badge.room { background: rgba(157, 78, 221, 0.2); color: #9d4edd; }
    .type-badge.idea { background: rgba(251, 191, 36, 0.2); color: #fbbf24; }
    .type-badge.decision { background: rgba(74, 222, 128, 0.2); color: #4ade80; }
    .type-badge.learning { background: rgba(0, 180, 216, 0.2); color: #00b4d8; }
    .type-badge.symbol { background: rgba(96, 165, 250, 0.2); color: #60a5fa; }

    .detail-panel {
      position: absolute;
      top: 12px;
      left: 12px;
      width: 260px;
      background: #1a1a2e;
      border-radius: 8px;
      border: 1px solid #2d2d44;
      overflow: hidden;
      z-index: 5;
    }

    .panel-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 10px 12px;
      border-bottom: 1px solid #2d2d44;
    }

    .close-btn {
      width: 22px;
      height: 22px;
      background: transparent;
      border: none;
      color: #64748b;
      cursor: pointer;
      font-size: 14px;
      border-radius: 4px;
    }

    .close-btn:hover {
      background: rgba(255, 255, 255, 0.1);
      color: #e2e8f0;
    }

    .detail-panel h4 {
      margin: 0;
      padding: 8px 12px 0;
      color: #e2e8f0;
      font-size: 0.9rem;
      font-weight: 500;
    }

    .panel-content {
      padding: 8px 12px 12px;
    }

    .detail-text {
      font-size: 0.8rem;
      color: #94a3b8;
      margin: 0 0 8px 0;
      line-height: 1.4;
    }

    .stats-row {
      display: flex;
      gap: 12px;
      font-size: 0.7rem;
      color: #64748b;
    }

    .scope-badge {
      display: inline-block;
      font-size: 0.65rem;
      padding: 2px 6px;
      background: #2d2d44;
      border-radius: 3px;
      color: #94a3b8;
    }

    .empty-state {
      position: absolute;
      inset: 0;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      color: #64748b;
      text-align: center;
    }

    .empty-state svg {
      width: 48px;
      height: 48px;
      margin-bottom: 12px;
      opacity: 0.5;
    }

    .empty-state p {
      margin: 0 0 4px 0;
      color: #94a3b8;
    }

    .empty-state span {
      font-size: 0.8rem;
    }
  `]
})
export class NeuralMapComponent implements OnInit, AfterViewInit, OnDestroy {
  @ViewChild('mapSvg') mapSvg!: ElementRef<SVGSVGElement>;
  @ViewChild('container') container!: ElementRef<HTMLDivElement>;

  height = input<number>(400);
  nodeSelected = output<NeuralNode>();

  private readonly neuralMapService = inject(NeuralMapService);

  loading = signal(true);
  hoveredNode = signal<NeuralNode | null>(null);
  selectedNode = signal<NeuralNode | null>(null);
  tooltipPos = signal({ x: 0, y: 0 });

  showRooms = true;
  showIdeas = true;
  showDecisions = true;
  showLearnings = true;

  nodes: NeuralNode[] = [];
  links: NeuralLink[] = [];
  private allNodes: NeuralNode[] = [];
  private allLinks: NeuralLink[] = [];

  private svg: d3.Selection<SVGSVGElement, unknown, null, undefined> | null = null;
  private g: d3.Selection<SVGGElement, unknown, null, undefined> | null = null;
  private simulation: d3.Simulation<NeuralNode, NeuralLink> | null = null;
  private zoom: d3.ZoomBehavior<SVGSVGElement, unknown> | null = null;

  ngOnInit() {
    this.loadData();
  }

  ngAfterViewInit() {
    this.initSvg();
  }

  ngOnDestroy() {
    if (this.simulation) {
      this.simulation.stop();
    }
  }

  private loadData() {
    this.loading.set(true);
    this.neuralMapService.fetchMapData().subscribe({
      next: (data) => {
        this.allNodes = data.nodes;
        this.allLinks = data.links;
        this.updateFilters();
        this.loading.set(false);
        setTimeout(() => this.renderGraph(), 50);
      },
      error: (err) => {
        console.error('Failed to load neural map data:', err);
        this.loading.set(false);
      }
    });
  }

  private initSvg() {
    if (!this.mapSvg) return;

    const svg = d3.select(this.mapSvg.nativeElement);
    this.svg = svg;

    this.zoom = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.2, 4])
      .on('zoom', (event) => {
        if (this.g) {
          this.g.attr('transform', event.transform);
        }
        this.hoveredNode.set(null);
      });

    svg.call(this.zoom);
    svg.on('click', () => this.deselectAll());

    this.g = svg.append('g');

    // Create defs for markers and filters
    this.createDefs(svg);
  }

  private createDefs(svg: d3.Selection<SVGSVGElement, unknown, null, undefined>) {
    const defs = svg.append('defs');

    // Arrow markers for different link types
    const markerConfigs = [
      { id: 'arrow-references', color: LINK_COLORS.references },
      { id: 'arrow-supports', color: LINK_COLORS.supports },
      { id: 'arrow-implements', color: LINK_COLORS.implements },
      { id: 'arrow-refines', color: LINK_COLORS.refines },
      { id: 'arrow-depends', color: LINK_COLORS.depends },
    ];

    markerConfigs.forEach(config => {
      defs.append('marker')
        .attr('id', config.id)
        .attr('viewBox', '0 -5 10 10')
        .attr('refX', 20)
        .attr('refY', 0)
        .attr('markerWidth', 5)
        .attr('markerHeight', 5)
        .attr('orient', 'auto')
        .append('path')
        .attr('d', 'M0,-5L10,0L0,5')
        .attr('fill', config.color);
    });

    // Glow filter for selected nodes
    const filter = defs.append('filter')
      .attr('id', 'glow')
      .attr('x', '-50%')
      .attr('y', '-50%')
      .attr('width', '200%')
      .attr('height', '200%');

    filter.append('feGaussianBlur')
      .attr('stdDeviation', '3')
      .attr('result', 'coloredBlur');

    const feMerge = filter.append('feMerge');
    feMerge.append('feMergeNode').attr('in', 'coloredBlur');
    feMerge.append('feMergeNode').attr('in', 'SourceGraphic');
  }

  updateFilters() {
    const visibleTypes = new Set<NodeType>();
    if (this.showRooms) visibleTypes.add('room');
    if (this.showIdeas) visibleTypes.add('idea');
    if (this.showDecisions) visibleTypes.add('decision');
    if (this.showLearnings) visibleTypes.add('learning');

    this.nodes = this.allNodes.filter(n => visibleTypes.has(n.type));
    const nodeIds = new Set(this.nodes.map(n => n.id));

    this.links = this.allLinks.filter(link => {
      const sourceId = typeof link.source === 'string' ? link.source : link.source.id;
      const targetId = typeof link.target === 'string' ? link.target : link.target.id;
      return nodeIds.has(sourceId) && nodeIds.has(targetId);
    });

    if (this.g) {
      this.renderGraph();
    }
  }

  private renderGraph() {
    if (!this.g || !this.svg || this.nodes.length === 0) return;

    this.g.selectAll('*').remove();

    // Use container dimensions for better sizing
    const containerRect = this.container.nativeElement.getBoundingClientRect();
    const width = containerRect.width || 800;
    const height = containerRect.height || this.height() || 420;

    // Reset node positions for fresh simulation - spread across full area
    this.nodes.forEach(node => {
      if (node.x === undefined) {
        node.x = width * 0.2 + Math.random() * width * 0.6;
        node.y = height * 0.2 + Math.random() * height * 0.6;
      }
    });

    // Create simulation with stronger forces to spread nodes
    this.simulation = d3.forceSimulation<NeuralNode>(this.nodes)
      .force('link', d3.forceLink<NeuralNode, NeuralLink>(this.links)
        .id(d => d.id)
        .distance(d => this.getLinkDistance(d) * 1.5)
        .strength(d => d.strength * 0.4))
      .force('charge', d3.forceManyBody<NeuralNode>()
        .strength(d => d.type === 'room' ? -500 : -200)
        .distanceMax(400))
      .force('center', d3.forceCenter(width / 2, height / 2).strength(0.03))
      .force('collision', d3.forceCollide<NeuralNode>()
        .radius(d => d.radius + 15)
        .strength(0.8))
      .force('x', d3.forceX(width / 2).strength(0.02))
      .force('y', d3.forceY(height / 2).strength(0.02))
      .alphaDecay(0.015)
      .velocityDecay(0.35);

    // Draw links
    const link = this.g.selectAll('.link')
      .data(this.links)
      .join('line')
      .attr('class', 'link')
      .attr('stroke', d => LINK_COLORS[d.type])
      .attr('stroke-width', d => d.type === 'contains' ? 1 : d.type === 'depends' ? 2.5 : 1.5)
      .attr('stroke-dasharray', d => d.type === 'contradicts' ? '4,4' : d.type === 'refines' ? '2,2' : 'none')
      .attr('stroke-opacity', d => d.type === 'depends' ? 0.8 : 0.6)
      .attr('marker-end', d => ['references', 'supports', 'implements', 'refines', 'depends'].includes(d.type) ? `url(#arrow-${d.type})` : null);

    // Draw nodes
    const node = this.g.selectAll<SVGGElement, NeuralNode>('.node')
      .data(this.nodes)
      .join('g')
      .attr('class', 'node')
      .style('cursor', 'pointer')
      .call(d3.drag<SVGGElement, NeuralNode>()
        .on('start', (event, d) => this.dragStarted(event, d))
        .on('drag', (event, d) => this.dragged(event, d))
        .on('end', (event, d) => this.dragEnded(event, d)) as any);

    // Node shapes based on type
    node.each((d, i, nodes) => {
      const g = d3.select(nodes[i]);

      if (d.type === 'room') {
        // Room: circle with thicker border
        g.append('circle')
          .attr('r', d.radius)
          .attr('fill', d.color)
          .attr('fill-opacity', 0.2)
          .attr('stroke', d.color)
          .attr('stroke-width', 3);
      } else if (d.type === 'idea') {
        // Idea: diamond
        const size = d.radius * 1.4;
        g.append('path')
          .attr('d', `M0,${-size} L${size},0 L0,${size} L${-size},0 Z`)
          .attr('fill', d.color)
          .attr('stroke', '#1a1a2e')
          .attr('stroke-width', 2);
      } else if (d.type === 'decision') {
        // Decision: hexagon
        const size = d.radius;
        const points = this.hexagonPoints(size);
        g.append('polygon')
          .attr('points', points)
          .attr('fill', d.color)
          .attr('stroke', '#1a1a2e')
          .attr('stroke-width', 2);
      } else if (d.type === 'learning') {
        // Learning: rounded rect
        const w = d.radius * 2;
        const h = d.radius * 1.4;
        g.append('rect')
          .attr('x', -w / 2)
          .attr('y', -h / 2)
          .attr('width', w)
          .attr('height', h)
          .attr('rx', 4)
          .attr('fill', d.color)
          .attr('stroke', '#1a1a2e')
          .attr('stroke-width', 2);
      } else {
        // Default: circle
        g.append('circle')
          .attr('r', d.radius)
          .attr('fill', d.color)
          .attr('stroke', '#1a1a2e')
          .attr('stroke-width', 2);
      }
    });

    // Node labels (only for rooms and important nodes)
    node.filter(d => d.type === 'room' || d.importance > 0.5)
      .append('text')
      .text(d => d.label.length > 12 ? d.label.slice(0, 12) + '...' : d.label)
      .attr('dy', d => d.radius + 14)
      .attr('text-anchor', 'middle')
      .attr('fill', '#94a3b8')
      .attr('font-size', '10px')
      .attr('pointer-events', 'none');

    // Interactions
    node
      .on('mouseenter', (event, d) => {
        const rect = this.container.nativeElement.getBoundingClientRect();
        this.tooltipPos.set({
          x: event.clientX - rect.left + 15,
          y: event.clientY - rect.top - 10
        });
        this.hoveredNode.set(d);
      })
      .on('mouseleave', () => {
        this.hoveredNode.set(null);
      })
      .on('click', (event, d) => {
        event.stopPropagation();
        this.selectNode(d);
      });

    // Simulation tick
    this.simulation.on('tick', () => {
      link
        .attr('x1', (d: any) => d.source.x)
        .attr('y1', (d: any) => d.source.y)
        .attr('x2', (d: any) => d.target.x)
        .attr('y2', (d: any) => d.target.y);

      node.attr('transform', (d: any) => `translate(${d.x},${d.y})`);
    });

    // Initial fit
    setTimeout(() => this.fitToScreen(), 500);
  }

  private hexagonPoints(radius: number): string {
    const points: [number, number][] = [];
    for (let i = 0; i < 6; i++) {
      const angle = (Math.PI / 3) * i - Math.PI / 2;
      points.push([radius * Math.cos(angle), radius * Math.sin(angle)]);
    }
    return points.map(p => p.join(',')).join(' ');
  }

  private getLinkDistance(link: NeuralLink): number {
    if (link.type === 'contains') return 50;
    if (link.type === 'depends') return 120;  // Longer for room-to-room dependencies
    if (link.type === 'references' || link.type === 'implements') return 80;
    return 100;
  }

  private dragStarted(event: d3.D3DragEvent<SVGGElement, NeuralNode, NeuralNode>, d: NeuralNode) {
    if (!event.active && this.simulation) {
      this.simulation.alphaTarget(0.3).restart();
    }
    d.fx = d.x;
    d.fy = d.y;
  }

  private dragged(event: d3.D3DragEvent<SVGGElement, NeuralNode, NeuralNode>, d: NeuralNode) {
    d.fx = event.x;
    d.fy = event.y;
  }

  private dragEnded(event: d3.D3DragEvent<SVGGElement, NeuralNode, NeuralNode>, d: NeuralNode) {
    if (!event.active && this.simulation) {
      this.simulation.alphaTarget(0);
    }
    d.fx = null;
    d.fy = null;
  }

  private selectNode(node: NeuralNode) {
    this.selectedNode.set(node);
    this.nodeSelected.emit(node);

    // Highlight connected
    const connectedIds = new Set<string>([node.id]);
    this.links.forEach(link => {
      const sourceId = typeof link.source === 'string' ? link.source : link.source.id;
      const targetId = typeof link.target === 'string' ? link.target : link.target.id;
      if (sourceId === node.id) connectedIds.add(targetId);
      if (targetId === node.id) connectedIds.add(sourceId);
    });

    this.g?.selectAll('.node')
      .attr('opacity', (d: any) => connectedIds.has(d.id) ? 1 : 0.25);

    this.g?.selectAll('.link')
      .attr('opacity', (d: any) => {
        const sourceId = typeof d.source === 'string' ? d.source : d.source.id;
        const targetId = typeof d.target === 'string' ? d.target : d.target.id;
        return sourceId === node.id || targetId === node.id ? 1 : 0.1;
      });

    // Glow effect
    this.g?.selectAll('.node')
      .filter((d: any) => d.id === node.id)
      .attr('filter', 'url(#glow)');
  }

  deselectAll() {
    this.selectedNode.set(null);
    this.g?.selectAll('.node').attr('opacity', 1).attr('filter', null);
    this.g?.selectAll('.link').attr('opacity', 0.6);
  }

  getFullContent(): string {
    const node = this.selectedNode();
    if (!node) return '';
    return node.data.content || node.data.description || node.label;
  }

  zoomIn() {
    if (!this.svg || !this.zoom) return;
    this.svg.transition().duration(300).call(this.zoom.scaleBy as any, 1.4);
  }

  zoomOut() {
    if (!this.svg || !this.zoom) return;
    this.svg.transition().duration(300).call(this.zoom.scaleBy as any, 0.7);
  }

  fitToScreen() {
    if (!this.svg || !this.zoom || this.nodes.length === 0) return;

    const width = this.mapSvg.nativeElement.clientWidth || 800;
    const height = this.mapSvg.nativeElement.clientHeight || 400;

    const xs = this.nodes.map(n => n.x || 0);
    const ys = this.nodes.map(n => n.y || 0);
    const minX = Math.min(...xs);
    const maxX = Math.max(...xs);
    const minY = Math.min(...ys);
    const maxY = Math.max(...ys);

    const graphWidth = Math.max(maxX - minX, 100);
    const graphHeight = Math.max(maxY - minY, 100);
    const padding = 60;

    const scale = Math.min(
      (width - padding * 2) / graphWidth,
      (height - padding * 2) / graphHeight,
      1.5
    );

    const centerX = (minX + maxX) / 2;
    const centerY = (minY + maxY) / 2;

    this.svg.transition().duration(500).call(
      this.zoom.transform as any,
      d3.zoomIdentity
        .translate(width / 2, height / 2)
        .scale(scale)
        .translate(-centerX, -centerY)
    );
  }
}
