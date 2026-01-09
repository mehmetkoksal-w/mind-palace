import { Component, OnInit, inject, signal, computed } from "@angular/core";
import { CommonModule } from "@angular/common";
import { FormsModule } from "@angular/forms";
import {
  ApiService,
  FileIntel,
  Learning,
} from "../../core/services/api.service";
import { LoggerService } from "../../core/services/logger.service";

interface TreeNode {
  name: string;
  path: string;
  children: TreeNode[];
  file?: FileIntel;
  isExpanded?: boolean;
}

@Component({
  selector: "app-intel",
  imports: [CommonModule, FormsModule],
  template: `
    <div class="intel-container">
      <header class="page-header">
        <h1>File Intelligence</h1>
        <p>Heat map visualization of file activity and learnings</p>
      </header>

      <div class="view-toggle">
        <button
          [class.active]="viewMode() === 'heatmap'"
          (click)="viewMode.set('heatmap')"
        >
          Heat Map
        </button>
        <button
          [class.active]="viewMode() === 'list'"
          (click)="viewMode.set('list')"
        >
          List View
        </button>
        <button
          [class.active]="viewMode() === 'tree'"
          (click)="viewMode.set('tree')"
        >
          Tree View
        </button>
      </div>

      @if (loading()) {
      <div class="loading">Loading file intelligence...</div>
      } @else {
      <div class="stats-row">
        <div class="stat-card">
          <span class="stat-value">{{ hotspots().length }}</span>
          <span class="stat-label">Files Tracked</span>
        </div>
        <div class="stat-card">
          <span class="stat-value">{{ totalEdits() }}</span>
          <span class="stat-label">Total Edits</span>
        </div>
        <div class="stat-card warning">
          <span class="stat-value">{{ fragile().length }}</span>
          <span class="stat-label">Fragile Files</span>
        </div>
        <div class="stat-card danger">
          <span class="stat-value">{{ totalFailures() }}</span>
          <span class="stat-label">Total Failures</span>
        </div>
      </div>

      @if (viewMode() === 'heatmap') {
      <div class="heatmap-container">
        <div class="heatmap-legend">
          <span class="legend-label">Fewer edits</span>
          <div class="legend-gradient"></div>
          <span class="legend-label">More edits</span>
        </div>

        <div class="heatmap-grid">
          @for (file of hotspots(); track file.path) {
          <div
            class="heatmap-cell"
            [class.has-failures]="file.failureCount > 0"
            [style.background-color]="getHeatColor(file)"
            [title]="file.path"
            (click)="selectFile(file)"
          >
            <span class="cell-name">{{ getFileName(file.path) }}</span>
            @if (file.failureCount > 0) {
            <span class="failure-badge">!</span>
            }
          </div>
          }
        </div>

        @if (hotspots().length === 0) {
        <div class="empty-state">
          <p>No file activity tracked yet.</p>
          <p class="hint">
            File intelligence is populated as agents work on your codebase.
          </p>
        </div>
        }
      </div>
      } @if (viewMode() === 'list') {
      <div class="sections-container">
        <div class="section">
          <h3>Hotspots (Most Edited)</h3>
          <div class="files-list">
            @for (file of hotspots(); track file.path) {
            <div class="file-card" (click)="selectFile(file)">
              <div class="file-header">
                <div
                  class="heat-indicator"
                  [style.background-color]="getHeatColor(file)"
                ></div>
                <div class="file-path">{{ file.path }}</div>
              </div>
              <div class="file-stats">
                <span class="edits">{{ file.editCount }} edits</span>
                @if (file.failureCount > 0) {
                <span class="failures">{{ file.failureCount }} failures</span>
                } @if (file.lastEditor) {
                <span class="editor">by {{ file.lastEditor }}</span>
                }
              </div>
              <div class="edit-bar">
                <div
                  class="edit-fill"
                  [style.width.%]="getEditPercentage(file)"
                ></div>
                @if (file.failureCount > 0) {
                <div
                  class="failure-fill"
                  [style.width.%]="getFailurePercentage(file)"
                ></div>
                }
              </div>
            </div>
            } @if (hotspots().length === 0) {
            <div class="empty">No files tracked yet</div>
            }
          </div>
        </div>

        <div class="section fragile-section">
          <h3>Fragile Files (High Failure Rate)</h3>
          <div class="files-list">
            @for (file of fragile(); track file.path) {
            <div class="file-card fragile" (click)="selectFile(file)">
              <div class="file-header">
                <div class="heat-indicator danger"></div>
                <div class="file-path">{{ file.path }}</div>
              </div>
              <div class="file-stats">
                <span class="failures">{{ file.failureCount }} failures</span>
                <span class="edits">{{ file.editCount }} edits</span>
                <span class="rate"
                  >{{ getFailureRate(file) }}% failure rate</span
                >
              </div>
            </div>
            } @if (fragile().length === 0) {
            <div class="empty">No fragile files detected</div>
            }
          </div>
        </div>
      </div>
      } @if (viewMode() === 'tree') {
      <div class="tree-container">
        @for (node of fileTree(); track node.path) {
        <ng-container
          *ngTemplateOutlet="treeNode; context: { node: node, depth: 0 }"
        ></ng-container>
        } @if (fileTree().length === 0) {
        <div class="empty-state">
          <p>No file activity tracked yet.</p>
        </div>
        }
      </div>

      <ng-template #treeNode let-node="node" let-depth="depth">
        <div class="tree-item" [style.padding-left.px]="depth * 20">
          @if (node.children.length > 0) {
          <button class="expand-btn" (click)="toggleNode(node)">
            {{ node.isExpanded ? "-" : "+" }}
          </button>
          } @else {
          <span class="leaf-space"></span>
          } @if (node.file) {
          <div class="tree-file" (click)="selectFile(node.file)">
            <div
              class="heat-indicator"
              [style.background-color]="getHeatColor(node.file)"
            ></div>
            <span class="node-name">{{ node.name }}</span>
            <span class="node-stats">
              {{ node.file.editCount }} edits @if (node.file.failureCount > 0) {
              <span class="failure-count"
                >{{ node.file.failureCount }} failures</span
              >
              }
            </span>
          </div>
          } @else {
          <div class="tree-folder" (click)="toggleNode(node)">
            <span class="folder-icon">{{ node.isExpanded ? "📂" : "📁" }}</span>
            <span class="node-name">{{ node.name }}</span>
            <span class="folder-count">({{ countFiles(node) }} files)</span>
          </div>
          }
        </div>

        @if (node.isExpanded && node.children.length > 0) { @for (child of
        node.children; track child.path) {
        <ng-container
          *ngTemplateOutlet="
            treeNode;
            context: { node: child, depth: depth + 1 }
          "
        ></ng-container>
        } }
      </ng-template>
      } @if (selectedFile()) {
      <div class="file-detail-panel">
        <div class="detail-header">
          <h3>{{ selectedFile()!.path }}</h3>
          <button class="close-btn" (click)="selectedFile.set(null)">
            Close
          </button>
        </div>
        <div class="detail-stats">
          <div class="detail-stat">
            <span class="value">{{ selectedFile()!.editCount }}</span>
            <span class="label">Edits</span>
          </div>
          <div
            class="detail-stat"
            [class.danger]="selectedFile()!.failureCount > 0"
          >
            <span class="value">{{ selectedFile()!.failureCount }}</span>
            <span class="label">Failures</span>
          </div>
          <div class="detail-stat">
            <span class="value">{{ getFailureRate(selectedFile()!) }}%</span>
            <span class="label">Failure Rate</span>
          </div>
        </div>
        @if (selectedFile()!.lastEditor) {
        <div class="last-edit-info">
          Last edited by <strong>{{ selectedFile()!.lastEditor }}</strong> @if
          (selectedFile()!.lastEdited) { on
          {{ selectedFile()!.lastEdited | date : "medium" }}
          }
        </div>
        } @if (fileLearnings().length > 0) {
        <div class="learnings-section">
          <h4>Associated Learnings</h4>
          <ul class="learnings-list">
            @for (learning of fileLearnings(); track learning.id) {
            <li>
              <span class="confidence" [style.opacity]="learning.confidence"
                >●</span
              >
              {{ learning.content }}
            </li>
            }
          </ul>
        </div>
        }
      </div>
      } }
    </div>
  `,
  styles: [
    `
      .intel-container {
        padding: 24px;
        max-width: 1400px;
        margin: 0 auto;
      }

      .page-header {
        margin-bottom: 24px;
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

      .view-toggle {
        display: flex;
        gap: 8px;
        margin-bottom: 24px;
      }

      .view-toggle button {
        padding: 8px 16px;
        background: #16213e;
        border: 1px solid #2d3748;
        border-radius: 6px;
        color: #888;
        cursor: pointer;
        transition: all 0.2s;
      }

      .view-toggle button:hover {
        border-color: #4ade80;
        color: #ccc;
      }

      .view-toggle button.active {
        background: #4ade80;
        border-color: #4ade80;
        color: #000;
      }

      .loading {
        text-align: center;
        padding: 48px;
        color: #888;
      }

      .stats-row {
        display: flex;
        gap: 16px;
        margin-bottom: 24px;
        flex-wrap: wrap;
      }

      .stat-card {
        background: #16213e;
        padding: 16px 24px;
        border-radius: 8px;
        display: flex;
        flex-direction: column;
        min-width: 140px;
      }

      .stat-card.warning .stat-value {
        color: #fbbf24;
      }

      .stat-card.danger .stat-value {
        color: #f87171;
      }

      .stat-value {
        font-size: 28px;
        font-weight: 600;
        color: #4ade80;
      }

      .stat-label {
        font-size: 12px;
        color: #888;
        text-transform: uppercase;
      }

      /* Heat Map Styles */
      .heatmap-container {
        background: #16213e;
        border-radius: 8px;
        padding: 24px;
      }

      .heatmap-legend {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 20px;
        justify-content: center;
      }

      .legend-label {
        font-size: 12px;
        color: #888;
      }

      .legend-gradient {
        width: 200px;
        height: 12px;
        border-radius: 6px;
        background: linear-gradient(
          to right,
          #1e3a5f,
          #3b82f6,
          #f59e0b,
          #ef4444
        );
      }

      .heatmap-grid {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
      }

      .heatmap-cell {
        position: relative;
        width: 80px;
        height: 60px;
        border-radius: 6px;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        transition: transform 0.2s, box-shadow 0.2s;
        overflow: hidden;
      }

      .heatmap-cell:hover {
        transform: scale(1.1);
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
        z-index: 1;
      }

      .heatmap-cell.has-failures {
        border: 2px solid #f87171;
      }

      .cell-name {
        font-size: 10px;
        color: #fff;
        text-align: center;
        text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
        padding: 4px;
        word-break: break-all;
      }

      .failure-badge {
        position: absolute;
        top: 4px;
        right: 4px;
        width: 14px;
        height: 14px;
        background: #f87171;
        border-radius: 50%;
        font-size: 10px;
        font-weight: bold;
        display: flex;
        align-items: center;
        justify-content: center;
        color: #fff;
      }

      /* empty-state styles in global styles.scss */

      /* List View Styles */
      .sections-container {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 24px;
      }

      @media (max-width: 900px) {
        .sections-container {
          grid-template-columns: 1fr;
        }
      }

      .section {
        background: #16213e;
        border-radius: 12px;
        padding: 20px;
      }

      .section h3 {
        color: #fff;
        margin: 0 0 16px 0;
        font-size: 16px;
      }

      .files-list {
        display: flex;
        flex-direction: column;
        gap: 12px;
      }

      .file-card {
        background: #0f172a;
        border-radius: 8px;
        padding: 12px 16px;
        cursor: pointer;
        transition: all 0.2s;
      }

      .file-card:hover {
        background: #1e293b;
      }

      .file-header {
        display: flex;
        align-items: center;
        gap: 10px;
        margin-bottom: 8px;
      }

      .heat-indicator {
        width: 12px;
        height: 12px;
        border-radius: 3px;
        flex-shrink: 0;
      }

      .heat-indicator.danger {
        background: #f87171;
      }

      .file-path {
        font-family: monospace;
        font-size: 13px;
        color: #fff;
        word-break: break-all;
      }

      .file-stats {
        display: flex;
        gap: 12px;
        font-size: 12px;
        margin-bottom: 8px;
      }

      .edits {
        color: #4ade80;
      }

      .failures {
        color: #f87171;
      }

      .editor {
        color: #888;
      }

      .rate {
        color: #f87171;
        font-weight: 600;
      }

      .edit-bar {
        height: 4px;
        background: #2d3748;
        border-radius: 2px;
        overflow: hidden;
        position: relative;
      }

      .edit-fill {
        height: 100%;
        background: #4ade80;
        border-radius: 2px;
      }

      .failure-fill {
        position: absolute;
        top: 0;
        right: 0;
        height: 100%;
        background: #f87171;
        border-radius: 2px;
      }

      .empty {
        text-align: center;
        color: #666;
        padding: 24px;
      }

      /* Tree View Styles */
      .tree-container {
        background: #16213e;
        border-radius: 8px;
        padding: 20px;
      }

      .tree-item {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 6px 0;
      }

      .expand-btn {
        width: 20px;
        height: 20px;
        background: #2d3748;
        border: none;
        border-radius: 4px;
        color: #fff;
        cursor: pointer;
        font-size: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
      }

      .expand-btn:hover {
        background: #4a5568;
      }

      .leaf-space {
        width: 20px;
      }

      .tree-file,
      .tree-folder {
        display: flex;
        align-items: center;
        gap: 8px;
        flex: 1;
        padding: 4px 8px;
        border-radius: 4px;
        cursor: pointer;
      }

      .tree-file:hover,
      .tree-folder:hover {
        background: #2d3748;
      }

      .folder-icon {
        font-size: 14px;
      }

      .node-name {
        color: #fff;
        font-size: 13px;
      }

      .node-stats {
        color: #4ade80;
        font-size: 11px;
        margin-left: auto;
      }

      .node-stats .failure-count {
        color: #f87171;
        margin-left: 8px;
      }

      .folder-count {
        color: #888;
        font-size: 11px;
      }

      /* File Detail Panel */
      .file-detail-panel {
        position: fixed;
        bottom: 0;
        right: 0;
        width: 400px;
        background: #16213e;
        border-top-left-radius: 12px;
        padding: 20px;
        box-shadow: -4px -4px 20px rgba(0, 0, 0, 0.3);
      }

      .detail-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 16px;
      }

      .detail-header h3 {
        margin: 0;
        font-size: 14px;
        font-family: monospace;
        color: #fff;
        word-break: break-all;
      }

      .close-btn {
        background: none;
        border: none;
        color: #888;
        cursor: pointer;
        font-size: 12px;
      }

      .close-btn:hover {
        color: #fff;
      }

      .detail-stats {
        display: flex;
        gap: 24px;
        margin-bottom: 16px;
      }

      .detail-stat {
        display: flex;
        flex-direction: column;
      }

      .detail-stat .value {
        font-size: 24px;
        font-weight: 600;
        color: #4ade80;
      }

      .detail-stat.danger .value {
        color: #f87171;
      }

      .detail-stat .label {
        font-size: 11px;
        color: #888;
        text-transform: uppercase;
      }

      .last-edit-info {
        font-size: 12px;
        color: #888;
        margin-bottom: 16px;
      }

      .last-edit-info strong {
        color: #ccc;
      }

      .learnings-section h4 {
        margin: 0 0 12px 0;
        font-size: 13px;
        color: #888;
        text-transform: uppercase;
      }

      .learnings-list {
        list-style: none;
        padding: 0;
        margin: 0;
      }

      .learnings-list li {
        display: flex;
        align-items: flex-start;
        gap: 8px;
        padding: 8px 0;
        border-bottom: 1px solid #2d3748;
        font-size: 13px;
        color: #ccc;
      }

      .learnings-list li:last-child {
        border-bottom: none;
      }

      .confidence {
        color: #4ade80;
      }
    `,
  ],
})
export class IntelComponent implements OnInit {
  private readonly api = inject(ApiService);
  private readonly logger = inject(LoggerService).forContext("IntelComponent");

  hotspots = signal<FileIntel[]>([]);
  fragile = signal<FileIntel[]>([]);
  loading = signal(true);
  viewMode = signal<"heatmap" | "list" | "tree">("heatmap");
  selectedFile = signal<FileIntel | null>(null);
  fileLearnings = signal<Learning[]>([]);
  fileTree = signal<TreeNode[]>([]);

  private maxEdits = 1;

  ngOnInit() {
    this.loadData();
  }

  loadData() {
    this.loading.set(true);
    this.api.getHotspots().subscribe({
      next: (data) => {
        const hotspots = data.hotspots || [];
        this.hotspots.set(hotspots);
        this.fragile.set(data.fragile || []);
        this.maxEdits = Math.max(1, ...hotspots.map((f) => f.editCount));
        this.buildFileTree(hotspots);
        this.loading.set(false);
      },
      error: (err) => {
        this.logger.error("Failed to load file intelligence", err, {
          endpoint: "/api/intel/hotspots",
        });
        this.loading.set(false);
      },
    });
  }

  totalEdits(): number {
    return this.hotspots().reduce((sum, f) => sum + f.editCount, 0);
  }

  totalFailures(): number {
    return this.hotspots().reduce((sum, f) => sum + f.failureCount, 0);
  }

  getHeatColor(file: FileIntel): string {
    const ratio = file.editCount / this.maxEdits;

    if (ratio < 0.25) return "#1e3a5f";
    if (ratio < 0.5) return "#3b82f6";
    if (ratio < 0.75) return "#f59e0b";
    return "#ef4444";
  }

  getFileName(path: string): string {
    const parts = path.split("/");
    return parts[parts.length - 1] || path;
  }

  getEditPercentage(file: FileIntel): number {
    return (file.editCount / this.maxEdits) * 100;
  }

  getFailurePercentage(file: FileIntel): number {
    if (file.editCount === 0) return 0;
    return Math.min(100, (file.failureCount / file.editCount) * 100);
  }

  getFailureRate(file: FileIntel): string {
    if (file.editCount === 0) return "0";
    return ((file.failureCount / file.editCount) * 100).toFixed(0);
  }

  selectFile(file: FileIntel) {
    this.selectedFile.set(file);
    this.loadFileLearnings(file.path);
  }

  loadFileLearnings(path: string) {
    this.api.getFileIntel(path).subscribe({
      next: (data) => {
        this.fileLearnings.set(data.learnings || []);
      },
      error: () => this.fileLearnings.set([]),
    });
  }

  buildFileTree(files: FileIntel[]) {
    const root: TreeNode[] = [];
    const nodeMap = new Map<string, TreeNode>();

    for (const file of files) {
      const parts = file.path.split("/").filter((p) => p);
      let currentPath = "";

      for (let i = 0; i < parts.length; i++) {
        const part = parts[i];
        const isLast = i === parts.length - 1;
        const parentPath = currentPath;
        currentPath = currentPath ? `${currentPath}/${part}` : part;

        if (!nodeMap.has(currentPath)) {
          const node: TreeNode = {
            name: part,
            path: currentPath,
            children: [],
            file: isLast ? file : undefined,
            isExpanded: i < 2,
          };
          nodeMap.set(currentPath, node);

          if (parentPath) {
            const parent = nodeMap.get(parentPath);
            if (parent) {
              parent.children.push(node);
            }
          } else {
            root.push(node);
          }
        } else if (isLast) {
          const existingNode = nodeMap.get(currentPath)!;
          existingNode.file = file;
        }
      }
    }

    this.fileTree.set(root);
  }

  toggleNode(node: TreeNode) {
    node.isExpanded = !node.isExpanded;
    this.fileTree.set([...this.fileTree()]);
  }

  countFiles(node: TreeNode): number {
    let count = node.file ? 1 : 0;
    for (const child of node.children) {
      count += this.countFiles(child);
    }
    return count;
  }
}
