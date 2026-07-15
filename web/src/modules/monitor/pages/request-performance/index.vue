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

      <section-card
        :title="t('monitor.requestPerformance.statusDistributionTitle')"
        :description="t('monitor.requestPerformance.statusDistributionHint')"
      >
        <div class="request-performance-page__status-groups">
          <div
            v-for="group in statusGroups"
            :key="group.status_group"
            class="request-performance-page__status-group"
            :data-status-group="group.status_group"
          >
            <div class="request-performance-page__status-group-heading">
              <strong>{{ group.label }}</strong>
              <small>{{ group.status_group }}</small>
            </div>
            <span>{{ formatCount(group.request_count) }}</span>
            <small class="request-performance-page__status-group-rate">{{ formatPercent(group.request_rate) }}</small>
          </div>
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
    </template>
    <t-empty v-else-if="!loading" :description="t('monitor.requestPerformance.empty')" />
  </server-status-page-shell>
</template>
<script setup lang="ts">
import { LineChart } from 'echarts/charts';
import { GridComponent, TooltipComponent } from 'echarts/components';
import * as echarts from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { InfoCircleIcon } from 'tdesign-icons-vue-next';
import type { PrimaryTableCol } from 'tdesign-vue-next';
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { RefreshControlBar, type RefreshControlOption, type RefreshControlStatus } from '@/shared/components/refresh';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { useRealtimeSchedulerStore, useSettingStore } from '@/store';

import { getRequestPerformance } from '../../api/request-performance';
import MonitorPageFeedback from '../../components/MonitorPageFeedback.vue';
import SectionCard from '../../components/SectionCard.vue';
import ServerStatusPageShell from '../../components/ServerStatusPageShell.vue';
import SummaryMetricCard from '../../components/SummaryMetricCard.vue';
import { useMonitorRefreshPreferences } from '../../composables/use-monitor-refresh-preferences';
import type { MonitorRefreshInterval } from '../../contract/refresh';
import { MONITOR_TREND_RANGE, type MonitorTrendRange } from '../../contract/trend';
import { formatChartTimeOnly } from '../../shared/time-display';
import type { RequestPerformanceResponse } from '../../types/request-performance';

echarts.use([GridComponent, TooltipComponent, LineChart, CanvasRenderer]);

defineOptions({ name: 'MonitorRequestPerformanceIndex' });

type ChartKey = 'traffic' | 'latency' | 'errors';
interface ChartDefinition {
  key: ChartKey;
  title: string;
  description: string;
  qualifier?: string;
  help?: string;
}

const MIN_PENDING_REFRESH_DISPLAY_MS = 500;

const { t } = useI18n();
const scheduler = useRealtimeSchedulerStore();
const settingStore = useSettingStore();
const { autoRefreshEnabled, refreshIntervalOptions, selectedRefreshInterval, toggleAutoRefresh } =
  useMonitorRefreshPreferences();
const selectedRange = ref<MonitorTrendRange>(MONITOR_TREND_RANGE.TEN_MINUTES);
const snapshot = ref<RequestPerformanceResponse | null>(null);
const loading = ref(false);
const errorMessage = ref('');
const remainingRefreshSeconds = ref<number | null>(null);
const chartRefs = ref<Partial<Record<ChartKey, HTMLDivElement | null>>>({});
const chartInstances = new Map<ChartKey, echarts.ECharts>();
let refreshTimer: number | null = null;
let refreshDeadline: number | null = null;
let pendingDisplayTimer: number | null = null;
let isMounted = false;

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
]);
const statusGroups = computed(() =>
  (snapshot.value?.status_groups ?? []).map((group) => ({
    ...group,
    label: statusGroupLabel(group.status_group),
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

function card(
  key: string,
  title: string,
  value: string,
  description: string,
): {
  key: string;
  title: string;
  value: string;
  description: string;
  status: 'healthy' | 'error';
  statusLabel: string;
} {
  return { key, title, value, description, status: performanceTone.value, statusLabel: performanceStatusLabel.value };
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
  if (loading.value) return;
  const pendingDisplayStartedAt = remainingRefreshSeconds.value === 0 ? Date.now() : null;
  loading.value = true;
  errorMessage.value = '';
  try {
    snapshot.value = await getRequestPerformance(selectedRange.value);
    await nextTick();
    renderCharts();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('monitor.requestPerformance.loadFailed'));
  } finally {
    loading.value = false;
    if (pendingDisplayStartedAt !== null) {
      await keepPendingStateVisible(pendingDisplayStartedAt);
    }
  }
  if (isMounted) {
    scheduleRefresh();
  }
}

async function keepPendingStateVisible(startedAt: number) {
  const remainingDelay = MIN_PENDING_REFRESH_DISPLAY_MS - (Date.now() - startedAt);
  if (remainingDelay <= 0) return;

  await new Promise<void>((resolve) => {
    pendingDisplayTimer = window.setTimeout(() => {
      pendingDisplayTimer = null;
      resolve();
    }, remainingDelay);
  });
}

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
  const definitions: Record<
    ChartKey,
    { name: string; values: number[]; color: string; value: (input: number) => string }
  > = {
    traffic: {
      name: t('monitor.requestPerformance.trafficSeries'),
      values: buckets.map((bucket) => bucket.requests_per_second),
      color: readChartThemeColor('--td-brand-color', '#0052d9'),
      value: (input) => `${input.toFixed(2)} RPS`,
    },
    latency: {
      name: t('monitor.requestPerformance.latencySeries'),
      values: buckets.map((bucket) => bucket.p95_latency_ms),
      color: readChartThemeColor('--td-warning-color-5', '#ebb105'),
      value: (input) => `${input.toFixed(0)} ms`,
    },
    errors: {
      name: t('monitor.requestPerformance.errorSeries'),
      values: buckets.map((bucket) => bucket.error_5xx_rate),
      color: readChartThemeColor('--td-error-color-5', '#e34d59'),
      value: (input) => `${input.toFixed(2)}%`,
    },
  };
  (Object.keys(definitions) as ChartKey[]).forEach((key) => {
    const element = chartRefs.value[key];
    if (!element) return;
    const instance = chartInstances.get(key) ?? echarts.init(element);
    chartInstances.set(key, instance);
    const definition = definitions[key];
    instance.setOption({
      color: [definition.color],
      tooltip: {
        trigger: 'axis',
        axisPointer: { lineStyle: { color: settingStore.chartColors.borderColor } },
        backgroundColor: settingStore.chartColors.containerColor,
        borderColor: settingStore.chartColors.borderColor,
        textStyle: { color: settingStore.chartColors.textColor },
        formatter: (params: Array<{ axisValueLabel?: string; color: string; data: number }>) => {
          const point = params[0];
          if (!point) return '';
          return `${point.axisValueLabel ?? ''}<br/><span style="color:${point.color}">●</span> ${definition.name}: <strong>${definition.value(point.data)}</strong>`;
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
      series: [
        {
          name: definition.name,
          type: 'line',
          smooth: true,
          showSymbol: false,
          lineStyle: { color: definition.color, width: 2 },
          itemStyle: { color: definition.color },
          emphasis: { focus: 'series' },
          areaStyle: { opacity: 0.1 },
          data: definition.values,
        },
      ],
    });
  });
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
  if (pendingDisplayTimer !== null) window.clearTimeout(pendingDisplayTimer);
  chartInstances.forEach((instance) => instance.dispose());
  window.removeEventListener('resize', resizeCharts);
});
</script>
<style scoped lang="less">
.request-performance-page__trends,
.request-performance-page__routes {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin-bottom: var(--graft-density-gap-16);
}

.request-performance-page__chart {
  height: 220px;
  width: 100%;
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
  .request-performance-page__routes {
    grid-template-columns: 1fr;
  }
}

@media (width <= 720px) {
  .request-performance-page__status-groups {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
