<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <section class="management-toolbar">
    <div class="management-toolbar__filters">
      <slot name="filters" />
    </div>
    <div v-if="$slots.actions" class="management-toolbar__actions">
      <slot name="actions" />
    </div>
  </section>
</template>
<style scoped lang="less">
.management-toolbar,
.management-toolbar__filters,
.management-toolbar__actions {
  --graft-list-search-width: clamp(240px, 28vw, 320px);
  --graft-list-select-width: clamp(160px, 18vw, 220px);
  --graft-query-search-width: clamp(360px, 36vw, 480px);

  display: flex;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.management-toolbar {
  align-items: center;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  box-shadow: var(--td-shadow-1);
  flex-wrap: wrap;
  justify-content: space-between;
  min-height: 72px;
  padding: var(--graft-density-gap-14) var(--graft-density-gap-20);
}

.management-toolbar__filters,
.management-toolbar__actions {
  align-items: center;
  flex: 1 1 auto;
  flex-wrap: wrap;
  min-width: 0;
}

.management-toolbar__actions {
  flex: 0 0 auto;
  justify-content: flex-end;
}

.management-toolbar :deep(.management-list-search) {
  flex: 0 1 var(--graft-list-search-width);
  min-width: min(100%, 220px);
  width: var(--graft-list-search-width);
}

.management-toolbar :deep(.management-query-search) {
  flex: 0 1 var(--graft-query-search-width);
  min-width: min(100%, 280px);
  width: var(--graft-query-search-width);
}

.management-toolbar :deep(.toolbar__select),
.management-toolbar :deep(.scheduled-task-toolbar__select) {
  flex: 0 1 var(--graft-list-select-width);
  min-width: min(100%, 150px);
  width: var(--graft-list-select-width);
}

@media (width <= 768px) {
  .management-toolbar {
    padding: var(--graft-density-gap-16);
  }

  .management-toolbar__filters,
  .management-toolbar__actions {
    justify-content: flex-start;
  }

  .management-toolbar :deep(.management-list-search),
  .management-toolbar :deep(.management-query-search),
  .management-toolbar :deep(.toolbar__select),
  .management-toolbar :deep(.scheduled-task-toolbar__select) {
    flex-basis: 100%;
    max-width: none;
    width: 100%;
  }
}
</style>
