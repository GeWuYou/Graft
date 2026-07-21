<template>
  <form
    ref="container"
    :class="['responsive-form', `responsive-form--${variant.density}`]"
    @submit="emit('submit', $event)"
  >
    <div class="responsive-form__fields"><slot :variant="variant" /></div>
    <footer v-if="$slots.actions" class="responsive-form__actions"><slot name="actions" :variant="variant" /></footer>
  </form>
</template>
<script setup lang="ts">
import { ref } from 'vue';

import { useResponsiveVariant } from '@/shared/composables';

/** Form 只管理字段栅格和动作区；校验、提交与字段模型仍由业务表单拥有。 */
const emit = defineEmits<{ (event: 'submit', value: SubmitEvent): void }>();
const container = ref<HTMLElement | null>(null);
const variant = useResponsiveVariant(container);
</script>
<style scoped lang="less">
.responsive-form {
  container-type: inline-size;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-20);
  min-width: 0;
}

.responsive-form__fields {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  min-width: 0;
}

.responsive-form__actions {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  min-width: 0;
  padding-bottom: env(safe-area-inset-bottom, 0);
}

@container (width < 48rem) {
  .responsive-form__fields {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
