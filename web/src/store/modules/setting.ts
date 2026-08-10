import { defineStore } from 'pinia';

import type { TChartColor, TColorSeries } from '@/config/color';
import { DEFAULT_CHART_COLORS } from '@/config/color';
import STYLE_CONFIG from '@/config/style';
import {
  DEFAULT_THEME_PRESET_ID,
  GRAFT_BASE_THEME_TOKENS,
  THEME_PRESET_DEFINITIONS,
  THEME_TOKEN_DEFINITIONS,
  THEME_WORKBENCH_GROUPS,
  THEME_WORKBENCH_SCENARIO_PRESETS,
} from '@/config/theme-workbench';
import type {
  ThemeAuthorityDiffItem,
  ThemeAuthorityState,
  ThemeIdentitySummary,
  ThemeModeTokenState,
  ThemePresetApplicationScope,
  ThemePresetDefinition,
  ThemeTokenGroupKey,
  ThemeTokenMap,
  ThemeWorkbenchAuthorityPatch,
  ThemeWorkbenchGroupKey,
  ThemeWorkbenchStylePatch,
} from '@/types/theme';
import { composeThemeTokenMap, generateBrandColorMap, insertThemeStylesheet } from '@/utils/color';
import {
  buildThemeModeSnapshot,
  cloneThemeModeTokenState,
  createEmptyThemeModeTokenState,
  resolveModeTokens,
  resolvePresetId,
} from '@/utils/theme-workbench';
import type { ModeType } from '@/utils/types';

import {
  buildUserThemeTokens,
  countThemeTokenOverrides,
  createPersistedThemeAuthoritySnapshot,
  createThemeAuthoritySourceSnapshot,
  createWorkbenchStyleConfigSnapshot,
  hasThemeAuthorityStateDiff,
  hasThemeTokenOverrideDiff,
  hasWorkbenchStyleConfigDiff,
  type PersistedThemeAuthoritySource,
  STYLE_CONFIG_KEYS,
  THEME_AUTHORITY_DIFF_KEYS,
  WORKBENCH_STYLE_CONFIG_KEYS,
  type WorkbenchStyleConfigSnapshot,
} from './setting-theme-authority';
import {
  applyThemeWorkbenchDraft,
  beginThemeWorkbenchDraft,
  buildSelectedThemePresetState,
  buildThemeTokenGroupUpdate,
  buildThemeTokenValueUpdate,
  buildUpdatedThemeDraft,
  clearThemeTokenGroupOverrides,
  closeThemeWorkbenchDraft,
  createCompleteThemePresetState,
  openThemeWorkbenchDraft,
  previewThemeWorkbenchDraft,
  resetThemeWorkbenchDraftToDefault,
} from './setting-theme-draft';
import {
  buildChartColorsFromTokens,
  runThemeTransition,
  THEME_RESET_FEEDBACK_DURATION_MS,
} from './setting-theme-runtime';

export type SettingState = typeof STYLE_CONFIG &
  PersistedThemeAuthoritySource & {
    showSettingPanel: boolean;
    showThemeWorkbench: boolean;
    themeWorkbenchDockPosition: { xRatio: number; yRatio: number } | null;
    themeWorkbenchRuntimeReady: boolean;
    activeThemeWorkbenchGroup: ThemeWorkbenchGroupKey;
    activeThemeTokenGroup: ThemeTokenGroupKey;
    themeWorkbenchStyleConfigBaseline: WorkbenchStyleConfigSnapshot | null;
    themeDraftBaseline: ThemeAuthorityState | null;
    themeDraft: ThemeAuthorityState | null;
    themeDraftApplied: boolean;
    themeResetting: boolean;
    themeResetFeedbackKey: number;
    selectedThemePresetId: string | null;
    themeResolvedTokens: ThemeModeTokenState;
    themeAuthorityLastModifiedAt: string | null;
    colorList: TColorSeries;
    chartColors: TChartColor;
  };

function createInitialSettingState(): SettingState {
  const defaultPreset = THEME_PRESET_DEFINITIONS.find((item) => item.id === DEFAULT_THEME_PRESET_ID);
  const defaultThemeState = defaultPreset
    ? createCompleteThemePresetState(defaultPreset, STYLE_CONFIG.mode as ModeType | 'auto')
    : null;

  return {
    ...STYLE_CONFIG,
    showSettingPanel: false,
    showThemeWorkbench: false,
    themeWorkbenchDockPosition: null,
    themeWorkbenchRuntimeReady: false,
    activeThemeWorkbenchGroup: 'overview',
    activeThemeTokenGroup: 'brand',
    themeWorkbenchStyleConfigBaseline: null,
    themeDraftBaseline: null,
    themeDraft: null,
    themeDraftApplied: false,
    themeResetting: false,
    themeResetFeedbackKey: 0,
    mode: defaultThemeState?.mode ?? STYLE_CONFIG.mode,
    brandTheme: defaultThemeState?.brandTheme ?? STYLE_CONFIG.brandTheme,
    selectedThemePresetId: defaultThemeState?.selectedThemePresetId ?? DEFAULT_THEME_PRESET_ID,
    themeSource: defaultThemeState?.themeSource ?? 'preset',
    fontFamilyPreset: defaultThemeState?.fontFamilyPreset ?? 'system',
    fontSizePreset: defaultThemeState?.fontSizePreset ?? 'standard',
    radiusPreset: defaultThemeState?.radiusPreset ?? 'standard',
    radiusOverride: defaultThemeState?.radiusOverride ?? null,
    shadowPreset: defaultThemeState?.shadowPreset ?? 'standard',
    shadowIntensity: defaultThemeState?.shadowIntensity ?? 'standard',
    shadowIntensityOverride: defaultThemeState?.shadowIntensityOverride ?? null,
    densityPreset: defaultThemeState?.densityPreset ?? 'standard',
    densityOverride: defaultThemeState?.densityOverride ?? null,
    themeTokenOverrides: defaultThemeState?.themeTokenOverrides ?? createEmptyThemeModeTokenState(),
    themeResolvedTokens: createEmptyThemeModeTokenState(),
    themeAuthorityLastModifiedAt: null,
    colorList: {},
    chartColors: DEFAULT_CHART_COLORS,
  };
}

export type TState = SettingState;
export type TStateKey = keyof SettingState;

export const useSettingStore = defineStore('setting', {
  state: createInitialSettingState,
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
    themeWorkbenchScenarioPresets: () => THEME_WORKBENCH_SCENARIO_PRESETS,
    selectedThemePreset(state): ThemePresetDefinition | null {
      return THEME_PRESET_DEFINITIONS.find((item) => item.id === resolvePresetId(state.selectedThemePresetId)) ?? null;
    },
    effectiveThemeState(state): ThemeAuthorityState {
      return state.themeDraft ?? createPersistedThemeAuthoritySnapshot(state);
    },
    effectiveSelectedThemePreset(state): ThemePresetDefinition | null {
      const effectivePresetId = state.themeDraft?.selectedThemePresetId ?? state.selectedThemePresetId;
      return THEME_PRESET_DEFINITIONS.find((item) => item.id === resolvePresetId(effectivePresetId)) ?? null;
    },
    effectiveThemeDisplayNameKey(): string {
      const preset = this.effectiveSelectedThemePreset;
      if (!preset) {
        return 'layout.setting.workbench.presets.customized.label';
      }

      return preset.labelKey;
    },
    effectiveThemeSourceLabelKey(): string {
      const preset = this.effectiveSelectedThemePreset;
      return preset?.labelKey ?? 'layout.setting.workbench.presets.customized.label';
    },
    themeAuthorityDiff(state): ThemeAuthorityDiffItem[] {
      const persistedSnapshot = createPersistedThemeAuthoritySnapshot(state);
      const current = state.themeDraft ?? persistedSnapshot;
      const sourcePreset =
        THEME_PRESET_DEFINITIONS.find((item) => item.id === resolvePresetId(current.selectedThemePresetId)) ?? null;
      const baseline = createThemeAuthoritySourceSnapshot(sourcePreset, current);

      const presetDiffItems = THEME_AUTHORITY_DIFF_KEYS.flatMap((key) => {
        const fromValue = baseline[key];
        const toValue = current[key];

        if (fromValue === toValue) {
          return [];
        }

        return [
          {
            key,
            labelKey: `layout.setting.workbench.diff.${key}`,
            fromValue: String(fromValue),
            toValue: String(toValue),
          },
        ];
      });

      if (!hasThemeTokenOverrideDiff(baseline.themeTokenOverrides, current.themeTokenOverrides)) {
        return presetDiffItems;
      }

      return [
        ...presetDiffItems,
        {
          key: 'themeTokenOverrides',
          labelKey: 'layout.setting.workbench.diff.themeTokenOverrides',
          fromValue: String(countThemeTokenOverrides(baseline.themeTokenOverrides)),
          toValue: String(countThemeTokenOverrides(current.themeTokenOverrides)),
        },
      ];
    },
    themeIdentitySummary(): ThemeIdentitySummary {
      return {
        currentLabelKey: this.effectiveThemeDisplayNameKey,
        sourceLabelKey: this.effectiveThemeSourceLabelKey,
        sourceType: this.effectiveThemeState.themeSource,
        modifiedCount: this.themeAuthorityDiff.length,
        lastModifiedAt: this.themeAuthorityLastModifiedAt,
      };
    },
    resolvedThemeTokensForDisplayMode(): ThemeTokenMap {
      return resolveModeTokens(this.themeResolvedTokens, this.displayMode);
    },
    hasThemeDraftPendingChanges(state): boolean {
      if (!state.themeDraft) {
        return false;
      }

      const baseline = state.themeDraftBaseline ?? createPersistedThemeAuthoritySnapshot(state);
      return hasThemeAuthorityStateDiff(baseline, state.themeDraft);
    },
    hasThemeWorkbenchPendingChanges(state): boolean {
      const themeHasChanges = this.hasThemeDraftPendingChanges;
      const baseline = state.themeWorkbenchStyleConfigBaseline;

      if (!baseline) {
        return themeHasChanges;
      }

      return themeHasChanges || hasWorkbenchStyleConfigDiff(baseline, createWorkbenchStyleConfigSnapshot(state));
    },
  },
  actions: {
    createThemeAuthoritySnapshot(): ThemeAuthorityState {
      return {
        mode: this.mode as ModeType | 'auto',
        brandTheme: this.brandTheme,
        selectedThemePresetId: this.selectedThemePresetId,
        themeSource: this.themeSource,
        fontFamilyPreset: this.fontFamilyPreset,
        fontSizePreset: this.fontSizePreset,
        radiusPreset: this.radiusPreset,
        radiusOverride: this.radiusOverride,
        shadowPreset: this.shadowPreset,
        shadowIntensity: this.shadowIntensity,
        shadowIntensityOverride: this.shadowIntensityOverride,
        densityPreset: this.densityPreset,
        densityOverride: this.densityOverride,
        themeTokenOverrides: cloneThemeModeTokenState(this.themeTokenOverrides),
      };
    },
    assignThemeAuthorityState(nextState: ThemeAuthorityState) {
      this.mode = nextState.mode;
      this.brandTheme = nextState.brandTheme;
      this.selectedThemePresetId = nextState.selectedThemePresetId;
      this.themeSource = nextState.themeSource;
      this.fontFamilyPreset = nextState.fontFamilyPreset;
      this.fontSizePreset = nextState.fontSizePreset;
      this.radiusPreset = nextState.radiusPreset;
      this.radiusOverride = nextState.radiusOverride;
      this.shadowPreset = nextState.shadowPreset;
      this.shadowIntensity = nextState.shadowIntensity;
      this.shadowIntensityOverride = nextState.shadowIntensityOverride;
      this.densityPreset = nextState.densityPreset;
      this.densityOverride = nextState.densityOverride;
      this.themeTokenOverrides = cloneThemeModeTokenState(nextState.themeTokenOverrides);
    },
    createWorkbenchStyleConfigSnapshot(): WorkbenchStyleConfigSnapshot {
      return createWorkbenchStyleConfigSnapshot(this);
    },
    assignWorkbenchStyleConfigSnapshot(snapshot: WorkbenchStyleConfigSnapshot) {
      WORKBENCH_STYLE_CONFIG_KEYS.forEach((key) => {
        this[key] = snapshot[key] as never;
      });
      this.changeSideMode(this.sideMode as ModeType);
    },
    markThemeCustomized() {
      this.themeSource = 'customized';
    },
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
      const brandTokens: ThemeModeTokenState = {
        light: this.getCachedBrandTokens(this.brandTheme, 'light'),
        dark: this.getCachedBrandTokens(this.brandTheme, 'dark'),
      };
      const userTokens = buildUserThemeTokens(this.createThemeAuthoritySnapshot());
      this.themeResolvedTokens = buildThemeModeSnapshot({
        baseTokens: GRAFT_BASE_THEME_TOKENS,
        brandTokens,
        userTokens,
        customTokens: this.themeTokenOverrides,
      });
    },
    applyResolvedThemeTokens(mode: ModeType) {
      const resolvedTokens = resolveModeTokens(this.themeResolvedTokens, mode);
      const tokenMap = composeThemeTokenMap(resolvedTokens);
      insertThemeStylesheet(this.brandTheme, tokenMap, mode);
      document.documentElement.setAttribute('theme-color', this.brandTheme);
      document.documentElement.toggleAttribute(
        'data-graft-hard-surface',
        this.radiusPreset === 'square' && this.radiusOverride === null && this.shadowPreset === 'hard-offset',
      );
    },
    refreshThemeWorkbenchRuntime(mode?: ModeType | 'auto') {
      const nextMode = mode ?? (this.mode as ModeType | 'auto');
      const displayMode = this.getDisplayModeByInput(nextMode);
      this.buildResolvedThemeTokens();
      this.applyResolvedThemeTokens(displayMode);
      this.chartColors = buildChartColorsFromTokens(resolveModeTokens(this.themeResolvedTokens, displayMode));
    },
    async changeMode(mode: ModeType | 'auto') {
      const theme = this.getDisplayModeByInput(mode);
      const isDarkMode = theme === 'dark';

      document.documentElement.setAttribute('theme-mode', isDarkMode ? 'dark' : '');

      this.refreshThemeWorkbenchRuntime(theme);
    },
    async changeModeWithTransition(mode: ModeType | 'auto', event?: MouseEvent) {
      await runThemeTransition(() => {
        this.changeMode(mode);
      }, event);
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
      this.brandTheme = brandTheme;
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
    setThemeWorkbenchDockPosition(position: { xRatio: number; yRatio: number }) {
      const xRatio = Math.min(1, Math.max(0, position.xRatio));
      const yRatio = Math.min(1, Math.max(0, position.yRatio));
      this.themeWorkbenchDockPosition = { xRatio, yRatio };
    },
    resetThemeWorkbenchDockPosition() {
      this.themeWorkbenchDockPosition = null;
    },
    openThemeWorkbench(group?: ThemeWorkbenchGroupKey) {
      openThemeWorkbenchDraft(this, group);
    },
    closeThemeWorkbench() {
      closeThemeWorkbenchDraft(this);
    },
    setActiveThemeWorkbenchGroup(group: ThemeWorkbenchGroupKey) {
      this.activeThemeWorkbenchGroup = group;
    },
    beginThemeDraft() {
      beginThemeWorkbenchDraft(this);
    },
    applyThemeDraftPreview() {
      previewThemeWorkbenchDraft(this);
    },
    updateThemeDraft(patch: Partial<ThemeAuthorityState>) {
      const base = this.themeDraft ?? this.createThemeAuthoritySnapshot();
      this.themeDraft = buildUpdatedThemeDraft(base, patch);
      this.applyThemeDraftPreview();
    },
    applyThemeDraft() {
      applyThemeWorkbenchDraft(this);
    },
    cancelThemeDraft() {
      this.closeThemeWorkbench();
    },
    resetThemeDraftToDefault(options: { preserveResettingFeedback?: boolean } = {}) {
      const defaultPreset = THEME_PRESET_DEFINITIONS.find((item) => item.id === DEFAULT_THEME_PRESET_ID);
      if (!defaultPreset) {
        return;
      }
      resetThemeWorkbenchDraftToDefault(this, defaultPreset, options);
      this.updateConfig(defaultPreset.stylePatch ?? {});
    },
    async resetDefaultThemeWithFeedback() {
      const feedbackKey = this.themeResetFeedbackKey + 1;

      this.themeResetting = true;
      this.themeResetFeedbackKey = feedbackKey;
      this.resetThemeDraftToDefault({ preserveResettingFeedback: true });

      await new Promise((resolve) => {
        window.setTimeout(resolve, THEME_RESET_FEEDBACK_DURATION_MS);
      });

      if (this.themeResetFeedbackKey === feedbackKey) {
        this.themeResetting = false;
      }
    },
    selectThemePreset(presetId: string | null, scope: ThemePresetApplicationScope = 'palette') {
      const resolvedPresetId = resolvePresetId(presetId);
      const preset = THEME_PRESET_DEFINITIONS.find((item) => item.id === resolvedPresetId);
      const currentPresetId = this.themeDraft?.selectedThemePresetId ?? this.selectedThemePresetId;
      const currentPreset =
        THEME_PRESET_DEFINITIONS.find((item) => item.id === resolvePresetId(currentPresetId)) ?? null;

      if (!preset) {
        return;
      }

      const nextState = buildSelectedThemePresetState(
        preset,
        this.themeDraft,
        this.createThemeAuthoritySnapshot(),
        scope,
        currentPreset,
      );
      this.updateThemeDraft(nextState);
      if (scope === 'complete' && preset.stylePatch) {
        this.updateConfig(preset.stylePatch);
      }
    },
    applyWorkbenchQuickAppearance(patch: ThemeWorkbenchAuthorityPatch) {
      this.updateThemeDraftAppearance(patch);
    },
    applyWorkbenchQuickLayout(patch: ThemeWorkbenchStylePatch) {
      this.updateConfig(patch);
    },
    applyThemeWorkbenchScenarioPreset(presetId: string) {
      const preset = THEME_WORKBENCH_SCENARIO_PRESETS.find((item) => item.id === presetId);

      if (!preset) {
        return;
      }

      if (preset.presetId) {
        this.selectThemePreset(preset.presetId);
      }

      if (preset.authorityPatch) {
        const nextPatch: ThemeWorkbenchAuthorityPatch = { ...preset.authorityPatch };
        this.updateThemeDraftAppearance(nextPatch);
      }

      if (preset.stylePatch) {
        this.updateConfig(preset.stylePatch);
      }
    },
    setCustomBrandTheme(brandTheme: string) {
      this.updateThemeDraft({
        brandTheme,
        themeSource: 'customized',
      });
    },
    updateThemeDraftAppearance(
      patch: Partial<
        Pick<
          ThemeAuthorityState,
          | 'mode'
          | 'fontFamilyPreset'
          | 'fontSizePreset'
          | 'radiusPreset'
          | 'radiusOverride'
          | 'shadowPreset'
          | 'shadowIntensity'
          | 'shadowIntensityOverride'
          | 'densityPreset'
          | 'densityOverride'
        >
      >,
    ) {
      const nextPatch: Partial<ThemeAuthorityState> = {
        ...patch,
        themeSource: 'customized',
      };
      this.updateThemeDraft(nextPatch);
    },
    async updateThemeDraftModeWithTransition(mode: ModeType | 'auto', event?: MouseEvent) {
      const base = this.themeDraft ?? this.createThemeAuthoritySnapshot();
      this.themeDraft = {
        ...base,
        mode,
        themeSource: 'customized',
        themeTokenOverrides: cloneThemeModeTokenState(base.themeTokenOverrides),
      };
      await runThemeTransition(() => {
        if (!this.themeDraft) {
          return;
        }

        this.assignThemeAuthorityState(this.themeDraft);
        this.changeMode(this.mode as ModeType | 'auto');
      }, event);
      this.themeDraftApplied = true;
    },
    updateThemeToken(mode: ModeType, tokenKey: string, tokenValue: string) {
      const baseState = this.themeDraft ?? this.createThemeAuthoritySnapshot();
      this.updateThemeDraft(buildThemeTokenValueUpdate(baseState, mode, tokenKey, tokenValue));
    },
    updateThemeTokenGroup(mode: ModeType, tokenGroup: ThemeTokenMap) {
      const baseState = this.themeDraft ?? this.createThemeAuthoritySnapshot();
      this.updateThemeDraft(buildThemeTokenGroupUpdate(baseState, mode, tokenGroup));
    },
    clearThemeTokenGroup(mode: ModeType, tokenKeys?: string[]) {
      const baseState = this.themeDraft ?? this.createThemeAuthoritySnapshot();
      this.updateThemeDraft(clearThemeTokenGroupOverrides(baseState, mode, tokenKeys));
    },
    resetThemeWorkbench() {
      this.activeThemeWorkbenchGroup = 'overview';
      this.activeThemeTokenGroup = 'brand';
      this.beginThemeDraft();
      this.resetThemeDraftToDefault();
    },
    initializeThemeWorkbenchRuntime() {
      if (this.themeWorkbenchRuntimeReady) {
        return;
      }

      this.selectedThemePresetId = resolvePresetId(this.selectedThemePresetId);
      this.themeTokenOverrides = cloneThemeModeTokenState(this.themeTokenOverrides);
      this.themeResolvedTokens = cloneThemeModeTokenState(this.themeResolvedTokens);
      if (this.layout === 'top') {
        this.menuAlwaysExpanded = false;
      }
      this.changeMode(this.mode as ModeType | 'auto');
      this.changeSideMode(this.sideMode as ModeType);
      this.themeWorkbenchRuntimeReady = true;
    },
    updateConfig(payload: Partial<TState>) {
      const normalizedPayload = { ...payload };

      if (normalizedPayload.menuAlwaysExpanded === true) {
        normalizedPayload.menuAutoCollapsed = false;
      }
      if (normalizedPayload.menuAutoCollapsed === true) {
        normalizedPayload.menuAlwaysExpanded = false;
      }
      if (normalizedPayload.layout === 'top') {
        normalizedPayload.menuAlwaysExpanded = false;
      }

      for (const key in normalizedPayload) {
        const stateKey = key as TStateKey;

        if (normalizedPayload[stateKey] !== undefined) {
          if (stateKey === 'showSettingPanel' || stateKey === 'showThemeWorkbench') {
            this.setThemeWorkbenchVisible(Boolean(normalizedPayload[stateKey]));
            continue;
          }

          this[stateKey] = normalizedPayload[stateKey] as never;
        }
        if (key === 'mode') {
          this.changeMode(normalizedPayload[stateKey] as ModeType);
        }
        if (key === 'sideMode') {
          this.changeSideMode(normalizedPayload[stateKey] as ModeType);
        }
        if (key === 'brandTheme') {
          this.changeBrandTheme(normalizedPayload[stateKey] as string);
        }
      }
    },
  },
  persist: {
    pick: [
      ...STYLE_CONFIG_KEYS,
      'colorList',
      'chartColors',
      'themeAuthorityLastModifiedAt',
      'selectedThemePresetId',
      'themeSource',
      'fontFamilyPreset',
      'fontSizePreset',
      'radiusPreset',
      'radiusOverride',
      'shadowPreset',
      'shadowIntensity',
      'shadowIntensityOverride',
      'densityPreset',
      'densityOverride',
      'themeTokenOverrides',
      'themeResolvedTokens',
      'activeThemeWorkbenchGroup',
      'activeThemeTokenGroup',
      'themeWorkbenchDockPosition',
    ],
  },
});
