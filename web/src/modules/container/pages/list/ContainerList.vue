<template>
  <section ref="container" class="container-list" :data-presentation="effectivePresentation">
    <container-resource-table
      v-bind="props"
      :current="current"
      :force-cards="effectivePresentation === 'card'"
      :page-size="pageSize"
      :presentation="effectivePresentation === 'card' ? 'entity' : 'data'"
      @update:current="$emit('update:current', $event)"
      @update:page-size="$emit('update:page-size', $event)"
      @action="$emit('action', $event)"
      @application-context="$emit('application-context', $event)"
      @page-change="$emit('page-change', $event)"
      @select-change="$emit('select-change', $event)"
    >
      <template #toolbar>
        <slot name="toolbar" :presentation="effectivePresentation" :desktop="desktop" />
      </template>
      <template #batch>
        <slot name="batch" />
      </template>
      <template #feedback>
        <slot name="feedback" />
      </template>
      <template #cards>
        <div class="container-list__cards">
          <container-card
            v-for="row in rows"
            :key="row.id"
            :row="row"
            :actions="rowActions(row)"
            :selected="selectedRowKeys.includes(row.id)"
            @detail="$emit('detail', $event)"
            @action="$emit('action', $event)"
            @select-change="toggleCardSelection(row.id, $event)"
          />
        </div>
      </template>
      <template #empty-action><slot name="empty-action" /></template>
    </container-resource-table>
  </section>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';

import { useResponsiveVariant } from '@/shared/composables';

import ContainerResourceTable from '../../components/ContainerResourceTable.vue';
import type { ContainerResourceRowAction } from '../../shared/resource-table';
import type { ContainerSummaryRecord } from '../../types/container';
import ContainerCard from './ContainerCard.vue';
/** 列表仅在展示层选择卡片或表格，数据加载和操作所有权始终在页面入口。 */
const props = defineProps<{
  rows: ContainerSummaryRecord[];
  rowActions: (row: ContainerSummaryRecord) => ContainerResourceRowAction[];
  presentation: 'table' | 'card';
  loading: boolean;
  total: number;
  current: number;
  pageSize: number;
  emptyTitle: string;
  emptyDescription: string;
  footerSummary: string;
  visibleColumnKeys: string[];
  selectedRowKeys: Array<string | number>;
  tableDensity: 'medium' | 'small';
  alwaysVisibleColumnKeys: string[];
  composeApplicationReferences: Map<string, { applicationId: string; displayName: string }>;
  headDescription: string;
  headSummary: string;
  moreActionsLabel: string;
}>();
const emit = defineEmits<{
  detail: [row: ContainerSummaryRecord];
  action: [payload: { action: string; row: ContainerSummaryRecord }];
  'application-context': [applicationId: string];
  'page-change': [pageInfo: { current?: number; pageSize?: number }];
  'select-change': [rowKeys: Array<string | number>];
  'update:current': [value: number];
  'update:page-size': [value: number];
}>();
const container = ref<HTMLElement | null>(null);
const variant = useResponsiveVariant(container, { presentation: 'entity' });
const desktop = computed(() => variant.value.density === 'spacious');
const effectivePresentation = computed(() => (desktop.value ? props.presentation : 'card'));

function toggleCardSelection(rowID: string, checked: boolean) {
  const next = new Set(props.selectedRowKeys.map((key) => String(key)));
  if (checked) {
    next.add(rowID);
  } else {
    next.delete(rowID);
  }
  emit('select-change', [...next]);
}
</script>
<style scoped lang="less">
.container-list {
  container-type: inline-size;
  min-width: 0;
}

.container-list__cards {
  display: grid;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.container-list__cards > :last-child {
  justify-self: end;
}

@container (width >= 768px) {
  .container-list__cards {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .container-list__cards > :last-child {
    grid-column: 1 / -1;
  }
}
</style>
