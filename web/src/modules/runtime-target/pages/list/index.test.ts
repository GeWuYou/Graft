import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';

import RuntimeTargetListPage from './index.vue';

const apiMocks = vi.hoisted(() => ({
  discoverLocalDocker: vi.fn(),
  listRuntimeTargetPage: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({ success: vi.fn() }));
const realtimeMocks = vi.hoisted(() => ({
  openRealtimeTopicSocket: vi.fn<
    (options: { topic: string; onMessage: (payload: { topic: string; items: unknown[] }) => void }) => {
      close: () => void;
      reconnect: () => void;
    }
  >(() => ({ close: vi.fn(), reconnect: vi.fn() })),
}));

vi.mock('../../api/runtime-target', () => apiMocks);
vi.mock('tdesign-vue-next/es/message', () => ({ MessagePlugin: messageMocks }));
vi.mock('@/shared/realtime', () => ({
  isRealtimePayloadObject: (value: unknown) => Boolean(value && typeof value === 'object' && !Array.isArray(value)),
  openRealtimeTopicSocket: realtimeMocks.openRealtimeTopicSocket,
  parseRealtimeEnvelopeData: (raw: unknown) => {
    try {
      const parsed = JSON.parse(String(raw)) as { data?: unknown };
      return parsed.data;
    } catch {
      return null;
    }
  },
}));
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string, values?: Record<string, unknown>) => `${key}:${values?.count ?? ''}` }),
}));

const passthrough = (name: string) =>
  defineComponent({
    name,
    template: '<div><slot name="actions" /><slot /><slot name="action" /></div>',
  });

function target(id: number) {
  return {
    id,
    displayName: `Target ${id}`,
    runtime: { provider: 'docker', type: 'container_runtime', version: '27.5.0', apiVersion: '1.46' },
    connection: { endpoint: `unix:///target-${id}.sock`, kind: 'unix_socket' },
    health: { status: 'healthy', lastCheckedAt: '2026-07-13T00:00:00Z', diagnostic: '' },
    resources: {
      workloads: { available: true, total: id, active: id, unavailableReason: '' },
      cpu: { available: true, usagePercent: id, usedBytes: 0, totalBytes: 0, unavailableReason: '' },
      memory: { available: true, usagePercent: id, usedBytes: id, totalBytes: 100, unavailableReason: '' },
      storage: { available: true, usagePercent: id, usedBytes: id, totalBytes: 100, unavailableReason: '' },
    },
  };
}

function mountPage() {
  return mount(RuntimeTargetListPage, {
    global: {
      stubs: {
        'management-page-content': passthrough('ManagementPageContent'),
        'management-page-header': passthrough('ManagementPageHeader'),
        'management-table-pagination': passthrough('ManagementTablePagination'),
        'responsive-table': defineComponent({
          name: 'ResponsiveTable',
          template: '<div data-testid="runtime-target-responsive-table"><slot name="cards" /><slot /></div>',
        }),
        'router-link': defineComponent({ name: 'RouterLink', template: '<a><slot /></a>' }),
        't-loading': passthrough('TLoading'),
        't-row': passthrough('TRow'),
        't-col': passthrough('TCol'),
        't-empty': passthrough('TEmpty'),
        't-tag': passthrough('TTag'),
        't-tooltip': passthrough('TTooltip'),
        't-table': defineComponent({
          name: 'TTable',
          props: {
            data: { type: Array, default: () => [] },
            loading: { type: Boolean, default: false },
          },
          template:
            '<div data-testid="runtime-target-table" :data-ids="data.map((row) => row.id).join(\',\')" :data-loading="String(loading)" />',
        }),
        't-button': defineComponent({
          name: 'TButton',
          emits: ['click'],
          template: '<button @click="$emit(\'click\')"><slot name="icon" /><slot /></button>',
        }),
        't-pagination': defineComponent({
          name: 'TPagination',
          props: { pageSizeOptions: { type: Array, default: () => [] } },
          emits: ['update:current'],
          template: '<div data-testid="pagination" :data-options="pageSizeOptions.join(\',\')" />',
        }),
        't-progress': defineComponent({ name: 'TProgress', template: '<div class="progress" />' }),
      },
    },
  });
}

describe('RuntimeTargetListPage', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('loads paged target overview cards with the shared page-size choices', async () => {
    apiMocks.listRuntimeTargetPage.mockResolvedValue({
      items: [target(7)],
      total: 21,
      limit: 10,
      offset: 0,
      summary: { total: 21, healthy: 18, unavailable: 3 },
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(apiMocks.listRuntimeTargetPage).toHaveBeenCalledWith({ limit: 10, offset: 0 });
    expect(wrapper.get('[data-testid="runtime-target-table"]').attributes('data-ids')).toBe('7');
    expect(wrapper.get('[data-testid="pagination"]').attributes('data-options')).toBe('10,20,50,100');
  });

  it('scans Local Docker and reloads the first page', async () => {
    apiMocks.listRuntimeTargetPage.mockResolvedValue({
      items: [],
      total: 0,
      limit: 10,
      offset: 0,
      summary: { total: 0, healthy: 0, unavailable: 0 },
    });
    apiMocks.discoverLocalDocker.mockResolvedValue(null);
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="runtime-target-discover-local-empty"]').trigger('click');
    await flushPromises();

    expect(apiMocks.discoverLocalDocker).toHaveBeenCalledOnce();
    expect(apiMocks.listRuntimeTargetPage).toHaveBeenCalledTimes(2);
    expect(messageMocks.success).toHaveBeenCalledWith('runtimeTarget.list.discoverSuccess:');
  });

  it('keeps every runtime resource metric and the detail link in the card presentation', async () => {
    apiMocks.listRuntimeTargetPage.mockResolvedValue({
      items: [target(7)],
      total: 1,
      limit: 10,
      offset: 0,
      summary: { total: 1, healthy: 1, unavailable: 0 },
    });
    const wrapper = mountPage();
    await flushPromises();

    const card = wrapper.get('[data-testid="runtime-target-card-7"]');
    expect(card.text()).toContain('Target 7');
    expect(card.text()).toContain('unix:///target-7.sock');
    expect(card.text()).toContain('runtimeTarget.metrics.workloads:');
    expect(card.text()).toContain('runtimeTarget.metrics.cpu:');
    expect(card.text()).toContain('runtimeTarget.metrics.memory:');
    expect(card.text()).toContain('runtimeTarget.metrics.storage:');
    expect(card.text()).toContain('runtimeTarget.list.viewDetail:');
    expect(wrapper.get('[data-testid="runtime-target-discover-local"]').text()).toBe('');
  });

  it('patches the current page from a realtime snapshot without reloading the table', async () => {
    apiMocks.listRuntimeTargetPage.mockResolvedValue({
      items: [target(7)],
      total: 1,
      limit: 10,
      offset: 0,
      summary: { total: 1, healthy: 1, unavailable: 0 },
    });
    const wrapper = mountPage();
    await flushPromises();

    const options = realtimeMocks.openRealtimeTopicSocket.mock.calls[0]?.[0];
    expect(options?.topic).toBe('runtime-target.summary.list');
    options?.onMessage({
      topic: 'runtime-target.summary.list',
      items: [target(7)],
    });
    await flushPromises();

    expect(apiMocks.listRuntimeTargetPage).toHaveBeenCalledOnce();
    expect(wrapper.get('[data-testid="runtime-target-table"]').attributes('data-ids')).toBe('7');
  });

  it('subscribes while the initial page is empty and fills it from a realtime snapshot', async () => {
    apiMocks.listRuntimeTargetPage.mockResolvedValue({
      items: [],
      total: 0,
      limit: 10,
      offset: 0,
      summary: { total: 0, healthy: 0, unavailable: 0 },
    });
    const wrapper = mountPage();
    await flushPromises();

    const options = realtimeMocks.openRealtimeTopicSocket.mock.calls[0]?.[0];
    expect(options?.topic).toBe('runtime-target.summary.list');

    options?.onMessage({ topic: 'runtime-target.summary.list', items: [target(1)] });
    await flushPromises();

    expect(wrapper.get('[data-testid="runtime-target-table"]').attributes('data-ids')).toBe('1');
    expect(apiMocks.listRuntimeTargetPage).toHaveBeenCalledOnce();
  });

  it('reconciles a realtime snapshot to the current page window', async () => {
    apiMocks.listRuntimeTargetPage.mockResolvedValue({
      items: [target(11)],
      total: 11,
      limit: 10,
      offset: 10,
      summary: { total: 11, healthy: 11, unavailable: 0 },
    });
    const wrapper = mountPage();
    await flushPromises();

    const options = realtimeMocks.openRealtimeTopicSocket.mock.calls[0]?.[0];
    const snapshot = Array.from({ length: 12 }, (_, index) => target(index + 1));
    wrapper.findComponent({ name: 'TPagination' }).vm.$emit('update:current', 2);
    options?.onMessage({ topic: 'runtime-target.summary.list', items: snapshot });
    await flushPromises();

    expect(wrapper.get('[data-testid="runtime-target-table"]').attributes('data-ids')).toBe('11,12');
  });
});
