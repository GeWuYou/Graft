import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import ProjectListPage from './index.vue';

const projectApiMocks = vi.hoisted(() => ({
  getProjects: vi.fn(),
  postProjectDown: vi.fn(),
  postProjectRefresh: vi.fn(),
  postProjectRestart: vi.fn(),
  postProjectUnregister: vi.fn(),
  postProjectUp: vi.fn(),
}));

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
  resolve: vi.fn(() => ({
    fullPath: '/ops/projects',
    path: '/ops/projects',
  })),
}));

const tabsRouterStoreMock = vi.hoisted(() => ({
  appendTab: vi.fn(),
}));

function slotStub(name: string) {
  return defineComponent({
    name,
    setup(_props, { slots }) {
      return () =>
        h('div', { 'data-stub': name }, [
          slots.meta?.(),
          slots.actions?.(),
          slots.filters?.(),
          slots.head?.(),
          slots.toolbar?.(),
          slots.footer?.(),
          slots.default?.(),
        ]);
    },
  });
}

vi.mock('../../api/project', () => ({
  getProjects: projectApiMocks.getProjects,
  postProjectDown: projectApiMocks.postProjectDown,
  postProjectRefresh: projectApiMocks.postProjectRefresh,
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
  useRouter: () => routerMocks,
}));

vi.mock('@/store/modules/tabs-router', () => ({
  useTabsRouterStore: () => tabsRouterStoreMock,
}));

vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    error: vi.fn(),
  }),
}));

vi.mock('@/utils/route/title', () => ({
  localizeRouteTitleKey: (key: string) => key,
}));

vi.mock('../../shared/navigation', () => ({
  appendResolvedTab: vi.fn(),
  buildDetailTitleWithFallback: (key: string) => key,
}));

vi.mock('@/shared/components/management', () => ({
  ManagementEmptyState: slotStub('ManagementEmptyState'),
  ManagementPageContent: slotStub('ManagementPageContent'),
  ManagementPageHeader: slotStub('ManagementPageHeader'),
  ManagementTableCard: slotStub('ManagementTableCard'),
  ManagementTablePagination: slotStub('ManagementTablePagination'),
  ManagementToolbar: slotStub('ManagementToolbar'),
  TableActionMenu: slotStub('TableActionMenu'),
  TableViewToolbar: slotStub('TableViewToolbar'),
  resolveTableWidthPolicy: () => ({ mode: 'fit', tableContentWidth: 'auto' }),
  useTableHostWidth: () => ({ tableHostRef: ref(null), tableHostWidth: ref(1280) }),
}));

describe('Project list page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    projectApiMocks.getProjects.mockResolvedValue({
      items: [
        {
          canonical_project_name: 'alpha',
          container_counts: { running: 3, stopped: 0, transitioning: 0, issue: 0, total: 3 },
          display_name: 'Alpha',
          drift_status: 'clean',
          id: 1,
          last_refresh_at: '2026-07-03T10:00:00Z',
          last_refresh_status: 'success',
          runtime_status: 'running',
          service_count: 3,
          source_kind: 'imported',
          working_directory: '/srv/alpha',
        },
        {
          canonical_project_name: 'beta',
          container_counts: { running: 0, stopped: 2, transitioning: 0, issue: 0, total: 2 },
          display_name: 'Beta',
          drift_status: 'clean',
          id: 2,
          last_refresh_at: '2026-07-03T10:05:00Z',
          last_refresh_status: 'success',
          runtime_status: 'degraded',
          service_count: 2,
          source_kind: 'managed',
          working_directory: '/srv/beta',
        },
        {
          canonical_project_name: 'gamma',
          container_counts: { running: 0, stopped: 1, transitioning: 1, issue: 0, total: 2 },
          display_name: 'Gamma',
          drift_status: 'clean',
          id: 3,
          last_refresh_at: '2026-07-03T10:10:00Z',
          last_refresh_status: 'failed',
          runtime_status: 'transitioning',
          service_count: 2,
          source_kind: 'git',
          working_directory: '/srv/gamma',
        },
        {
          canonical_project_name: 'delta',
          container_counts: { running: 0, stopped: 1, transitioning: 0, issue: 1, total: 1 },
          display_name: 'Delta',
          drift_status: 'unknown',
          id: 4,
          last_refresh_at: null,
          last_refresh_status: 'never',
          runtime_status: 'unknown',
          service_count: 1,
          source_kind: 'template',
          working_directory: '/srv/delta',
        },
      ],
      limit: 20,
      offset: 0,
      total: 4,
    });
  });

  it('renders only non-zero runtime summary badges in the header', async () => {
    const wrapper = mount(ProjectListPage, {
      global: {
        renderStubDefaultSlot: true,
        stubs: {
          'project-list-entry-actions': slotStub('ProjectListEntryActions'),
          'refresh-icon': true,
          't-button': slotStub('TButton'),
          't-checkbox': slotStub('TCheckbox'),
          't-checkbox-group': slotStub('TCheckboxGroup'),
          't-drawer': slotStub('TDrawer'),
          't-empty': slotStub('TEmpty'),
          't-input': slotStub('TInput'),
          't-option': slotStub('TOption'),
          't-pagination': slotStub('TPagination'),
          't-select': slotStub('TSelect'),
          't-space': slotStub('TSpace'),
          't-table': slotStub('TTable'),
          't-tag': slotStub('TTag'),
          't-tooltip': slotStub('TTooltip'),
        },
      },
    });

    await flushPromises();

    expect(projectApiMocks.getProjects).toHaveBeenCalledTimes(1);
    expect(wrapper.find('[data-testid="project-status-summary-total"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-status-summary-running"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-status-summary-degraded"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-status-summary-transitioning"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-status-summary-unknown"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-status-summary-stopped"]').exists()).toBe(false);
  });
});
