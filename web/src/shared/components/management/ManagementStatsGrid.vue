<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <section class="management-stats-grid" :class="`management-stats-grid--${layout}`">
    <article v-for="(item, index) in items" :key="`${item.label}-${index}`" class="management-stats-grid__item">
      <div class="management-stats-grid__head">
        <span class="management-stats-grid__label">{{ item.label }}</span>
        <span v-if="item.tip" class="management-stats-grid__tip">{{ item.tip }}</span>
      </div>
      <strong class="management-stats-grid__value">{{ item.value }}</strong>
      <p v-if="item.description" class="management-stats-grid__description">{{ item.description }}</p>
    </article>
  </section>
</template>
<script setup lang="ts">
export type ManagementStatItem = {
  label: string;
  value: string | number;
  description?: string;
  tip?: string;
};

withDefaults(
  defineProps<{
    items: ManagementStatItem[];
    layout?: 'auto' | 'dashboard' | 'compact';
  }>(),
  {
    layout: 'auto',
  },
);
</script>
<style scoped lang="less">
@import './card-surface.less';

.management-stats-grid {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(auto-fit, minmax(180px, minmax(0, 1fr)));
}

.management-stats-grid--dashboard {
  grid-template-columns: repeat(5, minmax(0, 1fr));
}

.management-stats-grid--compact {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.management-stats-grid__item {
  .management-card-surface();

  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  min-height: 112px;
  padding: var(--graft-density-gap-14) var(--graft-density-gap-16);
}

.management-stats-grid--compact .management-stats-grid__item {
  gap: var(--graft-density-gap-6);
  min-height: 96px;
  padding: var(--graft-density-gap-12) var(--graft-density-gap-14);
}

.management-stats-grid__head {
  color: var(--td-text-color-secondary);
  display: flex;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
}

.management-stats-grid__tip,
.management-stats-grid__description {
  color: var(--td-text-color-placeholder);
}

.management-stats-grid__value {
  color: var(--graft-metric-value-color);
  font: var(--td-font-title-large);
  line-height: 1.1;
}

.management-stats-grid__description {
  font: var(--td-font-body-small);
  margin: 0;
}

@media (width <= 1199px) {
  .management-stats-grid--dashboard,
  .management-stats-grid--compact {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 767px) {
  .management-stats-grid--dashboard,
  .management-stats-grid--compact {
    grid-template-columns: 1fr;
  }
}
</style>
