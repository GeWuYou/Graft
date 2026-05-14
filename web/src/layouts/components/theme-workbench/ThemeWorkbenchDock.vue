<template>
  <div class="theme-workbench-dock">
    <t-button
      class="theme-workbench-dock__main"
      :class="{ 'theme-workbench-dock__action--active': isGroupActive('overview') }"
      size="large"
      variant="outline"
      @click="toggleOverview"
    >
      <template #icon>
        <t-icon name="palette" />
      </template>
      {{ t('layout.setting.workbench.dock.title') }}
    </t-button>
    <t-button
      v-for="entry in quickEntries"
      :key="entry.group"
      class="theme-workbench-dock__action"
      :class="{ 'theme-workbench-dock__action--active': isGroupActive(entry.group) }"
      :title="t(entry.labelKey)"
      shape="circle"
      variant="outline"
      @click="openGroup(entry.group)"
    >
      <t-icon :name="entry.icon" />
    </t-button>
    <t-button shape="circle" variant="outline" @click="resetWorkbench">
      <t-icon name="rollback" />
    </t-button>
  </div>
</template>
<script setup lang="ts">
import { t } from '@/locales';
import { useSettingStore } from '@/store';
import type { ThemeWorkbenchGroupKey } from '@/types/theme';

const settingStore = useSettingStore();

const quickEntries = [
  { group: 'brand' as const, icon: 'edit-1', labelKey: 'layout.setting.workbench.groups.brand' },
  { group: 'semantic' as const, icon: 'color-picker', labelKey: 'layout.setting.workbench.groups.semantic' },
  { group: 'font' as const, icon: 'textformat', labelKey: 'layout.setting.workbench.groups.font' },
  { group: 'radius' as const, icon: 'chart-bubble', labelKey: 'layout.setting.workbench.groups.radius' },
];

const openGroup = (group: ThemeWorkbenchGroupKey) => {
  settingStore.openThemeWorkbench(group);
};

const isGroupActive = (group: ThemeWorkbenchGroupKey) => {
  return settingStore.showThemeWorkbench && settingStore.activeThemeWorkbenchGroup === group;
};

// 底部 dock 作为全局入口，概览按钮在工作台已打开且停留在概览页时直接承担关闭动作。
const toggleOverview = () => {
  if (isGroupActive('overview')) {
    settingStore.closeThemeWorkbench();
    return;
  }

  openGroup('overview');
};

const resetWorkbench = () => {
  settingStore.resetThemeWorkbench();
};
</script>
<style lang="less" scoped>
.theme-workbench-dock {
  align-items: center;
  backdrop-filter: blur(14px);
  background: color-mix(in srgb, var(--td-bg-color-container) 92%, transparent);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  border-radius: 999px;
  bottom: 28px;
  box-shadow: 0 14px 30px rgb(15 23 42 / 14%);
  display: inline-flex;
  gap: 10px;
  left: 50%;
  padding: 10px 12px;
  position: fixed;
  transform: translateX(-50%);
  z-index: 1100;
}

.theme-workbench-dock__main {
  min-width: 156px;
}

.theme-workbench-dock__action--active {
  border-color: var(--td-brand-color);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--td-brand-color) 18%, transparent);
  color: var(--td-brand-color);
}

:deep(.t-button--variant-outline) {
  backdrop-filter: blur(10px);
}
</style>
