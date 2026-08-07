import type STYLE_CONFIG from '@/config/style';
import type { ModeType } from '@/utils/types';

export type SettingStyleConfig = typeof STYLE_CONFIG;

export type ThemeWorkbenchGroupKey =
  'overview' | 'presets' | 'appearance' | 'layout' | 'typography' | 'style' | 'advanced';

export type ThemeTokenGroupKey = 'brand' | 'text' | 'background' | 'border' | 'chart' | 'component' | 'material';

/** 预设应用范围只在工作台当前交互中生效，不作为主题视觉状态持久化。 */
export type ThemePresetApplicationScope = 'palette' | 'complete';

export type ThemeSourceType = 'preset' | 'customized';

export type ThemePresetCategory = 'balanced' | 'focused' | 'operations' | 'night';

export type ThemeTokenMap = Record<string, string>;

export type ThemeWorkbenchAuthorityPatch = Partial<
  Pick<
    ThemeAuthorityState,
    'mode' | 'fontFamilyPreset' | 'fontSizePreset' | 'radiusPreset' | 'shadowPreset' | 'densityPreset'
  >
>;

export type ThemeWorkbenchStylePatch = Partial<
  Pick<
    SettingStyleConfig,
    | 'layout'
    | 'splitMenu'
    | 'isSidebarFixed'
    | 'isUseTabsRouter'
    | 'showFooter'
    | 'showHeader'
    | 'showBreadcrumb'
    | 'menuAutoCollapsed'
    | 'menuAlwaysExpanded'
    | 'isAcrylicEnabled'
  >
>;

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
  key:
    | 'brandTheme'
    | 'fontFamilyPreset'
    | 'fontSizePreset'
    | 'radiusPreset'
    | 'shadowPreset'
    | 'densityPreset'
    | 'themeTokenOverrides';
  labelKey: string;
  fromValue: string;
  toValue: string;
}

export interface ThemeIdentitySummary {
  currentLabelKey: string;
  sourceLabelKey: string;
  sourceType: ThemeSourceType;
  modifiedCount: number;
  lastModifiedAt: string | null;
}

export interface ThemeTokenDefinition {
  key: string;
  group: ThemeTokenGroupKey;
  labelKey: string;
  followsBrandColor?: boolean;
}

export interface ThemePresetDefinition {
  id: string;
  labelKey: string;
  descriptionKey: string;
  category: ThemePresetCategory;
  featured?: boolean;
  brandTheme: string;
  mode?: ModeType | 'auto';
  tokenOverrides?: Partial<ThemeModeTokenState>;
  materialTokenOverrides?: Partial<ThemeModeTokenState>;
  authorityPatch?: ThemeWorkbenchAuthorityPatch;
  stylePatch?: ThemeWorkbenchStylePatch;
}

export interface ThemeWorkbenchScenarioPresetDefinition {
  id: string;
  labelKey: string;
  descriptionKey: string;
  presetId?: string | null;
  authorityPatch?: ThemeWorkbenchAuthorityPatch;
  stylePatch?: ThemeWorkbenchStylePatch;
}

export interface ThemeAuthorityState {
  mode: ModeType | 'auto';
  brandTheme: string;
  selectedThemePresetId: string | null;
  themeSource: ThemeSourceType;
  fontFamilyPreset: 'system' | 'harmonyos' | 'inter' | 'source-han-sans';
  fontSizePreset: 'extra-small' | 'small' | 'standard' | 'large' | 'extra-large';
  radiusPreset: 'square' | 'business' | 'standard' | 'rounded' | 'capsule';
  shadowPreset: 'flat' | 'standard' | 'floating' | 'hard-offset';
  densityPreset: 'compact' | 'standard' | 'comfortable';
  themeTokenOverrides: ThemeModeTokenState;
}
