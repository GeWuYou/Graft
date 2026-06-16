// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export type LogLevel = 'FATAL' | 'ERROR' | 'WARN' | 'INFO' | 'DEBUG' | 'TRACE';
export type LogTokenType = 'text' | 'keyword' | 'field-key' | 'field-value' | 'level';
export type LogToken = {
  text: string;
  type: LogTokenType;
  level?: LogLevel;
};

const FIELD_PATTERN = /\b([A-Za-z_][\w.-]*)=("[^"]*"|'[^']*'|\S*)/g;
const LEVEL_PATTERN = /\blevel=(?:"|')?(fatal|error|warn|warning|info|debug|trace)(?:"|')?\b/i;
const STANDALONE_LEVEL_PATTERN = /\b(fatal|error|warn|warning|info|debug|trace)\b/i;

export function detectLogLevel(line: string): LogLevel | null {
  const fieldMatch = LEVEL_PATTERN.exec(line);
  const rawLevel = fieldMatch?.[1] ?? STANDALONE_LEVEL_PATTERN.exec(line)?.[1];
  return normalizeLogLevel(rawLevel);
}

export function getLogLevelTone(level: LogLevel | null) {
  if (level === 'FATAL' || level === 'ERROR') return 'danger';
  if (level === 'WARN') return 'warning';
  if (level === 'INFO') return 'info';
  if (level === 'DEBUG' || level === 'TRACE') return 'muted';
  return 'default';
}

export function tokenizeLogLine(line: string, keyword = ''): LogToken[] {
  const tokens: LogToken[] = [];
  const normalizedKeyword = keyword.trim();
  let cursor = 0;

  for (const match of line.matchAll(FIELD_PATTERN)) {
    const index = match.index ?? 0;
    const [fullText, key, value] = match;
    if (index > cursor) {
      tokens.push(...tokenizeKeyword(line.slice(cursor, index), normalizedKeyword));
    }

    const normalizedLevel = key.toLowerCase() === 'level' ? normalizeLogLevel(stripQuotes(value)) : null;
    tokens.push({ text: key, type: 'field-key' });
    tokens.push({ text: '=', type: 'text' });
    if (normalizedLevel) {
      tokens.push({
        text: value,
        type: 'level',
        level: normalizedLevel,
      });
    } else {
      tokens.push(...tokenizeKeyword(value, normalizedKeyword, 'field-value'));
    }
    cursor = index + fullText.length;
  }

  if (cursor < line.length) {
    tokens.push(...tokenizeKeyword(line.slice(cursor), normalizedKeyword));
  }

  return tokens.length ? tokens : [{ text: line, type: 'text' }];
}

function normalizeLogLevel(value?: string | null): LogLevel | null {
  if (!value) return null;
  const normalized = value.toUpperCase();
  if (normalized === 'WARNING') return 'WARN';
  if (
    normalized === 'FATAL' ||
    normalized === 'ERROR' ||
    normalized === 'WARN' ||
    normalized === 'INFO' ||
    normalized === 'DEBUG' ||
    normalized === 'TRACE'
  ) {
    return normalized;
  }
  return null;
}

function tokenizeKeyword(text: string, keyword: string, defaultType: LogTokenType = 'text'): LogToken[] {
  if (!keyword) {
    return text ? [{ text, type: defaultType }] : [];
  }

  const tokens: LogToken[] = [];
  const lowerText = text.toLowerCase();
  const lowerKeyword = keyword.toLowerCase();
  let cursor = 0;
  let nextIndex = lowerText.indexOf(lowerKeyword);

  while (nextIndex >= 0) {
    if (nextIndex > cursor) {
      tokens.push({ text: text.slice(cursor, nextIndex), type: defaultType });
    }
    tokens.push({ text: text.slice(nextIndex, nextIndex + keyword.length), type: 'keyword' });
    cursor = nextIndex + keyword.length;
    nextIndex = lowerText.indexOf(lowerKeyword, cursor);
  }

  if (cursor < text.length) {
    tokens.push({ text: text.slice(cursor), type: defaultType });
  }

  return tokens;
}

function stripQuotes(value: string) {
  return value.replace(/^["']|["']$/g, '');
}
