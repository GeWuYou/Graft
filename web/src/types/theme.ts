import type STYLE_CONFIG from '@/config/style';
import type { ModeType } from '@/utils/types';

export type SettingStyleConfig = typeof STYLE_CONFIG;

export type ThemeWorkbenchGroupKey =
  | 'overview'
  | 'brand'
  | 'semantic'
  | 'neutral'
  | 'font'
  | 'radius'
  | 'shadow'
  | 'size';

export type ThemeTokenGroupKey = Exclude<ThemeWorkbenchGroupKey, 'overview'>;

export type ThemeSourceType = 'preset' | 'custom';

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
