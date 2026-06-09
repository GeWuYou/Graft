<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <section :class="containerClasses">
    <main :class="mainClasses">
      <slot />
    </main>
    <t-footer v-if="showFooter" :class="`${prefix}-footer-layout`">
      <l-footer :content="footerText ?? ''" />
    </t-footer>
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import { prefix } from '@/config/global';

import LFooter from './Footer.vue';

const props = defineProps<{
  showFooter?: boolean;
  footerText?: string;
  surface?: 'shell' | 'overview-dashboard' | 'list-form-detail';
}>();

const containerClasses = computed(() => [
  'graft-page-container',
  `${prefix}-page-container`,
  `${prefix}-page-container--${props.surface ?? 'shell'}`,
]);

const mainClasses = computed(() => [
  'graft-page',
  `${prefix}-page-container__main`,
  `${prefix}-page-container__main--${props.surface ?? 'shell'}`,
]);
</script>
<style scoped lang="less">
@prefix-cls: ~'@{starter-prefix}-page-container';

.@{prefix-cls} {
  padding-bottom: var(--graft-page-bottom-safe-area);
}
</style>
