import { flushPromises } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';

const taskApiMocks = vi.hoisted(() => ({ getTask: vi.fn() }));
const realtimeMocks = vi.hoisted(() => ({
  controller: { close: vi.fn() },
  open: vi.fn(),
}));

vi.mock('./api/task', () => ({ getTask: taskApiMocks.getTask }));
vi.mock('@/shared/realtime', () => ({
  openRealtimeTopicSocket: realtimeMocks.open,
}));

import { isTerminalTaskStatus, observeTask } from './task-observer';

describe('observeTask', () => {
  afterEach(() => {
    vi.clearAllMocks();
    vi.useRealTimers();
  });

  it('refreshes from durable state after a matching realtime event and releases resources on stop', async () => {
    let options: Record<string, any> | undefined;
    realtimeMocks.open.mockImplementation((nextOptions: Record<string, any>) => {
      options = nextOptions;
      return realtimeMocks.controller;
    });
    taskApiMocks.getTask.mockResolvedValue({ id: 7, status: 'running' });
    const onTask = vi.fn();

    const observer = observeTask(7, { onTask, pollIntervalMs: 60_000 });
    await flushPromises();
    expect(onTask).toHaveBeenCalledWith({ id: 7, status: 'running' });

    options?.onMessage({ task_id: 7, type: 'task.updated' });
    await flushPromises();
    expect(taskApiMocks.getTask).toHaveBeenCalledTimes(2);

    observer.stop();
    expect(realtimeMocks.controller.close).toHaveBeenCalledOnce();
  });

  it('identifies every persisted terminal task outcome', () => {
    expect(isTerminalTaskStatus('success')).toBe(true);
    expect(isTerminalTaskStatus('failed')).toBe(true);
    expect(isTerminalTaskStatus('cancelled')).toBe(true);
    expect(isTerminalTaskStatus('needs_attention')).toBe(true);
    expect(isTerminalTaskStatus('running')).toBe(false);
  });
});
