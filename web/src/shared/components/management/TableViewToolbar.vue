<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="table-view-toolbar">
    <slot name="before" />
    <t-tooltip v-if="refreshLabel" :content="refreshLabel" placement="top">
      <t-button
        :aria-label="refreshLabel"
        :loading="refreshLoading"
        shape="square"
        theme="default"
        variant="outline"
        @click="$emit('refresh')"
      >
        <template #icon><refresh-icon /></template>
      </t-button>
    </t-tooltip>
    <t-tooltip v-if="columnSettingsLabel" :content="columnSettingsLabel" placement="top">
      <t-button
        :aria-label="columnSettingsLabel"
        shape="square"
        theme="default"
        variant="outline"
        @click="$emit('column-settings')"
      >
        <template #icon><view-column-icon /></template>
      </t-button>
    </t-tooltip>
    <t-tooltip v-if="densityLabel" :content="densityLabel" placement="top">
      <t-button :aria-label="densityLabel" shape="square" theme="default" variant="outline" @click="$emit('density')">
        <template #icon><view-module-icon /></template>
      </t-button>
    </t-tooltip>
    <slot />
  </div>
</template>
<script setup lang="ts">
import { RefreshIcon, ViewColumnIcon, ViewModuleIcon } from 'tdesign-icons-vue-next';

defineProps<{
  columnSettingsLabel?: string;
  densityLabel?: string;
  refreshLabel?: string;
  refreshLoading?: boolean;
}>();

defineEmits<{
  (e: 'column-settings'): void;
  (e: 'density'): void;
  (e: 'refresh'): void;
}>();
</script>
<style scoped lang="less">
.table-view-toolbar {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  justify-content: flex-end;
}

@media (width <= 768px) {
  .table-view-toolbar {
    justify-content: flex-start;
  }
}
</style>
