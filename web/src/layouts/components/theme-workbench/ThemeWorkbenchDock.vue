<template>
  <div class="theme-workbench-dock">
    <t-button
      class="theme-workbench-dock__main"
      :class="{ 'theme-workbench-dock__action--active': isGroupActive('overview') }"
      :title="isGroupActive('overview') ? undefined : t('layout.setting.workbench.dock.title')"
      variant="outline"
      @click="toggleOverview"
    >
      <template #icon>
        <t-icon name="app" />
      </template>
      <span class="theme-workbench-dock__action-label">{{ t('layout.setting.workbench.dock.title') }}</span>
    </t-button>
    <div class="theme-workbench-dock__group">
      <t-button
        v-for="entry in quickEntries"
        :key="entry.group"
        class="theme-workbench-dock__action"
        :class="{ 'theme-workbench-dock__action--active': isGroupActive(entry.group) }"
        :title="isGroupActive(entry.group) ? undefined : t(entry.labelKey)"
        variant="outline"
        @click="openGroup(entry.group)"
      >
        <template #icon>
          <t-icon :name="entry.icon" />
        </template>
        <span class="theme-workbench-dock__action-label">{{ t(entry.labelKey) }}</span>
      </t-button>
    </div>
    <t-button
      class="theme-workbench-dock__reset"
      :title="t('layout.setting.workbench.actions.reset')"
      shape="circle"
      variant="outline"
      @click="resetWorkbench"
    >
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
  { group: 'brand' as const, icon: 'fill-color', labelKey: 'layout.setting.workbench.groups.brand' },
  { group: 'semantic' as const, icon: 'component-grid', labelKey: 'layout.setting.workbench.groups.semantic' },
  { group: 'font' as const, icon: 'text', labelKey: 'layout.setting.workbench.groups.font' },
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
  backdrop-filter: blur(24px) saturate(160%);
  background:
    linear-gradient(135deg, rgb(255 255 255 / 56%), rgb(255 255 255 / 18%)),
    color-mix(in srgb, var(--td-bg-color-container) 84%, transparent);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 54%, rgb(255 255 255 / 52%));
  border-radius: 28px;
  bottom: 28px;
  box-shadow:
    0 18px 44px rgb(15 23 42 / 16%),
    inset 0 1px 0 rgb(255 255 255 / 44%);
  box-sizing: border-box;
  display: inline-flex;
  flex-wrap: nowrap;
  gap: 12px;
  justify-content: center;
  left: 50%;
  max-width: calc(100vw - 24px);
  overflow: visible;
  padding: 8px;
  position: fixed;
  scrollbar-width: none;
  transform: translateX(-50%);
  width: max-content;
  z-index: 1100;
}

.theme-workbench-dock::-webkit-scrollbar {
  display: none;
}

.theme-workbench-dock__main {
  flex: 0 0 auto;
  min-width: 44px;
}

.theme-workbench-dock__group {
  align-items: center;
  background: color-mix(in srgb, var(--td-bg-color-container) 72%, transparent);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 44%, transparent);
  border-radius: 22px;
  box-shadow: inset 0 1px 0 rgb(255 255 255 / 32%);
  display: inline-flex;
  flex: 0 0 auto;
  gap: 8px;
  min-width: 0;
  padding: 4px;
}

.theme-workbench-dock__action--active {
  background: color-mix(in srgb, var(--td-brand-color) 12%, var(--td-bg-color-container));
  border-color: color-mix(in srgb, var(--td-brand-color) 28%, transparent);
  box-shadow:
    0 10px 20px color-mix(in srgb, var(--td-brand-color) 16%, transparent),
    inset 0 1px 0 rgb(255 255 255 / 34%);
  color: var(--td-brand-color);
}

:deep(.t-button--variant-outline) {
  backdrop-filter: blur(14px);
  background: color-mix(in srgb, var(--td-bg-color-container) 76%, transparent);
  border-color: color-mix(in srgb, var(--td-component-stroke) 48%, transparent);
  box-shadow:
    0 8px 20px rgb(15 23 42 / 8%),
    inset 0 1px 0 rgb(255 255 255 / 30%);
  color: var(--td-text-color-secondary);
  transition:
    transform 0.18s ease,
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    background-color 0.18s ease,
    color 0.18s ease;
}

:deep(.t-button--variant-outline:hover) {
  background: color-mix(in srgb, var(--td-bg-color-container) 90%, transparent);
  border-color: color-mix(in srgb, var(--td-brand-color) 16%, var(--td-component-stroke));
  color: var(--td-text-color-primary);
  transform: translateY(-1px);
}

:deep(.theme-workbench-dock__main.t-button) {
  .theme-workbench-dock-button-base();

  font-weight: 600;
}

// 主入口与快捷入口都需要同一套展开/收起交互动效，统一收口避免后续样式漂移。
.theme-workbench-dock-button-base() {
  align-items: center;
  border-radius: 999px;
  display: inline-flex;
  flex: 0 0 auto;
  height: 44px;
  justify-content: center;
  min-width: 44px;
  overflow: hidden;
  padding: 0;
  transition:
    min-width 0.22s ease,
    padding-inline 0.22s ease,
    background-color 0.18s ease,
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    color 0.18s ease,
    transform 0.18s ease;
}

.theme-workbench-dock-button-text-base() {
  max-width: 0;
  opacity: 0;
  overflow: hidden;
  transition:
    max-width 0.22s ease,
    margin-inline-start 0.22s ease,
    opacity 0.16s ease;
  white-space: nowrap;
}

.theme-workbench-dock-button-label-margin-collapsed() {
  margin-left: 0;
}

.theme-workbench-dock-button-active-base(@min-width) {
  min-width: @min-width;
  padding-inline: 16px;
}

.theme-workbench-dock-button-label-margin-expanded() {
  margin-left: 8px;
}

.theme-workbench-dock-button-text-active() {
  max-width: 96px;
  opacity: 1;
}

// 激活态 pill 需要围绕按钮中心展开，否则图标和文字会整体向一侧偏移。
:deep(.theme-workbench-dock__main .t-button__text) {
  .theme-workbench-dock-button-text-base();
}

:deep(.theme-workbench-dock__main .t-icon + .t-button__text:not(:empty)) {
  .theme-workbench-dock-button-label-margin-collapsed();
}

:deep(.theme-workbench-dock__main.theme-workbench-dock__action--active.t-button) {
  .theme-workbench-dock-button-active-base(132px);
}

:deep(.theme-workbench-dock__main.theme-workbench-dock__action--active.t-button .t-icon + .t-button__text:not(:empty)) {
  .theme-workbench-dock-button-label-margin-expanded();
}

:deep(.theme-workbench-dock__main.theme-workbench-dock__action--active.t-button .t-button__text) {
  .theme-workbench-dock-button-text-active();
}

:deep(.theme-workbench-dock__action.t-button) {
  .theme-workbench-dock-button-base();
}

:deep(.theme-workbench-dock__action .t-button__text) {
  .theme-workbench-dock-button-text-base();
}

:deep(.theme-workbench-dock__action .t-icon + .t-button__text:not(:empty)) {
  .theme-workbench-dock-button-label-margin-collapsed();
}

:deep(.theme-workbench-dock__action--active.t-button) {
  .theme-workbench-dock-button-active-base(116px);
}

:deep(.theme-workbench-dock__action--active.t-button .t-icon + .t-button__text:not(:empty)) {
  .theme-workbench-dock-button-label-margin-expanded();
}

:deep(.theme-workbench-dock__action--active.t-button .t-button__text) {
  .theme-workbench-dock-button-text-active();
}

:deep(.theme-workbench-dock__reset.t-button) {
  background: color-mix(in srgb, var(--td-bg-color-container) 66%, transparent);
  border-radius: 18px;
  flex: 0 0 auto;
  height: 44px;
  justify-content: center;
  min-width: 44px;
  padding: 0;
  width: 44px;
}

@media (width <= 768px) {
  .theme-workbench-dock {
    gap: 8px;
    padding: 6px;
  }

  .theme-workbench-dock__main {
    min-width: 40px;
  }

  .theme-workbench-dock__group {
    gap: 6px;
    padding: 3px;
  }

  :deep(.theme-workbench-dock__main.t-button) {
    height: 40px;
    min-width: 40px;
  }

  :deep(.theme-workbench-dock__action.t-button),
  :deep(.theme-workbench-dock__reset.t-button) {
    height: 40px;
    min-width: 40px;
    width: 40px;
  }

  :deep(.theme-workbench-dock__action--active.t-button) {
    min-width: 104px;
    padding-inline: 14px;
    width: auto;
  }

  :deep(.theme-workbench-dock__main.theme-workbench-dock__action--active.t-button) {
    min-width: 116px;
  }

  :deep(.theme-workbench-dock__action--active.t-button .t-button__text) {
    max-width: 72px;
  }
}
</style>
