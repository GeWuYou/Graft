import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import type { ContainerDashboardSummary } from '@/modules/container/contract/dashboard-summary';

import DashboardContainerResources from './DashboardContainerResources.vue';

const observabilityMocks = vi.hoisted(() => ({
  formatLocaleDateTime: vi.fn((value: string | null | undefined) => (value ? `formatted:${value}` : '')),
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
      'dashboard.containerResources.hotspots.eyebrow': 'Hotspots',
      'dashboard.containerResources.hotspots.cpuTitle': 'CPU TOP3',
      'dashboard.containerResources.hotspots.memoryTitle': 'Memory TOP3',
      'dashboard.containerResources.hotspots.top3': 'TOP3',
      'dashboard.containerResources.hotspots.empty': 'No hotspots',
      'dashboard.containerResources.anomalies.eyebrow': 'Anomalies',
      'dashboard.containerResources.anomalies.title': 'Anomaly List',
      'dashboard.containerResources.anomalies.count': `${params?.count ?? 0} anomalies`,
      'dashboard.containerResources.anomalies.empty': 'No anomalies',
      'dashboard.containerResources.anomalies.kind.unhealthy': 'Unhealthy',
      'dashboard.containerResources.anomalies.kind.restarting': 'Restarting',
      'dashboard.containerResources.anomalies.kind.exited': 'Exited',
      'dashboard.containerResources.anomalies.kind.dead': 'Dead',
      'dashboard.containerResources.anomalies.kind.high_load': 'High Load',
      'dashboard.containerResources.cpu': 'CPU',
      'dashboard.containerResources.memory': 'Memory',
      'dashboard.containerResources.memoryUsage': `${params?.usage ?? ''} / ${params?.limit ?? ''}`,
      'dashboard.containerResources.unavailable': 'Unavailable',
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
  formatLocaleDateTime: observabilityMocks.formatLocaleDateTime,
  formatPercent: (value?: number | null, fallback?: string) =>
    value === null || value === undefined ? (fallback ?? '') : `${value}%`,
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
    percentage: {
      type: Number,
      default: 0,
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
          'data-title': props.title,
          'data-description': props.description,
          'data-percentage': String(props.percentage),
          'data-theme': props.theme,
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
          id: 'cpu-1',
          name: 'cpu-hot',
          image: 'graft/server:latest',
          shortId: 'cpu-1',
          restartCount: null,
          state: 'paused',
          health: null,
          collectedAt: '2026-06-24T00:02:00Z',
          cpuPercent: 42.5,
          memoryPercent: 12.5,
          memoryUsageBytes: 100,
          memoryLimitBytes: 200,
        },
      ],
      memory: [],
      ...(overrides?.hotspots ?? {}),
    },
    anomalies: [],
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
      stubs: {
        TCard: passthroughStub,
        TEmpty: passthroughStub,
        TProgress: passthroughStub,
        TSkeleton: passthroughStub,
        TSpace: passthroughStub,
        TTag: passthroughStub,
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
    observabilityMocks.formatLocaleDateTime.mockClear();
  });

  it('formats collectedAt with the locale-aware formatter', () => {
    const wrapper = mountComponent();

    expect(observabilityMocks.formatLocaleDateTime).toHaveBeenCalledWith('2026-06-24T00:02:00Z', expect.anything(), {});
    expect(wrapper.text()).toContain('Collected At formatted:2026-06-24T00:02:00Z');
    expect(wrapper.text()).not.toContain('Collected At 2026-06-24T00:02:00Z');
  });

  it('renders the paused status label for paused containers', () => {
    const wrapper = mountComponent();

    expect(wrapper.text()).toContain('Paused');
  });
});
