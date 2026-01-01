import { Component, inject, OnInit, OnDestroy, signal } from '@angular/core';

import { Router } from '@angular/router';
import { WebSocketService } from '../services/websocket.service';
import { Subscription } from 'rxjs';
import { filter } from 'rxjs/operators';

interface ContradictionDetail {
  conflictingId: string;
  conflictingKind: string;
  conflictingContent: string;
  confidence: number;
  type: string;
  explanation: string;
  autoLinked: boolean;
}

interface ContradictionAlert {
  id: string;
  recordId: string;
  recordKind: string;
  recordContent: string;
  contradictions: ContradictionDetail[];
  timestamp: Date;
}

@Component({
    selector: 'app-contradiction-alert',
    imports: [],
    template: `
    <div class="alerts-container">
      @for (alert of alerts(); track alert.id) {
        <div class="alert-toast" [class.entering]="!alert.id.includes('exiting')" @fadeSlide>
          <div class="alert-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
              <line x1="12" y1="9" x2="12" y2="13"/>
              <line x1="12" y1="17" x2="12.01" y2="17"/>
            </svg>
          </div>
          <div class="alert-content">
            <div class="alert-header">
              <span class="alert-title">Contradiction Detected</span>
              <span class="alert-badge">{{ alert.contradictions.length }} conflict{{ alert.contradictions.length > 1 ? 's' : '' }}</span>
            </div>
            <div class="alert-body">
              <p class="alert-record">
                <span class="record-kind">{{ alert.recordKind }}</span>
                {{ truncate(alert.recordContent, 80) }}
              </p>
              @if (alert.contradictions[0]; as c) {
                <p class="alert-conflict">
                  Conflicts with: <span class="conflict-kind">{{ c.conflictingKind }}</span>
                  <span class="conflict-confidence">({{ (c.confidence * 100).toFixed(0) }}%)</span>
                </p>
              }
            </div>
            <div class="alert-actions">
              <button class="action-btn view-btn" (click)="viewContradiction(alert)">
                View Details
              </button>
              <button class="action-btn dismiss-btn" (click)="dismissAlert(alert.id)">
                Dismiss
              </button>
            </div>
          </div>
          <button class="close-btn" (click)="dismissAlert(alert.id)">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"/>
              <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>
      }
    </div>
  `,
    styles: [`
    .alerts-container {
      position: fixed;
      bottom: 1.5rem;
      right: 1.5rem;
      display: flex;
      flex-direction: column-reverse;
      gap: 0.75rem;
      z-index: 1000;
      max-width: 400px;
    }

    .alert-toast {
      display: flex;
      gap: 0.75rem;
      background: linear-gradient(135deg, #2a1a1a 0%, #1a1a2e 100%);
      border: 1px solid #ef4444;
      border-radius: 12px;
      padding: 1rem;
      box-shadow: 0 8px 32px rgba(239, 68, 68, 0.2);
      animation: slideIn 0.3s ease-out;
    }

    @keyframes slideIn {
      from {
        transform: translateX(100%);
        opacity: 0;
      }
      to {
        transform: translateX(0);
        opacity: 1;
      }
    }

    .alert-icon {
      flex-shrink: 0;
      width: 40px;
      height: 40px;
      border-radius: 10px;
      background: rgba(239, 68, 68, 0.15);
      display: flex;
      align-items: center;
      justify-content: center;
    }

    .alert-icon svg {
      width: 22px;
      height: 22px;
      color: #ef4444;
    }

    .alert-content {
      flex: 1;
      min-width: 0;
    }

    .alert-header {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      margin-bottom: 0.5rem;
    }

    .alert-title {
      font-weight: 600;
      color: #ef4444;
      font-size: 0.9rem;
    }

    .alert-badge {
      font-size: 0.65rem;
      padding: 0.15rem 0.4rem;
      border-radius: 4px;
      background: rgba(239, 68, 68, 0.2);
      color: #fca5a5;
      text-transform: uppercase;
    }

    .alert-body {
      margin-bottom: 0.75rem;
    }

    .alert-record {
      color: #e2e8f0;
      font-size: 0.85rem;
      margin: 0 0 0.25rem 0;
      line-height: 1.4;
    }

    .record-kind {
      display: inline-block;
      font-size: 0.65rem;
      padding: 0.1rem 0.3rem;
      border-radius: 3px;
      background: #3d3d54;
      color: #94a3b8;
      text-transform: uppercase;
      margin-right: 0.35rem;
      vertical-align: middle;
    }

    .alert-conflict {
      color: #94a3b8;
      font-size: 0.75rem;
      margin: 0;
    }

    .conflict-kind {
      color: #fbbf24;
    }

    .conflict-confidence {
      color: #64748b;
    }

    .alert-actions {
      display: flex;
      gap: 0.5rem;
    }

    .action-btn {
      padding: 0.35rem 0.75rem;
      border-radius: 6px;
      font-size: 0.75rem;
      font-weight: 500;
      cursor: pointer;
      transition: all 0.2s ease;
      border: none;
    }

    .view-btn {
      background: rgba(239, 68, 68, 0.2);
      color: #fca5a5;
    }

    .view-btn:hover {
      background: rgba(239, 68, 68, 0.3);
    }

    .dismiss-btn {
      background: transparent;
      color: #64748b;
    }

    .dismiss-btn:hover {
      background: rgba(255, 255, 255, 0.05);
      color: #94a3b8;
    }

    .close-btn {
      flex-shrink: 0;
      width: 24px;
      height: 24px;
      padding: 0;
      background: transparent;
      border: none;
      cursor: pointer;
      color: #64748b;
      transition: color 0.2s ease;
    }

    .close-btn:hover {
      color: #94a3b8;
    }

    .close-btn svg {
      width: 16px;
      height: 16px;
    }

    @media (max-width: 480px) {
      .alerts-container {
        left: 1rem;
        right: 1rem;
        max-width: none;
      }
    }
  `]
})
export class ContradictionAlertComponent implements OnInit, OnDestroy {
  private ws = inject(WebSocketService);
  private router = inject(Router);
  private subscription?: Subscription;
  private alertIdCounter = 0;

  alerts = signal<ContradictionAlert[]>([]);

  ngOnInit() {
    this.subscription = this.ws.events.pipe(
      filter(event => event.type === 'contradiction_detected')
    ).subscribe(event => {
      this.addAlert(event.data);
    });
  }

  ngOnDestroy() {
    this.subscription?.unsubscribe();
  }

  private addAlert(payload: any) {
    if (!payload) return;

    const alert: ContradictionAlert = {
      id: `alert-${++this.alertIdCounter}`,
      recordId: payload.recordId || '',
      recordKind: payload.recordKind || 'unknown',
      recordContent: payload.recordContent || '',
      contradictions: payload.contradictions || [],
      timestamp: new Date()
    };

    this.alerts.update(current => [...current, alert]);

    // Auto-dismiss after 15 seconds
    setTimeout(() => {
      this.dismissAlert(alert.id);
    }, 15000);
  }

  dismissAlert(id: string) {
    this.alerts.update(current => current.filter(a => a.id !== id));
  }

  viewContradiction(alert: ContradictionAlert) {
    this.dismissAlert(alert.id);
    // Navigate to the insights page with contradiction filter
    this.router.navigate(['/insights'], {
      queryParams: {
        tab: 'contradictions',
        recordId: alert.recordId
      }
    });
  }

  truncate(text: string, maxLength: number): string {
    if (!text) return '';
    return text.length > maxLength ? text.substring(0, maxLength) + '...' : text;
  }
}
