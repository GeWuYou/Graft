<template>
  <t-drawer
    v-model:visible="drawerVisible"
    class="theme-workbench-panel"
    destroy-on-close
    placement="right"
    :size="drawerSize"
    :close-btn="false"
    :footer="false"
    :header="false"
  >
    <div class="theme-workbench-panel__shell">
      <header class="theme-workbench-panel__header">
        <div>
          <div class="panel-title">{{ t('layout.setting.workbench.title') }}</div>
          <div class="panel-subtitle">{{ t('layout.setting.workbench.subtitle') }}</div>
        </div>
        <t-button shape="square" variant="text" @click="closeWorkbench">
          <t-icon name="close" />
        </t-button>
      </header>

      <div class="theme-workbench-panel__body">
        <aside class="theme-workbench-panel__nav">
          <button
            v-for="group in groups"
            :key="group.key"
            type="button"
            class="nav-item"
            :class="{ 'nav-item--active': group.key === settingStore.activeThemeWorkbenchGroup }"
            @click="openGroup(group.key)"
          >
            <span class="nav-item__icon">
              <t-icon :name="groupIconMap[group.key]" />
            </span>
            <span class="nav-item__text">{{ t(group.labelKey) }}</span>
          </button>
        </aside>

        <section class="theme-workbench-panel__content">
          <template v-if="settingStore.activeThemeWorkbenchGroup === 'overview'">
            <div class="section">
              <div class="section-title">{{ t('layout.setting.theme.mode') }}</div>
              <div class="mode-grid">
                <button
                  v-for="item in modeOptions"
                  :key="item.type"
                  type="button"
                  class="mode-card"
                  :class="{ 'mode-card--active': settingStore.mode === item.type }"
                  @click="settingStore.updateConfig({ mode: item.type })"
                >
                  <span class="mode-card__preview">
                    <component :is="item.icon" class="mode-card__icon" />
                    <span v-if="settingStore.mode === item.type" class="mode-card__check">
                      <t-icon name="check" />
                    </span>
                  </span>
                  <span class="mode-card__label">{{ item.text }}</span>
                </button>
              </div>
            </div>

            <div class="section">
              <div class="section-title">{{ t('layout.setting.workbench.presets.title') }}</div>
              <div class="preset-grid">
                <button
                  v-for="preset in presetDefinitions"
                  :key="preset.id"
                  type="button"
                  class="preset-card"
                  :class="{ 'preset-card--active': selectedPresetId === preset.id }"
                  @click="settingStore.selectThemePreset(preset.id)"
                >
                  <span class="preset-card__swatch" :style="{ background: preset.brandTheme }" />
                  <span class="preset-card__title">{{ preset.label }}</span>
                  <span class="preset-card__desc">{{ preset.description }}</span>
                </button>
              </div>
            </div>

            <div class="section">
              <div class="section-title">{{ t('layout.setting.theme.color') }}</div>
              <div class="brand-palette">
                <button
                  v-for="color in brandOptions"
                  :key="color"
                  type="button"
                  class="brand-palette__item"
                  :class="{ 'brand-palette__item--active': settingStore.brandTheme === color }"
                  :style="{ background: color }"
                  @click="settingStore.setCustomBrandTheme(color)"
                />
              </div>
              <div class="brand-input">
                <input
                  type="color"
                  :value="settingStore.brandTheme"
                  @input="settingStore.setCustomBrandTheme(($event.target as HTMLInputElement).value)"
                />
                <t-input
                  :model-value="settingStore.brandTheme"
                  @update:model-value="(value) => settingStore.setCustomBrandTheme(value)"
                />
              </div>
            </div>

            <div class="section">
              <div class="section-heading">
                <div class="section-title">{{ t('layout.setting.navigationLayout') }}</div>
                <div class="section-tip">{{ t('layout.setting.workbench.layout.tip') }}</div>
              </div>
              <div class="layout-grid">
                <button
                  v-for="layoutOption in layoutOptions"
                  :key="layoutOption.value"
                  type="button"
                  class="layout-card"
                  :class="{
                    'layout-card--active': settingStore.layout === layoutOption.value,
                  }"
                  @click="selectLayout(layoutOption.value)"
                >
                  <thumbnail :src="layoutOption.thumbnail" />
                  <span class="layout-card__title">{{ layoutOption.label }}</span>
                </button>
              </div>
              <div v-if="settingStore.layout === 'mix'" class="switch-list switch-list--layout">
                <div class="switch-item">
                  <span>{{ t('layout.setting.splitMenu') }}</span>
                  <t-switch v-model="settingStore.splitMenu" />
                </div>
                <div class="switch-item">
                  <span>{{ t('layout.setting.fixedSidebar') }}</span>
                  <t-switch v-model="settingStore.isSidebarFixed" />
                </div>
              </div>
            </div>

            <div class="section">
              <div class="section-title">{{ t('layout.setting.element.title') }}</div>
              <div class="switch-list">
                <div class="switch-item">
                  <span>{{ t('layout.setting.element.showHeader') }}</span>
                  <t-switch v-model="settingStore.showHeader" />
                </div>
                <div class="switch-item">
                  <span>{{ t('layout.setting.element.showBreadcrumb') }}</span>
                  <t-switch v-model="settingStore.showBreadcrumb" />
                </div>
                <div class="switch-item">
                  <span>{{ t('layout.setting.element.showFooter') }}</span>
                  <t-switch v-model="settingStore.showFooter" />
                </div>
                <div class="switch-item">
                  <span>{{ t('layout.setting.element.useTagTabs') }}</span>
                  <t-switch v-model="settingStore.isUseTabsRouter" />
                </div>
                <div class="switch-item">
                  <span>{{ t('layout.setting.element.menuAutoCollapsed') }}</span>
                  <t-switch v-model="settingStore.menuAutoCollapsed" />
                </div>
              </div>
              <div class="section-actions">
                <t-button block variant="outline" @click="settingStore.resetThemeWorkbench()">
                  {{ t('layout.setting.workbench.actions.reset') }}
                </t-button>
              </div>
            </div>
          </template>

          <template v-else>
            <div class="section section--compact">
              <div class="section-title">{{ activeGroupLabel }}</div>
              <div class="section-desc">{{ t('layout.setting.workbench.token.description') }}</div>
            </div>
            <theme-token-editor
              :group-key="settingStore.activeThemeTokenGroup"
              :token-definitions="activeTokenDefinitions"
            />
          </template>
        </section>
      </div>
    </div>
  </t-drawer>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import SettingAutoIcon from '@/assets/assets-setting-auto.svg';
import SettingDarkIcon from '@/assets/assets-setting-dark.svg';
import SettingLightIcon from '@/assets/assets-setting-light.svg';
import { DEFAULT_COLOR_OPTIONS } from '@/config/color';
import { t } from '@/locales';
import Thumbnail from '@/shared/components/ThumbnailImage.vue';
import { useSettingStore } from '@/store';
import type { ThemeWorkbenchGroupKey } from '@/types/theme';

import ThemeTokenEditor from './ThemeTokenEditor.vue';

const settingStore = useSettingStore();

const groupIconMap: Record<ThemeWorkbenchGroupKey, string> = {
  overview: 'app',
  brand: 'fill-color',
  semantic: 'component-grid',
  neutral: 'layers',
  font: 'text',
  radius: 'chart-bubble',
  shadow: 'gesture-pray',
  size: 'fullscreen',
};

const groups = computed(() => settingStore.themeWorkbenchGroups);
const presetDefinitions = computed(() => settingStore.themePresetDefinitions);
const selectedPresetId = computed(() => settingStore.selectedThemePresetId);
const brandOptions = DEFAULT_COLOR_OPTIONS;
const drawerSize = 'min(520px, calc(100vw - 16px))';

const modeOptions = [
  { type: 'light' as const, text: t('layout.setting.theme.options.light'), icon: SettingLightIcon },
  { type: 'dark' as const, text: t('layout.setting.theme.options.dark'), icon: SettingDarkIcon },
  { type: 'auto' as const, text: t('layout.setting.theme.options.auto'), icon: SettingAutoIcon },
];

const drawerVisible = computed({
  get: () => settingStore.showThemeWorkbench,
  set: (visible: boolean) => {
    if (!visible) {
      settingStore.closeThemeWorkbench();
      return;
    }

    settingStore.openThemeWorkbench(settingStore.activeThemeWorkbenchGroup);
  },
});

const layoutOptions = computed(() => [
  {
    value: 'side' as const,
    label: t('layout.setting.workbench.layout.side'),
    description: t('layout.setting.workbench.layout.descriptions.side'),
    thumbnail: 'https://tdesign.gtimg.com/tdesign-pro/setting/side.png',
  },
  {
    value: 'top' as const,
    label: t('layout.setting.workbench.layout.top'),
    description: t('layout.setting.workbench.layout.descriptions.top'),
    thumbnail: 'https://tdesign.gtimg.com/tdesign-pro/setting/top.png',
  },
  {
    value: 'mix' as const,
    label: t('layout.setting.workbench.layout.mix'),
    description: t('layout.setting.workbench.layout.descriptions.mix'),
    thumbnail: 'https://tdesign.gtimg.com/tdesign-pro/setting/mix.png',
  },
]);

const activeTokenDefinitions = computed(() =>
  settingStore.themeTokenDefinitions.filter((item) => item.group === settingStore.activeThemeTokenGroup),
);

const activeGroupLabel = computed(() => {
  const group = groups.value.find((item) => item.key === settingStore.activeThemeWorkbenchGroup);
  return group ? t(group.labelKey) : t('layout.setting.workbench.title');
});

const openGroup = (group: ThemeWorkbenchGroupKey) => {
  settingStore.setActiveThemeWorkbenchGroup(group);
};

const closeWorkbench = () => {
  settingStore.closeThemeWorkbench();
};

const selectLayout = (layout: 'side' | 'top' | 'mix') => {
  settingStore.updateConfig({ layout });
};
</script>
<style lang="less" scoped>
.theme-workbench-panel {
  :deep(.t-drawer) {
    background: transparent;
    max-width: calc(100vw - 16px);
  }

  :deep(.t-drawer__content) {
    background: var(--td-bg-color-page);
  }

  :deep(.t-drawer__body) {
    padding: 0;
  }
}

.theme-workbench-panel__shell {
  background: var(--td-bg-color-page);
  display: flex;
  flex-direction: column;
  height: 100%;
}

.theme-workbench-panel__header {
  align-items: flex-start;
  background: linear-gradient(90deg, #6c42f6 0%, #5a33ec 100%);
  border-bottom: 1px solid rgb(255 255 255 / 14%);
  color: #fff;
  display: flex;
  justify-content: space-between;
  padding: 16px 18px 14px;
}

.theme-workbench-panel__header :deep(.t-button) {
  color: #fff;
}

.panel-title {
  font-size: 20px;
  font-weight: 700;
  line-height: 1.1;
}

.panel-subtitle {
  font-size: 13px;
  margin-top: 3px;
  opacity: 0.72;
}

.theme-workbench-panel__body {
  align-items: start;
  display: grid;
  flex: 1;
  gap: 14px;
  grid-template-columns: 64px minmax(0, 1fr);
  min-height: 0;
  padding: 16px 14px 0;
}

.theme-workbench-panel__nav {
  display: grid;
  gap: 8px;
  height: fit-content;
  min-width: 0;
}

.nav-item {
  appearance: none;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 16px;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  display: grid;
  gap: 5px;
  padding: 10px 4px;
  place-items: center center;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease,
    color 0.2s ease,
    transform 0.2s ease;
}

.nav-item--active {
  border-color: var(--td-brand-color);
  box-shadow: 0 6px 18px color-mix(in srgb, var(--td-brand-color) 12%, transparent);
  color: var(--td-brand-color);
  transform: translateY(-1px);
}

.nav-item__icon {
  align-items: center;
  display: inline-flex;
  font-size: 20px;
  justify-content: center;
}

.nav-item__text {
  font-size: 12px;
  line-height: 1.2;
  text-align: center;
}

.theme-workbench-panel__content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 0;
  min-width: 0;
  overflow: auto;
  padding-right: 2px;
}

.section {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 20px;
  display: grid;
  gap: 14px;
  padding: 16px;
}

.section--compact {
  gap: 8px;
}

.section-title {
  color: var(--td-text-color-primary);
  font-size: 16px;
  font-weight: 700;
}

.section-heading {
  display: grid;
  gap: 6px;
}

.section-tip {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.section-desc {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
}

.mode-grid,
.preset-grid,
.layout-grid {
  display: grid;
  gap: 12px;
}

.mode-grid {
  grid-template-columns: repeat(auto-fit, minmax(108px, 1fr));
}

.preset-grid {
  grid-template-columns: repeat(auto-fit, minmax(132px, 1fr));
}

.layout-grid {
  grid-template-columns: repeat(auto-fit, minmax(104px, 1fr));
}

.mode-card,
.preset-card,
.layout-card {
  appearance: none;
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-component-stroke);
  border-radius: 16px;
  cursor: pointer;
  display: grid;
  gap: 8px;
  min-width: 0;
  padding: 12px;
  text-align: left;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease,
    transform 0.2s ease;
}

.mode-card--active,
.preset-card--active,
.layout-card--active {
  border-color: var(--td-brand-color);
  box-shadow: 0 10px 24px color-mix(in srgb, var(--td-brand-color) 12%, transparent);
  transform: translateY(-1px);
}

.mode-card {
  align-content: start;
  justify-items: center;
  padding-bottom: 10px;
  text-align: center;
}

.mode-card__preview {
  display: grid;
  inline-size: min(100%, 88px);
  place-items: center;
  position: relative;
  width: 100%;
}

.mode-card__icon {
  aspect-ratio: 11 / 6;
  display: block;
  height: auto;
  max-width: 88px;
  width: 100%;
}

.mode-card__preview :deep(svg) {
  aspect-ratio: 11 / 6;
  border-radius: 12px;
  display: block;
  height: auto;
  max-width: 88px;
  overflow: hidden;
  width: 100%;
}

.mode-card__check {
  align-items: center;
  background: rgb(0 0 0 / 58%);
  border-radius: 999px;
  bottom: 4px;
  color: #fff;
  display: inline-flex;
  height: 22px;
  justify-content: center;
  position: absolute;
  right: 6px;
  width: 22px;
}

.mode-card__label {
  color: var(--td-text-color-primary);
  font-size: 13px;
  font-weight: 600;
}

.preset-card__swatch {
  border-radius: 12px;
  display: inline-flex;
  height: 34px;
  width: 100%;
}

.preset-card__title {
  color: var(--td-text-color-primary);
  font-weight: 600;
}

.preset-card__desc {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
}

.layout-card {
  align-content: start;
  justify-items: center;
  min-width: 0;
  padding: 10px 8px 12px;
  text-align: center;
}

.layout-card :deep(.thumbnail-layout) {
  aspect-ratio: 11 / 6;
  display: block;
  height: auto;
  width: min(88px, 100%);
}

.layout-card__title {
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 600;
}

.brand-palette {
  display: grid;
  gap: 10px;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.brand-palette__item {
  appearance: none;
  border: 2px solid transparent;
  border-radius: 14px;
  cursor: pointer;
  height: 40px;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease,
    transform 0.2s ease;
}

.brand-palette__item--active {
  border-color: var(--td-text-color-primary);
  box-shadow: 0 8px 18px rgb(15 23 42 / 12%);
  transform: translateY(-1px);
}

.brand-input {
  align-items: center;
  display: grid;
  gap: 10px;
  grid-template-columns: 40px 1fr;
}

.brand-input input[type='color'] {
  appearance: none;
  background: transparent;
  border: 0;
  border-radius: 10px;
  cursor: pointer;
  height: 40px;
  padding: 0;
  width: 40px;
}

.switch-list {
  display: grid;
  gap: 8px;
}

.switch-list--layout {
  margin-top: 2px;
}

.section-actions {
  display: grid;
  gap: 10px;
}

.switch-item {
  align-items: center;
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  display: flex;
  justify-content: space-between;
  min-height: 52px;
  padding: 10px 14px;
}

@media (width <= 860px) {
  .theme-workbench-panel__body {
    gap: 12px;
    grid-template-columns: 1fr;
    padding-inline: 12px;
  }

  .theme-workbench-panel__nav {
    grid-template-columns: repeat(auto-fit, minmax(68px, 1fr));
    padding-bottom: 4px;
  }

  .nav-item {
    min-height: 64px;
    min-width: 0;
  }

  .section {
    padding: 14px;
  }

  .brand-palette {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (width <= 560px) {
  .theme-workbench-panel {
    :deep(.t-drawer) {
      max-width: 100vw;
    }
  }

  .theme-workbench-panel__header {
    padding: 14px 14px 12px;
  }

  .panel-title {
    font-size: 18px;
  }

  .theme-workbench-panel__body {
    padding: 12px 12px 0;
  }

  .theme-workbench-panel__nav {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .mode-grid,
  .preset-grid,
  .layout-grid {
    grid-template-columns: 1fr;
  }

  .brand-palette {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .brand-input {
    grid-template-columns: 1fr;
  }
}
</style>
