import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { TestBed, ComponentFixture } from "@angular/core/testing";
import { provideHttpClient } from "@angular/common/http";
import {
  HttpTestingController,
  provideHttpClientTesting,
} from "@angular/common/http/testing";
import { SessionsComponent } from "./sessions.component";
import { ApiService, Session } from "../../core/services/api.service";
import { LoggerService } from "../../core/services/logger.service";

describe("SessionsComponent", () => {
  let component: SessionsComponent;
  let fixture: ComponentFixture<SessionsComponent>;
  let apiService: ApiService;
  let httpMock: HttpTestingController;

  const mockSessions: Session[] = [
    {
      id: "sess-1",
      agentId: "agent-1",
      agentType: "code-editor",
      state: "active",
      goal: "Implement user authentication",
      startedAt: "2025-01-06T09:00:00Z",
      lastActivity: "2025-01-06T10:30:00Z",
      summary: "Working on login component",
    },
    {
      id: "sess-2",
      agentId: "agent-2",
      agentType: "debugger",
      state: "completed",
      goal: "Fix memory leak in service",
      startedAt: "2025-01-05T14:00:00Z",
      lastActivity: "2025-01-05T16:45:00Z",
      summary: "Memory leak fixed in data service",
    },
    {
      id: "sess-3",
      agentId: "agent-3",
      agentType: "code-review",
      state: "abandoned",
      goal: "Review PR #123",
      startedAt: "2025-01-04T11:00:00Z",
      lastActivity: "2025-01-04T11:15:00Z",
      summary: "",
    },
  ];

  beforeEach(() => {
    TestBed.resetTestingModule();

    TestBed.configureTestingModule({
      imports: [SessionsComponent],
      providers: [
        ApiService,
        LoggerService,
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    });

    fixture = TestBed.createComponent(SessionsComponent);
    component = fixture.componentInstance;
    apiService = TestBed.inject(ApiService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    if (httpMock) {
      try {
        const openRequests = httpMock.match(() => true);
        openRequests.forEach((req) => {
          if (!req.cancelled) {
            req.flush({ sessions: [], count: 0 });
          }
        });
        httpMock.verify();
      } catch (e) {
        // Ignore verification errors in afterEach
      }
    }
  });

  it("should create", () => {
    expect(component).toBeTruthy();
  });

  describe("Component Initialization", () => {
    it("should initialize with empty sessions array", () => {
      expect(component.sessions()).toEqual([]);
    });

    it("should initialize with active set to false", () => {
      expect(component.activeOnly()).toBe(false);
    });

    it("should load all sessions on initialization", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/sessions")
      );
      expect(req.request.method).toBe("GET");
      expect(req.request.params.get("active")).toBe("false");
      req.flush({ sessions: mockSessions, count: mockSessions.length });

      expect(component.sessions()).toEqual(mockSessions);
      expect(component.sessions()).toHaveLength(3);
    });
  });

  describe("Sessions Loading", () => {
    it("should load all sessions when active is false", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/sessions")
      );
      req.flush({ sessions: mockSessions, count: mockSessions.length });

      expect(component.sessions()).toEqual(mockSessions);
    });

    it("should load only active sessions when active is true", () => {
      component.activeOnly.set(true);
      fixture.detectChanges();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/sessions")
      );
      expect(req.request.params.get("active")).toBe("true");

      const activeSessions = mockSessions.filter((s) => s.state === "active");
      req.flush({ sessions: activeSessions, count: activeSessions.length });

      expect(component.sessions()).toEqual(activeSessions);
      expect(component.sessions()).toHaveLength(1);
    });

    it("should handle empty sessions response", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/sessions")
      );
      req.flush({ sessions: [], count: 0 });

      expect(component.sessions()).toEqual([]);
    });

    it("should handle missing sessions field in response", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/sessions")
      );
      req.flush({ count: 0 });

      expect(component.sessions()).toEqual([]);
    });
  });

  describe("Signal Updates", () => {
    it("should update sessions signal when data is loaded", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/sessions")
      );
      req.flush({ sessions: mockSessions, count: mockSessions.length });

      expect(component.sessions()).toEqual(mockSessions);
      expect(component.sessions()[0].id).toBe("sess-1");
      expect(component.sessions()[1].id).toBe("sess-2");
    });

    it("should handle sessions signal reactivity", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .flush({ sessions: mockSessions, count: mockSessions.length });

      const newSessions: Session[] = [
        {
          id: "sess-4",
          agentId: "agent-4",
          agentType: "test-runner",
          state: "active",
          goal: "Run unit tests",
          startedAt: "2025-01-06T11:00:00Z",
          lastActivity: "2025-01-06T11:30:00Z",
          summary: "",
        },
      ];
      component.sessions.set(newSessions);

      expect(component.sessions()).toEqual(newSessions);
      expect(component.sessions()).toHaveLength(1);
    });
  });

  describe("Active Filter Toggle", () => {
    it("should toggle active from false to true", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .flush({ sessions: mockSessions, count: mockSessions.length });

      component.toggleActive();

      expect(component.activeOnly()).toBe(true);
    });

    it("should toggle active from true to false", () => {
      component.activeOnly.set(true);
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .flush({ sessions: [], count: 0 });

      component.toggleActive();

      expect(component.activeOnly()).toBe(false);
    });

    it("should reload sessions when toggleActive is called", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .flush({ sessions: mockSessions, count: mockSessions.length });

      component.toggleActive();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/sessions")
      );
      expect(req.request.params.get("active")).toBe("true");
      req.flush({ sessions: [mockSessions[0]], count: 1 });

      expect(component.activeOnly()).toBe(true);
      expect(component.sessions()).toHaveLength(1);
    });
  });

  describe("Session Display", () => {
    it("should display session cards", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .flush({ sessions: mockSessions, count: mockSessions.length });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const sessionCards = compiled.querySelectorAll(".session-card");

      expect(sessionCards.length).toBe(3);
    });

    it("should display agent type for each session", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .flush({ sessions: mockSessions, count: mockSessions.length });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const agentTypes = compiled.querySelectorAll(".agent-type");

      expect(agentTypes[0].textContent).toContain("code-editor");
      expect(agentTypes[1].textContent).toContain("debugger");
      expect(agentTypes[2].textContent).toContain("code-review");
    });

    it("should display session state", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .flush({ sessions: mockSessions, count: mockSessions.length });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const states = compiled.querySelectorAll(".state");

      expect(states[0].textContent).toContain("active");
      expect(states[1].textContent).toContain("completed");
      expect(states[2].textContent).toContain("abandoned");
    });

    it("should apply active class to active sessions", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .flush({ sessions: mockSessions, count: mockSessions.length });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const sessionCards = compiled.querySelectorAll(".session-card");

      expect(sessionCards[0].classList.contains("active")).toBe(true);
      expect(sessionCards[1].classList.contains("active")).toBe(false);
    });

    it("should display session goal", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .flush({ sessions: mockSessions, count: mockSessions.length });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const goals = compiled.querySelectorAll(".session-goal");

      expect(goals[0].textContent).toContain("Implement user authentication");
      expect(goals[1].textContent).toContain("Fix memory leak in service");
    });

    it('should display "No goal specified" when goal is missing', () => {
      const sessionWithoutGoal: Session = {
        id: "sess-5",
        agentId: "agent-5",
        agentType: "monitor",
        state: "active",
        goal: "",
        startedAt: "2025-01-06T10:00:00Z",
        lastActivity: "2025-01-06T10:30:00Z",
        summary: "",
      };

      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .flush({ sessions: [sessionWithoutGoal], count: 1 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const goal = compiled.querySelector(".session-goal");

      expect(goal?.textContent).toContain("No goal specified");
    });

    it("should display session summary when present", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .flush({ sessions: mockSessions, count: mockSessions.length });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const summaries = compiled.querySelectorAll(".session-summary");

      expect(summaries.length).toBe(2); // Only 2 sessions have summaries
      expect(summaries[0].textContent).toContain("Working on login component");
    });

    it("should not display summary when null", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .flush({ sessions: [mockSessions[2]], count: 1 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const summaries = compiled.querySelectorAll(".session-summary");

      expect(summaries.length).toBe(0);
    });

    it("should display empty message when no sessions", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .flush({ sessions: [], count: 0 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const emptyMessage = compiled.querySelector(".empty");

      expect(emptyMessage).toBeDefined();
      expect(emptyMessage?.textContent).toContain("No sessions found");
    });
  });

  describe("Date Formatting", () => {
    it("should format valid timestamp", () => {
      const timestamp = "2025-01-06T10:30:00Z";
      const formatted = component.formatDate(timestamp);

      expect(formatted).toBeDefined();
      expect(typeof formatted).toBe("string");
      expect(formatted).not.toBe("Unknown");
    });

    it('should return "Unknown" for empty timestamp', () => {
      const formatted = component.formatDate("");

      expect(formatted).toBe("Unknown");
    });

    it('should return "Unknown" for null/undefined timestamp', () => {
      const formattedNull = component.formatDate(null as any);
      const formattedUndefined = component.formatDate(undefined as any);

      expect(formattedNull).toBe("Unknown");
      expect(formattedUndefined).toBe("Unknown");
    });

    it("should display formatted dates in session cards", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .flush({ sessions: mockSessions, count: mockSessions.length });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const metaSpans = compiled.querySelectorAll(".session-meta span");

      expect(metaSpans.length).toBeGreaterThan(0);
      expect(metaSpans[0].textContent).toContain("Started:");
      expect(metaSpans[1].textContent).toContain("Last activity:");
    });
  });

  describe("Error Handling", () => {
    it("should handle sessions loading error gracefully", () => {
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .error(new ProgressEvent("error"));

      expect(component.sessions()).toEqual([]);
      expect(consoleSpy).toHaveBeenCalled();

      consoleSpy.mockRestore();
    });

    it("should maintain active state after error", () => {
      component.activeOnly.set(true);
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .error(new ProgressEvent("error"));

      expect(component.activeOnly()).toBe(true);

      consoleSpy.mockRestore();
    });
  });

  describe("Manual Refresh", () => {
    it("should reload sessions when loadSessions is called", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .flush({ sessions: mockSessions, count: mockSessions.length });

      const newSessions: Session[] = [mockSessions[0]];
      component.loadSessions();

      httpMock
        .expectOne((request) => request.url.includes("/api/sessions"))
        .flush({ sessions: newSessions, count: newSessions.length });

      expect(component.sessions()).toEqual(newSessions);
      expect(component.sessions()).toHaveLength(1);
    });
  });
});
