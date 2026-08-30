import { THEME_PRESET_DEFINITIONS } from '@/config/theme-workbench';
import type {
  ThemeAuthorityState,
  ThemeModeTokenState,
  ThemePresetApplicationScope,
  ThemePresetDefinition,
  ThemeTokenMap,
  ThemeWorkbenchGroupKey,
} from '@/types/theme';
import { cloneThemeModeTokenState, createEmptyThemeModeTokenState } from '@/utils/theme-workbench';
import type { ModeType } from '@/utils/types';

import type { WorkbenchStyleConfigSnapshot } from './setting-theme-authority';
import { hasThemeTokenOverrideDiff, normalizeThemeAuthorityOverrides } from './setting-theme-authority';

function createPresetThemeTokenOverrides(
  preset: ThemePresetDefinition,
  scope: ThemePresetApplicationScope,
  currentTokens?: ThemeModeTokenState,
  previousPreset?: ThemePresetDefinition | null,
): ThemeModeTokenState {
  const materialTokens = scope === 'complete' ? preset.materialTokenOverrides : undefined;
  const currentPaletteTokens = {
    light: { ...(currentTokens?.light ?? {}) },
    dark: { ...(currentTokens?.dark ?? {}) },
  };

  if (scope === 'palette' && previousPreset) {
    (['light', 'dark'] as const).forEach((mode) => {
      Object.entries({
        ...(previousPreset.tokenOverrides?.[mode] ?? {}),
        ...(previousPreset.materialTokenOverrides?.[mode] ?? {}),
      }).forEach(([tokenKey, tokenValue]) => {
        if (currentPaletteTokens[mode][tokenKey] === tokenValue) {
          delete currentPaletteTokens[mode][tokenKey];
        }
      });
    });
  }

  if (scope === 'palette') {
    // 即使其它 token 已被个性化导致无法识别旧预设，也只清掉仍等于内置材质值的残留；
    // 用户改过的材质 token 必须继续随调色板切换保留。
    (['light', 'dark'] as const).forEach((mode) => {
      const builtInMaterialValues = new Set(
        THEME_PRESET_DEFINITIONS.flatMap((item) => Object.values(item.materialTokenOverrides?.[mode] ?? {})),
      );
      const builtInMaterialKeys = new Set(
        THEME_PRESET_DEFINITIONS.flatMap((item) => Object.keys(item.materialTokenOverrides?.[mode] ?? {})),
      );
      Object.entries(currentPaletteTokens[mode]).forEach(([tokenKey, tokenValue]) => {
        if (builtInMaterialKeys.has(tokenKey) && builtInMaterialValues.has(tokenValue)) {
          delete currentPaletteTokens[mode][tokenKey];
        }
      });
    });
  }

  return {
    light: {
      ...(preset.tokenOverrides?.light ?? {}),
      ...(materialTokens?.light ?? {}),
      ...(scope === 'palette' ? currentPaletteTokens.light : {}),
    },
    dark: {
      ...(preset.tokenOverrides?.dark ?? {}),
      ...(materialTokens?.dark ?? {}),
      ...(scope === 'palette' ? currentPaletteTokens.dark : {}),
    },
  };
}

export function createCompleteThemePresetState(
  preset: ThemePresetDefinition,
  fallbackMode: ModeType | 'auto',
): ThemeAuthorityState {
  return {
    mode: preset.authorityPatch?.mode ?? preset.mode ?? fallbackMode,
    brandTheme: preset.brandTheme,
    selectedThemePresetId: preset.id,
    themeSource: 'preset',
    fontFamilyPreset: preset.authorityPatch?.fontFamilyPreset ?? 'system',
    fontSizePreset: preset.authorityPatch?.fontSizePreset ?? 'standard',
    radiusPreset: preset.authorityPatch?.radiusPreset ?? 'standard',
    radiusOverride: null,
    shadowPreset: preset.authorityPatch?.shadowPreset ?? 'standard',
    shadowIntensity: preset.authorityPatch?.shadowIntensity ?? 'standard',
    shadowIntensityOverride: null,
    densityPreset: preset.authorityPatch?.densityPreset ?? 'standard',
    densityOverride: null,
    themeTokenOverrides: createPresetThemeTokenOverrides(preset, 'complete'),
  };
}

export function buildUpdatedThemeDraft(
  base: ThemeAuthorityState,
  patch: Partial<ThemeAuthorityState>,
): ThemeAuthorityState {
  const nextState = normalizeThemeAuthorityOverrides({
    ...base,
    ...patch,
    themeTokenOverrides: patch.themeTokenOverrides
      ? cloneThemeModeTokenState(patch.themeTokenOverrides)
      : cloneThemeModeTokenState(base.themeTokenOverrides),
  });

  // 显式选择预设时保留选择结果；其余编辑入口都根据最终 authority 重新判定预设归属。
  if (patch.selectedThemePresetId !== undefined) {
    return nextState;
  }

  const matchingPreset = THEME_PRESET_DEFINITIONS.find((preset) => matchesCompleteThemePreset(nextState, preset));
  return {
    ...nextState,
    selectedThemePresetId: matchingPreset?.id ?? null,
    themeSource: matchingPreset ? 'preset' : 'customized',
  };
}

const THEME_PRESET_MATCH_KEYS = [
  'mode',
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
] as const;

function matchesCompleteThemePreset(state: ThemeAuthorityState, preset: ThemePresetDefinition): boolean {
  const expected = createCompleteThemePresetState(preset, state.mode);

  return (
    THEME_PRESET_MATCH_KEYS.every((key) => state[key] === expected[key]) &&
    !hasThemeTokenOverrideDiff(state.themeTokenOverrides, expected.themeTokenOverrides)
  );
}

/** 从仍保留的调色板 token 中恢复上一个预设，供下一次 palette 切换清理旧材质 token。 */
export function findThemePresetForPaletteTransition(state: ThemeAuthorityState): ThemePresetDefinition | null {
  return (
    THEME_PRESET_DEFINITIONS.find((preset) => {
      if (preset.brandTheme !== state.brandTheme || (preset.mode && preset.mode !== state.mode)) {
        return false;
      }

      return (['light', 'dark'] as const).every((mode) => {
        const expectedTokens = {
          ...(preset.tokenOverrides?.[mode] ?? {}),
          ...(preset.materialTokenOverrides?.[mode] ?? {}),
        };
        return Object.entries(expectedTokens).every(([key, value]) => state.themeTokenOverrides[mode][key] === value);
      });
    }) ?? null
  );
}

export function buildSelectedThemePresetState(
  preset: ThemePresetDefinition,
  draftState: ThemeAuthorityState | null,
  persistedState: Pick<
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
  >,
  scope: ThemePresetApplicationScope,
  previousPreset?: ThemePresetDefinition | null,
): ThemeAuthorityState {
  const current = draftState ?? {
    ...persistedState,
    themeTokenOverrides: createEmptyThemeModeTokenState(),
  };

  if (scope === 'complete') {
    return createCompleteThemePresetState(preset, persistedState.mode);
  }

  return {
    mode: preset.authorityPatch?.mode ?? preset.mode ?? current.mode,
    brandTheme: preset.brandTheme,
    selectedThemePresetId: preset.id,
    themeSource: 'preset',
    fontFamilyPreset: current.fontFamilyPreset,
    fontSizePreset: current.fontSizePreset,
    radiusPreset: current.radiusPreset,
    radiusOverride: current.radiusOverride,
    shadowPreset: current.shadowPreset,
    shadowIntensity: current.shadowIntensity,
    shadowIntensityOverride: current.shadowIntensityOverride,
    densityPreset: current.densityPreset,
    densityOverride: current.densityOverride,
    themeTokenOverrides: createPresetThemeTokenOverrides(
      preset,
      'palette',
      current.themeTokenOverrides,
      previousPreset,
    ),
  };
}

export function buildThemeTokenValueUpdate(
  baseState: ThemeAuthorityState,
  mode: ModeType,
  tokenKey: string,
  tokenValue: string,
): Partial<ThemeAuthorityState> {
  return {
    themeSource: 'customized',
    themeTokenOverrides: {
      ...baseState.themeTokenOverrides,
      [mode]: {
        ...baseState.themeTokenOverrides[mode],
        [tokenKey]: tokenValue,
      },
    },
  };
}

export function buildThemeTokenGroupUpdate(
  baseState: ThemeAuthorityState,
  mode: ModeType,
  tokenGroup: ThemeTokenMap,
): Partial<ThemeAuthorityState> {
  return {
    themeSource: 'customized',
    themeTokenOverrides: {
      ...baseState.themeTokenOverrides,
      [mode]: {
        ...baseState.themeTokenOverrides[mode],
        ...tokenGroup,
      },
    },
  };
}

export function clearThemeTokenGroupOverrides(
  baseState: ThemeAuthorityState,
  mode: ModeType,
  tokenKeys?: string[],
): Partial<ThemeAuthorityState> {
  const nextTokens = { ...baseState.themeTokenOverrides[mode] };
  const nextThemeTokenOverrides: ThemeModeTokenState = cloneThemeModeTokenState(baseState.themeTokenOverrides);

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

  return {
    themeTokenOverrides: nextThemeTokenOverrides,
    themeSource: hasOverrides ? 'customized' : baseState.selectedThemePresetId ? 'preset' : 'customized',
  };
}

export type ThemeWorkbenchDraftStore = {
  activeThemeWorkbenchGroup: ThemeWorkbenchGroupKey;
  themeWorkbenchStyleConfigBaseline: WorkbenchStyleConfigSnapshot | null;
  themeDraftBaseline: ThemeAuthorityState | null;
  themeDraft: ThemeAuthorityState | null;
  themeDraftApplied: boolean;
  themeResetting: boolean;
  themeAuthorityLastModifiedAt: string | null;
  readonly hasThemeWorkbenchPendingChanges: boolean;
  createThemeAuthoritySnapshot(): ThemeAuthorityState;
  createWorkbenchStyleConfigSnapshot(): WorkbenchStyleConfigSnapshot;
  assignThemeAuthorityState(nextState: ThemeAuthorityState): void;
  assignWorkbenchStyleConfigSnapshot(snapshot: WorkbenchStyleConfigSnapshot): void;
  changeMode(mode: ModeType | 'auto'): void;
  syncThemeWorkbenchVisibility(visible: boolean): void;
};

export function openThemeWorkbenchDraft(store: ThemeWorkbenchDraftStore, group?: ThemeWorkbenchGroupKey) {
  store.syncThemeWorkbenchVisibility(true);

  if (!store.themeWorkbenchStyleConfigBaseline) {
    store.themeWorkbenchStyleConfigBaseline = store.createWorkbenchStyleConfigSnapshot();
  }

  if (!store.themeDraft) {
    const snapshot = store.createThemeAuthoritySnapshot();
    store.themeDraftBaseline = snapshot;
    store.themeDraft = snapshot;
    store.themeDraftApplied = false;
  }

  if (group) {
    store.activeThemeWorkbenchGroup = group;
  }
}

export function closeThemeWorkbenchDraft(store: ThemeWorkbenchDraftStore) {
  if (store.themeWorkbenchStyleConfigBaseline) {
    store.assignWorkbenchStyleConfigSnapshot(store.themeWorkbenchStyleConfigBaseline);
  }

  if (store.themeDraftBaseline && store.themeDraftApplied) {
    store.assignThemeAuthorityState(store.themeDraftBaseline);
    store.changeMode(store.themeDraftBaseline.mode);
  }

  store.syncThemeWorkbenchVisibility(false);
  store.themeWorkbenchStyleConfigBaseline = null;
  store.themeDraftBaseline = null;
  store.themeDraft = null;
  store.themeDraftApplied = false;
  store.themeResetting = false;
}

export function beginThemeWorkbenchDraft(store: ThemeWorkbenchDraftStore) {
  const snapshot = store.createThemeAuthoritySnapshot();
  store.themeWorkbenchStyleConfigBaseline = store.createWorkbenchStyleConfigSnapshot();
  store.themeDraftBaseline = snapshot;
  store.themeDraft = snapshot;
  store.themeDraftApplied = false;
}

export function previewThemeWorkbenchDraft(store: ThemeWorkbenchDraftStore) {
  if (!store.themeDraft) {
    return;
  }

  store.assignThemeAuthorityState(store.themeDraft);
  store.changeMode(store.themeDraft.mode);
  store.themeDraftApplied = true;
}

export function applyThemeWorkbenchDraft(store: ThemeWorkbenchDraftStore) {
  if (!store.themeDraft) {
    return;
  }

  const hasPendingChanges = store.hasThemeWorkbenchPendingChanges;
  store.assignThemeAuthorityState(store.themeDraft);
  if (hasPendingChanges) {
    store.themeAuthorityLastModifiedAt = new Date().toISOString();
  }
  store.changeMode(store.themeDraft.mode);
  store.themeWorkbenchStyleConfigBaseline = null;
  store.themeDraftBaseline = null;
  store.themeDraft = null;
  store.themeDraftApplied = false;
  store.syncThemeWorkbenchVisibility(false);
}

export function resetThemeWorkbenchDraftToDefault(
  store: ThemeWorkbenchDraftStore,
  defaultPreset: ThemePresetDefinition,
  options: { preserveResettingFeedback?: boolean } = {},
) {
  if (!store.themeDraftBaseline) {
    store.themeDraftBaseline = store.createThemeAuthoritySnapshot();
  }

  const current = store.themeDraft ?? store.createThemeAuthoritySnapshot();
  store.themeDraft = createCompleteThemePresetState(defaultPreset, current.mode);
  previewThemeWorkbenchDraft(store);
  if (!options.preserveResettingFeedback) {
    store.themeResetting = false;
  }
}
