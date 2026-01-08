import * as vscode from "vscode";

/**
 * Safe wrapper around OutputChannel that prevents "Channel has been closed" errors.
 * Gracefully handles write operations after channel disposal.
 */
export class SafeOutputChannel {
  private isDisposed = false;

  constructor(private readonly channel: vscode.OutputChannel) {}

  appendLine(value: string): void {
    if (this.isDisposed) {
      // Silently ignore writes to disposed channel
      return;
    }
    try {
      this.channel.appendLine(value);
    } catch (error) {
      // Channel was disposed between check and write
      this.isDisposed = true;
    }
  }

  append(value: string): void {
    if (this.isDisposed) {
      return;
    }
    try {
      this.channel.append(value);
    } catch (error) {
      this.isDisposed = true;
    }
  }

  show(preserveFocus?: boolean): void {
    if (this.isDisposed) {
      return;
    }
    try {
      this.channel.show(preserveFocus);
    } catch (error) {
      this.isDisposed = true;
    }
  }

  dispose(): void {
    if (!this.isDisposed) {
      this.isDisposed = true;
      this.channel.dispose();
    }
  }

  get name(): string {
    return this.channel.name;
  }
}

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

class ExtensionLogger {
  private outputChannel: vscode.OutputChannel | null = null;
  private config: LoggerConfig = {
    minLevel: LogLevel.DEBUG,
    showTimestamp: true,
    showContext: true,
    showNotifications: true,
    outputChannelName: "Mind Palace",
  };

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
    if (metadata?.error) {
      entry.error = metadata.error;
    }
    this.writeToOutputChannel(entry);
  }

  private writeToOutputChannel(entry: LogEntry): void {
    if (!this.outputChannel) {
      return;
    }
    const parts: string[] = [];
    if (this.config.showTimestamp) {
      parts.push(`[${this.formatTimestamp(entry.timestamp)}]`);
    }
    parts.push(`[${LogLevel[entry.level].padEnd(5)}]`);
    if (this.config.showContext && entry.context) {
      parts.push(`[${entry.context}]`);
    }
    parts.push(entry.message);
    const line = parts.join(" ");
    this.outputChannel.appendLine(line);
    if (entry.metadata && Object.keys(entry.metadata).length > 0) {
      const metadataStr = JSON.stringify(entry.metadata, null, 2);
      this.outputChannel.appendLine(`  Metadata: ${metadataStr}`);
    }
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

export const logger = new ExtensionLogger();
