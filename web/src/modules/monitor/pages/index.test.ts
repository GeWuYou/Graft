import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import MonitorPage from './index.vue';

const monitorApiMocks = vi.hoisted(() => ({
  getServerStatus: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
}));

const translations = vi.hoisted(
  (): Record<string, string> => ({
    'monitor.serverStatus.summaryTitle': 'Current Status',
    'monitor.serverStatus.statusLabel': 'Overall status',
    'monitor.serverStatus.summaryHint': 'Current summary hint',
    'monitor.serverStatus.endpointTitle': 'Live Contract',
    'monitor.serverStatus.endpointLabel': 'Endpoint: ',
    'monitor.serverStatus.fieldsLabel': 'Minimal fields: ',
    'monitor.serverStatus.fieldsValue': 'status, observed_at, server, dependencies, plugins',
    'monitor.serverStatus.lastObserved': 'Last observed: {time}',
    'monitor.serverStatus.refresh': 'Refresh Status',
    'monitor.serverStatus.serverCardTitle': 'Server Information',
    'monitor.serverStatus.versionLabel': 'Version',
    'monitor.serverStatus.startedAtLabel': 'Started at',
    'monitor.serverStatus.uptimeLabel': 'Uptime',
    'monitor.serverStatus.goVersionLabel': 'Go Version',
    'monitor.serverStatus.appLabel': 'Application',
    'monitor.serverStatus.envLabel': 'Environment',
    'monitor.serverStatus.dependencyCardTitle': 'Dependency Status',
    'monitor.serverStatus.databaseLabel': 'Database',
    'monitor.serverStatus.redisLabel': 'Redis',
    'monitor.serverStatus.pluginCardTitle': 'Plugin Summary',
    'monitor.serverStatus.pluginName': 'Plugin',
    'monitor.serverStatus.pluginVersion': 'Version',
    'monitor.serverStatus.pluginStatus': 'Status',
    'monitor.serverStatus.statusHealthy': 'Healthy',
    'monitor.serverStatus.statusDegraded': 'Degraded',
    'monitor.serverStatus.statusDisabled': 'Disabled',
    'monitor.serverStatus.statusUnknown': 'Unknown',
    'monitor.serverStatus.loadFailed': 'Failed to load server status',
    'monitor.serverStatus.empty': 'No server-status data',
  }),
);

vi.mock('../api/server-status', () => ({
  getServerStatus: monitorApiMocks.getServerStatus,
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) => {
      const template = translations[key] ?? key;
      return params ? template.replace('{time}', params.time ?? '') : template;
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
      h('div', { 'data-theme': props.theme || undefined }, [props.title, props.description, slots.default?.()]);
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
        slots.default?.(),
      );
  },
});

const tableStub = defineComponent({
  name: 'TTableStub',
  props: {
    data: {
      type: Array,
      default: () => [],
    },
  },
  setup(props, { slots }) {
    return () => {
      if (props.data.length === 0) {
        return h('div', slots.empty?.());
      }

      return h(
        'div',
        props.data.map((row, index) =>
          h('div', { 'data-testid': `plugin-row-${index}` }, [
            h('span', String((row as { name?: string }).name ?? '')),
            h('span', String((row as { version?: string }).version ?? '')),
            slots.status?.({ row }),
          ]),
        ),
      );
    };
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
    dependencies: {
      database: {
        status: 'healthy',
      },
      redis: {
        status: 'disabled',
      },
    },
    plugins: [
      {
        name: 'monitor',
        version: '0.1.0',
        status: 'healthy',
      },
      {
        name: 'user',
        version: 'unknown',
        status: 'unknown',
      },
    ],
  };
}

function mountMonitorPage() {
  return mount(MonitorPage, {
    global: {
      stubs: {
        't-button': buttonStub,
        't-card': passthroughStub,
        't-col': passthroughStub,
        't-empty': passthroughStub,
        't-row': passthroughStub,
        't-table': tableStub,
        't-tag': passthroughStub,
      },
    },
  });
}

describe('MonitorPage', () => {
  beforeEach(() => {
    monitorApiMocks.getServerStatus.mockReset();
    messageMocks.error.mockReset();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('loads server status on mount and renders dependency and plugin summaries', async () => {
    monitorApiMocks.getServerStatus.mockResolvedValue(createServerStatusResponse());

    const wrapper = mountMonitorPage();
    await flushPromises();

    expect(monitorApiMocks.getServerStatus).toHaveBeenCalledTimes(1);
    expect(wrapper.text()).toContain('Overall status');
    expect(wrapper.text()).toContain('Healthy');
    expect(wrapper.text()).toContain('Disabled');
    expect(wrapper.text()).toContain('Unknown');
    expect(wrapper.text()).toContain('Database');
    expect(wrapper.text()).toContain('Redis');
    expect(wrapper.text()).toContain('monitor');
    expect(wrapper.text()).toContain('user');
    expect(wrapper.text()).toContain('1h 1m 1s');
    expect(wrapper.findAll('[data-testid^="plugin-row-"]')).toHaveLength(2);
    expect(wrapper.find('[data-theme="success"]').exists()).toBe(true);
    expect(wrapper.find('[data-theme="danger"]').exists()).toBe(true);
    expect(wrapper.find('[data-theme="default"]').exists()).toBe(true);
  });

  it('falls back to the localized load failure message when the error is empty', async () => {
    monitorApiMocks.getServerStatus.mockRejectedValue(new Error('   '));

    const wrapper = mountMonitorPage();
    await flushPromises();

    expect(messageMocks.error).toHaveBeenCalledWith('Failed to load server status');
    expect(wrapper.text()).toContain('No server-status data');
  });
});
