import keys from 'lodash/keys';

import STYLE_CONFIG from '@/config/style';
import type {
  ThemeAuthorityDiffItem,
  ThemeAuthorityState,
  ThemeModeTokenState,
  ThemePresetDefinition,
  ThemeTokenMap,
} from '@/types/theme';
import { createEmptyThemeModeTokenState } from '@/utils/theme-workbench';
import type { ModeType } from '@/utils/types';

export const STYLE_CONFIG_KEYS = keys(STYLE_CONFIG) as Array<keyof typeof STYLE_CONFIG>;

const THEME_AUTHORITY_STYLE_KEYS = ['mode', 'brandTheme'] as const;

export const WORKBENCH_STYLE_CONFIG_KEYS = STYLE_CONFIG_KEYS.filter(
  (key) => !(THEME_AUTHORITY_STYLE_KEYS as readonly string[]).includes(key),
);

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

type ThemeAuthorityPresetDiffKey = Exclude<ThemeAuthorityDiffItem['key'], 'themeTokenOverrides'>;

export const THEME_AUTHORITY_DIFF_KEYS = [
  'brandTheme',
  'fontFamilyPreset',
  'fontSizePreset',
  'radiusPreset',
  'shadowPreset',
  'densityPreset',
] as const satisfies ReadonlyArray<ThemeAuthorityPresetDiffKey>;

export type PersistedThemeAuthoritySource = {
  mode: string;
  brandTheme: string;
  selectedThemePresetId: string | null;
  themeSource: ThemeAuthorityState['themeSource'];
  fontFamilyPreset: ThemeAuthorityState['fontFamilyPreset'];
  fontSizePreset: ThemeAuthorityState['fontSizePreset'];
  radiusPreset: ThemeAuthorityState['radiusPreset'];
  shadowPreset: ThemeAuthorityState['shadowPreset'];
  densityPreset: ThemeAuthorityState['densityPreset'];
  themeTokenOverrides: ThemeModeTokenState;
};

type ThemeModeValue = ModeType | 'auto';

export type WorkbenchStyleConfigSnapshot = Pick<typeof STYLE_CONFIG, (typeof WORKBENCH_STYLE_CONFIG_KEYS)[number]>;

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

export function buildUserThemeTokens(authorityState: ThemeAuthorityState): ThemeModeTokenState {
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

export function countThemeTokenOverrides(tokens: ThemeModeTokenState) {
  return Object.keys(tokens.light).length + Object.keys(tokens.dark).length;
}

export function hasThemeTokenOverrideDiff(fromTokens: ThemeModeTokenState, toTokens: ThemeModeTokenState) {
  const modes: Array<keyof ThemeModeTokenState> = ['light', 'dark'];

  return modes.some((mode) => {
    const keys = new Set([...Object.keys(fromTokens[mode]), ...Object.keys(toTokens[mode])]);
    return [...keys].some((key) => fromTokens[mode][key] !== toTokens[mode][key]);
  });
}

export function createThemeAuthoritySourceSnapshot(
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
    mode: (preset?.mode ?? STYLE_CONFIG.mode) as ThemeModeValue,
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

export function createPersistedThemeAuthoritySnapshot(state: PersistedThemeAuthoritySource): ThemeAuthorityState {
  return {
    mode: state.mode as ThemeModeValue,
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

export function hasThemeAuthorityStateDiff(fromState: ThemeAuthorityState, toState: ThemeAuthorityState) {
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

export function createWorkbenchStyleConfigSnapshot(state: typeof STYLE_CONFIG): WorkbenchStyleConfigSnapshot {
  return WORKBENCH_STYLE_CONFIG_KEYS.reduce((snapshot, key) => {
    snapshot[key] = state[key] as never;
    return snapshot;
  }, {} as WorkbenchStyleConfigSnapshot);
}

export function hasWorkbenchStyleConfigDiff(
  fromState: WorkbenchStyleConfigSnapshot,
  toState: WorkbenchStyleConfigSnapshot,
) {
  return WORKBENCH_STYLE_CONFIG_KEYS.some((key) => fromState[key] !== toState[key]);
}
