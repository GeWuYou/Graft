<template>
  <section class="management-list-page" :data-page-type="pageType">
    <management-page-header
      v-if="headerVisible"
      :compact="compact"
      :description="description"
      :description-key="descriptionKey"
      :title="title"
      :title-key="titleKey"
      :source="source"
      :action-layout="actionLayout"
    >
      <template v-if="$slots.meta" #meta>
        <slot name="meta" />
      </template>
      <template v-if="$slots.actions" #actions>
        <slot name="actions" />
      </template>
      <template v-if="$slots.compactActions" #compactActions>
        <slot name="compactActions" />
      </template>
    </management-page-header>
    <slot name="statistics">
      <management-statistics-bar
        v-if="statsItems?.length"
        :items="statsItems"
        :compact-items="statsCompactItems"
        :label="statsLabel"
        :layout="statsLayout"
        aria-live="polite"
      />
    </slot>
    <slot name="filters" />
    <slot name="feedback" />
    <slot />
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import type { PageHeaderSource } from '@/shared/components/page';

import ManagementPageHeader from './ManagementPageHeader.vue';
import ManagementStatisticsBar from './ManagementStatisticsBar.vue';
import type { ManagementStatisticItem } from './statistics';

// 管理列表页只收敛页面骨架的节奏与表面：header、统计、筛选与反馈槽的顺序和间距；
// 列定义、查询状态、分页语义和业务操作仍由业务页面拥有。
const props = withDefaults(
  defineProps<{
    actionLayout?: 'responsive' | 'inline';
    compact?: boolean;
    description?: string;
    descriptionKey?: string;
    pageType?: string;
    source?: PageHeaderSource;
    statsCompactItems?: ManagementStatisticItem[];
    statsItems?: ManagementStatisticItem[];
    statsLabel?: string;
    statsLayout?: 'chips' | 'inline' | 'summary';
    title?: string;
    titleKey?: string;
  }>(),
  {
    actionLayout: 'responsive',
    compact: false,
    description: '',
    descriptionKey: '',
    pageType: 'list-form-detail',
    source: undefined,
    statsCompactItems: () => [],
    statsItems: () => [],
    statsLabel: '',
    statsLayout: 'inline',
    title: '',
    titleKey: '',
  },
);

const headerVisible = computed(() =>
  Boolean(props.title || props.titleKey || props.description || props.descriptionKey),
);
</script>
<style scoped lang="less">
.management-list-page {
  container-type: inline-size;
  display: grid;
  gap: var(--graft-density-gap-16);
  min-width: 0;
  width: 100%;
}

@media (width <= 768px) {
  .management-list-page {
    gap: var(--graft-density-gap-12);
  }
}
</style>
