import { Component, inject, signal, output } from "@angular/core";
import { OnboardingService } from "./onboarding.service";

@Component({
  selector: "app-sample-step",
  standalone: true,
  template: `
    <div class="sample-container" role="region" aria-label="Create Sample Room">
      <h2>Create Your First Room</h2>
      <p>Let's create a sample room to show you how Mind Palace works</p>

      <div class="preview">
        <h3>What we'll create:</h3>
        <ul>
          <li>📁 <strong>Getting Started</strong> room</li>
          <li>📝 Sample learning: "Welcome to Mind Palace"</li>
          <li>🔗 Example relationships and links</li>
          <li>📊 Neural map visualization</li>
        </ul>
      </div>

      <div class="actions">
        <button
          class="primary"
          [disabled]="loading()"
          (click)="createSample()"
          aria-busy="{{ loading() }}"
        >
          @if (loading()) {
          <span class="spinner" aria-hidden="true"></span>
          Creating... } @else { Create Sample Room }
        </button>

        <button class="secondary" (click)="skipSample()">Skip Sample</button>

        <button class="tertiary" (click)="back.emit()">Back</button>
      </div>

      @if (error()) {
      <div class="error-message" role="alert">{{ error() }}</div>
      }
    </div>
  `,
  styles: [],
})
export class SampleStepComponent {
  private onboarding = inject(OnboardingService);

  loading = signal(false);
  error = signal<string | null>(null);

  next = output<void>();
  back = output<void>();

  async createSample(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      await this.onboarding.createSampleRoom();
      this.next.emit();
    } catch (err: any) {
      this.error.set(err?.message || "Failed to create sample");
    } finally {
      this.loading.set(false);
    }
  }

  skipSample(): void {
    this.onboarding.skipOnboarding();
    this.next.emit();
  }
}
