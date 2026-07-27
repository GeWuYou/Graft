<template>
  <section
    ref="container"
    :class="[
      'responsive-table',
      `responsive-table--${presentation}`,
      `responsive-table--${entityCardLayout}`,
      `responsive-table--${variant.density}`,
    ]"
    :data-responsive-density="variant.density"
    :data-responsive-entity-card-layout="entityCardLayout"
    :data-responsive-presentation="presentation"
  >
    <div v-if="$slots.cards && (showCards || preserveInactive)" v-show="showCards" class="responsive-table__cards">
      <slot name="cards" :variant="variant" />
    </div>
    <div v-if="!showCards || preserveInactive" v-show="!showCards" class="responsive-table__scroll graft-scrollbar">
      <slot :variant="variant" />
    </div>
  </section>
</template>
<script lang="ts">
export type ResponsiveEntityCardLayout = 'compact' | 'adaptive';
</script>
<script setup lang="ts">
import { computed, ref, useSlots } from 'vue';

import { useResponsiveVariant, useViewportResponsiveVariant } from '@/shared/composables';
import type { ResponsivePresentation } from '@/shared/responsive';

/** Table 只按数据/实体展示语义选择横向滚动或卡片槽位，不解释列与业务操作。 */
const {
  densityScope = 'container',
  entityCardLayout = 'compact',
  preserveInactive = false,
  presentation = 'data',
} = defineProps<{
  densityScope?: 'container' | 'viewport';
  entityCardLayout?: ResponsiveEntityCardLayout;
  preserveInactive?: boolean;
  presentation?: ResponsivePresentation;
}>();
const container = ref<HTMLElement | null>(null);
const containerVariant = useResponsiveVariant(container, { presentation });
const viewportVariant = useViewportResponsiveVariant({ presentation });
const variant = computed(() => (densityScope === 'viewport' ? viewportVariant.value : containerVariant.value));
const slots = useSlots();
const showCards = computed(
  () =>
    presentation === 'entity' &&
    Boolean(slots.cards) &&
    (entityCardLayout === 'adaptive' ? variant.value.density !== 'spacious' : variant.value.density === 'compact'),
);
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

@container (width >= 768px) {
  .responsive-table--adaptive .responsive-table__cards {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
