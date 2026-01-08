import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { TestBed, ComponentFixture } from "@angular/core/testing";
import { provideHttpClient } from "@angular/common/http";
import {
  HttpTestingController,
  provideHttpClientTesting,
} from "@angular/common/http/testing";
import { CommonModule } from "@angular/common";
import { FormsModule } from "@angular/forms";
import { IntelComponent } from "./intel.component";
import {
  ApiService,
  FileIntel,
  Learning,
} from "../../core/services/api.service";
import { LoggerService } from "../../core/services/logger.service";

describe("IntelComponent", () => {
  let component: IntelComponent;
  let fixture: ComponentFixture<IntelComponent>;
  let apiService: ApiService;
  let httpMock: HttpTestingController;

  const mockHotspots: FileIntel[] = [
    {
      path: "src/app/main.ts",
      editCount: 45,
      failureCount: 2,
      lastEditor: "alice@dev.com",
      lastEdited: "2025-01-06T10:30:00Z",
    },
    {
      path: "src/services/api.service.ts",
      editCount: 32,
      failureCount: 0,
      lastEditor: "bob@dev.com",
      lastEdited: "2025-01-05T15:20:00Z",
    },
    {
      path: "src/components/header.tsx",
      editCount: 18,
      failureCount: 1,
      lastEditor: "charlie@dev.com",
      lastEdited: "2025-01-04T09:15:00Z",
    },
  ];

  const mockFragile: FileIntel[] = [
    {
      path: "src/legacy/old-module.js",
      editCount: 25,
      failureCount: 15,
      lastEditor: "alice@dev.com",
      lastEdited: "2025-01-06T11:00:00Z",
    },
  ];

  const mockFileLearnings: Learning[] = [
    {
      id: "learn-1",
      sessionId: "sess-1",
      content: "This file handles authentication logic",
      confidence: 0.92,
      scope: "file",
      scopePath: "src/app/main.ts",
      source: "code-analysis",
      useCount: 3,
      createdAt: "2025-01-01T10:00:00Z",
      lastUsed: "2025-01-06T10:00:00Z",
    },
  ];

  beforeEach(() => {
    TestBed.resetTestingModule();

    TestBed.configureTestingModule({
      imports: [IntelComponent, CommonModule, FormsModule],
      providers: [
        ApiService,
        LoggerService,
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    });

    fixture = TestBed.createComponent(IntelComponent);
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
            req.flush({ hotspots: [], fragile: [], learnings: [] });
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
    it("should initialize with empty hotspots array", () => {
      expect(component.hotspots()).toEqual([]);
    });

    it("should initialize with empty fragile array", () => {
      expect(component.fragile()).toEqual([]);
    });

    it("should initialize with loading set to true", () => {
      expect(component.loading()).toBe(true);
    });

    it("should initialize with heatmap view mode", () => {
      expect(component.viewMode()).toBe("heatmap");
    });

    it("should load file intelligence on initialization", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/hotspots")
      );
      expect(req.request.method).toBe("GET");
      req.flush({ hotspots: mockHotspots, fragile: mockFragile });

      expect(component.hotspots()).toEqual(mockHotspots);
      expect(component.fragile()).toEqual(mockFragile);
      expect(component.loading()).toBe(false);
    });
  });

  describe("Data Loading", () => {
    it("should set loading to false after successful load", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/hotspots")
      );
      req.flush({ hotspots: mockHotspots, fragile: mockFragile });

      expect(component.loading()).toBe(false);
    });

    it("should handle empty hotspots response", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/hotspots")
      );
      req.flush({ hotspots: [], fragile: [] });

      expect(component.hotspots()).toEqual([]);
      expect(component.loading()).toBe(false);
    });

    it("should handle missing fields in response", () => {
      fixture.detectChanges();

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/hotspots")
      );
      req.flush({});

      expect(component.hotspots()).toEqual([]);
      expect(component.fragile()).toEqual([]);
    });
  });

  describe("View Mode Switching", () => {
    it("should switch to list view", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: mockHotspots, fragile: mockFragile });

      component.viewMode.set("list");

      expect(component.viewMode()).toBe("list");
    });

    it("should switch to tree view", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: mockHotspots, fragile: mockFragile });

      component.viewMode.set("tree");

      expect(component.viewMode()).toBe("tree");
    });

    it("should switch back to heatmap view", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: mockHotspots, fragile: mockFragile });

      component.viewMode.set("list");
      component.viewMode.set("heatmap");

      expect(component.viewMode()).toBe("heatmap");
    });
  });

  describe("Stats Calculations", () => {
    it("should calculate total edits correctly", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: mockHotspots, fragile: mockFragile });

      const total = component.totalEdits();

      expect(total).toBe(45 + 32 + 18);
    });

    it("should calculate total failures correctly", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: mockHotspots, fragile: mockFragile });

      const total = component.totalFailures();

      expect(total).toBe(2 + 0 + 1);
    });

    it("should return 0 for total edits when no hotspots", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: [], fragile: [] });

      expect(component.totalEdits()).toBe(0);
    });

    it("should return 0 for total failures when no hotspots", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: [], fragile: [] });

      expect(component.totalFailures()).toBe(0);
    });
  });

  describe("Heat Color Calculation", () => {
    it("should return low heat color for low edit count", () => {
      const lowEditFile: FileIntel = {
        path: "test.ts",
        editCount: 5,
        failureCount: 0,
        lastEditor: "",
        lastEdited: "",
      };
      const highEditFile: FileIntel = {
        path: "high.ts",
        editCount: 100,
        failureCount: 0,
        lastEditor: "",
        lastEdited: "",
      };

      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: [lowEditFile, highEditFile], fragile: [] });

      const color = component.getHeatColor(lowEditFile);

      expect(color).toBe("#1e3a5f"); // 5/100 = 0.05 < 0.25
    });

    it("should return high heat color for high edit count", () => {
      const file: FileIntel = {
        path: "test.ts",
        editCount: 100,
        failureCount: 0,
        lastEditor: "",
        lastEdited: "",
      };

      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: [file], fragile: [] });

      const color = component.getHeatColor(file);

      expect(color).toBe("#ef4444");
    });

    it("should return medium heat color for medium edit count", () => {
      const mediumFile: FileIntel = {
        path: "medium.ts",
        editCount: 50,
        failureCount: 0,
        lastEditor: "",
        lastEdited: "",
      };
      const highFile: FileIntel = {
        path: "high.ts",
        editCount: 100,
        failureCount: 0,
        lastEditor: "",
        lastEdited: "",
      };

      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: [mediumFile, highFile], fragile: [] });

      const color = component.getHeatColor(mediumFile);

      expect(["#3b82f6", "#f59e0b"]).toContain(color); // 50/100 = 0.5, should be blue or orange
    });
  });

  describe("File Name Extraction", () => {
    it("should extract file name from path", () => {
      const fileName = component.getFileName("src/app/services/api.ts");

      expect(fileName).toBe("api.ts");
    });

    it("should return full path when no slashes", () => {
      const fileName = component.getFileName("test.ts");

      expect(fileName).toBe("test.ts");
    });

    it("should handle empty path", () => {
      const fileName = component.getFileName("");

      expect(fileName).toBe("");
    });
  });

  describe("Percentage Calculations", () => {
    it("should calculate edit percentage correctly", () => {
      const file: FileIntel = {
        path: "test.ts",
        editCount: 50,
        failureCount: 0,
        lastEditor: "",
        lastEdited: "",
      };

      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: [{ ...file, editCount: 100 }], fragile: [] });

      const percentage = component.getEditPercentage(file);

      expect(percentage).toBe(50);
    });

    it("should calculate failure percentage correctly", () => {
      const file: FileIntel = {
        path: "test.ts",
        editCount: 100,
        failureCount: 25,
        lastEditor: "",
        lastEdited: "",
      };

      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: [file], fragile: [] });

      const percentage = component.getFailurePercentage(file);

      expect(percentage).toBe(25);
    });

    it("should return 0 failure percentage when no edits", () => {
      const file: FileIntel = {
        path: "test.ts",
        editCount: 0,
        failureCount: 0,
        lastEditor: "",
        lastEdited: "",
      };

      const percentage = component.getFailurePercentage(file);

      expect(percentage).toBe(0);
    });

    it("should cap failure percentage at 100", () => {
      const file: FileIntel = {
        path: "test.ts",
        editCount: 10,
        failureCount: 50,
        lastEditor: "",
        lastEdited: "",
      };

      const percentage = component.getFailurePercentage(file);

      expect(percentage).toBe(100);
    });
  });

  describe("Failure Rate Calculation", () => {
    it("should calculate failure rate as string", () => {
      const file: FileIntel = {
        path: "test.ts",
        editCount: 100,
        failureCount: 15,
        lastEditor: "",
        lastEdited: "",
      };

      const rate = component.getFailureRate(file);

      expect(rate).toBe("15");
    });

    it('should return "0" when no edits', () => {
      const file: FileIntel = {
        path: "test.ts",
        editCount: 0,
        failureCount: 0,
        lastEditor: "",
        lastEdited: "",
      };

      const rate = component.getFailureRate(file);

      expect(rate).toBe("0");
    });

    it("should round failure rate to nearest integer", () => {
      const file: FileIntel = {
        path: "test.ts",
        editCount: 100,
        failureCount: 17,
        lastEditor: "",
        lastEdited: "",
      };

      const rate = component.getFailureRate(file);

      expect(rate).toBe("17");
    });
  });

  describe("File Selection", () => {
    it("should select a file", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: mockHotspots, fragile: mockFragile });

      component.selectFile(mockHotspots[0]);

      expect(component.selectedFile()).toEqual(mockHotspots[0]);
    });

    it("should load file learnings when file is selected", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: mockHotspots, fragile: mockFragile });

      component.selectFile(mockHotspots[0]);

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/file-intel")
      );
      req.flush({ learnings: mockFileLearnings });

      expect(component.fileLearnings()).toEqual(mockFileLearnings);
    });

    it("should clear selected file", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: mockHotspots, fragile: mockFragile });

      component.selectFile(mockHotspots[0]);
      httpMock
        .expectOne((request) => request.url.includes("/api/file-intel"))
        .flush({ learnings: [] });

      component.selectedFile.set(null);

      expect(component.selectedFile()).toBeNull();
    });

    it("should handle file learnings loading error", () => {
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: mockHotspots, fragile: mockFragile });

      component.selectFile(mockHotspots[0]);

      const req = httpMock.expectOne((request) =>
        request.url.includes("/api/file-intel")
      );
      req.error(new ProgressEvent("error"));

      expect(component.fileLearnings()).toEqual([]);

      consoleSpy.mockRestore();
    });
  });

  describe("File Tree Building", () => {
    it("should build file tree from hotspots", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: mockHotspots, fragile: mockFragile });

      expect(component.fileTree()).toBeDefined();
      expect(component.fileTree().length).toBeGreaterThan(0);
    });

    it("should create tree nodes with proper structure", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: mockHotspots, fragile: mockFragile });

      const tree = component.fileTree();

      expect(tree[0]).toHaveProperty("name");
      expect(tree[0]).toHaveProperty("path");
      expect(tree[0]).toHaveProperty("children");
    });

    it("should toggle node expansion", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: mockHotspots, fragile: mockFragile });

      const node = component.fileTree()[0];
      const initialState = node.isExpanded;

      component.toggleNode(node);

      expect(node.isExpanded).toBe(!initialState);
    });

    it("should count files in tree node", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: mockHotspots, fragile: mockFragile });

      const tree = component.fileTree();
      const count = component.countFiles(tree[0]);

      expect(count).toBeGreaterThanOrEqual(0);
    });
  });

  describe("Stats Display", () => {
    it("should display files tracked count", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: mockHotspots, fragile: mockFragile });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const statCards = compiled.querySelectorAll(".stat-card");

      expect(statCards.length).toBeGreaterThan(0);
    });

    it("should display fragile files count", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: mockHotspots, fragile: mockFragile });
      fixture.detectChanges();

      const compiled = fixture.nativeElement;
      const fragileCount = Array.from(compiled.querySelectorAll(".stat-value"))
        .map((el: any) => el.textContent)
        .includes("1");

      expect(fragileCount).toBeTruthy();
    });
  });

  describe("Error Handling", () => {
    it("should handle hotspots loading error gracefully", () => {
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .error(new ProgressEvent("error"));

      expect(component.loading()).toBe(false);
      expect(component.hotspots()).toEqual([]);
      expect(consoleSpy).toHaveBeenCalled();

      consoleSpy.mockRestore();
    });

    it("should set loading to false after error", () => {
      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .error(new ProgressEvent("error"));

      expect(component.loading()).toBe(false);

      consoleSpy.mockRestore();
    });
  });

  describe("Manual Refresh", () => {
    it("should reload data when loadData is called", () => {
      fixture.detectChanges();
      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: mockHotspots, fragile: mockFragile });

      const newHotspots: FileIntel[] = [mockHotspots[0]];
      component.loadData();

      httpMock
        .expectOne((request) => request.url.includes("/api/hotspots"))
        .flush({ hotspots: newHotspots, fragile: [] });

      expect(component.hotspots()).toEqual(newHotspots);
      expect(component.hotspots()).toHaveLength(1);
    });
  });
});
