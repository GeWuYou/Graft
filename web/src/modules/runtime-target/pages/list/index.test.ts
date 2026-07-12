import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';

import RuntimeTargetListPage from './index.vue';

const apiMocks = vi.hoisted(() => ({
  discoverLocalDocker: vi.fn(),
  listRuntimeTargetPage: vi.fn(),
  refreshRuntimeTarget: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({ success: vi.fn() }));

vi.mock('../../api/runtime-target', () => apiMocks);
vi.mock('tdesign-vue-next/es/message', () => ({ MessagePlugin: messageMocks }));
vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string, values?: Record<string, unknown>) => `${key}:${values?.count ?? ''}` }),
}));

const passthrough = (name: string) =>
  defineComponent({
    name,
    template: '<div><slot name="actions" /><slot /><slot name="action" /></div>',
  });

function mountPage() {
  return mount(RuntimeTargetListPage, {
    global: {
      stubs: {
        'management-page-content': passthrough('ManagementPageContent'),
        'management-page-header': passthrough('ManagementPageHeader'),
        'management-table-pagination': passthrough('ManagementTablePagination'),
        't-loading': passthrough('TLoading'),
        't-row': passthrough('TRow'),
        't-col': passthrough('TCol'),
        't-empty': passthrough('TEmpty'),
        't-tag': passthrough('TTag'),
        't-tooltip': passthrough('TTooltip'),
        't-button': defineComponent({
          name: 'TButton',
          emits: ['click'],
          template: '<button @click="$emit(\'click\')"><slot name="icon" /><slot /></button>',
        }),
        't-pagination': defineComponent({
          name: 'TPagination',
          props: { pageSizeOptions: { type: Array, default: () => [] } },
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
      items: [
        {
          id: 7,
          displayName: 'Local Docker',
          endpointLabel: 'unix:///var/run/docker.sock',
          availability: true,
          summary: {
            containers: { available: true, total: 12, running: 11, stopped: 1, unavailableReason: '' },
            images: { available: true, total: 21, used: 10, unused: 11, unavailableReason: '' },
            cpu: { available: true, usagePercent: 0.5, usedBytes: 0, totalBytes: 0, unavailableReason: '' },
            memory: {
              available: false,
              usagePercent: 0,
              usedBytes: 0,
              totalBytes: 0,
              unavailableReason: 'not supported',
            },
            disk: { available: true, usagePercent: 7, usedBytes: 1_024, totalBytes: 2_048, unavailableReason: '' },
          },
        },
      ],
      total: 21,
      limit: 10,
      offset: 0,
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(apiMocks.listRuntimeTargetPage).toHaveBeenCalledWith({ limit: 10, offset: 0 });
    expect(wrapper.find('t-table').exists()).toBe(true);
    expect(wrapper.get('[data-testid="pagination"]').attributes('data-options')).toBe('10,20,50,100');
  });

  it('scans Local Docker and reloads the first page', async () => {
    apiMocks.listRuntimeTargetPage.mockResolvedValue({ items: [], total: 0, limit: 10, offset: 0 });
    apiMocks.discoverLocalDocker.mockResolvedValue(null);
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="runtime-target-discover-local"]').trigger('click');
    await flushPromises();

    expect(apiMocks.discoverLocalDocker).toHaveBeenCalledOnce();
    expect(apiMocks.listRuntimeTargetPage).toHaveBeenCalledTimes(2);
    expect(messageMocks.success).toHaveBeenCalledWith('runtimeTarget.list.discoverSuccess:');
  });
});
