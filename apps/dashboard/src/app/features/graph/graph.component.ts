import { Component, OnInit, OnDestroy, inject, signal, ElementRef, ViewChild, AfterViewInit } from '@angular/core';

import { FormsModule } from '@angular/forms';
import { ApiService } from '../../core/services/api.service';
import * as d3 from 'd3';

interface Node {
  id: string;
  name: string;
  file: string;
  kind: string;
  x?: number;
  y?: number;
  fx?: number | null;
  fy?: number | null;
}

interface Link {
  source: string | Node;
  target: string | Node;
  type: string;
}

@Component({
    selector: 'app-graph',
    imports: [FormsModule],
    template: `
    <div class="graph-container">
      <header class="page-header">
        <h1>Call Graph Visualizer</h1>
        <p>Explore function call relationships in your codebase</p>
      </header>

      <div class="controls">
        <div class="search-group">
          <input
            type="text"
            [(ngModel)]="symbolInput"
            (keyup.enter)="searchSymbol()"
            placeholder="Enter function name (e.g., handleLogin)"
            class="search-input"
          />
          <button (click)="searchSymbol()" class="search-btn" [disabled]="loading()">
            @if (loading()) {
              Loading...
            } @else {
              Explore
            }
          </button>
        </div>

        <div class="view-controls">
          <label>
            <input type="checkbox" [(ngModel)]="showCallers" (change)="updateGraph()">
            Show Callers
          </label>
          <label>
            <input type="checkbox" [(ngModel)]="showCallees" (change)="updateGraph()">
            Show Callees
          </label>
          <button (click)="resetZoom()" class="reset-btn">Reset View</button>
        </div>
      </div>

      @if (error()) {
        <div class="error">{{ error() }}</div>
      }

      <div class="graph-wrapper">
        <svg #graphSvg class="graph-svg"></svg>

        @if (!currentSymbol() && !loading()) {
          <div class="placeholder">
            <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="12" r="10"/>
              <circle cx="12" cy="12" r="4"/>
              <path d="M12 2v2M12 20v2M2 12h2M20 12h2"/>
            </svg>
            <p>Enter a function name above to visualize its call graph</p>
          </div>
        }
      </div>

      @if (currentSymbol()) {
        <div class="info-panel">
          <h3>{{ currentSymbol() }}</h3>
          <div class="stats">
            <div class="stat">
              <span class="value">{{ callerCount() }}</span>
              <span class="label">Callers</span>
            </div>
            <div class="stat">
              <span class="value">{{ calleeCount() }}</span>
              <span class="label">Callees</span>
            </div>
          </div>

          @if (selectedNode()) {
            <div class="selected-info">
              <h4>Selected: {{ selectedNode()!.name }}</h4>
              <p class="file-path">{{ selectedNode()!.file }}</p>
              <button (click)="exploreNode(selectedNode()!)" class="explore-btn">
                Explore this function
              </button>
            </div>
          }
        </div>
      }

      <div class="legend">
        <div class="legend-item">
          <span class="dot center"></span>
          Center Node
        </div>
        <div class="legend-item">
          <span class="dot caller"></span>
          Callers
        </div>
        <div class="legend-item">
          <span class="dot callee"></span>
          Callees
        </div>
      </div>
    </div>
  `,
    styles: [`
    .graph-container {
      padding: 24px;
      height: 100%;
      display: flex;
      flex-direction: column;
    }

    .page-header {
      margin-bottom: 16px;
    }

    .page-header h1 {
      margin: 0 0 8px 0;
      font-size: 28px;
      color: #fff;
    }

    .page-header p {
      margin: 0;
      color: #888;
    }

    .controls {
      display: flex;
      gap: 24px;
      margin-bottom: 16px;
      flex-wrap: wrap;
    }

    .search-group {
      display: flex;
      gap: 8px;
      flex: 1;
      min-width: 300px;
    }

    .search-input {
      flex: 1;
      padding: 10px 14px;
      background: #16213e;
      border: 1px solid #2d3748;
      border-radius: 6px;
      color: #fff;
      font-size: 14px;
    }

    .search-input:focus {
      outline: none;
      border-color: #4ade80;
    }

    .search-btn {
      padding: 10px 20px;
      background: #4ade80;
      color: #000;
      border: none;
      border-radius: 6px;
      font-weight: 600;
      cursor: pointer;
      white-space: nowrap;
    }

    .search-btn:hover:not(:disabled) {
      background: #22c55e;
    }

    .search-btn:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }

    .view-controls {
      display: flex;
      align-items: center;
      gap: 16px;
    }

    .view-controls label {
      display: flex;
      align-items: center;
      gap: 6px;
      color: #ccc;
      font-size: 14px;
      cursor: pointer;
    }

    .reset-btn {
      padding: 8px 16px;
      background: transparent;
      border: 1px solid #2d3748;
      border-radius: 6px;
      color: #ccc;
      cursor: pointer;
    }

    .reset-btn:hover {
      border-color: #4ade80;
      color: #4ade80;
    }

    .error {
      background: rgba(248, 113, 113, 0.15);
      color: #f87171;
      padding: 12px 16px;
      border-radius: 6px;
      margin-bottom: 16px;
    }

    .graph-wrapper {
      flex: 1;
      min-height: 400px;
      background: #0f172a;
      border-radius: 8px;
      position: relative;
      overflow: hidden;
    }

    .graph-svg {
      width: 100%;
      height: 100%;
    }

    .placeholder {
      position: absolute;
      top: 50%;
      left: 50%;
      transform: translate(-50%, -50%);
      text-align: center;
      color: #666;
    }

    .placeholder svg {
      margin-bottom: 16px;
      opacity: 0.5;
    }

    .info-panel {
      margin-top: 16px;
      background: #16213e;
      padding: 16px 20px;
      border-radius: 8px;
    }

    .info-panel h3 {
      margin: 0 0 12px 0;
      color: #fff;
      font-family: monospace;
    }

    .stats {
      display: flex;
      gap: 24px;
    }

    .stat {
      display: flex;
      flex-direction: column;
    }

    .stat .value {
      font-size: 24px;
      font-weight: 600;
      color: #fff;
    }

    .stat .label {
      font-size: 12px;
      color: #888;
      text-transform: uppercase;
    }

    .selected-info {
      margin-top: 16px;
      padding-top: 16px;
      border-top: 1px solid #2d3748;
    }

    .selected-info h4 {
      margin: 0 0 4px 0;
      color: #4ade80;
      font-family: monospace;
    }

    .file-path {
      color: #888;
      font-size: 12px;
      margin: 0 0 12px 0;
      font-family: monospace;
    }

    .explore-btn {
      padding: 8px 16px;
      background: transparent;
      border: 1px solid #4ade80;
      border-radius: 6px;
      color: #4ade80;
      cursor: pointer;
      font-size: 13px;
    }

    .explore-btn:hover {
      background: rgba(74, 222, 128, 0.1);
    }

    .legend {
      margin-top: 16px;
      display: flex;
      gap: 24px;
      justify-content: center;
    }

    .legend-item {
      display: flex;
      align-items: center;
      gap: 8px;
      color: #888;
      font-size: 13px;
    }

    .dot {
      width: 12px;
      height: 12px;
      border-radius: 50%;
    }

    .dot.center {
      background: #fbbf24;
    }

    .dot.caller {
      background: #60a5fa;
    }

    .dot.callee {
      background: #4ade80;
    }
  `]
})
export class GraphComponent implements OnInit, AfterViewInit, OnDestroy {
  @ViewChild('graphSvg') graphSvg!: ElementRef<SVGSVGElement>;

  private readonly api = inject(ApiService);

  symbolInput = '';
  showCallers = true;
  showCallees = true;

  loading = signal(false);
  error = signal<string | null>(null);
  currentSymbol = signal<string | null>(null);
  callerCount = signal(0);
  calleeCount = signal(0);
  selectedNode = signal<Node | null>(null);

  private svg: d3.Selection<SVGSVGElement, unknown, null, undefined> | null = null;
  private simulation: d3.Simulation<Node, Link> | null = null;
  private nodes: Node[] = [];
  private links: Link[] = [];
  private callers: any[] = [];
  private callees: any[] = [];
  private zoom: d3.ZoomBehavior<SVGSVGElement, unknown> | null = null;
  private g: d3.Selection<SVGGElement, unknown, null, undefined> | null = null;

  ngOnInit() {}

  ngAfterViewInit() {
    this.initSvg();
  }

  ngOnDestroy() {
    if (this.simulation) {
      this.simulation.stop();
    }
  }

  initSvg() {
    const svg = d3.select(this.graphSvg.nativeElement);
    this.svg = svg;

    this.zoom = d3.zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.1, 4])
      .on('zoom', (event) => {
        if (this.g) {
          this.g.attr('transform', event.transform);
        }
      });

    svg.call(this.zoom);

    this.g = svg.append('g');
  }

  searchSymbol() {
    const symbol = this.symbolInput.trim();
    if (!symbol) return;

    this.loading.set(true);
    this.error.set(null);
    this.selectedNode.set(null);

    this.api.getGraph(symbol).subscribe({
      next: (data) => {
        this.currentSymbol.set(data.symbol);
        this.callers = data.callers || [];
        this.callees = data.callees || [];
        this.callerCount.set(this.callers.length);
        this.calleeCount.set(this.callees.length);
        this.buildGraph();
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.message || 'Failed to load graph data');
        this.loading.set(false);
      }
    });
  }

  updateGraph() {
    this.buildGraph();
  }

  buildGraph() {
    if (!this.g || !this.currentSymbol()) return;

    this.g.selectAll('*').remove();

    const centerNode: Node = {
      id: this.currentSymbol()!,
      name: this.currentSymbol()!,
      file: '',
      kind: 'center'
    };

    this.nodes = [centerNode];
    this.links = [];

    if (this.showCallers) {
      this.callers.forEach((caller, i) => {
        const node: Node = {
          id: `caller-${i}`,
          name: caller.name || caller,
          file: caller.file || '',
          kind: 'caller'
        };
        this.nodes.push(node);
        this.links.push({
          source: node.id,
          target: centerNode.id,
          type: 'calls'
        });
      });
    }

    if (this.showCallees) {
      this.callees.forEach((callee, i) => {
        const node: Node = {
          id: `callee-${i}`,
          name: callee.name || callee,
          file: callee.file || '',
          kind: 'callee'
        };
        this.nodes.push(node);
        this.links.push({
          source: centerNode.id,
          target: node.id,
          type: 'calls'
        });
      });
    }

    this.renderGraph();
  }

  renderGraph() {
    if (!this.g || !this.svg) return;

    const width = this.graphSvg.nativeElement.clientWidth || 800;
    const height = this.graphSvg.nativeElement.clientHeight || 500;

    // Create arrow marker
    this.svg.select('defs').remove();
    const defs = this.svg.append('defs');

    defs.append('marker')
      .attr('id', 'arrowhead')
      .attr('viewBox', '-0 -5 10 10')
      .attr('refX', 25)
      .attr('refY', 0)
      .attr('orient', 'auto')
      .attr('markerWidth', 6)
      .attr('markerHeight', 6)
      .append('path')
      .attr('d', 'M 0,-5 L 10,0 L 0,5')
      .attr('fill', '#666');

    // Create simulation
    this.simulation = d3.forceSimulation<Node>(this.nodes)
      .force('link', d3.forceLink<Node, Link>(this.links)
        .id(d => d.id)
        .distance(120))
      .force('charge', d3.forceManyBody().strength(-300))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collision', d3.forceCollide().radius(40));

    // Draw links
    const link = this.g.selectAll('.link')
      .data(this.links)
      .join('line')
      .attr('class', 'link')
      .attr('stroke', '#666')
      .attr('stroke-width', 1.5)
      .attr('marker-end', 'url(#arrowhead)');

    // Draw nodes
    const node = this.g.selectAll('.node')
      .data(this.nodes)
      .join('g')
      .attr('class', 'node')
      .style('cursor', 'pointer')
      .call(d3.drag<SVGGElement, Node>()
        .on('start', (event, d) => this.dragStarted(event, d))
        .on('drag', (event, d) => this.dragged(event, d))
        .on('end', (event, d) => this.dragEnded(event, d)) as any)
      .on('click', (event, d) => {
        event.stopPropagation();
        this.selectedNode.set(d);
      });

    // Node circles
    node.append('circle')
      .attr('r', d => d.kind === 'center' ? 20 : 14)
      .attr('fill', d => {
        switch (d.kind) {
          case 'center': return '#fbbf24';
          case 'caller': return '#60a5fa';
          case 'callee': return '#4ade80';
          default: return '#666';
        }
      })
      .attr('stroke', '#fff')
      .attr('stroke-width', 2);

    // Node labels
    node.append('text')
      .text(d => d.name.length > 15 ? d.name.slice(0, 15) + '...' : d.name)
      .attr('dy', 30)
      .attr('text-anchor', 'middle')
      .attr('fill', '#ccc')
      .attr('font-size', '11px')
      .attr('font-family', 'monospace');

    // Update positions
    this.simulation.on('tick', () => {
      link
        .attr('x1', (d: any) => d.source.x)
        .attr('y1', (d: any) => d.source.y)
        .attr('x2', (d: any) => d.target.x)
        .attr('y2', (d: any) => d.target.y);

      node.attr('transform', (d: any) => `translate(${d.x},${d.y})`);
    });

    // Reset zoom to fit
    this.resetZoom();
  }

  dragStarted(event: d3.D3DragEvent<SVGGElement, Node, Node>, d: Node) {
    if (!event.active && this.simulation) {
      this.simulation.alphaTarget(0.3).restart();
    }
    d.fx = d.x;
    d.fy = d.y;
  }

  dragged(event: d3.D3DragEvent<SVGGElement, Node, Node>, d: Node) {
    d.fx = event.x;
    d.fy = event.y;
  }

  dragEnded(event: d3.D3DragEvent<SVGGElement, Node, Node>, d: Node) {
    if (!event.active && this.simulation) {
      this.simulation.alphaTarget(0);
    }
    d.fx = null;
    d.fy = null;
  }

  resetZoom() {
    if (!this.svg || !this.zoom) return;

    const width = this.graphSvg.nativeElement.clientWidth || 800;
    const height = this.graphSvg.nativeElement.clientHeight || 500;

    this.svg.transition()
      .duration(750)
      .call(this.zoom.transform as any, d3.zoomIdentity.translate(0, 0).scale(1));
  }

  exploreNode(node: Node) {
    this.symbolInput = node.name;
    this.searchSymbol();
  }
}
