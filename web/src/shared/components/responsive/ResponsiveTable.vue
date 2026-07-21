<template>
  <section
    ref="container"
    :class="['responsive-table', `responsive-table--${presentation}`, `responsive-table--${variant.density}`]"
    :data-responsive-density="variant.density"
    :data-responsive-presentation="presentation"
  >
    <div
      v-if="presentation === 'entity' && variant.density === 'compact' && $slots.cards"
      class="responsive-table__cards"
    >
      <slot name="cards" :variant="variant" />
    </div>
    <div v-else class="responsive-table__scroll graft-scrollbar"><slot :variant="variant" /></div>
  </section>
</template>
<script setup lang="ts">
import { ref } from 'vue';

import { useResponsiveVariant } from '@/shared/composables';
import type { ResponsivePresentation } from '@/shared/responsive';

/** Table 只按数据/实体展示语义选择横向滚动或卡片槽位，不解释列与业务操作。 */
const { presentation = 'data' } = defineProps<{ presentation?: ResponsivePresentation }>();
const container = ref<HTMLElement | null>(null);
const variant = useResponsiveVariant(container, { presentation });
</script>
<style scoped lang="less">
.responsive-table {
  container-type: inline-size;
  min-width: 0;
  width: 100%;
}

.responsive-table__scroll {
  max-width: 100%;
  min-width: 0;
  overflow-x: auto;
  overscroll-behavior-inline: contain;
}

.responsive-table__cards {
  display: grid;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}
</style>
