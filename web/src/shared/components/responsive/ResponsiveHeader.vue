<template>
  <header
    ref="container"
    :class="['responsive-header', `responsive-header--${variant.density}`]"
    :data-responsive-density="variant.density"
  >
    <div class="responsive-header__copy">
      <div v-if="$slots.eyebrow" class="responsive-header__eyebrow"><slot name="eyebrow" /></div>
      <div class="responsive-header__title"><slot name="title" /></div>
      <div v-if="$slots.description" class="responsive-header__description"><slot name="description" /></div>
    </div>
    <div v-if="$slots.actions || $slots.extra" class="responsive-header__actions">
      <div v-if="$slots.extra" class="responsive-header__extra"><slot name="extra" /></div>
      <div v-if="$slots.actions" class="responsive-header__primary"><slot name="actions" /></div>
    </div>
  </header>
</template>
<script setup lang="ts">
import { ref } from 'vue';

import { useResponsiveVariant } from '@/shared/composables';

/** Header 只编排标题与操作 slots，不承担页面标题的 i18n 或业务来源。 */
const container = ref<HTMLElement | null>(null);
const variant = useResponsiveVariant(container);
</script>
<style scoped lang="less">
.responsive-header {
  align-items: flex-start;
  container-type: inline-size;
  display: flex;
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
  min-width: 0;
}

.responsive-header__copy,
.responsive-header__actions,
.responsive-header__extra,
.responsive-header__primary {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.responsive-header__copy {
  flex: 1 1 auto;
  flex-direction: column;
}

.responsive-header__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
  overflow-wrap: anywhere;
}

.responsive-header__eyebrow,
.responsive-header__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
  overflow-wrap: anywhere;
}

.responsive-header__actions {
  align-items: flex-end;
  flex: 0 1 auto;
  flex-direction: column;
}

.responsive-header__primary {
  justify-content: flex-end;
}

@container (width < 48rem) {
  .responsive-header,
  .responsive-header__actions {
    align-items: stretch;
    flex-direction: column;
  }

  .responsive-header__actions,
  .responsive-header__extra,
  .responsive-header__primary {
    width: 100%;
  }

  .responsive-header__primary {
    justify-content: flex-start;
  }
}
</style>
