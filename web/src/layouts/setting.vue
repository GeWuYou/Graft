<template>
  <scroll-edge-actions
    v-if="activeController"
    :controller="activeController"
    :labels="{
      toTop: t('layout.scrollActions.top'),
      toBottom: t('layout.scrollActions.bottom'),
    }"
  />
  <theme-workbench-dock v-if="settingStore.showThemeWorkbenchDock" />
  <theme-workbench-panel />
</template>
<script setup lang="ts">
import { computed } from 'vue';

import { t } from '@/locales';
import { ScrollEdgeActions } from '@/shared/components/scroll';
import { useScrollEdgeActionsContext } from '@/shared/composables';
import { useSettingStore } from '@/store';

import ThemeWorkbenchDock from './components/theme-workbench/ThemeWorkbenchDock.vue';
import ThemeWorkbenchPanel from './components/theme-workbench/ThemeWorkbenchPanel.vue';

const settingStore = useSettingStore();
const scrollEdgeActionsContext = useScrollEdgeActionsContext();
const activeController = computed(() => scrollEdgeActionsContext?.activeController.value ?? null);
</script>
