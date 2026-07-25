import { afterEach, describe, expect, it, vi } from 'vitest';

import type { TaskLogEntry, TaskLogResponse } from '../types/task';
import { TaskLogRealtimeBatcher } from './task-log-realtime-batcher';

function entry(sequence: number): TaskLogEntry {
  return {
    id: sequence,
    sequence,
    stream: 'stdout',
    level: 'info',
    line: `line-${sequence}`,
    occurred_at: `2026-07-11T00:00:${String(sequence).padStart(2, '0')}Z`,
  };
}

function response(items: TaskLogEntry[], nextAfterSequence: number): TaskLogResponse {
  return { items, next_after_sequence: nextAfterSequence };
}

describe('TaskLogRealtimeBatcher', () => {
  afterEach(() => vi.useRealTimers());

  it('uses the durable cursor and retains only the newest bounded log entries', () => {
    vi.useFakeTimers();
    const commits: ReturnType<typeof response>[] = [];
    const snapshots: string[][] = [];
    const batcher = new TaskLogRealtimeBatcher({
      capacity: 2,
      onCommit: (snapshot) => snapshots.push(snapshot.entries.map((item) => item.line)),
    });

    batcher.seed(response([entry(1), entry(2)], 2));
    batcher.append(response([entry(3)], 3));
    vi.advanceTimersByTime(100);

    expect(commits).toEqual([]);
    expect(batcher.nextAfterSequence()).toBe(3);
    expect(snapshots.at(-1)).toEqual(['line-2', 'line-3']);
  });

  it('keeps a newer cursor when an empty replay page races a later notification', () => {
    const batcher = new TaskLogRealtimeBatcher({ capacity: 2, onCommit: () => undefined });

    batcher.seed(response([entry(4)], 4));
    batcher.append(response([], 4));

    expect(batcher.nextAfterSequence()).toBe(4);
  });

  it('retains every loaded page by default for the virtual log viewport', () => {
    const snapshots: string[][] = [];
    const batcher = new TaskLogRealtimeBatcher({
      onCommit: (snapshot) => snapshots.push(snapshot.entries.map((item) => item.line)),
    });

    batcher.seed(response([entry(1), entry(2)], 2));
    batcher.append(response([entry(3)], 3));

    expect(snapshots.at(-1)).toEqual(['line-1', 'line-2']);
    expect(batcher.nextAfterSequence()).toBe(3);
  });

  it('does not commit a duplicate snapshot for an empty response with an unchanged cursor', () => {
    const commits: number[] = [];
    const batcher = new TaskLogRealtimeBatcher({
      onCommit: (snapshot) => commits.push(snapshot.contentVersion),
    });

    batcher.seed(response([entry(1)], 1));
    batcher.append(response([], 1));

    expect(commits).toEqual([1]);
  });

  it('commits a burst of durable replay pages once with the newest snapshot', () => {
    vi.useFakeTimers();
    const snapshots: string[][] = [];
    const batcher = new TaskLogRealtimeBatcher({
      onCommit: (snapshot) => snapshots.push(snapshot.entries.map((item) => item.line)),
    });

    batcher.seed(response([entry(1)], 1));
    batcher.append(response([entry(2)], 2));
    batcher.append(response([entry(3)], 3));

    expect(snapshots).toEqual([['line-1']]);
    vi.advanceTimersByTime(100);
    expect(snapshots).toEqual([['line-1'], ['line-1', 'line-2', 'line-3']]);
  });

  it('clears a pending visual commit when the drawer closes', () => {
    vi.useFakeTimers();
    const onCommit = vi.fn();
    const batcher = new TaskLogRealtimeBatcher({ onCommit });

    batcher.seed(response([entry(1)], 1));
    batcher.append(response([entry(2)], 2));
    batcher.destroy();
    vi.runAllTimers();

    expect(onCommit).toHaveBeenCalledOnce();
  });

  it('yields a large initial replay and abandons it after the drawer closes', async () => {
    vi.useFakeTimers();
    const onCommit = vi.fn();
    const batcher = new TaskLogRealtimeBatcher({ onCommit });
    const replay = batcher.seedDeferred(
      response(
        Array.from({ length: 100 }, (_, index) => entry(index + 1)),
        100,
      ),
    );

    await vi.advanceTimersByTimeAsync(0);
    batcher.clear();
    await vi.runAllTimersAsync();
    await replay;

    expect(onCommit).not.toHaveBeenCalled();
  });
});
