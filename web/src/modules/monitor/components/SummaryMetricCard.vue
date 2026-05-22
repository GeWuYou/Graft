<template>
  <article class="server-status-summary-card">
    <div class="server-status-summary-card__top">
      <span class="server-status-summary-card__title">{{ title }}</span>
      <status-tag v-if="statusLabel" :label="statusLabel" :status="status" />
    </div>

    <div class="server-status-summary-card__value-row">
      <strong class="server-status-summary-card__value">{{ value }}</strong>
      <span v-if="valueAside" class="server-status-summary-card__aside">{{ valueAside }}</span>
    </div>

    <p class="server-status-summary-card__description">{{ description }}</p>
  </article>
</template>
<script setup lang="ts">
import type { ServerStatusTone } from './server-status-ui';
import StatusTag from './StatusTag.vue';

withDefaults(
  defineProps<{
    title: string;
    value: string;
    description: string;
    status?: ServerStatusTone;
    statusLabel?: string;
    valueAside?: string;
  }>(),
  {
    status: 'unknown',
    statusLabel: '',
    valueAside: '',
  },
);
</script>
<style scoped lang="less">
.server-status-summary-card {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-border);
  border-radius: var(--td-radius-large);
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 128px;
  padding: 16px 18px;
}

.server-status-summary-card__top {
  align-items: flex-start;
  display: flex;
  gap: 12px;
  justify-content: space-between;
}

.server-status-summary-card__title {
  color: var(--td-text-color-secondary);
  font-size: 13px;
  font-weight: 500;
  line-height: 20px;
}

.server-status-summary-card__value-row {
  align-items: baseline;
  display: flex;
  flex: 1;
  gap: 10px;
}

.server-status-summary-card__value {
  color: var(--td-text-color-primary);
  font-size: 26px;
  font-variant-numeric: tabular-nums;
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.server-status-summary-card__aside {
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 20px;
}

.server-status-summary-card__description {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
  margin: 0;
}
</style>
