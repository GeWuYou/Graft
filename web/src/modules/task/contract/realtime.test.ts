import { describe, expect, it } from 'vitest';

import { buildTaskRealtimeTopicName, parseTaskRealtimeNotification, type TaskRealtimeEventType } from './realtime';

describe('task realtime contract', () => {
  it('builds the canonical task topic and parses its persisted-fact notification', () => {
    const knownEvent: TaskRealtimeEventType = 'task.log.appended';

    expect(buildTaskRealtimeTopicName(42)).toBe('task:42');
    expect(parseTaskRealtimeNotification(JSON.stringify({ data: { task_id: 42, type: knownEvent } }))).toEqual({
      task_id: 42,
      type: 'task.log.appended',
    });
  });

  it('rejects malformed task notifications before they can trigger a durable backfill', () => {
    expect(parseTaskRealtimeNotification(JSON.stringify({ data: { task_id: '42', type: 'task.updated' } }))).toBeNull();
    expect(parseTaskRealtimeNotification('not-json')).toBeNull();
  });

  it('preserves unknown event names for forward-compatible durable backfills', () => {
    expect(parseTaskRealtimeNotification(JSON.stringify({ data: { task_id: 42, type: 'task.future.event' } }))).toEqual(
      {
        task_id: 42,
        type: 'task.future.event',
      },
    );
  });
});
