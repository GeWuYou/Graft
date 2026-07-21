<template>
  <section
    ref="container"
    :class="['responsive-empty', `responsive-empty--${tone}`, `responsive-empty--${variant.density}`]"
    :data-responsive-density="variant.density"
  >
    <div v-if="$slots.icon" class="responsive-empty__icon"><slot name="icon" /></div>
    <div class="responsive-empty__copy">
      <div v-if="$slots.title" class="responsive-empty__title"><slot name="title" /></div>
      <div v-if="$slots.description" class="responsive-empty__description"><slot name="description" /></div>
    </div>
    <div v-if="$slots.actions" class="responsive-empty__actions"><slot name="actions" /></div>
  </section>
</template>
<script setup lang="ts">
import { ref } from 'vue';

import { useResponsiveVariant } from '@/shared/composables';

/** Empty 只提供通用反馈表面，加载、错误与重试的业务状态仍由调用方决定。 */
const { tone = 'default' } = defineProps<{ tone?: 'default' | 'error' | 'loading' }>();
const container = ref<HTMLElement | null>(null);
const variant = useResponsiveVariant(container);
</script>
<style scoped lang="less">
.responsive-empty {
  align-items: flex-start;
  border: 1px dashed var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  container-type: inline-size;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-width: 0;
  padding: var(--graft-density-gap-24);
}

.responsive-empty--default,
.responsive-empty--loading {
  background: var(--td-bg-color-secondarycontainer);
}

.responsive-empty--error {
  background: color-mix(in srgb, var(--td-error-color-5) 6%, var(--td-bg-color-container));
  border-color: color-mix(in srgb, var(--td-error-color-5) 26%, var(--td-component-stroke));
}

.responsive-empty__copy {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
}

.responsive-empty__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.responsive-empty__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
}

.responsive-empty__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
}
</style>
