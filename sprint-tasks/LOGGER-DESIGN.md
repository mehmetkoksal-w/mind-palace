# Production-Ready Logging Service Design

**Created:** January 6, 2026  
**Status:** Design Phase  
**Target:** Dashboard (Angular 21) & VS Code Extension (TypeScript)

---

## 📊 Current State Analysis

### Dashboard Logging Audit (17 console statements)

| File                       | Line | Type          | Severity | Context                    | Action            |
| -------------------------- | ---- | ------------- | -------- | -------------------------- | ----------------- |
| main.ts                    | 13   | console.error | ERROR    | Bootstrap failure          | Convert to logger |
| websocket.service.ts       | 45   | console.error | ERROR    | WebSocket error            | Convert to logger |
| websocket.service.ts       | 54   | console.error | ERROR    | Message parse failure      | Convert to logger |
| websocket.service.ts       | 58   | console.error | ERROR    | Connection failure         | Convert to logger |
| websocket.service.ts       | 88   | console.warn  | WARN     | Send when disconnected     | Convert to logger |
| overview.component.ts      | 205  | console.error | ERROR    | Stats load failure         | Convert to logger |
| overview.component.ts      | 212  | console.error | ERROR    | Agents load failure        | Convert to logger |
| sessions.component.ts      | 149  | console.error | ERROR    | Sessions load failure      | Convert to logger |
| learnings.component.ts     | 158  | console.error | ERROR    | Learnings load failure     | Convert to logger |
| intel.component.ts         | 763  | console.error | ERROR    | File intel load failure    | Convert to logger |
| corridors.component.ts     | 212  | console.error | ERROR    | Corridors load failure     | Convert to logger |
| corridors.component.ts     | 217  | console.error | ERROR    | Personal learnings failure | Convert to logger |
| conversations.component.ts | 837  | console.error | ERROR    | Conversations load failure | Convert to logger |
| conversations.component.ts | 853  | console.error | ERROR    | Conversation load failure  | Convert to logger |
| conversations.component.ts | 872  | console.error | ERROR    | Timeline load failure      | Convert to logger |
| conversations.component.ts | 902  | console.error | ERROR    | Search failure             | Convert to logger |
| neural-map.component.ts    | 423  | console.error | ERROR    | Neural map data failure    | Convert to logger |

**Pattern Analysis:**

- 16/17 are error-level API/data loading failures
- 1 is a warning for state validation
- All should be converted to structured logging
- Need user-facing error handling (toast notifications)
- Backend logging needed for production monitoring

### VS Code Extension Audit (9 console statements)

| File                        | Line | Type          | Severity | Context                   | Action                   |
| --------------------------- | ---- | ------------- | -------- | ------------------------- | ------------------------ |
| test/suite/index.ts         | 36   | console.error | ERROR    | Test runner error         | Keep (test infra)        |
| test/runTests.ts            | 22   | console.error | ERROR    | Test execution error      | Keep (test infra)        |
| sidebar.ts                  | 130  | console.error | ERROR    | MCP connection failure    | Convert to OutputChannel |
| sidebar.ts                  | 180  | console.error | ERROR    | Search failure            | Convert to OutputChannel |
| sidebar.ts                  | 294  | console.error | ERROR    | Room file parse error     | Convert to OutputChannel |
| decorator.ts                | 64   | console.error | ERROR    | Context pack parse error  | Convert to OutputChannel |
| fileIntelligenceProvider.ts | 145  | console.error | ERROR    | File intel API failure    | Convert to OutputChannel |
| config.ts                   | 126  | console.warn  | WARN     | Palace.jsonc parse errors | Convert to OutputChannel |
| config.ts                   | 132  | console.error | ERROR    | Palace.jsonc read error   | Convert to OutputChannel |

**Pattern Analysis:**

- 7/9 need conversion (2 are test infrastructure)
- Mix of parse errors, API failures, and config issues
- Need OutputChannel integration for debugging
- Some errors should show VS Code notifications
- Debug level needed for development workflow

---

## 🏗️ Architecture Overview

### Design Principles

1. **Structured Logging:** Consistent format with metadata (timestamp, level, context)
2. **Environment Awareness:** Different behavior for dev vs. production
3. **Performance:** Zero-cost abstraction in production when logging disabled
4. **Type Safety:** TypeScript interfaces for log metadata
5. **Extensibility:** Easy to add new transports (file, remote, etc.)
6. **Context Enrichment:** Automatic injection of component/module context
7. **Correlation:** Request IDs, session IDs for tracing

### Log Levels (Standard Severity)

```typescript
enum LogLevel {
  DEBUG = 0, // Verbose dev info, disabled in production
  INFO = 1, // General informational messages
  WARN = 2, // Warning conditions, potential issues
  ERROR = 3, // Error conditions, failures
  FATAL = 4, // Critical failures requiring immediate attention
}
```

---

## 📱 Dashboard Logger Service Design

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     Application Layer                       │
│  (Components, Services, Guards, Interceptors)              │
└──────────────────────┬──────────────────────────────────────┘
                       │ inject(LoggerService)
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                   LoggerService (Root)                       │
│  • Log level filtering                                      │
│  • Context enrichment (component name, metadata)            │
│  • Rate limiting / batching                                 │
└──────────────────────┬──────────────────────────────────────┘
                       │
         ┌─────────────┴─────────────┬─────────────────┐
         ▼                           ▼                 ▼
┌─────────────────┐      ┌──────────────────┐   ┌────────────┐
│ ConsoleTransport│      │  RemoteTransport │   │   Future   │
│  (Development)  │      │   (Production)   │   │ Transports │
│                 │      │                  │   │            │
│ • Pretty print  │      │ • POST /api/logs │   │ • File     │
│ • Color coding  │      │ • Batching       │   │ • Analytics│
│ • Stack traces  │      │ • Retry logic    │   │ • Sentry   │
└─────────────────┘      └──────────────────┘   └────────────┘
```

### Core Service Implementation

#### File: `apps/dashboard/src/app/core/services/logger.service.ts`

```typescript
import { Injectable, inject, isDevMode, InjectionToken } from "@angular/core";
import { HttpClient, HttpErrorResponse } from "@angular/common/http";
import { Observable, Subject, catchError, of, timer } from "rxjs";
import { bufferTime, filter, mergeMap } from "rxjs/operators";

// ============================================================================
// Types & Interfaces
// ============================================================================

export enum LogLevel {
  DEBUG = 0,
  INFO = 1,
  WARN = 2,
  ERROR = 3,
  FATAL = 4,
}

export interface LogEntry {
  timestamp: string;
  level: LogLevel;
  message: string;
  context?: string;
  metadata?: Record<string, any>;
  error?: Error | unknown;
  stackTrace?: string;
  sessionId?: string;
  userId?: string;
}

export interface LoggerConfig {
  minLevel: LogLevel;
  enableConsole: boolean;
  enableRemote: boolean;
  remoteEndpoint?: string;
  batchSize?: number;
  batchIntervalMs?: number;
  includeStackTrace?: boolean;
}

// Default configurations
export const LOGGER_CONFIG = new InjectionToken<LoggerConfig>("LOGGER_CONFIG");

export const DEFAULT_LOGGER_CONFIG: LoggerConfig = {
  minLevel: isDevMode() ? LogLevel.DEBUG : LogLevel.INFO,
  enableConsole: true,
  enableRemote: !isDevMode(),
  remoteEndpoint: "/api/logs",
  batchSize: 10,
  batchIntervalMs: 5000,
  includeStackTrace: isDevMode(),
};

// ============================================================================
// Logger Service
// ============================================================================

@Injectable({ providedIn: "root" })
export class LoggerService {
  private readonly http = inject(HttpClient, { optional: true });
  private readonly config =
    inject(LOGGER_CONFIG, { optional: true }) ?? DEFAULT_LOGGER_CONFIG;

  private readonly logStream = new Subject<LogEntry>();
  private sessionId = this.generateSessionId();

  constructor() {
    this.initializeRemoteLogging();
  }

  // --------------------------------------------------------------------------
  // Public API
  // --------------------------------------------------------------------------

  debug(
    message: string,
    context?: string,
    metadata?: Record<string, any>
  ): void {
    this.log(LogLevel.DEBUG, message, context, metadata);
  }

  info(
    message: string,
    context?: string,
    metadata?: Record<string, any>
  ): void {
    this.log(LogLevel.INFO, message, context, metadata);
  }

  warn(
    message: string,
    context?: string,
    metadata?: Record<string, any>
  ): void {
    this.log(LogLevel.WARN, message, context, metadata);
  }

  error(
    message: string,
    error?: Error | unknown,
    context?: string,
    metadata?: Record<string, any>
  ): void {
    this.log(LogLevel.ERROR, message, context, { ...metadata, error });
  }

  fatal(
    message: string,
    error?: Error | unknown,
    context?: string,
    metadata?: Record<string, any>
  ): void {
    this.log(LogLevel.FATAL, message, context, { ...metadata, error });
  }

  /**
   * Create a child logger with automatic context injection
   * Usage: private logger = this.loggerService.forContext('MyComponent');
   */
  forContext(context: string): ContextLogger {
    return new ContextLogger(this, context);
  }

  // --------------------------------------------------------------------------
  // Core Logging
  // --------------------------------------------------------------------------

  private log(
    level: LogLevel,
    message: string,
    context?: string,
    metadata?: Record<string, any>
  ): void {
    // Filter by minimum level
    if (level < this.config.minLevel) {
      return;
    }

    const entry: LogEntry = {
      timestamp: new Date().toISOString(),
      level,
      message,
      context,
      metadata,
      sessionId: this.sessionId,
    };

    // Extract error and stack trace if present
    if (metadata?.error) {
      entry.error = metadata.error;
      if (this.config.includeStackTrace && metadata.error instanceof Error) {
        entry.stackTrace = metadata.error.stack;
      }
    }

    // Console transport
    if (this.config.enableConsole) {
      this.logToConsole(entry);
    }

    // Remote transport (via stream)
    if (this.config.enableRemote) {
      this.logStream.next(entry);
    }
  }

  // --------------------------------------------------------------------------
  // Console Transport
  // --------------------------------------------------------------------------

  private logToConsole(entry: LogEntry): void {
    const prefix = `[${entry.timestamp}] [${LogLevel[entry.level]}]`;
    const contextStr = entry.context ? ` [${entry.context}]` : "";
    const fullMessage = `${prefix}${contextStr} ${entry.message}`;

    const style = this.getConsoleStyle(entry.level);
    const logFn = this.getConsoleMethod(entry.level);

    if (entry.metadata || entry.error) {
      logFn(fullMessage, entry.metadata, entry.error);
    } else {
      logFn(fullMessage);
    }

    // Stack trace in development
    if (entry.stackTrace && isDevMode()) {
      console.groupCollapsed("Stack Trace");
      console.log(entry.stackTrace);
      console.groupEnd();
    }
  }

  private getConsoleMethod(level: LogLevel): (...args: any[]) => void {
    switch (level) {
      case LogLevel.DEBUG:
        return console.debug.bind(console);
      case LogLevel.INFO:
        return console.info.bind(console);
      case LogLevel.WARN:
        return console.warn.bind(console);
      case LogLevel.ERROR:
      case LogLevel.FATAL:
        return console.error.bind(console);
      default:
        return console.log.bind(console);
    }
  }

  private getConsoleStyle(level: LogLevel): string {
    const styles: Record<LogLevel, string> = {
      [LogLevel.DEBUG]: "color: #888",
      [LogLevel.INFO]: "color: #0066cc",
      [LogLevel.WARN]: "color: #ff9900",
      [LogLevel.ERROR]: "color: #cc0000; font-weight: bold",
      [LogLevel.FATAL]: "color: #fff; background: #cc0000; font-weight: bold",
    };
    return styles[level] || "";
  }

  // --------------------------------------------------------------------------
  // Remote Transport
  // --------------------------------------------------------------------------

  private initializeRemoteLogging(): void {
    if (!this.config.enableRemote || !this.http) {
      return;
    }

    // Batch logs and send periodically
    this.logStream
      .pipe(
        bufferTime(this.config.batchIntervalMs!, this.config.batchSize!),
        filter((logs) => logs.length > 0),
        mergeMap((logs) => this.sendLogsToBackend(logs))
      )
      .subscribe({
        error: (err) => console.error("[Logger] Remote transport error:", err),
      });
  }

  private sendLogsToBackend(logs: LogEntry[]): Observable<void> {
    if (!this.http || !this.config.remoteEndpoint) {
      return of(undefined);
    }

    const payload = {
      logs: logs.map((log) => ({
        ...log,
        // Serialize error objects
        error:
          log.error instanceof Error
            ? { message: log.error.message, name: log.error.name }
            : log.error,
      })),
    };

    return this.http.post<void>(this.config.remoteEndpoint, payload).pipe(
      catchError((error: HttpErrorResponse) => {
        // Fallback to console if backend fails (avoid infinite loop)
        console.error("[Logger] Failed to send logs to backend:", error);
        return of(undefined);
      })
    );
  }

  // --------------------------------------------------------------------------
  // Utilities
  // --------------------------------------------------------------------------

  private generateSessionId(): string {
    return `session-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }
}

// ============================================================================
// Context Logger (Convenience Wrapper)
// ============================================================================

export class ContextLogger {
  constructor(
    private readonly logger: LoggerService,
    private readonly context: string
  ) {}

  debug(message: string, metadata?: Record<string, any>): void {
    this.logger.debug(message, this.context, metadata);
  }

  info(message: string, metadata?: Record<string, any>): void {
    this.logger.info(message, this.context, metadata);
  }

  warn(message: string, metadata?: Record<string, any>): void {
    this.logger.warn(message, this.context, metadata);
  }

  error(
    message: string,
    error?: Error | unknown,
    metadata?: Record<string, any>
  ): void {
    this.logger.error(message, error, this.context, metadata);
  }

  fatal(
    message: string,
    error?: Error | unknown,
    metadata?: Record<string, any>
  ): void {
    this.logger.fatal(message, error, this.context, metadata);
  }
}
```

### Configuration Provider

#### File: `apps/dashboard/src/app/app.config.ts`

```typescript
import {
  ApplicationConfig,
  isDevMode,
  provideZoneChangeDetection,
} from "@angular/core";
import { provideRouter } from "@angular/router";
import { provideHttpClient } from "@angular/common/http";
import { routes } from "./app.routes";
import {
  LOGGER_CONFIG,
  LogLevel,
  LoggerConfig,
} from "./core/services/logger.service";

const loggerConfig: LoggerConfig = {
  minLevel: isDevMode() ? LogLevel.DEBUG : LogLevel.WARN,
  enableConsole: true,
  enableRemote: !isDevMode(), // Only send to backend in production
  remoteEndpoint: "/api/logs",
  batchSize: 20,
  batchIntervalMs: 10000,
  includeStackTrace: isDevMode(),
};

export const appConfig: ApplicationConfig = {
  providers: [
    provideZoneChangeDetection({ eventCoalescing: true }),
    provideRouter(routes),
    provideHttpClient(),
    { provide: LOGGER_CONFIG, useValue: loggerConfig },
  ],
};
```

### Usage Examples

#### Basic Component Usage

```typescript
import { Component, inject, OnInit } from "@angular/core";
import { LoggerService } from "@core/services/logger.service";
import { ApiService } from "@core/services/api.service";

@Component({
  selector: "app-overview",
  templateUrl: "./overview.component.html",
})
export class OverviewComponent implements OnInit {
  private readonly api = inject(ApiService);
  private readonly logger =
    inject(LoggerService).forContext("OverviewComponent");

  ngOnInit(): void {
    this.logger.info("Component initialized");
    this.loadStats();
  }

  private loadStats(): void {
    this.logger.debug("Loading stats...");

    this.api.getStats().subscribe({
      next: (data) => {
        this.logger.info("Stats loaded successfully", { count: data.total });
        this.stats.set(data);
      },
      error: (err) => {
        this.logger.error("Failed to load stats", err, {
          endpoint: "/api/stats",
          retryAttempt: 0,
        });
        // Show user-facing error (toast/snackbar)
      },
    });
  }
}
```

#### Service Usage (WebSocket)

```typescript
import { Injectable, inject } from "@angular/core";
import { LoggerService } from "./logger.service";

@Injectable({ providedIn: "root" })
export class WebSocketService {
  private readonly logger =
    inject(LoggerService).forContext("WebSocketService");
  private socket: WebSocket | null = null;

  connect(): void {
    const wsUrl = `ws://${window.location.host}/api/ws`;
    this.logger.info("Connecting to WebSocket", { url: wsUrl });

    try {
      this.socket = new WebSocket(wsUrl);

      this.socket.onopen = () => {
        this.logger.info("WebSocket connected");
        this.connected.set(true);
      };

      this.socket.onerror = (error) => {
        this.logger.error("WebSocket error", error, { url: wsUrl });
      };

      this.socket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          this.eventsSubject.next(data);
        } catch (e) {
          this.logger.error("Failed to parse WebSocket message", e, {
            rawData: event.data?.substring(0, 100),
          });
        }
      };
    } catch (e) {
      this.logger.error("Failed to create WebSocket connection", e, {
        url: wsUrl,
      });
      this.attemptReconnect();
    }
  }
}
```

---

## 🔌 VS Code Extension Logger Design

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│              VS Code Extension Components                   │
│  (Commands, Providers, Webviews, etc.)                     │
└──────────────────────┬──────────────────────────────────────┘
                       │ import { logger }
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                 ExtensionLogger (Singleton)                  │
│  • Log level filtering                                      │
│  • Context enrichment                                       │
│  • Timestamp formatting                                     │
└──────────────────────┬──────────────────────────────────────┘
                       │
         ┌─────────────┴─────────────┬─────────────────┐
         ▼                           ▼                 ▼
┌─────────────────┐      ┌──────────────────┐   ┌────────────┐
│  OutputChannel  │      │   Notifications  │   │   Future   │
│  (Dev Console)  │      │  (User Visible)  │   │            │
│                 │      │                  │   │ • File log │
│ • Formatted     │      │ • Errors/Warns   │   │ • Telemetry│
│ • Filterable    │      │ • showErrorMsg() │   │            │
└─────────────────┘      └──────────────────┘   └────────────┘
```

### Core Logger Implementation

#### File: `apps/vscode/src/logger.ts`

```typescript
import * as vscode from "vscode";

// ============================================================================
// Types & Interfaces
// ============================================================================

export enum LogLevel {
  DEBUG = 0,
  INFO = 1,
  WARN = 2,
  ERROR = 3,
  FATAL = 4,
}

export interface LogEntry {
  timestamp: Date;
  level: LogLevel;
  message: string;
  context?: string;
  metadata?: Record<string, any>;
  error?: Error | unknown;
}

export interface LoggerConfig {
  minLevel: LogLevel;
  showTimestamp: boolean;
  showContext: boolean;
  showNotifications: boolean; // Show VS Code notifications for errors
  outputChannelName: string;
}

// ============================================================================
// Extension Logger (Singleton)
// ============================================================================

class ExtensionLogger {
  private outputChannel: vscode.OutputChannel | null = null;
  private config: LoggerConfig = {
    minLevel: LogLevel.DEBUG, // Will be updated based on extension mode
    showTimestamp: true,
    showContext: true,
    showNotifications: true,
    outputChannelName: "Mind Palace",
  };

  // --------------------------------------------------------------------------
  // Initialization
  // --------------------------------------------------------------------------

  /**
   * Initialize the logger (call in extension activate())
   */
  initialize(context: vscode.ExtensionContext): void {
    this.outputChannel = vscode.window.createOutputChannel(
      this.config.outputChannelName,
      "log" // Language ID for syntax highlighting
    );

    context.subscriptions.push(this.outputChannel);

    // Set log level based on extension mode
    const isDevelopment =
      context.extensionMode === vscode.ExtensionMode.Development;
    this.config.minLevel = isDevelopment ? LogLevel.DEBUG : LogLevel.INFO;

    this.info("Logger initialized", "ExtensionLogger", {
      mode: isDevelopment ? "development" : "production",
      minLevel: LogLevel[this.config.minLevel],
    });
  }

  /**
   * Update logger configuration
   */
  configure(config: Partial<LoggerConfig>): void {
    this.config = { ...this.config, ...config };
  }

  // --------------------------------------------------------------------------
  // Public API
  // --------------------------------------------------------------------------

  debug(
    message: string,
    context?: string,
    metadata?: Record<string, any>
  ): void {
    this.log(LogLevel.DEBUG, message, context, metadata);
  }

  info(
    message: string,
    context?: string,
    metadata?: Record<string, any>
  ): void {
    this.log(LogLevel.INFO, message, context, metadata);
  }

  warn(
    message: string,
    context?: string,
    metadata?: Record<string, any>
  ): void {
    this.log(LogLevel.WARN, message, context, metadata);
  }

  error(
    message: string,
    error?: Error | unknown,
    context?: string,
    metadata?: Record<string, any>
  ): void {
    this.log(LogLevel.ERROR, message, context, { ...metadata, error });

    // Show notification for errors
    if (this.config.showNotifications) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      vscode.window.showErrorMessage(
        `Mind Palace: ${message}${errorMsg ? ` - ${errorMsg}` : ""}`
      );
    }
  }

  fatal(
    message: string,
    error?: Error | unknown,
    context?: string,
    metadata?: Record<string, any>
  ): void {
    this.log(LogLevel.FATAL, message, context, { ...metadata, error });

    // Always show notification for fatal errors
    const errorMsg = error instanceof Error ? error.message : String(error);
    vscode.window
      .showErrorMessage(
        `Mind Palace (FATAL): ${message}${errorMsg ? ` - ${errorMsg}` : ""}`,
        "Show Logs"
      )
      .then((selection) => {
        if (selection === "Show Logs") {
          this.show();
        }
      });
  }

  /**
   * Create a child logger with automatic context
   */
  forContext(context: string): ContextLogger {
    return new ContextLogger(this, context);
  }

  /**
   * Show the output channel
   */
  show(): void {
    this.outputChannel?.show(true);
  }

  /**
   * Clear the output channel
   */
  clear(): void {
    this.outputChannel?.clear();
  }

  // --------------------------------------------------------------------------
  // Core Logging
  // --------------------------------------------------------------------------

  private log(
    level: LogLevel,
    message: string,
    context?: string,
    metadata?: Record<string, any>
  ): void {
    // Filter by minimum level
    if (level < this.config.minLevel) {
      return;
    }

    if (!this.outputChannel) {
      console.warn("[Logger] Not initialized - call initialize() first");
      return;
    }

    const entry: LogEntry = {
      timestamp: new Date(),
      level,
      message,
      context,
      metadata,
    };

    // Extract error if present
    if (metadata?.error) {
      entry.error = metadata.error;
    }

    const formattedMessage = this.formatLogEntry(entry);
    this.outputChannel.appendLine(formattedMessage);

    // Also log to debug console in development
    if (this.config.minLevel === LogLevel.DEBUG) {
      const consoleFn = this.getConsoleMethod(level);
      consoleFn(`[Mind Palace] ${formattedMessage}`);
    }
  }

  // --------------------------------------------------------------------------
  // Formatting
  // --------------------------------------------------------------------------

  private formatLogEntry(entry: LogEntry): string {
    const parts: string[] = [];

    // Timestamp
    if (this.config.showTimestamp) {
      parts.push(`[${this.formatTimestamp(entry.timestamp)}]`);
    }

    // Log level
    parts.push(`[${this.formatLogLevel(entry.level)}]`);

    // Context
    if (this.config.showContext && entry.context) {
      parts.push(`[${entry.context}]`);
    }

    // Message
    parts.push(entry.message);

    let output = parts.join(" ");

    // Metadata
    if (entry.metadata && Object.keys(entry.metadata).length > 0) {
      const metaCopy = { ...entry.metadata };
      delete metaCopy.error; // Handle error separately
      if (Object.keys(metaCopy).length > 0) {
        output += ` | ${JSON.stringify(metaCopy)}`;
      }
    }

    // Error details
    if (entry.error) {
      output += "\n  Error: ";
      if (entry.error instanceof Error) {
        output += `${entry.error.name}: ${entry.error.message}`;
        if (entry.error.stack) {
          output += `\n  Stack:\n${this.indentStack(entry.error.stack)}`;
        }
      } else {
        output += JSON.stringify(entry.error);
      }
    }

    return output;
  }

  private formatTimestamp(date: Date): string {
    return date.toISOString().replace("T", " ").substring(0, 23);
  }

  private formatLogLevel(level: LogLevel): string {
    const labels: Record<LogLevel, string> = {
      [LogLevel.DEBUG]: "DEBUG",
      [LogLevel.INFO]: "INFO ",
      [LogLevel.WARN]: "WARN ",
      [LogLevel.ERROR]: "ERROR",
      [LogLevel.FATAL]: "FATAL",
    };
    return labels[level];
  }

  private indentStack(stack: string): string {
    return stack
      .split("\n")
      .map((line) => `    ${line}`)
      .join("\n");
  }

  private getConsoleMethod(level: LogLevel): (...args: any[]) => void {
    switch (level) {
      case LogLevel.DEBUG:
        return console.debug.bind(console);
      case LogLevel.INFO:
        return console.info.bind(console);
      case LogLevel.WARN:
        return console.warn.bind(console);
      case LogLevel.ERROR:
      case LogLevel.FATAL:
        return console.error.bind(console);
      default:
        return console.log.bind(console);
    }
  }
}

// ============================================================================
// Context Logger (Convenience Wrapper)
// ============================================================================

class ContextLogger {
  constructor(
    private readonly logger: ExtensionLogger,
    private readonly context: string
  ) {}

  debug(message: string, metadata?: Record<string, any>): void {
    this.logger.debug(message, this.context, metadata);
  }

  info(message: string, metadata?: Record<string, any>): void {
    this.logger.info(message, this.context, metadata);
  }

  warn(message: string, metadata?: Record<string, any>): void {
    this.logger.warn(message, this.context, metadata);
  }

  error(
    message: string,
    error?: Error | unknown,
    metadata?: Record<string, any>
  ): void {
    this.logger.error(message, error, this.context, metadata);
  }

  fatal(
    message: string,
    error?: Error | unknown,
    metadata?: Record<string, any>
  ): void {
    this.logger.fatal(message, error, this.context, metadata);
  }
}

// ============================================================================
// Singleton Export
// ============================================================================

export const logger = new ExtensionLogger();
```

### Extension Activation Integration

#### File: `apps/vscode/src/extension.ts` (modifications)

```typescript
import * as vscode from "vscode";
import { logger } from "./logger";
import { PalaceBridge } from "./bridge";
// ... other imports

export function activate(context: vscode.ExtensionContext) {
  // Initialize logger FIRST
  logger.initialize(context);
  logger.info("Extension activating...", "Extension");

  // Check version compatibility
  warnIfIncompatible();

  // Initialize components
  const bridge = new PalaceBridge();
  const hud = new PalaceHUD();
  const decorator = new PalaceDecorator();

  logger.info("Core components initialized", "Extension", {
    bridgeReady: !!bridge,
    hudReady: !!hud,
    decoratorReady: !!decorator,
  });

  // ... rest of activation

  logger.info("Extension activated successfully", "Extension");
}

export function deactivate() {
  logger.info("Extension deactivating...", "Extension");
}
```

### Usage Examples

#### Basic Usage in Provider

```typescript
import * as vscode from "vscode";
import { logger } from "../logger";

export class FileIntelligenceProvider {
  private readonly logger = logger.forContext("FileIntelligenceProvider");

  async getFileIntel(filePath: string): Promise<FileIntel | null> {
    this.logger.debug("Fetching file intelligence", { filePath });

    try {
      const response = await fetch(
        `${this.baseUrl}/api/intel/files/${encodeURIComponent(filePath)}`
      );

      if (!response.ok) {
        this.logger.warn("File intel not found", {
          filePath,
          status: response.status,
        });
        return null;
      }

      const data = await response.json();
      this.logger.info("File intel retrieved", {
        filePath,
        editCount: data.editCount,
      });
      return data;
    } catch (error) {
      this.logger.error("Failed to get file intel", error, { filePath });
      return null;
    }
  }
}
```

#### Complex Error Handling

```typescript
import { logger } from "../logger";

export class PalaceConfig {
  private readonly logger = logger.forContext("PalaceConfig");

  async loadConfig(): Promise<Config | null> {
    const configPath = this.getConfigPath();
    this.logger.debug("Loading palace.jsonc", { path: configPath });

    try {
      const content = await vscode.workspace.fs.readFile(
        vscode.Uri.file(configPath)
      );
      const text = Buffer.from(content).toString("utf8");

      const { value, errors } = parseJSONC(text);

      if (errors.length > 0) {
        this.logger.warn("JSONC parse errors detected", {
          errorCount: errors.length,
          errors: errors.map((e) => e.message),
        });
      }

      this.logger.info("Configuration loaded successfully", {
        roomCount: value.rooms?.length ?? 0,
      });

      return value;
    } catch (error) {
      this.logger.error("Failed to load configuration", error, {
        path: configPath,
      });
      return null;
    }
  }
}
```

---

## 📋 Migration Plan

### Phase 1: Foundation (Week 1)

**Dashboard:**

1. ✅ Create `logger.service.ts`
2. ✅ Add to `app.config.ts` with providers
3. ✅ Add unit tests for logger service
4. ✅ Document in team wiki

**VS Code:**

1. ✅ Create `logger.ts`
2. ✅ Integrate in `extension.ts` activation
3. ✅ Add unit tests
4. ✅ Document usage patterns

### Phase 2: Critical Paths (Week 1-2)

**Priority Order (High Risk Areas):**

1. **WebSocket Service** (Dashboard)

   - Connection errors visible to users
   - Replace 4 console statements
   - Add structured error context

2. **Main Entry Points**

   - `main.ts` bootstrap error
   - Extension activation logging

3. **API Error Handling** (Dashboard)

   - All component API failures
   - 12 console.error statements
   - Add retry/circuit breaker context

4. **Configuration Loading** (VS Code)
   - Palace.jsonc parse errors
   - Critical for extension startup

### Phase 3: Component Migration (Week 2-3)

**Dashboard Components (systematic conversion):**

```
✅ WebSocketService
✅ main.ts
⬜ OverviewComponent (2 errors)
⬜ SessionsComponent (1 error)
⬜ LearningsComponent (1 error)
⬜ CorridorsComponent (2 errors)
⬜ ConversationsComponent (4 errors)
⬜ IntelComponent (1 error)
⬜ NeuralMapComponent (1 error)
```

**VS Code Files:**

```
✅ extension.ts
✅ config.ts (2 statements)
⬜ sidebar.ts (3 statements)
⬜ decorator.ts (1 statement)
⬜ fileIntelligenceProvider.ts (1 statement)
```

### Phase 4: Enhancement (Week 3-4)

1. **Add User-Facing Error Handling**

   - Toast/snackbar notifications (Dashboard)
   - VS Code information messages
   - Retry mechanisms

2. **Backend Integration** (Dashboard)

   - Implement `/api/logs` endpoint
   - Test batching and retry logic
   - Monitor production logs

3. **Performance Monitoring**
   - Add performance metrics to logger
   - Monitor bundle size impact
   - Optimize batching parameters

### Migration Script Template

```typescript
// BEFORE (❌ Bad)
this.api.getStats().subscribe({
  next: (data) => this.stats.set(data),
  error: (err) => console.error('Failed to load stats:', err)
});

// AFTER (✅ Good)
private readonly logger = inject(LoggerService).forContext('OverviewComponent');

this.api.getStats().subscribe({
  next: (data) => {
    this.logger.debug('Stats loaded', { count: data.total });
    this.stats.set(data);
  },
  error: (err) => {
    this.logger.error('Failed to load stats', err, {
      endpoint: '/api/stats',
      timestamp: new Date().toISOString()
    });
    // TODO: Show user-facing error (toast)
  }
});
```

---

## 🧪 Testing Strategy

### Unit Tests - Dashboard

#### File: `apps/dashboard/src/app/core/services/logger.service.spec.ts`

```typescript
import { TestBed } from "@angular/core/testing";
import {
  HttpClientTestingModule,
  HttpTestingController,
} from "@angular/common/http/testing";
import {
  LoggerService,
  LogLevel,
  LOGGER_CONFIG,
  LoggerConfig,
} from "./logger.service";

describe("LoggerService", () => {
  let service: LoggerService;
  let httpMock: HttpTestingController;
  let consoleSpy: jasmine.Spy;

  beforeEach(() => {
    consoleSpy = spyOn(console, "error");

    const testConfig: LoggerConfig = {
      minLevel: LogLevel.DEBUG,
      enableConsole: true,
      enableRemote: true,
      remoteEndpoint: "/api/logs",
      batchSize: 5,
      batchIntervalMs: 1000,
      includeStackTrace: true,
    };

    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [
        LoggerService,
        { provide: LOGGER_CONFIG, useValue: testConfig },
      ],
    });

    service = TestBed.inject(LoggerService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  it("should be created", () => {
    expect(service).toBeTruthy();
  });

  it("should filter logs below minimum level", () => {
    const testConfig: LoggerConfig = {
      minLevel: LogLevel.ERROR,
      enableConsole: true,
      enableRemote: false,
    };

    TestBed.resetTestingModule();
    TestBed.configureTestingModule({
      providers: [
        LoggerService,
        { provide: LOGGER_CONFIG, useValue: testConfig },
      ],
    });

    service = TestBed.inject(LoggerService);

    service.debug("Debug message");
    service.info("Info message");
    service.warn("Warning message");

    expect(consoleSpy).not.toHaveBeenCalled();

    service.error("Error message");
    expect(consoleSpy).toHaveBeenCalledTimes(1);
  });

  it("should create context logger with automatic context injection", () => {
    const contextLogger = service.forContext("TestComponent");

    contextLogger.info("Test message");

    // Verify console was called with context
    expect(consoleSpy).toHaveBeenCalled();
    const call = consoleSpy.calls.mostRecent();
    expect(call.args[0]).toContain("[TestComponent]");
  });

  it("should batch and send logs to backend", (done) => {
    service.error("Error 1");
    service.error("Error 2");

    setTimeout(() => {
      const req = httpMock.expectOne("/api/logs");
      expect(req.request.method).toBe("POST");
      expect(req.request.body.logs.length).toBe(2);
      req.flush({});
      done();
    }, 1500);
  });
});
```

### Integration Tests - VS Code

#### File: `apps/vscode/src/test/logger.test.ts`

```typescript
import * as assert from "assert";
import * as vscode from "vscode";
import { logger, LogLevel } from "../logger";

suite("Extension Logger Tests", () => {
  test("Logger initializes successfully", () => {
    const context = {
      subscriptions: [],
      extensionMode: vscode.ExtensionMode.Development,
    } as vscode.ExtensionContext;

    logger.initialize(context);

    // Should not throw
    assert.ok(true);
  });

  test("Context logger provides automatic context", () => {
    const contextLogger = logger.forContext("TestModule");

    // Should not throw
    contextLogger.info("Test message");
    contextLogger.debug("Debug message");

    assert.ok(true);
  });

  test("Error logging shows notification", async () => {
    // Mock vscode.window.showErrorMessage
    const originalShowError = vscode.window.showErrorMessage;
    let notificationShown = false;

    (vscode.window as any).showErrorMessage = () => {
      notificationShown = true;
      return Promise.resolve();
    };

    logger.error("Test error", new Error("Test exception"));

    // Restore original
    (vscode.window as any).showErrorMessage = originalShowError;

    assert.strictEqual(notificationShown, true);
  });
});
```

### E2E Testing Checklist

- [ ] Dashboard logger sends batched logs to backend
- [ ] VS Code logger shows in Output Channel
- [ ] Error notifications appear in VS Code
- [ ] Log levels filter correctly
- [ ] Production mode disables debug logs
- [ ] Stack traces appear in development only
- [ ] Remote logging fails gracefully without breaking app

---

## ⚡ Performance Considerations

### Bundle Size Impact

**Dashboard:**

- Logger service: ~5KB minified
- No external dependencies
- Tree-shakeable (unused log levels removed in prod)

**VS Code:**

- Logger module: ~3KB
- Uses only VS Code API (no deps)

### Runtime Performance

1. **Log Level Filtering (O(1))**

   ```typescript
   if (level < this.config.minLevel) return; // Early exit
   ```

2. **Lazy Serialization**

   ```typescript
   // Only serialize if actually logging
   const payload = logs.map((log) => serializeLog(log));
   ```

3. **Batching Strategy**

   - Dashboard: 10-20 logs per batch, 5-10s interval
   - Reduces HTTP requests by ~90%
   - Minimal memory overhead

4. **Production Optimizations**
   - DEBUG logs completely removed in production builds
   - Console output disabled when remote logging enabled
   - Stack trace collection disabled

### Memory Management

- Log batching buffer: Max 100 entries (auto-flush)
- No memory leaks from subscriptions (managed by service lifecycle)
- Error objects properly serialized (avoid circular references)

---

## 🎯 Best Practices & Guidelines

### DO ✅

```typescript
// Clear, actionable messages
logger.error("Failed to load user sessions", error, {
  endpoint: "/api/sessions",
  userId: user.id,
  retryAttempt: attempts,
});

// Structured metadata for filtering/searching
logger.info("WebSocket message received", {
  messageType: msg.type,
  payloadSize: msg.data.length,
  connectionId: this.connectionId,
});

// Use appropriate levels
logger.debug("Cache hit", { key, ttl }); // Development only
logger.info("User logged in", { userId }); // Production events
logger.warn("Rate limit approaching", { current: 95, limit: 100 });
logger.error("Payment processing failed", error, { orderId });
logger.fatal("Database connection lost", error); // Critical issues
```

### DON'T ❌

```typescript
// Vague messages
logger.error("Error"); // ❌ No context
logger.info("Done"); // ❌ What's done?

// Sensitive data in logs
logger.debug("User authenticated", { password: pwd }); // ❌ Security risk
logger.info("Payment processed", { creditCard: cc }); // ❌ PCI violation

// Excessive logging
for (const item of items) {
  logger.debug("Processing item", { item }); // ❌ Too noisy
}

// Logging in tight loops
items.map((item) => {
  logger.info("Mapped", { item }); // ❌ Performance impact
  return transform(item);
});
```

### Context Naming Conventions

```typescript
// Components: Use class name
private logger = inject(LoggerService).forContext('OverviewComponent');

// Services: Use class name
private logger = inject(LoggerService).forContext('ApiService');

// Providers (VS Code): Use descriptive name
private logger = logger.forContext('FileIntelligenceProvider');

// Utilities: Use module name
const logger = logger.forContext('ConfigParser');
```

---

## 🚀 Rollout Strategy

### Development Environment

1. Deploy logger services
2. Convert 2-3 components as proof-of-concept
3. Team review and feedback
4. Adjust configuration if needed

### Staging Environment

1. Full migration of all console statements
2. Enable remote logging to staging backend
3. Monitor log volume and performance
4. Test error notification UX

### Production Environment

1. Deploy with feature flag (initially disabled)
2. Enable for 10% of users (canary)
3. Monitor metrics:
   - Log volume (logs/minute)
   - Backend response time
   - Client performance impact
   - Error rate
4. Gradual rollout to 50% → 100%

### Rollback Plan

- Feature flag to disable remote logging
- Fallback to console.error for critical errors
- Backend endpoint can return 503 to disable logging

---

## 📊 Success Metrics

### Technical Metrics

- ✅ Zero console.log/error in production builds
- ✅ <5KB bundle size impact
- ✅ <10ms average logging overhead
- ✅ 99.9% log delivery success rate
- ✅ <100MB/day log volume per user

### Operational Metrics

- ✅ 50% reduction in "I can't reproduce" bug reports
- ✅ 30% faster bug diagnosis time
- ✅ Proactive error detection before user reports
- ✅ Improved production monitoring coverage

### Developer Experience

- ✅ Consistent logging API across projects
- ✅ Easy to add context to logs
- ✅ Self-documenting error messages
- ✅ Minimal boilerplate code

---

## 🔮 Future Enhancements

### Short Term (3-6 months)

1. **Log Aggregation Dashboard**

   - Real-time error monitoring
   - Log search and filtering
   - User session replay

2. **Integration with Error Tracking**

   - Sentry/Rollbar integration
   - Automatic issue creation
   - Error grouping and deduplication

3. **Performance Monitoring**
   - Timing decorators
   - Performance marks
   - Slow operation detection

### Long Term (6-12 months)

1. **Distributed Tracing**

   - Request correlation IDs
   - Cross-service tracing
   - Dependency graphs

2. **Machine Learning**

   - Anomaly detection
   - Predictive error alerts
   - Pattern recognition

3. **User Analytics Integration**
   - Combine logs with user behavior
   - Error impact on user journey
   - Conversion funnel debugging

---

## 📚 References & Resources

### Documentation

- [Angular Dependency Injection](https://angular.dev/guide/di)
- [VS Code Extension API - OutputChannel](https://code.visualstudio.com/api/references/vscode-api#OutputChannel)
- [Structured Logging Best Practices](https://www.loggly.com/ultimate-guide/node-logging-basics/)

### Related Files

- Dashboard: [api.service.ts](../apps/dashboard/src/app/core/services/api.service.ts)
- Dashboard: [websocket.service.ts](../apps/dashboard/src/app/core/services/websocket.service.ts)
- VS Code: [extension.ts](../apps/vscode/src/extension.ts)

### Team Contacts

- Architecture Review: TBD
- Security Review: TBD
- Production Deployment: TBD

---

## ✅ Implementation Checklist

### Pre-Implementation

- [ ] Review this design doc with team
- [ ] Get security approval for remote logging
- [ ] Set up `/api/logs` backend endpoint
- [ ] Create feature flags in config

### Implementation

- [ ] Create Dashboard logger service
- [ ] Create VS Code logger module
- [ ] Write unit tests (80%+ coverage)
- [ ] Update app.config.ts / extension.ts
- [ ] Create migration script/guidelines

### Migration

- [ ] Convert WebSocketService (Dashboard)
- [ ] Convert extension.ts activation (VS Code)
- [ ] Convert remaining Dashboard components
- [ ] Convert remaining VS Code files
- [ ] Remove all console.log/error statements

### Testing

- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing in dev environment
- [ ] Load testing (log volume)
- [ ] Security audit (no sensitive data logged)

### Documentation

- [ ] Update team wiki
- [ ] Create migration guide
- [ ] Record demo video
- [ ] Update onboarding docs

### Deployment

- [ ] Deploy to staging
- [ ] Monitor for 48 hours
- [ ] Deploy to production (canary)
- [ ] Full production rollout
- [ ] Post-deployment review

---

**END OF DESIGN DOCUMENT**

_Ready for team review and implementation approval._
