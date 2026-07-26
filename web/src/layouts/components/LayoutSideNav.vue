<template>
  <responsive-sidebar
    v-if="settingStore.showSidebar"
    :mode="presentation"
    :visible="drawerVisible"
    @update:visible="emit('update:drawerVisible', $event)"
  >
    <template #default="{ compact }">
      <l-side-nav
        :show-logo="settingStore.showSidebarLogo"
        :layout="settingStore.layout"
        :is-fixed="settingStore.isSidebarFixed"
        :menu="sideMenu"
        :theme="settingStore.displaySideMode"
        :is-compact="widthCompact || compact"
        :render-compact="renderCompact || compact"
        :motion-phase="motionPhase"
      />
    </template>
    <template #drawer>
      <l-side-nav
        :show-logo="settingStore.showSidebarLogo"
        :layout="settingStore.layout"
        drawer-mode
        :menu="sideMenu"
        :theme="settingStore.displaySideMode"
        :is-compact="false"
        :render-compact="false"
        motion-phase="expanded"
      />
    </template>
  </responsive-sidebar>
</template>
<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { computed } from 'vue';
import { useRoute } from 'vue-router';

import { selectMixSidebarMenu, type SidebarMotionPhase, type SidebarPresentation } from '@/layouts/layout-navigation';
import ResponsiveSidebar from '@/shared/components/responsive/ResponsiveSidebar.vue';
import { usePermissionStore, useSettingStore } from '@/store';
import type { MenuRoute } from '@/utils/types';

import LSideNav from './SideNav.vue';

defineProps<{
  drawerVisible: boolean;
  presentation: SidebarPresentation;
  renderCompact: boolean;
  widthCompact: boolean;
  motionPhase: SidebarMotionPhase;
}>();

const emit = defineEmits<{
  'update:drawerVisible': [visible: boolean];
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
