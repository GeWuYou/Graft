import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { useRealtimeSchedulerStore } from './realtime-scheduler';

describe('realtime scheduler store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.useFakeTimers();

    let frameId = 0;
    vi.stubGlobal('window', {
      cancelAnimationFrame: vi.fn(),
      requestAnimationFrame: vi.fn((callback: FrameRequestCallback) => {
        const nextFrameId = ++frameId;
        setTimeout(() => callback(16), 0);
        return nextFrameId;
      }),
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  it('enters freezing immediately and returns to running after the last release settles', () => {
    const store = useRealtimeSchedulerStore();

    const token = store.freeze('shell-sidebar-motion');

    expect(store.phase).toBe('freezing');
    expect(store.allowPolling).toBe(false);

    store.release(token);

    expect(store.phase).toBe('resuming');
    vi.runAllTimers();
    expect(store.phase).toBe('running');
    expect(store.allowSnapshotCommit).toBe(true);
  });

  it('keeps the scheduler frozen until all outstanding tokens are released', () => {
    const store = useRealtimeSchedulerStore();

    const first = store.freeze('shell-sidebar-motion');
    const second = store.freeze('shell-sidebar-motion');

    store.release(first);
    vi.runAllTimers();
    expect(store.phase).toBe('freezing');

    store.release(second);
    expect(store.phase).toBe('resuming');
    vi.runAllTimers();
    expect(store.phase).toBe('running');
  });
});
