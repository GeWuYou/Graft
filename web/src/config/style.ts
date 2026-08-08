export const TAB_INDICATOR_POSITIONS = ['none', 'top', 'bottom'] as const;
export type TabIndicatorPosition = (typeof TAB_INDICATOR_POSITIONS)[number];

export default {
  showFooter: true,
  isSidebarCompact: false,
  showBreadcrumb: false,
  menuAutoCollapsed: false,
  menuAlwaysExpanded: false,
  mode: 'dark',
  layout: 'mix',
  splitMenu: true,
  sideMode: 'dark',
  isFooterAside: false,
  isSidebarFixed: true,
  isHeaderFixed: true,
  isUseTabsRouter: true,
  tabIndicatorPosition: 'none' as TabIndicatorPosition,
  showHeader: true,
  showThemeWorkbenchDock: true,
  isAcrylicEnabled: true,
  // 仅决定下一次应用预设的范围，不参与运行时主题 token 合成。
  themePresetApplicationScope: 'palette' as 'palette' | 'complete',
  brandTheme: '#61AFEF',
};
