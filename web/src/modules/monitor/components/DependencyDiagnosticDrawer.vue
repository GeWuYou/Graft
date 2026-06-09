<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <t-drawer v-model:visible="visible" :header="title" :footer="false" placement="right" size="420px" destroy-on-close>
    <dl v-if="diagnostics" class="dependency-diagnostic-drawer__list">
      <div
        v-for="item in diagnostics.items"
        :key="item.key"
        class="dependency-diagnostic-drawer__item"
        :data-diagnostic-key="item.key"
      >
        <dt>{{ item.label }}</dt>
        <dd>{{ item.value }}</dd>
      </div>
    </dl>
  </t-drawer>
</template>
<script setup lang="ts">
import type { DependencyHealthDiagnostics } from './DependencyHealthCard.vue';

defineProps<{
  title: string;
  diagnostics: DependencyHealthDiagnostics | null;
}>();

const visible = defineModel<boolean>('visible', { required: true });
</script>
<style scoped lang="less">
.dependency-diagnostic-drawer__list {
  display: grid;
  gap: var(--graft-density-gap-12);
  margin: 0;
}

.dependency-diagnostic-drawer__item {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-default);
  display: grid;
  gap: var(--graft-density-gap-8);
  min-width: 0;
  padding: var(--graft-density-gap-12);
}

.dependency-diagnostic-drawer__item dt {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.dependency-diagnostic-drawer__item dd {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  font-variant-numeric: tabular-nums;
  margin: 0;
  min-width: 0;
  overflow-wrap: anywhere;
}
</style>
