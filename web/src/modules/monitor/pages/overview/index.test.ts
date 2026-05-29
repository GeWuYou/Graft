import { flushPromises, mount, type VueWrapper } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import { resetMonitorRefreshPreferencesForTests } from '../../composables/use-monitor-refresh-preferences';
import DependenciesPage from '../dependencies/index.vue';
import RuntimePage from '../runtime/index.vue';
import MonitorPage from './index.vue';

const monitorApiMocks = vi.hoisted(() => ({
  getServerStatus: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
}));

const correlationActionMocks = vi.hoisted(() => ({
  openCorrelationErrorNotification: vi.fn(),
  requestIdFromError: vi.fn(() => ''),
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

const resizeObserverMocks = vi.hoisted(() => {
  const observe = vi.fn();
  const unobserve = vi.fn();
  const disconnect = vi.fn();
  let callback: ResizeObserverCallback | null = null;

  class ResizeObserverMock {
    constructor(nextCallback: ResizeObserverCallback) {
      callback = nextCallback;
    }

    observe = observe;
    unobserve = unobserve;
    disconnect = disconnect;
  }

  return {
    ResizeObserverMock,
    observe,
    unobserve,
    disconnect,
    trigger() {
      callback?.([], {} as ResizeObserver);
    },
  };
});

const settingStoreMock = vi.hoisted(() => ({
  displayMode: 'light',
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
    'monitor.serverStatus.overviewTitle': 'Server Status Overview',
    'monitor.serverStatus.overviewHint':
      'Review overall server resources, dependency health, and the current Go process runtime summary.',
    'monitor.serverStatus.refreshIntervalLabel': 'Refresh cadence',
    'monitor.serverStatus.refreshInterval5Seconds': 'Every 5 sec',
    'monitor.serverStatus.refreshInterval10Seconds': 'Every 10 sec',
    'monitor.serverStatus.refreshInterval30Seconds': 'Every 30 sec',
    'monitor.serverStatus.refreshInterval1Minute': 'Every 1 min',
    'monitor.serverStatus.refreshFixedValue': 'Every 5 sec',
    'monitor.serverStatus.trendWindowLabel': 'Trend window',
    'monitor.serverStatus.refreshNow': 'Refresh now',
    'monitor.serverStatus.pauseRefresh': 'Pause auto refresh',
    'monitor.serverStatus.resumeRefresh': 'Resume auto refresh',
    'monitor.serverStatus.refreshStateLabel': 'Refresh state',
    'monitor.serverStatus.lastObserved': 'Last observed: {time}',
    'monitor.serverStatus.lastUpdated': 'Last updated: {time}',
    'monitor.serverStatus.observedAtLabel': 'Observed at',
    'monitor.serverStatus.nextRefreshPausedByUser': 'Auto refresh paused',
    'monitor.serverStatus.nextRefreshPaused': 'Next refresh paused while the page is hidden',
    'monitor.serverStatus.nextRefreshPending': 'Preparing the next refresh',
    'monitor.serverStatus.nextRefreshIn': 'Next refresh in {seconds}s',
    'monitor.serverStatus.nextRefreshRetryIn': 'Retry in {seconds}s · base interval {interval}',
    'monitor.serverStatus.loadFailed': 'Failed to load server status',
    'monitor.serverStatus.trendCardTitle': 'Resource and Load Trends',
    'monitor.serverStatus.trendRange10Minutes': '10 min',
    'monitor.serverStatus.trendRange30Minutes': '30 min',
    'monitor.serverStatus.trendRange1Hour': '1 hour',
    'monitor.serverStatus.trendModeOverview': 'Overview',
    'monitor.serverStatus.trendModeMulti': 'Small charts',
    'monitor.serverStatus.trendModeFocus': 'Focus metric',
    'monitor.serverStatus.trendMetricInventory': 'Trend metrics',
    'monitor.serverStatus.trendMetricInventoryValue': '{count} metrics grouped as {groups}.',
    'monitor.serverStatus.focusMetricLabel': 'Focus metric',
    'monitor.serverStatus.dependencyCardTitle': 'Dependency Status',
    'monitor.serverStatus.runtimeStatusTitle': 'Runtime status',
    'monitor.serverStatus.runtimeStatusSubtitle': 'Summary of service, dependencies, and sampling status',
    'monitor.serverStatus.runtimeStatusDependenciesTitle': 'Dependencies',
    'monitor.serverStatus.runtimeStatusProcessTitle': 'Process summary',
    'monitor.serverStatus.runtimeStatusSamplingTitle': 'Sampling status',
    'monitor.serverStatus.runtimeStatusUptimeLabel': 'Uptime',
    'monitor.serverStatus.runtimeStatusGoroutinesLabel': 'Goroutines',
    'monitor.serverStatus.runtimeStatusHeapLabel': 'Heap memory',
    'monitor.serverStatus.runtimeStatusRuntimeSysLabel': 'Runtime system memory',
    'monitor.serverStatus.runtimeStatusGcCountLabel': 'GC count',
    'monitor.serverStatus.runtimeStatusLastGcLabel': 'Last GC',
    'monitor.serverStatus.runtimeStatusLastUpdatedLabel': 'Last updated',
    'monitor.serverStatus.runtimeStatusAutoRefreshLabel': 'Auto refresh',
    'monitor.serverStatus.runtimeStatusTimeRangeLabel': 'Time range',
    'monitor.serverStatus.runtimeStatusSamplesLabel': 'Samples',
    'monitor.serverStatus.runtimeStatusTrendModeLabel': 'Trend mode',
    'monitor.serverStatus.runtimeStatusPaused': 'Paused',
    'monitor.serverStatus.runtimeStatusNotAvailable': 'N/A',
    'monitor.serverStatus.runtimeStatusRefreshValue': '5 sec',
    'monitor.serverStatus.trendGroupResourceUsage': 'Resource Usage',
    'monitor.serverStatus.trendGroupSystemLoad': 'System Load',
    'monitor.serverStatus.trendGroupGoRuntime': 'Go Runtime',
    'monitor.serverStatus.trendGroupResourceUsageInfo':
      'CPU and server memory share a 0-100% scale for quick pressure checks.',
    'monitor.serverStatus.trendGroupSystemLoadInfo':
      'Load is separated from percentage-based resources to avoid mixed semantics on one axis.',
    'monitor.serverStatus.runtimeSummaryTitle': 'Runtime Summary',
    'monitor.serverStatus.infoActionLabel': 'Info',
    'monitor.serverStatus.currentValue': 'Current value',
    'monitor.serverStatus.unitLabel': 'Unit',
    'monitor.serverStatus.referenceCoreCountValue': 'Reference: {count} cores',
    'monitor.serverStatus.referenceCoreCountMark': 'Ref {count} cores',
    'monitor.serverStatus.metricLoadLabel': 'Load',
    'monitor.serverStatus.metricLoadValueSide': '1 min avg',
    'monitor.serverStatus.metricLoadMeta': '5m {five} · 15m {fifteen}',
    'monitor.serverStatus.metricLoadStatusHealthy': 'Healthy',
    'monitor.serverStatus.metricLoadStatusWarning': 'Elevated',
    'monitor.serverStatus.metricLoadStatusCritical': 'High',
    'monitor.serverStatus.metricLoadDescriptionHealthy': 'System load is low',
    'monitor.serverStatus.metricLoadDescriptionWarning': 'System load is elevated',
    'monitor.serverStatus.metricLoadDescriptionCritical': 'System load is high',
    'monitor.serverStatus.metricCpuLabel': 'CPU',
    'monitor.serverStatus.metricCpuValue': '{count} cores',
    'monitor.serverStatus.metricCpuMeta': '{count} cores · latest sample',
    'monitor.serverStatus.metricCpuStatusHealthy': 'Idle',
    'monitor.serverStatus.metricCpuStatusWarning': 'Normal',
    'monitor.serverStatus.metricCpuStatusCritical': 'Busy',
    'monitor.serverStatus.metricCpuDescriptionHealthy': 'CPU usage is very low',
    'monitor.serverStatus.metricCpuDescriptionWarning': 'CPU usage is within the normal range',
    'monitor.serverStatus.metricCpuDescriptionCritical': 'CPU usage is high',
    'monitor.serverStatus.metricMemoryLabel': 'Memory',
    'monitor.serverStatus.metricMemoryValue': '{used} / {total}',
    'monitor.serverStatus.metricMemoryMeta': 'Available {available}',
    'monitor.serverStatus.metricMemoryStatusHealthy': 'Sufficient',
    'monitor.serverStatus.metricMemoryStatusWarning': 'Normal',
    'monitor.serverStatus.metricMemoryStatusCritical': 'Elevated',
    'monitor.serverStatus.metricMemoryDescriptionHealthy': 'Server memory is sufficient',
    'monitor.serverStatus.metricMemoryDescriptionWarning': 'Memory usage is normal',
    'monitor.serverStatus.metricMemoryDescriptionCritical': 'Memory pressure is elevated',
    'monitor.serverStatus.metricDiskLabel': 'Disk',
    'monitor.serverStatus.metricDiskValue': '{used} / {total}',
    'monitor.serverStatus.metricDiskMeta': 'Mount {path} · {free} free',
    'monitor.serverStatus.metricDiskStatusHealthy': 'Sufficient',
    'monitor.serverStatus.metricDiskStatusWarning': 'Normal',
    'monitor.serverStatus.metricDiskStatusCritical': 'Low',
    'monitor.serverStatus.metricDiskDescriptionHealthy': 'Disk capacity is sufficient',
    'monitor.serverStatus.metricDiskDescriptionWarning': 'Disk usage is normal',
    'monitor.serverStatus.metricDiskDescriptionCritical': 'Disk capacity is low',
    'monitor.serverStatus.diskRootLabel': 'Root partition',
    'monitor.serverStatus.diskRootPath': '/',
    'monitor.serverStatus.postgresqlLabel': 'PostgreSQL',
    'monitor.serverStatus.statusLabel': 'Overall status',
    'monitor.serverStatus.summaryDependencies': 'Healthy dependencies',
    'monitor.serverStatus.summaryDependenciesValue': '{healthy} / {total} healthy',
    'monitor.serverStatus.summaryDependenciesDetail': 'Dependency exceptions',
    'monitor.serverStatus.summaryDependenciesMeta': '{degraded} degraded · {disabled} disabled',
    'monitor.serverStatus.versionLabel': 'Version',
    'monitor.serverStatus.startedAtLabel': 'Started at',
    'monitor.serverStatus.uptimeLabel': 'Uptime',
    'monitor.serverStatus.goroutinesLabel': 'Goroutines',
    'monitor.serverStatus.goroutinesValue': '{count}',
    'monitor.serverStatus.gcLabel': 'GC cycles',
    'monitor.serverStatus.gcValue': '{count}',
    'monitor.serverStatus.goVersionLabel': 'Go version',
    'monitor.serverStatus.appLabel': 'Application',
    'monitor.serverStatus.envLabel': 'Environment',
    'monitor.serverStatus.hostLabel': 'Host',
    'monitor.serverStatus.platformLabel': 'Platform',
    'monitor.serverStatus.cpuLabel': 'CPU',
    'monitor.serverStatus.cpuValue': '{count} cores',
    'monitor.serverStatus.heapLabel': 'Heap in use',
    'monitor.serverStatus.hostMemoryLabel': 'Server memory',
    'monitor.serverStatus.hostMemoryValue': '{used} / {total}',
    'monitor.serverStatus.runtimeAllocLabel': 'Runtime alloc',
    'monitor.serverStatus.runtimeSysLabel': 'Runtime sys',
    'monitor.serverStatus.runtimeGroupRuntime': 'Runtime Status',
    'monitor.serverStatus.runtimeGroupProcess': 'Go Process',
    'monitor.serverStatus.runtimeGroupEnvironment': 'Environment and Capacity',
    'monitor.serverStatus.runtimeGroupPlugins': 'Plugin Summary',
    'monitor.serverStatus.databaseLabel': 'Database',
    'monitor.serverStatus.redisLabel': 'Redis',
    'monitor.serverStatus.pluginRegistered': 'Registered',
    'monitor.serverStatus.pluginHealthy': 'Healthy',
    'monitor.serverStatus.pluginAbnormal': 'Abnormal',
    'monitor.serverStatus.pluginName': 'Plugins',
    'monitor.serverStatus.noLatency': 'No latency sample',
    'monitor.serverStatus.latencyValue': '{value} ms',
    'monitor.serverStatus.chartCpu': 'CPU usage',
    'monitor.serverStatus.chartCpuShort': 'CPU usage',
    'monitor.serverStatus.chartCpuDescription': 'Track processor utilization over time.',
    'monitor.serverStatus.chartHostMemory': 'Server memory used',
    'monitor.serverStatus.chartHostMemoryShort': 'Server memory',
    'monitor.serverStatus.chartHostMemoryDescription': 'Track server memory utilization over time.',
    'monitor.serverStatus.chartLoad': '1m load average',
    'monitor.serverStatus.chartLoadShort': '1m load',
    'monitor.serverStatus.chartLoadDescription': 'Compare system load to available CPU cores.',
    'monitor.serverStatus.chartRuntimeAlloc': 'Runtime alloc',
    'monitor.serverStatus.chartRuntimeAllocShort': 'Runtime Alloc',
    'monitor.serverStatus.chartRuntimeAllocDescription': 'Current memory allocated by the Go runtime.',
    'monitor.serverStatus.chartRuntimeHeap': 'Runtime heap in use',
    'monitor.serverStatus.chartRuntimeHeapShort': 'Heap',
    'monitor.serverStatus.chartRuntimeHeapDescription': 'Go heap usage over time.',
    'monitor.serverStatus.chartRuntimeSys': 'Runtime sys',
    'monitor.serverStatus.chartRuntimeSysShort': 'Runtime Sys',
    'monitor.serverStatus.chartRuntimeSysDescription': 'Total memory requested from the system by the Go runtime.',
    'monitor.serverStatus.chartGoroutines': 'Goroutines',
    'monitor.serverStatus.chartGoroutinesShort': 'Goroutines',
    'monitor.serverStatus.chartGoroutinesDescription': 'Observe goroutine count changes.',
    'monitor.serverStatus.chartLoadAxis': 'Load',
    'monitor.serverStatus.statusHealthy': 'Healthy',
    'monitor.serverStatus.statusDegraded': 'Degraded',
    'monitor.serverStatus.statusUnknown': 'Unknown',
    'monitor.serverStatus.statusDisabled': 'Disabled',
    'monitor.serverStatus.empty': 'No server-status data',
    'monitor.serverStatus.emptyTrend': 'At least 2 samples are required before the trend chart is shown',
    'monitor.serverStatus.emptyMetric.meta': 'Waiting for the first successful snapshot',
    'monitor.serverStatus.emptyMetric.description': 'Waiting for status data',
    'monitor.serverStatus.diskDetailTitle': 'Disk Detail',
    'monitor.serverStatus.diskPathLabel': 'Mount',
    'monitor.serverStatus.diskTotalLabel': 'Total',
    'monitor.serverStatus.diskUsedLabel': 'Used',
    'monitor.serverStatus.diskFreeLabel': 'Free',
    'monitor.serverStatus.diskPercentLabel': 'Used percent',
  }),
);

vi.mock('../../api/server-status', () => ({
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

vi.mock('tdesign-vue-next', async () => {
  const actual = await vi.importActual<typeof import('tdesign-vue-next')>('tdesign-vue-next');
  return {
    ...actual,
    MessagePlugin: {
      error: messageMocks.error,
    },
  };
});

vi.mock('@/modules/audit/shared/correlation-actions', () => ({
  openCorrelationErrorNotification: correlationActionMocks.openCorrelationErrorNotification,
  requestIdFromError: correlationActionMocks.requestIdFromError,
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
      type: [Number, String],
      default: '',
    },
    options: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['update:modelValue'],
  setup(props, { attrs, emit }) {
    return () =>
      h(
        'select',
        {
          ...attrs,
          value: String(props.modelValue),
          onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
        },
        (props.options as Array<{ label: string; value: number | string }>).map((option) =>
          h('option', { value: String(option.value) }, option.label),
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

const popupStub = defineComponent({
  name: 'TPopupStub',
  setup(_props, { slots }) {
    return () => h('div', [slots.default?.(), slots.content?.()]);
  },
});

const tableStub = defineComponent({
  name: 'TTableStub',
  props: {
    data: {
      type: Array,
      default: () => [],
    },
    columns: {
      type: Array,
      default: () => [],
    },
  },
  setup(props) {
    return () =>
      h(
        'div',
        {
          'data-table-columns': JSON.stringify(props.columns),
          'data-table-rows': JSON.stringify(props.data),
        },
        [],
      );
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
      load_average: {
        one_minute: 0.42,
        five_minutes: 0.35,
        fifteen_minutes: 0.29,
      },
      disk_usage: {
        path: '/',
        total_bytes: 64317135257,
        used_bytes: 12670153523,
        free_bytes: 51646981734,
        used_percent: 18.99,
      },
      host_memory_total_bytes: 34359738368,
      host_memory_used_bytes: 16234976358,
      host_memory_free_bytes: 18124762010,
      host_memory_used_percent: 47.25,
      goroutines: 32,
      runtime_alloc_bytes: 104857600,
      runtime_heap_in_use_bytes: 52428800,
      runtime_sys_bytes: 157286400,
      runtime_gc_cycles: 12,
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
      healthy_plugins: 1,
    },
    trend: {
      range: '10m',
      retention_seconds: 600,
      sample_interval_seconds: 5,
      points: [
        {
          observed_at: '2026-05-20T08:55:00Z',
          cpu_percent: 14.5,
          host_memory_used_percent: 46.71,
          load_average_one_minute: 0.39,
          load_average_five_minutes: 0.32,
          load_average_fifteen_minutes: 0.28,
          goroutines: 28,
          runtime_alloc_bytes: 100663296,
          runtime_heap_in_use_bytes: 50331648,
          runtime_sys_bytes: 150994944,
        },
        {
          observed_at: '2026-05-20T09:00:00Z',
          cpu_percent: 21.2,
          host_memory_used_percent: 47.25,
          load_average_one_minute: 0.42,
          load_average_five_minutes: 0.31,
          load_average_fifteen_minutes: 0.29,
          goroutines: 32,
          runtime_alloc_bytes: 104857600,
          runtime_heap_in_use_bytes: 52428800,
          runtime_sys_bytes: 157286400,
        },
      ],
    },
    plugins: [
      {
        name: 'monitor',
        version: '0.1.0',
        status: 'healthy',
        status_detail: 'Runtime metadata is present and platform signals are healthy',
        depends_on: ['user', 'rbac'],
        missing_dependencies: [],
      },
      {
        name: 'user',
        version: '0.2.0',
        status: 'degraded',
        status_detail: 'Missing runtime dependencies: audit',
        depends_on: [],
        missing_dependencies: ['audit'],
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
  const wrapper = mountWithGlobalStubs(MonitorPage, { attachTo: document.body });

  mountedWrappers.push(wrapper);
  return wrapper;
}

function mountWithGlobalStubs(component: object, options: { attachTo?: Element } = {}) {
  return mount(component, {
    attachTo: options.attachTo,
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
        't-popup': popupStub,
        't-table': tableStub,
        't-tag': passthroughStub,
      },
    },
  });
}

function mountRuntimePage() {
  const wrapper = mountWithGlobalStubs(RuntimePage);
  mountedWrappers.push(wrapper);
  return wrapper;
}

function mountDependenciesPage() {
  const wrapper = mountWithGlobalStubs(DependenciesPage);

  mountedWrappers.push(wrapper);
  return wrapper;
}

function metricCardText(wrapper: VueWrapper, key: string) {
  return wrapper.find(`[data-card-key="${key}"]`).text();
}

function sidebarGroupText(wrapper: VueWrapper, key: string) {
  return wrapper.find(`[data-status-sidebar-group="${key}"]`).text();
}

function getLatestChartOption<T = unknown>() {
  return chartMocks.setOption.mock.calls.at(-1)?.[0] as T;
}

describe('MonitorPage', () => {
  beforeEach(() => {
    vi.useRealTimers();
    vi.stubGlobal('ResizeObserver', resizeObserverMocks.ResizeObserverMock);
    monitorApiMocks.getServerStatus.mockReset();
    messageMocks.error.mockReset();
    correlationActionMocks.openCorrelationErrorNotification.mockReset();
    correlationActionMocks.requestIdFromError.mockReset();
    correlationActionMocks.requestIdFromError.mockReturnValue('');
    chartMocks.init.mockClear();
    chartMocks.setOption.mockClear();
    chartMocks.resize.mockClear();
    chartMocks.dispose.mockClear();
    resizeObserverMocks.observe.mockClear();
    resizeObserverMocks.unobserve.mockClear();
    resizeObserverMocks.disconnect.mockClear();
    document.body.innerHTML = '';
    setVisibilityState('visible');
    document.documentElement.style.setProperty('--td-brand-color', '#0052D9');
    document.documentElement.style.setProperty('--td-brand-color-7', '#003cab');
    document.documentElement.style.setProperty('--td-success-color-5', '#00A870');
    document.documentElement.style.setProperty('--td-success-color-6', '#078d5c');
    document.documentElement.style.setProperty('--td-warning-color-5', '#ED7B2F');
    document.documentElement.style.setProperty('--td-warning-color-6', '#d96c1f');
    document.documentElement.style.setProperty('--td-text-color-brand', '#4f46e5');
  });

  afterEach(() => {
    while (mountedWrappers.length > 0) {
      mountedWrappers.pop()?.unmount();
    }
    resetMonitorRefreshPreferencesForTests();
    vi.clearAllMocks();
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });

  it('loads server status and renders the unified overview shell with trend and runtime sidebar', async () => {
    monitorApiMocks.getServerStatus.mockResolvedValue(createServerStatusResponse());

    const wrapper = mountMonitorPage();
    await flushPromises();
    await nextTick();

    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledTimes(1);
    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledWith('10m');
    expect(wrapper.text()).toContain('Server Status Overview');
    expect(wrapper.text()).toContain('Refresh cadence');
    expect(wrapper.text()).toContain('Every 5 sec');
    expect(wrapper.text()).toContain('Trend window');
    expect(wrapper.text()).toContain('Refresh now');
    expect(wrapper.text()).toContain('Pause auto refresh');
    expect(wrapper.text()).toContain('Load');
    expect(wrapper.text()).toContain('CPU');
    expect(wrapper.text()).toContain('Memory');
    expect(wrapper.text()).toContain('Disk');
    expect(wrapper.text()).toContain('0.42');
    expect(wrapper.text()).toContain('21%');
    expect(wrapper.text()).toContain('47%');
    expect(wrapper.text()).toContain('15.1 GB / 32.0 GB');
    expect(wrapper.text()).not.toContain('RAM');
    expect(wrapper.text()).toContain('Runtime Summary');
    expect(wrapper.text()).toContain('Runtime status');
    expect(wrapper.text()).not.toContain('Dependency Status');
    expect(wrapper.findAll('.server-status-summary-card')).toHaveLength(4);
    expect(wrapper.findAll('.server-status-overview-layout__trend')).toHaveLength(1);
    expect(wrapper.findAll('.server-status-overview-layout__status')).toHaveLength(1);
    expect(wrapper.find('.metric-card__ring').exists()).toBe(false);
    expect(chartMocks.init).toHaveBeenCalledTimes(2);

    const overviewChartOptions = chartMocks.setOption.mock.calls.map((call) => call[0]) as Array<{
      color?: string[];
    }>;
    expect(overviewChartOptions.some((option) => option.color?.includes('#0052D9'))).toBe(true);
    expect(overviewChartOptions.some((option) => option.color?.includes('#00A870'))).toBe(true);

    const loadCardText = metricCardText(wrapper, 'load');
    const cpuCardText = metricCardText(wrapper, 'cpu');
    const memoryCardText = metricCardText(wrapper, 'memory');
    const diskCardText = metricCardText(wrapper, 'disk');

    expect(loadCardText).toContain('Healthy');
    expect(loadCardText).toContain('1 min avg');
    expect(loadCardText).toContain('5m 0.35 · 15m 0.29');
    expect(loadCardText).toContain('System load is low');

    expect(cpuCardText).toContain('Normal');
    expect(cpuCardText).toContain('21%');
    expect(cpuCardText).toContain('8 cores · latest sample');
    expect(cpuCardText).toContain('CPU usage is within the normal range');
    expect(cpuCardText).not.toContain('1m load');

    expect(memoryCardText).toContain('Sufficient');
    expect(memoryCardText).toContain('47%');
    expect(memoryCardText).toContain('15.1 GB / 32.0 GB');
    expect(memoryCardText).toContain('Available 16.9 GB');
    expect(memoryCardText).toContain('Server memory is sufficient');
    expect(memoryCardText).not.toContain('Runtime sys');
    expect(memoryCardText).not.toContain('GC cycles');

    expect(diskCardText).toContain('Sufficient');
    expect(diskCardText).toContain('19%');
    expect(diskCardText).toContain('11.8 GB / 59.9 GB');
    expect(diskCardText).toContain('Mount / · 48.1 GB free');
    expect(diskCardText).toContain('Disk capacity is sufficient');
    expect(diskCardText).not.toContain('Root partition ·');
    expect(diskCardText).not.toContain('/ 11.8 GB / 59.9 GB');

    expect(wrapper.findAll('[data-status-sidebar-group]')).toHaveLength(3);
    expect(sidebarGroupText(wrapper, 'dependencies')).toContain('Dependencies');
    expect(sidebarGroupText(wrapper, 'dependencies')).toContain('PostgreSQL');
    expect(sidebarGroupText(wrapper, 'dependencies')).toContain('Healthy');
    expect(sidebarGroupText(wrapper, 'dependencies')).toContain('2.15 ms');
    expect(sidebarGroupText(wrapper, 'dependencies')).toContain('Redis');
    expect(sidebarGroupText(wrapper, 'dependencies')).toContain('Disabled');
    expect(sidebarGroupText(wrapper, 'dependencies')).toContain('No latency sample');

    expect(sidebarGroupText(wrapper, 'process')).toContain('Process summary');
    expect(sidebarGroupText(wrapper, 'process')).toContain('Uptime');
    expect(sidebarGroupText(wrapper, 'process')).toContain('1h 1m 1s');
    expect(sidebarGroupText(wrapper, 'process')).toContain('Goroutines');
    expect(sidebarGroupText(wrapper, 'process')).toContain('32');
    expect(sidebarGroupText(wrapper, 'process')).toContain('Heap memory');
    expect(sidebarGroupText(wrapper, 'process')).toContain('50 MB');
    expect(sidebarGroupText(wrapper, 'process')).toContain('Runtime system memory');
    expect(sidebarGroupText(wrapper, 'process')).toContain('150 MB');
    expect(sidebarGroupText(wrapper, 'process')).toContain('GC count');
    expect(sidebarGroupText(wrapper, 'process')).toContain('12');
    expect(sidebarGroupText(wrapper, 'process')).toContain('Last GC');
    expect(sidebarGroupText(wrapper, 'process')).toContain('N/A');

    expect(sidebarGroupText(wrapper, 'sampling')).toContain('Sampling status');
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('Last updated');
    expect(sidebarGroupText(wrapper, 'sampling')).not.toContain('N/A');
    expect(sidebarGroupText(wrapper, 'sampling')).toMatch(/\d{1,2}:\d{2}:\d{2}/);
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('Auto refresh');
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('5 sec');
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('Time range');
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('10 min');
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('Samples');
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('2');
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('Trend mode');
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('Overview');

    const option = getLatestChartOption<{
      series: Array<{ name: string; yAxisIndex?: number; data: number[] }>;
      yAxis: Array<{ name?: string; axisLabel?: { formatter?: (value: number) => string } }>;
      tooltip?: {
        formatter?: (
          params: Array<{ axisValueLabel: string; seriesName: string; color: string; data: number }>,
        ) => string;
      };
    }>();

    expect(chartMocks.init).toHaveBeenCalledTimes(2);
    expect(wrapper.findAll('[data-trend-overview-section]')).toHaveLength(3);
    expect(wrapper.find('[data-trend-overview-section="resourceUsage"]').text()).toContain('Resource Usage');
    expect(wrapper.find('[data-trend-overview-section="systemLoad"]').text()).toContain('System Load');
    expect(wrapper.find('[data-trend-overview-section="runtimeSummary"]').text()).toContain('Runtime Summary');
    expect(wrapper.find('[data-trend-overview-section="runtimeSummary"]').text()).not.toContain('Reference: 8 cores');
    expect(wrapper.find('[data-trend-legend-group="resourceUsage"]').text()).toContain('CPU usage');
    expect(wrapper.find('[data-trend-legend-group="resourceUsage"]').text()).toContain('Server memory');
    expect(wrapper.find('[data-trend-legend-group="systemLoad"]').text()).toContain('1m load');
    expect(wrapper.find('[data-trend-overview-section="systemLoad"]').text()).toContain('Reference: 8 cores');

    expect(option.series).toHaveLength(1);
    expect(option.series[0]?.name).toBe('1m load average');
    expect(option.series[0]?.data).toEqual([0.39, 0.42]);
    expect(option.yAxis[0]?.name).toBe('Load');
    expect(option.yAxis[0]?.axisLabel?.formatter?.(0.42)).toBe('0.4');
    expect(
      option.tooltip?.formatter?.([
        { axisValueLabel: '09:00', seriesName: '1m load average', color: '#D97706', data: 0.42 },
      ]),
    ).toContain('1m load average');

    const overviewOptions = chartMocks.setOption.mock.calls.map((call) => call[0]) as Array<{
      series?: Array<{ name: string }>;
      yAxis?: Array<{ name?: string }>;
    }>;

    const usageOption = overviewOptions.find((item) => item.series?.some((series) => series.name === 'CPU usage'));
    expect(usageOption?.series?.map((series) => series.name)).toEqual(['CPU usage', 'Server memory used']);
    expect(usageOption?.yAxis?.[0]?.name).toBe('%');

    const loadOption = overviewOptions.find((item) => item.series?.some((series) => series.name === '1m load average'));
    expect(loadOption?.series).toHaveLength(1);
    expect(loadOption?.yAxis?.[0]?.name).toBe('Load');

    const legendItems = wrapper.findAll('[data-trend-legend-item="true"]');
    expect(legendItems.length).toBeGreaterThan(0);
    expect(legendItems.some((item) => item.text().includes('CPU usage'))).toBe(true);
    expect(legendItems.some((item) => item.text().includes('Server memory'))).toBe(true);

    const allOverviewText = wrapper.text();
    expect(allOverviewText).toContain('7 metrics grouped as Resource Usage / System Load / Go Runtime.');
    expect(allOverviewText).toContain('Runtime Alloc');
    expect(allOverviewText).toContain('Heap');
    expect(allOverviewText).toContain('Runtime Sys');
    expect(allOverviewText).toContain('Goroutines');
  });

  it('renders no chart when fewer than two samples are available', async () => {
    const response = createServerStatusResponse();
    response.trend.points = response.trend.points.slice(0, 1);
    monitorApiMocks.getServerStatus.mockResolvedValue(response);

    const wrapper = mountMonitorPage();
    await flushPromises();
    await nextTick();

    expect(wrapper.text()).toContain('At least 2 samples are required before the trend chart is shown');
    expect(chartMocks.init).not.toHaveBeenCalled();
  });

  it('supports focus and multi trend modes with grouped metrics and dedicated legends', async () => {
    monitorApiMocks.getServerStatus.mockResolvedValue(createServerStatusResponse());

    const wrapper = mountMonitorPage();
    await flushPromises();
    await nextTick();

    wrapper.findAllComponents(radioGroupStub)[0]?.vm.$emit('update:modelValue', 'focus');
    await nextTick();
    const focusOptions = wrapper.findAll('[data-trend-focus-select="true"] option');
    expect(focusOptions).toHaveLength(7);
    expect(focusOptions.map((option) => option.text())).toEqual([
      'Resource Usage / CPU usage',
      'Resource Usage / Server memory used',
      'System Load / 1m load average',
      'Go Runtime / Runtime alloc',
      'Go Runtime / Runtime heap in use',
      'Go Runtime / Runtime sys',
      'Go Runtime / Goroutines',
    ]);

    await wrapper.find('[data-trend-focus-select="true"]').setValue('runtimeSys');
    await nextTick();

    const option = getLatestChartOption<{
      series: Array<{ name: string; areaStyle?: { opacity?: number }; data: number[] }>;
      yAxis: Array<{ name?: string; axisLabel?: { formatter?: (value: number) => string } }>;
      tooltip?: {
        formatter?: (
          params: Array<{ axisValueLabel: string; seriesName: string; color: string; data: number }>,
        ) => string;
      };
    }>();

    expect(option.series).toHaveLength(1);
    expect(option.series[0]?.name).toBe('Runtime sys');
    expect(option.series[0]?.data).toEqual([144, 150]);
    expect(option.series[0]?.areaStyle?.opacity).toBe(0.14);
    expect(option.yAxis[0]?.axisLabel?.formatter?.(150)).toBe('150 MB');
    expect(
      option.tooltip?.formatter?.([
        { axisValueLabel: '09:00', seriesName: 'Runtime sys', color: '#A56A2A', data: 150 },
      ]),
    ).toContain('150.0 MB');
    expect(wrapper.find('[data-trend-mode-panel="focus"]').text()).toContain('Runtime sys');
    expect(wrapper.find('[data-trend-legend-group="focus"]').text()).toContain('Runtime sys');
    expect(wrapper.find('[data-trend-mode-panel="focus"]').text()).toContain('Unit MB');
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('Trend mode');
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('Focus metric');

    wrapper.findAllComponents(radioGroupStub)[0]?.vm.$emit('update:modelValue', 'multi');
    await nextTick();
    await flushPromises();

    const cards = wrapper.findAll('[data-trend-small-card]');
    expect(cards).toHaveLength(7);
    expect(cards.map((card) => card.attributes('data-trend-small-card'))).toEqual([
      'cpu',
      'hostMemory',
      'load',
      'runtimeAlloc',
      'runtimeHeap',
      'runtimeSys',
      'goroutines',
    ]);
    expect(cards[0]?.text()).toContain('CPU usage');
    expect(cards[1]?.text()).toContain('Server memory used');
    expect(cards[2]?.text()).toContain('Reference: 8 cores');
    expect(cards[3]?.text()).toContain('Runtime alloc');
    expect(cards[6]?.text()).toContain('Goroutines');

    const multiOptions = chartMocks.setOption.mock.calls.slice(-7).map((call) => call[0]) as Array<{
      series: Array<{ name: string; data: number[] }>;
      yAxis: Array<{ name?: string; axisLabel?: { formatter?: (value: number) => string } }>;
      tooltip?: {
        formatter?: (
          params: Array<{ axisValueLabel: string; seriesName: string; color: string; data: number }>,
        ) => string;
      };
    }>;

    expect(multiOptions).toHaveLength(7);
    expect(multiOptions[0]?.series[0]?.name).toBe('CPU usage');
    expect(multiOptions[0]?.yAxis[0]?.name).toBe('%');
    expect(multiOptions[0]?.yAxis[0]?.axisLabel?.formatter?.(21.2)).toBe('21.2%');

    expect(multiOptions[2]?.series[0]?.name).toBe('1m load average');
    expect(multiOptions[2]?.yAxis[0]?.name).toBe('load');
    expect(
      multiOptions[2]?.tooltip?.formatter?.([
        { axisValueLabel: '09:00', seriesName: '1m load average', color: '#D97706', data: 0.42 },
      ]),
    ).toContain('0.42');

    expect(multiOptions[3]?.series[0]?.name).toBe('Runtime alloc');
    expect(multiOptions[3]?.yAxis[0]?.axisLabel?.formatter?.(100)).toBe('100 MB');
    expect(multiOptions[6]?.series[0]?.name).toBe('Goroutines');
    expect(multiOptions[6]?.yAxis[0]?.axisLabel?.formatter?.(32)).toBe('32');
  });

  it('keeps the selected trend mode when the range changes', async () => {
    monitorApiMocks.getServerStatus.mockResolvedValue(createServerStatusResponse());

    const wrapper = mountMonitorPage();
    await flushPromises();
    await nextTick();

    wrapper.findAllComponents(radioGroupStub)[0]?.vm.$emit('update:modelValue', 'multi');
    await nextTick();

    await wrapper.find('[data-monitor-refresh-extra-select="true"]').setValue('30m');
    await flushPromises();
    await nextTick();

    expect(wrapper.find('[data-trend-mode-panel="multi"]').exists()).toBe(true);
    expect(wrapper.findAll('[data-trend-small-card]')).toHaveLength(7);
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('Time range');
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('30 min');
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('Trend mode');
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('Small charts');
  });

  it('resizes trend charts when the container observer reports a layout change', async () => {
    monitorApiMocks.getServerStatus.mockResolvedValue(createServerStatusResponse());

    mountMonitorPage();
    await flushPromises();
    await nextTick();

    expect(resizeObserverMocks.observe).toHaveBeenCalled();

    chartMocks.resize.mockClear();
    resizeObserverMocks.trigger();

    expect(chartMocks.resize).toHaveBeenCalled();
  });

  it('counts down auto refresh, pauses while hidden or paused by the user, and resumes immediately', async () => {
    vi.useFakeTimers();
    monitorApiMocks.getServerStatus.mockResolvedValue(createServerStatusResponse());

    const wrapper = mountMonitorPage();
    await flushPromises();
    await nextTick();

    expect(wrapper.text()).toContain('Next refresh in 5s');

    await vi.advanceTimersByTimeAsync(2000);
    await flushPromises();
    expect(wrapper.text()).toContain('Next refresh in 3s');

    const buttons = wrapper.findAll('button');
    await buttons[1]?.trigger('click');
    await nextTick();
    expect(wrapper.text()).toContain('Auto refresh paused');
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('Auto refresh');
    expect(sidebarGroupText(wrapper, 'sampling')).toContain('Paused');

    await vi.advanceTimersByTimeAsync(5000);
    await flushPromises();
    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledTimes(1);

    await buttons[1]?.trigger('click');
    await flushPromises();
    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledTimes(2);
    expect(wrapper.text()).toContain('Pause auto refresh');

    setVisibilityState('hidden');
    document.dispatchEvent(new Event('visibilitychange'));
    await nextTick();
    expect(wrapper.text()).toContain('Next refresh paused while the page is hidden');
  });

  it('backs off the retry cadence after a failed auto refresh', async () => {
    vi.useFakeTimers();
    monitorApiMocks.getServerStatus.mockRejectedValue(new Error('Network down'));

    const wrapper = mountMonitorPage();
    await flushPromises();

    expect(wrapper.text()).toContain('Retry in 10s · base interval Every 5 sec');

    await vi.advanceTimersByTimeAsync(9000);
    await flushPromises();
    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(1000);
    await flushPromises();
    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledTimes(2);
  });

  it('falls back to the localized load failure message when the error is empty', async () => {
    monitorApiMocks.getServerStatus.mockRejectedValue(new Error('   '));

    const wrapper = mountMonitorPage();
    await flushPromises();

    expect(messageMocks.error).not.toHaveBeenCalled();
    expect(correlationActionMocks.openCorrelationErrorNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'audit.correlation.errorTitle',
        message: 'Failed to load server status',
        requestId: '',
      }),
    );
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

    await wrapper.find('[data-monitor-refresh-extra-select="true"]').setValue('30m');
    await flushPromises();

    resolveFirstRequest(createServerStatusResponse());
    await flushPromises();

    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledTimes(2);
    expect(monitorApiMocks.getServerStatus).toHaveBeenNthCalledWith(2, '30m');
  });

  it('shares the selected refresh cadence across overview, runtime, and dependencies pages', async () => {
    monitorApiMocks.getServerStatus.mockResolvedValue(createServerStatusResponse());

    const overviewWrapper = mountMonitorPage();
    await flushPromises();
    await nextTick();

    await overviewWrapper.find('[data-monitor-refresh-interval-select="true"]').setValue('10');
    await nextTick();

    const runtimeWrapper = mountRuntimePage();
    await flushPromises();
    await nextTick();

    const dependenciesWrapper = mountDependenciesPage();
    await flushPromises();
    await nextTick();

    expect(
      (overviewWrapper.find('[data-monitor-refresh-interval-select="true"]').element as HTMLSelectElement).value,
    ).toBe('10');
    expect(
      (runtimeWrapper.find('[data-monitor-refresh-interval-select="true"]').element as HTMLSelectElement).value,
    ).toBe('10');
    expect(
      (dependenciesWrapper.find('[data-monitor-refresh-interval-select="true"]').element as HTMLSelectElement).value,
    ).toBe('10');
  });
});
