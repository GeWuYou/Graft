import { stripAnsiControlSequences } from './ansi';
import type { LogStream, StructuredLogEntry } from './log-entry';
import type { LogLevel, LogToken } from './log-highlight';
import { detectLogLevel, getLogLevelTone, normalizeLogLevel, tokenizeLogLine } from './log-highlight';

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
  stream: LogStream;
  message: string;
  metadata: ParsedLogMetadata | null;
  raw: string;
  displayRaw: string;
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

/**
 * 解析原始容器日志行并自动识别其格式。
 *
 * @returns 包含识别格式、提取字段和计算元数据的解析结果
 */
export function parseContainerLogLine(rawLine: string): ParsedContainerLog {
  const raw = stripAnsiControlSequences(rawLine ?? '');
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

  return buildParsedLog({
    raw,
    level: detectLogLevel(trimmed) ?? 'LOG',
    message: trimmed,
    format: 'plain',
    fields: {},
  });
}

/**
 * 将结构化日志条目转换为包含解析消息、级别、来源和元数据的日志行。
 *
 * @param entry - 待解析的结构化日志条目
 * @param lineNo - 日志条目在源内容中的行号
 * @returns 包含标准化级别、时间戳、显示音调及容器日志解析结果的日志行
 */
export function parseLogLine(entry: StructuredLogEntry, lineNo: number): ParsedLogLine {
  const parsed = parseContainerLogLine(entry.line);
  const level = normalizeLogLevel(entry.level) ?? parsed.level ?? null;
  const timestamp = entry.occurredAt || parsed.time || '';

  return {
    lineNo,
    timestamp,
    level,
    source: parsed.source ?? '',
    sourceShort: shortenLogSource(parsed.source ?? ''),
    stream: entry.stream,
    message: parsed.message || parsed.raw,
    metadata: hasFields(parsed.fields) ? parsed.fields : null,
    raw: entry.line,
    displayRaw: parsed.raw,
    tone: getLogLevelTone(level),
    parsed: {
      ...parsed,
      time: timestamp,
    },
  };
}

/**
 * 将多个结构化日志条目解析为结构化日志行。
 *
 * @returns 解析后的日志行数组。
 */
export function parseLogLines(entries: StructuredLogEntry[]): ParsedLogLine[] {
  return entries.map((entry, index) => parseLogLine(entry, index + 1));
}

/**
 * 为解析后的日志行补充令牌化结果和搜索匹配信息。
 *
 * @param keyword - 用于高亮和统计匹配次数的可选搜索词
 * @returns 补充消息令牌、原始令牌和关键词匹配次数的 `DisplayLogLine`
 */
export function buildDisplayLogLine(line: ParsedLogLine, keyword = ''): DisplayLogLine {
  return {
    ...line,
    messageTokens: tokenizeLogLine(line.message, keyword),
    rawTokens: tokenizeLogLine(line.displayRaw, keyword),
    searchMatchCount: countKeywordMatches(line.displayRaw, keyword),
  };
}

/**
 * 通过选择重要字段并统计被排除字段数量来汇总元数据。
 *
 * 过滤低信号字段，只返回不超过指定上限的重要字段。
 *
 * @param maxVisible - 汇总中包含的重要字段数量上限
 * @returns 包含 `hiddenCount`（被排除字段数量）和 `tags`（可见元数据键值对数组）的对象
 */
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

/**
 * 将元数据值转换为显示字符串。
 *
 * @returns 转换后的显示字符串
 */
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

/**
 * 解析 JSON 日志行并提取标准日志字段。
 *
 * @param raw - 原始日志行
 * @param trimmed - 去除首尾空白后的日志行
 * @returns 提取字段后的容器日志；解析失败或 JSON 不是普通对象时返回 `null`
 */
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

/**
 * 解析 logfmt 格式的日志行。
 *
 * @param raw - 待解析的原始日志行
 * @returns 输入是有效 logfmt 时返回解析后的容器日志，否则返回 `null`
 */
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

/**
 * 解析结构化文本格式的日志行。
 *
 * 尝试匹配标准日志格式或通用结构化格式，提取时间戳、级别、来源和消息，
 * 并将末尾 JSON 对象提取为元数据。
 *
 * @returns 提取结构化字段后的容器日志；未识别到结构化格式时返回 `null`
 */
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

/**
 * 根据提取的字段和元数据组装容器日志解析结果。
 *
 * @returns 包含规范化字段、计算后的重要字段和显示数据的 `ParsedContainerLog`
 */
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

/**
 * 选择并排列用于显示的元数据字段。
 *
 * 过滤空值，可选移除低信号字段，根据预定义字段列表分配优先级，
 * 最后按优先级或原始顺序返回不超过 `maxVisible` 个字段。
 *
 * @param fields - 待处理的元数据
 * @param maxVisible - 返回字段数量上限
 * @param options.hideLowSignal - 为 `true` 时排除识别为低信号的字段
 * @param options.includeAll - 为 `true` 时包含所有字段，否则仅包含有已知优先级的字段
 * @param options.preserveFieldOrder - 为 `true` 时保留输入顺序，否则按优先级排序
 * @returns 带格式化值和优先级的不超过 `maxVisible` 个元数据字段
 */
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

/**
 * 从字符串末尾提取 JSON 对象作为元数据。
 *
 * 在输入末尾查找有效 JSON 对象，并将其与消息正文分离。
 *
 * @returns 包含消息正文和元数据对象的结果；未找到有效末尾 JSON 时返回完整输入和 `null` 元数据
 */
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

/**
 * 从原始文本解析 logfmt 格式的键值对。
 *
 * @returns 包含解析元数据字段的对象；输入不是有效 logfmt 时返回 `null`
 */
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

/**
 * 查找字符串中有效末尾 JSON 对象的起始索引。
 *
 * @param raw - 待搜索的字符串
 * @returns JSON 起始位置的索引；未找到有效 JSON 后缀时返回 `-1`
 */
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

/**
 * 使用不区分大小写的匹配统计文本中的关键词出现次数。
 *
 * @param text - 待搜索的文本
 * @param keyword - 待搜索的字符串；空字符串或仅含空白时返回 0
 * @returns 关键词在文本中不重叠出现的次数
 */
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

/**
 * 从元数据对象的指定键中返回首个非空字符串、数字或布尔值。
 *
 * @returns 首个匹配值的字符串形式；没有匹配值时返回空字符串
 */
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

/**
 * 去除字符串首尾空白，并将空值过滤为 `undefined`。
 *
 * @param value - 可选字符串值
 * @returns 非空的去空白字符串，否则返回 `undefined`
 */
function normalizeOptionalString(value?: string) {
  const normalized = value?.trim();
  return normalized || undefined;
}

/**
 * 判断值是否为普通对象。
 *
 * @returns 值是非空且非数组对象时返回 `true`，否则返回 `false`
 */
function isPlainRecord(value: unknown): value is ParsedLogMetadata {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}

/**
 * 判断元数据对象是否包含字段。
 *
 * @returns 元数据包含字段时返回 `true`，否则返回 `false`
 */
function hasFields(fields: ParsedLogMetadata) {
  return Object.keys(fields).length > 0;
}

/**
 * 判断值是否为空。
 *
 * @returns 值为 `undefined`、`null` 或空字符串时返回 `true`，否则返回 `false`
 */
function isEmptyFieldValue(value: unknown) {
  return value === undefined || value === null || value === '';
}

/**
 * 判断日志行是否具有堆栈跟踪特征。
 *
 * @returns 日志行匹配堆栈跟踪模式时返回 `true`，否则返回 `false`
 */
function isStackTraceLike(trimmed: string, raw: string) {
  return STACK_SYMBOL_PATTERN.test(trimmed) || STACK_FILE_PATTERN.test(raw);
}

/**
 * 从日志来源字符串提取缩短后的标识。
 *
 * 空格分隔的来源取最后一段，文件路径取其 basename；对于特定文件 `logger.go:61`，结果保留父目录。
 *
 * @param source - 待缩短的日志来源字符串
 * @returns 缩短后的来源标识；来源为空时返回空字符串
 */
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

/**
 * 判断元数据字段是否应作为低信号字段过滤。
 *
 * 请求 ID、trace ID、状态码、耗时、路径、方法和组件等已知重要字段不会被视为低信号；
 * 其他字段根据名称或值判断。
 *
 * @param key - 元数据字段名
 * @param value - 元数据字段值
 * @returns 字段属于低信号时返回 `true`，否则返回 `false`
 */
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

/**
 * 移除成对的外层引号，并将转义引号还原为普通字符。
 */
function stripQuotes(value: string) {
  const stripped = value.replace(/^["']|["']$/g, '');
  return stripped.replace(/\\"/g, '"').replace(/\\'/g, "'");
}
