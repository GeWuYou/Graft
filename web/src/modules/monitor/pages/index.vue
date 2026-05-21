<template>
  <div
    class="monitor-dashboard"
    data-page-type="overview-dashboard"
    :data-status="overallStatus"
    :data-theme-mode="settingStore.displayMode"
  >
    <header class="dashboard-header">
      <div class="dashboard-header__main">
        <div class="dashboard-header__title-row">
          <div class="dashboard-header__copy">
            <p class="dashboard-header__section">{{ t('monitor.sectionTitle') }}</p>
            <h1 class="dashboard-header__title">{{ t('monitor.serverStatus.overviewTitle') }}</h1>
          </div>
          <t-tag class="dashboard-header__status" :theme="statusTheme(overallStatus)" variant="light">
            {{ overallStatusLabel(serverStatus?.status) }}
          </t-tag>
        </div>
        <p class="dashboard-header__hint">{{ t('monitor.serverStatus.overviewHint') }}</p>
        <div class="dashboard-header__meta">
          <span>{{ t('monitor.serverStatus.lastUpdated', { time: formatTimestamp(lastUpdatedAt) }) }}</span>
          <span>{{
            t('monitor.serverStatus.lastObserved', { time: formatTimestamp(serverStatus?.observed_at) })
          }}</span>
        </div>
      </div>
      <aside class="dashboard-actions" :data-status="refreshFeedbackTone">
        <div class="dashboard-actions__summary">
          <div class="dashboard-actions__item">
            <span class="dashboard-actions__label">{{ t('monitor.serverStatus.refreshIntervalLabel') }}</span>
            <strong class="dashboard-actions__value">{{ refreshIntervalLabel }}</strong>
          </div>
          <div class="dashboard-actions__item">
            <span class="dashboard-actions__label">{{ t('monitor.serverStatus.trendWindowLabel') }}</span>
            <strong class="dashboard-actions__value">{{ selectedTrendRangeLabel }}</strong>
          </div>
        </div>
        <div class="dashboard-actions__controls">
          <t-select
            v-model="refreshIntervalMode"
            class="dashboard-actions__select"
            :options="refreshIntervalOptions"
            size="small"
          />
          <t-button
            class="dashboard-actions__button"
            theme="primary"
            :loading="loading"
            @click="() => fetchServerStatus({ manual: true })"
          >
            <template #icon>
              <refresh-icon />
            </template>
            {{ t('monitor.serverStatus.refresh') }}
          </t-button>
        </div>
      </aside>
    </header>

    <section class="dashboard-feedback" :data-status="refreshFeedbackTone">
      <article class="dashboard-feedback__item">
        <span class="dashboard-feedback__label">{{ t('monitor.serverStatus.refreshStateLabel') }}</span>
        <strong class="dashboard-feedback__value">{{ refreshCountdownText }}</strong>
      </article>
      <article class="dashboard-feedback__item">
        <span class="dashboard-feedback__label">{{ t('monitor.serverStatus.observedAtLabel') }}</span>
        <strong class="dashboard-feedback__value">{{ formatTimestamp(serverStatus?.observed_at) }}</strong>
      </article>
    </section>

    <section class="summary-grid">
      <t-card
        v-for="card in summaryCards"
        :key="card.key"
        class="summary-card"
        :bordered="false"
        :data-status="card.status"
      >
        <div class="summary-card__icon-wrap">
          <component :is="card.icon" class="summary-card__icon" />
        </div>
        <div class="summary-card__content">
          <span class="summary-card__label">{{ card.label }}</span>
          <strong class="summary-card__value">{{ card.value }}</strong>
          <span class="summary-card__meta">{{ card.meta }}</span>
        </div>
      </t-card>
    </section>

    <t-row :gutter="[16, 16]">
      <t-col :xs="12" :xl="8">
        <t-card class="panel-card" :bordered="false" :title="t('monitor.serverStatus.trendCardTitle')">
          <template #actions>
            <t-radio-group v-model="selectedTrendRange" variant="default-filled" size="small">
              <t-radio-button v-for="option in trendRangeOptions" :key="option.value" :value="option.value">
                {{ option.label }}
              </t-radio-button>
            </t-radio-group>
          </template>
          <div v-if="hasTrendData" ref="trendChartRef" class="trend-chart" />
          <t-empty v-else :description="t('monitor.serverStatus.emptyTrend')" />
        </t-card>
      </t-col>
      <t-col :xs="12" :xl="4">
        <t-card class="panel-card" :bordered="false" :title="t('monitor.serverStatus.dependencyCardTitle')">
          <div v-if="serverStatus" class="dependency-list">
            <article
              v-for="dependency in dependencyItems"
              :key="dependency.key"
              class="dependency-item"
              :data-status="dependency.status"
            >
              <div class="dependency-item__main">
                <span class="dependency-item__icon-wrap">
                  <component :is="dependency.icon" class="dependency-item__icon" />
                </span>
                <div class="dependency-item__body">
                  <div class="dependency-item__title-row">
                    <span class="dependency-item__title">{{ dependency.label }}</span>
                    <t-tag :theme="statusTheme(dependency.status)" variant="light">
                      {{ statusLabel(dependency.status) }}
                    </t-tag>
                  </div>
                  <p class="dependency-item__detail">{{ dependency.detail }}</p>
                </div>
              </div>
              <span class="dependency-item__latency">{{ dependency.latency }}</span>
            </article>
          </div>
          <t-empty v-else :description="t('monitor.serverStatus.empty')" />
        </t-card>
      </t-col>
    </t-row>

    <section class="runtime-grid">
      <t-card
        v-for="section in runtimeSections"
        :key="section.key"
        class="panel-card runtime-card"
        :bordered="false"
        :title="section.title"
      >
        <template #actions>
          <component :is="section.icon" class="runtime-card__icon" />
        </template>
        <div v-if="serverStatus" class="runtime-card__grid">
          <div v-for="item in section.items" :key="item.key" class="runtime-card__item">
            <span class="runtime-card__label">{{ item.label }}</span>
            <strong class="runtime-card__value">{{ item.value }}</strong>
          </div>
        </div>
        <t-empty v-else :description="t('monitor.serverStatus.empty')" />
      </t-card>
    </section>
  </div>
</template>
<script setup lang="ts">
import { LineChart } from 'echarts/charts';
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components';
import type { EChartsCoreOption } from 'echarts/core';
import * as echarts from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import {
  AppIcon,
  CheckCircleIcon,
  CpuIcon,
  DataBaseIcon,
  ErrorCircleIcon,
  LinkIcon,
  PauseCircleIcon,
  RefreshIcon,
  ServerIcon,
  TimeIcon,
} from 'tdesign-icons-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import type { Component } from 'vue';
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import type { TChartColor } from '@/config/color';
import { useSettingStore } from '@/store';

import { getServerStatus } from '../api/server-status';
import { MONITOR_TREND_RANGE, type MonitorTrendRange } from '../contract/trend';
import type { ServerStatusDependency, ServerStatusResponse, ServerStatusTrendPoint } from '../types/server-status';

defineOptions({
  name: 'MonitorServerStatusOverviewIndex',
});

echarts.use([TooltipComponent, LegendComponent, GridComponent, LineChart, CanvasRenderer]);

type MonitorStatus = 'healthy' | 'degraded' | 'disabled' | 'unknown';
type RefreshIntervalMode = 'manual' | '5' | '10' | '30' | '60' | '300';
type TrendRange = MonitorTrendRange;

interface SummaryCard {
  key: string;
  label: string;
  value: string;
  meta: string;
  status: MonitorStatus;
  icon: Component;
}

interface RuntimeSection {
  key: string;
  title: string;
  icon: Component;
  items: Array<{
    key: string;
    label: string;
    value: string;
  }>;
}

const { t, locale } = useI18n();
const settingStore = useSettingStore();
const loading = ref(false);
const serverStatus = ref<ServerStatusResponse | null>(null);
const trendChartRef = ref<HTMLDivElement | null>(null);
const refreshIntervalMode = ref<RefreshIntervalMode>('5');
const selectedTrendRange = ref<TrendRange>(MONITOR_TREND_RANGE.TEN_MINUTES);
const lastUpdatedAt = ref<string | null>(null);
const consecutiveFailures = ref(0);
const remainingRefreshSeconds = ref<number | null>(null);
const isPageVisible = ref(typeof document === 'undefined' ? true : document.visibilityState === 'visible');

let trendChart: echarts.ECharts | null = null;
let refreshTickTimer: number | null = null;
let nextRefreshAt: number | null = null;
let pendingTrendRange: TrendRange | null = null;

const trendRangeOptions = computed(() => [
  { label: t('monitor.serverStatus.trendRange10Minutes'), value: MONITOR_TREND_RANGE.TEN_MINUTES },
  { label: t('monitor.serverStatus.trendRange30Minutes'), value: MONITOR_TREND_RANGE.THIRTY_MINUTES },
  { label: t('monitor.serverStatus.trendRange1Hour'), value: MONITOR_TREND_RANGE.ONE_HOUR },
]);

const refreshIntervalOptions = computed(() => [
  { label: t('monitor.serverStatus.refreshIntervalManual'), value: 'manual' },
  { label: t('monitor.serverStatus.refreshInterval5Seconds'), value: '5' },
  { label: t('monitor.serverStatus.refreshInterval10Seconds'), value: '10' },
  { label: t('monitor.serverStatus.refreshInterval30Seconds'), value: '30' },
  { label: t('monitor.serverStatus.refreshInterval1Minute'), value: '60' },
  { label: t('monitor.serverStatus.refreshInterval5Minutes'), value: '300' },
]);

const currentRefreshSeconds = computed<number | null>(() => {
  if (refreshIntervalMode.value === 'manual') {
    return null;
  }

  return Number(refreshIntervalMode.value);
});

const refreshIntervalLabel = computed(() => {
  const seconds = currentRefreshSeconds.value;
  return seconds === null ? t('monitor.serverStatus.refreshIntervalManual') : formatDuration(seconds);
});

const refreshCountdownText = computed(() => {
  if (refreshIntervalMode.value === 'manual') {
    return t('monitor.serverStatus.nextRefreshManual');
  }
  if (!isPageVisible.value) {
    return t('monitor.serverStatus.nextRefreshPaused');
  }
  if (remainingRefreshSeconds.value === null) {
    return t('monitor.serverStatus.nextRefreshPending');
  }
  if (consecutiveFailures.value > 0) {
    return t('monitor.serverStatus.nextRefreshRetryIn', {
      seconds: String(remainingRefreshSeconds.value),
      interval: refreshIntervalLabel.value,
    });
  }

  return t('monitor.serverStatus.nextRefreshIn', {
    seconds: String(remainingRefreshSeconds.value),
  });
});

const trendPoints = computed<ServerStatusTrendPoint[]>(() => serverStatus.value?.trend.points ?? []);

const hasTrendData = computed(() => trendPoints.value.length >= 2);
const overallStatus = computed<MonitorStatus>(() => normalizeStatus(serverStatus.value?.status));
const selectedTrendRangeLabel = computed(() => {
  return trendRangeOptions.value.find((option) => option.value === selectedTrendRange.value)?.label ?? '--';
});
const refreshFeedbackTone = computed<MonitorStatus>(() => {
  if (consecutiveFailures.value > 0 || !isPageVisible.value) {
    return 'degraded';
  }

  if (refreshIntervalMode.value === 'manual') {
    return 'disabled';
  }

  return overallStatus.value;
});

const pluginCounts = computed(() => {
  const rows = serverStatus.value?.plugins ?? [];
  let healthy = 0;
  let abnormal = 0;
  let unreported = 0;

  rows.forEach((plugin) => {
    if (plugin.status === 'healthy') {
      healthy += 1;
      return;
    }
    if (plugin.status === 'degraded' || plugin.status === 'disabled') {
      abnormal += 1;
      return;
    }
    unreported += 1;
  });

  return {
    total: rows.length,
    healthy,
    abnormal,
    unreported,
  };
});

const summaryCards = computed<SummaryCard[]>(() => {
  const response = serverStatus.value;
  if (!response) {
    return [
      emptyMetricCard('overall', ErrorCircleIcon),
      emptyMetricCard('dependencies', DataBaseIcon),
      emptyMetricCard('plugins', AppIcon),
      emptyMetricCard('memory', CpuIcon),
    ];
  }

  const pluginCardStatus = derivePluginCardStatus(pluginCounts.value);
  const pluginCardMeta =
    pluginCounts.value.total > 0 && pluginCounts.value.unreported === pluginCounts.value.total
      ? t('monitor.serverStatus.summaryPluginsNoMetrics')
      : t('monitor.serverStatus.summaryPluginsMeta', {
          healthy: String(pluginCounts.value.healthy),
          abnormal: String(pluginCounts.value.abnormal),
          unreported: String(pluginCounts.value.unreported),
        });

  return [
    {
      key: 'overall',
      label: t('monitor.serverStatus.statusLabel'),
      value: overallStatusLabel(response.status),
      meta: t('monitor.serverStatus.lastObserved', { time: formatTimestamp(response.observed_at) }),
      status: normalizeStatus(response.status),
      icon: overallStatusIcon(response.status),
    },
    {
      key: 'dependencies',
      label: t('monitor.serverStatus.summaryDependencies'),
      value: t('monitor.serverStatus.summaryDependenciesValue', {
        healthy: String(response.summary.healthy_dependencies),
        total: String(response.summary.total_dependencies),
      }),
      meta: t('monitor.serverStatus.summaryDependenciesMeta', {
        degraded: String(response.summary.degraded_dependencies),
        disabled: String(response.summary.disabled_dependencies),
      }),
      status: dependencySummaryStatus(response),
      icon: DataBaseIcon,
    },
    {
      key: 'plugins',
      label: t('monitor.serverStatus.summaryPlugins'),
      value: t('monitor.serverStatus.summaryPluginsValue', {
        total: String(pluginCounts.value.total),
      }),
      meta: pluginCardMeta,
      status: pluginCardStatus,
      icon: AppIcon,
    },
    {
      key: 'memory',
      label: t('monitor.serverStatus.summaryMemory'),
      value: formatBytes(response.runtime.alloc_bytes),
      meta: t('monitor.serverStatus.summaryMemoryMeta', {
        goroutines: String(response.runtime.goroutines),
        gc: String(response.runtime.gc_cycles),
      }),
      status: normalizeStatus(response.status),
      icon: CpuIcon,
    },
  ];
});

const dependencyItems = computed(() => {
  const response = serverStatus.value;
  if (!response) {
    return [];
  }

  return [
    buildDependencyItem(
      'database',
      t('monitor.serverStatus.databaseLabel'),
      response.dependencies.database,
      DataBaseIcon,
    ),
    buildDependencyItem('redis', t('monitor.serverStatus.redisLabel'), response.dependencies.redis, LinkIcon),
  ];
});

const runtimeSections = computed<RuntimeSection[]>(() => {
  const response = serverStatus.value;
  if (!response) {
    return [
      {
        key: 'basic',
        title: t('monitor.serverStatus.runtimeGroupBasic'),
        icon: ServerIcon,
        items: [],
      },
      {
        key: 'runtime',
        title: t('monitor.serverStatus.runtimeGroupRuntime'),
        icon: TimeIcon,
        items: [],
      },
      {
        key: 'environment',
        title: t('monitor.serverStatus.runtimeGroupEnvironment'),
        icon: CpuIcon,
        items: [],
      },
    ];
  }

  return [
    {
      key: 'basic',
      title: t('monitor.serverStatus.runtimeGroupBasic'),
      icon: ServerIcon,
      items: [
        { key: 'version', label: t('monitor.serverStatus.versionLabel'), value: response.server.version || '-' },
        { key: 'app', label: t('monitor.serverStatus.appLabel'), value: response.server.app_name || '-' },
        {
          key: 'startedAt',
          label: t('monitor.serverStatus.startedAtLabel'),
          value: formatTimestamp(response.server.started_at),
        },
        {
          key: 'observedAt',
          label: t('monitor.serverStatus.observedAtLabel'),
          value: formatTimestamp(response.observed_at),
        },
      ],
    },
    {
      key: 'runtime',
      title: t('monitor.serverStatus.runtimeGroupRuntime'),
      icon: TimeIcon,
      items: [
        {
          key: 'uptime',
          label: t('monitor.serverStatus.uptimeLabel'),
          value: formatUptime(response.server.uptime_seconds),
        },
        {
          key: 'goroutines',
          label: t('monitor.serverStatus.goroutinesLabel'),
          value: t('monitor.serverStatus.goroutinesValue', {
            count: String(response.runtime.goroutines),
          }),
        },
        {
          key: 'heap',
          label: t('monitor.serverStatus.heapLabel'),
          value: formatBytes(response.runtime.heap_in_use_bytes),
        },
        {
          key: 'gc',
          label: t('monitor.serverStatus.gcLabel'),
          value: t('monitor.serverStatus.gcValue', {
            count: String(response.runtime.gc_cycles),
          }),
        },
      ],
    },
    {
      key: 'environment',
      title: t('monitor.serverStatus.runtimeGroupEnvironment'),
      icon: CpuIcon,
      items: [
        { key: 'goVersion', label: t('monitor.serverStatus.goVersionLabel'), value: response.runtime.go_version },
        { key: 'host', label: t('monitor.serverStatus.hostLabel'), value: response.runtime.host_name || '-' },
        {
          key: 'platform',
          label: t('monitor.serverStatus.platformLabel'),
          value: `${response.runtime.operating_system}/${response.runtime.architecture}`,
        },
        {
          key: 'cpu',
          label: t('monitor.serverStatus.cpuLabel'),
          value: t('monitor.serverStatus.cpuValue', { count: String(response.runtime.cpu_cores) }),
        },
      ],
    },
  ];
});

async function fetchServerStatus(options: { manual?: boolean } = {}) {
  const requestedTrendRange = selectedTrendRange.value;
  if (loading.value) {
    pendingTrendRange = requestedTrendRange;
    return;
  }

  let shouldRefetch = false;
  pendingTrendRange = null;
  stopRefreshTick();
  loading.value = true;
  try {
    serverStatus.value = await getServerStatus(requestedTrendRange);
    lastUpdatedAt.value = new Date().toISOString();
    consecutiveFailures.value = 0;
  } catch (error) {
    const previousFailures = consecutiveFailures.value;
    consecutiveFailures.value += 1;
    if (options.manual || previousFailures === 0) {
      const fallbackMessage = t('monitor.serverStatus.loadFailed');
      const message = error instanceof Error && error.message.trim() ? error.message : fallbackMessage;
      MessagePlugin.error(message);
    }
  } finally {
    loading.value = false;
    if (pendingTrendRange && pendingTrendRange !== requestedTrendRange) {
      shouldRefetch = true;
    } else {
      scheduleNextRefresh();
    }
  }

  if (shouldRefetch) {
    void fetchServerStatus();
  }
}

function scheduleNextRefresh() {
  stopRefreshTick();
  if (refreshIntervalMode.value === 'manual' || !isPageVisible.value) {
    remainingRefreshSeconds.value = null;
    return;
  }

  const intervalSeconds = currentRefreshSeconds.value;
  if (intervalSeconds === null) {
    remainingRefreshSeconds.value = null;
    return;
  }

  const backoffMultiplier = consecutiveFailures.value > 0 ? 2 ** consecutiveFailures.value : 1;
  const delaySeconds = Math.min(intervalSeconds * backoffMultiplier, 5 * 60);
  nextRefreshAt = Date.now() + delaySeconds * 1000;
  updateRemainingRefreshSeconds();
  refreshTickTimer = window.setInterval(() => {
    updateRemainingRefreshSeconds();
    if (remainingRefreshSeconds.value === 0) {
      void fetchServerStatus();
    }
  }, 1000);
}

function updateRemainingRefreshSeconds() {
  if (nextRefreshAt === null) {
    remainingRefreshSeconds.value = null;
    return;
  }

  const diffSeconds = Math.max(0, Math.ceil((nextRefreshAt - Date.now()) / 1000));
  remainingRefreshSeconds.value = diffSeconds;
}

function stopRefreshTick() {
  if (refreshTickTimer !== null) {
    window.clearInterval(refreshTickTimer);
    refreshTickTimer = null;
  }
  nextRefreshAt = null;
}

function handleVisibilityChange() {
  isPageVisible.value = document.visibilityState === 'visible';
  if (isPageVisible.value) {
    void fetchServerStatus();
    return;
  }

  stopRefreshTick();
  remainingRefreshSeconds.value = null;
}

function normalizeStatus(status?: string): MonitorStatus {
  switch (status) {
    case 'healthy':
    case 'degraded':
    case 'disabled':
      return status;
    default:
      return 'unknown';
  }
}

function derivePluginCardStatus(counts: {
  healthy: number;
  abnormal: number;
  unreported: number;
  total: number;
}): MonitorStatus {
  if (counts.abnormal > 0) {
    return 'degraded';
  }
  if (counts.unreported === counts.total && counts.total > 0) {
    return 'unknown';
  }
  if (counts.healthy > 0 && counts.abnormal === 0 && counts.unreported === 0) {
    return 'healthy';
  }

  return 'unknown';
}

function emptyMetricCard(key: string, icon: Component): SummaryCard {
  return {
    key,
    label: t(`monitor.serverStatus.emptyMetric.${key}`),
    value: '--',
    meta: t('monitor.serverStatus.emptyMetric.meta'),
    status: 'unknown',
    icon,
  };
}

function dependencySummaryStatus(response: ServerStatusResponse): MonitorStatus {
  if (response.summary.degraded_dependencies > 0) {
    return 'degraded';
  }
  if (response.summary.healthy_dependencies > 0) {
    return 'healthy';
  }
  if (response.summary.unknown_dependencies > 0) {
    return 'unknown';
  }

  return 'disabled';
}

function buildDependencyItem(key: string, label: string, dependency: ServerStatusDependency, icon: Component) {
  return {
    key,
    label,
    detail: dependency.detail,
    status: normalizeStatus(dependency.status),
    latency: formatLatency(dependency.latency_ms),
    icon,
  };
}

function ensureTrendChart() {
  if (!trendChartRef.value) {
    return null;
  }

  if (!trendChart) {
    trendChart = echarts.init(trendChartRef.value);
  }

  return trendChart;
}

function syncTrendChart() {
  if (!hasTrendData.value) {
    disposeTrendChart();
    return;
  }

  const chart = ensureTrendChart();
  if (!chart) {
    return;
  }

  chart.setOption(buildTrendChartOption(trendPoints.value, settingStore.chartColors), true);
  resizeTrendChart();
}

function resizeTrendChart() {
  trendChart?.resize({
    width: trendChartRef.value?.clientWidth,
    height: trendChartRef.value?.clientHeight,
  });
}

function disposeTrendChart() {
  trendChart?.dispose();
  trendChart = null;
}

function buildTrendChartOption(points: ServerStatusTrendPoint[], chartColors: TChartColor): EChartsCoreOption {
  const seriesColors = [
    readThemeToken('--td-brand-color', '#0052D9'),
    readThemeToken('--td-success-color-5', '#00A870'),
    readThemeToken('--td-warning-color-5', '#ED7B2F'),
  ];
  const buildTrendSeriesInteraction = () => ({
    areaStyle: {
      opacity: 0,
    },
    emphasis: {
      focus: 'series' as const,
      areaStyle: {
        opacity: 0.14,
      },
    },
  });

  return {
    color: seriesColors,
    tooltip: {
      trigger: 'axis',
      backgroundColor: chartColors.containerColor,
      borderColor: chartColors.borderColor,
      textStyle: {
        color: chartColors.textColor,
      },
    },
    legend: {
      bottom: 0,
      icon: 'roundRect',
      itemHeight: 8,
      itemWidth: 14,
      textStyle: {
        color: chartColors.placeholderColor,
      },
    },
    grid: {
      left: '12px',
      right: '18px',
      top: '20px',
      bottom: '48px',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: points.map((point) => formatChartTimestamp(point.observed_at)),
      axisLabel: {
        color: chartColors.placeholderColor,
      },
      axisLine: {
        lineStyle: {
          color: chartColors.borderColor,
        },
      },
      axisTick: {
        show: false,
      },
    },
    yAxis: [
      {
        type: 'value',
        name: t('monitor.serverStatus.chartCpu'),
        min: 0,
        axisLabel: {
          color: chartColors.placeholderColor,
          formatter: (value: number) => `${value}%`,
        },
        splitLine: {
          lineStyle: {
            color: chartColors.borderColor,
          },
        },
      },
      {
        type: 'value',
        name: t('monitor.serverStatus.chartMemory'),
        axisLabel: {
          color: chartColors.placeholderColor,
          formatter: (value: number) => `${value} MB`,
        },
        splitLine: {
          show: false,
        },
      },
    ],
    series: [
      {
        name: t('monitor.serverStatus.chartCpu'),
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 8,
        ...buildTrendSeriesInteraction(),
        data: points.map((point) => Number(point.cpu_percent.toFixed(2))),
      },
      {
        name: t('monitor.serverStatus.chartMemory'),
        type: 'line',
        smooth: true,
        yAxisIndex: 1,
        symbol: 'circle',
        symbolSize: 8,
        ...buildTrendSeriesInteraction(),
        data: points.map((point) => bytesToMegabytes(point.alloc_bytes)),
      },
      {
        name: t('monitor.serverStatus.chartGoroutines'),
        type: 'line',
        smooth: true,
        yAxisIndex: 1,
        symbol: 'circle',
        symbolSize: 8,
        ...buildTrendSeriesInteraction(),
        data: points.map((point) => point.goroutines),
      },
    ],
  };
}

function readThemeToken(token: string, fallback: string) {
  const value = getComputedStyle(document.documentElement).getPropertyValue(token).trim();
  return value || fallback;
}

function overallStatusIcon(status?: string) {
  switch (status) {
    case 'healthy':
      return CheckCircleIcon;
    case 'degraded':
      return PauseCircleIcon;
    default:
      return ErrorCircleIcon;
  }
}

function overallStatusLabel(status?: string) {
  switch (status) {
    case 'healthy':
      return t('monitor.serverStatus.statusHealthy');
    case 'degraded':
      return t('monitor.serverStatus.statusDegraded');
    default:
      return t('monitor.serverStatus.statusAbnormal');
  }
}

function formatUptime(totalSeconds: number) {
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  return `${hours}h ${minutes}m ${seconds}s`;
}

function formatTimestamp(value?: string | null) {
  if (!value) {
    return '--';
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'medium',
    timeStyle: 'medium',
  }).format(date);
}

function formatChartTimestamp(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(locale.value, {
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
}

function formatBytes(bytes: number) {
  if (!bytes) {
    return '0 B';
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let value = bytes;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  return `${value.toFixed(value >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}

function bytesToMegabytes(bytes: number) {
  return Number((bytes / 1024 / 1024).toFixed(2));
}

function formatDuration(totalSeconds: number) {
  const isZhLocale = locale.value.toLowerCase().startsWith('zh');
  if (!totalSeconds) {
    return '--';
  }
  if (totalSeconds < 60) {
    return isZhLocale ? `${totalSeconds}秒` : `${totalSeconds} sec`;
  }
  if (totalSeconds % 60 === 0) {
    return isZhLocale ? `${totalSeconds / 60}分钟` : `${totalSeconds / 60} min`;
  }

  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return isZhLocale ? `${minutes}分 ${seconds}秒` : `${minutes} min ${seconds} sec`;
}

function formatLatency(latency: number | null) {
  if (latency === null) {
    return t('monitor.serverStatus.noLatency');
  }

  return t('monitor.serverStatus.latencyValue', { value: String(latency) });
}

function statusLabel(status?: string) {
  switch (status) {
    case 'healthy':
      return t('monitor.serverStatus.statusHealthy');
    case 'degraded':
      return t('monitor.serverStatus.statusDegraded');
    case 'disabled':
      return t('monitor.serverStatus.statusDisabled');
    default:
      return t('monitor.serverStatus.statusUnknown');
  }
}

function statusTheme(status?: string) {
  switch (status) {
    case 'healthy':
      return 'success';
    case 'degraded':
      return 'warning';
    case 'disabled':
      return 'default';
    default:
      return 'danger';
  }
}

watch(
  [
    () => trendPoints.value,
    () => settingStore.brandTheme,
    () => settingStore.chartColors.textColor,
    () => settingStore.chartColors.placeholderColor,
    () => settingStore.chartColors.borderColor,
    () => locale.value,
  ],
  async () => {
    await nextTick();
    syncTrendChart();
  },
  { deep: true },
);

watch(refreshIntervalMode, () => {
  scheduleNextRefresh();
});

watch(selectedTrendRange, async (nextRange, previousRange) => {
  if (nextRange === previousRange) {
    return;
  }

  await fetchServerStatus();
});

onMounted(async () => {
  await fetchServerStatus();
  await nextTick();
  syncTrendChart();
  window.addEventListener('resize', resizeTrendChart, false);
  document.addEventListener('visibilitychange', handleVisibilityChange, false);
});

onUnmounted(() => {
  stopRefreshTick();
  window.removeEventListener('resize', resizeTrendChart);
  document.removeEventListener('visibilitychange', handleVisibilityChange);
  disposeTrendChart();
});
</script>
<style lang="less" scoped>
@import './index.less';
</style>
