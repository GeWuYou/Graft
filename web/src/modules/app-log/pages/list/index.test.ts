import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { computed, defineComponent, h, reactive, ref } from 'vue';

import { queryClient } from '@/shared/query';

import type { AppLogItem, AppLogListResponse } from '../../types/app-log';
import AppLogListIndex from './index.vue';

const mocks = vi.hoisted(() => ({
  getAppLogs: vi.fn(),
  routerReplace: vi.fn(),
  messageError: vi.fn(),
  loggerError: vi.fn(),
}));

const routeState = reactive<{ query: Record<string, unknown> }>({ query: {} });

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => ({ replace: mocks.routerReplace }),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}));

vi.mock('tdesign-vue-next', () => ({
  DialogPlugin: { confirm: vi.fn() },
  MessagePlugin: { error: mocks.messageError, success: vi.fn() },
}));

vi.mock('@/shared/components/management', () => ({
  TableViewToolbar: defineComponent({
    emits: ['refresh'],
    setup(_, { emit }) {
      return () => h('button', { 'data-testid': 'refresh', onClick: () => emit('refresh') }, 'refresh');
    },
  }),
}));

vi.mock('@/shared/components/query-list', () => ({
  AdvancedQueryColumnDrawer: defineComponent({ setup: () => () => null }),
  AdvancedQueryListPage: defineComponent({
    setup(_, { slots }) {
      return () => h('main', [slots.filters?.(), slots.table?.(), slots.detail?.()]);
    },
  }),
  applySavedQueryViewPresentation: vi.fn(),
  normalizeSavedQueryView: vi.fn(),
  useSavedQueryViews: () => ({
    applying: ref(false),
    deleting: ref(false),
    hasSelectedView: computed(() => false),
    isBusy: computed(() => false),
    load: vi.fn(async () => true),
    loading: ref(false),
    removeSelected: vi.fn(async () => true),
    save: vi.fn(async () => true),
    selectedId: ref(undefined),
    selectedView: computed(() => undefined),
    select: vi.fn(async () => true),
    submitting: ref(false),
    views: ref([]),
  }),
}));

vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

vi.mock('@/shared/observability', () => ({
  assignEncodedSorters: vi.fn(),
  buildRecentHoursLocalRange: vi.fn(() => []),
  createLogDetailErrorReporter: vi.fn(() => vi.fn()),
  createSingleSorter: vi.fn((field: string, order: string) => ({ field, order })),
  decodeSorters: vi.fn(() => []),
  encodeSorters: vi.fn(() => []),
  localDateTimeToUtcIso: vi.fn((value: string) => value),
  normalizePageStateRangeForRoute: vi.fn((value: string[]) => value),
  normalizeRouteRangeForPageState: vi.fn((value: unknown[]) => value.filter(Boolean)),
  normalizeSorters: vi.fn((value: unknown[]) => value),
  openLogDetailRow: vi.fn(),
  parseLogRouteQuery: vi.fn((query: Record<string, unknown>) => query),
  buildLogListLocation: vi.fn((_path: string, _keys: string[], query: Record<string, unknown>) => ({ query })),
  restartLogListQuery: vi.fn(),
}));

vi.mock('@/store', () => ({
  usePermissionStore: () => ({ hasPermission: () => true }),
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({ error: mocks.loggerError }),
}));

vi.mock('../../api/app-log', () => ({
  deleteAppLog: vi.fn(),
  deleteAppLogs: vi.fn(),
  deleteAppLogSavedView: vi.fn(),
  getAppLogDetail: vi.fn(),
  getAppLogs: mocks.getAppLogs,
  getAppLogSavedViews: vi.fn(async () => []),
  postAppLogSavedView: vi.fn(),
  putAppLogSavedView: vi.fn(),
}));

vi.mock('../../components/AppLogDetailDrawer.vue', () => ({ default: defineComponent({ setup: () => () => null }) }));
vi.mock('../../components/AppLogFilters.vue', () => ({ default: defineComponent({ setup: () => () => null }) }));
vi.mock('../../components/AppLogTable.vue', () => ({
  default: defineComponent({
    props: {
      emptyDescription: { type: String, default: '' },
      emptyTitle: { type: String, default: '' },
      filteredEmpty: { type: Boolean, default: false },
      rows: { type: Array, default: () => [] },
      total: { type: Number, default: 0 },
    },
    emits: ['clear-filters'],
    setup(props, { emit, slots }) {
      return () =>
        h('section', [
          h('output', { 'data-testid': 'rows' }, JSON.stringify(props.rows)),
          h('output', { 'data-testid': 'total' }, String(props.total)),
          h('output', { 'data-testid': 'empty-title' }, props.emptyTitle),
          h('output', { 'data-testid': 'empty-description' }, props.emptyDescription),
          h('output', { 'data-testid': 'filtered-empty' }, String(props.filteredEmpty)),
          h('button', { 'data-testid': 'clear-filters', onClick: () => emit('clear-filters') }, 'clear'),
          slots.toolbar?.(),
        ]);
    },
  }),
}));

function appLog(id: number): AppLogItem {
  return {
    category: 'application',
    component: 'app',
    error: '',
    fields: {},
    id,
    message: `message-${id}`,
    method: 'GET',
    occurred_at: '2026-07-16T00:00:00Z',
    operation: 'list',
    request_id: `request-${id}`,
    severity: 'info',
  } as AppLogItem;
}

function response(items: AppLogItem[]): AppLogListResponse {
  return { items, page: 1, page_size: 20, total: items.length } as AppLogListResponse;
}

function mountPage() {
  return mount(AppLogListIndex, {
    global: {
      directives: { permission: () => undefined },
      stubs: {
        't-button': true,
        't-tab-panel': true,
        't-tabs': true,
      },
    },
  });
}

describe('AppLogListIndex', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    queryClient.clear();
    queryClient.setDefaultOptions({ queries: { retry: false, staleTime: 0 } });
    routeState.query = {};
  });

  it('clears list state and reports an initial-load failure', async () => {
    const error = new Error('initial failure');
    mocks.getAppLogs.mockRejectedValue(error);

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.get('[data-testid="rows"]').text()).toBe('[]');
    expect(wrapper.get('[data-testid="total"]').text()).toBe('0');
    expect(mocks.messageError).toHaveBeenCalledWith('appLog.page.loadFailed');
    expect(mocks.loggerError).toHaveBeenCalledWith('failed to fetch app logs', error);
  });

  it('preserves the last successful list when a background refetch fails', async () => {
    const existing = response([appLog(7)]);
    const error = new Error('refetch failure');
    mocks.getAppLogs.mockResolvedValueOnce(existing).mockRejectedValueOnce(error);

    const wrapper = mountPage();
    await flushPromises();
    expect(wrapper.get('[data-testid="rows"]').text()).toContain('message-7');
    expect(wrapper.get('[data-testid="total"]').text()).toBe('1');

    await wrapper.get('[data-testid="refresh"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="rows"]').text()).toContain('message-7');
    expect(wrapper.get('[data-testid="total"]').text()).toBe('1');
    expect(mocks.messageError).toHaveBeenCalledWith('appLog.page.loadFailed');
    expect(mocks.loggerError).toHaveBeenCalledWith('failed to fetch app logs', error);
  });

  it('preserves an arbitrary category deep-link until server validation', async () => {
    routeState.query = { category: 'future.category' };
    mocks.getAppLogs.mockResolvedValue(response([]));

    mountPage();
    await flushPromises();

    expect(mocks.getAppLogs).toHaveBeenCalledWith(expect.objectContaining({ category: 'future.category' }));
  });

  it('keeps the result surface and supplies reset recovery for filtered empty responses', async () => {
    routeState.query = { category: 'future.category' };
    mocks.getAppLogs.mockResolvedValue(response([]));

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.get('[data-testid="rows"]').text()).toBe('[]');
    expect(wrapper.get('[data-testid="filtered-empty"]').text()).toBe('true');
    expect(wrapper.get('[data-testid="empty-title"]').text()).toBe('appLog.page.emptyFilteredTitle');

    await wrapper.get('[data-testid="clear-filters"]').trigger('click');
    await flushPromises();

    expect(mocks.getAppLogs).toHaveBeenLastCalledWith(expect.not.objectContaining({ category: 'future.category' }));
  });
});
