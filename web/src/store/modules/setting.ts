// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import keys from 'lodash/keys';
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
} from '@/config/theme-workbench';
import type {
  ThemeAuthorityDiffItem,
  ThemeAuthorityState,
  ThemeIdentitySummary,
  ThemeModeTokenState,
  ThemePresetDefinition,
  ThemeSourceType,
  ThemeTokenGroupKey,
  ThemeTokenMap,
  ThemeWorkbenchGroupKey,
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

const STYLE_CONFIG_KEYS = keys(STYLE_CONFIG) as Array<keyof typeof STYLE_CONFIG>;
const THEME_AUTHORITY_STYLE_KEYS = ['mode', 'brandTheme'] as const;
const WORKBENCH_STYLE_CONFIG_KEYS = STYLE_CONFIG_KEYS.filter(
  (key) => !(THEME_AUTHORITY_STYLE_KEYS as readonly string[]).includes(key),
);
const THEME_TRANSITION_DURATION_MS = 420;
const THEME_TRANSITION_EASING = 'cubic-bezier(0.4, 0, 0.2, 1)';
const THEME_VIEW_TRANSITION_CLASS = 'graft-theme-view-transition';
const THEME_CSS_TRANSITION_CLASS = 'graft-theme-css-transition';
const THEME_RESET_FEEDBACK_DURATION_MS = 640;

type ThemeViewTransition = {
  ready: Promise<void>;
  finished: Promise<void>;
};

type ThemeViewTransitionDocument = Document & {
  startViewTransition?: (callback: () => void) => ThemeViewTransition;
};

function prefersReducedMotion() {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

function resolveThemeTransitionOrigin(event?: MouseEvent) {
  const x = event?.clientX ?? window.innerWidth;
  const y = event?.clientY ?? 0;

  return { x, y };
}

async function runThemeCssFallbackTransition(applyThemeChange: () => void) {
  const root = document.documentElement;

  root.classList.add(THEME_CSS_TRANSITION_CLASS);
  applyThemeChange();

  await new Promise((resolve) => {
    window.setTimeout(resolve, THEME_TRANSITION_DURATION_MS);
  });

  root.classList.remove(THEME_CSS_TRANSITION_CLASS);
}

async function runThemeViewTransition(applyThemeChange: () => void, event?: MouseEvent) {
  const transitionDocument = document as ThemeViewTransitionDocument;

  if (!transitionDocument.startViewTransition || prefersReducedMotion()) {
    applyThemeChange();
    return;
  }

  const { x, y } = resolveThemeTransitionOrigin(event);
  const endRadius = Math.hypot(Math.max(x, window.innerWidth - x), Math.max(y, window.innerHeight - y));
  const root = document.documentElement;

  root.classList.add(THEME_VIEW_TRANSITION_CLASS);

  const transition = transitionDocument.startViewTransition(() => {
    applyThemeChange();
  });

  try {
    await transition.ready;
    root.animate(
      {
        clipPath: [`circle(0px at ${x}px ${y}px)`, `circle(${endRadius}px at ${x}px ${y}px)`],
      },
      {
        duration: THEME_TRANSITION_DURATION_MS,
        easing: THEME_TRANSITION_EASING,
        pseudoElement: '::view-transition-new(root)',
      },
    );
    await transition.finished;
  } finally {
    root.classList.remove(THEME_VIEW_TRANSITION_CLASS);
  }
}

async function runThemeTransition(applyThemeChange: () => void, event?: MouseEvent) {
  const transitionDocument = document as ThemeViewTransitionDocument;

  if (prefersReducedMotion()) {
    applyThemeChange();
    return;
  }

  if (!transitionDocument.startViewTransition) {
    await runThemeCssFallbackTransition(applyThemeChange);
    return;
  }

  await runThemeViewTransition(applyThemeChange, event);
}

const FONT_FAMILY_MAP: Record<ThemeAuthorityState['fontFamilyPreset'], string> = {
  system: '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif',
  harmonyos: '"HarmonyOS Sans SC", "HarmonyOS Sans", "PingFang SC", "Microsoft YaHei", sans-serif',
  inter: '"Inter", "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif',
  'source-han-sans': '"Source Han Sans SC", "Noto Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif',
};

const RADIUS_PRESET_MAP: Record<ThemeAuthorityState['radiusPreset'], ThemeTokenMap> = {
  business: {
    '--td-radius-small': '4px',
    '--td-radius-default': '4px',
    '--td-radius-medium': '6px',
    '--td-radius-large': '8px',
    '--td-radius-extraLarge': '10px',
    '--td-radius-circle': '999px',
  },
  standard: {
    '--td-radius-small': '6px',
    '--td-radius-default': '8px',
    '--td-radius-medium': '10px',
    '--td-radius-large': '12px',
    '--td-radius-extraLarge': '14px',
    '--td-radius-circle': '999px',
  },
  rounded: {
    '--td-radius-small': '8px',
    '--td-radius-default': '12px',
    '--td-radius-medium': '14px',
    '--td-radius-large': '16px',
    '--td-radius-extraLarge': '18px',
    '--td-radius-circle': '999px',
  },
  capsule: {
    '--td-radius-small': '10px',
    '--td-radius-default': '16px',
    '--td-radius-medium': '18px',
    '--td-radius-large': '20px',
    '--td-radius-extraLarge': '24px',
    '--td-radius-circle': '999px',
  },
};

const SHADOW_PRESET_MAP: Record<ThemeAuthorityState['shadowPreset'], ThemeTokenMap> = {
  flat: {
    '--td-shadow-1': 'none',
    '--td-shadow-2': 'none',
    '--td-shadow-3': 'none',
  },
  standard: {
    '--td-shadow-1': '0 2px 10px rgba(15, 23, 42, 0.08)',
    '--td-shadow-2': '0 10px 24px rgba(15, 23, 42, 0.12)',
    '--td-shadow-3': '0 18px 42px rgba(15, 23, 42, 0.18)',
  },
  floating: {
    '--td-shadow-1': '0 6px 16px rgba(15, 23, 42, 0.12)',
    '--td-shadow-2': '0 16px 36px rgba(15, 23, 42, 0.18)',
    '--td-shadow-3': '0 24px 56px rgba(15, 23, 42, 0.24)',
  },
};

const BASE_DENSITY_TOKENS = {
  '--td-comp-size-xs': 24,
  '--td-comp-size-s': 28,
  '--td-comp-size-m': 32,
  '--td-comp-size-l': 36,
  '--td-comp-size-xl': 40,
  '--td-comp-paddingTB-s': 4,
  '--td-comp-paddingTB-m': 6,
  '--td-comp-paddingTB-l': 8,
  '--td-comp-paddingTB-xl': 12,
  '--td-comp-paddingLR-s': 8,
  '--td-comp-paddingLR-m': 12,
  '--td-comp-paddingLR-l': 16,
  '--td-comp-paddingLR-xl': 20,
  '--td-comp-margin-xs': 4,
  '--td-comp-margin-s': 8,
  '--td-comp-margin-m': 12,
  '--td-comp-margin-l': 16,
  '--td-comp-margin-xl': 24,
  '--graft-density-gap-2': 2,
  '--graft-density-gap-4': 4,
  '--graft-density-gap-6': 6,
  '--graft-density-gap-8': 8,
  '--graft-density-gap-10': 10,
  '--graft-density-gap-12': 12,
  '--graft-density-gap-14': 14,
  '--graft-density-gap-16': 16,
  '--graft-density-gap-18': 18,
  '--graft-density-gap-20': 20,
  '--graft-density-gap-24': 24,
  '--graft-density-gap-28': 28,
  '--graft-density-gap-32': 32,
  '--graft-density-gap-48': 48,
  '--graft-density-padding-xs': 8,
  '--graft-density-padding-sm': 10,
  '--graft-density-padding-md': 12,
  '--graft-density-padding-lg': 14,
  '--graft-density-card-padding': 16,
  '--graft-density-card-padding-lg': 20,
  '--graft-density-section-padding': 24,
  '--graft-density-empty-padding': 28,
} as const satisfies Record<string, number>;

const DENSITY_SCALE_MAP: Record<ThemeAuthorityState['densityPreset'], number> = {
  compact: 0.88,
  standard: 1,
  comfortable: 1.12,
};

const FONT_SIZE_SCALE_MAP: Record<ThemeAuthorityState['fontSizePreset'], number> = {
  'extra-small': 0.88,
  small: 0.94,
  standard: 1,
  large: 1.06,
  'extra-large': 1.12,
};

const BASE_FONT_SIZE_TOKENS = {
  '--td-font-size-link-small': 12,
  '--td-font-size-link-medium': 14,
  '--td-font-size-link-large': 16,
  '--td-font-size-mark-small': 12,
  '--td-font-size-mark-medium': 14,
  '--td-font-size-body-small': 12,
  '--td-font-size-body-medium': 14,
  '--td-font-size-body-large': 16,
  '--td-font-size-title-small': 14,
  '--td-font-size-title-medium': 16,
  '--td-font-size-title-large': 18,
  '--td-font-size-title-extraLarge': 20,
  '--td-font-size-headline-small': 24,
  '--td-font-size-headline-medium': 28,
  '--td-font-size-headline-large': 36,
  '--td-font-size-display-medium': 48,
  '--td-font-size-display-large': 64,
} as const satisfies Record<string, number>;

const BASE_LINE_HEIGHT_TOKENS = {
  '--td-line-height-link-small': '20px',
  '--td-line-height-link-medium': '22px',
  '--td-line-height-link-large': '24px',
  '--td-line-height-mark-small': '20px',
  '--td-line-height-mark-medium': '22px',
  '--td-line-height-body-small': '20px',
  '--td-line-height-body-medium': '22px',
  '--td-line-height-body-large': '24px',
  '--td-line-height-title-small': '22px',
  '--td-line-height-title-medium': '24px',
  '--td-line-height-title-large': '26px',
  '--td-line-height-title-extraLarge': '28px',
  '--td-line-height-headline-small': '32px',
  '--td-line-height-headline-medium': '36px',
  '--td-line-height-headline-large': '44px',
  '--td-line-height-display-medium': '56px',
  '--td-line-height-display-large': '72px',
} as const satisfies ThemeTokenMap;

const FONT_TOKEN_ALIAS_MAP = {
  '--td-font-link-small': '--td-font-size-link-small / --td-line-height-link-small',
  '--td-font-link-medium': '--td-font-size-link-medium / --td-line-height-link-medium',
  '--td-font-link-large': '--td-font-size-link-large / --td-line-height-link-large',
  '--td-font-mark-small': '600 --td-font-size-mark-small / --td-line-height-mark-small',
  '--td-font-mark-medium': '600 --td-font-size-mark-medium / --td-line-height-mark-medium',
  '--td-font-body-small': '--td-font-size-body-small / --td-line-height-body-small',
  '--td-font-body-medium': '--td-font-size-body-medium / --td-line-height-body-medium',
  '--td-font-body-large': '--td-font-size-body-large / --td-line-height-body-large',
  '--td-font-title-small': '600 --td-font-size-title-small / --td-line-height-title-small',
  '--td-font-title-medium': '600 --td-font-size-title-medium / --td-line-height-title-medium',
  '--td-font-title-large': '600 --td-font-size-title-large / --td-line-height-title-large',
  '--td-font-title-extraLarge': '600 --td-font-size-title-extraLarge / --td-line-height-title-extraLarge',
  '--td-font-headline-small': '600 --td-font-size-headline-small / --td-line-height-headline-small',
  '--td-font-headline-medium': '600 --td-font-size-headline-medium / --td-line-height-headline-medium',
  '--td-font-headline-large': '600 --td-font-size-headline-large / --td-line-height-headline-large',
  '--td-font-display-medium': '600 --td-font-size-display-medium / --td-line-height-display-medium',
  '--td-font-display-large': '600 --td-font-size-display-large / --td-line-height-display-large',
} as const satisfies Record<string, string>;

const FONT_SCALE_PERCENT_MAP: Record<ThemeAuthorityState['fontSizePreset'], string> = {
  'extra-small': '88%',
  small: '94%',
  standard: '100%',
  large: '106%',
  'extra-large': '112%',
};

function px(value: number) {
  return `${Number(value.toFixed(2))}px`;
}

function buildFontSizeTokens(fontSizePreset: ThemeAuthorityState['fontSizePreset']): ThemeTokenMap {
  const scale = FONT_SIZE_SCALE_MAP[fontSizePreset];
  const scaledSizeTokens = Object.fromEntries(
    Object.entries(BASE_FONT_SIZE_TOKENS).map(([key, value]) => [key, px(value * scale)]),
  ) as ThemeTokenMap;
  const aliasTokens = Object.fromEntries(
    Object.entries(FONT_TOKEN_ALIAS_MAP).map(([key, template]) => [
      key,
      template.replaceAll(/--td-[a-zA-Z-]+/g, (tokenKey) => `var(${tokenKey})`) + ' var(--td-font-family)',
    ]),
  ) as ThemeTokenMap;

  return {
    '--graft-theme-font-scale': FONT_SCALE_PERCENT_MAP[fontSizePreset],
    ...scaledSizeTokens,
    ...BASE_LINE_HEIGHT_TOKENS,
    ...aliasTokens,
  };
}

function buildDensityTokens(densityPreset: ThemeAuthorityState['densityPreset']): ThemeTokenMap {
  const scale = DENSITY_SCALE_MAP[densityPreset];

  return {
    '--graft-theme-density-scale': String(scale),
    ...Object.fromEntries(Object.entries(BASE_DENSITY_TOKENS).map(([key, value]) => [key, px(value * scale)])),
  } as ThemeTokenMap;
}

function buildUserThemeTokens(authorityState: ThemeAuthorityState): ThemeModeTokenState {
  const sharedTokens: ThemeTokenMap = {
    '--td-font-family': FONT_FAMILY_MAP[authorityState.fontFamilyPreset],
    ...buildFontSizeTokens(authorityState.fontSizePreset),
    ...RADIUS_PRESET_MAP[authorityState.radiusPreset],
    ...SHADOW_PRESET_MAP[authorityState.shadowPreset],
    ...buildDensityTokens(authorityState.densityPreset),
  };

  return {
    light: sharedTokens,
    dark: sharedTokens,
  };
}

type ThemeAuthorityPresetDiffKey = Exclude<ThemeAuthorityDiffItem['key'], 'themeTokenOverrides'>;

const THEME_AUTHORITY_DIFF_KEYS = [
  'brandTheme',
  'fontFamilyPreset',
  'fontSizePreset',
  'radiusPreset',
  'shadowPreset',
  'densityPreset',
] as const satisfies ReadonlyArray<ThemeAuthorityPresetDiffKey>;

function countThemeTokenOverrides(tokens: ThemeModeTokenState) {
  return Object.keys(tokens.light).length + Object.keys(tokens.dark).length;
}

function hasThemeTokenOverrideDiff(fromTokens: ThemeModeTokenState, toTokens: ThemeModeTokenState) {
  const modes: Array<keyof ThemeModeTokenState> = ['light', 'dark'];

  return modes.some((mode) => {
    const keys = new Set([...Object.keys(fromTokens[mode]), ...Object.keys(toTokens[mode])]);
    return [...keys].some((key) => fromTokens[mode][key] !== toTokens[mode][key]);
  });
}

function createThemeAuthoritySourceSnapshot(
  preset: ThemePresetDefinition | null,
  currentState: Pick<
    ThemeAuthorityState,
    | 'selectedThemePresetId'
    | 'themeSource'
    | 'fontFamilyPreset'
    | 'fontSizePreset'
    | 'radiusPreset'
    | 'shadowPreset'
    | 'densityPreset'
  >,
): ThemeAuthorityState {
  return {
    mode: preset?.mode ?? (STYLE_CONFIG.mode as ModeType | 'auto'),
    brandTheme: preset?.brandTheme ?? STYLE_CONFIG.brandTheme,
    selectedThemePresetId: currentState.selectedThemePresetId,
    themeSource: currentState.themeSource,
    fontFamilyPreset: 'system',
    fontSizePreset: 'standard',
    radiusPreset: 'standard',
    shadowPreset: 'standard',
    densityPreset: 'standard',
    themeTokenOverrides: createEmptyThemeModeTokenState(),
  };
}

function createPersistedThemeAuthoritySnapshot(state: SettingState): ThemeAuthorityState {
  return {
    mode: state.mode as ModeType | 'auto',
    brandTheme: state.brandTheme,
    selectedThemePresetId: state.selectedThemePresetId,
    themeSource: state.themeSource,
    fontFamilyPreset: state.fontFamilyPreset,
    fontSizePreset: state.fontSizePreset,
    radiusPreset: state.radiusPreset,
    shadowPreset: state.shadowPreset,
    densityPreset: state.densityPreset,
    themeTokenOverrides: state.themeTokenOverrides,
  };
}

function hasThemeAuthorityStateDiff(fromState: ThemeAuthorityState, toState: ThemeAuthorityState) {
  return (
    fromState.mode !== toState.mode ||
    fromState.brandTheme !== toState.brandTheme ||
    fromState.selectedThemePresetId !== toState.selectedThemePresetId ||
    fromState.themeSource !== toState.themeSource ||
    fromState.fontFamilyPreset !== toState.fontFamilyPreset ||
    fromState.fontSizePreset !== toState.fontSizePreset ||
    fromState.radiusPreset !== toState.radiusPreset ||
    fromState.shadowPreset !== toState.shadowPreset ||
    fromState.densityPreset !== toState.densityPreset ||
    hasThemeTokenOverrideDiff(fromState.themeTokenOverrides, toState.themeTokenOverrides)
  );
}

type WorkbenchStyleConfigSnapshot = Pick<typeof STYLE_CONFIG, (typeof WORKBENCH_STYLE_CONFIG_KEYS)[number]>;

function createWorkbenchStyleConfigSnapshot(state: SettingState): WorkbenchStyleConfigSnapshot {
  return WORKBENCH_STYLE_CONFIG_KEYS.reduce((snapshot, key) => {
    snapshot[key] = state[key] as never;
    return snapshot;
  }, {} as WorkbenchStyleConfigSnapshot);
}

function hasWorkbenchStyleConfigDiff(fromState: WorkbenchStyleConfigSnapshot, toState: WorkbenchStyleConfigSnapshot) {
  return WORKBENCH_STYLE_CONFIG_KEYS.some((key) => fromState[key] !== toState[key]);
}

export type SettingState = typeof STYLE_CONFIG & {
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
  themeSource: ThemeSourceType;
  fontFamilyPreset: ThemeAuthorityState['fontFamilyPreset'];
  fontSizePreset: ThemeAuthorityState['fontSizePreset'];
  radiusPreset: ThemeAuthorityState['radiusPreset'];
  shadowPreset: ThemeAuthorityState['shadowPreset'];
  densityPreset: ThemeAuthorityState['densityPreset'];
  themeTokenOverrides: ThemeModeTokenState;
  themeResolvedTokens: ThemeModeTokenState;
  themeAuthorityLastModifiedAt: string | null;
  colorList: TColorSeries;
  chartColors: TChartColor;
};

function buildChartColorsFromTokens(tokens: ThemeTokenMap): TChartColor {
  return {
    textColor: tokens['--graft-chart-text-color'] ?? DEFAULT_CHART_COLORS.textColor,
    placeholderColor: tokens['--graft-chart-placeholder-color'] ?? DEFAULT_CHART_COLORS.placeholderColor,
    borderColor: tokens['--graft-chart-border-color'] ?? DEFAULT_CHART_COLORS.borderColor,
    containerColor: tokens['--graft-chart-container-color'] ?? DEFAULT_CHART_COLORS.containerColor,
  };
}

function createInitialSettingState(): SettingState {
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
    selectedThemePresetId: DEFAULT_THEME_PRESET_ID,
    themeSource: 'preset',
    fontFamilyPreset: 'system',
    fontSizePreset: 'standard',
    radiusPreset: 'standard',
    shadowPreset: 'standard',
    densityPreset: 'standard',
    themeTokenOverrides: createEmptyThemeModeTokenState(),
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
        shadowPreset: this.shadowPreset,
        densityPreset: this.densityPreset,
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
      this.shadowPreset = nextState.shadowPreset;
      this.densityPreset = nextState.densityPreset;
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
      const preset =
        THEME_PRESET_DEFINITIONS.find((item) => item.id === resolvePresetId(this.selectedThemePresetId)) ?? null;
      const brandTokens: ThemeModeTokenState = {
        light: this.getCachedBrandTokens(this.brandTheme, 'light'),
        dark: this.getCachedBrandTokens(this.brandTheme, 'dark'),
      };
      const userTokens = buildUserThemeTokens(this.createThemeAuthoritySnapshot());

      this.themeResolvedTokens = buildThemeModeSnapshot({
        baseTokens: GRAFT_BASE_THEME_TOKENS,
        brandTokens,
        preset,
        userTokens,
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
      this.syncThemeWorkbenchVisibility(true);
      if (!this.themeWorkbenchStyleConfigBaseline) {
        this.themeWorkbenchStyleConfigBaseline = this.createWorkbenchStyleConfigSnapshot();
      }
      if (!this.themeDraft) {
        const snapshot = this.createThemeAuthoritySnapshot();
        this.themeDraftBaseline = snapshot;
        this.themeDraft = snapshot;
        this.themeDraftApplied = false;
      }
      if (group) {
        this.activeThemeWorkbenchGroup = group;
      }
    },
    closeThemeWorkbench() {
      if (this.themeWorkbenchStyleConfigBaseline) {
        this.assignWorkbenchStyleConfigSnapshot(this.themeWorkbenchStyleConfigBaseline);
      }
      if (this.themeDraftBaseline && this.themeDraftApplied) {
        this.assignThemeAuthorityState(this.themeDraftBaseline);
        this.changeMode(this.mode as ModeType | 'auto');
      }
      this.syncThemeWorkbenchVisibility(false);
      this.themeWorkbenchStyleConfigBaseline = null;
      this.themeDraftBaseline = null;
      this.themeDraft = null;
      this.themeDraftApplied = false;
      this.themeResetting = false;
    },
    setActiveThemeWorkbenchGroup(group: ThemeWorkbenchGroupKey) {
      this.activeThemeWorkbenchGroup = group;
    },
    beginThemeDraft() {
      const snapshot = this.createThemeAuthoritySnapshot();
      this.themeWorkbenchStyleConfigBaseline = this.createWorkbenchStyleConfigSnapshot();
      this.themeDraftBaseline = snapshot;
      this.themeDraft = snapshot;
      this.themeDraftApplied = false;
    },
    applyThemeDraftPreview() {
      if (!this.themeDraft) {
        return;
      }

      this.assignThemeAuthorityState(this.themeDraft);
      this.changeMode(this.mode as ModeType | 'auto');
      this.themeDraftApplied = true;
    },
    updateThemeDraft(patch: Partial<ThemeAuthorityState>) {
      const base = this.themeDraft ?? this.createThemeAuthoritySnapshot();
      this.themeDraft = {
        ...base,
        ...patch,
        themeTokenOverrides: patch.themeTokenOverrides
          ? cloneThemeModeTokenState(patch.themeTokenOverrides)
          : cloneThemeModeTokenState(base.themeTokenOverrides),
      };
      this.applyThemeDraftPreview();
    },
    applyThemeDraft() {
      if (!this.themeDraft) {
        return;
      }

      const hasPendingChanges = this.hasThemeWorkbenchPendingChanges;
      this.assignThemeAuthorityState(this.themeDraft);
      if (hasPendingChanges) {
        this.themeAuthorityLastModifiedAt = new Date().toISOString();
      }
      this.changeMode(this.mode as ModeType | 'auto');
      this.themeWorkbenchStyleConfigBaseline = null;
      this.themeDraftBaseline = null;
      this.themeDraft = null;
      this.themeDraftApplied = false;
      this.syncThemeWorkbenchVisibility(false);
    },
    cancelThemeDraft() {
      this.closeThemeWorkbench();
    },
    resetThemeDraftToDefault(options: { preserveResettingFeedback?: boolean } = {}) {
      if (!this.themeDraftBaseline) {
        this.themeDraftBaseline = this.createThemeAuthoritySnapshot();
      }

      const defaultSnapshot: ThemeAuthorityState = {
        mode: STYLE_CONFIG.mode as ModeType | 'auto',
        brandTheme: STYLE_CONFIG.brandTheme,
        selectedThemePresetId: DEFAULT_THEME_PRESET_ID,
        themeSource: 'preset',
        fontFamilyPreset: 'system',
        fontSizePreset: 'standard',
        radiusPreset: 'standard',
        shadowPreset: 'standard',
        densityPreset: 'standard',
        themeTokenOverrides: createEmptyThemeModeTokenState(),
      };
      this.themeDraft = defaultSnapshot;
      this.applyThemeDraftPreview();
      if (!options.preserveResettingFeedback) {
        this.themeResetting = false;
      }
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
    selectThemePreset(presetId: string | null) {
      const resolvedPresetId = resolvePresetId(presetId);
      const preset = THEME_PRESET_DEFINITIONS.find((item) => item.id === resolvedPresetId);

      if (!preset) {
        return;
      }

      const nextState: ThemeAuthorityState = {
        mode: preset.mode ?? this.themeDraft?.mode ?? (this.mode as ModeType | 'auto'),
        brandTheme: preset.brandTheme,
        selectedThemePresetId: preset.id,
        themeSource: 'preset',
        fontFamilyPreset: this.themeDraft?.fontFamilyPreset ?? this.fontFamilyPreset,
        fontSizePreset: this.themeDraft?.fontSizePreset ?? this.fontSizePreset,
        radiusPreset: this.themeDraft?.radiusPreset ?? this.radiusPreset,
        shadowPreset: this.themeDraft?.shadowPreset ?? this.shadowPreset,
        densityPreset: this.themeDraft?.densityPreset ?? this.densityPreset,
        themeTokenOverrides: createEmptyThemeModeTokenState(),
      };
      this.updateThemeDraft(nextState);
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
          'mode' | 'fontFamilyPreset' | 'fontSizePreset' | 'radiusPreset' | 'shadowPreset' | 'densityPreset'
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
      this.updateThemeDraft({
        themeSource: 'customized',
        themeTokenOverrides: {
          ...baseState.themeTokenOverrides,
          [mode]: {
            ...baseState.themeTokenOverrides[mode],
            [tokenKey]: tokenValue,
          },
        },
      });
    },
    updateThemeTokenGroup(mode: ModeType, tokenGroup: ThemeTokenMap) {
      const baseState = this.themeDraft ?? this.createThemeAuthoritySnapshot();
      this.updateThemeDraft({
        themeSource: 'customized',
        themeTokenOverrides: {
          ...baseState.themeTokenOverrides,
          [mode]: {
            ...baseState.themeTokenOverrides[mode],
            ...tokenGroup,
          },
        },
      });
    },
    clearThemeTokenGroup(mode: ModeType, tokenKeys?: string[]) {
      const baseState = this.themeDraft ?? this.createThemeAuthoritySnapshot();
      const nextTokens = { ...baseState.themeTokenOverrides[mode] };
      const nextThemeTokenOverrides = cloneThemeModeTokenState(baseState.themeTokenOverrides);

      if (!tokenKeys?.length) {
        nextThemeTokenOverrides[mode] = {};
      } else {
        tokenKeys.forEach((tokenKey) => {
          delete nextTokens[tokenKey];
        });
        nextThemeTokenOverrides[mode] = nextTokens;
      }

      const hasOverrides =
        Object.keys(nextThemeTokenOverrides.light).length > 0 || Object.keys(nextThemeTokenOverrides.dark).length > 0;
      this.updateThemeDraft({
        themeTokenOverrides: nextThemeTokenOverrides,
        themeSource: hasOverrides ? 'customized' : baseState.selectedThemePresetId ? 'preset' : 'customized',
      });
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
      'themeAuthorityLastModifiedAt',
      'selectedThemePresetId',
      'themeSource',
      'fontFamilyPreset',
      'fontSizePreset',
      'radiusPreset',
      'shadowPreset',
      'densityPreset',
      'themeTokenOverrides',
      'themeResolvedTokens',
      'activeThemeWorkbenchGroup',
      'activeThemeTokenGroup',
      'themeWorkbenchDockPosition',
    ],
  },
});
