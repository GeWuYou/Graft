import { DEFAULT_THEME_PRESET_ID } from '@/config/theme-workbench';
import type { ThemeModeTokenState, ThemePresetDefinition, ThemeTokenMap } from '@/types/theme';
import type { ModeType } from '@/utils/types';

/**
 * 创建空的双模式 token 容器，避免调用方在 light/dark 分支上做空判断。
 */
export function createEmptyThemeModeTokenState(): ThemeModeTokenState {
  return {
    light: {},
    dark: {},
  };
}

export function cloneThemeModeTokenState(tokens?: Partial<ThemeModeTokenState>): ThemeModeTokenState {
  return {
    light: { ...(tokens?.light ?? {}) },
    dark: { ...(tokens?.dark ?? {}) },
  };
}

export function mergeThemeTokenMaps(...sources: Array<ThemeTokenMap | undefined>): ThemeTokenMap {
  return sources.reduce<ThemeTokenMap>((merged, current) => {
    if (!current) {
      return merged;
    }

    return {
      ...merged,
      ...current,
    };
  }, {});
}

export function mergeThemeModeTokenState(
  base: ThemeModeTokenState,
  patch?: Partial<ThemeModeTokenState>,
): ThemeModeTokenState {
  return {
    light: mergeThemeTokenMaps(base.light, patch?.light),
    dark: mergeThemeTokenMaps(base.dark, patch?.dark),
  };
}

export function buildThemeModeSnapshot(options: {
  brandTokens: ThemeModeTokenState;
  preset?: ThemePresetDefinition | null;
  customTokens: ThemeModeTokenState;
}): ThemeModeTokenState {
  const { brandTokens, preset, customTokens } = options;

  return {
    light: mergeThemeTokenMaps(brandTokens.light, preset?.tokenOverrides?.light, customTokens.light),
    dark: mergeThemeTokenMaps(brandTokens.dark, preset?.tokenOverrides?.dark, customTokens.dark),
  };
}

export function resolvePresetId(presetId: string | null | undefined): string {
  return presetId ?? DEFAULT_THEME_PRESET_ID;
}

export function resolveModeTokens(tokens: ThemeModeTokenState, mode: ModeType): ThemeTokenMap {
  return mode === 'dark' ? tokens.dark : tokens.light;
}
