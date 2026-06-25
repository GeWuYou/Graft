import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import type { ContainerDashboardSummary } from '@/modules/container/contract/dashboard-summary';

import DashboardContainerResources from './DashboardContainerResources.vue';

const observabilityMocks = vi.hoisted(() => ({
  formatLocaleDateTimeMock: vi.fn((value: string | null | undefined) => (value ? `formatted:${value}` : '')),
}));

vi.mock('@/locales', () => ({
  currentLocale: ref('en-US'),
  t: (key: string, params?: Record<string, unknown>) => {
    const translations: Record<string, string> = {
      'dashboard.containerResources.title': 'Container Resource Overview',
      'dashboard.containerResources.source': 'Shared Container Resource View',
      'dashboard.containerResources.collectedAt': 'Collected At',
      'dashboard.containerResources.empty': 'No container resource data',
      'dashboard.containerResources.overview.title': 'Container Resource Overview',
      'dashboard.containerResources.overview.running.label': 'Running Containers',
      'dashboard.containerResources.overview.running.value': String(params?.count ?? ''),
      'dashboard.containerResources.overview.running.description': 'Running description',
      'dashboard.containerResources.overview.abnormal.label': 'Abnormal Containers',
      'dashboard.containerResources.overview.abnormal.value': String(params?.count ?? ''),
      'dashboard.containerResources.overview.abnormal.description': 'Abnormal description',
      'dashboard.containerResources.overview.cpuTotal.label': 'CPU Total',
      'dashboard.containerResources.overview.cpuTotal.description': 'CPU description',
      'dashboard.containerResources.overview.memoryTotal.label': 'Memory Total',
      'dashboard.containerResources.overview.memoryTotal.description': 'Memory description',
      'dashboard.containerResources.consumers.eyebrow': 'Top consumers',
      'dashboard.containerResources.consumers.title': 'Top Resource Consumers',
      'dashboard.containerResources.consumers.topCount': `Top ${params?.count ?? 0}`,
      'dashboard.containerResources.consumers.empty': 'No ranked container resource data',
      'dashboard.containerResources.consumers.noRunning': 'No running containers.',
      'dashboard.containerResources.consumers.rankCpu': `CPU #${params?.rank ?? ''}`,
      'dashboard.containerResources.consumers.rankMemory': `Memory #${params?.rank ?? ''}`,
      'dashboard.containerResources.anomalies.eyebrow': 'Anomalies',
      'dashboard.containerResources.anomalies.title': 'Anomaly List',
      'dashboard.containerResources.anomalies.count': `${params?.count ?? 0} anomalies`,
      'dashboard.containerResources.anomalies.empty': 'No anomalies',
      'dashboard.containerResources.anomalies.kind.unhealthy': 'Unhealthy',
      'dashboard.containerResources.anomalies.kind.restarting': 'Restarting',
      'dashboard.containerResources.anomalies.kind.exited': 'Exited',
      'dashboard.containerResources.anomalies.kind.dead': 'Dead',
      'dashboard.containerResources.anomalies.kind.high_load': 'High Load',
      'dashboard.containerResources.anomalies.reasonCode.state_restarting': 'Restart Back-off',
      'dashboard.containerResources.anomalies.resourceSummary': `CPU ${params?.cpu ?? ''} / Memory ${params?.memory ?? ''}`,
      'dashboard.containerResources.anomalies.restartCount': `${params?.count ?? 0} restarts`,
      'dashboard.containerResources.anomalies.reasonFallback': 'Investigate the latest runtime state',
      'dashboard.containerResources.anomalies.noCollectedAt': 'Collected At Unknown',
      'dashboard.containerResources.cpu': 'CPU',
      'dashboard.containerResources.memory': 'Memory',
      'dashboard.containerResources.memoryUsage': `${params?.usage ?? ''} / ${params?.limit ?? ''}`,
      'dashboard.containerResources.metrics.cpuDescription': 'Current CPU usage share',
      'dashboard.containerResources.metricStateDescription.running': 'Collected from the latest snapshot',
      'dashboard.containerResources.metricStateDescription.notApplicable':
        'Metric does not apply while the container is not running',
      'dashboard.containerResources.metricStateDescription.notCollected': 'Waiting for the next resource sample',
      'dashboard.containerResources.metricStateDescription.unknown': 'Metric state is unknown',
      'dashboard.containerResources.notApplicable': 'N/A',
      'dashboard.containerResources.notCollected': 'Not Collected',
      'dashboard.containerResources.status.dead': 'Dead',
      'dashboard.containerResources.status.exited': 'Exited',
      'dashboard.containerResources.status.paused': 'Paused',
      'dashboard.containerResources.status.restarting': 'Restarting',
      'dashboard.containerResources.status.running': 'Running',
      'dashboard.containerResources.status.unknown': 'Unknown',
      'dashboard.containerResources.status.unhealthy': 'Unhealthy',
    };
    return translations[key] ?? key;
  },
}));

vi.mock('@/shared/observability', () => ({
  MEDIUM_DATE_TIME_WITH_SECONDS_FORMAT_OPTIONS: {},
  formatBytes: (value?: number | null, fallback?: string) =>
    value === null || value === undefined ? (fallback ?? '') : `${value} B`,
  formatLocaleDateTime: observabilityMocks.formatLocaleDateTimeMock,
  formatPercent: (value?: number | null, fallback?: string) =>
    value === null || value === undefined ? (fallback ?? '') : `${value}%`,
}));

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: {
    description: {
      type: String,
      default: '',
    },
    percentage: {
      type: Number,
      default: 0,
    },
    status: {
      type: String,
      default: '',
    },
    theme: {
      type: String,
      default: '',
    },
    title: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        {
          'data-description': props.description,
          'data-percentage': String(props.percentage),
          'data-status': props.status,
          'data-theme': props.theme,
          'data-title': props.title,
        },
        [slots.actions?.(), slots.default?.()],
      );
  },
});

function createSummary(overrides?: Partial<ContainerDashboardSummary>): ContainerDashboardSummary {
  return {
    overview: {
      runningContainers: 3,
      abnormalContainers: 1,
      cpuTotalPercent: 42.5,
      memoryTotalUsageBytes: 1024,
      memoryTotalLimitBytes: 2048,
      memoryTotalPercent: 50,
      collectedAt: '2026-06-24T00:02:00Z',
      ...(overrides?.overview ?? {}),
    },
    hotspots: {
      cpu: [
        {
          id: 'container-1',
          name: 'api',
          image: 'graft/api:latest',
          shortId: 'api',
          restartCount: null,
          state: 'running',
          health: null,
          collectedAt: '2026-06-24T00:02:00Z',
          cpuPercent: 92,
          memoryPercent: 45,
          memoryUsageBytes: 100,
          memoryLimitBytes: 200,
        },
      ],
      memory: [
        {
          id: 'container-2',
          name: 'worker',
          image: 'graft/worker:latest',
          shortId: 'worker',
          restartCount: null,
          state: 'paused',
          health: null,
          collectedAt: '2026-06-24T00:02:00Z',
          cpuPercent: null,
          memoryPercent: null,
          memoryUsageBytes: null,
          memoryLimitBytes: null,
        },
      ],
      ...(overrides?.hotspots ?? {}),
    },
    anomalies: [
      {
        id: 'anomaly-1',
        name: 'scheduler',
        image: 'graft/scheduler:latest',
        shortId: 'scheduler',
        restartCount: 5,
        state: 'running',
        health: null,
        status: 'Restarting',
        reasonCode: 'state.restarting',
        reasonLabel: 'Container is restarting repeatedly',
        collectedAt: '2026-06-24T00:05:00Z',
        cpuPercent: 10,
        memoryPercent: 20,
        memoryUsageBytes: 256,
        memoryLimitBytes: 512,
      },
      ...(overrides?.anomalies ?? []),
    ],
    ...(overrides ?? {}),
  };
}

function mountComponent(summary = createSummary(), loading = false) {
  return mount(DashboardContainerResources, {
    props: {
      summary,
      loading,
    },
    global: {
      components: {
        't-card': passthroughStub,
        't-empty': passthroughStub,
        't-progress': passthroughStub,
        't-skeleton': passthroughStub,
        't-space': passthroughStub,
        't-tag': passthroughStub,
      },
    },
  });
}

describe('DashboardContainerResources', () => {
  beforeEach(() => {
    observabilityMocks.formatLocaleDateTimeMock.mockClear();
  });

  it('formats collectedAt with the locale-aware formatter', () => {
    const wrapper = mountComponent();

    expect(observabilityMocks.formatLocaleDateTimeMock).toHaveBeenCalledWith(
      '2026-06-24T00:02:00Z',
      expect.anything(),
      {},
    );
    expect(wrapper.text()).toContain('Collected At formatted:2026-06-24T00:02:00Z');
    expect(wrapper.text()).not.toContain('Collected At 2026-06-24T00:02:00Z');
  });

  it('renders unified top consumer cards from cpu and memory hotspots', () => {
    const wrapper = mountComponent();

    const cards = wrapper.findAll('[data-testid="dashboard-container-resource-consumer-item"]');
    expect(cards).toHaveLength(2);
    expect(wrapper.text()).toContain('Top Resource Consumers');
    expect(wrapper.text()).toContain('CPU #1');
    expect(wrapper.text()).toContain('Memory #1');
  });

  it('shows N/A instead of unavailable for stopped or paused resource metrics', () => {
    const wrapper = mountComponent();

    expect(wrapper.text()).toContain('Paused');
    expect(wrapper.text()).toContain('N/A');
    expect(wrapper.text()).not.toContain('Unavailable');
  });

  it('does not render a duplicated status tag when the anomaly cause already matches the runtime state', () => {
    const wrapper = mountComponent(
      createSummary({
        anomalies: [
          {
            id: 'anomaly-2',
            name: 'cli-proxy-api',
            image: 'eceasy/cli-proxy-api:latest',
            shortId: 'cli-proxy-api',
            restartCount: null,
            state: 'exited',
            health: null,
            status: 'Exited',
            reasonCode: 'state.exited',
            reasonLabel: null,
            collectedAt: '2026-06-25T10:42:58Z',
            cpuPercent: null,
            memoryPercent: null,
            memoryUsageBytes: null,
            memoryLimitBytes: null,
          },
        ],
      }),
    );

    expect(wrapper.text()).toContain('Exited');
    expect(wrapper.findAll('[data-testid="dashboard-anomaly-primary-tag"]')).toHaveLength(1);
    expect(wrapper.findAll('[data-testid="dashboard-anomaly-status-tag"]')).toHaveLength(0);
  });

  it('shows skeleton surfaces during first load instead of empty metrics', () => {
    const wrapper = mountComponent(createSummary(), true);

    expect(wrapper.findAll('[data-percentage]').length).toBeGreaterThan(0);
    expect(wrapper.text()).not.toContain('No container resource data');
    expect(wrapper.findAll('[data-testid="dashboard-container-resource-consumer-item"]')).toHaveLength(0);
  });

  it('shows no-running empty state and hides consumer cards when no containers are running', () => {
    const wrapper = mountComponent(
      createSummary({
        overview: {
          runningContainers: 0,
          abnormalContainers: 1,
          cpuTotalPercent: null as never,
          memoryTotalPercent: null,
          collectedAt: '2026-06-24T00:02:00Z',
          memoryTotalUsageBytes: null,
          memoryTotalLimitBytes: null,
        },
        hotspots: {
          cpu: [],
          memory: [],
        },
        anomalies: [
          {
            id: 'stopped-1',
            name: 'stopped-worker',
            image: 'graft/worker:latest',
            shortId: 'stopped-worker',
            restartCount: null,
            state: 'paused',
            health: null,
            status: 'Paused',
            reasonCode: null,
            reasonLabel: null,
            collectedAt: null,
            cpuPercent: null,
            memoryPercent: null,
            memoryUsageBytes: null,
            memoryLimitBytes: null,
          },
        ],
      }),
    );

    expect(wrapper.findAll('[data-testid="dashboard-container-resource-consumer-item"]')).toHaveLength(0);
    expect(wrapper.text()).toContain('Top 0');
    expect(wrapper.text()).toContain('N/A');
  });

  it('shows anomaly cause hierarchy from existing reason fields', () => {
    const wrapper = mountComponent();

    expect(wrapper.text()).toContain('Restart Back-off');
    expect(wrapper.text()).toContain('Container is restarting repeatedly');
    expect(wrapper.text()).toContain('5 restarts');
    expect(wrapper.text()).toContain('CPU 10% / Memory 20%');
  });
});
