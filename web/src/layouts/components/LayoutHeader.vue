<template>
  <l-header
    v-if="settingStore.showHeader"
    :show-logo="settingStore.showHeaderLogo"
    :theme="settingStore.displayMode"
    :layout="settingStore.layout"
    :is-fixed="settingStore.isHeaderFixed"
    :menu="headerMenu"
    :is-compact="settingStore.isSidebarCompact"
  />
</template>
<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { computed } from 'vue';

import { flattenMixHeaderMenus } from '@/layouts/layout-navigation';
import { usePermissionStore, useSettingStore } from '@/store';
import type { MenuRoute } from '@/utils/types';

import LHeader from './Header.vue';

const permissionStore = usePermissionStore();
const settingStore = useSettingStore();
const { routers: menuRouters } = storeToRefs(permissionStore);
const headerMenu = computed<MenuRoute[]>(() => {
  return settingStore.layout === 'mix'
    ? flattenMixHeaderMenus(menuRouters.value as MenuRoute[])
    : (menuRouters.value as MenuRoute[]);
});
</script>
