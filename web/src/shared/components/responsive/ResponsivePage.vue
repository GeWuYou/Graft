<template>
  <main
    ref="container"
    :class="['responsive-page', `responsive-page--${variant.density}`, `responsive-page--${layout}`]"
    :data-responsive-density="variant.density"
    :data-responsive-layout="layout"
  >
    <slot :variant="variant" />
  </main>
</template>
<script setup lang="ts">
import { ref } from 'vue';

import { useResponsiveVariant } from '@/shared/composables';
import type { ResponsiveLayout } from '@/shared/responsive';

/** shared 页面外壳只收敛容器布局，业务页面继续拥有数据与操作语义。 */
const { layout = 'flow' } = defineProps<{ layout?: ResponsiveLayout }>();
const container = ref<HTMLElement | null>(null);
const variant = useResponsiveVariant(container, { layout });
</script>
<style scoped lang="less">
.responsive-page {
  box-sizing: border-box;
  container-type: inline-size;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-width: 0;
  padding: var(--graft-responsive-content-gutter);
  width: 100%;
}

.responsive-page--compact {
  gap: var(--graft-density-gap-12);
}

.responsive-page--stack {
  align-items: stretch;
}
</style>
