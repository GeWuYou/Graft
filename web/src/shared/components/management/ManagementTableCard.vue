<template>
  <section class="management-table-card">
    <div v-if="$slots.head" class="management-table-card__head">
      <slot name="head" />
    </div>
    <div v-if="$slots.batch" class="management-table-card__batch">
      <slot name="batch" />
    </div>
    <div class="management-table-card__body">
      <slot />
    </div>
    <div v-if="$slots.footer" class="management-table-card__footer">
      <slot name="footer" />
    </div>
  </section>
</template>
<style scoped lang="less">
.management-table-card {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  box-shadow: var(--td-shadow-1);
  display: flex;
  flex-direction: column;
  gap: 0;
  overflow: hidden;
  padding: 0;
}

.management-table-card__head,
.management-table-card__batch,
.management-table-card__footer {
  align-items: center;
  border-bottom: 1px solid var(--td-component-stroke);
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: space-between;
  padding: 16px 20px;
}

.management-table-card__body {
  min-width: 0;
  padding: 0 20px 16px;
}

.management-table-card__footer {
  border-bottom: 0;
  border-top: 1px solid var(--td-component-stroke);
}

.management-table-card__body :deep(.t-table) {
  --td-comp-paddingTB-m: 11px;

  width: 100%;
}

.management-table-card__body :deep(.t-table th) {
  height: 44px;
}

.management-table-card__body :deep(.t-table td) {
  min-height: 56px;
}

@media (width <= 768px) {
  .management-table-card__head,
  .management-table-card__batch,
  .management-table-card__footer {
    align-items: stretch;
    flex-direction: column;
    padding: 16px;
  }

  .management-table-card__body {
    padding: 0 16px 16px;
  }
}
</style>
