import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { TestBed, ComponentFixture } from "@angular/core/testing";
import { provideHttpClient } from "@angular/common/http";
import {
  HttpTestingController,
  provideHttpClientTesting,
} from "@angular/common/http/testing";
import { Component, input } from "@angular/core";
import { OverviewComponent } from "./overview.component";
import { NeuralMapComponent } from "./neural-map/neural-map.component";
import {
  ApiService,
  Stats,
  ActiveAgent,
} from "../../core/services/api.service";

// Mock NeuralMapComponent
@Component({
  selector: "app-neural-map",
  standalone: true,
  template: '<div class="neural-map-mock"></div>',
})
class MockNeuralMapComponent {
  height = input<number>(420);
}

describe("OverviewComponent", () => {
  let component: OverviewComponent;
  let fixture: ComponentFixture<OverviewComponent>;
  let apiService: ApiService;
  let httpMock: HttpTestingController;

  const mockStats: Stats = {
    sessions: { total: 25, active: 3 },
    learnings: 150,
    filesTracked: 75,
    rooms: 8,
    corridor: {
      learningCount: 40,
      linkedWorkspaces: 5,
      averageConfidence: 0.87,
    },
  };

  const mockAgents: ActiveAgent[] = [
    {
      agentId: "agent-1",
      agentType: "code-editor",
      sessionId: "sess-1",
      heartbeat: "2025-01-06T10:30:00Z",
      currentFile: "/src/app/main.ts",
    },
    {
      agentId: "agent-2",
      agentType: "debugger",
      sessionId: "sess-2",
      heartbeat: "2025-01-06T10:31:00Z",
      currentFile: "/src/app/service.ts",
    },
  ];

  beforeEach(() => {
    TestBed.resetTestingModule();

    TestBed.configureTestingModule({
      imports: [OverviewComponent],
      providers: [ApiService, provideHttpClient(), provideHttpClientTesting()],
    }).overrideComponent(OverviewComponent, {
      remove: {
        imports: [NeuralMapComponent],
      },
      add: {
        imports: [MockNeuralMapComponent],
      },
    });

    fixture = TestBed.createComponent(OverviewComponent);
    component = fixture.componentInstance;
    apiService = TestBed.inject(ApiService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    // Flush any pending HTTP requests
    if (httpMock) {
      try {
        const openRequests = httpMock.match(() => true);
        openRequests.forEach((req) => {
          if (!req.cancelled) {
            req.flush({
              rooms: [],
              ideas: [],
              decisions: [],
              learnings: [],
              count: 0,
            });
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
    it("should initialize with null stats", () => {
      // Assert
      expect(component.stats()).toBeNull();
    });

    it("should initialize with empty agents array", () => {
      // Assert
      expect(component.agents()).toEqual([]);
    });

    it("should load stats on initialization", () => {
      // Act
      fixture.detectChanges(); // Triggers ngOnInit

      // Assert
      const req = httpMock.expectOne("/api/stats");
      expect(req.request.method).toBe("GET");
      req.flush(mockStats);

      expect(component.stats()).toEqual(mockStats);
    });

    it("should load agents on initialization", () => {
      // Act
      fixture.detectChanges(); // Triggers ngOnInit

      // Assert
      httpMock.expectOne("/api/stats").flush(mockStats);

      const agentsReq = httpMock.expectOne("/api/agents");
      expect(agentsReq.request.method).toBe("GET");
      agentsReq.flush({ agents: mockAgents, count: mockAgents.length });

      expect(component.agents()).toEqual(mockAgents);
    });
  });

  describe("Signal Updates", () => {
    it("should update stats signal when data is loaded", () => {
      // Act
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock.expectOne("/api/agents").flush({ agents: [], count: 0 });

      // Assert
      expect(component.stats()).toEqual(mockStats);
      expect(component.stats()?.sessions.total).toBe(25);
      expect(component.stats()?.sessions.active).toBe(3);
    });

    it("should update agents signal when data is loaded", () => {
      // Act
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock
        .expectOne("/api/agents")
        .flush({ agents: mockAgents, count: mockAgents.length });

      // Assert
      expect(component.agents()).toEqual(mockAgents);
      expect(component.agents()).toHaveLength(2);
    });

    it("should handle stats signal reactivity", () => {
      // Arrange
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock.expectOne("/api/agents").flush({ agents: [], count: 0 });

      // Act - Update stats
      const newStats: Stats = {
        sessions: { total: 30, active: 5 },
        learnings: 200,
        filesTracked: 100,
        rooms: 10,
      };
      component.stats.set(newStats);

      // Assert
      expect(component.stats()).toEqual(newStats);
      expect(component.stats()?.sessions.total).toBe(30);
    });
  });

  describe("Stats Rendering", () => {
    it("should display total sessions count", () => {
      // Arrange
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock.expectOne("/api/agents").flush({ agents: [], count: 0 });
      fixture.detectChanges();

      // Act
      const compiled = fixture.nativeElement;
      const statCards = compiled.querySelectorAll(".stat-card");

      // Assert
      expect(statCards.length).toBeGreaterThan(0);
      const totalSessionsCard = Array.from(statCards).find((card: any) =>
        card.textContent.includes("Total Sessions")
      ) as HTMLElement;
      expect(totalSessionsCard).toBeDefined();
      expect(totalSessionsCard?.textContent).toContain("25");
    });

    it("should display active sessions count", () => {
      // Arrange
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock.expectOne("/api/agents").flush({ agents: [], count: 0 });
      fixture.detectChanges();

      // Act
      const compiled = fixture.nativeElement;

      // Assert
      expect(compiled.textContent).toContain("3 active");
    });

    it("should display learnings count", () => {
      // Arrange
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock.expectOne("/api/agents").flush({ agents: [], count: 0 });
      fixture.detectChanges();

      // Act
      const compiled = fixture.nativeElement;
      const statCards = compiled.querySelectorAll(".stat-card");

      // Assert
      const learningsCard = Array.from(statCards).find((card: any) =>
        card.textContent.includes("Learnings")
      ) as HTMLElement;
      expect(learningsCard).toBeDefined();
      expect(learningsCard?.textContent).toContain("150");
    });

    it("should display files tracked count", () => {
      // Arrange
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock.expectOne("/api/agents").flush({ agents: [], count: 0 });
      fixture.detectChanges();

      // Act
      const compiled = fixture.nativeElement;
      const statCards = compiled.querySelectorAll(".stat-card");

      // Assert
      const filesCard = Array.from(statCards).find((card: any) =>
        card.textContent.includes("Files Tracked")
      ) as HTMLElement;
      expect(filesCard).toBeDefined();
      expect(filesCard?.textContent).toContain("75");
    });

    it("should display rooms count", () => {
      // Arrange
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock.expectOne("/api/agents").flush({ agents: [], count: 0 });
      fixture.detectChanges();

      // Act
      const compiled = fixture.nativeElement;
      const statCards = compiled.querySelectorAll(".stat-card");

      // Assert
      const roomsCard = Array.from(statCards).find((card: any) =>
        card.textContent.includes("Rooms")
      ) as HTMLElement;
      expect(roomsCard).toBeDefined();
      expect(roomsCard?.textContent).toContain("8");
    });

    it("should display default values when stats are null", () => {
      // Arrange
      component.stats.set(null);
      fixture.detectChanges();

      // Act
      const compiled = fixture.nativeElement;
      const statValues = compiled.querySelectorAll(".stat-value");

      // Assert
      expect(statValues.length).toBeGreaterThan(0);
      statValues.forEach((value: HTMLElement) => {
        expect(value.textContent).toContain("0");
      });
    });
  });

  describe("Active Agents Display", () => {
    it("should display active agents when present", () => {
      // Arrange
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock
        .expectOne("/api/agents")
        .flush({ agents: mockAgents, count: mockAgents.length });
      fixture.detectChanges();

      // Act
      const compiled = fixture.nativeElement;
      const agentCards = compiled.querySelectorAll(".agent-card");

      // Assert
      expect(agentCards.length).toBe(2);
    });

    it("should display agent type correctly", () => {
      // Arrange
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock
        .expectOne("/api/agents")
        .flush({ agents: mockAgents, count: mockAgents.length });
      fixture.detectChanges();

      // Act
      const compiled = fixture.nativeElement;
      const agentTypes = compiled.querySelectorAll(".agent-type");

      // Assert
      expect(agentTypes[0].textContent).toContain("code-editor");
      expect(agentTypes[1].textContent).toContain("debugger");
    });

    it("should display current file for each agent", () => {
      // Arrange
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock
        .expectOne("/api/agents")
        .flush({ agents: mockAgents, count: mockAgents.length });
      fixture.detectChanges();

      // Act
      const compiled = fixture.nativeElement;
      const agentFiles = compiled.querySelectorAll(".agent-file");

      // Assert
      expect(agentFiles[0].textContent).toContain("/src/app/main.ts");
      expect(agentFiles[1].textContent).toContain("/src/app/service.ts");
    });

    it("should not display agents section when no agents are active", () => {
      // Arrange
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock.expectOne("/api/agents").flush({ agents: [], count: 0 });
      fixture.detectChanges();

      // Act
      const compiled = fixture.nativeElement;
      const agentsSection = compiled.querySelector(".agents-list");

      // Assert
      expect(agentsSection).toBeNull();
    });

    it('should display "No active file" when agent has no current file', () => {
      // Arrange
      const agentsWithoutFile: ActiveAgent[] = [
        {
          agentId: "agent-1",
          agentType: "monitor",
          sessionId: "sess-1",
          heartbeat: "2025-01-06T10:30:00Z",
          currentFile: "",
        },
      ];

      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock
        .expectOne("/api/agents")
        .flush({ agents: agentsWithoutFile, count: 1 });
      fixture.detectChanges();

      // Act
      const compiled = fixture.nativeElement;
      const agentFile = compiled.querySelector(".agent-file");

      // Assert
      expect(agentFile?.textContent).toContain("No active file");
    });
  });

  describe("Corridor Stats Display", () => {
    it("should display corridor stats when present", () => {
      // Arrange
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock.expectOne("/api/agents").flush({ agents: [], count: 0 });
      fixture.detectChanges();

      // Act
      const compiled = fixture.nativeElement;
      const corridorSection = compiled.querySelector(".corridor-stats");

      // Assert
      expect(corridorSection).toBeDefined();
      expect(corridorSection?.textContent).toContain("40");
      expect(corridorSection?.textContent).toContain("5");
    });

    it("should not display corridor section when not present in stats", () => {
      // Arrange
      const statsWithoutCorridor: Stats = {
        sessions: { total: 25, active: 3 },
        learnings: 150,
        filesTracked: 75,
        rooms: 8,
      };

      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(statsWithoutCorridor);
      httpMock.expectOne("/api/agents").flush({ agents: [], count: 0 });
      fixture.detectChanges();

      // Act
      const compiled = fixture.nativeElement;
      const corridorSection = compiled.querySelector(".corridor-stats");

      // Assert
      expect(corridorSection).toBeNull();
    });
  });

  describe("Error Handling", () => {
    it("should handle stats loading error gracefully", () => {
      // Arrange
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      // Act
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").error(new ProgressEvent("error"));
      httpMock.expectOne("/api/agents").flush({ agents: [], count: 0 });

      // Assert
      expect(component.stats()).toBeNull();
      // Logger service outputs formatted messages, verify it was called
      expect(consoleSpy).toHaveBeenCalled();
      expect(
        consoleSpy.mock.calls.some((call) =>
          call[0]?.toString().includes("Failed to load stats")
        )
      ).toBe(true);

      consoleSpy.mockRestore();
    });

    it("should handle agents loading error gracefully", () => {
      // Arrange
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      // Act
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock.expectOne("/api/agents").error(new ProgressEvent("error"));

      // Assert
      expect(component.agents()).toEqual([]);
      // Logger service outputs formatted messages, verify it was called
      expect(consoleSpy).toHaveBeenCalled();
      expect(
        consoleSpy.mock.calls.some(
          (call) =>
            call[0]?.toString().includes("Failed to load") &&
            call[0]?.toString().includes("agents")
        )
      ).toBe(true);

      consoleSpy.mockRestore();
    });
  });

  describe("Time Formatting", () => {
    it("should format timestamp correctly", () => {
      // Arrange
      const timestamp = "2025-01-06T14:30:45Z";

      // Act
      const formatted = component.formatTime(timestamp);

      // Assert
      expect(formatted).toBeDefined();
      expect(typeof formatted).toBe("string");
      // The exact format depends on locale, so just verify it's not 'Unknown'
      expect(formatted).not.toBe("Unknown");
    });

    it('should return "Unknown" for invalid timestamp', () => {
      // Arrange
      const invalidTimestamp = "";

      // Act
      const formatted = component.formatTime(invalidTimestamp);

      // Assert
      expect(formatted).toBe("Unknown");
    });

    it('should return "Unknown" for null/undefined timestamp', () => {
      // Act
      const formattedNull = component.formatTime(null as any);
      const formattedUndefined = component.formatTime(undefined as any);

      // Assert
      expect(formattedNull).toBe("Unknown");
      expect(formattedUndefined).toBe("Unknown");
    });
  });

  describe("Manual Data Refresh", () => {
    it("should reload stats when loadStats is called", () => {
      // Arrange
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock.expectOne("/api/agents").flush({ agents: [], count: 0 });

      const newStats: Stats = {
        sessions: { total: 50, active: 10 },
        learnings: 300,
        filesTracked: 150,
        rooms: 15,
      };

      // Act
      component.loadStats();
      httpMock.expectOne("/api/stats").flush(newStats);

      // Assert
      expect(component.stats()).toEqual(newStats);
      expect(component.stats()?.sessions.total).toBe(50);
    });

    it("should reload agents when loadAgents is called", () => {
      // Arrange
      fixture.detectChanges();
      httpMock.expectOne("/api/stats").flush(mockStats);
      httpMock.expectOne("/api/agents").flush({ agents: [], count: 0 });

      // Act
      component.loadAgents();
      httpMock
        .expectOne("/api/agents")
        .flush({ agents: mockAgents, count: mockAgents.length });

      // Assert
      expect(component.agents()).toEqual(mockAgents);
      expect(component.agents()).toHaveLength(2);
    });
  });
});
