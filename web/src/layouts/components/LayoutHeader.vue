<template>
  <l-header
    v-if="settingStore.showHeader"
    :show-logo="settingStore.showHeaderLogo"
    :theme="settingStore.displayMode"
    :layout="settingStore.layout"
    :is-fixed="settingStore.isHeaderFixed"
    :menu="headerMenu"
    :is-compact="renderCompact"
    :navigation-presentation="presentation"
    :sidebar-visible="presentation !== 'drawer'"
    :show-navigation-toggle="presentation !== 'drawer'"
    @open-navigation="emit('open-navigation')"
  />
</template>
<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { computed } from 'vue';

import { flattenMixHeaderMenus, type SidebarPresentation } from '@/layouts/layout-navigation';
import { usePermissionStore, useSettingStore } from '@/store';
import type { MenuRoute } from '@/utils/types';

defineProps<{
  presentation: SidebarPresentation;
  renderCompact: boolean;
}>();

const emit = defineEmits<{
  'open-navigation': [];
}>();

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
