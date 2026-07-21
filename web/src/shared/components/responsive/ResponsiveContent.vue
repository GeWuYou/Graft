<template>
  <section
    ref="container"
    :class="['responsive-content', `responsive-content--${layout}`, `responsive-content--${variant.density}`]"
    :data-responsive-density="variant.density"
    :data-responsive-layout="layout"
  >
    <slot :variant="variant" />
  </section>
</template>
<script setup lang="ts">
import { ref } from 'vue';

import { useResponsiveVariant } from '@/shared/composables';
import type { ResponsiveLayout } from '@/shared/responsive';

/** Content 将语义 layout 映射为容器栅格，不解释内容领域或字段关系。 */
const { layout = 'flow' } = defineProps<{ layout?: ResponsiveLayout }>();
const container = ref<HTMLElement | null>(null);
const variant = useResponsiveVariant(container, { layout });
</script>
<style scoped lang="less">
.responsive-content {
  container-type: inline-size;
  display: grid;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.responsive-content--stack {
  grid-template-columns: minmax(0, 1fr);
}

.responsive-content--split {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.responsive-content--grid {
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 18rem), 1fr));
}

.responsive-content--compact {
  gap: var(--graft-density-gap-12);
}

@container (width < 48rem) {
  .responsive-content--split {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
