import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { nextTick } from 'vue';

import DependencyHistoryChart from './DependencyHistoryChart.vue';

const chartMocks = vi.hoisted(() => {
  const dispose = vi.fn();
  const init = vi.fn(() => ({ dispose, resize: vi.fn(), setOption: vi.fn() }));
  return { dispose, init };
});

vi.mock('echarts/core', () => ({
  init: chartMocks.init,
  use: vi.fn(),
}));

vi.mock('echarts/charts', () => ({ LineChart: {} }));
vi.mock('echarts/components', () => ({ GridComponent: {}, TooltipComponent: {} }));
vi.mock('echarts/renderers', () => ({ CanvasRenderer: {} }));

vi.mock('@/store', () => ({
  useSettingStore: () => ({
    brandTheme: '#0052d9',
    chartColors: {
      borderColor: '#d9e0ea',
      containerColor: '#ffffff',
      textColor: '#111827',
    },
    displayMode: 'light',
    resolvedThemeTokensForDisplayMode: {},
  }),
}));

const points = [
  {
    availability_percent: 100,
    latency_p95_ms: 12,
    observed_at: '2026-07-14T09:00:00Z',
  },
];

afterEach(() => {
  vi.restoreAllMocks();
});

describe('DependencyHistoryChart', () => {
  it('recreates the chart when a visible state remounts its container', async () => {
    const wrapper = mount(DependencyHistoryChart, {
      props: {
        availabilityLabel: 'Availability',
        latencyLabel: 'Latency',
        message: 'No data',
        points,
        state: 'ready',
      },
    });
    await flushPromises();
    expect(chartMocks.init).toHaveBeenCalledTimes(1);

    await wrapper.setProps({ state: 'empty' });
    await nextTick();
    await flushPromises();
    expect(chartMocks.dispose).toHaveBeenCalledTimes(1);

    await wrapper.setProps({ state: 'ready' });
    await nextTick();
    await flushPromises();
    expect(chartMocks.init).toHaveBeenCalledTimes(2);

    wrapper.unmount();
  });
});
