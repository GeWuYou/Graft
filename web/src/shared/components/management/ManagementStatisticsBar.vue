<template>
  <section :class="['management-statistics-bar', `management-statistics-bar--${layout}`]" :aria-label="label">
    <template v-if="layout === 'summary'">
      <div class="management-statistics-bar__summary-content">
        <article v-for="(item, index) in items" :key="`${item.label}-${index}`" :class="itemClass(item)">
          <span class="management-statistics-bar__label">
            <span v-if="item.marker" class="management-statistics-bar__marker" aria-hidden="true">{{
              item.marker
            }}</span>
            {{ item.label }}
          </span>
          <strong class="management-statistics-bar__value">{{ item.value }}</strong>
        </article>
      </div>
    </template>
    <template v-else>
      <div class="management-statistics-bar__content">
        <template v-for="(item, index) in items" :key="`${item.label}-${index}`">
          <t-divider v-if="index > 0" layout="vertical" />
          <span :class="itemClass(item)">
            <span v-if="item.marker" class="management-statistics-bar__marker" aria-hidden="true">{{
              item.marker
            }}</span>
            <span class="management-statistics-bar__label">{{ item.label }}</span>
            <strong class="management-statistics-bar__value">{{ item.value }}</strong>
          </span>
        </template>
      </div>
      <div v-if="layout === 'chips'" class="management-statistics-bar__compact-content">
        <span
          v-for="(item, index) in compactItems.length ? compactItems : items"
          :key="`${item.label}-${index}`"
          :class="itemClass(item)"
        >
          <span v-if="item.marker" class="management-statistics-bar__marker" aria-hidden="true">{{ item.marker }}</span>
          <strong class="management-statistics-bar__value">{{ item.value }}</strong>
          <span class="management-statistics-bar__label">{{ item.label }}</span>
        </span>
      </div>
    </template>
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
function itemClass(item: ManagementStatisticItem) {
  return ['management-statistics-bar__item', item.tone && `management-statistics-bar__item--${item.tone}`];
}
</script>
<style scoped lang="less">
@import './card-surface.less';

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

// summary 布局在全部宽度下渲染为统计卡片：每项独立一列，窄容器自动换行。
.management-statistics-bar__summary-content {
  .management-card-surface();

  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(auto-fit, minmax(min(180px, 100%), 1fr));
  min-width: 0;
  padding: var(--graft-density-gap-14) var(--graft-density-gap-20);
  width: 100%;
}

.management-statistics-bar__summary-content .management-statistics-bar__item {
  align-items: flex-start;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.management-statistics-bar__summary-content .management-statistics-bar__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  min-width: 0;
  overflow-wrap: anywhere;
}

.management-statistics-bar__summary-content .management-statistics-bar__value {
  font: var(--td-font-title-large);
  font-variant-numeric: tabular-nums;
  line-height: 1.1;
}

.management-statistics-bar__item--success .management-statistics-bar__value {
  color: var(--td-success-color);
}

.management-statistics-bar__item--warning .management-statistics-bar__value {
  color: var(--td-warning-color);
}

.management-statistics-bar__item--danger .management-statistics-bar__value {
  color: var(--td-error-color);
}

.management-statistics-bar__item--muted .management-statistics-bar__value {
  color: var(--td-text-color-secondary);
}

.management-statistics-bar__item--muted .management-statistics-bar__label {
  color: var(--td-text-color-secondary);
}

.management-statistics-bar__summary-content .management-statistics-bar__marker {
  margin-inline-end: var(--graft-density-gap-4);
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

  .management-statistics-bar__summary-content {
    gap: var(--graft-density-gap-10);
    padding: var(--graft-density-gap-12) var(--graft-density-gap-16);
  }

  .management-statistics-bar__summary-content .management-statistics-bar__value {
    font: var(--td-font-title-medium);
  }
}
</style>
