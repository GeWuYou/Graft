<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <t-drawer v-model:visible="visible" :header="title" :footer="false" placement="right" size="320px">
    <t-checkbox-group v-model="selectedColumnKeys">
      <div class="advanced-query-column-drawer__grid">
        <t-checkbox
          v-for="column in columns"
          :key="column.value"
          :disabled="disabledKeySet.has(column.value)"
          :value="column.value"
        >
          {{ column.label }}
        </t-checkbox>
      </div>
    </t-checkbox-group>
    <div v-if="resetLabel && defaultSelectedKeys?.length" class="advanced-query-column-drawer__footer">
      <t-button theme="default" variant="outline" block @click="resetColumns">
        {{ resetLabel }}
      </t-button>
    </div>
  </t-drawer>
</template>
<script setup lang="ts">
import { computed } from 'vue';

export type AdvancedQueryColumnOption = {
  label: string;
  value: string;
};

const props = defineProps<{
  columns: AdvancedQueryColumnOption[];
  defaultSelectedKeys?: string[];
  disabledKeys?: string[];
  resetLabel?: string;
  title: string;
}>();

const visible = defineModel<boolean>('visible', { required: true });
const selectedKeys = defineModel<string[]>('selectedKeys', { required: true });

const disabledKeySet = computed(() => new Set(props.disabledKeys ?? []));

const selectedColumnKeys = computed({
  get: () => selectedKeys.value,
  set: (keys: string[]) => {
    selectedKeys.value = normalizeSelectedKeys(keys);
  },
});

function resetColumns() {
  selectedKeys.value = normalizeSelectedKeys(props.defaultSelectedKeys ?? []);
}

function normalizeSelectedKeys(keys: string[]) {
  const nextKeys = new Set(keys);
  for (const key of disabledKeySet.value) {
    nextKeys.add(key);
  }
  return Array.from(nextKeys);
}
</script>
<style scoped lang="less">
.advanced-query-column-drawer__grid {
  display: grid;
  gap: var(--graft-density-gap-12);
}

.advanced-query-column-drawer__footer {
  border-top: 1px solid var(--td-border-level-1-color);
  margin-top: var(--graft-density-gap-16);
  padding-top: var(--graft-density-gap-16);
}
</style>
