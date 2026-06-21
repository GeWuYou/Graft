<template>
  <header :class="['management-page-header', { 'management-page-header--compact': compact }]">
    <page-header
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
import { PageHeader, type PageHeaderSource } from '@/shared/components/page';

defineProps<{
  title?: string;
  description?: string;
  titleKey?: string;
  descriptionKey?: string;
  compact?: boolean;
  source?: PageHeaderSource;
}>();
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
