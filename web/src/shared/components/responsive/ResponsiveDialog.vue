<template>
  <section
    ref="container"
    :class="[
      'responsive-dialog',
      `responsive-dialog--${policy.surface}`,
      `responsive-dialog--${size}`,
      `responsive-dialog--${purpose}`,
    ]"
    :data-responsive-density="policy.density"
    :data-responsive-interaction="policy.interaction"
    :data-responsive-surface="policy.surface"
    :style="{ '--graft-responsive-dialog-max': `var(${dialogSizeToken})` }"
  >
    <div v-if="$slots.header" class="responsive-dialog__header"><slot name="header" :policy="policy" /></div>
    <div class="responsive-dialog__body"><slot :policy="policy" /></div>
    <div v-if="$slots.footer" class="responsive-dialog__footer"><slot name="footer" :policy="policy" /></div>
  </section>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';

import { useContainerSize } from '@/shared/composables';
import { RESPONSIVE_STYLE_TOKENS, type ResponsiveStyleToken } from '@/shared/responsive';

import {
  resolveResponsiveDialogPolicy,
  type ResponsiveDialogPurpose,
  type ResponsiveDialogSize,
} from './dialog-policy';

/** Dialog facade 只解析可用表面，实际 TDesign overlay 的创建仍归后续接入层。 */
const { purpose = 'detail', size = 'medium' } = defineProps<{
  purpose?: ResponsiveDialogPurpose;
  size?: ResponsiveDialogSize;
}>();
const container = ref<HTMLElement | null>(null);
const containerSize = useContainerSize(container);
const policy = computed(() => resolveResponsiveDialogPolicy(containerSize.value.width, purpose, size));
const dialogSizeToken = computed<ResponsiveStyleToken>(() => {
  if (size === 'compact') {
    return RESPONSIVE_STYLE_TOKENS.dialogCompactMax;
  }

  if (size === 'large') {
    return RESPONSIVE_STYLE_TOKENS.dialogLargeMax;
  }

  return RESPONSIVE_STYLE_TOKENS.dialogMediumMax;
});
</script>
<style scoped lang="less">
.responsive-dialog {
  box-sizing: border-box;
  container-type: inline-size;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  max-inline-size: var(--graft-responsive-dialog-max);
  max-width: 100%;
  min-width: 0;
}

.responsive-dialog--sheet,
.responsive-dialog--fullscreen {
  inline-size: 100%;
  max-inline-size: none;
}

.responsive-dialog__footer {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
}
</style>
