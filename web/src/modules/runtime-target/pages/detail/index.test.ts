import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';

import RuntimeTargetDetailPage from './index.vue';

const apiMocks = vi.hoisted(() => ({ getRuntimeTarget: vi.fn(), refreshRuntimeTarget: vi.fn() }));

vi.mock('../../api/runtime-target', () => apiMocks);
vi.mock('vue-router', () => ({ useRoute: () => ({ params: { id: '7' } }) }));
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }));
vi.mock('@/shared/observability', () => ({ formatBytes: (value: number) => `${value} B` }));

const passthrough = (name: string) => defineComponent({ name, template: '<div><slot /></div>' });

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
          't-alert': passthrough('TAlert'),
          't-card': passthrough('TCard'),
          't-descriptions': passthrough('TDescriptions'),
          't-descriptions-item': passthrough('TDescriptionsItem'),
          't-tag': passthrough('TTag'),
          't-statistic': passthrough('TStatistic'),
          't-empty': passthrough('TEmpty'),
          't-button': passthrough('TButton'),
        },
      },
    });
    await flushPromises();

    expect(apiMocks.getRuntimeTarget).toHaveBeenCalledWith(7);
    expect(wrapper.text()).toContain('11');
    expect(wrapper.text()).toContain('12');
    expect(wrapper.text()).toContain('network metrics unavailable');
  });
});
