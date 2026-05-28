import { createConsolaTransport } from '@/utils/logger/transports/consola';
import { noopTransport } from '@/utils/logger/transports/noop';
import type { LogEvent, Logger, LoggerContext, LoggerTransport, LogLevel } from '@/utils/logger/types';

const LOG_LEVEL_ORDER: Record<Exclude<LogLevel, 'silent'>, number> = {
  debug: 10,
  info: 20,
  warn: 30,
  error: 40,
};

function isPlainObject(value: unknown): value is Record<string, unknown> {
  if (value === null || typeof value !== 'object') {
    return false;
  }

  const prototype = Object.getPrototypeOf(value);
  return prototype === Object.prototype || prototype === null;
}

function normalizeModuleSegment(moduleName: string): string {
  const normalized = moduleName.trim();
  if (!normalized) {
    throw new Error('moduleName must not be empty');
  }
  return normalized;
}

function joinModuleName(moduleName: string, childName: string): string {
  return `${moduleName}:${normalizeModuleSegment(childName)}`;
}

function mergeContext(baseContext: LoggerContext, nextContext: LoggerContext): LoggerContext {
  return {
    ...baseContext,
    ...nextContext,
  };
}

let globalContext: LoggerContext = {};

// context 默认作为结构化字段参与输出；当单次 meta 也是对象时，让单次 meta 覆盖同名 context。
function mergeMeta(context: LoggerContext, meta: unknown): unknown {
  const hasContext = Object.keys(context).length > 0;
  if (!hasContext) {
    return meta;
  }

  if (meta === undefined) {
    return { ...context };
  }

  if (isPlainObject(meta)) {
    return {
      ...context,
      ...meta,
    };
  }

  return {
    ...context,
    value: meta,
  };
}

function resolveDefaultLevel(): LogLevel {
  return import.meta.env.PROD ? 'warn' : 'debug';
}

function isLogLevel(value: string): value is LogLevel {
  return value === 'debug' || value === 'info' || value === 'warn' || value === 'error' || value === 'silent';
}

function resolveLogLevel(rawLevel: string | undefined): LogLevel {
  if (!rawLevel) {
    return resolveDefaultLevel();
  }

  const normalized = rawLevel.trim().toLowerCase();
  if (isLogLevel(normalized)) {
    return normalized;
  }

  return resolveDefaultLevel();
}

function shouldLog(eventLevel: Exclude<LogLevel, 'silent'>, currentLevel: LogLevel): boolean {
  if (currentLevel === 'silent') {
    return false;
  }

  return LOG_LEVEL_ORDER[eventLevel] >= LOG_LEVEL_ORDER[currentLevel];
}

function createTransport(currentLevel: LogLevel): LoggerTransport {
  if (currentLevel === 'silent') {
    return noopTransport;
  }

  return createConsolaTransport();
}

function resolveGlobalContext(): LoggerContext {
  return { ...globalContext };
}

class LoggerCore implements Logger {
  constructor(
    private readonly moduleName: string,
    private readonly currentLevel: LogLevel,
    private readonly transport: LoggerTransport,
    private readonly context: LoggerContext = {},
  ) {}

  debug(message: string, meta?: unknown): void {
    this.emit('debug', message, meta);
  }

  info(message: string, meta?: unknown): void {
    this.emit('info', message, meta);
  }

  warn(message: string, meta?: unknown): void {
    this.emit('warn', message, meta);
  }

  error(messageOrError: string | Error, meta?: unknown): void {
    if (messageOrError instanceof Error) {
      this.emit('error', messageOrError.message || 'Unexpected error', meta, messageOrError);
      return;
    }

    this.emit('error', messageOrError, meta);
  }

  child(name: string): Logger {
    return new LoggerCore(joinModuleName(this.moduleName, name), this.currentLevel, this.transport, this.context);
  }

  withContext(context: LoggerContext): Logger {
    return new LoggerCore(this.moduleName, this.currentLevel, this.transport, mergeContext(this.context, context));
  }

  private emit(level: Exclude<LogLevel, 'silent'>, message: string, meta?: unknown, error?: Error): void {
    if (!shouldLog(level, this.currentLevel)) {
      return;
    }

    const mergedContext = mergeContext(resolveGlobalContext(), this.context);
    const mergedMeta = mergeMeta(mergedContext, meta);
    const event: LogEvent = {
      level,
      moduleName: this.moduleName,
      message,
      timestamp: new Date(),
      ...(mergedMeta === undefined ? {} : { meta: mergedMeta }),
      ...(error === undefined ? {} : { error }),
    };

    this.transport.log(event);
  }
}

const defaultLogLevel = resolveLogLevel(import.meta.env.VITE_LOG_LEVEL);
const defaultTransport = createTransport(defaultLogLevel);

export function createLogger(moduleName: string): Logger {
  return new LoggerCore(normalizeModuleSegment(moduleName), defaultLogLevel, defaultTransport);
}

export function patchGlobalLoggerContext(context: LoggerContext): void {
  const nextContext: LoggerContext = { ...globalContext };

  Object.entries(context).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') {
      delete nextContext[key];
      return;
    }
    nextContext[key] = value;
  });

  globalContext = nextContext;
}
