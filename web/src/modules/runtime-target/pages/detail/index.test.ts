import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';

import RuntimeTargetDetailPage from './index.vue';

const apiMocks = vi.hoisted(() => ({ getRuntimeTarget: vi.fn(), refreshRuntimeTarget: vi.fn() }));
const routeState = vi.hoisted(() => ({ params: { id: '7' } }));

vi.mock('../../api/runtime-target', () => apiMocks);
vi.mock('vue-router', () => ({ useRoute: () => routeState }));
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));
vi.mock('@/locales/useLocale', () => ({ useLocale: () => ({ locale: 'en-US' }) }));
vi.mock('@/shared/observability', () => ({
  formatBytes: (value: number) => `${value} B`,
  formatLocaleDateTime: (value: string, locale: string) => `${locale}:${value}`,
}));

const passthrough = (name: string) => defineComponent({ name, template: '<div><slot /><slot name="actions" /></div>' });
const statisticStub = defineComponent({
  name: 'TStatistic',
  props: {
    title: { type: String, default: '' },
    value: { type: [String, Number], default: '' },
    extra: { type: String, default: '' },
  },
  template: '<div>{{ title }} {{ value }} {{ extra }}</div>',
});
const buttonStub = defineComponent({
  name: 'TButton',
  emits: ['click'],
  template: '<button @click="$emit(\'click\')"><slot /></button>',
});
const alertStub = defineComponent({
  name: 'TAlert',
  props: { message: { type: String, default: '' } },
  template: '<div>{{ message }}<slot /></div>',
});

describe('RuntimeTargetDetailPage', () => {
  it('renders Docker provider details separately from neutral runtime resources', async () => {
    apiMocks.getRuntimeTarget.mockResolvedValue({
      id: 7,
      displayName: 'Local Docker',
      runtime: { provider: 'docker', type: 'container_runtime', version: '27.0', apiVersion: '1.46' },
      connection: { endpoint: 'unix:///var/run/docker.sock', kind: 'unix_socket' },
      health: { status: 'healthy', lastCheckedAt: '2026-07-13T00:00:00Z', diagnostic: '' },
      resources: {
        workloads: { available: true, total: 6, active: 5, unavailableReason: '' },
        cpu: { available: true, usagePercent: 10, usedBytes: 0, totalBytes: 0, unavailableReason: '' },
        memory: { available: true, usagePercent: 20, usedBytes: 20, totalBytes: 100, unavailableReason: '' },
        storage: { available: true, usagePercent: 30, usedBytes: 30, totalBytes: 100, unavailableReason: '' },
      },
      providerDetails: {
        provider: 'docker',
        docker: {
          images: { available: true, total: 11, unavailableReason: '' },
          volumes: { available: true, total: 12, unavailableReason: '' },
          networks: { available: false, total: 0, unavailableReason: 'network metrics unavailable' },
        },
      },
    });

    const wrapper = mount(RuntimeTargetDetailPage, {
      global: {
        stubs: {
          'management-page-content': passthrough('ManagementPageContent'),
          'management-page-header': passthrough('ManagementPageHeader'),
          't-alert': alertStub,
          't-card': passthrough('TCard'),
          't-descriptions': passthrough('TDescriptions'),
          't-descriptions-item': passthrough('TDescriptionsItem'),
          't-tag': passthrough('TTag'),
          't-statistic': statisticStub,
          't-empty': passthrough('TEmpty'),
          't-button': buttonStub,
        },
      },
    });
    await flushPromises();

    expect(apiMocks.getRuntimeTarget).toHaveBeenCalledWith(7);
    expect(wrapper.text()).toContain('11');
    expect(wrapper.text()).toContain('12');
    expect(wrapper.text()).toContain('network metrics unavailable');
    expect(wrapper.text()).toContain('en-US:2026-07-13T00:00:00Z');
  });

  it('shows unavailable reasons instead of zero for unavailable metrics', async () => {
    apiMocks.getRuntimeTarget.mockResolvedValue({
      id: 7,
      displayName: 'Local Docker',
      runtime: { provider: 'docker', type: 'container_runtime', version: '27.0', apiVersion: '1.46' },
      connection: { endpoint: 'unix:///var/run/docker.sock', kind: 'unix_socket' },
      health: { status: 'healthy', lastCheckedAt: null, diagnostic: '' },
      resources: {
        workloads: { available: false, total: 0, active: 0, unavailableReason: 'workloads unavailable' },
        cpu: { available: false, usagePercent: 0, usedBytes: 0, totalBytes: 0, unavailableReason: 'cpu unavailable' },
        memory: { available: true, usagePercent: 20, usedBytes: 20, totalBytes: 100, unavailableReason: '' },
        storage: { available: true, usagePercent: 30, usedBytes: 30, totalBytes: 100, unavailableReason: '' },
      },
      providerDetails: {
        provider: 'docker',
        docker: {
          images: { available: true, total: 1, unavailableReason: '' },
          volumes: { available: true, total: 2, unavailableReason: '' },
          networks: { available: true, total: 3, unavailableReason: '' },
        },
      },
    });

    const wrapper = mount(RuntimeTargetDetailPage, {
      global: {
        stubs: {
          'management-page-content': passthrough('ManagementPageContent'),
          'management-page-header': passthrough('ManagementPageHeader'),
          't-alert': alertStub,
          't-card': passthrough('TCard'),
          't-descriptions': passthrough('TDescriptions'),
          't-descriptions-item': passthrough('TDescriptionsItem'),
          't-tag': passthrough('TTag'),
          't-statistic': statisticStub,
          't-empty': passthrough('TEmpty'),
          't-button': buttonStub,
        },
      },
    });
    await flushPromises();

    expect(wrapper.text()).toContain('workloads unavailable');
    expect(wrapper.text()).toContain('cpu unavailable');
  });

  it('does not refresh when the route id is not a positive integer', async () => {
    routeState.params.id = 'not-an-id';
    const wrapper = mount(RuntimeTargetDetailPage, {
      global: {
        stubs: {
          'management-page-content': passthrough('ManagementPageContent'),
          'management-page-header': passthrough('ManagementPageHeader'),
          't-alert': alertStub,
          't-card': passthrough('TCard'),
          't-descriptions': passthrough('TDescriptions'),
          't-descriptions-item': passthrough('TDescriptionsItem'),
          't-tag': passthrough('TTag'),
          't-statistic': statisticStub,
          't-empty': passthrough('TEmpty'),
          't-button': buttonStub,
        },
      },
    });
    await flushPromises();

    await wrapper.get('button').trigger('click');

    expect(apiMocks.refreshRuntimeTarget).not.toHaveBeenCalled();
    expect(wrapper.text()).toContain('runtimeTarget.detail.refreshError');
    routeState.params.id = '7';
  });
});
