<template>
  <theme-workbench-dock v-if="showFloatingWorkbench" />
  <theme-workbench-panel />
</template>
<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useRoute } from 'vue-router';

import { useSettingStore } from '@/store';

import ThemeWorkbenchDock from './components/theme-workbench/ThemeWorkbenchDock.vue';
import ThemeWorkbenchPanel from './components/theme-workbench/ThemeWorkbenchPanel.vue';

const route = useRoute();
const settingStore = useSettingStore();
const showFloatingWorkbench = computed(() => route.path !== '/login');

onMounted(() => {
  // 主题工作台宿主提升到 App 根节点后，运行时初始化只保留这一条入口。
  settingStore.initializeThemeWorkbenchRuntime();
});
</script>
