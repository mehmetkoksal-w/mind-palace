import { Component, output } from "@angular/core";

@Component({
  selector: "app-welcome-step",
  standalone: true,
  template: `
    <div class="welcome-container" role="region" aria-label="Welcome">
      <div class="hero">
        <img src="assets/logo/logo.svg" alt="Mind Palace" class="logo" />
        <h1>Welcome to Mind Palace</h1>
        <p class="tagline">Your AI-powered second brain for code</p>
      </div>

      <div class="features" aria-label="Highlights">
        <div class="feature">
          <span class="icon" aria-hidden="true">🧠</span>
          <h3>Deep Knowledge</h3>
          <p>Capture decisions, learnings, and insights as you code</p>
        </div>

        <div class="feature">
          <span class="icon" aria-hidden="true">🔍</span>
          <h3>Semantic Search</h3>
          <p>Find relevant knowledge across your entire codebase</p>
        </div>

        <div class="feature">
          <span class="icon" aria-hidden="true">📊</span>
          <h3>Neural Maps</h3>
          <p>Visualize relationships and dependencies</p>
        </div>

        <div class="feature">
          <span class="icon" aria-hidden="true">🤖</span>
          <h3>AI Integration</h3>
          <p>LLM-powered insights and context generation</p>
        </div>
      </div>

      <div class="actions">
        <button class="primary" (click)="next.emit()" aria-label="Get Started">
          Get Started
        </button>
        <button
          class="secondary"
          (click)="skip.emit()"
          aria-label="Skip Tutorial"
        >
          Skip Tutorial
        </button>
      </div>
    </div>
  `,
  styles: [],
})
export class WelcomeStepComponent {
  next = output<void>();
  skip = output<void>();
}
