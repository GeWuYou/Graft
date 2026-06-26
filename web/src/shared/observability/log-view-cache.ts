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
  readonly #parsedByRaw = new Map<string, CachedParsedLogLine>();
  readonly #searchByRawKeyword = new Map<string, CachedSearchPayload>();

  buildView(options: {
    lines: readonly string[];
    lineLimit: number;
    level: LevelFilter;
    keyword: string;
  }): LogViewResult {
    const visibleLines = selectVisibleLines(options.lines, options.lineLimit);
    const visibleRawSet = new Set(visibleLines);

    this.#pruneParsedCache(visibleRawSet);
    this.#pruneSearchCache(visibleRawSet);

    const lineNoOffset = options.lines.length - visibleLines.length;
    const displayLines: DisplayLogLine[] = [];
    let matchCount = 0;

    visibleLines.forEach((raw, index) => {
      const parsedLine = this.#materializeParsedLine(raw, lineNoOffset + index + 1);
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

  #materializeParsedLine(raw: string, lineNo: number): ParsedLogLine {
    const cached = this.#parsedByRaw.get(raw) ?? this.#cacheParsedLine(raw);
    return {
      ...cached,
      lineNo,
    };
  }

  #cacheParsedLine(raw: string) {
    const parsedLine = parseLogLine(raw, 0);
    const cached: CachedParsedLogLine = {
      ...parsedLine,
    };
    this.#parsedByRaw.set(raw, cached);
    return cached;
  }

  #resolveSearchPayload(line: ParsedLogLine, keyword: string) {
    const cacheKey = `${keyword}\u0000${line.raw}`;
    const cached = this.#searchByRawKeyword.get(cacheKey);
    if (cached) {
      return cached;
    }

    const next = buildDisplayLogLine(line, keyword);
    const payload: CachedSearchPayload = {
      messageTokens: next.messageTokens,
      rawTokens: next.rawTokens,
      searchMatchCount: next.searchMatchCount,
    };
    this.#searchByRawKeyword.set(cacheKey, payload);
    return payload;
  }

  #pruneParsedCache(visibleRawSet: ReadonlySet<string>) {
    for (const raw of this.#parsedByRaw.keys()) {
      if (!visibleRawSet.has(raw)) {
        this.#parsedByRaw.delete(raw);
      }
    }
  }

  #pruneSearchCache(visibleRawSet: ReadonlySet<string>) {
    for (const cacheKey of this.#searchByRawKeyword.keys()) {
      const separatorIndex = cacheKey.indexOf('\u0000');
      const raw = separatorIndex >= 0 ? cacheKey.slice(separatorIndex + 1) : cacheKey;
      if (!visibleRawSet.has(raw)) {
        this.#searchByRawKeyword.delete(cacheKey);
      }
    }
  }
}

/**
 * 选取可见的日志行。
 *
 * @param lines - 原始日志行列表
 * @param lineLimit - 可见行数量上限
 * @returns 最多包含末尾 `lineLimit` 条日志行的数组；当 `lineLimit` 小于或等于 `0` 时返回空数组
 */
function selectVisibleLines(lines: readonly string[], lineLimit: number) {
  if (lineLimit <= 0) {
    return [] as readonly string[];
  }

  return lines.slice(-lineLimit);
}
