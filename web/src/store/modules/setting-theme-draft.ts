import type {
  ThemeAuthorityState,
  ThemeModeTokenState,
  ThemePresetDefinition,
  ThemeTokenMap,
  ThemeWorkbenchGroupKey,
} from '@/types/theme';
import { cloneThemeModeTokenState, createEmptyThemeModeTokenState } from '@/utils/theme-workbench';
import type { ModeType } from '@/utils/types';

import type { WorkbenchStyleConfigSnapshot } from './setting-theme-authority';

function createDefaultThemeAuthorityState(
  mode: ModeType | 'auto',
  brandTheme: string,
  selectedThemePresetId: string,
): ThemeAuthorityState {
  return {
    mode,
    brandTheme,
    selectedThemePresetId,
    themeSource: 'preset',
    fontFamilyPreset: 'system',
    fontSizePreset: 'standard',
    radiusPreset: 'standard',
    shadowPreset: 'standard',
    densityPreset: 'standard',
    themeTokenOverrides: createEmptyThemeModeTokenState(),
  };
}

export function buildUpdatedThemeDraft(
  base: ThemeAuthorityState,
  patch: Partial<ThemeAuthorityState>,
): ThemeAuthorityState {
  return {
    ...base,
    ...patch,
    themeTokenOverrides: patch.themeTokenOverrides
      ? cloneThemeModeTokenState(patch.themeTokenOverrides)
      : cloneThemeModeTokenState(base.themeTokenOverrides),
  };
}

export function buildSelectedThemePresetState(
  preset: ThemePresetDefinition,
  draftState: ThemeAuthorityState | null,
  persistedState: Pick<
    ThemeAuthorityState,
    'mode' | 'fontFamilyPreset' | 'fontSizePreset' | 'radiusPreset' | 'shadowPreset' | 'densityPreset'
  >,
): ThemeAuthorityState {
  return {
    mode: preset.authorityPatch?.mode ?? preset.mode ?? draftState?.mode ?? persistedState.mode,
    brandTheme: preset.brandTheme,
    selectedThemePresetId: preset.id,
    themeSource: 'preset',
    fontFamilyPreset:
      preset.authorityPatch?.fontFamilyPreset ?? draftState?.fontFamilyPreset ?? persistedState.fontFamilyPreset,
    fontSizePreset:
      preset.authorityPatch?.fontSizePreset ?? draftState?.fontSizePreset ?? persistedState.fontSizePreset,
    radiusPreset: preset.authorityPatch?.radiusPreset ?? draftState?.radiusPreset ?? persistedState.radiusPreset,
    shadowPreset: preset.authorityPatch?.shadowPreset ?? draftState?.shadowPreset ?? persistedState.shadowPreset,
    densityPreset: preset.authorityPatch?.densityPreset ?? draftState?.densityPreset ?? persistedState.densityPreset,
    themeTokenOverrides: createEmptyThemeModeTokenState(),
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
  defaultMode: ModeType | 'auto',
  defaultBrandTheme: string,
  defaultPresetId: string,
  options: { preserveResettingFeedback?: boolean } = {},
) {
  if (!store.themeDraftBaseline) {
    store.themeDraftBaseline = store.createThemeAuthoritySnapshot();
  }

  store.themeDraft = createDefaultThemeAuthorityState(defaultMode, defaultBrandTheme, defaultPresetId);
  previewThemeWorkbenchDraft(store);
  if (!options.preserveResettingFeedback) {
    store.themeResetting = false;
  }
}
