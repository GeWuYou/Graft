// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { LogLevel, LogToken } from './log-highlight';
import { getLogLevelTone, normalizeLogLevel, tokenizeLogLine } from './log-highlight';

export type ParsedLogMetadata = Record<string, unknown>;
export type ContainerLogFormat = 'json' | 'logfmt' | 'structured' | 'plain' | 'stack' | 'unknown';
export type ParsedContainerLogImportantField = {
  key: string;
  value: string;
  priority: number;
};
export type ParsedContainerLog = {
  raw: string;
  level?: LogLevel;
  time?: string;
  source?: string;
  message: string;
  format: ContainerLogFormat;
  fields: ParsedLogMetadata;
  importantFields: ParsedContainerLogImportantField[];
  display: {
    title: string;
    subtitleParts: string[];
    level?: LogLevel;
  };
};
export type ParsedLogLine = {
  lineNo: number;
  timestamp: string;
  level: LogLevel | null;
  source: string;
  sourceShort: string;
  message: string;
  metadata: ParsedLogMetadata | null;
  raw: string;
  tone: ReturnType<typeof getLogLevelTone>;
  parsed: ParsedContainerLog;
};
export type DisplayLogLine = ParsedLogLine & {
  messageTokens: LogToken[];
  rawTokens: LogToken[];
  searchMatchCount: number;
};

const STRUCTURED_HEAD_PATTERN =
  /^(\d{4}-\d{2}-\d{2}(?:[T\s]\d{2}:\d{2}:\d{2}(?:[.,]\d+)?(?:Z|[+-]\d{2}:?\d{2})?)?)\s+(\S+)(?:\s+(\S+:\d+))?(?:\s+(.*))?$/;
const STRUCTURED_STDLOG_HEAD_PATTERN =
  /^(\d{4}-\d{2}-\d{2}(?:[T\s]\d{2}:\d{2}:\d{2}(?:[.,]\d+)?(?:Z|[+-]\d{2}:?\d{2})?)?)\s+(\S+)\s+(\S+)\s+(\S+:\d+)(?:\s+(.*))?$/;
const SOURCE_PATTERN = /^(?:[\w.-]+\/)*[\w.-]+\.(?:go|ts|tsx|vue|js|jsx|mjs|cjs|py|rs|java|kt|php|rb):\d+$/;
const STACK_SYMBOL_PATTERN = /^(?:[\w.-]+\/)*[\w.-]+(?:\.[\w$-]+)*(?:\.\(\*?[\w$-]+\))?\.[\w$-]+/;
const STACK_FILE_PATTERN = /^\s+(?:\/|[A-Za-z]:\\|\.{1,2}\/).+:\d+(?::\d+)?/;
const LOGFMT_PAIR_PATTERN = /(?:^|\s)([A-Za-z_][\w.-]*)=("[^"\\]*(?:\\.[^"\\]*)*"|'[^'\\]*(?:\\.[^'\\]*)*'|[^\s]+)/g;
const FIELD_PRIORITY = [
  'request_id',
  'client_request_id',
  'trace_id',
  'span_id',
  'path',
  'method',
  'status_code',
  'status',
  'duration',
  'latency_ms',
  'user_id',
  'api_key_id',
  'group_id',
  'group_name',
  'model',
  'provider',
  'endpoint',
  'protocol',
  'error',
  'time',
  'timestamp',
  'ts',
  'datetime',
  'level',
  'severity',
  'msg',
  'message',
  'event',
] as const;
const MAX_IMPORTANT_FIELDS = 10;
const LOW_SIGNAL_METADATA_PATTERNS = [/^legacy_/i, /^service$/i, /^env$/i];

export function parseContainerLogLine(rawLine: string): ParsedContainerLog {
  const raw = rawLine ?? '';
  const trimmed = raw.trim();
  if (!trimmed) {
    return buildParsedLog({ raw, message: raw, format: 'plain', fields: {} });
  }

  const parsedJson = parseJsonLine(raw, trimmed);
  if (parsedJson) return parsedJson;

  const parsedLogfmt = parseLogfmtLine(raw);
  if (parsedLogfmt) return parsedLogfmt;

  const parsedStructured = parseStructuredTextLine(raw);
  if (parsedStructured) return parsedStructured;

  if (isStackTraceLike(trimmed, raw)) {
    return buildParsedLog({ raw, level: 'LOG', message: trimmed, format: 'stack', fields: {} });
  }

  return buildParsedLog({ raw, level: 'LOG', message: trimmed, format: 'plain', fields: {} });
}

export function parseLogLine(raw: string, lineNo: number): ParsedLogLine {
  const parsed = parseContainerLogLine(raw);
  const level = parsed.level ?? null;

  return {
    lineNo,
    timestamp: parsed.time ?? '',
    level,
    source: parsed.source ?? '',
    sourceShort: shortenLogSource(parsed.source ?? ''),
    message: parsed.message || parsed.raw,
    metadata: hasFields(parsed.fields) ? parsed.fields : null,
    raw: parsed.raw,
    tone: getLogLevelTone(level),
    parsed,
  };
}

export function parseLogLines(lines: string[]): ParsedLogLine[] {
  return lines.map((line, index) => parseLogLine(line, index + 1));
}

export function buildDisplayLogLine(line: ParsedLogLine, keyword = ''): DisplayLogLine {
  return {
    ...line,
    messageTokens: tokenizeLogLine(line.message, keyword),
    rawTokens: tokenizeLogLine(line.raw, keyword),
    searchMatchCount: countKeywordMatches(line.raw, keyword),
  };
}

export function summarizeMetadata(metadata: ParsedLogMetadata | null, maxVisible = 3) {
  if (!metadata) {
    return { hiddenCount: 0, tags: [] as Array<[string, unknown]> };
  }

  const importantFields = buildImportantFields(metadata, maxVisible, { hideLowSignal: true });
  const visibleKeys = new Set(importantFields.map((field) => field.key));

  return {
    hiddenCount: Math.max(0, Object.keys(metadata).length - visibleKeys.size),
    tags: importantFields.map((field) => [field.key, metadata[field.key]] as [string, unknown]),
  };
}

export function formatLogMetadataValue(value: unknown) {
  if (value === null) return 'null';
  if (value === undefined) return '';
  if (typeof value === 'string') return value;
  if (typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') return String(value);
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

function parseJsonLine(raw: string, trimmed: string): ParsedContainerLog | null {
  if (!trimmed.startsWith('{') || !trimmed.endsWith('}')) {
    return null;
  }

  try {
    const parsed = JSON.parse(trimmed);
    if (!isPlainRecord(parsed)) {
      return null;
    }

    return buildParsedLog({
      raw,
      level: normalizeLogLevel(readFirstString(parsed, ['level', 'severity'])) ?? undefined,
      time: readFirstString(parsed, ['time', 'timestamp', 'ts', 'datetime']),
      source: readFirstString(parsed, ['caller', 'source', 'file', 'logger']),
      message: readFirstString(parsed, ['msg', 'message', 'event']) || trimmed,
      format: 'json',
      fields: parsed,
    });
  } catch {
    return null;
  }
}

function parseLogfmtLine(raw: string): ParsedContainerLog | null {
  const parsedPairs = parseLogfmt(raw);
  if (!parsedPairs) {
    return null;
  }

  const fields = parsedPairs.fields;
  return buildParsedLog({
    raw,
    level: normalizeLogLevel(readFirstString(fields, ['level', 'severity'])) ?? undefined,
    time: readFirstString(fields, ['time', 'timestamp', 'ts', 'datetime']),
    source: readFirstString(fields, ['caller', 'source', 'file', 'logger']),
    message: readFirstString(fields, ['msg', 'message', 'event']) || raw.trim(),
    format: 'logfmt',
    fields,
    includeAllImportantFields: Object.keys(fields).length <= MAX_IMPORTANT_FIELDS,
    preserveFieldOrder: true,
  });
}

function parseStructuredTextLine(raw: string): ParsedContainerLog | null {
  const metadataResult = extractTrailingMetadata(raw);
  const body = metadataResult.body.trim();
  const stdlogHeadMatch = STRUCTURED_STDLOG_HEAD_PATTERN.exec(body);
  if (stdlogHeadMatch) {
    const [, time, rawLevel, stream, sourceCandidate, rest = ''] = stdlogHeadMatch;
    const level = normalizeLogLevel(rawLevel);
    if (level && SOURCE_PATTERN.test(sourceCandidate)) {
      const fields = metadataResult.metadata ?? {};
      return buildParsedLog({
        raw,
        level,
        time,
        source: `${stream} ${sourceCandidate}`,
        message: rest.trim() || body,
        format: 'structured',
        fields,
      });
    }
  }

  const headMatch = STRUCTURED_HEAD_PATTERN.exec(body);
  if (!headMatch) {
    return null;
  }

  const [, time, rawLevel, sourceCandidate = '', rest = ''] = headMatch;
  const level = normalizeLogLevel(rawLevel);
  if (!time || !level) {
    return null;
  }

  const source = SOURCE_PATTERN.test(sourceCandidate) ? sourceCandidate : '';
  const message = source ? rest.trim() : [sourceCandidate, rest].filter(Boolean).join(' ').trim();
  const fields = metadataResult.metadata ?? {};
  return buildParsedLog({
    raw,
    level,
    time,
    source,
    message: message || body,
    format: 'structured',
    fields,
  });
}

function buildParsedLog({
  raw,
  level,
  time,
  source,
  message,
  format,
  fields,
  includeAllImportantFields = false,
  preserveFieldOrder = false,
}: {
  raw: string;
  level?: LogLevel;
  time?: string;
  source?: string;
  message: string;
  format: ContainerLogFormat;
  fields: ParsedLogMetadata;
  includeAllImportantFields?: boolean;
  preserveFieldOrder?: boolean;
}): ParsedContainerLog {
  const normalizedMessage = message.trim() || raw.trim() || raw;
  const normalizedLevel = level ?? undefined;
  const normalizedTime = normalizeOptionalString(time);
  const normalizedSource = normalizeOptionalString(source);
  const subtitleParts = [normalizedTime, normalizedSource].filter(Boolean) as string[];

  return {
    raw,
    level: normalizedLevel,
    time: normalizedTime,
    source: normalizedSource,
    message: normalizedMessage,
    format,
    fields,
    importantFields: buildImportantFields(fields, includeAllImportantFields ? MAX_IMPORTANT_FIELDS : undefined, {
      includeAll: includeAllImportantFields,
      preserveFieldOrder,
    }),
    display: {
      title: normalizedMessage || raw,
      subtitleParts,
      level: normalizedLevel,
    },
  };
}

function buildImportantFields(
  fields: ParsedLogMetadata,
  maxVisible = MAX_IMPORTANT_FIELDS,
  options: { hideLowSignal?: boolean; includeAll?: boolean; preserveFieldOrder?: boolean } = {},
): ParsedContainerLogImportantField[] {
  const entries = Object.entries(fields).filter(([, value]) => !isEmptyFieldValue(value));
  const filteredEntries = options.hideLowSignal
    ? entries.filter(([key, value]) => !isLowSignalMetadata(key, value))
    : entries;
  const priority = new Map<string, number>(FIELD_PRIORITY.map((key, index) => [key, index + 1]));
  const prioritized = filteredEntries
    .map(([key, value], index) => ({
      key,
      value: formatLogMetadataValue(value),
      priority: priority.get(key) ?? FIELD_PRIORITY.length + index + 1,
    }))
    .filter((field) => options.includeAll || priority.has(field.key))
    .sort((left, right) => (options.preserveFieldOrder ? 0 : left.priority - right.priority));

  return prioritized.slice(0, maxVisible);
}

function extractTrailingMetadata(raw: string): { body: string; metadata: ParsedLogMetadata | null } {
  const jsonStart = findTrailingJsonStart(raw);
  if (jsonStart >= 0) {
    const jsonText = raw.slice(jsonStart).trim();
    try {
      const parsed = JSON.parse(jsonText);
      if (isPlainRecord(parsed)) {
        return {
          body: raw.slice(0, jsonStart).trimEnd(),
          metadata: parsed,
        };
      }
    } catch {
      return { body: raw, metadata: null };
    }
  }

  return { body: raw, metadata: null };
}

function parseLogfmt(raw: string): { fields: ParsedLogMetadata } | null {
  const matches = [...raw.matchAll(LOGFMT_PAIR_PATTERN)];
  if (!matches.length) {
    return null;
  }

  const nonWhitespaceRanges = raw.trim()
    ? [...raw.matchAll(/\S+/g)].map((match) => ({ start: match.index ?? 0, end: (match.index ?? 0) + match[0].length }))
    : [];
  const pairRanges = matches.map((match) => ({
    start: match.index ?? 0,
    end: (match.index ?? 0) + match[0].length,
  }));
  const allTextCovered = nonWhitespaceRanges.every((range) =>
    pairRanges.some((pair) => range.start >= pair.start && range.end <= pair.end),
  );
  if (!allTextCovered) {
    return null;
  }

  const fields: ParsedLogMetadata = {};
  for (const match of matches) {
    fields[match[1]] = stripQuotes(match[2]);
  }

  return { fields };
}

function findTrailingJsonStart(raw: string) {
  let cursor = raw.lastIndexOf('{');
  while (cursor >= 0) {
    const suffix = raw.slice(cursor).trim();
    try {
      JSON.parse(suffix);
      return cursor;
    } catch {
      cursor = raw.lastIndexOf('{', cursor - 1);
    }
  }

  return -1;
}

function countKeywordMatches(text: string, keyword = '') {
  const normalizedKeyword = keyword.trim().toLowerCase();
  if (!normalizedKeyword) {
    return 0;
  }

  const normalizedText = text.toLowerCase();
  let count = 0;
  let cursor = normalizedText.indexOf(normalizedKeyword);
  while (cursor >= 0) {
    count += 1;
    cursor = normalizedText.indexOf(normalizedKeyword, cursor + normalizedKeyword.length);
  }
  return count;
}

function readFirstString(fields: ParsedLogMetadata, keys: string[]) {
  for (const key of keys) {
    const value = fields[key];
    if (typeof value === 'string' && value.trim()) {
      return value;
    }
    if (typeof value === 'number' || typeof value === 'boolean') {
      return String(value);
    }
  }
  return '';
}

function normalizeOptionalString(value?: string) {
  const normalized = value?.trim();
  return normalized || undefined;
}

function isPlainRecord(value: unknown): value is ParsedLogMetadata {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

function hasFields(fields: ParsedLogMetadata) {
  return Object.keys(fields).length > 0;
}

function isEmptyFieldValue(value: unknown) {
  return value === undefined || value === null || value === '';
}

function isStackTraceLike(trimmed: string, raw: string) {
  return STACK_SYMBOL_PATTERN.test(trimmed) || STACK_FILE_PATTERN.test(raw);
}

function shortenLogSource(source: string) {
  if (!source) return '';
  const parts = source.split(/\s+/);
  const lastPart = parts.at(-1) ?? source;
  const pathParts = lastPart.split('/');
  const basename = pathParts.at(-1) ?? lastPart;
  if (parts.length > 1) {
    return basename;
  }
  if (pathParts.length >= 2 && basename === 'logger.go:61') {
    return `${pathParts.at(-2)}/${basename}`;
  }
  return basename;
}

function isLowSignalMetadata(key: string, value: unknown) {
  if (
    key === 'request_id' ||
    key === 'client_request_id' ||
    key === 'trace_id' ||
    key === 'span_id' ||
    key === 'status' ||
    key === 'status_code' ||
    key === 'duration' ||
    key === 'latency_ms' ||
    key === 'path' ||
    key === 'method'
  ) {
    return false;
  }
  if (key === 'component') {
    return false;
  }
  if (key === 'service' && typeof value === 'string') {
    return value === 'sub2api' || value === 'sub2api-api';
  }
  if (key === 'env' && typeof value === 'string') {
    return value === 'production';
  }
  return LOW_SIGNAL_METADATA_PATTERNS.some((pattern) => pattern.test(key));
}

function stripQuotes(value: string) {
  const stripped = value.replace(/^["']|["']$/g, '');
  return stripped.replace(/\\"/g, '"').replace(/\\'/g, "'");
}
