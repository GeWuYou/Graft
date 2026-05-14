import type { ThemePresetDefinition, ThemeTokenDefinition, ThemeWorkbenchGroupDefinition } from '@/types/theme';

export const DEFAULT_THEME_PRESET_ID = 'tdesign-default';

export const THEME_WORKBENCH_GROUPS: ThemeWorkbenchGroupDefinition[] = [
  { key: 'overview', labelKey: 'layout.setting.workbench.groups.overview' },
  { key: 'brand', labelKey: 'layout.setting.workbench.groups.brand' },
  { key: 'semantic', labelKey: 'layout.setting.workbench.groups.semantic' },
  { key: 'neutral', labelKey: 'layout.setting.workbench.groups.neutral' },
  { key: 'font', labelKey: 'layout.setting.workbench.groups.font' },
  { key: 'radius', labelKey: 'layout.setting.workbench.groups.radius' },
  { key: 'shadow', labelKey: 'layout.setting.workbench.groups.shadow' },
  { key: 'size', labelKey: 'layout.setting.workbench.groups.size' },
];

export const THEME_TOKEN_DEFINITIONS: ThemeTokenDefinition[] = [
  { key: '--td-brand-color', group: 'brand', label: 'Primary Brand', followsBrandColor: true },
  { key: '--td-brand-color-1', group: 'brand', label: 'Brand Scale 1', followsBrandColor: true },
  { key: '--td-brand-color-2', group: 'brand', label: 'Brand Scale 2', followsBrandColor: true },
  { key: '--td-brand-color-3', group: 'brand', label: 'Brand Scale 3', followsBrandColor: true },
  { key: '--td-brand-color-4', group: 'brand', label: 'Brand Scale 4', followsBrandColor: true },
  { key: '--td-brand-color-5', group: 'brand', label: 'Brand Scale 5', followsBrandColor: true },
  { key: '--td-brand-color-6', group: 'brand', label: 'Brand Scale 6', followsBrandColor: true },
  { key: '--td-brand-color-7', group: 'brand', label: 'Brand Scale 7', followsBrandColor: true },
  { key: '--td-brand-color-8', group: 'brand', label: 'Brand Scale 8', followsBrandColor: true },
  { key: '--td-brand-color-9', group: 'brand', label: 'Brand Scale 9', followsBrandColor: true },
  { key: '--td-brand-color-10', group: 'brand', label: 'Brand Scale 10', followsBrandColor: true },
  { key: '--td-success-color', group: 'semantic', label: 'Success Feedback' },
  { key: '--td-error-color', group: 'semantic', label: 'Error Feedback' },
  { key: '--td-warning-color', group: 'semantic', label: 'Warning Feedback' },
  { key: '--td-text-color-primary', group: 'neutral', label: 'Primary Text' },
  { key: '--td-text-color-secondary', group: 'neutral', label: 'Secondary Text' },
  { key: '--td-text-color-placeholder', group: 'neutral', label: 'Placeholder Text' },
  { key: '--td-bg-color-page', group: 'neutral', label: 'Page Background' },
  { key: '--td-bg-color-container', group: 'neutral', label: 'Container Background' },
  { key: '--td-bg-color-container-hover', group: 'neutral', label: 'Container Hover Background' },
  { key: '--td-component-border', group: 'neutral', label: 'Component Border' },
  { key: '--td-component-stroke', group: 'neutral', label: 'Component Stroke' },
  { key: '--td-border-level-1-color', group: 'neutral', label: 'Level 1 Border' },
  { key: '--td-scrollbar-color', group: 'neutral', label: 'Scrollbar' },
  { key: '--td-font-family', group: 'font', label: 'Font Family' },
  { key: '--td-font-body-medium', group: 'font', label: 'Body Medium' },
  { key: '--td-font-title-medium', group: 'font', label: 'Title Medium' },
  { key: '--td-font-title-large', group: 'font', label: 'Title Large' },
  { key: '--td-radius-small', group: 'radius', label: 'Small Radius' },
  { key: '--td-radius-default', group: 'radius', label: 'Default Radius' },
  { key: '--td-radius-medium', group: 'radius', label: 'Medium Radius' },
  { key: '--td-radius-large', group: 'radius', label: 'Large Radius' },
  { key: '--td-radius-extraLarge', group: 'radius', label: 'Extra Large Radius' },
  { key: '--td-radius-circle', group: 'radius', label: 'Circle Radius' },
  { key: '--td-shadow-1', group: 'shadow', label: 'Shadow Level 1' },
  { key: '--td-shadow-2', group: 'shadow', label: 'Shadow Level 2' },
  { key: '--td-shadow-3', group: 'shadow', label: 'Shadow Level 3' },
  { key: '--td-comp-size-xs', group: 'size', label: 'Component Size XS' },
  { key: '--td-comp-size-s', group: 'size', label: 'Component Size S' },
  { key: '--td-comp-size-m', group: 'size', label: 'Component Size M' },
  { key: '--td-comp-size-l', group: 'size', label: 'Component Size L' },
  { key: '--td-comp-size-xl', group: 'size', label: 'Component Size XL' },
];

// 推荐主题保持为纯前端本地配置，先服务当前工作台预览，不与后端契约耦合。
export const THEME_PRESET_DEFINITIONS: ThemePresetDefinition[] = [
  {
    id: 'tdesign-default',
    label: 'TDesign Original',
    description: '保留 starter 默认主题基线。',
    brandTheme: '#0052D9',
  },
  {
    id: 'tencent-cloud',
    label: 'Tencent Cloud',
    description: '收敛到更贴近腾讯云控制台的亮色主色。',
    brandTheme: '#0064FF',
    tokenOverrides: {
      light: {
        '--td-bg-color-page': '#f5f7ff',
        '--td-bg-color-container-hover': '#eef3ff',
      },
      dark: {
        '--td-bg-color-page': '#111827',
        '--td-bg-color-container': '#1b2333',
      },
    },
  },
  {
    id: 'mountain-green',
    label: 'Mountain Green',
    description: '用更温和的绿色主色与浅色容器强化信息层次。',
    brandTheme: '#2BA471',
    tokenOverrides: {
      light: {
        '--td-bg-color-page': '#f3faf7',
        '--td-bg-color-container-hover': '#e8f5ef',
      },
      dark: {
        '--td-bg-color-page': '#0f1f1a',
        '--td-bg-color-container': '#142922',
      },
    },
  },
  {
    id: 'midnight-blue',
    label: 'Midnight Blue',
    description: '默认以暗色工作区为主，适合长时间浏览控制台页面。',
    brandTheme: '#3B82F6',
    mode: 'dark',
    tokenOverrides: {
      dark: {
        '--td-bg-color-page': '#0b1220',
        '--td-bg-color-container': '#111b30',
        '--td-bg-color-container-hover': '#16233d',
      },
    },
  },
];
