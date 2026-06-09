<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <t-drawer v-model:visible="visible" :header="title" :footer="false" placement="right" size="320px">
    <t-checkbox-group v-model="selectedKeys">
      <div class="advanced-query-column-drawer__grid">
        <t-checkbox v-for="column in columns" :key="column.value" :value="column.value">
          {{ column.label }}
        </t-checkbox>
      </div>
    </t-checkbox-group>
  </t-drawer>
</template>
<script setup lang="ts">
export type AdvancedQueryColumnOption = {
  label: string;
  value: string;
};

defineProps<{
  columns: AdvancedQueryColumnOption[];
  title: string;
}>();

const visible = defineModel<boolean>('visible', { required: true });
const selectedKeys = defineModel<string[]>('selectedKeys', { required: true });
</script>
<style scoped lang="less">
.advanced-query-column-drawer__grid {
  display: grid;
  gap: var(--graft-density-gap-12);
}
</style>
