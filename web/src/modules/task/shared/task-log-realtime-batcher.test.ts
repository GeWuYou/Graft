import { describe, expect, it } from 'vitest';

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
  it('uses the durable cursor and retains only the newest bounded log entries', () => {
    const commits: ReturnType<typeof response>[] = [];
    const snapshots: string[][] = [];
    const batcher = new TaskLogRealtimeBatcher({
      capacity: 2,
      onCommit: (snapshot) => snapshots.push(snapshot.entries.map((item) => item.line)),
    });

    batcher.seed(response([entry(1), entry(2)], 2));
    batcher.append(response([entry(3)], 3));

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

  it('does not commit a duplicate snapshot for an empty response with an unchanged cursor', () => {
    const commits: number[] = [];
    const batcher = new TaskLogRealtimeBatcher({
      onCommit: (snapshot) => commits.push(snapshot.contentVersion),
    });

    batcher.seed(response([entry(1)], 1));
    batcher.append(response([], 1));

    expect(commits).toEqual([1]);
  });

  it('merges an older page ahead of the tail without duplicating a raced realtime entry', () => {
    const snapshots: string[][] = [];
    const batcher = new TaskLogRealtimeBatcher({
      capacity: 10,
      onCommit: (snapshot) => snapshots.push(snapshot.entries.map((item) => item.line)),
    });

    batcher.seed(response([entry(4), entry(5)], 5));
    batcher.prepend(response([entry(2), entry(3), entry(4)], 4));
    batcher.append(response([entry(6)], 6));

    expect(snapshots.at(-1)).toEqual(['line-2', 'line-3', 'line-4', 'line-5', 'line-6']);
    expect(batcher.oldestSequence()).toBe(2);
    expect(batcher.nextAfterSequence()).toBe(6);
  });
});
