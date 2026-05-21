<template>
  <server-status-page-shell
    class="monitor-dashboard"
    :eyebrow="t('monitor.sectionTitle')"
    :title="t('monitor.serverStatus.overviewTitle')"
    :description="t('monitor.serverStatus.overviewHint')"
  >
    <template #toolbar>
      <monitor-toolbar
        :auto-refresh-enabled="autoRefreshEnabled"
        :loading="loading"
        :pause-auto-refresh-label="t('monitor.serverStatus.pauseRefresh')"
        :refresh-interval-label="t('monitor.serverStatus.refreshIntervalLabel')"
        :refresh-interval-options="refreshIntervalOptions"
        :refresh-interval-value="selectedRefreshInterval"
        :refresh-now-label="t('monitor.serverStatus.refreshNow')"
        :resume-auto-refresh-label="t('monitor.serverStatus.resumeRefresh')"
        :show-trend-range="true"
        :status="toolbarStatus"
        :status-label="overallStatusLabel(overallStatus)"
        :trend-range-label="t('monitor.serverStatus.trendWindowLabel')"
        :trend-range-options="trendRangeOptions"
        :trend-range-value="selectedTrendRange"
        @refresh="() => fetchServerStatus({ manual: true })"
        @toggle-auto-refresh="toggleAutoRefresh"
        @update:refresh-interval-value="handleRefreshIntervalChange"
        @update:trend-range-value="handleTrendRangeChange"
      />
    </template>

    <template #summary>
      <summary-metric-card
        v-for="card in metricCards"
        :key="card.key"
        :data-card-key="card.key"
        :title="card.label"
        :value="card.value"
        :value-aside="card.valueSide"
        :description="`${card.meta} · ${card.description}`"
        :status="metricToneToServerStatusTone(card.tone)"
        :status-label="card.statusLabel"
      />
    </template>

    <div class="server-status-overview-layout">
      <section-card
        class="server-status-overview-layout__trend"
        :title="t('monitor.serverStatus.trendCardTitle')"
        :description="refreshCountdownText"
        :min-height="520"
      >
        <template #actions>
          <div class="trend-panel__actions">
            <t-radio-group v-model="selectedTrendMode" variant="default-filled" size="small">
              <t-radio-button v-for="option in trendModeOptions" :key="option.value" :value="option.value">
                {{ option.label }}
              </t-radio-button>
            </t-radio-group>
          </div>
        </template>

        <div class="trend-panel__shell" :data-mode="selectedTrendMode">
          <div class="trend-panel__summary-bar">
            <div class="trend-panel__summary-copy">
              <span class="trend-panel__summary-title">{{ t('monitor.serverStatus.trendMetricInventory') }}</span>
              <p class="trend-panel__summary-text">
                {{
                  t('monitor.serverStatus.trendMetricInventoryValue', {
                    count: String(visibleTrendMetricCount),
                    groups: trendGroupSummaryLabel,
                  })
                }}
              </p>
            </div>
            <div
              v-if="selectedTrendMode === 'focus'"
              class="trend-panel__focus-toolbar"
              data-trend-focus-toolbar="true"
            >
              <div class="trend-panel__focus-toolbar-copy">
                <span class="trend-panel__focus-label">{{ t('monitor.serverStatus.focusMetricLabel') }}</span>
                <span class="trend-panel__focus-group">{{ currentFocusMetric?.groupLabel }}</span>
              </div>
              <t-select
                v-model="selectedFocusMetric"
                class="trend-panel__focus-select"
                :options="focusMetricOptions"
                size="small"
                data-trend-focus-select="true"
              />
            </div>
          </div>

          <t-empty v-if="!hasTrendData" :description="t('monitor.serverStatus.emptyTrend')" />

          <transition v-else name="trend-mode-fade" mode="out-in">
            <div
              v-if="selectedTrendMode === 'overview'"
              key="overview"
              class="trend-panel__body trend-panel__body--overview"
              data-trend-mode-panel="overview"
            >
              <article
                v-for="section in overviewTrendSections"
                :key="section.key"
                class="trend-overview-section"
                :data-trend-overview-section="section.key"
              >
                <header class="trend-section-header">
                  <div class="trend-section-header__copy">
                    <div class="trend-section-header__title-row">
                      <h3 class="trend-section-header__title">{{ section.title }}</h3>
                      <t-popup v-if="section.infoText" expand-animation placement="top" show-arrow trigger="click">
                        <template #content>
                          <div class="trend-info-popup">{{ section.infoText }}</div>
                        </template>
                        <button
                          type="button"
                          class="trend-info-trigger"
                          :aria-label="`${section.title}${t('monitor.serverStatus.infoActionLabel')}`"
                        >
                          <info-circle-icon class="trend-info-trigger__icon" />
                        </button>
                      </t-popup>
                    </div>
                  </div>
                  <div v-if="section.helperText" class="trend-section-header__helper">
                    {{ section.helperText }}
                  </div>
                </header>
                <div class="trend-section-legend" :data-trend-legend-group="section.key">
                  <span
                    v-for="metric in section.metrics"
                    :key="metric.key"
                    class="trend-legend-item"
                    data-trend-legend-item="true"
                  >
                    <i class="trend-legend-item__dot" :style="{ backgroundColor: metric.color() }" />
                    <span class="trend-legend-item__text">{{ metric.shortLabel }}</span>
                    <strong class="trend-legend-item__value">{{ metric.currentValue }}</strong>
                  </span>
                </div>
                <div
                  :ref="(el) => setTrendChartRef(section.chartKey, el)"
                  class="trend-chart trend-chart--overview"
                  :data-trend-chart="section.chartKey"
                />
              </article>

              <article class="trend-runtime-summary" data-trend-overview-section="runtimeSummary">
                <header class="trend-section-header">
                  <div class="trend-section-header__copy">
                    <h3 class="trend-section-header__title">{{ t('monitor.serverStatus.runtimeSummaryTitle') }}</h3>
                  </div>
                </header>
                <div class="trend-runtime-summary__grid">
                  <article
                    v-for="metric in runtimeSummaryMetrics"
                    :key="metric.key"
                    class="trend-runtime-summary__item"
                    :data-runtime-summary-item="metric.key"
                  >
                    <span class="trend-runtime-summary__label">{{ metric.shortLabel }}</span>
                    <strong class="trend-runtime-summary__value">{{ metric.currentValue }}</strong>
                  </article>
                </div>
              </article>
            </div>

            <transition-group
              v-else-if="selectedTrendMode === 'multi'"
              key="multi"
              name="trend-metric-fade"
              tag="div"
              class="trend-panel__body trend-panel__body--multi trend-small-grid"
              data-trend-mode-panel="multi"
            >
              <article
                v-for="metric in smallMultipleMetrics"
                :key="metric.key"
                class="trend-small-card"
                :data-trend-small-card="metric.key"
              >
                <header class="trend-small-card__header">
                  <div class="trend-small-card__copy">
                    <div class="trend-small-card__title-row">
                      <h3 class="trend-small-card__title">{{ metric.label }}</h3>
                      <t-popup v-if="metric.infoText" expand-animation placement="top" show-arrow trigger="click">
                        <template #content>
                          <div class="trend-info-popup">{{ metric.infoText }}</div>
                        </template>
                        <button
                          type="button"
                          class="trend-info-trigger"
                          :aria-label="`${metric.label}${t('monitor.serverStatus.infoActionLabel')}`"
                        >
                          <info-circle-icon class="trend-info-trigger__icon" />
                        </button>
                      </t-popup>
                    </div>
                  </div>
                  <div class="trend-small-card__meta">
                    <span class="trend-small-card__meta-label">{{ t('monitor.serverStatus.currentValue') }}</span>
                    <strong class="trend-small-card__meta-value">{{ metric.currentValue }}</strong>
                    <span class="trend-small-card__meta-unit">
                      {{ t('monitor.serverStatus.unitLabel') }} {{ metric.unit }}
                    </span>
                  </div>
                </header>
                <div
                  :ref="(el) => setTrendChartRef(metric.chartKey, el)"
                  class="trend-chart trend-chart--small"
                  :data-trend-chart="metric.chartKey"
                />
                <footer class="trend-small-card__footer">
                  <span class="trend-legend-item" data-trend-legend-item="true">
                    <i class="trend-legend-item__dot" :style="{ backgroundColor: metric.color() }" />
                    <span class="trend-legend-item__text">{{ metric.shortLabel }}</span>
                  </span>
                  <span v-if="metric.helperText" class="trend-section-header__helper">
                    {{ metric.helperText }}
                  </span>
                </footer>
              </article>
            </transition-group>

            <div
              v-else
              key="focus"
              class="trend-panel__body trend-panel__body--focus trend-focus-panel"
              :data-trend-mode-panel="selectedTrendMode"
              :data-trend-focus-metric="currentFocusMetric?.key"
            >
              <header class="trend-focus-panel__header">
                <div class="trend-focus-panel__copy">
                  <div class="trend-focus-panel__title-row">
                    <h3 class="trend-focus-panel__title">{{ currentFocusMetric?.label }}</h3>
                    <t-popup
                      v-if="currentFocusMetric?.infoText"
                      expand-animation
                      placement="top"
                      show-arrow
                      trigger="click"
                    >
                      <template #content>
                        <div class="trend-info-popup">{{ currentFocusMetric?.infoText }}</div>
                      </template>
                      <button
                        type="button"
                        class="trend-info-trigger"
                        :aria-label="`${currentFocusMetric?.label ?? ''}${t('monitor.serverStatus.infoActionLabel')}`"
                      >
                        <info-circle-icon class="trend-info-trigger__icon" />
                      </button>
                    </t-popup>
                    <span class="trend-focus-panel__group">{{ currentFocusMetric?.groupLabel }}</span>
                  </div>
                </div>
                <div class="trend-focus-panel__meta">
                  <span class="trend-focus-panel__meta-label">{{ t('monitor.serverStatus.currentValue') }}</span>
                  <strong class="trend-focus-panel__meta-value">{{ currentFocusMetric?.currentValue }}</strong>
                  <span class="trend-focus-panel__meta-unit">
                    {{ t('monitor.serverStatus.unitLabel') }} {{ currentFocusMetric?.unit }}
                  </span>
                </div>
              </header>
              <div class="trend-section-legend" data-trend-legend-group="focus">
                <span class="trend-legend-item" data-trend-legend-item="true">
                  <i class="trend-legend-item__dot" :style="{ backgroundColor: currentFocusMetric?.color() }" />
                  <span class="trend-legend-item__text">{{ currentFocusMetric?.label }}</span>
                </span>
                <span v-if="focusReferenceText" class="trend-section-header__helper">
                  {{ focusReferenceText }}
                </span>
              </div>
              <div
                :ref="(el) => setTrendChartRef('focus', el)"
                class="trend-chart trend-chart--focus"
                data-trend-chart="focus"
              />
            </div>
          </transition>
        </div>
      </section-card>

      <section-card
        class="server-status-overview-layout__status"
        :title="t('monitor.serverStatus.runtimeStatusTitle')"
        :description="t('monitor.serverStatus.runtimeStatusSubtitle')"
        :min-height="520"
      >
        <div v-if="serverStatus" class="status-sidebar__content">
          <section class="status-sidebar__section" data-status-sidebar-group="dependencies">
            <header class="status-sidebar__section-header">
              <h3 class="status-sidebar__section-title">
                {{ t('monitor.serverStatus.runtimeStatusDependenciesTitle') }}
              </h3>
            </header>
            <div class="dependency-list">
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
          </section>

          <section class="status-sidebar__section" data-status-sidebar-group="process">
            <header class="status-sidebar__section-header">
              <h3 class="status-sidebar__section-title">{{ t('monitor.serverStatus.runtimeStatusProcessTitle') }}</h3>
            </header>
            <dl class="status-sidebar__summary-list">
              <div
                v-for="item in processSummaryItems"
                :key="item.key"
                class="status-sidebar__summary-item"
                :data-status-sidebar-item="item.key"
              >
                <dt class="status-sidebar__summary-key">{{ item.label }}</dt>
                <dd class="status-sidebar__summary-value">{{ item.value }}</dd>
              </div>
            </dl>
          </section>

          <section class="status-sidebar__section" data-status-sidebar-group="sampling">
            <header class="status-sidebar__section-header">
              <h3 class="status-sidebar__section-title">{{ t('monitor.serverStatus.runtimeStatusSamplingTitle') }}</h3>
            </header>
            <dl class="status-sidebar__summary-list">
              <div
                v-for="item in samplingStatusItems"
                :key="item.key"
                class="status-sidebar__summary-item"
                :data-status-sidebar-item="item.key"
              >
                <dt class="status-sidebar__summary-key">{{ item.label }}</dt>
                <dd class="status-sidebar__summary-value">{{ item.value }}</dd>
              </div>
            </dl>
          </section>
        </div>
        <t-empty v-else :description="t('monitor.serverStatus.empty')" />
      </section-card>
    </div>
  </server-status-page-shell>
</template>
<script setup lang="ts">
import { LineChart } from 'echarts/charts';
import { GridComponent, LegendComponent, MarkLineComponent, TooltipComponent } from 'echarts/components';
import * as echarts from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { DataBaseIcon, InfoCircleIcon, LinkIcon } from 'tdesign-icons-vue-next';
import type { SelectProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import type { Component } from 'vue';
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import type { TChartColor } from '@/config/color';
import { useSettingStore } from '@/store';

import { getServerStatus } from '../api/server-status';
import MonitorToolbar from '../components/MonitorToolbar.vue';
import SectionCard from '../components/SectionCard.vue';
import type { ServerStatusTone } from '../components/server-status-ui';
import ServerStatusPageShell from '../components/ServerStatusPageShell.vue';
import SummaryMetricCard from '../components/SummaryMetricCard.vue';
import { useMonitorRefreshPreferences } from '../composables/use-monitor-refresh-preferences';
import type { MonitorRefreshInterval } from '../contract/refresh';
import { MONITOR_TREND_RANGE, type MonitorTrendRange } from '../contract/trend';
import type { ServerStatusDependency, ServerStatusResponse, ServerStatusTrendPoint } from '../types/server-status';

defineOptions({
  name: 'MonitorServerStatusOverviewIndex',
});

echarts.use([TooltipComponent, LegendComponent, GridComponent, MarkLineComponent, LineChart, CanvasRenderer]);

type MonitorStatus = 'healthy' | 'degraded' | 'disabled' | 'unknown';
type MetricCardTone = 'healthy' | 'warning' | 'critical' | 'unknown';
type TrendRange = MonitorTrendRange;
type TrendMode = 'overview' | 'multi' | 'focus';
type FocusMetric = 'cpu' | 'hostMemory' | 'load' | 'runtimeAlloc' | 'runtimeHeap' | 'runtimeSys' | 'goroutines';
type TrendMetricGroup = 'resourceUsage' | 'systemLoad' | 'goRuntime';
type TrendMetricUnit = '%' | 'load' | 'MB' | 'count';
type TrendMetricAxis = 'percent' | 'load' | 'bytes' | 'count';
type TrendChartKey =
  | 'overviewUsage'
  | 'overviewLoad'
  | 'multi-cpu'
  | 'multi-hostMemory'
  | 'multi-load'
  | 'multi-runtimeAlloc'
  | 'multi-runtimeHeap'
  | 'multi-runtimeSys'
  | 'multi-goroutines'
  | 'focus';

interface MetricCard {
  key: string;
  label: string;
  value: string;
  valueSide: string;
  meta: string;
  description: string;
  statusLabel: string;
  tagTheme: 'success' | 'warning' | 'danger' | 'default';
  tone: MetricCardTone;
}

interface TrendMetricDefinition {
  key: FocusMetric;
  label: string;
  shortLabel: string;
  unit: TrendMetricUnit;
  group: TrendMetricGroup;
  groupLabel: string;
  color: () => string;
  axis: TrendMetricAxis;
  description: string;
  formatter: (value: number | null) => string;
  visibleInOverview: boolean;
  visibleInSmallMultiples: boolean;
  visibleInFocus: boolean;
  chartKey: TrendChartKey;
  infoText?: string;
  helperText?: string;
  values: number[];
  currentValue: string;
}

interface TrendOverviewSection {
  key: 'resourceUsage' | 'systemLoad';
  chartKey: TrendChartKey;
  title: string;
  infoText?: string;
  helperText?: string;
  metrics: TrendMetricDefinition[];
}

interface StatusSidebarSummaryItem {
  key: string;
  label: string;
  value: string;
}

const { t, locale } = useI18n();
const settingStore = useSettingStore();
const {
  autoRefreshEnabled,
  refreshIntervalOptions,
  selectedRefreshInterval,
  selectedRefreshIntervalLabel,
  toggleAutoRefresh: toggleSharedAutoRefresh,
} = useMonitorRefreshPreferences();
const loading = ref(false);
const serverStatus = ref<ServerStatusResponse | null>(null);
const selectedTrendRange = ref<TrendRange>(MONITOR_TREND_RANGE.TEN_MINUTES);
const selectedTrendMode = ref<TrendMode>('overview');
const selectedFocusMetric = ref<FocusMetric>('cpu');
const lastUpdatedAt = ref<string | null>(null);
const consecutiveFailures = ref(0);
const remainingRefreshSeconds = ref<number | null>(null);
const isPageVisible = ref(typeof document === 'undefined' ? true : document.visibilityState === 'visible');

const trendChartRefs = ref<Partial<Record<TrendChartKey, HTMLDivElement | null>>>({});
let refreshTickTimer: number | null = null;
let nextRefreshAt: number | null = null;
let pendingTrendRange: TrendRange | null = null;
const trendCharts = new Map<TrendChartKey, echarts.ECharts>();

const trendRangeOptions = computed(() => [
  { label: t('monitor.serverStatus.trendRange10Minutes'), value: MONITOR_TREND_RANGE.TEN_MINUTES },
  { label: t('monitor.serverStatus.trendRange30Minutes'), value: MONITOR_TREND_RANGE.THIRTY_MINUTES },
  { label: t('monitor.serverStatus.trendRange1Hour'), value: MONITOR_TREND_RANGE.ONE_HOUR },
]);

const trendModeOptions = computed(() => [
  { label: t('monitor.serverStatus.trendModeOverview'), value: 'overview' },
  { label: t('monitor.serverStatus.trendModeMulti'), value: 'multi' },
  { label: t('monitor.serverStatus.trendModeFocus'), value: 'focus' },
]);

const selectedTrendRangeLabel = computed(() => {
  return trendRangeOptions.value.find((option) => option.value === selectedTrendRange.value)?.label ?? '--';
});

const selectedTrendModeLabel = computed(() => {
  return trendModeOptions.value.find((option) => option.value === selectedTrendMode.value)?.label ?? '--';
});

function trendGroupInfoText(group: TrendMetricGroup) {
  switch (group) {
    case 'resourceUsage':
      return t('monitor.serverStatus.trendGroupResourceUsageInfo');
    case 'systemLoad':
      return t('monitor.serverStatus.trendGroupSystemLoadInfo');
    default:
      return undefined;
  }
}

const trendMetricConfigs = computed<TrendMetricDefinition[]>(() => {
  const points = trendPoints.value;
  const cpuCores = serverStatus.value?.runtime.cpu_cores ?? 0;

  return [
    {
      key: 'cpu',
      label: t('monitor.serverStatus.chartCpu'),
      shortLabel: t('monitor.serverStatus.chartCpuShort'),
      unit: '%',
      group: 'resourceUsage',
      groupLabel: t('monitor.serverStatus.trendGroupResourceUsage'),
      color: () => readMetricThemeColor('--td-brand-color', '#2F6BFF'),
      axis: 'percent',
      description: t('monitor.serverStatus.chartCpuDescription'),
      formatter: formatPercentPrecise,
      visibleInOverview: true,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-cpu',
      infoText: trendGroupInfoText('resourceUsage'),
      currentValue: formatPercentPrecise(latestTrendPoint.value?.cpu_percent ?? null),
      values: points.map((point) => Number(point.cpu_percent.toFixed(2))),
    },
    {
      key: 'hostMemory',
      label: t('monitor.serverStatus.chartHostMemory'),
      shortLabel: t('monitor.serverStatus.chartHostMemoryShort'),
      unit: '%',
      group: 'resourceUsage',
      groupLabel: t('monitor.serverStatus.trendGroupResourceUsage'),
      color: () => readMetricThemeColor('--td-success-color-5', '#1B9C6B'),
      axis: 'percent',
      description: t('monitor.serverStatus.chartHostMemoryDescription'),
      formatter: formatPercentPrecise,
      visibleInOverview: true,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-hostMemory',
      infoText: trendGroupInfoText('resourceUsage'),
      currentValue: formatPercentPrecise(latestTrendPoint.value?.host_memory_used_percent ?? null),
      values: points.map((point) => Number(point.host_memory_used_percent.toFixed(2))),
    },
    {
      key: 'load',
      label: t('monitor.serverStatus.chartLoad'),
      shortLabel: t('monitor.serverStatus.chartLoadShort'),
      unit: 'load',
      group: 'systemLoad',
      groupLabel: t('monitor.serverStatus.trendGroupSystemLoad'),
      color: () => readMetricThemeColor('--td-warning-color-5', '#D97706'),
      axis: 'load',
      description: t('monitor.serverStatus.chartLoadDescription'),
      formatter: formatLoadAverage,
      visibleInOverview: true,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-load',
      infoText: trendGroupInfoText('systemLoad'),
      helperText:
        cpuCores > 0 ? t('monitor.serverStatus.referenceCoreCountValue', { count: String(cpuCores) }) : undefined,
      currentValue: formatLoadAverage(latestTrendPoint.value?.load_average_one_minute ?? null),
      values: points.map((point) => Number(point.load_average_one_minute.toFixed(2))),
    },
    {
      key: 'runtimeAlloc',
      label: t('monitor.serverStatus.chartRuntimeAlloc'),
      shortLabel: t('monitor.serverStatus.chartRuntimeAllocShort'),
      unit: 'MB',
      group: 'goRuntime',
      groupLabel: t('monitor.serverStatus.trendGroupGoRuntime'),
      color: () => readMetricThemeColor('--td-brand-color-7', '#7B4DFF'),
      axis: 'bytes',
      description: t('monitor.serverStatus.chartRuntimeAllocDescription'),
      formatter: formatBytes,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-runtimeAlloc',
      currentValue: formatBytes(latestTrendPoint.value?.runtime_alloc_bytes ?? 0),
      values: points.map((point) => point.runtime_alloc_bytes),
    },
    {
      key: 'runtimeHeap',
      label: t('monitor.serverStatus.chartRuntimeHeap'),
      shortLabel: t('monitor.serverStatus.chartRuntimeHeapShort'),
      unit: 'MB',
      group: 'goRuntime',
      groupLabel: t('monitor.serverStatus.trendGroupGoRuntime'),
      color: () => readMetricThemeColor('--td-brand-color-5', '#17A2B8'),
      axis: 'bytes',
      description: t('monitor.serverStatus.chartRuntimeHeapDescription'),
      formatter: formatBytes,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-runtimeHeap',
      currentValue: formatBytes(latestTrendPoint.value?.runtime_heap_in_use_bytes ?? 0),
      values: points.map((point) => point.runtime_heap_in_use_bytes),
    },
    {
      key: 'runtimeSys',
      label: t('monitor.serverStatus.chartRuntimeSys'),
      shortLabel: t('monitor.serverStatus.chartRuntimeSysShort'),
      unit: 'MB',
      group: 'goRuntime',
      groupLabel: t('monitor.serverStatus.trendGroupGoRuntime'),
      color: () => readMetricThemeColor('--td-error-color-6', '#A56A2A'),
      axis: 'bytes',
      description: t('monitor.serverStatus.chartRuntimeSysDescription'),
      formatter: formatBytes,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-runtimeSys',
      currentValue: formatBytes(latestTrendPoint.value?.runtime_sys_bytes ?? 0),
      values: points.map((point) => point.runtime_sys_bytes),
    },
    {
      key: 'goroutines',
      label: t('monitor.serverStatus.chartGoroutines'),
      shortLabel: t('monitor.serverStatus.chartGoroutinesShort'),
      unit: 'count',
      group: 'goRuntime',
      groupLabel: t('monitor.serverStatus.trendGroupGoRuntime'),
      color: () => readMetricThemeColor('--td-error-color-5', '#D9488B'),
      axis: 'count',
      description: t('monitor.serverStatus.chartGoroutinesDescription'),
      formatter: formatCountValue,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-goroutines',
      currentValue: formatCountValue(latestTrendPoint.value?.goroutines ?? null),
      values: points.map((point) => point.goroutines),
    },
  ];
});

const focusMetricOptions = computed<SelectProps['options']>(() => {
  return trendMetricConfigs.value
    .filter((metric) => metric.visibleInFocus)
    .map((metric) => ({
      label: `${metric.groupLabel} / ${metric.label}`,
      value: metric.key,
    }));
});

const trendPoints = computed<ServerStatusTrendPoint[]>(() => serverStatus.value?.trend.points ?? []);
const latestTrendPoint = computed<ServerStatusTrendPoint | null>(() => trendPoints.value.at(-1) ?? null);
const hasTrendData = computed(() => trendPoints.value.length >= 2);
const visibleTrendMetricCount = computed(
  () => trendMetricConfigs.value.filter((metric) => metric.visibleInFocus).length,
);
const trendGroupSummaryLabel = computed(() =>
  [
    t('monitor.serverStatus.trendGroupResourceUsage'),
    t('monitor.serverStatus.trendGroupSystemLoad'),
    t('monitor.serverStatus.trendGroupGoRuntime'),
  ].join(' / '),
);
const overviewTrendSections = computed<TrendOverviewSection[]>(() => [
  {
    key: 'resourceUsage',
    chartKey: 'overviewUsage',
    title: t('monitor.serverStatus.trendGroupResourceUsage'),
    infoText: t('monitor.serverStatus.trendGroupResourceUsageInfo'),
    metrics: trendMetricConfigs.value.filter((metric) => metric.group === 'resourceUsage' && metric.visibleInOverview),
  },
  {
    key: 'systemLoad',
    chartKey: 'overviewLoad',
    title: t('monitor.serverStatus.trendGroupSystemLoad'),
    infoText: t('monitor.serverStatus.trendGroupSystemLoadInfo'),
    helperText:
      (serverStatus.value?.runtime.cpu_cores ?? 0) > 0
        ? t('monitor.serverStatus.referenceCoreCountValue', {
            count: String(serverStatus.value?.runtime.cpu_cores ?? 0),
          })
        : undefined,
    metrics: trendMetricConfigs.value.filter((metric) => metric.group === 'systemLoad' && metric.visibleInOverview),
  },
]);
const runtimeSummaryMetrics = computed(() =>
  trendMetricConfigs.value.filter((metric) => metric.group === 'goRuntime').slice(0, 4),
);
const smallMultipleMetrics = computed(() =>
  trendMetricConfigs.value.filter((metric) => metric.visibleInSmallMultiples),
);
const currentFocusMetric = computed(
  () =>
    trendMetricConfigs.value.find((metric) => metric.key === selectedFocusMetric.value) ?? trendMetricConfigs.value[0],
);
const focusReferenceText = computed(() =>
  currentFocusMetric.value?.group === 'systemLoad' ? currentFocusMetric.value.helperText : '',
);

const overallStatus = computed<MonitorStatus>(() => {
  const statuses = [
    normalizeStatus(serverStatus.value?.status),
    ...metricCards.value.map((card) => metricToneToMonitorStatus(card.tone)),
    ...dependencyItems.value.map((item) => item.status),
  ];

  if (statuses.includes('degraded')) {
    return 'degraded';
  }
  if (statuses.includes('healthy')) {
    return 'healthy';
  }
  if (statuses.includes('disabled')) {
    return 'disabled';
  }

  return 'unknown';
});

const refreshCountdownText = computed(() => {
  if (!autoRefreshEnabled.value) {
    return t('monitor.serverStatus.nextRefreshPausedByUser');
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
      interval: selectedRefreshIntervalLabel.value,
    });
  }

  return t('monitor.serverStatus.nextRefreshIn', {
    seconds: String(remainingRefreshSeconds.value),
  });
});

const metricCards = computed<MetricCard[]>(() => {
  const response = serverStatus.value;
  if (!response) {
    return [
      emptyMetricCard('load', t('monitor.serverStatus.metricLoadLabel')),
      emptyMetricCard('cpu', t('monitor.serverStatus.metricCpuLabel')),
      emptyMetricCard('memory', t('monitor.serverStatus.metricMemoryLabel')),
      emptyMetricCard('disk', t('monitor.serverStatus.metricDiskLabel')),
    ];
  }

  const loadAverage = response.runtime.load_average;
  const loadPercent =
    response.runtime.cpu_cores > 0 ? (loadAverage.one_minute / response.runtime.cpu_cores) * 100 : null;
  const cpuPercent = latestTrendPoint.value?.cpu_percent ?? null;
  const hostMemoryPercent = response.runtime.host_memory_used_percent;
  const diskPercent = response.runtime.disk_usage.total_bytes > 0 ? response.runtime.disk_usage.used_percent : null;
  const diskPath = normalizedDiskPath(response.runtime.disk_usage.path);

  return [
    {
      key: 'load',
      label: t('monitor.serverStatus.metricLoadLabel'),
      value: formatLoadAverage(loadAverage.one_minute),
      valueSide: t('monitor.serverStatus.metricLoadValueSide'),
      meta: t('monitor.serverStatus.metricLoadMeta', {
        five: formatLoadAverage(loadAverage.five_minutes),
        fifteen: formatLoadAverage(loadAverage.fifteen_minutes),
      }),
      ...buildLoadCardStatus(loadPercent),
    },
    {
      key: 'cpu',
      label: t('monitor.serverStatus.metricCpuLabel'),
      value: formatPercent(cpuPercent),
      valueSide: t('monitor.serverStatus.metricCpuValue', {
        count: String(response.runtime.cpu_cores),
      }),
      meta: t('monitor.serverStatus.metricCpuMeta', {
        count: String(response.runtime.cpu_cores),
      }),
      ...buildCpuCardStatus(cpuPercent),
    },
    {
      key: 'memory',
      label: t('monitor.serverStatus.metricMemoryLabel'),
      value: formatPercent(hostMemoryPercent),
      valueSide: t('monitor.serverStatus.metricMemoryValue', {
        used: formatBytes(response.runtime.host_memory_used_bytes),
        total: formatBytes(response.runtime.host_memory_total_bytes),
      }),
      meta: t('monitor.serverStatus.metricMemoryMeta', {
        available: formatBytes(response.runtime.host_memory_free_bytes),
      }),
      ...buildMemoryCardStatus(hostMemoryPercent),
    },
    {
      key: 'disk',
      label: t('monitor.serverStatus.metricDiskLabel'),
      value: formatPercent(diskPercent),
      valueSide: t('monitor.serverStatus.metricDiskValue', {
        used: formatBytes(response.runtime.disk_usage.used_bytes),
        total: formatBytes(response.runtime.disk_usage.total_bytes),
      }),
      meta: t('monitor.serverStatus.metricDiskMeta', {
        path: diskPath,
        free: formatBytes(response.runtime.disk_usage.free_bytes),
      }),
      ...buildDiskCardStatus(diskPercent),
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
      t('monitor.serverStatus.postgresqlLabel'),
      response.dependencies.database,
      DataBaseIcon,
    ),
    buildDependencyItem('redis', t('monitor.serverStatus.redisLabel'), response.dependencies.redis, LinkIcon),
  ];
});

const processSummaryItems = computed<StatusSidebarSummaryItem[]>(() => {
  const response = serverStatus.value;
  if (!response) {
    return [];
  }

  return [
    {
      key: 'uptime',
      label: t('monitor.serverStatus.runtimeStatusUptimeLabel'),
      value: formatUptime(response.server.uptime_seconds),
    },
    {
      key: 'goroutines',
      label: t('monitor.serverStatus.runtimeStatusGoroutinesLabel'),
      value: String(response.runtime.goroutines),
    },
    {
      key: 'heap',
      label: t('monitor.serverStatus.runtimeStatusHeapLabel'),
      value: formatBytes(response.runtime.runtime_heap_in_use_bytes),
    },
    {
      key: 'runtimeSys',
      label: t('monitor.serverStatus.runtimeStatusRuntimeSysLabel'),
      value: formatBytes(response.runtime.runtime_sys_bytes),
    },
    {
      key: 'gcCount',
      label: t('monitor.serverStatus.runtimeStatusGcCountLabel'),
      value: String(response.runtime.runtime_gc_cycles),
    },
    {
      key: 'lastGc',
      label: t('monitor.serverStatus.runtimeStatusLastGcLabel'),
      value: t('monitor.serverStatus.runtimeStatusNotAvailable'),
    },
  ];
});

const samplingStatusItems = computed<StatusSidebarSummaryItem[]>(() => [
  {
    key: 'lastUpdated',
    label: t('monitor.serverStatus.runtimeStatusLastUpdatedLabel'),
    value: formatTimeOnly(lastUpdatedAt.value ?? serverStatus.value?.observed_at),
  },
  {
    key: 'autoRefresh',
    label: t('monitor.serverStatus.runtimeStatusAutoRefreshLabel'),
    value: autoRefreshEnabled.value
      ? t('monitor.serverStatus.runtimeStatusRefreshValue')
      : t('monitor.serverStatus.runtimeStatusPaused'),
  },
  {
    key: 'timeRange',
    label: t('monitor.serverStatus.runtimeStatusTimeRangeLabel'),
    value: selectedTrendRangeLabel.value,
  },
  {
    key: 'samples',
    label: t('monitor.serverStatus.runtimeStatusSamplesLabel'),
    value: String(trendPoints.value.length),
  },
  {
    key: 'trendMode',
    label: t('monitor.serverStatus.runtimeStatusTrendModeLabel'),
    value: selectedTrendModeLabel.value,
  },
]);
const toolbarStatus = computed<ServerStatusTone>(() => {
  switch (overallStatus.value) {
    case 'healthy':
      return 'healthy';
    case 'degraded':
      return 'warning';
    case 'disabled':
      return 'disabled';
    default:
      return 'unknown';
  }
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

function toggleAutoRefresh() {
  toggleSharedAutoRefresh();

  if (autoRefreshEnabled.value && isPageVisible.value) {
    void fetchServerStatus({ manual: true });
    return;
  }

  stopRefreshTick();
  remainingRefreshSeconds.value = null;
}

function scheduleNextRefresh() {
  stopRefreshTick();
  if (!autoRefreshEnabled.value || !isPageVisible.value) {
    remainingRefreshSeconds.value = null;
    return;
  }

  const backoffMultiplier = consecutiveFailures.value > 0 ? 2 ** consecutiveFailures.value : 1;
  const delaySeconds = Math.min(selectedRefreshInterval.value * backoffMultiplier, 5 * 60);
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
  if (isPageVisible.value && autoRefreshEnabled.value) {
    void fetchServerStatus();
    return;
  }

  stopRefreshTick();
  remainingRefreshSeconds.value = null;
}

function handleRefreshIntervalChange(value: number | string) {
  selectedRefreshInterval.value = value as MonitorRefreshInterval;
}

function handleTrendRangeChange(value: number | string) {
  selectedTrendRange.value = value as TrendRange;
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

function emptyMetricCard(key: string, label: string): MetricCard {
  return {
    key,
    label,
    value: '--',
    valueSide: '--',
    meta: t('monitor.serverStatus.emptyMetric.meta'),
    description: t('monitor.serverStatus.emptyMetric.description'),
    statusLabel: t('monitor.serverStatus.statusUnknown'),
    tagTheme: 'default',
    tone: 'unknown',
  };
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

function loadTone(percent: number | null): MetricCardTone {
  if (percent === null || Number.isNaN(percent)) {
    return 'unknown';
  }
  if (percent >= 100) {
    return 'critical';
  }
  if (percent >= 60) {
    return 'warning';
  }
  return 'healthy';
}

function normalizedDiskPath(path?: string | null) {
  if (!path) {
    return t('monitor.serverStatus.diskRootPath');
  }
  return path;
}

function usageTone(percent: number | null, warningThreshold: number, criticalThreshold: number): MetricCardTone {
  if (percent === null || Number.isNaN(percent)) {
    return 'unknown';
  }
  if (percent >= criticalThreshold) {
    return 'critical';
  }
  if (percent >= warningThreshold) {
    return 'warning';
  }
  return 'healthy';
}

function metricToneToMonitorStatus(tone: MetricCardTone): MonitorStatus {
  switch (tone) {
    case 'healthy':
      return 'healthy';
    case 'warning':
    case 'critical':
      return 'degraded';
    default:
      return 'unknown';
  }
}

function metricToneToServerStatusTone(tone: MetricCardTone): ServerStatusTone {
  switch (tone) {
    case 'healthy':
      return 'healthy';
    case 'warning':
      return 'warning';
    case 'critical':
      return 'error';
    default:
      return 'unknown';
  }
}

function metricCardTagTheme(tone: MetricCardTone): MetricCard['tagTheme'] {
  switch (tone) {
    case 'healthy':
      return 'success';
    case 'warning':
      return 'warning';
    case 'critical':
      return 'danger';
    default:
      return 'default';
  }
}

function buildLoadCardStatus(
  percent: number | null,
): Pick<MetricCard, 'description' | 'statusLabel' | 'tagTheme' | 'tone'> {
  const tone = loadTone(percent);

  switch (tone) {
    case 'healthy':
      return {
        tone,
        statusLabel: t('monitor.serverStatus.metricLoadStatusHealthy'),
        description: t('monitor.serverStatus.metricLoadDescriptionHealthy'),
        tagTheme: metricCardTagTheme(tone),
      };
    case 'warning':
      return {
        tone,
        statusLabel: t('monitor.serverStatus.metricLoadStatusWarning'),
        description: t('monitor.serverStatus.metricLoadDescriptionWarning'),
        tagTheme: metricCardTagTheme(tone),
      };
    case 'critical':
      return {
        tone,
        statusLabel: t('monitor.serverStatus.metricLoadStatusCritical'),
        description: t('monitor.serverStatus.metricLoadDescriptionCritical'),
        tagTheme: metricCardTagTheme(tone),
      };
    default:
      return {
        tone,
        statusLabel: t('monitor.serverStatus.statusUnknown'),
        description: t('monitor.serverStatus.emptyMetric.description'),
        tagTheme: metricCardTagTheme(tone),
      };
  }
}

function buildCpuCardStatus(
  percent: number | null,
): Pick<MetricCard, 'description' | 'statusLabel' | 'tagTheme' | 'tone'> {
  const tone = usageTone(percent, 20, 70);

  switch (tone) {
    case 'healthy':
      return {
        tone,
        statusLabel: t('monitor.serverStatus.metricCpuStatusHealthy'),
        description: t('monitor.serverStatus.metricCpuDescriptionHealthy'),
        tagTheme: metricCardTagTheme(tone),
      };
    case 'warning':
      return {
        tone,
        statusLabel: t('monitor.serverStatus.metricCpuStatusWarning'),
        description: t('monitor.serverStatus.metricCpuDescriptionWarning'),
        tagTheme: metricCardTagTheme(tone),
      };
    case 'critical':
      return {
        tone,
        statusLabel: t('monitor.serverStatus.metricCpuStatusCritical'),
        description: t('monitor.serverStatus.metricCpuDescriptionCritical'),
        tagTheme: metricCardTagTheme(tone),
      };
    default:
      return {
        tone,
        statusLabel: t('monitor.serverStatus.statusUnknown'),
        description: t('monitor.serverStatus.emptyMetric.description'),
        tagTheme: metricCardTagTheme(tone),
      };
  }
}

function buildMemoryCardStatus(
  percent: number | null,
): Pick<MetricCard, 'description' | 'statusLabel' | 'tagTheme' | 'tone'> {
  const tone = usageTone(percent, 60, 85);

  switch (tone) {
    case 'healthy':
      return {
        tone,
        statusLabel: t('monitor.serverStatus.metricMemoryStatusHealthy'),
        description: t('monitor.serverStatus.metricMemoryDescriptionHealthy'),
        tagTheme: metricCardTagTheme(tone),
      };
    case 'warning':
      return {
        tone,
        statusLabel: t('monitor.serverStatus.metricMemoryStatusWarning'),
        description: t('monitor.serverStatus.metricMemoryDescriptionWarning'),
        tagTheme: metricCardTagTheme(tone),
      };
    case 'critical':
      return {
        tone,
        statusLabel: t('monitor.serverStatus.metricMemoryStatusCritical'),
        description: t('monitor.serverStatus.metricMemoryDescriptionCritical'),
        tagTheme: metricCardTagTheme(tone),
      };
    default:
      return {
        tone,
        statusLabel: t('monitor.serverStatus.statusUnknown'),
        description: t('monitor.serverStatus.emptyMetric.description'),
        tagTheme: metricCardTagTheme(tone),
      };
  }
}

function buildDiskCardStatus(
  percent: number | null,
): Pick<MetricCard, 'description' | 'statusLabel' | 'tagTheme' | 'tone'> {
  const tone = usageTone(percent, 70, 85);

  switch (tone) {
    case 'healthy':
      return {
        tone,
        statusLabel: t('monitor.serverStatus.metricDiskStatusHealthy'),
        description: t('monitor.serverStatus.metricDiskDescriptionHealthy'),
        tagTheme: metricCardTagTheme(tone),
      };
    case 'warning':
      return {
        tone,
        statusLabel: t('monitor.serverStatus.metricDiskStatusWarning'),
        description: t('monitor.serverStatus.metricDiskDescriptionWarning'),
        tagTheme: metricCardTagTheme(tone),
      };
    case 'critical':
      return {
        tone,
        statusLabel: t('monitor.serverStatus.metricDiskStatusCritical'),
        description: t('monitor.serverStatus.metricDiskDescriptionCritical'),
        tagTheme: metricCardTagTheme(tone),
      };
    default:
      return {
        tone,
        statusLabel: t('monitor.serverStatus.statusUnknown'),
        description: t('monitor.serverStatus.emptyMetric.description'),
        tagTheme: metricCardTagTheme(tone),
      };
  }
}

function formatPercent(percent: number | null) {
  if (percent === null || Number.isNaN(percent)) {
    return '--';
  }
  return `${Math.max(0, Math.round(percent))}%`;
}

function formatPercentPrecise(percent: number | null) {
  if (percent === null || Number.isNaN(percent)) {
    return '--';
  }

  return `${percent.toFixed(percent >= 10 ? 1 : 2)}%`;
}

function formatLoadAverage(value: number | null) {
  if (value === null || Number.isNaN(value)) {
    return '--';
  }
  return value.toFixed(2);
}

function formatCountValue(value: number | null) {
  if (value === null || Number.isNaN(value)) {
    return '--';
  }

  return `${Math.round(value)}`;
}

function setTrendChartRef(key: TrendChartKey, el: Element | Component | null) {
  trendChartRefs.value[key] = el instanceof HTMLDivElement ? el : null;
}

function ensureTrendChart(key: TrendChartKey) {
  const el = trendChartRefs.value[key];
  if (!el) {
    return null;
  }

  const existing = trendCharts.get(key);
  if (existing) {
    return existing;
  }

  const chart = echarts.init(el);
  trendCharts.set(key, chart);
  return chart;
}

function syncTrendChart() {
  if (!hasTrendData.value) {
    disposeTrendChart();
    return;
  }

  const options = buildTrendChartOptions(trendPoints.value, settingStore.chartColors);
  const activeKeys = new Set<TrendChartKey>(options.map((item) => item.key));

  options.forEach(({ key, option }) => {
    const chart = ensureTrendChart(key);
    if (!chart) {
      return;
    }

    chart.setOption(option, true);
  });

  for (const [key, chart] of trendCharts.entries()) {
    if (!activeKeys.has(key)) {
      chart.dispose();
      trendCharts.delete(key);
    }
  }

  resizeTrendChart();
}

function resizeTrendChart() {
  trendCharts.forEach((chart, key) => {
    chart.resize({
      width: trendChartRefs.value[key]?.clientWidth,
      height: trendChartRefs.value[key]?.clientHeight,
    });
  });
}

function disposeTrendChart() {
  trendCharts.forEach((chart) => chart.dispose());
  trendCharts.clear();
}

function buildTrendChartOptions(points: ServerStatusTrendPoint[], chartColors: TChartColor) {
  const metrics = trendMetricConfigs.value;
  const labels = points.map((point) => formatChartTimestamp(point.observed_at));

  if (selectedTrendMode.value === 'overview') {
    return overviewTrendSections.value.map((section) => ({
      key: section.chartKey,
      option:
        section.key === 'resourceUsage'
          ? buildOverviewUsageChartOption(labels, section.metrics, chartColors)
          : buildOverviewLoadChartOption(labels, section.metrics, chartColors),
    }));
  }

  if (selectedTrendMode.value === 'focus') {
    const focusMetric = metrics.find((metric) => metric.key === selectedFocusMetric.value) ?? metrics[0];
    return [
      {
        key: 'focus' as const,
        option: buildFocusTrendChartOption(labels, focusMetric, chartColors),
      },
    ];
  }

  return smallMultipleMetrics.value.map((metric) => ({
    key: metric.chartKey as TrendChartKey,
    option: buildSmallMultipleTrendChartOption(labels, metric, chartColors),
  }));
}

function buildOverviewUsageChartOption(labels: string[], metrics: TrendMetricDefinition[], chartColors: TChartColor) {
  return {
    color: metrics.map((metric) => metric.color()),
    tooltip: buildTooltip(chartColors, metrics),
    grid: {
      left: '18px',
      right: '18px',
      top: '12px',
      bottom: '28px',
      containLabel: true,
    },
    xAxis: buildXAxis(labels, chartColors),
    yAxis: [buildYAxis('%', 'percent', chartColors, { min: 0, max: 100 })],
    series: metrics.map((metric) => buildSeries(metric, 0)),
  };
}

function buildOverviewLoadChartOption(labels: string[], metrics: TrendMetricDefinition[], chartColors: TChartColor) {
  const loadMetric = metrics[0];

  return {
    color: [loadMetric.color()],
    tooltip: buildTooltip(chartColors, metrics),
    grid: {
      left: '18px',
      right: '18px',
      top: '12px',
      bottom: '28px',
      containLabel: true,
    },
    xAxis: buildXAxis(labels, chartColors),
    yAxis: [buildYAxis(t('monitor.serverStatus.chartLoadAxis'), 'load', chartColors)],
    series: [buildSeries(loadMetric, 0, { markLineValue: serverStatus.value?.runtime.cpu_cores ?? null })],
  };
}

function buildSmallMultipleTrendChartOption(labels: string[], metric: TrendMetricDefinition, chartColors: TChartColor) {
  return {
    color: [metric.color()],
    tooltip: buildTooltip(chartColors, [metric]),
    grid: {
      left: '18px',
      right: '18px',
      top: '18px',
      bottom: '28px',
      containLabel: true,
    },
    xAxis: buildXAxis(labels, chartColors),
    yAxis: [buildSingleAxis(metric, chartColors)],
    series: [
      buildSeries(metric, 0, {
        area: true,
        markLineValue: metric.key === 'load' ? (serverStatus.value?.runtime.cpu_cores ?? null) : null,
      }),
    ],
  };
}

function buildFocusTrendChartOption(labels: string[], metric: TrendMetricDefinition, chartColors: TChartColor) {
  return {
    color: [metric.color()],
    tooltip: buildTooltip(chartColors, [metric]),
    grid: {
      left: '18px',
      right: '18px',
      top: '18px',
      bottom: '28px',
      containLabel: true,
    },
    xAxis: buildXAxis(labels, chartColors),
    yAxis: [buildSingleAxis(metric, chartColors)],
    series: [
      buildSeries(metric, 0, {
        area: true,
        markLineValue: metric.key === 'load' ? (serverStatus.value?.runtime.cpu_cores ?? null) : null,
      }),
    ],
  };
}

function buildTooltip(chartColors: TChartColor, metrics: TrendMetricDefinition[]) {
  const metricMap = new Map(metrics.map((metric) => [metric.label, metric]));
  return {
    trigger: 'axis',
    backgroundColor: chartColors.containerColor,
    borderColor: chartColors.borderColor,
    textStyle: {
      color: chartColors.textColor,
    },
    formatter: (params: Array<{ axisValueLabel: string; seriesName: string; color: string; data: number }>) => {
      const rows = params
        .map((param) => {
          const metric = metricMap.get(param.seriesName);
          if (!metric) {
            return '';
          }

          const valueLabel =
            metric.axis === 'bytes'
              ? `${param.data.toFixed(1)} MB`
              : metric.axis === 'count'
                ? formatCountValue(param.data)
                : metric.axis === 'percent'
                  ? formatPercentPrecise(param.data)
                  : formatLoadAverage(param.data);

          return [
            `<div style="display:flex;align-items:center;justify-content:space-between;gap:16px;">`,
            `<span style="display:flex;align-items:center;gap:8px;">`,
            `<i style="width:8px;height:8px;border-radius:999px;background:${param.color};display:inline-block;"></i>`,
            `<span>${metric.label}</span>`,
            `</span>`,
            `<strong>${valueLabel}</strong>`,
            `</div>`,
          ].join('');
        })
        .filter(Boolean)
        .join('');

      return `<div style="display:flex;flex-direction:column;gap:8px;"><strong>${params[0]?.axisValueLabel ?? ''}</strong>${rows}</div>`;
    },
  };
}

function buildXAxis(labels: string[], chartColors: TChartColor) {
  return {
    type: 'category',
    data: labels,
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
  };
}

function buildSingleAxis(metric: TrendMetricDefinition, chartColors: TChartColor) {
  switch (metric.axis) {
    case 'percent':
      return buildYAxis(metric.unit, 'percent', chartColors, { min: 0, max: 100 });
    case 'load':
      return buildYAxis(metric.unit, 'load', chartColors);
    case 'bytes':
      return buildYAxis(metric.unit, 'bytes', chartColors);
    case 'count':
      return buildYAxis(metric.unit, 'count', chartColors);
    default:
      return buildYAxis(metric.unit, 'count', chartColors);
  }
}

function buildYAxis(
  name: string,
  axisType: 'percent' | 'load' | 'bytes' | 'count',
  chartColors: TChartColor,
  bounds?: { min?: number; max?: number },
) {
  return {
    type: 'value',
    name,
    min: bounds?.min ?? 0,
    max: bounds?.max,
    axisLabel: {
      color: chartColors.placeholderColor,
      formatter: (value: number) => formatAxisValue(value, axisType),
    },
    splitLine: {
      lineStyle: {
        color: chartColors.borderColor,
      },
    },
  };
}

function buildSeries(
  metric: TrendMetricDefinition,
  yAxisIndex: number,
  options: {
    area?: boolean;
    markLineValue?: number | null;
  } = {},
) {
  return {
    name: metric.label,
    type: 'line',
    smooth: true,
    yAxisIndex,
    symbol: 'circle',
    symbolSize: options.area ? 7 : 6,
    showSymbol: false,
    lineStyle: {
      width: 2.5,
    },
    areaStyle: {
      opacity: options.area ? 0.14 : 0,
    },
    emphasis: {
      focus: 'series',
      areaStyle: {
        opacity: options.area ? 0.18 : 0.12,
      },
    },
    markLine:
      options.markLineValue && options.markLineValue > 0
        ? {
            symbol: 'none',
            label: {
              formatter: t('monitor.serverStatus.referenceCoreCountMark', { count: String(options.markLineValue) }),
            },
            lineStyle: {
              type: 'dashed',
              opacity: 0.72,
            },
            data: [{ yAxis: options.markLineValue }],
          }
        : undefined,
    data: metric.values.map((value) =>
      Number(metric.axis === 'bytes' ? (value / 1024 / 1024).toFixed(2) : (value.toFixed?.(2) ?? value)),
    ),
  };
}

function formatAxisValue(value: number, axisType: 'percent' | 'load' | 'bytes' | 'count') {
  switch (axisType) {
    case 'percent':
      return `${value}%`;
    case 'load':
      return value.toFixed(1);
    case 'bytes':
      return `${value} MB`;
    default:
      return `${value}`;
  }
}

function readMetricThemeColor(token: string, fallback: string) {
  void settingStore.resolvedThemeTokensForDisplayMode;
  return readThemeToken(token, fallback);
}

function readThemeToken(token: string, fallback: string) {
  const value = getComputedStyle(document.documentElement).getPropertyValue(token).trim();
  return value || fallback;
}

function overallStatusLabel(status?: string) {
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

function formatUptime(totalSeconds: number) {
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  return `${hours}h ${minutes}m ${seconds}s`;
}

function formatTimeOnly(value?: string | null) {
  if (!value) {
    return t('monitor.serverStatus.runtimeStatusNotAvailable');
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(locale.value, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
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

function formatBytes(bytes: number | null) {
  if (bytes === null || Number.isNaN(bytes) || bytes === 0) {
    return '0 B';
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let value = bytes;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  const decimals = unitIndex >= 3 ? 1 : value >= 10 || unitIndex === 0 ? 0 : 1;
  return `${value.toFixed(decimals)} ${units[unitIndex]}`;
}

function formatLatency(latency: number | null) {
  if (latency === null) {
    return t('monitor.serverStatus.noLatency');
  }

  return t('monitor.serverStatus.latencyValue', { value: latency.toFixed(2) });
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
    default:
      return 'default';
  }
}

watch(
  [
    () => trendPoints.value,
    () => trendMetricConfigs.value,
    () => selectedTrendMode.value,
    () => selectedFocusMetric.value,
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

watch(selectedTrendRange, async (nextRange, previousRange) => {
  if (nextRange === previousRange) {
    return;
  }

  await fetchServerStatus();
});

watch(selectedRefreshInterval, (nextValue, previousValue) => {
  if (nextValue === previousValue) {
    return;
  }

  scheduleNextRefresh();
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

.server-status-overview-layout {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(12, minmax(0, 1fr));
}

.server-status-overview-layout__trend {
  grid-column: span 8;
}

.server-status-overview-layout__status {
  grid-column: span 4;
}

@media (width <= 991px) {
  .server-status-overview-layout__trend,
  .server-status-overview-layout__status {
    grid-column: span 12;
  }
}
</style>
