<template>
  <section :class="['management-statistics-bar', `management-statistics-bar--${layout}`]" :aria-label="label">
    <div class="management-statistics-bar__content">
      <template v-for="(item, index) in items" :key="`${item.label}-${index}`">
        <t-divider v-if="index > 0" layout="vertical" />
        <span class="management-statistics-bar__item">
          <span v-if="item.marker" class="management-statistics-bar__marker" aria-hidden="true">{{ item.marker }}</span>
          <span class="management-statistics-bar__label">{{ item.label }}</span>
          <strong class="management-statistics-bar__value">{{ item.value }}</strong>
        </span>
      </template>
    </div>
    <div v-if="layout === 'chips'" class="management-statistics-bar__compact-content">
      <span
        v-for="(item, index) in compactItems.length ? compactItems : items"
        :key="`${item.label}-${index}`"
        class="management-statistics-bar__item"
      >
        <span v-if="item.marker" class="management-statistics-bar__marker" aria-hidden="true">{{ item.marker }}</span>
        <strong class="management-statistics-bar__value">{{ item.value }}</strong>
        <span class="management-statistics-bar__label">{{ item.label }}</span>
      </span>
    </div>
  </section>
</template>
<script setup lang="ts">
import type { ManagementStatisticItem } from './statistics';

// 统计条只统一行内聚合信息的布局，资源统计口径和状态语义始终由所属页面提供。
withDefaults(
  defineProps<{
    items: ManagementStatisticItem[];
    compactItems?: ManagementStatisticItem[];
    label?: string;
    layout?: 'chips' | 'inline' | 'summary';
  }>(),
  {
    label: '',
    layout: 'inline',
    compactItems: () => [],
  },
);
</script>
<style scoped lang="less">
.management-statistics-bar {
  container-type: inline-size;
  min-width: 0;
  width: 100%;
}

.management-statistics-bar__content {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  gap: var(--graft-density-gap-12);
  min-width: 0;
  overflow-x: auto;
  padding: var(--graft-density-gap-4) 0;
  white-space: nowrap;
  width: 100%;
}

.management-statistics-bar__compact-content {
  display: none;
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

@container (width < 768px) {
  .management-statistics-bar--chips .management-statistics-bar__content {
    display: none;
  }

  .management-statistics-bar--chips .management-statistics-bar__compact-content {
    align-items: center;
    display: flex;
    flex-wrap: wrap;
    gap: var(--graft-density-gap-8);
    min-width: 0;
  }

  .management-statistics-bar--chips .management-statistics-bar__compact-content .management-statistics-bar__item {
    align-items: baseline;
    background: var(--td-bg-color-container);
    border: solid 1px var(--td-component-stroke);
    border-radius: var(--td-radius-medium);
    gap: var(--graft-density-gap-4);
    min-width: 0;
    padding-block: var(--graft-density-gap-6);
    padding-inline: var(--graft-density-gap-8);
  }

  .management-statistics-bar--chips .management-statistics-bar__compact-content .management-statistics-bar__label,
  .management-statistics-bar--chips .management-statistics-bar__compact-content .management-statistics-bar__value {
    font: var(--td-font-body-small);
  }

  .management-statistics-bar--summary .management-statistics-bar__content {
    border: 1px solid var(--td-component-stroke);
    border-radius: var(--td-radius-medium);
    display: grid;
    gap: 0;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    overflow: hidden;
    padding: 0;
    white-space: normal;
  }

  .management-statistics-bar--summary .management-statistics-bar__item {
    align-items: center;
    flex-direction: column;
    gap: var(--graft-density-gap-4);
    min-width: 0;
    padding: var(--graft-density-gap-12) var(--graft-density-gap-8);
    text-align: center;
  }

  .management-statistics-bar--summary .management-statistics-bar__item ~ .management-statistics-bar__item {
    border-left: 1px solid var(--td-component-stroke);
  }

  .management-statistics-bar--summary .management-statistics-bar__label {
    overflow-wrap: anywhere;
  }

  .management-statistics-bar--summary .management-statistics-bar__value {
    font: var(--td-font-title-small);
  }

  .management-statistics-bar--summary :deep(.t-divider--vertical) {
    display: none;
  }
}
</style>
