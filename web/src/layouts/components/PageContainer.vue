<template>
  <section :class="`${prefix}-page-container`">
    <main :class="`${prefix}-page-container__main`">
      <slot />
    </main>
    <t-footer v-if="showFooter" :class="`${prefix}-footer-layout`">
      <l-footer :content="footerText" />
    </t-footer>
    <div :class="`${prefix}-page-container__bottom-safe-spacer`" aria-hidden="true"></div>
  </section>
</template>
<script setup lang="ts">
import { prefix } from '@/config/global';

import LFooter from './Footer.vue';

defineProps<{
  showFooter: boolean;
  footerText: string;
}>();
</script>
