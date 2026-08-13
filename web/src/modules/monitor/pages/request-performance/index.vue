<template>
  <server-status-page-shell
    class="request-performance-page"
    :eyebrow="t('monitor.sectionTitle')"
    title-key="monitor.requestPerformance.title"
    description-key="monitor.requestPerformance.subtitle"
  >
    <template #toolbar>
      <refresh-control-bar
        :status="refreshStatus"
        :countdown-seconds="remainingRefreshSeconds"
        :interval="selectedRefreshInterval"
        :interval-options="refreshIntervalOptions"
        :refreshing="loading"
        :show-countdown="true"
        :show-trend-window="true"
        :status-tone="performanceTone"
        :status-label="performanceStatusLabel"
        :trend-window="selectedRange"
        :trend-window-label="t('monitor.serverStatus.trendWindowLabel')"
        :trend-window-options="rangeOptions"
        variant="page"
        @refresh="refresh"
        @pause="toggleAutoRefresh"
        @resume="toggleAutoRefresh"
        @update:interval="updateInterval"
        @update:trend-window="updateRange"
      />
    </template>

    <template #summary>
      <summary-metric-card
        v-for="item in summaryCards"
        :key="item.key"
        :title="item.title"
        :value="item.value"
        :description="item.description"
        :value-aside="item.valueAside"
        :status="item.status"
        :status-label="item.statusLabel"
      />
    </template>

    <monitor-page-feedback
      v-if="errorMessage"
      :title="t('monitor.requestPerformance.loadFailed')"
      :description="errorMessage"
      tone="error"
    />

    <template v-if="snapshot">
      <div class="request-performance-page__trends">
        <section-card
          v-for="chart in charts"
          :key="chart.key"
          :title="chart.title"
          :description="chart.description"
          :min-height="260"
        >
          <template v-if="chart.qualifier || chart.help" #title>
            <span class="request-performance-page__chart-title">
              <span>{{ chart.title }}</span>
              <small v-if="chart.qualifier" class="request-performance-page__chart-qualifier">
                {{ chart.qualifier }}
              </small>
              <t-tooltip v-if="chart.help" :content="chart.help" placement="top" theme="light">
                <button
                  type="button"
                  class="request-performance-page__chart-help"
                  :aria-label="`${chart.title} ${t('monitor.requestPerformance.infoActionLabel')}`"
                >
                  <info-circle-icon />
                </button>
              </t-tooltip>
            </span>
          </template>
          <div :ref="(element) => setChartRef(chart.key, element)" class="request-performance-page__chart" />
        </section-card>
      </div>

      <div class="request-performance-page__distributions">
        <section-card
          v-for="histogram in histograms"
          :key="histogram.key"
          :title="histogram.title"
          :description="histogram.description"
          :min-height="220"
        >
          <t-empty v-if="!histogram.available" :description="histogram.empty" />
          <div
            v-else
            :ref="(element) => setChartRef(histogram.key, element)"
            class="request-performance-page__chart request-performance-page__chart--compact"
          />
        </section-card>
      </div>

      <section-card
        :title="t('monitor.requestPerformance.statusDistributionTitle')"
        :description="t('monitor.requestPerformance.statusDistributionHint')"
      >
        <div class="request-performance-page__status-groups">
          <section
            v-for="group in statusGroups"
            :key="group.status_group"
            class="request-performance-page__status-group"
            :data-status-group="group.status_group"
          >
            <t-button
              class="request-performance-page__status-toggle"
              theme="default"
              variant="text"
              :aria-expanded="expandedStatusGroups.includes(group.status_group)"
              @click="toggleStatusGroup(group.status_group)"
            >
              <span class="request-performance-page__status-group-heading">
                <strong>{{ group.label }}</strong>
                <small>{{ group.status_group }}</small>
              </span>
            </t-button>
            <span>{{ formatCount(group.request_count) }}</span>
            <small class="request-performance-page__status-group-rate">{{ formatPercent(group.request_rate) }}</small>
            <div
              v-if="expandedStatusGroups.includes(group.status_group)"
              class="request-performance-page__status-codes"
            >
              <t-button
                v-for="code in group.codes"
                :key="code.status_code"
                size="small"
                theme="default"
                variant="outline"
                @click="openStatusCode(code.status_code)"
              >
                {{ code.status_code }} · {{ formatCount(code.request_count) }}
              </t-button>
              <small v-if="group.codes.length === 0">{{ t('monitor.requestPerformance.statusCodesEmpty') }}</small>
            </div>
          </section>
        </div>
      </section-card>

      <div class="request-performance-page__routes">
        <section-card v-for="list in routeLists" :key="list.key" :title="list.title" :description="list.description">
          <t-table
            class="request-performance-page__route-table"
            :class="{ 'request-performance-page__route-table--empty': list.rows.length === 0 }"
            :data="list.rows"
            :columns="routeColumns"
            row-key="route"
            size="small"
          />
        </section-card>
      </div>

      <div class="request-performance-page__instances">
        <section-card v-for="list in instanceLists" :key="list.key" :title="list.title" :description="list.description">
          <t-table
            class="request-performance-page__instance-table"
            :data="list.rows"
            :columns="instanceColumns"
            row-key="request_id"
            size="small"
            @row-click="openRequestInstance"
          >
            <template #operation="{ row }">
              <t-button size="small" theme="primary" variant="text" @click.stop="openRequestInstance({ row })">
                {{ t('monitor.requestPerformance.instances.view') }}
              </t-button>
            </template>
          </t-table>
        </section-card>
      </div>
    </template>
    <t-empty v-else-if="!loading" :description="t('monitor.requestPerformance.empty')" />
  </server-status-page-shell>
</template>
<script setup lang="ts">
// 请求性能页只消费 OpenAPI 聚合快照，并把下钻状态编码为 Access Log URL 查询；不在前端重算后端统计事实。
import { useQuery } from '@tanstack/vue-query';
import { BarChart, LineChart } from 'echarts/charts';
import { GridComponent, TooltipComponent } from 'echarts/components';
import * as echarts from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { InfoCircleIcon } from 'tdesign-icons-vue-next';
import type { PrimaryTableCol, TableRowData } from 'tdesign-vue-next';
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { buildAccessLogLocation, buildAccessLogRequestLocation } from '@/modules/access-log/contract/deep-link';
import { RefreshControlBar, type RefreshControlOption, type RefreshControlStatus } from '@/shared/components/refresh';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatBytes } from '@/shared/observability';
import { useRealtimeSchedulerStore, useSettingStore } from '@/store';

import { getRequestPerformance } from '../../api/request-performance';
import MonitorPageFeedback from '../../components/MonitorPageFeedback.vue';
import SectionCard from '../../components/SectionCard.vue';
import ServerStatusPageShell from '../../components/ServerStatusPageShell.vue';
import SummaryMetricCard from '../../components/SummaryMetricCard.vue';
import { useMonitorRefreshPreferences } from '../../composables/use-monitor-refresh-preferences';
import { type MonitorOriginContext, parseMonitorOriginQuery } from '../../contract/navigation';
import type { MonitorRefreshInterval } from '../../contract/refresh';
import { MONITOR_TREND_RANGE, type MonitorTrendRange } from '../../contract/trend';
import { formatChartTimeOnly } from '../../shared/time-display';

echarts.use([GridComponent, TooltipComponent, LineChart, BarChart, CanvasRenderer]);

defineOptions({ name: 'MonitorRequestPerformanceIndex' });

type ChartKey =
  | 'traffic'
  | 'latency'
  | 'errors'
  | 'bytes'
  | 'latencyDistribution'
  | 'requestSizeDistribution'
  | 'responseSizeDistribution';
interface ChartDefinition {
  key: ChartKey;
  title: string;
  description: string;
  qualifier?: string;
  help?: string;
}

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const scheduler = useRealtimeSchedulerStore();
const settingStore = useSettingStore();
const { autoRefreshEnabled, refreshIntervalOptions, selectedRefreshInterval, toggleAutoRefresh } =
  useMonitorRefreshPreferences();
const initialOrigin = parseMonitorOriginQuery(route.query as Record<string, unknown>);
const selectedRange = ref<MonitorTrendRange>(normalizeTrendRange(initialOrigin?.trendRange));
const expandedStatusGroups = ref<string[]>([]);
const remainingRefreshSeconds = ref<number | null>(null);
const chartRefs = ref<Partial<Record<ChartKey, HTMLDivElement | null>>>({});
const chartInstances = new Map<ChartKey, echarts.ECharts>();
const chartElements = new Map<ChartKey, HTMLDivElement>();
let refreshTimer: number | null = null;
let refreshDeadline: number | null = null;
let isMounted = false;

const {
  data: snapshot,
  error: snapshotError,
  isFetching: loading,
  refetch: refetchSnapshot,
} = useQuery({
  queryKey: computed(() => ['monitor', 'request-performance', selectedRange.value]),
  queryFn: () => getRequestPerformance(selectedRange.value),
  enabled: false,
  retry: false,
});
const errorMessage = computed(() =>
  snapshotError.value
    ? resolveLocalizedErrorMessage(t, snapshotError.value, t('monitor.requestPerformance.loadFailed'))
    : '',
);

const rangeOptions = computed<RefreshControlOption[]>(() => [
  { label: t('monitor.serverStatus.trendRange10Minutes'), value: MONITOR_TREND_RANGE.TEN_MINUTES },
  { label: t('monitor.serverStatus.trendRange30Minutes'), value: MONITOR_TREND_RANGE.THIRTY_MINUTES },
  { label: t('monitor.serverStatus.trendRange1Hour'), value: MONITOR_TREND_RANGE.ONE_HOUR },
]);
const refreshStatus = computed<RefreshControlStatus>(() => (autoRefreshEnabled.value ? 'running' : 'paused'));
const performanceTone = computed(() =>
  snapshot.value && snapshot.value.summary.error_5xx_count > 0 ? 'error' : 'healthy',
);
const performanceStatusLabel = computed(() =>
  performanceTone.value === 'error'
    ? t('monitor.requestPerformance.statusAttention')
    : t('monitor.requestPerformance.statusHealthy'),
);
const summaryCards = computed(() => {
  const summary = snapshot.value?.summary;
  return [
    card(
      'rps',
      t('monitor.requestPerformance.summary.rps'),
      formatRps(summary?.requests_per_second),
      t('monitor.requestPerformance.summary.rpsHint'),
    ),
    card(
      'latency',
      t('monitor.requestPerformance.summary.latency'),
      `${formatLatency(summary?.p50_latency_ms)} / ${formatLatency(summary?.p95_latency_ms)}`,
      t('monitor.requestPerformance.summary.latencyHint'),
    ),
    card(
      'average',
      t('monitor.requestPerformance.summary.average'),
      formatLatency(summary?.average_latency_ms),
      t('monitor.requestPerformance.summary.averageHint'),
      t('monitor.requestPerformance.summary.maxValue', { value: formatLatency(summary?.max_latency_ms) }),
    ),
    card(
      'p99',
      t('monitor.requestPerformance.summary.p99'),
      formatLatency(summary?.p99_latency_ms),
      t('monitor.requestPerformance.summary.p99Hint'),
    ),
    card(
      'errors',
      t('monitor.requestPerformance.summary.errors'),
      formatPercent(summary?.error_5xx_rate),
      t('monitor.requestPerformance.summary.errorsHint'),
    ),
    card(
      'slow',
      t('monitor.requestPerformance.summary.slow'),
      formatCount(summary?.slow_request_count),
      t('monitor.requestPerformance.summary.slowHint'),
    ),
    card(
      'active',
      t('monitor.requestPerformance.summary.active'),
      formatCount(summary?.active_requests),
      t('monitor.requestPerformance.summary.activeHint'),
    ),
    card(
      'total',
      t('monitor.requestPerformance.summary.total'),
      formatCount(summary?.total_requests),
      t('monitor.requestPerformance.summary.totalHint'),
    ),
  ];
});
const charts = computed<ChartDefinition[]>(() => [
  {
    key: 'traffic' as const,
    title: t('monitor.requestPerformance.trafficTitle'),
    description: t('monitor.requestPerformance.trafficHint'),
  },
  {
    key: 'latency' as const,
    title: t('monitor.requestPerformance.latencyTitle'),
    description: t('monitor.requestPerformance.latencyHint'),
    qualifier: t('monitor.requestPerformance.latencyQualifier'),
    help: t('monitor.requestPerformance.latencyHelp'),
  },
  {
    key: 'errors' as const,
    title: t('monitor.requestPerformance.errorTitle'),
    description: t('monitor.requestPerformance.errorHint'),
    qualifier: t('monitor.requestPerformance.errorQualifier'),
    help: t('monitor.requestPerformance.errorHelp'),
  },
  {
    key: 'bytes' as const,
    title: t('monitor.requestPerformance.bytesTitle'),
    description: t('monitor.requestPerformance.bytesHint'),
  },
]);
const histograms = computed(() => [
  {
    key: 'latencyDistribution' as const,
    title: t('monitor.requestPerformance.histogram.latencyTitle'),
    description: t('monitor.requestPerformance.histogram.latencyHint'),
    empty: t('monitor.requestPerformance.histogram.empty'),
    available: (snapshot.value?.summary.total_requests ?? 0) > 0,
  },
  {
    key: 'requestSizeDistribution' as const,
    title: t('monitor.requestPerformance.histogram.requestSizeTitle'),
    description: t('monitor.requestPerformance.histogram.requestSizeHint'),
    empty: t('monitor.requestPerformance.histogram.sizeUnknown'),
    available: (snapshot.value?.summary.request_bytes.measured_count ?? 0) > 0,
  },
  {
    key: 'responseSizeDistribution' as const,
    title: t('monitor.requestPerformance.histogram.responseSizeTitle'),
    description: t('monitor.requestPerformance.histogram.responseSizeHint'),
    empty: t('monitor.requestPerformance.histogram.sizeUnknown'),
    available: (snapshot.value?.summary.response_bytes.measured_count ?? 0) > 0,
  },
]);
const statusGroups = computed(() =>
  (snapshot.value?.status_groups ?? []).map((group) => ({
    ...group,
    label: statusGroupLabel(group.status_group),
    codes: (snapshot.value?.status_codes ?? []).filter(
      (code) => `${Math.floor(code.status_code / 100)}xx` === group.status_group,
    ),
  })),
);
const routeLists = computed(() => {
  const routes = snapshot.value?.top_routes;
  return [
    {
      key: 'traffic',
      title: t('monitor.requestPerformance.topTrafficTitle'),
      description: t('monitor.requestPerformance.topTrafficHint'),
      rows: routes?.traffic ?? [],
    },
    {
      key: 'errors',
      title: t('monitor.requestPerformance.topErrorsTitle'),
      description: t('monitor.requestPerformance.topErrorsHint'),
      rows: routes?.errors_5xx ?? [],
    },
    {
      key: 'latency',
      title: t('monitor.requestPerformance.topLatencyTitle'),
      description: t('monitor.requestPerformance.topLatencyHint'),
      rows: routes?.p95_latency ?? [],
    },
  ];
});
const routeColumns = computed<PrimaryTableCol[]>(() => [
  { colKey: 'method', title: t('monitor.requestPerformance.columns.method'), width: 84 },
  { colKey: 'route', title: t('monitor.requestPerformance.columns.route'), ellipsis: true },
  {
    colKey: 'total_requests',
    title: t('monitor.requestPerformance.columns.requests'),
    width: 96,
  },
  {
    colKey: 'error_5xx_count',
    title: t('monitor.requestPerformance.columns.errors'),
    width: 108,
  },
  {
    colKey: 'p95_latency_ms',
    title: t('monitor.requestPerformance.columns.p95'),
    width: 108,
  },
]);
type RequestInstance = NonNullable<typeof snapshot.value>['slowest_requests'][number];
type RequestInstanceRow = RequestInstance & { display_size_bytes: number | null };
const instanceLists = computed(() => {
  const response = snapshot.value;
  return [
    {
      key: 'slowest',
      title: t('monitor.requestPerformance.instances.slowestTitle'),
      description: t('monitor.requestPerformance.instances.slowestHint'),
      rows: (response?.slowest_requests ?? []).map((row) => ({
        ...row,
        display_size_bytes: row.response_size_bytes ?? row.request_size_bytes,
      })),
    },
    {
      key: 'request',
      title: t('monitor.requestPerformance.instances.requestTitle'),
      description: t('monitor.requestPerformance.instances.requestHint'),
      rows: (response?.largest_requests ?? []).map((row) => ({
        ...row,
        display_size_bytes: row.request_size_bytes,
      })),
    },
    {
      key: 'response',
      title: t('monitor.requestPerformance.instances.responseTitle'),
      description: t('monitor.requestPerformance.instances.responseHint'),
      rows: (response?.largest_responses ?? []).map((row) => ({
        ...row,
        display_size_bytes: row.response_size_bytes,
      })),
    },
  ];
});
const instanceColumns = computed<PrimaryTableCol[]>(() => [
  { colKey: 'method', title: t('monitor.requestPerformance.columns.method'), width: 72 },
  { colKey: 'path', title: t('monitor.requestPerformance.columns.path'), ellipsis: true },
  { colKey: 'status_code', title: t('monitor.requestPerformance.columns.status'), width: 76 },
  {
    colKey: 'duration_ms',
    title: t('monitor.requestPerformance.columns.duration'),
    width: 92,
    cell: (_h, { row }) => formatLatency(row.duration_ms as number),
  },
  {
    colKey: 'size',
    title: t('monitor.requestPerformance.columns.size'),
    width: 100,
    cell: (_h, { row }) => formatBytes((row as RequestInstanceRow).display_size_bytes, '--'),
  },
  { colKey: 'operation', title: t('monitor.requestPerformance.columns.operation'), width: 72 },
]);

function card(
  key: string,
  title: string,
  value: string,
  description: string,
  valueAside = '',
): {
  key: string;
  title: string;
  value: string;
  description: string;
  valueAside: string;
  status: 'healthy' | 'error';
  statusLabel: string;
} {
  return {
    key,
    title,
    value,
    description,
    valueAside,
    status: performanceTone.value,
    statusLabel: performanceStatusLabel.value,
  };
}
function formatCount(value?: number) {
  return value !== undefined && Number.isFinite(value) ? new Intl.NumberFormat().format(value) : '--';
}
function formatRps(value?: number) {
  return Number.isFinite(value) ? `${value!.toFixed(2)} RPS` : '--';
}
function formatLatency(value?: number) {
  return Number.isFinite(value) ? `${value!.toFixed(0)} ms` : '--';
}
function formatPercent(value?: number) {
  return Number.isFinite(value) ? `${value!.toFixed(2)}%` : '--';
}
function normalizeTrendRange(value?: string): MonitorTrendRange {
  return Object.values(MONITOR_TREND_RANGE).includes(value as MonitorTrendRange)
    ? (value as MonitorTrendRange)
    : MONITOR_TREND_RANGE.TEN_MINUTES;
}
function monitorOrigin(): MonitorOriginContext {
  return { view: 'request-performance', trendRange: selectedRange.value };
}
function toggleStatusGroup(statusGroup: string) {
  expandedStatusGroups.value = expandedStatusGroups.value.includes(statusGroup)
    ? expandedStatusGroups.value.filter((item) => item !== statusGroup)
    : [...expandedStatusGroups.value, statusGroup];
}
function openStatusCode(statusCode: number) {
  const response = snapshot.value;
  if (!response) return;
  void router.push(
    buildAccessLogLocation(
      {
        status_code: String(statusCode),
        occurred_from: response.window_start,
        occurred_to: response.window_end,
      },
      monitorOrigin(),
    ),
  );
}
function openRequestInstance({ row }: { row: TableRowData }) {
  const request = row as RequestInstanceRow;
  void router.push(buildAccessLogRequestLocation(request.request_id, monitorOrigin()));
}
function statusGroupLabel(statusGroup: string) {
  switch (statusGroup) {
    case '2xx':
      return t('monitor.requestPerformance.statusGroups.success');
    case '3xx':
      return t('monitor.requestPerformance.statusGroups.redirect');
    case '4xx':
      return t('monitor.requestPerformance.statusGroups.clientError');
    case '5xx':
      return t('monitor.requestPerformance.statusGroups.serverError');
    default:
      return statusGroup;
  }
}
function setChartRef(key: ChartKey, value: unknown) {
  chartRefs.value[key] = value instanceof HTMLDivElement ? value : null;
}
function updateInterval(value: number | string) {
  selectedRefreshInterval.value = Number(value) as MonitorRefreshInterval;
  scheduleRefresh();
}
function updateRange(value: number | string) {
  selectedRange.value = value as MonitorTrendRange;
  void refresh();
}

async function refresh() {
  await refetchSnapshot();
  if (isMounted) {
    scheduleRefresh();
  }
}

watch(snapshot, async (value) => {
  if (!value) return;
  await nextTick();
  renderCharts();
});

function scheduleRefresh() {
  if (refreshTimer !== null) window.clearInterval(refreshTimer);
  refreshTimer = null;
  refreshDeadline = null;
  remainingRefreshSeconds.value = null;
  if (!autoRefreshEnabled.value || !scheduler.allowPolling || selectedRefreshInterval.value <= 0) return;
  refreshDeadline = Date.now() + selectedRefreshInterval.value * 1000;
  updateRemainingRefreshSeconds();
  refreshTimer = window.setInterval(() => {
    updateRemainingRefreshSeconds();
    if (remainingRefreshSeconds.value === 0) void refresh();
  }, 1000);
}
function updateRemainingRefreshSeconds() {
  if (refreshDeadline === null) {
    remainingRefreshSeconds.value = null;
    return;
  }

  remainingRefreshSeconds.value = Math.max(0, Math.ceil((refreshDeadline - Date.now()) / 1000));
}
function renderCharts() {
  const buckets = snapshot.value?.minute_buckets ?? [];
  const labels = buckets.map((bucket) => formatChartTimeOnly(bucket.observed_at));
  const definitions = {
    traffic: {
      series: [
        {
          name: t('monitor.requestPerformance.trafficSeries'),
          values: buckets.map((bucket) => bucket.requests_per_second),
          color: readChartThemeColor('--td-brand-color', '#0052d9'),
          value: (input: number) => `${input.toFixed(2)} RPS`,
        },
      ],
    },
    latency: {
      series: [
        {
          name: t('monitor.requestPerformance.latencySeriesP95'),
          values: buckets.map((bucket) => bucket.p95_latency_ms),
          color: readChartThemeColor('--td-warning-color-5', '#ebb105'),
          value: (input: number) => `${input.toFixed(0)} ms`,
        },
        {
          name: t('monitor.requestPerformance.latencySeriesP99'),
          values: buckets.map((bucket) => bucket.p99_latency_ms),
          color: readChartThemeColor('--td-error-color-4', '#f36d78'),
          value: (input: number) => `${input.toFixed(0)} ms`,
        },
      ],
    },
    errors: {
      series: [
        {
          name: t('monitor.requestPerformance.errorSeries'),
          values: buckets.map((bucket) => bucket.error_5xx_rate),
          color: readChartThemeColor('--td-error-color-5', '#e34d59'),
          value: (input: number) => `${input.toFixed(2)}%`,
        },
      ],
    },
    bytes: {
      series: [
        {
          name: t('monitor.requestPerformance.bytesInSeries'),
          values: buckets.map((bucket) => bucket.request_bytes_per_second),
          color: readChartThemeColor('--td-brand-color', '#0052d9'),
          value: (input: number) => `${formatBytes(input)}/s`,
        },
        {
          name: t('monitor.requestPerformance.bytesOutSeries'),
          values: buckets.map((bucket) => bucket.response_bytes_per_second),
          color: readChartThemeColor('--td-success-color-5', '#2ba471'),
          value: (input: number) => `${formatBytes(input)}/s`,
        },
      ],
    },
  };
  renderChartDefinitions(definitions, (definition, instance) => {
    instance.setOption({
      color: definition.series.map((series) => series.color),
      tooltip: {
        trigger: 'axis',
        axisPointer: { lineStyle: { color: settingStore.chartColors.borderColor } },
        backgroundColor: settingStore.chartColors.containerColor,
        borderColor: settingStore.chartColors.borderColor,
        textStyle: { color: settingStore.chartColors.textColor },
        formatter: (params: Array<{ axisValueLabel?: string; color: string; data: number; seriesIndex?: number }>) => {
          const first = params[0];
          if (!first) return '';
          return [
            first.axisValueLabel ?? '',
            ...params.map((point) => {
              const series = definition.series[point.seriesIndex ?? 0];
              return series
                ? `<span style="color:${point.color}">●</span> ${series.name}: <strong>${series.value(point.data)}</strong>`
                : '';
            }),
          ].join('<br/>');
        },
      },
      grid: { left: 44, right: 16, top: 20, bottom: 28 },
      xAxis: {
        type: 'category',
        data: labels,
        boundaryGap: false,
        axisLabel: { color: settingStore.chartColors.placeholderColor },
        axisLine: { lineStyle: { color: settingStore.chartColors.borderColor } },
        axisTick: { show: false },
      },
      yAxis: {
        type: 'value',
        axisLabel: { color: settingStore.chartColors.placeholderColor },
        splitLine: { lineStyle: { color: settingStore.chartColors.borderColor } },
      },
      series: definition.series.map((series) => ({
        name: series.name,
        type: 'line',
        smooth: true,
        showSymbol: false,
        lineStyle: { color: series.color, width: 2 },
        itemStyle: { color: series.color },
        emphasis: { focus: 'series' },
        areaStyle: { opacity: 0.1 },
        data: series.values,
      })),
    });
  });
  renderHistogramCharts();
}

function renderHistogramCharts() {
  const response = snapshot.value;
  if (!response) return;
  const definitions = {
    latencyDistribution: {
      items: response.latency_distribution,
      color: readChartThemeColor('--td-warning-color-5', '#ebb105'),
      label: (lower: number, upper: number | null) => formatHistogramRange(lower, upper, formatLatencyBoundary),
    },
    requestSizeDistribution: {
      items: response.request_size_distribution,
      color: readChartThemeColor('--td-brand-color', '#0052d9'),
      label: (lower: number, upper: number | null) => formatHistogramRange(lower, upper, formatByteBoundary),
    },
    responseSizeDistribution: {
      items: response.response_size_distribution,
      color: readChartThemeColor('--td-success-color-5', '#2ba471'),
      label: (lower: number, upper: number | null) => formatHistogramRange(lower, upper, formatByteBoundary),
    },
  };
  renderChartDefinitions(definitions, (definition, instance) => {
    instance.setOption({
      color: [definition.color],
      tooltip: {
        trigger: 'axis',
        backgroundColor: settingStore.chartColors.containerColor,
        borderColor: settingStore.chartColors.borderColor,
        textStyle: { color: settingStore.chartColors.textColor },
      },
      grid: { left: 44, right: 12, top: 12, bottom: 44 },
      xAxis: {
        type: 'category',
        data: definition.items.map((item) => definition.label(item.lower_bound, item.upper_bound)),
        axisLabel: { color: settingStore.chartColors.placeholderColor, rotate: 24 },
        axisLine: { lineStyle: { color: settingStore.chartColors.borderColor } },
      },
      yAxis: {
        type: 'value',
        minInterval: 1,
        axisLabel: { color: settingStore.chartColors.placeholderColor },
        splitLine: { lineStyle: { color: settingStore.chartColors.borderColor } },
      },
      series: [
        {
          type: 'bar',
          barMaxWidth: 36,
          data: definition.items.map((item) => item.sample_count),
          itemStyle: { color: definition.color },
        },
      ],
    });
  });
}

function renderChartDefinitions<T>(
  definitions: Partial<Record<ChartKey, T>>,
  render: (definition: T, instance: echarts.ECharts) => void,
) {
  (Object.keys(definitions) as ChartKey[]).forEach((key) => {
    const definition = definitions[key];
    const instance = resolveChartInstance(key);
    if (!definition || !instance) return;
    render(definition, instance);
  });
}

function resolveChartInstance(key: ChartKey) {
  const element = chartRefs.value[key];
  if (!element) {
    disposeChart(key);
    return null;
  }
  return getChartInstance(key, element);
}

function getChartInstance(key: ChartKey, element: HTMLDivElement) {
  const currentInstance = chartInstances.get(key);
  if (currentInstance && chartElements.get(key) === element) return currentInstance;

  currentInstance?.dispose();
  const nextInstance = echarts.init(element);
  chartInstances.set(key, nextInstance);
  chartElements.set(key, element);
  return nextInstance;
}

function disposeChart(key: ChartKey) {
  chartInstances.get(key)?.dispose();
  chartInstances.delete(key);
  chartElements.delete(key);
}

function formatHistogramRange(lower: number, upper: number | null, formatter: (value: number) => string) {
  return upper === null ? `${formatter(lower)}+` : `${formatter(lower)}-${formatter(upper)}`;
}
function formatLatencyBoundary(value: number) {
  return `${value}ms`;
}
function formatByteBoundary(value: number) {
  if (value < 1024 * 1024) return `${value / 1024}KiB`;
  return formatBytes(value);
}
function readChartThemeColor(token: string, fallback: string) {
  void settingStore.resolvedThemeTokensForDisplayMode;
  return getComputedStyle(document.documentElement).getPropertyValue(token).trim() || fallback;
}
function resizeCharts() {
  chartInstances.forEach((instance) => instance.resize());
}
watch([selectedRefreshInterval, autoRefreshEnabled, () => scheduler.allowPolling], scheduleRefresh);
watch(
  () => [settingStore.displayMode, settingStore.brandTheme, settingStore.resolvedThemeTokensForDisplayMode],
  () => renderCharts(),
  { deep: true },
);
onMounted(() => {
  isMounted = true;
  void refresh();
  window.addEventListener('resize', resizeCharts);
});
onUnmounted(() => {
  isMounted = false;
  if (refreshTimer !== null) window.clearInterval(refreshTimer);
  chartInstances.forEach((instance) => instance.dispose());
  chartInstances.clear();
  chartElements.clear();
  window.removeEventListener('resize', resizeCharts);
});
</script>
<style scoped lang="less">
.request-performance-page__trends,
.request-performance-page__routes,
.request-performance-page__instances,
.request-performance-page__distributions {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin-bottom: var(--graft-density-gap-16);
}

.request-performance-page__chart {
  height: 220px;
  width: 100%;
}

.request-performance-page__chart--compact {
  height: 180px;
}

.request-performance-page__route-table:not(.request-performance-page__route-table--empty) {
  height: 100%;

  :deep(.t-table__content) {
    height: 100%;
  }
}

.request-performance-page__route-table--empty {
  height: 100%;

  :deep(.t-table__content),
  :deep(.t-table table) {
    height: 100%;
  }

  :deep(.t-table__content) {
    overflow-x: hidden;
  }
}

.request-performance-page__chart-title,
.request-performance-page__status-group-heading {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-6);
}

.request-performance-page__chart-qualifier,
.request-performance-page__status-group-heading small {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.request-performance-page__chart-help {
  align-items: center;
  background: transparent;
  border: 0;
  color: var(--td-text-color-secondary);
  cursor: help;
  display: inline-flex;
  padding: 0;
}

.request-performance-page__chart-help:hover,
.request-performance-page__chart-help:focus-visible {
  color: var(--td-brand-color);
}

.request-performance-page__status-groups {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.request-performance-page__status-group {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-component-stroke);
  display: grid;
  gap: var(--graft-density-gap-4);
  padding: var(--graft-density-gap-16);
}

.request-performance-page__status-toggle {
  justify-content: flex-start;
  min-width: 0;
  padding: 0;
}

.request-performance-page__status-codes {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-6);
  margin-top: var(--graft-density-gap-4);
}

.request-performance-page__instance-table :deep(tbody tr) {
  cursor: pointer;
}

.request-performance-page__status-group strong {
  color: var(--td-text-color-primary);
}

.request-performance-page__status-group span {
  font: var(--td-font-title-medium);
}

.request-performance-page__status-group small {
  color: var(--td-text-color-secondary);
}

.request-performance-page__status-group-rate {
  margin-top: calc(-1 * var(--graft-density-gap-4));
}

@media (width <= 1200px) {
  .request-performance-page__trends,
  .request-performance-page__routes,
  .request-performance-page__instances,
  .request-performance-page__distributions {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 720px) {
  .request-performance-page__status-groups {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .request-performance-page__trends,
  .request-performance-page__routes,
  .request-performance-page__instances,
  .request-performance-page__distributions {
    grid-template-columns: 1fr;
  }
}
</style>
