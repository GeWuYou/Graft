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
            <div class="audit-overview__trend-legend" aria-label="trend legend">
              <span v-for="item in trendLegendItems" :key="item.key" class="audit-overview__trend-legend-item">
                <i
                  :class="['audit-overview__trend-legend-swatch', `audit-overview__trend-legend-swatch--${item.key}`]"
                />
                {{ item.label }}
              </span>
            </div>
            <div class="audit-overview__trend-chart">
              <div class="audit-overview__trend-y-axis" aria-hidden="true">
                <span v-for="marker in trendView.yAxisMarkers" :key="marker.key">{{ marker.label }}</span>
              </div>
              <div class="audit-overview__trend-plot">
                <div
                  v-for="marker in trendView.yAxisMarkers"
                  :key="marker.key"
                  class="audit-overview__trend-grid-line"
                  :style="{ bottom: marker.offset }"
                  aria-hidden="true"
                />
                <div class="audit-overview__trend" :style="trendTrackStyle">
                  <t-tooltip v-for="point in trendView.points" :key="point.key" placement="top" theme="default">
                    <template #content>
                      <div class="audit-overview__trend-tooltip">
                        <strong>{{ point.tooltipLabel }}</strong>
                        <span>{{ t('audit.overview.trend.totalValue', { value: point.total }) }}</span>
                        <span>{{ t('audit.overview.trend.highRiskValue', { value: point.highRisk }) }}</span>
                        <span>{{ t('audit.overview.trend.securityValue', { value: point.security }) }}</span>
                      </div>
                    </template>
                    <div class="audit-overview__trend-point">
                      <div class="audit-overview__trend-bars">
                        <div
                          class="audit-overview__trend-bar audit-overview__trend-bar--total"
                          :style="{ height: point.totalHeight }"
                        />
                        <div
                          class="audit-overview__trend-bar audit-overview__trend-bar--risk"
                          :style="{ height: point.highRiskHeight }"
                        />
                        <div
                          class="audit-overview__trend-bar audit-overview__trend-bar--security"
                          :style="{ height: point.securityHeight }"
                        />
                      </div>
                      <strong>{{ point.axisLabel }}</strong>
                    </div>
                  </t-tooltip>
                </div>
              </div>
            </div>
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
                <strong>{{ item.action }}</strong>
                <p>{{ item.resource_name || item.resource_type || t('audit.common.unknownResource') }}</p>
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
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { buildAccessLogRequestLocation } from '@/modules/access-log/contract/deep-link';
import { buildAuditLogsLocation } from '@/modules/audit/contract/deep-link';
import { AUDIT_TIME_PRESET, type AuditTimePreset } from '@/modules/audit/contract/time-presets';
import { openCorrelationErrorNotification, requestIdFromError } from '@/modules/audit/shared/correlation-actions';
import { resolveLocalizedErrorMessage } from '@/modules/shared/localized-api-error';
import { GovernanceDashboardShell, GovernanceSection, GovernanceSummaryCard } from '@/shared/components/governance';
import { ManagementEmptyState } from '@/shared/components/management';
import { formatLocaleDateTime } from '@/shared/observability';
import { createLogger } from '@/utils/logger';

import { getAuditOverview } from '../../api/audit';
import type { AuditOverviewItem, AuditOverviewResponse } from '../../types/audit';

defineOptions({
  name: 'AuditOverviewIndex',
});

const { locale, t } = useI18n();
const router = useRouter();
const logger = createLogger('audit.overview');
const activeWindow = ref<AuditTimePreset>(AUDIT_TIME_PRESET.LAST_24H);
const loading = ref(false);
const errorMessage = ref('');
const overview = ref<AuditOverviewResponse | null>(null);

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
const trendView = computed(() => {
  const points = overview.value?.trend?.points ?? [];
  const activePoints = points.filter(
    (point) => point.total > 0 || point.failed > 0 || point.high_risk > 0 || point.security_events > 0,
  );
  const maxTotal = Math.max(...points.map((point) => point.total), 0);
  const nonSparsePoints = activePoints.filter(
    (point) => point.total > 1 || point.failed > 1 || point.high_risk > 0 || point.security_events > 0,
  );
  const hasEnoughPoints = points.length >= 3;
  const hasEnoughActivity = activePoints.length >= 3 && nonSparsePoints.length >= 3 && maxTotal > 1;

  if (!hasEnoughPoints || !hasEnoughActivity) {
    return {
      isRenderable: false,
      points: [],
      yAxisMarkers: [],
    };
  }

  const labelStep = resolveTrendLabelStep(points.length);
  const yAxisMarkers = buildTrendAxisMarkers(maxTotal);

  return {
    isRenderable: true,
    points: points.map((point, index) => ({
      key: `${point.bucket_start}-${point.bucket_end}`,
      axisLabel: index % labelStep === 0 || index === points.length - 1 ? formatTrendAxisLabel(point.bucket_start) : '',
      tooltipLabel: formatTrendTooltipLabel(point.bucket_start, point.bucket_end),
      totalHeight: resolveTrendBarHeight(point.total, maxTotal),
      highRiskHeight: resolveTrendBarHeight(point.high_risk, maxTotal),
      securityHeight: resolveTrendBarHeight(point.security_events, maxTotal),
      total: point.total,
      highRisk: point.high_risk,
      security: point.security_events,
    })),
    yAxisMarkers,
  };
});

const trendTrackStyle = computed(() => {
  const pointCount = Math.max(trendView.value.points.length, 1);
  return {
    gridTemplateColumns: `repeat(${pointCount}, minmax(0, 1fr))`,
  };
});

const trendLegendItems = computed(() => [
  { key: 'total', label: t('audit.overview.trend.legend.total') },
  { key: 'risk', label: t('audit.overview.trend.legend.highRisk') },
  { key: 'security', label: t('audit.overview.trend.legend.security') },
]);
const shortcuts = computed(() => [
  {
    key: 'failed',
    title: t('audit.overview.shortcuts.failedAuth.title'),
    description: t('audit.overview.shortcuts.failedAuth.description'),
    query: buildOverviewAuditQuery({
      success: 'false',
      resource_types: 'auth,session',
      action_keywords: 'auth,login',
      request_path_prefixes: '/api/auth',
    }),
  },
  {
    key: 'rbac',
    title: t('audit.overview.shortcuts.rbacChanges.title'),
    description: t('audit.overview.shortcuts.rbacChanges.description'),
    query: buildOverviewAuditQuery({
      action_prefixes: 'rbac.,role.,permission.',
    }),
  },
  {
    key: 'sensitive',
    title: t('audit.overview.shortcuts.sensitiveOps.title'),
    description: t('audit.overview.shortcuts.sensitiveOps.description'),
    query: buildOverviewAuditQuery({
      action_keywords: 'delete,reset,grant,assign,revoke,remove,replace,update_role,update_permission',
    }),
  },
]);

const riskGroupActionLabel = computed(() => t('audit.overview.riskGroups.action'));
const relatedRequestActionLabel = computed(() => t('audit.logList.drawer.actions.viewRelatedRequest'));

function buildOverviewAuditQuery(query: Record<string, string>) {
  return {
    preset: activeWindow.value,
    ...query,
  };
}

function openShortcut(query: Record<string, string>) {
  void router.push(buildAuditLogsLocation(query));
}

function openSummary(key: string) {
  switch (key) {
    case 'failed':
      void router.push(buildAuditLogsLocation(buildOverviewAuditQuery({ success: 'false' })));
      return;
    case 'risk':
      void router.push(buildAuditLogsLocation(buildOverviewAuditQuery({ risk_levels: 'HIGH,CRITICAL' })));
      return;
    case 'sensitive':
      void router.push(
        buildAuditLogsLocation(
          buildOverviewAuditQuery({
            action_keywords: 'delete,reset,grant,assign,revoke,remove,replace,update_role,update_permission',
          }),
        ),
      );
      return;
    default:
      void router.push(buildAuditLogsLocation(buildOverviewAuditQuery({})));
  }
}

function openRiskGroup(groupKey: string) {
  const queryByGroupKey: Record<string, Record<string, string>> = {
    critical_security: { results: 'DENIED,ERROR' },
    high_risk_operations: {
      action_keywords: 'delete,reset,grant,assign,revoke,remove,replace,update_role,update_permission',
    },
    auth_failures: {
      success: 'false',
      resource_types: 'auth,session',
      action_keywords: 'auth,login',
      request_path_prefixes: '/api/auth',
    },
    permission_denials: { results: 'DENIED' },
  };

  void router.push(buildAuditLogsLocation(buildOverviewAuditQuery(queryByGroupKey[groupKey] || {})));
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

function formatBucketLabel(value?: string) {
  return formatTrendDateTime(value, { month: '2-digit', day: '2-digit', hour: '2-digit' });
}

function formatTrendAxisLabel(value?: string) {
  return formatTrendDateTime(value, { hour: '2-digit', minute: '2-digit' });
}

function formatTrendTooltipLabel(start?: string, end?: string) {
  const startLabel = formatBucketLabel(start);
  if (!end) {
    return startLabel;
  }
  return `${startLabel} - ${formatBucketLabel(end)}`;
}

function resolveTrendLabelStep(pointCount: number) {
  if (pointCount <= 6) {
    return 1;
  }
  if (pointCount <= 12) {
    return 2;
  }
  return 3;
}

function resolveTrendBarHeight(value: number, maxTotal: number) {
  if (value <= 0 || maxTotal <= 0) {
    return '0%';
  }
  return `${(value / maxTotal) * 100}%`;
}

function buildTrendAxisMarkers(maxTotal: number) {
  const midpoint = Math.ceil(maxTotal / 2);
  return [
    { key: 'max', label: String(maxTotal), offset: '100%' },
    { key: 'mid', label: String(midpoint), offset: '50%' },
    { key: 'min', label: '0', offset: '0%' },
  ];
}

function formatTrendDateTime(value: string | undefined, timeOptions: Intl.DateTimeFormatOptions) {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  const currentLocale = locale.value || undefined;
  const options: Intl.DateTimeFormatOptions =
    overview.value?.trend?.bucket_unit === 'day' ? { month: '2-digit', day: '2-digit' } : timeOptions;
  return new Intl.DateTimeFormat(currentLocale, options).format(date);
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

onMounted(() => {
  void fetchOverview();
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
  cursor: pointer;
  text-align: left;
  width: 100%;
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
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.audit-overview__trend-panel,
.audit-overview__trend-empty {
  min-height: 240px;
}

.audit-overview__trend-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.audit-overview__trend-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}

.audit-overview__trend-legend-item {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: inline-flex;
  font-size: 12px;
  gap: 8px;
}

.audit-overview__trend-legend-swatch {
  border-radius: 999px;
  display: inline-block;
  height: 10px;
  width: 10px;
}

.audit-overview__trend-legend-swatch--total {
  background: color-mix(in srgb, var(--td-brand-color) 35%, white);
}

.audit-overview__trend-legend-swatch--risk {
  background: var(--td-warning-color);
}

.audit-overview__trend-legend-swatch--security {
  background: var(--td-error-color);
}

.audit-overview__trend-chart {
  display: grid;
  gap: 12px;
  grid-template-columns: 40px minmax(0, 1fr);
}

.audit-overview__trend-y-axis {
  color: var(--td-text-color-secondary);
  display: flex;
  flex-direction: column;
  font-size: 12px;
  justify-content: space-between;
  padding-block: 4px 20px;
}

.audit-overview__trend-plot {
  min-width: 0;
  position: relative;
}

.audit-overview__trend-grid-line {
  border-top: 1px dashed var(--td-component-stroke);
  inset-inline: 0;
  position: absolute;
}

.audit-overview__trend {
  align-items: end;
  display: grid;
  gap: 12px;
  min-height: 220px;
  position: relative;
}

.audit-overview__trend-point {
  align-items: center;
  cursor: default;
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.audit-overview__trend-bars {
  align-items: end;
  display: flex;
  gap: 4px;
  height: 160px;
  position: relative;
  z-index: 1;
}

.audit-overview__trend-bar {
  border-radius: 999px 999px 0 0;
  min-height: 0;
  width: 10px;
}

.audit-overview__trend-bar--total {
  background: color-mix(in srgb, var(--td-brand-color) 35%, white);
}

.audit-overview__trend-bar--risk {
  background: var(--td-warning-color);
}

.audit-overview__trend-bar--security {
  background: var(--td-error-color);
}

.audit-overview__trend-point strong {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  line-height: 20px;
  min-height: 20px;
  text-align: center;
}

.audit-overview__trend-tooltip {
  color: var(--td-text-color-anti);
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-width: 220px;
}

.audit-overview__trend-tooltip strong {
  color: var(--td-text-color-anti);
}

.audit-overview__trend-tooltip span {
  color: var(--td-text-color-anti);
  font-variant-numeric: tabular-nums;
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
    grid-template-columns: 32px minmax(0, 1fr);
  }
}
</style>
