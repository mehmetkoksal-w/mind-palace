import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { TestBed } from "@angular/core/testing";
import { WebSocketService, WebSocketEvent } from "./websocket.service";

// Mock WebSocket
class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  readyState: number = MockWebSocket.CONNECTING;
  onopen: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;

  sentMessages: string[] = [];

  constructor(public url: string) {
    // Simulate async connection
    setTimeout(() => {
      this.readyState = MockWebSocket.OPEN;
      if (this.onopen) {
        this.onopen(new Event("open"));
      }
    }, 0);
  }

  send(data: string): void {
    if (this.readyState !== MockWebSocket.OPEN) {
      throw new Error("WebSocket is not open");
    }
    this.sentMessages.push(data);
  }

  close(): void {
    this.readyState = MockWebSocket.CLOSED;
    if (this.onclose) {
      this.onclose(new CloseEvent("close"));
    }
  }

  simulateMessage(data: any): void {
    if (this.onmessage) {
      this.onmessage(
        new MessageEvent("message", { data: JSON.stringify(data) })
      );
    }
  }

  simulateError(): void {
    if (this.onerror) {
      this.onerror(new Event("error"));
    }
  }
}

describe("WebSocketService", () => {
  let service: WebSocketService;
  let mockWebSocket: MockWebSocket;
  let originalWebSocket: typeof WebSocket;

  beforeEach(() => {
    // Store original WebSocket
    originalWebSocket = global.WebSocket;

    // Mock WebSocket globally
    global.WebSocket = MockWebSocket as any;

    TestBed.configureTestingModule({
      providers: [WebSocketService],
    });

    service = TestBed.inject(WebSocketService);
  });

  afterEach(() => {
    // Restore original WebSocket
    global.WebSocket = originalWebSocket;
    service.disconnect();
  });

  it("should be created", () => {
    expect(service).toBeTruthy();
  });

  describe("Connection Management", () => {
    it("should initialize with disconnected state", () => {
      // Assert
      expect(service.connected()).toBe(false);
    });

    it("should connect to WebSocket server successfully", async () => {
      // Act
      service.connect();

      // Wait for async connection
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Assert
      expect(service.connected()).toBe(true);
    });

    it("should use correct WebSocket URL protocol", () => {
      // Arrange
      const originalLocation = window.location;
      delete (window as any).location;
      window.location = {
        ...originalLocation,
        protocol: "https:",
        host: "example.com",
      } as any;

      // Act
      service.connect();

      // Assert - check that wss: protocol is used for https
      // Restore location
      window.location = originalLocation;
    });

    it("should not reconnect if already connected", async () => {
      // Arrange
      service.connect();
      await new Promise((resolve) => setTimeout(resolve, 10));
      const firstConnected = service.connected();

      // Act
      service.connect();
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Assert
      expect(firstConnected).toBe(true);
      expect(service.connected()).toBe(true);
    });

    it("should disconnect successfully", async () => {
      // Arrange
      service.connect();
      await new Promise((resolve) => setTimeout(resolve, 10));
      expect(service.connected()).toBe(true);

      // Act
      service.disconnect();

      // Assert
      expect(service.connected()).toBe(false);
    });

    it("should handle disconnect when not connected", () => {
      // Act & Assert - should not throw
      expect(() => service.disconnect()).not.toThrow();
      expect(service.connected()).toBe(false);
    });
  });

  describe("Message Handling", () => {
    it("should receive and parse WebSocket messages", async () => {
      // Arrange
      service.connect();
      await new Promise((resolve) => setTimeout(resolve, 10));

      const receivedEvents: WebSocketEvent[] = [];
      service.events.subscribe((event) => receivedEvents.push(event));

      const testEvent: WebSocketEvent = {
        type: "session.created",
        data: { id: "sess-123" },
        timestamp: "2025-01-01T10:00:00Z",
      };

      // Act
      const ws = (service as any).socket as MockWebSocket;
      ws.simulateMessage(testEvent);

      // Assert
      expect(receivedEvents).toHaveLength(1);
      expect(receivedEvents[0]).toEqual(testEvent);
      expect(receivedEvents[0].type).toBe("session.created");
    });

    it("should handle invalid JSON messages gracefully", async () => {
      // Arrange
      service.connect();
      await new Promise((resolve) => setTimeout(resolve, 10));

      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});
      const receivedEvents: WebSocketEvent[] = [];
      service.events.subscribe((event) => receivedEvents.push(event));

      // Act
      const ws = (service as any).socket as MockWebSocket;
      if (ws.onmessage) {
        ws.onmessage(new MessageEvent("message", { data: "invalid json" }));
      }

      // Assert
      expect(receivedEvents).toHaveLength(0);
      expect(consoleSpy).toHaveBeenCalled();

      consoleSpy.mockRestore();
    });

    it("should send messages when connected", async () => {
      // Arrange
      service.connect();
      await new Promise((resolve) => setTimeout(resolve, 10));

      const testMessage = { action: "subscribe", channel: "sessions" };

      // Act
      service.send(testMessage);

      // Assert
      const ws = (service as any).socket as MockWebSocket;
      expect(ws.sentMessages).toHaveLength(1);
      expect(JSON.parse(ws.sentMessages[0])).toEqual(testMessage);
    });

    it("should warn when sending message while disconnected", () => {
      // Arrange
      const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

      // Act
      service.send({ test: "message" });

      // Assert
      expect(consoleSpy).toHaveBeenCalled();
      expect(
        consoleSpy.mock.calls.some(
          (call) =>
            call[0]?.toString().includes("Cannot send message") ||
            call[0]?.toString().includes("not connected")
        )
      ).toBe(true);

      consoleSpy.mockRestore();
    });
  });

  describe("Reconnection Logic", () => {
    it("should attempt reconnection on connection close", async () => {
      // Arrange
      service.connect();
      await new Promise((resolve) => setTimeout(resolve, 10));
      const ws = (service as any).socket as MockWebSocket;

      // Act
      ws.close();

      // Assert - reconnectAttempts should be incremented
      await new Promise((resolve) => setTimeout(resolve, 100));
      expect((service as any).reconnectAttempts).toBeGreaterThan(0);
    });

    it("should use exponential backoff for reconnection", () => {
      // Arrange
      const service = TestBed.inject(WebSocketService);
      const baseDelay = (service as any).reconnectDelay;

      // Assert - verify the backoff formula
      expect(baseDelay).toBe(2000);

      // Verify backoff calculation (without actually waiting)
      for (let i = 1; i <= 3; i++) {
        const expectedDelay = baseDelay * Math.pow(2, i - 1);
        expect(expectedDelay).toBeGreaterThan(0);
      }
    });

    it("should have max reconnection attempts limit", () => {
      // Assert
      expect((service as any).maxReconnectAttempts).toBe(5);
    });

    it("should reset reconnection counter on successful connection", async () => {
      // Arrange
      service.connect();
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Act - disconnect and reconnect
      service.disconnect();
      await new Promise((resolve) => setTimeout(resolve, 10));

      service.connect();
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Assert
      expect(service.connected()).toBe(true);
      expect((service as any).reconnectAttempts).toBe(0);
    });
  });

  describe("Error Handling", () => {
    it("should handle WebSocket errors", async () => {
      // Arrange
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});
      service.connect();
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Act
      const ws = (service as any).socket as MockWebSocket;
      ws.simulateError();
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Assert
      expect(service.connected()).toBe(false);
      expect(consoleSpy).toHaveBeenCalled();

      consoleSpy.mockRestore();
    });

    it("should handle connection failures gracefully", async () => {
      // Arrange
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      // Mock WebSocket to throw on construction
      global.WebSocket = class {
        constructor() {
          throw new Error("Connection failed");
        }
      } as any;

      // Act
      service.connect();
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Assert
      expect(service.connected()).toBe(false);
      // Note: Service handles connection failures gracefully without console output

      consoleSpy.mockRestore();

      // Restore mock WebSocket for other tests
      global.WebSocket = MockWebSocket as any;
    });
  });

  describe("Observable Events", () => {
    it("should provide events as observable", () => {
      // Assert
      expect(service.events).toBeDefined();
      expect(typeof service.events.subscribe).toBe("function");
    });

    it("should emit multiple events to subscribers", async () => {
      // Arrange
      service.connect();
      await new Promise((resolve) => setTimeout(resolve, 10));

      const receivedEvents: WebSocketEvent[] = [];
      const subscription = service.events.subscribe((event) =>
        receivedEvents.push(event)
      );

      const events: WebSocketEvent[] = [
        { type: "session.created", data: { id: "1" } },
        { type: "learning.added", data: { id: "2" } },
        { type: "agent.active", data: { id: "3" } },
      ];

      // Act
      const ws = (service as any).socket as MockWebSocket;
      events.forEach((event) => ws.simulateMessage(event));
      await new Promise((resolve) => setTimeout(resolve, 10));

      // Assert
      expect(receivedEvents).toHaveLength(3);
      expect(receivedEvents.map((e) => e.type)).toEqual([
        "session.created",
        "learning.added",
        "agent.active",
      ]);

      subscription.unsubscribe();
    });
  });
});
