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

    <div v-if="loading" class="dashboard-container-resources__summary-grid">
      <t-skeleton v-for="item in 4" :key="item" animation="gradient" :row-col="summarySkeletonRowCol" />
    </div>

    <t-empty
      v-else-if="isEmpty"
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

      <section class="dashboard-container-resources__groups">
        <div class="dashboard-container-resources__group">
          <header class="dashboard-container-resources__group-header">
            <div>
              <span>{{ t('dashboard.containerResources.hotspots.eyebrow') }}</span>
              <h3>{{ t('dashboard.containerResources.hotspots.cpuTitle') }}</h3>
            </div>
            <t-tag size="small" theme="warning" variant="light-outline">
              {{ t('dashboard.containerResources.hotspots.top3') }}
            </t-tag>
          </header>
          <div v-if="summary.hotspots.cpu.length" class="dashboard-container-resources__list">
            <article
              v-for="container in summary.hotspots.cpu"
              :key="`cpu-${container.id}`"
              class="dashboard-container-resources__list-item"
              data-testid="dashboard-container-hotspot-cpu-item"
            >
              <header class="dashboard-container-resources__list-item-header">
                <div>
                  <strong>{{ container.name }}</strong>
                  <p>{{ container.image || '-' }}</p>
                </div>
                <t-tag size="small" variant="light-outline" :theme="stateTheme(container.state, container.health)">
                  {{ containerStatusLabel(container.state, container.health) }}
                </t-tag>
              </header>
              <div class="dashboard-container-resources__metric-card">
                <span>{{ t('dashboard.containerResources.cpu') }}</span>
                <strong>{{ formatPercent(container.cpuPercent) }}</strong>
                <t-progress :percentage="clampPercent(container.cpuPercent)" :label="false" size="small" />
              </div>
            </article>
          </div>
          <t-empty
            v-else
            size="small"
            :description="t('dashboard.containerResources.hotspots.empty')"
            class="dashboard-container-resources__group-empty"
          />
        </div>

        <div class="dashboard-container-resources__group">
          <header class="dashboard-container-resources__group-header">
            <div>
              <span>{{ t('dashboard.containerResources.hotspots.eyebrow') }}</span>
              <h3>{{ t('dashboard.containerResources.hotspots.memoryTitle') }}</h3>
            </div>
            <t-tag size="small" theme="warning" variant="light-outline">
              {{ t('dashboard.containerResources.hotspots.top3') }}
            </t-tag>
          </header>
          <div v-if="summary.hotspots.memory.length" class="dashboard-container-resources__list">
            <article
              v-for="container in summary.hotspots.memory"
              :key="`memory-${container.id}`"
              class="dashboard-container-resources__list-item"
              data-testid="dashboard-container-hotspot-memory-item"
            >
              <header class="dashboard-container-resources__list-item-header">
                <div>
                  <strong>{{ container.name }}</strong>
                  <p>{{ container.image || '-' }}</p>
                </div>
                <t-tag size="small" variant="light-outline" :theme="stateTheme(container.state, container.health)">
                  {{ containerStatusLabel(container.state, container.health) }}
                </t-tag>
              </header>
              <div class="dashboard-container-resources__metric-card">
                <span>{{ t('dashboard.containerResources.memory') }}</span>
                <strong>{{ formatPercent(container.memoryPercent) }}</strong>
                <t-progress :percentage="clampPercent(container.memoryPercent)" :label="false" size="small" />
                <p>{{ formatMemoryUsage(container.memoryUsageBytes, container.memoryLimitBytes) }}</p>
              </div>
            </article>
          </div>
          <t-empty
            v-else
            size="small"
            :description="t('dashboard.containerResources.hotspots.empty')"
            class="dashboard-container-resources__group-empty"
          />
        </div>
      </section>

      <section class="dashboard-container-resources__anomalies">
        <header class="dashboard-container-resources__group-header">
          <div>
            <span>{{ t('dashboard.containerResources.anomalies.eyebrow') }}</span>
            <h3>{{ t('dashboard.containerResources.anomalies.title') }}</h3>
          </div>
          <t-tag size="small" theme="danger" variant="light-outline">
            {{ t('dashboard.containerResources.anomalies.count', { count: summary.anomalies.length }) }}
          </t-tag>
        </header>
        <div v-if="summary.anomalies.length" class="dashboard-container-resources__list">
          <article
            v-for="item in summary.anomalies"
            :key="`anomaly-${item.id}-${item.state}-${item.health || 'none'}`"
            class="dashboard-container-resources__list-item dashboard-container-resources__list-item--anomaly"
            data-testid="dashboard-container-anomaly-item"
          >
            <header class="dashboard-container-resources__list-item-header">
              <div>
                <strong>{{ item.name }}</strong>
                <p>{{ item.image || '-' }}</p>
              </div>
              <div class="dashboard-container-resources__anomaly-tags">
                <t-tag size="small" theme="danger" variant="light-outline">
                  {{ anomalyLabel(item) }}
                </t-tag>
                <t-tag size="small" variant="light-outline" :theme="stateTheme(item.state, item.health)">
                  {{ containerStatusLabel(item.state, item.health) }}
                </t-tag>
              </div>
            </header>
            <div class="dashboard-container-resources__anomaly-metrics">
              <span>{{ t('dashboard.containerResources.cpu') }} {{ formatPercent(item.cpuPercent) }}</span>
              <span>{{ t('dashboard.containerResources.memory') }} {{ formatPercent(item.memoryPercent) }}</span>
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
import type { ContainerDashboardSummary } from '@/modules/container/contract/dashboard-summary';
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

const summarySkeletonRowCol = [
  { width: '42%', height: '14px' },
  { width: '58%', height: '30px' },
  { width: '78%', height: '12px' },
];

const isEmpty = computed(
  () =>
    props.summary.overview.runningContainers <= 0 &&
    props.summary.overview.abnormalContainers <= 0 &&
    props.summary.hotspots.cpu.length === 0 &&
    props.summary.hotspots.memory.length === 0 &&
    props.summary.anomalies.length === 0,
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
    value: formatPercent(props.summary.overview.cpuTotalPercent),
    description: t('dashboard.containerResources.overview.cpuTotal.description'),
  },
  {
    key: 'memory-total',
    label: t('dashboard.containerResources.overview.memoryTotal.label'),
    value: formatPercent(props.summary.overview.memoryTotalPercent),
    description: t('dashboard.containerResources.overview.memoryTotal.description'),
  },
]);

const collectedAtLabel = computed(() =>
  formatLocaleDateTime(props.summary.overview.collectedAt, currentLocale, MEDIUM_DATE_TIME_WITH_SECONDS_FORMAT_OPTIONS),
);

function clampPercent(value?: number | null) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return 0;
  }
  return Math.min(100, Math.max(0, value));
}

function formatPercent(value?: number | null) {
  return formatResourcePercent(value, t('dashboard.containerResources.unavailable'));
}

function formatMemoryUsage(usageBytes?: number | null, limitBytes?: number | null) {
  const usage = formatBytes(usageBytes, t('dashboard.containerResources.unavailable'));
  const limit = formatBytes(limitBytes, t('dashboard.containerResources.unavailable'));
  return t('dashboard.containerResources.memoryUsage', { usage, limit });
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
  state?: string | null;
  cpuPercent?: number | null;
  memoryPercent?: number | null;
}) {
  const translationKey = `dashboard.containerResources.anomalies.kind.${resolveAnomalyKind(item)}`;
  const translated = t(translationKey);
  if (translated !== translationKey) {
    return translated;
  }
  return t('dashboard.containerResources.status.unknown');
}

function resolveAnomalyKind(item: {
  health?: string | null;
  state?: string | null;
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
.dashboard-container-resources__list-item {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  display: grid;
  gap: var(--td-comp-margin-s);
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
}

.dashboard-container-resources__summary-item span,
.dashboard-container-resources__group-header span,
.dashboard-container-resources__collected-at,
.dashboard-container-resources__metric-card span,
.dashboard-container-resources__anomaly-metrics span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.dashboard-container-resources__summary-item strong,
.dashboard-container-resources__metric-card strong,
.dashboard-container-resources__list-item-header strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.dashboard-container-resources__summary-item p,
.dashboard-container-resources__list-item-header p,
.dashboard-container-resources__metric-card p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.dashboard-container-resources__groups {
  display: grid;
  gap: var(--td-comp-margin-l);
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
}

.dashboard-container-resources__group,
.dashboard-container-resources__anomalies {
  display: grid;
  gap: var(--td-comp-margin-m);
}

.dashboard-container-resources__group-header {
  align-items: start;
  display: flex;
  justify-content: space-between;
}

.dashboard-container-resources__group-header h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  margin: 0;
}

.dashboard-container-resources__list {
  display: grid;
  gap: var(--td-comp-margin-s);
}

.dashboard-container-resources__list-item-header,
.dashboard-container-resources__anomaly-tags {
  align-items: start;
  display: flex;
  gap: var(--td-comp-margin-s);
  justify-content: space-between;
}

.dashboard-container-resources__metric-card,
.dashboard-container-resources__anomaly-metrics {
  background: color-mix(in srgb, var(--td-bg-color-container) 82%, transparent);
  border-radius: var(--td-radius-medium);
  display: grid;
  gap: var(--td-comp-margin-xxs);
  padding: var(--td-comp-paddingTB-s) var(--td-comp-paddingLR-s);
}

.dashboard-container-resources__empty {
  padding: var(--td-comp-paddingTB-xl) 0;
}

.dashboard-container-resources__group-empty {
  padding-block: var(--td-comp-paddingTB-l);
}

.dashboard-container-resources__list-item--anomaly {
  border-color: color-mix(in srgb, var(--td-error-color-5) 28%, var(--td-border-level-1-color));
}

@media (width <= 1024px) {
  .dashboard-container-resources__groups {
    grid-template-columns: 1fr;
  }
}
</style>
