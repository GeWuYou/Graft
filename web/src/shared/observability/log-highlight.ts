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
 * 从文本行中识别日志级别。
 *
 * @returns 识别出的日志级别；未识别到时返回 `null`
 */
export function detectLogLevel(line: string): LogLevel | null {
  const fieldMatch = LEVEL_PATTERN.exec(line);
  const rawLevel = fieldMatch?.[1] ?? STANDALONE_LEVEL_PATTERN.exec(line)?.[1];
  return normalizeLogLevel(rawLevel);
}

/**
 * 将日志级别映射为表示严重程度的视觉色调。
 *
 * @param level - 待映射的日志级别
 * @returns 对应的色调键：`danger`、`warning`、`info`、`muted` 或 `default`
 */
export function getLogLevelTone(level: LogLevel | null) {
  if (level === 'FATAL' || level === 'ERROR') return 'danger';
  if (level === 'WARN') return 'warning';
  if (level === 'INFO') return 'info';
  if (level === 'DEBUG' || level === 'TRACE' || level === 'LOG' || level === 'UNKNOWN') return 'muted';
  return 'default';
}

/**
 * 将日志行拆分为用于高亮和语义分析的令牌，同时提取字段对并识别日志级别。
 *
 * @param line - 待拆分的日志行文本
 * @param keyword - 可选的行内高亮关键词
 * @returns 日志令牌数组；未生成令牌时返回包含整行文本的单个令牌
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
 * 将日志级别字符串规范化为标准 `LogLevel` 值。
 *
 * 处理 `err` 到 `error`、`warning` 到 `warn` 等常见别名。
 *
 * @returns 输入匹配已知级别时返回标准 `LogLevel`，否则返回 `null`
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
