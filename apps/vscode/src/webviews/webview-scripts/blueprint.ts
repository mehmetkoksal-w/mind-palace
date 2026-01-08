/**
 * Blueprint webview script
 * Bundled with Cytoscape to run in VS Code webview
 */

import cytoscape from "cytoscape";

// Declare vscode API (injected by VS Code)
declare const acquireVsCodeApi: () => any;

// Initialize VS Code API
const vscode = acquireVsCodeApi();

// ══════════════════════════════════════════════════════════════════
// STATE
// ══════════════════════════════════════════════════════════════════

let currentView: "list" | "map" = "list";
let cy: any = null;
let isSearchMode = false;
let currentSearchResults: any = null;
let originalGraphData: any = null;
let treeData: any[] = [];

// ══════════════════════════════════════════════════════════════════
// DOM ELEMENTS
// ══════════════════════════════════════════════════════════════════

const contentArea = document.getElementById("content-area")!;
const treeView = document.getElementById("tree-view")!;
const mapView = document.getElementById("map-view")!;
const cyContainer = document.getElementById("cy")!;
const canvasWrapper = document.getElementById("canvas-wrapper")!;
const emptyState = document.getElementById("empty-state")!;
const noResults = document.getElementById("no-results")!;
const noResultsText = document.getElementById("no-results-text")!;
const loadingOverlay = document.getElementById("loading")!;
const tooltip = document.getElementById("tooltip")!;
const tooltipTitle = document.getElementById("tooltip-title")!;
const tooltipDescription = document.getElementById("tooltip-description")!;
const tooltipPath = document.getElementById("tooltip-path")!;
const tooltipSnippet = document.getElementById("tooltip-snippet")!;
const tooltipSnippetContent = document.getElementById(
  "tooltip-snippet-content"
)!;
const btnRefresh = document.getElementById("btn-refresh")!;
const btnFit = document.getElementById("btn-fit")!;
const btnExpandAll = document.getElementById("btn-expand-all")!;
const btnCollapseAll = document.getElementById("btn-collapse-all")!;
const btnListView = document.getElementById("btn-list-view")!;
const btnMapView = document.getElementById("btn-map-view")!;
const searchInput = document.getElementById("search-input") as HTMLInputElement;
const searchClear = document.getElementById("search-clear")!;
const searchStatus = document.getElementById("search-status")!;
const matchCount = document.getElementById("match-count")!;
const searchStatusText = document.getElementById("search-status-text")!;
const indicator = document.getElementById("indicator")!;

// ══════════════════════════════════════════════════════════════════
// VIEW SWITCHING
// ══════════════════════════════════════════════════════════════════

function switchView(view: "list" | "map") {
  currentView = view;

  if (view === "list") {
    treeView.classList.add("active");
    mapView.classList.remove("active");
    btnListView.classList.add("active");
    btnMapView.classList.remove("active");
    btnExpandAll.style.display = "";
    btnCollapseAll.style.display = "";
    btnFit.style.display = "none";
  } else {
    treeView.classList.remove("active");
    mapView.classList.add("active");
    btnListView.classList.remove("active");
    btnMapView.classList.add("active");
    btnExpandAll.style.display = "none";
    btnCollapseAll.style.display = "none";
    btnFit.style.display = "";

    // Initialize Cytoscape if needed
    if (originalGraphData && !cy) {
      initCytoscape({
        nodes: originalGraphData.nodes,
        edges: originalGraphData.edges || [],
      });
    }

    // Apply search state to map if in search mode
    if (isSearchMode && currentSearchResults) {
      applySearchResultsToMap(currentSearchResults);
    }
  }
}

// ══════════════════════════════════════════════════════════════════
// TREE VIEW RENDERING
// ══════════════════════════════════════════════════════════════════

function buildTreeData(graphData: any) {
  const rooms: any[] = [];
  const roomMap = new Map();

  // First pass: create rooms
  for (const node of graphData.nodes) {
    if (node.data.type === "room") {
      const room = {
        id: node.data.id,
        name: node.data.label,
        description: node.data.description,
        files: [],
        expanded: true,
        isMatch: false,
      };
      rooms.push(room);
      roomMap.set(node.data.id, room);
    }
  }

  // Second pass: add files to rooms
  for (const node of graphData.nodes) {
    if (node.data.type === "file" && node.data.parent) {
      const room = roomMap.get(node.data.parent);
      if (room) {
        room.files.push({
          id: node.data.id,
          name: node.data.label,
          fullPath: node.data.fullPath,
          snippet: node.data.snippet,
          lineNumber: node.data.lineNumber,
          isMatch: false,
        });
      }
    }
  }

  // Sort rooms alphabetically
  rooms.sort((a, b) => a.name.localeCompare(b.name));

  return rooms;
}

function renderTreeView(searchResults: any = null) {
  treeView.innerHTML = "";

  if (treeData.length === 0) {
    return;
  }

  // Build matching sets for search mode
  const matchingRooms = new Set();
  const matchingFiles = new Map();

  if (searchResults && searchResults.results) {
    for (const roomResult of searchResults.results) {
      matchingRooms.add(roomResult.roomName);
      for (const match of roomResult.matches) {
        matchingFiles.set(match.filePath, match);
      }
    }
  }

  for (const room of treeData) {
    const roomEl = document.createElement("div");
    roomEl.className = "tree-room" + (room.expanded ? " expanded" : "");
    roomEl.dataset.roomId = room.id;

    // Check if room matches search
    const roomMatches =
      searchResults &&
      (matchingRooms.has(room.name) ||
        matchingRooms.has(room.id.replace("room-", "")));
    const hasMatchingFiles = room.files.some((f: any) =>
      matchingFiles.has(f.fullPath)
    );
    const roomIsRelevant = roomMatches || hasMatchingFiles;

    // Room Header
    const headerEl = document.createElement("div");
    headerEl.className = "tree-room-header";
    if (searchResults) {
      headerEl.classList.add(roomIsRelevant ? "search-match" : "ghost-mode");
    }

    // Count matching files
    const matchingFileCount = room.files.filter((f: any) =>
      matchingFiles.has(f.fullPath)
    ).length;

    headerEl.innerHTML = `
            <div class="tree-chevron">
                <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                    <path d="M6 4l4 4-4 4V4z"/>
                </svg>
            </div>
            <div class="tree-room-icon">
                <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                    <path d="M14.5 3H7.71l-.85-.85L6.51 2h-5l-.5.5v11l.5.5h13l.5-.5v-10L14.5 3zm-.51 8.49V13h-12V3h4.29l.85.85.36.15H14v7.49z"/>
                </svg>
            </div>
            <span class="tree-room-label">${escapeHtml(room.name)}</span>
            <span class="tree-room-count">${
              searchResults && roomIsRelevant
                ? matchingFileCount
                : room.files.length
            }</span>
        `;

    headerEl.addEventListener("click", () => {
      room.expanded = !room.expanded;
      roomEl.classList.toggle("expanded", room.expanded);
    });

    roomEl.appendChild(headerEl);

    // Files Container
    const filesEl = document.createElement("div");
    filesEl.className = "tree-files";

    if (room.files.length === 0) {
      // Empty room placeholder
      const emptyEl = document.createElement("div");
      emptyEl.className = "tree-empty";
      emptyEl.innerHTML = `
                <div class="tree-empty-icon">
                    <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                        <circle cx="8" cy="8" r="6" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2,2"/>
                    </svg>
                </div>
                <span class="tree-empty-label">No entry point</span>
            `;
      filesEl.appendChild(emptyEl);
    } else {
      for (const file of room.files) {
        const matchData = matchingFiles.get(file.fullPath);
        const fileMatches = !!matchData;

        const fileEl = document.createElement("div");
        fileEl.className = "tree-file";
        if (searchResults) {
          fileEl.classList.add(fileMatches ? "search-match" : "ghost-mode");
        }
        fileEl.dataset.filePath = file.fullPath;

        let lineHtml = "";
        if (fileMatches && matchData.lineNumber) {
          lineHtml = `<span class="tree-file-line">L${matchData.lineNumber}</span>`;
        }

        fileEl.innerHTML = `
                    <div class="tree-file-icon">
                        <svg viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
                            <path d="M13.85 4.44l-3.28-3.3-.35-.14H3.5l-.5.5v13l.5.5h10l.5-.5V4.8l-.15-.36zM10.5 2l2.65 2.65H10.5V2zM13 14H4V2h5.5v3.5l.5.5H13v8z"/>
                        </svg>
                    </div>
                    <span class="tree-file-label">${escapeHtml(
                      file.name
                    )}</span>
                    ${lineHtml}
                `;

        fileEl.addEventListener("click", () => {
          const lineNum = matchData ? matchData.lineNumber : file.lineNumber;
          vscode.postMessage({
            command: "openFile",
            filePath: file.fullPath,
            lineNumber: lineNum,
          });
        });

        // Tooltip on hover for search matches
        if (fileMatches && matchData.snippet) {
          fileEl.addEventListener("mouseenter", (e) => {
            tooltipTitle.textContent = file.name;
            tooltipDescription.style.display = "none";
            tooltipPath.style.display = "block";
            tooltipPath.textContent = file.fullPath;

            tooltipSnippet.style.display = "block";
            let snippetHtml = "";
            if (matchData.lineNumber) {
              snippetHtml =
                '<span class="tooltip-line-number">L' +
                matchData.lineNumber +
                "</span>";
            }
            snippetHtml += escapeHtml(matchData.snippet);
            tooltipSnippetContent.innerHTML = snippetHtml;
            tooltip.classList.add("search-match");
            tooltip.classList.add("visible");

            positionTooltip(e);
          });

          fileEl.addEventListener("mousemove", positionTooltip);

          fileEl.addEventListener("mouseleave", () => {
            tooltip.classList.remove("visible");
            tooltip.classList.remove("search-match");
          });
        }

        filesEl.appendChild(fileEl);
      }
    }

    roomEl.appendChild(filesEl);
    treeView.appendChild(roomEl);
  }

  // Auto-expand rooms with matches in search mode
  if (searchResults) {
    for (const room of treeData) {
      const roomEl = treeView.querySelector(`[data-room-id="${room.id}"]`);
      if (roomEl) {
        const hasMatches = room.files.some((f: any) =>
          matchingFiles.has(f.fullPath)
        );
        if (hasMatches) {
          room.expanded = true;
          roomEl.classList.add("expanded");
        }
      }
    }
  }
}

function positionTooltip(e: MouseEvent) {
  const padding = 14;
  const x = e.clientX + padding;
  const y = e.clientY + padding;

  const rect = tooltip.getBoundingClientRect();
  const maxX = window.innerWidth - rect.width - 8;
  const maxY = window.innerHeight - rect.height - 8;

  tooltip.style.left = Math.min(x, maxX) + "px";
  tooltip.style.top = Math.min(y, maxY) + "px";
}

function expandAllRooms() {
  for (const room of treeData) {
    room.expanded = true;
  }
  const roomEls = treeView.querySelectorAll(".tree-room");
  roomEls.forEach((el) => el.classList.add("expanded"));
}

function collapseAllRooms() {
  for (const room of treeData) {
    room.expanded = false;
  }
  const roomEls = treeView.querySelectorAll(".tree-room");
  roomEls.forEach((el) => el.classList.remove("expanded"));
}

// ══════════════════════════════════════════════════════════════════
// CYTOSCAPE HELPERS
// ══════════════════════════════════════════════════════════════════

function getCssVar(name: string, fallback: string): string {
  const value = getComputedStyle(document.documentElement)
    .getPropertyValue(name)
    .trim();
  return value || fallback;
}

function getCytoscapeStyle() {
  const fg = getCssVar("--vscode-editor-foreground", "#d4d4d4");
  const fgDim = getCssVar("--vscode-descriptionForeground", "#808080");
  const fgMuted = getCssVar("--vscode-editorLineNumber-foreground", "#5a5a5a");
  const accent = getCssVar("--vscode-focusBorder", "#007fd4");
  const accentSoft = getCssVar("--vscode-charts-blue", "#4fc1ff");
  const green = getCssVar("--vscode-charts-green", "#89d185");
  const purple = getCssVar("--vscode-charts-purple", "#c586c0");
  const orange = getCssVar("--vscode-charts-orange", "#d19a66");

  return [
    // ROOM: Normal State
    {
      selector: 'node[type="room"]',
      style: {
        shape: "round-rectangle",
        "corner-radius": 8,
        "background-color": fgMuted,
        "background-opacity": 0.04,
        "border-width": 1,
        "border-style": "solid",
        "border-color": fgMuted,
        "border-opacity": 0.25,
        label: "data(label)",
        "text-valign": "top",
        "text-halign": "center",
        "text-margin-y": 12,
        "font-size": "9px",
        "font-weight": "600",
        "text-transform": "uppercase",
        color: fgDim,
        "text-opacity": 0.7,
        padding: "40px",
        "compound-sizing-wrt-labels": "include",
        "min-width": "120px",
        "min-height": "70px",
      },
    },
    // ROOM: Search Match
    {
      selector: 'node[type="room"].search-match',
      style: {
        "background-color": purple,
        "background-opacity": 0.15,
        "border-color": purple,
        "border-opacity": 0.8,
        "border-width": 2,
        color: purple,
        "text-opacity": 1,
        "shadow-blur": 20,
        "shadow-color": purple,
        "shadow-opacity": 0.4,
        "shadow-offset-x": 0,
        "shadow-offset-y": 0,
      },
    },
    // ROOM: Ghost Mode
    {
      selector: 'node[type="room"].ghost-mode',
      style: {
        "background-opacity": 0.01,
        "border-opacity": 0.08,
        "text-opacity": 0.2,
      },
    },
    // FILE: Normal State
    {
      selector: 'node[type="file"]',
      style: {
        shape: "round-rectangle",
        "corner-radius": 4,
        "background-color": accentSoft,
        "background-opacity": 0.12,
        "border-width": 1,
        "border-style": "solid",
        "border-color": accentSoft,
        "border-opacity": 0.5,
        label: "data(label)",
        "text-valign": "center",
        "text-halign": "center",
        "font-size": "11px",
        "font-family": "var(--vscode-editor-font-family, monospace)",
        "font-weight": "500",
        color: fg,
        width: "label",
        height: "26px",
        padding: "12px",
      },
    },
    // FILE: Search Match
    {
      selector: 'node[type="file"].search-match',
      style: {
        "background-color": purple,
        "background-opacity": 0.35,
        "border-color": "#ffffff",
        "border-opacity": 0.9,
        "border-width": 2,
        color: "#ffffff",
        "font-weight": "600",
        "shadow-blur": 25,
        "shadow-color": purple,
        "shadow-opacity": 0.6,
        "shadow-offset-x": 0,
        "shadow-offset-y": 0,
        "z-index": 100,
      },
    },
    // FILE: Ghost Mode
    {
      selector: 'node[type="file"].ghost-mode',
      style: {
        "background-opacity": 0.03,
        "border-opacity": 0.1,
        "text-opacity": 0.15,
      },
    },
    // FILE: Hover in Search Mode
    {
      selector: 'node[type="file"].search-match:active',
      style: {
        "background-opacity": 0.5,
        "shadow-opacity": 0.8,
      },
    },
    // GHOST: Empty Placeholder
    {
      selector: 'node[type="ghost"]',
      style: {
        shape: "round-rectangle",
        "corner-radius": 3,
        "background-color": fgMuted,
        "background-opacity": 0.02,
        "border-width": 1,
        "border-style": "dotted",
        "border-color": fgMuted,
        "border-opacity": 0.15,
        label: "data(label)",
        "text-valign": "center",
        "text-halign": "center",
        "font-size": "9px",
        "font-style": "italic",
        color: fgMuted,
        "text-opacity": 0.4,
        width: "60px",
        height: "20px",
        padding: "4px",
      },
    },
    // Ghost in ghost mode
    {
      selector: 'node[type="ghost"].ghost-mode',
      style: {
        "background-opacity": 0.005,
        "border-opacity": 0.05,
        "text-opacity": 0.1,
      },
    },
    // SELECTION STATE
    {
      selector: ":selected",
      style: {
        "border-width": 2,
        "border-color": green,
        "border-opacity": 0.9,
      },
    },
    // EDGES
    {
      selector: "edge",
      style: {
        width: 1,
        "line-color": fgMuted,
        "line-opacity": 0.4,
        "line-style": "solid",
        "curve-style": "bezier",
        "target-arrow-shape": "triangle",
        "target-arrow-color": fgMuted,
        "arrow-scale": 0.6,
      },
    },
    {
      selector: "edge.ghost-mode",
      style: {
        "line-opacity": 0.1,
      },
    },
  ];
}

function getLayoutConfig() {
  return {
    name: "cose",
    animate: false,
    fit: true,
    padding: 40,
    nodeDimensionsIncludeLabels: true,
    nodeRepulsion: function (node: any) {
      return node.data("type") === "room" ? 20000 : 8000;
    },
    idealEdgeLength: 120,
    componentSpacing: 80,
    nestingFactor: 5,
    gravity: 0.15,
    numIter: 500,
    initialTemp: 300,
    coolingFactor: 0.95,
    minTemp: 1.0,
  };
}

function initCytoscape(elements: any) {
  if (cy) {
    cy.destroy();
  }

  cy = cytoscape({
    container: cyContainer,
    elements: elements,
    style: getCytoscapeStyle() as any,
    layout: getLayoutConfig(),
    wheelSensitivity: 0.25,
    minZoom: 0.25,
    maxZoom: 4,
    boxSelectionEnabled: false,
    selectionType: "single",
  });

  // File Click Handler
  cy.on("tap", 'node[type="file"]', function (evt: any) {
    const node = evt.target;
    const filePath = node.data("fullPath");
    const lineNumber = node.data("lineNumber");
    if (filePath) {
      vscode.postMessage({
        command: "openFile",
        filePath: filePath,
        lineNumber: lineNumber,
      });
    }
  });

  // Tooltip: Show on hover
  cy.on("mouseover", "node", function (evt: any) {
    const node = evt.target;
    const nodeType = node.data("type");

    if (nodeType === "ghost") return;

    tooltipTitle.textContent = node.data("label");

    const description = node.data("description");
    tooltipDescription.style.display = description ? "block" : "none";
    tooltipDescription.textContent = description || "";

    const fullPath = node.data("fullPath");
    tooltipPath.style.display = fullPath ? "block" : "none";
    tooltipPath.textContent = fullPath || "";

    // Show snippet for search matches
    const snippet = node.data("snippet");
    const lineNum = node.data("lineNumber");
    if (snippet && isSearchMode) {
      tooltipSnippet.style.display = "block";
      let snippetHtml = "";
      if (lineNum) {
        snippetHtml =
          '<span class="tooltip-line-number">L' + lineNum + "</span>";
      }
      snippetHtml += escapeHtml(snippet);
      tooltipSnippetContent.innerHTML = snippetHtml;
      tooltip.classList.add("search-match");
    } else {
      tooltipSnippet.style.display = "none";
      tooltip.classList.remove("search-match");
    }

    tooltip.classList.add("visible");
  });

  // Tooltip: Hide
  cy.on("mouseout", "node", function () {
    tooltip.classList.remove("visible");
  });

  // Tooltip: Follow cursor
  cy.on("mousemove", function (evt: any) {
    if (tooltip.classList.contains("visible") && evt.originalEvent) {
      const padding = 14;
      const x = evt.originalEvent.clientX + padding;
      const y = evt.originalEvent.clientY + padding;

      const rect = tooltip.getBoundingClientRect();
      const maxX = window.innerWidth - rect.width - 8;
      const maxY = window.innerHeight - rect.height - 8;

      tooltip.style.left = Math.min(x, maxX) + "px";
      tooltip.style.top = Math.min(y, maxY) + "px";
    }
  });

  // Cursor: Pointer on clickable nodes
  cy.on("mouseover", 'node[type="file"]', function () {
    cyContainer.style.cursor = "pointer";
  });
  cy.on("mouseout", 'node[type="file"]', function () {
    cyContainer.style.cursor = "default";
  });
}

// ══════════════════════════════════════════════════════════════════
// GRAPH UPDATE
// ══════════════════════════════════════════════════════════════════

function updateGraph(data: any) {
  loadingOverlay.classList.remove("visible");
  noResults.classList.remove("visible");

  if (!data || !data.nodes || data.nodes.length === 0) {
    emptyState.classList.add("visible");
    treeView.style.display = "none";
    mapView.style.display = "none";
    return;
  }

  emptyState.classList.remove("visible");

  // Store original data
  originalGraphData = data;

  // Build tree data
  treeData = buildTreeData(data);

  // Render appropriate view
  if (currentView === "list") {
    treeView.style.display = "";
    mapView.style.display = "none";
    renderTreeView();
  } else {
    treeView.style.display = "none";
    mapView.style.display = "";
    initCytoscape({
      nodes: data.nodes,
      edges: data.edges || [],
    });
  }
}

// ══════════════════════════════════════════════════════════════════
// SEARCH MODE
// ══════════════════════════════════════════════════════════════════

function enterSearchMode() {
  isSearchMode = true;
  canvasWrapper.classList.add("search-mode");
  indicator.classList.add("search-active");
}

function exitSearchMode() {
  isSearchMode = false;
  currentSearchResults = null;
  canvasWrapper.classList.remove("search-mode");
  indicator.classList.remove("search-active");
  indicator.classList.remove("searching");
  searchStatus.classList.remove("visible");
  noResults.classList.remove("visible");

  // Re-render tree without search highlights
  if (currentView === "list") {
    renderTreeView();
  }

  // Remove all search classes from map nodes
  if (cy) {
    cy.nodes().removeClass("search-match ghost-mode");
    cy.edges().removeClass("ghost-mode");
  }
}

function applySearchResults(results: any) {
  currentSearchResults = results;
  enterSearchMode();

  // Update match count
  matchCount.textContent = results.totalMatches;
  searchStatusText.textContent =
    results.totalMatches === 1 ? "match found" : "matches found";
  searchStatus.classList.add("visible");

  if (results.totalMatches === 0) {
    noResults.classList.add("visible");
    noResultsText.textContent = 'No matches for "' + results.query + '"';
    if (currentView === "list") {
      treeView.style.display = "none";
    } else {
      cyContainer.style.display = "none";
    }
    return;
  }

  noResults.classList.remove("visible");

  if (currentView === "list") {
    treeView.style.display = "";
    renderTreeView(results);
  } else {
    cyContainer.style.display = "";
    applySearchResultsToMap(results);
  }
}

function applySearchResultsToMap(results: any) {
  if (!cy || !results) return;

  // Get matching file paths and room names
  const matchingRooms = new Set();
  const matchingFiles = new Map();

  for (const roomResult of results.results) {
    matchingRooms.add(roomResult.roomName);
    for (const match of roomResult.matches) {
      matchingFiles.set(match.filePath, match);
    }
  }

  // Apply classes to nodes
  cy.nodes().forEach((node: any) => {
    const nodeType = node.data("type");
    const nodeId = node.data("id");
    const fullPath = node.data("fullPath");
    const label = node.data("label");

    if (nodeType === "room") {
      const roomName = label;
      const roomId = nodeId.replace("room-", "");

      if (matchingRooms.has(roomName) || matchingRooms.has(roomId)) {
        node.removeClass("ghost-mode");
        node.addClass("search-match");
      } else {
        node.removeClass("search-match");
        node.addClass("ghost-mode");
      }
    } else if (nodeType === "file") {
      const match = matchingFiles.get(fullPath);
      if (match) {
        node.removeClass("ghost-mode");
        node.addClass("search-match");
        node.data("snippet", match.snippet);
        node.data("lineNumber", match.lineNumber);
      } else {
        node.removeClass("search-match");
        node.addClass("ghost-mode");
        node.data("snippet", null);
        node.data("lineNumber", null);
      }
    } else {
      node.addClass("ghost-mode");
    }
  });

  cy.edges().addClass("ghost-mode");

  // Fit camera to matching nodes
  const matchingNodes = cy.nodes(".search-match");
  if (matchingNodes.length > 0) {
    cy.animate({
      fit: {
        eles: matchingNodes,
        padding: 50,
      },
      duration: 300,
      easing: "ease-out-cubic",
    });
  }
}

function escapeHtml(text: string): string {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}

// ══════════════════════════════════════════════════════════════════
// EVENT HANDLERS
// ══════════════════════════════════════════════════════════════════

// View toggle buttons
btnListView.addEventListener("click", () => switchView("list"));
btnMapView.addEventListener("click", () => switchView("map"));

// Expand/Collapse buttons
btnExpandAll.addEventListener("click", expandAllRooms);
btnCollapseAll.addEventListener("click", collapseAllRooms);

// Search input handler
searchInput.addEventListener("input", function (e) {
  const query = (e.target as HTMLInputElement).value;

  if (query.length > 0) {
    searchClear.classList.add("visible");
  } else {
    searchClear.classList.remove("visible");
  }

  vscode.postMessage({ command: "search", query: query });
});

// Search input keyboard handlers
searchInput.addEventListener("keydown", function (e) {
  if (e.key === "Escape") {
    searchInput.value = "";
    searchClear.classList.remove("visible");
    vscode.postMessage({ command: "clearSearch" });
  }
});

// Clear button
searchClear.addEventListener("click", function () {
  searchInput.value = "";
  searchClear.classList.remove("visible");
  searchInput.focus();
  vscode.postMessage({ command: "clearSearch" });
});

// Refresh button
btnRefresh.addEventListener("click", function () {
  loadingOverlay.classList.add("visible");
  vscode.postMessage({ command: "refresh" });
});

// Fit button (map view only)
btnFit.addEventListener("click", function () {
  if (cy) {
    const targetNodes = isSearchMode ? cy.nodes(".search-match") : cy.nodes();
    if (targetNodes.length > 0) {
      cy.animate({
        fit: {
          eles: targetNodes.length > 0 ? targetNodes : cy.elements(),
          padding: 40,
        },
        duration: 200,
        easing: "ease-out-cubic",
      });
    }
  }
});

// ══════════════════════════════════════════════════════════════════
// MESSAGE HANDLER
// ══════════════════════════════════════════════════════════════════

window.addEventListener("message", function (event) {
  const message = event.data;

  switch (message.command) {
    case "updateGraph":
      updateGraph(message.data);
      break;

    case "searchState":
      if (message.state === "searching") {
        indicator.classList.add("searching");
        indicator.classList.remove("search-active");
      } else if (message.state === "error") {
        indicator.classList.remove("searching");
        indicator.classList.add("disconnected");
        searchStatus.classList.add("visible");
        matchCount.textContent = "!";
        searchStatusText.textContent = message.error || "Search failed";
      }
      break;

    case "searchResults":
      indicator.classList.remove("searching");
      applySearchResults(message.results);
      break;

    case "clearSearchResults":
      exitSearchMode();
      // Restore original graph
      if (originalGraphData) {
        updateGraph(originalGraphData);
      }
      break;

    case "connectionStatus":
      if (message.connected) {
        indicator.classList.remove("disconnected");
      } else {
        indicator.classList.add("disconnected");
      }
      break;
  }
});

// ══════════════════════════════════════════════════════════════════
// INITIAL LOAD
// ══════════════════════════════════════════════════════════════════

// Set initial button visibility
btnFit.style.display = "none"; // Hidden in list view by default

loadingOverlay.classList.add("visible");
vscode.postMessage({ command: "ready" });
