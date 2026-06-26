import { afterEach, describe, expect, it, vi } from 'vitest';

import * as logParser from './log-parser';
import { LogViewCache } from './log-view-cache';

describe('LogViewCache', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });
  it('reuses parsed lines for retained tail rows and only parses newly visible rows', () => {
    const parseSpy = vi.spyOn(logParser, 'parseLogLine');
    const cache = new LogViewCache();

    cache.buildView({
      lines: ['line-a', 'line-b', 'line-c'],
      lineLimit: 3,
      level: 'ALL',
      keyword: '',
    });
    expect(parseSpy).toHaveBeenCalledTimes(3);

    cache.buildView({
      lines: ['line-a', 'line-b', 'line-c', 'line-d'],
      lineLimit: 3,
      level: 'ALL',
      keyword: '',
    });

    expect(parseSpy).toHaveBeenCalledTimes(4);
    expect(parseSpy.mock.calls.at(-1)).toEqual(['line-d', 0]);
  });

  it('reuses search payloads for retained rows when the keyword is unchanged', () => {
    const buildSpy = vi.spyOn(logParser, 'buildDisplayLogLine');
    const cache = new LogViewCache();

    cache.buildView({
      lines: ['request-a', 'request-b', 'request-c'],
      lineLimit: 3,
      level: 'ALL',
      keyword: 'request',
    });
    expect(buildSpy).toHaveBeenCalledTimes(3);

    cache.buildView({
      lines: ['request-a', 'request-b', 'request-c', 'request-d'],
      lineLimit: 3,
      level: 'ALL',
      keyword: 'request',
    });

    expect(buildSpy).toHaveBeenCalledTimes(4);
  });

  it('drops prior keyword search payloads so keyword history does not accumulate unbounded cache entries', () => {
    const buildSpy = vi.spyOn(logParser, 'buildDisplayLogLine');
    const cache = new LogViewCache();

    cache.buildView({
      lines: ['request-a', 'request-b'],
      lineLimit: 2,
      level: 'ALL',
      keyword: 'request',
    });
    expect(buildSpy).toHaveBeenCalledTimes(2);

    cache.buildView({
      lines: ['request-a', 'request-b'],
      lineLimit: 2,
      level: 'ALL',
      keyword: 'error',
    });
    expect(buildSpy).toHaveBeenCalledTimes(4);

    cache.buildView({
      lines: ['request-a', 'request-b'],
      lineLimit: 2,
      level: 'ALL',
      keyword: 'error',
    });
    expect(buildSpy).toHaveBeenCalledTimes(4);

    cache.buildView({
      lines: ['request-a', 'request-b'],
      lineLimit: 2,
      level: 'ALL',
      keyword: 'request',
    });
    expect(buildSpy).toHaveBeenCalledTimes(6);
  });

  it('keeps visible line numbers aligned with the sliced tail', () => {
    const cache = new LogViewCache();

    const result = cache.buildView({
      lines: ['line-a', 'line-b', 'line-c', 'line-d'],
      lineLimit: 2,
      level: 'ALL',
      keyword: '',
    });

    expect(result.displayLines.map((line) => line.lineNo)).toEqual([3, 4]);
  });
});
