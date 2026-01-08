import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { TestBed, ComponentFixture } from "@angular/core/testing";
import { provideHttpClient } from "@angular/common/http";
import {
  HttpTestingController,
  provideHttpClientTesting,
} from "@angular/common/http/testing";
import { CorridorsComponent } from "./corridors.component";
import { ApiService } from "../../core/services/api.service";
import { LoggerService } from "../../core/services/logger.service";

interface PersonalLearning {
  id: string;
  originWorkspace: string;
  content: string;
  confidence: number;
  useCount: number;
}

interface LinkedWorkspace {
  name: string;
  path: string;
  addedAt: string;
  lastAccessed: string;
}

describe("CorridorsComponent", () => {
  let component: CorridorsComponent;
  let fixture: ComponentFixture<CorridorsComponent>;
  let apiService: ApiService;
  let httpMock: HttpTestingController;

  const mockCorridorStats = {
    learningCount: 42,
    averageConfidence: 0.85,
  };

  const mockLinks: LinkedWorkspace[] = [
    {
      name: "project-alpha",
      path: "/home/user/projects/alpha",
      addedAt: "2025-01-01T10:00:00Z",
      lastAccessed: "2025-01-06T09:30:00Z",
    },
    {
      name: "project-beta",
      path: "/home/user/projects/beta",
      addedAt: "2024-12-15T14:20:00Z",
      lastAccessed: "2025-01-05T11:45:00Z",
    },
  ];

  const mockPersonalLearnings: PersonalLearning[] = [
    {
      id: "pl-1",
      originWorkspace: "project-alpha",
      content: "Use composition over inheritance",
      confidence: 0.92,
      useCount: 15,
    },
    {
      id: "pl-2",
      originWorkspace: "project-beta",
      content: "Implement error boundaries for React components",
      confidence: 0.88,
      useCount: 8,
    },
    {
      id: "pl-3",
      originWorkspace: "",
      content: "Always sanitize user input",
      confidence: 0.95,
      useCount: 22,
    },
  ];

  beforeEach(() => {
    TestBed.resetTestingModule();

    TestBed.configureTestingModule({
      imports: [CorridorsComponent],
      providers: [
        ApiService,
        LoggerService,
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    });

    fixture = TestBed.createComponent(CorridorsComponent);
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
            req.flush({ stats: null, links: [], learnings: [] });
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
      expect(component.stats()).toBeNull();
    });

    it("should initialize with empty links array", () => {
      expect(component.links()).toEqual([]);
    });

    it("should initialize with empty personalLearnings array", () => {
      expect(component.personalLearnings()).toEqual([]);
    });

    it("should load corridor data on initialization", () => {
      fixture.detectChanges();

      const corridorReq = httpMock.expectOne(
        (request) =>
          request.url.includes("/api/corridors") &&
          !request.url.includes("/api/corridors/personal")
      );
      expect(corridorReq.request.method).toBe("GET");
      corridorReq.flush({
        stats: mockCorridorStats,
        links: mockLinks,
      });

      const learningsReq = httpMock.expectOne((request) =>
        request.url.includes("/api/corridors/personal")
      );
      learningsReq.flush({
        learnings: mockPersonalLearnings,
        count: mockPersonalLearnings.length,
      });

      expect(component.stats()).toEqual(mockCorridorStats);
      expect(component.links()).toEqual(mockLinks);
      expect(component.personalLearnings()).toEqual(mockPersonalLearnings);
    });
  });

  describe("Corridor Stats Loading", () => {
    it("should load corridor stats and links", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne(
        (request) =>
          request.url.includes("/api/corridors") &&
          !request.url.includes("/api/corridors/personal")
      );
      req.flush({
        stats: mockCorridorStats,
        links: mockLinks,
      });

      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({ learnings: [], count: 0 });

      expect(component.stats()).toEqual(mockCorridorStats);
      expect(component.links()).toEqual(mockLinks);
    });

    it("should handle empty links in response", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne(
        (request) =>
          request.url.includes("/api/corridors") &&
          !request.url.includes("/api/corridors/personal")
      );
      req.flush({
        stats: mockCorridorStats,
        links: [],
      });

      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({ learnings: [], count: 0 });

      expect(component.links()).toEqual([]);
    });

    it("should handle missing links field in response", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne(
        (request) =>
          request.url.includes("/api/corridors") &&
          !request.url.includes("/api/corridors/personal")
      );
      req.flush({
        stats: mockCorridorStats,
      });

      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({ learnings: [], count: 0 });

      expect(component.links()).toEqual([]);
    });
  });

  describe("Personal Learnings Loading", () => {
    it("should load personal learnings with limit of 10", () => {
      fixture.detectChanges();

      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/corridors/personal")
      );
      expect(req.request.params.get("limit")).toBe("10");
      req.flush({
        learnings: mockPersonalLearnings,
        count: mockPersonalLearnings.length,
      });

      expect(component.personalLearnings()).toEqual(mockPersonalLearnings);
    });

    it("should handle empty personal learnings response", () => {
      fixture.detectChanges();

      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/corridors/personal")
      );
      req.flush({ learnings: [], count: 0 });

      expect(component.personalLearnings()).toEqual([]);
    });

    it("should handle missing learnings field in response", () => {
      fixture.detectChanges();

      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/corridors/personal")
      );
      req.flush({ count: 0 });

      expect(component.personalLearnings()).toEqual([]);
    });
  });

  describe("Signal Updates", () => {
    it("should update stats signal when data is loaded", () => {
      fixture.detectChanges();

      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({ learnings: [], count: 0 });

      expect(component.stats()).toEqual(mockCorridorStats);
      expect(component.stats()?.learningCount).toBe(42);
      expect(component.stats()?.averageConfidence).toBe(0.85);
    });

    it("should update links signal when data is loaded", () => {
      fixture.detectChanges();

      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({ learnings: [], count: 0 });

      expect(component.links()).toEqual(mockLinks);
      expect(component.links()).toHaveLength(2);
    });

    it("should update personalLearnings signal when data is loaded", () => {
      fixture.detectChanges();

      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({
          learnings: mockPersonalLearnings,
          count: mockPersonalLearnings.length,
        });

      expect(component.personalLearnings()).toEqual(mockPersonalLearnings);
      expect(component.personalLearnings()).toHaveLength(3);
    });
  });

  describe("Stats Display", () => {
    it("should display learning count", () => {
      fixture.detectChanges();
      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({ learnings: [], count: 0 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const statValues = compiled.querySelectorAll(".stat .value");

      const learningCountFound = Array.from(statValues).some((el: any) =>
        el.textContent.includes("42")
      );
      expect(learningCountFound).toBeTruthy();
    });

    it("should display average confidence", () => {
      fixture.detectChanges();
      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({ learnings: [], count: 0 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const statValues = compiled.querySelectorAll(".stat .value");

      const confidenceFound = Array.from(statValues).some((el: any) =>
        el.textContent.includes("85%")
      );
      expect(confidenceFound).toBeTruthy();
    });

    it("should display 0 when stats are null", () => {
      component.stats.set(null);
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const statValues = compiled.querySelectorAll(".stat .value");

      expect(statValues.length).toBeGreaterThan(0);
      expect(statValues[0].textContent).toContain("0");
    });
  });

  describe("Personal Learnings Display", () => {
    it("should display personal learning cards", () => {
      fixture.detectChanges();
      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({
          learnings: mockPersonalLearnings,
          count: mockPersonalLearnings.length,
        });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const learningItems = compiled.querySelectorAll(".learning-item");

      expect(learningItems.length).toBe(3);
    });

    it("should display confidence badges", () => {
      fixture.detectChanges();
      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({
          learnings: mockPersonalLearnings,
          count: mockPersonalLearnings.length,
        });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const badges = compiled.querySelectorAll(".confidence-badge");

      expect(badges[0].textContent).toContain("92%");
      expect(badges[1].textContent).toContain("88%");
      expect(badges[2].textContent).toContain("95%");
    });

    it("should display learning content", () => {
      fixture.detectChanges();
      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({
          learnings: mockPersonalLearnings,
          count: mockPersonalLearnings.length,
        });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const contents = compiled.querySelectorAll(".content");

      expect(contents[0].textContent).toContain(
        "Use composition over inheritance"
      );
      expect(contents[1].textContent).toContain(
        "Implement error boundaries for React components"
      );
    });

    it("should display origin workspace when present", () => {
      fixture.detectChanges();
      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({
          learnings: mockPersonalLearnings,
          count: mockPersonalLearnings.length,
        });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const origins = compiled.querySelectorAll(".origin");

      expect(origins.length).toBe(2); // Only 2 have origin workspace
      expect(origins[0].textContent).toContain("from: project-alpha");
      expect(origins[1].textContent).toContain("from: project-beta");
    });

    it("should display empty message when no learnings", () => {
      fixture.detectChanges();
      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({ learnings: [], count: 0 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const emptyMessage = compiled.querySelector(".learnings-preview .empty");

      expect(emptyMessage).toBeDefined();
      expect(emptyMessage?.textContent).toContain("No personal learnings yet");
    });
  });

  describe("Linked Workspaces Display", () => {
    it("should display linked workspace cards", () => {
      fixture.detectChanges();
      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({ learnings: [], count: 0 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const linkCards = compiled.querySelectorAll(".link-card");

      expect(linkCards.length).toBe(2);
    });

    it("should display workspace names", () => {
      fixture.detectChanges();
      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({ learnings: [], count: 0 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const linkNames = compiled.querySelectorAll(".link-name");

      expect(linkNames[0].textContent).toContain("project-alpha");
      expect(linkNames[1].textContent).toContain("project-beta");
    });

    it("should display workspace paths", () => {
      fixture.detectChanges();
      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({ learnings: [], count: 0 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const linkPaths = compiled.querySelectorAll(".link-path");

      expect(linkPaths[0].textContent).toContain("/home/user/projects/alpha");
      expect(linkPaths[1].textContent).toContain("/home/user/projects/beta");
    });

    it("should display formatted dates", () => {
      fixture.detectChanges();
      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({ learnings: [], count: 0 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const linkMetas = compiled.querySelectorAll(".link-meta");

      expect(linkMetas[0].textContent).toContain("Added:");
      expect(linkMetas[0].textContent).toContain("Last accessed:");
    });

    it('should display "Never" when lastAccessed is null', () => {
      const linkWithoutAccess: LinkedWorkspace = {
        name: "unused-project",
        path: "/home/user/unused",
        addedAt: "2025-01-01T10:00:00Z",
        lastAccessed: null as any,
      };

      fixture.detectChanges();
      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: [linkWithoutAccess],
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({ learnings: [], count: 0 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const linkMeta = compiled.querySelector(".link-meta");

      expect(linkMeta?.textContent).toContain("Never");
    });

    it("should display empty message when no links", () => {
      fixture.detectChanges();
      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: [],
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({ learnings: [], count: 0 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const emptyMessage = compiled.querySelector(".links-list .empty");

      expect(emptyMessage).toBeDefined();
      expect(emptyMessage?.textContent).toContain("No linked workspaces");
      expect(emptyMessage?.textContent).toContain("palace corridor link");
    });
  });

  describe("Formatting Functions", () => {
    it("should format confidence as percentage", () => {
      const formatted = component.formatConfidence(0.85);

      expect(formatted).toBe("85%");
    });

    it('should return "0%" for undefined confidence', () => {
      const formatted = component.formatConfidence(undefined);

      expect(formatted).toBe("0%");
    });

    it('should return "0%" for null confidence', () => {
      const formatted = component.formatConfidence(null as any);

      expect(formatted).toBe("0%");
    });

    it("should format date correctly", () => {
      const formatted = component.formatDate("2025-01-06T10:30:00Z");

      expect(formatted).toBeDefined();
      expect(typeof formatted).toBe("string");
      expect(formatted).not.toBe("Unknown");
    });

    it('should return "Unknown" for empty date', () => {
      const formatted = component.formatDate("");

      expect(formatted).toBe("Unknown");
    });

    it('should return "Unknown" for null/undefined date', () => {
      const formattedNull = component.formatDate(null as any);
      const formattedUndefined = component.formatDate(undefined as any);

      expect(formattedNull).toBe("Unknown");
      expect(formattedUndefined).toBe("Unknown");
    });
  });

  describe("Error Handling", () => {
    it("should handle corridors loading error gracefully", () => {
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      fixture.detectChanges();
      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .error(new ProgressEvent("error"));
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({ learnings: [], count: 0 });

      expect(component.stats()).toBeNull();
      expect(component.links()).toEqual([]);
      expect(consoleSpy).toHaveBeenCalled();

      consoleSpy.mockRestore();
    });

    it("should handle personal learnings loading error gracefully", () => {
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      fixture.detectChanges();
      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .error(new ProgressEvent("error"));

      expect(component.personalLearnings()).toEqual([]);
      expect(consoleSpy).toHaveBeenCalled();

      consoleSpy.mockRestore();
    });
  });

  describe("Manual Refresh", () => {
    it("should reload data when loadData is called", () => {
      fixture.detectChanges();
      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: mockCorridorStats,
          links: mockLinks,
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({
          learnings: mockPersonalLearnings,
          count: mockPersonalLearnings.length,
        });

      const newStats = { learningCount: 50, averageConfidence: 0.9 };
      component.loadData();

      httpMock
        .expectOne(
          (request) =>
            request.url.includes("/api/corridors") &&
            !request.url.includes("/api/corridors/personal")
        )
        .flush({
          stats: newStats,
          links: [],
        });
      httpMock
        .expectOne((request) => request.url.includes("/api/corridors/personal"))
        .flush({ learnings: [], count: 0 });

      expect(component.stats()).toEqual(newStats);
      expect(component.links()).toEqual([]);
    });
  });
});
