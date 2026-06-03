import type { ThemePresetDefinition, ThemeTokenDefinition, ThemeWorkbenchGroupDefinition } from '@/types/theme';

export const DEFAULT_THEME_PRESET_ID = 'tdesign-default';

export const THEME_WORKBENCH_GROUPS: ThemeWorkbenchGroupDefinition[] = [
  { key: 'overview', labelKey: 'layout.setting.workbench.groups.overview' },
  { key: 'appearance', labelKey: 'layout.setting.workbench.groups.appearance' },
  { key: 'layout', labelKey: 'layout.setting.workbench.groups.layout' },
  { key: 'typography', labelKey: 'layout.setting.workbench.groups.typography' },
  { key: 'style', labelKey: 'layout.setting.workbench.groups.style' },
  { key: 'advanced', labelKey: 'layout.setting.workbench.groups.advanced' },
];

export const THEME_TOKEN_DEFINITIONS: ThemeTokenDefinition[] = [
  {
    key: '--td-brand-color',
    group: 'brand',
    labelKey: 'layout.setting.workbench.tokenDefinitions.primaryBrand',
    followsBrandColor: true,
  },
  {
    key: '--td-brand-color-1',
    group: 'brand',
    labelKey: 'layout.setting.workbench.tokenDefinitions.brandScale1',
    followsBrandColor: true,
  },
  {
    key: '--td-brand-color-2',
    group: 'brand',
    labelKey: 'layout.setting.workbench.tokenDefinitions.brandScale2',
    followsBrandColor: true,
  },
  {
    key: '--td-brand-color-3',
    group: 'brand',
    labelKey: 'layout.setting.workbench.tokenDefinitions.brandScale3',
    followsBrandColor: true,
  },
  {
    key: '--td-brand-color-4',
    group: 'brand',
    labelKey: 'layout.setting.workbench.tokenDefinitions.brandScale4',
    followsBrandColor: true,
  },
  {
    key: '--td-brand-color-5',
    group: 'brand',
    labelKey: 'layout.setting.workbench.tokenDefinitions.brandScale5',
    followsBrandColor: true,
  },
  {
    key: '--td-brand-color-6',
    group: 'brand',
    labelKey: 'layout.setting.workbench.tokenDefinitions.brandScale6',
    followsBrandColor: true,
  },
  {
    key: '--td-brand-color-7',
    group: 'brand',
    labelKey: 'layout.setting.workbench.tokenDefinitions.brandScale7',
    followsBrandColor: true,
  },
  {
    key: '--td-brand-color-8',
    group: 'brand',
    labelKey: 'layout.setting.workbench.tokenDefinitions.brandScale8',
    followsBrandColor: true,
  },
  {
    key: '--td-brand-color-9',
    group: 'brand',
    labelKey: 'layout.setting.workbench.tokenDefinitions.brandScale9',
    followsBrandColor: true,
  },
  {
    key: '--td-brand-color-10',
    group: 'brand',
    labelKey: 'layout.setting.workbench.tokenDefinitions.brandScale10',
    followsBrandColor: true,
  },
  { key: '--td-text-color-primary', group: 'text', labelKey: 'layout.setting.workbench.tokenDefinitions.primaryText' },
  {
    key: '--td-text-color-secondary',
    group: 'text',
    labelKey: 'layout.setting.workbench.tokenDefinitions.secondaryText',
  },
  {
    key: '--td-text-color-placeholder',
    group: 'text',
    labelKey: 'layout.setting.workbench.tokenDefinitions.placeholderText',
  },
  {
    key: '--td-bg-color-page',
    group: 'background',
    labelKey: 'layout.setting.workbench.tokenDefinitions.pageBackground',
  },
  {
    key: '--td-bg-color-container',
    group: 'background',
    labelKey: 'layout.setting.workbench.tokenDefinitions.containerBackground',
  },
  {
    key: '--td-bg-color-container-hover',
    group: 'background',
    labelKey: 'layout.setting.workbench.tokenDefinitions.containerHoverBackground',
  },
  {
    key: '--td-component-border',
    group: 'border',
    labelKey: 'layout.setting.workbench.tokenDefinitions.componentBorder',
  },
  {
    key: '--td-component-stroke',
    group: 'border',
    labelKey: 'layout.setting.workbench.tokenDefinitions.componentStroke',
  },
  {
    key: '--td-border-level-1-color',
    group: 'border',
    labelKey: 'layout.setting.workbench.tokenDefinitions.level1Border',
  },
  { key: '--td-scrollbar-color', group: 'border', labelKey: 'layout.setting.workbench.tokenDefinitions.scrollbar' },
  {
    key: '--td-success-color',
    group: 'component',
    labelKey: 'layout.setting.workbench.tokenDefinitions.successFeedback',
  },
  {
    key: '--td-error-color',
    group: 'component',
    labelKey: 'layout.setting.workbench.tokenDefinitions.errorFeedback',
  },
  {
    key: '--td-warning-color',
    group: 'component',
    labelKey: 'layout.setting.workbench.tokenDefinitions.warningFeedback',
  },
  { key: '--td-font-family', group: 'component', labelKey: 'layout.setting.workbench.tokenDefinitions.fontFamily' },
  {
    key: '--td-font-body-medium',
    group: 'component',
    labelKey: 'layout.setting.workbench.tokenDefinitions.bodyMedium',
  },
  {
    key: '--td-font-title-medium',
    group: 'component',
    labelKey: 'layout.setting.workbench.tokenDefinitions.titleMedium',
  },
  {
    key: '--td-font-title-large',
    group: 'component',
    labelKey: 'layout.setting.workbench.tokenDefinitions.titleLarge',
  },
  { key: '--td-radius-small', group: 'component', labelKey: 'layout.setting.workbench.tokenDefinitions.smallRadius' },
  {
    key: '--td-radius-default',
    group: 'component',
    labelKey: 'layout.setting.workbench.tokenDefinitions.defaultRadius',
  },
  {
    key: '--td-radius-medium',
    group: 'component',
    labelKey: 'layout.setting.workbench.tokenDefinitions.mediumRadius',
  },
  { key: '--td-radius-large', group: 'component', labelKey: 'layout.setting.workbench.tokenDefinitions.largeRadius' },
  {
    key: '--td-radius-extraLarge',
    group: 'component',
    labelKey: 'layout.setting.workbench.tokenDefinitions.extraLargeRadius',
  },
  {
    key: '--td-radius-circle',
    group: 'component',
    labelKey: 'layout.setting.workbench.tokenDefinitions.circleRadius',
  },
  { key: '--td-shadow-1', group: 'component', labelKey: 'layout.setting.workbench.tokenDefinitions.shadowLevel1' },
  { key: '--td-shadow-2', group: 'component', labelKey: 'layout.setting.workbench.tokenDefinitions.shadowLevel2' },
  { key: '--td-shadow-3', group: 'component', labelKey: 'layout.setting.workbench.tokenDefinitions.shadowLevel3' },
  {
    key: '--td-comp-size-xs',
    group: 'component',
    labelKey: 'layout.setting.workbench.tokenDefinitions.componentSizeXs',
  },
  {
    key: '--td-comp-size-s',
    group: 'component',
    labelKey: 'layout.setting.workbench.tokenDefinitions.componentSizeS',
  },
  {
    key: '--td-comp-size-m',
    group: 'component',
    labelKey: 'layout.setting.workbench.tokenDefinitions.componentSizeM',
  },
  {
    key: '--td-comp-size-l',
    group: 'component',
    labelKey: 'layout.setting.workbench.tokenDefinitions.componentSizeL',
  },
  {
    key: '--td-comp-size-xl',
    group: 'component',
    labelKey: 'layout.setting.workbench.tokenDefinitions.componentSizeXl',
  },
];

// 推荐主题保持为纯前端本地配置，先服务当前工作台预览，不与后端契约耦合。
export const THEME_PRESET_DEFINITIONS: ThemePresetDefinition[] = [
  {
    id: 'tdesign-default',
    labelKey: 'layout.setting.workbench.presets.tdesignDefault.label',
    descriptionKey: 'layout.setting.workbench.presets.tdesignDefault.description',
    brandTheme: '#0052D9',
  },
  {
    id: 'tencent-cloud',
    labelKey: 'layout.setting.workbench.presets.tencentCloud.label',
    descriptionKey: 'layout.setting.workbench.presets.tencentCloud.description',
    brandTheme: '#0064FF',
  },
  {
    id: 'mountain-green',
    labelKey: 'layout.setting.workbench.presets.mountainGreen.label',
    descriptionKey: 'layout.setting.workbench.presets.mountainGreen.description',
    brandTheme: '#2BA471',
  },
  {
    id: 'midnight-blue',
    labelKey: 'layout.setting.workbench.presets.midnightBlue.label',
    descriptionKey: 'layout.setting.workbench.presets.midnightBlue.description',
    brandTheme: '#3B82F6',
    mode: 'dark',
  },
];
