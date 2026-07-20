<template>
  <div class="management-batch-bar">
    <span class="management-batch-bar__summary">{{ selectedLabel }}</span>
    <div class="management-batch-bar__actions">
      <t-space>
        <slot />
        <t-button :data-testid="clearTestId" size="small" theme="default" variant="text" @click="emit('clear')">
          {{ clearLabel }}
        </t-button>
      </t-space>
    </div>
  </div>
</template>
<script setup lang="ts">
// 页面持有选择状态；组件只统一批量操作布局，并通过 clear 事件交还清除选择的责任。
defineProps<{
  clearLabel: string;
  clearTestId?: string;
  selectedLabel: string;
}>();

const emit = defineEmits<{
  clear: [];
}>();
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

@media (width <= 768px) {
  .management-batch-bar,
  .management-batch-bar__actions {
    align-items: stretch;
    flex-direction: column;
  }

  .management-batch-bar__actions,
  .management-batch-bar__actions :deep(.t-space) {
    width: 100%;
  }

  .management-batch-bar__actions :deep(.t-space) {
    flex-wrap: wrap;
  }
}
</style>
