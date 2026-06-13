<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <header :class="['management-page-header', { 'management-page-header--compact': compact }]">
    <page-header
      :breadcrumb="resolvedBreadcrumb"
      :source="source"
      :title-key="titleKey"
      :title-fallback="title"
      :description-key="descriptionKey"
      :description-fallback="description"
    >
      <template v-if="$slots.meta" #extra>
        <slot name="meta" />
      </template>
      <template v-if="$slots.actions" #actions>
        <slot name="actions" />
      </template>
    </page-header>
  </header>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import { PageHeader, type PageHeaderBreadcrumbItem, type PageHeaderSource } from '@/shared/components/page';

const props = defineProps<{
  title?: string;
  description?: string;
  titleKey?: string;
  descriptionKey?: string;
  breadcrumb?: PageHeaderBreadcrumbItem[];
  compact?: boolean;
  source?: PageHeaderSource;
}>();

const resolvedBreadcrumb = computed<PageHeaderBreadcrumbItem[]>(() => {
  if (props.breadcrumb) {
    return props.breadcrumb;
  }

  const titleKey = props.titleKey || props.title || '';
  const titleFallback = props.title || '';
  if (!props.source) {
    return [{ labelKey: titleKey, fallback: titleFallback }];
  }

  if (props.source.labelKey === titleKey || props.source.fallback === titleFallback) {
    return [{ labelKey: titleKey, fallback: titleFallback }];
  }

  return [
    { labelKey: props.source.labelKey, fallback: props.source.fallback },
    { labelKey: titleKey, fallback: titleFallback },
  ];
});
</script>
<style scoped lang="less">
@import './card-surface.less';

.management-page-header {
  .management-card-surface();

  padding: var(--graft-density-gap-18) var(--graft-density-gap-20);
}

.management-page-header--compact {
  padding: var(--graft-density-gap-14) var(--graft-density-gap-18);
}

.management-page-header--compact :deep(.page-header__main) {
  gap: var(--graft-density-gap-8);
}

.management-page-header--compact :deep(.page-header__description) {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (width <= 768px) {
  .management-page-header {
    padding: var(--graft-density-gap-16);
  }

  .management-page-header--compact :deep(.page-header__description) {
    white-space: normal;
  }
}
</style>
