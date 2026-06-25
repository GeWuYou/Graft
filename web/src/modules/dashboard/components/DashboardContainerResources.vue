<template>
  <t-card class="dashboard-container-resources" :title="t('dashboard.containerResources.title')" :bordered="false">
    <template #actions>
      <t-space size="small" align="center">
        <t-tag v-if="summary.overview.collectedAt" size="small" variant="light-outline" theme="primary">
          {{ t('dashboard.containerResources.source') }}
        </t-tag>
        <span v-if="summary.overview.collectedAt" class="dashboard-container-resources__collected-at">
          {{ t('dashboard.containerResources.collectedAt') }} {{ collectedAtLabel }}
        </span>
      </t-space>
    </template>

    <div v-if="loading" class="dashboard-container-resources__content">
      <section class="dashboard-container-resources__summary-grid">
        <t-skeleton v-for="item in 4" :key="`summary-${item}`" animation="gradient" :row-col="summarySkeletonRowCol" />
      </section>

      <section v-if="showConsumersSection" class="dashboard-container-resources__section">
        <header class="dashboard-container-resources__section-header">
          <div>
            <span>{{ t('dashboard.containerResources.consumers.eyebrow') }}</span>
            <h3>{{ t('dashboard.containerResources.consumers.title') }}</h3>
          </div>
        </header>
        <div class="dashboard-container-resources__consumer-grid">
          <t-skeleton
            v-for="item in 3"
            :key="`consumer-${item}`"
            animation="gradient"
            :row-col="consumerSkeletonRowCol"
          />
        </div>
      </section>

      <section class="dashboard-container-resources__section">
        <header class="dashboard-container-resources__section-header">
          <div>
            <span>{{ t('dashboard.containerResources.anomalies.eyebrow') }}</span>
            <h3>{{ t('dashboard.containerResources.anomalies.title') }}</h3>
          </div>
        </header>
        <div class="dashboard-container-resources__anomaly-list">
          <t-skeleton
            v-for="item in 2"
            :key="`anomaly-${item}`"
            animation="gradient"
            :row-col="anomalySkeletonRowCol"
          />
        </div>
      </section>
    </div>

    <t-empty
      v-else-if="isCompletelyEmpty"
      size="small"
      class="dashboard-container-resources__empty"
      :description="t('dashboard.containerResources.empty')"
    />

    <div v-else class="dashboard-container-resources__content">
      <section
        class="dashboard-container-resources__summary-grid"
        :aria-label="t('dashboard.containerResources.overview.title')"
      >
        <article
          v-for="item in overviewItems"
          :key="item.key"
          class="dashboard-container-resources__summary-item"
          :data-testid="`dashboard-container-overview-${item.key}`"
        >
          <span>{{ item.label }}</span>
          <strong>{{ item.value }}</strong>
          <p>{{ item.description }}</p>
        </article>
      </section>

      <section class="dashboard-container-resources__section">
        <header class="dashboard-container-resources__section-header">
          <div>
            <span>{{ t('dashboard.containerResources.consumers.eyebrow') }}</span>
            <h3>{{ t('dashboard.containerResources.consumers.title') }}</h3>
          </div>
          <t-tag size="small" theme="warning" variant="light-outline">
            {{ t('dashboard.containerResources.consumers.topCount', { count: unifiedConsumers.length }) }}
          </t-tag>
        </header>

        <t-empty
          v-if="showConsumersEmpty"
          size="small"
          :description="consumerEmptyDescription"
          class="dashboard-container-resources__group-empty"
        />

        <div v-else class="dashboard-container-resources__consumer-grid">
          <article
            v-for="consumer in unifiedConsumers"
            :key="consumer.id"
            class="dashboard-container-resources__consumer-card"
            :class="consumerCardClasses(consumer)"
            :data-testid="'dashboard-container-resource-consumer-item'"
          >
            <header class="dashboard-container-resources__consumer-header">
              <div class="dashboard-container-resources__consumer-title">
                <div class="dashboard-container-resources__consumer-name-row">
                  <strong>{{ consumer.name }}</strong>
                  <span v-if="consumer.rankBadge" class="dashboard-container-resources__rank-badge">
                    {{ consumer.rankBadge }}
                  </span>
                </div>
                <p>{{ consumer.image || '-' }}</p>
              </div>
              <div class="dashboard-container-resources__consumer-tags">
                <t-tag
                  v-if="consumer.leadingMetric"
                  size="small"
                  variant="light-outline"
                  :theme="consumer.leadingMetric.theme"
                >
                  {{ consumer.leadingMetric.label }}
                </t-tag>
                <t-tag size="small" variant="light-outline" :theme="stateTheme(consumer.state, consumer.health)">
                  {{ containerStatusLabel(consumer.state, consumer.health) }}
                </t-tag>
              </div>
            </header>

            <div class="dashboard-container-resources__metric-stack">
              <div
                v-for="metric in consumer.metrics"
                :key="`${consumer.id}-${metric.key}`"
                class="dashboard-container-resources__metric-card"
                :data-testid="`dashboard-container-resource-metric-${metric.key}`"
              >
                <div class="dashboard-container-resources__metric-head">
                  <span>{{ metric.label }}</span>
                  <strong>{{ metric.value }}</strong>
                </div>
                <t-progress
                  v-if="metric.showProgress"
                  :percentage="metric.percentage"
                  :label="false"
                  size="small"
                  :status="metric.progressStatus"
                />
                <p>{{ metric.description }}</p>
              </div>
            </div>
          </article>
        </div>
      </section>

      <section class="dashboard-container-resources__section dashboard-container-resources__section--anomalies">
        <header class="dashboard-container-resources__section-header">
          <div>
            <span>{{ t('dashboard.containerResources.anomalies.eyebrow') }}</span>
            <h3>{{ t('dashboard.containerResources.anomalies.title') }}</h3>
          </div>
          <t-tag size="small" theme="danger" variant="light-outline">
            {{ t('dashboard.containerResources.anomalies.count', { count: summary.anomalies.length }) }}
          </t-tag>
        </header>

        <div v-if="summary.anomalies.length" class="dashboard-container-resources__anomaly-list">
          <article
            v-for="item in anomalyCards"
            :key="`anomaly-${item.id}-${item.state}-${item.health || 'none'}`"
            class="dashboard-container-resources__anomaly-card"
            data-testid="dashboard-container-anomaly-item"
          >
            <header class="dashboard-container-resources__consumer-header">
              <div class="dashboard-container-resources__consumer-title">
                <strong>{{ item.name }}</strong>
                <p>{{ item.image || '-' }}</p>
              </div>
              <div class="dashboard-container-resources__consumer-tags">
                <t-tag size="small" theme="danger" variant="light-outline" data-testid="dashboard-anomaly-primary-tag">
                  {{ item.primaryCause }}
                </t-tag>
                <t-tag
                  v-if="hasDistinctAnomalyStatus(item)"
                  data-testid="dashboard-anomaly-status-tag"
                  size="small"
                  variant="light-outline"
                  :theme="stateTheme(item.state, item.health)"
                >
                  {{ containerStatusLabel(item.state, item.health) }}
                </t-tag>
              </div>
            </header>

            <div class="dashboard-container-resources__anomaly-body">
              <div class="dashboard-container-resources__anomaly-summary">
                <strong>{{ item.secondaryCause }}</strong>
                <p>{{ item.resourceSummary }}</p>
              </div>
              <div class="dashboard-container-resources__anomaly-meta">
                <span>{{ item.collectedAtLabel }}</span>
                <span v-if="item.restartLabel">{{ item.restartLabel }}</span>
              </div>
            </div>
          </article>
        </div>
        <t-empty
          v-else
          size="small"
          :description="t('dashboard.containerResources.anomalies.empty')"
          class="dashboard-container-resources__group-empty"
        />
      </section>
    </div>
  </t-card>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import { currentLocale, t } from '@/locales';
import type {
  ContainerDashboardAnomalyItem,
  ContainerDashboardHotspotItem,
  ContainerDashboardSummary,
} from '@/modules/container/contract/dashboard-summary';
import {
  formatBytes,
  formatLocaleDateTime,
  formatPercent as formatResourcePercent,
  MEDIUM_DATE_TIME_WITH_SECONDS_FORMAT_OPTIONS,
} from '@/shared/observability';

defineOptions({
  name: 'DashboardContainerResources',
});

const props = defineProps<{
  summary: ContainerDashboardSummary;
  loading: boolean;
}>();

type ConsumerMetric = {
  description: string;
  key: 'cpu' | 'memory';
  label: string;
  percentage: number;
  progressStatus: 'warning' | 'error' | 'active' | undefined;
  showProgress: boolean;
  theme: 'danger' | 'warning' | 'success' | 'default';
  value: string;
};

type ConsumerCard = {
  health: string | null;
  id: string;
  image: string;
  leadingMetric: ConsumerMetric | null;
  metrics: ConsumerMetric[];
  name: string;
  rankBadge: string | null;
  state: string;
};

const summarySkeletonRowCol = [
  { width: '42%', height: '14px' },
  { width: '58%', height: '30px' },
  { width: '78%', height: '12px' },
];

const consumerSkeletonRowCol = [
  { width: '48%', height: '14px' },
  { width: '72%', height: '12px' },
  { width: '100%', height: '54px', margin: '12px 0 0' },
  { width: '100%', height: '54px' },
];

const anomalySkeletonRowCol = [
  { width: '46%', height: '14px' },
  { width: '68%', height: '12px' },
  { width: '100%', height: '18px', margin: '10px 0 0' },
  { width: '82%', height: '12px' },
];

const isCompletelyEmpty = computed(
  () =>
    props.summary.overview.runningContainers <= 0 &&
    props.summary.overview.abnormalContainers <= 0 &&
    props.summary.hotspots.cpu.length === 0 &&
    props.summary.hotspots.memory.length === 0 &&
    props.summary.anomalies.length === 0,
);

const showConsumersSection = computed(
  () => props.summary.overview.runningContainers > 0 || props.summary.overview.abnormalContainers > 0,
);

const showConsumersEmpty = computed(
  () => props.summary.overview.runningContainers <= 0 || unifiedConsumers.value.length === 0,
);

const consumerEmptyDescription = computed(() =>
  props.summary.overview.runningContainers <= 0
    ? t('dashboard.containerResources.consumers.noRunning')
    : t('dashboard.containerResources.consumers.empty'),
);

const overviewItems = computed(() => [
  {
    key: 'running',
    label: t('dashboard.containerResources.overview.running.label'),
    value: t('dashboard.containerResources.overview.running.value', {
      count: props.summary.overview.runningContainers,
    }),
    description: t('dashboard.containerResources.overview.running.description'),
  },
  {
    key: 'abnormal',
    label: t('dashboard.containerResources.overview.abnormal.label'),
    value: t('dashboard.containerResources.overview.abnormal.value', {
      count: props.summary.overview.abnormalContainers,
    }),
    description: t('dashboard.containerResources.overview.abnormal.description'),
  },
  {
    key: 'cpu-total',
    label: t('dashboard.containerResources.overview.cpuTotal.label'),
    value: formatRunningMetricValue(
      props.summary.overview.cpuTotalPercent,
      normalizeOverviewMetricState(props.summary.overview.runningContainers, props.summary.overview.cpuTotalPercent),
    ),
    description: t('dashboard.containerResources.overview.cpuTotal.description'),
  },
  {
    key: 'memory-total',
    label: t('dashboard.containerResources.overview.memoryTotal.label'),
    value: formatRunningMetricValue(
      props.summary.overview.memoryTotalPercent,
      normalizeOverviewMetricState(props.summary.overview.runningContainers, props.summary.overview.memoryTotalPercent),
    ),
    description: t('dashboard.containerResources.overview.memoryTotal.description'),
  },
]);

const collectedAtLabel = computed(() =>
  formatLocaleDateTime(props.summary.overview.collectedAt, currentLocale, MEDIUM_DATE_TIME_WITH_SECONDS_FORMAT_OPTIONS),
);

const unifiedConsumers = computed<ConsumerCard[]>(() => {
  const byId = new Map<
    string,
    {
      cpuRank: number | null;
      cpuSource: ContainerDashboardHotspotItem | null;
      memoryRank: number | null;
      memorySource: ContainerDashboardHotspotItem | null;
      order: number;
    }
  >();

  props.summary.hotspots.cpu.forEach((item, index) => {
    const existing = byId.get(item.id);
    byId.set(item.id, {
      cpuRank: index + 1,
      cpuSource: item,
      memoryRank: existing?.memoryRank ?? null,
      memorySource: existing?.memorySource ?? null,
      order: Math.min(existing?.order ?? Number.MAX_SAFE_INTEGER, index),
    });
  });

  props.summary.hotspots.memory.forEach((item, index) => {
    const existing = byId.get(item.id);
    byId.set(item.id, {
      cpuRank: existing?.cpuRank ?? null,
      cpuSource: existing?.cpuSource ?? null,
      memoryRank: index + 1,
      memorySource: item,
      order: Math.min(existing?.order ?? Number.MAX_SAFE_INTEGER, index),
    });
  });

  return [...byId.entries()]
    .map(([id, item]) => {
      const source = item.cpuSource ?? item.memorySource;
      if (!source) {
        return null;
      }

      const cpuMetric = buildConsumerMetric('cpu', source);
      const memoryMetric = buildConsumerMetric('memory', source);
      const metrics = [cpuMetric, memoryMetric];
      const sortedMetrics = [...metrics].sort((left, right) => right.percentage - left.percentage);
      const leadingMetric = sortedMetrics.find((metric) => metric.showProgress) ?? null;

      return {
        id,
        health: source.health,
        image: source.image,
        leadingMetric,
        metrics,
        name: source.name,
        rankBadge: buildRankBadge(item.cpuRank, item.memoryRank),
        state: source.state,
        sortValue: Math.max(cpuMetric.percentage, memoryMetric.percentage),
        sortOrder: item.order,
      };
    })
    .filter((item): item is ConsumerCard & { sortOrder: number; sortValue: number } => Boolean(item))
    .sort((left, right) => {
      if (right.sortValue !== left.sortValue) {
        return right.sortValue - left.sortValue;
      }
      return left.sortOrder - right.sortOrder;
    })
    .map(({ sortOrder: _sortOrder, sortValue: _sortValue, ...item }) => item);
});

const anomalyCards = computed(() =>
  props.summary.anomalies.map((item) => ({
    ...item,
    collectedAtLabel: formatCollectedAt(item.collectedAt),
    primaryCause: anomalyLabel(item),
    resourceSummary: buildAnomalyResourceSummary(item),
    restartLabel:
      typeof item.restartCount === 'number'
        ? t('dashboard.containerResources.anomalies.restartCount', { count: item.restartCount })
        : '',
    secondaryCause: buildAnomalySecondaryCause(item),
  })),
);

function clampPercent(value?: number | null) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return 0;
  }
  return Math.min(100, Math.max(0, value));
}

function formatCollectedAt(value?: string | null) {
  return value
    ? formatLocaleDateTime(value, currentLocale, MEDIUM_DATE_TIME_WITH_SECONDS_FORMAT_OPTIONS)
    : t('dashboard.containerResources.anomalies.noCollectedAt');
}

function buildConsumerMetric(key: 'cpu' | 'memory', source: ContainerDashboardHotspotItem): ConsumerMetric {
  const percentage = key === 'cpu' ? source.cpuPercent : source.memoryPercent;
  const runningState = normalizeResourceState(source.state, percentage);
  const value = formatRunningMetricValue(percentage, runningState);
  const progressStatus = buildMetricProgressStatus(percentage);
  const theme = buildMetricTheme(percentage);

  return {
    description:
      key === 'cpu'
        ? t('dashboard.containerResources.metrics.cpuDescription')
        : buildMemoryMetricDescription(source, runningState),
    key,
    label: key === 'cpu' ? t('dashboard.containerResources.cpu') : t('dashboard.containerResources.memory'),
    percentage: clampPercent(percentage),
    progressStatus,
    showProgress: runningState === 'running' && typeof percentage === 'number',
    theme,
    value,
  };
}

function buildMetricTheme(value?: number | null) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return 'default';
  }
  if (value >= 90) {
    return 'danger';
  }
  if (value >= 70) {
    return 'warning';
  }
  return 'success';
}

function buildMetricProgressStatus(value?: number | null) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return undefined;
  }
  if (value >= 90) {
    return 'error';
  }
  if (value >= 70) {
    return 'warning';
  }
  return 'active';
}

function buildMemoryMetricDescription(source: ContainerDashboardHotspotItem, state: ResourceDisplayState) {
  if (state === 'running' && source.memoryUsageBytes !== null && source.memoryLimitBytes !== null) {
    return t('dashboard.containerResources.memoryUsage', {
      limit: formatBytes(source.memoryLimitBytes, t('dashboard.containerResources.notCollected')),
      usage: formatBytes(source.memoryUsageBytes, t('dashboard.containerResources.notCollected')),
    });
  }
  return t(`dashboard.containerResources.metricStateDescription.${state}`);
}

function buildRankBadge(cpuRank: number | null, memoryRank: number | null) {
  const labels: string[] = [];
  if (cpuRank) {
    labels.push(t('dashboard.containerResources.consumers.rankCpu', { rank: cpuRank }));
  }
  if (memoryRank) {
    labels.push(t('dashboard.containerResources.consumers.rankMemory', { rank: memoryRank }));
  }
  return labels.join(' · ') || null;
}

function consumerCardClasses(consumer: ConsumerCard) {
  const theme = consumer.leadingMetric?.theme ?? 'default';
  return {
    'dashboard-container-resources__consumer-card--danger': theme === 'danger',
    'dashboard-container-resources__consumer-card--warning': theme === 'warning',
  };
}

function normalizeResourceState(runtimeState?: string | null, metricValue?: number | null): ResourceDisplayState {
  if (runtimeState && ['exited', 'dead', 'paused', 'restarting'].includes(runtimeState)) {
    return 'notApplicable';
  }
  if (runtimeState === 'running') {
    return typeof metricValue === 'number' && !Number.isNaN(metricValue) ? 'running' : 'notCollected';
  }
  if (typeof metricValue === 'number' && !Number.isNaN(metricValue)) {
    return 'running';
  }
  return 'unknown';
}

function normalizeOverviewMetricState(runningContainers: number, metricValue?: number | null): ResourceDisplayState {
  if (runningContainers <= 0) {
    return 'notApplicable';
  }
  if (typeof metricValue === 'number' && !Number.isNaN(metricValue)) {
    return 'running';
  }
  return 'notCollected';
}

function formatRunningMetricValue(value?: number | null, state: ResourceDisplayState = 'running') {
  if (state === 'notApplicable') {
    return t('dashboard.containerResources.notApplicable');
  }
  if (state === 'notCollected') {
    return t('dashboard.containerResources.notCollected');
  }
  if (state === 'unknown') {
    return t('dashboard.containerResources.status.unknown');
  }
  return formatResourcePercent(value, t('dashboard.containerResources.notCollected'));
}

function buildAnomalySecondaryCause(item: ContainerDashboardAnomalyItem) {
  if (item.reasonLabel) {
    return item.reasonLabel;
  }
  if (item.status) {
    return item.status;
  }
  return t('dashboard.containerResources.anomalies.reasonFallback');
}

function buildAnomalyResourceSummary(item: ContainerDashboardAnomalyItem) {
  const cpuState = normalizeResourceState(item.state, item.cpuPercent);
  const memoryState = normalizeResourceState(item.state, item.memoryPercent);
  return t('dashboard.containerResources.anomalies.resourceSummary', {
    cpu: formatRunningMetricValue(item.cpuPercent, cpuState),
    memory: formatRunningMetricValue(item.memoryPercent, memoryState),
  });
}

function stateTheme(state?: string | null, health?: string | null) {
  if (health === 'unhealthy' || state === 'exited' || state === 'dead') {
    return 'danger';
  }
  if (state === 'restarting' || state === 'paused') {
    return 'warning';
  }
  if (state === 'running') {
    return 'success';
  }
  return 'default';
}

function containerStatusLabel(state?: string | null, health?: string | null) {
  if (health === 'unhealthy') {
    return t('dashboard.containerResources.status.unhealthy');
  }
  if (state === 'paused') {
    return t('dashboard.containerResources.status.paused');
  }
  if (state === 'restarting') {
    return t('dashboard.containerResources.status.restarting');
  }
  if (state === 'exited') {
    return t('dashboard.containerResources.status.exited');
  }
  if (state === 'dead') {
    return t('dashboard.containerResources.status.dead');
  }
  if (state === 'running') {
    return t('dashboard.containerResources.status.running');
  }
  return state || t('dashboard.containerResources.status.unknown');
}

function anomalyLabel(item: {
  health?: string | null;
  reasonCode?: string | null;
  state?: string | null;
  status?: string | null;
  cpuPercent?: number | null;
  memoryPercent?: number | null;
}) {
  const reasonCodeKey = item.reasonCode
    ? `dashboard.containerResources.anomalies.reasonCode.${sanitizeReasonCodeKey(item.reasonCode)}`
    : '';
  if (reasonCodeKey) {
    const translated = t(reasonCodeKey);
    if (translated !== reasonCodeKey) {
      return translated;
    }
  }

  const translationKey = `dashboard.containerResources.anomalies.kind.${resolveAnomalyKind(item)}`;
  const translated = t(translationKey);
  if (translated !== translationKey) {
    return translated;
  }
  return item.status || t('dashboard.containerResources.status.unknown');
}

function hasDistinctAnomalyStatus(item: ContainerDashboardAnomalyItem) {
  const primary = normalizeDisplayLabel(anomalyLabel(item));
  const status = normalizeDisplayLabel(containerStatusLabel(item.state, item.health));
  return Boolean(status && status !== primary);
}

function sanitizeReasonCodeKey(reasonCode: string) {
  return reasonCode.replaceAll('.', '_').replaceAll('-', '_');
}

function normalizeDisplayLabel(value?: string | null) {
  return value?.trim().toLowerCase() || '';
}

function resolveAnomalyKind(item: {
  health?: string | null;
  state?: string | null;
  status?: string | null;
  cpuPercent?: number | null;
  memoryPercent?: number | null;
}) {
  if (item.health === 'unhealthy') {
    return 'unhealthy';
  }
  if (item.state === 'restarting') {
    return 'restarting';
  }
  if (item.state === 'exited') {
    return 'exited';
  }
  if (item.state === 'dead') {
    return 'dead';
  }
  if ((item.cpuPercent ?? 0) > 0 || (item.memoryPercent ?? 0) > 0) {
    return 'high_load';
  }
  return 'unknown';
}

type ResourceDisplayState = 'running' | 'notApplicable' | 'notCollected' | 'unknown';
</script>
<style lang="less" scoped>
.dashboard-container-resources {
  border-radius: var(--td-radius-large);
}

.dashboard-container-resources__content {
  display: grid;
  gap: var(--td-comp-margin-l);
}

.dashboard-container-resources__summary-grid {
  display: grid;
  gap: var(--td-comp-margin-m);
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
}

.dashboard-container-resources__summary-item,
.dashboard-container-resources__consumer-card,
.dashboard-container-resources__anomaly-card {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
}

.dashboard-container-resources__summary-item {
  display: grid;
  gap: var(--td-comp-margin-s);
}

.dashboard-container-resources__summary-item span,
.dashboard-container-resources__section-header span,
.dashboard-container-resources__collected-at,
.dashboard-container-resources__metric-card span,
.dashboard-container-resources__anomaly-meta span,
.dashboard-container-resources__rank-badge {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.dashboard-container-resources__summary-item strong,
.dashboard-container-resources__metric-head strong,
.dashboard-container-resources__consumer-title strong,
.dashboard-container-resources__anomaly-summary strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.dashboard-container-resources__summary-item p,
.dashboard-container-resources__consumer-title p,
.dashboard-container-resources__metric-card p,
.dashboard-container-resources__anomaly-summary p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.dashboard-container-resources__section {
  display: grid;
  gap: var(--td-comp-margin-m);
}

.dashboard-container-resources__section--anomalies {
  gap: var(--td-comp-margin-s);
}

.dashboard-container-resources__section-header {
  align-items: start;
  display: flex;
  justify-content: space-between;
}

.dashboard-container-resources__section-header h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  margin: 0;
}

.dashboard-container-resources__consumer-grid {
  display: grid;
  gap: var(--td-comp-margin-m);
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
}

.dashboard-container-resources__consumer-card,
.dashboard-container-resources__anomaly-card {
  display: grid;
  gap: var(--td-comp-margin-m);
}

.dashboard-container-resources__consumer-card--warning {
  border-color: color-mix(in srgb, var(--td-warning-color-5) 35%, var(--td-border-level-1-color));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--td-warning-color-5) 15%, transparent);
}

.dashboard-container-resources__consumer-card--danger {
  border-color: color-mix(in srgb, var(--td-error-color-5) 35%, var(--td-border-level-1-color));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--td-error-color-5) 16%, transparent);
}

.dashboard-container-resources__consumer-header,
.dashboard-container-resources__consumer-tags,
.dashboard-container-resources__anomaly-meta {
  align-items: start;
  display: flex;
  gap: var(--td-comp-margin-s);
  justify-content: space-between;
}

.dashboard-container-resources__consumer-title,
.dashboard-container-resources__anomaly-summary {
  display: grid;
  gap: var(--td-comp-margin-xxs);
  min-width: 0;
}

.dashboard-container-resources__consumer-name-row {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-xs);
}

.dashboard-container-resources__metric-stack {
  display: grid;
  gap: var(--td-comp-margin-s);
}

.dashboard-container-resources__metric-card,
.dashboard-container-resources__anomaly-body {
  background: color-mix(in srgb, var(--td-bg-color-container) 82%, transparent);
  border-radius: var(--td-radius-medium);
  display: grid;
  gap: var(--td-comp-margin-xs);
  padding: var(--td-comp-paddingTB-s) var(--td-comp-paddingLR-s);
}

.dashboard-container-resources__metric-head {
  align-items: center;
  display: flex;
  justify-content: space-between;
}

.dashboard-container-resources__anomaly-list {
  display: grid;
  gap: var(--td-comp-margin-s);
}

.dashboard-container-resources__anomaly-card {
  border-color: color-mix(in srgb, var(--td-error-color-5) 28%, var(--td-border-level-1-color));
}

.dashboard-container-resources__anomaly-meta {
  flex-wrap: wrap;
}

.dashboard-container-resources__empty {
  padding: var(--td-comp-paddingTB-xl) 0;
}

.dashboard-container-resources__group-empty {
  padding-block: var(--td-comp-paddingTB-l);
}

@media (width <= 1024px) {
  .dashboard-container-resources__consumer-grid {
    grid-template-columns: 1fr;
  }

  .dashboard-container-resources__section-header,
  .dashboard-container-resources__consumer-header,
  .dashboard-container-resources__consumer-tags,
  .dashboard-container-resources__anomaly-meta {
    flex-direction: column;
  }
}
</style>
