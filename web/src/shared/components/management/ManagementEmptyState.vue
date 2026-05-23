<template>
  <div :class="['management-empty-state', toneClass]">
    <h3 class="management-empty-state__title">{{ title }}</h3>
    <p class="management-empty-state__description">{{ description }}</p>
    <div v-if="$slots.actions" class="management-empty-state__actions">
      <slot name="actions" />
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue';

const props = withDefaults(
  defineProps<{
    title: string;
    description: string;
    tone?: 'default' | 'error';
  }>(),
  {
    tone: 'default',
  },
);

const toneClass = computed(() =>
  props.tone === 'error' ? 'management-empty-state--error' : 'management-empty-state--default',
);
</script>
<style scoped lang="less">
.management-empty-state {
  align-items: flex-start;
  border: 1px dashed var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 28px 24px;
}

.management-empty-state--default {
  background: var(--td-bg-color-secondarycontainer);
}

.management-empty-state--error {
  background: color-mix(in srgb, var(--td-error-color-5) 6%, var(--td-bg-color-container));
  border-color: color-mix(in srgb, var(--td-error-color-5) 26%, var(--td-component-stroke));
}

.management-empty-state__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0;
}

.management-empty-state__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
  margin: 0;
}

.management-empty-state__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}
</style>
