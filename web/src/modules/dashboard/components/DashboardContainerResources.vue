<template>
  <t-card class="dashboard-container-resources" :title="t('dashboard.containerResources.title')" :bordered="false">
    <template #actions>
      <t-tag size="small" variant="light-outline" theme="primary">
        {{ t('dashboard.containerResources.source') }}
      </t-tag>
    </template>

    <div v-if="loading" class="dashboard-container-resources__grid">
      <t-skeleton v-for="item in 3" :key="item" animation="gradient" :row-col="skeletonRowCol" />
    </div>

    <t-empty
      v-else-if="!containers.length"
      size="small"
      class="dashboard-container-resources__empty"
      :description="t('dashboard.containerResources.empty')"
    />

    <div v-else class="dashboard-container-resources__grid">
      <article
        v-for="container in containers"
        :key="container.id"
        class="dashboard-container-resources__item"
        data-testid="dashboard-container-resource-item"
      >
        <header class="dashboard-container-resources__item-header">
          <div>
            <strong>{{ container.name || container.short_id }}</strong>
            <p>{{ container.image }}</p>
          </div>
          <t-tag size="small" variant="light-outline" :theme="stateTheme(container.state)">
            {{ container.state }}
          </t-tag>
        </header>

        <div class="dashboard-container-resources__metrics">
          <div class="dashboard-container-resources__metric">
            <span>{{ t('dashboard.containerResources.cpu') }}</span>
            <strong>{{ formatPercent(container.resource?.cpu_percent) }}</strong>
            <t-progress :percentage="clampPercent(container.resource?.cpu_percent)" :label="false" size="small" />
          </div>
          <div class="dashboard-container-resources__metric">
            <span>{{ t('dashboard.containerResources.memory') }}</span>
            <strong>{{ formatPercent(container.resource?.memory_percent) }}</strong>
            <t-progress :percentage="clampPercent(container.resource?.memory_percent)" :label="false" size="small" />
          </div>
        </div>

        <footer class="dashboard-container-resources__footer">
          <span>{{ t('dashboard.containerResources.collectedAt') }}</span>
          <time>{{ container.resource?.collected_at || '-' }}</time>
        </footer>
      </article>
    </div>
  </t-card>
</template>
<script setup lang="ts">
import { t } from '@/locales';

import type { DashboardContainerResourceView } from '../types/container-resource';

defineOptions({
  name: 'DashboardContainerResources',
});

defineProps<{
  containers: DashboardContainerResourceView[];
  loading: boolean;
}>();

const skeletonRowCol = [
  { width: '56%', height: '16px' },
  { width: '80%', height: '12px' },
  { width: '100%', height: '12px' },
  { width: '100%', height: '12px' },
];

function clampPercent(value?: number) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return 0;
  }
  return Math.min(100, Math.max(0, value));
}

function formatPercent(value?: number) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return t('dashboard.containerResources.unavailable');
  }
  return `${value.toFixed(1)}%`;
}

function stateTheme(state?: string) {
  if (state === 'running') {
    return 'success';
  }
  if (state === 'paused') {
    return 'warning';
  }
  return 'default';
}
</script>
<style lang="less" scoped>
.dashboard-container-resources {
  border-radius: var(--td-radius-large);
}

.dashboard-container-resources__grid {
  display: grid;
  gap: var(--td-comp-margin-m);
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
}

.dashboard-container-resources__item {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  display: grid;
  gap: var(--td-comp-margin-s);
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
}

.dashboard-container-resources__item-header {
  align-items: start;
  display: flex;
  gap: var(--td-comp-margin-s);
  justify-content: space-between;
}

.dashboard-container-resources__item-header strong {
  color: var(--td-text-color-primary);
  display: block;
  font: var(--td-font-title-small);
}

.dashboard-container-resources__item-header p,
.dashboard-container-resources__footer,
.dashboard-container-resources__metric span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.dashboard-container-resources__metrics {
  display: grid;
  gap: var(--td-comp-margin-s);
}

.dashboard-container-resources__metric {
  display: grid;
  gap: var(--td-comp-margin-xxs);
}

.dashboard-container-resources__metric strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.dashboard-container-resources__footer {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-xs);
  justify-content: space-between;
}

.dashboard-container-resources__empty {
  padding: var(--td-comp-paddingTB-xl) 0;
}
</style>
