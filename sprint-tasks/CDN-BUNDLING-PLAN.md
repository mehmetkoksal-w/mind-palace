# CDN to Local Bundling Plan: D3.js & Cytoscape

**Date Created:** January 6, 2026  
**Status:** PREPARATION PHASE - DO NOT IMPLEMENT YET  
**Priority:** HIGH - Security & Offline Functionality

---

## Executive Summary

The Mind Palace VS Code extension currently loads D3.js and Cytoscape from external CDNs, creating security risks and offline failures. This document outlines a comprehensive plan to bundle these libraries locally while maintaining VS Code extension best practices.

**Current Risk Level:** 🔴 HIGH

- External script execution bypasses CSP protections
- Extension fails offline
- Unpredictable CDN availability
- No version locking guarantees

---

## 1. Current State Analysis

### 1.1 CDN Dependencies Found

| File                                                 | Library   | Version     | CDN URL                                                                    |
| ---------------------------------------------------- | --------- | ----------- | -------------------------------------------------------------------------- |
| `src/sidebar.ts`                                     | Cytoscape | 3.28.1      | `https://cdnjs.cloudflare.com/ajax/libs/cytoscape/3.28.1/cytoscape.min.js` |
| `src/webviews/knowledgeGraph/knowledgeGraphPanel.ts` | D3.js     | v7 (latest) | `https://d3js.org/d3.v7.min.js`                                            |

**Exact code locations:**

**sidebar.ts (Line 1229):**

```typescript
<script src="https://cdnjs.cloudflare.com/ajax/libs/cytoscape/3.28.1/cytoscape.min.js"></script>
```

**knowledgeGraphPanel.ts (Line 426):**

```typescript
<script src="https://d3js.org/d3.v7.min.js" nonce="${nonce}"></script>
```

### 1.2 Content Security Policy Issues

**sidebar.ts (Line 309):**

```typescript
content =
  "default-src 'none'; style-src 'unsafe-inline'; script-src 'nonce-${nonce}' https://cdnjs.cloudflare.com; connect-src https://cdnjs.cloudflare.com;";
```

**knowledgeGraphPanel.ts (Line 254):**

```typescript
content =
  "default-src 'none'; script-src 'nonce-${nonce}' https://d3js.org; style-src 'unsafe-inline'; img-src data:;";
```

**Problem:** Both CSPs allow external script sources, violating VS Code security best practices.

### 1.3 Package.json Review

**Current dependencies:**

```json
"devDependencies": {
  "@types/cytoscape": "^3.21.9",  // ✅ Type definitions present
  "cytoscape": "^3.33.1",         // ⚠️ MISMATCH - types for 3.21, using 3.28 from CDN
}
```

**Missing:**

- `d3` package (not installed at all)
- `@types/d3` (no TypeScript types)

**Version Discrepancies:**

- Cytoscape types: 3.21.9
- Cytoscape CDN: 3.28.1
- Cytoscape package: 3.33.1 (never used!)
- D3.js CDN: v7 (latest from d3js.org)

**Key Finding:** Cytoscape is already installed as a dev dependency but never bundled!

---

## 2. Recommended Bundling Strategy

### 2.1 Build Tool Selection: **esbuild**

**Rationale:**

- ✅ Already VS Code's recommended bundler (2024+)
- ✅ 10-100x faster than webpack
- ✅ Built-in minification and tree-shaking
- ✅ Native TypeScript support
- ✅ Simple configuration for webview bundling
- ✅ Lower bundle sizes with proper splitting

**Alternative considered:**

- ❌ Webpack: Slower, complex config, legacy tool
- ❌ Rollup: Good but esbuild is faster and simpler
- ❌ Vite: Overkill for extension bundling

### 2.2 Architecture Overview

```
apps/vscode/
├── src/
│   ├── extension.ts              # Extension host (existing)
│   ├── sidebar.ts                # Generates webview HTML
│   ├── webviews/
│   │   ├── knowledgeGraph/
│   │   │   └── knowledgeGraphPanel.ts
│   │   └── webview-scripts/      # 🆕 NEW DIRECTORY
│   │       ├── blueprint.ts      # 🆕 Blueprint webview script
│   │       └── knowledge-graph.ts # 🆕 Knowledge graph webview script
├── out/
│   ├── extension.js              # Compiled extension
│   └── webviews/                 # 🆕 Bundled webview assets
│       ├── blueprint.js          # 🆕 Includes Cytoscape
│       └── knowledge-graph.js    # 🆕 Includes D3.js
└── package.json
```

**Key Principle:** Separate bundles for extension host and webviews to avoid bloating the extension.

---

## 3. Step-by-Step Implementation Guide

### Phase 1: Dependencies & Configuration

#### Step 1.1: Install Missing Dependencies

**Add to package.json:**

```json
{
  "devDependencies": {
    "@types/cytoscape": "^3.21.9", // Keep existing
    "@types/d3": "^7.4.3", // 🆕 ADD
    "cytoscape": "^3.33.1", // Keep (align with types later)
    "d3": "^7.9.0", // 🆕 ADD
    "esbuild": "^0.20.0", // 🆕 ADD
    "@types/vscode": "^1.80.0", // Keep existing
    "typescript": "^5.1.3" // Keep existing
  }
}
```

**Command to run:**

```powershell
npm install --save-dev d3@^7.9.0 @types/d3@^7.4.3 esbuild@^0.20.0
```

#### Step 1.2: Create esbuild Configuration

**Create `esbuild.config.js`:**

```javascript
const esbuild = require("esbuild");

const watch = process.argv.includes("--watch");

// Extension bundle (runs in Node.js)
const extensionConfig = {
  entryPoints: ["src/extension.ts"],
  bundle: true,
  outfile: "out/extension.js",
  external: ["vscode"],
  format: "cjs",
  platform: "node",
  sourcemap: true,
  minify: process.env.NODE_ENV === "production",
};

// Blueprint webview bundle (runs in browser)
const blueprintWebviewConfig = {
  entryPoints: ["src/webviews/webview-scripts/blueprint.ts"],
  bundle: true,
  outfile: "out/webviews/blueprint.js",
  format: "iife", // Immediately Invoked Function Expression
  platform: "browser",
  sourcemap: true,
  minify: process.env.NODE_ENV === "production",
  target: ["es2020"],
};

// Knowledge graph webview bundle (runs in browser)
const knowledgeGraphWebviewConfig = {
  entryPoints: ["src/webviews/webview-scripts/knowledge-graph.ts"],
  bundle: true,
  outfile: "out/webviews/knowledge-graph.js",
  format: "iife",
  platform: "browser",
  sourcemap: true,
  minify: process.env.NODE_ENV === "production",
  target: ["es2020"],
};

async function build() {
  const builders = [
    esbuild.build(extensionConfig),
    esbuild.build(blueprintWebviewConfig),
    esbuild.build(knowledgeGraphWebviewConfig),
  ];

  if (watch) {
    // Watch mode for development
    const contexts = await Promise.all([
      esbuild.context(extensionConfig),
      esbuild.context(blueprintWebviewConfig),
      esbuild.context(knowledgeGraphWebviewConfig),
    ]);
    await Promise.all(contexts.map((ctx) => ctx.watch()));
    console.log("Watching for changes...");
  } else {
    // Production build
    await Promise.all(builders);
    console.log("Build complete");
  }
}

build().catch(() => process.exit(1));
```

#### Step 1.3: Update package.json Scripts

**Replace existing scripts:**

```json
{
  "scripts": {
    "vscode:prepublish": "NODE_ENV=production node esbuild.config.js",
    "compile": "node esbuild.config.js",
    "watch": "node esbuild.config.js --watch",
    "test": "node ./out/test/runTests.js",
    "test:unit": "mocha --require ts-node/register 'src/test/unit/**/*.test.ts'",
    "test:coverage": "nyc npm run test:unit",
    "pretest": "npm run compile"
  }
}
```

**Note:** For Windows PowerShell, use `$env:NODE_ENV="production"` instead of `NODE_ENV=production` or use `cross-env` package.

---

### Phase 2: Extract Webview Scripts

#### Step 2.1: Create Blueprint Webview Script

**Create `src/webviews/webview-scripts/blueprint.ts`:**

```typescript
/**
 * Blueprint webview script
 * Bundled with Cytoscape to run in VS Code webview
 */

import cytoscape from "cytoscape";

// Declare vscode API (injected by VS Code)
declare const acquireVsCodeApi: () => any;

interface BlueprintData {
  type: string;
  data?: any;
  [key: string]: any;
}

// Initialize VS Code API
const vscode = acquireVsCodeApi();

// State management
let currentView: "list" | "map" = "list";
let cy: any = null;
let isSearchMode = false;
let currentSearchResults: any = null;
let originalGraphData: any = null;
let treeData: any[] = [];

// DOM references (will be initialized after DOM loads)
let contentArea: HTMLElement;
let treeView: HTMLElement;
let mapView: HTMLElement;
let graphContainer: HTMLElement;
let tooltipElement: HTMLElement;

/**
 * Initialize Cytoscape graph
 */
function initCytoscape(elements: any) {
  if (!graphContainer) return;

  cy = cytoscape({
    container: graphContainer,
    elements: elements,
    style: getCytoscapeStyle(),
    layout: {
      name: "cose",
      animate: false,
      idealEdgeLength: 100,
      nodeOverlap: 20,
      refresh: 20,
      fit: true,
      padding: 30,
      randomize: false,
      componentSpacing: 100,
      nodeRepulsion: 400000,
      edgeElasticity: 100,
      nestingFactor: 5,
      gravity: 80,
    },
  });

  // Add interaction handlers
  cy.on("tap", "node", function (evt: any) {
    const node = evt.target;
    const data = node.data();
    vscode.postMessage({
      type: "nodeClicked",
      data: data,
    });
  });

  // Tooltip on hover
  cy.on("mouseover", "node", function (evt: any) {
    const node = evt.target;
    showTooltip(evt.renderedPosition, node.data());
  });

  cy.on("mouseout", "node", function () {
    hideTooltip();
  });
}

/**
 * Get Cytoscape stylesheet
 */
function getCytoscapeStyle() {
  return [
    {
      selector: "node",
      style: {
        "background-color": "#4a5568",
        label: "data(label)",
        "font-size": "10px",
        "text-valign": "center",
        "text-halign": "center",
        color: "#e2e8f0",
        width: "label",
        height: "label",
        padding: "8px",
        shape: "roundrectangle",
      },
    },
    {
      selector: 'node[type="room"]',
      style: {
        "background-color": "#3b82f6",
        shape: "hexagon",
      },
    },
    {
      selector: 'node[type="file"]',
      style: {
        "background-color": "#10b981",
      },
    },
    {
      selector: "edge",
      style: {
        width: 1,
        "line-color": "#4a5568",
        "target-arrow-color": "#4a5568",
        "target-arrow-shape": "triangle",
        "curve-style": "bezier",
      },
    },
    {
      selector: ":selected",
      style: {
        "border-width": 2,
        "border-color": "#f59e0b",
      },
    },
  ];
}

/**
 * Show tooltip
 */
function showTooltip(position: any, data: any) {
  if (!tooltipElement) return;

  tooltipElement.style.display = "block";
  tooltipElement.style.left = position.x + 10 + "px";
  tooltipElement.style.top = position.y + 10 + "px";
  tooltipElement.innerHTML = `
    <strong>${data.label}</strong><br/>
    Type: ${data.type}
  `;
}

/**
 * Hide tooltip
 */
function hideTooltip() {
  if (tooltipElement) {
    tooltipElement.style.display = "none";
  }
}

/**
 * Handle messages from extension
 */
window.addEventListener("message", (event) => {
  const message: BlueprintData = event.data;

  switch (message.type) {
    case "updateData":
      treeData = message.data;
      renderCurrentView();
      break;
    case "switchView":
      currentView = message.view;
      renderCurrentView();
      break;
    // Add other message handlers as needed
  }
});

/**
 * Render the current view
 */
function renderCurrentView() {
  if (currentView === "list") {
    renderTreeView();
  } else {
    renderMapView();
  }
}

/**
 * Render tree view (implementation to be extracted from sidebar.ts)
 */
function renderTreeView() {
  // TODO: Extract tree rendering logic from sidebar.ts
}

/**
 * Render map view with Cytoscape
 */
function renderMapView() {
  // TODO: Extract map rendering logic from sidebar.ts
}

/**
 * Initialize on DOM load
 */
document.addEventListener("DOMContentLoaded", () => {
  contentArea = document.getElementById("content-area")!;
  treeView = document.getElementById("tree-view")!;
  mapView = document.getElementById("map-view")!;
  graphContainer = document.getElementById("cy")!;
  tooltipElement = document.getElementById("tooltip")!;

  // Request initial data
  vscode.postMessage({ type: "ready" });
});
```

**Note:** This is a starter template. The full implementation will require extracting all Cytoscape-related logic from `sidebar.ts`.

#### Step 2.2: Create Knowledge Graph Webview Script

**Create `src/webviews/webview-scripts/knowledge-graph.ts`:**

```typescript
/**
 * Knowledge Graph webview script
 * Bundled with D3.js to run in VS Code webview
 */

import * as d3 from "d3";

// Declare vscode API
declare const acquireVsCodeApi: () => any;

interface GraphNode {
  id: string;
  kind: "idea" | "decision" | "learning";
  content: string;
  confidence?: number;
  status?: string;
  links: number;
}

interface GraphLink {
  source: string | GraphNode;
  target: string | GraphNode;
  relation: string;
}

// Initialize VS Code API
const vscode = acquireVsCodeApi();

// State
let nodes: GraphNode[] = [];
let links: GraphLink[] = [];
let simulation: any;
let svg: any;
let g: any;
let link: any;
let node: any;
let zoom: any;

// DOM elements
const container = document.getElementById("graph")!;
const tooltip = document.getElementById("tooltip")!;

/**
 * Handle messages from extension
 */
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

/**
 * Render the D3 force-directed graph
 */
function renderGraph() {
  // Clear previous graph
  container.innerHTML = "";

  if (nodes.length === 0) {
    container.innerHTML = '<div class="loading">No data to display</div>';
    return;
  }

  // Set up SVG
  const width = container.clientWidth;
  const height = container.clientHeight;

  svg = d3
    .select(container)
    .append("svg")
    .attr("width", width)
    .attr("height", height);

  // Add zoom behavior
  zoom = d3
    .zoom()
    .scaleExtent([0.1, 4])
    .on("zoom", (event) => {
      g.attr("transform", event.transform);
    });

  svg.call(zoom as any);

  g = svg.append("g");

  // Create force simulation
  simulation = d3
    .forceSimulation(nodes as any)
    .force(
      "link",
      d3
        .forceLink(links)
        .id((d: any) => d.id)
        .distance(100)
    )
    .force("charge", d3.forceManyBody().strength(-300))
    .force("center", d3.forceCenter(width / 2, height / 2))
    .force("collision", d3.forceCollide().radius(30));

  // Draw links
  link = g
    .append("g")
    .selectAll("line")
    .data(links)
    .enter()
    .append("line")
    .attr("class", (d: GraphLink) => `link ${d.relation}`)
    .style("stroke", getLinkColor)
    .style("stroke-width", 1.5);

  // Draw nodes
  node = g
    .append("g")
    .selectAll("circle")
    .data(nodes)
    .enter()
    .append("circle")
    .attr("class", (d: GraphNode) => `node ${d.kind}`)
    .attr("r", (d: GraphNode) => Math.min(8 + d.links * 2, 20))
    .style("fill", getNodeColor)
    .call(
      d3
        .drag<any, any>()
        .on("start", dragStarted)
        .on("drag", dragged)
        .on("end", dragEnded) as any
    )
    .on("mouseover", showTooltip)
    .on("mouseout", hideTooltip)
    .on("click", handleNodeClick);

  // Update positions on each tick
  simulation.on("tick", () => {
    link
      .attr("x1", (d: any) => d.source.x)
      .attr("y1", (d: any) => d.source.y)
      .attr("x2", (d: any) => d.target.x)
      .attr("y2", (d: any) => d.target.y);

    node.attr("cx", (d: any) => d.x).attr("cy", (d: any) => d.y);
  });
}

/**
 * Get node color based on kind
 */
function getNodeColor(d: GraphNode): string {
  switch (d.kind) {
    case "idea":
      return "#3b82f6"; // blue
    case "decision":
      return "#10b981"; // green
    case "learning":
      return "#8b5cf6"; // purple
    default:
      return "#6b7280"; // gray
  }
}

/**
 * Get link color based on relation
 */
function getLinkColor(d: GraphLink): string {
  switch (d.relation) {
    case "supports":
      return "#10b981";
    case "contradicts":
      return "#ef4444";
    case "implements":
      return "#3b82f6";
    default:
      return "#8b5cf6";
  }
}

/**
 * Drag handlers
 */
function dragStarted(event: any) {
  if (!event.active) simulation.alphaTarget(0.3).restart();
  event.subject.fx = event.subject.x;
  event.subject.fy = event.subject.y;
}

function dragged(event: any) {
  event.subject.fx = event.x;
  event.subject.fy = event.y;
}

function dragEnded(event: any) {
  if (!event.active) simulation.alphaTarget(0);
  event.subject.fx = null;
  event.subject.fy = null;
}

/**
 * Show tooltip
 */
function showTooltip(event: any, d: GraphNode) {
  tooltip.style.display = "block";
  tooltip.style.left = event.pageX + 10 + "px";
  tooltip.style.top = event.pageY + 10 + "px";
  tooltip.innerHTML = `
    <h4>${d.kind.toUpperCase()}</h4>
    <p>${d.content.substring(0, 100)}${d.content.length > 100 ? "..." : ""}</p>
    <p style="margin-top: 4px; font-size: 11px;">Links: ${d.links}</p>
  `;
}

/**
 * Hide tooltip
 */
function hideTooltip() {
  tooltip.style.display = "none";
}

/**
 * Handle node click
 */
function handleNodeClick(event: any, d: GraphNode) {
  vscode.postMessage({
    type: "showDetail",
    node: d,
  });
}

/**
 * Button handlers
 */
document.getElementById("refresh")?.addEventListener("click", () => {
  vscode.postMessage({ type: "refresh" });
});

document.getElementById("zoomIn")?.addEventListener("click", () => {
  svg.transition().call(zoom.scaleBy, 1.3);
});

document.getElementById("zoomOut")?.addEventListener("click", () => {
  svg.transition().call(zoom.scaleBy, 0.7);
});

document.getElementById("reset")?.addEventListener("click", () => {
  svg
    .transition()
    .call(
      zoom.transform,
      d3.zoomIdentity.translate(
        container.clientWidth / 2,
        container.clientHeight / 2
      )
    );
  simulation.alpha(1).restart();
});
```

---

### Phase 3: Update Webview HTML Generation

#### Step 3.1: Update sidebar.ts to Use Bundled Scripts

**Current code (line ~1229):**

```typescript
<script src="https://cdnjs.cloudflare.com/ajax/libs/cytoscape/3.28.1/cytoscape.min.js"></script>
<script nonce="${nonce}">
  // Inline script code...
</script>
```

**Replace with:**

```typescript
private _getHtmlForWebview(webview: vscode.Webview): string {
  const nonce = this._getNonce();

  // Get bundled script URI
  const scriptUri = webview.asWebviewUri(
    vscode.Uri.joinPath(this._extensionUri, 'out', 'webviews', 'blueprint.js')
  );

  return /*html*/ `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src ${webview.cspSource} 'unsafe-inline'; script-src 'nonce-${nonce}';">
    <title>Mind Palace Blueprint</title>
    <style>
        /* Keep existing styles */
    </style>
</head>
<body>
    <!-- Keep existing HTML structure -->

    <!-- Replace CDN script with bundled script -->
    <script nonce="${nonce}" src="${scriptUri}"></script>
</body>
</html>`;
}
```

**Key changes:**

1. Use `webview.asWebviewUri()` to get proper VS Code webview URI
2. Remove CDN URL completely
3. Update CSP to remove `https://cdnjs.cloudflare.com`
4. Remove inline script (moved to bundle)

#### Step 3.2: Update knowledgeGraphPanel.ts

**Current code (line ~426):**

```typescript
<script src="https://d3js.org/d3.v7.min.js" nonce="${nonce}"></script>
<script nonce="${nonce}">
  // Inline script code...
</script>
```

**Replace with:**

```typescript
private getHtmlContent(): string {
  const nonce = getNonce();

  // Get bundled script URI
  const scriptUri = this.panel.webview.asWebviewUri(
    vscode.Uri.joinPath(this.extensionUri, 'out', 'webviews', 'knowledge-graph.js')
  );

  return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'nonce-${nonce}'; style-src 'unsafe-inline'; img-src data:;">
    <title>Knowledge Graph</title>
    <style>
        /* Keep existing styles */
    </style>
</head>
<body>
    <!-- Keep existing HTML structure -->

    <!-- Replace CDN script with bundled script -->
    <script nonce="${nonce}" src="${scriptUri}"></script>
</body>
</html>`;
}
```

**Key changes:**

1. Use `this.panel.webview.asWebviewUri()` for proper URI
2. Remove `https://d3js.org` from CSP
3. Remove inline D3 script (moved to bundle)

---

### Phase 4: Content Security Policy Updates

#### Updated CSP for Blueprint (sidebar.ts)

**Before:**

```typescript
content =
  "default-src 'none'; style-src 'unsafe-inline'; script-src 'nonce-${nonce}' https://cdnjs.cloudflare.com; connect-src https://cdnjs.cloudflare.com;";
```

**After:**

```typescript
content =
  "default-src 'none'; style-src ${webview.cspSource} 'unsafe-inline'; script-src 'nonce-${nonce}';";
```

**Improvements:**

- ✅ No external script sources
- ✅ Nonce-based script execution only
- ✅ Compliant with VS Code security guidelines

#### Updated CSP for Knowledge Graph

**Before:**

```typescript
content =
  "default-src 'none'; script-src 'nonce-${nonce}' https://d3js.org; style-src 'unsafe-inline'; img-src data:;";
```

**After:**

```typescript
content =
  "default-src 'none'; script-src 'nonce-${nonce}'; style-src 'unsafe-inline'; img-src data:;";
```

**Improvements:**

- ✅ No external D3.js domain
- ✅ All scripts nonce-validated
- ✅ Offline-compatible

---

### Phase 5: Build System Integration

#### Update .vscodeignore

**Create/Update `.vscodeignore`:**

```
.vscode/**
.vscode-test/**
src/**
tsconfig.json
esbuild.config.js
node_modules/**
out/test/**
.gitignore
.eslintrc.json
**/*.map
*.vsix
```

**Key points:**

- Include `out/` directory (compiled code)
- Exclude `src/` (source TypeScript)
- Exclude `node_modules` (dependencies bundled into out/)

#### Update TypeScript Configuration

**tsconfig.json (no changes needed, but verify):**

```jsonc
{
  "compilerOptions": {
    "module": "commonjs",
    "target": "es2020",
    "outDir": "out", // esbuild overrides this
    "lib": ["es2020", "dom"],
    "sourceMap": true,
    "rootDir": "src",
    "strict": true
  },
  "exclude": ["node_modules", ".vscode-test", "out"]
}
```

**Note:** esbuild ignores this but TypeScript language service still uses it.

---

## 4. Bundle Size Analysis

### Expected Bundle Sizes

| Bundle               | Libraries        | Estimated Size (minified) | Gzipped |
| -------------------- | ---------------- | ------------------------- | ------- |
| `extension.js`       | VS Code API only | ~50 KB                    | ~15 KB  |
| `blueprint.js`       | Cytoscape 3.33.1 | ~600 KB                   | ~150 KB |
| `knowledge-graph.js` | D3.js v7         | ~250 KB                   | ~75 KB  |
| **Total**            |                  | ~900 KB                   | ~240 KB |

**Current (CDN):** 0 KB bundled, ~850 KB network download  
**After Bundling:** ~900 KB bundled, 0 KB network

**Impact:**

- ✅ Extension package size: +900 KB (~1 MB increase in .vsix)
- ✅ Offline functionality: Full support
- ✅ Load time: Faster (local filesystem vs network)
- ✅ Security: Improved (no external scripts)

### Optimization Opportunities

1. **Tree-shaking:** D3.js is modular; only import needed modules
2. **Code splitting:** Load webview bundles only when views open
3. **Lazy loading:** Defer Cytoscape init until map view is activated

**Optimized D3 imports (for knowledge-graph.ts):**

```typescript
// Instead of:
import * as d3 from "d3";

// Use specific imports:
import { select } from "d3-selection";
import {
  forceSimulation,
  forceLink,
  forceManyBody,
  forceCenter,
  forceCollide,
} from "d3-force";
import { zoom } from "d3-zoom";
import { drag } from "d3-drag";

// Estimated savings: ~100 KB
```

---

## 5. Testing Strategy

### 5.1 Unit Tests

**Test file:** `src/test/unit/webview-bundling.test.ts`

```typescript
import * as assert from "assert";
import * as vscode from "vscode";
import * as path from "path";
import * as fs from "fs";

suite("Webview Bundling Tests", () => {
  test("Bundled scripts exist in out/ directory", () => {
    const extensionPath = path.resolve(__dirname, "../../..");
    const blueprintPath = path.join(
      extensionPath,
      "out",
      "webviews",
      "blueprint.js"
    );
    const knowledgeGraphPath = path.join(
      extensionPath,
      "out",
      "webviews",
      "knowledge-graph.js"
    );

    assert.ok(fs.existsSync(blueprintPath), "blueprint.js should exist");
    assert.ok(
      fs.existsSync(knowledgeGraphPath),
      "knowledge-graph.js should exist"
    );
  });

  test("Bundled scripts contain expected libraries", () => {
    const blueprintPath = path.resolve(
      __dirname,
      "../../../out/webviews/blueprint.js"
    );
    const content = fs.readFileSync(blueprintPath, "utf8");

    // Check for Cytoscape signatures
    assert.ok(content.includes("cytoscape"), "Should bundle Cytoscape");
  });

  test("Webview URIs are generated correctly", () => {
    const extensionUri = vscode.Uri.file("/fake/path");
    const webview = {
      asWebviewUri: (uri: vscode.Uri) => {
        // Mock implementation
        return vscode.Uri.parse(`vscode-webview://fake/${uri.fsPath}`);
      },
    } as any;

    const scriptUri = webview.asWebviewUri(
      vscode.Uri.joinPath(extensionUri, "out", "webviews", "blueprint.js")
    );

    assert.ok(scriptUri.toString().includes("blueprint.js"));
  });
});
```

### 5.2 Integration Tests

**Test scenarios:**

1. **Extension Activation**

   - Verify extension activates without errors
   - Check that bundled files are accessible

2. **Blueprint View**

   - Open blueprint sidebar
   - Verify Cytoscape graph renders
   - Test map/list view switching
   - Validate tooltip interactions

3. **Knowledge Graph View**

   - Trigger `mindPalace.showKnowledgeGraph`
   - Verify D3 graph renders
   - Test zoom/pan controls
   - Validate force simulation

4. **Offline Mode**
   - Disable network
   - Restart VS Code
   - Verify both webviews load successfully

### 5.3 Manual Testing Checklist

- [ ] Build extension: `npm run compile`
- [ ] Package extension: `vsce package`
- [ ] Install .vsix in VS Code
- [ ] Open Mind Palace project
- [ ] Test blueprint sidebar (list view)
- [ ] Test blueprint sidebar (map view)
- [ ] Test knowledge graph panel
- [ ] Test all Cytoscape interactions (zoom, pan, click)
- [ ] Test all D3 interactions (drag nodes, zoom, reset)
- [ ] Check DevTools console for errors
- [ ] Verify CSP violations (should be none)
- [ ] Test offline (disconnect network, reload VS Code)

### 5.4 Performance Testing

**Metrics to measure:**

1. **Extension activation time**

   - Before: Measure current activation
   - After: Should be similar (bundles load lazily)

2. **Webview load time**

   - Blueprint sidebar: Measure time to first render
   - Knowledge graph: Measure time to first render
   - Target: <500ms for both

3. **Bundle load time**
   - Measure `blueprint.js` load time
   - Measure `knowledge-graph.js` load time
   - Compare to CDN load times

**Testing tool:**

```typescript
// Add to webview script
const loadStart = performance.now();
window.addEventListener("DOMContentLoaded", () => {
  const loadTime = performance.now() - loadStart;
  console.log(`Webview loaded in ${loadTime}ms`);
});
```

---

## 6. Rollback Plan

### 6.1 Git Strategy

**Before starting implementation:**

```powershell
git checkout -b feature/bundle-cdn-libraries
git commit -m "Checkpoint: Before CDN bundling migration"
```

**After each phase:**

```powershell
git add .
git commit -m "Phase X: [description]"
git tag bundling-phase-X
```

### 6.2 Rollback Triggers

**Abort migration if:**

- Bundle size > 2 MB
- Extension activation time > 3 seconds
- Webview load time > 2 seconds
- Tests fail consistently
- Breaking changes to existing functionality

### 6.3 Rollback Commands

**Full rollback:**

```powershell
git reset --hard bundling-phase-0
git clean -fd
npm install
```

**Partial rollback (keep config, revert code):**

```powershell
git checkout main -- src/
npm run compile
```

### 6.4 Emergency Hotfix

**If bundling breaks production:**

1. Revert package.json scripts to TypeScript compilation
2. Restore CDN script tags in HTML
3. Restore original CSP policies
4. Publish emergency patch version

**Quick revert patch:**

```typescript
// sidebar.ts
<script src="https://cdnjs.cloudflare.com/ajax/libs/cytoscape/3.28.1/cytoscape.min.js"></script>

// knowledgeGraphPanel.ts
<script src="https://d3js.org/d3.v7.min.js" nonce="${nonce}"></script>
```

---

## 7. Post-Implementation Tasks

### 7.1 Documentation Updates

**Files to update:**

- `README.md` - Add bundling information
- `CHANGELOG.md` - Document security improvements
- `package.json` - Update description to mention offline support

### 7.2 Version Bump

**Semantic versioning:**

- Current: `0.0.2-alpha`
- After bundling: `0.0.3-alpha` (patch)
- Or: `0.1.0-alpha` (minor - new feature: offline support)

### 7.3 CI/CD Updates

**If CI pipeline exists:**

- Update build steps to use esbuild
- Add bundle size checks
- Add webview asset validation

### 7.4 Future Optimizations

1. **Upgrade to Cytoscape 3.33.1**

   - Currently using 3.28.1 from CDN
   - Package.json has 3.33.1 installed
   - Align versions after bundling

2. **D3 Modular Imports**

   - Switch from full D3 bundle to specific modules
   - Estimated savings: ~100 KB

3. **Cytoscape Extensions**

   - If using extensions (layouts, etc.), bundle them too
   - Examples: cytoscape-cose-bilkent, cytoscape-dagre

4. **Source Maps**
   - Enable in production for debugging
   - Configure proper paths for VS Code debugging

---

## 8. Timeline & Milestones

### Recommended Implementation Schedule

**Week 1: Setup & Configuration**

- [ ] Install dependencies (d3, @types/d3, esbuild)
- [ ] Create esbuild.config.js
- [ ] Update package.json scripts
- [ ] Test basic builds

**Week 2: Extract Webview Scripts**

- [ ] Create `src/webviews/webview-scripts/` directory
- [ ] Extract blueprint.ts from sidebar.ts inline code
- [ ] Extract knowledge-graph.ts from knowledgeGraphPanel.ts
- [ ] Implement message handlers

**Week 3: Update HTML Generation**

- [ ] Modify sidebar.ts to use bundled script
- [ ] Modify knowledgeGraphPanel.ts to use bundled script
- [ ] Update CSP policies
- [ ] Test webview.asWebviewUri() paths

**Week 4: Testing & Validation**

- [ ] Write unit tests
- [ ] Run integration tests
- [ ] Manual testing checklist
- [ ] Performance benchmarks
- [ ] Fix bugs and issues

**Week 5: Documentation & Release**

- [ ] Update documentation
- [ ] Create changelog entries
- [ ] Package extension (.vsix)
- [ ] Internal testing
- [ ] Tag release

---

## 9. Risk Assessment & Mitigation

### Risk Matrix

| Risk                            | Probability | Impact | Mitigation                           |
| ------------------------------- | ----------- | ------ | ------------------------------------ |
| Bundle size too large           | Low         | Medium | Use tree-shaking, modular imports    |
| Breaking existing functionality | Medium      | High   | Comprehensive testing, rollback plan |
| Performance degradation         | Low         | Medium | Benchmark before/after, optimize     |
| CSP violations                  | Low         | High   | Test thoroughly, use CSP validator   |
| Build complexity                | Medium      | Low    | Document build process, automate     |
| Type definition conflicts       | Medium      | Low    | Align package versions               |

### Critical Success Factors

1. ✅ **No functionality loss** - All existing features work identically
2. ✅ **Security improvement** - CSP compliant, no external scripts
3. ✅ **Offline support** - Extension works without network
4. ✅ **Performance parity** - Load times similar or better than CDN
5. ✅ **Maintainability** - Build process is simple and documented

---

## 10. Code Examples & Snippets

### Helper Function: Generate Nonce

```typescript
/**
 * Generate a cryptographically secure nonce for CSP
 */
function getNonce(): string {
  let text = "";
  const possible =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  for (let i = 0; i < 32; i++) {
    text += possible.charAt(Math.floor(Math.random() * possible.length));
  }
  return text;
}
```

### Webview Resource URI Helper

```typescript
/**
 * Get webview URI for a bundled resource
 */
function getWebviewUri(
  webview: vscode.Webview,
  extensionUri: vscode.Uri,
  ...pathSegments: string[]
): vscode.Uri {
  return webview.asWebviewUri(
    vscode.Uri.joinPath(extensionUri, ...pathSegments)
  );
}

// Usage:
const scriptUri = getWebviewUri(
  this.panel.webview,
  this.extensionUri,
  "out",
  "webviews",
  "knowledge-graph.js"
);
```

### CSP Validator

```typescript
/**
 * Validate Content Security Policy compliance
 */
function validateCSP(csp: string): boolean {
  const warnings: string[] = [];

  if (csp.includes("unsafe-eval")) {
    warnings.push("CSP contains unsafe-eval");
  }

  if (csp.includes("http://") || csp.includes("https://")) {
    warnings.push("CSP allows external resources");
  }

  if (!csp.includes("script-src 'nonce-")) {
    warnings.push("CSP missing nonce-based script-src");
  }

  if (warnings.length > 0) {
    console.warn("CSP validation warnings:", warnings);
    return false;
  }

  return true;
}
```

---

## 11. References & Resources

### VS Code Extension Documentation

- [Webview API](https://code.visualstudio.com/api/extension-guides/webview)
- [Webview Security](https://code.visualstudio.com/api/extension-guides/webview#security)
- [Content Security Policy](https://code.visualstudio.com/api/extension-guides/webview#content-security-policy)
- [Bundling Extensions](https://code.visualstudio.com/api/working-with-extensions/bundling-extension)

### Library Documentation

- [D3.js v7 Documentation](https://d3js.org/)
- [Cytoscape.js Documentation](https://js.cytoscape.org/)
- [esbuild Documentation](https://esbuild.github.io/)

### Security Best Practices

- [OWASP CSP Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Content_Security_Policy_Cheat_Sheet.html)
- [VS Code Extension Security](https://code.visualstudio.com/api/references/extension-manifest#extension-security)

### Example Extensions Using Bundled Libraries

- [vscode-drawio](https://github.com/hediet/vscode-drawio) - Uses bundled libraries
- [vscode-markdown-mermaid](https://github.com/mjbvz/vscode-markdown-mermaid) - Bundles Mermaid.js

---

## 12. Implementation Checklist

### Pre-Implementation

- [ ] Review this document thoroughly
- [ ] Create feature branch
- [ ] Backup current working state
- [ ] Run existing tests (baseline)

### Phase 1: Dependencies

- [ ] Install d3@^7.9.0
- [ ] Install @types/d3@^7.4.3
- [ ] Install esbuild@^0.20.0
- [ ] Verify package.json updates
- [ ] Run `npm install`

### Phase 2: Build Configuration

- [ ] Create esbuild.config.js
- [ ] Update package.json scripts
- [ ] Test `npm run compile`
- [ ] Test `npm run watch`
- [ ] Verify out/ directory structure

### Phase 3: Webview Scripts

- [ ] Create src/webviews/webview-scripts/ directory
- [ ] Create blueprint.ts
- [ ] Create knowledge-graph.ts
- [ ] Extract Cytoscape logic from sidebar.ts
- [ ] Extract D3 logic from knowledgeGraphPanel.ts

### Phase 4: HTML Updates

- [ ] Update sidebar.ts script tags
- [ ] Update sidebar.ts CSP
- [ ] Update knowledgeGraphPanel.ts script tags
- [ ] Update knowledgeGraphPanel.ts CSP
- [ ] Add webview.asWebviewUri() calls

### Phase 5: Testing

- [ ] Write unit tests
- [ ] Run `npm test`
- [ ] Manual testing - blueprint list view
- [ ] Manual testing - blueprint map view
- [ ] Manual testing - knowledge graph
- [ ] Test offline mode
- [ ] Check for CSP violations
- [ ] Performance benchmarks

### Phase 6: Documentation

- [ ] Update README.md
- [ ] Update CHANGELOG.md
- [ ] Add inline code comments
- [ ] Document build process

### Phase 7: Release

- [ ] Bump version number
- [ ] Create git tag
- [ ] Build production bundle
- [ ] Package .vsix
- [ ] Test installed extension
- [ ] Merge to main branch

---

## 13. Success Criteria

**The migration is successful when:**

1. ✅ Extension builds without errors
2. ✅ All webviews load and render correctly
3. ✅ No CDN URLs in source code
4. ✅ CSP policies compliant (no external script sources)
5. ✅ Extension works offline
6. ✅ Bundle size < 2 MB
7. ✅ Load times similar or better than CDN
8. ✅ All existing features functional
9. ✅ Tests pass
10. ✅ Documentation complete

---

## 14. Next Steps

**After this plan is approved:**

1. **Schedule implementation** - Allocate 2-3 weeks
2. **Assign developers** - 1-2 developers recommended
3. **Set up tracking** - Create GitHub issues/tasks
4. **Schedule reviews** - Code review after each phase
5. **Plan testing** - QA testing before release

**Questions to resolve:**

1. Should we optimize D3 imports immediately or in Phase 2?
2. Should we upgrade Cytoscape to 3.33.1 now or separately?
3. Do we need CI/CD pipeline changes?
4. Should we add bundle size limits to CI?

---

## Appendix A: File Structure After Implementation

```
apps/vscode/
├── .gitignore
├── .vscodeignore          # ✏️ Updated
├── esbuild.config.js      # 🆕 NEW
├── Makefile
├── package.json           # ✏️ Updated (deps & scripts)
├── package-lock.json      # ✏️ Updated
├── README.md              # ✏️ Updated
├── CHANGELOG.md           # ✏️ Updated
├── tsconfig.json
├── images/
├── out/                   # Build output
│   ├── extension.js       # Extension host
│   ├── extension.js.map
│   └── webviews/          # 🆕 NEW
│       ├── blueprint.js
│       ├── blueprint.js.map
│       ├── knowledge-graph.js
│       └── knowledge-graph.js.map
└── src/
    ├── extension.ts
    ├── sidebar.ts         # ✏️ Modified (updated HTML)
    ├── bridge.ts
    ├── config.ts
    ├── types.ts
    ├── commands/
    ├── providers/
    ├── test/
    │   └── unit/
    │       └── webview-bundling.test.ts  # 🆕 NEW
    └── webviews/
        ├── knowledgeGraph/
        │   └── knowledgeGraphPanel.ts     # ✏️ Modified (updated HTML)
        └── webview-scripts/               # 🆕 NEW DIRECTORY
            ├── blueprint.ts               # 🆕 NEW
            └── knowledge-graph.ts         # 🆕 NEW
```

**Legend:**

- 🆕 NEW - File/directory to create
- ✏️ Modified - File to update
- No icon - Unchanged

---

## Appendix B: Dependency Version Lock

**Recommended versions (as of Jan 2026):**

```json
{
  "devDependencies": {
    "@types/cytoscape": "^3.21.9",
    "@types/d3": "^7.4.3",
    "@types/vscode": "^1.80.0",
    "cytoscape": "^3.33.1",
    "d3": "^7.9.0",
    "esbuild": "^0.20.0",
    "typescript": "^5.1.3"
  }
}
```

**Important:** Pin versions for production builds to avoid breaking changes.

---

## Appendix C: Build Performance Metrics

**Target benchmarks:**

| Metric               | Target | Acceptable | Unacceptable |
| -------------------- | ------ | ---------- | ------------ |
| Initial build time   | <10s   | <15s       | >20s         |
| Incremental build    | <2s    | <5s        | >10s         |
| Watch mode rebuild   | <1s    | <3s        | >5s          |
| Extension activation | <500ms | <1s        | >2s          |
| Blueprint load       | <300ms | <500ms     | >1s          |
| Knowledge graph load | <300ms | <500ms     | >1s          |

**Measurement command:**

```powershell
Measure-Command { npm run compile }
```

---

**END OF BUNDLING PLAN**

---

**Status:** ✅ READY FOR REVIEW  
**Next Action:** Present to team, get approval, schedule implementation  
**Estimated Implementation Time:** 2-3 weeks  
**Risk Level:** LOW (comprehensive rollback plan in place)
