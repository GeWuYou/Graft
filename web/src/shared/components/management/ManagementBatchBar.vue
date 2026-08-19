<template>
  <div
    class="management-batch-bar"
    :class="{
      'management-batch-bar--has-compact-actions': compactMenuOptions.length,
    }"
  >
    <span class="management-batch-bar__summary">{{ selectedLabel }}</span>
    <div class="management-batch-bar__actions">
      <t-space class="management-batch-bar__desktop-actions">
        <t-button
          v-if="props.selectCurrentPageLabel"
          :data-testid="props.selectCurrentPageTestId"
          :size="props.buttonSize"
          theme="default"
          variant="outline"
          @click="emit('select-current-page')"
        >
          {{ props.selectCurrentPageLabel }}
        </t-button>
        <t-button
          v-if="props.invertCurrentPageLabel"
          :data-testid="props.invertCurrentPageTestId"
          :size="props.buttonSize"
          theme="default"
          variant="outline"
          @click="emit('invert-current-page')"
        >
          {{ props.invertCurrentPageLabel }}
        </t-button>
        <slot />
      </t-space>
      <t-space v-if="compactMenuOptions.length" class="management-batch-bar__compact-actions">
        <t-dropdown :options="compactMenuOptions" trigger="click" @click="handleCompactAction">
          <t-button :data-testid="props.compactActionTestId" :size="props.buttonSize" theme="primary" variant="outline">
            {{ props.compactActionLabel }}
          </t-button>
        </t-dropdown>
      </t-space>
      <t-button
        :data-testid="clearTestId"
        :size="props.buttonSize"
        theme="default"
        variant="text"
        @click="emit('clear')"
      >
        {{ clearLabel }}
      </t-button>
    </div>
  </div>
</template>
<script setup lang="ts">
import type { DropdownProps } from 'tdesign-vue-next';
import { computed } from 'vue';

type ManagementBatchAction = NonNullable<DropdownProps['options']>[number];

// 页面持有选择状态；组件只统一批量操作布局，并通过 clear / action 事件交还业务处理责任。
const props = withDefaults(
  defineProps<{
    clearLabel: string;
    clearTestId?: string;
    compactActionLabel?: string;
    compactActionTestId?: string;
    compactActions?: ManagementBatchAction[];
    buttonSize?: 'small' | 'medium' | 'large';
    invertCurrentPageLabel?: string;
    invertCurrentPageTestId?: string;
    selectCurrentPageLabel?: string;
    selectCurrentPageTestId?: string;
    selectedLabel: string;
  }>(),
  {
    compactActionLabel: '',
    compactActionTestId: 'management-batch-actions',
    compactActions: () => [],
    clearTestId: '',
    buttonSize: 'small',
    invertCurrentPageLabel: '',
    invertCurrentPageTestId: 'management-batch-invert-current-page',
    selectCurrentPageLabel: '',
    selectCurrentPageTestId: 'management-batch-select-current-page',
  },
);

const emit = defineEmits<{
  action: [value: string];
  clear: [];
  'invert-current-page': [];
  'select-current-page': [];
}>();

const compactMenuOptions = computed<ManagementBatchAction[]>(() => [
  ...(props.selectCurrentPageLabel ? [{ content: props.selectCurrentPageLabel, value: 'select-current-page' }] : []),
  ...(props.invertCurrentPageLabel ? [{ content: props.invertCurrentPageLabel, value: 'invert-current-page' }] : []),
  ...props.compactActions,
]);

const handleCompactAction: NonNullable<DropdownProps['onClick']> = (action) => {
  const value = typeof action === 'object' && action ? action.value : action;
  if (value === 'select-current-page') emit('select-current-page');
  else if (value === 'invert-current-page') emit('invert-current-page');
  else if (typeof value === 'string') emit('action', value);
};
</script>
<style scoped lang="less">
.management-batch-bar,
.management-batch-bar__actions {
  align-items: center;
  display: flex;
  min-width: 0;
}

.management-batch-bar {
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  width: 100%;
}

.management-batch-bar__summary {
  color: var(--td-text-color-primary);
  flex: 1 1 auto;
  min-width: 0;
}

.management-batch-bar__actions {
  flex: 0 0 auto;
  justify-content: flex-end;
}

.management-batch-bar__compact-actions {
  display: none;
}

@media (width <= 768px) {
  .management-batch-bar {
    align-items: center;
  }

  .management-batch-bar__desktop-actions {
    flex-wrap: wrap;
  }

  .management-batch-bar__compact-actions {
    display: inline-flex;
  }

  .management-batch-bar--has-compact-actions .management-batch-bar__desktop-actions {
    display: none;
  }
}
</style>
