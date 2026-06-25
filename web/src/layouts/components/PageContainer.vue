<template>
  <section :class="containerClasses">
    <main :class="mainClasses">
      <div :class="`${prefix}-page-container__content`">
        <breadcrumb />
        <slot />
      </div>
    </main>
    <t-footer v-if="showFooter" :class="`${prefix}-footer-layout`">
      <l-footer :content="footerText ?? ''" />
    </t-footer>
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import { prefix } from '@/config/global';
import type { PageSurfaceType } from '@/utils/route/meta';

import Breadcrumb from './Breadcrumb.vue';
import LFooter from './Footer.vue';

const props = defineProps<{
  showFooter?: boolean;
  footerText?: string;
  surface?: PageSurfaceType;
}>();

const containerClasses = computed(() => [
  'graft-page-container',
  'page-scroll',
  'graft-scrollbar',
  `${prefix}-page-container`,
  `${prefix}-page-scroll`,
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
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
  overflow: hidden auto;
  padding-bottom: var(--graft-page-bottom-safe-area);
}

.@{prefix-cls}__main,
.@{prefix-cls}__content {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.@{prefix-cls}__main {
  flex: 1 0 auto;
  min-height: 0;
}

.@{prefix-cls}__content {
  flex: 1 0 auto;
  min-height: 100%;
}
</style>
