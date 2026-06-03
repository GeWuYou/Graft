import type STYLE_CONFIG from '@/config/style';
import type { ModeType } from '@/utils/types';

export type SettingStyleConfig = typeof STYLE_CONFIG;

export type ThemeWorkbenchGroupKey = 'overview' | 'appearance' | 'layout' | 'typography' | 'style' | 'advanced';

export type ThemeTokenGroupKey = 'brand' | 'text' | 'background' | 'border' | 'component';

export type ThemeSourceType = 'preset' | 'customized';

export type ThemeTokenMap = Record<string, string>;

export interface ThemeModeTokenState {
  light: ThemeTokenMap;
  dark: ThemeTokenMap;
}

export interface ThemeWorkbenchGroupDefinition {
  key: ThemeWorkbenchGroupKey;
  labelKey: string;
  descriptionKey?: string;
}

export interface ThemeAuthorityDiffItem {
  key: 'brandTheme' | 'fontFamilyPreset' | 'fontSizePreset' | 'radiusPreset' | 'shadowPreset' | 'densityPreset';
  labelKey: string;
  fromValue: string;
  toValue: string;
}

export interface ThemeIdentitySummary {
  currentLabel: string;
  sourceLabel: string;
  sourceType: ThemeSourceType;
  modifiedCount: number;
  lastModifiedAt: string | null;
}

export interface ThemeTokenDefinition {
  key: string;
  group: ThemeTokenGroupKey;
  label: string;
  followsBrandColor?: boolean;
}

export interface ThemePresetDefinition {
  id: string;
  label: string;
  description: string;
  brandTheme: string;
  mode?: ModeType | 'auto';
  tokenOverrides?: Partial<ThemeModeTokenState>;
}

export interface ThemeAuthorityState {
  mode: ModeType | 'auto';
  brandTheme: string;
  selectedThemePresetId: string | null;
  themeSource: ThemeSourceType;
  fontFamilyPreset: 'system' | 'harmonyos' | 'inter' | 'source-han-sans';
  fontSizePreset: 'extra-small' | 'small' | 'standard' | 'large' | 'extra-large';
  radiusPreset: 'business' | 'standard' | 'rounded' | 'capsule';
  shadowPreset: 'flat' | 'standard' | 'floating';
  densityPreset: 'compact' | 'standard' | 'comfortable';
  themeTokenOverrides: ThemeModeTokenState;
}
