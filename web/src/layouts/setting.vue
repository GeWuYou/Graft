<template>
  <theme-workbench-dock v-if="settingStore.showThemeWorkbenchDock" />
  <theme-workbench-panel />
</template>
<script setup lang="ts">
import { useSettingStore } from '@/store';

import ThemeWorkbenchDock from './components/theme-workbench/ThemeWorkbenchDock.vue';
import ThemeWorkbenchPanel from './components/theme-workbench/ThemeWorkbenchPanel.vue';

const settingStore = useSettingStore();
</script>
