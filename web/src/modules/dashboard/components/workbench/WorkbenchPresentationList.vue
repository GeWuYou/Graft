<template>
  <t-list class="workbench-list" :class="{ 'workbench-list--quiet': variant === 'health' }" split>
    <workbench-presentation-row
      v-for="item in visibleItems"
      :key="item.id"
      :item="item"
      :retrying-id="props.retryingId"
      :variant="variant"
      @activate="handleAction"
    />
  </t-list>

  <p v-if="!items.length && emptyKey" class="workbench-empty">
    {{ t(emptyKey) }}
  </p>

  <t-collapse
    v-if="remainingItems.length"
    v-model="expandedPanels"
    borderless
    :expand-icon="false"
    :expand-on-row-click="false"
    class="workbench-collapse"
  >
    <t-collapse-panel :value="panelValue">
      <template #header>
        <t-button
          block
          theme="primary"
          type="button"
          variant="text"
          :aria-controls="panelContentId"
          :aria-expanded="overflowExpanded"
          :data-collapse-trigger="panelValue"
          @click.stop="toggleOverflow"
          @keydown.enter.prevent.stop="toggleOverflow"
          @keydown.space.prevent.stop="toggleOverflow"
        >
          {{ t(expandKey, { count: remainingItems.length }) }}
          <template #suffix>
            <chevron-down-icon :style="{ transform: overflowExpanded ? 'rotate(180deg)' : undefined }" />
          </template>
        </t-button>
      </template>
      <div :id="panelContentId" class="workbench-collapse__content" :data-collapse-content="panelValue">
        <t-list class="workbench-list" :class="{ 'workbench-list--quiet': variant === 'health' }" split>
          <workbench-presentation-row
            v-for="item in remainingItems"
            :key="item.id"
            :item="item"
            :retrying-id="props.retryingId"
            :variant="variant"
            @activate="handleAction"
          />
        </t-list>
      </div>
    </t-collapse-panel>
  </t-collapse>
</template>
<script setup lang="ts">
import { ChevronDownIcon } from 'tdesign-icons-vue-next';
import { computed, ref } from 'vue';

import { t } from '@/locales';

import type { PresentationItem } from '../../presentation/workbench';
import WorkbenchPresentationRow from './WorkbenchPresentationRow.vue';

// 展示列表统一工作台事实行、折叠预算和下钻语义，父组件只决定区域信息层级。
const props = withDefaults(
  defineProps<{
    emptyKey?: string;
    expandKey?: string;
    items: PresentationItem[];
    retryingId?: string;
    variant: 'attention' | 'health' | 'activity';
    visibleLimit?: number;
  }>(),
  {
    emptyKey: '',
    expandKey: 'dashboard.workbench.expand.attention',
    retryingId: '',
    visibleLimit: Number.MAX_SAFE_INTEGER,
  },
);

const emit = defineEmits<{
  navigate: [route: string];
  'retry-item': [item: PresentationItem];
}>();

const expandedPanels = ref<string[]>([]);
const visibleItems = computed(() => props.items.slice(0, props.visibleLimit));
const remainingItems = computed(() => props.items.slice(props.visibleLimit));
const panelValue = computed(() => `${props.variant}-more`);
const panelContentId = computed(() => `workbench-${panelValue.value}-content`);
const overflowExpanded = computed(() => expandedPanels.value.includes(panelValue.value));

function toggleOverflow() {
  expandedPanels.value = overflowExpanded.value ? [] : [panelValue.value];
}

function handleAction(item: PresentationItem) {
  if (item.action?.kind === 'retry') {
    emit('retry-item', item);
    return;
  }
  if (item.action?.kind === 'navigate') {
    emit('navigate', item.action.route);
  }
}
</script>
<style scoped lang="less">
.workbench-list,
.workbench-list :deep(.t-list-item) {
  background: transparent;
}

.workbench-list :deep(.t-list-item) {
  padding: var(--graft-density-gap-16) 0;
}

.workbench-list--quiet :deep(.t-list-item) {
  padding: var(--graft-density-gap-8) 0;
}

.workbench-empty {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-large);
  margin: 0;
  padding: var(--graft-density-gap-16) 0;
}

.workbench-collapse {
  background: transparent;
  margin-top: var(--graft-density-gap-8);
}

.workbench-collapse :deep(.t-collapse-panel__body) {
  background: transparent;
}

.workbench-collapse :deep(.t-collapse-panel__content) {
  color: inherit;
  padding: 0;
}
</style>
