export type DebugFlagId =
  | 'tabs'
  | 'tabs.layout'
  | 'tabs.store'
  | 'project.logs'
  | 'project.monaco'
  | 'project.templates'
  | 'project.workspace'
  | 'container.raw-json';

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
    flagId: 'project.logs',
    envKeys: ['VITE_DEBUG_PROJECT_LOGS'],
    owner: 'project log snapshot and realtime runtime',
    summary: '项目日志 HTTP 快照、实时订阅、批处理与视图归一化诊断日志。',
    relatedPaths: [
      'src/modules/project/pages/detail/index.vue',
      'src/modules/project/shared/project-log-debug.ts',
      'src/modules/project/shared/project-log-realtime-batcher.ts',
    ],
    defaultEnabled: false,
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
  {
    flagId: 'project.workspace',
    envKeys: ['VITE_DEBUG_PROJECT_WORKSPACE'],
    owner: 'project creation and configuration workspace runtime',
    summary: '项目工作台会话、文件树、标签缓冲区与共享编辑器渲染状态诊断日志。',
    relatedPaths: [
      'src/modules/project/store/workspace.ts',
      'src/modules/project/components/ProjectCreateWorkspaceEditor.vue',
      'src/modules/project/components/ProjectWorkspaceEditor.vue',
    ],
    defaultEnabled: false,
  },
  {
    flagId: 'project.templates',
    envKeys: ['VITE_DEBUG_PROJECT_TEMPLATES'],
    owner: 'project template catalog and detail runtime',
    summary: '应用模板创建、详情路由与详情加载诊断日志。',
    relatedPaths: [
      'src/modules/project/pages/templates/index.vue',
      'src/modules/project/pages/template-detail/index.vue',
      'src/modules/project/shared/project-template-debug.ts',
    ],
    defaultEnabled: false,
  },
  {
    flagId: 'container.raw-json',
    envKeys: ['VITE_DEBUG_CONTAINER_RAW_JSON'],
    owner: 'container detail JSON tree viewer',
    summary: '容器详情 JSON 树数据刷新与展开状态保留诊断日志。',
    relatedPaths: ['src/modules/container/components/JsonTreeViewer.vue'],
    defaultEnabled: false,
  },
] as const satisfies readonly DebugFlagDefinition[];

export const DEBUG_FLAG_MAP = new Map<DebugFlagId, DebugFlagDefinition>(
  DEBUG_FLAG_REGISTRY.map((definition) => [definition.flagId, definition]),
);
