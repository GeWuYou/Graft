<template>
  <aside ref="container" :class="['responsive-sidebar', `responsive-sidebar--${variant.density}`]">
    <slot :variant="variant" />
  </aside>
</template>
<script setup lang="ts">
import { ref } from 'vue';

import { useResponsiveVariant } from '@/shared/composables';

/** Sidebar 仅提供壳层可用宽度语义，导航数据、展开态和路由行为仍属于 layouts。 */
const container = ref<HTMLElement | null>(null);
const variant = useResponsiveVariant(container, { surface: 'drawer' });
</script>
<style scoped lang="less">
.responsive-sidebar {
  container-type: inline-size;
  min-width: 0;
}
</style>
