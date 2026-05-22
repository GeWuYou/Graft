import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { resetMonitorRefreshPreferencesForTests } from '../composables/use-monitor-refresh-preferences';
import DependenciesPage from './dependencies.vue';

const monitorApiMocks = vi.hoisted(() => ({
  getServerStatus: vi.fn(),
}));

const translations = vi.hoisted(
  (): Record<string, string> => ({
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
    'monitor.serverStatus.refreshStateLabel': 'Refresh state',
    'monitor.serverStatus.nextRefreshPausedByUser': 'Auto refresh paused',
    'monitor.serverStatus.nextRefreshPaused': 'Next refresh paused while the page is hidden',
    'monitor.serverStatus.nextRefreshPending': 'Preparing the next refresh',
    'monitor.serverStatus.nextRefreshIn': 'Next refresh in {seconds}s',
    'monitor.serverStatus.nextRefreshRetryIn': 'Retry in {seconds}s · base interval {interval}',
    'monitor.dependenciesPage.title': 'Dependencies',
    'monitor.dependenciesPage.subtitle':
      'Review health signals for PostgreSQL, Redis, and future plugin-owned dependency checks.',
    'monitor.dependenciesPage.noteTitle': 'Dependency health scope',
    'monitor.dependenciesPage.noteDescription':
      'The current page reflects the latest aggregated snapshot. Additional services can show their own health checks here as they become available.',
    'monitor.dependenciesPage.statusHealthy': 'Healthy',
    'monitor.dependenciesPage.statusAbnormal': 'Abnormal',
    'monitor.dependenciesPage.statusNotConfigured': 'Not configured',
    'monitor.dependenciesPage.statusUnknown': 'Unknown',
    'monitor.dependenciesPage.postgresqlSubtitle': 'Primary relational database health',
    'monitor.dependenciesPage.redisSubtitle': 'Cache and lightweight KV health',
    'monitor.dependenciesPage.futureEntryTitle': 'Plugin dependency extension',
    'monitor.dependenciesPage.futureEntrySubtitle': 'Reserved for plugin-owned health probes',
    'monitor.dependenciesPage.futureEntryLabel': 'Reserved entry',
    'monitor.dependenciesPage.futureEntryHint':
      'Future plugins can plug their own dependency checks in here without further menu restructuring.',
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
    'monitor.dependenciesPage.fields.checkedAt': 'Last checked',
    'monitor.dependenciesPage.fields.errorInfo': 'Error info',
    'monitor.dependenciesPage.fields.detail': 'Probe detail',
    'monitor.dependenciesPage.fields.extensionEntry': 'Extension entry',
    'monitor.dependenciesPage.fieldDescriptions.latency': 'Most recent probe response duration.',
    'monitor.dependenciesPage.fieldDescriptions.checkedAt': 'Currently follows the latest server-status snapshot time.',
    'monitor.dependenciesPage.fieldDescriptions.errorInfo':
      'Shows the latest error only when the dependency is abnormal or unknown.',
    'monitor.dependenciesPage.fieldDescriptions.detail': 'Raw probe detail returned by the current backend snapshot.',
    'monitor.serverStatus.postgresqlLabel': 'PostgreSQL',
    'monitor.serverStatus.redisLabel': 'Redis',
  }),
);

vi.mock('../api/server-status', () => ({
  getServerStatus: monitorApiMocks.getServerStatus,
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => translations[key] ?? key,
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

function mountDependenciesPage() {
  return mount(DependenciesPage, {
    global: {
      stubs: {
        't-card': passthroughStub,
        't-tag': passthroughStub,
        't-button': buttonStub,
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
      database: { status: 'healthy', detail: 'Database ping succeeded', latency_ms: 2.1 },
      redis: { status: 'disabled', detail: 'Redis client is not configured', latency_ms: null },
    },
    summary: {
      total_dependencies: 2,
      healthy_dependencies: 1,
      degraded_dependencies: 0,
      unknown_dependencies: 0,
      disabled_dependencies: 1,
      total_plugins: 5,
      healthy_plugins: 4,
    },
    trend: {
      range: '10m',
      retention_seconds: 600,
      sample_interval_seconds: 5,
      points: [],
    },
    plugins: [],
  };
}

describe('monitor dependencies page', () => {
  it('renders dependency states, last check details, and future extension entry', async () => {
    monitorApiMocks.getServerStatus.mockResolvedValue(createResponse());

    const wrapper = mountDependenciesPage();
    await flushPromises();
    const observedAt = new Date('2026-05-21T10:30:00Z');
    const expectedTime = new Intl.DateTimeFormat(undefined, {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    }).format(observedAt);
    const expectedDate = new Intl.DateTimeFormat(undefined, {
      year: 'numeric',
      month: 'numeric',
      day: 'numeric',
    }).format(observedAt);

    expect(wrapper.attributes('data-page-type')).toBe('overview-dashboard');
    expect(wrapper.text()).toContain('Dependencies');
    expect(wrapper.text()).toContain('PostgreSQL');
    expect(wrapper.text()).toContain('Redis');
    expect(wrapper.text()).toContain('Healthy');
    expect(wrapper.text()).toContain('Not configured');
    expect(wrapper.text()).toContain('Refresh cadence');
    expect(wrapper.text()).toContain('Every 5 sec');
    expect(wrapper.text()).toContain('Pause auto refresh');
    expect(wrapper.text()).toContain('Plugin dependency extension');
    expect(wrapper.text()).toContain('Redis client is not configured');
    expect(wrapper.text()).toContain(expectedTime);
    expect(wrapper.text()).toContain(expectedDate);
    expect(wrapper.find('[data-monitor-refresh-extra-select="true"]').exists()).toBe(false);
  });
});
