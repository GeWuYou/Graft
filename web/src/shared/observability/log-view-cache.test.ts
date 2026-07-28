import { afterEach, describe, expect, it, vi } from 'vitest';

import type { StructuredLogEntry } from './log-entry';
import * as logParser from './log-parser';
import { LogViewCache } from './log-view-cache';

function createEntry(line: string, stream: 'stdout' | 'stderr' = 'stdout', occurredAt = '2026-06-26T03:00:00Z') {
  return {
    line,
    occurredAt,
    stream,
  } satisfies StructuredLogEntry;
}

describe('LogViewCache', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });
  it('reuses parsed lines for retained tail rows and only parses newly visible rows', () => {
    const parseSpy = vi.spyOn(logParser, 'parseLogLine');
    const cache = new LogViewCache();

    cache.buildView({
      entries: [createEntry('line-a'), createEntry('line-b'), createEntry('line-c')],
      lineLimit: 3,
      level: 'ALL',
      keyword: '',
    });
    expect(parseSpy).toHaveBeenCalledTimes(3);

    cache.buildView({
      entries: [createEntry('line-a'), createEntry('line-b'), createEntry('line-c'), createEntry('line-d')],
      lineLimit: 3,
      level: 'ALL',
      keyword: '',
    });

    expect(parseSpy).toHaveBeenCalledTimes(4);
    expect(parseSpy.mock.calls.at(-1)).toEqual([createEntry('line-d'), 0]);
  });

  it('reuses search payloads for retained rows when the keyword is unchanged', () => {
    const buildSpy = vi.spyOn(logParser, 'buildDisplayLogLine');
    const cache = new LogViewCache();

    cache.buildView({
      entries: [createEntry('request-a'), createEntry('request-b'), createEntry('request-c')],
      lineLimit: 3,
      level: 'ALL',
      keyword: 'request',
    });
    expect(buildSpy).toHaveBeenCalledTimes(3);

    cache.buildView({
      entries: [createEntry('request-a'), createEntry('request-b'), createEntry('request-c'), createEntry('request-d')],
      lineLimit: 3,
      level: 'ALL',
      keyword: 'request',
    });

    expect(buildSpy).toHaveBeenCalledTimes(4);
  });

  it('reuses search payloads when occurredAt is empty but the line carries a parseable inline timestamp', () => {
    const buildSpy = vi.spyOn(logParser, 'buildDisplayLogLine');
    const cache = new LogViewCache();
    const inlineTimestampLine = '2026-06-26T03:00:00Z INFO request-a';

    cache.buildView({
      entries: [createEntry(inlineTimestampLine, 'stdout', '')],
      lineLimit: 1,
      level: 'ALL',
      keyword: 'request',
    });
    expect(buildSpy).toHaveBeenCalledTimes(1);

    cache.buildView({
      entries: [createEntry(inlineTimestampLine, 'stdout', '')],
      lineLimit: 1,
      level: 'ALL',
      keyword: 'request',
    });

    expect(buildSpy).toHaveBeenCalledTimes(1);
  });

  it('drops prior keyword search payloads so keyword history does not accumulate unbounded cache entries', () => {
    const buildSpy = vi.spyOn(logParser, 'buildDisplayLogLine');
    const cache = new LogViewCache();

    cache.buildView({
      entries: [createEntry('request-a'), createEntry('request-b')],
      lineLimit: 2,
      level: 'ALL',
      keyword: 'request',
    });
    expect(buildSpy).toHaveBeenCalledTimes(2);

    cache.buildView({
      entries: [createEntry('request-a'), createEntry('request-b')],
      lineLimit: 2,
      level: 'ALL',
      keyword: 'error',
    });
    expect(buildSpy).toHaveBeenCalledTimes(4);

    cache.buildView({
      entries: [createEntry('request-a'), createEntry('request-b')],
      lineLimit: 2,
      level: 'ALL',
      keyword: 'error',
    });
    expect(buildSpy).toHaveBeenCalledTimes(4);

    cache.buildView({
      entries: [createEntry('request-a'), createEntry('request-b')],
      lineLimit: 2,
      level: 'ALL',
      keyword: 'request',
    });
    expect(buildSpy).toHaveBeenCalledTimes(6);
  });

  it('keeps visible line numbers aligned with the sliced tail', () => {
    const cache = new LogViewCache();

    const result = cache.buildView({
      entries: [createEntry('line-a'), createEntry('line-b'), createEntry('line-c'), createEntry('line-d')],
      lineLimit: 2,
      level: 'ALL',
      keyword: '',
    });

    expect(result.displayLines.map((line) => line.lineNo)).toEqual([3, 4]);
  });

  it('filters to matching lines while retaining the pre-search candidate count', () => {
    const cache = new LogViewCache();
    const result = cache.buildView({
      entries: [createEntry('request accepted'), createEntry('connection closed'), createEntry('request failed')],
      lineLimit: 3,
      level: 'ALL',
      keyword: 'request',
    });

    expect(result.totalCount).toBe(3);
    expect(result.matchCount).toBe(2);
    expect(result.displayLines.map((line) => line.message)).toEqual(['request accepted', 'request failed']);
  });

  it('counts level-filtered candidates before applying the keyword filter', () => {
    const cache = new LogViewCache();
    const result = cache.buildView({
      entries: [
        createEntry('2026-06-26T03:00:00Z ERROR request failed'),
        createEntry('2026-06-26T03:00:01Z ERROR connection closed'),
        createEntry('2026-06-26T03:00:02Z INFO request completed'),
      ],
      lineLimit: 3,
      level: 'ERROR',
      keyword: 'request',
    });

    expect(result.totalCount).toBe(2);
    expect(result.matchCount).toBe(1);
    expect(result.displayLines).toHaveLength(1);
  });

  it('keeps row keys stable for retained rows after a tail append', () => {
    const cache = new LogViewCache();
    const first = cache.buildView({
      entries: [createEntry('line-a'), createEntry('line-b'), createEntry('line-c')],
      lineLimit: 3,
      level: 'ALL',
      keyword: '',
    });
    const second = cache.buildView({
      entries: [createEntry('line-a'), createEntry('line-b'), createEntry('line-c'), createEntry('line-d')],
      lineLimit: 3,
      level: 'ALL',
      keyword: '',
    });

    expect(second.displayLines[0]?.rowKey).toBe(first.displayLines[1]?.rowKey);
    expect(second.displayLines[1]?.rowKey).toBe(first.displayLines[2]?.rowKey);
  });
});
