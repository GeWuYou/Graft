<template>
  <responsive-sidebar v-if="settingStore.showSidebar">
    <l-side-nav
      :show-logo="settingStore.showSidebarLogo"
      :layout="settingStore.layout"
      :is-fixed="settingStore.isSidebarFixed"
      :menu="sideMenu"
      :theme="settingStore.displaySideMode"
      :is-compact="widthCompact"
      :render-compact="renderCompact"
      :motion-phase="motionPhase"
    />
  </responsive-sidebar>
</template>
<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { computed } from 'vue';
import { useRoute } from 'vue-router';

import { selectMixSidebarMenu, type SidebarMotionPhase } from '@/layouts/layout-navigation';
import ResponsiveSidebar from '@/shared/components/responsive/ResponsiveSidebar.vue';
import { usePermissionStore, useSettingStore } from '@/store';
import type { MenuRoute } from '@/utils/types';

import LSideNav from './SideNav.vue';

defineProps<{
  renderCompact: boolean;
  widthCompact: boolean;
  motionPhase: SidebarMotionPhase;
}>();

const route = useRoute();
const permissionStore = usePermissionStore();
const settingStore = useSettingStore();
const { routers: menuRouters } = storeToRefs(permissionStore);

const sideMenu = computed(() => {
  const { layout, splitMenu } = settingStore;
  const newMenuRouters = menuRouters.value as Array<MenuRoute>;
  if (layout === 'mix' && splitMenu) {
    return selectMixSidebarMenu(newMenuRouters, route.path);
  }
  return newMenuRouters;
});
</script>
