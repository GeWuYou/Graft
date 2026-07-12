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
});
