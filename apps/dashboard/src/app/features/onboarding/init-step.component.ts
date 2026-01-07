import { Component, inject, signal, output } from "@angular/core";
import { FormsModule } from "@angular/forms";
import { OnboardingService } from "./onboarding.service";

@Component({
  selector: "app-init-step",
  standalone: true,
  imports: [FormsModule],
  template: `
    <div class="init-container" role="form" aria-label="Initialize Palace">
      <h2>Initialize Your Palace</h2>
      <p>Give your Mind Palace a name to get started</p>

      <div class="form">
        <label for="projectName">Project Name</label>
        <input
          id="projectName"
          type="text"
          [(ngModel)]="projectName"
          placeholder="e.g., My Awesome Project"
          (input)="validateName()"
          [class.invalid]="!isValid()"
          aria-invalid="{{ !isValid() }}"
          aria-describedby="nameHelp"
        />

        @if (!isValid() && projectName().length > 0) {
        <span id="nameHelp" class="error">
          Project name must be at least 3 characters
        </span>
        }

        <div class="actions">
          <button
            class="primary"
            [disabled]="!isValid() || loading()"
            (click)="initialize()"
            aria-busy="{{ loading() }}"
          >
            @if (loading()) {
            <span class="spinner" aria-hidden="true"></span>
            Initializing... } @else { Initialize Project }
          </button>

          <button class="secondary" (click)="back.emit()">Back</button>
        </div>

        @if (error()) {
        <div class="error-message" role="alert">
          {{ error() }}
        </div>
        }
      </div>
    </div>
  `,
  styles: [],
})
export class InitStepComponent {
  private onboarding = inject(OnboardingService);

  projectName = signal("");
  loading = signal(false);
  error = signal<string | null>(null);

  next = output<void>();
  back = output<void>();

  isValid(): boolean {
    return this.projectName().trim().length >= 3;
  }

  validateName(): void {
    this.error.set(null);
  }

  async initialize(): Promise<void> {
    if (!this.isValid()) return;

    this.loading.set(true);
    this.error.set(null);

    try {
      await this.onboarding.initializeProject(this.projectName().trim());
      this.next.emit();
    } catch (err: any) {
      this.error.set(err?.message || "Initialization failed");
    } finally {
      this.loading.set(false);
    }
  }
}
