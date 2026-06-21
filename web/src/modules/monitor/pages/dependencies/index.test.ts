import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { resetMonitorRefreshPreferencesForTests } from '../../composables/use-monitor-refresh-preferences';
import DependenciesPage from './index.vue';

const monitorApiMocks = vi.hoisted(() => ({
  getServerStatus: vi.fn(),
}));

const translations = vi.hoisted(
  (): Record<string, string> => ({
    'app.refreshControl.labels.interval': '自动刷新：',
    'app.refreshControl.labels.trendWindow': '趋势窗口：',
    'app.refreshControl.status.running': '自动刷新：{interval}',
    'app.refreshControl.status.paused': '自动刷新已暂停',
    'app.refreshControl.status.off': '自动刷新关闭',
    'app.refreshControl.countdown': '{countdown} 后刷新',
    'app.refreshControl.pending': '等待下次刷新',
    'app.refreshControl.actions.refresh': 'Refresh now',
    'app.refreshControl.actions.pause': 'Pause auto refresh',
    'app.refreshControl.actions.resume': 'Resume auto refresh',
    'app.refreshControl.actions.enable': 'Enable auto refresh',
    'app.refreshControl.actions.pauseCompact': 'Pause',
    'app.refreshControl.actions.resumeCompact': 'Resume',
    'app.refreshControl.actions.enableCompact': 'Enable',
    'monitor.sectionTitle': 'Server Management',
    'monitor.shared.loadFailed': 'Failed to load server status',
    'monitor.shared.empty': 'No server-status snapshot is available yet',
    'monitor.shared.errorTitle': 'Snapshot request failed',
    'monitor.shared.notReported': 'Not reported',
    'monitor.serverStatus.refreshIntervalLabel': 'Refresh cadence',
    'monitor.serverStatus.refreshInterval5Seconds': 'Every 5 sec',
    'monitor.serverStatus.refreshInterval10Seconds': 'Every 10 sec',
    'monitor.serverStatus.refreshInterval30Seconds': 'Every 30 sec',
    'monitor.serverStatus.refreshInterval1Minute': 'Every 1 min',
    'monitor.serverStatus.refreshNow': 'Refresh now',
    'monitor.serverStatus.pauseRefresh': 'Pause auto refresh',
    'monitor.serverStatus.resumeRefresh': 'Resume auto refresh',
    'monitor.serverStatus.nextRefreshLabel': 'Next refresh',
    'monitor.serverStatus.refreshStateLabel': 'Refresh state',
    'monitor.serverStatus.nextRefreshPausedByUser': 'Auto refresh paused',
    'monitor.serverStatus.nextRefreshPaused': 'Next refresh paused while the page is hidden',
    'monitor.serverStatus.nextRefreshPending': 'Preparing the next refresh',
    'monitor.serverStatus.nextRefreshIn': 'Next refresh in {seconds}s',
    'monitor.serverStatus.nextRefreshRetryIn': 'Retry in {seconds}s · base interval {interval}',
    'monitor.dependenciesPage.title': 'Dependencies',
    'monitor.dependenciesPage.subtitle':
      'Review health signals for PostgreSQL, Redis, and future module-owned dependency checks.',
    'monitor.dependenciesPage.noteTitle': 'Dependency health scope',
    'monitor.dependenciesPage.noteDescription':
      'The current page reflects the latest aggregated snapshot. Additional services can show their own health checks here as they become available.',
    'monitor.dependenciesPage.statusHealthy': 'Healthy',
    'monitor.dependenciesPage.statusAbnormal': 'Abnormal',
    'monitor.dependenciesPage.statusNotConfigured': 'Not configured',
    'monitor.dependenciesPage.statusUnknown': 'Unknown',
    'monitor.dependenciesPage.postgresqlSubtitle': 'Primary relational database health',
    'monitor.dependenciesPage.redisSubtitle': 'Cache and lightweight KV health',
    'monitor.dependenciesPage.futureEntryTitle': 'Module dependency extension',
    'monitor.dependenciesPage.futureEntrySubtitle': 'Reserved for module-owned health probes',
    'monitor.dependenciesPage.futureEntryLabel': 'Reserved entry',
    'monitor.dependenciesPage.futureEntryHint':
      'Future modules can plug their own dependency checks in here without further menu restructuring.',
    'monitor.dependenciesPage.futureEntryDescription':
      'This card will show new dependency checks when they are available.',
    'monitor.dependenciesPage.noError': 'No current error',
    'monitor.dependenciesPage.summary.healthy': 'Healthy',
    'monitor.dependenciesPage.summary.healthyDescription': 'Dependencies responding normally',
    'monitor.dependenciesPage.summary.abnormal': 'Abnormal',
    'monitor.dependenciesPage.summary.abnormalDescription': 'Dependencies returning degraded probes',
    'monitor.dependenciesPage.summary.notConfigured': 'Not configured',
    'monitor.dependenciesPage.summary.notConfiguredDescription': 'Dependencies intentionally not wired in',
    'monitor.dependenciesPage.summary.lastCheck': 'Last check',
    'monitor.dependenciesPage.summary.lastCheckDescription': 'Time of the latest aggregated snapshot',
    'monitor.dependenciesPage.fields.latency': 'Response latency',
    'monitor.dependenciesPage.fields.poolWait': 'Pool waits',
    'monitor.dependenciesPage.fields.timeoutCount': 'Wait timeouts',
    'monitor.dependenciesPage.fields.staleCount': 'Connections recycled',
    'monitor.dependenciesPage.fields.checkedAt': 'Last checked',
    'monitor.dependenciesPage.fields.errorInfo': 'Error info',
    'monitor.dependenciesPage.fields.detail': 'Probe detail',
    'monitor.dependenciesPage.fields.extensionEntry': 'Extension entry',
    'monitor.dependenciesPage.pool.title': 'Connection pool',
    'monitor.dependenciesPage.pool.stateTitle': 'Pool state',
    'monitor.dependenciesPage.pool.usageLabel': '{label} pool usage',
    'monitor.dependenciesPage.pool.usageTooltip': '{label} pool {value} · usage {percent}',
    'monitor.dependenciesPage.pool.inUse': 'In use',
    'monitor.dependenciesPage.pool.idle': 'Idle',
    'monitor.dependenciesPage.pool.open': 'Total connections',
    'monitor.dependenciesPage.pool.capacity': 'Max connections',
    'monitor.dependenciesPage.pool.riskHealthy': 'Pool pressure is normal',
    'monitor.dependenciesPage.pool.riskWarning': 'Pool is approaching a high watermark',
    'monitor.dependenciesPage.pool.riskCritical': 'Pool is close to exhaustion',
    'monitor.dependenciesPage.pool.riskUnknown': 'Pool data is unavailable',
    'monitor.dependenciesPage.diagnostics.title': 'Advanced diagnostics',
    'monitor.dependenciesPage.fieldDescriptions.latency': 'Most recent probe response duration.',
    'monitor.dependenciesPage.fieldDescriptions.checkedAt': 'Currently follows the latest server-status snapshot time.',
    'monitor.dependenciesPage.fieldDescriptions.errorInfo':
      'Shows the latest error only when the dependency is abnormal or unknown.',
    'monitor.dependenciesPage.fieldDescriptions.detail': 'Raw probe detail returned by the current backend snapshot.',
    'monitor.serverStatus.postgresqlLabel': 'PostgreSQL',
    'monitor.serverStatus.redisLabel': 'Redis',
  }),
);

vi.mock('../../api/server-status', () => ({
  getServerStatus: monitorApiMocks.getServerStatus,
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: 'en-US',
    t: (key: string, params?: Record<string, unknown>) => {
      const template = translations[key] ?? key;
      if (!params) {
        return template;
      }

      return Object.entries(params).reduce(
        (result, [token, value]) => result.replace(`{${token}}`, String(value)),
        template,
      );
    },
  }),
}));

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: {
    title: {
      type: String,
      default: '',
    },
    theme: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () => h('div', { 'data-theme': props.theme || undefined }, [props.title, slots.default?.()]);
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_props, { attrs, emit, slots }) {
    return () => h('button', { ...attrs, onClick: (event: MouseEvent) => emit('click', event) }, slots.default?.());
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

const drawerStub = defineComponent({
  name: 'TDrawerStub',
  props: {
    header: {
      type: String,
      default: '',
    },
    visible: {
      type: Boolean,
      default: false,
    },
    footer: {
      type: [Boolean, String],
      default: true,
    },
  },
  setup(props, { slots }) {
    return () =>
      props.visible
        ? h(
            'aside',
            {
              'data-testid': 'diagnostic-drawer',
              'data-header': props.header,
              'data-footer': String(props.footer),
            },
            [props.header, slots.default?.()],
          )
        : null;
  },
});

function mountDependenciesPage() {
  return mount(DependenciesPage, {
    global: {
      stubs: {
        't-card': passthroughStub,
        't-tag': passthroughStub,
        't-button': buttonStub,
        't-drawer': drawerStub,
        't-select': selectStub,
        't-empty': passthroughStub,
      },
    },
  });
}

afterEach(() => {
  resetMonitorRefreshPreferencesForTests();
});

function createResponse() {
  return {
    status: 'degraded',
    observed_at: '2026-05-21T10:30:00Z',
    server: {
      version: 'v0.3.2',
      started_at: '2026-05-21T08:30:00Z',
      uptime_seconds: 7200,
      go_version: 'go1.26.0',
      app_name: 'graft',
      app_env: 'prod',
    },
    runtime: {
      go_version: 'go1.26.0',
      host_name: 'node-a',
      operating_system: 'linux',
      architecture: 'amd64',
      cpu_cores: 8,
      load_average: {
        one_minute: 0.34,
        five_minutes: 0.28,
        fifteen_minutes: 0.22,
      },
      disk_usage: {
        path: '/',
        total_bytes: 0,
        used_bytes: 0,
        free_bytes: 0,
        used_percent: 0,
      },
      host_memory_total_bytes: 17179869184,
      host_memory_used_bytes: 8589934592,
      host_memory_free_bytes: 8589934592,
      host_memory_used_percent: 50,
      goroutines: 37,
      runtime_alloc_bytes: 41943040,
      runtime_heap_in_use_bytes: 31457280,
      runtime_sys_bytes: 83886080,
      runtime_gc_cycles: 12,
    },
    dependencies: {
      database: {
        status: 'healthy',
        detail: 'Database ping succeeded',
        latency_ms: 2.1,
        pool: {
          capacity: 25,
          max_active_connections: 25,
          open_connections: 8,
          in_use_connections: 3,
          idle_connections: 5,
          usage_percent: 12,
          wait_count: 2,
          wait_duration_ms: 4.25,
          timeout_count: 0,
          stale_count: 1,
        },
      },
      redis: {
        status: 'healthy',
        detail: 'Redis ping succeeded',
        latency_ms: 0.23,
        pool: {
          capacity: 280,
          max_active_connections: 0,
          open_connections: 280,
          in_use_connections: 252,
          idle_connections: 28,
          usage_percent: 90,
          wait_count: 0,
          wait_duration_ms: 0,
          timeout_count: 3,
          stale_count: 2,
        },
      },
    },
    summary: {
      total_dependencies: 2,
      healthy_dependencies: 2,
      degraded_dependencies: 0,
      unknown_dependencies: 0,
      disabled_dependencies: 0,
      total_modules: 5,
      healthy_modules: 4,
    },
    trend: {
      range: '10m',
      retention_seconds: 600,
      sample_interval_seconds: 5,
      points: [],
    },
    modules: [],
  };
}

describe('monitor dependencies page', () => {
  it('renders aligned dependency cards and opens selected diagnostics in a drawer', async () => {
    monitorApiMocks.getServerStatus.mockResolvedValue(createResponse());

    const wrapper = mountDependenciesPage();
    await flushPromises();
    const observedAt = new Date('2026-05-21T10:30:00Z');
    const expectedTime = new Intl.DateTimeFormat('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    }).format(observedAt);
    const expectedDate = new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: 'numeric',
      day: 'numeric',
    }).format(observedAt);

    expect(wrapper.attributes('data-page-type')).toBe('overview-dashboard');
    expect(wrapper.text()).toContain('Dependencies');
    expect(wrapper.text()).toContain('PostgreSQL');
    expect(wrapper.text()).toContain('Redis');
    expect(wrapper.text()).toContain('Healthy');
    expect(wrapper.text()).toContain('Response latency');
    expect(wrapper.text()).toContain('2.10 ms');
    expect(wrapper.text()).toContain('Connection pool');
    expect(wrapper.text()).toContain('3 / 25');
    expect(wrapper.text()).toContain('12%');
    expect(wrapper.text()).toContain('Pool pressure is normal');
    expect(wrapper.text()).toContain('252 / 280');
    expect(wrapper.text()).toContain('90%');
    expect(wrapper.text()).toContain('Pool is close to exhaustion');
    expect(wrapper.text()).toContain('Pool state');
    expect(wrapper.text()).toContain('In use');
    expect(wrapper.text()).toContain('Idle');
    expect(wrapper.text()).toContain('Total connections');
    expect(wrapper.text()).toContain('Max connections');
    expect(wrapper.text()).toContain('Advanced diagnostics');
    expect(wrapper.find('[data-testid="diagnostic-drawer"]').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('2 · 4.25 ms');
    expect(wrapper.text()).not.toContain('Wait timeouts');
    expect(wrapper.text()).not.toContain('Connections recycled');
    expect(wrapper.text()).toContain('Every 5 sec');
    expect(wrapper.text()).toContain('5s 后刷新');
    expect(wrapper.text()).toContain('Pause auto refresh');
    expect(wrapper.text()).toContain('Module dependency extension');
    expect(wrapper.text()).not.toContain('Redis ping succeeded');
    expect(wrapper.text()).toContain(expectedTime);
    expect(wrapper.text()).toContain(expectedDate);
    expect(wrapper.find('[data-dependency-key="redis"] [data-usage-status="danger"]').exists()).toBe(true);
    expect(wrapper.find('[data-dependency-key="postgresql"] [data-usage-status="healthy"]').exists()).toBe(true);
    expect(wrapper.find('[data-refresh-trend-window-select="true"]').exists()).toBe(false);

    await wrapper.get('[data-dependency-key="redis"] .dependency-health-card__diagnostic-action').trigger('click');

    const drawer = wrapper.get('[data-testid="diagnostic-drawer"]');
    expect(drawer.attributes('data-header')).toBe('Redis Advanced diagnostics');
    expect(drawer.attributes('data-footer')).toBe('false');
    expect(drawer.text()).toContain('Pool waits');
    expect(drawer.text()).toContain('0 · 0.00 ms');
    expect(drawer.text()).toContain('Wait timeouts');
    expect(drawer.text()).toContain('3');
    expect(drawer.text()).toContain('Connections recycled');
    expect(drawer.text()).toContain('2');
    expect(drawer.text()).toContain('Redis ping succeeded');
  });
});
