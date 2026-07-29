<template>
  <div class="management-table-pagination">
    <div class="management-table-pagination__summary">
      <slot name="summary">
        <span>{{ summary }}</span>
      </slot>
    </div>
    <div class="management-table-pagination__controls">
      <slot />
    </div>
  </div>
</template>
<script setup lang="ts">
// 管理表格分页只承载摘要与分页控件的布局；数据统计和分页状态仍由业务页面拥有，避免共享壳层反向持有查询状态。
defineProps<{
  summary: string;
}>();
</script>
<style scoped lang="less">
.management-table-pagination {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-20);
  justify-content: space-between;
  min-height: 60px;
  width: 100%;
}

.management-table-pagination__summary {
  color: var(--td-text-color-secondary);
  flex: 0 0 auto;
  font: var(--td-font-body-small);
  min-width: 0;
  white-space: nowrap;
}

.management-table-pagination__controls {
  align-items: center;
  display: flex;
  flex: 1 1 auto;
  justify-content: flex-end;
  min-width: 0;
}

.management-table-pagination__controls :deep(.t-pagination) {
  align-items: center;
  display: flex;
  flex-wrap: nowrap;
  gap: var(--graft-density-gap-14);
  justify-content: flex-end;
  margin-left: auto;
  min-width: 0;
  width: 100%;
}

// 这些选择器依赖 tdesign-vue-next 1.20.5 的 Pagination DOM，用于保持分页各段在窄屏下的布局约束。
.management-table-pagination__controls :deep(.t-pagination__select),
.management-table-pagination__controls :deep(.t-pagination__pager),
.management-table-pagination__controls :deep(.t-pagination__jump),
.management-table-pagination__controls :deep(.t-pagination__btn),
.management-table-pagination__controls :deep(.t-select) {
  align-items: center;
  display: inline-flex;
  margin: 0;
  white-space: nowrap;
}

.management-table-pagination__controls :deep(.t-pagination__pager) {
  align-items: center;
  display: inline-flex;
}

@media (width <= 768px) {
  .management-table-pagination {
    align-items: stretch;
    flex-direction: column;
  }

  .management-table-pagination__summary,
  .management-table-pagination__controls {
    white-space: normal;
    width: 100%;
  }

  .management-table-pagination__controls :deep(.t-pagination) {
    flex-wrap: wrap;
    justify-content: flex-start;
  }
}
</style>
