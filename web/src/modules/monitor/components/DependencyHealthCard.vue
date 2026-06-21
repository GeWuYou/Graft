<template>
  <article class="dependency-health-card" :data-dependency-key="serviceKey" :data-status="status">
    <header class="dependency-health-card__header">
      <div class="dependency-health-card__copy">
        <h3 class="dependency-health-card__title">{{ title }}</h3>
        <p v-if="description" class="dependency-health-card__description">{{ description }}</p>
      </div>
      <status-tag :label="statusLabel" :status="status" />
    </header>

    <section class="dependency-health-card__hero" :aria-label="primaryMetric.label">
      <span class="dependency-health-card__metric-label">{{ primaryMetric.label }}</span>
      <strong class="dependency-health-card__metric-value">{{ primaryMetric.value }}</strong>
      <span class="dependency-health-card__metric-hint">{{ primaryMetric.description }}</span>
    </section>

    <section class="dependency-health-card__pool" :aria-label="pool.title">
      <div class="dependency-health-card__pool-header">
        <span class="dependency-health-card__section-label">{{ pool.title }}</span>
        <strong class="dependency-health-card__pool-value">{{ pool.usageText }}</strong>
        <span class="dependency-health-card__pool-percent">{{ pool.usagePercentText }}</span>
      </div>
      <metric-usage-bar
        :value="pool.usagePercent"
        :label="pool.usageLabel"
        :status="pool.usageStatus"
        :tooltip="pool.usageTooltip"
        :empty-text="pool.emptyText"
      />
      <p class="dependency-health-card__pool-summary" :data-pool-risk="pool.usageStatus">{{ pool.summary }}</p>
    </section>

    <section class="dependency-health-card__pool-state" :aria-label="pool.stateTitle">
      <span class="dependency-health-card__section-label">{{ pool.stateTitle }}</span>
      <dl class="dependency-health-card__pool-grid">
        <div v-for="item in pool.items" :key="item.key" class="dependency-health-card__pool-item">
          <dt>{{ item.label }}</dt>
          <dd>{{ item.value }}</dd>
        </div>
      </dl>
    </section>

    <footer class="dependency-health-card__actions">
      <t-button
        class="dependency-health-card__diagnostic-action"
        variant="text"
        block
        type="button"
        @click="emit('show-diagnostics')"
      >
        {{ diagnosticsTitle }}
      </t-button>
    </footer>
  </article>
</template>
<script setup lang="ts">
import MetricUsageBar, { type MetricUsageStatus } from './MetricUsageBar.vue';
import type { ServerStatusTone } from './server-status-ui';
import StatusTag from './StatusTag.vue';

export type DependencyHealthMetric = {
  label: string;
  value: string;
  description: string;
};

export type DependencyHealthPoolItem = {
  key: string;
  label: string;
  value: string;
};

export type DependencyHealthPool = {
  title: string;
  stateTitle: string;
  usageText: string;
  usagePercent: number | null;
  usagePercentText: string;
  usageStatus: MetricUsageStatus;
  usageLabel: string;
  usageTooltip: string;
  summary: string;
  emptyText: string;
  items: DependencyHealthPoolItem[];
};

export type DependencyHealthDiagnostics = {
  title: string;
  items: DependencyHealthPoolItem[];
};

defineProps<{
  serviceKey: string;
  title: string;
  description?: string;
  status: ServerStatusTone;
  statusLabel: string;
  primaryMetric: DependencyHealthMetric;
  pool: DependencyHealthPool;
  diagnosticsTitle: string;
}>();

const emit = defineEmits<{
  (event: 'show-diagnostics'): void;
}>();
</script>
<style scoped lang="less">
.dependency-health-card {
  background: var(--server-status-card-background-subtle, var(--td-bg-color-container-hover));
  border: 1px solid var(--server-status-card-border, var(--td-component-stroke));
  border-radius: calc(var(--td-radius-large) - 2px);
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  height: 100%;
  min-width: 0;
  padding: var(--graft-density-gap-16);
  width: 100%;
}

.dependency-health-card[data-status='error'] {
  border-color: color-mix(in srgb, var(--td-error-color-5) 36%, var(--td-component-stroke));
}

.dependency-health-card[data-status='warning'] {
  border-color: color-mix(in srgb, var(--td-warning-color-5) 36%, var(--td-component-stroke));
}

.dependency-health-card__header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.dependency-health-card__copy {
  min-width: 0;
}

.dependency-health-card__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0;
}

.dependency-health-card__description,
.dependency-health-card__metric-hint,
.dependency-health-card__pool-summary {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.dependency-health-card__description {
  margin-top: var(--graft-density-gap-4);
}

.dependency-health-card__hero,
.dependency-health-card__pool-item {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-default);
}

.dependency-health-card__hero {
  display: grid;
  gap: var(--graft-density-gap-6);
  padding: var(--graft-density-gap-14);
}

.dependency-health-card__metric-label,
.dependency-health-card__section-label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  font-weight: 600;
}

.dependency-health-card__metric-value {
  color: var(--graft-metric-value-color, var(--td-brand-color-7));
  font: var(--td-font-headline-small);
  font-variant-numeric: tabular-nums;
  line-height: 1.1;
}

.dependency-health-card__pool {
  display: grid;
  gap: var(--graft-density-gap-8);
}

.dependency-health-card__pool-header {
  align-items: baseline;
  display: grid;
  gap: var(--graft-density-gap-8);
  grid-template-columns: minmax(0, 1fr) auto auto;
}

.dependency-health-card__pool-value,
.dependency-health-card__pool-percent,
.dependency-health-card__pool-item dd {
  color: var(--td-text-color-primary);
  font-variant-numeric: tabular-nums;
}

.dependency-health-card__pool-value {
  font: var(--td-font-title-small);
}

.dependency-health-card__pool-percent {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.dependency-health-card__pool-summary[data-pool-risk='warning'] {
  color: var(--td-warning-color-7);
}

.dependency-health-card__pool-summary[data-pool-risk='danger'] {
  color: var(--td-error-color-7);
}

.dependency-health-card__pool-state {
  display: grid;
  gap: var(--graft-density-gap-10);
}

.dependency-health-card__pool-grid {
  display: grid;
  gap: var(--graft-density-gap-8);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
}

.dependency-health-card__pool-item {
  display: grid;
  gap: var(--graft-density-gap-4);
  min-width: 0;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
}

.dependency-health-card__pool-item dt {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.dependency-health-card__pool-item dd {
  font: var(--td-font-title-small);
  margin: 0;
  min-width: 0;
  overflow-wrap: anywhere;
}

.dependency-health-card__actions {
  border-top: 1px solid var(--td-component-stroke);
  margin-top: auto;
  padding-top: var(--graft-density-gap-8);
}

.dependency-health-card__diagnostic-action {
  justify-content: space-between;
  min-height: 32px;
  padding: 0;
}

.dependency-health-card__diagnostic-action::after {
  content: '>';
}

@media (width <= 767px) {
  .dependency-health-card__pool-header {
    grid-template-columns: 1fr;
  }
}
</style>
