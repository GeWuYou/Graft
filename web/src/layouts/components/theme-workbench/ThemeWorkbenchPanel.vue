<template>
  <t-drawer
    v-model:visible="drawerVisible"
    class="theme-workbench-panel"
    destroy-on-close
    placement="right"
    size="720px"
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
        <aside class="theme-workbench-panel__nav narrow-scrollbar">
          <t-tooltip v-for="group in groups" :key="group.key" :content="t(group.labelKey)" placement="right" show-arrow>
            <button
              type="button"
              class="nav-item"
              :class="{ 'nav-item--active': group.key === activeGroup }"
              @click="openGroup(group.key)"
            >
              <span class="nav-item__icon">
                <t-icon :name="groupIconMap[group.key]" />
              </span>
              <span class="nav-item__text">{{ t(group.labelKey) }}</span>
            </button>
          </t-tooltip>
        </aside>

        <section class="theme-workbench-panel__content narrow-scrollbar">
          <div v-if="activeGroup === 'overview'" class="overview-layout">
            <div class="section overview-layout__summary">
              <div class="section-title">{{ t('layout.setting.workbench.overview.currentConfig') }}</div>
              <div class="section-desc">{{ t('layout.setting.workbench.overview.description') }}</div>
              <div class="config-summary-card">
                <div v-for="item in overviewSummaryItems" :key="item.key" class="config-summary-row">
                  <span class="config-summary-row__label">{{ item.label }}</span>
                  <span v-if="item.key !== 'brandTheme'" class="config-summary-row__value">{{ item.value }}</span>
                  <span v-else class="config-summary-row__value config-summary-row__value--color">
                    <span class="config-summary-color" :style="{ background: item.value }" />
                    <span class="config-summary-color__value">{{ item.value }}</span>
                  </span>
                </div>
              </div>
            </div>

            <theme-workbench-preset-section
              class="overview-layout__presets"
              :title="t('layout.setting.workbench.presets.title')"
              :presets="presetDefinitions"
              :active-preset-id="effectivePresetId"
              @select="settingStore.selectThemePreset"
            />
          </div>

          <div v-else-if="activeGroup === 'appearance'" class="settings-layout settings-layout--appearance">
            <div class="section settings-layout__section settings-layout__section--mode">
              <div class="section-heading">
                <div class="section-title">{{ t('layout.setting.theme.mode') }}</div>
                <div class="section-desc">{{ t('layout.setting.workbench.appearance.description') }}</div>
              </div>
              <div class="choice-grid choice-grid--mode">
                <button
                  v-for="item in modeOptions"
                  :key="item.type"
                  type="button"
                  class="choice-card choice-card--mode"
                  :class="{ 'choice-card--active': effectiveTheme.mode === item.type }"
                  @click="handleModeSelect(item.type, $event)"
                >
                  <span class="choice-card__check">
                    <t-icon v-if="effectiveTheme.mode === item.type" name="check" />
                  </span>
                  <span class="mode-thumbnail" :class="`mode-thumbnail--${item.type}`">
                    <component :is="item.icon" class="mode-thumbnail__icon" />
                  </span>
                  <span class="choice-card__title">{{ item.text }}</span>
                </button>
              </div>
            </div>

            <div class="section settings-layout__section settings-layout__section--color">
              <div class="section-heading">
                <div class="section-title">{{ t('layout.setting.theme.color') }}</div>
                <div class="section-desc">{{ t('layout.setting.workbench.appearance.colorDescription') }}</div>
              </div>
              <div class="brand-palette">
                <button
                  v-for="color in brandOptions"
                  :key="color"
                  type="button"
                  class="brand-palette__item"
                  :class="{ 'brand-palette__item--active': effectiveTheme.brandTheme === color }"
                  :style="{ background: color }"
                  @click="settingStore.setCustomBrandTheme(color)"
                >
                  <t-icon v-if="effectiveTheme.brandTheme === color" name="check" />
                </button>
              </div>
              <div class="brand-input">
                <div
                  class="brand-input__preview"
                  aria-hidden="true"
                  :style="{ background: effectiveTheme.brandTheme }"
                />
                <div class="brand-input__content">
                  <span class="brand-input__title">{{
                    t('layout.setting.workbench.appearance.customBrandColor')
                  }}</span>
                  <span class="brand-input__value">{{ effectiveTheme.brandTheme }}</span>
                </div>
                <div class="brand-input__picker">
                  <t-color-picker
                    class="brand-input__picker-control"
                    :color-modes="colorPickerModes"
                    format="HEX"
                    :model-value="effectiveTheme.brandTheme"
                    :popup-props="{ placement: 'bottom-right' }"
                    :show-primary-color-preview="false"
                    :swatch-colors="brandOptions"
                    @change="(value) => settingStore.setCustomBrandTheme(value)"
                  />
                  <span class="brand-input__picker-icon" aria-hidden="true">
                    <t-icon name="palette" />
                  </span>
                </div>
              </div>
            </div>

            <div class="section settings-layout__section settings-layout__section--nav">
              <div class="section-heading">
                <div class="section-title">{{ t('layout.setting.workbench.appearance.navigationAppearance') }}</div>
                <div class="section-desc">{{ t('layout.setting.workbench.appearance.navigationDescription') }}</div>
              </div>
              <div class="choice-grid">
                <button
                  v-for="item in layoutOptions"
                  :key="item.value"
                  type="button"
                  class="choice-card"
                  :class="{ 'choice-card--active': settingStore.layout === item.value }"
                  @click="settingStore.updateConfig({ layout: item.value })"
                >
                  <span class="choice-card__check">
                    <t-icon v-if="settingStore.layout === item.value" name="check" />
                  </span>
                  <span class="layout-thumbnail" :class="`layout-thumbnail--${item.value}`">
                    <span class="layout-thumbnail__header" />
                    <span class="layout-thumbnail__sidebar" />
                    <span class="layout-thumbnail__content" />
                  </span>
                  <span class="choice-card__title">{{ item.label }}</span>
                </button>
              </div>
            </div>

            <div class="section settings-layout__section settings-layout__section--content">
              <div class="section-heading">
                <div class="section-title">{{ t('layout.setting.workbench.appearance.contentAppearance') }}</div>
                <div class="section-desc">{{ t('layout.setting.workbench.appearance.contentDescription') }}</div>
              </div>
              <div class="appearance-summary-grid">
                <button type="button" class="appearance-summary-card" @click="openGroup('typography')">
                  <span class="appearance-summary-card__label">{{
                    t('layout.setting.workbench.typography.fontFamily')
                  }}</span>
                  <span class="appearance-summary-card__value">{{ activeFontLabel }}</span>
                </button>
                <button type="button" class="appearance-summary-card" @click="openGroup('typography')">
                  <span class="appearance-summary-card__label">{{
                    t('layout.setting.workbench.typography.fontSize')
                  }}</span>
                  <span class="appearance-summary-card__value">{{ activeFontSizeLabel }}</span>
                </button>
                <button type="button" class="appearance-summary-card" @click="openGroup('style')">
                  <span class="appearance-summary-card__label">{{ t('layout.setting.workbench.style.radius') }}</span>
                  <span class="appearance-summary-card__value">{{ activeRadiusLabel }}</span>
                </button>
                <button type="button" class="appearance-summary-card" @click="openGroup('style')">
                  <span class="appearance-summary-card__label">{{ t('layout.setting.workbench.style.density') }}</span>
                  <span class="appearance-summary-card__value">{{ activeDensityLabel }}</span>
                </button>
              </div>
            </div>

            <theme-workbench-preset-section
              class="settings-layout__presets"
              :title="t('layout.setting.workbench.presets.title')"
              :presets="presetDefinitions"
              :active-preset-id="effectivePresetId"
              @select="settingStore.selectThemePreset"
            />
          </div>

          <div v-else-if="activeGroup === 'layout'" class="settings-layout settings-layout--layout">
            <div class="section settings-layout__section settings-layout__section--layout-choices">
              <div class="section-title">{{ t('layout.setting.navigationLayout') }}</div>
              <div class="section-desc">{{ t('layout.setting.workbench.layout.tip') }}</div>
              <div class="choice-grid">
                <button
                  v-for="item in layoutOptions"
                  :key="item.value"
                  type="button"
                  class="choice-card"
                  :class="{ 'choice-card--active': settingStore.layout === item.value }"
                  @click="settingStore.updateConfig({ layout: item.value })"
                >
                  <span class="choice-card__check">
                    <t-icon v-if="settingStore.layout === item.value" name="check" />
                  </span>
                  <span class="layout-thumbnail" :class="`layout-thumbnail--${item.value}`">
                    <span class="layout-thumbnail__header" />
                    <span class="layout-thumbnail__sidebar" />
                    <span class="layout-thumbnail__content" />
                  </span>
                  <span class="choice-card__title">{{ item.label }}</span>
                </button>
              </div>
            </div>

            <div class="section">
              <div class="section-title">{{ t('layout.setting.workbench.layout.navigationBehavior') }}</div>
              <div class="section-desc">{{ t('layout.setting.workbench.layout.behaviorDescription') }}</div>
              <div class="switch-list">
                <div class="switch-item">
                  <div class="switch-item__content">
                    <div class="switch-item__label">{{ t('layout.setting.splitMenuShort') }}</div>
                    <div class="switch-item__hint">{{ splitMenuHint }}</div>
                  </div>
                  <t-switch
                    :model-value="settingStore.splitMenu"
                    :disabled="!splitMenuAvailable"
                    @update:model-value="(value) => settingStore.updateConfig({ splitMenu: value })"
                  />
                </div>
                <div class="switch-item">
                  <div class="switch-item__content">
                    <div class="switch-item__label">{{ t('layout.setting.fixedSidebar') }}</div>
                    <div class="switch-item__hint">{{ fixedSidebarHint }}</div>
                  </div>
                  <t-switch
                    :model-value="settingStore.isSidebarFixed"
                    :disabled="!fixedSidebarAvailable"
                    @update:model-value="(value) => settingStore.updateConfig({ isSidebarFixed: value })"
                  />
                </div>
                <div class="switch-item">
                  <div class="switch-item__content">
                    <div class="switch-item__label">{{ t('layout.setting.element.menuAutoCollapsed') }}</div>
                    <div class="switch-item__hint">
                      {{ t('layout.setting.workbench.layout.menuAutoCollapsedHint') }}
                    </div>
                  </div>
                  <t-switch
                    :model-value="settingStore.menuAutoCollapsed"
                    @update:model-value="(value) => settingStore.updateConfig({ menuAutoCollapsed: value })"
                  />
                </div>
              </div>
            </div>

            <div class="section">
              <div class="section-title">{{ t('layout.setting.element.title') }}</div>
              <div class="section-desc">{{ t('layout.setting.workbench.layout.elementsDescription') }}</div>
              <div class="switch-list">
                <div class="switch-item">
                  <div class="switch-item__content">
                    <div class="switch-item__label">{{ t('layout.setting.element.showHeader') }}</div>
                    <div class="switch-item__hint">{{ t('layout.setting.workbench.layout.showHeaderHint') }}</div>
                  </div>
                  <t-switch
                    :model-value="settingStore.showHeader"
                    @update:model-value="(value) => settingStore.updateConfig({ showHeader: value })"
                  />
                </div>
                <div class="switch-item">
                  <div class="switch-item__content">
                    <div class="switch-item__label">{{ t('layout.setting.element.showBreadcrumb') }}</div>
                    <div class="switch-item__hint">{{ t('layout.setting.workbench.layout.showBreadcrumbHint') }}</div>
                  </div>
                  <t-switch
                    :model-value="settingStore.showBreadcrumb"
                    @update:model-value="(value) => settingStore.updateConfig({ showBreadcrumb: value })"
                  />
                </div>
                <div v-if="footerOptionVisible" class="switch-item">
                  <div class="switch-item__content">
                    <div class="switch-item__label">{{ t('layout.setting.element.showFooter') }}</div>
                    <div class="switch-item__hint">{{ t('layout.setting.workbench.layout.showFooterHint') }}</div>
                  </div>
                  <t-switch
                    :model-value="settingStore.showFooter"
                    @update:model-value="(value) => settingStore.updateConfig({ showFooter: value })"
                  />
                </div>
                <div class="switch-item">
                  <div class="switch-item__content">
                    <div class="switch-item__label">{{ t('layout.setting.element.useTagTabs') }}</div>
                    <div class="switch-item__hint">{{ t('layout.setting.workbench.layout.useTagTabsHint') }}</div>
                  </div>
                  <t-switch
                    :model-value="settingStore.isUseTabsRouter"
                    @update:model-value="(value) => settingStore.updateConfig({ isUseTabsRouter: value })"
                  />
                </div>
              </div>
            </div>
          </div>

          <div v-else-if="activeGroup === 'typography'" class="settings-layout settings-layout--typography">
            <div class="section">
              <div class="section-title">{{ t('layout.setting.workbench.typography.fontFamily') }}</div>
              <div class="section-desc">{{ t('layout.setting.workbench.typography.description') }}</div>
              <div
                class="font-option-list"
                role="radiogroup"
                :aria-label="t('layout.setting.workbench.typography.fontFamily')"
              >
                <label
                  v-for="item in fontFamilyOptions"
                  :key="item.value"
                  class="font-option"
                  :class="{ 'font-option--active': effectiveTheme.fontFamilyPreset === item.value }"
                >
                  <input
                    class="font-option__input"
                    type="radio"
                    name="theme-font-family"
                    :value="item.value"
                    :checked="effectiveTheme.fontFamilyPreset === item.value"
                    @change="
                      settingStore.updateThemeDraftAppearance({
                        fontFamilyPreset: item.value,
                      })
                    "
                  />
                  <span class="font-option__main">
                    <span class="font-option__title">{{ t(item.labelKey) }}</span>
                    <span class="font-option__preview" :style="{ fontFamily: item.previewFamily }">
                      {{ t('layout.setting.workbench.typography.previewLine') }}
                    </span>
                  </span>
                  <span class="font-option__check">
                    <t-icon v-if="effectiveTheme.fontFamilyPreset === item.value" name="check" />
                  </span>
                </label>
              </div>
            </div>

            <div class="section">
              <div class="section-heading">
                <div class="section-title">{{ t('layout.setting.workbench.typography.fontSize') }}</div>
                <div class="section-desc">{{ t('layout.setting.workbench.typography.fontSizeDescription') }}</div>
              </div>
              <div class="font-size-control-stack">
                <div class="font-size-control">
                  <span class="font-size-control__icon font-size-control__icon--small" aria-hidden="true">Aa</span>
                  <t-slider
                    class="font-size-control__slider"
                    :label="false"
                    :max="fontSizeOptions.length - 1"
                    :min="0"
                    :model-value="activeFontSizeIndex"
                    :step="1"
                    @change="handleFontSizeSliderChange"
                  />
                  <span class="font-size-control__icon font-size-control__icon--large" aria-hidden="true">Aa</span>
                  <t-select
                    class="font-size-control__select"
                    :model-value="effectiveTheme.fontSizePreset"
                    :options="fontSizeSelectOptions"
                    size="small"
                    @change="handleFontSizeSelectChange"
                  />
                </div>
                <div class="font-size-control__marks" aria-hidden="true">
                  <span
                    v-for="item in fontSizeOptions"
                    :key="item.value"
                    class="font-size-control__mark"
                    :class="{ 'font-size-control__mark--active': item.value === effectiveTheme.fontSizePreset }"
                  >
                    {{ item.label }}
                  </span>
                </div>
              </div>
              <div class="font-size-preview" :style="fontSizePreviewStyle">
                <span class="font-size-preview__sample">{{
                  t('layout.setting.workbench.typography.previewLine')
                }}</span>
              </div>
            </div>
          </div>

          <div v-else-if="activeGroup === 'style'" class="settings-layout settings-layout--style">
            <div class="section">
              <div class="section-heading">
                <div class="section-title">{{ t('layout.setting.workbench.style.radius') }}</div>
                <div class="section-desc">{{ t('layout.setting.workbench.style.description') }}</div>
              </div>
              <div class="style-preview-grid">
                <button
                  v-for="item in radiusOptions"
                  :key="item.value"
                  type="button"
                  class="style-preview-card"
                  :class="{ 'style-preview-card--active': effectiveTheme.radiusPreset === item.value }"
                  @click="
                    settingStore.updateThemeDraftAppearance({
                      radiusPreset: item.value,
                    })
                  "
                >
                  <span class="style-preview-card__label">{{ item.label }}</span>
                  <span class="radius-preview" :class="`radius-preview--${item.value}`">
                    <span class="radius-preview__surface radius-preview__surface--main" />
                    <span class="radius-preview__surface radius-preview__surface--sub" />
                  </span>
                </button>
              </div>
            </div>

            <div class="section">
              <div class="section-heading">
                <div class="section-title">{{ t('layout.setting.workbench.style.shadow') }}</div>
              </div>
              <div class="style-preview-grid">
                <button
                  v-for="item in shadowOptions"
                  :key="item.value"
                  type="button"
                  class="style-preview-card"
                  :class="{ 'style-preview-card--active': effectiveTheme.shadowPreset === item.value }"
                  @click="
                    settingStore.updateThemeDraftAppearance({
                      shadowPreset: item.value,
                    })
                  "
                >
                  <span class="style-preview-card__label">{{ item.label }}</span>
                  <span class="shadow-preview" :class="`shadow-preview--${item.value}`">
                    <span class="shadow-preview__card shadow-preview__card--back" />
                    <span class="shadow-preview__card shadow-preview__card--front" />
                  </span>
                </button>
              </div>
            </div>

            <div class="section">
              <div class="section-heading">
                <div class="section-title">{{ t('layout.setting.workbench.style.density') }}</div>
              </div>
              <div class="style-preview-grid">
                <button
                  v-for="item in densityOptions"
                  :key="item.value"
                  type="button"
                  class="style-preview-card"
                  :class="{ 'style-preview-card--active': effectiveTheme.densityPreset === item.value }"
                  @click="
                    settingStore.updateThemeDraftAppearance({
                      densityPreset: item.value,
                    })
                  "
                >
                  <span class="style-preview-card__label">{{ item.label }}</span>
                  <span class="density-preview" :class="`density-preview--${item.value}`">
                    <span v-for="line in densityPreviewLines" :key="line" class="density-preview__line">
                      {{ line }}
                    </span>
                  </span>
                </button>
              </div>
            </div>
          </div>

          <div v-else class="advanced-layout">
            <div class="advanced-settings-card">
              <div class="advanced-settings-card__content">
                <div class="section-title">{{ t('layout.setting.workbench.advanced.title') }}</div>
                <div class="section-desc advanced-settings-card__desc">
                  {{ t('layout.setting.workbench.advanced.description') }}
                </div>
              </div>
              <t-switch :model-value="advancedVisible" @update:model-value="toggleAdvancedVisible" />
            </div>

            <div v-if="advancedVisible" class="advanced-mode-toolbar">
              <span class="advanced-mode-toolbar__label">{{ t('layout.setting.workbench.token.targetMode') }}</span>
              <t-radio-group v-model="activeTokenEditorMode" theme="button" variant="default-filled">
                <t-radio-button value="light">{{ t('layout.setting.workbench.token.light') }}</t-radio-button>
                <t-radio-button value="dark">{{ t('layout.setting.workbench.token.dark') }}</t-radio-button>
              </t-radio-group>
            </div>

            <div v-if="advancedVisible" class="advanced-sections">
              <section v-for="section in tokenSections" :key="section.key" class="advanced-section">
                <div class="advanced-section__title">{{ section.label }}</div>
                <t-collapse
                  v-model:value="expandedAdvancedGroups"
                  borderless
                  class="advanced-collapse"
                  expand-icon-placement="right"
                >
                  <t-collapse-panel
                    v-for="group in section.groups"
                    :key="group.value"
                    :value="group.value"
                    class="advanced-collapse-panel"
                  >
                    <template #header>
                      <span class="advanced-group__header">
                        <span class="advanced-group__icon">
                          <t-icon :name="group.icon" />
                        </span>
                        <span class="advanced-group__title">{{ group.label }}</span>
                        <span class="advanced-group__count">
                          {{ t(group.countLabelKey, { count: tokenDefinitionsByGroup[group.value].length }) }}
                        </span>
                      </span>
                    </template>
                    <template #default>
                      <theme-token-editor
                        :group-key="group.value"
                        :mode="activeTokenEditorMode"
                        :token-definitions="tokenDefinitionsByGroup[group.value]"
                      />
                    </template>
                  </t-collapse-panel>
                </t-collapse>
              </section>
            </div>
          </div>
        </section>
      </div>

      <footer class="theme-workbench-panel__footer">
        <t-button variant="outline" @click="settingStore.resetThemeDraftToDefault()">
          {{ t('layout.setting.workbench.actions.reset') }}
        </t-button>
        <div class="theme-workbench-panel__footer-actions">
          <t-button variant="outline" @click="settingStore.cancelThemeDraft()">
            {{ t('layout.setting.workbench.actions.cancel') }}
          </t-button>
          <t-button
            :theme="hasPendingChanges ? 'primary' : 'default'"
            :disabled="!hasPendingChanges"
            @click="settingStore.applyThemeDraft()"
          >
            {{ t('layout.setting.workbench.actions.apply') }}
          </t-button>
        </div>
      </footer>
    </div>
  </t-drawer>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useRoute } from 'vue-router';

import SettingAutoIcon from '@/assets/assets-setting-auto.svg';
import SettingDarkIcon from '@/assets/assets-setting-dark.svg';
import SettingLightIcon from '@/assets/assets-setting-light.svg';
import { DEFAULT_COLOR_OPTIONS } from '@/config/color';
import { t } from '@/locales';
import { warnTranslationLengthBudget } from '@/locales/length-budgets';
import { useLocale } from '@/locales/useLocale';
import { useSettingStore } from '@/store';
import type { ThemeAuthorityState, ThemeTokenGroupKey, ThemeWorkbenchGroupKey } from '@/types/theme';
import type { ModeType } from '@/utils/types';

import ThemeTokenEditor from './ThemeTokenEditor.vue';
import ThemeWorkbenchPresetSection from './ThemeWorkbenchPresetSection.vue';

const settingStore = useSettingStore();
const route = useRoute();
const { locale } = useLocale();
const advancedVisible = ref(false);
const expandedAdvancedGroups = ref<Array<string | number>>(['brand']);

const groupIconMap: Record<ThemeWorkbenchGroupKey, string> = {
  overview: 'palette',
  appearance: 'fill-color',
  layout: 'view-list',
  typography: 'text',
  style: 'chart-bubble',
  advanced: 'tools',
};

const groups = computed(() => settingStore.themeWorkbenchGroups);
const presetDefinitions = computed(() => settingStore.themePresetDefinitions);
const effectiveTheme = computed(() => settingStore.effectiveThemeState);
const effectivePresetId = computed(() => settingStore.effectiveThemeState.selectedThemePresetId);
const activeGroup = computed(() => settingStore.activeThemeWorkbenchGroup);
const themeIdentity = computed(() => settingStore.themeIdentitySummary);
const brandOptions = DEFAULT_COLOR_OPTIONS;
const colorPickerModes: Array<'monochrome'> = ['monochrome'];

const modeOptions = computed(() => [
  { type: 'light' as const, text: t('layout.setting.theme.options.light'), icon: SettingLightIcon },
  { type: 'dark' as const, text: t('layout.setting.theme.options.dark'), icon: SettingDarkIcon },
  { type: 'auto' as const, text: t('layout.setting.theme.options.auto'), icon: SettingAutoIcon },
]);

const fontFamilyOptions = [
  {
    value: 'system',
    labelKey: 'layout.setting.workbench.typography.fontFamilies.system',
    previewFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif',
  },
  {
    value: 'harmonyos',
    labelKey: 'layout.setting.workbench.typography.fontFamilies.harmonyos',
    previewFamily: '"HarmonyOS Sans SC", "HarmonyOS Sans", "PingFang SC", "Microsoft YaHei", sans-serif',
  },
  {
    value: 'inter',
    labelKey: 'layout.setting.workbench.typography.fontFamilies.inter',
    previewFamily: '"Inter", "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif',
  },
  {
    value: 'source-han-sans',
    labelKey: 'layout.setting.workbench.typography.fontFamilies.sourceHanSans',
    previewFamily: '"Source Han Sans SC", "Noto Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif',
  },
] as const;

const fontSizeOptions = computed<
  Array<{
    value: ThemeAuthorityState['fontSizePreset'];
    label: string;
    scale: string;
  }>
>(() => [
  { value: 'extra-small', label: t('layout.setting.workbench.typography.extraSmall'), scale: '88%' },
  { value: 'small', label: t('layout.setting.workbench.typography.small'), scale: '94%' },
  { value: 'standard', label: t('layout.setting.workbench.typography.standard'), scale: '100%' },
  { value: 'large', label: t('layout.setting.workbench.typography.large'), scale: '106%' },
  { value: 'extra-large', label: t('layout.setting.workbench.typography.extraLarge'), scale: '112%' },
]);

const fontSizeSelectOptions = computed(() => fontSizeOptions.value.map(({ label, value }) => ({ label, value })));

const radiusOptions = computed(
  () =>
    [
      { value: 'business', label: t('layout.setting.workbench.style.business') },
      { value: 'standard', label: t('layout.setting.workbench.style.standard') },
      { value: 'rounded', label: t('layout.setting.workbench.style.rounded') },
      { value: 'capsule', label: t('layout.setting.workbench.style.capsule') },
    ] as const,
);

const shadowOptions = computed(
  () =>
    [
      { value: 'flat', label: t('layout.setting.workbench.style.flat') },
      { value: 'standard', label: t('layout.setting.workbench.style.standard') },
      { value: 'floating', label: t('layout.setting.workbench.style.floating') },
    ] as const,
);

const densityOptions = computed(
  () =>
    [
      { value: 'compact', label: t('layout.setting.workbench.style.compact') },
      { value: 'standard', label: t('layout.setting.workbench.style.standard') },
      { value: 'comfortable', label: t('layout.setting.workbench.style.comfortable') },
    ] as const,
);

const densityPreviewLines = computed(() => [
  t('layout.setting.workbench.style.densityPreviewLinePrimary'),
  t('layout.setting.workbench.style.densityPreviewLineSecondary'),
  t('layout.setting.workbench.style.densityPreviewLineMeta'),
]);

const tokenSections = computed<
  Array<{
    key: 'color' | 'component';
    label: string;
    groups: Array<{
      value: ThemeTokenGroupKey;
      label: string;
      icon: string;
      countLabelKey: string;
    }>;
  }>
>(() => [
  {
    key: 'color',
    label: t('layout.setting.workbench.advanced.colorSystem'),
    groups: [
      {
        value: 'brand',
        label: t('layout.setting.workbench.groups.brand'),
        icon: 'palette',
        countLabelKey: 'layout.setting.workbench.advanced.colorVariableCount',
      },
      {
        value: 'text',
        label: t('layout.setting.workbench.groups.text'),
        icon: 'text',
        countLabelKey: 'layout.setting.workbench.advanced.colorVariableCount',
      },
      {
        value: 'background',
        label: t('layout.setting.workbench.groups.background'),
        icon: 'image',
        countLabelKey: 'layout.setting.workbench.advanced.colorVariableCount',
      },
      {
        value: 'border',
        label: t('layout.setting.workbench.groups.border'),
        icon: 'component-divider-horizontal',
        countLabelKey: 'layout.setting.workbench.advanced.colorVariableCount',
      },
    ],
  },
  {
    key: 'component',
    label: t('layout.setting.workbench.advanced.componentStyle'),
    groups: [
      {
        value: 'component',
        label: t('layout.setting.workbench.groups.component'),
        icon: 'setting',
        countLabelKey: 'layout.setting.workbench.advanced.styleVariableCount',
      },
    ],
  },
]);

const tokenGroups = computed(() => tokenSections.value.flatMap((section) => section.groups));

const tokenDefinitionsByGroup = computed(() =>
  tokenGroups.value.reduce(
    (acc, group) => {
      acc[group.value] = settingStore.themeTokenDefinitions.filter((item) => item.group === group.value);
      return acc;
    },
    {} as Record<ThemeTokenGroupKey, typeof settingStore.themeTokenDefinitions>,
  ),
);

const layoutOptions = computed(
  () =>
    [
      { value: 'side', label: t('layout.setting.workbench.layout.side') },
      { value: 'top', label: t('layout.setting.workbench.layout.top') },
      { value: 'mix', label: t('layout.setting.workbench.layout.mix') },
    ] as const,
);

const modeLabel = computed(() => {
  const matched = modeOptions.value.find((item) => item.type === effectiveTheme.value.mode);
  return matched?.text ?? effectiveTheme.value.mode;
});

const layoutLabel = computed(() => {
  const matched = layoutOptions.value.find((item) => item.value === settingStore.layout);
  return matched?.label ?? settingStore.layout;
});

const activeFontLabel = computed(() => {
  const matched = fontFamilyOptions.find((item) => item.value === effectiveTheme.value.fontFamilyPreset);
  return t(matched?.labelKey ?? fontFamilyOptions[0].labelKey);
});

const activeFontSizeIndex = computed(() => {
  const matchedIndex = fontSizeOptions.value.findIndex((item) => item.value === effectiveTheme.value.fontSizePreset);
  return Math.max(matchedIndex, 0);
});

const activeFontSizeLabel = computed(() => fontSizeOptions.value[activeFontSizeIndex.value].label);

const activeFontSizeScale = computed(() => fontSizeOptions.value[activeFontSizeIndex.value].scale);

const fontSizePreviewStyle = computed(() => ({
  '--font-size-preview-scale': activeFontSizeScale.value,
}));

const activeRadiusLabel = computed(() => {
  const matched = radiusOptions.value.find((item) => item.value === effectiveTheme.value.radiusPreset);
  return matched?.label ?? radiusOptions.value[0].label;
});

const activeDensityLabel = computed(() => {
  const matched = densityOptions.value.find((item) => item.value === effectiveTheme.value.densityPreset);
  return matched?.label ?? densityOptions.value[1].label;
});

const overviewSummaryItems = computed(() => [
  {
    key: 'mode',
    label: t('layout.setting.theme.mode'),
    value: modeLabel.value,
  },
  {
    key: 'theme',
    label: t('layout.setting.workbench.overview.currentTheme'),
    value: t(themeIdentity.value.currentLabelKey),
  },
  {
    key: 'layout',
    label: t('layout.setting.navigationLayout'),
    value: layoutLabel.value,
  },
  {
    key: 'brandTheme',
    label: t('layout.setting.theme.color'),
    value: effectiveTheme.value.brandTheme,
  },
  {
    key: 'font',
    label: t('layout.setting.workbench.typography.fontFamily'),
    value: activeFontLabel.value,
  },
  {
    key: 'fontSize',
    label: t('layout.setting.workbench.typography.fontSize'),
    value: activeFontSizeLabel.value,
  },
]);

const hasPendingChanges = computed(() => settingStore.hasThemeDraftPendingChanges);
const splitMenuAvailable = computed(() => settingStore.layout === 'mix');
const splitMenuHint = computed(() =>
  splitMenuAvailable.value
    ? t('layout.setting.workbench.layout.splitMenuHint')
    : t('layout.setting.workbench.layout.onlyMix'),
);
const fixedSidebarAvailable = computed(() => settingStore.layout !== 'top');
const fixedSidebarHint = computed(() =>
  fixedSidebarAvailable.value
    ? t('layout.setting.workbench.layout.fixedSidebarHint')
    : t('layout.setting.workbench.layout.notIntegrated'),
);
const footerOptionVisible = computed(() => route.meta.footer !== false);
const activeTokenEditorMode = ref<ModeType>(settingStore.displayMode);

const drawerVisible = computed({
  get: () => settingStore.showThemeWorkbench,
  set: (visible: boolean) => {
    if (!visible) {
      settingStore.cancelThemeDraft();
      return;
    }

    settingStore.openThemeWorkbench(settingStore.activeThemeWorkbenchGroup);
  },
});

const themeWorkbenchBudgetKeys = computed(() => [
  ...groups.value.map((group) => ({ scope: 'navigation' as const, key: group.labelKey })),
  { scope: 'button' as const, key: 'layout.setting.workbench.actions.reset' },
  { scope: 'button' as const, key: 'layout.setting.workbench.actions.cancel' },
  { scope: 'button' as const, key: 'layout.setting.workbench.actions.apply' },
]);

watch(
  [drawerVisible, locale, themeWorkbenchBudgetKeys],
  ([visible]) => {
    if (!visible) {
      return;
    }

    themeWorkbenchBudgetKeys.value.forEach((item) => {
      warnTranslationLengthBudget(item.scope, locale.value, item.key, t(item.key));
    });
  },
  { immediate: true },
);

watch(
  () => settingStore.displayMode,
  (mode) => {
    activeTokenEditorMode.value = mode;
  },
);

const openGroup = (group: ThemeWorkbenchGroupKey) => {
  settingStore.setActiveThemeWorkbenchGroup(group);
};

const updateFontSizePreset = (value: unknown) => {
  const matched = fontSizeOptions.value.find((item) => item.value === value);
  if (!matched) {
    return;
  }

  settingStore.updateThemeDraftAppearance({
    fontSizePreset: matched.value,
  });
};

const handleFontSizeSelectChange = (value: unknown) => {
  updateFontSizePreset(value);
};

const handleFontSizeSliderChange = (value: unknown) => {
  if (typeof value !== 'number') {
    return;
  }

  updateFontSizePreset(fontSizeOptions.value[value]?.value);
};

const closeWorkbench = () => {
  settingStore.cancelThemeDraft();
};

const toggleAdvancedVisible = (value: boolean) => {
  advancedVisible.value = value;
};

const handleModeSelect = (mode: ModeType | 'auto', event: MouseEvent) => {
  void settingStore.updateThemeDraftModeWithTransition(mode, event);
};
</script>
<style lang="less" scoped>
@import '../../../shared/components/management/card-surface.less';
@import './theme-surface.less';

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
  min-height: 0;
  overflow: hidden;
}

.theme-workbench-panel__header {
  align-items: flex-start;
  background: var(--td-bg-color-container);
  border-bottom: 1px solid var(--td-component-stroke);
  display: flex;
  justify-content: space-between;
  padding: var(--graft-density-gap-18) var(--graft-density-gap-20) var(--graft-density-gap-16);
}

.theme-workbench-panel__header :deep(.t-button) {
  flex: 0 0 auto;
}

.panel-title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  font-weight: 700;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  word-break: keep-all;
}

.panel-subtitle {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  line-height: 22px;
  margin-top: var(--graft-density-gap-4);
  min-height: 44px;
  overflow: hidden;
  word-break: keep-all;
}

.theme-workbench-panel__body {
  display: grid;
  flex: 1;
  gap: var(--graft-density-gap-12);
  grid-template-columns: minmax(120px, 140px) minmax(0, 1fr);
  min-height: 0;
  overflow: hidden;
  padding: var(--graft-density-gap-16);
}

.theme-workbench-panel__nav {
  align-content: start;
  align-self: stretch;
  display: grid;
  gap: var(--graft-density-gap-8);
  max-height: 100%;
  min-height: 0;
  overflow: hidden auto;
  overscroll-behavior: contain;
  padding-right: var(--graft-density-gap-2);
}

.theme-workbench-selectable-card() {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 16px;
  color: var(--td-text-color-primary);
  cursor: pointer;
}

.nav-item {
  align-items: center;
  appearance: none;
  .theme-workbench-selectable-card();

  color: var(--td-text-color-secondary);
  display: flex;
  gap: var(--graft-density-gap-8);
  min-height: 48px;
  min-width: 120px;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
  width: 100%;
}

.nav-item--active {
  border-color: var(--td-brand-color);
  box-shadow: var(--td-shadow-1);
  color: var(--td-brand-color);
}

.nav-item__icon {
  flex: 0 0 auto;
  font-size: 20px;
  line-height: 1;
}

.nav-item__text {
  font: var(--td-font-body-small);
  line-height: 1.3;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  overflow-wrap: normal;
  text-align: center;
  text-overflow: ellipsis;
  white-space: nowrap;
  word-break: keep-all;
}

.theme-workbench-panel__content {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden auto;
  padding-bottom: calc(var(--graft-density-gap-48) + var(--graft-density-gap-32) + var(--graft-density-gap-24));
  padding-right: var(--graft-density-gap-4);
}

.section {
  .theme-workbench-surface();

  gap: var(--graft-density-gap-12);
  max-width: 100%;
  min-width: 0;
  padding: var(--graft-density-gap-16);
}

.section-title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  font-weight: 700;
  min-width: 0;
  overflow: hidden;
  overflow-wrap: normal;
  text-overflow: ellipsis;
  white-space: nowrap;
  word-break: keep-all;
}

.section-heading {
  display: grid;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.section-desc {
  -webkit-box-orient: vertical;
  color: var(--td-text-color-secondary);
  display: -webkit-box;
  font: var(--td-font-body-small);
  -webkit-line-clamp: 2;
  line-height: 22px;
  max-width: 100%;
  min-height: 44px;
  min-width: 0;
  overflow: hidden;
  overflow-wrap: normal;
  word-break: keep-all;
}

.overview-layout,
.settings-layout,
.advanced-layout {
  display: grid;
  gap: var(--graft-density-gap-12);
  max-width: 100%;
  min-width: 0;
}

.choice-grid,
.brand-palette {
  display: grid;
  gap: var(--graft-density-gap-12);
  max-width: 100%;
  min-width: 0;
}

.overview-layout .section {
  padding: var(--graft-density-gap-14);
}

.config-summary-card {
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  display: grid;
  gap: var(--graft-density-gap-8) var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  padding: var(--graft-density-gap-12);
}

.config-summary-row {
  align-items: center;
  display: grid;
  gap: var(--graft-density-gap-8);
  grid-template-columns: minmax(72px, 0.75fr) minmax(0, 1fr);
  min-width: 0;
}

.config-summary-row__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.config-summary-row__value {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 600;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.config-summary-row__value--color {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-10);
  max-width: 100%;
}

.config-summary-color {
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 74%, transparent);
  border-radius: 10px;
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 18%),
    0 4px 12px rgb(15 23 42 / 10%);
  display: inline-flex;
  flex: 0 0 auto;
  height: 24px;
  width: 40px;
}

.config-summary-color__value {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.choice-grid {
  grid-template-columns: repeat(auto-fit, minmax(136px, 1fr));
}

.choice-card {
  appearance: none;
  .theme-workbench-selectable-card();

  display: grid;
  gap: var(--graft-density-gap-10);
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  padding: var(--graft-density-gap-12);
  position: relative;
  text-align: left;
}

.choice-card--active {
  border-color: var(--td-brand-color);
  box-shadow: var(--td-shadow-1);
}

.settings-layout__section--nav .choice-card,
.settings-layout__section--layout-choices .choice-card {
  isolation: isolate;
  transition:
    border-color 220ms ease,
    box-shadow 220ms ease,
    transform 220ms ease;
}

.settings-layout__section--nav .choice-card::before,
.settings-layout__section--layout-choices .choice-card::before {
  background: radial-gradient(
    circle at 50% 42%,
    color-mix(in srgb, var(--td-brand-color) 18%, transparent),
    transparent 62%
  );
  content: '';
  inset: 0;
  opacity: 0;
  pointer-events: none;
  position: absolute;
  transition: opacity 220ms ease;
  z-index: 0;
}

.settings-layout__section--nav .choice-card > *,
.settings-layout__section--layout-choices .choice-card > * {
  position: relative;
  z-index: 1;
}

.settings-layout__section--nav .choice-card:hover,
.settings-layout__section--layout-choices .choice-card:hover,
.settings-layout__section--nav .choice-card:focus-visible,
.settings-layout__section--layout-choices .choice-card:focus-visible {
  border-color: color-mix(in srgb, var(--td-brand-color) 68%, var(--td-component-stroke));
  box-shadow:
    0 0 0 3px color-mix(in srgb, var(--td-brand-color) 18%, transparent),
    0 14px 30px color-mix(in srgb, var(--td-brand-color) 16%, transparent),
    var(--td-shadow-1);
  transform: translateY(-2px);
}

.settings-layout__section--nav .choice-card:hover::before,
.settings-layout__section--layout-choices .choice-card:hover::before,
.settings-layout__section--nav .choice-card:focus-visible::before,
.settings-layout__section--layout-choices .choice-card:focus-visible::before {
  opacity: 1;
}

.settings-layout__section--mode .choice-grid,
.settings-layout__section--nav .choice-grid,
.settings-layout--layout .choice-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.settings-layout--appearance .section {
  gap: var(--graft-density-gap-10);
  padding: var(--graft-density-gap-16);
}

.settings-layout--layout .section {
  gap: var(--graft-density-gap-10);
  padding: var(--graft-density-gap-14) var(--graft-density-gap-16);
}

.choice-card__check {
  color: var(--td-brand-color);
  position: absolute;
  right: 8px;
  top: 8px;
}

.choice-card__title {
  font: var(--td-font-body-small);
  font-weight: 600;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mode-thumbnail,
.layout-thumbnail {
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  display: block;
  height: 92px;
  overflow: hidden;
  position: relative;
}

.mode-thumbnail {
  align-items: center;
  display: flex;
  justify-content: center;
  padding: var(--graft-density-gap-12);
}

.mode-thumbnail--light {
  background: linear-gradient(180deg, #fff, #f3f5f9);
}

.mode-thumbnail--dark {
  background: linear-gradient(180deg, #1f2937, #111827);
}

.mode-thumbnail--auto {
  background: linear-gradient(135deg, #fff 0 50%, #111827 50% 100%);
}

.mode-thumbnail__icon {
  max-height: 68px;
  max-width: 92px;
  width: 100%;
}

.layout-thumbnail {
  display: grid;
  grid-template-columns: 24px 1fr;
  grid-template-rows: 18px 1fr;
  padding: var(--graft-density-gap-10);
}

.layout-thumbnail__header,
.layout-thumbnail__sidebar,
.layout-thumbnail__content {
  border-radius: 8px;
  display: block;
}

.layout-thumbnail__header {
  background: color-mix(in srgb, var(--td-brand-color) 18%, var(--td-bg-color-container));
  grid-column: 1 / span 2;
}

.layout-thumbnail__sidebar {
  background: color-mix(in srgb, var(--td-brand-color) 10%, var(--td-bg-color-container));
  grid-row: 2;
  margin-top: var(--graft-density-gap-8);
}

.layout-thumbnail__content {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  grid-row: 2;
  margin-left: var(--graft-density-gap-8);
  margin-top: var(--graft-density-gap-8);
}

.layout-thumbnail--top {
  grid-template-columns: 1fr;
  grid-template-rows: 18px 1fr;
}

.layout-thumbnail--top .layout-thumbnail__header {
  grid-column: 1;
}

.layout-thumbnail--top .layout-thumbnail__sidebar {
  display: none;
}

.layout-thumbnail--top .layout-thumbnail__content {
  grid-column: 1;
  margin-left: 0;
}

.layout-thumbnail--mix::after {
  background: color-mix(in srgb, var(--td-brand-color) 16%, var(--td-bg-color-container));
  border-radius: 6px;
  content: '';
  height: 38px;
  left: 42px;
  position: absolute;
  top: 34px;
  width: 16px;
}

.switch-list {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
}

.switch-item {
  align-items: center;
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  display: flex;
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
  min-height: 52px;
  min-width: 0;
  overflow: hidden;
  padding: var(--graft-density-gap-8) var(--graft-density-gap-12);
}

.switch-item__label {
  color: var(--td-text-color-primary);
  font-weight: 500;
}

.switch-item__content {
  display: grid;
  gap: var(--graft-density-gap-2);
  min-width: 0;
}

.switch-item__hint {
  -webkit-box-orient: vertical;
  color: var(--td-text-color-secondary);
  display: -webkit-box;
  font: var(--td-font-body-small);
  -webkit-line-clamp: 2;
  line-height: 1.35;
  overflow: hidden;
}

.brand-palette {
  grid-template-columns: repeat(8, minmax(0, 1fr));
  margin-bottom: var(--graft-density-gap-6);
}

.brand-palette__item {
  align-items: center;
  appearance: none;
  border: 2px solid transparent;
  border-radius: 14px;
  color: #fff;
  cursor: pointer;
  display: inline-flex;
  height: 40px;
  justify-content: center;
}

.brand-palette__item--active {
  border-color: var(--td-text-color-primary);
}

.brand-input {
  align-items: center;
  background: color-mix(in srgb, var(--td-brand-color) 4%, var(--td-bg-color-page));
  border: 1px solid color-mix(in srgb, var(--td-brand-color) 16%, var(--td-component-stroke));
  border-radius: 10px;
  box-shadow:
    0 0 0 1px color-mix(in srgb, var(--td-brand-color) 6%, transparent),
    0 8px 18px color-mix(in srgb, var(--td-brand-color) 5%, transparent);
  box-sizing: border-box;
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: 42px minmax(0, 1fr) 34px;
  height: 68px;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  padding: var(--graft-density-gap-12) var(--graft-density-gap-14);
  position: relative;
  transition:
    border-color 200ms ease,
    box-shadow 200ms ease,
    transform 200ms ease;
}

.brand-input::before {
  background: radial-gradient(
    circle at 28% 34%,
    color-mix(in srgb, var(--td-brand-color) 12%, transparent),
    transparent 68%
  );
  content: '';
  inset: 0;
  opacity: 0;
  pointer-events: none;
  position: absolute;
  transition: opacity 200ms ease;
}

.brand-input > * {
  position: relative;
  z-index: 1;
}

.brand-input:hover,
.brand-input:focus-within {
  border-color: color-mix(in srgb, var(--td-brand-color) 24%, var(--td-component-stroke));
  box-shadow:
    0 0 0 3px color-mix(in srgb, var(--td-brand-color) 12%, transparent),
    0 12px 24px color-mix(in srgb, var(--td-brand-color) 8%, transparent);
  transform: translateY(-1px);
}

.brand-input:hover::before,
.brand-input:focus-within::before {
  opacity: 1;
}

.brand-input__content {
  display: grid;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.brand-input__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 700;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.brand-input__picker {
  flex: 0 0 auto;
  height: 34px;
  overflow: hidden;
  position: relative;
  width: 34px;
}

.brand-input__picker-control {
  display: block;
}

.brand-input__picker :deep(.t-color-picker__trigger) {
  max-width: 34px;
  overflow: hidden;
  width: 34px;
}

.brand-input__picker :deep(.t-color-picker__trigger--default) {
  max-width: 34px;
  overflow: hidden;
  width: 34px;
}

.brand-input__picker :deep(.t-input__wrap) {
  max-width: 34px;
  overflow: hidden;
  width: 34px;
}

.brand-input__picker :deep(.t-input) {
  background: color-mix(in srgb, var(--td-bg-color-container) 82%, var(--td-brand-color) 18%);
  border-color: color-mix(in srgb, var(--td-brand-color) 24%, var(--td-component-stroke));
  border-radius: 9px;
  box-shadow: none;
  color: var(--td-brand-color);
  max-width: 34px;
  overflow: hidden;
  padding: 0;
  width: 34px;
}

.brand-input__picker:hover :deep(.t-input),
.brand-input__picker:focus-within :deep(.t-input) {
  border-color: color-mix(in srgb, var(--td-brand-color) 42%, var(--td-component-stroke));
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--td-brand-color) 14%, transparent);
}

.brand-input__picker :deep(.t-input__inner),
.brand-input__picker :deep(.t-input__input-pre) {
  flex: 0 0 0 !important;
  height: 0 !important;
  margin: 0 !important;
  max-width: 0 !important;
  min-width: 0 !important;
  opacity: 0;
  padding: 0 !important;
  pointer-events: none;
  visibility: hidden;
  width: 0 !important;
}

.brand-input__picker :deep(.t-input__prefix) {
  display: inline-flex;
  margin-right: 0;
}

.brand-input__picker :deep(.t-color-picker__trigger--default__color) {
  height: 0;
  opacity: 0;
  width: 0;
}

.brand-input__picker-icon {
  align-items: center;
  color: var(--td-brand-color);
  display: inline-flex;
  inset: 0;
  justify-content: center;
  pointer-events: none;
  position: absolute;
  z-index: 1;
}

.brand-input__value {
  color: var(--td-text-color-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  font-size: 12px;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.brand-input__preview {
  align-items: center;
  border: 1px solid color-mix(in srgb, white 42%, transparent);
  border-radius: 10px;
  box-shadow:
    inset 0 0 0 1px color-mix(in srgb, var(--td-text-color-primary) 12%, transparent),
    0 8px 16px color-mix(in srgb, var(--td-brand-color) 16%, transparent);
  display: inline-flex;
  height: 40px;
  justify-content: center;
  min-width: 0;
  overflow: hidden;
  width: 40px;
}

.appearance-summary-grid {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(3, minmax(0, 1fr));
  min-width: 0;
}

.appearance-summary-card {
  appearance: none;
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-component-stroke);
  border-radius: 14px;
  color: inherit;
  cursor: pointer;
  display: grid;
  gap: var(--graft-density-gap-6);
  min-width: 0;
  overflow: hidden;
  padding: var(--graft-density-gap-12);
  text-align: left;
}

.appearance-summary-card:hover {
  background: color-mix(in srgb, var(--td-brand-color) 4%, var(--td-bg-color-page));
  border-color: color-mix(in srgb, var(--td-brand-color) 24%, var(--td-component-stroke));
}

.appearance-summary-card__label,
.appearance-summary-card__value {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.appearance-summary-card__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.appearance-summary-card__value {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 600;
}

.font-option-list {
  display: grid;
  gap: var(--graft-density-gap-10);
}

.font-option {
  align-items: center;
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-component-stroke);
  border-radius: 14px;
  cursor: pointer;
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: minmax(0, 1fr) auto;
  padding: var(--graft-density-gap-12) var(--graft-density-gap-14);
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    background-color 0.18s ease;
}

.font-option:hover {
  background: color-mix(in srgb, var(--td-bg-color-container) 82%, var(--td-bg-color-page));
  border-color: color-mix(in srgb, var(--td-brand-color) 20%, var(--td-component-stroke));
}

.font-option--active {
  border-color: color-mix(in srgb, var(--td-brand-color) 40%, var(--td-component-stroke));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--td-brand-color) 12%, transparent);
}

.font-option__input {
  opacity: 0;
  pointer-events: none;
  position: absolute;
}

.font-option__main {
  display: grid;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.font-option__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 600;
}

.font-option__preview {
  color: var(--td-text-color-secondary);
  display: block;
  font: var(--td-font-body-small);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.font-option__check {
  color: var(--td-brand-color);
}

.font-size-control-stack {
  display: grid;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.font-size-control {
  align-items: center;
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-component-stroke);
  border-radius: 14px;
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-template-columns: auto minmax(132px, 1fr) auto minmax(136px, 160px);
  max-width: 100%;
  min-height: 56px;
  min-width: 0;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
}

.font-size-control__icon {
  color: var(--td-text-color-primary);
  font-weight: 700;
  line-height: 1;
  min-width: max-content;
  white-space: nowrap;
}

.font-size-control__icon--small {
  font: var(--td-font-body-small);
}

.font-size-control__icon--large {
  font: var(--td-font-title-large);
}

.font-size-control__slider {
  min-width: 0;
}

.font-size-control__slider :deep(.t-slider__container) {
  min-width: 0;
}

.font-size-control__marks {
  color: var(--td-text-color-secondary);
  display: grid;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-8);
  grid-template-columns: repeat(5, minmax(0, 1fr));
  min-width: 0;
  padding: 0 var(--graft-density-gap-12);
}

.font-size-control__mark {
  min-width: 0;
  overflow: hidden;
  text-align: center;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.font-size-control__mark--active {
  color: var(--td-brand-color);
  font-weight: 700;
}

.font-size-control__select {
  min-width: 136px;
}

.font-size-control__select :deep(.t-input__inner) {
  min-width: 0;
  text-overflow: ellipsis;
}

.font-size-preview {
  --font-size-preview-scale: 100%;

  align-items: start;
  background: color-mix(in srgb, var(--td-bg-color-container) 72%, var(--td-bg-color-page));
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 86%, transparent);
  border-radius: 14px;
  display: grid;
  gap: var(--graft-density-gap-6);
  grid-template-columns: minmax(0, 1fr);
  min-width: 0;
  overflow-wrap: normal;
  padding: var(--graft-density-gap-12) var(--graft-density-gap-14);
  word-break: keep-all;
}

.font-size-preview__sample {
  color: var(--td-text-color-primary);
  display: block;
  font: var(--td-font-body-medium);
  font-size: var(--font-size-preview-scale);
  font-weight: 600;
  line-height: 1.45;
  min-width: 0;
  overflow: hidden;
  overflow-wrap: normal;
  text-overflow: ellipsis;
  white-space: nowrap;
  word-break: keep-all;
}

.style-preview-grid {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(auto-fit, minmax(132px, 1fr));
  min-width: 0;
}

.style-preview-card {
  align-items: stretch;
  appearance: none;
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-component-stroke);
  border-radius: 14px;
  color: var(--td-text-color-primary);
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-rows: auto 1fr;
  min-height: 144px;
  min-width: 0;
  overflow: hidden;
  padding: var(--graft-density-gap-12);
  text-align: left;
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    transform 0.18s ease,
    background-color 0.18s ease;
}

.style-preview-card:hover {
  border-color: color-mix(in srgb, var(--td-brand-color) 24%, var(--td-component-stroke));
  transform: translateY(-1px);
}

.style-preview-card--active {
  background: color-mix(in srgb, var(--td-brand-color) 5%, var(--td-bg-color-page));
  border-color: color-mix(in srgb, var(--td-brand-color) 44%, var(--td-component-stroke));
  box-shadow:
    0 0 0 1px color-mix(in srgb, var(--td-brand-color) 14%, transparent),
    var(--td-shadow-1);
}

.style-preview-card__label {
  font: var(--td-font-body-small);
  font-weight: 600;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.radius-preview,
.shadow-preview,
.density-preview {
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--td-bg-color-container) 88%, transparent), transparent),
    color-mix(in srgb, var(--td-bg-color-container) 65%, var(--td-bg-color-page));
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 86%, transparent);
  border-radius: 12px;
  box-sizing: border-box;
  height: 88px;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  padding: var(--graft-density-gap-14);
}

.radius-preview {
  align-items: end;
  display: flex;
  gap: var(--graft-density-gap-12);
}

.radius-preview__surface {
  background: color-mix(in srgb, var(--td-brand-color) 12%, var(--td-bg-color-container));
  border: 1px solid color-mix(in srgb, var(--td-brand-color) 18%, var(--td-component-stroke));
  display: block;
  min-width: 0;
}

.radius-preview__surface--main {
  flex: 1;
  height: 42px;
}

.radius-preview__surface--sub {
  height: 28px;
  width: 32%;
}

.radius-preview--business .radius-preview__surface--main,
.radius-preview--business .radius-preview__surface--sub {
  border-radius: 6px;
}

.radius-preview--standard .radius-preview__surface--main,
.radius-preview--standard .radius-preview__surface--sub {
  border-radius: 12px;
}

.radius-preview--rounded .radius-preview__surface--main,
.radius-preview--rounded .radius-preview__surface--sub {
  border-radius: 20px;
}

.radius-preview--capsule .radius-preview__surface--main,
.radius-preview--capsule .radius-preview__surface--sub {
  border-radius: 999px;
}

.shadow-preview {
  display: grid;
  place-items: center;
  position: relative;
}

.shadow-preview__card {
  background: color-mix(in srgb, var(--td-bg-color-container) 94%, white 3%);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 86%, transparent);
  border-radius: 14px;
  display: block;
  position: absolute;
}

.shadow-preview__card--back {
  height: 34px;
  transform: translate(-10px, -8px);
  width: 56px;
}

.shadow-preview__card--front {
  height: 42px;
  transform: translate(10px, 8px);
  width: 74px;
}

.shadow-preview--flat .shadow-preview__card {
  box-shadow: none;
}

.shadow-preview--standard .shadow-preview__card--back {
  box-shadow: 0 6px 16px rgb(15 23 42 / 10%);
}

.shadow-preview--standard .shadow-preview__card--front {
  box-shadow: 0 12px 24px rgb(15 23 42 / 12%);
}

.shadow-preview--floating .shadow-preview__card--back {
  box-shadow: 0 10px 22px rgb(15 23 42 / 14%);
}

.shadow-preview--floating .shadow-preview__card--front {
  box-shadow: 0 18px 40px rgb(15 23 42 / 18%);
}

[theme-mode='dark'] .shadow-preview--standard .shadow-preview__card--back {
  box-shadow: 0 6px 18px rgb(0 0 0 / 26%);
}

[theme-mode='dark'] .shadow-preview--standard .shadow-preview__card--front {
  box-shadow: 0 14px 28px rgb(0 0 0 / 32%);
}

[theme-mode='dark'] .shadow-preview--floating .shadow-preview__card--back {
  box-shadow: 0 10px 26px rgb(0 0 0 / 34%);
}

[theme-mode='dark'] .shadow-preview--floating .shadow-preview__card--front {
  box-shadow: 0 18px 42px rgb(0 0 0 / 40%);
}

.density-preview {
  color: var(--td-text-color-secondary);
  display: grid;
  font: var(--td-font-body-small);
  grid-auto-rows: min-content;
  width: 100%;
}

.density-preview--compact {
  line-height: 1.16;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
}

.density-preview--standard {
  line-height: 1.42;
  padding: var(--graft-density-gap-12) var(--graft-density-gap-14);
}

.density-preview--comfortable {
  line-height: 1.5;
  padding: var(--graft-density-gap-14);
}

.density-preview__line {
  display: block;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.density-preview__line + .density-preview__line {
  margin-top: var(--graft-density-gap-2);
}

.density-preview--comfortable .density-preview__line + .density-preview__line {
  margin-top: var(--graft-density-gap-6);
}

.advanced-layout .advanced-settings-card {
  align-items: center;
  background: color-mix(in srgb, var(--td-bg-color-container) 96%, var(--td-bg-color-page));
  border: 1px solid var(--td-component-stroke);
  border-radius: 16px;
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: minmax(0, 1fr) auto;
  padding: var(--graft-density-gap-14) var(--graft-density-gap-16);
}

.advanced-layout .advanced-settings-card__content {
  display: grid;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.advanced-layout .advanced-settings-card__desc {
  color: color-mix(in srgb, var(--td-text-color-secondary) 90%, var(--td-text-color-primary));
}

.advanced-layout .advanced-mode-toolbar {
  align-items: center;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 16px;
  display: flex;
  gap: var(--graft-density-gap-14);
  justify-content: space-between;
  padding: var(--graft-density-gap-14) var(--graft-density-gap-16);
}

.advanced-layout .advanced-mode-toolbar__label {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 600;
}

.advanced-layout .advanced-sections {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.advanced-layout .advanced-section {
  display: grid;
  gap: var(--graft-density-gap-10);
}

.advanced-layout .advanced-section__title {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  font-weight: 700;
}

.advanced-layout .advanced-collapse {
  background: transparent;
  border: 0;
  display: grid;
  gap: var(--graft-density-gap-12);
}

.advanced-layout .advanced-collapse :deep(.t-collapse-panel) {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 16px;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
}

.advanced-layout .advanced-collapse :deep(.t-collapse-panel__wrapper) {
  border: 0;
}

.advanced-layout .advanced-collapse :deep(.t-collapse-panel__header) {
  background: transparent;
  border: 0;
  min-height: 58px;
  padding: var(--graft-density-gap-12) var(--graft-density-gap-14);
}

.advanced-layout .advanced-collapse :deep(.t-collapse-panel__header:hover) {
  background: color-mix(in srgb, var(--td-brand-color) 4%, var(--td-bg-color-container));
}

.advanced-layout .advanced-collapse :deep(.t-collapse-panel__icon--active) {
  color: var(--td-brand-color);
}

.advanced-layout .advanced-collapse :deep(.t-collapse-panel__body) {
  border-top: 1px solid var(--td-component-stroke);
}

.advanced-layout .advanced-collapse :deep(.t-collapse-panel__content) {
  padding: var(--graft-density-gap-14);
}

.advanced-layout .advanced-group__header {
  align-items: center;
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: auto minmax(0, 1fr) auto;
  min-width: 0;
  width: 100%;
}

.advanced-layout .advanced-group__icon {
  align-items: center;
  background: color-mix(in srgb, var(--td-brand-color) 10%, var(--td-bg-color-page));
  border: 1px solid color-mix(in srgb, var(--td-brand-color) 16%, var(--td-component-stroke));
  border-radius: 14px;
  color: var(--td-brand-color);
  display: inline-flex;
  font-size: 20px;
  height: 40px;
  justify-content: center;
  width: 40px;
}

.advanced-layout .advanced-group__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 700;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.advanced-layout .advanced-group__count {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  white-space: nowrap;
}

.overview-layout :deep(.preset-grid),
.settings-layout--appearance :deep(.preset-grid) {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.overview-layout :deep(.preset-card),
.settings-layout--appearance :deep(.preset-card) {
  gap: var(--graft-density-gap-10);
  padding: var(--graft-density-gap-12);
}

.overview-layout :deep(.preset-card__thumb-shell),
.settings-layout--appearance :deep(.preset-card__thumb-shell) {
  min-height: 112px;
}

.overview-layout :deep(.preset-card__thumbnail),
.settings-layout--appearance :deep(.preset-card__thumbnail) {
  padding: var(--graft-density-gap-8);
}

.overview-layout :deep(.preset-card__desc),
.settings-layout--appearance :deep(.preset-card__desc) {
  -webkit-box-orient: vertical;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-height: 1.45;
  overflow: hidden;
}

.theme-workbench-panel__footer {
  align-items: center;
  background: var(--td-bg-color-container);
  border-top: 1px solid var(--td-component-stroke);
  bottom: 0;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  left: 0;
  padding: var(--graft-density-gap-12) var(--graft-density-gap-16)
    calc(var(--graft-density-gap-12) + env(safe-area-inset-bottom, 0px));
  position: sticky;
  z-index: 2;
}

.theme-workbench-panel__footer-actions {
  display: flex;
  gap: var(--graft-density-gap-12);
}

.theme-workbench-panel__footer :deep(.t-button) {
  flex: 0 0 auto;
  white-space: nowrap;
}

@media (width <= 768px) {
  .theme-workbench-panel__body {
    grid-template-columns: 1fr;
    grid-template-rows: auto minmax(0, 1fr);
  }

  .theme-workbench-panel__nav {
    align-content: initial;
    display: flex;
    max-width: 100%;
    overflow: auto hidden;
    padding-bottom: var(--graft-density-gap-2);
    padding-right: 0;
  }

  .nav-item {
    flex: 0 0 64px;
    justify-content: center;
    min-height: 48px;
    min-width: 48px;
    padding: var(--graft-density-gap-8);
    width: 48px;
  }

  .nav-item__text {
    clip: rect(0 0 0 0);
    clip-path: inset(50%);
    height: 1px;
    overflow: hidden;
    position: absolute;
    white-space: nowrap;
    width: 1px;
  }

  .choice-grid,
  .settings-layout__section--mode .choice-grid,
  .settings-layout__section--nav .choice-grid,
  .settings-layout--layout .choice-grid,
  .style-preview-grid,
  .appearance-summary-grid,
  .overview-layout :deep(.preset-grid),
  .settings-layout--appearance :deep(.preset-grid) {
    grid-template-columns: 1fr;
  }

  .config-summary-card {
    grid-template-columns: 1fr;
  }

  .config-summary-row {
    gap: var(--graft-density-gap-4);
    grid-template-columns: 1fr;
  }

  .advanced-layout .advanced-settings-card,
  .advanced-layout .advanced-mode-toolbar {
    align-items: flex-start;
    flex-direction: column;
    grid-template-columns: 1fr;
  }

  .brand-input {
    grid-template-columns: 42px minmax(0, 1fr) 34px;
  }

  .font-size-control {
    grid-template-columns: auto minmax(0, 1fr) auto;
  }

  .font-size-control__select {
    grid-column: 1 / -1;
    width: 100%;
  }

  .font-size-control__marks {
    padding-inline: 0;
  }

  .font-size-preview {
    grid-template-columns: 1fr;
  }
}
</style>
