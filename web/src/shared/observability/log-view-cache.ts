import type { StructuredLogEntry } from './log-entry';
import type { LogLevel } from './log-highlight';
import { buildDisplayLogLine, type DisplayLogLine, type ParsedLogLine, parseLogLine } from './log-parser';

type LevelFilter = 'ALL' | LogLevel;

type CachedParsedLogLine = Omit<ParsedLogLine, 'lineNo'>;
type CachedSearchPayload = Pick<DisplayLogLine, 'messageTokens' | 'rawTokens' | 'searchMatchCount'>;

export type LogViewResult = Readonly<{
  displayLines: DisplayLogLine[];
  matchCount: number;
}>;

export class LogViewCache {
  readonly #parsedByKey = new Map<string, CachedParsedLogLine>();
  readonly #searchByKey = new Map<string, CachedSearchPayload>();
  #activeKeyword = '';

  buildView(options: {
    entries: readonly StructuredLogEntry[];
    lineLimit: number;
    level: LevelFilter;
    keyword: string;
  }): LogViewResult {
    const visibleEntries = selectVisibleEntries(options.entries, options.lineLimit);
    const visibleKeySet = new Set(visibleEntries.map(buildEntryCacheKey));

    this.#pruneParsedCache(visibleKeySet);
    this.#resetSearchCacheForKeyword(options.keyword);
    this.#pruneSearchCache(visibleKeySet);

    const lineNoOffset = options.entries.length - visibleEntries.length;
    const displayLines: DisplayLogLine[] = [];
    let matchCount = 0;

    visibleEntries.forEach((entry, index) => {
      const parsedLine = this.#materializeParsedLine(entry, lineNoOffset + index + 1);
      if (options.level !== 'ALL' && parsedLine.level !== options.level) {
        return;
      }

      const searchPayload = this.#resolveSearchPayload(parsedLine, options.keyword);
      matchCount += searchPayload.searchMatchCount;
      displayLines.push({
        ...parsedLine,
        ...searchPayload,
      });
    });

    return {
      displayLines,
      matchCount,
    };
  }

  #materializeParsedLine(entry: StructuredLogEntry, lineNo: number): ParsedLogLine {
    const cacheKey = buildEntryCacheKey(entry);
    const cached = this.#parsedByKey.get(cacheKey) ?? this.#cacheParsedLine(entry, cacheKey);
    return {
      ...cached,
      lineNo,
    };
  }

  #cacheParsedLine(entry: StructuredLogEntry, cacheKey: string) {
    const parsedLine = parseLogLine(entry, 0);
    const cached: CachedParsedLogLine = {
      ...parsedLine,
    };
    this.#parsedByKey.set(cacheKey, cached);
    return cached;
  }

  #resolveSearchPayload(line: ParsedLogLine, keyword: string) {
    const cacheKey = buildEntryCacheKey({
      line: line.raw,
      occurredAt: line.timestamp,
      stream: line.stream,
    });
    const cached = this.#searchByKey.get(cacheKey);
    if (cached) {
      return cached;
    }

    const next = buildDisplayLogLine(line, keyword);
    const payload: CachedSearchPayload = {
      messageTokens: next.messageTokens,
      rawTokens: next.rawTokens,
      searchMatchCount: next.searchMatchCount,
    };
    this.#searchByKey.set(cacheKey, payload);
    return payload;
  }

  #resetSearchCacheForKeyword(keyword: string) {
    if (this.#activeKeyword === keyword) {
      return;
    }

    this.#activeKeyword = keyword;
    this.#searchByKey.clear();
  }

  #pruneParsedCache(visibleKeySet: ReadonlySet<string>) {
    for (const key of this.#parsedByKey.keys()) {
      if (!visibleKeySet.has(key)) {
        this.#parsedByKey.delete(key);
      }
    }
  }

  #pruneSearchCache(visibleKeySet: ReadonlySet<string>) {
    for (const key of this.#searchByKey.keys()) {
      if (!visibleKeySet.has(key)) {
        this.#searchByKey.delete(key);
      }
    }
  }
}

/**
 * 选取末尾的可见日志条目。
 *
 * @param entries - 日志条目列表
 * @param lineLimit - 可见条目数量上限
 * @returns 最多包含末尾 `lineLimit` 条日志条目的数组；当 `lineLimit` 小于或等于 `0` 时返回空数组
 */
function selectVisibleEntries(entries: readonly StructuredLogEntry[], lineLimit: number) {
  if (lineLimit <= 0) {
    return [] as readonly StructuredLogEntry[];
  }

  return entries.slice(-lineLimit);
}

/**
 * 生成日志条目的稳定缓存键。
 *
 * @param entry - 要生成缓存键的结构化日志条目
 * @returns 由 `occurredAt`、`stream` 和 `line` 组成的字符串缓存键
 */
function buildEntryCacheKey(entry: StructuredLogEntry) {
  return `${entry.occurredAt}\u0000${entry.stream}\u0000${entry.line}`;
}
