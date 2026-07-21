<template>
  <section
    ref="container"
    :class="['responsive-toolbar', `responsive-toolbar--${variant.density}`]"
    :data-responsive-density="variant.density"
  >
    <div v-if="$slots.filters" class="responsive-toolbar__filters"><slot name="filters" /></div>
    <div v-if="$slots.batch" class="responsive-toolbar__batch"><slot name="batch" /></div>
    <div v-if="$slots.primary || $slots.secondary || $slots.overflow" class="responsive-toolbar__actions">
      <div v-if="$slots.primary" class="responsive-toolbar__primary"><slot name="primary" /></div>
      <div v-if="$slots.secondary" class="responsive-toolbar__secondary"><slot name="secondary" /></div>
      <div v-if="$slots.overflow" class="responsive-toolbar__overflow"><slot name="overflow" /></div>
    </div>
  </section>
</template>
<script setup lang="ts">
import { ref } from 'vue';

import { useResponsiveVariant } from '@/shared/composables';

/** Toolbar 保留业务操作的所有权，只为各类操作 slots 提供容器内重排。 */
const container = ref<HTMLElement | null>(null);
const variant = useResponsiveVariant(container);
</script>
<style scoped lang="less">
.responsive-toolbar,
.responsive-toolbar__filters,
.responsive-toolbar__batch,
.responsive-toolbar__actions,
.responsive-toolbar__primary,
.responsive-toolbar__secondary,
.responsive-toolbar__overflow {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.responsive-toolbar {
  container-type: inline-size;
  justify-content: space-between;
  padding: var(--graft-density-gap-14) 0;
}

.responsive-toolbar__filters {
  flex: 1 1 20rem;
}

.responsive-toolbar__batch {
  flex: 1 1 100%;
}

.responsive-toolbar__actions {
  flex: 0 1 auto;
  justify-content: flex-end;
}

@container (width < 48rem) {
  .responsive-toolbar,
  .responsive-toolbar__actions {
    align-items: stretch;
    flex-direction: column;
  }

  .responsive-toolbar__filters,
  .responsive-toolbar__batch,
  .responsive-toolbar__actions,
  .responsive-toolbar__primary,
  .responsive-toolbar__secondary,
  .responsive-toolbar__overflow {
    width: 100%;
  }

  .responsive-toolbar__primary,
  .responsive-toolbar__secondary,
  .responsive-toolbar__overflow {
    justify-content: flex-start;
  }
}
</style>
