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
  showNotifications: boolean;
  outputChannelName: string;
}

// ============================================================================
// Extension Logger (Singleton)
// ============================================================================

class ExtensionLogger {
  private outputChannel: vscode.OutputChannel | null = null;
  private config: LoggerConfig = {
    minLevel: LogLevel.DEBUG,
    showTimestamp: true,
    showContext: true,
    showNotifications: true,
    outputChannelName: "Mind Palace",
  };

  // --------------------------------------------------------------------------
  // Initialization
  // --------------------------------------------------------------------------

  initialize(context: vscode.ExtensionContext): void {
    this.outputChannel = vscode.window.createOutputChannel(
      this.config.outputChannelName,
      "log"
    );

    context.subscriptions.push(this.outputChannel);

    const isDevelopment =
      context.extensionMode === vscode.ExtensionMode.Development;
    this.config.minLevel = isDevelopment ? LogLevel.DEBUG : LogLevel.INFO;

    this.info("Logger initialized", "ExtensionLogger", {
      mode: isDevelopment ? "development" : "production",
      minLevel: LogLevel[this.config.minLevel],
    });
  }

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

    // Show notification for errors if enabled
    if (this.config.showNotifications) {
      vscode.window.showErrorMessage(`Mind Palace: ${message}`);
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
    vscode.window
      .showErrorMessage(`Mind Palace (FATAL): ${message}`, "Show Logs")
      .then((selection) => {
        if (selection === "Show Logs") {
          this.show();
        }
      });
  }

  forContext(context: string): ContextLogger {
    return new ContextLogger(this, context);
  }

  show(): void {
    this.outputChannel?.show();
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
    if (level < this.config.minLevel) {
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

    this.writeToOutputChannel(entry);
  }

  // --------------------------------------------------------------------------
  // Output Channel Transport
  // --------------------------------------------------------------------------

  private writeToOutputChannel(entry: LogEntry): void {
    if (!this.outputChannel) {
      return;
    }

    const parts: string[] = [];

    // Timestamp
    if (this.config.showTimestamp) {
      parts.push(`[${this.formatTimestamp(entry.timestamp)}]`);
    }

    // Level
    parts.push(`[${LogLevel[entry.level].padEnd(5)}]`);

    // Context
    if (this.config.showContext && entry.context) {
      parts.push(`[${entry.context}]`);
    }

    // Message
    parts.push(entry.message);

    const line = parts.join(" ");
    this.outputChannel.appendLine(line);

    // Metadata
    if (entry.metadata && Object.keys(entry.metadata).length > 0) {
      const metadataStr = JSON.stringify(entry.metadata, null, 2);
      this.outputChannel.appendLine(`  Metadata: ${metadataStr}`);
    }

    // Error details
    if (entry.error) {
      if (entry.error instanceof Error) {
        this.outputChannel.appendLine(`  Error: ${entry.error.message}`);
        if (entry.error.stack) {
          this.outputChannel.appendLine(`  Stack:\n${entry.error.stack}`);
        }
      } else {
        this.outputChannel.appendLine(
          `  Error: ${JSON.stringify(entry.error)}`
        );
      }
    }
  }

  private formatTimestamp(date: Date): string {
    const pad = (n: number) => n.toString().padStart(2, "0");
    return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(
      date.getSeconds()
    )}.${date.getMilliseconds().toString().padStart(3, "0")}`;
  }
}

// ============================================================================
// Context Logger
// ============================================================================

export class ContextLogger {
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

  show(): void {
    this.logger.show();
  }
}

// ============================================================================
// Singleton Export
// ============================================================================

export const logger = new ExtensionLogger();
