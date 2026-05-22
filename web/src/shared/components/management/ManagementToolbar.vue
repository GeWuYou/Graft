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
  display: flex;
  gap: 12px;
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
  padding: 14px 20px;
}

.management-toolbar__filters,
.management-toolbar__actions {
  align-items: center;
  flex: 1 1 auto;
  flex-wrap: wrap;
}

.management-toolbar__actions {
  flex: 0 0 auto;
  justify-content: flex-end;
}

@media (width <= 768px) {
  .management-toolbar {
    padding: 16px;
  }

  .management-toolbar__filters,
  .management-toolbar__actions {
    justify-content: flex-start;
  }
}
</style>
