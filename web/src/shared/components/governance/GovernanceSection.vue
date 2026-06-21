<template>
  <section class="governance-section" :data-section-kind="kind" :style="sectionStyle">
    <header v-if="title || description || $slots.actions" class="governance-section__header">
      <div class="governance-section__copy">
        <h2 v-if="title" class="governance-section__title">{{ title }}</h2>
        <p v-if="description" class="governance-section__description">{{ description }}</p>
      </div>
      <div v-if="$slots.actions" class="governance-section__actions">
        <slot name="actions" />
      </div>
    </header>

    <div class="governance-section__body">
      <slot />
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue';

const props = withDefaults(
  defineProps<{
    title?: string;
    description?: string;
    minHeight?: number | string;
    kind?: 'trend' | 'investigation' | 'status' | 'workflow' | 'navigation' | 'default';
  }>(),
  {
    title: '',
    description: '',
    minHeight: undefined,
    kind: 'default',
  },
);

const sectionStyle = computed(() => {
  if (props.minHeight === undefined) {
    return undefined;
  }

  return {
    minHeight: typeof props.minHeight === 'number' ? `${props.minHeight}px` : props.minHeight,
  };
});
</script>
<style scoped lang="less">
@import './card-surface.less';

.governance-section {
  .governance-card-surface();

  display: flex;
  flex-direction: column;
  padding: var(--graft-density-gap-18);
}

.governance-section__header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
  margin-bottom: var(--graft-density-gap-14);
}

.governance-section__copy {
  min-width: 0;
}

.governance-section__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
  margin: 0;
}

.governance-section__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: var(--graft-density-gap-4) 0 0;
}

.governance-section__actions {
  flex: 0 0 auto;
}

.governance-section__body {
  flex: 1;
  min-height: 0;
}
</style>
