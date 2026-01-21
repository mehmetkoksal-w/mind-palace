import { Component, OnInit, inject, signal, computed } from "@angular/core";
import { FormsModule } from "@angular/forms";
import {
  ApiService,
  Contract,
  ContractStats,
} from "../../core/services/api.service";
import { LoggerService } from "../../core/services/logger.service";

@Component({
  selector: "app-contracts",
  imports: [FormsModule],
  template: `
    <div class="contracts">
      <h2>API Contracts</h2>

      <!-- Stats Summary -->
      <div class="stats-bar">
        @if (stats()) {
          <div class="stat">
            <span class="value">{{ stats()!.total }}</span>
            <span class="label">Total</span>
          </div>
          <div class="stat discovered">
            <span class="value">{{ stats()!.discovered }}</span>
            <span class="label">Discovered</span>
          </div>
          <div class="stat verified">
            <span class="value">{{ stats()!.verified }}</span>
            <span class="label">Verified</span>
          </div>
          <div class="stat mismatch">
            <span class="value">{{ stats()!.mismatch }}</span>
            <span class="label">Mismatch</span>
          </div>
          <div class="stat ignored">
            <span class="value">{{ stats()!.ignored }}</span>
            <span class="label">Ignored</span>
          </div>
          @if (stats()!.totalErrors > 0 || stats()!.totalWarnings > 0) {
            <div class="stat errors">
              <span class="value">{{ stats()!.totalErrors }}</span>
              <span class="label">Errors</span>
            </div>
            <div class="stat warnings">
              <span class="value">{{ stats()!.totalWarnings }}</span>
              <span class="label">Warnings</span>
            </div>
          }
        }
      </div>

      <!-- Filters -->
      <div class="filters">
        <select [(ngModel)]="methodFilter" (change)="loadContracts()">
          <option value="">All Methods</option>
          <option value="GET">GET</option>
          <option value="POST">POST</option>
          <option value="PUT">PUT</option>
          <option value="PATCH">PATCH</option>
          <option value="DELETE">DELETE</option>
        </select>

        <select [(ngModel)]="statusFilter" (change)="loadContracts()">
          <option value="">All Status</option>
          <option value="discovered">Discovered</option>
          <option value="verified">Verified</option>
          <option value="mismatch">Mismatch</option>
          <option value="ignored">Ignored</option>
        </select>

        <input
          type="text"
          placeholder="Filter by endpoint..."
          [(ngModel)]="endpointFilter"
          (input)="loadContracts()"
        />

        <label class="mismatch-toggle">
          <input
            type="checkbox"
            [(ngModel)]="mismatchesOnly"
            (change)="loadContracts()"
          />
          <span>Mismatches Only</span>
        </label>
      </div>

      <!-- Contracts List -->
      <div class="contracts-list">
        @for (contract of contracts(); track contract.id) {
          <div
            class="contract-card"
            [class.has-mismatches]="contract.mismatches.length > 0"
            [class.expanded]="expandedId === contract.id"
          >
            <div class="contract-header" (click)="toggleExpanded(contract.id)">
              <span class="method-badge" [attr.data-method]="contract.method">
                {{ contract.method }}
              </span>
              <span class="endpoint">{{ contract.endpoint }}</span>
              <span class="status-badge" [attr.data-status]="contract.status">
                {{ contract.status }}
              </span>
              @if (contract.mismatches.length > 0) {
                <span class="mismatch-count">
                  {{ contract.mismatches!.length }} issues
                </span>
              }
              <span class="expand-icon">{{
                expandedId === contract.id ? "-" : "+"
              }}</span>
            </div>

            <div class="confidence-section">
              <div class="confidence-bar">
                <div
                  class="fill"
                  [style.width.%]="contract.confidence * 100"
                  [class.high]="contract.confidence >= 0.85"
                  [class.medium]="
                    contract.confidence >= 0.7 && contract.confidence < 0.85
                  "
                  [class.low]="contract.confidence < 0.7"
                ></div>
              </div>
              <span class="confidence-text"
                >{{ (contract.confidence * 100).toFixed(0) }}%</span
              >
            </div>

            <div class="contract-meta">
              <span class="calls"
                >{{ contract.frontendCalls.length || 0 }} frontend calls</span
              >
              <span class="framework">{{ contract.backend.framework }}</span>
              <span class="handler">{{ contract.backend.handler }}</span>
            </div>

            @if (expandedId === contract.id) {
              <div class="contract-details">
                <!-- Backend Info -->
                <div class="detail-section">
                  <h4>Backend</h4>
                  <div class="detail-row">
                    <span class="label">File:</span>
                    <span class="value file-link"
                      >{{ contract.backend.file }}:{{
                        contract.backend.line
                      }}</span
                    >
                  </div>
                  <div class="detail-row">
                    <span class="label">Handler:</span>
                    <span class="value">{{ contract.backend.handler }}</span>
                  </div>
                  @if (contract.backend.responseSchema) {
                    <div class="detail-row">
                      <span class="label">Response:</span>
                      <span class="value schema">{{
                        formatSchema(contract.backend.responseSchema)
                      }}</span>
                    </div>
                  }
                </div>

                <!-- Frontend Calls -->
                @if (contract.frontendCalls.length > 0) {
                  <div class="detail-section">
                    <h4>
                      Frontend Calls ({{ contract.frontendCalls!.length }})
                    </h4>
                    @for (
                      call of contract.frontendCalls!.slice(0, 5);
                      track call.id
                    ) {
                      <div class="call-row">
                        <span class="file-link"
                          >{{ call.file }}:{{ call.line }}</span
                        >
                        <span class="call-type">{{ call.callType }}</span>
                      </div>
                    }
                    @if (contract.frontendCalls!.length > 5) {
                      <div class="more">
                        ... and {{ contract.frontendCalls!.length - 5 }} more
                      </div>
                    }
                  </div>
                }

                <!-- Mismatches -->
                @if (contract.mismatches.length > 0) {
                  <div class="detail-section mismatches">
                    <h4>Mismatches ({{ contract.mismatches!.length }})</h4>
                    @for (mismatch of contract.mismatches!; track mismatch.id) {
                      <div
                        class="mismatch-row"
                        [class.error]="mismatch.severity === 'error'"
                        [class.warning]="mismatch.severity === 'warning'"
                      >
                        <span class="severity-badge">{{
                          mismatch.severity
                        }}</span>
                        <span class="field-path">{{ mismatch.fieldPath }}</span>
                        <span class="description">{{
                          mismatch.description
                        }}</span>
                        @if (mismatch.backendType && mismatch.frontendType) {
                          <span class="types"
                            >Backend: {{ mismatch.backendType }}, Frontend:
                            {{ mismatch.frontendType }}</span
                          >
                        }
                      </div>
                    }
                  </div>
                }

                <!-- Actions -->
                @if (
                  contract.status === "discovered" ||
                  contract.status === "mismatch"
                ) {
                  <div class="contract-actions">
                    <button
                      class="verify"
                      (click)="verify(contract.id); $event.stopPropagation()"
                    >
                      Verify
                    </button>
                    <button
                      class="ignore"
                      (click)="ignore(contract.id); $event.stopPropagation()"
                    >
                      Ignore
                    </button>
                  </div>
                }
              </div>
            }
          </div>
        }
        @if (contracts().length === 0) {
          <div class="empty">
            No contracts found. Run <code>palace contracts scan</code> to detect
            API contracts.
          </div>
        }
      </div>
    </div>
  `,
  styles: [
    `
      .contracts h2 {
        color: #00b4d8;
        margin-bottom: 1.5rem;
      }

      .stats-bar {
        display: flex;
        gap: 1.5rem;
        margin-bottom: 1.5rem;
        padding: 1rem;
        background: #16213e;
        border-radius: 8px;
        flex-wrap: wrap;
      }

      .stat {
        display: flex;
        flex-direction: column;
        align-items: center;
      }

      .stat .value {
        font-size: 1.5rem;
        font-weight: bold;
        color: #eee;
      }

      .stat .label {
        font-size: 0.75rem;
        color: #718096;
        text-transform: uppercase;
      }

      .stat.discovered .value {
        color: #00b4d8;
      }

      .stat.verified .value {
        color: #00d26a;
      }

      .stat.mismatch .value {
        color: #e53e3e;
      }

      .stat.ignored .value {
        color: #718096;
      }

      .stat.errors .value {
        color: #e53e3e;
      }

      .stat.warnings .value {
        color: #d69e2e;
      }

      .filters {
        display: flex;
        gap: 0.75rem;
        margin-bottom: 1.5rem;
        flex-wrap: wrap;
      }

      .filters select,
      .filters input {
        padding: 0.5rem 0.75rem;
        background: #16213e;
        border: 1px solid #2d3748;
        border-radius: 6px;
        color: #eee;
        font-size: 0.875rem;
      }

      .filters input[type="text"] {
        width: 200px;
      }

      .mismatch-toggle {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.5rem 0.75rem;
        background: #16213e;
        border: 1px solid #2d3748;
        border-radius: 6px;
        color: #eee;
        font-size: 0.875rem;
        cursor: pointer;
      }

      .mismatch-toggle:hover {
        border-color: #e53e3e;
      }

      .contracts-list {
        display: flex;
        flex-direction: column;
        gap: 1rem;
      }

      .contract-card {
        background: #16213e;
        border-radius: 12px;
        padding: 1.25rem;
        border: 2px solid transparent;
        transition: border-color 0.2s;
        cursor: pointer;
      }

      .contract-card:hover {
        border-color: #2d3748;
      }

      .contract-card.has-mismatches {
        border-color: #e53e3e33;
      }

      .contract-card.expanded {
        border-color: #00b4d8;
      }

      .contract-header {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        margin-bottom: 0.75rem;
      }

      .method-badge {
        padding: 0.25rem 0.5rem;
        border-radius: 4px;
        font-size: 0.75rem;
        font-weight: 600;
        text-transform: uppercase;
        min-width: 50px;
        text-align: center;
      }

      .method-badge[data-method="GET"] {
        background: #38a169;
        color: #fff;
      }

      .method-badge[data-method="POST"] {
        background: #3182ce;
        color: #fff;
      }

      .method-badge[data-method="PUT"] {
        background: #dd6b20;
        color: #fff;
      }

      .method-badge[data-method="PATCH"] {
        background: #805ad5;
        color: #fff;
      }

      .method-badge[data-method="DELETE"] {
        background: #e53e3e;
        color: #fff;
      }

      .endpoint {
        flex: 1;
        font-weight: 500;
        color: #eee;
        font-family: monospace;
      }

      .status-badge {
        padding: 0.25rem 0.5rem;
        border-radius: 4px;
        font-size: 0.75rem;
        font-weight: 500;
      }

      .status-badge[data-status="discovered"] {
        background: #00b4d8;
        color: #fff;
      }

      .status-badge[data-status="verified"] {
        background: #00d26a;
        color: #fff;
      }

      .status-badge[data-status="mismatch"] {
        background: #e53e3e;
        color: #fff;
      }

      .status-badge[data-status="ignored"] {
        background: #718096;
        color: #fff;
      }

      .mismatch-count {
        color: #e53e3e;
        font-size: 0.8rem;
        font-weight: 500;
      }

      .expand-icon {
        color: #718096;
        font-size: 1.2rem;
        width: 24px;
        text-align: center;
      }

      .confidence-section {
        display: flex;
        align-items: center;
        gap: 0.75rem;
        margin-bottom: 0.75rem;
      }

      .confidence-bar {
        flex: 1;
        height: 6px;
        background: #2d3748;
        border-radius: 3px;
        overflow: hidden;
      }

      .confidence-bar .fill {
        height: 100%;
        transition: width 0.3s;
      }

      .confidence-bar .fill.high {
        background: linear-gradient(90deg, #00d26a, #00b4d8);
      }

      .confidence-bar .fill.medium {
        background: linear-gradient(90deg, #d69e2e, #dd6b20);
      }

      .confidence-bar .fill.low {
        background: linear-gradient(90deg, #e53e3e, #dd6b20);
      }

      .confidence-text {
        color: #00d26a;
        font-weight: bold;
        min-width: 40px;
        font-size: 0.875rem;
      }

      .contract-meta {
        display: flex;
        gap: 1rem;
        color: #718096;
        font-size: 0.8rem;
      }

      .contract-details {
        margin-top: 1rem;
        padding-top: 1rem;
        border-top: 1px solid #2d3748;
      }

      .detail-section {
        margin-bottom: 1rem;
      }

      .detail-section h4 {
        color: #a0aec0;
        font-size: 0.875rem;
        margin-bottom: 0.5rem;
        text-transform: uppercase;
        letter-spacing: 0.05em;
      }

      .detail-row {
        display: flex;
        gap: 0.5rem;
        padding: 0.25rem 0;
        font-size: 0.875rem;
      }

      .detail-row .label {
        color: #718096;
        min-width: 80px;
      }

      .detail-row .value {
        color: #eee;
      }

      .file-link {
        color: #00b4d8;
        font-family: monospace;
      }

      .schema {
        font-family: monospace;
        font-size: 0.8rem;
        color: #a0aec0;
      }

      .call-row {
        display: flex;
        gap: 1rem;
        padding: 0.25rem 0;
        font-size: 0.875rem;
      }

      .call-type {
        color: #805ad5;
        font-size: 0.75rem;
        padding: 0.125rem 0.5rem;
        background: #805ad533;
        border-radius: 4px;
      }

      .more {
        color: #718096;
        font-size: 0.8rem;
        font-style: italic;
        padding-top: 0.25rem;
      }

      .detail-section.mismatches h4 {
        color: #e53e3e;
      }

      .mismatch-row {
        display: flex;
        flex-direction: column;
        gap: 0.25rem;
        padding: 0.5rem;
        margin-bottom: 0.5rem;
        background: #1a1a2e;
        border-radius: 6px;
        border-left: 3px solid #d69e2e;
      }

      .mismatch-row.error {
        border-left-color: #e53e3e;
      }

      .mismatch-row.warning {
        border-left-color: #d69e2e;
      }

      .severity-badge {
        font-size: 0.65rem;
        text-transform: uppercase;
        font-weight: 600;
        padding: 0.125rem 0.375rem;
        border-radius: 3px;
        align-self: flex-start;
      }

      .mismatch-row.error .severity-badge {
        background: #e53e3e;
        color: #fff;
      }

      .mismatch-row.warning .severity-badge {
        background: #d69e2e;
        color: #fff;
      }

      .field-path {
        font-family: monospace;
        color: #eee;
        font-size: 0.875rem;
      }

      .description {
        color: #a0aec0;
        font-size: 0.8rem;
      }

      .types {
        color: #718096;
        font-size: 0.75rem;
        font-family: monospace;
      }

      .contract-actions {
        display: flex;
        gap: 0.5rem;
        margin-top: 1rem;
        padding-top: 1rem;
        border-top: 1px solid #2d3748;
      }

      .contract-actions button {
        padding: 0.5rem 1rem;
        border: none;
        border-radius: 6px;
        cursor: pointer;
        font-weight: 500;
        transition: background 0.2s;
      }

      .contract-actions .verify {
        background: #00d26a;
        color: #fff;
      }

      .contract-actions .verify:hover {
        background: #00b85c;
      }

      .contract-actions .ignore {
        background: #4a5568;
        color: #fff;
      }

      .contract-actions .ignore:hover {
        background: #2d3748;
      }

      .empty {
        text-align: center;
        color: #718096;
        padding: 2rem;
      }

      .empty code {
        background: #2d3748;
        padding: 0.25rem 0.5rem;
        border-radius: 4px;
        color: #00b4d8;
      }
    `,
  ],
})
export class ContractsComponent implements OnInit {
  private readonly api = inject(ApiService);
  private readonly logger =
    inject(LoggerService).forContext("ContractsComponent");

  contracts = signal<Contract[]>([]);
  stats = signal<ContractStats | null>(null);
  expandedId: string | null = null;

  methodFilter = "";
  statusFilter = "";
  endpointFilter = "";
  mismatchesOnly = false;

  ngOnInit() {
    this.loadContracts();
    this.loadStats();
  }

  loadContracts() {
    const params: any = {};
    if (this.methodFilter) params.method = this.methodFilter;
    if (this.statusFilter) params.status = this.statusFilter;
    if (this.endpointFilter) params.endpoint = this.endpointFilter;
    if (this.mismatchesOnly) params.hasMismatches = true;

    this.api.getContracts(params).subscribe({
      next: (data) => this.contracts.set(data.contracts || []),
      error: (err) =>
        this.logger.error("Failed to load contracts", err, {
          endpoint: "/api/contracts",
        }),
    });
  }

  loadStats() {
    this.api.getContractStats().subscribe({
      next: (data) => this.stats.set(data),
      error: (err) =>
        this.logger.error("Failed to load contract stats", err, {
          endpoint: "/api/contracts/stats",
        }),
    });
  }

  toggleExpanded(id: string) {
    this.expandedId = this.expandedId === id ? null : id;
  }

  verify(id: string) {
    this.api.verifyContract(id).subscribe({
      next: () => {
        this.loadContracts();
        this.loadStats();
      },
      error: (err) =>
        this.logger.error("Failed to verify contract", err, { contractId: id }),
    });
  }

  ignore(id: string) {
    this.api.ignoreContract(id).subscribe({
      next: () => {
        this.loadContracts();
        this.loadStats();
      },
      error: (err) =>
        this.logger.error("Failed to ignore contract", err, { contractId: id }),
    });
  }

  formatSchema(schema: any): string {
    if (!schema) return "unknown";
    if (schema.type === "object" && schema.properties) {
      const props = Object.keys(schema.properties).join(", ");
      return `{ ${props} }`;
    }
    if (schema.type === "array" && schema.items) {
      return `${this.formatSchema(schema.items)}[]`;
    }
    return schema.type || "unknown";
  }
}
