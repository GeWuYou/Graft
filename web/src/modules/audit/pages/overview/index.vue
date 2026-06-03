<template>
  <div class="audit-overview" data-page-type="overview-dashboard">
    <governance-dashboard-shell
      domain="audit"
      :eyebrow="t('menu.audit.title')"
      :title="t('audit.overview.title')"
      :description="t('audit.overview.description')"
    >
      <template #actions>
        <t-space size="small" wrap>
          <t-radio-group v-model="activeWindow" size="small" variant="default-filled">
            <t-radio-button v-for="option in timeRangeOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </t-radio-button>
          </t-radio-group>
          <t-button theme="default" variant="outline" :loading="loading" @click="fetchOverview">
            {{ t('audit.overview.refresh') }}
          </t-button>
        </t-space>
      </template>

      <management-empty-state
        v-if="errorMessage && !loading"
        tone="error"
        :title="t('audit.overview.errorTitle')"
        :description="errorMessage"
      >
        <template #actions>
          <t-button theme="primary" variant="outline" @click="fetchOverview">
            {{ t('audit.overview.retry') }}
          </t-button>
        </template>
      </management-empty-state>

      <template #summary>
        <button
          v-for="item in stats"
          :key="item.key"
          class="audit-overview__summary-action"
          type="button"
          @click="openSummary(item.key)"
        >
          <governance-summary-card kind="activity" :title="item.title" :value="item.value" :value-aside="item.unit" />
        </button>
      </template>

      <section class="audit-overview__grid">
        <governance-section :title="t('audit.overview.sections.failedAuth')">
          <div class="audit-overview__list">
            <article v-for="item in failedAuthItems" :key="item.key" class="audit-overview__list-item">
              <div>
                <strong>{{ item.actor }}</strong>
                <p>{{ item.resource }}</p>
              </div>
              <div class="audit-overview__item-meta">
                <span>{{ item.time }}</span>
                <t-tag theme="danger" variant="light-outline" size="small">{{ item.result }}</t-tag>
              </div>
            </article>
          </div>
        </governance-section>

        <governance-section :title="t('audit.overview.sections.permissionDenied')">
          <div class="audit-overview__list">
            <article v-for="item in permissionDeniedItems" :key="item.key" class="audit-overview__list-item">
              <div>
                <strong>{{ item.actor }}</strong>
                <p>{{ item.resource }}</p>
              </div>
              <div class="audit-overview__item-meta">
                <span>{{ item.time }}</span>
                <t-tag theme="warning" variant="light-outline" size="small">{{ item.result }}</t-tag>
              </div>
            </article>
          </div>
        </governance-section>
      </section>

      <section class="audit-overview__grid audit-overview__grid--bottom">
        <governance-section :title="t('audit.overview.sections.sensitiveOps')">
          <div class="audit-overview__list">
            <article v-for="item in sensitiveItems" :key="item.key" class="audit-overview__list-item">
              <div>
                <strong>{{ item.actor }}</strong>
                <p>{{ item.resource }}</p>
              </div>
              <div class="audit-overview__item-meta">
                <span>{{ item.time }}</span>
                <t-tag theme="warning" variant="light-outline" size="small">{{ item.result }}</t-tag>
              </div>
            </article>
          </div>
        </governance-section>

        <div class="audit-overview__stack">
          <governance-section :title="t('audit.overview.sections.riskWatch')">
            <div class="audit-overview__watch-list">
              <article v-for="group in riskGroups" :key="group.key" class="audit-overview__watch-item">
                <div class="audit-overview__watch-content">
                  <strong>{{ t(group.label_key) }}</strong>
                  <p>{{ t('audit.overview.riskGroups.meta', { count: group.count }) }}</p>
                </div>
                <div class="audit-overview__watch-actions">
                  <t-tag :theme="riskTheme(group.risk_level)" variant="light-outline" size="small">
                    {{ t(`audit.common.risk.${group.risk_level}`) }}
                  </t-tag>
                  <t-button size="small" theme="primary" variant="text" @click="openRiskGroup(group.key)">
                    {{ riskGroupActionLabel }}
                  </t-button>
                </div>
              </article>
            </div>
          </governance-section>

          <governance-section :title="t('audit.overview.sections.shortcuts')">
            <div class="audit-overview__shortcut-list">
              <button
                v-for="entry in shortcuts"
                :key="entry.key"
                class="audit-overview__shortcut"
                type="button"
                @click="openShortcut(entry.query)"
              >
                <strong>{{ entry.title }}</strong>
                <span>{{ entry.description }}</span>
              </button>
            </div>
          </governance-section>
        </div>
      </section>

      <section class="audit-overview__grid audit-overview__grid--bottom">
        <governance-section :title="t('audit.overview.sections.trend')">
          <management-empty-state
            v-if="!trendView.isRenderable"
            class="audit-overview__trend-empty"
            :title="t('audit.overview.trend.emptyTitle')"
            :description="t('audit.overview.trend.emptyDescription')"
          />
          <div v-else class="audit-overview__trend-panel">
            <div class="audit-overview__trend-metrics">
              <article v-for="item in trendSummaryItems" :key="item.key" class="audit-overview__trend-metric">
                <span>{{ item.label }}</span>
                <strong>{{ item.value }}</strong>
              </article>
            </div>
            <div
              ref="trendChartRef"
              class="audit-overview__trend-chart"
              data-audit-trend-chart="true"
              role="img"
              :aria-label="t('audit.overview.sections.trend')"
            />
          </div>
        </governance-section>

        <governance-section :title="t('audit.overview.sections.securityTimeline')">
          <t-timeline class="audit-overview__timeline" mode="same">
            <t-timeline-item
              v-for="item in securityTimeline"
              :key="item.id"
              :label="formatTime(item.created_at)"
              :dot-color="timelineDotColor(item.risk_level)"
            >
              <div class="audit-overview__timeline-item">
                <strong>{{ actionTitle(item, t) }}</strong>
                <p>{{ resourceLabel(item, t) }}</p>
                <div class="audit-overview__timeline-meta">
                  <t-tag :theme="riskTheme(item.risk_level)" variant="light-outline" size="small">
                    {{ t(`audit.common.risk.${item.risk_level}`) }}
                  </t-tag>
                  <t-tag theme="default" variant="light-outline" size="small">
                    {{ t(`audit.common.source.${item.source}`) }}
                  </t-tag>
                </div>
                <div class="audit-overview__timeline-actions">
                  <t-button
                    v-if="item.request_id"
                    size="small"
                    theme="default"
                    variant="text"
                    @click="openSecurityTimelineRequest(item.request_id)"
                  >
                    {{ relatedRequestActionLabel }}
                  </t-button>
                </div>
              </div>
            </t-timeline-item>
          </t-timeline>
        </governance-section>
      </section>
    </governance-dashboard-shell>
  </div>
</template>
<script setup lang="ts">
import { LineChart } from 'echarts/charts';
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components';
import type { EChartsCoreOption } from 'echarts/core';
import * as echarts from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { buildAccessLogRequestLocation } from '@/modules/access-log/contract/deep-link';
import { buildAuditLogsLocation } from '@/modules/audit/contract/deep-link';
import { AUDIT_DRILLDOWN_SCOPE } from '@/modules/audit/contract/presets';
import { AUDIT_TIME_PRESET, type AuditTimePreset } from '@/modules/audit/contract/time-presets';
import { openCorrelationErrorNotification, requestIdFromError } from '@/modules/audit/shared/correlation-actions';
import { resolveLocalizedErrorMessage } from '@/modules/shared/localized-api-error';
import { GovernanceDashboardShell, GovernanceSection, GovernanceSummaryCard } from '@/shared/components/governance';
import { ManagementEmptyState } from '@/shared/components/management';
import {
  buildRecentHoursLocalRange,
  buildTrendAxisLabels,
  formatLocaleDateTime,
  formatTrendTooltipDateTime,
  localDateTimeToUtcIso,
} from '@/shared/observability';
import { useSettingStore } from '@/store';
import { createLogger } from '@/utils/logger';

import { getAuditOverview } from '../../api/audit';
import { actionTitle, resourceLabel } from '../../shared/presentation';
import type { AuditOverviewItem, AuditOverviewResponse } from '../../types/audit';

defineOptions({
  name: 'AuditOverviewIndex',
});

echarts.use([TooltipComponent, LegendComponent, GridComponent, LineChart, CanvasRenderer]);

const { locale, t } = useI18n();
const router = useRouter();
const logger = createLogger('audit.overview');
const settingStore = useSettingStore();
const activeWindow = ref<AuditTimePreset>(AUDIT_TIME_PRESET.LAST_24H);
const loading = ref(false);
const errorMessage = ref('');
const overview = ref<AuditOverviewResponse | null>(null);
const trendChartRef = ref<HTMLDivElement | null>(null);
let trendChart: echarts.ECharts | null = null;
let trendChartResizeObserver: ResizeObserver | null = null;

const timeRangeOptions = computed(() => [
  { label: t('audit.overview.timeRanges.24h'), value: AUDIT_TIME_PRESET.LAST_24H },
  { label: t('audit.overview.timeRanges.7d'), value: AUDIT_TIME_PRESET.LAST_7D },
  { label: t('audit.overview.timeRanges.30d'), value: AUDIT_TIME_PRESET.LAST_30D },
]);

const stats = computed(() => [
  {
    key: 'total',
    title: t('audit.overview.stats.totalLogs.title'),
    value: String(overview.value?.summary.total_logs ?? 0),
    unit: t('audit.overview.stats.totalLogs.unit'),
  },
  {
    key: 'failed',
    title: t('audit.overview.stats.failedWindow.title'),
    value: String(overview.value?.summary.failed_operations ?? 0),
    unit: t('audit.overview.stats.failedWindow.unit'),
  },
  {
    key: 'risk',
    title: t('audit.overview.stats.highRisk.title'),
    value: String(overview.value?.summary.high_risk_events ?? 0),
    unit: t('audit.overview.stats.highRisk.unit'),
  },
  {
    key: 'sensitive',
    title: t('audit.overview.stats.sensitiveOps.title'),
    value: String(overview.value?.summary.sensitive_operations ?? 0),
    unit: t('audit.overview.stats.sensitiveOps.unit'),
  },
]);

const failedAuthItems = computed(() =>
  toOverviewCards(overview.value?.failed_auth, t('audit.overview.itemResult.failed')),
);

const permissionDeniedItems = computed(() =>
  toOverviewCards(overview.value?.permission_denied, t('audit.overview.itemResult.denied')),
);

const sensitiveItems = computed(() =>
  toOverviewCards(overview.value?.sensitive_operations, t('audit.overview.itemResult.sensitive')),
);

const riskGroups = computed(() => overview.value?.risk_groups ?? []);
const securityTimeline = computed(() => overview.value?.security_timeline ?? []);
const securityEventCount = computed(() =>
  (overview.value?.trend?.points ?? []).reduce((total, point) => total + (point.security_events ?? 0), 0),
);
const trendSummaryItems = computed(() => [
  {
    key: 'total',
    label: t('audit.overview.trend.totalMetric'),
    value: String(overview.value?.summary.total_logs ?? 0),
  },
  {
    key: 'risk',
    label: t('audit.overview.trend.highRiskMetric'),
    value: String(overview.value?.summary.high_risk_events ?? 0),
  },
  {
    key: 'security',
    label: t('audit.overview.trend.securityMetric'),
    value: String(securityEventCount.value),
  },
]);
const trendView = computed(() => {
  const points = overview.value?.trend?.points ?? [];
  const activePoints = points.filter((point) => point.total > 0);
  const hasEnoughPoints = points.length >= 3;
  const hasEnoughActivity = activePoints.length >= 3;

  if (!hasEnoughPoints || !hasEnoughActivity) {
    return {
      isRenderable: false,
      points: [],
    };
  }

  const axisLabels = buildTrendAxisLabels(
    points.map((point) => ({
      key: `${point.bucket_start}-${point.bucket_end}`,
      start: point.bucket_start,
      end: point.bucket_end,
    })),
    activeWindow.value,
    locale.value,
  );
  const normalizedPoints = points.map((point, index) => ({
    key: `${point.bucket_start}-${point.bucket_end}`,
    axisLabel: axisLabels[index]?.axisLabel ?? '',
    tooltipLabel: formatTrendTooltipLabel(point.bucket_start, point.bucket_end),
    total: point.total,
    highRisk: point.high_risk,
    security: point.security_events,
  }));

  return {
    isRenderable: true,
    points: normalizedPoints,
  };
});

const shortcuts = computed(() => [
  {
    key: 'failed',
    title: t('audit.overview.shortcuts.failedAuth.title'),
    description: t('audit.overview.shortcuts.failedAuth.description'),
    query: buildOverviewAuditQuery({
      scope: AUDIT_DRILLDOWN_SCOPE.AUTH_FAILURES,
    }),
  },
  {
    key: 'rbac',
    title: t('audit.overview.shortcuts.rbacChanges.title'),
    description: t('audit.overview.shortcuts.rbacChanges.description'),
    query: buildOverviewAuditQuery({
      scope: AUDIT_DRILLDOWN_SCOPE.RBAC_CHANGES,
    }),
  },
  {
    key: 'sensitive',
    title: t('audit.overview.shortcuts.sensitiveOps.title'),
    description: t('audit.overview.shortcuts.sensitiveOps.description'),
    query: buildOverviewAuditQuery({
      scope: AUDIT_DRILLDOWN_SCOPE.SENSITIVE_OPERATIONS,
    }),
  },
]);

const riskGroupActionLabel = computed(() => t('audit.overview.riskGroups.action'));
const relatedRequestActionLabel = computed(() => t('audit.logList.drawer.actions.viewRelatedRequest'));

function buildOverviewAuditQuery(query: Record<string, string>) {
  return {
    ...buildFrozenOverviewWindow(),
    ...query,
  };
}

function openShortcut(query: Record<string, string>) {
  void router.push(buildAuditLogsLocation(query));
}

function openSummary(key: string) {
  switch (key) {
    case 'failed':
      void router.push(
        buildAuditLogsLocation(buildOverviewAuditQuery({ scope: AUDIT_DRILLDOWN_SCOPE.FAILED_OPERATIONS })),
      );
      return;
    case 'risk':
      void router.push(
        buildAuditLogsLocation(buildOverviewAuditQuery({ scope: AUDIT_DRILLDOWN_SCOPE.HIGH_RISK_OPERATIONS })),
      );
      return;
    case 'sensitive':
      void router.push(
        buildAuditLogsLocation(
          buildOverviewAuditQuery({
            scope: AUDIT_DRILLDOWN_SCOPE.SENSITIVE_OPERATIONS,
          }),
        ),
      );
      return;
    default:
      void router.push(buildAuditLogsLocation(buildOverviewAuditQuery({})));
  }
}

function openRiskGroup(groupKey: string) {
  const riskGroupQueries: Record<string, Record<string, string>> = {
    critical_security: { scope: AUDIT_DRILLDOWN_SCOPE.CRITICAL_SECURITY },
    high_risk_operations: { scope: AUDIT_DRILLDOWN_SCOPE.HIGH_RISK_OPERATIONS },
    auth_failures: { scope: AUDIT_DRILLDOWN_SCOPE.AUTH_FAILURES },
    permission_denials: { scope: AUDIT_DRILLDOWN_SCOPE.PERMISSION_DENIALS },
  };

  void router.push(buildAuditLogsLocation(buildOverviewAuditQuery(groupKey ? (riskGroupQueries[groupKey] ?? {}) : {})));
}

function openSecurityTimelineRequest(requestId?: string) {
  if (!requestId) {
    return;
  }

  void router.push(buildAccessLogRequestLocation(requestId));
}

async function fetchOverview() {
  loading.value = true;
  errorMessage.value = '';

  try {
    overview.value = await getAuditOverview({ preset: activeWindow.value });
  } catch (error) {
    overview.value = null;
    logger.error('failed to fetch audit overview', error);
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('audit.overview.loadFailed'));
    MessagePlugin.error(errorMessage.value);
    openCorrelationErrorNotification({
      router,
      title: t('audit.correlation.errorTitle'),
      message: errorMessage.value,
      requestId: requestIdFromError(error),
      translate: t,
    });
  } finally {
    loading.value = false;
  }
}

function toOverviewCards(items: AuditOverviewItem[] | undefined, result: string) {
  return (items ?? []).map((item) => ({
    key: String(item.id),
    actor: item.actor_display_name || item.actor_username || t('audit.common.unknownActor'),
    resource:
      item.resource_name ||
      [item.resource_type, item.resource_id].filter(Boolean).join(' / ') ||
      String(item.metadata?.request_path ?? t('audit.common.unknownResource')),
    time: formatTime(item.created_at),
    result,
  }));
}

function riskTheme(level?: string) {
  if (level === 'CRITICAL') {
    return 'danger';
  }
  if (level === 'HIGH') {
    return 'warning';
  }
  if (level === 'MEDIUM') {
    return 'primary';
  }
  return 'default';
}

function timelineDotColor(level?: string) {
  if (level === 'CRITICAL') {
    return 'var(--td-error-color)';
  }
  if (level === 'HIGH') {
    return 'var(--td-warning-color)';
  }
  if (level === 'MEDIUM') {
    return 'var(--td-brand-color)';
  }
  return 'var(--td-text-color-placeholder)';
}

function formatTrendTooltipLabel(start?: string, end?: string) {
  const startLabel = formatTrendTooltipDateTime(start, locale.value);
  if (!end) {
    return startLabel;
  }
  return `${startLabel} - ${formatTrendTooltipDateTime(end, locale.value)}`;
}

function formatTrendValue(value: number) {
  return String(value);
}

function resolveTrendColor(variableName: string, fallback: string) {
  if (typeof window === 'undefined') {
    return fallback;
  }

  const value = getComputedStyle(document.documentElement).getPropertyValue(variableName).trim();
  return value || fallback;
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

function disposeTrendChart() {
  trendChart?.dispose();
  trendChart = null;
}

function buildTrendChartOption(): EChartsCoreOption {
  const chartColors = settingStore.chartColors;
  const points = trendView.value.points;
  const totalLabel = t('audit.overview.trend.totalMetric');
  const highRiskLabel = t('audit.overview.trend.highRiskMetric');
  const securityLabel = t('audit.overview.trend.securityMetric');
  const seriesColors = [
    resolveTrendColor('--td-brand-color', '#0052D9'),
    resolveTrendColor('--td-warning-color', '#ED7B2F'),
    resolveTrendColor('--td-error-color', '#E34D59'),
  ];

  return {
    color: seriesColors,
    legend: {
      top: 0,
      itemWidth: 10,
      itemHeight: 10,
      textStyle: {
        color: chartColors.textColor,
      },
      data: [totalLabel, highRiskLabel, securityLabel],
    },
    tooltip: {
      trigger: 'axis',
      backgroundColor: chartColors.containerColor,
      borderColor: chartColors.borderColor,
      textStyle: {
        color: chartColors.textColor,
      },
      formatter: (
        params: Array<{ axisValue: string; axisValueLabel?: string; seriesName: string; color: string; data: number }>,
      ) => {
        const activePoint = points.find((point) => point.key === params[0]?.axisValue) ?? points[0];
        const rows = params
          .map((param) => {
            return [
              `<div style="display:flex;align-items:center;justify-content:space-between;gap:16px;">`,
              `<span style="display:flex;align-items:center;gap:8px;">`,
              `<i style="width:8px;height:8px;border-radius:999px;background:${param.color};display:inline-block;"></i>`,
              `<span>${param.seriesName}</span>`,
              `</span>`,
              `<strong>${formatTrendValue(param.data)}</strong>`,
              `</div>`,
            ].join('');
          })
          .join('');

        return [
          `<div style="display:flex;flex-direction:column;gap:8px;">`,
          `<strong>${activePoint?.tooltipLabel ?? params[0]?.axisValueLabel ?? ''}</strong>`,
          rows,
          `</div>`,
        ].join('');
      },
    },
    grid: {
      left: '18px',
      right: '18px',
      top: '52px',
      bottom: '12px',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: points.map((point) => point.key),
      axisLabel: {
        color: chartColors.placeholderColor,
        margin: 8,
        hideOverlap: true,
        formatter: (value: string) => points.find((point) => point.key === value)?.axisLabel ?? '',
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
    yAxis: {
      type: 'value',
      min: 0,
      axisLabel: {
        color: chartColors.placeholderColor,
        formatter: (value: number) => formatTrendValue(value),
      },
      splitLine: {
        lineStyle: {
          color: chartColors.borderColor,
        },
      },
    },
    series: [
      buildTrendSeries(
        totalLabel,
        points.map((point) => point.total),
      ),
      buildTrendSeries(
        highRiskLabel,
        points.map((point) => point.highRisk),
      ),
      buildTrendSeries(
        securityLabel,
        points.map((point) => point.security),
      ),
    ],
  };
}

function buildTrendSeries(name: string, data: number[]) {
  return {
    name,
    type: 'line',
    smooth: true,
    symbol: 'circle',
    symbolSize: 7,
    showSymbol: false,
    lineStyle: {
      width: 2.5,
    },
    areaStyle: {
      opacity: 0.14,
    },
    emphasis: {
      focus: 'series',
      areaStyle: {
        opacity: 0.18,
      },
    },
    data,
  };
}

async function syncTrendChart() {
  if (!trendView.value.isRenderable) {
    disposeTrendChart();
    return;
  }

  await nextTick();
  const chart = ensureTrendChart();
  if (!chart) {
    return;
  }
  setupTrendChartResizeObserver();
  observeTrendChartResize();

  chart.setOption(buildTrendChartOption(), true);
  chart.resize({
    width: trendChartRef.value?.clientWidth,
    height: trendChartRef.value?.clientHeight,
  });
}

function setupTrendChartResizeObserver() {
  if (trendChartResizeObserver || typeof ResizeObserver === 'undefined') {
    return;
  }

  trendChartResizeObserver = new ResizeObserver(() => {
    trendChart?.resize({
      width: trendChartRef.value?.clientWidth,
      height: trendChartRef.value?.clientHeight,
    });
  });
}

function observeTrendChartResize() {
  if (!trendChartResizeObserver || !trendChartRef.value) {
    return;
  }
  trendChartResizeObserver.disconnect();
  trendChartResizeObserver.observe(trendChartRef.value);
}

function teardownTrendChartResizeObserver() {
  trendChartResizeObserver?.disconnect();
  trendChartResizeObserver = null;
}

function buildFrozenOverviewWindow() {
  const [createdFrom = '', createdTo = ''] = buildPresetLocalRange(activeWindow.value);
  return {
    created_from: createdFrom ? localDateTimeToUtcIso(createdFrom) : '',
    created_to: createdTo ? localDateTimeToUtcIso(createdTo) : '',
  };
}

function buildPresetLocalRange(preset: AuditTimePreset) {
  const now = new Date();
  switch (preset) {
    case AUDIT_TIME_PRESET.LAST_24H:
      return buildRecentHoursLocalRange(now, 24);
    case AUDIT_TIME_PRESET.LAST_7D:
      return buildRecentHoursLocalRange(now, 24 * 7);
    case AUDIT_TIME_PRESET.LAST_30D:
      return buildRecentHoursLocalRange(now, 24 * 30);
    default:
      return [];
  }
}

function formatTime(value?: string) {
  return formatLocaleDateTime(value, locale.value, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

watch(activeWindow, () => {
  void fetchOverview();
});

watch(
  [
    () => trendView.value,
    () => locale.value,
    () => settingStore.displayMode,
    () => settingStore.brandTheme,
    () => settingStore.chartColors.textColor,
    () => settingStore.chartColors.placeholderColor,
    () => settingStore.chartColors.borderColor,
    () => settingStore.chartColors.containerColor,
  ],
  () => {
    void syncTrendChart();
  },
  { deep: true },
);

onMounted(() => {
  setupTrendChartResizeObserver();
  void fetchOverview();
});

onUnmounted(() => {
  teardownTrendChartResizeObserver();
  disposeTrendChart();
});
</script>
<style scoped lang="less">
.audit-overview,
.audit-overview__stack,
.audit-overview__list,
.audit-overview__watch-list {
  display: flex;
  flex-direction: column;
}

.audit-overview__grid {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.audit-overview__grid--bottom {
  grid-template-columns: minmax(0, 1.3fr) minmax(320px, 0.9fr);
}

.audit-overview__stack,
.audit-overview__list,
.audit-overview__watch-list,
.audit-overview__timeline {
  gap: 16px;
}

.audit-overview__list-item,
.audit-overview__watch-item,
.audit-overview__shortcut {
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  display: flex;
  gap: 12px;
  justify-content: space-between;
  padding: 14px 16px;
}

.audit-overview__list-item,
.audit-overview__shortcut,
.audit-overview__watch-actions {
  align-items: center;
}

.audit-overview__watch-content,
.audit-overview__watch-actions {
  display: flex;
}

.audit-overview__watch-content {
  flex: 1;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.audit-overview__watch-actions {
  flex-shrink: 0;
  gap: 8px;
}

.audit-overview__list-item p,
.audit-overview__watch-item p,
.audit-overview__shortcut span,
.audit-overview__item-meta span {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.audit-overview__item-meta {
  align-items: flex-end;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.audit-overview__shortcut {
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  cursor: pointer;
  text-align: left;
  transition:
    border-color 0.2s ease,
    background-color 0.2s ease,
    box-shadow 0.2s ease;
  width: 100%;
}

.audit-overview__shortcut strong {
  color: var(--td-text-color-primary);
}

.audit-overview__shortcut:hover {
  background: color-mix(in srgb, var(--td-brand-color-light) 14%, var(--td-bg-color-container) 86%);
  border-color: color-mix(in srgb, var(--td-brand-color) 32%, var(--td-component-stroke) 68%);
}

.audit-overview__shortcut:focus-visible {
  border-color: var(--td-brand-color);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--td-brand-color) 18%, transparent);
  outline: none;
}

.audit-overview__shortcut:active {
  background: color-mix(in srgb, var(--td-brand-color-light) 20%, var(--td-bg-color-container) 80%);
}

.audit-overview__summary-action,
.audit-overview__timeline-item--button {
  background: transparent;
  border: 0;
  cursor: pointer;
  padding: 0;
  text-align: left;
  width: 100%;
}

.audit-overview__window-label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  font-variant-numeric: tabular-nums;
}

.audit-overview__trend-panel,
.audit-overview__trend-empty {
  min-height: 280px;
}

.audit-overview__trend-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.audit-overview__trend-metrics {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}

.audit-overview__trend-metric {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 120px;
  padding: 12px 14px;
}

.audit-overview__trend-metric span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.audit-overview__trend-metric strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  font-variant-numeric: tabular-nums;
}

.audit-overview__trend-chart {
  min-height: 320px;
  width: 100%;
}

.audit-overview__timeline-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.audit-overview__timeline-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.audit-overview__timeline-item p {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.audit-overview__timeline-meta {
  display: flex;
  gap: 8px;
}

@media (width <= 1280px) {
  .audit-overview__grid,
  .audit-overview__grid--bottom {
    grid-template-columns: 1fr;
  }
}

@media (width <= 768px) {
  .audit-overview__list-item,
  .audit-overview__watch-item,
  .audit-overview__shortcut {
    align-items: flex-start;
    flex-direction: column;
  }

  .audit-overview__watch-actions {
    width: 100%;
  }

  .audit-overview__trend-chart {
    min-height: 280px;
  }
}
</style>
