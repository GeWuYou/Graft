<template>
  <section class="management-statistics-bar" :aria-label="label">
    <template v-for="(item, index) in items" :key="`${item.label}-${index}`">
      <t-divider v-if="index > 0" layout="vertical" />
      <span class="management-statistics-bar__item">
        <span v-if="item.marker" class="management-statistics-bar__marker" aria-hidden="true">{{ item.marker }}</span>
        <span class="management-statistics-bar__label">{{ item.label }}</span>
        <strong class="management-statistics-bar__value">{{ item.value }}</strong>
      </span>
    </template>
  </section>
</template>
<script lang="ts">
export type ManagementStatisticItem = {
  label: string;
  marker?: string;
  value: number | string;
};
</script>
<script setup lang="ts">
// 统计条只统一行内聚合信息的布局，资源统计口径和状态语义始终由所属页面提供。
withDefaults(
  defineProps<{
    items: ManagementStatisticItem[];
    label?: string;
  }>(),
  {
    label: '',
  },
);
</script>
<style scoped lang="less">
.management-statistics-bar {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  gap: var(--graft-density-gap-12);
  min-width: 0;
  overflow-x: auto;
  padding: var(--graft-density-gap-4) 0;
  white-space: nowrap;
}

.management-statistics-bar__item {
  align-items: baseline;
  display: inline-flex;
  flex: 0 0 auto;
  gap: var(--graft-density-gap-4);
}

.management-statistics-bar__label,
.management-statistics-bar__marker {
  font: var(--td-font-body-medium);
}

.management-statistics-bar__value {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 600;
}

.management-statistics-bar :deep(.t-divider--vertical) {
  height: 1em;
  margin: 0;
}
</style>
