<template>
  <t-drawer
    v-model:visible="drawerVisible"
    class="theme-workbench-panel"
    destroy-on-close
    placement="right"
    size="460px"
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
                  <component :is="item.icon" class="mode-card__icon" />
                  <span>{{ item.text }}</span>
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
                    'layout-card--disabled': layoutOption.disabled,
                  }"
                  :disabled="layoutOption.disabled"
                  @click="selectLayout(layoutOption.value, layoutOption.disabled)"
                >
                  <thumbnail :src="layoutOption.thumbnail" />
                  <span class="layout-card__title">{{ layoutOption.label }}</span>
                  <span class="layout-card__desc">{{ layoutOption.description }}</span>
                  <span v-if="layoutOption.disabled" class="layout-card__badge">
                    {{ t('layout.setting.workbench.layout.pending') }}
                  </span>
                </button>
              </div>
              <div v-if="showMixLayoutNotice" class="layout-warning">
                {{ t('layout.setting.workbench.layout.warning') }}
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

      <footer class="theme-workbench-panel__footer">
        <t-button theme="primary" @click="copyThemeConfig">
          {{ t('layout.setting.workbench.actions.copy') }}
        </t-button>
        <t-button variant="outline" @click="settingStore.resetThemeWorkbench()">
          {{ t('layout.setting.workbench.actions.reset') }}
        </t-button>
      </footer>
    </div>
  </t-drawer>
</template>
<script setup lang="ts">
import { useClipboard } from '@vueuse/core';
import { MessagePlugin } from 'tdesign-vue-next';
import { computed } from 'vue';

import SettingAutoIcon from '@/assets/assets-setting-auto.svg';
import SettingDarkIcon from '@/assets/assets-setting-dark.svg';
import SettingLightIcon from '@/assets/assets-setting-light.svg';
import Thumbnail from '@/components/thumbnail/index.vue';
import { DEFAULT_COLOR_OPTIONS } from '@/config/color';
import { t } from '@/locales';
import { useSettingStore } from '@/store';
import type { ThemeWorkbenchGroupKey } from '@/types/theme';

import ThemeTokenEditor from './ThemeTokenEditor.vue';

const settingStore = useSettingStore();
const { copy } = useClipboard();

const groupIconMap: Record<ThemeWorkbenchGroupKey, string> = {
  overview: 'app',
  brand: 'palette',
  semantic: 'color-picker',
  neutral: 'layers',
  font: 'textformat',
  radius: 'chart-bubble',
  shadow: 'gesture-pray',
  size: 'fullscreen',
};

const groups = computed(() => settingStore.themeWorkbenchGroups);
const presetDefinitions = computed(() => settingStore.themePresetDefinitions);
const selectedPresetId = computed(() => settingStore.selectedThemePresetId);
const brandOptions = DEFAULT_COLOR_OPTIONS;

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
    disabled: true,
  },
]);

const activeTokenDefinitions = computed(() =>
  settingStore.themeTokenDefinitions.filter((item) => item.group === settingStore.activeThemeTokenGroup),
);
const showMixLayoutNotice = computed(() => settingStore.layout === 'mix');

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

// 当前布局壳只实现了 side / top 两种真实预览，禁用 mix 入口避免用户误判配置已生效。
const selectLayout = (layout: 'side' | 'top' | 'mix', disabled?: boolean) => {
  if (disabled) {
    return;
  }

  settingStore.updateConfig({ layout });
};

const copyThemeConfig = async () => {
  try {
    await copy(settingStore.themeConfigCopyText);
    MessagePlugin.success(t('layout.setting.copy.success'));
  } catch {
    MessagePlugin.error(t('layout.setting.copy.fail'));
  }
};
</script>
<style lang="less" scoped>
.theme-workbench-panel {
  :deep(.t-drawer__body) {
    padding: 0;
  }
}

.theme-workbench-panel__shell {
  background:
    linear-gradient(180deg, rgb(84 39 243 / 96%) 0, rgb(84 39 243 / 86%) 64px, var(--td-bg-color-page) 64px),
    var(--td-bg-color-page);
  display: flex;
  flex-direction: column;
  height: 100%;
}

.theme-workbench-panel__header {
  align-items: flex-start;
  color: #fff;
  display: flex;
  justify-content: space-between;
  padding: 18px 20px 14px;
}

.panel-title {
  font-size: 28px;
  font-weight: 700;
  line-height: 1.1;
}

.panel-subtitle {
  font-size: 18px;
  margin-top: 4px;
  opacity: 0.88;
}

.theme-workbench-panel__body {
  display: grid;
  flex: 1;
  gap: 18px;
  grid-template-columns: 74px minmax(0, 1fr);
  min-height: 0;
  padding: 12px 14px 0;
}

.theme-workbench-panel__nav {
  display: grid;
  gap: 10px;
  height: fit-content;
}

.nav-item {
  appearance: none;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 22px;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  display: grid;
  gap: 6px;
  padding: 10px 6px;
  place-items: center center;
}

.nav-item--active {
  border-color: var(--td-brand-color);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--td-brand-color) 18%, transparent);
  color: var(--td-brand-color);
}

.nav-item__icon {
  align-items: center;
  display: inline-flex;
  font-size: 22px;
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
  overflow: auto;
  padding-right: 4px;
}

.section {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 24px;
  display: grid;
  gap: 14px;
  padding: 18px;
}

.section--compact {
  gap: 8px;
}

.section-title {
  color: var(--td-text-color-primary);
  font-size: 18px;
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
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.preset-grid,
.layout-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.mode-card,
.preset-card,
.layout-card {
  appearance: none;
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-component-stroke);
  border-radius: 18px;
  cursor: pointer;
  display: grid;
  gap: 8px;
  padding: 12px;
  text-align: left;
}

.mode-card--active,
.preset-card--active,
.layout-card--active {
  border-color: var(--td-brand-color);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--td-brand-color) 22%, transparent);
}

.layout-card--disabled {
  cursor: not-allowed;
  opacity: 0.68;
}

.mode-card__icon {
  height: 48px;
  width: 88px;
}

.preset-card__swatch {
  border-radius: 12px;
  display: inline-flex;
  height: 40px;
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

.layout-card__title {
  color: var(--td-text-color-primary);
  font-weight: 600;
}

.layout-card__desc {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.layout-card__badge {
  color: var(--td-brand-color);
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
  border-radius: 16px;
  cursor: pointer;
  height: 44px;
}

.brand-palette__item--active {
  border-color: var(--td-text-color-primary);
}

.brand-input {
  align-items: center;
  display: grid;
  gap: 10px;
  grid-template-columns: 42px 1fr;
}

.brand-input input[type='color'] {
  appearance: none;
  background: transparent;
  border: 0;
  cursor: pointer;
  height: 36px;
  padding: 0;
  width: 36px;
}

.switch-list {
  display: grid;
  gap: 12px;
}

.switch-item {
  align-items: center;
  display: flex;
  justify-content: space-between;
}

.layout-warning {
  background: color-mix(in srgb, var(--td-warning-color, #ed7b2f) 10%, var(--td-bg-color-container));
  border: 1px solid color-mix(in srgb, var(--td-warning-color, #ed7b2f) 28%, transparent);
  border-radius: 16px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 1.6;
  padding: 12px 14px;
}

.theme-workbench-panel__footer {
  display: grid;
  gap: 10px;
  grid-template-columns: 1fr 1fr;
  padding: 16px 18px 18px 106px;
}
</style>
