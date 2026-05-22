<template>
  <section :class="`${prefix}-page-container`">
    <main :class="`${prefix}-page-container__main`">
      <slot />
    </main>
    <div :class="`${prefix}-page-container__bottom-safe-spacer`" aria-hidden="true"></div>
  </section>
</template>
<script setup lang="ts">
import { prefix } from '@/config/global';

defineProps<{
  showFooter?: boolean;
  footerText?: string;
}>();
</script>
