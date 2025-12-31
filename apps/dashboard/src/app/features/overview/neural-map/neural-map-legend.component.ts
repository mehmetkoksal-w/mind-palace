import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-neural-map-legend',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="legend">
      <div class="legend-section">
        <span class="section-title">Nodes</span>
        <div class="legend-item">
          <svg width="16" height="16" viewBox="0 0 16 16">
            <circle cx="8" cy="8" r="6" fill="none" stroke="#9d4edd" stroke-width="2"/>
          </svg>
          <span>Room</span>
        </div>
        <div class="legend-item">
          <svg width="16" height="16" viewBox="0 0 16 16">
            <path d="M8,2 L14,8 L8,14 L2,8 Z" fill="#fbbf24"/>
          </svg>
          <span>Idea</span>
        </div>
        <div class="legend-item">
          <svg width="16" height="16" viewBox="0 0 16 16">
            <polygon points="8,2 13,5 13,11 8,14 3,11 3,5" fill="#4ade80"/>
          </svg>
          <span>Decision</span>
        </div>
        <div class="legend-item">
          <svg width="16" height="16" viewBox="0 0 16 16">
            <rect x="3" y="5" width="10" height="6" rx="2" fill="#00b4d8"/>
          </svg>
          <span>Learning</span>
        </div>
      </div>

      <div class="legend-section">
        <span class="section-title">Links</span>
        <div class="legend-item">
          <svg width="24" height="12" viewBox="0 0 24 12">
            <line x1="2" y1="6" x2="22" y2="6" stroke="#4ade80" stroke-width="2"/>
            <polygon points="22,6 17,3 17,9" fill="#4ade80"/>
          </svg>
          <span>Supports</span>
        </div>
        <div class="legend-item">
          <svg width="24" height="12" viewBox="0 0 24 12">
            <line x1="2" y1="6" x2="22" y2="6" stroke="#f87171" stroke-width="2" stroke-dasharray="3,3"/>
          </svg>
          <span>Contradicts</span>
        </div>
        <div class="legend-item">
          <svg width="24" height="12" viewBox="0 0 24 12">
            <line x1="2" y1="6" x2="22" y2="6" stroke="#60a5fa" stroke-width="1.5"/>
            <polygon points="22,6 17,3 17,9" fill="#60a5fa"/>
          </svg>
          <span>References</span>
        </div>
        <div class="legend-item">
          <svg width="24" height="12" viewBox="0 0 24 12">
            <line x1="2" y1="6" x2="22" y2="6" stroke="#f472b6" stroke-width="2"/>
            <polygon points="22,6 17,3 17,9" fill="#f472b6"/>
          </svg>
          <span>Depends</span>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .legend {
      background: rgba(26, 26, 46, 0.95);
      border-radius: 6px;
      border: 1px solid #2d2d44;
      padding: 10px;
      min-width: 120px;
    }

    .legend-section {
      margin-bottom: 8px;
    }

    .legend-section:last-child {
      margin-bottom: 0;
    }

    .section-title {
      display: block;
      font-size: 0.6rem;
      font-weight: 600;
      text-transform: uppercase;
      color: #64748b;
      margin-bottom: 6px;
      letter-spacing: 0.5px;
    }

    .legend-item {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 0.7rem;
      color: #94a3b8;
      margin-bottom: 4px;
    }

    .legend-item:last-child {
      margin-bottom: 0;
    }

    .legend-item svg {
      flex-shrink: 0;
    }
  `]
})
export class NeuralMapLegendComponent {}
