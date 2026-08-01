<template>
  <section class="resource-query-panel" :data-resource="config.resource" data-testid="resource-query-panel">
    <advanced-query-filter-builder-frame v-if="frame" :frame="frame" v-bind="{ messagePrefix }">
      <template #saved-query-views><slot name="saved-query-views" /></template>
      <template #toolbar-after-search><slot name="toolbar-after-search" /></template>
    </advanced-query-filter-builder-frame>
    <graft-query-bar
      v-else
      :config="barConfig"
      :loading="loading"
      :model-value="modelValue ?? emptyQueryState"
      @reset="emit('reset', $event)"
      @search="emit('search', $event)"
      @update:model-value="emit('update:modelValue', $event)"
    >
      <template #quick="slotProps"><slot name="quick" v-bind="slotProps" /></template>
    </graft-query-bar>
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import AdvancedQueryFilterBuilderFrame, {
  type AdvancedQueryFilterBuilderFrameState,
} from '../AdvancedQueryFilterBuilderFrame.vue';
import type { GraftQueryState } from '../graft-query-bar';
import GraftQueryBar from '../GraftQueryBar.vue';
import type { ResourceQueryConfig } from './types';

// Resource Query 提供基础查询模型；复杂日志页通过 frame 保留其领域字段与事件契约。
const props = withDefaults(
  defineProps<{
    config: ResourceQueryConfig;
    frame?: AdvancedQueryFilterBuilderFrameState;
    loading?: boolean;
    messagePrefix?: string;
    modelValue?: GraftQueryState;
  }>(),
  { frame: undefined, loading: false, messagePrefix: '', modelValue: undefined },
);

const emit = defineEmits<{
  (e: 'reset', value: GraftQueryState): void;
  (e: 'search', value: GraftQueryState): void;
  (e: 'update:modelValue', value: GraftQueryState): void;
}>();

const emptyQueryState: GraftQueryState = { keyword: '', filters: {}, page: 1, pageSize: 20 };

const barConfig = computed(() => ({
  ...props.config,
  filters: props.config.filterBuilder?.enabled === false ? [] : props.config.filters,
}));
</script>
<style scoped lang="less">
.resource-query-panel {
  min-width: 0;
}
</style>
