import { describe, expect, it } from 'vitest';

import { LogRingBuffer } from './log-ring-buffer';

describe('LogRingBuffer', () => {
  it('keeps insertion order before reaching capacity', () => {
    const buffer = new LogRingBuffer<string>(4);

    buffer.append('line-1');
    buffer.append('line-2');
    buffer.append('line-3');

    const snapshot = buffer.snapshot();

    expect(snapshot.size).toBe(3);
    expect(snapshot.capacity).toBe(4);
    expect(snapshot.oldestSeq).toBe(1);
    expect(snapshot.newestSeq).toBe(3);
    expect(snapshot.at(0)).toBe('line-1');
    expect(snapshot.at(1)).toBe('line-2');
    expect(snapshot.at(2)).toBe('line-3');
    expect(snapshot.seqAt(0)).toBe(1);
    expect(snapshot.seqAt(2)).toBe(3);
    expect(snapshot.toArray()).toEqual(['line-1', 'line-2', 'line-3']);
  });

  it('wraps around without moving existing slots and reports overwritten entries', () => {
    const buffer = new LogRingBuffer<string>(3);

    buffer.append('line-1');
    buffer.append('line-2');
    buffer.append('line-3');
    const result = buffer.append('line-4');

    const snapshot = buffer.snapshot();

    expect(result).toEqual({
      overwritten: 'line-1',
      overwrittenSeq: 1,
      seq: 4,
      version: 4,
    });
    expect(snapshot.size).toBe(3);
    expect(snapshot.oldestSeq).toBe(2);
    expect(snapshot.newestSeq).toBe(4);
    expect(snapshot.toArray()).toEqual(['line-2', 'line-3', 'line-4']);
  });

  it('treats overwrite as append and maintains monotonic seq and version', () => {
    const buffer = new LogRingBuffer<number>(2);

    const first = buffer.append(10);
    const second = buffer.overwrite(20);
    const third = buffer.overwrite(30);

    const snapshot = buffer.snapshot();

    expect(first.seq).toBe(1);
    expect(second.seq).toBe(2);
    expect(third.seq).toBe(3);
    expect(buffer.version()).toBe(3);
    expect(snapshot.oldestSeq).toBe(2);
    expect(snapshot.newestSeq).toBe(3);
    expect(snapshot.toArray()).toEqual([20, 30]);
  });

  it('clears contents in O(1) and bumps the version for future UI commits', () => {
    const buffer = new LogRingBuffer<string>(2);

    buffer.append('line-1');
    buffer.append('line-2');
    buffer.clear();
    const cleared = buffer.snapshot();
    expect(cleared.size).toBe(0);
    expect(cleared.oldestSeq).toBeNull();
    expect(cleared.newestSeq).toBeNull();
    expect(cleared.at(0)).toBeUndefined();
    expect(cleared.seqAt(0)).toBeUndefined();
    expect(cleared.toArray()).toEqual([]);

    const next = buffer.append('line-3');

    expect(buffer.size()).toBe(1);
    expect(buffer.capacity()).toBe(2);
    expect(next.seq).toBe(3);
    expect(buffer.version()).toBe(4);

    const snapshot = buffer.snapshot();

    expect(snapshot.size).toBe(1);
    expect(snapshot.oldestSeq).toBe(3);
    expect(snapshot.newestSeq).toBe(3);
    expect(snapshot.at(0)).toBe('line-3');
    expect(snapshot.seqAt(0)).toBe(3);
    expect(snapshot.toArray()).toEqual(['line-3']);
  });

  it('returns a readonly accessor snapshot view tied to buffer order at read time', () => {
    const buffer = new LogRingBuffer<string>(3);

    buffer.append('line-1');
    buffer.append('line-2');

    const snapshot = buffer.snapshot();

    expect(Object.isFrozen(snapshot)).toBe(true);
    expect(snapshot.version).toBe(2);
    expect(snapshot.toArray()).toEqual(['line-1', 'line-2']);

    buffer.append('line-3');

    expect(snapshot.version).toBe(3);
    expect(snapshot.size).toBe(3);
    expect(snapshot.toArray()).toEqual(['line-1', 'line-2', 'line-3']);
  });

  it('rejects invalid capacity values', () => {
    expect(() => new LogRingBuffer(0)).toThrow('RingBuffer capacity must be a positive integer');
    expect(() => new LogRingBuffer(1.5)).toThrow('RingBuffer capacity must be a positive integer');
  });
});
