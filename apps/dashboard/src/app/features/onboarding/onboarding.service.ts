import { Injectable, inject } from "@angular/core";
import { HttpClient } from "@angular/common/http";
import { BehaviorSubject } from "rxjs";

export interface OnboardingState {
  currentStep: "welcome" | "init" | "sample" | "complete";
  projectName?: string;
  initialized: boolean;
  sampleCreated: boolean;
}

@Injectable({ providedIn: "root" })
export class OnboardingService {
  private http = inject(HttpClient);

  private stateSubject = new BehaviorSubject<OnboardingState>({
    currentStep: "welcome",
    initialized: false,
    sampleCreated: false,
  });

  state$ = this.stateSubject.asObservable();

  async checkFirstRun(): Promise<boolean> {
    try {
      const response = await this.http.get<any>("/api/status").toPromise();
      return !response?.initialized;
    } catch {
      return true;
    }
  }

  async initializeProject(projectName: string): Promise<void> {
    await this.http.post("/api/init", { name: projectName }).toPromise();
    this.updateState({ initialized: true, projectName, currentStep: "sample" });
  }

  async createSampleRoom(): Promise<void> {
    const sampleRoom = {
      name: "Getting Started",
      summary: "Your first Mind Palace room - exploring the basics",
      description:
        "This sample room demonstrates how to organize knowledge in Mind Palace.",
      entryPoints: ["README.md"],
    };

    await this.http.post("/api/rooms", sampleRoom).toPromise();

    await this.http
      .post("/api/knowledge", {
        room: "Getting Started",
        title: "Welcome to Mind Palace",
        content:
          "Mind Palace helps you build and maintain deep knowledge about your codebase...",
        type: "learning",
      })
      .toPromise();

    this.updateState({ sampleCreated: true, currentStep: "complete" });
  }

  private updateState(partial: Partial<OnboardingState>): void {
    this.stateSubject.next({
      ...this.stateSubject.value,
      ...partial,
    });
  }

  completeOnboarding(): void {
    localStorage.setItem("onboarding_complete", "true");
  }

  skipOnboarding(): void {
    this.completeOnboarding();
  }
}
