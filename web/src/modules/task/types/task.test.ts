import { describe, expect, it } from 'vitest';

import { taskLogEntriesToStructured, type TaskLogEntry } from './task';

describe('taskLogEntriesToStructured', () => {
  it('preserves the snake_case fields returned by the task log API', () => {
    const entries = [
      {
        id: 1,
        sequence: 2,
        stage_id: 3,
        stream: 'stderr',
        level: 'error',
        line: 'compose failed',
        occurred_at: '2026-07-10T13:36:29Z',
      },
    ] as TaskLogEntry[];

    expect(taskLogEntriesToStructured(entries)).toEqual([
      {
        level: 'error',
        line: 'compose failed',
        occurredAt: '2026-07-10T13:36:29Z',
        stream: 'stderr',
      },
    ]);
  });
});
