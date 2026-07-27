import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { computed, defineComponent, h, reactive, ref } from 'vue';

import { queryClient } from '@/shared/query';

import AccessLogListIndex from './index.vue';

const mocks = vi.hoisted(() => ({
  getAccessLogs: vi.fn(),
  routerReplace: vi.fn(),
}));

const routeState = reactive<{ query: Record<string, unknown> }>({ query: {} });

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => ({ push: vi.fn(), replace: mocks.routerReplace }),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}));

vi.mock('tdesign-icons-vue-next', () => ({ ArrowLeftIcon: defineComponent({ setup: () => () => null }) }));
vi.mock('tdesign-vue-next', () => ({ MessagePlugin: { error: vi.fn() } }));

vi.mock('@/modules/app-log/contract/deep-link', () => ({ buildAppLogLocation: vi.fn() }));
vi.mock('@/modules/audit/contract/deep-link', () => ({ buildAuditRequestLocation: vi.fn() }));
vi.mock('@/modules/auth/store', () => ({ useAuthSessionStore: () => ({ userInfo: { username: 'admin' } }) }));
vi.mock('@/modules/monitor/contract/navigation', () => ({
  buildMonitorLocationFromOrigin: vi.fn(),
  parseMonitorOriginQuery: vi.fn(() => null),
}));

vi.mock('@/shared/components/management', () => ({
  TableViewToolbar: defineComponent({ setup: () => () => null }),
}));
vi.mock('@/shared/components/query-list', () => ({
  AdvancedQueryColumnDrawer: defineComponent({ setup: () => () => null }),
  AdvancedQueryListPage: defineComponent({
    setup(_, { slots }) {
      return () => h('main', [slots.filters?.(), slots.table?.(), slots.detail?.()]);
    },
  }),
  applySavedQueryViewPresentation: vi.fn(),
  normalizeSavedQueryView: vi.fn((value) => value),
  useSavedQueryViews: () => ({
    load: vi.fn(async () => true),
    selectedId: ref(undefined),
    selectedView: computed(() => undefined),
  }),
}));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));
vi.mock('@/shared/observability', () => ({
  assignEncodedSorters: vi.fn(),
  buildRecentHoursLocalRange: vi.fn(() => []),
  buildTodayLocalRange: vi.fn(() => []),
  createLogDetailErrorReporter: vi.fn(() => vi.fn()),
  createSingleSorter: vi.fn((field: string, direction: string) => ({ field, direction })),
  decodeSorters: vi.fn(() => []),
  encodeSorters: vi.fn(() => []),
  localDateTimeToUtcIso: vi.fn((value: string) => value),
  normalizePageStateRangeForRoute: vi.fn((value: string[]) => value),
  normalizeRouteRangeForPageState: vi.fn((value: unknown[]) => value.filter(Boolean)),
  normalizeSorters: vi.fn((value: unknown[]) => value),
  openLogDetailRow: vi.fn(),
  parseLogRouteQuery: vi.fn((query: Record<string, unknown>) => query),
  buildLogListLocation: vi.fn((_path: string, _keys: string[], query: Record<string, unknown>) => ({ query })),
  restartLogListQuery: vi.fn((config) => {
    config.activePreset.value = config.preset ?? 'all';
    config.pagination.value.current = 1;
    return config.updateRouteQuery();
  }),
}));
vi.mock('@/utils/logger', () => ({ createLogger: () => ({ error: vi.fn() }) }));

vi.mock('../../api/access-log', () => ({
  deleteAccessLogSavedView: vi.fn(),
  getAccessLogDetail: vi.fn(),
  getAccessLogs: mocks.getAccessLogs,
  getAccessLogSavedViews: vi.fn(async () => []),
  postAccessLogSavedView: vi.fn(),
  putAccessLogSavedView: vi.fn(),
}));
vi.mock('../../components/AccessLogDetailDrawer.vue', () => ({
  default: defineComponent({ setup: () => () => null }),
}));
vi.mock('../../components/AccessLogFilters.vue', () => ({
  default: defineComponent({
    props: ['activePreset'],
    emits: ['apply-preset', 'reset'],
    setup(props, { emit }) {
      return () =>
        h('section', [
          h('output', { 'data-testid': 'active-preset' }, props.activePreset),
          h('button', { 'data-testid': 'apply-status5xx', onClick: () => emit('apply-preset', 'status5xx') }, '5xx'),
          h('button', { 'data-testid': 'reset-filters', onClick: () => emit('reset') }, 'reset'),
        ]);
    },
  }),
}));
vi.mock('../../components/AccessLogTable.vue', () => ({ default: defineComponent({ setup: () => () => null }) }));
vi.mock('../../shared/presentation', () => ({
  buildAccessLogSortOptions: vi.fn(() => [{ label: 'started', value: 'started_at' }]),
}));

function mountPage() {
  return mount(AccessLogListIndex, {
    global: {
      stubs: { 't-button': true },
    },
  });
}

describe('AccessLogListIndex', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    queryClient.clear();
    queryClient.setDefaultOptions({ queries: { retry: false, staleTime: 0 } });
    routeState.query = {};
    mocks.routerReplace.mockImplementation((location: { query: Record<string, unknown> }) => {
      routeState.query = location.query;
      return Promise.resolve();
    });
    mocks.getAccessLogs.mockResolvedValue({ items: [], page: 1, page_size: 20, total: 0 });
  });

  it('restores the quick preset from the route and clears it for the default route', async () => {
    routeState.query = { quick_preset: 'status5xx', status_group: '5xx' };
    const wrapper = mountPage();
    await flushPromises();
    expect(wrapper.get('[data-testid="active-preset"]').text()).toBe('status5xx');

    routeState.query = {};
    await flushPromises();
    expect(wrapper.get('[data-testid="active-preset"]').text()).toBe('all');
  });

  it('keeps the quick preset out of the access-log API query', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="apply-status5xx"]').trigger('click');
    await flushPromises();
    expect(mocks.routerReplace).toHaveBeenLastCalledWith(
      expect.objectContaining({ query: expect.objectContaining({ quick_preset: 'status5xx', status_group: '5xx' }) }),
    );
    expect(mocks.getAccessLogs).toHaveBeenLastCalledWith(expect.not.objectContaining({ quick_preset: 'status5xx' }));

    await wrapper.get('[data-testid="reset-filters"]').trigger('click');
    await flushPromises();
    expect(mocks.routerReplace).toHaveBeenLastCalledWith(
      expect.objectContaining({ query: expect.objectContaining({ quick_preset: '' }) }),
    );
  });
});
