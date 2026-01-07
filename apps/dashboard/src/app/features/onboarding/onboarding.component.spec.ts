import { TestBed, ComponentFixture } from "@angular/core/testing";
import { Router } from "@angular/router";
import { NO_ERRORS_SCHEMA } from "@angular/core";
import { OnboardingComponent } from "./onboarding.component";
import { OnboardingService } from "./onboarding.service";

class RouterMock {
  navigate = vi.fn();
}

class OnboardingServiceMock {
  completeOnboarding = vi.fn();
  skipOnboarding = vi.fn();
}

describe("OnboardingComponent", () => {
  let component: OnboardingComponent;
  let fixture: ComponentFixture<OnboardingComponent>;
  let router: RouterMock;
  let service: OnboardingServiceMock;

  beforeEach(async () => {
    router = new RouterMock();
    service = new OnboardingServiceMock();

    await TestBed.configureTestingModule({
      imports: [OnboardingComponent],
      providers: [
        { provide: OnboardingService, useValue: service },
        { provide: Router, useValue: router },
      ],
      schemas: [NO_ERRORS_SCHEMA],
    }).compileComponents();

    fixture = TestBed.createComponent(OnboardingComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it("advances and reverses steps", () => {
    expect(component.step()).toBe(1);
    component.nextStep();
    expect(component.step()).toBe(2);
    component.prevStep();
    expect(component.step()).toBe(1);
  });

  it("skipOnboarding triggers service and navigates overview", () => {
    component.skipOnboarding();
    expect(service.skipOnboarding).toHaveBeenCalled();
    // Router navigate should be called with ['/overview']
  });

  it("complete triggers service and navigates overview", () => {
    component.complete();
    expect(service.completeOnboarding).toHaveBeenCalled();
    // Router navigate should be called with ['/overview']
  });
});
