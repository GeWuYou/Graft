<template>
  <div class="monitor-dashboard" :data-responsive-density="dashboardVariant.density">
    <server-status-page-shell
      :eyebrow="t('monitor.sectionTitle')"
      title-key="monitor.serverStatus.overviewTitle"
      description-key="monitor.serverStatus.overviewHint"
    >
      <template #toolbar>
        <refresh-control-bar
          :status="refreshControlStatus"
          :countdown-seconds="remainingRefreshSeconds"
          :interval="selectedRefreshInterval"
          :interval-options="refreshIntervalOptions"
          :refreshing="loading"
          :show-countdown="true"
          :show-trend-window="true"
          :status-tone="toolbarStatus"
          :status-label="overallStatusLabel(overallStatus)"
          :trend-window="selectedTrendRange"
          :trend-window-label="t('monitor.serverStatus.trendWindowLabel')"
          :trend-window-options="trendRangeOptions"
          variant="page"
          @refresh="() => fetchServerStatus({ manual: true })"
          @pause="toggleAutoRefresh"
          @resume="toggleAutoRefresh"
          @update:interval="handleRefreshIntervalChange"
          @update:trend-window="handleTrendRangeChange"
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
          :description="card.meta"
          :status="metricToneToServerStatusTone(card.tone)"
          :status-label="card.statusLabel"
        >
          <metric-usage-bar
            :value="card.usage.value"
            :label="card.usage.label"
            :status="card.usage.status"
            :tooltip="card.usage.tooltip"
            :loading="card.usage.loading"
            :empty-text="t('monitor.serverStatus.metricUsageNoData')"
          />
          <p class="metric-card__usage-description">{{ card.description }}</p>
        </summary-metric-card>
      </template>

      <responsive-content class="server-status-overview-layout" layout="wide-split">
        <section-card
          class="server-status-overview-layout__trend"
          :title="t('monitor.serverStatus.trendCardTitle')"
          :min-height="520"
        >
          <template #actions>
            <div v-if="!isCompactDashboard" class="trend-panel__actions">
              <t-radio-group v-model="selectedTrendMode" variant="default-filled" size="small">
                <t-radio-button v-for="option in trendModeOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </t-radio-button>
              </t-radio-group>
            </div>
          </template>

          <div class="trend-panel__shell" :data-mode="activeTrendMode">
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
                v-if="activeTrendMode === 'focus'"
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
                v-if="activeTrendMode === 'overview'"
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

                <article class="trend-runtime-summary" data-trend-overview-section="requestPerformance">
                  <header class="trend-section-header">
                    <div class="trend-section-header__copy">
                      <h3 class="trend-section-header__title">
                        {{ t('monitor.serverStatus.requestPerformanceTitle') }}
                      </h3>
                    </div>
                    <t-button theme="primary" variant="text" size="small" @click="openRequestPerformance">
                      {{ t('monitor.serverStatus.openRequestPerformance') }}
                    </t-button>
                  </header>
                  <div class="trend-runtime-summary__grid">
                    <article
                      v-for="metric in requestPerformanceMetrics"
                      :key="metric.key"
                      class="trend-runtime-summary__item"
                    >
                      <span class="trend-runtime-summary__label">{{ metric.label }}</span>
                      <strong class="trend-runtime-summary__value">{{ metric.value }}</strong>
                    </article>
                  </div>
                </article>
              </div>

              <transition-group
                v-else-if="activeTrendMode === 'multi'"
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
                :data-trend-mode-panel="activeTrendMode"
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

            <article class="trend-runtime-summary" data-trend-overview-section="runtimeSummary">
              <header class="trend-section-header">
                <div class="trend-section-header__copy">
                  <h3 class="trend-section-header__title">{{ t('monitor.serverStatus.runtimeSummaryTitle') }}</h3>
                </div>
                <t-button theme="primary" variant="text" size="small" @click="openServiceStatus">
                  {{ t('monitor.serverStatus.openServiceStatus') }}
                </t-button>
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
        </section-card>

        <section-card
          class="server-status-overview-layout__status"
          :title="t('monitor.serverStatus.runtimeStatusDependenciesTitle')"
          :min-height="520"
        >
          <template #actions>
            <t-button theme="primary" variant="text" size="small" @click="openDependencies">
              {{ t('monitor.serverStatus.openDependencies') }}
            </t-button>
          </template>
          <div v-if="serverStatus" class="status-sidebar__content">
            <dependency-health-card
              v-for="service in overviewDependencyCards"
              :key="service.key"
              variant="summary"
              :service-key="service.key"
              :title="service.name"
              :description="service.description"
              :status="service.status"
              :status-label="service.statusLabel"
              :primary-metric="service.primaryMetric"
              :pool="service.pool"
              :diagnostics-title="t('monitor.dependenciesPage.diagnostics.title')"
            />
          </div>
          <t-empty v-else :description="t('monitor.serverStatus.empty')" />
        </section-card>
      </responsive-content>

      <section-card class="host-observability-section" :title="t('monitor.serverStatus.hostObservabilityTitle')">
        <div class="host-observability-section__groups">
          <article
            v-for="group in hostObservabilityGroups"
            :key="group.key"
            class="host-observability-group"
            :data-host-observability-group="group.key"
          >
            <h3 class="host-observability-group__title">{{ group.title }}</h3>
            <div class="host-observability-group__metrics">
              <div v-for="metric in group.metrics" :key="metric.key" class="host-observability-metric">
                <span class="host-observability-metric__label">{{ metric.label }}</span>
                <strong class="host-observability-metric__value">{{ metric.value }}</strong>
              </div>
            </div>
          </article>
        </div>
      </section-card>
    </server-status-page-shell>
  </div>
</template>
<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { LineChart } from 'echarts/charts';
import { GridComponent, LegendComponent, MarkLineComponent, TooltipComponent } from 'echarts/components';
import * as echarts from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { InfoCircleIcon } from 'tdesign-icons-vue-next';
import type { SelectProps } from 'tdesign-vue-next';
import { type Component, computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import type { TChartColor } from '@/config/color';
import { openCorrelationErrorNotification, requestIdFromError } from '@/modules/audit/shared/correlation-actions';
import { RefreshControlBar } from '@/shared/components/refresh';
import ResponsiveContent from '@/shared/components/responsive/ResponsiveContent.vue';
import { useViewportResponsiveVariant } from '@/shared/composables';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { useRealtimeSchedulerStore, useSettingStore } from '@/store';

import { getRequestPerformance } from '../../api/request-performance';
import { getServerStatus } from '../../api/server-status';
import DependencyHealthCard, {
  type DependencyHealthMetric,
  type DependencyHealthPool,
} from '../../components/DependencyHealthCard.vue';
import MetricUsageBar from '../../components/MetricUsageBar.vue';
import SectionCard from '../../components/SectionCard.vue';
import type { ServerStatusTone } from '../../components/server-status-ui';
import ServerStatusPageShell from '../../components/ServerStatusPageShell.vue';
import SummaryMetricCard from '../../components/SummaryMetricCard.vue';
import { useMonitorRefreshPreferences } from '../../composables/use-monitor-refresh-preferences';
import type { MonitorRefreshInterval } from '../../contract/refresh';
import type { MonitorTrendRange } from '../../contract/trend';
import { MONITOR_TREND_RANGE } from '../../contract/trend';
import {
  formatDependencyPoolUsage,
  formatPoolCount,
  poolUsagePercent,
  poolUsageStatus,
} from '../../shared/pool-metrics';
import { formatLatency, normalizeDependencyStatus } from '../../shared/server-status-snapshot';
import { formatChartTimeOnly } from '../../shared/time-display';
import type { RequestPerformanceResponse } from '../../types/request-performance';
import type { ServerStatusAnomaly, ServerStatusResponse, ServerStatusTrendPoint } from '../../types/server-status';

defineOptions({
  name: 'MonitorServerStatusOverviewIndex',
});

// 系统运行概览复用同一快照；窄容器只收敛展示层级，不改变监控数据或刷新语义。
echarts.use([TooltipComponent, LegendComponent, GridComponent, MarkLineComponent, LineChart, CanvasRenderer]);
const router = useRouter();

type MonitorStatus = 'healthy' | 'degraded' | 'disabled' | 'unknown';
type MetricCardTone = 'healthy' | 'warning' | 'critical' | 'unknown';
type MetricUsageStatus = 'healthy' | 'warning' | 'danger' | 'unknown';
type MetricUsageKind = 'percent' | 'loadPressure';
type TrendRange = MonitorTrendRange;
type TrendMode = 'overview' | 'multi' | 'focus';
type FocusMetric =
  | 'cpu'
  | 'hostMemory'
  | 'load'
  | 'runtimeAlloc'
  | 'runtimeHeap'
  | 'runtimeSys'
  | 'goroutines'
  | 'networkSend'
  | 'networkReceive'
  | 'networkSendPackets'
  | 'networkReceivePackets'
  | 'diskRead'
  | 'diskWrite'
  | 'diskReadIops'
  | 'diskWriteIops'
  | 'diskReadLatency'
  | 'diskWriteLatency';
type TrendMetricGroup = 'resourceUsage' | 'systemLoad' | 'goRuntime' | 'network' | 'diskIo';
type TrendMetricUnit = '%' | 'load' | 'MB' | 'count' | 'B/s' | 'pps' | 'IOPS' | 'ms';
type TrendMetricAxis = 'percent' | 'load' | 'bytes' | 'count' | 'bytesPerSecond' | 'rate' | 'latency';
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
  | 'multi-networkSend'
  | 'multi-networkReceive'
  | 'multi-networkSendPackets'
  | 'multi-networkReceivePackets'
  | 'multi-diskRead'
  | 'multi-diskWrite'
  | 'multi-diskReadIops'
  | 'multi-diskWriteIops'
  | 'multi-diskReadLatency'
  | 'multi-diskWriteLatency'
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
  usage: MetricCardUsage;
}

interface MetricCardUsage {
  value: number | null;
  label: string;
  status: MetricUsageStatus;
  tooltip: string;
  loading: boolean;
  kind: MetricUsageKind;
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
  values: Array<number | null>;
  currentValue: string;
}

interface HostObservabilityGroup {
  key: 'network' | 'diskIo' | 'tcpProcess';
  title: string;
  metrics: Array<{ key: string; label: string; value: string }>;
}

interface TrendOverviewSection {
  key: 'resourceUsage' | 'systemLoad';
  chartKey: TrendChartKey;
  title: string;
  infoText?: string;
  helperText?: string;
  metrics: TrendMetricDefinition[];
}

interface OverviewDependencyCard {
  key: string;
  name: string;
  description: string;
  status: ServerStatusTone;
  statusLabel: string;
  primaryMetric: DependencyHealthMetric;
  pool: DependencyHealthPool;
}

const { t, locale } = useI18n();
const settingStore = useSettingStore();
const realtimeSchedulerStore = useRealtimeSchedulerStore();
const {
  autoRefreshEnabled,
  refreshIntervalOptions,
  selectedRefreshInterval,
  toggleAutoRefresh: toggleSharedAutoRefresh,
} = useMonitorRefreshPreferences();
const selectedTrendRange = ref<TrendRange>(MONITOR_TREND_RANGE.TEN_MINUTES);
const selectedTrendMode = ref<TrendMode>('overview');
const selectedFocusMetric = ref<FocusMetric>('cpu');
const dashboardVariant = useViewportResponsiveVariant({ layout: 'flow' });
const isCompactDashboard = computed(() => dashboardVariant.value.density === 'compact');
const activeTrendMode = computed<TrendMode>(() => (isCompactDashboard.value ? 'overview' : selectedTrendMode.value));
const consecutiveFailures = ref(0);
const remainingRefreshSeconds = ref<number | null>(null);
const isPageVisible = ref(typeof document === 'undefined' ? true : document.visibilityState === 'visible');

const trendChartRefs = ref<Partial<Record<TrendChartKey, HTMLDivElement | null>>>({});
let refreshTickTimer: number | null = null;
let nextRefreshAt: number | null = null;
const trendCharts = new Map<TrendChartKey, echarts.ECharts>();
let trendChartResizeObserver: ResizeObserver | null = null;

const {
  data: overviewSnapshot,
  isFetching: loading,
  refetch: refetchOverview,
} = useQuery({
  queryKey: computed(() => ['monitor', 'overview', selectedTrendRange.value]),
  queryFn: async () => {
    const [serverStatus, requestPerformance] = await Promise.all([
      getServerStatus(selectedTrendRange.value),
      getRequestPerformance(selectedTrendRange.value),
    ]);
    return { serverStatus, requestPerformance };
  },
  enabled: false,
  retry: false,
});
const serverStatus = computed<ServerStatusResponse | null>(() => overviewSnapshot.value?.serverStatus ?? null);
const requestPerformance = computed<RequestPerformanceResponse | null>(
  () => overviewSnapshot.value?.requestPerformance ?? null,
);

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

const monitorAnomalies = computed<ServerStatusAnomaly[]>(() => serverStatus.value?.anomalies ?? []);

function trendGroupInfoText(group: TrendMetricGroup) {
  switch (group) {
    case 'resourceUsage':
      return t('monitor.serverStatus.trendGroupResourceUsageInfo');
    case 'systemLoad':
      return t('monitor.serverStatus.trendGroupSystemLoadInfo');
    case 'network':
      return t('monitor.serverStatus.trendGroupNetworkInfo');
    case 'diskIo':
      return t('monitor.serverStatus.trendGroupDiskIoInfo');
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
      color: () => readMetricThemeColor('--graft-monitor-cpu-color'),
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
      color: () => readMetricThemeColor('--graft-monitor-memory-color'),
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
      color: () => readMetricThemeColor('--graft-monitor-load-color'),
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
      color: () => readMetricThemeColor('--graft-monitor-runtime-alloc-color'),
      axis: 'bytes',
      description: t('monitor.serverStatus.chartRuntimeAllocDescription'),
      formatter: formatBytes,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-runtimeAlloc',
      currentValue: formatBytes(serverStatus.value?.runtime.runtime_alloc_bytes ?? 0),
      values: points.map((point) => point.runtime_alloc_bytes),
    },
    {
      key: 'runtimeHeap',
      label: t('monitor.serverStatus.chartRuntimeHeap'),
      shortLabel: t('monitor.serverStatus.chartRuntimeHeapShort'),
      unit: 'MB',
      group: 'goRuntime',
      groupLabel: t('monitor.serverStatus.trendGroupGoRuntime'),
      color: () => readMetricThemeColor('--graft-monitor-runtime-heap-color'),
      axis: 'bytes',
      description: t('monitor.serverStatus.chartRuntimeHeapDescription'),
      formatter: formatBytes,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-runtimeHeap',
      currentValue: formatBytes(serverStatus.value?.runtime.runtime_heap_in_use_bytes ?? 0),
      values: points.map((point) => point.runtime_heap_in_use_bytes),
    },
    {
      key: 'runtimeSys',
      label: t('monitor.serverStatus.chartRuntimeSys'),
      shortLabel: t('monitor.serverStatus.chartRuntimeSysShort'),
      unit: 'MB',
      group: 'goRuntime',
      groupLabel: t('monitor.serverStatus.trendGroupGoRuntime'),
      color: () => readMetricThemeColor('--graft-monitor-runtime-sys-color'),
      axis: 'bytes',
      description: t('monitor.serverStatus.chartRuntimeSysDescription'),
      formatter: formatBytes,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-runtimeSys',
      currentValue: formatBytes(serverStatus.value?.runtime.runtime_sys_bytes ?? 0),
      values: points.map((point) => point.runtime_sys_bytes),
    },
    {
      key: 'goroutines',
      label: t('monitor.serverStatus.chartGoroutines'),
      shortLabel: t('monitor.serverStatus.chartGoroutinesShort'),
      unit: 'count',
      group: 'goRuntime',
      groupLabel: t('monitor.serverStatus.trendGroupGoRuntime'),
      color: () => readMetricThemeColor('--graft-monitor-goroutines-color'),
      axis: 'count',
      description: t('monitor.serverStatus.chartGoroutinesDescription'),
      formatter: formatCountValue,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-goroutines',
      currentValue: formatCountValue(serverStatus.value?.runtime.goroutines ?? null),
      values: points.map((point) => point.goroutines),
    },
    {
      key: 'networkSend',
      label: t('monitor.serverStatus.chartNetworkSend'),
      shortLabel: t('monitor.serverStatus.chartNetworkSendShort'),
      unit: 'B/s',
      group: 'network',
      groupLabel: t('monitor.serverStatus.trendGroupNetwork'),
      color: () => readMetricThemeColor('--graft-monitor-cpu-color'),
      axis: 'bytesPerSecond',
      description: t('monitor.serverStatus.chartNetworkSendDescription'),
      formatter: formatBytesPerSecond,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-networkSend',
      infoText: trendGroupInfoText('network'),
      currentValue: formatBytesPerSecond(serverStatus.value?.host_observability.network.sent_bytes_per_second ?? null),
      values: points.map((point) => point.network_sent_bytes_per_second ?? null),
    },
    {
      key: 'networkReceive',
      label: t('monitor.serverStatus.chartNetworkReceive'),
      shortLabel: t('monitor.serverStatus.chartNetworkReceiveShort'),
      unit: 'B/s',
      group: 'network',
      groupLabel: t('monitor.serverStatus.trendGroupNetwork'),
      color: () => readMetricThemeColor('--graft-monitor-memory-color'),
      axis: 'bytesPerSecond',
      description: t('monitor.serverStatus.chartNetworkReceiveDescription'),
      formatter: formatBytesPerSecond,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-networkReceive',
      infoText: trendGroupInfoText('network'),
      currentValue: formatBytesPerSecond(
        serverStatus.value?.host_observability.network.received_bytes_per_second ?? null,
      ),
      values: points.map((point) => point.network_received_bytes_per_second ?? null),
    },
    {
      key: 'networkSendPackets',
      label: t('monitor.serverStatus.chartNetworkSendPackets'),
      shortLabel: t('monitor.serverStatus.chartNetworkSendPacketsShort'),
      unit: 'pps',
      group: 'network',
      groupLabel: t('monitor.serverStatus.trendGroupNetwork'),
      color: () => readMetricThemeColor('--graft-monitor-runtime-alloc-color'),
      axis: 'rate',
      description: t('monitor.serverStatus.chartNetworkSendPacketsDescription'),
      formatter: formatRate,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-networkSendPackets',
      infoText: trendGroupInfoText('network'),
      currentValue: formatRate(serverStatus.value?.host_observability.network.sent_packets_per_second ?? null),
      values: points.map((point) => point.network_sent_packets_per_second ?? null),
    },
    {
      key: 'networkReceivePackets',
      label: t('monitor.serverStatus.chartNetworkReceivePackets'),
      shortLabel: t('monitor.serverStatus.chartNetworkReceivePacketsShort'),
      unit: 'pps',
      group: 'network',
      groupLabel: t('monitor.serverStatus.trendGroupNetwork'),
      color: () => readMetricThemeColor('--graft-monitor-runtime-heap-color'),
      axis: 'rate',
      description: t('monitor.serverStatus.chartNetworkReceivePacketsDescription'),
      formatter: formatRate,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-networkReceivePackets',
      infoText: trendGroupInfoText('network'),
      currentValue: formatRate(serverStatus.value?.host_observability.network.received_packets_per_second ?? null),
      values: points.map((point) => point.network_received_packets_per_second ?? null),
    },
    {
      key: 'diskRead',
      label: t('monitor.serverStatus.chartDiskRead'),
      shortLabel: t('monitor.serverStatus.chartDiskReadShort'),
      unit: 'B/s',
      group: 'diskIo',
      groupLabel: t('monitor.serverStatus.trendGroupDiskIo'),
      color: () => readMetricThemeColor('--graft-monitor-load-color'),
      axis: 'bytesPerSecond',
      description: t('monitor.serverStatus.chartDiskReadDescription'),
      formatter: formatBytesPerSecond,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-diskRead',
      infoText: trendGroupInfoText('diskIo'),
      currentValue: formatBytesPerSecond(serverStatus.value?.host_observability.disk_io.read_bytes_per_second ?? null),
      values: points.map((point) => point.disk_read_bytes_per_second ?? null),
    },
    {
      key: 'diskWrite',
      label: t('monitor.serverStatus.chartDiskWrite'),
      shortLabel: t('monitor.serverStatus.chartDiskWriteShort'),
      unit: 'B/s',
      group: 'diskIo',
      groupLabel: t('monitor.serverStatus.trendGroupDiskIo'),
      color: () => readMetricThemeColor('--graft-monitor-runtime-sys-color'),
      axis: 'bytesPerSecond',
      description: t('monitor.serverStatus.chartDiskWriteDescription'),
      formatter: formatBytesPerSecond,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-diskWrite',
      infoText: trendGroupInfoText('diskIo'),
      currentValue: formatBytesPerSecond(serverStatus.value?.host_observability.disk_io.write_bytes_per_second ?? null),
      values: points.map((point) => point.disk_write_bytes_per_second ?? null),
    },
    {
      key: 'diskReadIops',
      label: t('monitor.serverStatus.chartDiskReadIops'),
      shortLabel: t('monitor.serverStatus.chartDiskReadIopsShort'),
      unit: 'IOPS',
      group: 'diskIo',
      groupLabel: t('monitor.serverStatus.trendGroupDiskIo'),
      color: () => readMetricThemeColor('--graft-monitor-goroutines-color'),
      axis: 'rate',
      description: t('monitor.serverStatus.chartDiskReadIopsDescription'),
      formatter: formatRate,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-diskReadIops',
      infoText: trendGroupInfoText('diskIo'),
      currentValue: formatRate(serverStatus.value?.host_observability.disk_io.read_iops ?? null),
      values: points.map((point) => point.disk_read_iops ?? null),
    },
    {
      key: 'diskWriteIops',
      label: t('monitor.serverStatus.chartDiskWriteIops'),
      shortLabel: t('monitor.serverStatus.chartDiskWriteIopsShort'),
      unit: 'IOPS',
      group: 'diskIo',
      groupLabel: t('monitor.serverStatus.trendGroupDiskIo'),
      color: () => readMetricThemeColor('--td-warning-color-5'),
      axis: 'rate',
      description: t('monitor.serverStatus.chartDiskWriteIopsDescription'),
      formatter: formatRate,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-diskWriteIops',
      infoText: trendGroupInfoText('diskIo'),
      currentValue: formatRate(serverStatus.value?.host_observability.disk_io.write_iops ?? null),
      values: points.map((point) => point.disk_write_iops ?? null),
    },
    {
      key: 'diskReadLatency',
      label: t('monitor.serverStatus.chartDiskReadLatency'),
      shortLabel: t('monitor.serverStatus.chartDiskReadLatencyShort'),
      unit: 'ms',
      group: 'diskIo',
      groupLabel: t('monitor.serverStatus.trendGroupDiskIo'),
      color: () => readMetricThemeColor('--td-success-color-5'),
      axis: 'latency',
      description: t('monitor.serverStatus.chartDiskReadLatencyDescription'),
      formatter: formatLatencyValue,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-diskReadLatency',
      infoText: trendGroupInfoText('diskIo'),
      currentValue: formatLatencyValue(serverStatus.value?.host_observability.disk_io.read_average_latency_ms ?? null),
      values: points.map((point) => point.disk_read_average_latency_ms ?? null),
    },
    {
      key: 'diskWriteLatency',
      label: t('monitor.serverStatus.chartDiskWriteLatency'),
      shortLabel: t('monitor.serverStatus.chartDiskWriteLatencyShort'),
      unit: 'ms',
      group: 'diskIo',
      groupLabel: t('monitor.serverStatus.trendGroupDiskIo'),
      color: () => readMetricThemeColor('--td-error-color-5'),
      axis: 'latency',
      description: t('monitor.serverStatus.chartDiskWriteLatencyDescription'),
      formatter: formatLatencyValue,
      visibleInOverview: false,
      visibleInSmallMultiples: true,
      visibleInFocus: true,
      chartKey: 'multi-diskWriteLatency',
      infoText: trendGroupInfoText('diskIo'),
      currentValue: formatLatencyValue(serverStatus.value?.host_observability.disk_io.write_average_latency_ms ?? null),
      values: points.map((point) => point.disk_write_average_latency_ms ?? null),
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
    t('monitor.serverStatus.trendGroupNetwork'),
    t('monitor.serverStatus.trendGroupDiskIo'),
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
  trendMetricConfigs.value
    .filter((metric) => metric.group === 'goRuntime')
    .slice(0, 4)
    .map((metric) => ({ ...metric, shortLabel: serviceSummaryMetricLabel(metric.key) })),
);
const hostObservabilityGroups = computed<HostObservabilityGroup[]>(() => {
  const host = serverStatus.value?.host_observability;

  return [
    {
      key: 'network',
      title: t('monitor.serverStatus.hostNetworkTitle'),
      metrics: [
        {
          key: 'send-throughput',
          label: t('monitor.serverStatus.hostNetworkSendThroughput'),
          value: formatHostMetric(host?.network.sent_bytes_per_second ?? null, formatBytesPerSecond),
        },
        {
          key: 'receive-throughput',
          label: t('monitor.serverStatus.hostNetworkReceiveThroughput'),
          value: formatHostMetric(host?.network.received_bytes_per_second ?? null, formatBytesPerSecond),
        },
        {
          key: 'send-packet-rate',
          label: t('monitor.serverStatus.hostNetworkSendPacketRate'),
          value: formatHostMetric(host?.network.sent_packets_per_second ?? null, formatPacketRate),
        },
        {
          key: 'receive-packet-rate',
          label: t('monitor.serverStatus.hostNetworkReceivePacketRate'),
          value: formatHostMetric(host?.network.received_packets_per_second ?? null, formatPacketRate),
        },
      ],
    },
    {
      key: 'diskIo',
      title: t('monitor.serverStatus.hostDiskIoTitle'),
      metrics: [
        {
          key: 'read-throughput',
          label: t('monitor.serverStatus.hostDiskReadThroughput'),
          value: formatHostMetric(host?.disk_io.read_bytes_per_second ?? null, formatBytesPerSecond),
        },
        {
          key: 'write-throughput',
          label: t('monitor.serverStatus.hostDiskWriteThroughput'),
          value: formatHostMetric(host?.disk_io.write_bytes_per_second ?? null, formatBytesPerSecond),
        },
        {
          key: 'read-iops',
          label: t('monitor.serverStatus.hostDiskReadIops'),
          value: formatHostMetric(host?.disk_io.read_iops ?? null, formatRate),
        },
        {
          key: 'write-iops',
          label: t('monitor.serverStatus.hostDiskWriteIops'),
          value: formatHostMetric(host?.disk_io.write_iops ?? null, formatRate),
        },
        {
          key: 'read-latency',
          label: t('monitor.serverStatus.hostDiskReadLatency'),
          value: formatHostMetric(host?.disk_io.read_average_latency_ms ?? null, formatLatencyValue),
        },
        {
          key: 'write-latency',
          label: t('monitor.serverStatus.hostDiskWriteLatency'),
          value: formatHostMetric(host?.disk_io.write_average_latency_ms ?? null, formatLatencyValue),
        },
      ],
    },
    {
      key: 'tcpProcess',
      title: t('monitor.serverStatus.hostTcpProcessTitle'),
      metrics: [
        {
          key: 'tcp-total',
          label: t('monitor.serverStatus.hostTcpTotal'),
          value: formatHostMetric(host?.tcp.total ?? null, formatCountValue),
        },
        {
          key: 'tcp-established',
          label: t('monitor.serverStatus.hostTcpEstablished'),
          value: formatHostMetric(host?.tcp.established ?? null, formatCountValue),
        },
        {
          key: 'tcp-time-wait',
          label: t('monitor.serverStatus.hostTcpTimeWait'),
          value: formatHostMetric(host?.tcp.time_wait ?? null, formatCountValue),
        },
        {
          key: 'tcp-close-wait',
          label: t('monitor.serverStatus.hostTcpCloseWait'),
          value: formatHostMetric(host?.tcp.close_wait ?? null, formatCountValue),
        },
        {
          key: 'rss',
          label: t('monitor.serverStatus.hostProcessRss'),
          value: formatHostMetric(host?.process.rss_bytes ?? null, formatBytes),
        },
        {
          key: 'file-descriptors',
          label: t('monitor.serverStatus.hostProcessFileDescriptors'),
          value: formatHostMetric(host?.process.open_file_descriptors ?? null, formatCountValue),
        },
        {
          key: 'threads',
          label: t('monitor.serverStatus.hostProcessThreads'),
          value: formatHostMetric(host?.process.os_threads ?? null, formatCountValue),
        },
      ],
    },
  ];
});
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
  return normalizeStatus(serverStatus.value?.status);
});

const metricCards = computed<MetricCard[]>(() => {
  const response = serverStatus.value;
  if (!response) {
    return [
      emptyMetricCard('load', t('monitor.serverStatus.metricLoadLabel'), 'loadPressure'),
      emptyMetricCard('cpu', t('monitor.serverStatus.metricCpuLabel'), 'percent'),
      emptyMetricCard('memory', t('monitor.serverStatus.metricMemoryLabel'), 'percent'),
      emptyMetricCard('disk', t('monitor.serverStatus.metricDiskLabel'), 'percent'),
    ];
  }

  const loadAverage = response.runtime.load_average;
  const loadPercent =
    response.runtime.cpu_cores > 0 ? Math.min((loadAverage.one_minute / response.runtime.cpu_cores) * 100, 100) : null;
  const cpuPercent = latestTrendPoint.value?.cpu_percent ?? null;
  const hostMemoryPercent = response.runtime.host_memory_used_percent;
  const diskPercent = response.runtime.disk_usage.total_bytes > 0 ? response.runtime.disk_usage.used_percent : null;
  const diskPath = normalizedDiskPath(response.runtime.disk_usage.path);
  const loadStatus = buildMetricCardStatus(resolveAnomalyByKey('system_load_pressure'), {
    hasValue: loadPercent !== null,
    usagePercent: loadPercent,
    healthyDescription: t('monitor.serverStatus.metricLoadDescriptionHealthy'),
    healthyLabel: t('monitor.serverStatus.metricLoadStatusHealthy'),
    warningDescription: t('monitor.serverStatus.metricLoadDescriptionWarning'),
    warningLabel: t('monitor.serverStatus.metricLoadStatusWarning'),
    criticalDescription: t('monitor.serverStatus.metricLoadDescriptionCritical'),
    criticalLabel: t('monitor.serverStatus.metricLoadStatusCritical'),
  });
  const cpuStatus = buildMetricCardStatus(resolveAnomalyByKey('resource_cpu_pressure'), {
    hasValue: cpuPercent !== null,
    usagePercent: cpuPercent,
    healthyDescription: t('monitor.serverStatus.metricCpuDescriptionHealthy'),
    healthyLabel: t('monitor.serverStatus.metricCpuStatusHealthy'),
    warningDescription: t('monitor.serverStatus.metricCpuDescriptionWarning'),
    warningLabel: t('monitor.serverStatus.metricCpuStatusWarning'),
    criticalDescription: t('monitor.serverStatus.metricCpuDescriptionCritical'),
    criticalLabel: t('monitor.serverStatus.metricCpuStatusCritical'),
  });
  const memoryStatus = buildMetricCardStatus(resolveAnomalyByKey('resource_memory_pressure'), {
    hasValue: hostMemoryPercent !== null,
    usagePercent: hostMemoryPercent,
    healthyDescription: t('monitor.serverStatus.metricMemoryDescriptionHealthy'),
    healthyLabel: t('monitor.serverStatus.metricMemoryStatusHealthy'),
    warningDescription: t('monitor.serverStatus.metricMemoryDescriptionWarning'),
    warningLabel: t('monitor.serverStatus.metricMemoryStatusWarning'),
    criticalDescription: t('monitor.serverStatus.metricMemoryDescriptionCritical'),
    criticalLabel: t('monitor.serverStatus.metricMemoryStatusCritical'),
  });
  const diskStatus = buildMetricCardStatus(resolveAnomalyByKey('resource_disk_pressure'), {
    hasValue: diskPercent !== null,
    usagePercent: diskPercent,
    healthyDescription: t('monitor.serverStatus.metricDiskDescriptionHealthy'),
    healthyLabel: t('monitor.serverStatus.metricDiskStatusHealthy'),
    warningDescription: t('monitor.serverStatus.metricDiskDescriptionWarning'),
    warningLabel: t('monitor.serverStatus.metricDiskStatusWarning'),
    criticalDescription: t('monitor.serverStatus.metricDiskDescriptionCritical'),
    criticalLabel: t('monitor.serverStatus.metricDiskStatusCritical'),
  });

  return [
    {
      key: 'load',
      label: t('monitor.serverStatus.metricLoadLabel'),
      value: formatLoadAverage(loadAverage.one_minute),
      valueSide: t('monitor.serverStatus.metricLoadValueSide'),
      meta: t('monitor.serverStatus.metricLoadMeta', {
        load: formatLoadAverage(loadAverage.one_minute),
        cores: String(response.runtime.cpu_cores),
        percent: formatPercentPrecise(loadPercent),
      }),
      ...loadStatus,
      usage: buildMetricUsage({
        kind: 'loadPressure',
        label: t('monitor.serverStatus.metricLoadLabel'),
        value: loadPercent,
        tone: loadStatus.tone,
        loadAverage: loadAverage.one_minute,
        cpuCores: response.runtime.cpu_cores,
      }),
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
      ...cpuStatus,
      usage: buildMetricUsage({
        kind: 'percent',
        label: t('monitor.serverStatus.metricCpuLabel'),
        value: cpuPercent,
        tone: cpuStatus.tone,
      }),
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
      ...memoryStatus,
      usage: buildMetricUsage({
        kind: 'percent',
        label: t('monitor.serverStatus.metricMemoryLabel'),
        value: hostMemoryPercent,
        tone: memoryStatus.tone,
      }),
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
      ...diskStatus,
      usage: buildMetricUsage({
        kind: 'percent',
        label: t('monitor.serverStatus.metricDiskLabel'),
        value: diskPercent,
        tone: diskStatus.tone,
      }),
    },
  ];
});

const requestPerformanceMetrics = computed(() => {
  const summary = requestPerformance.value?.summary;
  return [
    {
      key: 'rps',
      label: t('monitor.requestPerformance.summary.rps'),
      value: summary ? `${summary.requests_per_second.toFixed(2)} RPS` : '--',
    },
    {
      key: 'latency',
      label: t('monitor.requestPerformance.summary.latency'),
      value: summary
        ? `${t('monitor.requestPerformance.summary.p50')} ${summary.p50_latency_ms.toFixed(0)} ms / ${t('monitor.requestPerformance.summary.p95')} ${summary.p95_latency_ms.toFixed(0)} ms`
        : '--',
    },
    {
      key: 'errors',
      label: t('monitor.requestPerformance.summary.errors'),
      value: summary ? formatRequestErrorRate(summary.error_5xx_rate) : '--',
    },
    {
      key: 'slow',
      label: t('monitor.requestPerformance.summary.slow'),
      value: summary ? String(summary.slow_request_count) : '--',
    },
  ];
});
const overviewDependencyCards = computed<OverviewDependencyCard[]>(() => {
  const dependencies = serverStatus.value?.dependencies;

  return [
    buildOverviewDependencyCard({
      key: 'postgresql',
      name: t('monitor.serverStatus.postgresqlLabel'),
      description: t('monitor.dependenciesPage.postgresqlSubtitle'),
      dependency: dependencies?.database,
    }),
    buildOverviewDependencyCard({
      key: 'redis',
      name: t('monitor.serverStatus.redisLabel'),
      description: t('monitor.dependenciesPage.redisSubtitle'),
      dependency: dependencies?.redis,
    }),
  ];
});
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

const refreshControlStatus = computed(() => {
  if (canRunAutoRefreshCycle()) {
    return 'running' as const;
  }
  return selectedRefreshInterval.value <= 0 ? ('off' as const) : ('paused' as const);
});

function canRunAutoRefreshCycle() {
  return (
    autoRefreshEnabled.value &&
    isPageVisible.value &&
    selectedRefreshInterval.value > 0 &&
    realtimeSchedulerStore.allowPolling
  );
}

function clearRefreshSchedule() {
  stopRefreshTick();
  remainingRefreshSeconds.value = null;
}

let isMounted = false;

async function fetchServerStatus(options: { manual?: boolean } = {}) {
  if (!isMounted) return;
  stopRefreshTick();
  const result = await refetchOverview();
  if (!isMounted) return;
  if (!result.error) {
    consecutiveFailures.value = 0;
  } else {
    const previousFailures = consecutiveFailures.value;
    consecutiveFailures.value += 1;

    if (options.manual || previousFailures === 0) {
      const message = resolveLocalizedErrorMessage(t, result.error, t('monitor.serverStatus.loadFailed'));
      openCorrelationErrorNotification({
        router,
        title: t('audit.correlation.errorTitle'),
        message,
        requestId: requestIdFromError(result.error),
        translate: t,
      });
    }
  }
  scheduleNextRefresh();
}

function toggleAutoRefresh() {
  toggleSharedAutoRefresh();

  if (canRunAutoRefreshCycle()) {
    void fetchServerStatus({ manual: true });
    return;
  }

  clearRefreshSchedule();
}

function scheduleNextRefresh() {
  stopRefreshTick();
  if (!canRunAutoRefreshCycle()) {
    clearRefreshSchedule();
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
  if (canRunAutoRefreshCycle()) {
    void fetchServerStatus();
    return;
  }

  clearRefreshSchedule();
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

function emptyMetricCard(key: string, label: string, kind: MetricUsageKind): MetricCard {
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
    usage: buildMetricUsage({
      kind,
      label,
      value: null,
      loading: loading.value,
    }),
  };
}

function openDependencies() {
  void router.push({ path: '/observability/dependencies' });
}

function openServiceStatus() {
  void router.push({ path: '/observability/service-status' });
}

function openRequestPerformance() {
  void router.push({ path: '/observability/request-performance' });
}

function serviceSummaryMetricLabel(metric: FocusMetric) {
  switch (metric) {
    case 'runtimeAlloc':
      return t('monitor.serverStatus.serviceSummaryRuntimeAlloc');
    case 'runtimeHeap':
      return t('monitor.serverStatus.serviceSummaryRuntimeHeap');
    case 'runtimeSys':
      return t('monitor.serverStatus.serviceSummaryRuntimeSys');
    case 'goroutines':
      return t('monitor.serverStatus.serviceSummaryGoroutines');
    default:
      return '';
  }
}

function buildOverviewDependencyCard(options: {
  key: string;
  name: string;
  description: string;
  dependency?: ServerStatusResponse['dependencies']['database'];
}): OverviewDependencyCard {
  const pool = options.dependency?.pool;
  const usagePercent = pool ? poolUsagePercent(pool) : null;
  const usageText = pool ? formatDependencyPoolUsage(pool, dependencyNoDataText()) : dependencyNoDataText();
  const usagePercentText = usagePercent === null ? dependencyNoDataText() : `${usagePercent.toFixed(0)}%`;
  const status = overviewDependencyTone(options.dependency?.status ?? undefined);

  return {
    key: options.key,
    name: options.name,
    description: options.description,
    status,
    statusLabel: overviewDependencyStatusLabel(status),
    primaryMetric: {
      label: t('monitor.dependenciesPage.fields.latency'),
      value: formatLatency(options.dependency?.latency_ms),
      description: t('monitor.dependenciesPage.fieldDescriptions.latency'),
    },
    pool: {
      title: t('monitor.dependenciesPage.pool.title'),
      stateTitle: t('monitor.dependenciesPage.pool.stateTitle'),
      usageText,
      usagePercent,
      usagePercentText,
      usageStatus: poolUsageStatus(usagePercent),
      usageLabel: t('monitor.dependenciesPage.pool.usageLabel', { label: options.name }),
      usageTooltip: t('monitor.dependenciesPage.pool.usageTooltip', {
        label: options.name,
        value: usageText,
        percent: usagePercentText,
      }),
      summary: overviewDependencyPoolSummary(usagePercent),
      emptyText: dependencyNoDataText(),
      items: [
        {
          key: 'inUse',
          label: t('monitor.dependenciesPage.pool.inUse'),
          value: formatPoolCount(pool?.in_use_connections, dependencyNoDataText()),
        },
        {
          key: 'idle',
          label: t('monitor.dependenciesPage.pool.idle'),
          value: formatPoolCount(pool?.idle_connections, dependencyNoDataText()),
        },
        {
          key: 'open',
          label: t('monitor.dependenciesPage.pool.open'),
          value: formatPoolCount(pool?.open_connections, dependencyNoDataText()),
        },
        {
          key: 'capacity',
          label: t('monitor.dependenciesPage.pool.capacity'),
          value: formatPoolCount(pool?.capacity, dependencyNoDataText()),
        },
      ],
    },
  };
}

function overviewDependencyTone(status?: string | null): ServerStatusTone {
  switch (normalizeDependencyStatus(status ?? undefined)) {
    case 'healthy':
      return 'healthy';
    case 'abnormal':
      return 'error';
    case 'notConfigured':
      return 'disabled';
    default:
      return 'unknown';
  }
}

function overviewDependencyStatusLabel(status: ServerStatusTone) {
  switch (status) {
    case 'healthy':
      return t('monitor.dependenciesPage.statusHealthy');
    case 'error':
      return t('monitor.dependenciesPage.statusAbnormal');
    case 'disabled':
      return t('monitor.dependenciesPage.statusNotConfigured');
    default:
      return t('monitor.dependenciesPage.statusUnknown');
  }
}

function overviewDependencyPoolSummary(percent: number | null) {
  switch (poolUsageStatus(percent)) {
    case 'danger':
      return t('monitor.dependenciesPage.pool.riskCritical');
    case 'warning':
      return t('monitor.dependenciesPage.pool.riskWarning');
    case 'healthy':
      return t('monitor.dependenciesPage.pool.riskHealthy');
    default:
      return t('monitor.dependenciesPage.pool.riskUnknown');
  }
}

function dependencyNoDataText() {
  return t('monitor.serverStatus.metricUsageNoData');
}

function formatRequestErrorRate(value: number) {
  return `${value.toFixed(2)}%`;
}

function resolveAnomalyByKey(anomalyKey: string) {
  return monitorAnomalies.value.find((anomaly) => anomaly.anomaly_key === anomalyKey);
}

function normalizedDiskPath(path?: string | null) {
  if (!path) {
    return t('monitor.serverStatus.diskRootPath');
  }
  return path;
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

function buildMetricCardStatus(
  anomaly: ServerStatusAnomaly | undefined,
  copy: {
    hasValue: boolean;
    usagePercent: number | null;
    healthyDescription: string;
    healthyLabel: string;
    warningDescription: string;
    warningLabel: string;
    criticalDescription: string;
    criticalLabel: string;
  },
): Pick<MetricCard, 'description' | 'statusLabel' | 'tagTheme' | 'tone'> {
  if (anomaly?.severity === 'critical') {
    return {
      tone: 'critical',
      statusLabel: copy.criticalLabel,
      description: localizedAnomalyDescription(anomaly) || copy.criticalDescription,
      tagTheme: metricCardTagTheme('critical'),
    };
  }
  if (anomaly?.severity === 'warning') {
    return {
      tone: 'warning',
      statusLabel: copy.warningLabel,
      description: localizedAnomalyDescription(anomaly) || copy.warningDescription,
      tagTheme: metricCardTagTheme('warning'),
    };
  }
  if (!copy.hasValue) {
    return {
      tone: 'unknown',
      statusLabel: t('monitor.serverStatus.statusUnknown'),
      description: t('monitor.serverStatus.emptyMetric.description'),
      tagTheme: metricCardTagTheme('unknown'),
    };
  }
  const thresholdTone = metricUsageTone(copy.usagePercent);
  if (thresholdTone === 'critical') {
    return {
      tone: 'critical',
      statusLabel: copy.criticalLabel,
      description: copy.criticalDescription,
      tagTheme: metricCardTagTheme('critical'),
    };
  }
  if (thresholdTone === 'warning') {
    return {
      tone: 'warning',
      statusLabel: copy.warningLabel,
      description: copy.warningDescription,
      tagTheme: metricCardTagTheme('warning'),
    };
  }

  return {
    tone: 'healthy',
    statusLabel: copy.healthyLabel,
    description: copy.healthyDescription,
    tagTheme: metricCardTagTheme('healthy'),
  };
}

function localizedAnomalyDescription(anomaly: ServerStatusAnomaly) {
  if (!anomaly.summary_key) {
    return '';
  }

  return t(anomaly.summary_key, anomaly.summary_params ?? {});
}

function metricUsageTone(percent: number | null): MetricCardTone {
  if (percent === null || Number.isNaN(percent)) {
    return 'unknown';
  }
  if (percent >= 85) {
    return 'critical';
  }
  if (percent >= 70) {
    return 'warning';
  }

  return 'healthy';
}

function metricUsageStatus(percent: number | null): MetricUsageStatus {
  const tone = metricUsageTone(percent);
  return metricToneToUsageStatus(tone);
}

function metricToneToUsageStatus(tone: MetricCardTone): MetricUsageStatus {
  switch (tone) {
    case 'healthy':
      return 'healthy';
    case 'warning':
      return 'warning';
    case 'critical':
      return 'danger';
    default:
      return 'unknown';
  }
}

function buildMetricUsage(options: {
  kind: MetricUsageKind;
  label: string;
  value: number | null;
  tone?: MetricCardTone;
  loading?: boolean;
  loadAverage?: number;
  cpuCores?: number;
}): MetricCardUsage {
  const hasValue = options.value !== null && Number.isFinite(options.value);
  const status = options.tone ? metricToneToUsageStatus(options.tone) : metricUsageStatus(options.value);
  let tooltip = t('monitor.serverStatus.metricUsageNoData');

  if (hasValue && options.kind === 'loadPressure') {
    tooltip = t('monitor.serverStatus.metricLoadUsageTooltip', {
      load: formatLoadAverage(options.loadAverage ?? null),
      cores: String(options.cpuCores ?? 0),
      percent: formatPercentPrecise(options.value),
    });
  } else if (hasValue) {
    tooltip = t('monitor.serverStatus.metricPercentUsageTooltip', {
      label: options.label,
      percent: formatPercentPrecise(options.value),
    });
  }

  return {
    kind: options.kind,
    label: options.label,
    value: hasValue ? options.value : null,
    status,
    tooltip,
    loading: options.loading ?? false,
  };
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
  const previous = trendChartRefs.value[key];
  if (previous && trendChartResizeObserver) {
    trendChartResizeObserver.unobserve(previous);
  }

  const nextElement = el instanceof HTMLDivElement ? el : null;
  trendChartRefs.value[key] = nextElement;

  if (nextElement && trendChartResizeObserver) {
    trendChartResizeObserver.observe(nextElement);
  }
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

function ensureTrendChartResizeObserver() {
  if (trendChartResizeObserver || typeof ResizeObserver === 'undefined') {
    return;
  }

  trendChartResizeObserver = new ResizeObserver(() => {
    resizeTrendChart();
  });
}

function reconnectTrendChartResizeObserver() {
  if (!trendChartResizeObserver) {
    return;
  }

  trendChartResizeObserver.disconnect();
  Object.values(trendChartRefs.value).forEach((element) => {
    if (element) {
      trendChartResizeObserver?.observe(element);
    }
  });
}

function buildTrendChartOptions(points: ServerStatusTrendPoint[], chartColors: TChartColor) {
  const metrics = trendMetricConfigs.value;
  const labels = points.map((point) => formatChartTimestamp(point.observed_at));

  if (activeTrendMode.value === 'overview') {
    return overviewTrendSections.value.map((section) => ({
      key: section.chartKey,
      option:
        section.key === 'resourceUsage'
          ? buildOverviewUsageChartOption(labels, section.metrics, chartColors)
          : buildOverviewLoadChartOption(labels, section.metrics, chartColors),
    }));
  }

  if (activeTrendMode.value === 'focus') {
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
    xAxis: buildXAxis(labels, chartColors, isCompactDashboard.value),
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
    xAxis: buildXAxis(labels, chartColors, isCompactDashboard.value),
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
    xAxis: buildXAxis(labels, chartColors, isCompactDashboard.value),
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
    xAxis: buildXAxis(labels, chartColors, isCompactDashboard.value),
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
    formatter: (params: Array<{ axisValueLabel: string; seriesName: string; color: string; data: number | null }>) => {
      const rows = params
        .map((param) => {
          const metric = metricMap.get(param.seriesName);
          if (!metric) {
            return '';
          }

          const valueLabel = formatTrendValue(metric, param.data);

          return [
            `<div style="display:flex;align-items:center;justify-content:space-between;gap: var(--graft-density-gap-16);">`,
            `<span style="display:flex;align-items:center;gap: var(--graft-density-gap-8);">`,
            `<i style="width:8px;height:8px;border-radius:999px;background:${param.color};display:inline-block;"></i>`,
            `<span>${metric.label}</span>`,
            `</span>`,
            `<strong>${valueLabel}</strong>`,
            `</div>`,
          ].join('');
        })
        .filter(Boolean)
        .join('');

      return `<div style="display:flex;flex-direction:column;gap: var(--graft-density-gap-8);"><strong>${params[0]?.axisValueLabel ?? ''}</strong>${rows}</div>`;
    },
  };
}

function buildXAxis(labels: string[], chartColors: TChartColor, compact: boolean) {
  return {
    type: 'category',
    data: labels,
    axisLabel: {
      color: chartColors.placeholderColor,
      interval: compact ? compactTrendAxisLabelInterval(labels.length) : 'auto',
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

function compactTrendAxisLabelInterval(labelCount: number) {
  if (labelCount <= 4) {
    return 0;
  }

  const lastIndex = labelCount - 1;
  const step = Math.ceil(lastIndex / 3);
  return (index: number) => index === 0 || index === lastIndex || index % step === 0;
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
    case 'bytesPerSecond':
      return buildYAxis(metric.unit, 'bytesPerSecond', chartColors);
    case 'rate':
      return buildYAxis(metric.unit, 'rate', chartColors);
    case 'latency':
      return buildYAxis(metric.unit, 'latency', chartColors);
    default:
      return buildYAxis(metric.unit, 'count', chartColors);
  }
}

function buildYAxis(
  name: string,
  axisType: TrendMetricAxis,
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
    data: metric.values.map((value) => formatTrendDataValue(metric.axis, value)),
  };
}

function formatTrendDataValue(axis: TrendMetricAxis, value: number | null) {
  if (value === null || !Number.isFinite(value)) {
    return null;
  }

  return Number(axis === 'bytes' ? (value / 1024 / 1024).toFixed(2) : value.toFixed(2));
}

function formatTrendValue(metric: TrendMetricDefinition, value: number | null) {
  if (metric.axis === 'bytes') {
    return value === null ? '--' : `${value.toFixed(1)} MB`;
  }
  return metric.formatter(value);
}

function formatAxisValue(value: number, axisType: TrendMetricAxis) {
  switch (axisType) {
    case 'percent':
      return `${value}%`;
    case 'load':
      return value.toFixed(1);
    case 'bytes':
      return `${value} MB`;
    case 'bytesPerSecond':
      return formatBytesPerSecond(value);
    case 'rate':
      return formatRate(value);
    case 'latency':
      return formatLatencyValue(value);
    default:
      return `${value}`;
  }
}

function readMetricThemeColor(token: string) {
  void settingStore.resolvedThemeTokensForDisplayMode;
  return readThemeToken(token);
}

function readThemeToken(token: string) {
  const value = getComputedStyle(document.documentElement).getPropertyValue(token).trim();
  return value || `var(${token})`;
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

function formatChartTimestamp(value: string) {
  return formatChartTimeOnly(value, locale) || t('monitor.serverStatus.runtimeStatusNotAvailable');
}

function formatBytes(bytes: number | null) {
  if (bytes === null || Number.isNaN(bytes)) {
    return '--';
  }
  if (bytes === 0) {
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

function formatBytesPerSecond(value: number | null) {
  const formatted = formatBytes(value);
  return formatted === '--' ? formatted : `${formatted}/s`;
}

function formatRate(value: number | null) {
  if (value === null || Number.isNaN(value)) {
    return '--';
  }
  return value.toFixed(value >= 10 ? 0 : 1);
}

function formatPacketRate(value: number | null) {
  const formatted = formatRate(value);
  return formatted === '--' ? formatted : `${formatted} pps`;
}

function formatHostMetric(value: number | null, formatter: (metricValue: number | null) => string) {
  if (value === null || Number.isNaN(value)) {
    return t('monitor.serverStatus.metricUsageNoData');
  }
  return formatter(value);
}

function formatLatencyValue(value: number | null) {
  if (value === null || Number.isNaN(value)) {
    return '--';
  }
  return `${value.toFixed(value >= 10 ? 0 : 2)} ms`;
}

watch(
  [
    () => trendPoints.value,
    () => trendMetricConfigs.value,
    () => activeTrendMode.value,
    () => isCompactDashboard.value,
    () => selectedFocusMetric.value,
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

watch(
  [
    () => settingStore.layout,
    () => settingStore.splitMenu,
    () => settingStore.isSidebarCompact,
    () => settingStore.isSidebarFixed,
    () => settingStore.showHeader,
  ],
  async () => {
    await nextTick();
    reconnectTrendChartResizeObserver();
    resizeTrendChart();
  },
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

watch(
  () => realtimeSchedulerStore.allowPolling,
  (allowPolling) => {
    if (!allowPolling) {
      clearRefreshSchedule();
      return;
    }
    scheduleNextRefresh();
  },
);

onMounted(async () => {
  isMounted = true;
  await fetchServerStatus();
  await nextTick();
  ensureTrendChartResizeObserver();
  reconnectTrendChartResizeObserver();
  syncTrendChart();
  window.addEventListener('resize', resizeTrendChart, false);
  document.addEventListener('visibilitychange', handleVisibilityChange, false);
});

onUnmounted(() => {
  isMounted = false;
  stopRefreshTick();
  window.removeEventListener('resize', resizeTrendChart);
  document.removeEventListener('visibilitychange', handleVisibilityChange);
  trendChartResizeObserver?.disconnect();
  trendChartResizeObserver = null;
  disposeTrendChart();
});
</script>
<style lang="less" scoped>
@import './index.less';
</style>
