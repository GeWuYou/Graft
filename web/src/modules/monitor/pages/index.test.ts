import { flushPromises, mount, type VueWrapper } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import MonitorPage from './index.vue';

const monitorApiMocks = vi.hoisted(() => ({
  getServerStatus: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
}));

const chartMocks = vi.hoisted(() => {
  const setOption = vi.fn();
  const resize = vi.fn();
  const dispose = vi.fn();
  const init = vi.fn(() => ({
    setOption,
    resize,
    dispose,
  }));

  return {
    init,
    setOption,
    resize,
    dispose,
  };
});

const settingStoreMock = vi.hoisted(() => ({
  brandTheme: '#0052D9',
  chartColors: {
    textColor: 'rgba(0, 0, 0, 0.9)',
    placeholderColor: 'rgba(0, 0, 0, 0.35)',
    borderColor: '#dcdcdc',
    containerColor: '#ffffff',
  },
}));

const translations = vi.hoisted(
  (): Record<string, string> => ({
    'monitor.sectionTitle': 'Server Management',
    'monitor.serverStatus.heroEyebrow': 'Runtime Monitor',
    'monitor.serverStatus.overviewTitle': 'Server Status Overview',
    'monitor.serverStatus.overviewHint':
      'Review runtime health, dependency state, plugin summary, and short-window trends from a single compact dashboard.',
    'monitor.serverStatus.refresh': 'Refresh',
    'monitor.serverStatus.refreshIntervalLabel': 'Refresh cadence',
    'monitor.serverStatus.refreshStateLabel': 'Refresh state',
    'monitor.serverStatus.trendWindowLabel': 'Trend window',
    'monitor.serverStatus.lastObserved': 'Last observed: {time}',
    'monitor.serverStatus.lastUpdated': 'Last updated: {time}',
    'monitor.serverStatus.observedAtLabel': 'Observed at',
    'monitor.serverStatus.nextRefreshManual': 'Next refresh: manual only',
    'monitor.serverStatus.nextRefreshPaused': 'Next refresh paused while the page is hidden',
    'monitor.serverStatus.nextRefreshPending': 'Next refresh is being scheduled',
    'monitor.serverStatus.nextRefreshIn': 'Next refresh in {seconds}s',
    'monitor.serverStatus.nextRefreshRetryIn': 'Retry in {seconds}s · base interval {interval}',
    'monitor.serverStatus.refreshIntervalManual': 'Manual',
    'monitor.serverStatus.refreshInterval5Seconds': '5 sec',
    'monitor.serverStatus.refreshInterval10Seconds': '10 sec',
    'monitor.serverStatus.refreshInterval30Seconds': '30 sec',
    'monitor.serverStatus.refreshInterval1Minute': '1 min',
    'monitor.serverStatus.refreshInterval5Minutes': '5 min',
    'monitor.serverStatus.loadFailed': 'Failed to load server status',
    'monitor.serverStatus.trendCardTitle': 'Runtime Trend',
    'monitor.serverStatus.trendRange10Minutes': '10 min',
    'monitor.serverStatus.trendRange30Minutes': '30 min',
    'monitor.serverStatus.trendRange1Hour': '1 hour',
    'monitor.serverStatus.dependencyCardTitle': 'Dependency Status',
    'monitor.serverStatus.statusLabel': 'Overall status',
    'monitor.serverStatus.summaryDependencies': 'Healthy dependencies',
    'monitor.serverStatus.summaryDependenciesValue': '{healthy} / {total} healthy',
    'monitor.serverStatus.summaryDependenciesMeta': '{degraded} degraded · {disabled} disabled',
    'monitor.serverStatus.summaryPlugins': 'Plugin status',
    'monitor.serverStatus.summaryPluginsValue': '{total} registered',
    'monitor.serverStatus.summaryPluginsMeta': '{healthy} healthy · {abnormal} abnormal · {unreported} unreported',
    'monitor.serverStatus.summaryPluginsNoMetrics': 'Runtime status has not been reported',
    'monitor.serverStatus.summaryMemory': 'Allocated memory',
    'monitor.serverStatus.summaryMemoryMeta': '{goroutines} goroutines · {gc} GC cycles',
    'monitor.serverStatus.versionLabel': 'Version',
    'monitor.serverStatus.startedAtLabel': 'Started at',
    'monitor.serverStatus.uptimeLabel': 'Uptime',
    'monitor.serverStatus.goroutinesLabel': 'Goroutines',
    'monitor.serverStatus.goroutinesValue': '{count}',
    'monitor.serverStatus.gcLabel': 'GC cycles',
    'monitor.serverStatus.gcValue': '{count}',
    'monitor.serverStatus.goVersionLabel': 'Go Version',
    'monitor.serverStatus.appLabel': 'Application',
    'monitor.serverStatus.hostLabel': 'Host',
    'monitor.serverStatus.platformLabel': 'Platform',
    'monitor.serverStatus.cpuLabel': 'CPU',
    'monitor.serverStatus.cpuValue': '{count} cores',
    'monitor.serverStatus.heapLabel': 'Heap in use',
    'monitor.serverStatus.runtimeGroupBasic': 'Basic',
    'monitor.serverStatus.runtimeGroupRuntime': 'Runtime',
    'monitor.serverStatus.runtimeGroupEnvironment': 'Environment',
    'monitor.serverStatus.databaseLabel': 'Database',
    'monitor.serverStatus.redisLabel': 'Redis',
    'monitor.serverStatus.noDependencies': 'No dependencies',
    'monitor.serverStatus.latencyValue': '{value} ms',
    'monitor.serverStatus.noLatency': 'No latency sample',
    'monitor.serverStatus.chartCpu': 'CPU',
    'monitor.serverStatus.chartGoroutines': 'Goroutines',
    'monitor.serverStatus.chartMemory': 'Memory',
    'monitor.serverStatus.statusHealthy': 'Healthy',
    'monitor.serverStatus.statusDegraded': 'Degraded',
    'monitor.serverStatus.statusAbnormal': 'Abnormal',
    'monitor.serverStatus.statusUnknown': 'Unknown',
    'monitor.serverStatus.statusDisabled': 'Disabled',
    'monitor.serverStatus.empty': 'No server-status data',
    'monitor.serverStatus.emptyTrend': 'At least 2 samples are required before the trend chart is shown',
    'monitor.serverStatus.emptyMetric.overall': 'Overall status',
    'monitor.serverStatus.emptyMetric.dependencies': 'Healthy dependencies',
    'monitor.serverStatus.emptyMetric.plugins': 'Plugin status',
    'monitor.serverStatus.emptyMetric.memory': 'Allocated memory',
    'monitor.serverStatus.emptyMetric.meta': 'Waiting for the first successful snapshot',
  }),
);

vi.mock('../api/server-status', () => ({
  getServerStatus: monitorApiMocks.getServerStatus,
}));

vi.mock('@/store', () => ({
  useSettingStore: () => settingStoreMock,
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) => {
      const template = translations[key] ?? key;
      if (!params) {
        return template;
      }

      return Object.entries(params).reduce((result, [token, value]) => result.replace(`{${token}}`, value), template);
    },
    locale: {
      value: 'en-US',
    },
  }),
}));

vi.mock('tdesign-vue-next', () => ({
  MessagePlugin: {
    error: messageMocks.error,
  },
}));

vi.mock('echarts/core', async () => {
  const actual = await vi.importActual<typeof import('echarts/core')>('echarts/core');
  return {
    ...actual,
    init: chartMocks.init,
    use: vi.fn(),
  };
});

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: {
    title: {
      type: String,
      default: '',
    },
    description: {
      type: String,
      default: '',
    },
    theme: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        {
          'data-theme': props.theme || undefined,
        },
        [props.title, props.description, slots.icon?.(), slots.actions?.(), slots.default?.()],
      );
  },
});

const rowStub = defineComponent({
  name: 'TRowStub',
  setup(_props, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const colStub = defineComponent({
  name: 'TColStub',
  setup(_props, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  props: {
    loading: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'button',
        {
          'data-loading': String(props.loading),
          onClick: (event: MouseEvent) => emit('click', event),
        },
        [slots.icon?.(), slots.default?.()],
      );
  },
});

const selectStub = defineComponent({
  name: 'TSelectStub',
  props: {
    modelValue: {
      type: String,
      default: '',
    },
    options: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h(
        'select',
        {
          value: props.modelValue,
          onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
        },
        (props.options as Array<{ label: string; value: string }>).map((option) =>
          h('option', { value: option.value }, option.label),
        ),
      );
  },
});

const radioGroupStub = defineComponent({
  name: 'TRadioGroupStub',
  props: {
    modelValue: {
      type: String,
      default: '',
    },
  },
  emits: ['update:modelValue'],
  setup(_props, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const radioButtonStub = defineComponent({
  name: 'TRadioButtonStub',
  props: {
    value: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () => h('button', { 'data-range': props.value }, slots.default?.());
  },
});

function createServerStatusResponse() {
  return {
    status: 'healthy',
    observed_at: '2026-05-20T09:00:00Z',
    server: {
      version: 'dev',
      started_at: '2026-05-20T08:00:00Z',
      uptime_seconds: 3661,
      go_version: 'go1.26.0',
      app_name: 'graft',
      app_env: 'local',
    },
    runtime: {
      go_version: 'go1.26.0',
      host_name: 'test-host',
      operating_system: 'linux',
      architecture: 'amd64',
      cpu_cores: 8,
      goroutines: 32,
      alloc_bytes: 104857600,
      heap_in_use_bytes: 52428800,
      system_memory_bytes: 157286400,
      gc_cycles: 12,
    },
    dependencies: {
      database: {
        status: 'healthy',
        detail: 'Database ping succeeded',
        latency_ms: 2.15,
      },
      redis: {
        status: 'disabled',
        detail: 'Redis client is not configured',
        latency_ms: null,
      },
    },
    summary: {
      total_dependencies: 2,
      healthy_dependencies: 1,
      degraded_dependencies: 0,
      unknown_dependencies: 0,
      disabled_dependencies: 1,
      total_plugins: 2,
      healthy_plugins: 0,
    },
    trend: {
      range: '10m',
      retention_seconds: 600,
      sample_interval_seconds: 30,
      points: [
        {
          observed_at: '2026-05-20T08:55:00Z',
          cpu_percent: 14.5,
          goroutines: 28,
          alloc_bytes: 100663296,
          heap_in_use_bytes: 50331648,
          system_memory_bytes: 150994944,
        },
        {
          observed_at: '2026-05-20T09:00:00Z',
          cpu_percent: 21.2,
          goroutines: 32,
          alloc_bytes: 104857600,
          heap_in_use_bytes: 52428800,
          system_memory_bytes: 157286400,
        },
      ],
    },
    plugins: [
      {
        name: 'monitor',
        version: '0.1.0',
        status: 'healthy',
        depends_on: ['user', 'rbac'],
      },
      {
        name: 'user',
        version: '0.2.0',
        status: 'unknown',
        depends_on: [],
      },
    ],
  };
}

const mountedWrappers: VueWrapper[] = [];

function setVisibilityState(state: 'visible' | 'hidden') {
  Object.defineProperty(document, 'visibilityState', {
    configurable: true,
    value: state,
  });
}

function mountMonitorPage() {
  const wrapper = mount(MonitorPage, {
    attachTo: document.body,
    global: {
      stubs: {
        't-button': buttonStub,
        't-card': passthroughStub,
        't-col': colStub,
        't-empty': passthroughStub,
        't-radio-button': radioButtonStub,
        't-radio-group': radioGroupStub,
        't-row': rowStub,
        't-select': selectStub,
        't-tag': passthroughStub,
      },
    },
  });
  mountedWrappers.push(wrapper);
  return wrapper;
}

describe('MonitorPage', () => {
  beforeEach(() => {
    vi.useRealTimers();
    monitorApiMocks.getServerStatus.mockReset();
    messageMocks.error.mockReset();
    chartMocks.init.mockClear();
    chartMocks.setOption.mockClear();
    chartMocks.resize.mockClear();
    chartMocks.dispose.mockClear();
    document.body.innerHTML = '';
    setVisibilityState('visible');
    document.documentElement.style.setProperty('--td-brand-color', '#0052D9');
    document.documentElement.style.setProperty('--td-success-color-5', '#00A870');
    document.documentElement.style.setProperty('--td-warning-color-5', '#ED7B2F');
  });

  afterEach(() => {
    while (mountedWrappers.length > 0) {
      mountedWrappers.pop()?.unmount();
    }
    vi.clearAllMocks();
    vi.useRealTimers();
  });

  it('loads server status on mount and renders the compact summary, dependency and runtime cards', async () => {
    monitorApiMocks.getServerStatus.mockResolvedValue(createServerStatusResponse());

    const wrapper = mountMonitorPage();
    await flushPromises();
    await nextTick();

    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledTimes(1);
    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledWith('10m');
    expect(wrapper.text()).toContain('Server Status Overview');
    expect(wrapper.text()).toContain('Server Management');
    expect(wrapper.text()).toContain('Refresh cadence');
    expect(wrapper.text()).toContain('Trend window');
    expect(wrapper.text()).toContain('Refresh state');
    expect(wrapper.text()).toContain('Last updated:');
    expect(wrapper.text()).not.toContain('Endpoint:');
    expect(wrapper.text()).not.toContain('Runtime Monitor');
    expect(wrapper.text()).toContain('1 / 2 healthy');
    expect(wrapper.text()).toContain('2 registered');
    expect(wrapper.text()).toContain('1 healthy · 0 abnormal · 1 unreported');
    expect(wrapper.text()).toContain('Allocated memory');
    expect(wrapper.text()).toContain('100 MB');
    expect(wrapper.text()).toContain('Database ping succeeded');
    expect(wrapper.text()).toContain('Redis client is not configured');
    expect(wrapper.text()).toContain('2.15 ms');
    expect(wrapper.text()).toContain('No latency sample');
    expect(wrapper.text()).toContain('test-host');
    expect(wrapper.text()).toContain('linux/amd64');
    expect(wrapper.text()).toContain('8 cores');
    expect(wrapper.text()).toContain('32');
    expect(wrapper.text()).toContain('12');
    expect(wrapper.text()).toContain('Observed at');
    expect(wrapper.text()).toContain('10 min');
    expect(wrapper.text()).toContain('30 min');
    expect(wrapper.text()).toContain('1 hour');
    expect(wrapper.text()).toContain('Next refresh in 5s');
    expect(wrapper.text()).not.toContain('Plugin Summary');
    expect(chartMocks.init).toHaveBeenCalledTimes(1);
    expect(chartMocks.setOption).toHaveBeenCalled();
    const option = chartMocks.setOption.mock.calls.at(-1)?.[0] as {
      color: string[];
      yAxis: Array<{
        axisLabel?: {
          formatter?: (value: number) => string;
        };
      }>;
      series: Array<{
        data: number[];
        yAxisIndex?: number;
        areaStyle?: { opacity?: number };
        emphasis?: { focus?: string; areaStyle?: { opacity?: number } };
      }>;
    };
    expect(option.color).toEqual(['#0052D9', '#00A870', '#ED7B2F']);
    expect(option.yAxis[0]?.axisLabel?.formatter?.(32)).toBe('32%');
    expect(option.series).toHaveLength(3);
    expect(option.series[0]?.data).toEqual([14.5, 21.2]);
    expect(option.series[1]?.data).toEqual([96, 100]);
    expect(option.series[2]?.data).toEqual([28, 32]);
    expect(option.series[1]?.yAxisIndex).toBe(1);
    expect(option.series[2]?.yAxisIndex).toBe(1);
    expect(option.series[0]?.areaStyle?.opacity).toBe(0);
    expect(option.series[1]?.areaStyle?.opacity).toBe(0);
    expect(option.series[2]?.areaStyle?.opacity).toBe(0);
    expect(option.series[0]?.emphasis).toEqual({ focus: 'series', areaStyle: { opacity: 0.14 } });
    expect(option.series[1]?.emphasis).toEqual({ focus: 'series', areaStyle: { opacity: 0.14 } });
    expect(option.series[2]?.emphasis).toEqual({ focus: 'series', areaStyle: { opacity: 0.14 } });
  });

  it('does not render the trend chart when fewer than two samples remain after range filtering', async () => {
    const response = createServerStatusResponse();
    response.trend.points = response.trend.points.slice(0, 1);
    monitorApiMocks.getServerStatus.mockResolvedValue(response);

    const wrapper = mountMonitorPage();
    await flushPromises();
    await nextTick();

    expect(wrapper.text()).toContain('At least 2 samples are required before the trend chart is shown');
    expect(chartMocks.init).not.toHaveBeenCalled();
  });

  it('counts down auto refresh, pauses while hidden, and refreshes immediately after becoming visible', async () => {
    vi.useFakeTimers();
    monitorApiMocks.getServerStatus.mockResolvedValue(createServerStatusResponse());

    const wrapper = mountMonitorPage();
    await flushPromises();
    await nextTick();

    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledTimes(1);
    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledWith('10m');
    expect(wrapper.text()).toContain('Next refresh in 5s');

    await vi.advanceTimersByTimeAsync(2000);
    await flushPromises();
    expect(wrapper.text()).toContain('Next refresh in 3s');

    setVisibilityState('hidden');
    document.dispatchEvent(new Event('visibilitychange'));
    await nextTick();
    expect(wrapper.text()).toContain('Next refresh paused while the page is hidden');

    await vi.advanceTimersByTimeAsync(5000);
    await flushPromises();
    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledTimes(1);

    setVisibilityState('visible');
    document.dispatchEvent(new Event('visibilitychange'));
    await flushPromises();
    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledTimes(2);
    expect(monitorApiMocks.getServerStatus).toHaveBeenNthCalledWith(2, '10m');
  });

  it('backs off the retry cadence after a failed auto refresh', async () => {
    vi.useFakeTimers();
    monitorApiMocks.getServerStatus.mockRejectedValue(new Error('Network down'));

    const wrapper = mountMonitorPage();
    await flushPromises();

    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledTimes(1);
    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledWith('10m');
    expect(wrapper.text()).toContain('Retry in 10s · base interval 5 sec');

    await vi.advanceTimersByTimeAsync(9000);
    await flushPromises();
    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(1000);
    await flushPromises();
    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledTimes(2);
    expect(monitorApiMocks.getServerStatus).toHaveBeenNthCalledWith(2, '10m');
  });

  it('falls back to the localized load failure message when the error is empty', async () => {
    monitorApiMocks.getServerStatus.mockRejectedValue(new Error('   '));

    const wrapper = mountMonitorPage();
    await flushPromises();

    expect(messageMocks.error).toHaveBeenCalledWith('Failed to load server status');
    expect(wrapper.text()).toContain('No server-status data');
    expect(chartMocks.init).not.toHaveBeenCalled();
  });

  it('refetches the last selected trend range after an in-flight request finishes', async () => {
    let firstRequestSettled = false;
    let resolveFirstRequest!: (value: ReturnType<typeof createServerStatusResponse>) => void;
    monitorApiMocks.getServerStatus.mockImplementation(
      () =>
        new Promise((resolve) => {
          if (!firstRequestSettled) {
            resolveFirstRequest = resolve;
            firstRequestSettled = true;
            return;
          }

          resolve(createServerStatusResponse());
        }),
    );

    const wrapper = mountMonitorPage();
    await nextTick();

    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledTimes(1);
    expect(monitorApiMocks.getServerStatus).toHaveBeenNthCalledWith(1, '10m');

    await wrapper.find('select').setValue('manual');
    wrapper.findComponent(radioGroupStub).vm.$emit('update:modelValue', '30m');
    await flushPromises();

    resolveFirstRequest(createServerStatusResponse());
    await flushPromises();

    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledTimes(2);
    expect(monitorApiMocks.getServerStatus).toHaveBeenNthCalledWith(2, '30m');
  });
});
