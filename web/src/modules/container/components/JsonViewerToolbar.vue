<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="json-viewer-toolbar">
    <div class="json-viewer-toolbar__search">
      <t-input
        :model-value="searchValue"
        clearable
        :placeholder="searchPlaceholder"
        @update:model-value="handleSearchChange"
      />
    </div>

    <div class="json-viewer-toolbar__actions">
      <t-radio-group
        :model-value="viewMode"
        variant="default-filled"
        theme="button"
        :options="viewOptions"
        @update:model-value="handleViewModeChange"
      />
      <t-tooltip :content="copyTooltip">
        <span class="json-viewer-toolbar__copy-wrap">
          <t-button theme="default" variant="outline" :disabled="copyDisabled" @click="$emit('copy')">
            {{ copyLabel }}
          </t-button>
        </span>
      </t-tooltip>
      <t-button
        v-if="viewMode === 'tree'"
        theme="default"
        variant="outline"
        :disabled="expandDisabled"
        @click="$emit('toggle-expand-all')"
      >
        {{ expandedAll ? collapseAllLabel : expandAllLabel }}
      </t-button>
      <t-button v-else theme="default" variant="outline" :disabled="formatDisabled" @click="$emit('format')">
        {{ formatLabel }}
      </t-button>
    </div>
  </div>
</template>
<script setup lang="ts">
type RawJsonViewMode = 'tree' | 'source';

const props = defineProps<{
  collapseAllLabel: string;
  copyDisabled?: boolean;
  copyLabel: string;
  copyTooltip: string;
  expandAllLabel: string;
  expandDisabled?: boolean;
  expandedAll: boolean;
  formatDisabled?: boolean;
  formatLabel: string;
  searchPlaceholder: string;
  searchValue: string;
  sourceLabel: string;
  treeLabel: string;
  viewMode: RawJsonViewMode;
}>();

const emit = defineEmits<{
  copy: [];
  format: [];
  'toggle-expand-all': [];
  'update:search-value': [value: string];
  'update:view-mode': [value: RawJsonViewMode];
}>();

const viewOptions = [
  { label: props.treeLabel, value: 'tree' },
  { label: props.sourceLabel, value: 'source' },
];

function handleSearchChange(value: string | number) {
  emit('update:search-value', String(value ?? ''));
}

function handleViewModeChange(value: string | number | boolean) {
  emit('update:view-mode', value === 'source' ? 'source' : 'tree');
}
</script>
<style scoped lang="less">
.json-viewer-toolbar {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-10);
  justify-content: space-between;
  min-width: 0;
}

.json-viewer-toolbar__search {
  flex: 1 1 280px;
  min-width: min(100%, 280px);
}

.json-viewer-toolbar__actions {
  align-items: center;
  display: flex;
  flex: 0 1 auto;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  justify-content: flex-end;
  min-width: 0;
}

.json-viewer-toolbar__copy-wrap {
  display: inline-flex;
}

.json-viewer-toolbar__search :deep(.t-input) {
  width: 100%;
}
</style>
