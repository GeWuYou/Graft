<template>
  <div
    class="management-batch-bar"
    :class="{ 'management-batch-bar--has-compact-actions': props.compactActions.length }"
  >
    <span class="management-batch-bar__summary">{{ selectedLabel }}</span>
    <div class="management-batch-bar__actions">
      <t-space class="management-batch-bar__desktop-actions">
        <slot />
      </t-space>
      <t-space v-if="props.compactActions.length" class="management-batch-bar__compact-actions">
        <t-dropdown :options="props.compactActions" trigger="click" @click="handleCompactAction">
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
    selectedLabel: string;
  }>(),
  {
    compactActionLabel: '',
    compactActionTestId: 'management-batch-actions',
    compactActions: () => [],
    clearTestId: '',
    buttonSize: 'small',
  },
);

const emit = defineEmits<{
  action: [value: string];
  clear: [];
}>();

function handleCompactAction(action: ManagementBatchAction) {
  const { value } = action;
  if (typeof value === 'string') emit('action', value);
}
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
