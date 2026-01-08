/**
 * Knowledge Graph webview script
 * Bundled with D3.js to run in VS Code webview
 */

import * as d3 from "d3";

// Declare vscode API
declare const acquireVsCodeApi: () => any;

interface GraphNode extends d3.SimulationNodeDatum {
  id: string;
  kind: "idea" | "decision" | "learning";
  content: string;
  confidence?: number;
  status?: string;
  links: number;
}

interface GraphLink extends d3.SimulationLinkDatum<GraphNode> {
  source: string | GraphNode;
  target: string | GraphNode;
  relation: string;
}

// Initialize VS Code API
const vscode = acquireVsCodeApi();

// State
let nodes: GraphNode[] = [];
let links: GraphLink[] = [];
let simulation: d3.Simulation<GraphNode, GraphLink> | null = null;
let svg: d3.Selection<SVGSVGElement, unknown, HTMLElement, any> | null = null;
let g: d3.Selection<SVGGElement, unknown, HTMLElement, any> | null = null;
let link: d3.Selection<SVGLineElement, GraphLink, SVGGElement, unknown> | null =
  null;
let node: d3.Selection<SVGGElement, GraphNode, SVGGElement, unknown> | null =
  null;
let zoom: d3.ZoomBehavior<SVGSVGElement, unknown> | null = null;

// DOM elements
const container = document.getElementById("graph")!;
const tooltip = document.getElementById("tooltip")!;

// ══════════════════════════════════════════════════════════════════
// MESSAGE HANDLER
// ══════════════════════════════════════════════════════════════════

window.addEventListener("message", (event) => {
  const message = event.data;
  switch (message.type) {
    case "data":
      nodes = message.nodes;
      links = message.links;
      renderGraph();
      break;
    case "error":
      container.innerHTML = '<div class="error">' + message.message + "</div>";
      break;
  }
});

// ══════════════════════════════════════════════════════════════════
// BUTTON HANDLERS
// ══════════════════════════════════════════════════════════════════

document.getElementById("refresh")?.addEventListener("click", () => {
  container.innerHTML = '<div class="loading">Loading...</div>';
  vscode.postMessage({ type: "refresh" });
});

document.getElementById("zoomIn")?.addEventListener("click", () => {
  if (svg && zoom) {
    svg.transition().call(zoom.scaleBy, 1.3);
  }
});

document.getElementById("zoomOut")?.addEventListener("click", () => {
  if (svg && zoom) {
    svg.transition().call(zoom.scaleBy, 0.7);
  }
});

document.getElementById("reset")?.addEventListener("click", () => {
  if (svg && zoom) {
    svg.transition().call(zoom.transform, d3.zoomIdentity);
  }
});

// ══════════════════════════════════════════════════════════════════
// GRAPH RENDERING
// ══════════════════════════════════════════════════════════════════

function renderGraph() {
  container.innerHTML = "";

  if (nodes.length === 0) {
    container.innerHTML =
      '<div class="loading">No knowledge records found</div>';
    return;
  }

  const width = container.clientWidth;
  const height = container.clientHeight;

  // Create SVG
  svg = d3
    .select("#graph")
    .append("svg")
    .attr("width", width)
    .attr("height", height);

  // Add zoom behavior
  zoom = d3
    .zoom<SVGSVGElement, unknown>()
    .scaleExtent([0.1, 4])
    .on("zoom", (event) => {
      if (g) {
        g.attr("transform", event.transform);
      }
    });

  svg.call(zoom);

  g = svg.append("g");

  // Create simulation
  simulation = d3
    .forceSimulation(nodes)
    .force(
      "link",
      d3
        .forceLink<GraphNode, GraphLink>(links)
        .id((d) => d.id)
        .distance(100)
    )
    .force("charge", d3.forceManyBody<GraphNode>().strength(-300))
    .force("center", d3.forceCenter(width / 2, height / 2))
    .force("collision", d3.forceCollide<GraphNode>().radius(30));

  // Create links
  link = g
    .append("g")
    .selectAll("line")
    .data(links)
    .join("line")
    .attr("class", (d) => "link " + d.relation) as d3.Selection<
    SVGLineElement,
    GraphLink,
    SVGGElement,
    unknown
  >;

  // Create nodes
  node = g
    .append("g")
    .selectAll("g")
    .data(nodes)
    .join("g")
    .attr("class", (d) => "node " + d.kind)
    .call(drag(simulation) as any)
    .on("click", (event, d) => {
      vscode.postMessage({ type: "showDetail", node: d });
    })
    .on("mouseover", (event, d) => {
      showTooltip(event, d);
    })
    .on("mouseout", () => {
      tooltip.style.display = "none";
    }) as d3.Selection<SVGGElement, GraphNode, SVGGElement, unknown>;

  if (node) {
    node
      .append("circle")
      .attr("r", (d) => 8 + d.links * 2)
      .attr("opacity", (d) => (d.confidence ? 0.5 + d.confidence * 0.5 : 0.8));
  }

  if (node) {
    node
      .append("text")
      .attr("dx", 12)
      .attr("dy", 4)
      .text((d) => truncate(d.content, 30));
  }

  // Update positions on tick
  simulation.on("tick", () => {
    if (link) {
      link
        .attr("x1", (d) => (d.source as GraphNode).x || 0)
        .attr("y1", (d) => (d.source as GraphNode).y || 0)
        .attr("x2", (d) => (d.target as GraphNode).x || 0)
        .attr("y2", (d) => (d.target as GraphNode).y || 0);
    }

    if (node) {
      node.attr("transform", (d) => `translate(${d.x || 0},${d.y || 0})`);
    }
  });
}

// ══════════════════════════════════════════════════════════════════
// DRAG BEHAVIOR
// ══════════════════════════════════════════════════════════════════

function drag(simulation: d3.Simulation<GraphNode, GraphLink>) {
  function dragstarted(
    event: d3.D3DragEvent<SVGGElement, GraphNode, GraphNode>
  ) {
    if (!event.active && simulation) simulation.alphaTarget(0.3).restart();
    event.subject.fx = event.subject.x;
    event.subject.fy = event.subject.y;
  }

  function dragged(event: d3.D3DragEvent<SVGGElement, GraphNode, GraphNode>) {
    event.subject.fx = event.x;
    event.subject.fy = event.y;
  }

  function dragended(event: d3.D3DragEvent<SVGGElement, GraphNode, GraphNode>) {
    if (!event.active && simulation) simulation.alphaTarget(0);
    event.subject.fx = null;
    event.subject.fy = null;
  }

  return d3
    .drag<SVGGElement, GraphNode>()
    .on("start", dragstarted)
    .on("drag", dragged)
    .on("end", dragended);
}

// ══════════════════════════════════════════════════════════════════
// TOOLTIP
// ══════════════════════════════════════════════════════════════════

function showTooltip(event: MouseEvent, d: GraphNode) {
  const kind = d.kind.charAt(0).toUpperCase() + d.kind.slice(1);
  let extra = "";
  if (d.confidence)
    extra = "<br>Confidence: " + Math.round(d.confidence * 100) + "%";
  if (d.status) extra += "<br>Status: " + d.status;
  if (d.links > 0) extra += "<br>Links: " + d.links;

  tooltip.innerHTML = "<h4>" + kind + "</h4><p>" + d.content + extra + "</p>";
  tooltip.style.display = "block";
  tooltip.style.left = event.pageX + 10 + "px";
  tooltip.style.top = event.pageY + 10 + "px";
}

// ══════════════════════════════════════════════════════════════════
// UTILITY FUNCTIONS
// ══════════════════════════════════════════════════════════════════

function truncate(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength - 3) + "...";
}

// ══════════════════════════════════════════════════════════════════
// RESIZE HANDLER
// ══════════════════════════════════════════════════════════════════

window.addEventListener("resize", () => {
  if (nodes.length > 0) {
    renderGraph();
  }
});
