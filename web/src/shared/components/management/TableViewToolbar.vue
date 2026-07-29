<template>
  <div :class="['table-view-toolbar', { 'table-view-toolbar--compact': compact }]">
    <slot name="before" />
    <t-tooltip v-if="refreshLabel" :content="refreshLabel" placement="top">
      <t-button
        :aria-label="refreshLabel"
        :loading="refreshLoading"
        class="table-view-toolbar__button"
        theme="default"
        variant="outline"
        @click="$emit('refresh')"
      >
        <template #icon><refresh-icon /></template>
        <span v-if="!compact">{{ refreshLabel }}</span>
      </t-button>
    </t-tooltip>
    <t-tooltip
      v-if="columnSettingsLabel && variant.density !== 'compact'"
      :content="columnSettingsLabel"
      placement="top"
    >
      <t-button
        :aria-label="columnSettingsLabel"
        class="table-view-toolbar__button"
        theme="default"
        variant="outline"
        @click="$emit('column-settings')"
      >
        <template #icon><view-column-icon /></template>
        <span v-if="!compact">{{ columnSettingsLabel }}</span>
      </t-button>
    </t-tooltip>
    <t-dropdown
      v-if="compactOverflowOptions.length"
      :options="compactOverflowOptions"
      trigger="click"
      @click="handleOverflowAction"
    >
      <t-button :aria-label="resolvedMoreLabel" shape="square" theme="default" variant="outline">
        <template #icon><ellipsis-icon /></template>
      </t-button>
    </t-dropdown>
    <t-tooltip v-if="densityLabel && !compact" :content="densityLabel" placement="top">
      <t-button :aria-label="densityLabel" shape="square" theme="default" variant="outline" @click="$emit('density')">
        <template #icon><view-module-icon /></template>
      </t-button>
    </t-tooltip>
    <slot />
  </div>
</template>
<script setup lang="ts">
import { EllipsisIcon, RefreshIcon, ViewColumnIcon, ViewModuleIcon } from 'tdesign-icons-vue-next';
import { computed } from 'vue';

import { useViewportResponsiveVariant } from '@/shared/composables';

const {
  columnSettingsLabel = '',
  densityLabel = '',
  moreLabel = '',
  refreshLabel = '',
  refreshLoading = false,
} = defineProps<{
  columnSettingsLabel?: string;
  densityLabel?: string;
  moreLabel?: string;
  refreshLabel?: string;
  refreshLoading?: boolean;
}>();

const emit = defineEmits<{
  (e: 'column-settings'): void;
  (e: 'density'): void;
  (e: 'refresh'): void;
}>();

const variant = useViewportResponsiveVariant();
const compact = computed(() => variant.value.density === 'compact');
const resolvedMoreLabel = computed(() => moreLabel || columnSettingsLabel || densityLabel);
const compactOverflowOptions = computed(() => {
  if (variant.value.density !== 'compact') return [];

  return [
    ...(columnSettingsLabel ? [{ content: columnSettingsLabel, value: 'column-settings' }] : []),
    ...(densityLabel ? [{ content: densityLabel, value: 'density' }] : []),
  ];
});

function handleOverflowAction(payload: unknown) {
  const action =
    typeof payload === 'object' && payload !== null && 'value' in payload
      ? (payload as { value?: unknown }).value
      : payload;
  if (action === 'column-settings') emit('column-settings');
  if (action === 'density') emit('density');
}
</script>
<style scoped lang="less">
.table-view-toolbar {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  justify-content: flex-end;
}

.table-view-toolbar__button {
  flex: 0 0 auto;
}

.table-view-toolbar--compact .table-view-toolbar__button {
  min-width: auto;
}

@media (width <= 768px) {
  .table-view-toolbar {
    justify-content: flex-start;
  }
}
</style>
