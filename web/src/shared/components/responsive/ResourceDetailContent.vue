<template>
  <section class="resource-detail-content">
    <header class="resource-detail-content__header">
      <t-button shape="square" variant="text" :aria-label="backLabel" @click="emit('back')">
        <template #icon><chevron-left-icon /></template>
      </t-button>
      <h1>{{ title }}</h1>
      <div v-if="$slots.actions" class="resource-detail-content__actions"><slot name="actions" /></div>
    </header>
    <div class="resource-detail-content__scroll graft-scrollbar">
      <div class="resource-detail-content__body"><slot /></div>
    </div>
    <footer v-if="$slots.footer" class="resource-detail-content__footer"><slot name="footer" /></footer>
  </section>
</template>
<script setup lang="ts">
import { ChevronLeftIcon } from 'tdesign-icons-vue-next';

/** 详情内容骨架统一拥有固定标题栏和独立滚动区域，业务页只组合资源语义内容。 */
defineProps<{ backLabel: string; title: string }>();
const emit = defineEmits<{ back: [] }>();
</script>
<style scoped lang="less">
.resource-detail-content {
  block-size: 100%;
  display: flex;
  flex-direction: column;
  min-block-size: 0;
  min-inline-size: 0;
}

.resource-detail-content__header {
  align-items: flex-start;
  border-bottom: 1px solid var(--td-component-stroke);
  display: flex;
  flex: 0 0 auto;
  gap: var(--graft-density-gap-8);
  padding: var(--td-comp-paddingTB-m) var(--td-comp-paddingLR-l);
}

.resource-detail-content__header h1 {
  color: var(--td-text-color-primary);
  flex: 1;
  font-size: var(--td-font-size-title-medium);
  line-height: var(--td-line-height-title-medium);
  margin: 0;
  min-inline-size: 0;
  overflow-wrap: anywhere;
}

.resource-detail-content__actions {
  flex: 0 0 auto;
}

.resource-detail-content__scroll {
  flex: 1;
  min-block-size: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.resource-detail-content__body {
  display: grid;
  gap: var(--td-comp-margin-xl);
  min-inline-size: 0;
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
}

.resource-detail-content__footer {
  border-top: 1px solid var(--td-component-stroke);
  display: grid;
  flex: 0 0 auto;
  gap: var(--graft-density-gap-12);
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
}

@media (width < 768px) {
  .resource-detail-content__footer {
    padding-bottom: max(var(--td-comp-paddingTB-l), env(safe-area-inset-bottom));
  }

  .resource-detail-content__footer :deep(.t-button) {
    inline-size: 100%;
  }
}
</style>
