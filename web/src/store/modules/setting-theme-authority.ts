import keys from 'lodash/keys';

import STYLE_CONFIG from '@/config/style';
import type {
  ThemeAuthorityDiffItem,
  ThemeAuthorityState,
  ThemeModeTokenState,
  ThemePresetDefinition,
  ThemeTokenMap,
} from '@/types/theme';
import type { ModeType } from '@/utils/types';

export const STYLE_CONFIG_KEYS = keys(STYLE_CONFIG) as Array<keyof typeof STYLE_CONFIG>;

const THEME_AUTHORITY_STYLE_KEYS = ['mode', 'brandTheme'] as const;
const WORKBENCH_APPLICATION_PREFERENCE_KEYS = ['themePresetApplicationScope'] as const;

export const WORKBENCH_STYLE_CONFIG_KEYS = STYLE_CONFIG_KEYS.filter(
  (key) =>
    !(THEME_AUTHORITY_STYLE_KEYS as readonly string[]).includes(key) &&
    !(WORKBENCH_APPLICATION_PREFERENCE_KEYS as readonly string[]).includes(key),
);

const FONT_FAMILY_MAP: Record<ThemeAuthorityState['fontFamilyPreset'], string> = {
  system: '-apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif',
  harmonyos: '"HarmonyOS Sans SC", "HarmonyOS Sans", "PingFang SC", "Microsoft YaHei", sans-serif',
  inter: '"Inter", "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif',
  'source-han-sans': '"Source Han Sans SC", "Noto Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif',
};

const RADIUS_PRESET_MAP: Record<ThemeAuthorityState['radiusPreset'], ThemeTokenMap> = {
  square: {
    '--td-radius-small': '0',
    '--td-radius-default': '0',
    '--td-radius-medium': '0',
    '--td-radius-large': '0',
    '--td-radius-extraLarge': '0',
    '--td-radius-circle': '999px',
  },
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

const RADIUS_PRESET_ANCHORS = [
  { preset: 'square', value: 0 },
  { preset: 'business', value: 4 },
  { preset: 'standard', value: 8 },
  { preset: 'rounded', value: 12 },
  { preset: 'capsule', value: 16 },
] as const satisfies ReadonlyArray<{ preset: ThemeAuthorityState['radiusPreset']; value: number }>;

const SHADOW_INTENSITY_ANCHORS = [
  { preset: 'subtle', value: 0.5 },
  { preset: 'standard', value: 1 },
  { preset: 'strong', value: 1.5 },
] as const satisfies ReadonlyArray<{ preset: ThemeAuthorityState['shadowIntensity']; value: number }>;

const SHADOW_PRESET_MAP: Record<
  ThemeAuthorityState['shadowPreset'],
  Record<ThemeAuthorityState['shadowIntensity'], ThemeTokenMap>
> = {
  'hard-offset': {
    subtle: {
      '--td-shadow-1': '1px 1px 0 var(--graft-neo-ink, var(--td-text-color-primary))',
      '--td-shadow-2': '2px 2px 0 var(--graft-neo-ink, var(--td-text-color-primary))',
      '--td-shadow-3': '3px 3px 0 var(--graft-neo-ink, var(--td-text-color-primary))',
    },
    standard: {
      '--td-shadow-1': '2px 2px 0 var(--graft-neo-ink, var(--td-text-color-primary))',
      '--td-shadow-2': '4px 4px 0 var(--graft-neo-ink, var(--td-text-color-primary))',
      '--td-shadow-3': '6px 6px 0 var(--graft-neo-ink, var(--td-text-color-primary))',
    },
    strong: {
      '--td-shadow-1': '3px 3px 0 var(--graft-neo-ink, var(--td-text-color-primary))',
      '--td-shadow-2': '6px 6px 0 var(--graft-neo-ink, var(--td-text-color-primary))',
      '--td-shadow-3': '9px 9px 0 var(--graft-neo-ink, var(--td-text-color-primary))',
    },
  },
  flat: {
    subtle: {
      '--td-shadow-1': 'none',
      '--td-shadow-2': 'none',
      '--td-shadow-3': 'none',
    },
    standard: {
      '--td-shadow-1': 'none',
      '--td-shadow-2': 'none',
      '--td-shadow-3': 'none',
    },
    strong: {
      '--td-shadow-1': 'none',
      '--td-shadow-2': 'none',
      '--td-shadow-3': 'none',
    },
  },
  standard: {
    subtle: {
      '--td-shadow-1': '0 1px 6px rgba(15, 23, 42, 0.05)',
      '--td-shadow-2': '0 6px 16px rgba(15, 23, 42, 0.08)',
      '--td-shadow-3': '0 12px 30px rgba(15, 23, 42, 0.12)',
    },
    standard: {
      '--td-shadow-1': '0 2px 10px rgba(15, 23, 42, 0.08)',
      '--td-shadow-2': '0 10px 24px rgba(15, 23, 42, 0.12)',
      '--td-shadow-3': '0 18px 42px rgba(15, 23, 42, 0.18)',
    },
    strong: {
      '--td-shadow-1': '0 3px 14px rgba(15, 23, 42, 0.12)',
      '--td-shadow-2': '0 14px 32px rgba(15, 23, 42, 0.18)',
      '--td-shadow-3': '0 26px 58px rgba(15, 23, 42, 0.24)',
    },
  },
  floating: {
    subtle: {
      '--td-shadow-1': '0 3px 10px rgba(15, 23, 42, 0.08)',
      '--td-shadow-2': '0 10px 24px rgba(15, 23, 42, 0.12)',
      '--td-shadow-3': '0 16px 40px rgba(15, 23, 42, 0.18)',
    },
    standard: {
      '--td-shadow-1': '0 6px 16px rgba(15, 23, 42, 0.12)',
      '--td-shadow-2': '0 16px 36px rgba(15, 23, 42, 0.18)',
      '--td-shadow-3': '0 24px 56px rgba(15, 23, 42, 0.24)',
    },
    strong: {
      '--td-shadow-1': '0 8px 22px rgba(15, 23, 42, 0.16)',
      '--td-shadow-2': '0 22px 48px rgba(15, 23, 42, 0.24)',
      '--td-shadow-3': '0 34px 72px rgba(15, 23, 42, 0.3)',
    },
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

const DENSITY_PRESET_ANCHORS = [
  { preset: 'compact', value: 0.88 },
  { preset: 'standard', value: 1 },
  { preset: 'comfortable', value: 1.12 },
] as const satisfies ReadonlyArray<{ preset: ThemeAuthorityState['densityPreset']; value: number }>;

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
  'radiusOverride',
  'shadowPreset',
  'shadowIntensity',
  'shadowIntensityOverride',
  'densityPreset',
  'densityOverride',
] as const satisfies ReadonlyArray<ThemeAuthorityPresetDiffKey>;

export type PersistedThemeAuthoritySource = {
  mode: string;
  brandTheme: string;
  selectedThemePresetId: string | null;
  themeSource: ThemeAuthorityState['themeSource'];
  fontFamilyPreset: ThemeAuthorityState['fontFamilyPreset'];
  fontSizePreset: ThemeAuthorityState['fontSizePreset'];
  radiusPreset: ThemeAuthorityState['radiusPreset'];
  radiusOverride: ThemeAuthorityState['radiusOverride'];
  shadowPreset: ThemeAuthorityState['shadowPreset'];
  shadowIntensity: ThemeAuthorityState['shadowIntensity'];
  shadowIntensityOverride: ThemeAuthorityState['shadowIntensityOverride'];
  densityPreset: ThemeAuthorityState['densityPreset'];
  densityOverride: ThemeAuthorityState['densityOverride'];
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

function findPresetAnchor<TPreset extends string>(
  value: number,
  anchors: ReadonlyArray<{ preset: TPreset; value: number }>,
): TPreset | null {
  return anchors.find((anchor) => Math.abs(anchor.value - value) < 0.001)?.preset ?? null;
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}

export function normalizeThemeAuthorityOverrides(state: ThemeAuthorityState): ThemeAuthorityState {
  const radiusValue = state.radiusOverride === null ? null : clamp(state.radiusOverride, 0, 16);
  const radiusPreset = radiusValue === null ? null : findPresetAnchor(radiusValue, RADIUS_PRESET_ANCHORS);
  const shadowIntensityValue =
    state.shadowIntensityOverride === null ? null : clamp(state.shadowIntensityOverride, 0.5, 1.5);
  const shadowIntensity =
    shadowIntensityValue === null ? null : findPresetAnchor(shadowIntensityValue, SHADOW_INTENSITY_ANCHORS);
  const densityValue = state.densityOverride === null ? null : clamp(state.densityOverride, 0.88, 1.12);
  const densityPreset = densityValue === null ? null : findPresetAnchor(densityValue, DENSITY_PRESET_ANCHORS);

  return {
    ...state,
    radiusPreset: radiusPreset ?? state.radiusPreset,
    radiusOverride: radiusPreset ? null : radiusValue,
    shadowIntensity: shadowIntensity ?? state.shadowIntensity,
    shadowIntensityOverride: shadowIntensity ? null : shadowIntensityValue,
    densityPreset: densityPreset ?? state.densityPreset,
    densityOverride: densityPreset ? null : densityValue,
  };
}

function buildInterpolatedRadiusTokens(radiusOverride: number): ThemeTokenMap {
  const radius = clamp(radiusOverride, 0, 16);
  const upperIndex = RADIUS_PRESET_ANCHORS.findIndex((anchor) => anchor.value >= radius);
  const upper = RADIUS_PRESET_ANCHORS[upperIndex === -1 ? RADIUS_PRESET_ANCHORS.length - 1 : upperIndex];
  const lower = RADIUS_PRESET_ANCHORS[Math.max(0, RADIUS_PRESET_ANCHORS.indexOf(upper) - 1)];
  const progress = lower.value === upper.value ? 0 : (radius - lower.value) / (upper.value - lower.value);
  const lowerTokens = RADIUS_PRESET_MAP[lower.preset];
  const upperTokens = RADIUS_PRESET_MAP[upper.preset];

  return Object.fromEntries(
    Object.keys(lowerTokens).map((key) => {
      if (key === '--td-radius-circle') {
        return [key, lowerTokens[key]];
      }

      const lowerValue = Number.parseFloat(lowerTokens[key]);
      const upperValue = Number.parseFloat(upperTokens[key]);
      return [key, px(lowerValue + (upperValue - lowerValue) * progress)];
    }),
  );
}

function buildRadiusTokens(authorityState: ThemeAuthorityState): ThemeTokenMap {
  return authorityState.radiusOverride === null
    ? RADIUS_PRESET_MAP[authorityState.radiusPreset]
    : buildInterpolatedRadiusTokens(authorityState.radiusOverride);
}

function buildDensityTokens(authorityState: ThemeAuthorityState): ThemeTokenMap {
  const scale = authorityState.densityOverride ?? DENSITY_SCALE_MAP[authorityState.densityPreset];

  return {
    '--graft-theme-density-scale': String(scale),
    ...Object.fromEntries(Object.entries(BASE_DENSITY_TOKENS).map(([key, value]) => [key, px(value * scale)])),
  } as ThemeTokenMap;
}

function interpolateSoftShadowToken(from: string, to: string, progress: number) {
  const pattern = /^0 ([\d.]+)px ([\d.]+)px rgba\(15, 23, 42, ([\d.]+)\)$/;
  const fromMatch = from.match(pattern);
  const toMatch = to.match(pattern);

  if (!fromMatch || !toMatch) {
    return from;
  }

  const interpolate = (index: number) =>
    Number(fromMatch[index]) + (Number(toMatch[index]) - Number(fromMatch[index])) * progress;
  return `0 ${px(interpolate(1))} ${px(interpolate(2))} rgba(15, 23, 42, ${Number(interpolate(3).toFixed(3))})`;
}

function buildContinuousSoftShadowTokens(
  shadowPreset: Extract<ThemeAuthorityState['shadowPreset'], 'standard' | 'floating'>,
  shadowIntensityOverride: number,
): ThemeTokenMap {
  const intensity = clamp(shadowIntensityOverride, 0.5, 1.5);
  const upperIndex = SHADOW_INTENSITY_ANCHORS.findIndex((anchor) => anchor.value >= intensity);
  const upper = SHADOW_INTENSITY_ANCHORS[upperIndex === -1 ? SHADOW_INTENSITY_ANCHORS.length - 1 : upperIndex];
  const lower = SHADOW_INTENSITY_ANCHORS[Math.max(0, SHADOW_INTENSITY_ANCHORS.indexOf(upper) - 1)];
  const progress = lower.value === upper.value ? 0 : (intensity - lower.value) / (upper.value - lower.value);
  const lowerTokens = SHADOW_PRESET_MAP[shadowPreset][lower.preset];
  const upperTokens = SHADOW_PRESET_MAP[shadowPreset][upper.preset];

  return Object.fromEntries(
    Object.keys(lowerTokens).map((key) => [
      key,
      interpolateSoftShadowToken(lowerTokens[key], upperTokens[key], progress),
    ]),
  );
}

function buildShadowTokens(
  shadowPreset: ThemeAuthorityState['shadowPreset'],
  shadowIntensity: ThemeAuthorityState['shadowIntensity'],
  shadowIntensityOverride: ThemeAuthorityState['shadowIntensityOverride'],
): ThemeTokenMap {
  const intensity =
    shadowIntensityOverride ?? SHADOW_INTENSITY_ANCHORS.find((anchor) => anchor.preset === shadowIntensity)?.value ?? 1;

  if (shadowPreset === 'flat') {
    return SHADOW_PRESET_MAP.flat.standard;
  }

  if (shadowPreset === 'hard-offset') {
    const hardOffsetIntensity = clamp(intensity, 0.5, 1.5);
    const hardOffsetTokens = [1, 2, 3].reduce<ThemeTokenMap>((tokens, level) => {
      const offset = px(level * 2 * hardOffsetIntensity);
      tokens[`--td-shadow-${level}`] = `${offset} ${offset} 0 var(--graft-neo-ink, var(--td-text-color-primary))`;
      return tokens;
    }, {});

    return {
      ...hardOffsetTokens,
      '--graft-neo-shadow': `${px(4 * hardOffsetIntensity)} ${px(4 * hardOffsetIntensity)} 0 var(--graft-neo-ink)`,
    };
  }

  const shadowTokens =
    shadowIntensityOverride === null
      ? SHADOW_PRESET_MAP[shadowPreset][shadowIntensity]
      : buildContinuousSoftShadowTokens(shadowPreset, intensity);

  return shadowTokens;
}

export function buildUserThemeTokens(authorityState: ThemeAuthorityState): ThemeModeTokenState {
  const sharedTokens: ThemeTokenMap = {
    '--td-font-family': FONT_FAMILY_MAP[authorityState.fontFamilyPreset],
    ...buildFontSizeTokens(authorityState.fontSizePreset),
    ...buildRadiusTokens(authorityState),
    ...buildShadowTokens(
      authorityState.shadowPreset,
      authorityState.shadowIntensity,
      authorityState.shadowIntensityOverride,
    ),
    ...buildDensityTokens(authorityState),
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
    | 'radiusOverride'
    | 'shadowPreset'
    | 'shadowIntensity'
    | 'shadowIntensityOverride'
    | 'densityPreset'
    | 'densityOverride'
  >,
): ThemeAuthorityState {
  return {
    mode: (preset?.mode ?? STYLE_CONFIG.mode) as ThemeModeValue,
    brandTheme: preset?.brandTheme ?? STYLE_CONFIG.brandTheme,
    selectedThemePresetId: currentState.selectedThemePresetId,
    themeSource: currentState.themeSource,
    fontFamilyPreset: preset?.authorityPatch?.fontFamilyPreset ?? 'system',
    fontSizePreset: preset?.authorityPatch?.fontSizePreset ?? 'standard',
    radiusPreset: preset?.authorityPatch?.radiusPreset ?? 'standard',
    radiusOverride: null,
    shadowPreset: preset?.authorityPatch?.shadowPreset ?? 'standard',
    shadowIntensity: preset?.authorityPatch?.shadowIntensity ?? 'standard',
    shadowIntensityOverride: null,
    densityPreset: preset?.authorityPatch?.densityPreset ?? 'standard',
    densityOverride: null,
    themeTokenOverrides: {
      light: {
        ...(preset?.tokenOverrides?.light ?? {}),
        ...(preset?.materialTokenOverrides?.light ?? {}),
      },
      dark: {
        ...(preset?.tokenOverrides?.dark ?? {}),
        ...(preset?.materialTokenOverrides?.dark ?? {}),
      },
    },
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
    radiusOverride: state.radiusOverride ?? null,
    shadowPreset: state.shadowPreset,
    shadowIntensity: state.shadowIntensity,
    shadowIntensityOverride: state.shadowIntensityOverride ?? null,
    densityPreset: state.densityPreset,
    densityOverride: state.densityOverride ?? null,
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
    fromState.radiusOverride !== toState.radiusOverride ||
    fromState.shadowPreset !== toState.shadowPreset ||
    fromState.shadowIntensity !== toState.shadowIntensity ||
    fromState.shadowIntensityOverride !== toState.shadowIntensityOverride ||
    fromState.densityPreset !== toState.densityPreset ||
    fromState.densityOverride !== toState.densityOverride ||
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
