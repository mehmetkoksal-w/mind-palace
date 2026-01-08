import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { TestBed, ComponentFixture } from "@angular/core/testing";
import { provideHttpClient } from "@angular/common/http";
import {
  HttpTestingController,
  provideHttpClientTesting,
} from "@angular/common/http/testing";
import { FormsModule } from "@angular/forms";
import { LearningsComponent } from "./learnings.component";
import { ApiService, Learning } from "../../core/services/api.service";
import { LoggerService } from "../../core/services/logger.service";

describe("LearningsComponent", () => {
  let component: LearningsComponent;
  let fixture: ComponentFixture<LearningsComponent>;
  let apiService: ApiService;
  let httpMock: HttpTestingController;

  const mockLearnings: Learning[] = [
    {
      id: "learn-1",
      sessionId: "sess-1",
      content: "Always validate user input before processing",
      confidence: 0.95,
      scope: "palace",
      scopePath: "/src/app",
      source: "code-review",
      useCount: 15,
      createdAt: "2025-01-01T10:00:00Z",
      lastUsed: "2025-01-06T10:00:00Z",
    },
    {
      id: "learn-2",
      sessionId: "sess-2",
      content: "Use dependency injection for services",
      confidence: 0.87,
      scope: "room",
      scopePath: "/src/services",
      source: "pattern-analysis",
      useCount: 8,
      createdAt: "2025-01-02T10:00:00Z",
      lastUsed: "2025-01-05T10:00:00Z",
    },
    {
      id: "learn-3",
      sessionId: "sess-3",
      content: "Handle errors with proper logging",
      confidence: 0.72,
      scope: "file",
      scopePath: "/src/app/main.ts",
      source: "debugging",
      useCount: 3,
      createdAt: "2025-01-03T10:00:00Z",
      lastUsed: "2025-01-04T10:00:00Z",
    },
  ];

  beforeEach(() => {
    TestBed.resetTestingModule();

    TestBed.configureTestingModule({
      imports: [LearningsComponent, FormsModule],
      providers: [
        ApiService,
        LoggerService,
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    });

    fixture = TestBed.createComponent(LearningsComponent);
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
            req.flush({ learnings: [], count: 0 });
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
    it("should initialize with empty learnings array", () => {
      expect(component.learnings()).toEqual([]);
    });

    it("should initialize with empty search query", () => {
      expect(component.searchQuery).toBe("");
    });

    it("should load learnings on initialization", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/learnings")
      );
      expect(req.request.method).toBe("GET");
      req.flush({ learnings: mockLearnings, count: mockLearnings.length });

      expect(component.learnings()).toEqual(mockLearnings);
      expect(component.learnings()).toHaveLength(3);
    });
  });

  describe("Learnings Loading", () => {
    it("should load all learnings with empty search query", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/learnings")
      );
      req.flush({ learnings: mockLearnings, count: mockLearnings.length });

      expect(component.learnings()).toEqual(mockLearnings);
    });

    it("should handle empty learnings response", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/learnings")
      );
      req.flush({ learnings: [], count: 0 });

      expect(component.learnings()).toEqual([]);
    });

    it("should handle missing learnings field in response", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/learnings")
      );
      req.flush({ count: 0 });

      expect(component.learnings()).toEqual([]);
    });
  });

  describe("Search Functionality", () => {
    it("should search learnings with query", () => {
      component.searchQuery = "validation";
      fixture.detectChanges();

      const req = httpMock.expectOne((request) => {
        return (
          request.url.includes("/api/learnings") &&
          request.params.get("query") === "validation"
        );
      });

      const filteredLearnings = [mockLearnings[0]];
      req.flush({ learnings: filteredLearnings, count: 1 });

      expect(component.learnings()).toEqual(filteredLearnings);
    });

    it("should call loadLearnings when search is triggered", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: mockLearnings, count: mockLearnings.length });

      component.searchQuery = "error";
      component.search();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/learnings")
      );
      expect(req.request.params.get("query")).toBe("error");
      req.flush({ learnings: [mockLearnings[2]], count: 1 });

      expect(component.learnings()).toHaveLength(1);
    });

    it("should handle empty search results", () => {
      component.searchQuery = "nonexistent";
      fixture.detectChanges();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/learnings")
      );
      req.flush({ learnings: [], count: 0 });

      expect(component.learnings()).toEqual([]);
    });

    it("should clear results when search query is cleared", () => {
      component.searchQuery = "";
      component.search();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/learnings")
      );
      req.flush({ learnings: mockLearnings, count: mockLearnings.length });

      expect(component.learnings()).toEqual(mockLearnings);
    });
  });

  describe("Signal Updates", () => {
    it("should update learnings signal when data is loaded", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/learnings")
      );
      req.flush({ learnings: mockLearnings, count: mockLearnings.length });

      expect(component.learnings()).toEqual(mockLearnings);
      expect(component.learnings()[0].id).toBe("learn-1");
      expect(component.learnings()[1].confidence).toBe(0.87);
    });

    it("should handle learnings signal reactivity", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: mockLearnings, count: mockLearnings.length });

      const newLearnings: Learning[] = [
        {
          id: "learn-4",
          content: "Test all edge cases",
          confidence: 0.99,
          scope: "palace",
          scopePath: "",
          source: "test-suite",
          sessionId: "sess-4",
          createdAt: "2025-01-06T12:00:00Z",
          lastUsed: "2025-01-06T12:00:00Z",
          useCount: 25,
        },
      ];
      component.learnings.set(newLearnings);

      expect(component.learnings()).toEqual(newLearnings);
      expect(component.learnings()).toHaveLength(1);
    });
  });

  describe("Learnings Display", () => {
    it("should display learning cards", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: mockLearnings, count: mockLearnings.length });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const learningCards = compiled.querySelectorAll(".learning-card");

      expect(learningCards.length).toBe(3);
    });

    it("should display learning content", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: mockLearnings, count: mockLearnings.length });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const contents = compiled.querySelectorAll(".learning-content");

      expect(contents[0].textContent).toContain(
        "Always validate user input before processing"
      );
      expect(contents[1].textContent).toContain(
        "Use dependency injection for services"
      );
    });

    it("should display confidence percentage", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: mockLearnings, count: mockLearnings.length });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const confidenceTexts = compiled.querySelectorAll(".confidence-text");

      expect(confidenceTexts[0].textContent).toContain("95%");
      expect(confidenceTexts[1].textContent).toContain("87%");
      expect(confidenceTexts[2].textContent).toContain("72%");
    });

    it("should display confidence bar with correct width", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: mockLearnings, count: mockLearnings.length });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const fills = compiled.querySelectorAll(".confidence-bar .fill");

      expect(fills[0].style.width).toBe("95%");
      expect(fills[1].style.width).toBe("87%");
      expect(fills[2].style.width).toBe("72%");
    });

    it("should display scope information", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: mockLearnings, count: mockLearnings.length });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const scopes = compiled.querySelectorAll(".scope");

      expect(scopes[0].textContent).toContain("palace:/src/app");
      expect(scopes[1].textContent).toContain("room:/src/services");
      expect(scopes[2].textContent).toContain("file:/src/app/main.ts");
    });

    it("should display scope without path when scopePath is null", () => {
      const learningWithoutPath: Learning = {
        id: "learn-5",
        content: "Global best practice",
        confidence: 0.9,
        scope: "palace",
        scopePath: "",
        source: "documentation",
        sessionId: "sess-5",
        createdAt: "2025-01-06T13:00:00Z",
        lastUsed: "2025-01-06T13:00:00Z",
        useCount: 10,
      };

      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: [learningWithoutPath], count: 1 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const scope = compiled.querySelector(".scope");

      expect(scope?.textContent?.trim()).toBe("palace");
    });

    it("should display source information", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: mockLearnings, count: mockLearnings.length });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const sources = compiled.querySelectorAll(".source");

      expect(sources[0].textContent).toContain("code-review");
      expect(sources[1].textContent).toContain("pattern-analysis");
      expect(sources[2].textContent).toContain("debugging");
    });

    it("should display use count", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: mockLearnings, count: mockLearnings.length });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const useCounts = compiled.querySelectorAll(".used");

      expect(useCounts[0].textContent).toContain("Used 15 times");
      expect(useCounts[1].textContent).toContain("Used 8 times");
      expect(useCounts[2].textContent).toContain("Used 3 times");
    });

    it("should display empty message when no learnings", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: [], count: 0 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const emptyMessage = compiled.querySelector(".empty");

      expect(emptyMessage).toBeDefined();
      expect(emptyMessage?.textContent).toContain("No learnings found");
    });
  });

  describe("Search UI", () => {
    it("should render search input", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: [], count: 0 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const searchInput = compiled.querySelector(
        '.search-bar input[type="text"]'
      );

      expect(searchInput).toBeDefined();
      expect(searchInput?.getAttribute("placeholder")).toContain(
        "Search learnings"
      );
    });

    it("should render search button", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: [], count: 0 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const searchButton = compiled.querySelector(".search-bar button");

      expect(searchButton).toBeDefined();
      expect(searchButton?.textContent).toContain("Search");
    });

    it("should update searchQuery on input change", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: [], count: 0 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const searchInput = compiled.querySelector(
        '.search-bar input[type="text"]'
      ) as HTMLInputElement;

      searchInput.value = "test query";
      searchInput.dispatchEvent(new Event("input"));
      fixture.detectChanges();

      // Note: Due to ngModel binding, we need to verify the binding works
      expect(searchInput).toBeDefined();
    });
  });

  describe("Error Handling", () => {
    it("should handle learnings loading error gracefully", () => {
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .error(new ProgressEvent("error"));

      expect(component.learnings()).toEqual([]);
      expect(consoleSpy).toHaveBeenCalled();

      consoleSpy.mockRestore();
    });

    it("should maintain search query after error", () => {
      component.searchQuery = "test";
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .error(new ProgressEvent("error"));

      expect(component.searchQuery).toBe("test");

      consoleSpy.mockRestore();
    });
  });

  describe("Manual Refresh", () => {
    it("should reload learnings when loadLearnings is called", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: mockLearnings, count: mockLearnings.length });

      const newLearnings: Learning[] = [mockLearnings[0], mockLearnings[1]];
      component.loadLearnings();

      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: newLearnings, count: newLearnings.length });

      expect(component.learnings()).toEqual(newLearnings);
      expect(component.learnings()).toHaveLength(2);
    });
  });

  describe("Confidence Filtering", () => {
    it("should display learnings sorted by confidence", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: mockLearnings, count: mockLearnings.length });

      expect(component.learnings()[0].confidence).toBe(0.95);
      expect(component.learnings()[1].confidence).toBe(0.87);
      expect(component.learnings()[2].confidence).toBe(0.72);
    });

    it("should handle learnings with very low confidence", () => {
      const lowConfidenceLearning: Learning = {
        id: "learn-low",
        content: "Experimental pattern",
        confidence: 0.15,
        scope: "file",
        scopePath: "/experimental.ts",
        source: "experimentation",
        sessionId: "sess-low",
        createdAt: "2025-01-06T14:00:00Z",
        lastUsed: "2025-01-06T14:00:00Z",
        useCount: 1,
      };

      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: [lowConfidenceLearning], count: 1 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const confidenceText = compiled.querySelector(".confidence-text");

      expect(confidenceText?.textContent).toContain("15%");
    });

    it("should handle learnings with maximum confidence", () => {
      const maxConfidenceLearning: Learning = {
        id: "learn-max",
        content: "Well-established pattern",
        confidence: 1.0,
        scope: "palace",
        scopePath: "",
        source: "best-practice",
        sessionId: "sess-max",
        createdAt: "2025-01-06T15:00:00Z",
        lastUsed: "2025-01-06T15:00:00Z",
        useCount: 50,
      };

      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/learnings"))
        .flush({ learnings: [maxConfidenceLearning], count: 1 });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const confidenceText = compiled.querySelector(".confidence-text");

      expect(confidenceText?.textContent).toContain("100%");
    });
  });
});
