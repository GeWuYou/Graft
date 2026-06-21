<template>
  <section class="management-table-card">
    <div v-if="hasHead" class="management-table-card__head">
      <div class="management-table-card__head-main">
        <slot name="head">
          <div v-if="title || description" class="management-table-card__summary">
            <p v-if="title" class="management-table-card__title">{{ title }}</p>
            <p v-if="description" class="management-table-card__description">{{ description }}</p>
          </div>
        </slot>
      </div>
      <div v-if="$slots.toolbar" class="management-table-card__toolbar">
        <slot name="toolbar" />
      </div>
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
<script setup lang="ts">
import { computed, useSlots } from 'vue';

const props = defineProps<{
  description?: string;
  title?: string;
}>();

const slots = useSlots();

const hasHead = computed(() => Boolean(slots.head || slots.toolbar || props.title || props.description));
</script>
<style scoped lang="less">
@import './card-surface.less';

.management-table-card {
  .management-card-surface();

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
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  padding: var(--graft-density-gap-16) var(--graft-density-gap-20);
  width: 100%;
}

.management-table-card__head-main,
.management-table-card__toolbar,
.management-table-card__batch,
.management-table-card__body,
.management-table-card__footer {
  min-width: 0;
}

.management-table-card__head-main {
  flex: 1 1 240px;
}

.management-table-card__toolbar {
  align-items: center;
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  justify-content: flex-end;
}

.management-table-card__summary {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.management-table-card__title,
.management-table-card__description {
  margin: 0;
}

.management-table-card__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.management-table-card__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.management-table-card__body {
  --td-comp-paddingTB-m: 11px;

  display: block;
  max-width: 100%;
  min-width: 0;
  overflow-x: hidden;
  padding: 0 var(--graft-density-gap-20) var(--graft-density-gap-16);
  width: 100%;
}

.management-table-card__footer {
  border-bottom: 0;
  border-top: 1px solid var(--td-component-stroke);
}

.management-table-card__body :deep(.t-table),
.management-table-card__body :deep(.t-table__content) {
  max-width: 100%;
  width: 100%;
}

.management-table-card__body :deep(.t-table__content) {
  min-width: 0;
}

.management-table-card__body :deep(.t-table__content table) {
  min-width: 100%;
  width: 100%;
}

@media (width <= 768px) {
  .management-table-card__head,
  .management-table-card__batch,
  .management-table-card__footer {
    align-items: stretch;
    flex-direction: column;
    padding: var(--graft-density-gap-16);
  }

  .management-table-card__toolbar {
    justify-content: flex-start;
  }

  .management-table-card__body {
    padding: 0 var(--graft-density-gap-16) var(--graft-density-gap-16);
  }
}
</style>
