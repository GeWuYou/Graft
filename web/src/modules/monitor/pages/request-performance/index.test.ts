import { flushPromises, shallowMount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { nextTick, reactive } from 'vue';

import { resetMonitorRefreshPreferencesForTests } from '../../composables/use-monitor-refresh-preferences';
import RequestPerformancePage from './index.vue';

const requestPerformanceMock = vi.hoisted(() => vi.fn());
const schedulerStoreMock = vi.hoisted(() => ({
  store: null as { allowPolling: boolean } | null,
}));

vi.mock('../../api/request-performance', () => ({
  getRequestPerformance: requestPerformanceMock,
}));

vi.mock('@/store', () => {
  const store = reactive({ allowPolling: true });
  schedulerStoreMock.store = store;
  return {
    useRealtimeSchedulerStore: () => store,
  };
});

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      const translations: Record<string, string> = {
        'app.refreshControl.countdown': '{countdown} Until refresh',
        'app.refreshControl.pending': 'Waiting for next refresh',
      };
      const template = translations[key] ?? key;
      return Object.entries(params ?? {}).reduce(
        (message, [name, value]) => message.replace(`{${name}}`, String(value)),
        template,
      );
    },
  }),
}));

vi.mock('echarts/core', () => ({
  init: vi.fn(),
  use: vi.fn(),
}));

function createResponse() {
  return {
    summary: {
      error_5xx_count: 0,
      error_5xx_rate: 0,
      p50_latency_ms: 1,
      p95_latency_ms: 2,
      requests_per_second: 1,
      slow_request_count: 0,
    },
    minute_buckets: [],
    status_groups: [],
    top_routes: {
      errors_5xx: [],
      p95_latency: [],
      traffic: [],
    },
  };
}

describe('MonitorRequestPerformanceIndex', () => {
  beforeEach(() => {
    resetMonitorRefreshPreferencesForTests();
    requestPerformanceMock.mockReset();
    requestPerformanceMock.mockResolvedValue(createResponse());
    schedulerStoreMock.store!.allowPolling = true;
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('keeps pending visible at the refresh deadline before starting the next countdown', async () => {
    vi.useFakeTimers();
    const wrapper = shallowMount(RequestPerformancePage);
    await flushPromises();
    await nextTick();

    const page = wrapper.vm as unknown as { remainingRefreshSeconds: number | null };
    expect(page.remainingRefreshSeconds).toBe(5);

    let resolveRefresh: ((value: ReturnType<typeof createResponse>) => void) | undefined;
    requestPerformanceMock.mockImplementationOnce(
      () =>
        new Promise<ReturnType<typeof createResponse>>((resolve) => {
          resolveRefresh = resolve;
        }),
    );

    await vi.advanceTimersByTimeAsync(5000);
    await nextTick();
    expect(page.remainingRefreshSeconds).toBe(0);

    resolveRefresh?.(createResponse());
    await flushPromises();
    await nextTick();
    expect(page.remainingRefreshSeconds).toBe(0);

    await vi.advanceTimersByTimeAsync(500);
    await flushPromises();
    await nextTick();
    expect(page.remainingRefreshSeconds).toBe(5);

    wrapper.unmount();
  });
});
