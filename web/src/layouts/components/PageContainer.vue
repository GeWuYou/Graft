<template>
  <section :class="`${prefix}-page-container`">
    <main :class="`${prefix}-page-container__main`">
      <slot />
    </main>
    <t-footer v-if="showFooter" :class="`${prefix}-footer-layout`">
      <l-footer :content="footerText" />
    </t-footer>
  </section>
</template>
<script setup lang="ts">
import { prefix } from '@/config/global';

import LFooter from './Footer.vue';

withDefaults(
  defineProps<{
    showFooter?: boolean;
    footerText?: string;
  }>(),
  {
    showFooter: false,
    footerText: '',
  },
);
</script>
