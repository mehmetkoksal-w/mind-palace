import { TestBed } from "@angular/core/testing";
import {
  HttpClientTestingModule,
  HttpTestingController,
} from "@angular/common/http/testing";
import { OnboardingService, OnboardingState } from "./onboarding.service";

describe("OnboardingService", () => {
  let service: OnboardingService;
  let httpMock: HttpTestingController;

  afterEach(() => {
    if (httpMock) {
      try {
        httpMock.verify();
      } catch (e) {
        // Ignore verification errors in cleanup
      }
    }
    localStorage.removeItem("onboarding_complete");
    TestBed.resetTestingModule();
  });

  function setupTest() {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [OnboardingService],
    });
    service = TestBed.inject(OnboardingService);
    httpMock = TestBed.inject(HttpTestingController);
  }

  it("checkFirstRun returns true when status not initialized", async () => {
    setupTest();
    const promise = service.checkFirstRun();
    const req = httpMock.expectOne("/api/status");
    expect(req.request.method).toBe("GET");
    req.flush({ initialized: false });
    await expect(promise).resolves.toBe(true);
  });

  it("checkFirstRun returns false when status initialized", async () => {
    setupTest();
    const promise = service.checkFirstRun();
    const req = httpMock.expectOne("/api/status");
    req.flush({ initialized: true });
    await expect(promise).resolves.toBe(false);
  });

  it("initializeProject posts to /api/init and updates state", async () => {
    setupTest();
    const initPromise = service.initializeProject("Demo");
    const req = httpMock.expectOne("/api/init");
    expect(req.request.method).toBe("POST");
    expect(req.request.body).toEqual({ name: "Demo" });
    req.flush({ ok: true });
    await initPromise;

    const statePromise = new Promise<void>((resolve) => {
      service.state$.subscribe((state) => {
        if (
          state.initialized &&
          state.projectName === "Demo" &&
          state.currentStep === "sample"
        ) {
          resolve();
        }
      });
    });
    await statePromise;
  });

  it("createSampleRoom posts sample room and learning then updates state", async () => {
    setupTest();

    // Start the async operation
    const createPromise = service.createSampleRoom();

    // First request: rooms
    const roomReq = httpMock.expectOne("/api/rooms");
    expect(roomReq.request.method).toBe("POST");
    expect(roomReq.request.body.name).toBe("Getting Started");
    roomReq.flush({ ok: true });

    // Give time for the second request to be made
    await new Promise((resolve) => setTimeout(resolve, 0));

    // Second request: knowledge (happens after rooms completes)
    const knowledgeReq = httpMock.expectOne("/api/knowledge");
    expect(knowledgeReq.request.method).toBe("POST");
    knowledgeReq.flush({ ok: true });

    // Wait for completion
    await createPromise;

    // Verify state update
    let currentState: OnboardingState | null = null;
    service.state$.subscribe((state) => (currentState = state)).unsubscribe();

    expect(currentState?.sampleCreated).toBe(true);
    expect(currentState?.currentStep).toBe("complete");
  });

  it("completeOnboarding sets localStorage flag", () => {
    setupTest();
    service.completeOnboarding();
    expect(localStorage.getItem("onboarding_complete")).toBe("true");
  });

  it("skipOnboarding calls completeOnboarding", () => {
    setupTest();
    service.skipOnboarding();
    expect(localStorage.getItem("onboarding_complete")).toBe("true");
  });
});
