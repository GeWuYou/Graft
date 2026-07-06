import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { useRealtimeSchedulerStore } from '@/store/modules/realtime-scheduler';

import { createRealtimeSnapshotGate } from './snapshot-gate';

describe('realtime snapshot gate', () => {
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

  it('buffers the latest snapshot while frozen and flushes it once on resume', async () => {
    const scheduler = useRealtimeSchedulerStore();
    const applied: number[] = [];
    const gate = createRealtimeSnapshotGate<number>({
      apply(snapshot) {
        applied.push(snapshot);
      },
    });

    const token = scheduler.freeze('shell-sidebar-motion');
    gate.commit(1);
    gate.commit(2);

    expect(applied).toEqual([]);

    scheduler.release(token);
    vi.runAllTimers();
    await Promise.resolve();

    expect(applied).toEqual([2]);

    gate.dispose();
  });
});
