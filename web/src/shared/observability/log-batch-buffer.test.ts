import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { LogBatchBuffer } from './log-batch-buffer';

describe('LogBatchBuffer', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('flushes pending items once after the default time window', () => {
    const flushed: string[][] = [];
    const buffer = new LogBatchBuffer<string>({
      onFlush: (batch) => {
        flushed.push([...batch]);
      },
    });

    buffer.append('line-1');
    buffer.append('line-2');

    expect(buffer.pendingSize()).toBe(2);
    vi.advanceTimersByTime(99);
    expect(flushed).toEqual([]);

    vi.advanceTimersByTime(1);

    expect(flushed).toEqual([['line-1', 'line-2']]);
    expect(buffer.pendingSize()).toBe(0);
  });

  it('flushes early when the count threshold is reached', () => {
    const flushed: number[][] = [];
    const buffer = new LogBatchBuffer<number>({
      flushIntervalMs: 500,
      maxBatchSize: 3,
      onFlush: (batch) => {
        flushed.push([...batch]);
      },
    });

    buffer.appendMany([1, 2]);
    expect(flushed).toEqual([]);

    buffer.append(3);

    expect(flushed).toEqual([[1, 2, 3]]);
    expect(buffer.pendingSize()).toBe(0);
    vi.advanceTimersByTime(500);
    expect(flushed).toEqual([[1, 2, 3]]);
  });

  it('splits appendMany input into threshold-sized flushes and keeps only the remainder pending', () => {
    const flushed: string[][] = [];
    const buffer = new LogBatchBuffer<string>({
      flushIntervalMs: 500,
      maxBatchSize: 3,
      onFlush: (batch) => {
        flushed.push([...batch]);
      },
    });

    const pendingSize = buffer.appendMany(['line-1', 'line-2', 'line-3', 'line-4', 'line-5', 'line-6', 'line-7']);

    expect(flushed).toEqual([
      ['line-1', 'line-2', 'line-3'],
      ['line-4', 'line-5', 'line-6'],
    ]);
    expect(pendingSize).toBe(1);
    expect(buffer.pendingSize()).toBe(1);

    vi.advanceTimersByTime(500);
    expect(flushed).toEqual([['line-1', 'line-2', 'line-3'], ['line-4', 'line-5', 'line-6'], ['line-7']]);
  });

  it('supports manual flush and cancels the scheduled timer', () => {
    const flushed: string[][] = [];
    const buffer = new LogBatchBuffer<string>({
      flushIntervalMs: 250,
      onFlush: (batch) => {
        flushed.push([...batch]);
      },
    });

    buffer.appendMany(['line-1', 'line-2']);
    const manual = buffer.flush();

    expect([...manual]).toEqual(['line-1', 'line-2']);
    expect(flushed).toEqual([['line-1', 'line-2']]);
    vi.advanceTimersByTime(250);
    expect(flushed).toEqual([['line-1', 'line-2']]);
  });

  it('clear drops pending items without flushing them', () => {
    const flushed: string[][] = [];
    const buffer = new LogBatchBuffer<string>({
      onFlush: (batch) => {
        flushed.push([...batch]);
      },
    });

    buffer.appendMany(['line-1', 'line-2']);
    buffer.clear();

    vi.runOnlyPendingTimers();
    expect(flushed).toEqual([]);
    expect(buffer.pendingSize()).toBe(0);
  });

  it('destroy clears timers and ignores future appends', () => {
    const flushed: string[][] = [];
    const buffer = new LogBatchBuffer<string>({
      onFlush: (batch) => {
        flushed.push([...batch]);
      },
    });

    buffer.append('line-1');
    buffer.destroy();
    buffer.append('line-2');

    vi.runOnlyPendingTimers();
    expect(flushed).toEqual([]);
    expect(buffer.pendingSize()).toBe(0);
  });

  it('rejects invalid scheduler configuration', () => {
    expect(() => new LogBatchBuffer({ flushIntervalMs: 0, onFlush: () => undefined })).toThrow(
      'LogBatchBuffer flushIntervalMs must be a positive integer',
    );
    expect(() => new LogBatchBuffer({ maxBatchSize: 1.5, onFlush: () => undefined })).toThrow(
      'LogBatchBuffer maxBatchSize must be a positive integer',
    );
  });
});
