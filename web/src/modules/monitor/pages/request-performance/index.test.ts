import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query';
import { flushPromises, mount, shallowMount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick, reactive } from 'vue';

import { resetMonitorRefreshPreferencesForTests } from '../../composables/use-monitor-refresh-preferences';
import RequestPerformancePage from './index.vue';

const monitorApiMocks = vi.hoisted(() => ({
  getRequestPerformance: vi.fn(),
}));

const chartMocks = vi.hoisted(() => {
  const setOption = vi.fn();
  const resize = vi.fn();
  const dispose = vi.fn();
  const init = vi.fn(() => ({ setOption, resize, dispose }));
  return { dispose, init, resize, setOption };
});

const schedulerStoreMock = vi.hoisted(() => ({
  store: null as { allowPolling: boolean } | null,
}));
const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
  query: {} as Record<string, string>,
}));

const translations = vi.hoisted((): Record<string, string> => ({
  'monitor.sectionTitle': 'Observability',
  'monitor.serverStatus.trendWindowLabel': 'Trend window',
  'monitor.serverStatus.trendRange10Minutes': '10 min',
  'monitor.serverStatus.trendRange30Minutes': '30 min',
  'monitor.serverStatus.trendRange1Hour': '1 hour',
  'monitor.serverStatus.refreshInterval5Seconds': 'Every 5 sec',
  'monitor.serverStatus.refreshInterval10Seconds': 'Every 10 sec',
  'monitor.serverStatus.refreshInterval30Seconds': 'Every 30 sec',
  'monitor.serverStatus.refreshInterval1Minute': 'Every 1 min',
  'monitor.requestPerformance.title': 'Request Performance',
  'monitor.requestPerformance.subtitle': 'Inspect request performance.',
  'monitor.requestPerformance.loadFailed': 'Failed to load request performance',
  'monitor.requestPerformance.empty': 'No request performance data.',
  'monitor.requestPerformance.statusHealthy': 'Healthy',
  'monitor.requestPerformance.statusAttention': 'Attention',
  'monitor.requestPerformance.summary.rps': 'Request Rate',
  'monitor.requestPerformance.summary.rpsHint': 'Average request rate',
  'monitor.requestPerformance.summary.latency': 'Latency Profile',
  'monitor.requestPerformance.summary.latencyHint': 'P50 and P95 latency',
  'monitor.requestPerformance.summary.errors': 'Server Error Rate',
  'monitor.requestPerformance.summary.errorsHint': '5xx affects health',
  'monitor.requestPerformance.summary.slow': 'Slow Requests',
  'monitor.requestPerformance.summary.slowHint': 'Over the slow-request threshold',
  'monitor.requestPerformance.trafficTitle': 'Request Volume',
  'monitor.requestPerformance.trafficHint': 'Request rate by minute.',
  'monitor.requestPerformance.trafficSeries': 'Requests per second',
  'monitor.requestPerformance.latencyTitle': 'High-Percentile Latency',
  'monitor.requestPerformance.latencyQualifier': 'P95',
  'monitor.requestPerformance.latencyHint': 'High-percentile latency by minute.',
  'monitor.requestPerformance.latencyHelp': 'P95 explanation',
  'monitor.requestPerformance.latencySeries': 'P95 latency (ms)',
  'monitor.requestPerformance.errorTitle': 'Server Error Rate',
  'monitor.requestPerformance.errorQualifier': '5xx',
  'monitor.requestPerformance.errorHint': '4xx is reference-only.',
  'monitor.requestPerformance.errorHelp': '5xx explanation',
  'monitor.requestPerformance.errorSeries': '5xx error rate (%)',
  'monitor.requestPerformance.infoActionLabel': 'Info',
  'monitor.requestPerformance.statusDistributionTitle': 'Request Outcome Overview',
  'monitor.requestPerformance.statusDistributionHint': 'Status groups.',
  'monitor.requestPerformance.statusGroups.success': 'Success',
  'monitor.requestPerformance.statusGroups.redirect': 'Redirect',
  'monitor.requestPerformance.statusGroups.clientError': 'Client Error',
  'monitor.requestPerformance.statusGroups.serverError': 'Server Error',
  'monitor.requestPerformance.topTrafficTitle': 'Highest Traffic Routes',
  'monitor.requestPerformance.topTrafficHint': 'Top traffic routes.',
  'monitor.requestPerformance.topErrorsTitle': 'Routes With Most Server Errors',
  'monitor.requestPerformance.topErrorsHint': 'Top 5xx routes.',
  'monitor.requestPerformance.topLatencyTitle': 'High-Latency Routes',
  'monitor.requestPerformance.topLatencyHint': 'Top P95 routes.',
  'monitor.requestPerformance.columns.method': 'Method',
  'monitor.requestPerformance.columns.route': 'Route',
  'monitor.requestPerformance.columns.requests': 'Requests',
  'monitor.requestPerformance.columns.errors': 'Server Errors',
  'monitor.requestPerformance.columns.p95': 'P95',
}));

vi.mock('../../api/request-performance', () => ({
  getRequestPerformance: monitorApiMocks.getRequestPerformance,
}));

vi.mock('echarts/core', () => ({
  init: chartMocks.init,
  use: vi.fn(),
}));

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n');
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => translations[key] ?? key,
    }),
  };
});

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: routerMocks.query }),
  useRouter: () => ({ push: routerMocks.push }),
}));

vi.mock('@/store', () => {
  const schedulerStore = reactive({ allowPolling: true });
  schedulerStoreMock.store = schedulerStore;
  return {
    useRealtimeSchedulerStore: () => schedulerStore,
    useSettingStore: () => ({
      brandTheme: '#0052d9',
      chartColors: {
        borderColor: '#d9e0ea',
        containerColor: '#ffffff',
        placeholderColor: '#8a94a6',
        textColor: '#111827',
      },
      displayMode: 'light',
      resolvedThemeTokensForDisplayMode: {},
    }),
  };
});

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  setup(_props, { attrs, slots }) {
    return () => h('div', attrs, slots.default?.());
  },
});

const sectionCardStub = defineComponent({
  name: 'SectionCardStub',
  props: {
    description: { default: '', type: String },
    title: { default: '', type: String },
  },
  setup(props, { slots }) {
    return () => h('section', [slots.title?.(), h('p', props.description), slots.default?.()]);
  },
});

const tooltipStub = defineComponent({
  name: 'TooltipStub',
  props: { content: { default: '', type: String } },
  setup(props, { slots }) {
    return () => h('span', { 'data-tooltip-content': props.content }, slots.default?.());
  },
});

function createResponse() {
  return {
    latency_distribution: [
      { lower_bound: 0, upper_bound: 5, sample_count: 120, sample_rate: 80 },
      { lower_bound: 100, upper_bound: null, sample_count: 3, sample_rate: 2 },
    ],
    largest_requests: [
      {
        request_id: 'req-large',
        observed_at: '2026-07-14T09:00:00Z',
        method: 'POST',
        path: '/upload',
        route: '/upload',
        status_code: 201,
        duration_ms: 20,
        request_size_bytes: 4096,
        response_size_bytes: 128,
      },
    ],
    largest_responses: [],
    minute_buckets: [
      {
        error_5xx_count: 0,
        error_5xx_rate: 2,
        observed_at: '2026-07-14T09:00:00Z',
        p95_latency_ms: 42,
        p99_latency_ms: 84,
        request_bytes_per_second: 1024,
        requests_per_second: 2.5,
        response_bytes_per_second: 2048,
        total_requests: 150,
      },
    ],
    observed_at: '2026-07-14T09:01:00Z',
    range: '10m',
    request_size_distribution: [{ lower_bound: 0, upper_bound: 1024, sample_count: 150, sample_rate: 100 }],
    response_size_distribution: [{ lower_bound: 0, upper_bound: 1024, sample_count: 150, sample_rate: 100 }],
    slowest_requests: [],
    status_codes: [
      { status_code: 200, request_count: 120, request_rate: 80 },
      { status_code: 500, request_count: 3, request_rate: 2 },
    ],
    status_groups: [
      { request_count: 120, request_rate: 80, status_group: '2xx' },
      { request_count: 3, request_rate: 2, status_group: '3xx' },
      { request_count: 24, request_rate: 16, status_group: '4xx' },
      { request_count: 3, request_rate: 2, status_group: '5xx' },
    ],
    summary: {
      active_requests: 2,
      average_latency_ms: 10,
      error_5xx_count: 3,
      error_5xx_rate: 2,
      max_latency_ms: 500,
      p50_latency_ms: 7,
      p95_latency_ms: 42,
      p99_latency_ms: 84,
      request_bytes: {
        measured_count: 150,
        total_bytes: 614400,
        average_bytes: 4096,
        bytes_per_second: 1024,
      },
      requests_per_second: 2.5,
      response_bytes: {
        measured_count: 150,
        total_bytes: 1228800,
        average_bytes: 8192,
        bytes_per_second: 2048,
      },
      slow_request_count: 1,
      total_requests: 150,
    },
    top_routes: { errors_5xx: [], p95_latency: [], traffic: [] },
    window_end: '2026-07-14T09:01:00Z',
    window_start: '2026-07-14T08:51:00Z',
  };
}

function mountPage() {
  return mount(RequestPerformancePage, {
    global: {
      plugins: [[VueQueryPlugin, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) }]],
      stubs: {
        'info-circle-icon': passthroughStub,
        'monitor-page-feedback': passthroughStub,
        'refresh-control-bar': passthroughStub,
        'section-card': sectionCardStub,
        'server-status-page-shell': passthroughStub,
        'summary-metric-card': passthroughStub,
        't-button': passthroughStub,
        't-empty': passthroughStub,
        't-table': passthroughStub,
        't-tooltip': tooltipStub,
      },
    },
  });
}

beforeEach(() => {
  resetMonitorRefreshPreferencesForTests();
  monitorApiMocks.getRequestPerformance.mockReset();
  monitorApiMocks.getRequestPerformance.mockResolvedValue(createResponse());
  schedulerStoreMock.store!.allowPolling = true;
  routerMocks.push.mockReset();
  Object.keys(routerMocks.query).forEach((key) => delete routerMocks.query[key]);
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.useRealTimers();
});

describe('request performance page', () => {
  it('keeps pending visible at the refresh deadline before starting the next countdown', async () => {
    vi.useFakeTimers();
    const wrapper = shallowMount(RequestPerformancePage, {
      global: {
        plugins: [
          [VueQueryPlugin, { queryClient: new QueryClient({ defaultOptions: { queries: { retry: false } } }) }],
        ],
      },
    });
    await flushPromises();
    await nextTick();

    const page = wrapper.vm as unknown as { remainingRefreshSeconds: number | null };
    expect(page.remainingRefreshSeconds).toBe(5);

    let resolveRefresh: ((value: ReturnType<typeof createResponse>) => void) | undefined;
    monitorApiMocks.getRequestPerformance.mockImplementationOnce(
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

  it('uses resolved semantic chart colors and keeps tooltip values unit-aware', async () => {
    schedulerStoreMock.store!.allowPolling = false;
    vi.spyOn(window, 'getComputedStyle').mockReturnValue({
      getPropertyValue: (token: string) =>
        ({
          '--td-brand-color': '#0052d9',
          '--td-error-color-5': '#e34d59',
          '--td-warning-color-5': '#ebb105',
        })[token] ?? '',
    } as CSSStyleDeclaration);

    const wrapper = mountPage();
    await flushPromises();

    const options = chartMocks.setOption.mock.calls.map(([option]) => option) as Array<{
      color: string[];
      series: Array<{ areaStyle?: { opacity: number }; lineStyle?: { color: string; width: number } }>;
      tooltip: {
        formatter?: (
          params: Array<{ axisValueLabel: string; color: string; data: number; seriesIndex?: number }>,
        ) => string;
      };
    }>;

    expect(options).toHaveLength(7);
    expect(options.slice(0, 4).map((option) => option.color[0])).toEqual(['#0052d9', '#ebb105', '#e34d59', '#0052d9']);
    expect(options.every((option) => !option.color[0].startsWith('var('))).toBe(true);
    expect(options.slice(0, 4).every((option) => option.series[0]?.lineStyle?.width === 2)).toBe(true);
    expect(options.slice(0, 4).every((option) => option.series[0]?.areaStyle?.opacity === 0.1)).toBe(true);
    expect(options[0]?.tooltip.formatter?.([{ axisValueLabel: '09:00', color: '#0052d9', data: 2.5 }])).toContain(
      '2.50 RPS',
    );
    expect(options[1]?.tooltip.formatter?.([{ axisValueLabel: '09:00', color: '#ebb105', data: 42 }])).toContain(
      '42 ms',
    );
    expect(options[2]?.tooltip.formatter?.([{ axisValueLabel: '09:00', color: '#e34d59', data: 2 }])).toContain(
      '2.00%',
    );
    expect(wrapper.text()).toContain('Success');
    expect(wrapper.text()).toContain('Server Error');
    expect(wrapper.find('[data-tooltip-content="P95 explanation"]').exists()).toBe(true);
    expect(wrapper.find('[data-tooltip-content="5xx explanation"]').exists()).toBe(true);
    wrapper.unmount();
  });

  it('renders eight summaries and builds status/request drilldowns with monitor origin', async () => {
    schedulerStoreMock.store!.allowPolling = false;
    const wrapper = mountPage();
    await flushPromises();
    const page = wrapper.vm as unknown as {
      summaryCards: Array<{ key: string }>;
      toggleStatusGroup: (group: string) => void;
      expandedStatusGroups: string[];
      openStatusCode: (code: number) => void;
      openRequestInstance: (context: { row: ReturnType<typeof createResponse>['largest_requests'][number] }) => void;
    };

    expect(page.summaryCards.map((item) => item.key)).toEqual([
      'rps',
      'latency',
      'average',
      'p99',
      'errors',
      'slow',
      'active',
      'total',
    ]);
    page.toggleStatusGroup('5xx');
    expect(page.expandedStatusGroups).toEqual(['5xx']);
    page.openStatusCode(500);
    expect(routerMocks.push).toHaveBeenLastCalledWith({
      path: '/observability/access-logs',
      query: {
        status_code: '500',
        occurred_from: '2026-07-14T08:51:00Z',
        occurred_to: '2026-07-14T09:01:00Z',
        monitorView: 'request-performance',
        monitorTrendRange: '10m',
      },
    });
    page.openRequestInstance({ row: createResponse().largest_requests[0] });
    expect(routerMocks.push).toHaveBeenLastCalledWith({
      path: '/observability/access-logs',
      query: {
        request_id: 'req-large',
        monitorView: 'request-performance',
        monitorTrendRange: '10m',
      },
    });
    wrapper.unmount();
  });

  it('marks empty route tables so their horizontal scrollbars stay hidden', async () => {
    schedulerStoreMock.store!.allowPolling = false;

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.findAll('.request-performance-page__route-table')).toHaveLength(3);
    expect(wrapper.findAll('.request-performance-page__route-table--empty')).toHaveLength(3);
    wrapper.unmount();
  });

  it('restores the selected trend range from monitor origin context', async () => {
    schedulerStoreMock.store!.allowPolling = false;
    routerMocks.query.monitorView = 'request-performance';
    routerMocks.query.monitorTrendRange = '30m';

    const wrapper = mountPage();
    await flushPromises();

    expect(monitorApiMocks.getRequestPerformance).toHaveBeenCalledWith('30m');
    wrapper.unmount();
  });
});
