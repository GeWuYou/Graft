import keys from 'lodash/keys';
import { defineStore } from 'pinia';

import type { TColorSeries } from '@/config/color';
import { DARK_CHART_COLORS, LIGHT_CHART_COLORS } from '@/config/color';
import STYLE_CONFIG from '@/config/style';
import {
  DEFAULT_THEME_PRESET_ID,
  THEME_PRESET_DEFINITIONS,
  THEME_TOKEN_DEFINITIONS,
  THEME_WORKBENCH_GROUPS,
} from '@/config/theme-workbench';
import { store } from '@/store';
import type {
  ThemeModeTokenState,
  ThemePresetDefinition,
  ThemeSourceType,
  ThemeTokenGroupKey,
  ThemeTokenMap,
  ThemeWorkbenchGroupKey,
} from '@/types/theme';
import { composeThemeTokenMap, generateBrandColorMap, insertThemeStylesheet } from '@/utils/color';
import {
  buildThemeConfigCopyPayload,
  buildThemeModeSnapshot,
  cloneThemeModeTokenState,
  createEmptyThemeModeTokenState,
  resolveModeTokens,
  resolvePresetId,
  stringifyThemeConfigCopyPayload,
} from '@/utils/theme-workbench';
import type { ModeType } from '@/utils/types';

const STYLE_CONFIG_KEYS = keys(STYLE_CONFIG) as Array<keyof typeof STYLE_CONFIG>;

export type SettingState = typeof STYLE_CONFIG & {
  showSettingPanel: boolean;
  showThemeWorkbench: boolean;
  themeWorkbenchRuntimeReady: boolean;
  activeThemeWorkbenchGroup: ThemeWorkbenchGroupKey;
  activeThemeTokenGroup: ThemeTokenGroupKey;
  selectedThemePresetId: string | null;
  themeSource: ThemeSourceType;
  themeTokenOverrides: ThemeModeTokenState;
  themeResolvedTokens: ThemeModeTokenState;
  colorList: TColorSeries;
  chartColors: typeof LIGHT_CHART_COLORS;
};

const state: SettingState = {
  ...STYLE_CONFIG,
  showSettingPanel: false,
  showThemeWorkbench: false,
  themeWorkbenchRuntimeReady: false,
  activeThemeWorkbenchGroup: 'overview',
  activeThemeTokenGroup: 'brand',
  selectedThemePresetId: DEFAULT_THEME_PRESET_ID,
  themeSource: 'preset',
  themeTokenOverrides: createEmptyThemeModeTokenState(),
  themeResolvedTokens: createEmptyThemeModeTokenState(),
  colorList: {},
  chartColors: LIGHT_CHART_COLORS,
};

export type TState = SettingState;
export type TStateKey = keyof SettingState;

export const useSettingStore = defineStore('setting', {
  state: () => state,
  getters: {
    showSidebar: (state) => state.layout !== 'top',
    showSidebarLogo: (state) => state.layout === 'side',
    showHeaderLogo: (state) => state.layout !== 'side',
    displayMode: (state): ModeType => {
      if (state.mode === 'auto') {
        const media = window.matchMedia('(prefers-color-scheme:dark)');
        if (media.matches) {
          return 'dark';
        }
        return 'light';
      }
      return state.mode as ModeType;
    },
    displaySideMode: (state): ModeType => {
      return state.sideMode as ModeType;
    },
    themeWorkbenchGroups: () => THEME_WORKBENCH_GROUPS,
    themeTokenDefinitions: () => THEME_TOKEN_DEFINITIONS,
    themePresetDefinitions: () => THEME_PRESET_DEFINITIONS,
    selectedThemePreset(state): ThemePresetDefinition | null {
      return THEME_PRESET_DEFINITIONS.find((item) => item.id === resolvePresetId(state.selectedThemePresetId)) ?? null;
    },
    resolvedThemeTokensForDisplayMode(state): ThemeTokenMap {
      return resolveModeTokens(state.themeResolvedTokens, state.mode === 'dark' ? 'dark' : 'light');
    },
    themeConfigCopyPayload(state) {
      const styleConfig = STYLE_CONFIG_KEYS.reduce<Partial<typeof STYLE_CONFIG>>((config, key) => {
        config[key] = state[key] as never;
        return config;
      }, {});

      return buildThemeConfigCopyPayload({
        styleConfig,
        activeGroup: state.activeThemeWorkbenchGroup,
        selectedPresetId: state.selectedThemePresetId,
        source: state.themeSource,
        customTokens: state.themeTokenOverrides,
        resolvedTokens: state.themeResolvedTokens,
      });
    },
    themeConfigCopyText(): string {
      return stringifyThemeConfigCopyPayload(this.themeConfigCopyPayload);
    },
  },
  actions: {
    getDisplayModeByInput(mode: ModeType | 'auto') {
      return mode === 'auto' ? this.getMediaColor() : mode;
    },
    getCachedBrandTokens(brandTheme: string, mode: ModeType) {
      const colorKey = `${brandTheme}[${mode}]`;
      const cached = this.colorList[colorKey];

      if (cached) {
        return cached;
      }

      const colorMap = generateBrandColorMap(brandTheme, mode);
      this.colorList[colorKey] = colorMap;
      return colorMap;
    },
    buildResolvedThemeTokens() {
      const preset =
        THEME_PRESET_DEFINITIONS.find((item) => item.id === resolvePresetId(this.selectedThemePresetId)) ?? null;
      const brandTokens: ThemeModeTokenState = {
        light: this.getCachedBrandTokens(this.brandTheme, 'light'),
        dark: this.getCachedBrandTokens(this.brandTheme, 'dark'),
      };

      this.themeResolvedTokens = buildThemeModeSnapshot({
        brandTokens,
        preset,
        customTokens: this.themeTokenOverrides,
      });
    },
    applyResolvedThemeTokens(mode: ModeType) {
      const resolvedTokens = resolveModeTokens(this.themeResolvedTokens, mode);
      const tokenMap = composeThemeTokenMap(resolvedTokens);
      insertThemeStylesheet(this.brandTheme, tokenMap, mode);
      document.documentElement.setAttribute('theme-color', this.brandTheme);
    },
    refreshThemeWorkbenchRuntime(mode?: ModeType | 'auto') {
      const nextMode = mode ?? (this.mode as ModeType | 'auto');
      const displayMode = this.getDisplayModeByInput(nextMode);
      this.buildResolvedThemeTokens();
      this.applyResolvedThemeTokens(displayMode);
    },
    async changeMode(mode: ModeType | 'auto') {
      const theme = this.getDisplayModeByInput(mode);
      const isDarkMode = theme === 'dark';

      document.documentElement.setAttribute('theme-mode', isDarkMode ? 'dark' : '');

      this.chartColors = isDarkMode ? DARK_CHART_COLORS : LIGHT_CHART_COLORS;
      this.refreshThemeWorkbenchRuntime(theme);
    },
    async changeSideMode(mode: ModeType) {
      const isDarkMode = mode === 'dark';

      document.documentElement.setAttribute('side-mode', isDarkMode ? 'dark' : '');
    },
    getMediaColor() {
      const media = window.matchMedia('(prefers-color-scheme:dark)');

      if (media.matches) {
        return 'dark';
      }
      return 'light';
    },
    changeBrandTheme(brandTheme: string) {
      const mode = this.displayMode;
      this.getCachedBrandTokens(brandTheme, 'light');
      this.getCachedBrandTokens(brandTheme, 'dark');
      this.refreshThemeWorkbenchRuntime(mode);
      document.documentElement.setAttribute('theme-color', brandTheme);
    },
    syncThemeWorkbenchVisibility(visible: boolean) {
      // 旧 showSettingPanel 仅保留给尚未迁移的壳层读取，真实来源收口到 showThemeWorkbench。
      this.showThemeWorkbench = visible;
      this.showSettingPanel = visible;
    },
    setThemeWorkbenchVisible(visible: boolean) {
      this.syncThemeWorkbenchVisibility(visible);
    },
    openThemeWorkbench(group?: ThemeWorkbenchGroupKey) {
      this.syncThemeWorkbenchVisibility(true);
      if (group) {
        this.activeThemeWorkbenchGroup = group;
        if (group !== 'overview') {
          this.activeThemeTokenGroup = group;
        }
      }
    },
    closeThemeWorkbench() {
      this.syncThemeWorkbenchVisibility(false);
    },
    setActiveThemeWorkbenchGroup(group: ThemeWorkbenchGroupKey) {
      this.activeThemeWorkbenchGroup = group;
      if (group !== 'overview') {
        this.activeThemeTokenGroup = group;
      }
    },
    selectThemePreset(presetId: string | null) {
      const resolvedPresetId = resolvePresetId(presetId);
      const preset = THEME_PRESET_DEFINITIONS.find((item) => item.id === resolvedPresetId);

      if (!preset) {
        return;
      }

      this.selectedThemePresetId = preset.id;
      this.themeSource = 'preset';
      this.themeTokenOverrides = createEmptyThemeModeTokenState();
      this.brandTheme = preset.brandTheme;
      if (preset.mode) {
        this.mode = preset.mode;
        this.changeMode(preset.mode);
      } else {
        this.refreshThemeWorkbenchRuntime();
      }
    },
    setCustomBrandTheme(brandTheme: string) {
      this.selectedThemePresetId = null;
      this.themeSource = 'custom';
      this.brandTheme = brandTheme;
      this.changeBrandTheme(brandTheme);
    },
    updateThemeToken(mode: ModeType, tokenKey: string, tokenValue: string) {
      this.themeSource = 'custom';
      this.themeTokenOverrides = {
        ...this.themeTokenOverrides,
        [mode]: {
          ...this.themeTokenOverrides[mode],
          [tokenKey]: tokenValue,
        },
      };
      this.refreshThemeWorkbenchRuntime();
    },
    updateThemeTokenGroup(mode: ModeType, tokenGroup: ThemeTokenMap) {
      this.themeSource = 'custom';
      this.themeTokenOverrides = {
        ...this.themeTokenOverrides,
        [mode]: {
          ...this.themeTokenOverrides[mode],
          ...tokenGroup,
        },
      };
      this.refreshThemeWorkbenchRuntime();
    },
    clearThemeTokenGroup(mode: ModeType, tokenKeys?: string[]) {
      const nextTokens = { ...this.themeTokenOverrides[mode] };

      if (!tokenKeys?.length) {
        this.themeTokenOverrides = {
          ...this.themeTokenOverrides,
          [mode]: {},
        };
      } else {
        tokenKeys.forEach((tokenKey) => {
          delete nextTokens[tokenKey];
        });
        this.themeTokenOverrides = {
          ...this.themeTokenOverrides,
          [mode]: nextTokens,
        };
      }

      this.themeSource =
        Object.keys(this.themeTokenOverrides.light).length || Object.keys(this.themeTokenOverrides.dark).length
          ? 'custom'
          : 'preset';
      this.refreshThemeWorkbenchRuntime();
    },
    resetThemeWorkbench() {
      this.closeThemeWorkbench();
      this.activeThemeWorkbenchGroup = 'overview';
      this.activeThemeTokenGroup = 'brand';
      this.selectedThemePresetId = DEFAULT_THEME_PRESET_ID;
      this.themeSource = 'preset';
      this.themeTokenOverrides = createEmptyThemeModeTokenState();
      this.mode = STYLE_CONFIG.mode;
      this.sideMode = STYLE_CONFIG.sideMode;
      this.layout = STYLE_CONFIG.layout;
      this.showHeader = STYLE_CONFIG.showHeader;
      this.showBreadcrumb = STYLE_CONFIG.showBreadcrumb;
      this.showFooter = STYLE_CONFIG.showFooter;
      this.isUseTabsRouter = STYLE_CONFIG.isUseTabsRouter;
      this.menuAutoCollapsed = STYLE_CONFIG.menuAutoCollapsed;
      this.brandTheme = STYLE_CONFIG.brandTheme;
      this.refreshThemeWorkbenchRuntime();
      this.changeMode(this.mode as ModeType | 'auto');
      this.changeSideMode(this.sideMode as ModeType);
    },
    initializeThemeWorkbenchRuntime() {
      if (this.themeWorkbenchRuntimeReady) {
        return;
      }

      this.selectedThemePresetId = resolvePresetId(this.selectedThemePresetId);
      this.themeTokenOverrides = cloneThemeModeTokenState(this.themeTokenOverrides);
      this.themeResolvedTokens = cloneThemeModeTokenState(this.themeResolvedTokens);
      this.changeMode(this.mode as ModeType | 'auto');
      this.changeSideMode(this.sideMode as ModeType);
      this.themeWorkbenchRuntimeReady = true;
    },
    updateConfig(payload: Partial<TState>) {
      for (const key in payload) {
        const stateKey = key as TStateKey;

        if (payload[stateKey] !== undefined) {
          if (stateKey === 'showSettingPanel' || stateKey === 'showThemeWorkbench') {
            this.setThemeWorkbenchVisible(Boolean(payload[stateKey]));
            continue;
          }

          this[stateKey] = payload[stateKey] as never;
        }
        if (key === 'mode') {
          this.changeMode(payload[stateKey] as ModeType);
        }
        if (key === 'sideMode') {
          this.changeSideMode(payload[stateKey] as ModeType);
        }
        if (key === 'brandTheme') {
          this.changeBrandTheme(payload[stateKey] as string);
        }
      }
    },
  },
  persist: {
    pick: [
      ...STYLE_CONFIG_KEYS,
      'colorList',
      'chartColors',
      'selectedThemePresetId',
      'themeSource',
      'themeTokenOverrides',
      'themeResolvedTokens',
      'activeThemeWorkbenchGroup',
      'activeThemeTokenGroup',
    ],
  },
});

export function getSettingStore() {
  return useSettingStore(store);
}
