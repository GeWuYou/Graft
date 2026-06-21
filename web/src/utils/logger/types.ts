export type LogLevel = 'debug' | 'info' | 'warn' | 'error' | 'silent';

export interface LogEvent {
  level: LogLevel;
  moduleName: string;
  message: string;
  timestamp: Date;
  meta?: unknown;
  error?: Error;
}

export interface LoggerTransport {
  log(event: LogEvent): void;
}

export type LoggerContext = Record<string, unknown>;

export interface Logger {
  debug(message: string, meta?: unknown): void;
  info(message: string, meta?: unknown): void;
  warn(message: string, meta?: unknown): void;
  error(messageOrError: string | Error, meta?: unknown): void;
  child(name: string): Logger;
  withContext(context: LoggerContext): Logger;
}
