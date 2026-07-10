import { describe, expect, it } from 'vitest';

import { buildTaskRealtimeTopicName, parseTaskRealtimeNotification } from './realtime';

describe('task realtime contract', () => {
  it('builds the canonical task topic and parses its persisted-fact notification', () => {
    expect(buildTaskRealtimeTopicName(42)).toBe('task:42');
    expect(parseTaskRealtimeNotification(JSON.stringify({ data: { task_id: 42, type: 'task.log.appended' } }))).toEqual(
      {
        task_id: 42,
        type: 'task.log.appended',
      },
    );
  });

  it('rejects malformed task notifications before they can trigger a durable backfill', () => {
    expect(parseTaskRealtimeNotification(JSON.stringify({ data: { task_id: '42', type: 'task.updated' } }))).toBeNull();
    expect(parseTaskRealtimeNotification('not-json')).toBeNull();
  });
});
