<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="server-status-kv-row">
    <div class="server-status-kv-row__copy">
      <span class="server-status-kv-row__label">{{ label }}</span>
      <span v-if="description" class="server-status-kv-row__description">{{ description }}</span>
    </div>
    <strong class="server-status-kv-row__value">{{ value }}</strong>
  </div>
</template>
<script setup lang="ts">
defineProps<{
  label: string;
  value: string;
  description?: string;
}>();
</script>
<style scoped lang="less">
.server-status-kv-row {
  align-items: flex-start;
  border-top: 1px solid var(--td-component-border);
  display: flex;
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
  padding: var(--graft-density-gap-12) 0;
}

.server-status-kv-row:first-child {
  border-top: 0;
  padding-top: 0;
}

.server-status-kv-row:last-child {
  padding-bottom: 0;
}

.server-status-kv-row__copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.server-status-kv-row__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.server-status-kv-row__description {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
}

.server-status-kv-row__value {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  max-width: 56%;
  overflow-wrap: anywhere;
  text-align: right;
}

@media (width <= 575px) {
  .server-status-kv-row {
    flex-direction: column;
  }

  .server-status-kv-row__value {
    max-width: none;
    text-align: left;
  }
}
</style>
