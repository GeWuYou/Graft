import { createConsola } from 'consola';

import type { LogEvent, LoggerTransport } from '@/utils/logger/types';

const logger = createConsola();
const MAX_FIELD_COUNT = 12;
const MAX_VALUE_LENGTH = 240;

function toSingleLine(value: string) {
  return value.replaceAll(/[\r\n\t]+/g, ' ').trim();
}

function truncate(value: string) {
  return value.length > MAX_VALUE_LENGTH ? `${value.slice(0, MAX_VALUE_LENGTH - 3)}...` : value;
}

function formatValue(value: unknown): string {
  if (value === null) {
    return 'null';
  }

  if (value === undefined) {
    return 'undefined';
  }

  if (value instanceof Date) {
    return value.toISOString();
  }

  if (typeof value === 'string') {
    return truncate(JSON.stringify(toSingleLine(value)));
  }

  if (typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') {
    return String(value);
  }

  try {
    return truncate(toSingleLine(JSON.stringify(value)));
  } catch {
    return '"[unserializable]"';
  }
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  if (value === null || typeof value !== 'object') {
    return false;
  }

  const prototype = Object.getPrototypeOf(value);
  return prototype === Object.prototype || prototype === null;
}

function appendFields(value: unknown, prefix: string, fields: string[], seen: WeakSet<object>) {
  if (fields.length >= MAX_FIELD_COUNT) {
    return;
  }

  if (value instanceof Error) {
    const errorPrefix = prefix || 'error';
    fields.push(`${errorPrefix}.name=${formatValue(value.name)}`);
    if (fields.length < MAX_FIELD_COUNT) {
      fields.push(`${errorPrefix}.message=${formatValue(value.message)}`);
    }
    return;
  }

  if (isPlainObject(value)) {
    if (seen.has(value)) {
      fields.push(`${prefix}="[circular]"`);
      return;
    }

    seen.add(value);
    for (const [key, child] of Object.entries(value)) {
      appendFields(child, prefix ? `${prefix}.${key}` : key, fields, seen);
      if (fields.length >= MAX_FIELD_COUNT) {
        break;
      }
    }
    seen.delete(value);
    return;
  }

  fields.push(`${prefix}=${formatValue(value)}`);
}

function formatConsolaLine(event: LogEvent) {
  const fields: string[] = [];
  const seen = new WeakSet<object>();

  if (event.meta !== undefined) {
    appendFields(event.meta, '', fields, seen);
  }
  if (event.error !== undefined) {
    appendFields(event.error, 'error', fields, seen);
  }

  const eventLine = `${event.timestamp.toISOString()} | ${toSingleLine(event.message)}`;
  return fields.length > 0 ? `${eventLine} | ${fields.join(' ')}` : eventLine;
}

export function createConsolaTransport(): LoggerTransport {
  return {
    log(event) {
      const taggedLogger = logger.withTag(event.moduleName);
      const eventLine = formatConsolaLine(event);

      switch (event.level) {
        case 'debug':
          taggedLogger.debug(eventLine);
          return;
        case 'info':
          taggedLogger.info(eventLine);
          return;
        case 'warn':
          taggedLogger.warn(eventLine);
          return;
        case 'error':
          taggedLogger.error(eventLine);
          return;
        case 'silent':
          return;
      }
    },
  };
}
