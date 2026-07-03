import { flushPromises, shallowMount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ref } from 'vue';

import ProjectDetailPage from './index.vue';

const containerApiMocks = vi.hoisted(() => ({
  getContainerEvents: vi.fn(),
  getContainerLogs: vi.fn(),
}));

const projectApiMocks = vi.hoisted(() => ({
  getProject: vi.fn(),
  getProjectConfiguration: vi.fn(),
  getProjectConfigurationFile: vi.fn(),
  getProjectConfigurationPreview: vi.fn(),
  getProjectServices: vi.fn(),
  postProjectConfigurationDiff: vi.fn(),
  postProjectConfigurationValidate: vi.fn(),
  postProjectDeploy: vi.fn(),
  postProjectDown: vi.fn(),
  postProjectRestart: vi.fn(),
  postProjectUnregister: vi.fn(),
  postProjectUp: vi.fn(),
}));

const tabsRouterStoreMock = vi.hoisted(() => ({
  tabRouterList: [
    {
      fullPath: '/ops/projects/7',
      path: '/ops/projects/7',
      tabKey: '/ops/projects/7',
      title: { 'en-US': 'Project Detail', 'zh-CN': '项目详情' },
    },
  ],
}));

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
  replace: vi.fn(),
  resolve: vi.fn((target: { params?: { id?: string } }) => ({
    fullPath: `/ops/containers/${target.params?.id ?? ''}`,
    path: `/ops/containers/${target.params?.id ?? ''}`,
  })),
}));

vi.mock('@/modules/container/api/container', () => ({
  getContainerEvents: containerApiMocks.getContainerEvents,
  getContainerLogs: containerApiMocks.getContainerLogs,
}));

vi.mock('../../api/project', () => ({
  getProject: projectApiMocks.getProject,
  getProjectConfiguration: projectApiMocks.getProjectConfiguration,
  getProjectConfigurationFile: projectApiMocks.getProjectConfigurationFile,
  getProjectConfigurationPreview: projectApiMocks.getProjectConfigurationPreview,
  getProjectServices: projectApiMocks.getProjectServices,
  postProjectConfigurationDiff: projectApiMocks.postProjectConfigurationDiff,
  postProjectConfigurationValidate: projectApiMocks.postProjectConfigurationValidate,
  postProjectDeploy: projectApiMocks.postProjectDeploy,
  postProjectDown: projectApiMocks.postProjectDown,
  postProjectRestart: projectApiMocks.postProjectRestart,
  postProjectUnregister: projectApiMocks.postProjectUnregister,
  postProjectUp: projectApiMocks.postProjectUp,
}));

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>();
  const locale = ref('en-US');
  return {
    ...actual,
    useI18n: () => ({
      locale,
      t: (key: string) => key,
    }),
  };
});

vi.mock('vue-router', () => ({
  useRoute: () => ({
    fullPath: '/ops/projects/7',
    params: { id: '7' },
    path: '/ops/projects/7',
    query: {},
  }),
  useRouter: () => routerMocks,
}));

vi.mock('@/store/modules/tabs-router', () => ({
  useTabsRouterStore: () => tabsRouterStoreMock,
}));

vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

vi.mock('@/shared/observability', () => ({
  copyText: vi.fn(),
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    error: vi.fn(),
  }),
}));

vi.mock('../../shared/display', () => ({
  formatProjectTime: (_locale: string, value?: string | null) => value || '-',
  projectCanonicalNameSourceLabel: () => 'explicit',
  projectDriftStatusLabel: () => 'in-sync',
  projectDriftStatusTheme: () => 'success',
  projectHostScopeLabel: () => 'local',
  projectOwnershipModeLabel: () => 'external',
  projectRefreshStatusLabel: () => 'success',
  projectRefreshStatusTheme: () => 'success',
  projectRuntimeStatusLabel: () => 'running',
  projectRuntimeStatusTheme: () => 'success',
}));

describe('Project detail page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    tabsRouterStoreMock.tabRouterList = [
      {
        fullPath: '/ops/projects/7',
        path: '/ops/projects/7',
        tabKey: '/ops/projects/7',
        title: { 'en-US': 'Project Detail', 'zh-CN': '项目详情' },
      },
    ];
    projectApiMocks.getProject.mockResolvedValue({
      activity_authority: 'frontend-fanout',
      canonical_project_name: 'compose-demo',
      canonical_project_name_source: 'explicit',
      compose_files: [],
      container_counts: { running: 1, total: 1 },
      display_name: 'Compose Demo',
      drift_status: 'in-sync',
      env_files: [],
      host_scope: 'local',
      id: 7,
      last_observed_config_hash: null,
      last_refresh_at: '2026-07-03T10:00:00Z',
      last_refresh_config_hash: null,
      last_refresh_status: 'success',
      ownership_mode: 'external',
      runtime_status: 'running',
      service_count: 1,
      working_directory: '/srv/compose-demo',
    });
    projectApiMocks.getProjectConfiguration.mockResolvedValue({
      compose_files: [],
      diagnostics_summary: [],
      drift_status: 'in-sync',
      env_files: [],
      last_refresh_status: 'success',
      ownership_mode: 'external',
    });
    projectApiMocks.getProjectConfigurationPreview.mockResolvedValue(null);
    projectApiMocks.getProjectServices.mockResolvedValue({
      items: [
        {
          build_context: null,
          container_members: [{ container_id: 'container-1', container_name: 'compose-demo-1', state: 'running' }],
          declared_networks: [],
          declared_ports: [],
          declared_volumes: [],
          image: 'demo:latest',
          running_count: 1,
          service_name: 'app',
          stopped_count: 0,
        },
      ],
    });
    containerApiMocks.getContainerEvents.mockResolvedValue({
      items: [],
    });
    containerApiMocks.getContainerLogs.mockResolvedValue({
      entries: [],
    });
  });

  it('loads activity fan-out on initial detail refresh', async () => {
    shallowMount(ProjectDetailPage);
    await flushPromises();

    expect(projectApiMocks.getProject).toHaveBeenCalledWith(7);
    expect(projectApiMocks.getProjectConfiguration).toHaveBeenCalledWith(7);
    expect(projectApiMocks.getProjectConfigurationPreview).toHaveBeenCalledWith(7);
    expect(projectApiMocks.getProjectServices).toHaveBeenCalledTimes(1);
    expect(projectApiMocks.getProjectServices).toHaveBeenCalledWith(7);
    expect(containerApiMocks.getContainerEvents).toHaveBeenCalledWith('container-1');
    expect(containerApiMocks.getContainerLogs).toHaveBeenCalledWith('container-1', {
      since: '1h',
      stderr: true,
      stdout: true,
      tail: 40,
      timestamps: true,
    });
  });
});
