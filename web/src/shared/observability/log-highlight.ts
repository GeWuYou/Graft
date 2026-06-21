export type LogLevel = 'FATAL' | 'ERROR' | 'WARN' | 'INFO' | 'DEBUG' | 'TRACE' | 'LOG' | 'UNKNOWN';
export type LogTokenType = 'text' | 'keyword' | 'field-key' | 'field-value' | 'level';
export type LogToken = {
  text: string;
  type: LogTokenType;
  level?: LogLevel;
};

const FIELD_PATTERN = /\b([A-Za-z_][\w.-]*)=("[^"]*"|'[^']*'|\S*)/g;
const LEVEL_PATTERN = /\blevel=(?:"|')?(fatal|error|err|warn|warning|info|debug|trace|log|unknown)(?:"|')?\b/i;
const STANDALONE_LEVEL_PATTERN = /\b(fatal|error|err|warn|warning|info|debug|trace)\b/i;

/**
 * Detects the log level in a line of text.
 *
 * @returns The detected log level, or `null` if no level was found.
 */
export function detectLogLevel(line: string): LogLevel | null {
  const fieldMatch = LEVEL_PATTERN.exec(line);
  const rawLevel = fieldMatch?.[1] ?? STANDALONE_LEVEL_PATTERN.exec(line)?.[1];
  return normalizeLogLevel(rawLevel);
}

/**
 * Maps a log level to a visual tone indicating severity.
 *
 * @param level - The log level to map
 * @returns The tone string corresponding to the log level: 'danger', 'warning', 'info', 'muted', or 'default'
 */
export function getLogLevelTone(level: LogLevel | null) {
  if (level === 'FATAL' || level === 'ERROR') return 'danger';
  if (level === 'WARN') return 'warning';
  if (level === 'INFO') return 'info';
  if (level === 'DEBUG' || level === 'TRACE' || level === 'LOG' || level === 'UNKNOWN') return 'muted';
  return 'default';
}

/**
 * Breaks a log line into tokens for highlighting and semantic analysis, extracting field pairs and identifying log levels.
 *
 * @param line - The log line text to tokenize
 * @param keyword - An optional keyword to highlight within the line
 * @returns An array of log tokens; if no tokens are generated, returns a single token containing the entire line
 */
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

/**
 * Normalizes a log level string to a standard LogLevel value.
 *
 * Maps common aliases such as 'err' to 'error' and 'warning' to 'warn'.
 *
 * @returns A standard LogLevel if the input matches a recognized level, or null otherwise.
 */
export function normalizeLogLevel(value?: string | null): LogLevel | null {
  if (!value) return null;
  const normalized = value.toUpperCase();
  if (normalized === 'ERR') return 'ERROR';
  if (normalized === 'WARNING') return 'WARN';
  if (
    normalized === 'FATAL' ||
    normalized === 'ERROR' ||
    normalized === 'WARN' ||
    normalized === 'INFO' ||
    normalized === 'DEBUG' ||
    normalized === 'TRACE' ||
    normalized === 'LOG' ||
    normalized === 'UNKNOWN'
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
