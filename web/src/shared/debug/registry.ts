export type DebugFlagId = 'tabs' | 'tabs.layout' | 'tabs.store' | 'project.monaco';

export type DebugFlagDefinition = {
  defaultEnabled: boolean;
  summary: string;
  envKeys?: readonly (keyof ImportMetaEnv)[];
  flagId: DebugFlagId;
  owner: string;
  parentFlagId?: 'tabs';
  relatedPaths: string[];
};

export const DEBUG_FLAG_REGISTRY = [
  {
    flagId: 'tabs',
    envKeys: ['VITE_DEBUG_TABS'],
    owner: 'layout/store tabs runtime',
    summary: '多标签页调试总开关，覆盖 tabs layout 与 tabs store 链路。',
    relatedPaths: ['src/utils/tabs-debug.ts', 'src/layouts/index.vue', 'src/store/modules/tabs-router.ts'],
    defaultEnabled: false,
  },
  {
    flagId: 'tabs.layout',
    envKeys: ['VITE_DEBUG_TABS_LAYOUT'],
    owner: 'layout tabs runtime',
    summary: 'layout.tabs 路由追加、恢复与激活链路调试日志。',
    relatedPaths: ['src/layouts/index.vue', 'src/utils/tabs-debug.ts'],
    defaultEnabled: false,
    parentFlagId: 'tabs',
  },
  {
    flagId: 'tabs.store',
    envKeys: ['VITE_DEBUG_TABS_STORE'],
    owner: 'tabs router store',
    summary: 'store.tabsRouter 恢复、追加、刷新、固定标签链路调试日志。',
    relatedPaths: ['src/store/modules/tabs-router.ts', 'src/utils/tabs-debug.ts'],
    defaultEnabled: false,
    parentFlagId: 'tabs',
  },
  {
    flagId: 'project.monaco',
    envKeys: ['VITE_DEBUG_PROJECT_MONACO'],
    owner: 'project monaco runtime',
    summary: 'Monaco worker、surface、diff、layout 与模型处置调试日志。',
    relatedPaths: [
      'src/modules/project/shared/project-monaco-debug.ts',
      'src/modules/project/shared/project-monaco.ts',
      'src/modules/project/components/ProjectMonacoSurface.vue',
    ],
    defaultEnabled: false,
  },
] as const satisfies readonly DebugFlagDefinition[];

export const DEBUG_FLAG_MAP = new Map<DebugFlagId, DebugFlagDefinition>(
  DEBUG_FLAG_REGISTRY.map((definition) => [definition.flagId, definition]),
);
