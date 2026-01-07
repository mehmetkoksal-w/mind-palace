import { beforeAll, afterEach, vi } from "vitest";
import "@testing-library/jest-dom/vitest";
import { TestBed } from "@angular/core/testing";
import {
  BrowserDynamicTestingModule,
  platformBrowserDynamicTesting,
} from "@angular/platform-browser-dynamic/testing";
import { ɵresetCompiledComponents } from "@angular/core";

// Mock all SCSS imports
vi.mock("*.scss", () => ({}));

// Initialize Angular testing environment
beforeAll(() => {
  // Initialize TestBed once for all tests
  TestBed.initTestEnvironment(
    BrowserDynamicTestingModule,
    platformBrowserDynamicTesting(),
    {
      teardown: { destroyAfterEach: true },
    }
  );

  // Mock window.matchMedia for responsive components
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => true,
    }),
  });

  // Mock IntersectionObserver for lazy loading
  global.IntersectionObserver = class IntersectionObserver {
    constructor() {}
    disconnect() {}
    observe() {}
    takeRecords() {
      return [];
    }
    unobserve() {}
  } as any;

  // Mock ResizeObserver for responsive components
  global.ResizeObserver = class ResizeObserver {
    constructor() {}
    disconnect() {}
    observe() {}
    unobserve() {}
  } as any;
});

// Clean up after each test
afterEach(() => {
  // Clear any timers
  vi.clearAllTimers();
  // Clear all mocks
  vi.clearAllMocks();
  // Reset TestBed modules
  TestBed.resetTestingModule();
});
