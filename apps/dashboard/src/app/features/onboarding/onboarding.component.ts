import { Component, inject, signal } from "@angular/core";
import { Router } from "@angular/router";
import { OnboardingService } from "./onboarding.service";
import { WelcomeStepComponent } from "./welcome-step.component";
import { InitStepComponent } from "./init-step.component";
import { SampleStepComponent } from "./sample-step.component";

@Component({
  selector: "app-onboarding",
  standalone: true,
  imports: [WelcomeStepComponent, InitStepComponent, SampleStepComponent],
  template: `
    <div class="onboarding-wrapper">
      <div class="progress-bar" aria-label="Progress">
        <div
          class="step"
          [class.active]="step() >= 1"
          [class.complete]="step() > 1"
        >
          <span class="number" aria-hidden="true">1</span>
          <span class="label">Welcome</span>
        </div>
        <div
          class="step"
          [class.active]="step() >= 2"
          [class.complete]="step() > 2"
        >
          <span class="number" aria-hidden="true">2</span>
          <span class="label">Initialize</span>
        </div>
        <div
          class="step"
          [class.active]="step() >= 3"
          [class.complete]="step() > 3"
        >
          <span class="number" aria-hidden="true">3</span>
          <span class="label">Sample</span>
        </div>
      </div>

      <div class="step-content">
        @switch (step()) { @case (1) {
        <app-welcome-step (next)="nextStep()" (skip)="skipOnboarding()">
        </app-welcome-step>
        } @case (2) {
        <app-init-step (next)="nextStep()" (back)="prevStep()"> </app-init-step>
        } @case (3) {
        <app-sample-step (next)="complete()" (back)="prevStep()">
        </app-sample-step>
        } }
      </div>
    </div>
  `,
  styles: [],
})
export class OnboardingComponent {
  private router = inject(Router);
  private onboarding = inject(OnboardingService);

  step = signal(1);

  nextStep(): void {
    this.step.update((s) => s + 1);
  }

  prevStep(): void {
    this.step.update((s) => Math.max(1, s - 1));
  }

  skipOnboarding(): void {
    this.onboarding.skipOnboarding();
    this.router.navigate(["/overview"]);
  }

  complete(): void {
    this.onboarding.completeOnboarding();
    this.router.navigate(["/overview"]);
  }
}
