<template>
  <t-drawer v-model:visible="visible" :header="title" :footer="false" placement="right" size="320px">
    <div v-if="viewPresets?.length" class="advanced-query-column-drawer__presets">
      <p class="advanced-query-column-drawer__presets-title">{{ presetsLabel }}</p>
      <t-space break-line size="small">
        <t-button
          v-for="preset in viewPresets"
          :key="preset.value"
          size="small"
          theme="default"
          variant="outline"
          @click="applyPreset(preset.keys)"
        >
          {{ preset.label }}
        </t-button>
      </t-space>
    </div>
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
  presetsLabel?: string;
  resetLabel?: string;
  title: string;
  viewPresets?: Array<{
    keys: string[];
    label: string;
    value: string;
  }>;
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

function applyPreset(keys: string[]) {
  selectedKeys.value = normalizeSelectedKeys(keys);
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

.advanced-query-column-drawer__presets {
  border-bottom: 1px solid var(--td-border-level-1-color);
  margin-bottom: var(--graft-density-gap-16);
  padding-bottom: var(--graft-density-gap-16);
}

.advanced-query-column-drawer__presets-title {
  color: var(--td-text-color-secondary);
  margin: 0 0 var(--graft-density-gap-8);
}

.advanced-query-column-drawer__footer {
  border-top: 1px solid var(--td-border-level-1-color);
  margin-top: var(--graft-density-gap-16);
  padding-top: var(--graft-density-gap-16);
}
</style>
