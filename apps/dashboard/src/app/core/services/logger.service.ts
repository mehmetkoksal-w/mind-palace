import { Injectable, inject, isDevMode, InjectionToken } from "@angular/core";
import { HttpClient } from "@angular/common/http";
import { Observable, Subject, catchError, of } from "rxjs";
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
  minLevel: isDevMode() ? LogLevel.DEBUG : LogLevel.WARN,
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
    if (metadata?.["error"]) {
      entry.error = metadata["error"];
      if (this.config.includeStackTrace && metadata["error"] instanceof Error) {
        entry.stackTrace = metadata["error"].stack;
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
      catchError(() => of(undefined)) // Silently fail to avoid loops
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
