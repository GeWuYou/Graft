<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <article class="server-status-dependency-card">
    <header class="server-status-dependency-card__header">
      <div class="server-status-dependency-card__copy">
        <h3 class="server-status-dependency-card__title">{{ title }}</h3>
        <p v-if="description" class="server-status-dependency-card__description">{{ description }}</p>
      </div>
      <status-tag :label="statusLabel" :status="status" />
    </header>

    <div class="server-status-dependency-card__rows">
      <key-value-row
        v-for="item in items"
        :key="item.key"
        :label="item.label"
        :value="item.value"
        :description="item.description"
      />
    </div>
  </article>
</template>
<script setup lang="ts">
import KeyValueRow from './KeyValueRow.vue';
import type { ServerStatusTone } from './server-status-ui';
import StatusTag from './StatusTag.vue';

defineProps<{
  title: string;
  description?: string;
  status: ServerStatusTone;
  statusLabel: string;
  items: Array<{
    key: string;
    label: string;
    value: string;
    description?: string;
  }>;
}>();
</script>
<style scoped lang="less">
.server-status-dependency-card {
  background: var(--server-status-card-background-subtle, var(--td-bg-color-container-hover));
  border: 1px solid var(--server-status-card-border, var(--td-component-stroke));
  border-radius: calc(var(--td-radius-large) - 2px);
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: var(--graft-density-gap-16);
}

.server-status-dependency-card__header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  margin-bottom: var(--graft-density-gap-12);
}

.server-status-dependency-card__copy {
  min-width: 0;
}

.server-status-dependency-card__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0;
}

.server-status-dependency-card__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: var(--graft-density-gap-4) 0 0;
}
</style>
