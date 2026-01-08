import { provideZoneChangeDetection } from "@angular/core";
import { bootstrapApplication } from "@angular/platform-browser";
import { provideRouter } from "@angular/router";
import { provideHttpClient } from "@angular/common/http";
import { AppComponent } from "./app/app.component";
import { routes } from "./app/app.routes";
import { LoggerService } from "./app/core/services/logger.service";

bootstrapApplication(AppComponent, {
  providers: [
    provideZoneChangeDetection(),
    provideRouter(routes),
    provideHttpClient(),
  ],
}).catch((err) => {
  // Use console.error for bootstrap failures since DI isn't available yet
  console.error("[Bootstrap] Application failed to start:", err);
});
