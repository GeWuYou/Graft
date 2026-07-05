<template>
  <l-side-nav
    v-if="settingStore.showSidebar"
    :show-logo="settingStore.showSidebarLogo"
    :layout="settingStore.layout"
    :is-fixed="settingStore.isSidebarFixed"
    :menu="sideMenu"
    :theme="settingStore.displaySideMode"
    :is-compact="widthCompact"
    :render-compact="renderCompact"
    :motion-phase="motionPhase"
  />
</template>
<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { computed } from 'vue';
import { useRoute } from 'vue-router';

import { usePermissionStore, useSettingStore } from '@/store';
import type { MenuRoute } from '@/utils/types';

defineProps<{
  renderCompact: boolean;
  widthCompact: boolean;
  motionPhase:
    | 'expanded'
    | 'collapsing-width'
    | 'collapsing-submenu'
    | 'collapsing-topmenu'
    | 'compact'
    | 'expanding-width'
    | 'expanding-topmenu'
    | 'expanding-submenu';
}>();

import LSideNav from './SideNav.vue';

const route = useRoute();
const permissionStore = usePermissionStore();
const settingStore = useSettingStore();
const { routers: menuRouters } = storeToRefs(permissionStore);

const sideMenu = computed(() => {
  const { layout, splitMenu } = settingStore;
  let newMenuRouters = menuRouters.value as Array<MenuRoute>;
  if (layout === 'mix' && splitMenu) {
    newMenuRouters.forEach((menu) => {
      if (route.path.indexOf(menu.path) === 0 && menu.children?.length) {
        newMenuRouters = [
          {
            ...menu,
            meta: {
              ...menu.meta,
              expanded: true,
            },
          },
        ];
      }
    });
  }
  return newMenuRouters;
});
</script>
