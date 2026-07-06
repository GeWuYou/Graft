<template>
  <span
    class="brand-identity"
    :class="{
      'brand-identity--compact': compact,
      'brand-identity--label-hidden': labelHidden,
    }"
  >
    <span class="brand-identity__mark" :class="{ 'brand-identity__mark--compact': compact }">
      <brand-mark />
    </span>
    <span class="brand-identity__label" :class="{ 'brand-identity__label--hidden': labelHidden }">{{ label }}</span>
  </span>
</template>
<script setup lang="ts">
import BrandMark from './BrandMark.vue';

defineOptions({
  name: 'BrandIdentity',
});

withDefaults(
  defineProps<{
    compact?: boolean;
    label: string;
    labelHidden?: boolean;
  }>(),
  {
    compact: false,
    labelHidden: false,
  },
);
</script>
<style lang="less" scoped>
.brand-identity {
  align-items: center;
  color: var(--td-text-color-primary);
  display: inline-flex;
  gap: var(--graft-density-gap-12);
  min-width: 0;
  overflow: hidden;
  transition: gap 0.18s cubic-bezier(0.38, 0, 0.24, 1);
}

.brand-identity--compact {
  gap: 0;
  justify-content: center;
}

.brand-identity--label-hidden {
  gap: 0;
}

.brand-identity__mark {
  color: var(--td-brand-color);
  display: inline-flex;
  flex: 0 0 auto;
  height: 32px;
  width: 32px;
}

.brand-identity__mark--compact {
  height: 28px;
  width: 28px;
}

.brand-identity__label {
  color: inherit;
  font: var(--td-font-title-medium);
  font-style: italic;
  font-weight: 700;
  letter-spacing: 0.01em;
  line-height: 1;
  max-width: 112px;
  opacity: 1;
  overflow: hidden;
  transform: translateX(0);
  transform-origin: left center;
  transition:
    max-width 0.16s cubic-bezier(0.38, 0, 0.24, 1),
    opacity 0.1s cubic-bezier(0.38, 0, 0.24, 1),
    transform 0.16s cubic-bezier(0.38, 0, 0.24, 1);
  white-space: nowrap;
}

.brand-identity__label--hidden {
  max-width: 0;
  opacity: 0;
  transform: translateX(-10px);
}
</style>
