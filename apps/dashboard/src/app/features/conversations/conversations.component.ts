import { Component, inject, OnInit, signal } from "@angular/core";

import { FormsModule } from "@angular/forms";
import { ActivatedRoute, Router } from "@angular/router";
import { ApiService } from "../../core/services/api.service";
import { LoggerService } from "../../core/services/logger.service";

interface ConversationItem {
  id: string;
  sessionId: string;
  agentType: string;
  summary: string;
  messageCount: number;
  duration: string;
  createdAt: string;
}

interface Message {
  role: string;
  content: string;
  timestamp: string;
}

interface ExtractedRecord {
  id: string;
  kind: string;
  content: string;
  status?: string;
  confidence?: number;
  scope: string;
}

interface ConversationDetail {
  id: string;
  agentType: string;
  summary: string;
  messages: Message[];
  sessionId: string;
  createdAt: string;
  duration?: string;
  extracted?: ExtractedRecord[];
}

interface TimelineEvent {
  timestamp: string;
  type: string;
  role?: string;
  content: string;
  recordId?: string;
  recordKind?: string;
}

@Component({
  selector: "app-conversations",
  imports: [FormsModule],
  template: `
    <div class="conversations-container">
      <!-- Header -->
      <div class="page-header">
        <div class="header-content">
          <h1>
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z"
              />
            </svg>
            Conversations
          </h1>
          <p class="subtitle">
            Review past AI conversations and extracted knowledge
          </p>
        </div>
        <div class="header-actions">
          <div class="search-box">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <circle cx="11" cy="11" r="8" />
              <path d="M21 21l-4.35-4.35" />
            </svg>
            <input
              type="text"
              [(ngModel)]="searchQuery"
              (input)="onSearch()"
              placeholder="Search conversations..."
            />
          </div>
        </div>
      </div>

      <!-- Main Content -->
      <div class="content-grid" [class.detail-open]="selectedConversation()">
        <!-- Conversation List -->
        <div class="conversation-list">
          @if (loading()) {
          <div class="loading-state">
            <div class="spinner"></div>
            <span>Loading conversations...</span>
          </div>
          } @else if (conversations().length === 0) {
          <div class="empty-state">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.5"
            >
              <path
                d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z"
              />
            </svg>
            <h3>No conversations yet</h3>
            <p>AI conversations will appear here once recorded</p>
          </div>
          } @else { @for (conv of conversations(); track conv.id) {
          <div
            class="conversation-card"
            [class.selected]="selectedConversation()?.id === conv.id"
            (click)="selectConversation(conv.id)"
          >
            <div class="conv-header">
              <span class="agent-badge" [attr.data-agent]="conv.agentType">
                {{ conv.agentType }}
              </span>
              <span class="conv-date">{{ formatDate(conv.createdAt) }}</span>
            </div>
            <h3 class="conv-summary">{{ conv.summary }}</h3>
            <div class="conv-meta">
              <span class="meta-item">
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path
                    d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z"
                  />
                </svg>
                {{ conv.messageCount }} messages
              </span>
              @if (conv.duration) {
              <span class="meta-item">
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 6v6l4 2" />
                </svg>
                {{ conv.duration }}
              </span>
              }
            </div>
          </div>
          } }
        </div>

        <!-- Conversation Detail -->
        @if (selectedConversation()) {
        <div class="conversation-detail">
          <div class="detail-header">
            <button class="back-btn" (click)="closeDetail()">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <path d="M19 12H5M12 19l-7-7 7-7" />
              </svg>
            </button>
            <div class="detail-title">
              <h2>{{ selectedConversation()!.summary }}</h2>
              <div class="detail-meta">
                <span
                  class="agent-badge"
                  [attr.data-agent]="selectedConversation()!.agentType"
                >
                  {{ selectedConversation()!.agentType }}
                </span>
                <span>{{ formatDate(selectedConversation()!.createdAt) }}</span>
                @if (selectedConversation()!.duration) {
                <span>{{ selectedConversation()!.duration }}</span>
                }
              </div>
            </div>
            <div class="view-toggle">
              <button
                [class.active]="viewMode() === 'chat'"
                (click)="viewMode.set('chat')"
              >
                Chat
              </button>
              <button
                [class.active]="viewMode() === 'timeline'"
                (click)="loadTimeline()"
              >
                Timeline
              </button>
            </div>
          </div>

          <!-- Chat View -->
          @if (viewMode() === 'chat') {
          <div class="chat-view">
            <div class="messages-container">
              @for (msg of selectedConversation()!.messages; track $index) {
              <div class="message" [class]="msg.role">
                <div class="message-header">
                  <span class="role-label">{{ msg.role }}</span>
                  <span class="msg-time">{{ formatTime(msg.timestamp) }}</span>
                </div>
                <div class="message-content">{{ msg.content }}</div>
              </div>
              }
            </div>

            <!-- Extracted Records Sidebar -->
            @if (selectedConversation()!.extracted &&
            selectedConversation()!.extracted!.length > 0) {
            <div class="extracted-sidebar">
              <h3>Extracted Knowledge</h3>
              @for (record of selectedConversation()!.extracted; track
              record.id) {
              <div class="extracted-card" [attr.data-kind]="record.kind">
                <div class="record-header">
                  <span class="kind-badge">{{ record.kind }}</span>
                  @if (record.confidence) {
                  <span class="confidence"
                    >{{ (record.confidence * 100).toFixed(0) }}%</span
                  >
                  }
                </div>
                <p class="record-content">{{ record.content }}</p>
                <span class="record-id">{{ record.id }}</span>
              </div>
              }
            </div>
            }
          </div>
          }

          <!-- Timeline View -->
          @if (viewMode() === 'timeline') {
          <div class="timeline-view">
            @if (loadingTimeline()) {
            <div class="loading-state">
              <div class="spinner"></div>
              <span>Loading timeline...</span>
            </div>
            } @else {
            <div class="timeline">
              @for (event of timelineEvents(); track $index) {
              <div class="timeline-event" [attr.data-type]="event.type">
                <div class="event-marker">
                  @if (event.type === 'message') {
                  <svg
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <path
                      d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z"
                    />
                  </svg>
                  } @else {
                  <svg
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <path
                      d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
                    />
                  </svg>
                  }
                </div>
                <div class="event-content">
                  <div class="event-header">
                    @if (event.type === 'message') {
                    <span class="event-role">{{ event.role }}</span>
                    } @else {
                    <span
                      class="event-kind"
                      [attr.data-kind]="event.recordKind"
                    >
                      {{ event.recordKind }} extracted
                    </span>
                    }
                    <span class="event-time">{{
                      formatTime(event.timestamp)
                    }}</span>
                  </div>
                  <p class="event-text">{{ event.content }}</p>
                  @if (event.recordId) {
                  <span class="event-id">{{ event.recordId }}</span>
                  }
                </div>
              </div>
              }
            </div>
            }
          </div>
          }
        </div>
        }
      </div>
    </div>
  `,
  styles: [
    `
      .conversations-container {
        padding: 1.5rem;
        max-width: 1600px;
        margin: 0 auto;
      }

      .page-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 1.5rem;
        gap: 1rem;
        flex-wrap: wrap;
      }

      .header-content h1 {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        font-size: 1.5rem;
        font-weight: 600;
        color: #f1f5f9;
        margin: 0;
      }

      .header-content h1 svg {
        width: 24px;
        height: 24px;
        color: #8b5cf6;
      }

      .subtitle {
        color: #64748b;
        font-size: 0.9rem;
        margin: 0.25rem 0 0 0;
      }

      /* search-box styles in global styles.scss */

      .content-grid {
        display: grid;
        grid-template-columns: 1fr;
        gap: 1.5rem;
      }

      .content-grid.detail-open {
        grid-template-columns: 350px 1fr;
      }

      .conversation-list {
        display: flex;
        flex-direction: column;
        gap: 0.75rem;
        max-height: calc(100vh - 200px);
        overflow-y: auto;
      }

      .conversation-card {
        background: linear-gradient(135deg, #1e293b 0%, #0f172a 100%);
        border: 1px solid #334155;
        border-radius: 12px;
        padding: 1rem;
        cursor: pointer;
        transition: all 0.2s ease;
      }

      .conversation-card:hover {
        border-color: #8b5cf6;
        transform: translateX(4px);
      }

      .conversation-card.selected {
        border-color: #8b5cf6;
        background: linear-gradient(135deg, #1e1b4b 0%, #0f172a 100%);
      }

      .conv-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 0.5rem;
      }

      .agent-badge {
        font-size: 0.7rem;
        padding: 0.2rem 0.5rem;
        border-radius: 4px;
        background: #334155;
        color: #94a3b8;
        text-transform: uppercase;
      }

      .agent-badge[data-agent="claude-code"] {
        background: rgba(139, 92, 246, 0.2);
        color: #a78bfa;
      }

      .agent-badge[data-agent="cursor"] {
        background: rgba(34, 211, 238, 0.2);
        color: #22d3ee;
      }

      .conv-date {
        font-size: 0.75rem;
        color: #64748b;
      }

      .conv-summary {
        font-size: 0.95rem;
        font-weight: 500;
        color: #e2e8f0;
        margin: 0 0 0.75rem 0;
        line-height: 1.4;
      }

      .conv-meta {
        display: flex;
        gap: 1rem;
      }

      /* meta-item styles in global styles.scss */

      /* Detail View - loading-state, empty-state, spinner are in global styles */
      .conversation-detail {
        background: linear-gradient(135deg, #1e293b 0%, #0f172a 100%);
        border: 1px solid #334155;
        border-radius: 12px;
        display: flex;
        flex-direction: column;
        max-height: calc(100vh - 200px);
      }

      .detail-header {
        display: flex;
        align-items: center;
        gap: 1rem;
        padding: 1rem;
        border-bottom: 1px solid #334155;
      }

      .back-btn {
        background: transparent;
        border: none;
        color: #64748b;
        cursor: pointer;
        padding: 0.5rem;
        border-radius: 6px;
        transition: all 0.2s;
      }

      .back-btn:hover {
        background: #334155;
        color: #f1f5f9;
      }

      .back-btn svg {
        width: 20px;
        height: 20px;
      }

      .detail-title {
        flex: 1;
      }

      .detail-title h2 {
        font-size: 1rem;
        font-weight: 600;
        color: #f1f5f9;
        margin: 0 0 0.25rem 0;
      }

      .detail-meta {
        display: flex;
        gap: 0.75rem;
        font-size: 0.8rem;
        color: #64748b;
      }

      .view-toggle {
        display: flex;
        background: #0f172a;
        border-radius: 6px;
        padding: 2px;
      }

      .view-toggle button {
        padding: 0.4rem 0.75rem;
        background: transparent;
        border: none;
        color: #64748b;
        font-size: 0.8rem;
        cursor: pointer;
        border-radius: 4px;
        transition: all 0.2s;
      }

      .view-toggle button.active {
        background: #8b5cf6;
        color: white;
      }

      .chat-view {
        display: grid;
        grid-template-columns: 1fr 280px;
        flex: 1;
        overflow: hidden;
      }

      .messages-container {
        padding: 1rem;
        overflow-y: auto;
        display: flex;
        flex-direction: column;
        gap: 1rem;
      }

      .message {
        max-width: 80%;
        padding: 0.75rem 1rem;
        border-radius: 12px;
      }

      .message.user {
        background: #334155;
        align-self: flex-end;
        border-bottom-right-radius: 4px;
      }

      .message.assistant {
        background: linear-gradient(135deg, #1e1b4b 0%, #312e81 100%);
        align-self: flex-start;
        border-bottom-left-radius: 4px;
      }

      .message.system {
        background: rgba(251, 191, 36, 0.1);
        border-left: 3px solid #fbbf24;
        align-self: center;
        max-width: 90%;
      }

      .message-header {
        display: flex;
        justify-content: space-between;
        margin-bottom: 0.35rem;
      }

      .role-label {
        font-size: 0.7rem;
        text-transform: uppercase;
        color: #8b5cf6;
        font-weight: 600;
      }

      .message.user .role-label {
        color: #22d3ee;
      }

      .msg-time {
        font-size: 0.7rem;
        color: #64748b;
      }

      .message-content {
        color: #e2e8f0;
        font-size: 0.9rem;
        line-height: 1.5;
        white-space: pre-wrap;
        word-wrap: break-word;
      }

      .extracted-sidebar {
        padding: 1rem;
        border-left: 1px solid #334155;
        overflow-y: auto;
        background: rgba(0, 0, 0, 0.2);
      }

      .extracted-sidebar h3 {
        font-size: 0.85rem;
        color: #94a3b8;
        margin: 0 0 1rem 0;
        text-transform: uppercase;
      }

      .extracted-card {
        background: #1e293b;
        border: 1px solid #334155;
        border-radius: 8px;
        padding: 0.75rem;
        margin-bottom: 0.75rem;
      }

      .extracted-card[data-kind="idea"] {
        border-left: 3px solid #fbbf24;
      }

      .extracted-card[data-kind="decision"] {
        border-left: 3px solid #8b5cf6;
      }

      .extracted-card[data-kind="learning"] {
        border-left: 3px solid #10b981;
      }

      .record-header {
        display: flex;
        justify-content: space-between;
        margin-bottom: 0.5rem;
      }

      .kind-badge {
        font-size: 0.65rem;
        padding: 0.15rem 0.4rem;
        border-radius: 4px;
        background: #334155;
        color: #94a3b8;
        text-transform: uppercase;
      }

      .confidence {
        font-size: 0.7rem;
        color: #10b981;
      }

      .record-content {
        font-size: 0.8rem;
        color: #cbd5e1;
        margin: 0 0 0.5rem 0;
        line-height: 1.4;
      }

      .record-id {
        font-size: 0.65rem;
        color: #475569;
        font-family: monospace;
      }

      /* Timeline View */
      .timeline-view {
        flex: 1;
        overflow-y: auto;
        padding: 1rem;
      }

      .timeline {
        position: relative;
        padding-left: 2rem;
      }

      .timeline::before {
        content: "";
        position: absolute;
        left: 11px;
        top: 0;
        bottom: 0;
        width: 2px;
        background: #334155;
      }

      .timeline-event {
        position: relative;
        padding-bottom: 1.5rem;
      }

      .event-marker {
        position: absolute;
        left: -2rem;
        width: 24px;
        height: 24px;
        border-radius: 50%;
        background: #1e293b;
        border: 2px solid #334155;
        display: flex;
        align-items: center;
        justify-content: center;
      }

      .event-marker svg {
        width: 12px;
        height: 12px;
        color: #64748b;
      }

      .timeline-event[data-type="message"] .event-marker {
        border-color: #8b5cf6;
      }

      .timeline-event[data-type="message"] .event-marker svg {
        color: #8b5cf6;
      }

      .timeline-event[data-type="extraction"] .event-marker {
        border-color: #10b981;
        background: rgba(16, 185, 129, 0.1);
      }

      .timeline-event[data-type="extraction"] .event-marker svg {
        color: #10b981;
      }

      .event-content {
        background: #1e293b;
        border: 1px solid #334155;
        border-radius: 8px;
        padding: 0.75rem 1rem;
      }

      .event-header {
        display: flex;
        justify-content: space-between;
        margin-bottom: 0.5rem;
      }

      .event-role {
        font-size: 0.75rem;
        font-weight: 600;
        color: #8b5cf6;
        text-transform: uppercase;
      }

      .event-kind {
        font-size: 0.75rem;
        font-weight: 600;
        color: #10b981;
      }

      .event-time {
        font-size: 0.7rem;
        color: #64748b;
      }

      .event-text {
        font-size: 0.85rem;
        color: #cbd5e1;
        margin: 0;
        line-height: 1.5;
      }

      .event-id {
        display: block;
        margin-top: 0.5rem;
        font-size: 0.65rem;
        color: #475569;
        font-family: monospace;
      }

      @media (max-width: 1024px) {
        .content-grid.detail-open {
          grid-template-columns: 1fr;
        }

        .conversation-list {
          display: none;
        }

        .content-grid.detail-open .conversation-list {
          display: none;
        }

        .chat-view {
          grid-template-columns: 1fr;
        }

        .extracted-sidebar {
          display: none;
        }
      }
    `,
  ],
})
export class ConversationsComponent implements OnInit {
  private api = inject(ApiService);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private readonly logger = inject(LoggerService).forContext(
    "ConversationsComponent"
  );

  conversations = signal<ConversationItem[]>([]);
  selectedConversation = signal<ConversationDetail | null>(null);
  timelineEvents = signal<TimelineEvent[]>([]);
  loading = signal(true);
  loadingTimeline = signal(false);
  viewMode = signal<"chat" | "timeline">("chat");
  searchQuery = "";

  ngOnInit() {
    this.loadConversations();

    // Check for ID in route params
    this.route.params.subscribe((params) => {
      if (params["id"]) {
        this.selectConversation(params["id"]);
      }
    });
  }

  loadConversations() {
    this.loading.set(true);
    this.api.getConversations({ timeline: true, limit: 50 }).subscribe({
      next: (response) => {
        this.conversations.set(response.conversations || []);
      },
      error: (error) => {
        this.logger.error("Failed to load conversations", error, {
          endpoint: "/api/conversations",
          limit: 50,
        });
      },
      complete: () => {
        this.loading.set(false);
      },
    });
  }

  selectConversation(id: string) {
    this.api.getConversation(id).subscribe({
      next: (detail) => {
        this.selectedConversation.set(detail);
        this.viewMode.set("chat");
        this.timelineEvents.set([]);
      },
      error: (error) => {
        this.logger.error("Failed to load conversation detail", error, {
          endpoint: `/api/conversations/${id}`,
          conversationId: id,
        });
      },
    });
  }

  loadTimeline() {
    const conv = this.selectedConversation();
    if (!conv) return;

    this.viewMode.set("timeline");

    if (this.timelineEvents().length > 0) return;

    this.loadingTimeline.set(true);
    this.api.getConversationTimeline(conv.id).subscribe({
      next: (response) => {
        this.timelineEvents.set(response.events || []);
      },
      error: (error) => {
        this.logger.error("Failed to load conversation timeline", error, {
          endpoint: `/api/conversations/${conv.id}/timeline`,
          conversationId: conv.id,
        });
      },
      complete: () => {
        this.loadingTimeline.set(false);
      },
    });
  }

  closeDetail() {
    this.selectedConversation.set(null);
    this.timelineEvents.set([]);
    this.router.navigate(["/conversations"]);
  }

  onSearch() {
    // Debounced search could be implemented here
    if (this.searchQuery.trim()) {
      this.searchConversationsQuery(this.searchQuery);
    } else {
      this.loadConversations();
    }
  }

  searchConversationsQuery(query: string) {
    this.loading.set(true);
    this.api.getConversations({ query, timeline: true }).subscribe({
      next: (response) => {
        this.conversations.set(response.conversations || []);
      },
      error: (error) => {
        this.logger.error("Failed to search conversations", error, {
          endpoint: "/api/conversations",
          query: query,
        });
      },
      complete: () => {
        this.loading.set(false);
      },
    });
  }

  formatDate(dateStr: string): string {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (days === 0) return "Today";
    if (days === 1) return "Yesterday";
    if (days < 7) return `${days} days ago`;

    return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
  }

  formatTime(dateStr: string): string {
    return new Date(dateStr).toLocaleTimeString("en-US", {
      hour: "2-digit",
      minute: "2-digit",
    });
  }
}
