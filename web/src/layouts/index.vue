<template>
  <div class="app-shell" v-bind="shellSurfaceAttrs">
    <template v-if="setting.layout.value === 'side'">
      <t-layout key="side" :class="['app-shell__layout', mainLayoutCls]">
        <t-aside><layout-side-nav /></t-aside>
        <t-layout class="app-shell__main">
          <t-header class="app-shell__header"><layout-header /></t-header>
          <t-content class="app-shell__content"><layout-content /></t-content>
        </t-layout>
      </t-layout>
    </template>

    <template v-else>
      <t-layout key="no-side" class="app-shell__layout">
        <t-header class="app-shell__header"><layout-header /> </t-header>
        <t-layout :class="['app-shell__main', mainLayoutCls]">
          <layout-side-nav />
          <layout-content />
        </t-layout>
      </t-layout>
    </template>
  </div>
  <force-password-change-dialog />
</template>
<script setup lang="ts">
import '@/style/layout.less';

import { storeToRefs } from 'pinia';
import { computed, onMounted, watch } from 'vue';
import { useRoute } from 'vue-router';

import { prefix } from '@/config/global';
import { LOCALE } from '@/contracts/i18n/locales';
import { useSettingStore, useTabsRouterStore } from '@/store';
import type { AppRouteMeta } from '@/utils/types';

import ForcePasswordChangeDialog from './components/ForcePasswordChangeDialog.vue';
import LayoutContent from './components/LayoutContent.vue';
import LayoutHeader from './components/LayoutHeader.vue';
import LayoutSideNav from './components/LayoutSideNav.vue';

const route = useRoute();
const settingStore = useSettingStore();
const tabsRouterStore = useTabsRouterStore();
const setting = storeToRefs(settingStore);

const shellSurfaceAttrs = computed(() => ({
  'data-layout-mode': settingStore.layout,
  'data-page-type': 'shell',
  'data-theme-mode': settingStore.displayMode,
}));

const mainLayoutCls = computed(() => [
  {
    't-layout--with-sider': settingStore.showSidebar,
  },
]);

const appendNewRoute = () => {
  const {
    path,
    query,
    meta: { hidden, title },
    name,
  } = route;

  if (hidden) {
    return;
  }

  const titleObj = typeof title === 'string' ? { [LOCALE.ZH_CN]: title, [LOCALE.EN_US]: title } : title;
  tabsRouterStore.appendTabRouterList({
    path,
    query,
    title: titleObj,
    name,
    isAlive: true,
    meta: route.meta as AppRouteMeta,
  });
};

onMounted(() => {
  appendNewRoute();
});

watch(
  () => route.path,
  () => {
    appendNewRoute();
    document.querySelector(`.${prefix}-layout`)?.scrollTo({ top: 0, behavior: 'smooth' });
  },
);
</script>
<style lang="less" scoped>
.app-shell {
  background: var(--td-bg-color-page);
  color: var(--td-text-color-primary);
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.app-shell__layout,
.app-shell__main {
  background: transparent;
  flex: 1;
  min-height: 0;
}

.app-shell__content {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}

.app-shell :deep(.t-layout),
.app-shell :deep(.t-layout__content) {
  min-height: 0;
}
</style>
