<template>
  <div :class="['table-view-toolbar', { 'table-view-toolbar--compact': compact }]">
    <slot name="before" />
    <t-tooltip v-if="refreshVisible && refreshLabel" :content="refreshLabel" placement="top">
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
      v-if="columnSettingsVisible && resolvedColumnSettingsLabel && variant.density !== 'compact'"
      :content="resolvedColumnSettingsLabel"
      placement="top"
    >
      <t-button
        :aria-label="resolvedColumnSettingsLabel"
        class="table-view-toolbar__button"
        theme="default"
        variant="outline"
        @click="handleColumnSettings"
      >
        <template #icon><view-column-icon /></template>
        <span v-if="!compact">{{ resolvedColumnSettingsLabel }}</span>
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
    <t-tooltip
      v-if="densityVisible && resolvedDensityLabel && !compact"
      :content="resolvedDensityLabel"
      placement="top"
    >
      <t-button
        :aria-label="resolvedDensityLabel"
        class="table-view-toolbar__button"
        theme="default"
        variant="outline"
        @click="handleDensity"
      >
        <template #icon><view-module-icon /></template>
        <span>{{ resolvedDensityLabel }}</span>
      </t-button>
    </t-tooltip>
    <slot />
  </div>
</template>
<script setup lang="ts">
import { EllipsisIcon, RefreshIcon, ViewColumnIcon, ViewModuleIcon } from 'tdesign-icons-vue-next';
import { computed, inject } from 'vue';

import { useViewportResponsiveVariant } from '@/shared/composables';

import { managementTableViewToolsKey } from './table-view-tools';

// 标准工具栏统一按钮顺序与响应式折叠；页面传入标签时保留自有行为，否则复用分页表格上下文。
const {
  columnSettingsLabel = '',
  columnSettingsVisible = true,
  densityLabel = '',
  densityVisible = true,
  moreLabel = '',
  refreshLabel = '',
  refreshLoading = false,
  refreshVisible = true,
} = defineProps<{
  columnSettingsLabel?: string;
  columnSettingsVisible?: boolean;
  densityLabel?: string;
  densityVisible?: boolean;
  moreLabel?: string;
  refreshLabel?: string;
  refreshLoading?: boolean;
  refreshVisible?: boolean;
}>();

const emit = defineEmits<{
  (e: 'column-settings'): void;
  (e: 'density'): void;
  (e: 'refresh'): void;
}>();

const variant = useViewportResponsiveVariant();
const managedTools = inject(managementTableViewToolsKey, null);
const compact = computed(() => variant.value.density === 'compact');
const resolvedColumnSettingsLabel = computed(
  () => columnSettingsLabel || managedTools?.columnSettingsLabel.value || '',
);
const resolvedDensityLabel = computed(() => densityLabel || managedTools?.densityLabel.value || '');
const resolvedMoreLabel = computed(() => moreLabel || resolvedColumnSettingsLabel.value || resolvedDensityLabel.value);
const compactOverflowOptions = computed(() => {
  if (variant.value.density !== 'compact') return [];

  return [
    ...(columnSettingsVisible && resolvedColumnSettingsLabel.value
      ? [{ content: resolvedColumnSettingsLabel.value, value: 'column-settings' }]
      : []),
    ...(densityVisible && resolvedDensityLabel.value
      ? [{ content: resolvedDensityLabel.value, value: 'density' }]
      : []),
  ];
});

function handleColumnSettings() {
  if (columnSettingsLabel || !managedTools) {
    emit('column-settings');
    return;
  }
  managedTools.openColumnSettings();
}

function handleDensity() {
  if (densityLabel || !managedTools) {
    emit('density');
    return;
  }
  managedTools.toggleDensity();
}

function handleOverflowAction(payload: unknown) {
  const action =
    typeof payload === 'object' && payload !== null && 'value' in payload
      ? (payload as { value?: unknown }).value
      : payload;
  if (action === 'column-settings') handleColumnSettings();
  if (action === 'density') handleDensity();
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
