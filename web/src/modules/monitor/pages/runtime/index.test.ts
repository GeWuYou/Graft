// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { resetMonitorRefreshPreferencesForTests } from '../../composables/use-monitor-refresh-preferences';
import RuntimePage from './index.vue';

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
    'monitor.runtimePage.title': 'Runtime',
    'monitor.runtimePage.subtitle':
      'Inspect the current Go process snapshot, build details, and server environment while keeping server memory separate from Go Runtime memory.',
    'monitor.runtimePage.snapshotReady': 'Snapshot ready',
    'monitor.runtimePage.snapshotPending': 'Waiting for snapshot',
    'monitor.runtimePage.memoryBoundaryNotice':
      'Memory scope: server memory covers the whole server, while Go Runtime memory covers only the current service process.',
    'monitor.runtimePage.runtimeMemoryTitle': 'Go Runtime Memory',
    'monitor.runtimePage.runtimeMemoryDescription': 'Current in-process memory usage and garbage collection snapshot.',
    'monitor.runtimePage.processBuildTitle': 'Process and Build',
    'monitor.runtimePage.hostEnvironmentTitle': 'Server Environment',
    'monitor.runtimePage.serverEnvironmentDescription':
      'Whole-server memory snapshot, not the same as Go Runtime memory.',
    'monitor.runtimePage.summary.uptime': 'Uptime',
    'monitor.runtimePage.summary.uptimeDescription': 'Process running duration',
    'monitor.runtimePage.summary.goroutines': 'Goroutines',
    'monitor.runtimePage.summary.goroutinesDescription': 'Current Go scheduler concurrency',
    'monitor.runtimePage.summary.runtimeAlloc': 'Current alloc',
    'monitor.runtimePage.summary.runtimeAllocDescription': 'Currently allocated by the Go runtime',
    'monitor.runtimePage.summary.gcCycles': 'GC cycles',
    'monitor.runtimePage.summary.gcCyclesDescription': 'Completed garbage collection rounds',
    'monitor.runtimePage.fields.runtimeAlloc': 'Current alloc',
    'monitor.runtimePage.fields.runtimeHeap': 'Heap in use',
    'monitor.runtimePage.fields.runtimeSys': 'Runtime sys',
    'monitor.runtimePage.fields.gcCycles': 'GC count',
    'monitor.runtimePage.fields.lastGc': 'Last GC time',
    'monitor.runtimePage.fields.buildVersion': 'Build version',
    'monitor.runtimePage.fields.gitCommit': 'Git commit',
    'monitor.runtimePage.fields.appName': 'Application',
    'monitor.runtimePage.fields.appEnv': 'Environment',
    'monitor.runtimePage.fields.startedAt': 'Started at',
    'monitor.runtimePage.fields.goVersion': 'Go version',
    'monitor.runtimePage.fields.hostName': 'Host name',
    'monitor.runtimePage.fields.platform': 'Platform',
    'monitor.runtimePage.fields.cpuCores': 'CPU cores',
    'monitor.runtimePage.fields.hostMemory': 'Server memory',
    'monitor.runtimePage.fields.hostMemoryUsage': 'Memory usage',
    'monitor.runtimePage.fields.observedAt': 'Last observed',
    'monitor.runtimePage.fields.loadAverage': 'Load average',
    'monitor.runtimePage.fieldDescriptions.runtimeAlloc': 'Bytes currently allocated by the Go runtime.',
    'monitor.runtimePage.fieldDescriptions.runtimeHeap': 'Heap bytes actively used by the Go process.',
    'monitor.runtimePage.fieldDescriptions.runtimeSys': 'Total memory requested from the system by the Go runtime.',
    'monitor.runtimePage.fieldDescriptions.gcCycles': 'Current snapshot of completed GC cycles.',
    'monitor.runtimePage.fieldDescriptions.lastGc': 'Reserved until the backend exposes the most recent GC timestamp.',
    'monitor.runtimePage.fieldDescriptions.buildVersion': 'Uses the current backend version field when present.',
    'monitor.runtimePage.fieldDescriptions.gitCommit':
      'Reserved until the backend exposes an explicit commit identifier.',
    'monitor.runtimePage.fieldDescriptions.appName': 'Application identifier reported by the service.',
    'monitor.runtimePage.fieldDescriptions.appEnv': 'Current runtime environment label.',
    'monitor.runtimePage.fieldDescriptions.startedAt': 'First stable process startup time.',
    'monitor.runtimePage.fieldDescriptions.hostName': 'Server host name reported by the runtime snapshot.',
    'monitor.runtimePage.fieldDescriptions.platform': 'Operating system and architecture for the host.',
    'monitor.runtimePage.fieldDescriptions.cpuCores': 'Logical CPU core count visible to the process.',
    'monitor.runtimePage.fieldDescriptions.hostMemory': 'Whole-server memory snapshot, not Go Runtime memory.',
    'monitor.runtimePage.fieldDescriptions.observedAt': 'Timestamp of the latest aggregated snapshot.',
    'monitor.runtimePage.fieldDescriptions.loadAverage': '1m / 5m / 15m load averages from the current server.',
  }),
);

vi.mock('../../api/server-status', () => ({
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

function mountRuntimePage() {
  return mount(RuntimePage, {
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
    status: 'healthy',
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
      database: { status: 'healthy', detail: 'ok', latency_ms: 2.1 },
      redis: { status: 'disabled', detail: 'not configured', latency_ms: null },
    },
    summary: {
      total_dependencies: 2,
      healthy_dependencies: 1,
      degraded_dependencies: 0,
      unknown_dependencies: 0,
      disabled_dependencies: 1,
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

describe('monitor runtime page', () => {
  it('renders runtime and host sections with explicit memory separation and reserved fields', async () => {
    monitorApiMocks.getServerStatus.mockResolvedValue(createResponse());

    const wrapper = mountRuntimePage();
    await flushPromises();

    expect(wrapper.attributes('data-page-type')).toBe('overview-dashboard');
    expect(wrapper.text()).toContain('Runtime');
    expect(wrapper.text()).toContain('Go Runtime Memory');
    expect(wrapper.text()).toContain('Server Environment');
    expect(wrapper.text()).toContain('Server memory');
    expect(wrapper.text()).toContain('Memory usage');
    expect(wrapper.text()).toContain('Git commit');
    expect(wrapper.text()).toContain('Not reported');
    expect(wrapper.text()).toContain('Build version');
    expect(wrapper.text()).toContain('Go version');
    expect(wrapper.text()).toContain('Refresh cadence');
    expect(wrapper.text()).toContain('Every 5 sec');
    expect(wrapper.text()).toContain('Pause auto refresh');
    expect(wrapper.text()).toContain('Current alloc');
    expect(wrapper.text()).toContain(
      'Memory scope: server memory covers the whole server, while Go Runtime memory covers only the current service process.',
    );
    expect(wrapper.text()).toContain('8.00 GB / 16.0 GB');
    expect(wrapper.text()).toContain('50%');
    expect(wrapper.text()).toContain('v0.3.2');
    expect(wrapper.findAll('.server-status-summary-card')).toHaveLength(4);
    expect(wrapper.findAll('.server-status-runtime-grid__card')).toHaveLength(3);
    expect(wrapper.findAll('.server-status-kv-row')).toHaveLength(16);
    expect(wrapper.find('[data-refresh-trend-window-select="true"]').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('Snapshot Context');
    expect(wrapper.text()).not.toContain('Metric scope');
    expect(wrapper.text().match(/Snapshot ready/g)).toHaveLength(1);
    expect(wrapper.text()).not.toContain('Data status');
  });
});
