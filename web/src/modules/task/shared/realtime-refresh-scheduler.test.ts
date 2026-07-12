import { afterEach, describe, expect, it, vi } from 'vitest';

import { TaskRealtimeRefreshScheduler } from './realtime-refresh-scheduler';

describe('TaskRealtimeRefreshScheduler', () => {
  afterEach(() => vi.useRealTimers());

  it('coalesces a notification burst into one durable refresh', async () => {
    vi.useFakeTimers();
    const onRefresh = vi.fn();
    const scheduler = new TaskRealtimeRefreshScheduler({ intervalMs: 250, onRefresh });

    scheduler.request();
    scheduler.request();
    scheduler.request();
    await vi.advanceTimersByTimeAsync(250);

    expect(onRefresh).toHaveBeenCalledOnce();
  });

  it('schedules one follow-up when notifications arrive during a refresh', async () => {
    vi.useFakeTimers();
    let complete!: () => void;
    const onRefresh = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          complete = resolve;
        }),
    );
    const scheduler = new TaskRealtimeRefreshScheduler({ intervalMs: 250, onRefresh });

    scheduler.request(true);
    scheduler.request();
    scheduler.request();
    complete();
    await Promise.resolve();
    await vi.advanceTimersByTimeAsync(250);

    expect(onRefresh).toHaveBeenCalledTimes(2);
  });

  it('drops a pending refresh after cancellation', async () => {
    vi.useFakeTimers();
    const onRefresh = vi.fn();
    const scheduler = new TaskRealtimeRefreshScheduler({ onRefresh });

    scheduler.request();
    scheduler.cancel();
    await vi.runAllTimersAsync();

    expect(onRefresh).not.toHaveBeenCalled();
  });
});
