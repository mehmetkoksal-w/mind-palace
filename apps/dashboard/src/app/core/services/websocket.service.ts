import { Injectable, signal } from "@angular/core";
import { Subject, Observable } from "rxjs";

export interface WebSocketEvent {
  type: string;
  data?: any;
  timestamp?: string;
}

@Injectable({ providedIn: "root" })
export class WebSocketService {
  private socket: WebSocket | null = null;
  private eventsSubject = new Subject<WebSocketEvent>();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 2000;

  connected = signal(false);

  /**
   * Connect to the WebSocket server
   */
  connect(): void {
    if (this.socket?.readyState === WebSocket.OPEN) {
      return;
    }

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/api/ws`;

    try {
      this.socket = new WebSocket(wsUrl);

      this.socket.onopen = () => {
        this.connected.set(true);
        this.reconnectAttempts = 0;
      };

      this.socket.onclose = (event) => {
        this.connected.set(false);
        this.attemptReconnect();
      };

      this.socket.onerror = (error) => {
        console.error("[WebSocket] Error:", error);
        this.connected.set(false);
      };

      this.socket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          this.eventsSubject.next(data);
        } catch (e) {
          console.error("[WebSocket] Failed to parse message:", e);
        }
      };
    } catch (e) {
      console.error("[WebSocket] Failed to connect:", e);
      this.attemptReconnect();
    }
  }

  /**
   * Disconnect from the WebSocket server
   */
  disconnect(): void {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
    this.connected.set(false);
  }

  /**
   * Get observable of WebSocket events
   */
  get events(): Observable<WebSocketEvent> {
    return this.eventsSubject.asObservable();
  }

  /**
   * Send a message to the server
   */
  send(message: any): void {
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(message));
    } else {
      console.warn("[WebSocket] Cannot send message - not connected");
    }
  }

  /**
   * Attempt to reconnect with exponential backoff
   */
  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    setTimeout(() => {
      this.connect();
    }, delay);
  }
}
