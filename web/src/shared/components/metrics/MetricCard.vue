<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <article class="metric-card">
    <div class="metric-card__content">
      <span class="metric-card__title">{{ title }}</span>
      <t-statistic
        v-if="typeof numericValue === 'number'"
        class="metric-card__statistic"
        :decimal-places="decimalPlaces"
        :title="undefined"
        :unit="unit"
        :value="numericValue"
      />
      <strong v-else class="metric-card__value">{{ value }}</strong>
      <span v-if="description" class="metric-card__description">{{ description }}</span>
    </div>
    <t-progress
      v-if="typeof progress === 'number'"
      class="metric-card__progress"
      theme="circle"
      size="small"
      :label="progressLabel"
      :percentage="progress"
      :status="progressStatus"
    />
  </article>
</template>
<script setup lang="ts">
defineProps<{
  title: string;
  value?: string;
  description?: string;
  numericValue?: number;
  decimalPlaces?: number;
  unit?: string;
  progress?: number;
  progressLabel?: string;
  progressStatus?: 'success' | 'error' | 'warning' | 'active';
}>();
</script>
<style scoped lang="less">
.metric-card {
  align-items: center;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
  padding: var(--graft-density-gap-14);
}

.metric-card__content {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.metric-card__title,
.metric-card__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  min-width: 0;
  overflow-wrap: anywhere;
}

.metric-card__value {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  min-width: 0;
  overflow-wrap: anywhere;
}

.metric-card__statistic {
  min-width: 0;
}

.metric-card__progress {
  flex: 0 0 auto;
}

.metric-card :deep(.t-statistic__content) {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}
</style>
