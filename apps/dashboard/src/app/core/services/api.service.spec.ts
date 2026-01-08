import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { TestBed } from "@angular/core/testing";
import { provideHttpClient } from "@angular/common/http";
import {
  HttpTestingController,
  provideHttpClientTesting,
} from "@angular/common/http/testing";
import { HttpErrorResponse } from "@angular/common/http";
import {
  ApiService,
  Stats,
  Session,
  Learning,
  ActiveAgent,
} from "./api.service";

describe("ApiService", () => {
  let service: ApiService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [ApiService, provideHttpClient(), provideHttpClientTesting()],
    });

    service = TestBed.inject(ApiService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  it("should be created", () => {
    expect(service).toBeTruthy();
  });

  describe("Health & Stats", () => {
    it("should fetch health status successfully", () => {
      // Arrange
      const mockHealth = {
        status: "healthy",
        timestamp: "2025-01-01T00:00:00Z",
      };

      // Act
      service.getHealth().subscribe((health) => {
        // Assert
        expect(health).toEqual(mockHealth);
        expect(health.status).toBe("healthy");
      });

      const req = httpMock.expectOne("/api/health");
      expect(req.request.method).toBe("GET");
      req.flush(mockHealth);
    });

    it("should handle health check errors", () => {
      // Arrange
      const errorMessage = "Server error";
      let errorResponse: HttpErrorResponse | undefined;

      // Act
      service.getHealth().subscribe({
        next: () => fail("should have failed"),
        error: (error) => (errorResponse = error),
      });

      // Assert
      const req = httpMock.expectOne("/api/health");
      req.flush(errorMessage, {
        status: 500,
        statusText: "Internal Server Error",
      });

      expect(errorResponse).toBeDefined();
      expect(errorResponse?.status).toBe(500);
    });

    it("should fetch stats successfully", () => {
      // Arrange
      const mockStats: Stats = {
        sessions: { total: 10, active: 2 },
        learnings: 100,
        filesTracked: 50,
        rooms: 5,
        corridor: {
          learningCount: 25,
          linkedWorkspaces: 3,
          averageConfidence: 0.85,
        },
      };

      // Act
      service.getStats().subscribe((stats) => {
        // Assert
        expect(stats).toEqual(mockStats);
        expect(stats.sessions.total).toBe(10);
        expect(stats.sessions.active).toBe(2);
        expect(stats.corridor?.learningCount).toBe(25);
      });

      const req = httpMock.expectOne("/api/stats");
      expect(req.request.method).toBe("GET");
      req.flush(mockStats);
    });

    it("should handle stats fetch errors", () => {
      // Arrange
      let errorResponse: HttpErrorResponse | undefined;

      // Act
      service.getStats().subscribe({
        next: () => fail("should have failed"),
        error: (error) => (errorResponse = error),
      });

      // Assert
      const req = httpMock.expectOne("/api/stats");
      req.flush("Not Found", { status: 404, statusText: "Not Found" });

      expect(errorResponse).toBeDefined();
      expect(errorResponse?.status).toBe(404);
    });
  });

  describe("Sessions", () => {
    it("should fetch sessions with default params", () => {
      // Arrange
      const mockResponse = { sessions: [], count: 0 };

      // Act
      service.getSessions().subscribe((response) => {
        // Assert
        expect(response).toEqual(mockResponse);
      });

      const req = httpMock.expectOne((r) => r.url === "/api/sessions");
      expect(req.request.params.get("active")).toBe("false");
      expect(req.request.params.get("limit")).toBe("50");
      req.flush(mockResponse);
    });

    it("should fetch active sessions only", () => {
      // Arrange
      const mockSessions: Session[] = [
        {
          id: "sess-1",
          agentType: "code-editor",
          agentId: "agent-1",
          goal: "Implement feature",
          startedAt: "2025-01-01T10:00:00Z",
          lastActivity: "2025-01-01T11:00:00Z",
          state: "active",
          summary: "Working on feature",
        },
      ];

      // Act
      service.getSessions(true, 10).subscribe((response) => {
        // Assert
        expect(response.sessions).toEqual(mockSessions);
        expect(response.count).toBe(1);
      });

      const req = httpMock.expectOne((r) => r.url === "/api/sessions");
      expect(req.request.params.get("active")).toBe("true");
      expect(req.request.params.get("limit")).toBe("10");
      req.flush({ sessions: mockSessions, count: 1 });
    });

    it("should fetch single session by id", () => {
      // Arrange
      const sessionId = "test-123";
      const mockSession: Session = {
        id: sessionId,
        agentType: "code-editor",
        agentId: "agent-1",
        goal: "Test goal",
        startedAt: "2025-01-01T10:00:00Z",
        lastActivity: "2025-01-01T11:00:00Z",
        state: "active",
        summary: "Test summary",
      };

      // Act
      service.getSession(sessionId).subscribe((response) => {
        // Assert
        expect(response.session).toEqual(mockSession);
        expect(response.session.id).toBe(sessionId);
        expect(response.activities).toEqual([]);
      });

      const req = httpMock.expectOne(`/api/sessions/${sessionId}`);
      expect(req.request.method).toBe("GET");
      req.flush({ session: mockSession, activities: [] });
    });

    it("should handle session not found error", () => {
      // Arrange
      const sessionId = "non-existent";
      let errorResponse: HttpErrorResponse | undefined;

      // Act
      service.getSession(sessionId).subscribe({
        next: () => fail("should have failed"),
        error: (error) => (errorResponse = error),
      });

      // Assert
      const req = httpMock.expectOne(`/api/sessions/${sessionId}`);
      req.flush("Session not found", { status: 404, statusText: "Not Found" });

      expect(errorResponse).toBeDefined();
      expect(errorResponse?.status).toBe(404);
    });
  });

  describe("Learnings", () => {
    it("should fetch learnings with default params", () => {
      // Arrange
      const mockResponse = { learnings: [], count: 0 };

      // Act
      service.getLearnings().subscribe((response) => {
        // Assert
        expect(response).toEqual(mockResponse);
        expect(response.count).toBe(0);
      });

      const req = httpMock.expectOne((r) => r.url === "/api/learnings");
      expect(req.request.params.get("limit")).toBe("50");
      expect(req.request.params.has("scope")).toBe(false);
      expect(req.request.params.has("query")).toBe(false);
      req.flush(mockResponse);
    });

    it("should fetch learnings with scope and query", () => {
      // Arrange
      const mockLearnings: Learning[] = [
        {
          id: "learn-1",
          sessionId: "sess-1",
          scope: "workspace",
          scopePath: "/project",
          content: "Test learning",
          confidence: 0.9,
          source: "code-analysis",
          createdAt: "2025-01-01T10:00:00Z",
          lastUsed: "2025-01-01T11:00:00Z",
          useCount: 5,
        },
      ];

      // Act
      service.getLearnings("workspace", "test", 10).subscribe((response) => {
        // Assert
        expect(response.learnings).toEqual(mockLearnings);
        expect(response.count).toBe(1);
      });

      const req = httpMock.expectOne((r) => r.url === "/api/learnings");
      expect(req.request.params.get("scope")).toBe("workspace");
      expect(req.request.params.get("query")).toBe("test");
      expect(req.request.params.get("limit")).toBe("10");
      req.flush({ learnings: mockLearnings, count: 1 });
    });
  });

  describe("Active Agents", () => {
    it("should fetch active agents successfully", () => {
      // Arrange
      const mockAgents: ActiveAgent[] = [
        {
          agentId: "agent-1",
          agentType: "code-editor",
          sessionId: "sess-1",
          heartbeat: "2025-01-01T12:00:00Z",
          currentFile: "/src/app.ts",
        },
      ];

      // Act
      service.getActiveAgents().subscribe((response) => {
        // Assert
        expect(response.agents).toEqual(mockAgents);
        expect(response.count).toBe(1);
        expect(response.agents[0].agentType).toBe("code-editor");
      });

      const req = httpMock.expectOne("/api/agents");
      expect(req.request.method).toBe("GET");
      req.flush({ agents: mockAgents, count: 1 });
    });

    it("should handle no active agents", () => {
      // Arrange
      const mockResponse = { agents: [], count: 0 };

      // Act
      service.getActiveAgents().subscribe((response) => {
        // Assert
        expect(response.agents).toEqual([]);
        expect(response.count).toBe(0);
      });

      const req = httpMock.expectOne("/api/agents");
      req.flush(mockResponse);
    });
  });

  describe("Search", () => {
    it("should perform search with query", () => {
      const query = "test search";

      service.search(query).subscribe((response) => {
        expect(response).toBeDefined();
      });

      const req = httpMock.expectOne((r) => r.url === "/api/search");
      expect(req.request.params.get("q")).toBe(query);
      expect(req.request.params.get("limit")).toBe("20");
      req.flush({ results: [] });
    });
  });

  describe("Rooms", () => {
    it("should fetch rooms", () => {
      const mockResponse = { rooms: [], count: 0 };

      service.getRooms().subscribe((response) => {
        expect(response).toEqual(mockResponse);
      });

      const req = httpMock.expectOne("/api/rooms");
      expect(req.request.method).toBe("GET");
      req.flush(mockResponse);
    });
  });

  describe("Error Handling", () => {
    it("should handle HTTP errors", () => {
      service.getHealth().subscribe({
        error: (error) => {
          expect(error.status).toBe(500);
        },
      });

      const req = httpMock.expectOne("/api/health");
      req.flush("Server error", {
        status: 500,
        statusText: "Internal Server Error",
      });
    });
  });
});
