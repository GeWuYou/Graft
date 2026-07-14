<template>
  <theme-workbench-dock v-if="settingStore.showThemeWorkbenchDock" />
  <theme-workbench-panel />
</template>
<script setup lang="ts">
import { onMounted } from 'vue';

import { useSettingStore } from '@/store';

import ThemeWorkbenchDock from './components/theme-workbench/ThemeWorkbenchDock.vue';
import ThemeWorkbenchPanel from './components/theme-workbench/ThemeWorkbenchPanel.vue';

const settingStore = useSettingStore();

onMounted(() => {
  // 主题工作台宿主提升到 App 根节点后，运行时初始化只保留这一条入口。
  settingStore.initializeThemeWorkbenchRuntime();
});
</script>
