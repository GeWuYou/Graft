<template>
  <section ref="container" :class="['responsive-card-list', `responsive-card-list--${variant.density}`]">
    <slot :variant="variant" />
  </section>
</template>
<script setup lang="ts">
import { ref } from 'vue';

import { useResponsiveVariant } from '@/shared/composables';

/** CardList 是 entity 表格的 shared 呈现槽位，不能承载独立页面状态或数据请求。 */
const container = ref<HTMLElement | null>(null);
const variant = useResponsiveVariant(container, { presentation: 'entity' });
</script>
<style scoped lang="less">
.responsive-card-list {
  container-type: inline-size;
  display: grid;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}
</style>
